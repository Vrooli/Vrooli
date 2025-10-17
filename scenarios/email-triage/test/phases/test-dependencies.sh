#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Checking scenario dependencies..."

# Check required binaries
testing::phase::check_binary "go" "Go compiler required for API"
testing::phase::check_binary "node" "Node.js required for UI"
testing::phase::check_binary "npm" "NPM required for UI dependencies"

# Check Go module dependencies
if [ -f "api/go.mod" ]; then
    testing::phase::log "Verifying Go module dependencies..."
    cd api

    # Check for missing dependencies
    if ! go mod verify &>/dev/null; then
        testing::phase::error "Go module verification failed"
        testing::phase::end_with_summary "Dependency check failed" 1
    fi

    # Check for tidy modules
    if ! go mod tidy -diff &>/dev/null; then
        testing::phase::warn "go.mod or go.sum may need tidying"
    fi

    cd ..
    testing::phase::success "Go dependencies verified"
else
    testing::phase::error "api/go.mod not found"
    testing::phase::end_with_summary "Missing go.mod" 1
fi

# Check Node.js dependencies
if [ -f "ui/package.json" ]; then
    testing::phase::log "Checking Node.js dependencies..."
    cd ui

    if [ ! -d "node_modules" ]; then
        testing::phase::warn "node_modules not found, dependencies may need installation"
    else
        testing::phase::success "Node.js dependencies present"
    fi

    cd ..
else
    testing::phase::warn "ui/package.json not found, UI dependencies not checked"
fi

# Check required directories
testing::phase::log "Verifying directory structure..."
required_dirs=(
    "api"
    "api/handlers"
    "api/services"
    "api/models"
    "cli"
    "initialization"
    "initialization/postgres"
    "ui"
    "test/phases"
)

for dir in "${required_dirs[@]}"; do
    if [ ! -d "$dir" ]; then
        testing::phase::error "Required directory missing: $dir"
        testing::phase::end_with_summary "Missing directory: $dir" 1
    fi
done
testing::phase::success "All required directories present"

# Check required files
testing::phase::log "Verifying required files..."
required_files=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "Makefile"
    "api/main.go"
    "api/go.mod"
    "initialization/postgres/schema.sql"
)

for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        testing::phase::error "Required file missing: $file"
        testing::phase::end_with_summary "Missing file: $file" 1
    fi
done
testing::phase::success "All required files present"

# Check external resource dependencies
testing::phase::log "Checking external resource dependencies..."

# Check PostgreSQL dependency
if command -v resource-postgres &>/dev/null; then
    if resource-postgres status | grep -q "running\|healthy"; then
        testing::phase::success "PostgreSQL resource available"
    else
        testing::phase::warn "PostgreSQL resource not running (required for full functionality)"
    fi
else
    testing::phase::warn "resource-postgres CLI not available"
fi

# Check Qdrant dependency
if command -v resource-qdrant &>/dev/null; then
    if resource-qdrant status | grep -q "running\|healthy"; then
        testing::phase::success "Qdrant resource available"
    else
        testing::phase::warn "Qdrant resource not running (required for semantic search)"
    fi
else
    testing::phase::warn "resource-qdrant CLI not available"
fi

# Check Ollama dependency (optional)
if command -v resource-ollama &>/dev/null; then
    if resource-ollama status | grep -q "running\|healthy"; then
        testing::phase::success "Ollama resource available (optional AI features enabled)"
    else
        testing::phase::warn "Ollama resource not running (AI features will use fallback)"
    fi
else
    testing::phase::log "Ollama not available (optional - fallback mode will be used)"
fi

# Check scenario-authenticator dependency (optional in DEV_MODE)
testing::phase::log "Checking authentication dependency..."
if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
    testing::phase::success "scenario-authenticator available"
else
    testing::phase::warn "scenario-authenticator not available (DEV_MODE can be used)"
fi

# Check CLI binary
testing::phase::log "Checking CLI binary..."
if [ -f "cli/email-triage" ]; then
    testing::phase::success "CLI binary built"
else
    testing::phase::warn "CLI binary not found (run 'make build' or setup)"
fi

# Check API binary
testing::phase::log "Checking API binary..."
if [ -f "api/email-triage-api" ]; then
    testing::phase::success "API binary built"
else
    testing::phase::warn "API binary not found (run 'make build' or setup)"
fi

testing::phase::end_with_summary "Dependency check completed" 0
