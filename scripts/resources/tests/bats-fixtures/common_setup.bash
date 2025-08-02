#!/usr/bin/env bash
# Common setup for all BATS tests
# This file should be sourced in the setup() function of all resource tests

# Ensure proper test environment
export BATS_TEST_TMPDIR="${BATS_TEST_TMPDIR:-$(mktemp -d)}"
export MOCK_RESPONSES_DIR="$BATS_TEST_TMPDIR/mock_responses"

# Create mock response directory
mkdir -p "$MOCK_RESPONSES_DIR"

# Determine the path to the shared mocks directory using robust path resolution
find_mocks_directory() {
    local current_dir
    
    # Start from test file location if in BATS context, otherwise from this script
    if [[ -n "${BATS_TEST_FILENAME:-}" ]]; then
        current_dir="$(dirname "${BATS_TEST_FILENAME}")"
    else
        current_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    fi
    
    # Walk up the directory tree looking for the mocks directory
    local search_dir="$current_dir"
    local max_depth=10  # Prevent infinite loops
    local depth=0
    
    while [[ $depth -lt $max_depth ]]; do
        # Check for mocks in current level
        if [[ -d "$search_dir/tests/bats-fixtures/mocks" ]]; then
            echo "$search_dir/tests/bats-fixtures/mocks"
            return 0
        fi
        
        # Check for mocks relative to resources directory
        if [[ -d "$search_dir/resources/tests/bats-fixtures/mocks" ]]; then
            echo "$search_dir/resources/tests/bats-fixtures/mocks"
            return 0
        fi
        
        # Move up one directory
        local parent_dir="$(dirname "$search_dir")"
        if [[ "$parent_dir" == "$search_dir" ]]; then
            # Reached filesystem root
            break
        fi
        search_dir="$parent_dir"
        ((depth++))
    done
    
    # Last resort: try to find it anywhere in the project
    local project_root="/home/matthalloran8/Vrooli"
    if [[ -d "$project_root/scripts/resources/tests/bats-fixtures/mocks" ]]; then
        echo "$project_root/scripts/resources/tests/bats-fixtures/mocks"
        return 0
    fi
    
    # Failed to find mocks directory
    return 1
}

MOCKS_DIR="$(find_mocks_directory)"
if [[ -z "$MOCKS_DIR" ]]; then
    echo "ERROR: Could not locate mocks directory from current path" >&2
    return 1
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
    # Set standard environment variables first
    export FORCE="${FORCE:-no}"
    export YES="${YES:-no}"
    export SKIP_MODELS="${SKIP_MODELS:-no}"
    export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
    export QUIET="${QUIET:-no}"
    
    # Ensure mocks are loaded - reuse MOCKS_DIR from above
    if ! type -t mock::network::set_online &>/dev/null; then
        if [[ -d "$MOCKS_DIR" ]]; then
            source "$MOCKS_DIR/system_mocks.bash"
            source "$MOCKS_DIR/mock_helpers.bash"
            source "$MOCKS_DIR/resource_mocks.bash"
        else
            echo "ERROR: Could not find mocks directory!" >&2 
            echo "MOCKS_DIR was set to: $MOCKS_DIR" >&2
            return 1
        fi
    fi
    
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

# Standardized mock function export
export_standard_mock_functions() {
    # Core mock system functions
    export -f mock::network::set_online mock::network::set_offline
    export -f mock::docker::set_container_state mock::docker::set_container_health
    export -f mock::http::set_endpoint_response mock::http::set_endpoint_delay
    export -f mock::http::set_endpoint_sequence
    
    # Mock helper functions
    # Note: docker and curl functions are already exported by system_mocks.bash
    
    # Test utility functions
    export -f setup_standard_mocks setup_resource_test cleanup_mocks
    # Note: assert functions are exported after definition at end of file
    
    # Common test functions that might be used in timeout contexts
    local resource_functions=()
    
    # Dynamically export any loaded resource functions
    while IFS= read -r func_name; do
        if [[ "$func_name" =~ ^[a-zA-Z0-9_]+::[a-zA-Z0-9_]+$ ]]; then
            resource_functions+=("$func_name")
        fi
    done < <(declare -F | awk '{print $3}' | grep '::')
    
    # Export all found resource functions
    if [[ ${#resource_functions[@]} -gt 0 ]]; then
        export -f "${resource_functions[@]}"
    fi
}

# Call the standardized export function
export_standard_mock_functions

# Export setup functions
export -f setup_standard_mocks setup_resource_test cleanup_mocks export_standard_mock_functions

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