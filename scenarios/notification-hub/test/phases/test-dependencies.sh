#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Testing dependency availability..."

# Test PostgreSQL dependency
testing::phase::log "Checking PostgreSQL dependency..."

if command -v psql >/dev/null 2>&1; then
    testing::phase::success "PostgreSQL client available"
else
    testing::phase::warn "PostgreSQL client not found (optional for development)"
fi

# Test Redis dependency
testing::phase::log "Checking Redis dependency..."

if command -v redis-cli >/dev/null 2>&1; then
    testing::phase::success "Redis client available"
else
    testing::phase::warn "Redis client not found (optional for development)"
fi

# Check service.json dependencies
testing::phase::log "Validating service.json dependencies..."

if [ -f .vrooli/service.json ]; then
    if grep -q '"postgres"' .vrooli/service.json; then
        testing::phase::success "PostgreSQL dependency declared"
    else
        testing::phase::error "PostgreSQL dependency not declared"
        testing::phase::end_with_summary "Missing dependency declaration" 1
    fi

    if grep -q '"redis"' .vrooli/service.json; then
        testing::phase::success "Redis dependency declared"
    else
        testing::phase::error "Redis dependency not declared"
        testing::phase::end_with_summary "Missing dependency declaration" 1
    fi
else
    testing::phase::error "service.json not found"
    testing::phase::end_with_summary "Missing service configuration" 1
fi

# Test Go module dependencies
testing::phase::log "Checking Go module dependencies..."

cd api

if [ -f go.mod ]; then
    testing::phase::success "go.mod found"

    # Check for required packages
    required_packages=("github.com/gin-gonic/gin" "github.com/lib/pq" "github.com/go-redis/redis")
    for pkg in "${required_packages[@]}"; do
        if grep -q "$pkg" go.mod; then
            testing::phase::success "  - $pkg declared"
        else
            testing::phase::warn "  - $pkg not found in go.mod"
        fi
    done
else
    testing::phase::error "go.mod not found"
    cd ..
    testing::phase::end_with_summary "Missing Go module configuration" 1
fi

cd ..

# Test initialization scripts
testing::phase::log "Checking database initialization scripts..."

if [ -f initialization/storage/postgres/schema.sql ]; then
    testing::phase::success "PostgreSQL schema found"
else
    testing::phase::warn "PostgreSQL schema not found"
fi

if [ -f initialization/storage/postgres/seed.sql ]; then
    testing::phase::success "PostgreSQL seed data found"
else
    testing::phase::warn "PostgreSQL seed data not found"
fi

testing::phase::end_with_summary "Dependency tests completed"
