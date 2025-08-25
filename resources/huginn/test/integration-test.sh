#!/usr/bin/env bash
# Huginn Integration Test
# Tests real Huginn agent platform functionality
# Tests web interface, API endpoints, agent operations, and workflows

set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/huginn/test"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_DIR}/resources/tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load Huginn configuration
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../config/defaults.sh"
huginn::export_config

# Override library defaults with Huginn-specific settings
SERVICE_NAME="huginn"
BASE_URL="${HUGINN_BASE_URL:-http://localhost:4111}"
HEALTH_ENDPOINT="/"
REQUIRED_TOOLS=("curl" "jq" "docker")
SERVICE_METADATA=(
    "Port: ${HUGINN_PORT:-4111}"
    "Container: ${CONTAINER_NAME:-huginn}"
    "Database Container: ${DB_CONTAINER_NAME:-huginn-postgres}"
    "Admin User: ${DEFAULT_ADMIN_USERNAME:-admin}"
)

#######################################
# HUGINN-SPECIFIC TEST FUNCTIONS
#######################################

test_web_interface() {
    local test_name="web interface accessibility"
    
    local response
    if response=$(make_api_request "/" "GET" 10); then
        if echo "$response" | grep -qi "huginn\|login\|sign in\|<!DOCTYPE html>"; then
            log_test_result "$test_name" "PASS" "web interface accessible"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "web interface not accessible"
    return 1
}

test_login_page() {
    local test_name="login page availability"
    
    local response
    local endpoints=("/users/sign_in" "/login" "/")
    
    for endpoint in "${endpoints[@]}"; do
        if response=$(make_api_request "$endpoint" "GET" 5); then
            if echo "$response" | grep -qi "email\|password\|sign in\|login"; then
                log_test_result "$test_name" "PASS" "login page found at $endpoint"
                return 0
            fi
        fi
    done
    
    log_test_result "$test_name" "FAIL" "login page not accessible"
    return 1
}

test_api_endpoints() {
    local test_name="API endpoints structure"
    
    # Test common Huginn API endpoints (should return 401/403 without auth)
    local endpoints=("/api/v1/agents" "/agents" "/admin")
    local found_api=false
    
    for endpoint in "${endpoints[@]}"; do
        local status_code
        if status_code=$(check_http_status "$endpoint" "401" "GET" 2>/dev/null || 
                         check_http_status "$endpoint" "403" "GET" 2>/dev/null ||
                         check_http_status "$endpoint" "302" "GET" 2>/dev/null ||
                         check_http_status "$endpoint" "200" "GET" 2>/dev/null); then
            found_api=true
            break
        fi
    done
    
    if [[ "$found_api" == "true" ]]; then
        log_test_result "$test_name" "PASS" "API endpoints responsive"
        return 0
    else
        log_test_result "$test_name" "FAIL" "API endpoints not accessible"
        return 1
    fi
}

test_rails_application() {
    local test_name="Rails application status"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check if Rails is running inside the container
    local rails_check
    if rails_check=$(docker exec "${CONTAINER_NAME}" ps aux 2>/dev/null | grep -i "rails\|ruby\|puma\|unicorn" | head -1); then
        if [[ -n "$rails_check" ]]; then
            log_test_result "$test_name" "PASS" "Rails application running"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "Rails application not detected"
    return 1
}

test_database_connectivity() {
    local test_name="database connectivity"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check if database container is running
    if docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER_NAME}$"; then
        # Test database connection from Huginn container
        local db_test
        if db_test=$(docker exec "${CONTAINER_NAME}" bash -c "
            cd /app && 
            bundle exec rails runner 'puts ActiveRecord::Base.connection.execute(\"SELECT 1\").first' 2>/dev/null
        " 2>/dev/null); then
            if echo "$db_test" | grep -q "1"; then
                log_test_result "$test_name" "PASS" "database connection working"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "database connection failed"
    return 1
}

test_agent_system() {
    local test_name="agent system availability"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Test if agent system is accessible (check for Agent model)
    local agent_test
    if agent_test=$(docker exec "${CONTAINER_NAME}" bash -c "
        cd /app && 
        bundle exec rails runner 'puts Agent.count rescue \"error\"' 2>/dev/null
    " 2>/dev/null); then
        if echo "$agent_test" | grep -q "^[0-9]\+$"; then
            local count
            count=$(echo "$agent_test" | head -1)
            log_test_result "$test_name" "PASS" "agent system working (agents: $count)"
            return 0
        elif echo "$agent_test" | grep -q "error"; then
            log_test_result "$test_name" "FAIL" "agent system error"
            return 1
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "agent system status unclear"
    return 2
}

test_worker_processes() {
    local test_name="background worker processes"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check for background job processors
    local worker_check
    if worker_check=$(docker exec "${CONTAINER_NAME}" ps aux 2>/dev/null | grep -E "job|worker|delayed|sidekiq"); then
        if [[ -n "$worker_check" ]]; then
            log_test_result "$test_name" "PASS" "background workers detected"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "background workers not detected (may use different system)"
    return 2
}

test_container_health() {
    local test_name="Docker container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check main container
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        local container_status
        container_status=$(docker inspect "${CONTAINER_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
        if [[ "$container_status" == "running" ]]; then
            log_test_result "$test_name" "PASS" "containers running"
            return 0
        else
            log_test_result "$test_name" "FAIL" "main container status: $container_status"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "main container not found"
        return 1
    fi
}

test_log_output() {
    local test_name="application log health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check container logs for critical errors
    local logs_output
    if logs_output=$(docker logs "${CONTAINER_NAME}" --tail 20 2>&1); then
        # Look for critical error patterns
        if echo "$logs_output" | grep -qi "fatal\|critical\|exception.*error\|database.*error"; then
            log_test_result "$test_name" "FAIL" "critical errors detected in logs"
            return 1
        elif echo "$logs_output" | grep -qi "started\|running\|listening\|ready"; then
            log_test_result "$test_name" "PASS" "healthy log output"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "log status unclear"
    return 2
}

#######################################
# SERVICE-SPECIFIC VERBOSE INFO
#######################################

show_verbose_info() {
    echo
    echo "Huginn Information:"
    echo "  Web Interface: $BASE_URL"
    echo "  Admin Credentials:"
    echo "    Username: ${DEFAULT_ADMIN_USERNAME}"
    echo "    Password: ${DEFAULT_ADMIN_PASSWORD}"
    echo "  API Endpoints:"
    echo "    - Login: GET $BASE_URL/users/sign_in"
    echo "    - Agents: GET $BASE_URL/agents (requires auth)"
    echo "    - Admin: GET $BASE_URL/admin (requires auth)"
    echo "  Containers:"
    echo "    Main: ${CONTAINER_NAME}"
    echo "    Database: ${DB_CONTAINER_NAME}"
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register Huginn-specific tests
register_tests \
    "test_web_interface" \
    "test_login_page" \
    "test_api_endpoints" \
    "test_rails_application" \
    "test_database_connectivity" \
    "test_agent_system" \
    "test_worker_processes" \
    "test_container_health" \
    "test_log_output"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi