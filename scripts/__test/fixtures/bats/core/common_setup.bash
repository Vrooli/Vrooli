#!/usr/bin/env bash
# Unified Common Setup for All BATS Tests
# This file consolidates and improves the functionality from both existing common_setup.bash files
# providing a consistent, fast, and reliable test environment setup

# Prevent duplicate loading
if [[ "${COMMON_SETUP_LOADED:-}" == "true" ]]; then
    return 0
fi
export COMMON_SETUP_LOADED="true"

# Load the centralized path resolver first
# This ensures consistent path resolution across all fixtures
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/path_resolver.bash"

# Initialize test environment using centralized path functions
export BATS_TEST_TMPDIR="$(vrooli_test_tmpdir)"
export MOCK_RESPONSES_DIR="${BATS_TEST_TMPDIR}/mock_responses"
mkdir -p "${MOCK_RESPONSES_DIR}"

# Load core infrastructure using the path resolver
vrooli_source_fixture "core/error_handling.bash"
vrooli_source_fixture "core/benchmarking.bash"
vrooli_source_fixture "mocks/mock_registry.bash"
vrooli_source_fixture "core/assertions.bash"

#######################################
# Enhanced setup functions with improved performance and reliability
#######################################

#######################################
# Setup standard test environment with basic defaults
# This is the primary function most tests should use
# Globals: Standard environment variables
# Arguments: None
# Returns: 0 on success
#######################################
setup_standard_mocks() {
    echo "[COMMON_SETUP] Setting up standard test environment"
    
    # Set standard environment variables
    export FORCE="${FORCE:-no}"
    export YES="${YES:-no}"
    export SKIP_MODELS="${SKIP_MODELS:-no}"
    export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
    export QUIET="${QUIET:-no}"
    export DRY_RUN="${DRY_RUN:-no}"
    
    # Test isolation
    export TEST_NAMESPACE="test_$$_${RANDOM}"
    export TEST_PORT_BASE=$((8000 + (RANDOM % 1000)))
    
    # Use the new mock registry for minimal setup
    mock::setup_minimal
    
    echo "[COMMON_SETUP] Standard test environment ready"
}

#######################################
# Setup a resource test environment with optimized loading
# Arguments: 
#   $1 - resource name (e.g., "ollama", "n8n")
#   $2 - resource state (optional, defaults to "healthy")
#######################################
setup_resource_test() {
    local resource="$1"
    local state="${2:-healthy}"
    
    if ! validate_required_param "resource" "$resource" "COMMON_SETUP"; then
        return 1
    fi
    
    echo "[COMMON_SETUP] Setting up test environment for resource: $resource"
    
    # Initialize benchmarking system if enabled
    benchmark::init
    
    # Use the new mock registry for resource setup
    mock::setup_resource "$resource"
    
    # Configure resource state
    if declare -f "mock::${resource}::setup" >/dev/null 2>&1; then
        "mock::${resource}::setup" "$state"
    else
        echo "[COMMON_SETUP] WARNING: No specific mock setup found for $resource, using generic setup"
        mock::setup_generic_resource "$resource" "$state"
    fi
    
    echo "[COMMON_SETUP] Resource test environment ready for $resource"
}

#######################################
# Setup integration test environment for multiple resources
# Arguments: $@ - list of resource names
#######################################
setup_integration_test() {
    local resources=("$@")
    
    if [[ ${#resources[@]} -eq 0 ]]; then
        common_setup_error "At least one resource required for integration test"
        return 1
    fi
    
    echo "[COMMON_SETUP] Setting up integration test environment for: ${resources[*]}"
    
    # Use the new mock registry for integration setup
    mock::setup_integration "${resources[@]}"
    
    echo "[COMMON_SETUP] Integration test environment ready"
}

#######################################
# Setup performance test environment (minimal overhead)
# For tests that need maximum speed
#######################################
setup_performance_test() {
    local resource="${1:-generic}"
    
    echo "[COMMON_SETUP] Setting up performance test environment"
    
    # Minimal environment setup
    export FORCE="yes"
    export YES="yes"
    export QUIET="yes"
    export TEST_PERFORMANCE_MODE="true"
    
    # Load only essential mocks
    mock::load system docker
    mock::load system commands
    
    # Configure basic resource environment if specified
    if [[ "$resource" != "generic" ]]; then
        mock::configure_resource_environment "$resource"
    fi
    
    echo "[COMMON_SETUP] Performance test environment ready"
}

#######################################
# Generic resource mock setup (fallback)
# Arguments: $1 - resource name, $2 - state
#######################################
mock::setup_generic_resource() {
    local resource="$1"
    local state="${2:-healthy}"
    
    # Set up basic resource environment
    export RESOURCE_NAME="$resource"
    export RESOURCE_STATE="$state"
    export RESOURCE_PORT="${RESOURCE_PORT:-8080}"
    export RESOURCE_BASE_URL="http://localhost:${RESOURCE_PORT}"
    export RESOURCE_CONTAINER_NAME="${TEST_NAMESPACE}_${resource}"
    
    # Configure basic mocks based on state
    case "$state" in
        "healthy")
            export RESOURCE_DOCKER_STATE="running"
            export RESOURCE_HTTP_STATE="healthy"
            ;;
        "stopped")
            export RESOURCE_DOCKER_STATE="stopped"
            export RESOURCE_HTTP_STATE="unavailable"
            ;;
        "installing")
            export RESOURCE_DOCKER_STATE="creating"
            export RESOURCE_HTTP_STATE="unavailable"
            ;;
        *)
            export RESOURCE_DOCKER_STATE="unknown"
            export RESOURCE_HTTP_STATE="unknown"
            ;;
    esac
}

