#!/bin/bash
# Unit tests for video-tools scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "📊 Running video-tools unit tests..."

# Run Go tests with coverage
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

testing::phase::end_with_summary "Unit tests completed"
