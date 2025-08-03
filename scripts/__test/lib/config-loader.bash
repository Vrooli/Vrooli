#!/usr/bin/env bash
# Centralized Configuration Loader for Vrooli Test Infrastructure
# Replaces hard-coded values throughout the test system

# Prevent duplicate loading
if [[ "${CONFIG_LOADER_LOADED:-}" == "true" ]]; then
    return 0
fi
export CONFIG_LOADER_LOADED="true"

# Configuration file paths
readonly CONFIG_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../config" && pwd)"
readonly MAIN_CONFIG_FILE="${CONFIG_DIR}/test-config.yaml"
readonly TIMEOUTS_CONFIG_FILE="${CONFIG_DIR}/timeouts.yaml"

# Global configuration cache
declare -A CONFIG_CACHE
declare -A RESOURCE_CONFIG_CACHE
declare -A TIMEOUT_CACHE

# Environment detection
TEST_ENVIRONMENT="${TEST_ENVIRONMENT:-development}"

#######################################
# Load and parse YAML configuration
# Globals: CONFIG_CACHE
# Arguments: 
#   $1 - YAML file path
#   $2 - cache prefix (optional)
# Returns: 0 on success, 1 on failure
#######################################
config::load_yaml() {
    local yaml_file="$1"
    local cache_prefix="${2:-}"
    
    if [[ ! -f "$yaml_file" ]]; then
        echo "[CONFIG] ERROR: Configuration file not found: $yaml_file" >&2
        return 1
    fi
    
    # Check if yq is available
    if ! command -v yq >/dev/null 2>&1; then
        echo "[CONFIG] ERROR: yq command not found. Please install yq for YAML parsing." >&2
        return 1
    fi
    
    # Parse and cache configuration
    local temp_file="/tmp/config_parsed_$$"
    if ! yq eval -o=shell "$yaml_file" > "$temp_file" 2>/dev/null; then
        echo "[CONFIG] ERROR: Failed to parse YAML file: $yaml_file" >&2
        rm -f "$temp_file"
        return 1
    fi
    
    # Source the parsed configuration
    source "$temp_file"
    rm -f "$temp_file"
    
    echo "[CONFIG] Loaded configuration from: $yaml_file"
    return 0
}

#######################################
# Get configuration value with fallback
# Arguments:
#   $1 - config key (dot notation)
#   $2 - fallback value (optional)
# Output: configuration value or fallback
#######################################
config::get() {
    local key="$1"
    local fallback="${2:-}"
    
    # Convert dot notation to shell variable format
    local shell_key
    shell_key=$(echo "$key" | sed 's/\./_/g' | tr '[:lower:]' '[:upper:]')
    
    # Check cache first
    if [[ -n "${CONFIG_CACHE[$shell_key]:-}" ]]; then
        echo "${CONFIG_CACHE[$shell_key]}"
        return 0
    fi
    
    # Try to get from yq if config file exists
    if [[ -f "$MAIN_CONFIG_FILE" ]] && command -v yq >/dev/null 2>&1; then
        local value
        if value=$(yq eval ".$key" "$MAIN_CONFIG_FILE" 2>/dev/null) && [[ "$value" != "null" ]]; then
            CONFIG_CACHE["$shell_key"]="$value"
            echo "$value"
            return 0
        fi
    fi
    
    # Return fallback
    echo "$fallback"
}

#######################################
# Get resource configuration
# Arguments:
#   $1 - resource name (e.g., "ollama", "n8n")
#   $2 - property name (e.g., "default_port", "health_endpoint")
#   $3 - fallback value (optional)
#######################################
config::get_resource() {
    local resource="$1"
    local property="$2"
    local fallback="${3:-}"
    
    config::get "resources.${resource}.${property}" "$fallback"
}

#######################################
# Get dynamic port for resource (enables parallel execution)
# Arguments: $1 - resource name
# Output: available port number
#######################################
config::get_dynamic_port() {
    local resource="$1"
    local base_port
    base_port=$(config::get_resource "$resource" "default_port" "8080")
    
    # For parallel execution, calculate offset based on test namespace
    local port_offset=0
    if [[ -n "${TEST_NAMESPACE:-}" ]]; then
        # Extract numeric part from test namespace for offset
        local namespace_num
        namespace_num=$(echo "$TEST_NAMESPACE" | grep -o '[0-9]*' | tail -1)
        port_offset=$((namespace_num % 100))
    fi
    
    local dynamic_port=$((base_port + port_offset))
    
    # Ensure port is not in reserved ranges
    while config::port_is_reserved "$dynamic_port"; do
        dynamic_port=$((dynamic_port + 1))
    done
    
    echo "$dynamic_port"
}

