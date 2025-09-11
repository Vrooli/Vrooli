#!/usr/bin/env bash
################################################################################
# ESPHome Integration Test Phase
################################################################################

set -euo pipefail

# Get paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Run integration tests via the main test runner
"$TEST_DIR/run-tests.sh" integration