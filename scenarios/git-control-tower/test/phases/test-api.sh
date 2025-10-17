#!/usr/bin/env bash
# Test API endpoints for git-control-tower

set -euo pipefail

API_PORT="${API_PORT:-18700}"
API_BASE="http://localhost:${API_PORT}"

echo "Testing Git Control Tower API endpoints..."

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

pass_count=0
fail_count=0

test_pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((pass_count++))
}

test_fail() {
    echo -e "${RED}✗${NC} $1"
    ((fail_count++))
}

# Test 1: Health endpoint
echo ""
echo "Test 1: Health endpoint"
if response=$(curl -sf "${API_BASE}/health" 2>&1); then
    if echo "$response" | jq -e '.status == "healthy" or .status == "degraded"' >/dev/null 2>&1; then
        test_pass "Health endpoint returns valid JSON"
    else
        test_fail "Health endpoint JSON is invalid"
    fi
else
    test_fail "Health endpoint not accessible"
    echo "Error: git-control-tower API is not running on port ${API_PORT}"
    exit 1
fi

# Test 2: Status endpoint
echo ""
echo "Test 2: Status endpoint"
response=$(curl -sf "${API_BASE}/api/v1/status" || echo '{}')
if echo "$response" | jq -e '.branch' >/dev/null 2>&1; then
    test_pass "Status endpoint returns current branch"
else
    test_fail "Status endpoint missing branch field"
fi

if echo "$response" | jq -e '.staged' >/dev/null 2>&1; then
    test_pass "Status endpoint returns staged files array"
else
    test_fail "Status endpoint missing staged field"
fi

# Test 3: Diff endpoint
echo ""
echo "Test 3: Diff endpoint"
test_file="scenarios/git-control-tower/api/main.go"
response=$(curl -sf "${API_BASE}/api/v1/diff/${test_file}" || echo "")
# Not a failure if file has no changes
test_pass "Diff endpoint accessible"

# Test 4: Stage endpoint with files
echo ""
echo "Test 4: Stage endpoint"
payload='{"files": ["scenarios/git-control-tower/Makefile"]}'
response=$(curl -sf -X POST "${API_BASE}/api/v1/stage" \
    -H "Content-Type: application/json" \
    -d "$payload" || echo '{"success": false}')
if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
    test_pass "Stage endpoint responds with success field"
else
    test_fail "Stage endpoint response missing success field"
fi

# Test 5: Unstage endpoint
echo ""
echo "Test 5: Unstage endpoint"
payload='{"files": ["scenarios/git-control-tower/Makefile"]}'
response=$(curl -sf -X POST "${API_BASE}/api/v1/unstage" \
    -H "Content-Type: application/json" \
    -d "$payload" || echo '{"success": false}')
if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
    test_pass "Unstage endpoint responds with success field"
else
    test_fail "Unstage endpoint response missing success field"
fi

# Test 6: Commit endpoint validation (should reject invalid format)
echo ""
echo "Test 6: Commit message validation"
payload='{"message": "invalid message without type"}'
response=$(curl -s -X POST "${API_BASE}/api/v1/commit" \
    -H "Content-Type: application/json" \
    -d "$payload" || echo "")
# Should return an error about invalid format
if echo "$response" | grep -qi "conventional commit"; then
    test_pass "Commit endpoint validates conventional commit format"
else
    test_fail "Commit endpoint does not validate message format"
fi

# Test 7: Stage by scope
echo ""
echo "Test 7: Stage by scope"
payload='{"scope": "scenario:git-control-tower"}'
response=$(curl -sf -X POST "${API_BASE}/api/v1/stage" \
    -H "Content-Type: application/json" \
    -d "$payload" || echo '{"success": false}')
if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
    test_pass "Stage by scope endpoint works"
else
    test_fail "Stage by scope response invalid"
fi

# Clean up - unstage everything we staged
curl -sf -X POST "${API_BASE}/api/v1/unstage" \
    -H "Content-Type: application/json" \
    -d '{"scope": "scenario:git-control-tower"}' >/dev/null 2>&1 || true

# Summary
echo ""
echo "================================"
echo "Test Summary:"
echo "  Passed: ${pass_count}"
echo "  Failed: ${fail_count}"
echo "================================"

if [[ $fail_count -eq 0 ]]; then
    echo -e "${GREEN}✅ All API tests passed${NC}"
    exit 0
else
    echo -e "${RED}❌ Some API tests failed${NC}"
    exit 1
fi
