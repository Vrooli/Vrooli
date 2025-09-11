#!/usr/bin/env bash
# LNbits Unit Test

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

echo "==================================="
echo "LNbits Unit Test (<60s)"
echo "==================================="

# Run the unit test from the test library
test_unit