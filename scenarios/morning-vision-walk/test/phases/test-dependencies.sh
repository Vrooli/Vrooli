#!/bin/bash
# Dependency tests for Morning Vision Walk scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::step "Checking Go dependencies"

if [[ -f "api/go.mod" ]]; then
    cd api

    # Verify go.mod is valid
    if go mod verify 2>/dev/null; then
        testing::phase::success "Go module verified successfully"
    else
        testing::phase::warn "Go module verification issues (may need: go mod download)"
    fi

    # Check if dependencies are downloadable
    testing::phase::info "Checking dependency availability..."
    if go mod download 2>&1 | grep -q "error"; then
        testing::phase::warn "Some dependencies may have issues"
    else
        testing::phase::success "All Go dependencies available"
    fi

    # List direct dependencies
    DEP_COUNT=$(go list -m all 2>/dev/null | wc -l)
    testing::phase::info "Total dependencies: $DEP_COUNT"

    cd ..
else
    testing::phase::error "No go.mod file found"
fi

testing::phase::step "Checking Node.js dependencies"

if [[ -f "ui/package.json" ]]; then
    cd ui

    # Check if package.json is valid
    if jq empty package.json 2>/dev/null; then
        testing::phase::success "package.json is valid JSON"

        # Check for required dependencies
        DEPS=$(jq -r '.dependencies | keys | .[]' package.json 2>/dev/null)
        DEP_COUNT=$(echo "$DEPS" | wc -l)
        testing::phase::info "Node.js dependencies: $DEP_COUNT"

        # Check if node_modules exists
        if [[ -d "node_modules" ]]; then
            testing::phase::success "node_modules directory exists"
        else
            testing::phase::warn "node_modules not found (may need: npm install)"
        fi
    else
        testing::phase::error "package.json is invalid JSON"
    fi

    cd ..
else
    testing::phase::warn "No package.json file found in ui/"
fi

testing::phase::step "Checking resource dependencies"

# Check PostgreSQL dependency
if command -v psql &>/dev/null || [[ -f "$APP_ROOT/.vrooli/postgres/data/PG_VERSION" ]]; then
    testing::phase::success "PostgreSQL available"
else
    testing::phase::warn "PostgreSQL not detected"
fi

# Check Redis dependency
if command -v redis-cli &>/dev/null || pgrep -f redis-server &>/dev/null; then
    testing::phase::success "Redis available"
else
    testing::phase::warn "Redis not detected"
fi

# Check Qdrant dependency
if pgrep -f qdrant &>/dev/null || [[ -d "$APP_ROOT/.vrooli/qdrant" ]]; then
    testing::phase::success "Qdrant available"
else
    testing::phase::warn "Qdrant not detected"
fi

# Check n8n dependency
if pgrep -f "n8n" &>/dev/null || command -v n8n &>/dev/null; then
    testing::phase::success "n8n available"
else
    testing::phase::warn "n8n not detected"
fi

testing::phase::step "Checking system dependencies"

# Check required system commands
REQUIRED_COMMANDS=("curl" "jq" "grep" "awk")

for cmd in "${REQUIRED_COMMANDS[@]}"; do
    if command -v "$cmd" &>/dev/null; then
        testing::phase::success "Command available: $cmd"
    else
        testing::phase::error "Missing command: $cmd"
    fi
done

# Check Go installation
if command -v go &>/dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    testing::phase::success "Go installed: $GO_VERSION"

    # Check minimum Go version (1.18+)
    GO_MINOR=$(echo "$GO_VERSION" | sed 's/go1\.\([0-9]*\).*/\1/')
    if [[ $GO_MINOR -ge 18 ]]; then
        testing::phase::success "Go version meets minimum requirement (1.18+)"
    else
        testing::phase::warn "Go version may be too old: $GO_VERSION"
    fi
else
    testing::phase::error "Go not installed"
fi

# Check Node.js installation
if command -v node &>/dev/null; then
    NODE_VERSION=$(node --version)
    testing::phase::success "Node.js installed: $NODE_VERSION"

    # Check minimum Node version (14+)
    NODE_MAJOR=$(echo "$NODE_VERSION" | sed 's/v\([0-9]*\).*/\1/')
    if [[ $NODE_MAJOR -ge 14 ]]; then
        testing::phase::success "Node.js version meets minimum requirement (14+)"
    else
        testing::phase::warn "Node.js version may be too old: $NODE_VERSION"
    fi
else
    testing::phase::warn "Node.js not installed"
fi

testing::phase::step "Checking scenario integrations"

# Check if referenced scenarios exist
INTEGRATIONS=("stream-of-consciousness-analyzer" "task-planner" "mind-maps")

for integration in "${INTEGRATIONS[@]}"; do
    if [[ -d "$APP_ROOT/scenarios/$integration" ]]; then
        testing::phase::success "Integration scenario exists: $integration"
    else
        testing::phase::warn "Integration scenario missing: $integration"
    fi
done

testing::phase::end_with_summary "Dependency checks completed"
