#!/bin/bash
# Performance test phase - <60 seconds
# Tests response times, throughput, and resource utilization
set -euo pipefail

# Setup paths and utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "=== Performance Tests Phase ==="
start_time=$(date +%s)

error_count=0
test_count=0
scenario_name=$(basename "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)")

# Get ports for testing
if command -v vrooli >/dev/null 2>&1; then
    API_PORT=$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null || echo "")
    UI_PORT=$(vrooli scenario port "$scenario_name" UI_PORT 2>/dev/null || echo "")
else
    API_PORT=""
    UI_PORT=""
fi

API_BASE_URL="http://localhost:$API_PORT"

# Performance test configuration
RESPONSE_TIME_THRESHOLD=1000  # 1 second in milliseconds
CONCURRENT_REQUESTS=5
TEST_DURATION=10  # seconds

# Helper function to measure response time
measure_response_time() {
    local url="$1"
    local method="${2:-GET}"
    local data="${3:-}"
    
    if [ -n "$data" ]; then
        curl_time=$(curl -s -w "%{time_total}" -X "$method" -H "Content-Type: application/json" -d "$data" "$url" -o /dev/null 2>/dev/null)
    else
        curl_time=$(curl -s -w "%{time_total}" -X "$method" "$url" -o /dev/null 2>/dev/null)
    fi
    
    # Convert to milliseconds
    echo "scale=0; $curl_time * 1000 / 1" | bc 2>/dev/null || echo "0"
}

# Test API response times
if [ -n "$API_PORT" ] && [ -d "api" ]; then
    echo "🧪 Testing API response times..."
    
    # Test health endpoint response time
    echo "  Testing health endpoint performance..."
    health_time=$(measure_response_time "$API_BASE_URL/health")
    
    if [ "$health_time" -gt 0 ]; then
        if [ "$health_time" -lt "$RESPONSE_TIME_THRESHOLD" ]; then
            log::success "   ✅ Health endpoint: ${health_time}ms (< ${RESPONSE_TIME_THRESHOLD}ms)"
            test_count=$((test_count + 1))
        else
            log::error "   ❌ Health endpoint: ${health_time}ms (> ${RESPONSE_TIME_THRESHOLD}ms)"
            error_count=$((error_count + 1))
        fi
    else
        log::error "   ❌ Health endpoint not responding"
        error_count=$((error_count + 1))
    fi
    
    # Test main API endpoints (customize for your scenario)
    echo "  Testing main API endpoint performance..."
    # Example: Test list endpoint
    # api_time=$(measure_response_time "$API_BASE_URL/api/v1/items")
    # if [ "$api_time" -gt 0 ]; then
    #     if [ "$api_time" -lt "$RESPONSE_TIME_THRESHOLD" ]; then
    #         log::success "   ✅ List endpoint: ${api_time}ms (< ${RESPONSE_TIME_THRESHOLD}ms)"
    #         test_count=$((test_count + 1))
    #     else
    #         log::error "   ❌ List endpoint: ${api_time}ms (> ${RESPONSE_TIME_THRESHOLD}ms)"
    #         error_count=$((error_count + 1))
    #     fi
    # else
    #     log::warning "   ⚠️  List endpoint not available"
    # fi
    
    log::info "   💡 Add scenario-specific API performance tests"
    
    # Test concurrent request handling (if curl supports it)
    echo "  Testing concurrent request handling..."
    if command -v xargs >/dev/null 2>&1; then
        concurrent_start=$(date +%s%3N)
        
        # Run concurrent requests
        seq 1 $CONCURRENT_REQUESTS | xargs -n1 -P$CONCURRENT_REQUESTS -I{} \
            curl -s "$API_BASE_URL/health" -o /dev/null 2>/dev/null || true
        
        concurrent_end=$(date +%s%3N)
        concurrent_time=$((concurrent_end - concurrent_start))
        
        # Calculate throughput (requests per second)
        if [ "$concurrent_time" -gt 0 ]; then
            throughput=$(echo "scale=2; $CONCURRENT_REQUESTS * 1000 / $concurrent_time" | bc 2>/dev/null || echo "0")
            log::success "   ✅ Concurrent requests: ${CONCURRENT_REQUESTS} in ${concurrent_time}ms (${throughput} req/s)"
            test_count=$((test_count + 1))
        else
            log::warning "   ⚠️  Concurrent test completed too quickly to measure"
        fi
    else
        log::info "   ℹ️  xargs not available, skipping concurrent tests"
    fi
    
else
    log::info "ℹ️  API not available, skipping API performance tests"
fi

# Test UI loading performance
if [ -n "$UI_PORT" ] && [ -d "ui" ]; then
    echo "🧪 Testing UI loading performance..."
    
    echo "  Testing UI page load time..."
    ui_time=$(measure_response_time "http://localhost:$UI_PORT")
    
    if [ "$ui_time" -gt 0 ]; then
        # UI loading typically has higher acceptable thresholds
        ui_threshold=$((RESPONSE_TIME_THRESHOLD * 3))  # 3 seconds for UI
        
        if [ "$ui_time" -lt "$ui_threshold" ]; then
            log::success "   ✅ UI load time: ${ui_time}ms (< ${ui_threshold}ms)"
            test_count=$((test_count + 1))
        else
            log::error "   ❌ UI load time: ${ui_time}ms (> ${ui_threshold}ms)"
            error_count=$((error_count + 1))
        fi
    else
        log::error "   ❌ UI not responding"
        error_count=$((error_count + 1))
    fi
