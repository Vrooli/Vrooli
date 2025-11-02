#!/bin/bash
# Unit tests for news-aggregator-bias-analysis scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

# Source centralized unit test runner
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🧪 Running unit tests..."

if testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50; then
    testing::phase::add_test passed
else
    testing::phase::add_test failed
fi

testing::phase::end_with_summary "Unit tests completed"
