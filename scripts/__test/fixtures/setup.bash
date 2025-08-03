#!/usr/bin/env bash
# Vrooli Test Setup - Single Entry Point for All BATS Tests
# This is the ONLY file you need to source in BATS tests

# Prevent duplicate loading
if [[ "${VROOLI_TEST_SETUP_LOADED:-}" == "true" ]]; then
    return 0
fi
export VROOLI_TEST_SETUP_LOADED="true"

# Determine the test root directory
VROOLI_TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export VROOLI_TEST_ROOT

echo "[SETUP] Vrooli Test Setup - Loading from: $VROOLI_TEST_ROOT"

#######################################
# Load required dependencies
#######################################

# Load configuration system first (using simple variable-based system for BATS compatibility)
source "$VROOLI_TEST_ROOT/shared/config-simple.bash"

# Initialize configuration
if ! vrooli_config_init; then
    echo "[SETUP] ERROR: Failed to initialize configuration" >&2
    return 1
fi

# Load other shared utilities
source "$VROOLI_TEST_ROOT/shared/utils.bash" 2>/dev/null || echo "[SETUP] Utils not loaded (optional)"
source "$VROOLI_TEST_ROOT/shared/logging.bash" 2>/dev/null || echo "[SETUP] Logging not loaded (optional)"

# Load assertions
source "$VROOLI_TEST_ROOT/fixtures/assertions.bash"

# Load cleanup functions
source "$VROOLI_TEST_ROOT/fixtures/cleanup.bash"

# Load enhanced cleanup manager for automatic cleanup
if [[ -f "$VROOLI_TEST_ROOT/fixtures/cleanup-manager.sh" ]]; then
    source "$VROOLI_TEST_ROOT/fixtures/cleanup-manager.sh"
    # Register automatic cleanup handlers
    vrooli_register_cleanup
    echo "[SETUP] Cleanup manager loaded with automatic cleanup registration"
fi

#######################################
# Main setup functions for tests
#######################################

#######################################
# Setup a basic unit test with minimal mocks
# This is the most common setup function
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
vrooli_setup_unit_test() {
    echo "[SETUP] Setting up unit test environment"
    
    # Configure test environment
    _vrooli_setup_test_environment
    
    # Load basic system mocks
    _vrooli_load_system_mocks
    
    echo "[SETUP] Unit test environment ready"
    return 0
}

#######################################
# Setup a service-specific test
# Arguments: $1 - service name (e.g., "ollama", "postgres")
# Returns: 0 on success, 1 on failure
#######################################
vrooli_setup_service_test() {
    local service="$1"
    
    if [[ -z "$service" ]]; then
        echo "[SETUP] ERROR: Service name required for service test" >&2
        return 1
    fi
    
    echo "[SETUP] Setting up service test for: $service"
    
    # Basic unit test setup
    vrooli_setup_unit_test
    
    # Load service-specific mock
    _vrooli_load_service_mock "$service"
    
    # Configure service environment
    _vrooli_configure_service_environment "$service"
    
    echo "[SETUP] Service test environment ready for: $service"
    return 0
}