else
    log::info "ℹ️  UI not available, skipping UI performance tests"
fi

# Test CLI performance (if CLI exists)
if [ -d "cli" ]; then
    echo "🧪 Testing CLI performance..."
    cli_binary="cli/$scenario_name"
    
    if [ -f "$cli_binary" ] && [ -x "$cli_binary" ]; then
        echo "  Testing CLI command performance..."
        
        # Test help command performance
        cli_start=$(date +%s%3N)
        "$cli_binary" --help >/dev/null 2>&1 || "$cli_binary" help >/dev/null 2>&1 || true
        cli_end=$(date +%s%3N)
        cli_time=$((cli_end - cli_start))
        
        # CLI should be very fast (< 500ms)
        cli_threshold=500
        if [ "$cli_time" -lt "$cli_threshold" ]; then
            log::success "   ✅ CLI help: ${cli_time}ms (< ${cli_threshold}ms)"
            test_count=$((test_count + 1))
        else
            log::warning "   ⚠️  CLI help: ${cli_time}ms (> ${cli_threshold}ms)"
        fi
    else
        log::info "   ℹ️  CLI binary not available for performance testing"
    fi
else
    log::info "ℹ️  No CLI directory found, skipping CLI performance tests"
fi

# Test memory usage (basic check)
echo "🧪 Testing resource utilization..."
scenario_pids=$(pgrep -f "$scenario_name" 2>/dev/null || echo "")

if [ -n "$scenario_pids" ]; then
    echo "  Checking memory usage..."
    total_memory=0
    process_count=0
    
    while IFS= read -r pid; do
        [ -z "$pid" ] && continue
        
        # Get memory usage in KB
        if [ -f "/proc/$pid/status" ]; then
            mem_kb=$(grep "VmRSS:" /proc/$pid/status 2>/dev/null | awk '{print $2}' || echo "0")
            total_memory=$((total_memory + mem_kb))
            process_count=$((process_count + 1))
        fi
    done <<< "$scenario_pids"
    
    if [ "$process_count" -gt 0 ]; then
        # Convert to MB
        total_memory_mb=$((total_memory / 1024))
        
        # Reasonable memory threshold (adjust based on scenario complexity)
        memory_threshold=512  # 512 MB
        
        if [ "$total_memory_mb" -lt "$memory_threshold" ]; then
            log::success "   ✅ Memory usage: ${total_memory_mb}MB (< ${memory_threshold}MB) across $process_count processes"
            test_count=$((test_count + 1))
        else
            log::warning "   ⚠️  Memory usage: ${total_memory_mb}MB (> ${memory_threshold}MB) across $process_count processes"
        fi
    else
        log::info "   ℹ️  No memory data available"
    fi
else
    log::info "   ℹ️  No scenario processes found for resource monitoring"
fi

# Test disk I/O performance (basic check)
echo "  Testing basic I/O performance..."
if [ -w "/tmp" ]; then
    io_start=$(date +%s%3N)
    
    # Write and read a small test file
    test_file="/tmp/${scenario_name}_perf_test_$$"
    echo "test data for performance testing" > "$test_file" 2>/dev/null
    cat "$test_file" >/dev/null 2>&1
    rm -f "$test_file" 2>/dev/null
    
    io_end=$(date +%s%3N)
    io_time=$((io_end - io_start))
    
    # I/O should be very fast (< 100ms)
    io_threshold=100
    if [ "$io_time" -lt "$io_threshold" ]; then
        log::success "   ✅ Basic I/O: ${io_time}ms (< ${io_threshold}ms)"
        test_count=$((test_count + 1))
    else
        log::warning "   ⚠️  Basic I/O: ${io_time}ms (> ${io_threshold}ms)"
    fi
else
    log::info "   ℹ️  Cannot test I/O performance (no write access to /tmp)"
fi

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))
echo ""

if [ $error_count -eq 0 ]; then
    log::success "✅ Performance tests completed successfully in ${duration}s"
    log::success "   Tests run: $test_count, Errors: $error_count"
    
    if [ $test_count -eq 0 ]; then
        log::warning "⚠️  No performance tests were actually run - customize this phase!"
    fi
else
    log::error "❌ Performance tests failed with $error_count errors in ${duration}s"
    log::error "   Tests run: $test_count, Errors: $error_count"
    echo ""
    log::info "💡 Performance improvement tips:"
    echo "   • Check for resource bottlenecks: vrooli resource status"
    echo "   • Review scenario logs for performance issues: vrooli scenario logs $scenario_name"
    echo "   • Consider caching strategies for frequently accessed data"
    echo "   • Optimize database queries and API response sizes"
    echo "   • Add performance monitoring to identify bottlenecks"
fi

if [ $duration -gt 60 ]; then
    log::warning "⚠️  Performance phase exceeded 60s target"
fi

echo ""
log::info "💡 Customize performance tests for your scenario:"
echo "   • Add scenario-specific API endpoint tests"
echo "   • Test with realistic data volumes"
echo "   • Add load testing for production readiness"
echo "   • Monitor resource usage during peak operations"

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi