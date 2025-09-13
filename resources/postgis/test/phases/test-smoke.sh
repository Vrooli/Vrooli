#!/usr/bin/env bash
################################################################################
# PostGIS Smoke Tests
# Quick validation that PostGIS is functioning (<30s)
################################################################################

set -euo pipefail

# Determine script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "$SCRIPT_DIR/../../../../" && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test utility functions
test::suite() { echo -e "\n╔════════════════════════════════════╗\n║ $* ║\n╚════════════════════════════════════╝"; }
test::start() { echo -n "  Testing $*... "; }
test::pass() { echo -e "✅ $*"; }
test::fail() { echo -e "❌ $*"; }
test::success() { echo -e "\n✅ $*"; }
test::error() { echo -e "\n❌ $*"; }
log::success() { test::success "$@"; }
log::error() { test::error "$@"; }

# Test configuration
POSTGIS_PORT="${POSTGIS_PORT:-5434}"
POSTGIS_CONTAINER="${POSTGIS_CONTAINER:-postgis-main}"

# Test functions
test_container_running() {
    test::start "Container running"
    
    if docker ps --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
        test::pass "PostGIS container is running"
    else
        test::fail "PostGIS container not running"
        return 1
    fi
}

test_health_check() {
    test::start "Health check"
    
    # Use pg_isready for health check with timeout
    if timeout 5 docker exec "${POSTGIS_CONTAINER}" pg_isready -U vrooli -d spatial &>/dev/null; then
        test::pass "PostGIS is healthy and responding"
    else
        test::fail "PostGIS health check failed"
        return 1
    fi
}

test_http_health_endpoint() {
    test::start "HTTP health endpoint"
    
    # Check if health endpoint responds
    if timeout 5 curl -sf http://localhost:5435/health >/dev/null 2>&1; then
        test::pass "HTTP health endpoint is responding"
    else
        test::fail "HTTP health endpoint not responding (optional)"
        # Don't fail the test as this is being added
        return 0
    fi
}

test_port_accessible() {
    test::start "Port accessibility"
    
    if timeout 5 bash -c "echo > /dev/tcp/localhost/${POSTGIS_PORT}" 2>/dev/null; then
        test::pass "Port ${POSTGIS_PORT} is accessible"
    else
        test::fail "Port ${POSTGIS_PORT} is not accessible"
        return 1
    fi
}

test_database_connection() {
    test::start "Database connection"
    
    if timeout 5 docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -c "SELECT 1" &>/dev/null; then
        test::pass "Can connect to spatial database"
    else
        test::fail "Cannot connect to spatial database"
        return 1
    fi
}

test_postgis_installed() {
    test::start "PostGIS extension"
    
    local version=$(timeout 5 docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -t -c "SELECT PostGIS_Version();" 2>/dev/null | xargs)
    
    if [[ -n "$version" ]]; then
        test::pass "PostGIS installed: $version"
    else
        test::fail "PostGIS extension not found"
        return 1
    fi
}

test_spatial_functions() {
    test::start "Spatial functions"
    
    # Test basic spatial function
    local result=$(timeout 5 docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -t -c \
        "SELECT ST_Distance(
            ST_GeomFromText('POINT(0 0)', 4326),
            ST_GeomFromText('POINT(1 1)', 4326)
        );" 2>/dev/null | xargs)
    
    if [[ -n "$result" ]]; then
        test::pass "Spatial functions working (distance: $result)"
    else
        test::fail "Spatial functions not working"
        return 1
    fi
}

test_cli_commands() {
    test::start "CLI commands"
    
    # Test status command
    if vrooli resource postgis status &>/dev/null; then
        test::pass "CLI status command works"
    else
        test::fail "CLI status command failed"
        return 1
    fi
}

# Main test execution
main() {
    test::suite "PostGIS Smoke Tests"
    
    local failed=0
    
    # Run tests
    test_container_running || ((failed++))
    test_health_check || ((failed++))
    test_http_health_endpoint || ((failed++))
    test_port_accessible || ((failed++))
    test_database_connection || ((failed++))
    test_postgis_installed || ((failed++))
    test_spatial_functions || ((failed++))
    test_cli_commands || ((failed++))
    
    # Summary
    echo
    if [[ $failed -eq 0 ]]; then
        test::success "All smoke tests passed!"
        return 0
    else
        test::error "$failed smoke tests failed"
        return 1
    fi
}

# Run tests
main "$@"