#!/bin/bash
# Structure test phase - validates project structure and organization
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running structure validation tests..."

# Check required directories
log::step "Checking directory structure"
required_dirs=(
    "api"
    "ui"
    "test"
    "test/phases"
    ".vrooli"
)

for dir in "${required_dirs[@]}"; do
    if [ -d "${TESTING_PHASE_SCENARIO_DIR}/${dir}" ]; then
        log::success "Found directory: $dir"
    else
        log::error "Missing directory: $dir"
        testing::phase::add_error "Missing required directory: $dir"
    fi
done

# Check required files
log::step "Checking required files"
required_files=(
    "api/main.go"
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
    "test/phases/test-unit.sh"
    "PRD.md"
    "README.md"
)

for file in "${required_files[@]}"; do
    if [ -f "${TESTING_PHASE_SCENARIO_DIR}/${file}" ]; then
        log::success "Found file: $file"
    else
        log::error "Missing file: $file"
        testing::phase::add_error "Missing required file: $file"
    fi
done

# Check test file naming conventions
log::step "Validating test file naming conventions"
if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && find . -name "*_test.go" | grep -v "/vendor/" > /dev/null; then
    test_count=$(find . -name "*_test.go" | grep -v "/vendor/" | wc -l)
    log::success "Found $test_count test files"

    if [ "$test_count" -lt 3 ]; then
        log::warning "Low number of test files ($test_count), consider adding more tests"
    fi
else
    log::error "No Go test files found"
    testing::phase::add_error "No test files found in api directory"
fi

# Check for test helper functions
log::step "Checking test infrastructure"
if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && grep -q "setupTestLogger" test_helpers.go 2>/dev/null; then
    log::success "Test helper functions found"
else
    log::warning "Test helper functions may be missing"
fi

# Check test patterns
if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && grep -q "TestScenarioBuilder" test_patterns.go 2>/dev/null; then
    log::success "Test pattern builder found"
else
    log::warning "Test pattern builder may be missing"
fi

# Validate Go code structure
log::step "Checking Go code organization"
if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && go list ./... > /dev/null 2>&1; then
    log::success "Go code structure is valid"
else
    log::error "Go code structure has issues"
    testing::phase::add_error "Invalid Go code structure"
fi

# Check for documentation
log::step "Checking documentation"
docs_to_check=(
    "PRD.md"
    "README.md"
)

for doc in "${docs_to_check[@]}"; do
    if [ -f "${TESTING_PHASE_SCENARIO_DIR}/${doc}" ]; then
        word_count=$(wc -w < "${TESTING_PHASE_SCENARIO_DIR}/${doc}")
        if [ "$word_count" -gt 50 ]; then
            log::success "Documentation found: $doc ($word_count words)"
        else
            log::warning "Documentation is sparse: $doc ($word_count words)"
        fi
    else
        log::warning "Documentation missing: $doc"
    fi
done

# Check lifecycle compliance
log::step "Checking lifecycle compliance"
if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && grep -q "VROOLI_LIFECYCLE_MANAGED" main.go; then
    log::success "Lifecycle management check found in main.go"
else
    log::warning "Lifecycle management check may be missing"
fi

# End with summary
testing::phase::end_with_summary "Structure validation completed"
