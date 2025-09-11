#!/usr/bin/env bash
################################################################################
# CNCjs Integration Test Phase
# Complete functionality validation (<120s)
################################################################################

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Run integration tests
log::info "Starting CNCjs integration test phase..."
cncjs::test_integration
exit $?