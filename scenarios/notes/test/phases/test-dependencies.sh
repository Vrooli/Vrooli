#!/usr/bin/env bash
# Dependency validation for SmartNotes scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

require_resource() {
  local resource=$1
  testing::phase::check "Resource available: ${resource}" vrooli resource status "$resource"
}

require_command() {
  local cmd=$1
  testing::phase::check "Command available: ${cmd}" command -v "$cmd"
}

# Critical runtime resources
require_resource postgres || true
require_resource qdrant || true
require_resource ollama || true
require_resource n8n || true

# Optional redis (warn only)
if vrooli resource status redis >/dev/null 2>&1; then
  testing::phase::add_test passed
  log::success "Redis cache available"
else
  testing::phase::add_warning "Redis cache not installed (optional)"
  testing::phase::add_test skipped
fi

# Tooling requirements
require_command go || true
require_command node || true
require_command npm || true
require_command jq || true

# Go modules resolve
if [ -f api/go.mod ]; then
  if testing::phase::check "Go modules verify" bash -c 'cd api && go mod verify >/dev/null'; then
    :
  else
    :
  fi
else
  testing::phase::add_error "api/go.mod missing"
  testing::phase::add_test failed
fi

# Node modules present or installable
if [ -f ui/package.json ]; then
  if [ -d ui/node_modules ]; then
    testing::phase::add_test passed
    log::success "Node modules already installed"
  else
    testing::phase::add_warning "ui/node_modules missing; run npm install"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_error "ui/package.json missing"
  testing::phase::add_test failed
fi

# Ensure lifecycle configuration includes test hook
if jq -e '.lifecycle.test.steps[]? | select(.run == "./test/run-tests.sh")' .vrooli/service.json >/dev/null 2>&1; then
  testing::phase::add_test passed
else
  testing::phase::add_warning "Lifecycle test step not configured"
  testing::phase::add_test skipped
fi


testing::phase::end_with_summary "Dependency validation completed"
