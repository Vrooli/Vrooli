#!/bin/bash
# Validate scenario structure and critical assets
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Ensure required top-level directories exist
required_dirs=(api cli initialization test ui .vrooli)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Ensure critical files exist
required_files=(
  PRD.md
  README.md
  Makefile
  .vrooli/service.json
  test/run-tests.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Guard against legacy testing configuration
if testing::phase::check "Legacy scenario-test.yaml removed" bash -c '[[ ! -f scenario-test.yaml ]]'; then
  :
fi

testing::phase::end_with_summary "Structure validation completed"
