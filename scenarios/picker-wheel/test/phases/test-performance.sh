#!/bin/bash
set -e

echo "=== Performance Tests for Picker Wheel ==="

# Validate required environment variables
if [ -z "${API_PORT:-}" ]; then
    echo "ERROR: API_PORT environment variable is required for tests"
    exit 1
fi

API_URL="http://localhost:${API_PORT}"

# Test API response time
echo "Testing API response times..."
HEALTH_TIME=$(curl -o /dev/null -s -w '%{time_total}' "${API_URL}/health")
echo "Health endpoint: ${HEALTH_TIME}s"
if (( $(echo "$HEALTH_TIME > 0.5" | bc -l) )); then
  echo "⚠️ Health endpoint slow (>500ms)"
fi

# Test spin endpoint performance
echo "Testing spin performance..."
for i in {1..10}; do
  START=$(date +%s%N)
  curl -sf -X POST "${API_URL}/api/spin" \
    -H "Content-Type: application/json" \
    -d '{"wheel_id": "yes-or-no"}' > /dev/null
  END=$(date +%s%N)
  ELAPSED=$((($END - $START) / 1000000))
  echo "Spin $i: ${ELAPSED}ms"
  if [ $ELAPSED -gt 500 ]; then
    echo "⚠️ Spin endpoint slow (>500ms)"
  fi
done

# Test concurrent requests
echo "Testing concurrent spin requests..."
for i in {1..20}; do
  curl -sf -X POST "${API_URL}/api/spin" \
    -H "Content-Type: application/json" \
    -d '{"wheel_id": "dinner-decider"}' > /dev/null &
done
wait
echo "✅ Handled 20 concurrent requests"

# Test memory usage
echo "Testing memory usage..."
API_PID=$(pgrep -f "picker-wheel-api" || echo "")
if [ -n "$API_PID" ]; then
  MEM_USAGE=$(ps -o rss= -p $API_PID | awk '{print $1/1024 "MB"}')
  echo "API memory usage: $MEM_USAGE"
fi

echo "✅ Performance tests completed"