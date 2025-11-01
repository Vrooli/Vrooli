#!/bin/bash
# Validates core project layout and required files.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

required_dirs=(api ui cli initialization test)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  README.md
  Makefile
  test/run-tests.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Ensure phase scripts exist for the standard lineup.
missing_phases=()
for phase in structure dependencies unit integration business performance; do
  if [ ! -f "test/phases/test-${phase}.sh" ]; then
    missing_phases+=("test/phases/test-${phase}.sh")
  fi
done

if [ ${#missing_phases[@]} -eq 0 ]; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Missing phase scripts: ${missing_phases[*]}"
  testing::phase::add_test failed
fi

# Check that the shared runner is executable.
if [ -x "test/run-tests.sh" ]; then
  testing::phase::add_test passed
else
  testing::phase::add_error "test/run-tests.sh is not executable"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Structure validation completed"
