#!/bin/bash
# Test: Unit tests
# Tests individual components and functions

set -euo pipefail

# Initialize testing phase
APP_ROOT="${APP_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

# Source centralized unit test runner
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Navigate to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

echo "  ✓ Running unit tests for image-tools..."

# Run all unit tests using centralized runner
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50 \
    --verbose

testing::phase::end_with_summary "Unit tests completed"
