#!/bin/bash

set -euo pipefail

# Integrate with centralized testing infrastructure
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Run all unit tests with centralized runner
testing::unit::run_all_tests \
    --go-dir "api" \
    --node-dir "ui" \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

testing::phase::end_with_summary "Unit tests completed"
