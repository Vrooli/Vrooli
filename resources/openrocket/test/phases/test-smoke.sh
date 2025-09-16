#!/usr/bin/env bash
# OpenRocket Smoke Tests

set -euo pipefail

RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

echo "==================== OpenRocket Smoke Tests ===================="

# Run smoke tests
openrocket_test_smoke

exit $?