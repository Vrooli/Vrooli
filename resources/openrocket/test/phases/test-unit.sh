#!/usr/bin/env bash
# OpenRocket Unit Tests

set -euo pipefail

RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

echo "==================== OpenRocket Unit Tests ===================="

# Run unit tests
openrocket_test_unit

exit $?