#!/usr/bin/env bash
# Common setup for all BATS tests
# This file should be sourced in the setup() function of all resource tests

# Ensure proper test environment
export BATS_TEST_TMPDIR="${BATS_TEST_TMPDIR:-$(mktemp -d)}"
export MOCK_RESPONSES_DIR="$BATS_TEST_TMPDIR/mock_responses"

# Create mock response directory
mkdir -p "$MOCK_RESPONSES_DIR"

# Determine the path to the shared mocks directory
if [[ -n "${BATS_TEST_FILENAME:-}" ]]; then
    # Running in BATS context
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    
    # Find the path to the shared mocks
    # Look for the mocks directory relative to current test
    MOCKS_DIR=""
    
    # Try different relative paths to find mocks directory
    for try_path in \
        "$SCRIPT_DIR/../../tests/bats-fixtures/mocks" \
        "$SCRIPT_DIR/../../../tests/bats-fixtures/mocks" \
        "$SCRIPT_DIR/../../../../tests/bats-fixtures/mocks" \
        "$SCRIPT_DIR/../../../../../tests/bats-fixtures/mocks" \
        "$SCRIPT_DIR/tests/bats-fixtures/mocks"; do
        
        if [[ -d "$try_path" ]]; then
            MOCKS_DIR="$try_path"
            break
        fi
    done
    
    # Fallback to absolute path if relative path search fails
    if [[ -z "$MOCKS_DIR" ]]; then
        MOCKS_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests/bats-fixtures/mocks"
    fi
else
    # Running outside BATS context
    MOCKS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/mocks" && pwd)"
fi

# Verify mocks directory exists
if [[ ! -d "$MOCKS_DIR" ]]; then
    echo "ERROR: Could not find mocks directory. Looked for: $MOCKS_DIR" >&2
    return 1
fi

# Source all mock files
source "$MOCKS_DIR/system_mocks.bash"
source "$MOCKS_DIR/mock_helpers.bash"
source "$MOCKS_DIR/resource_mocks.bash"

#######################################
# Convenient setup functions for tests
#######################################

# Setup standard test environment with basic defaults
setup_standard_mocks() {
    # Set basic environment variables commonly used in tests
    export FORCE="${FORCE:-no}"
    export YES="${YES:-no}"
    export SKIP_MODELS="${SKIP_MODELS:-no}"
    
    # Default output settings
    export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
    export QUIET="${QUIET:-no}"
    
    # Set up network as online by default
    mock::network::set_online
}

# Setup a resource with common defaults
setup_resource_test() {
    local resource="$1"
    local state="${2:-healthy}"
    local port="${3:-}"
    
    setup_standard_mocks
    
    # Use resource-specific setup if available
    if declare -f "mock::${resource}::setup" >/dev/null 2>&1; then
        "mock::${resource}::setup" "$state"
    else
        # Fallback to generic resource setup
        mock::resource::setup "$resource" "installed_running" "$port"
    fi
}

# Cleanup function for tests
cleanup_mocks() {
    if [[ -d "${MOCK_RESPONSES_DIR:-}" ]]; then
        rm -rf "$MOCK_RESPONSES_DIR"
    fi
}

# Export setup functions
export -f setup_standard_mocks setup_resource_test cleanup_mocks

#######################################
# Validation helpers
#######################################

# Validate that a command was called with specific arguments
assert_command_called() {
    local command="$1"
    local expected_args="$2"
    
    # This is a placeholder for command validation
    # In a real implementation, you might log command calls to files
    # and validate them here
    return 0
}

# Check if a mock response was used
assert_mock_response_used() {
    local mock_file="$1"
    
    if [[ -f "$MOCK_RESPONSES_DIR/$mock_file" ]]; then
        return 0
    else
        echo "Mock response file not found: $mock_file" >&2
        return 1
    fi
}

export -f assert_command_called assert_mock_response_used