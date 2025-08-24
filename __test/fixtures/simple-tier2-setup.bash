#!/usr/bin/env bash
# Simple Tier 2 Mock Setup for BATS tests
# Minimal setup without the full Vrooli test infrastructure

# Determine root directory
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../.." && builtin pwd)}"
MOCK_BASE_DIR="${APP_ROOT}/__test/mocks"
export APP_ROOT MOCK_BASE_DIR

# Load the test helper
source "${MOCK_BASE_DIR}/test_helper.sh"

# Simple setup function for BATS tests
vrooli_setup_unit_test() {
    echo "[SETUP] Simple Tier 2 setup for BATS tests" >&2
    return 0
}

# Simple cleanup function  
vrooli_cleanup_test() {
    echo "[CLEANUP] Simple cleanup for BATS tests" >&2
    return 0
}

# Export functions
export -f vrooli_setup_unit_test
export -f vrooli_cleanup_test

echo "[SETUP] Simple Tier 2 setup loaded successfully" >&2