#!/usr/bin/env bash
# Standard Test Setup for Resource Bats Tests
# Provides consistent test environment initialization across all resources

# Determine paths relative to test file
_determine_paths() {
    # Start from test file directory
    local test_file_dir="$(dirname "${BATS_TEST_FILENAME}")"
    local current_dir="$test_file_dir"
    
    # Walk up directory tree to find resources directory 
    # Look for the main common.sh that has resource management functions
    while [[ "$current_dir" != "/" ]]; do
        if [[ -f "$current_dir/common.sh" ]] && [[ -f "$current_dir/port-registry.sh" ]]; then
            # This is likely the resources root directory
            break
        fi
        current_dir="$(dirname "$current_dir")"
    done
    
    if [[ ! -f "$current_dir/common.sh" ]] || [[ ! -f "$current_dir/port-registry.sh" ]]; then
        echo "ERROR: Could not find resources directory from ${BATS_TEST_FILENAME}" >&2
        echo "ERROR: Started search from: $test_file_dir" >&2
        echo "ERROR: Looking for directory with both common.sh and port-registry.sh" >&2
        return 1
    fi
    
    # Export discovered paths
    export RESOURCES_DIR="$current_dir"
    
    # For tests in lib/ subdirectory, resource dir is parent of lib
    if [[ "$(basename "$test_file_dir")" == "lib" ]]; then
        export RESOURCE_DIR="$(dirname "$test_file_dir")"
    else
        # For other test locations, adapt as needed
        export RESOURCE_DIR="$test_file_dir"
    fi
    
    export RESOURCE_NAME="$(basename "$RESOURCE_DIR")"
    export BATS_FIXTURES_DIR="$RESOURCES_DIR/test/bats-fixtures"
}

# Load core testing utilities and mocks
load_test_utilities() {
    # Ensure paths are set
    if [[ -z "${RESOURCES_DIR:-}" ]]; then
        _determine_paths || return 1
    fi
    
    # Load mocks first (before any other sourcing)
    if [[ -f "$BATS_FIXTURES_DIR/mocks/system_mocks.bash" ]]; then
        source "$BATS_FIXTURES_DIR/mocks/system_mocks.bash"
    else
        echo "ERROR: system_mocks.bash not found at $BATS_FIXTURES_DIR/mocks/" >&2
        return 1
    fi
    
    if [[ -f "$BATS_FIXTURES_DIR/mocks/mock_helpers.bash" ]]; then
        source "$BATS_FIXTURES_DIR/mocks/mock_helpers.bash"
    else
        echo "ERROR: mock_helpers.bash not found" >&2
        return 1
    fi
    
    if [[ -f "$BATS_FIXTURES_DIR/mocks/resource_mocks.bash" ]]; then
        source "$BATS_FIXTURES_DIR/mocks/resource_mocks.bash"
    else
        echo "ERROR: resource_mocks.bash not found" >&2
        return 1
    fi
    
    # Load common resource functions
    if [[ -f "$RESOURCES_DIR/common.sh" ]]; then
        source "$RESOURCES_DIR/common.sh"
    else
        echo "ERROR: common.sh not found at $RESOURCES_DIR/" >&2
        return 1
    fi
    
    # Load test helpers if available
    if [[ -f "$BATS_FIXTURES_DIR/helpers/assertions.bash" ]]; then
        source "$BATS_FIXTURES_DIR/helpers/assertions.bash"
    fi
    
    if [[ -f "$BATS_FIXTURES_DIR/helpers/test_utilities.bash" ]]; then
        source "$BATS_FIXTURES_DIR/helpers/test_utilities.bash"
    fi
}

