#!/bin/bash
# Validates project structure and required artifacts for the scenario-dependency-analyzer

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

required_dirs=(
  api
  cli
  initialization
  initialization/storage/postgres
  test
  ui
  .vrooli
)

required_files=(
  PRD.md
  README.md
  api/main.go
  api/go.mod
  cli/scenario-dependency-analyzer
  cli/install.sh
  initialization/storage/postgres/schema.sql
  initialization/storage/postgres/seed.sql
  .vrooli/service.json
)

if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Ensure CLI script is executable so subsequent phases can invoke it safely.
if [ -f "cli/scenario-dependency-analyzer" ]; then
  testing::phase::check "CLI wrapper is executable" test -x "cli/scenario-dependency-analyzer"
fi

testing::phase::end_with_summary "Structure validation completed"
