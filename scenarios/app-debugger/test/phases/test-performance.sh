#!/bin/bash
# Lightweight performance baseline checks for app-debugger

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_URL=""

if API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  testing::phase::check "API responds within 2s" curl --max-time 2 -sf "${API_URL}/health"
else
  testing::phase::add_error "Unable to determine API URL for ${SCENARIO_NAME}"
fi

if [ -d "api" ]; then
  testing::phase::check "Go benchmark smoke" bash -c 'cd api && go test -run=^$ -bench=. -count=1 ./... >/dev/null'
else
  testing::phase::add_warning "API directory missing; skipping Go benchmark"
  testing::phase::add_test skipped
fi


testing::phase::end_with_summary "Performance checks completed"
