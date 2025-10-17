#!/usr/bin/env bash
################################################################################
# CNCjs Smoke Test Phase
# Quick validation that service is operational (<30s)
################################################################################

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Run smoke tests
log::info "Starting CNCjs smoke test phase..."
cncjs::test_smoke
exit $?