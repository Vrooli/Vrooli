#!/bin/bash
# Unit test phase leveraging centralized Vrooli infrastructure

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "150s"

source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

PREV_GOFLAGS="${GOFLAGS:-}"
if [ -n "$PREV_GOFLAGS" ]; then
  export GOFLAGS="$PREV_GOFLAGS -tags=testing"
else
  export GOFLAGS="-tags=testing"
fi

testing::unit::run_all_tests \
  --go-dir "api" \
  --skip-node \
  --skip-python \
  --coverage-warn 80 \
  --coverage-error 50

if [ -n "$PREV_GOFLAGS" ]; then
  export GOFLAGS="$PREV_GOFLAGS"
else
  unset GOFLAGS
fi

testing::phase::end_with_summary "Unit tests completed"
