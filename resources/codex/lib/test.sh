#!/usr/bin/env bash
# Codex Test Functions - Delegates to test/run-tests.sh

# Set script directory for sourcing
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
CODEX_DIR="${RESOURCE_DIR}"
CODEX_TEST_DIR="${CODEX_DIR}/test"

# Source required utilities
# shellcheck disable=SC1091
source "${CODEX_DIR}/lib/common.sh"

#######################################
# Smoke test - Delegates to test runner
# Returns:
#   0 if healthy, 1 if unhealthy
#######################################
codex::test::smoke() {
    # Delegate to the test runner
    if [[ -f "${CODEX_TEST_DIR}/run-tests.sh" ]]; then
        bash "${CODEX_TEST_DIR}/run-tests.sh" smoke
        return $?
    else
        log::error "Test runner not found: ${CODEX_TEST_DIR}/run-tests.sh"
        return 1
    fi
}

#######################################
# Integration test - Delegates to test runner
# Returns:
#   0 if all tests pass, 1 if any fail
#######################################
codex::test::integration() {
    # Delegate to the test runner
    if [[ -f "${CODEX_TEST_DIR}/run-tests.sh" ]]; then
        bash "${CODEX_TEST_DIR}/run-tests.sh" integration
        return $?
    else
        log::error "Test runner not found: ${CODEX_TEST_DIR}/run-tests.sh"
        return 1
    fi
}

#######################################
# Unit test - Delegates to test runner
# Returns:
#   0 if all tests pass, 1 if any fail
#######################################
codex::test::unit() {
    # Delegate to the test runner
    if [[ -f "${CODEX_TEST_DIR}/run-tests.sh" ]]; then
        bash "${CODEX_TEST_DIR}/run-tests.sh" unit
        return $?
    else
        log::error "Test runner not found: ${CODEX_TEST_DIR}/run-tests.sh"
        return 1
    fi
}

#######################################
# All tests - Delegates to test runner
# Returns:
#   0 if all tests pass, 1 if any fail
#######################################
codex::test::all() {
    # Delegate to the test runner
    if [[ -f "${CODEX_TEST_DIR}/run-tests.sh" ]]; then
        bash "${CODEX_TEST_DIR}/run-tests.sh" all
        return $?
    else
        log::error "Test runner not found: ${CODEX_TEST_DIR}/run-tests.sh"
        return 1
    fi
}