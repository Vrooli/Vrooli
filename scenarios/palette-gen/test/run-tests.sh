#!/bin/bash
# Palette Gen Test Runner
# Runs all test phases in sequence

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Running Palette Gen Test Suite ===${NC}"
echo ""

# Track results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Run test phases if they exist
if [ -d "$SCRIPT_DIR/phases" ]; then
    echo -e "${YELLOW}Running test phases...${NC}"
    cd "$SCRIPT_DIR/phases"

    for test in test-*.sh; do
        if [ -f "$test" ]; then
            TOTAL_TESTS=$((TOTAL_TESTS + 1))
            echo ""
            echo -e "${BLUE}→ Running $test${NC}"

            if ./"$test"; then
                echo -e "${GREEN}✓ $test passed${NC}"
                PASSED_TESTS=$((PASSED_TESTS + 1))
            else
                echo -e "${RED}✗ $test failed${NC}"
                FAILED_TESTS=$((FAILED_TESTS + 1))
            fi
        fi
    done

    cd "$SCENARIO_DIR"
else
    echo -e "${YELLOW}No test phases directory found${NC}"
fi

# Summary
echo ""
echo -e "${BLUE}=== Test Summary ===${NC}"
echo "Total: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
    exit 1
else
    echo -e "${GREEN}✅ All tests completed successfully${NC}"
fi
