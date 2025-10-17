#!/bin/bash

# OpenEMS API Validation Tests
# Tests REST, JSON-RPC, and automation workflow APIs

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/api_validator.sh"

echo "🔍 OpenEMS API Validation Tests"
echo "================================"

# Ensure service is running for tests
echo "Starting OpenEMS for API tests..."
"${RESOURCE_DIR}/cli.sh" manage start --wait &>/dev/null || {
    echo "⚠️  Using existing OpenEMS instance"
}

# Run full API validation suite
api::validate_all

# Store exit code
EXIT_CODE=$?

# Cleanup
echo ""
echo "Stopping test instance..."
"${RESOURCE_DIR}/cli.sh" manage stop &>/dev/null || true

exit $EXIT_CODE