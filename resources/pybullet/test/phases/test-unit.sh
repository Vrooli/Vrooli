#!/bin/bash

# PyBullet Unit Test
# Library function validation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source test library
source "$RESOURCE_DIR/lib/test.sh"

# Run unit test
test_unit