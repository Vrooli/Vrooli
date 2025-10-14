#!/bin/bash
# Structure tests for accessibility-compliance-hub scenario

set -euo pipefail

# Colors for output
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

TESTS_PASSED=0
TESTS_FAILED=0

echo -e "${YELLOW}=== Structure Tests ===${NC}\n"

# Define required files
required_files=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "PROBLEMS.md"
    "Makefile"
    "api/main.go"
    "api/go.mod"
    "cli/accessibility-compliance-hub"
    "cli/install.sh"
)

# Check required files exist
for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} $file"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC} $file (missing)"
        ((TESTS_FAILED++))
    fi
done

# Check service.json is valid JSON
echo -n "Validating service.json... "
if jq empty .vrooli/service.json 2>/dev/null; then
    echo -e "${GREEN}✓ Valid JSON${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Invalid JSON${NC}"
    ((TESTS_FAILED++))
fi

# Check CLI is executable
echo -n "Checking CLI permissions... "
if [ -x "cli/accessibility-compliance-hub" ]; then
    echo -e "${GREEN}✓ Executable${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Not executable${NC}"
    ((TESTS_FAILED++))
fi

# Check .gitignore exists and protects artifacts
echo -n "Checking .gitignore... "
if [ -f ".gitignore" ]; then
    if grep -q "coverage" .gitignore && grep -q "api/.*-api" .gitignore; then
        echo -e "${GREEN}✓ Protects artifacts${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${YELLOW}⚠ Incomplete protection${NC}"
        ((TESTS_PASSED++))
    fi
else
    echo -e "${RED}✗ Missing${NC}"
    ((TESTS_FAILED++))
fi

# Summary
echo -e "\n${YELLOW}=== Structure Test Summary ===${NC}"
echo -e "${GREEN}Passed: ${TESTS_PASSED}${NC}"
echo -e "${RED}Failed: ${TESTS_FAILED}${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}All structure tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some structure tests failed!${NC}"
    exit 1
fi
