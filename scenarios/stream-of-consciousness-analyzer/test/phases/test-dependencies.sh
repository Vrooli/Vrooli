#!/bin/bash
# Dependency test phase - validates required dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Checking dependencies..."

# Check Go dependencies
log::info "Checking Go modules..."
cd api
if go mod verify; then
    log::success "Go modules verified"
else
    testing::phase::add_error "Go module verification failed"
fi

if go mod tidy -diff; then
    log::success "Go modules are tidy"
else
    log::warn "Go modules may need tidying"
fi

# Check required Go packages
log::info "Checking required packages..."
required_packages=(
    "github.com/gorilla/mux"
    "github.com/lib/pq"
    "github.com/rs/cors"
    "github.com/google/uuid"
)

for pkg in "${required_packages[@]}"; do
    if go list -m "$pkg" &>/dev/null; then
        log::success "Package $pkg found"
    else
        testing::phase::add_error "Required package $pkg not found"
    fi
done

# Check database schema
log::info "Checking database schema..."
if [ -f ../initialization/postgres/schema.sql ]; then
    log::success "Database schema file exists"
else
    testing::phase::add_error "Database schema file missing"
fi

# Check n8n workflows
log::info "Checking n8n workflows..."
workflow_count=$(find ../initialization/n8n -name "*.json" 2>/dev/null | wc -l)
if [ "$workflow_count" -ge 5 ]; then
    log::success "Found $workflow_count n8n workflows"
else
    testing::phase::add_error "Expected at least 5 n8n workflows, found $workflow_count"
fi

testing::phase::end_with_summary "Dependency tests completed"