#######################################
# Check if port is in reserved range
# Arguments: $1 - port number
# Returns: 0 if reserved, 1 if available
#######################################
config::port_is_reserved() {
    local port="$1"
    
    # Get reserved ranges from config
    local reserved_ranges
    reserved_ranges=$(config::get "ports.reserved_ranges" "")
    
    # For simplicity, check common reserved ranges
    # TODO: Parse YAML array properly
    if [[ "$port" -ge 8080 && "$port" -le 8090 ]]; then
        return 0  # Reserved
    fi
    
    if [[ "$port" -ge 3000 && "$port" -le 3010 ]]; then
        return 0  # Reserved
    fi
    
    return 1  # Available
}

#######################################
# Get timeout for test pattern
# Arguments: $1 - test file path or pattern
# Output: timeout in seconds
#######################################
config::get_timeout() {
    local test_pattern="$1"
    local base_timeout
    base_timeout=$(config::get "timeouts.base.integration" "120")
    
    # Check for specific pattern matches
    local patterns
    patterns=$(config::get "timeouts.patterns" "")
    
    # Simple pattern matching (could be enhanced with proper YAML array parsing)
    case "$test_pattern" in
        *api*.bats)
            echo "120"
            ;;
        *models*.bats)
            echo "240"
            ;;
        *install*.bats)
            echo "180"
            ;;
        *docker*.bats)
            echo "150"
            ;;
        *process*.bats)
            echo "90"
            ;;
        *status*.bats)
            echo "60"
            ;;
        *)
            echo "$base_timeout"
            ;;
    esac
}

#######################################
# Setup resource environment variables
# Arguments: $1 - resource name
#######################################
config::setup_resource_env() {
    local resource="$1"
    
    if [[ -z "$resource" ]]; then
        echo "[CONFIG] ERROR: Resource name required" >&2
        return 1
    fi
    
    # Get resource configuration
    local category port base_url health_endpoint container_name
    category=$(config::get_resource "$resource" "category" "unknown")
    port=$(config::get_dynamic_port "$resource")
    health_endpoint=$(config::get_resource "$resource" "health_endpoint" "/health")
    
    # Calculate base URL
    base_url="http://localhost:${port}"
    
    # Generate container name with namespace
    container_name="${TEST_NAMESPACE:-test}_${resource}"
    
    # Set standard environment variables
    export RESOURCE_NAME="$resource"
    export RESOURCE_CATEGORY="$category"
    export RESOURCE_PORT="$port"
    export RESOURCE_BASE_URL="$base_url"
    export RESOURCE_HEALTH_ENDPOINT="$health_endpoint"
    export RESOURCE_CONTAINER_NAME="$container_name"
    
    # Set resource-specific variables (uppercase resource name)
    local resource_upper
    resource_upper=$(echo "$resource" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
    
    declare -g "${resource_upper}_PORT=$port"
    declare -g "${resource_upper}_BASE_URL=$base_url"
    declare -g "${resource_upper}_HEALTH_ENDPOINT=$health_endpoint"
    declare -g "${resource_upper}_CONTAINER_NAME=$container_name"
    
    # Export them
    export "${resource_upper}_PORT"
    export "${resource_upper}_BASE_URL"
    export "${resource_upper}_HEALTH_ENDPOINT"
    export "${resource_upper}_CONTAINER_NAME"
    
    echo "[CONFIG] Configured environment for resource: $resource (port: $port)"
}

#######################################
# Setup global test environment variables
#######################################
config::setup_global_env() {
    local env="${TEST_ENVIRONMENT:-development}"
    
    # Global settings
    export TEST_NAMESPACE_PREFIX=$(config::get "global.test_namespace_prefix" "vrooli_test")
    export TEST_TIMEOUT_DEFAULT=$(config::get "global.default_timeout" "60")
    export TEST_MAX_PARALLEL=$(config::get "global.max_parallel_tests" "4")
    export TEST_CLEANUP_ON_EXIT=$(config::get "global.cleanup_on_exit" "true")
    
    # Environment-specific settings
    export TEST_LOG_LEVEL=$(config::get "environments.${env}.log_level" "info")
    export TEST_PRESERVE_TEMP_FILES=$(config::get "environments.${env}.preserve_temp_files" "false")
    export TEST_RESOURCE_CLEANUP_AGGRESSIVE=$(config::get "environments.${env}.resource_cleanup_aggressive" "false")
    
    # Temporary directory setup
    local temp_base
    temp_base=$(config::get "global.temp_dir_base" "/dev/shm")
    local temp_fallback
    temp_fallback=$(config::get "global.temp_dir_fallback" "/tmp")
    
    if [[ -d "$temp_base" && -w "$temp_base" ]]; then
        export TEST_TEMP_DIR_BASE="$temp_base"
    else
        export TEST_TEMP_DIR_BASE="$temp_fallback"
    fi
    
    # Generate unique test namespace if not set
    if [[ -z "${TEST_NAMESPACE:-}" ]]; then
        export TEST_NAMESPACE="${TEST_NAMESPACE_PREFIX}_$$_${RANDOM}"
    fi
    
    echo "[CONFIG] Global environment configured for: $env"
}

#######################################
# Setup mock configuration
#######################################
config::setup_mock_env() {
    # Docker mock settings
    export DOCKER_MOCK_MODE=$(config::get "mocks.docker.mock_mode" "normal")
    export DOCKER_MOCK_SIMULATE_DELAYS=$(config::get "mocks.docker.simulate_delays" "false")
    export DOCKER_MOCK_TRACK_CALLS=$(config::get "mocks.docker.track_calls" "true")
    
    # HTTP mock settings  
    export HTTP_MOCK_TIMEOUT=$(config::get "mocks.http.default_timeout" "5")
    export HTTP_MOCK_SIMULATE_ERRORS=$(config::get "mocks.http.simulate_network_errors" "false")
    
    # Filesystem mock settings
    export FS_MOCK_USE_TMPFS=$(config::get "mocks.filesystem.use_tmpfs" "true")
    export FS_MOCK_CLEANUP_ON_FAILURE=$(config::get "mocks.filesystem.cleanup_on_failure" "true")
    
    echo "[CONFIG] Mock environment configured"
}

#######################################
# Get resource dependencies
# Arguments: $1 - resource name
# Output: space-separated list of required dependencies
#######################################
config::get_resource_dependencies() {
    local resource="$1"
    
    # Get required dependencies
    local required
    required=$(config::get "dependencies.${resource}.requires" "")
    
    # For now, return a simple list (could be enhanced with proper YAML array parsing)
    case "$resource" in
        "n8n"|"windmill")
            echo "postgres"
            ;;
        "agent-s2")
            echo "postgres redis"
            ;;
        *)
            echo ""
            ;;
    esac
}

