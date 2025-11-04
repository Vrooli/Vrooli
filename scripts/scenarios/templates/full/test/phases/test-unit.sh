#!/bin/bash
# Orchestrates Go/Node/Python unit tests with coverage thresholds.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

testing::phase::init --target-time "120s"
cd "$TESTING_PHASE_SCENARIO_DIR"

# Add requirement IDs here when you need coarse-grained coverage (e.g. when all
# unit tests collectively prove a requirement). Individual tests can also tag
# themselves with `REQ:<ID>` (for example, `t.Run("renders workflow [REQ:SAMPLE-FUNC-001]", ...)` or
# `it('handles response [REQ:SAMPLE-FUNC-001]', () => { ... })`). The unit test
# runner automatically records those tags and updates requirement coverage.
UNIT_REQUIREMENTS=()

if testing::unit::run_all_tests \
  --go-dir "api" \
  --node-dir "ui" \
  --python-dir "." \
  --coverage-warn 80 \
  --coverage-error 70; then
  testing::phase::add_test passed
  for requirement_id in "${UNIT_REQUIREMENTS[@]}"; do
    testing::phase::add_requirement --id "$requirement_id" --status passed --evidence "Unit test suites"
  done
else
  testing::phase::add_error "Unit test runner reported failures"
  testing::phase::add_test failed
  for requirement_id in "${UNIT_REQUIREMENTS[@]}"; do
    testing::phase::add_requirement --id "$requirement_id" --status failed --evidence "Unit test suites"
  done
fi

# Update UNIT_REQUIREMENTS once the requirements registry is defined, or rely on
# per-test `REQ:` tags for finer-grained tracking.

testing::phase::end_with_summary "Unit tests completed"
