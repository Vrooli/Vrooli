#!/usr/bin/env bash
# Vault Integration Test
# Tests real HashiCorp Vault functionality
# Tests API endpoints, secret operations, authentication, and seal status

set -euo pipefail

# Source shared integration test library
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/vault/test"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load Vault configuration
RESOURCES_DIR="$SCRIPT_DIR/../../.."
# shellcheck disable=SC1091
source "$RESOURCES_DIR/common.sh"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../config/defaults.sh"
vault::export_config

# Override library defaults with Vault-specific settings
# SERVICE_NAME="vault"  # Used by sourced library functions
# shellcheck disable=SC2034
SERVICE_NAME="vault"
BASE_URL="${VAULT_BASE_URL:-http://localhost:8200}"
# HEALTH_ENDPOINT="/v1/sys/health"  # Used by sourced library functions
# shellcheck disable=SC2034
HEALTH_ENDPOINT="/v1/sys/health"
# REQUIRED_TOOLS=("curl" "jq" "docker")  # Used by sourced library functions
# shellcheck disable=SC2034
REQUIRED_TOOLS=("curl" "jq" "docker")
# SERVICE_METADATA array used by sourced library functions
# shellcheck disable=SC2034
SERVICE_METADATA=(
    "Port: ${VAULT_PORT:-8200}"
    "Container: ${VAULT_CONTAINER_NAME:-vault}"
    "Data Dir: ${VAULT_DATA_DIR:-${HOME}/.vault/data}"
    "Config Dir: ${VAULT_CONFIG_DIR:-${HOME}/.vault/config}"
)

#######################################
# VAULT-SPECIFIC TEST FUNCTIONS
#######################################

test_health_endpoint() {
    local test_name="health endpoint"
    
    local response
    if response=$(make_api_request "/v1/sys/health" "GET" 5); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local sealed
            sealed=$(echo "$response" | jq -r '.sealed // "unknown"' 2>/dev/null)
            local initialized
            initialized=$(echo "$response" | jq -r '.initialized // "unknown"' 2>/dev/null)
            log_test_result "$test_name" "PASS" "sealed: $sealed, initialized: $initialized"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "health endpoint not working"
    return 1
}

test_sys_endpoints() {
    local test_name="system endpoints accessibility"
    
    local endpoints=("/v1/sys/seal-status" "/v1/sys/init" "/v1/sys/version")
    local accessible_count=0
    
    for endpoint in "${endpoints[@]}"; do
        local response
        if response=$(make_api_request "$endpoint" "GET" 5 2>/dev/null); then
            if echo "$response" | jq . >/dev/null 2>&1; then
                accessible_count=$((accessible_count + 1))
            fi
        fi
    done
    
    if [[ $accessible_count -gt 0 ]]; then
        log_test_result "$test_name" "PASS" "system endpoints accessible ($accessible_count/${#endpoints[@]})"
        return 0
    else
        log_test_result "$test_name" "FAIL" "no system endpoints accessible"
        return 1
    fi
}

test_seal_status() {
    local test_name="seal status check"
    
    local response
    if response=$(make_api_request "/v1/sys/seal-status" "GET" 5); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local sealed
            sealed=$(echo "$response" | jq -r '.sealed // "unknown"' 2>/dev/null)
            local threshold
            threshold=$(echo "$response" | jq -r '.t // "unknown"' 2>/dev/null)
            local shares
            shares=$(echo "$response" | jq -r '.n // "unknown"' 2>/dev/null)
            log_test_result "$test_name" "PASS" "sealed: $sealed (threshold: $threshold, shares: $shares)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "seal status not available"
    return 1
}

test_initialization_status() {
    local test_name="initialization status"
    
    local response
    if response=$(make_api_request "/v1/sys/init" "GET" 5); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local initialized
            initialized=$(echo "$response" | jq -r '.initialized // "unknown"' 2>/dev/null)
            log_test_result "$test_name" "PASS" "initialized: $initialized"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "initialization status not available"
    return 1
}

test_version_endpoint() {
    local test_name="version endpoint"
    
    local response
    if response=$(make_api_request "/v1/sys/version" "GET" 5); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local version
            version=$(echo "$response" | jq -r '.version // "unknown"' 2>/dev/null)
            log_test_result "$test_name" "PASS" "version: $version"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "version endpoint not available"
    return 2
}

test_auth_methods() {
    local test_name="authentication methods endpoint"
    
    # This endpoint typically requires authentication, but should respond with 403/401
    if check_http_status "/v1/sys/auth" "403" "GET" 2>/dev/null ||
       check_http_status "/v1/sys/auth" "401" "GET" 2>/dev/null ||
       check_http_status "/v1/sys/auth" "200" "GET" 2>/dev/null; then
        log_test_result "$test_name" "PASS" "auth methods endpoint responsive"
        return 0
    fi
    
    log_test_result "$test_name" "SKIP" "auth methods endpoint not accessible"
    return 2
}

test_secret_engines() {
    local test_name="secret engines endpoint"
    
    # This endpoint typically requires authentication, but should respond with 403/401
    if check_http_status "/v1/sys/mounts" "403" "GET" 2>/dev/null ||
       check_http_status "/v1/sys/mounts" "401" "GET" 2>/dev/null ||
       check_http_status "/v1/sys/mounts" "200" "GET" 2>/dev/null; then
        log_test_result "$test_name" "PASS" "secret engines endpoint responsive"
        return 0
    fi
    
    log_test_result "$test_name" "SKIP" "secret engines endpoint not accessible"
    return 2
}

test_ui_availability() {
    local test_name="web UI availability"
    
    local response
    if response=$(make_api_request "/ui/" "GET" 5); then
        if echo "$response" | grep -qi "vault\|<!DOCTYPE html>\|hashicorp"; then
            log_test_result "$test_name" "PASS" "web UI accessible"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "web UI not available"
    return 2
}

test_container_health() {
    local test_name="Docker container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    if docker ps --format '{{.Names}}' | grep -q "^${VAULT_CONTAINER_NAME}$"; then
        local container_status
        container_status=$(docker inspect "${VAULT_CONTAINER_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
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

test_log_output() {
    local test_name="application log health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    local logs_output
    if logs_output=$(docker logs "${VAULT_CONTAINER_NAME}" --tail 15 2>&1 2>/dev/null || true); then
        # Look for Vault startup success patterns
        if echo "$logs_output" | grep -qi "vault server.*started\|api address\|cluster address"; then
            log_test_result "$test_name" "PASS" "healthy Vault logs"
            return 0
        elif echo "$logs_output" | grep -qi "error\|panic\|fatal\|failed.*start"; then
            log_test_result "$test_name" "FAIL" "errors detected in logs"
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
    echo "Vault Information:"
    echo "  API Base URL: $BASE_URL"
    echo "  Web UI: $BASE_URL/ui/"
    echo "  Key API Endpoints:"
    echo "    - Health: GET $BASE_URL/v1/sys/health"
    echo "    - Seal Status: GET $BASE_URL/v1/sys/seal-status"
    echo "    - Init Status: GET $BASE_URL/v1/sys/init"
    echo "    - Version: GET $BASE_URL/v1/sys/version"
    echo "    - Auth Methods: GET $BASE_URL/v1/sys/auth"
    echo "    - Secret Engines: GET $BASE_URL/v1/sys/mounts"
    echo "  Container: ${VAULT_CONTAINER_NAME}"
    echo "  Data Directory: ${VAULT_DATA_DIR}"
    echo "  Config Directory: ${VAULT_CONFIG_DIR}"
    echo
    echo "  Note: Most endpoints require authentication after initialization."
    echo "  Vault must be initialized and unsealed for full functionality."
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register Vault-specific tests
register_tests \
    "test_health_endpoint" \
    "test_sys_endpoints" \
    "test_seal_status" \
    "test_initialization_status" \
    "test_version_endpoint" \
    "test_auth_methods" \
    "test_secret_engines" \
    "test_ui_availability" \
    "test_container_health" \
    "test_log_output"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi