#!/bin/bash
# SmartNotes smoke tests - basic health and connectivity checks

# Test configuration
API_URL="${API_URL:-http://localhost:${API_PORT:-16993}}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASSED=0
FAILED=0

echo "🔥 Smoke Tests - Basic Health Checks"
echo "======================================"

# Test 1: API health check
echo -n "Test 1: API health check... "
if curl -sf "${API_URL}/health" 2>/dev/null | grep -q "healthy"; then
    echo -e "${GREEN}✓ PASS${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    FAILED=$((FAILED + 1))
fi

# Test 2: Notes endpoint
echo -n "Test 2: Notes endpoint... "
if curl -sf "${API_URL}/api/notes" &>/dev/null; then
    echo -e "${GREEN}✓ PASS${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    FAILED=$((FAILED + 1))
fi

# Test 3: Folders endpoint
echo -n "Test 3: Folders endpoint... "
if curl -sf "${API_URL}/api/folders" &>/dev/null; then
    echo -e "${GREEN}✓ PASS${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    FAILED=$((FAILED + 1))
fi

# Test 4: Tags endpoint
echo -n "Test 4: Tags endpoint... "
if curl -sf "${API_URL}/api/tags" &>/dev/null; then
    echo -e "${GREEN}✓ PASS${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    FAILED=$((FAILED + 1))
fi

# Test 5: Search endpoint
echo -n "Test 5: Search endpoint... "
if curl -sf -X POST "${API_URL}/api/search" -H "Content-Type: application/json" -d '{"query":"test"}' &>/dev/null; then
    echo -e "${GREEN}✓ PASS${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    FAILED=$((FAILED + 1))
fi

# Summary
echo ""
echo "======================================"
echo "Tests passed: $PASSED"
echo "Tests failed: $FAILED"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All smoke tests passed${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi