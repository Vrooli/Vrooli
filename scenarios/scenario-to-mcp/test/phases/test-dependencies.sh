#!/bin/bash
# Dependencies test phase
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Checking dependencies..."

# Test 1: Go dependencies
log::info "Checking Go dependencies..."
if [ -f "api/go.mod" ]; then
    cd api

    # Check if go.sum exists
    if [ -f "go.sum" ]; then
        log::success "go.sum present"
    else
        testing::phase::add_warning "go.sum missing, running go mod tidy..."
        go mod tidy &>/dev/null || testing::phase::add_error "go mod tidy failed"
    fi

    # Verify dependencies
    if go mod verify &>/dev/null; then
        log::success "Go dependencies verified"
    else
        testing::phase::add_error "Go dependency verification failed"
    fi

    # Check for outdated dependencies
    if command -v go &>/dev/null; then
        OUTDATED=$(go list -u -m all 2>/dev/null | grep '\[' | wc -l)
        if [ "$OUTDATED" -eq 0 ]; then
            log::success "All Go dependencies up to date"
        else
            testing::phase::add_warning "$OUTDATED Go dependencies have updates available"
        fi
    fi

    cd ..
else
    testing::phase::add_error "No go.mod found"
fi

# Test 2: Node.js dependencies (for lib/)
log::info "Checking Node.js dependencies..."
if [ -f "lib/detector.js" ]; then
    # Check if Node.js is available
    if command -v node &>/dev/null; then
        NODE_VERSION=$(node --version)
        log::success "Node.js available: $NODE_VERSION"

        # Test if detector can run
        if node lib/detector.js scan &>/dev/null 2>&1 || [ $? -eq 1 ]; then
            log::success "Detector script is executable"
        else
            testing::phase::add_warning "Detector script may have issues"
        fi
    else
        testing::phase::add_error "Node.js not available"
    fi
else
    testing::phase::add_warning "No Node.js detector found"
fi

# Test 3: UI dependencies (if UI exists)
log::info "Checking UI dependencies..."
if [ -f "ui/package.json" ]; then
    cd ui

    if [ -d "node_modules" ]; then
        log::success "UI node_modules present"

        # Check package-lock.json
        if [ -f "package-lock.json" ] || [ -f "yarn.lock" ]; then
            log::success "Dependency lock file present"
        else
            testing::phase::add_warning "No dependency lock file found"
        fi
    else
        testing::phase::add_warning "UI node_modules missing (may need npm install)"
    fi

    # Verify package.json structure
    if command -v jq &>/dev/null; then
        if jq -e '.scripts.build' package.json &>/dev/null; then
            log::success "UI has build script"
        else
            testing::phase::add_warning "UI missing build script"
        fi

        if jq -e '.scripts.dev or .scripts.start' package.json &>/dev/null; then
            log::success "UI has dev/start script"
        else
            testing::phase::add_warning "UI missing dev script"
        fi
    fi

    cd ..
else
    log::info "No UI package.json (not required)"
fi

# Test 4: Database dependency
log::info "Checking database dependency..."
DB_URL="${DATABASE_URL:-}"
if [ -n "$DB_URL" ]; then
    # Try to connect to database
    if command -v psql &>/dev/null; then
        if psql "$DB_URL" -c "SELECT 1" &>/dev/null; then
            log::success "Database connection successful"

            # Check for required schemas
            if psql "$DB_URL" -c "SELECT 1 FROM information_schema.schemata WHERE schema_name = 'mcp'" | grep -q 1; then
                log::success "MCP schema exists"
            else
                testing::phase::add_warning "MCP schema may need initialization"
            fi
        else
            testing::phase::add_warning "Database connection failed (may not be started)"
        fi
    else
        testing::phase::add_warning "psql not available for database check"
    fi
else
    testing::phase::add_warning "DATABASE_URL not set"
fi

# Test 5: System dependencies
log::info "Checking system dependencies..."

# Check for required system tools
REQUIRED_TOOLS=("curl" "jq" "grep" "awk")
for tool in "${REQUIRED_TOOLS[@]}"; do
    if command -v "$tool" &>/dev/null; then
        log::success "$tool available"
    else
        testing::phase::add_error "$tool not available"
    fi
done

# Check for optional tools
OPTIONAL_TOOLS=("git" "docker" "make")
for tool in "${OPTIONAL_TOOLS[@]}"; do
    if command -v "$tool" &>/dev/null; then
        log::success "$tool available (optional)"
    else
        log::info "$tool not available (optional)"
    fi
done

# Test 6: Resource dependencies
log::info "Checking resource dependencies..."
if [ -f ".vrooli/service.json" ]; then
    if jq -e '.dependencies.resources' .vrooli/service.json &>/dev/null; then
        REQUIRED_RESOURCES=$(jq -r '.dependencies.resources.required[]?' .vrooli/service.json 2>/dev/null)
        if [ -n "$REQUIRED_RESOURCES" ]; then
            for resource in $REQUIRED_RESOURCES; do
                log::info "Required resource: $resource"
            done
        else
            log::info "No required resources specified"
        fi

        OPTIONAL_RESOURCES=$(jq -r '.dependencies.resources.optional[]?' .vrooli/service.json 2>/dev/null)
        if [ -n "$OPTIONAL_RESOURCES" ]; then
            for resource in $OPTIONAL_RESOURCES; do
                log::info "Optional resource: $resource"
            done
        fi
    fi
fi

testing::phase::end_with_summary "Dependency checks completed"
