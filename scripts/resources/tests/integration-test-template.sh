#!/usr/bin/env bash
# RESOURCE_NAME Integration Test Template
# Tests real RESOURCE_NAME service functionality
# 
# This template should be customized for each resource:
# 1. Replace RESOURCE_NAME with actual resource name
# 2. Update SERVICE_SPECIFIC_CONFIGURATION section
# 3. Implement service-specific test functions
# 4. Register tests using register_tests function
# 5. Ensure script calls integration_test_main "$@" at the end

set -euo pipefail

_HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${_HERE}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_DIR}/resources/tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load resource configuration (optional - only if resource has config/defaults.sh)
# # shellcheck disable=SC1091
# source "${var_RESOURCES_COMMON_FILE}"
# # shellcheck disable=SC1091
# source "${integration_test_template::SCRIPT_DIR}/../config/defaults.sh"
# resource_name::export_config

# Override library defaults with service-specific settings
SERVICE_NAME="RESOURCE_NAME"
BASE_URL="${RESOURCE_BASE_URL:-http://localhost:PORT}"
HEALTH_ENDPOINT="/health"  # or service-specific health endpoint
REQUIRED_TOOLS=("curl" "jq")  # add any additional required tools
SERVICE_METADATA=(
    "API Port: ${RESOURCE_PORT:-PORT}"
    "Container: ${RESOURCE_CONTAINER_NAME:-RESOURCE_NAME}"
    # Add any additional metadata
)

# Test configuration constants
readonly TEST_TIMEOUT="${RESOURCE_TEST_TIMEOUT:-30}"

#######################################
# RESOURCE-SPECIFIC TEST FUNCTIONS
#######################################

test_basic_health() {
    local test_name="basic health check"
    
    local response
    if response=$(make_api_request "$HEALTH_ENDPOINT" "GET" 10); then
        log_test_result "$test_name" "PASS"
        return 0
    fi
    
    log_test_result "$test_name" "FAIL" "health endpoint not accessible"
    return 1
}

test_service_version() {
    local test_name="service version information"
    
    # Try common version endpoints
    local version_endpoints=("/version" "/api/version" "/v1/version")
    
    for endpoint in "${version_endpoints[@]}"; do
        local response
        if response=$(make_api_request "$endpoint" "GET" 5); then
            if validate_json_field "$response" ".version"; then
                local version
                version=$(echo "$response" | jq -r '.version')
                log_test_result "$test_name" "PASS" "version: $version"
                return 0
            fi
        fi
    done
    
    log_test_result "$test_name" "SKIP" "no version endpoint found"
    return 2
}

test_api_endpoints() {
    local test_name="API endpoints availability"
    
    # Test common API endpoints (customize for each resource)
    local endpoints=("/api" "/api/v1")
    local accessible_count=0
    
    for endpoint in "${endpoints[@]}"; do
        if make_api_request "$endpoint" "GET" 5 >/dev/null 2>&1; then
            accessible_count=$((accessible_count + 1))
        fi
    done
    
    if [[ $accessible_count -gt 0 ]]; then
        log_test_result "$test_name" "PASS" "$accessible_count/${#endpoints[@]} endpoints accessible"
        return 0
    fi
    
    log_test_result "$test_name" "FAIL" "no API endpoints accessible"
    return 1
}

test_docker_container_health() {
    local test_name="Docker container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    local container_name="${RESOURCE_CONTAINER_NAME:-$SERVICE_NAME}"
    
    # Check if container exists and is running
    if docker ps --format '{{.Names}}' | grep -q "^${container_name}$"; then
        local container_status
        container_status=$(docker inspect "${container_name}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
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

# Add more service-specific test functions here
# Examples:
# - test_authentication()
# - test_database_connection()
# - test_file_upload()
# - test_specific_api_operations()

#######################################
# SERVICE-SPECIFIC VERBOSE INFO
#######################################

show_verbose_info() {
    echo
    echo "$SERVICE_NAME Information:"
    echo "  Base URL: $BASE_URL"
    echo "  Health Endpoint: $HEALTH_ENDPOINT"
    echo "  API Endpoints:"
    echo "    - Health: GET $BASE_URL$HEALTH_ENDPOINT"
    echo "    - Version: GET $BASE_URL/version (if available)"
    # Add service-specific endpoint documentation
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register service-specific tests
register_tests \
    "test_basic_health" \
    "test_service_version" \
    "test_api_endpoints" \
    "test_docker_container_health"
    # Add additional test function names here

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi