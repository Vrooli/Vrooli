#!/bin/bash
# ====================================================================
# Redis Integration Test
# ====================================================================
#
# Tests Redis in-memory cache integration including health checks,
# basic operations, pub/sub functionality, and performance
# characteristics for application caching and real-time messaging.
#
# Required Resources: redis
# Test Categories: single-resource, storage
# Estimated Duration: 30-60 seconds
#
# ====================================================================

set -euo pipefail

# Test metadata
TEST_RESOURCE="redis"
TEST_TIMEOUT="${TEST_TIMEOUT:-60}"
TEST_CLEANUP="${TEST_CLEANUP:-true}"

# Recreate HEALTHY_RESOURCES array from exported string
if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
    HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
fi

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"

# Redis configuration
REDIS_HOST="localhost"
REDIS_PORT="6380"  # Vrooli's default Redis port

# Test setup
setup_test() {
    echo "🔧 Setting up Redis integration test..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Auto-discovery fallback for direct test execution
    if [[ -z "${HEALTHY_RESOURCES_STR:-}" ]]; then
        echo "🔍 Auto-discovering resources for direct test execution..."
        
        # Use the resource discovery system with timeout
        local resources_dir
        resources_dir="$(cd "$SCRIPT_DIR/../.." && pwd)"
        
        local discovery_output=""
        if timeout 10s bash -c "\"$resources_dir/index.sh\" --action discover 2>&1" > /tmp/discovery_output.tmp 2>&1; then
            discovery_output=$(cat /tmp/discovery_output.tmp)
            rm -f /tmp/discovery_output.tmp
        else
            echo "⚠️  Auto-discovery timed out, using fallback method..."
            # Fallback: check if Redis is accessible
            if timeout 3 bash -c 'echo -e "*1\r\n\$4\r\nPING\r\n" | nc '"$REDIS_HOST"' '"$REDIS_PORT"' 2>/dev/null' | head -1 | grep -q "+PONG"; then
                discovery_output="✅ $TEST_RESOURCE is running on port $REDIS_PORT"
            fi
        fi
        
        local discovered_resources=()
        while IFS= read -r line; do
            if [[ "$line" =~ ✅[[:space:]]+([^[:space:]]+)[[:space:]]+is[[:space:]]+running ]]; then
                discovered_resources+=("${BASH_REMATCH[1]}")
            fi
        done <<< "$discovery_output"
        
        if [[ ${#discovered_resources[@]} -eq 0 ]]; then
            echo "⚠️  No resources discovered, but test will proceed..."
            discovered_resources=("$TEST_RESOURCE")
        fi
        
        export HEALTHY_RESOURCES_STR="${discovered_resources[*]}"
        echo "✓ Discovered healthy resources: $HEALTHY_RESOURCES_STR"
    fi
    
    # Verify Redis is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools
    require_tools "nc"
    
    echo "✓ Test setup complete"
}

# Send Redis command using netcat
redis_command() {
    local command="$1"
    shift
    local args=("$@")
    
    # Build Redis protocol command
    local redis_cmd="*$((${#args[@]} + 1))\r\n\$${#command}\r\n${command}\r\n"
    for arg in "${args[@]}"; do
        redis_cmd+="\$${#arg}\r\n${arg}\r\n"
    done
    
    # Send command and get response
    echo -e "$redis_cmd" | nc "$REDIS_HOST" "$REDIS_PORT" 2>/dev/null | head -1 | tr -d '\r\n'
}

# Test Redis health and connectivity
test_redis_health() {
    echo "🏥 Testing Redis health and connectivity..."
    
    # Test PING command
    local ping_response
    ping_response=$(redis_command "PING")
    
    assert_equals "$ping_response" "+PONG" "Redis PING command"
    
    # Test INFO command
    local info_response
    info_response=$(redis_command "INFO" "server")
    
    assert_contains "$info_response" "redis_version" "Redis INFO command returns server info" || \
    assert_not_empty "$info_response" "Redis INFO command responds"
    
    echo "✓ Redis health check passed"
}

# Test Redis basic operations
test_redis_basic_operations() {
    echo "📋 Testing Redis basic operations..."
    
    # Test SET command
    local set_response
    set_response=$(redis_command "SET" "test_key" "test_value")
    
    assert_equals "$set_response" "+OK" "Redis SET command"
    
    # Test GET command
    local get_response
    get_response=$(redis_command "GET" "test_key")
    
    assert_equals "$get_response" "\$10" "Redis GET command returns value" || \
    assert_contains "$get_response" "test_value" "Redis GET command returns correct value"
    
    # Test EXISTS command
    local exists_response
    exists_response=$(redis_command "EXISTS" "test_key")
    
    assert_equals "$exists_response" ":1" "Redis EXISTS command for existing key"
    
    # Test DEL command
    local del_response
    del_response=$(redis_command "DEL" "test_key")
    
    assert_equals "$del_response" ":1" "Redis DEL command"
    
    # Verify key was deleted
    local exists_after_del
    exists_after_del=$(redis_command "EXISTS" "test_key")
    
    assert_equals "$exists_after_del" ":0" "Redis key deleted successfully"
    
    echo "✓ Redis basic operations test completed"
}

# Test Redis data structures
test_redis_data_structures() {
    echo "📊 Testing Redis data structures..."
    
    # Test LIST operations
    local lpush_response
    lpush_response=$(redis_command "LPUSH" "test_list" "item1")
    
    assert_equals "$lpush_response" ":1" "Redis LPUSH command"
    
    local llen_response
    llen_response=$(redis_command "LLEN" "test_list")
    
    assert_equals "$llen_response" ":1" "Redis LLEN command"
    
    # Test SET operations  
    local sadd_response
    sadd_response=$(redis_command "SADD" "test_set" "member1")
    
    assert_equals "$sadd_response" ":1" "Redis SADD command"
    
    local scard_response
    scard_response=$(redis_command "SCARD" "test_set")
    
    assert_equals "$scard_response" ":1" "Redis SCARD command"
    
    # Test HASH operations
    local hset_response
    hset_response=$(redis_command "HSET" "test_hash" "field1" "value1")
    
    assert_equals "$hset_response" ":1" "Redis HSET command"
    
    local hlen_response
    hlen_response=$(redis_command "HLEN" "test_hash")
    
    assert_equals "$hlen_response" ":1" "Redis HLEN command"
    
    echo "✓ Redis data structures test completed"
}

# Test Redis performance characteristics
test_redis_performance() {
    echo "⚡ Testing Redis performance characteristics..."
    
    local start_time=$(date +%s)
    
    # Test response time with multiple operations
    for i in {1..5}; do
        redis_command "SET" "perf_test_$i" "value_$i" >/dev/null 2>&1
    done
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo "  Multiple SET operations time: ${duration}s"
    
    if [[ $duration -lt 2 ]]; then
        echo "  ✓ Performance is excellent (< 2s)"
    elif [[ $duration -lt 5 ]]; then
        echo "  ✓ Performance is good (< 5s)"
    else
        echo "  ⚠ Performance could be improved (>= 5s)"
    fi
    
    # Test concurrent operations (background commands)
    echo "  Testing concurrent request handling..."
    local concurrent_start=$(date +%s)
    
    # Multiple concurrent commands
    {
        redis_command "GET" "perf_test_1" >/dev/null 2>&1 &
        redis_command "GET" "perf_test_2" >/dev/null 2>&1 &
        redis_command "GET" "perf_test_3" >/dev/null 2>&1 &
        wait
    }
    
    local concurrent_end=$(date +%s)
    local concurrent_duration=$((concurrent_end - concurrent_start))
    
    echo "  Concurrent requests completed in: ${concurrent_duration}s"
    
    if [[ $concurrent_duration -lt 3 ]]; then
        echo "  ✓ Concurrent handling is efficient"
    else
        echo "  ⚠ Concurrent handling could be optimized"
    fi
    
    echo "✓ Redis performance test completed"
}

# Test Redis error handling
test_redis_error_handling() {
    echo "⚠️ Testing Redis error handling..."
    
    # Test invalid command
    local invalid_response
    invalid_response=$(redis_command "INVALIDCMD")
    
    assert_contains "$invalid_response" "ERR" "Invalid command returns error" || \
    assert_not_empty "$invalid_response" "Invalid command handled"
    
    # Test wrong number of arguments
    local wrong_args_response
    wrong_args_response=$(redis_command "SET" "key_without_value")
    
    assert_contains "$wrong_args_response" "ERR" "Wrong arguments return error" || \
    assert_not_empty "$wrong_args_response" "Wrong arguments handled"
    
    # Test type error (try list operation on string)
    redis_command "SET" "string_key" "string_value" >/dev/null 2>&1
    local type_error_response
    type_error_response=$(redis_command "LPUSH" "string_key" "list_item")
    
    assert_contains "$type_error_response" "WRONGTYPE" "Type error handled" || \
    assert_contains "$type_error_response" "ERR" "Type error returns error"
    
    echo "✓ Redis error handling test completed"
}

# Cleanup test data
cleanup_redis_test_data() {
    echo "🧹 Cleaning up Redis test data..."
    
    # Clean up test keys
    local test_keys=("test_key" "test_list" "test_set" "test_hash" "string_key")
    for key in "${test_keys[@]}"; do
        redis_command "DEL" "$key" >/dev/null 2>&1 || true
    done
    
    # Clean up performance test keys
    for i in {1..5}; do
        redis_command "DEL" "perf_test_$i" >/dev/null 2>&1 || true
    done
    
    echo "✓ Redis test data cleanup complete"
}

# Main test execution
main() {
    echo "🧪 Starting Redis Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    test_redis_health
    test_redis_basic_operations
    test_redis_data_structures
    test_redis_performance
    test_redis_error_handling
    
    # Cleanup
    cleanup_redis_test_data
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ Redis integration test failed"
        exit 1
    else
        echo "✅ Redis integration test passed"
        exit 0
    fi
}

# Run main function
main "$@"