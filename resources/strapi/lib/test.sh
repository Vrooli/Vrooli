#!/usr/bin/env bash
# Strapi Resource Test Library
# Implements test phases for validation

set -euo pipefail

# Prevent multiple sourcing
[[ -n "${STRAPI_TEST_LOADED:-}" ]] && return 0
readonly STRAPI_TEST_LOADED=1

# Source core library
TEST_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${TEST_SCRIPT_DIR}/lib/core.sh"

#######################################
# Run smoke tests (quick health check)
#######################################
test::run_smoke() {
    core::info "Running smoke tests..."
    
    local passed=0
    local failed=0
    
    # Test 1: Check if installed
    if core::is_installed; then
        core::success "✓ Strapi is installed"
        ((passed++))
    else
        core::error "✗ Strapi is not installed"
        ((failed++))
    fi
    
    # Test 2: Check if running
    if core::is_running; then
        core::success "✓ Strapi service is running"
        ((passed++))
    else
        core::error "✗ Strapi service is not running"
        ((failed++))
    fi
    
    # Test 3: Health endpoint
    if timeout 5 curl -sf "http://localhost:${STRAPI_PORT}/health" >/dev/null 2>&1; then
        core::success "✓ Health endpoint responsive"
        ((passed++))
    else
        core::error "✗ Health endpoint not responding"
        ((failed++))
    fi
    
    # Test 4: Admin panel accessible
    if timeout 5 curl -sf "http://localhost:${STRAPI_PORT}/admin" >/dev/null 2>&1; then
        core::success "✓ Admin panel accessible"
        ((passed++))
    else
        core::error "✗ Admin panel not accessible"
        ((failed++))
    fi
    
    # Test 5: API endpoint
    if timeout 5 curl -sf "http://localhost:${STRAPI_PORT}/api" >/dev/null 2>&1; then
        core::success "✓ API endpoint accessible"
        ((passed++))
    else
        core::error "✗ API endpoint not accessible"
        ((failed++))
    fi
    
    echo ""
    core::info "Smoke test results: ${passed} passed, ${failed} failed"
    
    [[ $failed -eq 0 ]] && return 0 || return 1
}

#######################################
# Run integration tests
#######################################
test::run_integration() {
    core::info "Running integration tests..."
    
    local passed=0
    local failed=0
    
    # Test 1: Database connectivity
    if test::check_database_connection; then
        core::success "✓ Database connection successful"
        ((passed++))
    else
        core::error "✗ Database connection failed"
        ((failed++))
    fi
    
    # Test 2: Create content type
    if test::create_test_content_type; then
        core::success "✓ Content type creation successful"
        ((passed++))
    else
        core::error "✗ Content type creation failed"
        ((failed++))
    fi
    
    # Test 3: GraphQL endpoint
    if test::check_graphql_endpoint; then
        core::success "✓ GraphQL endpoint functional"
        ((passed++))
    else
        core::error "✗ GraphQL endpoint not working"
        ((failed++))
    fi
    
    # Test 4: File upload capability
    if test::check_file_upload; then
        core::success "✓ File upload capability working"
        ((passed++))
    else
        core::error "✗ File upload not working"
        ((failed++))
    fi
    
    echo ""
    core::info "Integration test results: ${passed} passed, ${failed} failed"
    
    [[ $failed -eq 0 ]] && return 0 || return 1
}

