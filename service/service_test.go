package service

import (
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/companieshouse/chs.go/avro"
	consumer "github.com/companieshouse/chs.go/kafka/consumer/cluster"
	"github.com/companieshouse/chs.go/kafka/producer"
	"github.com/companieshouse/refund-request-consumer/data"
	"github.com/companieshouse/refund-request-consumer/payment"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

const paymentsAPIUrl = "paymentsAPIUrl"
const apiKey = "apiKey"
const paymentResourceID = "paymentResourceID"

func createMockService(mockPayment *payment.MockPayments) *Service {
	return &Service{
		Producer:            createMockProducer(),
		RefundRequestSchema: getDefaultSchema(),
		Payments:            mockPayment,
		PaymentsAPIURL:      paymentsAPIUrl,
		ApiKey:              apiKey,
		Client:              &http.Client{},
		Topic:               "test",
	}
}

func createMockConsumerWithRefundMessage(paymentId string) *consumer.GroupConsumer {
	return createMockConsumerWithMessage(1, paymentId, "100.00", "ref")
}

func createMockConsumerWithMessage(attempt int32, paymentId string, refundAmount string, refundReference string) *consumer.GroupConsumer {
	return &consumer.GroupConsumer{
		GConsumer: MockConsumer{Attempt: attempt, PaymentID: paymentId, RefundAmount: refundAmount, RefundReference: refundReference},
		Group:     MockGroup{},
	}
}

func createMockProducer() *producer.Producer {
	return &producer.Producer{
		SyncProducer: MockProducer{},
	}
}

type MockProducer struct {
	sarama.SyncProducer
}

func (m MockProducer) Close() error {
	return nil
}

func getDefaultSchema() string {
	return "{\"type\":\"record\",\"name\":\"refund_request\",\"namespace\":\"payments\",\"fields\":[{\"name\":\"attempt\",\"type\":\"int\"},{\"name\":\"payment_id\",\"type\":\"string\"},{\"name\":\"refund_amount\",\"type\":\"string\"},{\"name\":\"refund_reference\",\"type\":\"string\"}]}"
}

var MockSchema = &avro.Schema{
	Definition: getDefaultSchema(),
}

// endConsumerProcess facilitates service termination
func endConsumerProcess(svc *Service, c chan os.Signal) {

	// Increment the offset to escape an endless loop in the service
	svc.InitialOffset = int64(100)

	// Send a kill command to the input channel to terminate program execution
	go func() {
		c <- os.Kill
		close(c)
	}()
}

type MockConsumer struct {
	Attempt         int32
	PaymentID       string
	RefundAmount    string
	RefundReference string
}

func (m MockConsumer) prepareTestKafkaMessage() ([]byte, error) {
	return MockSchema.Marshal(data.RefundRequest{
		Attempt:         m.Attempt,
		PaymentID:       m.PaymentID,
		RefundAmount:    m.RefundAmount,
		RefundReference: m.RefundReference,
	})
}

func (m MockConsumer) Close() error {
	return nil
}

func (m MockConsumer) Messages() <-chan *sarama.ConsumerMessage {
	out := make(chan *sarama.ConsumerMessage)

	bytes, _ := m.prepareTestKafkaMessage()
	go func() {
		out <- &sarama.ConsumerMessage{
			Value: bytes,
		}
		close(out)
	}()

	return out
}

func (m MockConsumer) Errors() <-chan error {
	return nil
}

type MockGroup struct{}

func (m MockGroup) MarkOffset(msg *sarama.ConsumerMessage, metadata string) {}

func (m MockGroup) CommitOffsets() error {
	return nil
}

func TestUnitStart(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("Process of a single Kafka message for a refund", t, func() {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		c := make(chan os.Signal)

		mockPayment := payment.NewMockPayments(ctrl)

		svc := createMockService(mockPayment)

		Convey("Given a message containing refund id is readily available for the service to consume", func() {
			svc.Consumer = createMockConsumerWithRefundMessage(paymentResourceID)

			Convey("Then a refund request is sent to the Payments API", func() {
				mockPayment.EXPECT().RefundRequestPost(paymentsAPIUrl+"/payments/"+paymentResourceID+"/refunds", gomock.Any(), svc.Client, apiKey).Do(func(postURL string, postBody data.RefundPostRequest, HTTPClient *http.Client, apiKey string) {
					endConsumerProcess(svc, c)
				}).Times(1)

				svc.Start(wg, c)
			})
		})

		Convey("Error topic - Given a message containing refund id is readily available for the service to consume", func() {
			svc.Consumer = createMockConsumerWithRefundMessage(paymentResourceID)
			svc.IsErrorConsumer = true

			Convey("Then a refund request is sent to the Payments API", func() {
				mockPayment.EXPECT().RefundRequestPost(paymentsAPIUrl+"/payments/"+paymentResourceID+"/refunds", gomock.Any(), svc.Client, apiKey).Do(func(postURL string, postBody data.RefundPostRequest, HTTPClient *http.Client, apiKey string) {
					endConsumerProcess(svc, c)
				}).Times(1)

				svc.Start(wg, c)
			})
		})
	})
}

func TestUnitConvertToPenceFromDecimal(t *testing.T) {
	Convey("Convert decimal payment in pounds to pence", t, func() {
		amount, err := convertDecimalAmountToPence("116.32")
		So(err, ShouldBeNil)
		So(amount, ShouldEqual, 11632)
	})
}
