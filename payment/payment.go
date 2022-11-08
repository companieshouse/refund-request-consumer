package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/refund-request-consumer/data"
)

// InvalidPaymentAPIResponse is returned when an invalid status is returned
// from the payments api.
type InvalidPaymentAPIResponse struct {
	status int
}

func (e *InvalidPaymentAPIResponse) Error() string {
	return fmt.Sprintf("unexpected status returned from payments api: [%d]", e.status)
}

// Payments implements the payments endpoints.
type Payments interface {
	RefundRequestPost(refundRequestURL string, patchBody data.RefundPostRequest, HTTPClient *http.Client, apiKey string) error
}

// Payment implements the Payment Interface.
type Payment struct{}

// New returns a new implementation of the Payment Interface.
func New() *Payment {
	return &Payment{}
}

// RefundRequestPost executes a POST request to the specified URL.
func (impl *Payment) RefundRequestPost(patchURL string, patchBody data.RefundPostRequest, httpClient *http.Client, apiKey string) error {
	jsonValue, err := json.Marshal(patchBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", patchURL, bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	req.SetBasicAuth(apiKey, "")
	log.Trace("POST request to the refund request endpoint of the resource", log.Data{"Request": patchURL, "Body": patchBody})

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		return &InvalidPaymentAPIResponse{res.StatusCode}
	}

	return nil
}
