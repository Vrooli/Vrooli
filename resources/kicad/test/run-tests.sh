#!/usr/bin/env bash
################################################################################
# KiCad Resource Test Runner - v2.0 Contract Compliant
# 
# Main test entry point for KiCad resource validation
#
# Usage:
#   ./run-tests.sh [--smoke|--integration|--unit|--all]
#
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KICAD_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${KICAD_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test phase directory
PHASES_DIR="${SCRIPT_DIR}/phases"

# Default to all tests if no argument provided
TEST_TYPE="${1:---all}"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a test phase
run_test_phase() {
    local phase_name="$1"
    local phase_script="$2"
    local max_duration="${3:-60}"
    
    echo -e "\n${YELLOW}Running ${phase_name} tests...${NC}"
    
    if [[ ! -f "$phase_script" ]]; then
        echo -e "${RED}Error: Test script not found: $phase_script${NC}"
        ((FAILED_TESTS++))
        ((TOTAL_TESTS++))
        return 1
    fi
    
    if [[ ! -x "$phase_script" ]]; then
        chmod +x "$phase_script"
    fi
    
    # Run test with timeout and capture output
    local start_time=$(date +%s)
    local test_output
    local test_exit_code
    
    # Capture output to parse test counts
    # Use tee only if we have a TTY, otherwise just capture
    if [ -t 1 ]; then
        test_output=$(timeout "${max_duration}" "$phase_script" 2>&1 | tee /dev/tty)
        test_exit_code=${PIPESTATUS[0]}
    else
        test_output=$(timeout "${max_duration}" "$phase_script" 2>&1)
        test_exit_code=$?
        echo "$test_output"
    fi
    
    # Extract test counts from output (format: "Results: X/Y tests passed")
    local phase_passed=0
    local phase_total=0
    if echo "$test_output" | grep -q "Results:.*tests passed"; then
        local results_line=$(echo "$test_output" | grep "Results:.*tests passed" | tail -1)
        phase_passed=$(echo "$results_line" | sed -n 's/.*Results: \([0-9]*\)\/[0-9]* tests passed.*/\1/p')
        phase_total=$(echo "$results_line" | sed -n 's/.*Results: [0-9]*\/\([0-9]*\) tests passed.*/\1/p')
        
        # Add to global counters
        TOTAL_TESTS=$((TOTAL_TESTS + phase_total))
        PASSED_TESTS=$((PASSED_TESTS + phase_passed))
        FAILED_TESTS=$((FAILED_TESTS + (phase_total - phase_passed)))
    else
        # Fallback to simple phase counting if no detailed results
        ((TOTAL_TESTS++))
        if [[ $test_exit_code -eq 0 ]]; then
            ((PASSED_TESTS++))
        else
            ((FAILED_TESTS++))
        fi
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ $test_exit_code -eq 0 ]]; then
        echo -e "${GREEN}✓ ${phase_name} tests passed (${duration}s)${NC}"
        return 0
    else
        if [[ $test_exit_code -eq 124 ]]; then
            echo -e "${RED}✗ ${phase_name} tests timed out after ${max_duration}s${NC}"
        else
            echo -e "${RED}✗ ${phase_name} tests failed with exit code ${test_exit_code}${NC}"
        fi
        return 1
    fi
}

# Function to save test results to JSON
save_test_results() {
    local exit_code=0
    if [[ $FAILED_TESTS -gt 0 ]]; then
        exit_code=1
    fi
    
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local kicad_data_dir="${APP_ROOT}/data/resources/kicad"
    
    # Ensure data directory exists
    mkdir -p "$kicad_data_dir"
    
    # Create JSON results file
    cat > "${kicad_data_dir}/test_results.json" <<EOF
{
  "timestamp": "${timestamp}",
  "total": ${TOTAL_TESTS},
  "passed": ${PASSED_TESTS},
  "failed": ${FAILED_TESTS},
  "exit_code": ${exit_code},
  "resource": "kicad"
}
EOF
}

# Function to display results summary
display_summary() {
    echo -e "\n=================================="
    echo "Test Results Summary"
    echo "=================================="
    echo "Total tests run: ${TOTAL_TESTS}"
    echo -e "Passed: ${GREEN}${PASSED_TESTS}${NC}"
    echo -e "Failed: ${RED}${FAILED_TESTS}${NC}"
    
    # Save results to JSON for status reporting
    save_test_results
    
    if [[ $FAILED_TESTS -eq 0 && $TOTAL_TESTS -gt 0 ]]; then
        echo -e "\n${GREEN}✅ All ${TOTAL_TESTS} tests passed!${NC}"
        return 0
    elif [[ $TOTAL_TESTS -eq 0 ]]; then
        echo -e "\n${YELLOW}⚠️ No tests were run${NC}"
        return 2
    else
        echo -e "\n${RED}❌ ${FAILED_TESTS} of ${TOTAL_TESTS} tests failed${NC}"
        return 1
    fi
}

# Main test execution
main() {
    echo "KiCad Resource Test Suite"
    echo "========================="
    echo "Test type: ${TEST_TYPE}"
    
    case "$TEST_TYPE" in
        --smoke|-s)
            run_test_phase "Smoke" "${PHASES_DIR}/test-smoke.sh" 30
            ;;
        --integration|-i)
            run_test_phase "Integration" "${PHASES_DIR}/test-integration.sh" 120
            ;;
        --unit|-u)
            run_test_phase "Unit" "${PHASES_DIR}/test-unit.sh" 60
            ;;
        --all|-a|all)
            # Run all test phases in order
            run_test_phase "Smoke" "${PHASES_DIR}/test-smoke.sh" 30 || true
            run_test_phase "Integration" "${PHASES_DIR}/test-integration.sh" 120 || true
            run_test_phase "Unit" "${PHASES_DIR}/test-unit.sh" 60 || true
            ;;
        --help|-h|help)
            cat <<EOF
Usage: $0 [OPTIONS]

Run KiCad resource validation tests.

OPTIONS:
    --smoke, -s        Run quick health validation (30s max)
    --integration, -i  Run end-to-end functionality tests (120s max)
    --unit, -u        Run library function tests (60s max)
    --all, -a         Run all test phases (default)
    --help, -h        Show this help message

EXAMPLES:
    $0                 # Run all tests
    $0 --smoke         # Quick health check only
    $0 --integration   # Full functionality test

EXIT CODES:
    0  All selected tests passed
    1  One or more tests failed
    2  Invalid arguments or configuration error
EOF
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Invalid test type: ${TEST_TYPE}${NC}"
            echo "Use --help for usage information"
            exit 2
            ;;
    esac
    
    # Display summary and exit with appropriate code
    display_summary
}

# Run main function
main "$@"