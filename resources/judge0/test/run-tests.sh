#!/usr/bin/env bash
################################################################################
# Judge0 Test Runner - v2.0 Contract Compliant
# 
# Main test orchestrator for Judge0 resource testing
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
APP_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# Source common utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"

# Test phase to run
PHASE="${1:-all}"
VERBOSE="${VERBOSE:-false}"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

log_test() {
    local status="$1"
    local message="$2"
    
    case "$status" in
        PASS)
            echo -e "${GREEN}✓${NC} $message"
            ((PASSED_TESTS++))
            ;;
        FAIL)
            echo -e "${RED}✗${NC} $message"
            ((FAILED_TESTS++))
            ;;
        INFO)
            echo -e "${YELLOW}ℹ${NC} $message"
            ;;
        *)
            echo "$message"
            ;;
    esac
    ((TOTAL_TESTS++))
}

run_phase() {
    local phase_name="$1"
    local phase_script="${SCRIPT_DIR}/phases/test-${phase_name}.sh"
    
    if [[ ! -f "$phase_script" ]]; then
        log_test "FAIL" "Test phase '$phase_name' not found at $phase_script"
        return 1
    fi
    
    echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Running ${phase_name} tests...${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    
    # Make script executable
    chmod +x "$phase_script"
    
    # Run the phase
    if bash "$phase_script"; then
        log_test "PASS" "${phase_name} phase completed successfully"
        return 0
    else
        log_test "FAIL" "${phase_name} phase failed"
        return 1
    fi
}

print_summary() {
    echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Test Summary${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "Total Tests: $TOTAL_TESTS"
    echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
    
    if [[ $FAILED_TESTS -eq 0 ]]; then
        echo -e "\n${GREEN}✅ All tests passed!${NC}"
        return 0
    else
        echo -e "\n${RED}❌ Some tests failed${NC}"
        return 1
    fi
}

main() {
    echo -e "${GREEN}🧪 Judge0 Test Suite${NC}"
    echo -e "Phase: ${PHASE}"
    echo -e "Time: $(date '+%Y-%m-%d %H:%M:%S')\n"
    
    local exit_code=0
    
    case "$PHASE" in
        smoke)
            run_phase "smoke" || exit_code=$?
            ;;
        integration)
            run_phase "integration" || exit_code=$?
            ;;
        unit)
            run_phase "unit" || exit_code=$?
            ;;
        all)
            # Run all phases in order
            run_phase "smoke" || exit_code=$?
            
            if [[ $exit_code -eq 0 ]]; then
                run_phase "integration" || exit_code=$?
            fi
            
            if [[ $exit_code -eq 0 ]]; then
                run_phase "unit" || exit_code=$?
            fi
            ;;
        *)
            echo -e "${RED}Unknown test phase: $PHASE${NC}"
            echo "Available phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
    
    print_summary
    exit $exit_code
}

# Run main function
main "$@"