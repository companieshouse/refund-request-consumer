# refund-request-consumer
[![GoDoc](https://godoc.org/github.com/companieshouse/refund-request-consumer?status.svg)](https://godoc.org/github.com/companieshouse/refund-request-consumer)
[![Go Report Card](https://goreportcard.com/badge/github.com/companieshouse/refund-request-consumer)](https://goreportcard.com/report/github.com/companieshouse/refund-request-consumer)

Consumer that reads the refund-request topic and starts the refund process in the payment service.

## Running locally with Docker CHS
Clone [docker-chs-development](https://github.com/companieshouse/docker-chs-development) and follow the steps in the README.

Enable the `refund-request-consumer` service. 

Send a kafka message to the `refund-request` topic

Development mode is available for this service in Docker CHS Development. 

`./bin/chs-dev development enable refund-request-consumer`

