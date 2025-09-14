#!/bin/bash

# PyBullet Smoke Test
# Quick health validation (<30s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source test library
source "$RESOURCE_DIR/lib/test.sh"

# Run smoke test
test_smoke