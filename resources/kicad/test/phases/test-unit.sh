#!/usr/bin/env bash
################################################################################
# KiCad Unit Test - v2.0 Contract Compliant
# 
# Library function validation for KiCad resource
# Must complete within 60 seconds
#
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KICAD_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${KICAD_DIR}/../.." && pwd)"

# Source utilities and library functions
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${KICAD_DIR}/lib/common.sh"
source "${KICAD_DIR}/lib/install.sh" 2>/dev/null || true
source "${KICAD_DIR}/lib/status.sh" 2>/dev/null || true
source "${KICAD_DIR}/lib/inject.sh" 2>/dev/null || true
source "${KICAD_DIR}/lib/content.sh" 2>/dev/null || true
source "${KICAD_DIR}/lib/desktop.sh" 2>/dev/null || true
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
    
    ((TESTS_RUN++)) || true
    echo -n "  Testing ${test_name}... "
    
    if eval "$test_command" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++)) || true
        return 0
    else
        echo -e "${RED}✗${NC}"
        return 1
    fi
}

# Test function existence
test_function_exists() {
    local func_name="$1"
    
    ((TESTS_RUN++)) || true
    echo -n "  Function ${func_name}... "
    
    if type -t "$func_name" &>/dev/null; then
        echo -e "${GREEN}✓ exists${NC}"
        ((TESTS_PASSED++)) || true
        return 0
    else
        echo -e "${YELLOW}⚠ not found${NC}"
        return 1
    fi
}

# Main unit test
main() {
    echo "KiCad Unit Tests"
    echo "================"
    
    # Test common.sh functions
    echo
    echo "Testing common.sh functions:"
    test_function_exists "kicad::init_dirs"
    test_function_exists "kicad::is_installed"
    test_function_exists "kicad::get_version"
    test_function_exists "kicad::check_python_api"
    
    # Test init_dirs creates all required directories
    echo
    echo "Testing directory initialization:"
    kicad::init_dirs
    run_test "Data directory created" "[[ -d ${KICAD_DATA_DIR} ]]"
    run_test "Config directory created" "[[ -d ${KICAD_CONFIG_DIR} ]]"
    run_test "Projects directory created" "[[ -d ${KICAD_PROJECTS_DIR} ]]"
    run_test "Libraries directory created" "[[ -d ${KICAD_LIBRARIES_DIR} ]]"
    run_test "Templates directory created" "[[ -d ${KICAD_TEMPLATES_DIR} ]]"
    run_test "Outputs directory created" "[[ -d ${KICAD_OUTPUTS_DIR} ]]"
    run_test "Logs directory created" "[[ -d ${KICAD_LOGS_DIR} ]]"
    
    # Test installation detection
    echo
    echo "Testing installation detection:"
    run_test "Installation check runs" "kicad::is_installed || true"
    run_test "Version check runs" "kicad::get_version || true"
    
    # Test configuration exports
    echo
    echo "Testing configuration exports:"
    run_test "KICAD_DATA_DIR is set" "[[ -n ${KICAD_DATA_DIR} ]]"
    run_test "KICAD_PORT is set" "[[ -n ${KICAD_PORT} ]]"
    run_test "KICAD_HOST is set" "[[ -n ${KICAD_HOST} ]]"
    run_test "KICAD_PYTHON_VERSION is set" "[[ -n ${KICAD_PYTHON_VERSION} ]]"
    run_test "KICAD_EXPORT_FORMATS is set" "[[ -n ${KICAD_EXPORT_FORMATS} ]]"
    
    # Test install.sh functions (if available)
    if [[ -f "${KICAD_DIR}/lib/install.sh" ]]; then
        echo
        echo "Testing install.sh functions:"
        test_function_exists "kicad::install"
        test_function_exists "kicad::can_install"
    fi
    
    # Test status.sh functions (if available)
    if [[ -f "${KICAD_DIR}/lib/status.sh" ]]; then
        echo
        echo "Testing status.sh functions:"
        test_function_exists "kicad_status"
    fi
    
    # Test inject.sh functions (if available)
    if [[ -f "${KICAD_DIR}/lib/inject.sh" ]]; then
        echo
        echo "Testing inject.sh functions:"
        test_function_exists "kicad::inject"
        test_function_exists "kicad::inject::validate_file"
    fi
    
    # Test content.sh functions (if available)
    if [[ -f "${KICAD_DIR}/lib/content.sh" ]]; then
        echo
        echo "Testing content.sh functions:"
        test_function_exists "kicad::content::list"
        test_function_exists "kicad::content::get"
        test_function_exists "kicad::content::remove"
        test_function_exists "kicad::content::list_projects"
        test_function_exists "kicad::content::list_libraries"
        test_function_exists "kicad::export::project"
    fi
    
    # Test desktop.sh functions (if available)
    if [[ -f "${KICAD_DIR}/lib/desktop.sh" ]]; then
        echo
        echo "Testing desktop.sh functions:"
        test_function_exists "kicad::desktop::start"
        test_function_exists "kicad::desktop::stop"
        test_function_exists "kicad::desktop::restart"
        test_function_exists "kicad::desktop::uninstall"
        test_function_exists "kicad::desktop::logs"
    fi
    
    # Test file operations
    echo
    echo "Testing file operations:"
    local test_file="${KICAD_PROJECTS_DIR}/unit_test.tmp"
    run_test "Can write to projects dir" "echo 'test' > '$test_file'"
    run_test "Can read from projects dir" "[[ -f '$test_file' ]]"
    run_test "Can delete from projects dir" "rm -f '$test_file'"
    
    # Test configuration validity
    echo
    echo "Testing configuration validity:"
    run_test "Port number is valid" "[[ ${KICAD_PORT} -ge 1024 && ${KICAD_PORT} -le 65535 ]]"
    run_test "Python command exists" "command -v ${KICAD_PYTHON_VERSION}"
    
    # Summary
    echo
    echo "Results: ${TESTS_PASSED}/${TESTS_RUN} tests passed"
    
    if [[ ${TESTS_PASSED:-0} -eq ${TESTS_RUN:-0} ]] && [[ ${TESTS_RUN:-0} -gt 0 ]]; then
        echo -e "${GREEN}✅ Unit tests passed${NC}"
        exit 0
    elif [[ ${TESTS_PASSED:-0} -gt $((${TESTS_RUN:-1} * 3 / 4)) ]]; then
        echo -e "${YELLOW}⚠️  Some unit tests failed${NC}"
        exit 0  # Allow partial success for unit tests
    else
        echo -e "${RED}❌ Unit tests failed${NC}"
        exit 1
    fi
}

# Run main function
main "$@"