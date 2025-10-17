#!/usr/bin/env bash
# OpenRocket Integration Tests

set -euo pipefail

RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

echo "==================== OpenRocket Integration Tests ===================="

# Run integration tests
openrocket_test_integration

exit $?