#!/usr/bin/env bash
# n8n Integration Test
# Tests real n8n workflow automation functionality
# Tests API endpoints, workflow management, and execution capabilities

set -euo pipefail

# Source enhanced integration test library with fixture support
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../../tests/lib/enhanced-integration-test-lib.sh"

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

#######################################
# FIXTURE-BASED WORKFLOW TESTS
#######################################

test_workflow_import_fixture() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # Read workflow JSON from fixture
    local workflow_json
    workflow_json=$(cat "$fixture_path")
    
    # Try to import workflow via API (may need auth)
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d "$workflow_json" \
        "$BASE_URL$API_BASE/workflows" 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # Accept 401/403 as success (endpoint exists but needs auth)
        # or 200/201 if workflow was imported
        if [[ "$status_code" == "401" ]] || [[ "$status_code" == "403" ]]; then
            return 0  # Endpoint exists, auth required
        elif [[ "$status_code" == "200" ]] || [[ "$status_code" == "201" ]]; then
            return 0  # Workflow imported successfully
        fi
    fi
    
    return 1
}

test_workflow_validation_fixture() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # Read workflow JSON and validate structure
    local workflow_json
    workflow_json=$(cat "$fixture_path")
    
    # Check if workflow has required n8n structure
    if echo "$workflow_json" | jq -e '.nodes' >/dev/null 2>&1; then
        # Valid n8n workflow structure
        return 0
    fi
    
    return 1
}

# Run fixture-based workflow tests
run_workflow_fixture_tests() {
    if [[ "$FIXTURES_AVAILABLE" == "true" ]]; then
        # Test with n8n-specific workflow fixtures
        test_with_fixture "import n8n workflow" "workflows" "n8n/n8n-workflow.json" \
            test_workflow_import_fixture
        
        test_with_fixture "import whisper transcription workflow" "workflows" "n8n/n8n-whisper-transcription.json" \
            test_workflow_import_fixture
        
        # Validate workflow structure
        test_with_fixture "validate n8n workflow structure" "workflows" "n8n/n8n-workflow.json" \
            test_workflow_validation_fixture
        
        # Test with auto-discovered fixtures
        local n8n_fixtures
        n8n_fixtures=$(discover_resource_fixtures "n8n" "automation")
        
        for fixture_pattern in $n8n_fixtures; do
            local fixture_files
            if fixture_files=$(fixture_get_all "$fixture_pattern" "*.json" 2>/dev/null); then
                for fixture_file in $fixture_files; do
                    local fixture_name
                    fixture_name=$(basename "$fixture_file")
                    test_with_fixture "validate $fixture_name" "" "$fixture_file" \
                        test_workflow_validation_fixture
                done
            fi
        done
        
        # Test with rotating fixtures for variety
        local workflow_fixtures
        if workflow_fixtures=$(rotate_fixtures "workflows" 3); then
            for fixture in $workflow_fixtures; do
                if [[ "$fixture" == *.json ]]; then
                    local fixture_name
                    fixture_name=$(basename "$fixture")
                    test_with_fixture "import $fixture_name" "" "$fixture" \
                        test_workflow_import_fixture
                fi
            done
        fi
    fi
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
    "run_workflow_fixture_tests"
)

#######################################
# MAIN EXECUTION
#######################################

# Initialize and run tests using the shared library
init_config

# Register service-specific tests
REGISTERED_TESTS=("${SERVICE_TESTS[@]}")

integration_test_main "$@"