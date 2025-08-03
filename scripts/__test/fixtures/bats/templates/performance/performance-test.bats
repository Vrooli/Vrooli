#!/usr/bin/env bats
# Template: Performance BATS Test
# Use this template for testing performance characteristics and benchmarks
#
# Copy this file and customize:
# 1. Update RESOURCE_NAME if testing a specific resource
# 2. Add your performance test cases
# 3. Adjust performance thresholds for your needs
# 4. Add custom timing and measurement logic

bats_require_minimum_version 1.5.0

# ================================
# CONFIGURATION - CHANGE THIS
# ================================
# Set target resource (or "generic" for system-level tests)
RESOURCE_NAME="generic"  # CHANGE THIS

# Performance thresholds (in milliseconds)
FAST_THRESHOLD=100       # Operations that should be very fast
NORMAL_THRESHOLD=1000    # Normal operations
SLOW_THRESHOLD=5000      # Slow operations (network, heavy processing)

# Load the unified testing infrastructure  
if [[ -n "${VROOLI_TEST_FIXTURES_DIR:-}" ]]; then
    source "${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash"
else
    # Adjust the path based on where you place this test file
    TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${TEST_DIR}/../core/common_setup.bash"
fi

setup() {
    # Use performance test setup for minimal overhead
    if [[ "$RESOURCE_NAME" == "generic" ]]; then
        setup_performance_test
    else
        setup_performance_test "$RESOURCE_NAME"
    fi
    
    # Add any custom setup here
    # export PERFORMANCE_MODE="true"
}

teardown() {
    # Always clean up test environment
    cleanup_mocks
    
    # Add any custom cleanup here
}

# ================================
# HELPER FUNCTIONS
# ================================

# Measure execution time in milliseconds
measure_time() {
    local start_time end_time duration
    start_time=$(date +%s%N)
    
    # Execute the command
    "$@"
    local exit_code=$?
    
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
    
    echo "$duration"
    return $exit_code
}

# Run multiple iterations and calculate average
benchmark() {
    local iterations="$1"
    shift
    local total_time=0
    local successful_runs=0
    
    for ((i=1; i<=iterations; i++)); do
        local duration
        if duration=$(measure_time "$@"); then
            total_time=$((total_time + duration))
            ((successful_runs++))
        fi
    done
    
    if [[ $successful_runs -gt 0 ]]; then
        local average=$((total_time / successful_runs))
        echo "$average"
    else
        echo "0"
        return 1
    fi
}

# ================================
# PERFORMANCE TESTS - Customize these
# ================================

@test "mock system startup is fast" {
    # Test that the mock system itself starts quickly
    local duration
    duration=$(benchmark 5 setup_standard_mocks)
    
    echo "Mock startup time: ${duration}ms"
    assert_less_than "$duration" "$NORMAL_THRESHOLD"
}

