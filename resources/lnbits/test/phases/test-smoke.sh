#!/usr/bin/env bash
# LNbits Smoke Test

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

echo "==================================="
echo "LNbits Smoke Test (<30s)"
echo "==================================="

# Run the smoke test from the test library
test_smoke