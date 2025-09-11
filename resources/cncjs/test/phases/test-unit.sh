#!/usr/bin/env bash
################################################################################
# CNCjs Unit Test Phase
# Library function validation (<60s)
################################################################################

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Run unit tests
log::info "Starting CNCjs unit test phase..."
cncjs::test_unit
exit $?