#!/bin/bash
set -euo pipefail

# Recipe Book Unit Test Runner
# Integrates with Vrooli's centralized testing infrastructure

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

unit_args=(--go-dir "api" --node-dir "ui" --skip-python --coverage-warn 80 --coverage-error 0)
if ! find api -name '*_test.go' -type f | grep -q .; then
  unit_args+=(--skip-go)
  testing::phase::add_warning "No Go unit tests detected; skipping Go coverage enforcement"
fi

if testing::unit::run_all_tests "${unit_args[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Unit tests completed"
