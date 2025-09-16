#!/usr/bin/env bash
# Open MCT Test Runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source the test library
source "${RESOURCE_DIR}/lib/test.sh"

# Determine which test phase to run
PHASE="${1:-all}"

# Run the tests
handle_test "$PHASE"