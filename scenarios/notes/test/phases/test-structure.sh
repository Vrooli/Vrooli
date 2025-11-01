#!/usr/bin/env bash
# Structural validation for SmartNotes scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

REQUIRED_DIRS=(
  api
  ui
  cli
  initialization
  test
)

REQUIRED_FILES=(
  PRD.md
  README.md
  Makefile
  .vrooli/service.json
  api/main.go
  api/go.mod
  ui/package.json
  ui/server.js
  cli/notes
  cli/install.sh
  initialization/postgres/schema.sql
  initialization/qdrant/collections.json
)

if testing::phase::check_directories "${REQUIRED_DIRS[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if testing::phase::check_files "${REQUIRED_FILES[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if command -v jq >/dev/null 2>&1; then
  if testing::phase::check "service.json is valid JSON" jq empty .vrooli/service.json; then
    :
  else
    :
  fi
else
  testing::phase::add_warning "jq not installed; JSON validation skipped"
  testing::phase::add_test skipped
fi

# Ensure CLI documentation exists (optional but useful)
if [ -f cli/README.md ]; then
  testing::phase::add_test passed
else
  testing::phase::add_warning "CLI README missing"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
