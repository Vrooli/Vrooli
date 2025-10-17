#!/bin/bash
# Integration tests for test-scenario
set -euo pipefail

# Initialize phase environment
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running integration tests..."

# Test 1: Verify scenario binary exists
log::info "Checking binary existence"
if [ -f "api/test-scenario" ]; then
    log::success "Binary exists"
else
    log::warning "Binary not found, building..."
    cd api && go build -o test-scenario . && cd ..
fi

# Test 2: Verify lifecycle check works
log::info "Testing lifecycle enforcement"
if (VROOLI_LIFECYCLE_MANAGED= timeout 1 api/test-scenario 2>&1 || true) | grep -q "must be run through the Vrooli lifecycle system"; then
    log::success "Lifecycle check working correctly"
else
    log::error "Lifecycle check failed"
    ((TESTING_PHASE_ERROR_COUNT++))
fi

# Test 3: Verify hardcoded secrets are present (for scanner testing)
log::info "Verifying test secrets are present"
if grep -q "DatabasePassword" api/main.go && grep -q "APIKey" api/main.go; then
    log::success "Test secrets present in source"
else
    log::error "Test secrets missing"
    ((TESTING_PHASE_ERROR_COUNT++))
fi

# Test 4: Verify edge case handlers exist
log::info "Checking edge case handlers"
if [ -f "api/test_edge_cases.go" ]; then
    edge_case_count=$(grep -c "func handle" api/test_edge_cases.go || true)
    if [ "$edge_case_count" -ge 10 ]; then
        log::success "Found $edge_case_count edge case handlers"
    else
        log::warning "Only found $edge_case_count edge case handlers"
        ((TESTING_PHASE_WARNING_COUNT++))
    fi
else
    log::error "Edge case file not found"
    ((TESTING_PHASE_ERROR_COUNT++))
fi

# Test 5: Verify test infrastructure
log::info "Checking test infrastructure"
if [ -f "api/test_helpers.go" ] && [ -f "api/test_patterns.go" ] && [ -f "api/main_test.go" ]; then
    log::success "Test infrastructure complete"
else
    log::error "Test infrastructure incomplete"
    ((TESTING_PHASE_ERROR_COUNT++))
fi

testing::phase::end_with_summary "Integration tests completed"
