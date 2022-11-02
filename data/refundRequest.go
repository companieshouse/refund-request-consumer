// Package data contains the required data structures.
package data

// RefundRequest represents the avro schema.
type RefundRequest struct {
	Attempt         int32  `avro:"attempt"`
	PaymentID       string `avro:"payment_id"`
	RefundAmount    string `avro:"refund_amount"`
	RefundReference string `avro:"refund_reference"`
}
