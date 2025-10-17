#!/usr/bin/env bash
################################################################################
# Mifos Integration Test
# 
# End-to-end testing of Mifos functionality
################################################################################

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIFOS_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${MIFOS_DIR}/../.." && pwd)"

# Use the Mifos CLI for testing
MIFOS_CLI="${MIFOS_DIR}/cli.sh"

# Run integration tests via CLI
"${MIFOS_CLI}" test integration