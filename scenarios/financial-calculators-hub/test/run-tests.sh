#!/usr/bin/env bash
# Test runner for financial-calculators-hub
# This is a wrapper script that executes all test phases

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🧪 Running Financial Calculators Hub test suite${NC}"

# Track test results
FAILED_TESTS=()

# Function to run a test and track results
run_test() {
    local test_name="$1"
    local test_command="$2"

    echo -e "\n${YELLOW}Running: $test_name${NC}"
    if eval "$test_command"; then
        echo -e "${GREEN}✓ $test_name passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $test_name failed${NC}"
        FAILED_TESTS+=("$test_name")
        return 1
    fi
}

# Test 1: Calculation library tests
run_test "Calculation Library Tests" "cd '$SCENARIO_DIR/lib' && go test ./..."

# Test 2: API tests (may fail on database if not configured)
if [ -n "$POSTGRES_HOST" ]; then
    run_test "API Tests (with database)" "cd '$SCENARIO_DIR/api' && go test ./..."
else
    echo -e "${YELLOW}⚠️  Skipping database tests (POSTGRES_HOST not set)${NC}"
    run_test "API Tests (basic)" "cd '$SCENARIO_DIR/api' && go test -short ./..."
fi

# Test 3: CLI functionality test
if command -v financial-calculators-hub &> /dev/null; then
    run_test "CLI Functionality" "financial-calculators-hub fire --age 30 --savings 100000 --income 100000 --expenses 50000 > /dev/null"
else
    echo -e "${YELLOW}⚠️  Skipping CLI test (financial-calculators-hub not installed)${NC}"
fi

# Test 4: Health check (if service is running)
if [ -n "$API_PORT" ]; then
    run_test "API Health Check" "curl -sf http://localhost:${API_PORT}/health > /dev/null"
else
    echo -e "${YELLOW}⚠️  Skipping health check (API_PORT not set or service not running)${NC}"
fi

# Summary
echo -e "\n${GREEN}═══════════════════════════════════════${NC}"
if [ ${#FAILED_TESTS[@]} -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ ${#FAILED_TESTS[@]} test(s) failed:${NC}"
    for test in "${FAILED_TESTS[@]}"; do
        echo -e "  ${RED}✗${NC} $test"
    done
    exit 1
fi
