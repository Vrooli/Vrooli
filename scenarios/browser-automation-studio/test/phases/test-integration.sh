#!/bin/bash
# Runs Browser Automation Studio workflow automations from requirements registry.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

# Run BAS workflow automation validations discovered from requirements/*.json
# Runtime is auto-managed by testing::phase::init with --require-runtime flag
if ! testing::phase::run_bas_automation_validations --scenario "$TESTING_PHASE_SCENARIO_NAME" --manage-runtime skip; then
  bas_rc=$?
  if [ "$bas_rc" -ne 0 ] && [ "$bas_rc" -ne 200 ]; then
    testing::phase::add_error "Browser Automation Studio workflow validations failed"
  fi
fi

testing::phase::end_with_summary "Integration tests completed"
