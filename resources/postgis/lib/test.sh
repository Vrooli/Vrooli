#!/bin/bash
# PostGIS Test Runner

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
POSTGIS_TEST_LIB_DIR="${APP_ROOT}/resources/postgis/lib"

# Source common functions
source "${POSTGIS_TEST_LIB_DIR}/common.sh"

#######################################
# Run PostGIS integration tests and save results
# Globals:
#   POSTGIS_DATA_DIR
# Returns:
#   0 on success, 1 on failure
#######################################
postgis_run_tests() {
    # Initialize directories
    postgis_init_dirs
    
    local test_file="${POSTGIS_TEST_LIB_DIR}/../test/integration.bats"
    local results_file="${POSTGIS_DATA_DIR}/test_results.json"
    
    # Check if test file exists
    if [[ ! -f "$test_file" ]]; then
        echo "Error: Test file not found: $test_file" >&2
        return 1
    fi
    
    # Run tests with timeout
    local test_output
    local test_exit_code
    
    echo "Running PostGIS integration tests..."
    test_output=$(timeout 60 bats "$test_file" 2>&1)
    test_exit_code=$?
    
    # Parse test results
    local total_tests=0
    local passed_tests=0
    
    # Extract test counts from bats output
    if echo "$test_output" | grep -q "^[0-9]\+\.\.[0-9]\+$"; then
        # Format: 1..6 (TAP format)
        total_tests=$(echo "$test_output" | grep -o "^[0-9]\+\.\.[0-9]\+$" | cut -d. -f3)
        passed_tests=$(echo "$test_output" | grep -c "^ok [0-9]\+")
    fi
    
    # Create JSON results
    cat > "$results_file" <<EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "total": $total_tests,
  "passed": $passed_tests,
  "failed": $((total_tests - passed_tests)),
  "exit_code": $test_exit_code,
  "resource": "postgis"
}
EOF
    
    # Display results
    echo
    echo "Test Results:"
    echo "  Total: $total_tests"
    echo "  Passed: $passed_tests"
    echo "  Failed: $((total_tests - passed_tests))"
    echo
    
    if [[ $test_exit_code -eq 0 ]]; then
        echo "✅ All tests passed!"
        return 0
    else
        echo "❌ Some tests failed"
        echo
        echo "Test output:"
        echo "$test_output"
        return 1
    fi
}

# Export function
export -f postgis_run_tests