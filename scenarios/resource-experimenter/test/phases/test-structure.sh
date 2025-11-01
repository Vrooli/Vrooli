#!/bin/bash
# Validates foundational files and directories for resource-experimenter
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(api ui cli initialization test .vrooli)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  Makefile
  README.md
  PRD.md
  api/main.go
  api/go.mod
  test/run-tests.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if command -v jq >/dev/null 2>&1; then
  testing::phase::check "service.json is valid JSON" jq empty .vrooli/service.json
  testing::phase::check "service.json has lifecycle" jq -e '.lifecycle' .vrooli/service.json
else
  testing::phase::add_warning "jq not installed; skipping service.json validation"
  testing::phase::add_test skipped
fi

if [ -d "test/phases" ]; then
  testing::phase::check "unit phase exists" test -f test/phases/test-unit.sh
  testing::phase::check "integration phase exists" test -f test/phases/test-integration.sh
else
  testing::phase::add_error "test/phases directory missing"
fi

testing::phase::end_with_summary "Structure validation completed"
