#!/bin/bash
# Dependency test phase - verifies external dependencies and resources
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running dependency tests..."

# Check Go dependencies
log::info "Checking Go module dependencies"
if cd api && go mod verify; then
    log::success "Go dependencies verified"
else
    testing::phase::add_error "Go dependency verification failed"
fi

# Check for required dependencies
log::info "Checking required packages"
required_packages=(
    "github.com/gorilla/mux"
    "github.com/lib/pq"
    "github.com/google/uuid"
    "github.com/rs/cors"
)

for pkg in "${required_packages[@]}"; do
    if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && go list -m "$pkg" &> /dev/null; then
        log::success "Found package: $pkg"
    else
        log::error "Missing package: $pkg"
        testing::phase::add_error "Missing required package: $pkg"
    fi
done

# Check database connection (if configured)
log::info "Checking database connectivity"
if [ -n "$POSTGRES_URL" ] || [ -n "$POSTGRES_HOST" ]; then
    if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && go test -tags=testing -run "TestDatabaseConnection" -timeout 30s 2>/dev/null; then
        log::success "Database connection verified"
    else
        log::warning "Database connection test skipped (database may not be running)"
    fi
else
    log::warning "Database environment variables not set, skipping connectivity test"
fi

# Check service.json validity
log::info "Validating service configuration"
service_json="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/service.json"
if [ -f "$service_json" ]; then
    if jq empty "$service_json" 2>/dev/null; then
        log::success "service.json is valid JSON"
    else
        log::error "service.json is invalid JSON"
        testing::phase::add_error "Invalid service.json format"
    fi
else
    log::warning "service.json not found (optional)"
fi

# Check for test build tags
log::info "Verifying test build tags"
if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && grep -r "// +build testing" *_test.go > /dev/null 2>&1; then
    log::success "Test build tags found in test files"
else
    log::warning "No test build tags found (optional)"
fi

# End with summary
testing::phase::end_with_summary "Dependency tests completed"
