#!/usr/bin/env bash
################################################################################
# Windmill Test Runner - v2.0 Universal Contract Compliant
# 
# Main test orchestrator for all Windmill test phases
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WINDMILL_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${WINDMILL_DIR}/../.." && pwd)"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
source "${WINDMILL_DIR}/config/defaults.sh"

# Test configuration
readonly TEST_TIMEOUT_SMOKE=30
readonly TEST_TIMEOUT_INTEGRATION=120
readonly TEST_TIMEOUT_UNIT=60
readonly TEST_TIMEOUT_ALL=600

# Color codes for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

################################################################################
# Test Phase Functions
################################################################################

run_smoke_tests() {
    log::info "Running smoke tests..."
    
    if [[ -f "${SCRIPT_DIR}/phases/test-smoke.sh" ]]; then
        timeout "${TEST_TIMEOUT_SMOKE}" bash "${SCRIPT_DIR}/phases/test-smoke.sh"
        return $?
    else
        log::warning "Smoke test file not found"
        return 2
    fi
}

run_integration_tests() {
    log::info "Running integration tests..."
    
    if [[ -f "${SCRIPT_DIR}/phases/test-integration.sh" ]]; then
        timeout "${TEST_TIMEOUT_INTEGRATION}" bash "${SCRIPT_DIR}/phases/test-integration.sh"
        return $?
    else
        log::warning "Integration test file not found"
        return 2
    fi
}

run_unit_tests() {
    log::info "Running unit tests..."
    
    if [[ -f "${SCRIPT_DIR}/phases/test-unit.sh" ]]; then
        timeout "${TEST_TIMEOUT_UNIT}" bash "${SCRIPT_DIR}/phases/test-unit.sh"
        return $?
    else
        log::warning "Unit test file not found"
        return 2
    fi
}

run_all_tests() {
    log::info "Running all test phases..."
    
    local failed_tests=()
    local exit_code=0
    
    # Run smoke tests
    echo -e "${YELLOW}Phase 1/3: Smoke Tests${NC}"
    if run_smoke_tests; then
        echo -e "${GREEN}✓ Smoke tests passed${NC}"
    else
        echo -e "${RED}✗ Smoke tests failed${NC}"
        failed_tests+=("smoke")
        exit_code=1
    fi
    
    # Run integration tests
    echo -e "${YELLOW}Phase 2/3: Integration Tests${NC}"
    if run_integration_tests; then
        echo -e "${GREEN}✓ Integration tests passed${NC}"
    else
        echo -e "${RED}✗ Integration tests failed${NC}"
        failed_tests+=("integration")
        exit_code=1
    fi
    
    # Run unit tests
    echo -e "${YELLOW}Phase 3/3: Unit Tests${NC}"
    if run_unit_tests; then
        echo -e "${GREEN}✓ Unit tests passed${NC}"
    else
        local unit_exit=$?
        if [[ $unit_exit -eq 2 ]]; then
            echo -e "${YELLOW}⚠ Unit tests skipped (not available)${NC}"
        else
            echo -e "${RED}✗ Unit tests failed${NC}"
            failed_tests+=("unit")
            exit_code=1
        fi
    fi
    
    # Summary
    echo ""
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}✓ All tests passed successfully!${NC}"
    else
        echo -e "${RED}✗ Test failures detected:${NC}"
        for test in "${failed_tests[@]}"; do
            echo -e "  ${RED}- ${test}${NC}"
        done
    fi
    
    return $exit_code
}

################################################################################
# Main Execution
################################################################################

main() {
    local phase="${1:-all}"
    
    case "$phase" in
        smoke)
            run_smoke_tests
            ;;
        integration)
            run_integration_tests
            ;;
        unit)
            run_unit_tests
            ;;
        all)
            run_all_tests
            ;;
        *)
            echo "Usage: $0 {smoke|integration|unit|all}"
            echo ""
            echo "Test phases:"
            echo "  smoke       - Quick health validation (<30s)"
            echo "  integration - End-to-end functionality tests (<120s)"
            echo "  unit        - Library function tests (<60s)"
            echo "  all         - Run all test phases (<600s)"
            exit 1
            ;;
    esac
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi