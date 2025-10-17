#!/bin/bash
# Structure validation test phase - validates project structure
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 30-second target
testing::phase::init --target-time "30s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Validating project structure for prompt-manager..."

# Check required directories
log::info "Checking required directories..."
required_dirs=(
    "api"
    "ui"
    "cli"
    "test/phases"
    "initialization/storage/postgres"
    ".vrooli"
)

all_dirs_exist=true
for dir in "${required_dirs[@]}"; do
    if [ -d "$dir" ]; then
        log::success "Directory exists: $dir"
    else
        log::error "Missing directory: $dir"
        testing::phase::add_error "Missing directory: $dir"
        all_dirs_exist=false
    fi
done

# Check required files
log::info "Checking required files..."
required_files=(
    ".vrooli/service.json"
    "api/main.go"
    "api/main_test.go"
    "api/test_helpers.go"
    "api/test_patterns.go"
    "cli/prompt-manager"
    "ui/server.js"
    "initialization/storage/postgres/schema_with_prefix.sql"
    "test/phases/test-unit.sh"
)

all_files_exist=true
for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        log::success "File exists: $file"
    else
        log::error "Missing file: $file"
        testing::phase::add_error "Missing file: $file"
        all_files_exist=false
    fi
done

# Check service.json schema validation
log::info "Validating service.json..."
if [ -f ".vrooli/service.json" ]; then
    if jq empty .vrooli/service.json 2>/dev/null; then
        log::success "service.json is valid JSON"
    else
        log::error "service.json is not valid JSON"
        testing::phase::add_error "Invalid service.json"
    fi
fi

# Check Go module structure
log::info "Checking Go module structure..."
if [ -f "api/go.mod" ]; then
    log::success "Go module initialized"
else
    log::warn "Go module not initialized in api/"
fi

# Validate test file structure
log::info "Validating test file structure..."
if [ -f "api/test_helpers.go" ] && [ -f "api/test_patterns.go" ]; then
    log::success "Test infrastructure files present"
else
    log::error "Missing test infrastructure files"
    testing::phase::add_error "Missing test infrastructure"
fi

# End with summary
testing::phase::end_with_summary "Structure validation completed"
