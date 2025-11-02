#!/bin/bash
# Validate required files and directories for accessibility-compliance-hub.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(api cli initialization test)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  Makefile
  PRD.md
  README.md
  PROBLEMS.md
  api/main.go
  api/go.mod
  cli/install.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if command -v jq >/dev/null 2>&1; then
  testing::phase::check "service.json is valid JSON" jq empty .vrooli/service.json || true
else
  testing::phase::add_warning "jq not available; skipping service.json validation"
  testing::phase::add_test skipped
fi

if [ -f "cli/accessibility-compliance-hub" ]; then
  if [ -x "cli/accessibility-compliance-hub" ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_warning "CLI binary exists but is not executable; run cli/install.sh"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "CLI binary missing; run cli/install.sh to generate it"
  testing::phase::add_test skipped
fi

if [ -d ui ]; then
  if [ -f "ui/package.json" ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_warning "UI directory detected without package.json; treating as static prototype"
    testing::phase::add_test skipped
  fi
fi

testing::phase::end_with_summary "Structure validation completed"
