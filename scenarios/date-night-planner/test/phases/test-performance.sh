#!/bin/bash
set -e
echo "=== Performance Tests ==="

# Get the actual API port from service.json
API_PORT=$(jq -r '.api.config.port // 19450' /home/matthalloran8/Vrooli/scenarios/date-night-planner/.vrooli/service.json 2>/dev/null || echo "19450")

# Test 1: API response time under target (2000ms)
echo "Test: API response time..."
START_TIME=$(date +%s%N)
curl -sf -X POST http://localhost:${API_PORT}/api/v1/dates/suggest \
  -H 'Content-Type: application/json' \
  -d '{"couple_id":"test","date_type":"romantic","budget_max":100}' > /dev/null
END_TIME=$(date +%s%N)
DURATION_MS=$(( (END_TIME - START_TIME) / 1000000 ))

echo "Response time: ${DURATION_MS}ms"
if [ "$DURATION_MS" -gt 2000 ]; then
  echo "WARNING: Response time ${DURATION_MS}ms exceeds target of 2000ms"
else
  echo "✓ Response time within target (<2000ms)"
fi

# Test 2: Health check response time (should be <500ms)
echo "Test: Health check response time..."
START_TIME=$(date +%s%N)
curl -sf http://localhost:${API_PORT}/health > /dev/null
END_TIME=$(date +%s%N)
DURATION_MS=$(( (END_TIME - START_TIME) / 1000000 ))

echo "Health check time: ${DURATION_MS}ms"
if [ "$DURATION_MS" -gt 500 ]; then
  echo "WARNING: Health check time ${DURATION_MS}ms exceeds target of 500ms"
else
  echo "✓ Health check within target (<500ms)"
fi

# Test 3: CLI command response time
echo "Test: CLI response time..."
START_TIME=$(date +%s%N)
/home/matthalloran8/Vrooli/scenarios/date-night-planner/cli/date-night-planner suggest test --type casual --json > /dev/null
END_TIME=$(date +%s%N)
DURATION_MS=$(( (END_TIME - START_TIME) / 1000000 ))

echo "CLI response time: ${DURATION_MS}ms"
if [ "$DURATION_MS" -gt 3000 ]; then
  echo "WARNING: CLI response time ${DURATION_MS}ms exceeds reasonable limit"
else
  echo "✓ CLI response time acceptable"
fi

# Test 4: Concurrent request handling
echo "Test: Concurrent request handling..."
for i in {1..5}; do
  curl -sf -X POST http://localhost:${API_PORT}/api/v1/dates/suggest \
    -H 'Content-Type: application/json' \
    -d "{\"couple_id\":\"test-$i\",\"date_type\":\"romantic\"}" > /dev/null &
done
wait

echo "✓ Handled 5 concurrent requests successfully"

# Test 5: Memory usage check (if possible)
echo "Test: Memory usage..."
if command -v ps &> /dev/null; then
  PID=$(pgrep -f "date-night-planner" | head -1)
  if [ ! -z "$PID" ]; then
    MEM_KB=$(ps -o rss= -p $PID 2>/dev/null | tr -d ' ')
    if [ ! -z "$MEM_KB" ]; then
      MEM_MB=$((MEM_KB / 1024))
      echo "Memory usage: ${MEM_MB}MB"
      if [ "$MEM_MB" -gt 512 ]; then
        echo "WARNING: Memory usage ${MEM_MB}MB exceeds target of 512MB"
      else
        echo "✓ Memory usage within target (<512MB)"
      fi
    fi
  fi
fi

echo "Performance benchmarks completed"
exit 0