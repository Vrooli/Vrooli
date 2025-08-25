#!/usr/bin/env bash

#######################################
# Browserless Adapter Framework - Common Utilities
#
# Shared utilities for cross-resource adapters that allow
# browserless to provide UI automation interfaces for other resources.
#
# This framework enables the "for" pattern:
#   resource-browserless for <target-resource> <command> [args]
#
# Adapters transform browserless capabilities into interfaces
# for resources that may be unavailable, broken, or limited.
#######################################

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
ADAPTER_COMMON_DIR="${APP_ROOT}/resources/browserless/adapters"
BROWSERLESS_RESOURCE_DIR="${APP_ROOT}/resources/browserless"

# Source browserless core functionality
source "${BROWSERLESS_RESOURCE_DIR}/lib/common.sh"
source "${BROWSERLESS_RESOURCE_DIR}/lib/core.sh"
source "${BROWSERLESS_RESOURCE_DIR}/lib/api.sh"

#######################################
# Initialize adapter framework
# Sets up common adapter environment
# Globals:
#   BROWSERLESS_ADAPTER_MODE - Set to true when running in adapter mode
#   BROWSERLESS_TARGET_RESOURCE - The resource being adapted
# Arguments:
#   $1 - Target resource name (e.g., "n8n", "vault")
# Returns:
#   0 on success, 1 on failure
#######################################
adapter::init() {
    local target_resource="${1:-}"
    
    if [[ -z "$target_resource" ]]; then
        log::error "Target resource required for adapter initialization"
        return 1
    fi
    
    export BROWSERLESS_ADAPTER_MODE="true"
    export BROWSERLESS_TARGET_RESOURCE="$target_resource"
    
    log::debug "Adapter framework initialized for resource: $target_resource"
    return 0
}

#######################################
# Check if browserless is healthy
# Wrapper around core health check for adapter context
# Returns:
#   0 if healthy, 1 if not
#######################################
adapter::check_browserless_health() {
    # Check if browserless container is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${BROWSERLESS_CONTAINER_NAME}$"; then
        log::error "Browserless container is not running. Cannot run adapter for ${BROWSERLESS_TARGET_RESOURCE:-unknown}"
        log::info "Start browserless with: vrooli resource start browserless"
        return 1
    fi
    
    # Check if browserless API is responding
    if ! curl -s -f "http://localhost:${BROWSERLESS_PORT}/pressure" > /dev/null 2>&1; then
        log::error "Browserless API is not responding on port ${BROWSERLESS_PORT}"
        log::info "Check browserless logs: docker logs ${BROWSERLESS_CONTAINER_NAME}"
        return 1
    fi
    
    return 0
}

#######################################
# Get target resource configuration
# Loads configuration for the target resource being adapted
# Arguments:
#   $1 - Resource name
# Returns:
#   0 on success, 1 on failure
# Outputs:
#   Sets environment variables for target resource
#######################################
adapter::load_target_config() {
    local resource="${1:-}"
    
    if [[ -z "$resource" ]]; then
        log::error "Resource name required for configuration loading"
        return 1
    fi
    
    # Try to load resource configuration from common locations
    local config_locations=(
        "${var_ROOT_DIR}/scripts/resources/automation/${resource}/config/defaults.sh"
        "${var_ROOT_DIR}/scripts/resources/agents/${resource}/config/defaults.sh"
        "${var_ROOT_DIR}/scripts/resources/ai/${resource}/config/defaults.sh"
        "${var_ROOT_DIR}/scripts/resources/data/${resource}/config/defaults.sh"
        "${var_ROOT_DIR}/scripts/resources/ui/${resource}/config/defaults.sh"
    )
    
    local config_loaded=false
    for config_path in "${config_locations[@]}"; do
        if [[ -f "$config_path" ]]; then
            source "$config_path"
            config_loaded=true
            log::debug "Loaded configuration from: $config_path"
            break
        fi
    done
    
    if [[ "$config_loaded" == "false" ]]; then
        log::warn "No configuration found for resource: $resource"
        log::debug "Searched locations: ${config_locations[*]}"
    fi
    
    return 0
}

