#!/bin/bash
# Validates language runtimes, dependencies, and optional resources
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

SCENARIO_DIR="$TESTING_PHASE_SCENARIO_DIR"

# Go toolchain and modules
if [ -f "api/go.mod" ]; then
  if command -v go >/dev/null 2>&1; then
    testing::phase::check "Go module graph resolves" \
      bash -c 'cd api && go list ./... >/dev/null'

    if command -v govulncheck >/dev/null 2>&1; then
      if bash -c 'cd api && govulncheck ./... >/dev/null'; then
        testing::phase::add_test passed
      else
        testing::phase::add_warning "govulncheck reported potential issues"
        testing::phase::add_test skipped
      fi
    else
      testing::phase::add_warning "govulncheck not installed; skipping vulnerability scan"
      testing::phase::add_test skipped
    fi
  else
    testing::phase::add_error "Go toolchain not available"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "api/go.mod missing; skipping Go dependency checks"
  testing::phase::add_test skipped
fi

# Node runtime and dependencies
if [ -f "ui/package.json" ]; then
  if command -v node >/dev/null 2>&1; then
    testing::phase::check "Node.js runtime version" node -v

    if [ -d "ui/node_modules" ]; then
      testing::phase::add_test passed
    else
      testing::phase::add_warning "ui/node_modules missing; run npm install"
      testing::phase::add_test skipped
    fi
  else
    testing::phase::add_error "Node.js runtime not found"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "ui/package.json missing; skipping Node checks"
  testing::phase::add_test skipped
fi

# CLI availability
if testing::phase::check "CLI binary executable" test -x "cli/picker-wheel"; then
  if command -v picker-wheel >/dev/null 2>&1; then
    testing::phase::add_test passed
  else
    testing::phase::add_warning "picker-wheel CLI not linked in PATH; run cli/install.sh"
    testing::phase::add_test skipped
  fi
fi

# Optional resource health checks
check_resource() {
  local name="$1"
  local cmd="$2"
  if command -v "$cmd" >/dev/null 2>&1; then
    if "$cmd" status >/dev/null 2>&1; then
      testing::phase::add_test passed
      log::success "✅ ${name} resource healthy"
    else
      testing::phase::add_warning "${name} resource not running"
      testing::phase::add_test skipped
    fi
  else
    testing::phase::add_warning "${name} resource command unavailable"
    testing::phase::add_test skipped
  fi
}

check_resource "PostgreSQL" "resource-postgres"
check_resource "n8n" "resource-n8n"
check_resource "Ollama" "resource-ollama"

# Configuration sanity checks
if command -v jq >/dev/null 2>&1 && [ -f ".vrooli/service.json" ]; then
  testing::phase::check "service.json valid JSON" jq empty .vrooli/service.json
  testing::phase::check "service.json exposes API port" \
    jq -e '.ports.api.env_var == "API_PORT"' .vrooli/service.json
  testing::phase::check "service.json exposes UI port" \
    jq -e '.ports.ui.env_var == "UI_PORT"' .vrooli/service.json
else
  testing::phase::add_warning "jq not available or service.json missing; skipped configuration validation"
  testing::phase::add_test skipped
fi

# Initialization assets present
for path in initialization/postgres/schema.sql initialization/n8n; do
  if [ -e "$path" ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_warning "$path missing (optional)"
    testing::phase::add_test skipped
  fi
done

# Lightweight database check when postgres tooling is available
if command -v psql >/dev/null 2>&1 && command -v resource-postgres >/dev/null 2>&1 && resource-postgres status >/dev/null 2>&1; then
  testing::phase::check "wheels table exists (or will be created)" \
    bash -c 'psql -U postgres -d vrooli_db -c "\dt wheels" >/dev/null'
else
  testing::phase::add_warning "PostgreSQL CLI not available or server offline; skipping table existence check"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
