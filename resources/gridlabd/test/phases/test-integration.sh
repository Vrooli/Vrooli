#!/usr/bin/env bash
# GridLAB-D Resource - Integration Test Phase

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

# Run integration tests
test_integration
exit $?