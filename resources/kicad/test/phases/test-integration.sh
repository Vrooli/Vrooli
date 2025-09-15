#!/usr/bin/env bash
################################################################################
# KiCad Integration Test - v2.0 Contract Compliant
# 
# End-to-end functionality testing for KiCad resource
# Must complete within 120 seconds
#
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KICAD_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${KICAD_DIR}/../.." && pwd)"

# Source utilities and common functions
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${KICAD_DIR}/lib/common.sh"
source "${KICAD_DIR}/lib/inject.sh" 2>/dev/null || true
source "${KICAD_DIR}/lib/content.sh" 2>/dev/null || true
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

# Extended test function with output capture
run_test_with_output() {
    local test_name="$1"
    local test_command="$2"
    local expected_pattern="$3"
    
    ((TESTS_RUN++)) || true
    echo -n "  Testing ${test_name}... "
    
    local output
    if output=$(eval "$test_command" 2>&1); then
        if [[ -z "$expected_pattern" ]] || echo "$output" | grep -q "$expected_pattern"; then
            echo -e "${GREEN}✓${NC}"
            ((TESTS_PASSED++)) || true
            return 0
        else
            echo -e "${RED}✗ (output mismatch)${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗${NC}"
        return 1
    fi
}

# Main integration test
main() {
    echo "KiCad Integration Tests"
    echo "======================="
    
    # Initialize environment
    echo "Initializing test environment..."
    kicad::init_dirs
    
    # Test 1: CLI command structure
    run_test "CLI help command" "${KICAD_DIR}/cli.sh help"
    
    # Test 2: Status command
    run_test "Status command" "${KICAD_DIR}/cli.sh status"
    
    # Test 3: Info command
    run_test_with_output "Info command" "${KICAD_DIR}/cli.sh info --json" "startup_order"
    
    # Test 4: Manage subcommands exist
    run_test_with_output "Manage help" "${KICAD_DIR}/cli.sh manage help 2>&1 || true" "install"
    
    # Test 5: Test subcommands exist
    run_test_with_output "Test help" "${KICAD_DIR}/cli.sh test help 2>&1 || true" "smoke"
    
    # Test 6: Content subcommands exist
    run_test_with_output "Content help" "${KICAD_DIR}/cli.sh content help 2>&1 || true" "list"
    
    # Test 7: Create a test project file
    local test_project="${KICAD_PROJECTS_DIR}/test_project.kicad_pro"
    run_test "Create test project" "echo '{\"meta\":{\"version\":1}}' > '$test_project'"
    
    # Test 8: List projects (should find test project)
    if type -t kicad::content::list_projects &>/dev/null; then
        run_test "List projects" "kicad::content::list_projects"
    else
        echo -e "  Testing List projects... ${YELLOW}SKIPPED (function not found)${NC}"
    fi
    
    # Test 9: Check example file exists
    run_test "Example file exists" "[[ -f ${KICAD_DIR}/examples/led-blinker.kicad_sch ]]"
    
    # Test 10: Import example file (if inject function exists)
    if type -t kicad::inject &>/dev/null; then
        run_test "Import example" "kicad::inject '${KICAD_DIR}/examples/led-blinker.kicad_sch'"
    else
        echo -e "  Testing Import example... ${YELLOW}SKIPPED (inject not implemented)${NC}"
    fi
    
    # Test 11: Test directory structure
    run_test "Templates directory" "[[ -d ${KICAD_TEMPLATES_DIR} ]]"
    
    # Test 12: Test log directory
    run_test "Logs directory" "[[ -d ${KICAD_LOGS_DIR} ]]"
    
    # Test 13: Configuration validation
    run_test "Runtime config valid" "jq '.' ${KICAD_DIR}/config/runtime.json"
    
    # Test 14: Schema validation
    run_test "Schema valid" "jq '.' ${KICAD_DIR}/config/schema.json"
    
    # Test 15: Python API check (if KiCad is actually installed)
    if command -v kicad &>/dev/null || command -v kicad-cli &>/dev/null; then
        run_test "Python API" "kicad::check_python_api"
    else
        echo -e "  Testing Python API... ${YELLOW}SKIPPED (KiCad not installed)${NC}"
    fi
    
    # Test P2 features (cloud backup, simulation, autoroute)
    echo
    echo "Testing P2 Features:"
    run_test "Simulation models creation" \
        "${KICAD_DIR}/cli.sh simulation models"
    
    run_test "Backup commands available" \
        "${KICAD_DIR}/cli.sh backup --help"
    
    run_test "Autoroute commands available" \
        "${KICAD_DIR}/cli.sh autoroute --help"
    
    run_test "Version control commands available" \
        "${KICAD_DIR}/cli.sh version --help"
    
    # Cleanup test files
    echo
    echo "Cleaning up test files..."
    rm -f "$test_project"
    rm -rf "${KICAD_DATA_DIR}/libraries/spice_models"
    # Clean up any test artifacts from simulation tests
    rm -f "${KICAD_DIR}"/*.net 2>/dev/null || true
    rm -f "${KICAD_DIR}"/*.log 2>/dev/null || true
    
    # Summary
    echo
    echo "Results: ${TESTS_PASSED}/${TESTS_RUN} tests passed"
    
    if [[ ${TESTS_PASSED:-0} -eq ${TESTS_RUN:-0} ]] && [[ ${TESTS_RUN:-0} -gt 0 ]]; then
        echo -e "${GREEN}✅ Integration tests passed${NC}"
        exit 0
    elif [[ ${TESTS_PASSED:-0} -gt $((${TESTS_RUN:-1} / 2)) ]]; then
        echo -e "${YELLOW}⚠️  Some integration tests failed${NC}"
        exit 1
    else
        echo -e "${RED}❌ Integration tests failed${NC}"
        exit 1
    fi
}

# Run main function
main "$@"