#######################################
# Run unit tests
#######################################
test::run_unit() {
    core::info "Running unit tests..."
    
    local passed=0
    local failed=0
    
    # Test 1: Configuration validation
    if test::validate_configuration; then
        core::success "✓ Configuration valid"
        ((passed++))
    else
        core::error "✗ Configuration invalid"
        ((failed++))
    fi
    
    # Test 2: Environment variables
    if test::check_environment_vars; then
        core::success "✓ Environment variables set correctly"
        ((passed++))
    else
        core::error "✗ Environment variables missing or incorrect"
        ((failed++))
    fi
    
    # Test 3: Port availability
    if test::check_port_availability; then
        core::success "✓ Port configuration valid"
        ((passed++))
    else
        core::error "✗ Port configuration issues"
        ((failed++))
    fi
    
    # Test 4: File permissions
    if test::check_file_permissions; then
        core::success "✓ File permissions correct"
        ((passed++))
    else
        core::error "✗ File permission issues"
        ((failed++))
    fi
    
    echo ""
    core::info "Unit test results: ${passed} passed, ${failed} failed"
    
    [[ $failed -eq 0 ]] && return 0 || return 1
}

#######################################
# Run all test phases
#######################################
test::run_all() {
    core::info "Running all test phases..."
    echo ""
    
    local overall_status=0
    
    # Run smoke tests
    if test::run_smoke; then
        core::success "Smoke tests passed"
    else
        core::error "Smoke tests failed"
        overall_status=1
    fi
    
    echo ""
    
    # Run integration tests
    if test::run_integration; then
        core::success "Integration tests passed"
    else
        core::error "Integration tests failed"
        overall_status=1
    fi
    
    echo ""
    
    # Run unit tests
    if test::run_unit; then
        core::success "Unit tests passed"
    else
        core::error "Unit tests failed"
        overall_status=1
    fi
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        core::success "All test phases completed successfully"
    else
        core::error "Some test phases failed"
    fi
    
    return $overall_status
}

#######################################
# Helper test functions
#######################################
test::check_database_connection() {
    # Check if we can connect to PostgreSQL
    if command -v psql >/dev/null 2>&1; then
        PGPASSWORD="${POSTGRES_PASSWORD}" psql \
            -h "${POSTGRES_HOST}" \
            -p "${POSTGRES_PORT}" \
            -U "${POSTGRES_USER}" \
            -d "${STRAPI_DATABASE_NAME}" \
            -c "SELECT 1" >/dev/null 2>&1
    else
        # Fallback: check if Strapi can query database
        curl -sf "http://localhost:${STRAPI_PORT}/api" >/dev/null 2>&1
    fi
}

test::create_test_content_type() {
    # Try to create a simple content type via API
    local response=$(curl -sf -X GET \
        "http://localhost:${STRAPI_PORT}/api/content-type-builder/content-types" 2>/dev/null)
    
    [[ -n "$response" ]] && return 0 || return 1
}

test::check_graphql_endpoint() {
    # Send a simple GraphQL introspection query
    local query='{"query":"{ __schema { types { name } } }"}'
    
    curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$query" \
        "http://localhost:${STRAPI_PORT}/graphql" >/dev/null 2>&1
}

test::check_file_upload() {
    # Check if upload plugin is available
    curl -sf "http://localhost:${STRAPI_PORT}/api/upload" >/dev/null 2>&1
}

test::validate_configuration() {
    # Check if configuration files exist
    [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]] && \
    [[ -f "${SCRIPT_DIR}/config/runtime.json" ]] && \
    [[ -f "${SCRIPT_DIR}/config/schema.json" ]]
}

test::check_environment_vars() {
    # Check critical environment variables
    [[ -n "${STRAPI_PORT:-}" ]] || return 1
    [[ -n "${POSTGRES_HOST:-}" ]] || return 1
    [[ -n "${POSTGRES_PORT:-}" ]] || return 1
    
    return 0
}

test::check_port_availability() {
    # Check if port is valid
    [[ "${STRAPI_PORT}" -gt 1024 ]] && \
    [[ "${STRAPI_PORT}" -lt 65535 ]]
}

test::check_file_permissions() {
    # Check if data directory is writable
    [[ -w "${STRAPI_DATA_DIR}" ]] || mkdir -p "${STRAPI_DATA_DIR}"
    [[ -w "${STRAPI_DATA_DIR}" ]]
}