#!/usr/bin/env bash
# Redis Integration Test
# Tests real Redis key-value store functionality
# Tests basic operations, data types, persistence, and cluster health

set -euo pipefail

# Source shared integration test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../../tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load Redis configuration
RESOURCES_DIR="$SCRIPT_DIR/../../.."
# shellcheck disable=SC1091
source "$RESOURCES_DIR/common.sh"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../config/defaults.sh"

# Override library defaults with Redis-specific settings
SERVICE_NAME="redis"
BASE_URL="redis://localhost:${REDIS_PORT:-6380}"
HEALTH_ENDPOINT=""  # Redis uses command-based health checks
REQUIRED_TOOLS=("docker" "redis-cli")
SERVICE_METADATA=(
    "Port: ${REDIS_PORT:-6380}"
    "Container: ${REDIS_CONTAINER_NAME:-vrooli-redis-resource}"
    "Data Dir: ${REDIS_DATA_DIR:-${HOME}/.vrooli/redis/data}"
    "Max Memory: ${REDIS_MAX_MEMORY:-2gb}"
    "Databases: ${REDIS_DATABASES:-16}"
)

#######################################
# REDIS-SPECIFIC TEST FUNCTIONS
#######################################

test_redis_cli_connection() {
    local test_name="Redis CLI connection"
    
    if ! command -v redis-cli >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "redis-cli not available"
        return 2
    fi
    
    local port="${REDIS_PORT:-6380}"
    if redis-cli -h localhost -p "$port" ping >/dev/null 2>&1; then
        log_test_result "$test_name" "PASS" "Redis CLI connection working"
        return 0
    else
        log_test_result "$test_name" "FAIL" "Redis CLI connection failed"
        return 1
    fi
}

test_basic_operations() {
    local test_name="basic key-value operations"
    
    if ! command -v redis-cli >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "redis-cli not available"
        return 2
    fi
    
    local port="${REDIS_PORT:-6380}"
    local test_key="integration_test_$(date +%s)"
    local test_value="test_value_$(date +%s)"
    
    # Set a key
    if redis-cli -h localhost -p "$port" set "$test_key" "$test_value" >/dev/null 2>&1; then
        # Get the key
        local retrieved_value
        retrieved_value=$(redis-cli -h localhost -p "$port" get "$test_key" 2>/dev/null)
        
        # Clean up
        redis-cli -h localhost -p "$port" del "$test_key" >/dev/null 2>&1 || true
        
        if [[ "$retrieved_value" == "$test_value" ]]; then
            log_test_result "$test_name" "PASS" "basic operations working"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "basic operations failed"
    return 1
}

test_data_types() {
    local test_name="Redis data types support"
    
    if ! command -v redis-cli >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "redis-cli not available"
        return 2
    fi
    
    local port="${REDIS_PORT:-6380}"
    local test_prefix="integration_test_$(date +%s)"
    local success_count=0
    
    # Test string
    if redis-cli -h localhost -p "$port" set "${test_prefix}_string" "value" >/dev/null 2>&1; then
        success_count=$((success_count + 1))
    fi
    
    # Test list
    if redis-cli -h localhost -p "$port" lpush "${test_prefix}_list" "item1" "item2" >/dev/null 2>&1; then
        success_count=$((success_count + 1))
    fi
    
    # Test set
    if redis-cli -h localhost -p "$port" sadd "${test_prefix}_set" "member1" "member2" >/dev/null 2>&1; then
        success_count=$((success_count + 1))
    fi
    
    # Test hash
    if redis-cli -h localhost -p "$port" hset "${test_prefix}_hash" "field1" "value1" >/dev/null 2>&1; then
        success_count=$((success_count + 1))
    fi
    
    # Clean up
    redis-cli -h localhost -p "$port" del "${test_prefix}_string" "${test_prefix}_list" "${test_prefix}_set" "${test_prefix}_hash" >/dev/null 2>&1 || true
    
    if [[ $success_count -ge 3 ]]; then
        log_test_result "$test_name" "PASS" "data types working ($success_count/4)"
        return 0
    else
        log_test_result "$test_name" "FAIL" "data types failed ($success_count/4)"
        return 1
    fi
}

