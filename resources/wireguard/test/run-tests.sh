#!/usr/bin/env bash
# WireGuard Test Runner - Main entry point for all tests

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source test phases
source "${SCRIPT_DIR}/phases/test-smoke.sh"
source "${SCRIPT_DIR}/phases/test-integration.sh"
source "${SCRIPT_DIR}/phases/test-unit.sh"

# Test configuration
TEST_PHASE="${1:-all}"
VERBOSE="${2:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ====================
# Test Execution
# ====================
run_test_phase() {
    local phase="$1"
    local description="$2"
    
    echo -e "${YELLOW}Running $description...${NC}"
    echo "================================================"
    
    local start_time=$(date +%s)
    local result=0
    
    case "$phase" in
        smoke)
            test_smoke || result=$?
            ;;
        integration)
            test_integration || result=$?
            ;;
        unit)
            test_unit || result=$?
            ;;
        *)
            echo -e "${RED}Unknown test phase: $phase${NC}"
            return 1
            ;;
    esac
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    if [[ $result -eq 0 ]]; then
        echo -e "${GREEN}✓ $description PASSED (${duration}s)${NC}"
    else
        echo -e "${RED}✗ $description FAILED (${duration}s)${NC}"
    fi
    echo ""
    
    return $result
}

# ====================
# Main Test Runner
# ====================
main() {
    echo "WireGuard Resource Test Suite"
    echo "=============================="
    echo "Test Phase: $TEST_PHASE"
    echo ""
    
    local overall_result=0
    
    case "$TEST_PHASE" in
        smoke)
            run_test_phase "smoke" "Smoke Tests (<30s)" || overall_result=$?
            ;;
        integration)
            run_test_phase "integration" "Integration Tests (<120s)" || overall_result=$?
            ;;
        unit)
            run_test_phase "unit" "Unit Tests (<60s)" || overall_result=$?
            ;;
        all)
            run_test_phase "smoke" "Smoke Tests (<30s)" || overall_result=$?
            run_test_phase "integration" "Integration Tests (<120s)" || overall_result=$?
            run_test_phase "unit" "Unit Tests (<60s)" || overall_result=$?
            ;;
        *)
            echo -e "${RED}Invalid test phase: $TEST_PHASE${NC}"
            echo "Usage: $0 [smoke|integration|unit|all] [verbose]"
            exit 1
            ;;
    esac
    
    echo "================================================"
    if [[ $overall_result -eq 0 ]]; then
        echo -e "${GREEN}All tests PASSED${NC}"
    else
        echo -e "${RED}Some tests FAILED${NC}"
    fi
    
    exit $overall_result
}

# Run main function
main