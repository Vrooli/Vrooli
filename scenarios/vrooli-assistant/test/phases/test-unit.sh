#!/bin/bash
set -euo pipefail

# Get root directory (test/phases -> test -> scenario -> scenarios -> root)
APP_ROOT="${APP_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "60s"

# Source centralized testing library
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Navigate to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

echo "=== Running Unit Tests for Vrooli Assistant ==="

# Run all unit tests using centralized runner
# Note: Coverage thresholds are lowered because most functionality requires database
# Without database: ~18% coverage (model tests, health checks)
# With database: Expected ~72% coverage (full handler tests)
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 15 \
    --verbose

# End test phase with summary
testing::phase::end_with_summary "Unit tests completed"
