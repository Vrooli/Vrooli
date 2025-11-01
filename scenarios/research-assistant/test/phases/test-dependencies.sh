#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

if [ -f "api/go.mod" ] && command -v go >/dev/null 2>&1; then
  testing::phase::check "Go modules resolve" bash -c 'cd api && go list ./... >/dev/null'
  testing::phase::check "Go vendor/security scan" bash -c 'cd api && go vet ./... >/dev/null'
else
  testing::phase::add_warning "Skipped Go dependency checks (missing go.mod or Go toolchain)"
  testing::phase::add_test skipped
fi

if [ -f "ui/package.json" ]; then
  if command -v npm >/dev/null 2>&1; then
    testing::phase::check "UI dependency install dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
  else
    testing::phase::add_warning "npm CLI missing; UI dependency check skipped"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "UI package.json missing; skipping Node dependency validation"
  testing::phase::add_test skipped
fi

if [ -f "initialization/storage/postgres/schema.sql" ]; then
  if command -v psql >/dev/null 2>&1; then
    testing::phase::check "Postgres CLI available" psql --version
  else
    testing::phase::add_warning "psql not available; unable to validate Postgres schema"
    testing::phase::add_test skipped
  fi
fi

testing::phase::end_with_summary "Dependency validation completed"
