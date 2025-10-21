#!/bin/bash
# Master test runner for Web Scraper Manager
# Runs tests in proper order following phased testing architecture

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="$SCRIPT_DIR/phases"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
NC='\033[0m' # No Color

# Track results
PASSED=0
FAILED=0
SKIPPED=0

# Test phases in order
PHASES=(
    "structure"
    "dependencies"
    "unit"
    "api"
    "integration"
    "business"
    "performance"
)

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Web Scraper Manager - Phased Test Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Run each phase
for phase in "${PHASES[@]}"; do
    test_script="$PHASES_DIR/test-${phase}.sh"

    if [ ! -f "$test_script" ]; then
        echo -e "${YELLOW}⚠️  Phase $phase: SKIPPED (no test script)${NC}"
        SKIPPED=$((SKIPPED + 1))
        continue
    fi

    echo -e "${BLUE}Running phase: $phase${NC}"

    if bash "$test_script"; then
        echo -e "${GREEN}✅ Phase $phase: PASSED${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ Phase $phase: FAILED${NC}"
        FAILED=$((FAILED + 1))

        # Stop on first failure unless --continue flag is set
        if [ "$1" != "--continue" ]; then
            echo ""
            echo -e "${RED}Tests stopped after first failure. Use --continue to run all phases.${NC}"
            exit 1
        fi
    fi
    echo ""
done

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Passed:  $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed:  $FAILED${NC}"
fi
if [ $SKIPPED -gt 0 ]; then
    echo -e "${YELLOW}Skipped: $SKIPPED${NC}"
fi
echo ""

# Exit with appropriate code
if [ $FAILED -gt 0 ]; then
    exit 1
else
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
fi