@test "basic commands are fast" {
    # Test that basic mocked commands respond quickly
    local docker_time curl_time jq_time
    
    docker_time=$(measure_time docker --version)
    curl_time=$(measure_time curl -s http://localhost:8080/health)
    jq_time=$(measure_time jq -r '.status' <<< '{"status":"ok"}')
    
    echo "Docker time: ${docker_time}ms"
    echo "Curl time: ${curl_time}ms" 
    echo "JQ time: ${jq_time}ms"
    
    assert_less_than "$docker_time" "$FAST_THRESHOLD"
    assert_less_than "$curl_time" "$FAST_THRESHOLD"
    assert_less_than "$jq_time" "$FAST_THRESHOLD"
}

@test "file operations are fast" {
    # Test file operation performance
    local write_time read_time delete_time
    local test_file="$TEST_TMPDIR/perf_test.txt"
    local test_data="This is test data for performance testing"
    
    write_time=$(measure_time bash -c "echo '$test_data' > '$test_file'")
    read_time=$(measure_time cat "$test_file")
    delete_time=$(measure_time rm "$test_file")
    
    echo "File write time: ${write_time}ms"
    echo "File read time: ${read_time}ms"
    echo "File delete time: ${delete_time}ms"
    
    assert_less_than "$write_time" "$FAST_THRESHOLD"
    assert_less_than "$read_time" "$FAST_THRESHOLD"
    assert_less_than "$delete_time" "$FAST_THRESHOLD"
}

@test "assertion functions are fast" {
    # Test that assertion functions themselves are performant
    local assertion_time
    
    # Test multiple assertions
    assertion_time=$(measure_time bash -c '
        assert_success
        output="test output"
        assert_output_contains "test"
        assert_env_set "PATH"
        assert_json_valid "{\"test\": true}"
    ')
    
    echo "Assertion time: ${assertion_time}ms"
    assert_less_than "$assertion_time" "$FAST_THRESHOLD"
}

@test "resource health checks are reasonably fast" {
    if [[ "$RESOURCE_NAME" != "generic" ]]; then
        # Test resource-specific health check performance
        local health_time
        
        health_time=$(measure_time assert_resource_healthy "$RESOURCE_NAME")
        
        echo "Resource health check time: ${health_time}ms"
        assert_less_than "$health_time" "$NORMAL_THRESHOLD"
    else
        skip "Generic performance test - no specific resource configured"
    fi
}

@test "bulk operations scale reasonably" {
    # Test that bulk operations don't have terrible performance characteristics
    local small_batch_time large_batch_time
    
    # Small batch: 10 operations
    small_batch_time=$(measure_time bash -c '
        for i in {1..10}; do
            curl -s http://localhost:8080/test$i >/dev/null
        done
    ')
    
    # Large batch: 100 operations  
    large_batch_time=$(measure_time bash -c '
        for i in {1..100}; do
            curl -s http://localhost:8080/test$i >/dev/null
        done
    ')
    
    echo "Small batch (10 ops): ${small_batch_time}ms"
    echo "Large batch (100 ops): ${large_batch_time}ms"
    
    # Large batch should not be more than 20x slower than small batch
    local ratio=$((large_batch_time / small_batch_time))
    echo "Performance ratio: ${ratio}x"
    assert_less_than "$ratio" "20"
    
    # Both should complete within reasonable time
    assert_less_than "$small_batch_time" "$NORMAL_THRESHOLD"
    assert_less_than "$large_batch_time" "$SLOW_THRESHOLD"
}

@test "memory usage is reasonable" {
    # Test that the test infrastructure doesn't use excessive memory
    # Note: This is a simplified check - in real scenarios you'd use more sophisticated memory monitoring
    
    local before_memory after_memory memory_growth
    
    # Get memory usage before setup (rough approximation)
    before_memory=$(ps -o rss= -p $$ 2>/dev/null || echo "0")
    
    # Do some work
    for i in {1..50}; do
        docker ps >/dev/null
        curl -s http://localhost:8080/health >/dev/null
        jq -r '.status' <<< '{"status":"ok"}' >/dev/null
    done
    
    # Get memory usage after
    after_memory=$(ps -o rss= -p $$ 2>/dev/null || echo "0")
    
    memory_growth=$((after_memory - before_memory))
    
    echo "Memory before: ${before_memory}KB"
    echo "Memory after: ${after_memory}KB"
    echo "Memory growth: ${memory_growth}KB"
    
    # Memory growth should be reasonable (less than 50MB)
    assert_less_than "$memory_growth" "51200"  # 50MB in KB
}

@test "concurrent operations work efficiently" {
    # Test that multiple concurrent operations don't block each other
    local concurrent_time sequential_time
    
    # Sequential execution
    sequential_time=$(measure_time bash -c '
        curl -s http://localhost:8080/api1 >/dev/null
        curl -s http://localhost:8080/api2 >/dev/null  
        curl -s http://localhost:8080/api3 >/dev/null
        curl -s http://localhost:8080/api4 >/dev/null
    ')
    
    # Concurrent execution (background processes)
    concurrent_time=$(measure_time bash -c '
        curl -s http://localhost:8080/api1 >/dev/null &
        curl -s http://localhost:8080/api2 >/dev/null &
        curl -s http://localhost:8080/api3 >/dev/null &
        curl -s http://localhost:8080/api4 >/dev/null &
        wait
    ')
    
    echo "Sequential time: ${sequential_time}ms"
    echo "Concurrent time: ${concurrent_time}ms"
    
    # Concurrent should be faster than sequential (or at least not much slower)
    local ratio=$((concurrent_time * 100 / sequential_time))
    echo "Concurrent efficiency: ${ratio}% of sequential time"
    
    # Concurrent should be no more than 150% of sequential time
    assert_less_than "$ratio" "150"
}

# ================================
# ADD YOUR CUSTOM PERFORMANCE TESTS HERE
# ================================

# @test "my custom performance test" {
#     # Your performance test logic here
#     local duration
#     
#     duration=$(measure_time your_command_here)
#     
#     echo "Custom operation time: ${duration}ms"
#     assert_less_than "$duration" "$NORMAL_THRESHOLD"
# }

# @test "load test simulation" {
#     # Simulate load testing scenarios
#     local max_concurrent=10
#     local operations_per_worker=20
#     
#     echo "Running load test: $max_concurrent workers, $operations_per_worker ops each"
#     
#     local total_time
#     total_time=$(measure_time bash -c "
#         for worker in {1..$max_concurrent}; do
#             (
#                 for op in {1..$operations_per_worker}; do
#                     your_load_test_operation_here
#                 done
#             ) &
#         done
#         wait
#     ")
#     
#     echo "Load test completed in: ${total_time}ms"
#     
#     # Load test should complete within reasonable time
#     local max_expected=$((SLOW_THRESHOLD * max_concurrent))
#     assert_less_than "$total_time" "$max_expected"
# }