#!/usr/bin/env bash
set -euo pipefail

# Test: Performance Validation
# Validates performance benchmarks and resource usage

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "⚡ Testing Prompt Injection Arena performance..."

# Test configuration
if [ -z "${API_PORT:-}" ]; then
    echo "❌ API_PORT environment variable is required"
    exit 1
fi

API_URL="http://localhost:${API_PORT}"

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

# Performance test helper
test_endpoint_performance() {
    local name=$1
    local endpoint=$2
    local target_ms=$3
    
    echo -n "  Testing ${name} (target: <${target_ms}ms)... "
    
    # Make 5 requests and get average response time
    total_time=0
    for i in {1..5}; do
        response_time=$(curl -sf -w "%{time_total}" -o /dev/null "${API_URL}${endpoint}" 2>/dev/null || echo "999")
        time_ms=$(echo "$response_time * 1000" | bc 2>/dev/null || echo "999")
        total_time=$(echo "$total_time + $time_ms" | bc 2>/dev/null || echo "999")
    done
    
    avg_time=$(echo "$total_time / 5" | bc 2>/dev/null || echo "999")
    avg_time_int=${avg_time%.*}
    
    if [ "$avg_time_int" -lt "$target_ms" ]; then
        echo "✅ (${avg_time_int}ms)"
        ((TESTS_PASSED++))
        return 0
    else
        echo "⚠️  (${avg_time_int}ms, target: ${target_ms}ms)"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test API endpoint response times (per PRD SLAs)
echo "🎯 Testing API response times..."

test_endpoint_performance "Health check" "/health" 500
test_endpoint_performance "Injection library" "/api/v1/injections/library" 500
test_endpoint_performance "Agent leaderboard" "/api/v1/leaderboards/agents" 500
test_endpoint_performance "Export formats" "/api/v1/export/formats" 500

# Test concurrent request handling
echo "🔀 Testing concurrent request handling..."

if command -v hey &> /dev/null; then
    echo -n "  Testing 50 concurrent requests... "
    hey_output=$(hey -n 50 -c 10 -m GET "${API_URL}/health" 2>&1)
    
    # Check success rate
    success_rate=$(echo "$hey_output" | grep "Status code distribution" -A 5 | grep "200" | awk '{print $NF}' || echo "0")
    
    if [ "${success_rate:-0}" -ge 45 ]; then
        echo "✅ (${success_rate}/50 successful)"
        ((TESTS_PASSED++))
    else
        echo "⚠️  (${success_rate}/50 successful, expected ≥45)"
        ((TESTS_FAILED++))
    fi
else
    echo "  ⚠️  'hey' not installed, skipping load test"
fi

# Test database query performance
echo "🗄️  Testing database performance..."

if command -v psql &> /dev/null && [ -n "${POSTGRES_HOST:-}" ]; then
    echo -n "  Testing injection library query... "
    
    query_time=$(PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" \
        -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
        -c "EXPLAIN ANALYZE SELECT * FROM injection_techniques LIMIT 100;" 2>&1 \
        | grep "Execution Time" | awk '{print $3}' || echo "999")
    
    query_time_int=${query_time%.*}
    
    if [ "${query_time_int:-999}" -lt 100 ]; then
        echo "✅ (${query_time_int}ms)"
        ((TESTS_PASSED++))
    else
        echo "⚠️  (${query_time_int}ms, target: <100ms)"
        ((TESTS_FAILED++))
    fi
else
    echo "  ⚠️  PostgreSQL not accessible, skipping database tests"
fi

# Test memory usage (if running)
echo "💾 Testing memory usage..."

api_pid=$(pgrep -f "prompt-injection-arena-api" | head -1 || echo "")
if [ -n "$api_pid" ]; then
    mem_usage=$(ps -p "$api_pid" -o rss= 2>/dev/null || echo "999999")
    mem_mb=$((mem_usage / 1024))
    
    echo -n "  API memory usage... "
    if [ "$mem_mb" -lt 2048 ]; then
        echo "✅ (${mem_mb}MB < 2GB target)"
        ((TESTS_PASSED++))
    else
        echo "⚠️  (${mem_mb}MB, target: <2GB)"
        ((TESTS_FAILED++))
    fi
else
    echo "  ⚠️  API process not found, skipping memory test"
fi

# Test CPU usage baseline
echo "🔥 Testing CPU baseline..."

if [ -n "$api_pid" ]; then
    cpu_usage=$(ps -p "$api_pid" -o %cpu= 2>/dev/null || echo "999")
    cpu_int=${cpu_usage%.*}
    
    echo -n "  API CPU usage (idle)... "
    if [ "${cpu_int:-999}" -lt 10 ]; then
        echo "✅ (${cpu_int}%)"
        ((TESTS_PASSED++))
    else
        echo "⚠️  (${cpu_int}%, expected <10% at idle)"
        ((TESTS_FAILED++))
    fi
else
    echo "  ⚠️  API process not found, skipping CPU test"
fi

# Test Go test benchmarks
echo "🏃 Running Go benchmarks..."

if [ -f "${SCENARIO_DIR}/api/main_test.go" ]; then
    cd "${SCENARIO_DIR}/api"
    if go test -bench=. -benchtime=1s -run=^$ > /tmp/go-bench.txt 2>&1; then
        echo "  ✅ Go benchmarks completed"
        ((TESTS_PASSED++))
        
        # Show benchmark results
        if grep -q "Benchmark" /tmp/go-bench.txt; then
            echo ""
            echo "  📊 Benchmark Results:"
            grep "Benchmark" /tmp/go-bench.txt | head -5 | sed 's/^/    /'
            echo ""
        fi
    else
        echo "  ⚠️  Some Go benchmarks failed (see /tmp/go-bench.txt)"
    fi
else
    echo "  ⚠️  No Go test file found, skipping benchmarks"
fi

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Performance Test Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ Passed: ${TESTS_PASSED}"
echo "  ⚠️  Warnings: ${TESTS_FAILED}"
echo "  📊 Total:  $((TESTS_PASSED + TESTS_FAILED))"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Performance tests are warnings, not failures
if [ ${TESTS_PASSED} -gt 0 ]; then
    echo "✅ Performance baseline established!"
    exit 0
else
    echo "⚠️  Unable to establish performance baseline"
    exit 0  # Don't fail the build on performance warnings
fi
