#!/bin/bash
# Dependency validation for travel-map-filler.
# Verifies language toolchains and external services are reachable.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "🔗 Validating build toolchains and dependencies"

# --- Go toolchain ---
if [ -f "api/go.mod" ]; then
  if command -v go >/dev/null 2>&1; then
    if (cd api && go mod verify >/dev/null); then
      log::success "✅ Go modules verified"
      testing::phase::add_test passed
    else
      log::error "❌ go mod verify failed"
      testing::phase::add_error "Go module verification failed"
      testing::phase::add_test failed
    fi

    if (cd api && go build -o /tmp/travel-map-filler-build ./... >/dev/null); then
      log::success "✅ Go sources compile"
      testing::phase::add_test passed
    else
      log::error "❌ go build failed"
      testing::phase::add_error "Go sources failed to compile"
      testing::phase::add_test failed
    fi
    rm -f /tmp/travel-map-filler-build 2>/dev/null || true
  else
    log::warning "⚠️  Go toolchain not found; skipping Go dependency checks"
    testing::phase::add_warning "Go toolchain unavailable"
    testing::phase::add_test skipped
  fi
else
  log::warning "ℹ️  No api/go.mod present; skipping Go dependency checks"
  testing::phase::add_test skipped
fi

# --- Node toolchain ---
if [ -f "ui/package.json" ]; then
  if command -v npm >/dev/null 2>&1; then
    if (cd ui && npm install --package-lock-only --dry-run >/dev/null 2>&1); then
      log::success "✅ npm dependency resolution succeeded"
      testing::phase::add_test passed
    else
      log::error "❌ npm install --dry-run failed"
      testing::phase::add_error "Node dependency install failed"
      testing::phase::add_test failed
    fi
  else
    log::warning "⚠️  npm not available; skipping Node dependency checks"
    testing::phase::add_warning "npm unavailable"
    testing::phase::add_test skipped
  fi
else
  log::warning "ℹ️  No ui/package.json found; skipping Node dependency checks"
  testing::phase::add_test skipped
fi

# --- Postgres connectivity (optional: relies on env variables) ---
if command -v pg_isready >/dev/null 2>&1 && [ -n "${POSTGRES_HOST:-}" ] && [ -n "${POSTGRES_PORT:-}" ]; then
  if pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "${POSTGRES_USER:-postgres}" >/dev/null 2>&1; then
    log::success "✅ Postgres reachable at $POSTGRES_HOST:$POSTGRES_PORT"
    testing::phase::add_test passed
  else
    log::warning "⚠️  Postgres not reachable (may require runtime)"
    testing::phase::add_warning "Postgres connectivity check failed"
    testing::phase::add_test skipped
  fi
else
  log::info "ℹ️  Skipping Postgres connectivity check (pg_isready or env not available)"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
