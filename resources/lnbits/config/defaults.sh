#!/usr/bin/env bash
# LNbits Resource Configuration Defaults

# Get base directories from environment or use defaults
export var_DATA_DIR="${DATA_DIR:-${HOME}/.vrooli/data}"

# LNbits constants
export LNBITS_CONTAINER_NAME="lnbits-server"
export LNBITS_POSTGRES_CONTAINER="lnbits-postgres"
export LNBITS_NETWORK="lnbits-network"
export LNBITS_IMAGE="lnbits/lnbits:latest"
export POSTGRES_IMAGE="postgres:14-alpine"

# Source port registry
PORT_REGISTRY="${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh"
if [[ -f "$PORT_REGISTRY" ]]; then
    source "$PORT_REGISTRY"
fi

# Port configuration (from port registry or environment)
export LNBITS_PORT="${LNBITS_PORT:-$(ports::get_resource_port 'lnbits')}"
if [[ -z "${LNBITS_PORT}" ]]; then
    echo "Error: LNBITS_PORT not found in port registry or environment" >&2
    exit 1
fi

# Data directories
export LNBITS_DATA_DIR="${var_DATA_DIR}/resources/lnbits"
export LNBITS_CONFIG_DIR="${LNBITS_DATA_DIR}/config"
export LNBITS_POSTGRES_DATA="${LNBITS_DATA_DIR}/postgres"
export LNBITS_LOGS_DIR="${LNBITS_DATA_DIR}/logs"
export LNBITS_EXTENSIONS_DIR="${LNBITS_DATA_DIR}/extensions"

# LNbits configuration
export LNBITS_HOST="localhost:${LNBITS_PORT}"
export LNBITS_PROTOCOL="http"
export LNBITS_BASE_URL="${LNBITS_PROTOCOL}://${LNBITS_HOST}"
export LNBITS_ADMIN_UI="${LNBITS_ADMIN_UI:-true}"
export LNBITS_SITE_TITLE="${LNBITS_SITE_TITLE:-Vrooli LNbits}"

# PostgreSQL configuration
export LNBITS_POSTGRES_USER="${LNBITS_POSTGRES_USER:-lnbits}"
export LNBITS_POSTGRES_PASSWORD="${LNBITS_POSTGRES_PASSWORD:-lnbits_secure_password_$(date +%s)}"
export LNBITS_POSTGRES_DB="${LNBITS_POSTGRES_DB:-lnbits}"

# Lightning backend configuration
export LNBITS_BACKEND_WALLET="${LNBITS_BACKEND_WALLET:-FakeWallet}"
export FAKE_WALLET_SECRET="${FAKE_WALLET_SECRET:-supersecretkey}"

# Resource metadata
export LNBITS_RESOURCE_NAME="lnbits"
export LNBITS_RESOURCE_DESCRIPTION="LNbits - Lightning Network wallet and payments system"
export RESOURCE_NAME="lnbits"