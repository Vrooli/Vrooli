#!/bin/bash
# Structure validation test phase
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Validating scenario structure..."

# Test 1: Required files exist
log::info "Checking required files..."
REQUIRED_FILES=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "Makefile"
    "api/main.go"
    "api/go.mod"
    "cli/scenario-to-mcp"
    "cli/install.sh"
    "lib/detector.js"
)

MISSING_FILES=()
for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        MISSING_FILES+=("$file")
    fi
done

if [ ${#MISSING_FILES[@]} -eq 0 ]; then
    log::success "All required files present"
else
    for file in "${MISSING_FILES[@]}"; do
        testing::phase::add_error "Missing required file: $file"
    done
fi

# Test 2: Required directories exist
log::info "Checking required directories..."
REQUIRED_DIRS=(
    "api"
    "cli"
    "lib"
    "ui"
    "test/phases"
    "initialization"
)

MISSING_DIRS=()
for dir in "${REQUIRED_DIRS[@]}"; do
    if [ ! -d "$dir" ]; then
        MISSING_DIRS+=("$dir")
    fi
done

if [ ${#MISSING_DIRS[@]} -eq 0 ]; then
    log::success "All required directories present"
else
    for dir in "${MISSING_DIRS[@]}"; do
        testing::phase::add_error "Missing required directory: $dir"
    done
fi

# Test 3: API structure
log::info "Checking API structure..."
if [ -f "api/main.go" ]; then
    if grep -q "func main()" api/main.go; then
        log::success "API has main function"
    else
        testing::phase::add_error "API missing main function"
    fi

    if grep -q "VROOLI_LIFECYCLE_MANAGED" api/main.go; then
        log::success "API has lifecycle protection"
    else
        testing::phase::add_warning "API missing lifecycle protection"
    fi
fi

# Test 4: CLI structure
log::info "Checking CLI structure..."
if [ -x "cli/scenario-to-mcp" ]; then
    log::success "CLI is executable"

    # Check for required commands
    REQUIRED_COMMANDS=("help" "version" "list" "add" "test" "registry")
    for cmd in "${REQUIRED_COMMANDS[@]}"; do
        if grep -q "\"$cmd\"" cli/scenario-to-mcp; then
            log::success "CLI has $cmd command"
        else
            testing::phase::add_warning "CLI missing $cmd command definition"
        fi
    done
else
    testing::phase::add_error "CLI is not executable"
fi

# Test 5: Test infrastructure
log::info "Checking test infrastructure..."
if [ -f "test/phases/test-unit.sh" ]; then
    log::success "Unit test phase exists"
else
    testing::phase::add_error "Missing unit test phase"
fi

if [ -f "test/phases/test-integration.sh" ]; then
    log::success "Integration test phase exists"
else
    testing::phase::add_warning "Missing integration test phase"
fi

# Test 6: Go module structure
log::info "Checking Go module..."
if [ -f "api/go.mod" ]; then
    if grep -q "module" api/go.mod; then
        log::success "go.mod has module declaration"
    else
        testing::phase::add_error "go.mod missing module declaration"
    fi

    # Check for required dependencies
    if grep -q "github.com/gorilla/mux" api/go.mod; then
        log::success "Has gorilla/mux dependency"
    else
        testing::phase::add_warning "Missing gorilla/mux dependency"
    fi

    if grep -q "github.com/lib/pq" api/go.mod; then
        log::success "Has postgres driver dependency"
    else
        testing::phase::add_warning "Missing postgres driver dependency"
    fi
fi

# Test 7: Documentation
log::info "Checking documentation..."
if [ -f "PRD.md" ]; then
    if grep -q "Product Requirements" PRD.md; then
        log::success "PRD has proper structure"
    else
        testing::phase::add_warning "PRD may be incomplete"
    fi
fi

if [ -f "README.md" ]; then
    if grep -q "scenario-to-mcp" README.md; then
        log::success "README references scenario"
    else
        testing::phase::add_warning "README may be incomplete"
    fi
fi

testing::phase::end_with_summary "Structure validation completed"
