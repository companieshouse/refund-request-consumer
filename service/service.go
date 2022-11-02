// Package service contains the controllers to consume and process kafka messages.
package service

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/companieshouse/chs.go/avro"
	"github.com/companieshouse/chs.go/avro/schema"
	"github.com/companieshouse/chs.go/kafka/client"
	consumer "github.com/companieshouse/chs.go/kafka/consumer/cluster"
	"github.com/companieshouse/chs.go/kafka/producer"
	"github.com/companieshouse/chs.go/kafka/resilience"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/refund-request-consumer/config"
	"github.com/companieshouse/refund-request-consumer/data"
	"github.com/davecgh/go-spew/spew"
)

// Service represents service config for refund-request-consumer.
type Service struct {
	Consumer            *consumer.GroupConsumer
	Producer            *producer.Producer
	RefundRequestSchema string
	InitialOffset       int64
	HandleError         func(err error, offset int64, str interface{}) error
	Topic               string
	Retry               *resilience.ServiceRetry
	IsErrorConsumer     bool
	BrokerAddr          []string
	//	    Client          *http.Client
	//		ApiKey          string
}

// New creates a new instance of service with a given consumerGroup name,
// consumerTopic, throttleRate and refund-request-consumer config.
func New(consumerTopic, consumerGroupName string, InitialOffset int64, cfg *config.Config, retry *resilience.ServiceRetry) (*Service, error) {

	schemaName := "refund-request"
	refundRequestSchema, err := schema.Get(cfg.SchemaRegistryURL, schemaName)
	if err != nil {
		e := fmt.Errorf("error receiving %s schema: %w", schemaName, err)
		log.Error(e)

		return nil, e
	}

	log.Info(fmt.Sprintf("Successfully received %s schema", schemaName))

	appName := cfg.Namespace()

	p, err := producer.New(&producer.Config{Acks: &producer.WaitForAll, BrokerAddrs: cfg.BrokerAddr})
	if err != nil {
		e := fmt.Errorf("error initialising producer: %w", err)
		log.Error(e)

		return nil, e
	}

	maxRetries := 0
	if retry != nil {
		maxRetries = retry.MaxRetries
	}

	log.Info("Start Request Create resilient Kafka service", log.Data{"base_topic": consumerTopic, "app_name": appName, "maxRetries": maxRetries, "producer": p})
	rh := resilience.NewHandler(consumerTopic, "consumer", retry, p, &avro.Schema{Definition: refundRequestSchema})

	// Work out what topic we're consuming from, depending on whether were processing resilience or error input
	topicName := consumerTopic
	if retry != nil {
		topicName = rh.GetRetryTopicName()
	}
	if cfg.IsErrorConsumer {
		topicName = rh.GetErrorTopicName()
	}

	consumerConfig := &consumer.Config{
		Topics:       []string{topicName},
		ZookeeperURL: cfg.ZookeeperURL,
		BrokerAddr:   cfg.BrokerAddr,
	}

	log.Info(fmt.Sprintf("attempting to join consumer group [%s], topic [%s]", consumerGroupName, topicName))
	var resetOffset bool

	groupConfig := &consumer.GroupConfig{
		GroupName:   consumerGroupName,
		ResetOffset: resetOffset,
		Chroot:      cfg.ZookeeperChroot,
	}

	c := consumer.NewConsumerGroup(consumerConfig)
	if err = c.JoinGroup(groupConfig); err != nil {
		log.Error(fmt.Errorf("error joining '"+consumerGroupName+"' consumer group", err))
		return nil, err
	}

	return &Service{
		Consumer: c,

		RefundRequestSchema: refundRequestSchema,
		HandleError:         rh.HandleError,
		Topic:               topicName,
		Retry:               retry,
		IsErrorConsumer:     cfg.IsErrorConsumer,
		BrokerAddr:          cfg.BrokerAddr,
		//		Client:              &http.Client{},
		//		ApiKey:              cfg.ChsAPIKey,
		//		PaymentsAPIURL:      cfg.PaymentsAPIURL,
	}, nil
}

// Start begins the service.
// Messages are consumed from the refund-request topic.
func (svc *Service) Start(wg *sync.WaitGroup, c chan os.Signal) {
	log.Info("service starting, consuming from " + svc.Topic + " topic")

	// If we're an error consumer, then capture the tail of the topic, and only consume up to that offset.
	stopAtOffset := int64(-1)
	var err error
	if svc.IsErrorConsumer {
		stopAtOffset, err = client.TopicOffset(svc.BrokerAddr, svc.Topic)
		if err != nil {
			log.Error(err)
		}
		log.Info(fmt.Sprintf("error queue consumer will stop when backlog offset reached: %d", stopAtOffset))
	}

	var message *sarama.ConsumerMessage

	// We want to stop the processing of the service if consuming from an
	// error queue if all messages that were initially in the queue have
	// been cleared using the stopAtOffset

	running := true
	for running && (stopAtOffset == -1 || message == nil || message.Offset < stopAtOffset) {

		if message != nil {
			// Commit the message we've just been processing before starting the next
			log.Trace(fmt.Sprintf("Committing message, offset: %d", message.Offset))
			svc.Consumer.MarkOffset(message, "")
			if err := svc.Consumer.CommitOffsets(); err != nil {
				log.Error(err, log.Data{"offset": message.Offset})
			}
		}

		if svc.Retry != nil && svc.Retry.ThrottleRate > 0 {
			time.Sleep(svc.Retry.ThrottleRate * time.Second)
		}

		select {
		case <-c:
			running = false

		case message = <-svc.Consumer.Messages():
			// Falls into this block when a message becomes available from consumer
			if message != nil {
				if message.Offset >= svc.InitialOffset {
					var rr data.RefundRequest
					refundRequestSchema := &avro.Schema{
						Definition: svc.RefundRequestSchema,
					}

					err = refundRequestSchema.Unmarshal(message.Value, &rr)
					if err != nil {
						log.Error(err, log.Data{"message_offset": message.Offset})
						handleErr := svc.HandleError(err, message.Offset, refundRequestSchema)
						if handleErr != nil {
							log.Error(fmt.Errorf("error handling error: %w", handleErr))
						}

						continue
					}

					// TODO ROE-1461 Call Payments API
					spew.Dump(rr) // TODO remove this temporary logging
				}
			}

		case err = <-svc.Consumer.Errors():
			log.Error(err, log.Data{"topic": svc.Topic})
		}
	}

	// We only get here if we're an error consumer and we've reached out stop offset
	// We will not consume any further messages, so disconnect consumer.
	svc.Shutdown()

	// The app must not exit until explicitly asked. If this happens in
	// a managed environment such as Mesos/Marathon, the app will get
	// restarted and will go on to consume further messages in the error
	// topic and chasing its own tail, if something is really broken.
	if running {
		select {
		case <-c: // Just wait for a shutdown event
			log.Info("Received close notification")
		}
	}

	wg.Done()

	log.Info("Service successfully shutdown")

}

func (svc *Service) Shutdown() {
	log.Info("Shutting down service")

	log.Info("Closing producer")
	err := svc.Producer.Close()
	if err != nil {
		log.Error(fmt.Errorf("error closing producer: %w", err))
	}
	log.Info("Producer successfully closed")

	log.Info("Closing consumer")
	err = svc.Consumer.Close()
	if err != nil {
		log.Error(fmt.Errorf("error closing consumer: %w", err))
	}
	log.Info("Consumer successfully closed")
}
