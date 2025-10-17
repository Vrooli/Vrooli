#!/usr/bin/env bash
# Open MCT Unit Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source the test library
source "${RESOURCE_DIR}/lib/test.sh"

# Run unit tests
run_unit_tests