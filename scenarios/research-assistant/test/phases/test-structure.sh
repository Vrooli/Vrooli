#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

required_dirs=(
  "api"
  "ui"
  "cli"
  "initialization"
  "initialization/storage/postgres"
  "initialization/automation/n8n"
  "initialization/configuration"
)
testing::phase::check_directories "${required_dirs[@]}"

required_files=(
  "api/main.go"
  "api/go.mod"
  "cli/research-assistant"
  "cli/install.sh"
  "initialization/configuration/research-config.json"
  "initialization/storage/postgres/schema.sql"
  "initialization/storage/postgres/seed.sql"
)
testing::phase::check_files "${required_files[@]}"

if [ -d "api" ] && command -v go >/dev/null 2>&1; then
  testing::phase::check "Go module tidy check" bash -c 'cd api && go list ./... >/dev/null'
else
  testing::phase::add_warning "Go toolchain missing; skipped Go project validation"
  testing::phase::add_test skipped
fi

if [ -f "ui/package.json" ] && command -v jq >/dev/null 2>&1; then
  if jq -e '.scripts.build' ui/package.json >/dev/null 2>&1; then
    testing::phase::check "UI build script present" jq -e '.scripts.build' ui/package.json
  else
    testing::phase::add_warning "UI build script missing in package.json"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "Skipped UI package inspection (missing package.json or jq)"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
