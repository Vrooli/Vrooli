#!/bin/bash
# Validate scenario structure and required assets
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

required_dirs=(
  api
  cli
  docs
  initialization
  initialization/storage/postgres
  ui
  ui/sdk
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
  cli/comment-system
  cli/install.sh
  docs/integration.md
  initialization/storage/postgres/schema.sql
  ui/index.html
  ui/package.json
  ui/sdk/vrooli-comments.js
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if [[ ! -x cli/comment-system ]]; then
  testing::phase::add_warning "CLI binary cli/comment-system is not executable"
  testing::phase::add_test failed
fi

if [[ ! -x cli/install.sh ]]; then
  testing::phase::add_warning "CLI installer cli/install.sh is not executable"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Structure validation completed"
