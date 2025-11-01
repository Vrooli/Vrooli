#!/bin/bash
# Validates required files and directories for the symbol-search scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(
  api
  cli
  data
  initialization
  initialization/postgres
  test
  ui
  ui/src
  ui/src/components
  ui/src/hooks
)

if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  PRD.md
  README.md
  api/main.go
  api/go.mod
  cli/install.sh
  cli/symbol-search
  initialization/postgres/schema.sql
  initialization/postgres/seed.sql
  ui/package.json
  ui/vite.config.js
  ui/src/main.jsx
  test/phases/test-unit.sh
  test/phases/test-integration.sh
  test/phases/test-performance.sh
)

if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Warn if legacy files remain
if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml detected – migrate references to phased testing and remove the file"
  testing::phase::add_test skipped
fi

# Basic README contract
if testing::phase::check "README documents testing command" grep -q "test/run-tests.sh" README.md; then
  :
else
  testing::phase::add_warning "README does not mention phased test runner"
fi

# Summarise results
testing::phase::end_with_summary "Structure validation completed"
