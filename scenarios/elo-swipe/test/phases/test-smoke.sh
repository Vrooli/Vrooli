#!/bin/bash
# test-smoke.sh - Quick smoke tests to verify basic functionality

set -e

# Detect actual API port from running process
if DETECTED_PORT=$(lsof -i -P -n | grep "elo-swipe.*LISTEN" | awk '{print $9}' | cut -d: -f2 | head -1); then
    API_PORT=${DETECTED_PORT:-19304}
else
    API_PORT=${API_PORT:-19304}
fi

API_URL="http://localhost:$API_PORT/api/v1"

echo "🔥 Smoke Tests"
echo "=============="

# Test 1: API Health
echo -n "✓ API health check... "
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/health")
if [ "$STATUS" = "200" ]; then
    echo "PASS"
else
    echo "FAIL (HTTP $STATUS)"
    exit 1
fi

# Test 2: CLI availability
echo -n "✓ CLI status check... "
if API_PORT=$API_PORT VROOLI_LIFECYCLE_MANAGED=true elo-swipe status &>/dev/null; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Test 3: Database connectivity (via list creation)
echo -n "✓ Database connectivity... "
TEST_RESPONSE=$(curl -s -X POST "$API_URL/lists" \
    -H "Content-Type: application/json" \
    -d '{"name":"DB Test","description":"Test","items":[{"content":"Test"}]}')
if echo "$TEST_RESPONSE" | grep -q "list_id"; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

echo ""
echo "All smoke tests passed! ✅"
