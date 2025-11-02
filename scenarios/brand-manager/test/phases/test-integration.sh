#!/bin/bash
# Exercises integration-level health checks against the running scenario.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"

if ! testing::connectivity::test_api "$SCENARIO_NAME"; then
  testing::phase::add_error "API connectivity test failed"
  testing::phase::add_test failed
else
  testing::phase::add_test passed
fi

testing_script="$TESTING_PHASE_SCENARIO_DIR/test/test-brand-generation.sh"
if [ -x "$testing_script" ]; then
  testing::phase::check "Brand generation workflow" bash -c 'cd "${TESTING_PHASE_SCENARIO_DIR}" && ./test/test-brand-generation.sh'
else
  testing::phase::add_warning "Workflow integration script missing; skipping end-to-end integration"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
