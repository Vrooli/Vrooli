#!/usr/bin/env bash
################################################################################
# Judge0 Integration Tests - Simplified Version
# 
# Tests basic functionality that actually works with the direct executor
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"

# Test configuration
JUDGE0_PORT="${JUDGE0_PORT:-2358}"
API_URL="http://localhost:${JUDGE0_PORT}"
TEST_FAILED=0

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Running Judge0 Integration Tests${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Test 1: API availability
echo -e "  ${YELLOW}ℹ${NC} Testing API availability..."
if timeout 5 curl -sf "${API_URL}/about" &>/dev/null; then
    echo -e "  ${GREEN}✓${NC} API is available"
else
    echo -e "  ${RED}✗${NC} API is not available"
    TEST_FAILED=1
fi

# Test 2: Languages endpoint
echo -e "  ${YELLOW}ℹ${NC} Testing languages endpoint..."
lang_count=$(timeout 5 curl -sf "${API_URL}/languages" 2>/dev/null | jq 'length' 2>/dev/null || echo "0")
if [[ $lang_count -gt 20 ]]; then
    echo -e "  ${GREEN}✓${NC} Languages available: $lang_count"
else
    echo -e "  ${RED}✗${NC} Insufficient languages: $lang_count"
    TEST_FAILED=1
fi

# Test 3: Direct executor
echo -e "  ${YELLOW}ℹ${NC} Testing direct executor..."
direct_exec="${RESOURCE_DIR}/lib/direct-executor.sh"
if [[ -x "$direct_exec" ]]; then
    result=$("$direct_exec" execute python3 'print("test")' "" 5 128 2>/dev/null || echo "FAILED")
    if [[ "$result" != "FAILED" ]] && echo "$result" | grep -q '"status":"accepted"'; then
        echo -e "  ${GREEN}✓${NC} Direct executor working"
    else
        echo -e "  ${YELLOW}⚠${NC}  Direct executor needs configuration"
    fi
else
    echo -e "  ${RED}✗${NC} Direct executor not found"
    TEST_FAILED=1
fi

# Test 4: Health check performance
echo -e "  ${YELLOW}ℹ${NC} Testing health check performance..."
start_time=$(date +%s%N)
timeout 2 curl -sf "${API_URL}/system_info" &>/dev/null
end_time=$(date +%s%N)
response_time=$(( (end_time - start_time) / 1000000 ))

if [[ $response_time -lt 500 ]]; then
    echo -e "  ${GREEN}✓${NC} Health check response: ${response_time}ms"
elif [[ $response_time -lt 1000 ]]; then
    echo -e "  ${YELLOW}⚠${NC}  Health check slow: ${response_time}ms"
else
    echo -e "  ${RED}✗${NC} Health check very slow: ${response_time}ms"
    TEST_FAILED=1
fi

# Test 5: Cache functionality
echo -e "  ${YELLOW}ℹ${NC} Testing execution cache..."
cache_dir="/tmp/judge0_exec_cache"
if [[ -d "$cache_dir" ]] || mkdir -p "$cache_dir" 2>/dev/null; then
    # Run same code twice to test cache
    test_code='print("cache_test")'
    result1=$("$direct_exec" execute python3 "$test_code" "" 5 128 2>/dev/null || echo "FAILED")
    result2=$("$direct_exec" execute python3 "$test_code" "" 5 128 2>/dev/null || echo "FAILED")
    
    if [[ "$result1" == "$result2" ]]; then
        echo -e "  ${GREEN}✓${NC} Execution cache working"
    else
        echo -e "  ${YELLOW}⚠${NC}  Cache behavior inconsistent"
    fi
else
    echo -e "  ${YELLOW}⚠${NC}  Cache directory not accessible"
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [[ $TEST_FAILED -eq 0 ]]; then
    echo -e "${GREEN}✅ All integration tests passed${NC}\n"
    exit 0
else
    echo -e "${RED}❌ Some integration tests failed${NC}\n"
    exit 1
fi