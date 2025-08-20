#!/bin/bash

# OBS Studio Test Runner

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OBS_DATA_DIR="${HOME}/.vrooli/obs-studio"

# Run tests
run_tests() {
    echo "[INFO] Running OBS Studio integration tests..."
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local test_status="passed"
    
    # Check if bats is installed
    if ! command -v bats &>/dev/null; then
        echo "[WARNING] Bats test framework not installed. Skipping tests."
        echo "not_run" > "$OBS_DATA_DIR/.test-status"
        return 0
    fi
    
    # Run tests with timeout
    if timeout 60 bats "$TEST_DIR/integration.bats" 2>&1; then
        echo "[SUCCESS] All tests passed"
        test_status="passed"
    else
        echo "[WARNING] Some tests failed"
        test_status="failed"
    fi
    
    # Save test results
    echo "$timestamp" > "$OBS_DATA_DIR/.last-test"
    echo "$test_status" > "$OBS_DATA_DIR/.test-status"
    
    return 0
}

# Main
if [[ "${1:-}" == "--quiet" ]]; then
    run_tests >/dev/null 2>&1
else
    run_tests
fi