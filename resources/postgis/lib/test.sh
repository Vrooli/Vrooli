#!/bin/bash
# PostGIS Test Functions - v2.0 Compliant

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
POSTGIS_TEST_LIB_DIR="${APP_ROOT}/resources/postgis/lib"
POSTGIS_TEST_DIR="${APP_ROOT}/resources/postgis/test"

# Source common functions
source "${POSTGIS_TEST_LIB_DIR}/common.sh"

#######################################
# Run PostGIS tests - delegates to test/run-tests.sh
# Arguments:
#   $1 - Test phase (smoke|integration|unit|all)
# Returns:
#   0 on success, 1 on failure
#######################################
postgis_run_tests() {
    local phase="${1:-all}"
    
    # Initialize directories
    postgis_init_dirs
    
    # Delegate to the main test runner
    if [[ -f "${POSTGIS_TEST_DIR}/run-tests.sh" ]]; then
        "${POSTGIS_TEST_DIR}/run-tests.sh" "$phase"
        return $?
    else
        echo "Error: Test runner not found: ${POSTGIS_TEST_DIR}/run-tests.sh" >&2
        return 1
    fi
}

#######################################
# Run smoke tests specifically
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::test::smoke() {
    postgis_run_tests "smoke"
}

#######################################
# Run integration tests specifically
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::test::integration() {
    postgis_run_tests "integration"
}

#######################################
# Run unit tests specifically
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::test::unit() {
    postgis_run_tests "unit"
}

#######################################
# Run all tests
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::test::all() {
    postgis_run_tests "all"
}

#######################################
# Run extended tests (core + P2 features)
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::test::extended() {
    postgis_run_tests "extended"
}

#######################################
# Run geocoding tests (P2 feature)
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::test::geocoding() {
    postgis_run_tests "geocoding"
}

#######################################
# Run spatial analysis tests (P2 feature)
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::test::spatial() {
    postgis_run_tests "spatial"
}

#######################################
# Run visualization tests (P2 feature)
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::test::visualization() {
    postgis_run_tests "visualization"
}

# Export functions
export -f postgis_run_tests
export -f postgis::test::smoke
export -f postgis::test::integration
export -f postgis::test::unit
export -f postgis::test::all
export -f postgis::test::extended
export -f postgis::test::geocoding
export -f postgis::test::spatial
export -f postgis::test::visualization