#!/bin/bash
# Validate required project files and directories

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

required_dirs=(
  .vrooli
  api
  cli
  docs
  initialization
  test
  "test/phases"
  ui
)
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
  api/main.go
  cli/install.sh
  test/run-tests.sh
  ui/package.json
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

for script in test/run-tests.sh cli/install.sh; do
  if testing::phase::check "script executable: ${script}" test -x "$script"; then
    :
  fi
done

if command -v jq >/dev/null 2>&1; then
  if testing::phase::check "service.json validates" jq . .vrooli/service.json >/dev/null; then
    :
  fi

  for key in lifecycle service ports; do
    if testing::phase::check "service.json has key: ${key}" jq -e ".${key}" .vrooli/service.json >/dev/null; then
      :
    fi
  done
else
  testing::phase::add_warning "jq not available; skipping service.json validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
