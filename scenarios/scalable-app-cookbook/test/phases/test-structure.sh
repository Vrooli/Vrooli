#!/bin/bash
# Validates required files and directories for the scenario shell.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(
  api
  cli
  ui
  ui/src
  initialization
  initialization/postgres
  docs
  tests
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
  cli/scalable-app-cookbook
  cli/install.sh
  initialization/postgres/schema.sql
  initialization/postgres/seed.sql
  ui/package.json
  ui/src/App.jsx
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Structure validation completed"
