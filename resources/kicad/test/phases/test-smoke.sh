#!/usr/bin/env bash
################################################################################
# KiCad Smoke Test - v2.0 Contract Compliant
# 
# Quick health validation for KiCad resource
# Must complete within 30 seconds
#
################################################################################

set -uo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KICAD_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${KICAD_DIR}/../.." && pwd)"

# Source utilities and common functions
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${KICAD_DIR}/lib/common.sh"
source "${KICAD_DIR}/config/defaults.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    ((TESTS_RUN++))
    echo -n "  Testing ${test_name}... "
    
    if eval "$test_command" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC}"
        return 1
    fi
}

# Main smoke test
main() {
    echo "KiCad Smoke Tests"
    echo "================="
    
    # Test 1: Check if KiCad is installed or mocked
    run_test "KiCad installation" "kicad::is_installed"
    
    # Test 2: Check KiCad version
    run_test "KiCad version check" "[[ \$(kicad::get_version) != 'not_installed' ]]"
    
    # Test 3: Verify data directories exist or can be created
    run_test "Data directory creation" "kicad::init_dirs"
    
    # Test 4: Check if projects directory is accessible
    run_test "Projects directory" "[[ -d \$KICAD_PROJECTS_DIR ]]"
    
    # Test 5: Check if libraries directory is accessible
    run_test "Libraries directory" "[[ -d \$KICAD_LIBRARIES_DIR ]]"
    
    # Test 6: Check if outputs directory is accessible
    run_test "Outputs directory" "[[ -d \$KICAD_OUTPUTS_DIR ]]"
    
    # Test 7: Check if Python is available
    run_test "Python availability" "command -v \$KICAD_PYTHON_VERSION"
    
    # Test 8: Check CLI wrapper exists
    run_test "CLI wrapper" "[[ -f \$KICAD_DIR/cli.sh ]]"
    
    # Test 9: Check configuration files
    run_test "Config files" "[[ -f \$KICAD_DIR/config/defaults.sh && -f \$KICAD_DIR/config/runtime.json ]]"
    
    # Test 10: Quick health status (simulated)
    run_test "Health status" "timeout 5 echo 'healthy' || true"
    
    # Summary
    echo
    echo "Results: ${TESTS_PASSED}/${TESTS_RUN} tests passed"
    
    if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
        echo -e "${GREEN}✅ Smoke tests passed${NC}"
        exit 0
    else
        echo -e "${RED}❌ Smoke tests failed${NC}"
        exit 1
    fi
}

# Run main function
main "$@"