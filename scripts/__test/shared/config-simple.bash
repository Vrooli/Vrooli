#!/usr/bin/env bash
# Vrooli Test Configuration Loader - Simple Variable-Based Version
# This version uses regular bash variables instead of associative arrays for BATS compatibility

# Prevent duplicate loading
if [[ "${VROOLI_CONFIG_SIMPLE_LOADED:-}" == "true" ]]; then
    return 0
fi
export VROOLI_CONFIG_SIMPLE_LOADED="true"

export VROOLI_TEST_CONFIG_DIR
export VROOLI_TEST_ENVIRONMENT

#######################################
# Initialize configuration loader
#######################################
vrooli_config_init() {
    # Determine config directory
    if [[ -z "${VROOLI_TEST_CONFIG_DIR:-}" ]]; then
        # Find the __test directory by looking for config/test-config.yaml
        local current_dir="$(pwd)"
        local search_dir="$current_dir"
        
        while [[ "$search_dir" != "/" ]]; do
            if [[ -f "$search_dir/config/test-config.yaml" ]]; then
                VROOLI_TEST_CONFIG_DIR="$search_dir/config"
                break
            fi
            if [[ -f "$search_dir/../config/test-config.yaml" ]]; then
                VROOLI_TEST_CONFIG_DIR="$search_dir/../config"
                break
            fi
            search_dir="$(dirname "$search_dir")"
        done
        
        # If not found, fall back to the old method
        if [[ -z "${VROOLI_TEST_CONFIG_DIR:-}" ]]; then
            local script_dir
            script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
            VROOLI_TEST_CONFIG_DIR="$(dirname "$script_dir")/config"
        fi
    fi
    
    # Determine environment
    if [[ -z "${VROOLI_TEST_ENVIRONMENT:-}" ]]; then
        if [[ "${CI:-}" == "true" ]]; then
            VROOLI_TEST_ENVIRONMENT="ci"
        elif [[ -n "${DOCKER_CONTAINER:-}" ]] || [[ -f "/.dockerenv" ]]; then
            VROOLI_TEST_ENVIRONMENT="docker"
        else
            VROOLI_TEST_ENVIRONMENT="local"
        fi
    fi
    
    echo "[CONFIG] Initializing configuration (env: $VROOLI_TEST_ENVIRONMENT, config_dir: $VROOLI_TEST_CONFIG_DIR)"
    
    # Validate config directory exists
    if [[ ! -d "$VROOLI_TEST_CONFIG_DIR" ]]; then
        echo "[CONFIG] ERROR: Configuration directory not found: $VROOLI_TEST_CONFIG_DIR" >&2
        return 1
    fi
    
    # Load configurations
    _vrooli_load_config_files
    
    echo "[CONFIG] Configuration loaded successfully"
    return 0
}

#######################################
# Load configuration files with environment overrides
#######################################
_vrooli_load_config_files() {
    local main_config="$VROOLI_TEST_CONFIG_DIR/test-config.yaml"
    local env_config="$VROOLI_TEST_CONFIG_DIR/environments/${VROOLI_TEST_ENVIRONMENT}.yaml"
    
    # Load main configuration
    if [[ -f "$main_config" ]]; then
        echo "[CONFIG] Loading main configuration: $main_config"
        _vrooli_parse_yaml_file "$main_config"
    else
        echo "[CONFIG] ERROR: Main configuration file not found: $main_config" >&2
        return 1
    fi
    
    # Load environment-specific overrides
    if [[ -f "$env_config" ]]; then
        echo "[CONFIG] Loading environment configuration: $env_config"
        _vrooli_parse_yaml_file "$env_config" "override"
    else
        echo "[CONFIG] WARNING: Environment configuration not found: $env_config"
    fi
    
    # Set derived configuration values
    _vrooli_set_derived_config
    
    return 0
}

#######################################
# Parse YAML configuration file and set bash variables
#######################################
_vrooli_parse_yaml_file() {
    local file="$1"
    local mode="${2:-base}"
    local current_section=""
    local current_subsection=""
    
    if [[ ! -f "$file" ]]; then
        echo "[CONFIG] ERROR: Configuration file not found: $file" >&2
        return 1
    fi
    
    while IFS= read -r line; do
        # Skip comments and empty lines
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ "$line" =~ ^[[:space:]]*$ ]] && continue
        
        # Parse sections (top-level keys)
        if [[ "$line" =~ ^([a-zA-Z_][a-zA-Z0-9_]*):([[:space:]]*#.*)?$ ]]; then
            current_section="${BASH_REMATCH[1]}"
            current_subsection=""
            continue
        fi
        
        # Parse subsections (indented keys)
        if [[ "$line" =~ ^[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*):([[:space:]]*#.*)?$ ]]; then
            current_subsection="${BASH_REMATCH[1]}"
            continue
        fi
        
        # Parse key-value pairs
        if [[ "$line" =~ ^[[:space:]]*([a-zA-Z_][a-zA-Z0-9_]*):[[:space:]]*(.+)$ ]]; then
            local key="${BASH_REMATCH[1]}"
            local value="${BASH_REMATCH[2]}"
            
            # Clean up value (remove quotes and comments)
            value=$(echo "$value" | sed 's/[[:space:]]*#.*$//' | sed 's/^"//;s/"$//' | sed "s/^'//;s/'$//")
            
            # Build configuration variable name
            local config_var
            if [[ -n "$current_subsection" ]]; then
                config_var="VROOLI_CONFIG_${current_section^^}_${current_subsection^^}_${key^^}"
            elif [[ -n "$current_section" ]]; then
                config_var="VROOLI_CONFIG_${current_section^^}_${key^^}"
            else
                config_var="VROOLI_CONFIG_${key^^}"
            fi
            
            # Set the variable and export it
            declare -g "$config_var=$value"
            export "$config_var"
            
            # For debugging: show what we're setting
            if [[ "${VROOLI_CONFIG_DEBUG:-}" == "true" ]]; then
                echo "[CONFIG] Set: $config_var = $value"
            fi
        fi
    done < "$file"
    
    return 0
}

#######################################
# Set derived configuration values
#######################################
_vrooli_set_derived_config() {
    # Set test namespace with timestamp
    local timestamp
    timestamp=$(date +%s)
    local namespace_prefix="${VROOLI_CONFIG_ENVIRONMENT_NAMESPACE_PREFIX:-vrooli_test}"
    
    declare -g "VROOLI_CONFIG_DERIVED_TEST_NAMESPACE=${namespace_prefix}_${timestamp}_$$"
    export VROOLI_CONFIG_DERIVED_TEST_NAMESPACE
    
    # Set temporary directory
    local tmpdir_base="${VROOLI_CONFIG_ENVIRONMENT_TMPDIR_BASE:-/tmp}"
    local tmpdir_fallback="${VROOLI_CONFIG_ENVIRONMENT_TMPDIR_FALLBACK:-/tmp}"
    
    # Try preferred tmpdir, fallback if needed
    if [[ -d "$tmpdir_base" && -w "$tmpdir_base" ]]; then
        declare -g "VROOLI_CONFIG_DERIVED_TMPDIR=$tmpdir_base/$VROOLI_CONFIG_DERIVED_TEST_NAMESPACE"
    else
        declare -g "VROOLI_CONFIG_DERIVED_TMPDIR=$tmpdir_fallback/$VROOLI_CONFIG_DERIVED_TEST_NAMESPACE"
    fi
    export VROOLI_CONFIG_DERIVED_TMPDIR
    
    # Set log file with namespace
    local log_base="${VROOLI_CONFIG_LOGGING_FILE:-/tmp/vrooli-test.log}"
    declare -g "VROOLI_CONFIG_DERIVED_LOG_FILE=${log_base%.log}-$VROOLI_CONFIG_DERIVED_TEST_NAMESPACE.log"
    export VROOLI_CONFIG_DERIVED_LOG_FILE
    
    return 0
}

#######################################
# Get configuration value by key
#######################################
vrooli_config_get() {
    local key="$1"
    local default="${2:-}"
    
    # Convert key to variable name format (replace dots with underscores)
    local var_name="VROOLI_CONFIG_${key^^}"
    var_name="${var_name//./_}"
    
    # Get the value using indirect variable expansion
    local value="${!var_name:-$default}"
    echo "$value"
}

#######################################
# Get configuration value as integer
#######################################
vrooli_config_get_int() {
    local key="$1"
    local default="${2:-0}"
    local value
    
    value=$(vrooli_config_get "$key" "$default")
    
    # Validate it's a number
    if [[ "$value" =~ ^[0-9]+$ ]]; then
        echo "$value"
    else
        echo "$default"
    fi
}

#######################################
# Get configuration value as boolean
#######################################
vrooli_config_get_bool() {
    local key="$1"
    local default="${2:-false}"
    local value
    
    value=$(vrooli_config_get "$key" "$default")
    
    # Normalize boolean values
    case "${value,,}" in
        true|yes|1|on|enabled)
            echo "true"
            ;;
        *)
            echo "false"
            ;;
    esac
}

#######################################
# Get port for a service
#######################################
vrooli_config_get_port() {
    local service="$1"
    vrooli_config_get "ports_$service"
}

#######################################
# Get timeout for a category
#######################################
vrooli_config_get_timeout() {
    local category="$1"
    vrooli_config_get_int "timeouts_$category" 60
}

#######################################
# Check if parallel execution is enabled
#######################################
vrooli_config_parallel_enabled() {
    local enabled
    enabled=$(vrooli_config_get_bool "performance_parallel_enabled" "false")
    [[ "$enabled" == "true" ]]
}

#######################################
# Get maximum parallel tests
#######################################
vrooli_config_max_parallel() {
    vrooli_config_get_int "performance_max_parallel_tests" 1
}

#######################################
# Export configuration as environment variables
#######################################
vrooli_config_export_env() {
    # Export commonly used values
    export VROOLI_TEST_NAMESPACE="$VROOLI_CONFIG_DERIVED_TEST_NAMESPACE"
    export VROOLI_TEST_TMPDIR="$VROOLI_CONFIG_DERIVED_TMPDIR"
    export VROOLI_TEST_LOG_FILE="$VROOLI_CONFIG_DERIVED_LOG_FILE"
    export VROOLI_TEST_LOG_LEVEL="$VROOLI_CONFIG_LOGGING_LEVEL"
    
    # Export service ports as environment variables
    for var in $(compgen -v VROOLI_CONFIG_PORTS_); do
        local service_name="${var#VROOLI_CONFIG_PORTS_}"
        local port="${!var}"
        local env_var="VROOLI_TEST_${service_name}_PORT"
        export "$env_var=$port"
    done
    
    echo "[CONFIG] Environment variables exported"
    return 0
}

#######################################
# Show current configuration (for debugging)
#######################################
vrooli_config_show() {
    echo "[CONFIG] Current Configuration:"
    echo "  Environment: $VROOLI_TEST_ENVIRONMENT"
    echo "  Config Dir: $VROOLI_TEST_CONFIG_DIR"
    echo "  Test Namespace: $VROOLI_CONFIG_DERIVED_TEST_NAMESPACE"
    echo "  Temp Dir: $VROOLI_CONFIG_DERIVED_TMPDIR"
    echo "  Log File: $VROOLI_CONFIG_DERIVED_LOG_FILE"
    echo "  Parallel Enabled: $(vrooli_config_get_bool 'performance_parallel_enabled')"
    
    if [[ "${VROOLI_CONFIG_DEBUG:-}" == "true" ]]; then
        echo ""
        echo "[CONFIG] All Configuration Variables:"
        for var in $(compgen -v VROOLI_CONFIG_ | sort); do
            echo "  $var = ${!var}"
        done
    fi
}

# Export functions for use in tests
export -f vrooli_config_init vrooli_config_get vrooli_config_get_int vrooli_config_get_bool
export -f vrooli_config_get_port vrooli_config_get_timeout vrooli_config_parallel_enabled
export -f vrooli_config_max_parallel vrooli_config_export_env vrooli_config_show
export -f _vrooli_load_config_files _vrooli_parse_yaml_file _vrooli_set_derived_config

echo "[CONFIG] Simple configuration loader functions loaded" >&2