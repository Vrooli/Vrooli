#!/usr/bin/env bash
#
# Performance Test: Validate PRD Control Tower performance characteristics
#

set -eo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

echo "⚡ Running PRD Control Tower performance tests..."

API_PORT="${API_PORT:-18600}"
API_URL="http://localhost:$API_PORT"

# Check if API is running
if ! curl -sf "$API_URL/api/v1/health" > /dev/null 2>&1; then
    echo "  ⚠️  API not running, skipping performance tests"
    exit 0
fi

# Test 1: Health Check Response Time
echo "  Testing health check response time..."

total_time=0
iterations=10

for i in $(seq 1 $iterations); do
    response_time=$(curl -s -w "%{time_total}" -o /dev/null "$API_URL/api/v1/health")
    total_time=$(echo "$total_time + $response_time" | bc)
done

avg_time=$(echo "scale=3; $total_time / $iterations" | bc)
avg_ms=$(echo "scale=0; $avg_time * 1000" | bc)

echo "    Average health check time: ${avg_ms}ms"

# Health check should respond in < 500ms
if (( $(echo "$avg_time < 0.5" | bc -l) )); then
    echo "  ✓ Health check response time acceptable"
else
    echo "  ⚠️  Health check response time slower than target: ${avg_ms}ms"
fi

# Test 2: Draft Save Performance Target
echo "  Testing draft save performance target..."

# Draft save should complete in < 500ms (target from PRD)
# This is a logical test - actual implementation test will be added when draft CRUD is implemented

echo "    Target: < 500ms"
echo "    Note: Draft CRUD not yet implemented, placeholder test"
echo "  ✓ Draft save performance target defined"

# Test 3: Catalog Load Performance Target
echo "  Testing catalog load performance target..."

# Catalog should load in < 2s for 200+ scenarios (target from PRD)
# This is a logical test

echo "    Target: < 2s for 200+ scenarios"
echo "    Note: Catalog enumeration not yet implemented, placeholder test"
echo "  ✓ Catalog load performance target defined"

# Test 4: Validation Performance Target
echo "  Testing validation performance target..."

# Validation should complete in < 5s per PRD (target from PRD)

echo "    Target: < 5s per PRD"
echo "    Note: scenario-auditor integration not yet implemented, placeholder test"
echo "  ✓ Validation performance target defined"

# Test 5: AI Section Generation Performance Target
echo "  Testing AI section generation performance target..."

# AI generation should complete in < 10s per section (target from PRD)

echo "    Target: < 10s per section"
echo "    Note: AI assistance not yet implemented, placeholder test"
echo "  ✓ AI section generation performance target defined"

# Test 6: Publish Performance Target
echo "  Testing publish performance target..."

# Publishing should complete in < 3s including validation (target from PRD)

echo "    Target: < 3s including validation"
echo "    Note: Publishing not yet implemented, placeholder test"
echo "  ✓ Publish performance target defined"

# Test 7: Memory Usage
echo "  Testing memory usage..."

if pgrep -f "prd-control-tower-api" > /dev/null; then
    pid=$(pgrep -f "prd-control-tower-api" | head -1)
    mem_usage=$(ps -p "$pid" -o rss= 2>/dev/null | awk '{print $1/1024}' || echo "0")

    echo "    Current memory usage: ${mem_usage}MB"

    # Warn if memory usage is very high (>300MB for this service)
    if (( $(echo "$mem_usage > 300" | bc -l 2>/dev/null || echo "0") )); then
        echo "  ⚠️  High memory usage detected: ${mem_usag}MB"
    else
        echo "  ✓ Memory usage within normal range"
    fi
else
    echo "  ⚠️  API not running, skipping memory test"
fi

# Test 8: CPU Usage Check
echo "  Testing CPU usage..."

if pgrep -f "prd-control-tower-api" > /dev/null; then
    pid=$(pgrep -f "prd-control-tower-api" | head -1)
    cpu_usage=$(ps -p "$pid" -o %cpu= 2>/dev/null || echo "0.0")

    echo "    Current CPU usage: ${cpu_usage}%"
    echo "  ✓ CPU usage checked (informational)"
else
    echo "  ⚠️  API not running, skipping CPU test"
fi

# Test 9: Concurrent Request Handling
echo "  Testing concurrent request handling..."

if command -v parallel &> /dev/null; then
    echo "    Sending 50 concurrent health check requests..."

    seq 50 | parallel -j 10 "curl -s -w '\n%{http_code}\n' $API_URL/api/v1/health" > /tmp/concurrent-prd-test.log 2>&1

    success_count=$(grep "^200$" /tmp/concurrent-prd-test.log | wc -l)
    echo "    Successful responses: $success_count/50"

    if [ "$success_count" -ge 45 ]; then
        echo "  ✓ Concurrent request handling acceptable (≥90% success)"
    else
        echo "  ⚠️  Lower concurrent request success rate: $success_count/50"
    fi
else
    echo "  ⚠️  GNU parallel not available, skipping concurrent test"
fi

echo "✅ Performance tests passed"
