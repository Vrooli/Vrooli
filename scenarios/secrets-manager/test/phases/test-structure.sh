#!/bin/bash
# Validate scenario structure and required assets for Secrets Manager
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

required_dirs=(
  .vrooli
  api
  cli
  initialization
  initialization/storage/postgres
  initialization/automation
  initialization/automation/n8n
  ui
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
  Makefile
  api/main.go
  api/go.mod
  cli/install.sh
  cli/secrets-manager
  initialization/storage/postgres/schema.sql
  initialization/storage/postgres/seed.sql
  initialization/automation/n8n/resource-scanner.json
  initialization/automation/n8n/secret-validator.json
  ui/index.html
  ui/server.js
  custom-tests.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Warn if legacy scenario-test.yaml still exists elsewhere
if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml detected; phased tests should be authoritative"
fi

testing::phase::end_with_summary "Structure validation completed"
