#!/usr/bin/env bash

# Unit Tests for Device Sync Hub
# Integrates with centralized Vrooli testing infrastructure

set -euo pipefail

# Initialize testing infrastructure
APP_ROOT="${APP_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Check if centralized testing infrastructure is available
if [[ -f "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh" ]]; then
    source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
    testing::phase::init --target-time "120s"
else
    # Fallback to basic testing if centralized infrastructure not available
    echo "⚠️  Centralized testing infrastructure not found, using basic testing"
fi

# Test environment
export SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export TEST_DIR="$SCENARIO_DIR/test"
export API_PORT="${API_PORT:-17808}"
export UI_PORT="${UI_PORT:-37197}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Device Sync Hub Unit Tests ===${NC}"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"

    echo -n "Testing ${test_name}... "
    if eval "$test_command" &>/dev/null; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Structure tests
echo -e "\n${BLUE}Structure Tests:${NC}"

run_test "API binary exists" "[[ -f '$SCENARIO_DIR/api/device-sync-hub-api' ]]"
run_test "API source exists" "[[ -f '$SCENARIO_DIR/api/main.go' ]]"
run_test "Go module configured" "[[ -f '$SCENARIO_DIR/api/go.mod' ]]"

# CLI tests
echo -e "\n${BLUE}CLI Tests:${NC}"

if command -v device-sync-hub &>/dev/null; then
    run_test "CLI installed globally" "command -v device-sync-hub"
else
    run_test "CLI exists locally" "[[ -x '$SCENARIO_DIR/cli/device-sync-hub' ]]"
fi

run_test "CLI install script exists" "[[ -f '$SCENARIO_DIR/cli/install.sh' ]]"

# Run BATS CLI tests if available
if command -v bats &>/dev/null && [[ -f "$SCENARIO_DIR/cli/device-sync-hub.bats" ]]; then
    echo -e "\n${BLUE}Running BATS CLI Tests:${NC}"
    if cd "$SCENARIO_DIR" && bats cli/device-sync-hub.bats 2>&1 | tee /tmp/cli-tests.log; then
        BATS_PASSED=$(grep -oP '\d+ tests?, 0 failures' /tmp/cli-tests.log | grep -oP '^\d+' || echo "0")
        echo -e "${GREEN}✓ BATS CLI tests passed (${BATS_PASSED} tests)${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ BATS CLI tests failed${NC}"
        ((TESTS_FAILED++))
    fi
else
    if ! command -v bats &>/dev/null; then
        echo -e "${YELLOW}⚠ BATS not available, skipping CLI tests${NC}"
    fi
fi

# UI tests
echo -e "\n${BLUE}UI Tests:${NC}"

run_test "UI index.html exists" "[[ -f '$SCENARIO_DIR/ui/index.html' ]]"
run_test "UI app.js exists" "[[ -f '$SCENARIO_DIR/ui/app.js' ]]"
run_test "UI server.js exists" "[[ -f '$SCENARIO_DIR/ui/server.js' ]]"

# Database tests
echo -e "\n${BLUE}Database Tests:${NC}"

run_test "Database schema exists" "[[ -f '$SCENARIO_DIR/initialization/postgres/schema.sql' ]]"

# Configuration tests
echo -e "\n${BLUE}Configuration Tests:${NC}"

run_test "Service configuration exists" "[[ -f '$SCENARIO_DIR/.vrooli/service.json' ]]"
run_test "PRD exists" "[[ -f '$SCENARIO_DIR/PRD.md' ]]"
run_test "README exists" "[[ -f '$SCENARIO_DIR/README.md' ]]"
run_test "Makefile exists" "[[ -f '$SCENARIO_DIR/Makefile' ]]"

# Test files validation
echo -e "\n${BLUE}Test Infrastructure:${NC}"

run_test "Test helpers exist" "[[ -f '$SCENARIO_DIR/api/test_helpers.go' ]]"
run_test "Test patterns exist" "[[ -f '$SCENARIO_DIR/api/test_patterns.go' ]]"
run_test "Main tests exist" "[[ -f '$SCENARIO_DIR/api/main_test.go' ]]"
run_test "Performance tests exist" "[[ -f '$SCENARIO_DIR/api/performance_test.go' ]]"
run_test "Integration tests exist" "[[ -f '$SCENARIO_DIR/test/integration.sh' ]]"
run_test "Smoke tests exist" "[[ -f '$SCENARIO_DIR/test/phases/test-smoke.sh' ]]"

# Go tests if available
echo -e "\n${BLUE}Go Unit Tests:${NC}"

if [[ -f "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh" ]]; then
    # Use centralized testing infrastructure
    source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

    cd "$SCENARIO_DIR"

    if testing::unit::run_all_tests \
        --go-dir "api" \
        --skip-node \
        --skip-python \
        --coverage-warn 80 \
        --coverage-error 50; then
        echo -e "${GREEN}✓ Go unit tests passed${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ Go unit tests failed${NC}"
        ((TESTS_FAILED++))
    fi
else
    # Fallback to direct go test
    if cd "$SCENARIO_DIR/api" && go test -v -tags=testing -coverprofile=coverage.out ./... 2>&1 | tee test-output.txt; then
        echo -e "${GREEN}✓ Go unit tests passed${NC}"
        ((TESTS_PASSED++))

        # Display coverage
        if [[ -f coverage.out ]]; then
            COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
            echo -e "${BLUE}Test Coverage: ${COVERAGE}%${NC}"

            if (( $(echo "$COVERAGE >= 80" | bc -l) )); then
                echo -e "${GREEN}✓ Coverage target met (≥80%)${NC}"
            elif (( $(echo "$COVERAGE >= 50" | bc -l) )); then
                echo -e "${YELLOW}⚠ Coverage below target but acceptable (≥50%)${NC}"
            else
                echo -e "${RED}✗ Coverage below minimum threshold (<50%)${NC}"
                ((TESTS_FAILED++))
            fi
        fi
    else
        echo -e "${RED}✗ Go unit tests failed${NC}"
        ((TESTS_FAILED++))
    fi
fi

# Results summary
echo ""
echo -e "${BLUE}════════════════════════════════════${NC}"
echo -e "${BLUE}Unit Test Results:${NC}"
echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"
echo -e "${BLUE}════════════════════════════════════${NC}"

# End phase if using centralized infrastructure
if type -t testing::phase::end_with_summary &>/dev/null; then
    testing::phase::end_with_summary "Unit tests completed"
fi

if [[ $TESTS_FAILED -gt 0 ]]; then
    exit 1
fi

exit 0
