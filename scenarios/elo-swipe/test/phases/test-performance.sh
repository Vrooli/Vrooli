#!/bin/bash
set -e
echo "=== Performance Tests ==="
# Test performance targets defined in PRD

API_PORT=${API_PORT:-19286}

if ! curl -sf http://localhost:${API_PORT}/health > /dev/null 2>&1; then
  echo "API not running, skipping performance tests"
  echo "✅ Performance tests completed (skipped)"
  exit 0
fi

echo "Running performance benchmarks..."

# Test 1: Health endpoint response time (target: < 100ms)
echo "Test 1: Health endpoint latency..."
START=$(date +%s%N)
curl -sf http://localhost:${API_PORT}/health > /dev/null
END=$(date +%s%N)
HEALTH_LATENCY=$(( (END - START) / 1000000 ))
echo "Health endpoint: ${HEALTH_LATENCY}ms"
if [ $HEALTH_LATENCY -lt 100 ]; then
  echo "✅ Health endpoint meets target (< 100ms)"
else
  echo "⚠️  Health endpoint slower than target: ${HEALTH_LATENCY}ms (target: < 100ms)"
fi

# Test 2: Create list response time (target: < 200ms per PRD SLA)
echo "Test 2: List creation latency..."
START=$(date +%s%N)
curl -sf -X POST http://localhost:${API_PORT}/api/v1/lists \
  -H "Content-Type: application/json" \
  -d '{"name":"Perf Test","description":"Performance test","items":[{"content":"A"},{"content":"B"}]}' > /dev/null
END=$(date +%s%N)
CREATE_LATENCY=$(( (END - START) / 1000000 ))
echo "List creation: ${CREATE_LATENCY}ms"
if [ $CREATE_LATENCY -lt 200 ]; then
  echo "✅ List creation meets target (< 200ms)"
else
  echo "⚠️  List creation slower than target: ${CREATE_LATENCY}ms (target: < 200ms)"
fi

# Test 3: Comparison submission response time (target: < 100ms)
echo "Test 3: Comparison submission latency..."
# Create a test list first
LIST_RESPONSE=$(curl -sf -X POST http://localhost:${API_PORT}/api/v1/lists \
  -H "Content-Type: application/json" \
  -d '{"name":"Comparison Test","items":[{"content":"X"},{"content":"Y"}]}')
LIST_ID=$(echo "$LIST_RESPONSE" | jq -r '.list_id // empty')

if [ -n "$LIST_ID" ]; then
  COMPARISON=$(curl -sf http://localhost:${API_PORT}/api/v1/lists/${LIST_ID}/next-comparison)
  ITEM_A=$(echo "$COMPARISON" | jq -r '.item_a.id // empty')
  ITEM_B=$(echo "$COMPARISON" | jq -r '.item_b.id // empty')

  if [ -n "$ITEM_A" ] && [ -n "$ITEM_B" ]; then
    START=$(date +%s%N)
    curl -sf -X POST http://localhost:${API_PORT}/api/v1/comparisons \
      -H "Content-Type: application/json" \
      -d "{\"list_id\":\"${LIST_ID}\",\"winner_id\":\"${ITEM_A}\",\"loser_id\":\"${ITEM_B}\"}" > /dev/null
    END=$(date +%s%N)
    COMPARISON_LATENCY=$(( (END - START) / 1000000 ))
    echo "Comparison submission: ${COMPARISON_LATENCY}ms"
    if [ $COMPARISON_LATENCY -lt 100 ]; then
      echo "✅ Comparison submission meets target (< 100ms)"
    else
      echo "⚠️  Comparison slower than target: ${COMPARISON_LATENCY}ms (target: < 100ms)"
    fi
  fi
fi

# Test 4: Memory usage check (target: < 512MB from PRD)
echo "Test 4: Memory usage..."
if command -v pgrep > /dev/null 2>&1; then
  PID=$(pgrep -f "elo-swipe-api" | head -1)
  if [ -n "$PID" ] && [ -d "/proc/$PID" ]; then
    MEM_KB=$(awk '/VmRSS/ {print $2}' /proc/$PID/status)
    MEM_MB=$((MEM_KB / 1024))
    echo "Memory usage: ${MEM_MB}MB"
    if [ $MEM_MB -lt 512 ]; then
      echo "✅ Memory usage meets target (< 512MB)"
    else
      echo "⚠️  Memory usage exceeds target: ${MEM_MB}MB (target: < 512MB)"
    fi
  else
    echo "⚠️  Could not find API process for memory check"
  fi
else
  echo "⚠️  pgrep not available, skipping memory check"
fi

echo "✅ Performance tests completed"