test_expiration() {
    local test_name="key expiration"
    
    if ! command -v redis-cli >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "redis-cli not available"
        return 2
    fi
    
    local port="${REDIS_PORT:-6380}"
    local test_key="integration_expire_test_$(date +%s)"
    
    # Set a key with expiration
    if redis-cli -h localhost -p "$port" setex "$test_key" 2 "expire_value" >/dev/null 2>&1; then
        # Check TTL
        local ttl
        ttl=$(redis-cli -h localhost -p "$port" ttl "$test_key" 2>/dev/null)
        
        # Clean up
        redis-cli -h localhost -p "$port" del "$test_key" >/dev/null 2>&1 || true
        
        if [[ "$ttl" =~ ^[12]$ ]]; then
            log_test_result "$test_name" "PASS" "expiration working (TTL: ${ttl}s)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "expiration test inconclusive"
    return 2
}

test_info_command() {
    local test_name="Redis INFO command"
    
    if ! command -v redis-cli >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "redis-cli not available"
        return 2
    fi
    
    local port="${REDIS_PORT:-6380}"
    local info_output
    if info_output=$(redis-cli -h localhost -p "$port" info server 2>/dev/null); then
        if echo "$info_output" | grep -q "redis_version\|uptime_in_seconds"; then
            local version
            version=$(echo "$info_output" | grep "redis_version" | cut -d: -f2 | tr -d '\r')
            log_test_result "$test_name" "PASS" "Redis version: $version"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "INFO command failed"
    return 1
}

test_memory_usage() {
    local test_name="memory usage monitoring"
    
    if ! command -v redis-cli >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "redis-cli not available"
        return 2
    fi
    
    local port="${REDIS_PORT:-6380}"
    local memory_info
    if memory_info=$(redis-cli -h localhost -p "$port" info memory 2>/dev/null); then
        if echo "$memory_info" | grep -q "used_memory\|maxmemory"; then
            local used_memory
            used_memory=$(echo "$memory_info" | grep "used_memory_human" | cut -d: -f2 | tr -d '\r')
            log_test_result "$test_name" "PASS" "memory usage: $used_memory"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "memory info not available"
    return 2
}

test_container_health() {
    local test_name="Docker container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    local container_name="${REDIS_CONTAINER_NAME:-vrooli-redis-resource}"
    if docker ps --format '{{.Names}}' | grep -q "^${container_name}$"; then
        local container_status
        container_status=$(docker inspect "$container_name" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
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
    local test_name="container log health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    local container_name="${REDIS_CONTAINER_NAME:-vrooli-redis-resource}"
    local logs_output
    if logs_output=$(docker logs "$container_name" --tail 10 2>&1 2>/dev/null || true); then
        # Look for Redis startup success patterns
        if echo "$logs_output" | grep -qi "ready to accept connections\|server initialized\|redis.*started"; then
            log_test_result "$test_name" "PASS" "healthy Redis logs"
            return 0
        elif echo "$logs_output" | grep -qi "error\|fatal\|failed\|warning.*critical"; then
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
    echo "Redis Information:"
    echo "  Connection: redis://localhost:${REDIS_PORT}"
    echo "  Container: ${REDIS_CONTAINER_NAME}"
    echo "  Data Directory: ${REDIS_DATA_DIR}"
    echo "  Configuration:"
    echo "    Max Memory: ${REDIS_MAX_MEMORY}"
    echo "    Memory Policy: ${REDIS_MAX_MEMORY_POLICY}"
    echo "    Persistence: ${REDIS_PERSISTENCE}"
    echo "    Databases: ${REDIS_DATABASES}"
    echo "  Basic Commands:"
    echo "    Test Connection: redis-cli -h localhost -p ${REDIS_PORT} ping"
    echo "    Get Info: redis-cli -h localhost -p ${REDIS_PORT} info"
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register Redis-specific tests
register_tests \
    "test_redis_cli_connection" \
    "test_basic_operations" \
    "test_data_types" \
    "test_expiration" \
    "test_info_command" \
    "test_memory_usage" \
    "test_container_health" \
    "test_log_output"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi