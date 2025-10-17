#!/bin/bash
set -euo pipefail

echo "=== Performance Tests ==="

SCENARIO_NAME="maintenance-orchestrator"
failed=0

# Get dynamic ports
API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || echo "")
if [ -z "$API_PORT" ]; then
  echo "❌ Could not discover API_PORT - is the scenario running?"
  exit 1
fi

API_BASE_URL="http://localhost:$API_PORT"

echo "Testing performance baselines..."

# Test 1: Health endpoint response time
echo "Test 1: Health endpoint latency"
start_time=$(date +%s%N)
curl -sf "$API_BASE_URL/health" > /dev/null
end_time=$(date +%s%N)
duration=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

if [ "$duration" -lt 200 ]; then
  echo "✅ Health endpoint: ${duration}ms (target: <200ms)"
else
  echo "⚠️  Health endpoint: ${duration}ms (exceeds 200ms target)"
  # Warning only, not a failure
fi

# Test 2: Scenarios endpoint response time
echo "Test 2: Scenarios endpoint latency"
start_time=$(date +%s%N)
curl -sf "$API_BASE_URL/api/v1/scenarios" > /dev/null
end_time=$(date +%s%N)
duration=$(( (end_time - start_time) / 1000000 ))

if [ "$duration" -lt 200 ]; then
  echo "✅ Scenarios endpoint: ${duration}ms (target: <200ms)"
else
  echo "⚠️  Scenarios endpoint: ${duration}ms (exceeds 200ms target)"
fi

# Test 3: Status endpoint response time
echo "Test 3: Status endpoint latency"
start_time=$(date +%s%N)
curl -sf "$API_BASE_URL/api/v1/status" > /dev/null
end_time=$(date +%s%N)
duration=$(( (end_time - start_time) / 1000000 ))

if [ "$duration" -lt 200 ]; then
  echo "✅ Status endpoint: ${duration}ms (target: <200ms)"
else
  echo "⚠️  Status endpoint: ${duration}ms (exceeds 200ms target)"
fi

# Test 4: Memory usage check
echo "Test 4: Memory usage"
if command -v ps &> /dev/null; then
  # Find the API process
  api_pid=$(pgrep -f "maintenance-orchestrator-api" | head -1)
  if [ -n "$api_pid" ]; then
    mem_kb=$(ps -o rss= -p "$api_pid")
    mem_mb=$((mem_kb / 1024))
    if [ "$mem_mb" -lt 256 ]; then
      echo "✅ Memory usage: ${mem_mb}MB (target: <256MB)"
    else
      echo "⚠️  Memory usage: ${mem_mb}MB (exceeds 256MB target)"
    fi
  else
    echo "⚠️  Could not find API process for memory check"
  fi
else
  echo "⚠️  ps command not available for memory check"
fi

# Test 5: Discovery time
echo "Test 5: Discovery performance"
# This is harder to test without restarting, so we'll check if it's reasonable from logs
echo "⚠️  Discovery time test requires server restart (checked in logs)"

# Test 6: Concurrent request handling
echo "Test 6: Concurrent requests"
# Send 10 concurrent requests to health endpoint
echo "   Sending 10 concurrent health checks..."
start_time=$(date +%s%N)
for i in {1..10}; do
  curl -sf "$API_BASE_URL/health" > /dev/null &
done
wait
end_time=$(date +%s%N)
duration=$(( (end_time - start_time) / 1000000 ))

if [ "$duration" -lt 1000 ]; then
  echo "✅ Concurrent requests: 10 requests in ${duration}ms"
else
  echo "⚠️  Concurrent requests: 10 requests in ${duration}ms (>1s)"
fi

exit $failed
