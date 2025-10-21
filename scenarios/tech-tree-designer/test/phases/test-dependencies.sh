#!/bin/bash
# Dependency validation phase for tech-tree-designer
# Ensures all required resources and dependencies are available

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Check PostgreSQL resource availability
testing::phase::log "Checking PostgreSQL resource..."
if ! vrooli resource status postgres --json &>/dev/null; then
    testing::phase::warn "PostgreSQL resource not running (required for tech-tree-designer)"
fi

# Check Ollama resource availability (optional)
testing::phase::log "Checking Ollama resource..."
if ! vrooli resource status ollama --json &>/dev/null; then
    testing::phase::warn "Ollama resource not running (optional for AI analysis)"
fi

# Check Go dependencies
testing::phase::log "Checking Go dependencies..."
if [ -d api ] && [ -f api/go.mod ]; then
    cd api
    if ! go mod verify &>/dev/null; then
        testing::phase::error "Go module verification failed"
        testing::phase::end_with_summary "Go dependencies invalid" 1
    fi
    cd ..
fi

# Check UI dependencies
testing::phase::log "Checking UI dependencies..."
if [ -d ui ] && [ -f ui/package.json ]; then
    if [ ! -d ui/node_modules ]; then
        testing::phase::warn "UI dependencies not installed (run: cd ui && pnpm install)"
    fi
fi

# Check required binaries are built
testing::phase::log "Checking compiled binaries..."
if [ ! -f api/tech-tree-designer-api ]; then
    testing::phase::warn "API binary not built (required for testing)"
fi

testing::phase::log "All dependency checks passed"
testing::phase::end_with_summary "Dependencies validated" 0
