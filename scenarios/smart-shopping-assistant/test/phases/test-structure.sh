#!/bin/bash
# Test structure phase - validates scenario directory structure

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "15s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Validating smart-shopping-assistant structure"

# Required files
REQUIRED_FILES=(
    ".vrooli/service.json"
    "api/main.go"
    "api/db.go"
    "PRD.md"
    "README.md"
)

# Required test files
REQUIRED_TEST_FILES=(
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
    "test/phases/test-unit.sh"
)

# Check required files
testing::phase::add_test "Validate required files exist"
for file in "${REQUIRED_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        log::success "✓ $file"
    else
        testing::phase::add_error "Missing required file: $file"
    fi
done

# Check test files
testing::phase::add_test "Validate test infrastructure"
for file in "${REQUIRED_TEST_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        log::success "✓ $file"
    else
        testing::phase::add_error "Missing test file: $file"
    fi
done

# Check service.json structure
testing::phase::add_test "Validate service.json structure"
if command -v jq &> /dev/null && [[ -f ".vrooli/service.json" ]]; then
    if jq -e '.service.name' .vrooli/service.json &> /dev/null; then
        log::success "service.json has valid structure"
    else
        testing::phase::add_error "service.json missing required fields"
    fi
else
    testing::phase::add_warning "Cannot validate service.json (jq not available)"
fi

# Check API structure
testing::phase::add_test "Validate API directory structure"
if [[ -d "api" ]]; then
    if ls api/*.go &> /dev/null; then
        log::success "API source files present"
    else
        testing::phase::add_error "No Go source files in api/"
    fi
fi

# Check initialization scripts
testing::phase::add_test "Check database initialization"
if [[ -d "initialization/storage/postgres" ]]; then
    if [[ -f "initialization/storage/postgres/schema.sql" ]]; then
        log::success "PostgreSQL schema file exists"
    else
        testing::phase::add_warning "PostgreSQL schema file missing"
    fi
fi

# Check CLI if it exists
testing::phase::add_test "Check CLI structure"
if [[ -d "cli" ]]; then
    if [[ -f "cli/smart-shopping-assistant" ]] || [[ -f "cli/install.sh" ]]; then
        log::success "CLI components present"
    else
        testing::phase::add_warning "CLI files not found"
    fi
fi

testing::phase::end_with_summary "Structure validation completed"
