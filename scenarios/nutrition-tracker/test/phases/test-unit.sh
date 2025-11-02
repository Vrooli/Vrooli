#!/bin/bash
# Run unit tests across supported languages for nutrition-tracker

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

previous_goflags="${GOFLAGS-}"
export GOFLAGS="${GOFLAGS:+$GOFLAGS }-tags=testing"

if testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 25 \
    --coverage-error 10 \
    --scenario "nutrition-tracker"; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Unit tests failed"
  testing::phase::add_test failed
fi

if [ -z "${previous_goflags+x}" ] || [ -z "$previous_goflags" ]; then
  unset GOFLAGS
else
  export GOFLAGS="$previous_goflags"
fi

testing::phase::end_with_summary "Unit tests completed"
