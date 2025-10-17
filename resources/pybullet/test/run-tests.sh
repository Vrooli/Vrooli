#!/bin/bash

# PyBullet Test Runner
# Main entry point for all test phases

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source test library
source "$RESOURCE_DIR/lib/test.sh"

# Parse test type
TEST_TYPE="${1:-all}"

# Run requested tests
handle_test "$TEST_TYPE"