package payment

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/refund-request-consumer/data"
	"github.com/stretchr/testify/assert"
)

// Mock data for testing
var mockRefundPostRequest = data.RefundPostRequest{
	Amount:          100,
	RefundReference: "Test refund",
}

func TestInvalidPaymentAPIResponse_Error(t *testing.T) {
	err := &InvalidPaymentAPIResponse{status: http.StatusBadRequest}
	expected := "unexpected status returned from payments api: [400]"
	assert.Equal(t, expected, err.Error())
}

func TestNew(t *testing.T) {
	payment := New()
	assert.NotNil(t, payment)
}

func TestRefundRequestPost_Success(t *testing.T) {
	payment := New()
	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) *http.Response {
			recorder := httptest.NewRecorder()
			recorder.WriteHeader(http.StatusCreated)
			return recorder.Result()
		}),
	}

	err := payment.RefundRequestPost("http://example.com", mockRefundPostRequest, mockClient, "test-api-key")
	assert.NoError(t, err)
}

func TestRefundRequestPost_Failure(t *testing.T) {
	payment := New()
	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) *http.Response {
			recorder := httptest.NewRecorder()
			recorder.WriteHeader(http.StatusBadRequest)
			return recorder.Result()
		}),
	}

	err := payment.RefundRequestPost("http://example.com", mockRefundPostRequest, mockClient, "test-api-key")
	assert.Error(t, err)
	assert.IsType(t, &InvalidPaymentAPIResponse{}, err)
}

// roundTripFunc is a helper function to mock http.Client
type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}
