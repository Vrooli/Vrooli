#!/bin/bash
# Validate scenario structure and required assets
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_files=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
  "api/main.go"
  "api/go.mod"
  "cli/install.sh"
  "ui/index.html"
  "ui/package.json"
  "ui/server.js"
  "initialization/storage/postgres/schema.sql"
)

required_dirs=(
  "api"
  "cli"
  "ui"
  "data"
  "docs"
  "initialization/storage/postgres"
  "test/phases"
)

if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Missing required files"
  testing::phase::add_test failed
fi

if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Missing required directories"
  testing::phase::add_test failed
fi

# Ensure main.go has proper entrypoint semantics
testing::phase::check "api/main.go declares package main" \
  bash -c 'grep -q "^package main" api/main.go'

testing::phase::check "api/main.go defines main()" \
  bash -c 'grep -q "func main()" api/main.go'

# Scenario-specific sanity: confirm wallet lifecycle documentation exists
if [ -f "docs/integration.md" ]; then
  testing::phase::check "docs/integration.md references wallet lifecycle" \
    bash -c 'grep -iq "wallet" docs/integration.md'
else
  testing::phase::add_warning "docs/integration.md missing; skipping wallet lifecycle doc check"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
