#!/bin/bash
# Validates core files and directories for the Email Outreach Manager scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

cd "$TESTING_PHASE_SCENARIO_DIR"

required_dirs=(
  "api"
  ".vrooli"
  "test"
  "test/phases"
)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  ".vrooli/service.json"
  "README.md"
  "PRD.md"
  "test/run-tests.sh"
  "test/phases/test-unit.sh"
  "api/main.go"
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if command -v jq >/dev/null 2>&1; then
  testing::phase::check "service.json parses" jq empty .vrooli/service.json
else
  testing::phase::add_warning "jq unavailable; skipping service.json validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
