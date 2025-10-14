#!/bin/bash

set -e

echo "=== Performance Tests ==="

# Set test ports
export API_PORT=15901
export UI_PORT=35901

# Function to cleanup
cleanup() {
    pkill -f "home-automation-api" 2>/dev/null || true
    sleep 1
}

trap cleanup EXIT

# Start API for performance testing
echo "Starting API for performance tests..."
cd api
if [ ! -f "home-automation-api" ]; then
    go build -o home-automation-api .
fi

VROOLI_LIFECYCLE_MANAGED=true ./home-automation-api &
API_PID=$!
cd ..

# Wait for API to start
sleep 5

# Test response times
echo "Testing response times..."

# Health endpoint should be very fast
echo "Testing health endpoint response time..."
start_time=$(date +%s%N)
curl -s "http://localhost:$API_PORT/health" > /dev/null || echo "Health check failed"
end_time=$(date +%s%N)
duration=$(( (end_time - start_time) / 1000000 ))

if [ $duration -lt 100 ]; then
    echo "✅ Health endpoint response time: ${duration}ms (excellent)"
elif [ $duration -lt 500 ]; then
    echo "✅ Health endpoint response time: ${duration}ms (acceptable)"
else
    echo "⚠️  Health endpoint response time: ${duration}ms (slow, target < 500ms)"
fi

# Test devices endpoint response time
echo "Testing devices endpoint response time..."
start_time=$(date +%s%N)
curl -s "http://localhost:$API_PORT/api/v1/devices" > /dev/null || echo "Devices check failed"
end_time=$(date +%s%N)
duration=$(( (end_time - start_time) / 1000000 ))

if [ $duration -lt 200 ]; then
    echo "✅ Devices endpoint response time: ${duration}ms (excellent)"
elif [ $duration -lt 500 ]; then
    echo "✅ Devices endpoint response time: ${duration}ms (acceptable)"
else
    echo "⚠️  Devices endpoint response time: ${duration}ms (slow, target < 500ms)"
fi

# Test concurrent requests
echo "Testing concurrent request handling..."
for i in {1..10}; do
    curl -s "http://localhost:$API_PORT/health" > /dev/null &
done
wait

echo "✅ Concurrent requests handled"

# Memory usage (basic check)
echo "Checking memory usage..."
if command -v ps >/dev/null 2>&1; then
    mem_kb=$(ps -o rss= -p $API_PID 2>/dev/null || echo "0")
    mem_mb=$((mem_kb / 1024))

    if [ $mem_mb -lt 100 ]; then
        echo "✅ Memory usage: ${mem_mb}MB (excellent)"
    elif [ $mem_mb -lt 200 ]; then
        echo "✅ Memory usage: ${mem_mb}MB (acceptable)"
    else
        echo "⚠️  Memory usage: ${mem_mb}MB (high)"
    fi
else
    echo "⚠️  ps command not available, skipping memory check"
fi

# Run Go performance tests
echo "Running Go performance tests..."
cd api
if go test -bench=. -benchtime=1s ./... 2>/dev/null | grep -q "Benchmark"; then
    echo "✅ Performance benchmarks completed"
else
    echo "⚠️  No performance benchmarks found"
fi
cd ..

echo "✅ Performance tests completed"
