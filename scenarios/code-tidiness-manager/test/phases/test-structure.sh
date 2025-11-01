#!/bin/bash
# Validate key files and directories exist for code-tidiness-manager

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(
  api
  cli
  ui
  lib
  docs
  initialization
  initialization/automation
  initialization/automation/n8n
  initialization/storage
  initialization/storage/postgres
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
  initialization/storage/postgres/schema.sql
  initialization/storage/postgres/seed.sql
  initialization/automation/n8n/code-scanner.json
  initialization/automation/n8n/cleanup-executor.json
  initialization/automation/n8n/pattern-analyzer.json
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if [ -f cli/install.sh ] && [ ! -x cli/install.sh ]; then
  testing::phase::add_warning "cli/install.sh is not executable"
fi

testing::phase::end_with_summary "Structure validation completed"
