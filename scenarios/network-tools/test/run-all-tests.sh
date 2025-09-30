#!/bin/bash
# Comprehensive Test Runner for Network Tools
# Runs all test suites and provides aggregated results

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test configuration
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="${TEST_DIR}/phases"
RESULTS_FILE="/tmp/network-tools-test-results.txt"
TOTAL_SUITES=0
PASSED_SUITES=0
FAILED_SUITES=0

echo "======================================================================"
echo -e "${CYAN}Network Tools - Comprehensive Test Suite${NC}"
echo "======================================================================"
echo ""

# Function to run a test suite
run_test_suite() {
    local suite_name="$1"
    local suite_file="$2"

    ((TOTAL_SUITES++))
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${MAGENTA}Running: ${suite_name}${NC}"
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if bash "$suite_file"; then
        echo -e "${GREEN}✅ ${suite_name} - PASSED${NC}" | tee -a "$RESULTS_FILE"
        ((PASSED_SUITES++))
        echo ""
        return 0
    else
        echo -e "${RED}❌ ${suite_name} - FAILED${NC}" | tee -a "$RESULTS_FILE"
        ((FAILED_SUITES++))
        echo ""
        return 1
    fi
}

# Clear previous results
> "$RESULTS_FILE"

# Check if API is running
echo -e "${BLUE}Pre-flight check: Verifying API is running...${NC}"
if ! curl -sf "http://localhost:${API_PORT:-17125}/health" >/dev/null 2>&1; then
    echo -e "${RED}❌ Error: API is not responding. Please start the scenario first.${NC}"
    echo "  Run: make run"
    exit 1
fi
echo -e "${GREEN}✓ API is responding${NC}"
echo ""

# Run all test suites
echo -e "${CYAN}Starting test execution...${NC}"
echo ""

# Core functionality tests
if [[ -f "${PHASES_DIR}/test-unit.sh" ]]; then
    run_test_suite "Unit Tests" "${PHASES_DIR}/test-unit.sh" || true
fi

if [[ -f "${PHASES_DIR}/test-api.sh" ]]; then
    run_test_suite "API Tests" "${PHASES_DIR}/test-api.sh" || true
fi

if [[ -f "${PHASES_DIR}/test-integration.sh" ]]; then
    run_test_suite "Integration Tests" "${PHASES_DIR}/test-integration.sh" || true
fi

# Quality tests
if [[ -f "${PHASES_DIR}/test-security.sh" ]]; then
    run_test_suite "Security Tests" "${PHASES_DIR}/test-security.sh" || true
fi

if [[ -f "${PHASES_DIR}/test-performance.sh" ]]; then
    run_test_suite "Performance Tests" "${PHASES_DIR}/test-performance.sh" || true
fi

# Generate test report
echo ""
echo "======================================================================"
echo -e "${CYAN}TEST REPORT${NC}"
echo "======================================================================"
echo ""
echo -e "${BLUE}Test Suites Run:${NC} ${TOTAL_SUITES}"
echo -e "${GREEN}Passed:${NC} ${PASSED_SUITES}"
echo -e "${RED}Failed:${NC} ${FAILED_SUITES}"
echo ""

# Calculate success rate
if [[ ${TOTAL_SUITES} -gt 0 ]]; then
    success_rate=$((PASSED_SUITES * 100 / TOTAL_SUITES))
    echo -e "${BLUE}Success Rate:${NC} ${success_rate}%"

    # Quality gate
    if [[ ${success_rate} -ge 80 ]]; then
        echo -e "${GREEN}✅ Quality Gate: PASSED${NC}"
    else
        echo -e "${RED}❌ Quality Gate: FAILED (< 80%)${NC}"
    fi
fi

echo ""
echo "======================================================================"
echo -e "${CYAN}Test Results Summary${NC}"
echo "======================================================================"
cat "$RESULTS_FILE"

echo ""
echo "======================================================================"

# Exit with appropriate code
if [[ ${FAILED_SUITES} -gt 0 ]]; then
    echo -e "${YELLOW}⚠️  Some test suites failed. Review the output above for details.${NC}"
    exit 1
fi

echo -e "${GREEN}🎉 All test suites passed successfully!${NC}"
exit 0