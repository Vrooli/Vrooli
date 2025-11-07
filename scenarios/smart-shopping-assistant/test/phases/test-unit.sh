#!/bin/bash
# Orchestrates Go/Node/Python unit tests with coverage thresholds.
# Configuration is read from .vrooli/testing.json
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/config.sh"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

testing::phase::init --target-time "120s"
cd "$TESTING_PHASE_SCENARIO_DIR"

# Read unit test configuration from .vrooli/testing.json
# Configuration specifies which languages to test, coverage thresholds, and options
mapfile -t UNIT_ARGS < <(testing::config::get_unit_test_args)

if [ ${#UNIT_ARGS[@]} -eq 0 ]; then
  testing::phase::add_error "Failed to load unit test configuration from .vrooli/testing.json"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Unit test configuration error"
fi

# Run unit tests with config-driven arguments
# Individual tests can tag themselves with [REQ:ID] (e.g., `t.Run("test [REQ:SAMPLE-001]", ...)`)
# for automatic requirement tracking via the unit test runner
if testing::unit::run_all_tests "${UNIT_ARGS[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Unit test runner reported failures"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Unit tests completed"
