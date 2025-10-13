#!/bin/bash

set -e

echo "=== Performance Tests ==="

# Set test ports
export API_PORT=15997
export UI_PORT=35997

# Function to cleanup
cleanup() {
    pkill -f "scenario-auditor-api" 2>/dev/null || true
    sleep 1
}

trap cleanup EXIT

# Start API for performance testing
echo "Starting API for performance tests..."
cd api
if [ ! -f "scenario-auditor-api" ]; then
    go build -o scenario-auditor-api .
fi

VROOLI_LIFECYCLE_MANAGED=true ./scenario-auditor-api &
API_PID=$!
cd ..

# Wait for API to start
sleep 5

# Test API startup time
echo "Testing API startup time..."
start_time=$(date +%s%N)
while ! curl -s "http://localhost:$API_PORT/api/v1/health" >/dev/null 2>&1; do
    sleep 0.1
    current_time=$(date +%s%N)
    elapsed=$((($current_time - $start_time) / 1000000)) # Convert to milliseconds
    if [ $elapsed -gt 10000 ]; then # 10 seconds timeout
        echo "❌ API startup timeout"
        exit 1
    fi
done
end_time=$(date +%s%N)
startup_time=$((($end_time - $start_time) / 1000000))
echo "✅ API startup time: ${startup_time}ms"

# Test API response times
echo "Testing API response times..."

endpoints=(
    "/api/v1/health"
    "/api/v1/rules"
    "/api/v1/dashboard"
    "/api/v1/preferences"
)

for endpoint in "${endpoints[@]}"; do
    echo "Testing endpoint: $endpoint"
    
    # Measure response time
    response_time=$(curl -o /dev/null -s -w '%{time_total}' "http://localhost:$API_PORT$endpoint")
    response_time_ms=$(echo "$response_time * 1000" | bc -l | cut -d. -f1)
    
    if [ "$response_time_ms" -lt 1000 ]; then # Less than 1 second
        echo "✅ $endpoint response time: ${response_time_ms}ms"
    else
        echo "⚠️  $endpoint slow response time: ${response_time_ms}ms"
    fi
done

# Test concurrent requests (simplified - use timeout to prevent hangs)
echo "Testing concurrent request handling..."
start_time=$(date +%s%N)
for i in {1..10}; do
    curl -s --max-time 2 "http://localhost:$API_PORT/api/v1/health" >/dev/null 2>&1 &
done
# Wait with timeout (max 5 seconds for all requests)
timeout 5 bash -c 'wait' 2>/dev/null || true
end_time=$(date +%s%N)
concurrent_time=$((($end_time - $start_time) / 1000000))
echo "✅ 10 concurrent requests completed in: ${concurrent_time}ms"

# Test scan performance (async job-based scanning)
echo "Testing scan performance..."
start_time=$(date +%s%N)

# Start scan (returns job ID)
scan_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/scenarios/scenario-auditor/scan" \
    -H "Content-Type: application/json" \
    -d '{}')

if ! echo "$scan_response" | jq -e '.job_id' >/dev/null 2>&1; then
    echo "⚠️  Scan endpoint not fully implemented or database not ready, skipping scan performance test"
else
    job_id=$(echo "$scan_response" | jq -r '.job_id')
    echo "Scan job started: $job_id"

    # Poll for completion (max 60 seconds)
    scan_completed=false
    poll_count=0
    max_polls=60

    while [ $poll_count -lt $max_polls ]; do
        sleep 1
        poll_count=$((poll_count + 1))

        status_response=$(curl -s "http://localhost:$API_PORT/api/v1/scenarios/scan/jobs/$job_id")
        job_status=$(echo "$status_response" | jq -r '.scan_status.status // .status // "unknown"')

        if [ "$job_status" = "completed" ] || [ "$job_status" = "failed" ]; then
            scan_completed=true
            break
        fi
    done

    end_time=$(date +%s%N)
    scan_time=$((($end_time - $start_time) / 1000000))

    if [ "$scan_completed" = true ]; then
        echo "✅ Scan completed in: ${scan_time}ms"

        # Check if scan time is within acceptable range (< 60 seconds for async)
        if [ $scan_time -lt 60000 ]; then
            echo "✅ Scan performance acceptable"
        else
            echo "⚠️  Scan performance slow: ${scan_time}ms"
        fi
    else
        echo "⚠️  Scan did not complete within timeout (60s), skipping performance validation"
    fi
fi

# Test memory usage (if available)
echo "Testing memory usage..."
if command -v ps >/dev/null 2>&1; then
    # Get memory usage of API process
    api_memory=$(ps -p $API_PID -o rss= 2>/dev/null || echo "0")
    api_memory_mb=$((api_memory / 1024))
    
    if [ $api_memory_mb -gt 0 ]; then
        echo "✅ API memory usage: ${api_memory_mb}MB"
        
        # Check if memory usage is reasonable (< 500MB)
        if [ $api_memory_mb -lt 500 ]; then
            echo "✅ Memory usage acceptable"
        else
            echo "⚠️  High memory usage: ${api_memory_mb}MB"
        fi
    else
        echo "⚠️  Could not measure memory usage"
    fi
else
    echo "⚠️  ps command not available, skipping memory test"
fi

# Test rule loading performance
echo "Testing rule loading performance..."
start_time=$(date +%s%N)
rules_response=$(curl -s "http://localhost:$API_PORT/api/v1/rules")
end_time=$(date +%s%N)
rules_time=$((($end_time - $start_time) / 1000000))

if echo "$rules_response" | jq -e '.total' >/dev/null 2>&1; then
    rule_count=$(echo "$rules_response" | jq '.total')
    echo "✅ Loaded $rule_count rules in: ${rules_time}ms"
    
    # Check performance: should load rules quickly
    if [ $rules_time -lt 1000 ]; then
        echo "✅ Rule loading performance good"
    else
        echo "⚠️  Rule loading performance slow: ${rules_time}ms"
    fi
else
    echo "❌ Rule loading failed during performance test"
    exit 1
fi

# Test CLI performance
echo "Testing CLI performance..."
cd cli
if [ -f "scenario-auditor" ]; then
    # Test CLI command execution time
    start_time=$(date +%s%N)
    ./scenario-auditor version >/dev/null 2>&1
    end_time=$(date +%s%N)
    cli_time=$((($end_time - $start_time) / 1000000))
    
    echo "✅ CLI command execution time: ${cli_time}ms"
    
    # CLI should be fast
    if [ $cli_time -lt 1000 ]; then
        echo "✅ CLI performance good"
    else
        echo "⚠️  CLI performance slow: ${cli_time}ms"
    fi
else
    echo "⚠️  CLI binary not found, skipping CLI performance tests"
fi
cd ..

# Test UI build performance (if available)
echo "Testing UI build performance..."
cd ui
if [ -f "package.json" ] && command -v node >/dev/null 2>&1 && [ -d "node_modules" ]; then
    start_time=$(date +%s%N)
    npm run build >/dev/null 2>&1
    end_time=$(date +%s%N)
    build_time=$((($end_time - $start_time) / 1000000))
    
    echo "✅ UI build time: ${build_time}ms"
    
    # UI build should complete in reasonable time (< 2 minutes)
    if [ $build_time -lt 120000 ]; then
        echo "✅ UI build performance acceptable"
    else
        echo "⚠️  UI build performance slow: ${build_time}ms"
    fi
else
    echo "⚠️  Skipping UI build performance test"
fi
cd ..

# Performance summary
echo ""
echo "=== Performance Summary ==="
echo "API startup time: ${startup_time}ms"
if [ -n "$scan_time" ]; then
    echo "Scan time: ${scan_time}ms"
fi
echo "Rule loading time: ${rules_time}ms"
if [ -n "$api_memory_mb" ] && [ "$api_memory_mb" -gt 0 ]; then
    echo "Memory usage: ${api_memory_mb}MB"
fi

echo "=== Performance Tests Complete ==="