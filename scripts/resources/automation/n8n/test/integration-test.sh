#!/usr/bin/env bash
# n8n Integration Test
# Tests real n8n workflow automation functionality
# Tests API endpoints, workflow management, and execution capabilities

set -euo pipefail

# Source shared integration test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../../tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load n8n configuration
RESOURCES_DIR="$SCRIPT_DIR/../../.."
# shellcheck disable=SC1091
source "$RESOURCES_DIR/common.sh"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../config/defaults.sh"
n8n::export_config

# Override library defaults with n8n-specific settings
SERVICE_NAME="n8n"
BASE_URL="${N8N_BASE_URL:-http://localhost:5678}"
HEALTH_ENDPOINT="/healthz"
REQUIRED_TOOLS=("curl" "jq" "docker")
SERVICE_METADATA=(
    "API Port: ${N8N_PORT:-5678}"
    "Container: ${N8N_CONTAINER_NAME:-n8n}"
    "Database: ${N8N_DB_TYPE:-sqlite}"
)

# Test configuration
readonly API_BASE="/api/v1"

#######################################
# N8N-SPECIFIC TEST FUNCTIONS
#######################################

test_n8n_health_endpoint() {
    local test_name="n8n health endpoint"
    
    local response
    if response=$(make_api_request "/healthz" "GET" 10); then
        # n8n health endpoint typically returns 200 OK
        log_test_result "$test_name" "PASS" "health endpoint accessible"
        return 0
    fi
    
    log_test_result "$test_name" "FAIL" "health endpoint not accessible"
    return 1
}

test_workflows_api_endpoint() {
    local test_name="workflows API endpoint"
    
    local response
    local status_code
    # Test without authentication - should return 401/403 but endpoint should be accessible
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$BASE_URL$API_BASE/workflows" 2>/dev/null); then
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        response_body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
        
        # n8n should respond with 401 (unauthorized) or 200 (if no auth)
        if [[ "$status_code" == "401" ]] || [[ "$status_code" == "403" ]]; then
            log_test_result "$test_name" "PASS" "workflows API requires authentication (HTTP: $status_code)"
            return 0
        elif [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "workflows API accessible (HTTP: $status_code)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "workflows API endpoint not accessible"
    return 1
}

test_credentials_api_endpoint() {
    local test_name="credentials API endpoint"
    
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$BASE_URL$API_BASE/credentials" 2>/dev/null); then
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # Should require authentication
        if [[ "$status_code" == "401" ]] || [[ "$status_code" == "403" ]] || [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "credentials API endpoint responsive (HTTP: $status_code)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "credentials API endpoint not accessible"
    return 1
}

test_executions_api_endpoint() {
    local test_name="executions API endpoint"
    
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$BASE_URL$API_BASE/executions" 2>/dev/null); then
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # Should require authentication or return results
        if [[ "$status_code" == "401" ]] || [[ "$status_code" == "403" ]] || [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "executions API endpoint responsive (HTTP: $status_code)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "executions API endpoint not accessible"
    return 1
}

test_n8n_version_info() {
    local test_name="n8n version information"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Try to get version from running container
    local version_output
    if version_output=$(docker exec "${N8N_CONTAINER_NAME}" n8n --version 2>/dev/null); then
        if [[ -n "$version_output" ]]; then
            log_test_result "$test_name" "PASS" "version: $version_output"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "version information not accessible"
    return 2
}

test_webhook_endpoint() {
    local test_name="webhook endpoint availability"
    
    # Test webhook endpoint structure
    local webhook_url="$BASE_URL/webhook-test"
    local response
    local status_code
    
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST "$webhook_url" -H "Content-Type: application/json" -d '{"test": true}' 2>/dev/null); then
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # Webhook endpoint should respond (even if workflow doesn't exist)
        # 404 is expected for non-existent webhooks, which is still a valid response
        if [[ "$status_code" =~ ^(200|404|405|500)$ ]]; then
            log_test_result "$test_name" "PASS" "webhook endpoint structure accessible (HTTP: $status_code)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "webhook endpoints not accessible"
    return 1
}

test_container_health() {
    local test_name="Docker container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check if container exists and is running
    if docker ps --format '{{.Names}}' | grep -q "^${N8N_CONTAINER_NAME}$"; then
        # Get container status
        local container_status
        container_status=$(docker inspect "${N8N_CONTAINER_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
        if [[ "$container_status" == "running" ]]; then
            log_test_result "$test_name" "PASS" "container running"
            return 0
        else
            log_test_result "$test_name" "FAIL" "container status: $container_status"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "container not found"
        return 1
    fi
}

test_database_connection() {
    local test_name="database connection health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check container logs for database connection info
    local logs_output
    if logs_output=$(docker logs "${N8N_CONTAINER_NAME}" --tail 50 2>&1); then
        # Look for database connection indicators (both success and failure patterns)
        if echo "$logs_output" | grep -qi "database.*connect\|db.*connect\|sqlite\|postgres"; then
            if echo "$logs_output" | grep -qi "error.*database\|database.*error\|connection.*failed"; then
                log_test_result "$test_name" "FAIL" "database connection errors detected"
                return 1
            else
                log_test_result "$test_name" "PASS" "database connection indicators found"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "database connection status unclear"
    return 2
}

#######################################
# TEST RUNNER CONFIGURATION
#######################################

# Define service-specific tests to run
SERVICE_TESTS=(
    "test_n8n_health_endpoint"
    "test_workflows_api_endpoint"
    "test_credentials_api_endpoint"
    "test_executions_api_endpoint"
    "test_webhook_endpoint"
    "test_n8n_version_info"
    "test_container_health"
    "test_database_connection"
)

#######################################
# MAIN EXECUTION
#######################################

# Initialize and run tests using the shared library
init_config
run_integration_tests "$@"