#######################################
# Setup an integration test with multiple services
# Arguments: $@ - list of service names
# Returns: 0 on success, 1 on failure
#######################################
vrooli_setup_integration_test() {
    local services=("$@")
    
    if [[ ${#services[@]} -eq 0 ]]; then
        echo "[SETUP] ERROR: At least one service required for integration test" >&2
        return 1
    fi
    
    echo "[SETUP] Setting up integration test for services: ${services[*]}"
    
    # Basic unit test setup
    vrooli_setup_unit_test
    
    # Load mocks for all services
    for service in "${services[@]}"; do
        _vrooli_load_service_mock "$service"
        _vrooli_configure_service_environment "$service"
    done
    
    echo "[SETUP] Integration test environment ready"
    return 0
}

#######################################
# Setup a performance test (minimal overhead)
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
vrooli_setup_performance_test() {
    echo "[SETUP] Setting up performance test environment"
    
    # Minimal environment setup
    _vrooli_setup_test_environment "performance"
    
    # Load only essential mocks
    _vrooli_load_minimal_mocks
    
    echo "[SETUP] Performance test environment ready"
    return 0
}

# Note: vrooli_cleanup_test is provided by cleanup.bash which is sourced above

#######################################
# Internal helper functions
#######################################

#######################################
# Setup basic test environment
# Arguments: $1 - test type (optional)
# Returns: 0 on success, 1 on failure
#######################################
_vrooli_setup_test_environment() {
    local test_type="${1:-unit}"
    
    # Export configuration to environment
    vrooli_config_export_env
    
    # Create test temporary directory
    local tmpdir
    tmpdir=$(vrooli_config_get "derived_tmpdir")
    
    if ! mkdir -p "$tmpdir"; then
        echo "[SETUP] ERROR: Failed to create temp directory: $tmpdir" >&2
        return 1
    fi
    
    export VROOLI_TEST_TMPDIR="$tmpdir"
    export BATS_TEST_TMPDIR="$tmpdir"  # For BATS compatibility
    
    # Set up test isolation
    export TEST_NAMESPACE
    TEST_NAMESPACE=$(vrooli_config_get "derived_test_namespace")
    
    # Configure test type specific settings
    case "$test_type" in
        "performance")
            export VROOLI_TEST_PERFORMANCE_MODE="true"
            export VROOLI_TEST_QUIET="true"
            ;;
        *)
            export VROOLI_TEST_PERFORMANCE_MODE="false"
            export VROOLI_TEST_QUIET="false"
            ;;
    esac
    
    echo "[SETUP] Test environment configured (type: $test_type, namespace: $TEST_NAMESPACE)"
    return 0
}

#######################################
# Load system mocks (docker, http, etc.)
# Arguments: None
# Returns: 0 on success
#######################################
_vrooli_load_system_mocks() {
    local mocks_enabled
    mocks_enabled=$(vrooli_config_get_bool "mocks_enabled" "true")
    
    if [[ "$mocks_enabled" != "true" ]]; then
        echo "[SETUP] Mocks disabled, skipping system mock loading"
        return 0
    fi
    
    # Load system mocks if they exist
    local system_mocks=("docker" "http" "system")
    
    for mock in "${system_mocks[@]}"; do
        local mock_file="$VROOLI_TEST_ROOT/fixtures/mocks/$mock.bash"
        if [[ -f "$mock_file" ]]; then
            source "$mock_file"
            echo "[SETUP] Loaded system mock: $mock"
        else
            echo "[SETUP] System mock not found: $mock (file: $mock_file)"
        fi
    done
    
    return 0
}

#######################################
# Load service-specific mock
# Arguments: $1 - service name
# Returns: 0 on success
#######################################
_vrooli_load_service_mock() {
    local service="$1"
    local mock_file="$VROOLI_TEST_ROOT/fixtures/mocks/$service.bash"
    
    if [[ -f "$mock_file" ]]; then
        source "$mock_file"
        echo "[SETUP] Loaded service mock: $service"
    else
        echo "[SETUP] WARNING: Service mock not found: $service (file: $mock_file)"
        # Create a generic mock environment for the service
        _vrooli_create_generic_service_mock "$service"
    fi
    
    return 0
}

#######################################
# Configure service environment variables
# Arguments: $1 - service name
# Returns: 0
#######################################
_vrooli_configure_service_environment() {
    local service="$1"
    local port
    
    # Get service port from configuration
    port=$(vrooli_config_get_port "$service")
    
    if [[ -n "$port" ]]; then
        # Set standard environment variables for the service
        local service_upper
        service_upper=$(echo "$service" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
        
        export "VROOLI_${service_upper}_PORT=$port"
        export "VROOLI_${service_upper}_BASE_URL=http://localhost:$port"
        export "VROOLI_${service_upper}_CONTAINER_NAME=${TEST_NAMESPACE}_$service"
        
        # Legacy compatibility variables
        export "${service_upper}_PORT=$port"
        export "${service_upper}_BASE_URL=http://localhost:$port"
        
        echo "[SETUP] Configured environment for $service (port: $port)"
    else
        echo "[SETUP] WARNING: No port configured for service: $service"
    fi
    
    return 0
}

#######################################
# Create generic mock for unknown service
# Arguments: $1 - service name
# Returns: 0
#######################################
_vrooli_create_generic_service_mock() {
    local service="$1"
    
    # Create basic mock function for the service
    eval "${service}() { echo 'Mock ${service} called with: \$*'; return 0; }"
    
    echo "[SETUP] Created generic mock for service: $service"
    return 0
}

#######################################
# Load minimal mocks for performance tests
# Arguments: None
# Returns: 0
#######################################
_vrooli_load_minimal_mocks() {
    # Only load essential system mocks
    local essential_mocks=("system")
    
    for mock in "${essential_mocks[@]}"; do
        local mock_file="$VROOLI_TEST_ROOT/fixtures/mocks/$mock.bash"
        if [[ -f "$mock_file" ]]; then
            source "$mock_file"
        fi
    done
    
    return 0
}

# Note: Cleanup helper functions (_vrooli_cleanup_mocks, _vrooli_cleanup_temp_files, 
# _vrooli_cleanup_processes) are provided by cleanup.bash which is sourced above

#######################################
# Auto-detect test type and setup accordingly
# This provides intelligent defaults based on test context
# Arguments: None
# Returns: 0 on success
#######################################
vrooli_auto_setup() {
    local test_file="${BATS_TEST_FILENAME:-}"
    local test_name="${BATS_TEST_NAME:-}"
    
    # Auto-detect based on test file path
    if [[ "$test_file" =~ /services/ ]]; then
        # Extract service name from path
        local service
        service=$(basename "$(dirname "$test_file")" .bats)
        echo "[SETUP] Auto-detected service test: $service"
        vrooli_setup_service_test "$service"
    elif [[ "$test_file" =~ /integration/ ]] || [[ "$test_name" =~ integration ]]; then
        echo "[SETUP] Auto-detected integration test"
        vrooli_setup_unit_test  # Start with basic setup, services can be added explicitly
    elif [[ "$test_name" =~ performance|benchmark ]]; then
        echo "[SETUP] Auto-detected performance test"
        vrooli_setup_performance_test
    else
        echo "[SETUP] Default unit test setup"
        vrooli_setup_unit_test
    fi
}

#######################################
# Export all functions for use in tests
#######################################
export -f vrooli_setup_unit_test vrooli_setup_service_test vrooli_setup_integration_test
export -f vrooli_setup_performance_test vrooli_cleanup_test vrooli_auto_setup

# Export internal helper functions defined in this file
export -f _vrooli_setup_test_environment _vrooli_load_system_mocks _vrooli_load_service_mock
export -f _vrooli_configure_service_environment _vrooli_create_generic_service_mock _vrooli_load_minimal_mocks
# Note: Cleanup functions are exported by cleanup.bash

echo "[SETUP] Vrooli Test Setup loaded successfully"