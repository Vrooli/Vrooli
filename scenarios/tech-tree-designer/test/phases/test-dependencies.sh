#!/bin/bash
# Verifies toolchains and module dependencies for tech-tree-designer
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"
cd "${TESTING_PHASE_SCENARIO_DIR}"

testing::phase::check "Go toolchain available" bash -c 'command -v go >/dev/null'

testing::phase::check "Node.js available" bash -c 'command -v node >/dev/null'

testing::phase::check "npm or pnpm present" bash -c 'command -v pnpm >/dev/null || command -v npm >/dev/null'

testing::phase::check "jq available" bash -c 'command -v jq >/dev/null'

if [ -f api/go.mod ]; then
  testing::phase::check "Go module graph verifies" bash -c 'cd api && go mod verify'
else
  testing::phase::add_warning "api/go.mod missing; skipping Go dependency verification"
  testing::phase::add_test skipped
fi

if [ -f ui/package.json ]; then
  testing::phase::check "UI manifest parses" bash -c 'cd ui && jq -r ".name" package.json >/dev/null'
  if [ ! -d ui/node_modules ] && [ ! -f ui/pnpm-lock.yaml ] && [ ! -f ui/package-lock.json ]; then
    testing::phase::add_warning "UI dependencies not installed (run pnpm|npm install)"
    testing::phase::add_test skipped
  fi
fi

if command -v vrooli >/dev/null 2>&1; then
  if ! vrooli resource status postgres --json >/dev/null 2>&1; then
    testing::phase::add_warning "Postgres resource unavailable; integration tests may need it"
  fi
  if ! vrooli resource status ollama --json >/dev/null 2>&1; then
    testing::phase::add_warning "Ollama resource unavailable; AI features may be degraded"
  fi
else
  testing::phase::add_warning "vrooli CLI not on PATH; skipping resource health checks"
fi

testing::phase::end_with_summary "Dependency validation completed"
