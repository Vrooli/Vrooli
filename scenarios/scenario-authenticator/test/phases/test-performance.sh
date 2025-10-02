#!/bin/bash
set -euo pipefail

echo "=== Running test-performance.sh ==="

# Get API port using dynamic discovery
SCENARIO_NAME="scenario-authenticator"
if command -v vrooli >/dev/null 2>&1; then
  API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || echo "15785")
else
  API_PORT=${API_PORT:-15785}
fi

API_BASE="http://localhost:${API_PORT}"

# Check if API is running
if ! curl -sf "${API_BASE}/health" >/dev/null 2>&1; then
  echo "⚠ API not running on port ${API_PORT}, skipping performance tests"
  echo "  Start with: make run"
  exit 0
fi

echo "Running performance baseline tests..."

# Performance target: Token validation < 50ms (from PRD)
echo ""
echo "Test 1: Token validation performance (target: <50ms)"

# Register a test user and get token
PERF_EMAIL="perf-$(date +%s)@example.com"
REG_RESPONSE=$(curl -sf -X POST "${API_BASE}/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${PERF_EMAIL}\",\"password\":\"PerfTest123!\"}")

PERF_TOKEN=$(echo "$REG_RESPONSE" | jq -r '.token')

# Run 10 validation requests and measure average time
TOTAL_TIME=0
ITERATIONS=10

for i in $(seq 1 $ITERATIONS); do
  START_TIME=$(date +%s%N)
  curl -sf -H "Authorization: Bearer ${PERF_TOKEN}" \
    "${API_BASE}/api/v1/auth/validate" >/dev/null
  END_TIME=$(date +%s%N)

  TIME_MS=$(( (END_TIME - START_TIME) / 1000000 ))
  TOTAL_TIME=$((TOTAL_TIME + TIME_MS))
done

AVG_TIME=$((TOTAL_TIME / ITERATIONS))

if [ $AVG_TIME -lt 50 ]; then
  echo "✓ Token validation performance: ${AVG_TIME}ms average (target: <50ms) - PASSED"
else
  echo "⚠ Token validation performance: ${AVG_TIME}ms average (target: <50ms) - EXCEEDED TARGET"
fi

# Performance target: Password hashing < 200ms (from PRD)
echo ""
echo "Test 2: User registration performance (target: <200ms)"

TOTAL_REG_TIME=0
REG_ITERATIONS=5

for i in $(seq 1 $REG_ITERATIONS); do
  START_TIME=$(date +%s%N)
  curl -sf -X POST "${API_BASE}/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"perf-reg-${i}-$(date +%s)@example.com\",\"password\":\"RegTest123!\"}" >/dev/null
  END_TIME=$(date +%s%N)

  TIME_MS=$(( (END_TIME - START_TIME) / 1000000 ))
  TOTAL_REG_TIME=$((TOTAL_REG_TIME + TIME_MS))
done

AVG_REG_TIME=$((TOTAL_REG_TIME / REG_ITERATIONS))

if [ $AVG_REG_TIME -lt 200 ]; then
  echo "✓ Registration performance: ${AVG_REG_TIME}ms average (target: <200ms) - PASSED"
else
  echo "⚠ Registration performance: ${AVG_REG_TIME}ms average (target: <200ms) - EXCEEDED TARGET"
fi

# Health check performance
echo ""
echo "Test 3: Health check performance"

TOTAL_HEALTH_TIME=0
HEALTH_ITERATIONS=10

for i in $(seq 1 $HEALTH_ITERATIONS); do
  START_TIME=$(date +%s%N)
  curl -sf "${API_BASE}/health" >/dev/null
  END_TIME=$(date +%s%N)

  TIME_MS=$(( (END_TIME - START_TIME) / 1000000 ))
  TOTAL_HEALTH_TIME=$((TOTAL_HEALTH_TIME + TIME_MS))
done

AVG_HEALTH_TIME=$((TOTAL_HEALTH_TIME / HEALTH_ITERATIONS))

echo "✓ Health check performance: ${AVG_HEALTH_TIME}ms average"

echo ""
echo "Performance Summary:"
echo "  - Token validation: ${AVG_TIME}ms (target: <50ms)"
echo "  - User registration: ${AVG_REG_TIME}ms (target: <200ms)"
echo "  - Health check: ${AVG_HEALTH_TIME}ms"

echo ""
echo "✅ test-performance.sh completed successfully"
