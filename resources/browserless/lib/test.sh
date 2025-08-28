#!/usr/bin/env bash
#
# Browserless test runner
#
set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="${SCRIPT_DIR}/../test"

# Source common utilities
source "${SCRIPT_DIR}/common.sh"

# Run tests
function run_tests() {
    echo "🧪 Running browserless tests..."
    
    # Check if browserless is running
    if ! is_running; then
        echo "❌ Error: Browserless is not running. Please start it first with: browserless start"
        exit 1
    fi
    
    # Run integration tests if they exist
    if [[ -f "$TEST_DIR/integration-test.sh" ]]; then
        echo "  Running integration tests..."
        bash "$TEST_DIR/integration-test.sh"
    else
        echo "⚠️ No integration tests found at $TEST_DIR/integration-test.sh"
    fi
    
    # Run bats tests if they exist
    if command -v bats >/dev/null 2>&1; then
        if [[ -f "${SCRIPT_DIR}/../inject.bats" ]]; then
            echo "  Running bats tests..."
            bats "${SCRIPT_DIR}/../inject.bats"
        fi
        if [[ -f "${SCRIPT_DIR}/../manage.bats" ]]; then
            bats "${SCRIPT_DIR}/../manage.bats"
        fi
    else
        echo "ℹ️ Bats not installed, skipping unit tests"
    fi
    
    echo "✅ Tests completed"
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run_tests "$@"
fi