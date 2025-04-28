#!/bin/bash
#
# Start script for refund-request-consumer
APP_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
exec "${APP_DIR}/refund-request-consumer" "-bind-addr=:${PORT}"