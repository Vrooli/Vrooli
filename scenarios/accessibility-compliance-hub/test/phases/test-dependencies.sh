#!/bin/bash
# Dependency tests for accessibility-compliance-hub scenario

set -euo pipefail

# Colors for output
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

TESTS_PASSED=0
TESTS_FAILED=0

echo -e "${YELLOW}=== Dependency Tests ===${NC}\n"

# Check required Go version
echo -n "Checking Go version... "
if command -v go &> /dev/null; then
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    echo -e "${GREEN}✓ Go ${go_version}${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Go not found${NC}"
    ((TESTS_FAILED++))
fi

# Check if curl is available
echo -n "Checking curl... "
if command -v curl &> /dev/null; then
    echo -e "${GREEN}✓ Available${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Not found${NC}"
    ((TESTS_FAILED++))
fi

# Check if jq is available
echo -n "Checking jq... "
if command -v jq &> /dev/null; then
    echo -e "${GREEN}✓ Available${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Not found${NC}"
    ((TESTS_FAILED++))
fi

# Check Go module dependencies
echo -n "Checking Go module integrity... "
if cd api && go mod verify &> /dev/null; then
    echo -e "${GREEN}✓ Verified${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Failed${NC}"
    ((TESTS_FAILED++))
fi

# Note about future dependencies
echo -e "\n${YELLOW}Future dependencies (not yet integrated):${NC}"
echo "  - PostgreSQL: Database storage (declared but not connected)"
echo "  - Browserless: UI analysis (declared but not connected)"
echo "  - Ollama: AI suggestions (declared but not connected)"
echo "  - N8n: Workflow orchestration (declared but not connected)"

# Summary
echo -e "\n${YELLOW}=== Dependency Test Summary ===${NC}"
echo -e "${GREEN}Passed: ${TESTS_PASSED}${NC}"
echo -e "${RED}Failed: ${TESTS_FAILED}${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}All dependency tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some dependency tests failed!${NC}"
    exit 1
fi
