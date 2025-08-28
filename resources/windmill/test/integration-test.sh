#!/usr/bin/env bash
# Windmill Integration Test
# Tests real Windmill workflow automation functionality
# Tests web interface, API endpoints, workers, and workflows

set -euo pipefail

# Setup APP_ROOT with cached pattern
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Source var.sh using APP_ROOT
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source shared integration test library (if it exists)
if [[ -f "${var_TEST_DIR}/lib/integration-test-lib.sh" ]]; then
    # shellcheck disable=SC1091
    source "${var_TEST_DIR}/lib/integration-test-lib.sh"
else
    # Simple fallback implementation for basic functions if library doesn't exist
    log_test_result() {
        local test_name="$1"
        local result="$2"
        local message="$3"
        echo "[$result] $test_name: $message"
    }
    
    register_tests() {
        # Simple no-op for fallback
        :
    }
    
    register_standard_interface_tests() {
        # Simple no-op for fallback
        :
    }
    
    integration_test_main() {
        echo "Integration test library not available"
        exit 1
    }
    
    make_api_request() {
        local endpoint="$1"
        local method="${2:-GET}"
        local timeout="${3:-10}"
        curl -s --max-time "$timeout" -X "$method" "$BASE_URL$endpoint"
    }
    
    check_service_available() {
        local base_url="$1"
        local timeout="${2:-10}"
        local endpoint="${3:-/}"
        curl -s --max-time "$timeout" "$base_url$endpoint" >/dev/null 2>&1
    }
    
    check_http_status() {
        local endpoint="$1"
        local expected_status="$2"
        local method="${3:-GET}"
        local status
        status=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE_URL$endpoint")
        [[ "$status" == "$expected_status" ]]
    }
fi

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Source common resources using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/config/defaults.sh"
windmill::export_config

# Override library defaults with Windmill-specific settings
# shellcheck disable=SC2034
SERVICE_NAME="windmill"
BASE_URL="${WINDMILL_BASE_URL:-http://localhost:8000}"
# shellcheck disable=SC2034
HEALTH_ENDPOINT="/api/health"
# shellcheck disable=SC2034
REQUIRED_TOOLS=("curl" "jq" "docker")
# shellcheck disable=SC2034
SERVICE_METADATA=(
    "Port: ${WINDMILL_SERVER_PORT:-8000}"
    "Server Container: ${WINDMILL_SERVER_CONTAINER:-windmill-vrooli-server}"
    "Worker Container: ${WINDMILL_WORKER_CONTAINER:-windmill-vrooli-worker}"
    "Database Container: ${WINDMILL_DB_CONTAINER_NAME:-windmill-vrooli-db}"
    "Workers: ${WINDMILL_WORKER_REPLICAS:-3}"
)

#######################################
# WINDMILL-SPECIFIC TEST FUNCTIONS
#######################################

test_web_interface() {
    local test_name="web interface accessibility"
    
    local response
    if response=$(make_api_request "/" "GET" 10); then
        if echo "$response" | grep -qi "windmill\|<!DOCTYPE html>\|login\|sign"; then
            log_test_result "$test_name" "PASS" "web interface accessible"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "web interface not accessible"
    return 1
}

test_api_version() {
    local test_name="API version endpoint"
    
    local response
    if response=$(make_api_request "/api/version" "GET" 5); then
        if [[ -n "$response" ]] && echo "$response" | grep -q "."; then
            log_test_result "$test_name" "PASS" "version: $response"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "version endpoint not responding"
    return 1
}

test_health_endpoint() {
    local test_name="health endpoint"
    
    if check_service_available "$BASE_URL" 5 "/api/health"; then
        log_test_result "$test_name" "PASS" "health endpoint responding"
        return 0
    else
        log_test_result "$test_name" "FAIL" "health endpoint not responding"
        return 1
    fi
}

test_openapi_spec() {
    local test_name="OpenAPI specification"
    
    local response
    if response=$(make_api_request "/api/openapi.yaml" "GET" 5); then
        if echo "$response" | grep -qi "openapi\|swagger\|paths:"; then
            log_test_result "$test_name" "PASS" "OpenAPI spec available"
            return 0
        fi
    fi
    
    # Try alternative endpoint
    if response=$(make_api_request "/openapi.html" "GET" 5); then
        if echo "$response" | grep -qi "swagger\|openapi\|<!DOCTYPE html>"; then
            log_test_result "$test_name" "PASS" "Swagger UI available"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "OpenAPI documentation not available"
    return 2
}

test_authentication_endpoints() {
    local test_name="authentication endpoints"
    
    # Test endpoints that should require authentication (should return 401/403)
    local endpoints=("/api/w/list" "/api/users/whoami")
    local auth_working=false
    
    for endpoint in "${endpoints[@]}"; do
        local status_code
        # shellcheck disable=SC2034
        if status_code=$(check_http_status "$endpoint" "401" "GET" 2>/dev/null ||
                         check_http_status "$endpoint" "403" "GET" 2>/dev/null); then
            auth_working=true
            break
        fi
    done
    
    if [[ "$auth_working" == "true" ]]; then
        log_test_result "$test_name" "PASS" "authentication system working"
        return 0
    else
        log_test_result "$test_name" "FAIL" "authentication endpoints not responding properly"
        return 1
    fi
}

test_workers_endpoint() {
    local test_name="workers endpoint"
    
    # Workers endpoint might require admin auth, but should respond with 401/403
    local status_code
    # shellcheck disable=SC2034
    if status_code=$(check_http_status "/api/workers/list" "401" "GET" 2>/dev/null ||
                     check_http_status "/api/workers/list" "403" "GET" 2>/dev/null ||
                     check_http_status "/api/workers/list" "200" "GET" 2>/dev/null); then
        log_test_result "$test_name" "PASS" "workers endpoint responsive"
        return 0
    fi
    
    log_test_result "$test_name" "SKIP" "workers endpoint not accessible"
    return 2
}

test_server_container() {
    local test_name="server container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    if docker ps --format '{{.Names}}' | grep -q "${WINDMILL_SERVER_CONTAINER}"; then
        local container_status
        container_status=$(docker inspect "${WINDMILL_SERVER_CONTAINER}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
        if [[ "$container_status" == "running" ]]; then
            log_test_result "$test_name" "PASS" "server container running"
            return 0
        else
            log_test_result "$test_name" "FAIL" "server container status: $container_status"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "server container not found"
        return 1
    fi
}

test_worker_containers() {
    local test_name="worker containers health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check for worker containers (might have different naming patterns)
    local worker_count
    worker_count=$(docker ps --format '{{.Names}}' | grep -c "worker\|${WINDMILL_WORKER_CONTAINER}" || echo "0")
    
    if [[ $worker_count -gt 0 ]]; then
        log_test_result "$test_name" "PASS" "worker containers running (count: $worker_count)"
        return 0
    else
        log_test_result "$test_name" "SKIP" "no worker containers detected"
        return 2
    fi
}

test_database_container() {
    local test_name="database container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    if docker ps --format '{{.Names}}' | grep -q "${WINDMILL_DB_CONTAINER_NAME}"; then
        local container_status
        container_status=$(docker inspect "${WINDMILL_DB_CONTAINER_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
        if [[ "$container_status" == "running" ]]; then
            log_test_result "$test_name" "PASS" "database container running"
            return 0
        else
            log_test_result "$test_name" "FAIL" "database container status: $container_status"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "database container not found"
        return 1
    fi
}

test_compose_stack() {
    local test_name="Docker Compose stack health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check if compose project exists
    local project_containers
    project_containers=$(docker ps --filter "label=com.docker.compose.project=${WINDMILL_PROJECT_NAME}" --format '{{.Names}}' | wc -l)
    
    if [[ $project_containers -gt 0 ]]; then
        log_test_result "$test_name" "PASS" "compose stack running (containers: $project_containers)"
        return 0
    else
        log_test_result "$test_name" "SKIP" "compose stack not detected"
        return 2
    fi
}

test_log_output() {
    local test_name="application log health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check server container logs
    local logs_output
    if logs_output=$(docker logs "${WINDMILL_SERVER_CONTAINER}" --tail 15 2>&1 2>/dev/null || true); then
        # Look for startup success patterns
        if echo "$logs_output" | grep -qi "server.*start\|listening\|ready\|running"; then
            log_test_result "$test_name" "PASS" "healthy server logs"
            return 0
        elif echo "$logs_output" | grep -qi "error\|panic\|fatal\|exception"; then
            log_test_result "$test_name" "FAIL" "errors detected in server logs"
            return 1
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
    echo "Windmill Information:"
    echo "  Web Interface: $BASE_URL"
    echo "  Admin Credentials:"
    echo "    Email: ${WINDMILL_SUPERADMIN_EMAIL}"
    echo "    Password: ${WINDMILL_SUPERADMIN_PASSWORD}"
    echo "  API Endpoints:"
    echo "    - Version: GET $BASE_URL/api/version"
    echo "    - Health: GET $BASE_URL/api/health"
    echo "    - OpenAPI: GET $BASE_URL/api/openapi.yaml"
    echo "    - Swagger UI: GET $BASE_URL/openapi.html"
    echo "  Containers:"
    echo "    Server: ${WINDMILL_SERVER_CONTAINER}"
    echo "    Workers: ${WINDMILL_WORKER_CONTAINER} (${WINDMILL_WORKER_REPLICAS} replicas)"
    echo "    Database: ${WINDMILL_DB_CONTAINER_NAME}"
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register Windmill-specific tests
register_tests \
    "test_web_interface" \
    "test_api_version" \
    "test_health_endpoint" \
    "test_openapi_spec" \
    "test_authentication_endpoints" \
    "test_workers_endpoint" \
    "test_server_container" \
    "test_worker_containers" \
    "test_database_container" \
    "test_compose_stack" \
    "test_log_output"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi