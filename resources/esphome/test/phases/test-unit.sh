#!/usr/bin/env bash
################################################################################
# ESPHome Unit Test Phase
################################################################################

set -euo pipefail

# Get paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Run unit tests via the main test runner
"$TEST_DIR/run-tests.sh" unit