# refund-request-consumer
Consumer that reads the refund-request topic and starts the refund process in the payment service.

## Running locally with Docker CHS
Clone Docker CHS Development and follow the steps in the README.

Enable the `platform` module 

Send a kafka message to the `refund-request` topic

Development mode is available for this service in Docker CHS Development. 

`./bin/chs-dev development enable refund-request-consumer`

