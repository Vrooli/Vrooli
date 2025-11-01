#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"
source "$TESTING_PHASE_SCENARIO_DIR/test/utils/phase-extensions.sh"

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
        testing::phase::warn "go.mod or go.sum needs tidying"
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
    "tasks/active"
    "tasks/backlog/manual"
    "tasks/backlog/generated"
    "tasks/staged"
    "tasks/completed"
    "tasks/failed"
    "api"
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
    "api/main.go"
    "ui/server.js"
    "Makefile"
)

for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        testing::phase::error "Required file missing: $file"
        testing::phase::end_with_summary "Missing file: $file" 1
    fi
done
testing::phase::success "All required files present"

# Check for resource dependencies in service.json
testing::phase::log "Checking resource dependencies..."
if command -v jq &>/dev/null; then
    required_resources=$(jq -r '.resources | to_entries[] | select(.value.required == true) | .key' .vrooli/service.json 2>/dev/null)

    if [ -n "$required_resources" ]; then
        testing::phase::log "Required resources:"
        echo "$required_resources" | while read -r resource; do
            testing::phase::log "  - $resource"
        done
    fi
else
    testing::phase::warn "jq not available, skipping resource dependency check"
fi

testing::phase::end_with_summary "Dependency check completed"
