#!/bin/bash
#
# Start script for refund-request-consumer

APP_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# Read brokers from environment and split on comma
IFS=',' read -a BROKERS <<< "${KAFKA_BROKER_ADDR}"
# Ensure we only populate the broker address via application arguments
unset KAFKA_BROKER_ADDR
exec "${APP_DIR}/refund-request-consumer" $(for b in "${BROKERS[@]}"; do echo -n "-broker-addr=${b} "; done)
