#!/usr/bin/env bash
# Runs all *.bats files in parallel and provides a summary
# This helps reduce test execution time significantly

# Determine this script directory and set up library path for BATS
TESTS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# Ensure BATS helper libraries are discoverable by bats_load_library
export BATS_LIB_PATH="${TESTS_DIR}/helpers:${BATS_LIB_PATH-}"

# shellcheck disable=SC1091
source "${TESTS_DIR}/../helpers/utils/exit_codes.sh"
# shellcheck disable=SC1091
source "${TESTS_DIR}/../helpers/utils/log.sh"

# Disable exit on error to allow handling test failures manually
set +e

SCRIPTS_DIR=$(dirname "${TESTS_DIR}")

# Configuration - auto-detect optimal job count for powerful machines
if [[ -z "${PARALLEL_JOBS:-}" ]]; then
    # Get CPU count
    CPU_COUNT=$(nproc 2>/dev/null || echo 4)
    
    # For powerful machines, use more aggressive parallelization
    if [[ $CPU_COUNT -ge 16 ]]; then
        PARALLEL_JOBS=32  # High-end machines
    elif [[ $CPU_COUNT -ge 8 ]]; then
        PARALLEL_JOBS=16  # Mid-range machines
    elif [[ $CPU_COUNT -ge 4 ]]; then
        PARALLEL_JOBS=8   # Standard machines
    else
        PARALLEL_JOBS=4   # Low-end machines
    fi
    
    log::info "Auto-detected ${CPU_COUNT} CPUs, using ${PARALLEL_JOBS} parallel jobs"
else
    log::info "Using custom PARALLEL_JOBS=${PARALLEL_JOBS}"
fi
TEMP_DIR="/tmp/bats-parallel-$$"
mkdir -p "${TEMP_DIR}"

# Initialize counters
total_tests=0
total_failures=0
total_files=0
failed_files=()

log::header "Running bats tests in parallel (${PARALLEL_JOBS} jobs)..."

# Function to run a single test file
run_test_file() {
    local test_file="$1"
    local output_file="${TEMP_DIR}/$(basename "$test_file").out"
    local status_file="${TEMP_DIR}/$(basename "$test_file").status"
    
    # Run bats with TAP output and capture it
    bats --tap "${test_file}" > "${output_file}" 2>&1
    echo $? > "${status_file}"
}

# Export the function so it can be used by parallel
export -f run_test_file
export TEMP_DIR

# Find all test files
mapfile -t test_files < <(find "${SCRIPTS_DIR}" -path "${SCRIPTS_DIR}/__tests/helpers" -prune -o -type f -name '*.bats' -print)
total_files=${#test_files[@]}

log::info "Found ${total_files} test files to run"

# Run tests in parallel using xargs
printf '%s\n' "${test_files[@]}" | xargs -P "${PARALLEL_JOBS}" -I {} bash -c 'run_test_file "$@"' _ {}

# Process results
log::info "Processing test results..."

for test_file in "${test_files[@]}"; do
    output_file="${TEMP_DIR}/$(basename "$test_file").out"
    status_file="${TEMP_DIR}/$(basename "$test_file").status"
    
    if [[ -f "${output_file}" ]] && [[ -f "${status_file}" ]]; then
        exit_code=$(cat "${status_file}")
        output=$(cat "${output_file}")
        
        # Count tests and failures
        tests=$(echo "${output}" | grep -c "^ok\|^not ok" || echo 0)
        failures=$(echo "${output}" | grep -c "^not ok" || echo 0)
        
        # Add to totals
        total_tests=$((total_tests + tests))
        total_failures=$((total_failures + failures))
        
        # Track failed files
        if [[ "${failures}" -gt 0 ]] || [[ "${exit_code}" -ne 0 ]]; then
            failed_files+=("${test_file}")
        fi
        
        # Show file result
        if [[ "${failures}" -gt 0 ]]; then
            log::error "$(basename "${test_file}"): ${tests} tests, ${failures} failures"
        else
            log::success "$(basename "${test_file}"): ${tests} tests, all passed"
        fi
    else
        log::error "Failed to get results for ${test_file}"
        failed_files+=("${test_file}")
    fi
done

# Clean up temp directory
rm -rf "${TEMP_DIR}"

# Print summary
echo ""
log::info "Test Summary:"
log::info "  Total files: ${total_files}"
log::info "  Total tests: ${total_tests}"

if [ "${total_failures}" -eq 0 ]; then
    log::success "All tests passed successfully!"
else
    log::error "Total failures: ${total_failures}"
    log::error "Failed files:"
    for file in "${failed_files[@]}"; do
        echo "  - ${file}"
    done
fi

# Exit with appropriate code
if [ "${total_failures}" -eq 0 ]; then
    exit "${EXIT_SUCCESS}"
else
    exit "${ERROR_DEFAULT}"
fi