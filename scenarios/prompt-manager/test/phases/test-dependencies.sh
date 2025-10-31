#!/bin/bash
# Dependencies test phase - validates all dependencies are available
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 30-second target
testing::phase::init --target-time "30s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Checking dependencies for prompt-manager..."

# Check Go dependencies
log::info "Checking Go dependencies..."
if (cd api && go mod verify) 2>&1; then
    log::success "Go dependencies verified"
else
    testing::phase::add_error "Go dependency verification failed"
fi

# Check if binaries can be built
log::info "Checking Go build..."
if (cd api && go build -o /tmp/prompt-manager-test .) 2>&1; then
    log::success "Go build successful"
    rm -f /tmp/prompt-manager-test
else
    testing::phase::add_error "Go build failed"
fi

# Check Node.js dependencies
log::info "Checking Node.js dependencies..."
if [ -d "ui/node_modules" ]; then
    log::success "Node.js dependencies installed"
else
    log::warn "Node.js dependencies not installed"
fi

# Check required resources
log::info "Checking required resources..."
if vrooli resource status postgres &>/dev/null; then
    log::success "PostgreSQL available"
else
    testing::phase::add_error "PostgreSQL not available"
fi

# Check optional resources
if vrooli resource status qdrant &>/dev/null; then
    log::info "Qdrant vector database available (optional)"
fi

if vrooli resource status ollama &>/dev/null; then
    log::info "Ollama LLM available (optional)"
fi

# End with summary
testing::phase::end_with_summary "Dependency checks completed"
