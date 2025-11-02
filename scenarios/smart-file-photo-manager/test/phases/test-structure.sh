#!/bin/bash
# Structural validation for smart-file-photo-manager

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(api ui cli initialization)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  api/main.go
  api/go.mod
  cli/install.sh
  cli/smart-file-photo-manager
  initialization/storage/postgres/schema.sql
  initialization/storage/postgres/seed.sql
  test/phases/test-unit.sh
  test/run-tests.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Warn if legacy scenario-test.yaml lingers
if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml detected; phased testing should be canonical"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
