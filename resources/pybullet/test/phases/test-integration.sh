#!/bin/bash

# PyBullet Integration Test
# Full functionality validation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source test library
source "$RESOURCE_DIR/lib/test.sh"

# Run integration test
test_integration