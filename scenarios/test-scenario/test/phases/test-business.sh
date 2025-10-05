#!/bin/bash
# Business logic tests for test-scenario
set -euo pipefail

# Initialize phase environment
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Testing business logic for test-scenario"

# Test 1: Verify scenario purpose
log::info "Verifying scenario purpose"
if grep -q "testing vulnerability scanner" api/main.go || grep -q "intentional hardcoded secrets" api/main.go; then
    log::success "Scenario correctly implements test fixtures for vulnerability scanning"
else
    log::error "Scenario purpose not clear"
    ((TESTING_PHASE_ERROR_COUNT++))
fi

# Test 2: Verify hardcoded secrets are detectable
log::info "Checking that secrets are in detectable format"
secret_count=0
grep -q "DatabasePassword" api/main.go && ((secret_count++)) || true
grep -q "APIKey.*sk-" api/main.go && ((secret_count++)) || true
grep -q "JWTSecret" api/main.go && ((secret_count++)) || true
grep -q "postgres://.*:.*@" api/main.go && ((secret_count++)) || true

if [ "$secret_count" -ge 4 ]; then
    log::success "Found $secret_count detectable secrets (expected for test scenario)"
else
    log::warning "Only found $secret_count secrets, expected 4+"
    ((TESTING_PHASE_WARNING_COUNT++))
fi

# Test 3: Verify edge case coverage
log::info "Checking edge case coverage"
edge_case_handlers=$(grep -c "^func handle" api/test_edge_cases.go 2>/dev/null || echo "0")
if [ "$edge_case_handlers" -ge 10 ]; then
    log::success "Implements $edge_case_handlers edge case handlers"
else
    log::warning "Only $edge_case_handlers edge case handlers found"
    ((TESTING_PHASE_WARNING_COUNT++))
fi

# Test 4: Verify lifecycle enforcement
log::info "Checking lifecycle enforcement logic"
if grep -q "VROOLI_LIFECYCLE_MANAGED" api/main.go; then
    log::success "Lifecycle enforcement implemented"
else
    log::error "Lifecycle enforcement missing"
    ((TESTING_PHASE_ERROR_COUNT++))
fi

# Test 5: Verify error message quality
log::info "Checking error messages"
if grep -q "vrooli scenario start" api/main.go; then
    log::success "User-friendly error messages present"
else
    log::warning "Error messages could be improved"
    ((TESTING_PHASE_WARNING_COUNT++))
fi

testing::phase::end_with_summary "Business logic tests completed"
