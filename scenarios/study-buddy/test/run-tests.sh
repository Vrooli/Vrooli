#!/usr/bin/env bash
# Main test runner for study-buddy scenario

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Load environment
if [ -f "$SCENARIO_DIR/.env" ]; then
    export $(grep -v '^#' "$SCENARIO_DIR/.env" | xargs)
fi

echo -e "${BLUE}🧪 Running study-buddy tests...${NC}"

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test phase
run_test() {
    local phase=$1
    local script="$SCRIPT_DIR/phases/$phase"

    if [ -f "$script" ]; then
        echo -e "${BLUE}▶ Running $phase...${NC}"
        if bash "$script"; then
            echo -e "${GREEN}✅ $phase passed${NC}"
            ((TESTS_PASSED++))
        else
            echo -e "${RED}❌ $phase failed${NC}"
            ((TESTS_FAILED++))
        fi
    else
        echo -e "${YELLOW}⚠️ Skipping $phase (not found)${NC}"
    fi
}

# Run test phases in order
run_test "test-smoke.sh"
run_test "test-unit.sh"
run_test "test-integration.sh"
run_test "test-api.sh"
run_test "test-cli.sh"
run_test "test-ui.sh"

# Print summary
echo ""
echo -e "${BLUE}📊 Test Summary${NC}"
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -gt 0 ]; then
    exit 1
fi

echo -e "${GREEN}✅ All tests passed!${NC}"
exit 0