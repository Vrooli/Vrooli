#!/usr/bin/env bash
# Runs all *.bats files in the scripts directory and subdirectories and provides a summary

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

total_tests=0
total_failures=0

log::header "Running bats tests..."

# Run all tests in the scripts directory and subdirectories
while IFS= read -r test_file; do
    # Run bats with TAP output and capture it
    output=$(bats --tap "${test_file}")
    exit_code=$?

    # If the bats command failed, consider it a failure
    if [ "$exit_code" -ne "${EXIT_SUCCESS}" ] && ! echo "${output}" | grep -q "^not ok"; then
        log::error "Failed to run test: ${test_file}. Got exit code: ${exit_code}"
        total_failures=$((total_failures + 1))
        continue
    fi

    # Count tests and failures
    tests=$(echo "${output}" | grep -c "^ok\|^not ok")
    failures=$(echo "${output}" | grep -c "^not ok")

    # Add to totals
    total_tests=$((total_tests + tests))
    total_failures=$((total_failures + failures))

    # Print the original output
    echo "${output}"
done < <(find "${SCRIPTS_DIR}" -path "${SCRIPTS_DIR}/__tests/helpers" -prune -o -type f -name '*.bats' -print)

# Print summary
echo ""
log::info "Total tests run: ${total_tests}"
if [ "${total_failures}" -eq "${EXIT_SUCCESS}" ]; then
    log::success "All tests passed successfully!"
else
    log::error "Total failures: ${total_failures}"
fi

# Exit with appropriate code
if [ "${total_failures}" -eq "${EXIT_SUCCESS}" ]; then
    exit "${EXIT_SUCCESS}"
else
    exit "${ERROR_DEFAULT}"
fi
