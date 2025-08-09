#!/usr/bin/env bash
# Vrooli Test Helper
# Common utilities and setup for BATS tests

# Setup directory and source var.sh first
_HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${_HERE}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_SYSTEM_COMMANDS_FILE}"

# Test assertion helpers
assert_success() {
    if [[ $status -ne 0 ]]; then
        echo "Command failed with exit code $status"
        echo "Output: $output"
        return 1
    fi
}

assert_failure() {
    if [[ $status -eq 0 ]]; then
        echo "Expected command to fail, but it succeeded"
        echo "Output: $output"
        return 1
    fi
}

assert_output() {
    # Handle --partial flag first
    if [[ "$1" == "--partial" ]]; then
        local expected="$2"
        if [[ "$output" != *"$expected"* ]]; then
            echo "Expected output to contain: $expected"
            echo "Actual output: $output"
            return 1
        fi
    else
        local expected="$1"
        if [[ "$output" != "$expected" ]]; then
            echo "Expected output: $expected"
            echo "Actual output: $output"
            return 1
        fi
    fi
}

# Skip test helper
skip() {
    local reason="${1:-Test skipped}"
    # In BATS, we need a different approach for skips
    if [[ -n "${BATS_TEST_NAME:-}" ]]; then
        # We're in a BATS test - just return without failure
        return 0
    else
        echo "SKIP: $reason" >&3
        exit 0
    fi
}