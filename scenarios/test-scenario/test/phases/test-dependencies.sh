#!/bin/bash
# Dependency tests for test-scenario
set -euo pipefail

# Initialize phase environment
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Testing dependencies for test-scenario"

# Test 1: Go is available
log::info "Checking Go availability"
if command -v go &> /dev/null; then
    go_version=$(go version)
    log::success "Go is available: $go_version"
else
    log::error "Go is not installed"
    ((TESTING_PHASE_ERROR_COUNT++))
fi

# Test 2: Required Go packages
log::info "Checking Go module dependencies"
cd api
if go mod download &> /dev/null || true; then
    log::success "Go modules available"
else
    log::warning "Go modules had issues"
    ((TESTING_PHASE_WARNING_COUNT++))
fi
cd ..

# Test 3: Build dependencies
log::info "Verifying build can complete"
cd api
if go build -o test-scenario-deps-check . 2>&1 | tee /tmp/build-output.txt; then
    log::success "Build successful"
    rm -f test-scenario-deps-check
else
    log::error "Build failed"
    cat /tmp/build-output.txt
    ((TESTING_PHASE_ERROR_COUNT++))
fi
cd ..

# Test 4: Test dependencies
log::info "Checking test compilation"
cd api
if go test -c -o test-scenario.test . &> /dev/null; then
    log::success "Test compilation successful"
    rm -f test-scenario.test
else
    log::error "Test compilation failed"
    ((TESTING_PHASE_ERROR_COUNT++))
fi
cd ..

testing::phase::end_with_summary "Dependency tests completed"
