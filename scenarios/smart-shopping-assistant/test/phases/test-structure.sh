#!/bin/bash
# Validates required files and directory layout for smart-shopping-assistant.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

REQUIRED_FILES=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
  "api/main.go"
  "api/go.mod"
  "cli/smart-shopping-assistant"
  "cli/install.sh"
  "initialization/storage/postgres/schema.sql"
  "initialization/storage/postgres/seed.sql"
  "ui/package.json"
  "ui/src/main.jsx"
  "ui/src/App.jsx"
)

REQUIRED_DIRS=(
  "api"
  "cli"
  "initialization/storage/postgres"
  "ui/src"
  "tests"
)

if testing::phase::check_files "${REQUIRED_FILES[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if testing::phase::check_directories "${REQUIRED_DIRS[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Validate service.json structure when jq is present.
testing::phase::add_test "service.json schema validation"
if command -v jq >/dev/null 2>&1 && [[ -f ".vrooli/service.json" ]]; then
  if jq -e '.service.name and .ports.api' .vrooli/service.json >/dev/null; then
    log::success "service.json contains required keys"
  else
    testing::phase::add_error "service.json missing required keys"
  fi
else
  testing::phase::add_warning "Skipping service.json structure check (jq unavailable)"
fi

# Ensure Go source set is non-empty.
testing::phase::check "API Go sources present" bash -c 'shopt -s nullglob; files=(api/*.go); (( ${#files[@]} > 0 ))'

# Confirm database initialization assets exist.
testing::phase::check "PostgreSQL schema present" test -f "initialization/storage/postgres/schema.sql"
testing::phase::check "PostgreSQL seed present" test -f "initialization/storage/postgres/seed.sql"

# Optional CLI binary/script check.
if [[ -d cli ]]; then
  if testing::phase::check "CLI install script present" test -f "cli/install.sh"; then
    :
  fi
fi

testing::phase::end_with_summary "Structure validation completed"
