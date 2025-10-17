#!/usr/bin/env bash
# Temporal Resource - Unit Tests
# Test individual functions and configuration

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo "==============================="
echo "Temporal Resource - Unit Tests"
echo "==============================="
echo ""

TESTS_PASSED=0
TESTS_FAILED=0

# Test 1: CLI script exists and is executable
echo -n "1. Checking CLI script exists and is executable... "
if [[ -x "${RESOURCE_DIR}/cli.sh" ]]; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - CLI not found or not executable"
    ((TESTS_FAILED++))
fi

# Test 2: Core library exists
echo -n "2. Checking core library exists... "
if [[ -f "${RESOURCE_DIR}/lib/core.sh" ]]; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Core library not found"
    ((TESTS_FAILED++))
fi

# Test 3: Test library exists
echo -n "3. Checking test library exists... "
if [[ -f "${RESOURCE_DIR}/lib/test.sh" ]]; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Test library not found"
    ((TESTS_FAILED++))
fi

# Test 4: Configuration files exist
echo -n "4. Checking configuration files... "
if [[ -f "${RESOURCE_DIR}/config/defaults.sh" ]] && \
   [[ -f "${RESOURCE_DIR}/config/runtime.json" ]] && \
   [[ -f "${RESOURCE_DIR}/config/schema.json" ]]; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Configuration files missing"
    ((TESTS_FAILED++))
fi

# Test 5: Runtime.json is valid JSON
echo -n "5. Validating runtime.json... "
if jq empty "${RESOURCE_DIR}/config/runtime.json" 2>/dev/null; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Invalid JSON in runtime.json"
    ((TESTS_FAILED++))
fi

# Test 6: Schema.json is valid JSON
echo -n "6. Validating schema.json... "
if jq empty "${RESOURCE_DIR}/config/schema.json" 2>/dev/null; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Invalid JSON in schema.json"
    ((TESTS_FAILED++))
fi

# Test 7: Required directories exist
echo -n "7. Checking required directories... "
if [[ -d "${RESOURCE_DIR}/lib" ]] && \
   [[ -d "${RESOURCE_DIR}/config" ]] && \
   [[ -d "${RESOURCE_DIR}/test" ]] && \
   [[ -d "${RESOURCE_DIR}/test/phases" ]]; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Required directories missing"
    ((TESTS_FAILED++))
fi

# Test 8: PRD.md exists
echo -n "8. Checking PRD documentation... "
if [[ -f "${RESOURCE_DIR}/PRD.md" ]]; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - PRD.md not found"
    ((TESTS_FAILED++))
fi

# Test 9: README.md exists
echo -n "9. Checking README documentation... "
if [[ -f "${RESOURCE_DIR}/README.md" ]]; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - README.md not found"
    ((TESTS_FAILED++))
fi

# Test 10: Port configuration is correct
echo -n "10. Checking port configuration... "
source "${RESOURCE_DIR}/config/defaults.sh"
if [[ "$TEMPORAL_PORT" == "7233" ]] && [[ "$TEMPORAL_GRPC_PORT" == "7234" ]]; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Incorrect port configuration"
    ((TESTS_FAILED++))
fi

# Summary
echo ""
echo "==============================="
echo "Test Summary"
echo "==============================="
echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}All unit tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some unit tests failed!${NC}"
    exit 1
fi