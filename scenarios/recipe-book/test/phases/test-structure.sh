#!/bin/bash
# Validate scenario structure and essential assets
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

required_dirs=(api ui cli test initialization)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  Makefile
  README.md
  PRD.md
  .vrooli/service.json
  api/go.mod
  api/main.go
  cli/install.sh
  test/run-tests.sh
  test/phases/test-integration.sh
  test/phases/test-business.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Ensure initialization assets exist so resources can be provisioned.
init_files=(
  initialization/storage/postgres/schema.sql
  initialization/storage/postgres/seed.sql
  initialization/storage/qdrant/collection.json
)
missing_init=0
for file in "${init_files[@]}"; do
  if [ -f "$file" ]; then
    log::info "📦 Found initialization asset: $file"
  else
    log::warning "⚠️  Missing initialization asset: $file"
    missing_init=1
  fi
done
if [ "$missing_init" -eq 0 ]; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Guard against lingering legacy testing artifacts.
if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml still present"
  testing::phase::add_test skipped
else
  testing::phase::add_test passed
fi

testing::phase::end_with_summary "Structure validation completed"