#######################################
# Initialize configuration system
#######################################
config::init() {
    echo "[CONFIG] Initializing configuration system..."
    
    # Check if configuration file exists
    if [[ ! -f "$MAIN_CONFIG_FILE" ]]; then
        echo "[CONFIG] ERROR: Main configuration file not found: $MAIN_CONFIG_FILE" >&2
        echo "[CONFIG] Please ensure test-config.yaml exists in the config directory." >&2
        return 1
    fi
    
    # Load main configuration
    if ! config::load_yaml "$MAIN_CONFIG_FILE"; then
        echo "[CONFIG] ERROR: Failed to load main configuration" >&2
        return 1
    fi
    
    # Setup global environment
    config::setup_global_env
    
    # Setup mock environment
    config::setup_mock_env
    
    echo "[CONFIG] Configuration system initialized successfully"
    return 0
}

#######################################
# Print configuration summary (for debugging)
#######################################
config::debug() {
    echo "[CONFIG] Configuration Summary:"
    echo "  Environment: ${TEST_ENVIRONMENT}"
    echo "  Namespace: ${TEST_NAMESPACE:-not-set}"
    echo "  Temp Base: ${TEST_TEMP_DIR_BASE:-not-set}"
    echo "  Default Timeout: ${TEST_TIMEOUT_DEFAULT:-not-set}"
    echo "  Max Parallel: ${TEST_MAX_PARALLEL:-not-set}"
    echo "  Log Level: ${TEST_LOG_LEVEL:-not-set}"
    echo "  Config File: $MAIN_CONFIG_FILE"
}

# Export all functions
export -f config::load_yaml config::get config::get_resource config::get_dynamic_port
export -f config::port_is_reserved config::get_timeout config::setup_resource_env
export -f config::setup_global_env config::setup_mock_env config::get_resource_dependencies
export -f config::init config::debug

# Auto-initialize if config file exists
if [[ -f "$MAIN_CONFIG_FILE" ]]; then
    config::init
else
    echo "[CONFIG] WARNING: Configuration file not found, using fallback values"
fi

echo "[CONFIG] Configuration loader loaded"