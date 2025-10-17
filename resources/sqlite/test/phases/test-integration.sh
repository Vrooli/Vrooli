#!/usr/bin/env bash
################################################################################
# SQLite Resource - Integration Test Phase
#
# End-to-end testing of SQLite functionality
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Run integration test via test runner
"${RESOURCE_DIR}/test/run-tests.sh" integration