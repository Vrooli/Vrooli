#!/bin/bash
# Validate language/runtime dependencies and schema contracts
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

# Go module health
if [ -f "api/go.mod" ]; then
  testing::phase::check "Go modules resolve" \
    bash -c 'cd api && go list ./... >/dev/null'

  if command -v govulncheck >/dev/null 2>&1; then
    if testing::phase::check "govulncheck passes" bash -c 'cd api && govulncheck ./... >/dev/null'; then
      :
    else
      testing::phase::add_warning "govulncheck reported issues; review above output"
    fi
  else
    testing::phase::add_warning "govulncheck not installed; skipping vulnerability scan"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "api/go.mod missing; skipping Go dependency checks"
  testing::phase::add_test skipped
fi

# Node dependency health (UI)
if [ -f "ui/package.json" ]; then
  if command -v npm >/dev/null 2>&1; then
    testing::phase::check "npm install --dry-run" \
      bash -c 'cd ui && npm install --dry-run >/dev/null'
  else
    testing::phase::add_warning "npm CLI not available; skipping Node dependency verification"
    testing::phase::add_test skipped
  fi

  if [ ! -f "ui/package-lock.json" ] && [ ! -f "ui/pnpm-lock.yaml" ]; then
    testing::phase::add_warning "UI lockfile missing; enable deterministic installs"
  fi
else
  testing::phase::add_warning "ui/package.json missing; skipping UI dependency checks"
  testing::phase::add_test skipped
fi

# Schema sanity for Postgres
tables=(tokens wallets balances transactions achievements exchange_rates)
if [ -f "initialization/storage/postgres/schema.sql" ]; then
  for table in "${tables[@]}"; do
    testing::phase::check "Schema defines table ${table}" \
      bash -c "grep -qi 'CREATE TABLE[^;]*${table}' initialization/storage/postgres/schema.sql"
  done
else
  testing::phase::add_error "Database schema file missing"
  testing::phase::add_test failed
fi

# Redis optional dependency
if command -v redis-cli >/dev/null 2>&1; then
  if testing::phase::check "redis-cli ping" redis-cli ping; then
    :
  else
    testing::phase::add_warning "Redis ping failed (optional dependency)"
  fi
else
  testing::phase::add_warning "redis-cli unavailable; skipping Redis check"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