#######################################
# Execute browser function with adapter context
# Wrapper around browserless::run_function with adapter metadata
# Arguments:
#   $1 - JavaScript function code
#   $2 - Timeout in milliseconds
#   $3 - Use persistent session
#   $4 - Session ID
#   $5 - Launch options JSON
# Returns:
#   Function execution result
#######################################
adapter::execute_browser_function() {
    local function_code="${1:?Function code required}"
    local timeout="${2:-60000}"
    local use_persistent="${3:-true}"
    local session_id="${4:-adapter_${BROWSERLESS_TARGET_RESOURCE}_$(date +%s)}"
    local launch_json="${5:-}"
    
    # Add adapter metadata to function
    local adapter_metadata="
    // Adapter Context
    const adapterContext = {
        mode: 'adapter',
        target: '${BROWSERLESS_TARGET_RESOURCE:-unknown}',
        session: '${session_id}',
        timestamp: new Date().toISOString()
    };
    console.log('Browserless Adapter Context:', adapterContext);
    "
    
    # Prepend metadata to function code
    function_code="${adapter_metadata}${function_code}"
    
    # Execute via browserless
    browserless::run_function "$function_code" "$timeout" "$use_persistent" "$session_id" "$launch_json"
}

#######################################
# Register adapter capability
# Records that browserless can adapt a specific resource capability
# Arguments:
#   $1 - Resource name
#   $2 - Capability name
#   $3 - Priority (1-10, 1 is highest)
#   $4 - Description
# Returns:
#   0 on success
#######################################
adapter::register_capability() {
    local resource="${1:?Resource name required}"
    local capability="${2:?Capability name required}"
    local priority="${3:-5}"
    local description="${4:-}"
    
    local registry_file="${var_ROOT_DIR}/.vrooli/adapter-registry.json"
    
    # Ensure registry exists
    if [[ ! -f "$registry_file" ]]; then
        echo '{"adapters": []}' > "$registry_file"
    fi
    
    # Add capability to registry
    local temp_file=$(mktemp)
    jq --arg resource "$resource" \
       --arg capability "$capability" \
       --arg priority "$priority" \
       --arg description "$description" \
       --arg provider "browserless" \
       '.adapters += [{
           provider: $provider,
           resource: $resource,
           capability: $capability,
           priority: ($priority | tonumber),
           description: $description,
           timestamp: now | strftime("%Y-%m-%dT%H:%M:%SZ")
       }]' "$registry_file" > "$temp_file"
    
    mv "$temp_file" "$registry_file"
    log::debug "Registered adapter capability: browserless → $resource::$capability"
    return 0
}

#######################################
# List available adapters
# Shows all adapters that browserless provides
# Returns:
#   0 on success
#######################################
adapter::list() {
    local adapters_dir="${BROWSERLESS_RESOURCE_DIR}/adapters"
    
    log::header "Available Browserless Adapters"
    echo
    
    # List all subdirectories in adapters/
    for adapter_dir in "$adapters_dir"/*; do
        if [[ -d "$adapter_dir" ]]; then
            local adapter_name=$(basename "$adapter_dir")
            local adapter_api="${adapter_dir}/api.sh"
            
            if [[ -f "$adapter_api" ]]; then
                # Try to get description from the api.sh file
                local description=$(grep -m1 "^# Description:" "$adapter_api" 2>/dev/null | sed 's/^# Description: //')
                if [[ -z "$description" ]]; then
                    description="UI automation adapter for $adapter_name"
                fi
                
                echo "  ${adapter_name}: ${description}"
                
                # List available commands if api.sh exports them
                if grep -q "adapter::list_commands" "$adapter_api" 2>/dev/null; then
                    source "$adapter_api"
                    if declare -f adapter::list_commands >/dev/null; then
                        adapter::list_commands | sed 's/^/    - /'
                    fi
                fi
            fi
        fi
    done
    
    echo
    log::info "Usage: resource-browserless for <adapter> <command> [args]"
    return 0
}

#######################################
# Load adapter for a specific resource
# Sources the adapter implementation
# Arguments:
#   $1 - Adapter/resource name
# Returns:
#   0 on success, 1 on failure
#######################################
adapter::load() {
    local adapter_name="${1:?Adapter name required}"
    local adapter_dir="${BROWSERLESS_RESOURCE_DIR}/adapters/${adapter_name}"
    local adapter_api="${adapter_dir}/api.sh"
    
    if [[ ! -d "$adapter_dir" ]]; then
        log::error "Adapter not found: $adapter_name"
        log::info "Available adapters:"
        adapter::list
        return 1
    fi
    
    if [[ ! -f "$adapter_api" ]]; then
        log::error "Adapter API not found: $adapter_api"
        return 1
    fi
    
    # Initialize adapter framework
    adapter::init "$adapter_name"
    
    # Source the adapter
    source "$adapter_api"
    
    log::debug "Loaded adapter: $adapter_name"
    return 0
}

# Export functions for use by adapters
export -f adapter::init
export -f adapter::check_browserless_health
export -f adapter::load_target_config
export -f adapter::execute_browser_function
export -f adapter::register_capability
export -f adapter::list
export -f adapter::load