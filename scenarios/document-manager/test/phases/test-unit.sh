#!/bin/bash
# Phase 3: Unit tests - <60 seconds
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Note: document-manager coverage is limited to ~36% without database
# Many handlers (getApplications, getAgents, getQueue, etc.) require database connection
# Full coverage requires integration tests with database
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 40 \
    --coverage-error 30

testing::phase::end_with_summary "Unit tests completed"
