#!/bin/bash
# Validates critical files and configuration for picker-wheel
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

required_dirs=(api ui cli initialization)
required_files=(
  .vrooli/service.json
  PRD.md
  README.md
  Makefile
  api/main.go
  api/go.mod
  ui/package.json
  ui/server.js
  initialization/postgres/schema.sql
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

if command -v jq >/dev/null 2>&1; then
  testing::phase::check "service.json declares picker-wheel" \
    jq -e '.service.name == "picker-wheel"' .vrooli/service.json
  testing::phase::check "Lifecycle config uses v2.0.0" \
    jq -e '.lifecycle.version == "2.0.0"' .vrooli/service.json
else
  testing::phase::add_warning "jq unavailable; skipped JSON validation"
  testing::phase::add_test skipped
fi

if command -v grep >/dev/null 2>&1; then
  for target in run test stop status logs help; do
    testing::phase::check "Makefile target: $target" grep -q "^${target}:" Makefile
  done
else
  testing::phase::add_warning "grep unavailable; skipped Makefile target checks"
  testing::phase::add_test skipped
fi

if command -v grep >/dev/null 2>&1; then
  testing::phase::check "Schema defines wheels table" \
    grep -q "CREATE TABLE IF NOT EXISTS wheels" initialization/postgres/schema.sql
else
  testing::phase::add_warning "grep unavailable; skipped schema check"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
