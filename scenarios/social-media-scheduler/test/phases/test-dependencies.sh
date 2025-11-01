#!/bin/bash
# Ensure language runtimes and critical tooling are available

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

# Go toolchain
if command -v go >/dev/null 2>&1; then
  if testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'; then
    :
  fi
else
  testing::phase::add_warning "Go toolchain missing; skipping Go dependency verification"
  testing::phase::add_test skipped
fi

# Node toolchain
if [ -f "ui/package.json" ]; then
  if command -v npm >/dev/null 2>&1; then
    if testing::phase::check "npm dependency graph" bash -c 'cd ui && npm install --ignore-scripts --dry-run >/dev/null'; then
      :
    fi
  else
    testing::phase::add_warning "npm CLI not available; cannot verify UI dependencies"
    testing::phase::add_test skipped
  fi
fi

# PostgreSQL client
if command -v psql >/dev/null 2>&1; then
  testing::phase::check "psql client available" psql --version >/dev/null
else
  testing::phase::add_warning "psql client not found; database setup scripts may fail"
  testing::phase::add_test skipped
fi

# Redis client
if command -v redis-cli >/dev/null 2>&1; then
  testing::phase::check "redis-cli available" redis-cli --version >/dev/null
else
  testing::phase::add_warning "redis-cli not installed; queue diagnostics will be limited"
  testing::phase::add_test skipped
fi

# MinIO client (optional)
if [ -n "${MC_REQUIRED:-}" ] || command -v mc >/dev/null 2>&1; then
  if command -v mc >/dev/null 2>&1; then
    testing::phase::check "mc client available" mc --version >/dev/null
  else
    testing::phase::add_warning "MinIO client (mc) not found; skip storage smoke checks"
    testing::phase::add_test skipped
  fi
fi

testing::phase::end_with_summary "Dependency validation completed"
