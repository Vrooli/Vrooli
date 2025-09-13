#!/usr/bin/env bash
# BTCPay Server Resource Configuration Defaults

# Source port registry
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/resources/port_registry.sh"

# BTCPay constants
export BTCPAY_CONTAINER_NAME="btcpay-server"
export BTCPAY_POSTGRES_CONTAINER="btcpay-postgres"
export BTCPAY_NETWORK="btcpay-network"
export BTCPAY_IMAGE="btcpayserver/btcpayserver:1.13.5"
export BTCPAY_POSTGRES_IMAGE="postgres:14-alpine"
export BTCPAY_PORT="${BTCPAY_PORT:-$(ports::get_resource_port 'btcpay')}"
export BTCPAY_DATA_DIR="${var_DATA_DIR}/resources/btcpay"
export BTCPAY_CONFIG_DIR="${BTCPAY_DATA_DIR}/config"
export BTCPAY_POSTGRES_DATA="${BTCPAY_DATA_DIR}/postgres"
export BTCPAY_LOGS_DIR="${BTCPAY_DATA_DIR}/logs"

# BTCPay configuration
export BTCPAY_HOST="localhost:${BTCPAY_PORT}"
export BTCPAY_PROTOCOL="http"
export BTCPAY_BASE_URL="${BTCPAY_PROTOCOL}://${BTCPAY_HOST}"

# PostgreSQL configuration
export BTCPAY_POSTGRES_USER="btcpay"
export BTCPAY_POSTGRES_PASSWORD="btcpay_secure_password"
export BTCPAY_POSTGRES_DB="btcpayserver"

# Resource metadata
export BTCPAY_RESOURCE_NAME="btcpay"
export BTCPAY_RESOURCE_DESCRIPTION="BTCPay Server - Self-hosted, open-source cryptocurrency payment processor"