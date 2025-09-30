#!/bin/bash
# SmartNotes smoke tests - basic health and connectivity checks

set -euo pipefail

# Import test framework
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../../../../scripts/test/framework.sh"

# Test configuration
API_URL="http://localhost:${API_PORT:-17009}"
UI_URL="http://localhost:${UI_PORT:-36529}"

# Start test suite
test::suite "SmartNotes Smoke Tests"

# Test 1: API health check
test::case "API health check"
test::expect_json "${API_URL}/health" '.status' "healthy"

# Test 2: API endpoints availability
test::case "Notes endpoint available"
response=$(curl -sf "${API_URL}/api/notes" || echo "FAILED")
if [[ "$response" == "FAILED" ]]; then
    test::fail "Notes endpoint not responding"
else
    test::pass "Notes endpoint responding"
fi

# Test 3: UI server health
test::case "UI server responding"
response=$(curl -sf -o /dev/null -w "%{http_code}" "${UI_URL}" || echo "000")
if [[ "$response" == "200" ]]; then
    test::pass "UI server healthy (HTTP 200)"
else
    test::fail "UI server not responding (HTTP ${response})"
fi

# Test 4: Resource connectivity
test::case "Postgres connectivity"
response=$(curl -sf "${API_URL}/health" | jq -r '.status' || echo "FAILED")
if [[ "$response" == "healthy" ]]; then
    test::pass "Database connection working"
else
    test::fail "Database connection failed"
fi

# Test 5: Qdrant availability
test::case "Qdrant service check"
response=$(curl -sf "http://localhost:6333/collections" -o /dev/null -w "%{http_code}" || echo "000")
if [[ "$response" == "200" ]]; then
    test::pass "Qdrant service available"
else
    test::warn "Qdrant service not available (semantic search disabled)"
fi

# Complete test suite
test::summary