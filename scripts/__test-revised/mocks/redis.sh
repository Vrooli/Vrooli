#!/usr/bin/env bash
# Simplified Redis Mock for Resource Testing
# 
# Provides basic Redis command mocking for testing resource functionality
# Focus: Connection testing, basic operations, health checks
#
# NAMING CONVENTIONS: This mock follows the standard naming pattern expected
# by the convention-based resource testing framework:
#
#   test_redis_connection() : Test Redis connectivity via ping
#   test_redis_health()     : Test Redis health/status
#   test_redis_basic()      : Test basic Redis operations (set/get)
#
# This allows the testing framework to automatically discover and run these
# tests without requiring hardcoded knowledge of Redis specifics.

# Mock redis-cli command
redis-cli() {
    case "$*" in
        # Ping test
        "ping"|"-c ping"*)
            echo "PONG"
            return 0
            ;;
        # Info command
        "info"|"-c info"*)
            echo "# Server"
            echo "redis_version:7.0.0"
            echo "redis_git_sha1:00000000"
            echo "redis_mode:standalone"
            echo "os:Linux 5.15.0 x86_64"
            echo "# Clients"
            echo "connected_clients:1"
            return 0
            ;;
        # Set operation
        "set "*|"-c set"*)
            echo "OK"
            return 0
            ;;
        # Get operation
        "get "*|"-c get"*)
            echo "mock_value"
            return 0
            ;;
        # Delete operation
        "del "*|"-c del"*)
            echo "(integer) 1"
            return 0
            ;;
        # Keys listing
        "keys "*|"-c keys"*)
            echo "test_key"
            echo "another_key"
            return 0
            ;;
        # Flush operations
        "flushall"|"flushdb"|"-c flushall"*|"-c flushdb"*)
            echo "OK"
            return 0
            ;;
        # List operations
        "lpush "*|"-c lpush"*)
            echo "(integer) 1"
            return 0
            ;;
        "llen "*|"-c llen"*)
            echo "(integer) 1"
            return 0
            ;;
        "lrange "*|"-c lrange"*)
            echo "list_item_1"
            echo "list_item_2"
            return 0
            ;;
        # Hash operations
        "hset "*|"-c hset"*)
            echo "(integer) 1"
            return 0
            ;;
        "hget "*|"-c hget"*)
            echo "hash_value"
            return 0
            ;;
        "hgetall "*|"-c hgetall"*)
            echo "field1"
            echo "value1"
            echo "field2"
            echo "value2"
            return 0
            ;;
        # Set operations
        "sadd "*|"-c sadd"*)
            echo "(integer) 1"
            return 0
            ;;
        "smembers "*|"-c smembers"*)
            echo "set_member_1"
            echo "set_member_2"
            return 0
            ;;
        # PubSub operations
        "publish "*|"-c publish"*)
            echo "(integer) 1"
            return 0
            ;;
        # Generic operations
        "exists "*|"-c exists"*)
            echo "(integer) 1"
            return 0
            ;;
        "ttl "*|"-c ttl"*)
            echo "(integer) -1"
            return 0
            ;;
        "expire "*|"-c expire"*)
            echo "(integer) 1"
            return 0
            ;;
        # Default case
        *)
            echo "OK"
            return 0
            ;;
    esac
}

# Mock redis-server command
redis-server() {
    case "$*" in
        "--version")
            echo "Redis server v=7.0.0 sha=00000000:0 malloc=jemalloc-5.2.1 bits=64 build=1234567890"
            return 0
            ;;
        "--test-memory")
            echo "Please keep the test running for at least 60 seconds."
            echo "Memory usage test completed successfully"
            return 0
            ;;
        *)
            echo "Starting Redis server..."
            echo "Server initialized"
            return 0
            ;;
    esac
}

# Mock redis-benchmark command
redis-benchmark() {
    echo "====== PING_INLINE ======"
    echo "  100000 requests completed in 1.20 seconds"
    echo "  Requests per second: 83333.33"
    echo ""
    echo "====== SET ======"
    echo "  100000 requests completed in 1.30 seconds" 
    echo "  Requests per second: 76923.08"
    return 0
}

# Test functions for resource validation
test_redis_connection() {
    # Simulate connection test via ping
    local result
    result=$(redis-cli ping 2>/dev/null)
    
    if [[ "$result" == "PONG" ]]; then
        return 0
    else
        return 1
    fi
}


test_redis_basic() {
    # Simulate basic functionality test with set/get
    redis-cli set test_key "test_value" >/dev/null 2>&1 || return 1
    
    local result
    result=$(redis-cli get test_key 2>/dev/null)
    
    if [[ -n "$result" ]]; then
        return 0
    else
        return 1
    fi
}

test_redis_health() {
    # Simulate comprehensive health check
    local ping_result
    ping_result=$(redis-cli ping 2>/dev/null)
    
    if [[ "$ping_result" == "PONG" ]]; then
        log_test_pass "redis health check"
        return 0
    else
        log_test_fail "redis health check" "Ping failed"
        return 1
    fi
}

test_redis_info() {
    # Test Redis info command
    redis-cli info >/dev/null 2>&1
    return $?
}

# Export mock functions
export -f redis-cli redis-server redis-benchmark
export -f test_redis_connection test_redis_basic test_redis_health test_redis_info