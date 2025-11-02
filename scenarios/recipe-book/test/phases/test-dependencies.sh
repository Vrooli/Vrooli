#!/bin/bash
# Verify language toolchains and critical service dependencies
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if command -v go >/dev/null 2>&1 && [ -f "api/go.mod" ]; then
  if testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'; then
    :
  fi
  if testing::phase::check "Go dependencies download" bash -c 'cd api && go mod download >/dev/null'; then
    :
  fi
else
  testing::phase::add_warning "Go toolchain or go.mod missing; skipping Go dependency checks"
  testing::phase::add_test skipped
fi

if command -v npm >/dev/null 2>&1 && [ -f "ui/package.json" ]; then
  if testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run --silent >/dev/null'; then
    :
  fi
else
  testing::phase::add_warning "npm or ui/package.json not available; skipping Node dependency checks"
  testing::phase::add_test skipped
fi

# Database connectivity is required for API flows but may depend on resources being up.
if command -v psql >/dev/null 2>&1; then
  db_host="${POSTGRES_HOST:-localhost}"
  db_port="${POSTGRES_PORT:-5432}"
  db_user="${POSTGRES_USER:-recipe_book}" # align with scenario defaults
  db_name="${POSTGRES_DB:-recipe_book}"
  testing::phase::check "Postgres responds (${db_host}:${db_port})" \
    bash -c 'PGPASSWORD="${POSTGRES_PASSWORD:-recipe_book}" psql -h "$1" -p "$2" -U "$3" -d "$4" -c "SELECT 1" >/dev/null' \
    _ "$db_host" "$db_port" "$db_user" "$db_name" || testing::phase::add_warning "Postgres unavailable; API integration tests may skip DB assertions"
else
  testing::phase::add_warning "psql not installed; cannot verify Postgres connectivity"
  testing::phase::add_test skipped
fi

# Optional service checks surface warnings without failing the phase.
if command -v curl >/dev/null 2>&1; then
  qdrant_host="${QDRANT_HOST:-localhost}"
  qdrant_port="${QDRANT_PORT:-6333}"
  if curl -sf "http://${qdrant_host}:${qdrant_port}/collections" >/dev/null; then
    log::info "✅ Qdrant reachable at ${qdrant_host}:${qdrant_port}"
  else
    testing::phase::add_warning "Qdrant not reachable; semantic search automation will be skipped"
  fi

  n8n_host="${N8N_HOST:-localhost}"
  n8n_port="${N8N_PORT:-5678}"
  if curl -sf "http://${n8n_host}:${n8n_port}/healthz" >/dev/null; then
    log::info "✅ n8n health endpoint responding"
  else
    testing::phase::add_warning "n8n workflow runner unavailable"
  fi
else
  testing::phase::add_warning "curl not installed; skipping optional resource checks"
fi

testing::phase::end_with_summary "Dependency validation completed"
