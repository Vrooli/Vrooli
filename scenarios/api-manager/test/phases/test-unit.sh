#!/bin/bash

# API Manager - Unit Tests
# Tests individual functions and components in isolation

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Running API Manager Unit Tests${NC}"
echo "=============================="

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Track test results
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test Go code if present
if [[ -d "$PROJECT_ROOT/api" ]]; then
    echo -e "${YELLOW}Testing Go API code...${NC}"
    cd "$PROJECT_ROOT/api"
    
    # Check if there are any test files
    if find . -name "*_test.go" | grep -q .; then
        if go test -v ./...; then
            echo -e "${GREEN}✓ Go unit tests passed${NC}"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            echo -e "${RED}✗ Go unit tests failed${NC}"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
        TESTS_RUN=$((TESTS_RUN + 1))
    else
        echo -e "${YELLOW}No Go test files found${NC}"
    fi
fi

# Test TypeScript/React code if present
if [[ -f "$PROJECT_ROOT/ui/package.json" ]]; then
    echo -e "${YELLOW}Testing UI code...${NC}"
    cd "$PROJECT_ROOT/ui"
    
    # Check if test script exists in package.json
    if grep -q '"test"' package.json; then
        if npm test -- --watchAll=false 2>/dev/null; then
            echo -e "${GREEN}✓ UI unit tests passed${NC}"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            echo -e "${RED}✗ UI unit tests failed${NC}"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
        TESTS_RUN=$((TESTS_RUN + 1))
    else
        echo -e "${YELLOW}No UI test script found${NC}"
    fi
fi

# Summary
echo
echo "=============================="
echo -e "${BLUE}Unit Test Summary${NC}"
echo "=============================="
echo "Tests Run: $TESTS_RUN"
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 && $TESTS_RUN -gt 0 ]]; then
    echo -e "${GREEN}✓ All unit tests passed${NC}"
    exit 0
elif [[ $TESTS_RUN -eq 0 ]]; then
    echo -e "${YELLOW}⚠ No unit tests found${NC}"
    exit 0
else
    echo -e "${RED}✗ Some unit tests failed${NC}"
    exit 1
fi