#######################################
# Enhanced cleanup function with comprehensive resource cleanup
#######################################
cleanup_mocks() {
    echo "[COMMON_SETUP] Cleaning up test environment"
    
    # Cleanup benchmarking system
    benchmark::cleanup
    
    # Use mock registry cleanup
    mock::cleanup
    
    # Additional cleanup for common setup
    if [[ -d "${BATS_TEST_TMPDIR:-}" ]]; then
        rm -rf "$BATS_TEST_TMPDIR"
    fi
    
    # Clean up any stray processes
    jobs -p | xargs -r kill 2>/dev/null || true
    
    # Reset common environment variables
    unset TEST_NAMESPACE TEST_PORT_BASE RESOURCE_NAME RESOURCE_STATE
    unset RESOURCE_PORT RESOURCE_BASE_URL RESOURCE_CONTAINER_NAME
    
    echo "[COMMON_SETUP] Cleanup complete"
}

#######################################
# Validation helpers for test environment
#######################################

# Validate that required mock functions are available
validate_mock_environment() {
    local required_functions=(
        "mock::load"
        "mock::setup_minimal" 
        "assert_output_contains"
        "assert_file_exists"
    )
    
    for func in "${required_functions[@]}"; do
        if ! validate_function_exists "$func" "COMMON_SETUP"; then
            return 1
        fi
    done
    
    echo "[COMMON_SETUP] Mock environment validation passed"
    return 0
}

# Check if running in performance mode
is_performance_mode() {
    [[ "${TEST_PERFORMANCE_MODE:-}" == "true" ]]
}

# Check if test isolation is properly configured
validate_test_isolation() {
    if [[ -z "${TEST_NAMESPACE:-}" ]]; then
        echo "[COMMON_SETUP] WARNING: TEST_NAMESPACE not set, tests may interfere with each other" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Backward compatibility functions
# These ensure existing tests continue to work
#######################################

# Legacy function aliases for backward compatibility
setup_standard_mock_framework() {
    echo "[COMMON_SETUP] Legacy function called: setup_standard_mock_framework"
    setup_standard_mocks
}

cleanup_standard_mock_framework() {
    echo "[COMMON_SETUP] Legacy function called: cleanup_standard_mock_framework"
    cleanup_mocks
}

# Legacy environment variable setup
setup_legacy_environment() {
    # Variables from the old system mock framework
    export MOCK_CALL_COUNT=0
    export READ_CALL_COUNT=0
    export MOCK_FRAMEWORK_LOADED="true"
}

#######################################
# Auto-detection of test type based on context
#######################################
auto_setup_test_environment() {
    local test_file="${BATS_TEST_FILENAME:-}"
    local test_name="${BATS_TEST_NAME:-}"
    
    # Auto-detect test type from file path or test name
    if [[ "$test_file" =~ /resources/.*/(ai|automation|agents|storage|search|execution)/ ]]; then
        local resource_name
        resource_name=$(basename "$(dirname "$(dirname "$test_file")")")
        
        echo "[COMMON_SETUP] Auto-detected resource test: $resource_name"
        setup_resource_test "$resource_name"
    elif [[ "$test_name" =~ integration|workflow|pipeline ]]; then
        echo "[COMMON_SETUP] Auto-detected integration test"
        # Call without arguments - will require manual resource specification
        setup_standard_mocks
    elif [[ "$test_name" =~ performance|speed|benchmark ]]; then
        echo "[COMMON_SETUP] Auto-detected performance test"
        setup_performance_test "generic"
    else
        echo "[COMMON_SETUP] Using standard test setup"
        setup_standard_mocks
    fi
}

#######################################
# Export all functions for use in tests
#######################################
export -f setup_standard_mocks setup_resource_test setup_integration_test
export -f setup_performance_test cleanup_mocks validate_mock_environment
export -f is_performance_mode validate_test_isolation auto_setup_test_environment
export -f setup_standard_mock_framework cleanup_standard_mock_framework
export -f setup_legacy_environment mock::setup_generic_resource

# Validate environment on load
if ! validate_mock_environment; then
    common_setup_error "Mock environment validation failed"
    return 1
fi

echo "[COMMON_SETUP] Unified common setup loaded successfully"