# Load all library files for a resource
load_resource_libraries() {
    local resource_name="${1:-$RESOURCE_NAME}"
    local lib_dir="$RESOURCE_DIR/lib"
    
    if [[ ! -d "$lib_dir" ]]; then
        echo "WARNING: No lib directory found at $lib_dir" >&2
        return 0
    fi
    
    # Define standard load order (dependencies first)
    local load_order=(
        "config.sh"      # Configuration constants
        "common.sh"      # Resource-specific common functions
        "docker.sh"      # Docker management
        "api.sh"         # API interactions
        "install.sh"     # Installation functions
        "status.sh"      # Status checking
        "usage.sh"       # Usage information
    )
    
    # Load files in order if they exist
    for lib_file in "${load_order[@]}"; do
        if [[ -f "$lib_dir/$lib_file" ]]; then
            source "$lib_dir/$lib_file"
        fi
    done
    
    # Load any additional .sh files not in standard order
    for lib_file in "$lib_dir"/*.sh; do
        local basename="$(basename "$lib_file")"
        if [[ ! " ${load_order[@]} " =~ " ${basename} " ]]; then
            source "$lib_file"
        fi
    done
}

# Load configuration files
load_resource_config() {
    local resource_name="${1:-$RESOURCE_NAME}"
    local config_dir="$RESOURCE_DIR/config"
    
    if [[ -d "$config_dir" ]]; then
        # Load defaults first
        [[ -f "$config_dir/defaults.sh" ]] && source "$config_dir/defaults.sh"
        
        # Load messages
        [[ -f "$config_dir/messages.sh" ]] && source "$config_dir/messages.sh"
        
        # Load any other config files
        for config_file in "$config_dir"/*.sh; do
            local basename="$(basename "$config_file")"
            if [[ "$basename" != "defaults.sh" ]] && [[ "$basename" != "messages.sh" ]]; then
                source "$config_file"
            fi
        done
    fi
}

# Standard setup function for tests
standard_test_setup() {
    local resource_name="${1:-}"
    
    # Set up test environment
    export BATS_TEST_TMPDIR="${BATS_TEST_TMPDIR:-$(mktemp -d)}"
    export MOCK_RESPONSES_DIR="$BATS_TEST_TMPDIR/mock_responses"
    mkdir -p "$MOCK_RESPONSES_DIR"
    
    # Determine paths
    _determine_paths || return 1
    
    # Override resource name if provided
    [[ -n "$resource_name" ]] && export RESOURCE_NAME="$resource_name"
    
    # Load all required components
    load_test_utilities || return 1
    load_resource_config || return 0  # Config is optional
    load_resource_libraries || return 0  # Libraries might be optional for some tests
    
    # Set standard test environment variables
    export YES="no"                    # Disable auto-confirmation
    export LOG_LEVEL="error"          # Reduce noise in tests
    export DOCKER_HOST="${DOCKER_HOST:-unix:///var/run/docker.sock}"
    
    # Initialize mock state
    mock::cleanup
    
    # Set up standard mocked commands that should always succeed
    mock::network::set_online
    
    return 0
}

# Standard teardown function
standard_test_teardown() {
    # Clean up mock responses
    [[ -d "$MOCK_RESPONSES_DIR" ]] && rm -rf "$MOCK_RESPONSES_DIR"
    
    # Clean up test temporary directory if it exists and is safe to remove
    if [[ -n "${BATS_TEST_TMPDIR:-}" ]] && [[ -d "$BATS_TEST_TMPDIR" ]] && [[ "$BATS_TEST_TMPDIR" =~ ^/tmp/ ]]; then
        rm -rf "$BATS_TEST_TMPDIR"
    fi
    
    # Resource-specific cleanup can be added by tests after calling this
}

# Helper to skip test if dependencies are missing
require_commands() {
    local missing=()
    
    for cmd in "$@"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing+=("$cmd")
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        skip "Required commands not available: ${missing[*]}"
    fi
}

# Helper to run test with specific mock state
with_mock_state() {
    local resource="$1"
    local state="$2"
    shift 2
    
    # Set up mock state
    if declare -f "mock::${resource}::setup" >/dev/null 2>&1; then
        "mock::${resource}::setup" "$state"
    else
        mock::resource::setup "$resource" "$state"
    fi
    
    # Run the test commands
    "$@"
}

# Export functions for use in tests
export -f standard_test_setup standard_test_teardown
export -f load_test_utilities load_resource_libraries load_resource_config
export -f require_commands with_mock_state
export -f _determine_paths