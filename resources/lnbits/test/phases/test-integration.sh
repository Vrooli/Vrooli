#!/usr/bin/env bash
# LNbits Integration Test

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

echo "==================================="
echo "LNbits Integration Test (<120s)"
echo "==================================="

# Run the integration test from the test library
test_integration