#!/usr/bin/env bash
# Simple configuration system for BATS tests
# Minimal implementation to support test infrastructure

# Configuration storage
declare -gA VROOLI_TEST_CONFIG=()

# Initialize configuration
vrooli_config_init() {
    # Set default values
    VROOLI_TEST_CONFIG[mocks_enabled]="true"
    VROOLI_TEST_CONFIG[test_namespace]="test_$$"
    VROOLI_TEST_CONFIG[debug]="${DEBUG:-false}"
    return 0
}

# Get boolean configuration value
vrooli_config_get_bool() {
    local key="$1"
    local default="${2:-false}"
    local value="${VROOLI_TEST_CONFIG[$key]:-$default}"
    echo "$value"
}

# Get string configuration value
vrooli_config_get() {
    local key="$1"
    local default="${2:-}"
    echo "${VROOLI_TEST_CONFIG[$key]:-$default}"
}

# Get port for a service
vrooli_config_get_port() {
    local service="$1"
    # Return a default test port based on service name
    case "$service" in
        redis) echo "6379" ;;
        postgres) echo "5432" ;;
        n8n) echo "5678" ;;
        docker) echo "2375" ;;
        *) echo "8080" ;;
    esac
}

# Set configuration value
vrooli_config_set() {
    local key="$1"
    local value="$2"
    VROOLI_TEST_CONFIG[$key]="$value"
}

# Export configuration to environment variables
vrooli_config_export_env() {
    # Set basic environment variables for tests
    export TEST_NAMESPACE="${VROOLI_TEST_CONFIG[test_namespace]:-test_$$}"
    export VROOLI_TEST_MODE="${VROOLI_TEST_CONFIG[test_mode]:-unit}"
    export VROOLI_TEST_DEBUG="${VROOLI_TEST_CONFIG[debug]:-false}"
    
    # Set derived values
    VROOLI_TEST_CONFIG[derived_tmpdir]="${BATS_TEST_TMPDIR:-/tmp/vrooli_test_$$}"
    
    return 0
}

export -f vrooli_config_init
export -f vrooli_config_get_bool
export -f vrooli_config_get
export -f vrooli_config_get_port
export -f vrooli_config_set
export -f vrooli_config_export_env