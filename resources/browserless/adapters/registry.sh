#!/usr/bin/env bash

#######################################
# Browserless Adapter Registry
#
# Manages registration, discovery, and routing of adapters.
# This allows the system to know what capabilities browserless
# can provide as fallbacks or alternatives for other resources.
#######################################

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
REGISTRY_DIR="${APP_ROOT}/resources/browserless/adapters"
BROWSERLESS_RESOURCE_DIR="${APP_ROOT}/resources/browserless"

# Source common utilities
source "${REGISTRY_DIR}/common.sh"

#######################################
# Initialize adapter registry
# Creates or validates the registry file
# Returns:
#   0 on success
#######################################
registry::init() {
    local registry_file="${var_ROOT_DIR}/.vrooli/adapter-registry.json"
    local registry_dir=${registry_file%/*
    
    # Ensure directory exists
    mkdir -p "$registry_dir"
    
    # Create registry if it doesn't exist
    if [[ ! -f "$registry_file" ]]; then
        cat > "$registry_file" <<EOF
{
  "version": "1.0.0",
  "adapters": [],
  "capabilities": {},
  "created": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "updated": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
        log::debug "Created adapter registry at: $registry_file"
    fi
    
    return 0
}

#######################################
# Discover and register all available adapters
# Scans the adapters directory and registers capabilities
# Returns:
#   0 on success
#######################################
registry::discover() {
    local adapters_dir="${BROWSERLESS_RESOURCE_DIR}/adapters"
    local discovered=0
    
    log::debug "Discovering adapters in: $adapters_dir"
    
    # Initialize registry
    registry::init
    
    # Scan each adapter directory
    for adapter_dir in "$adapters_dir"/*; do
        if [[ -d "$adapter_dir" ]]; then
            local adapter_name=$(basename "$adapter_dir")
            local manifest_file="${adapter_dir}/manifest.json"
            
            # Skip common and registry
            if [[ "$adapter_name" == "common.sh" ]] || [[ "$adapter_name" == "registry.sh" ]]; then
                continue
            fi
            
            # Check for manifest file
            if [[ -f "$manifest_file" ]]; then
                log::debug "Found adapter manifest: $adapter_name"
                registry::register_from_manifest "$adapter_name" "$manifest_file"
                ((discovered++))
            elif [[ -f "${adapter_dir}/api.sh" ]]; then
                # Auto-discover from api.sh if no manifest
                log::debug "Auto-discovering adapter: $adapter_name"
                registry::auto_discover "$adapter_name" "${adapter_dir}/api.sh"
                ((discovered++))
            fi
        fi
    done
    
    log::info "Discovered $discovered adapter(s)"
    return 0
}

#######################################
# Register adapter from manifest file
# Arguments:
#   $1 - Adapter name
#   $2 - Manifest file path
# Returns:
#   0 on success
#######################################
registry::register_from_manifest() {
    local adapter_name="${1:?Adapter name required}"
    local manifest_file="${2:?Manifest file required}"
    
    if [[ ! -f "$manifest_file" ]]; then
        log::error "Manifest file not found: $manifest_file"
        return 1
    fi
    
    # Validate manifest JSON
    if ! jq . "$manifest_file" >/dev/null 2>&1; then
        log::error "Invalid manifest JSON: $manifest_file"
        return 1
    fi
    
    # Extract capabilities from manifest
    local capabilities=$(jq -r '.capabilities[]' "$manifest_file" 2>/dev/null)
    
    if [[ -n "$capabilities" ]]; then
        while IFS= read -r capability; do
            adapter::register_capability "$adapter_name" "$capability" "5" "From manifest"
        done <<< "$capabilities"
    fi
    
    return 0
}

#######################################
# Auto-discover adapter capabilities from api.sh
# Arguments:
#   $1 - Adapter name
#   $2 - API file path
# Returns:
#   0 on success
#######################################
registry::auto_discover() {
    local adapter_name="${1:?Adapter name required}"
    local api_file="${2:?API file required}"
    
    if [[ ! -f "$api_file" ]]; then
        log::error "API file not found: $api_file"
        return 1
    fi
    
    # Look for exported functions that match adapter patterns
    local functions=$(grep -E "^[[:space:]]*${adapter_name}::|^[[:space:]]*adapter::" "$api_file" | \
                     sed -E 's/^[[:space:]]*//; s/\(\).*//' | \
                     grep -v "^#" | \
                     sort -u)
    
    if [[ -n "$functions" ]]; then
        while IFS= read -r function_name; do
            # Extract capability name from function
            local capability="${function_name#*::}"
            if [[ -n "$capability" ]] && [[ "$capability" != "init" ]] && [[ "$capability" != "load" ]]; then
                adapter::register_capability "$adapter_name" "$capability" "5" "Auto-discovered"
            fi
        done <<< "$functions"
    fi
    
    # Always register base capability
    adapter::register_capability "$adapter_name" "ui-automation" "1" \
        "Provides UI automation interface for $adapter_name"
    
    return 0
}

#######################################
# Find adapter for a capability
# Searches registry for best adapter to provide capability
# Arguments:
#   $1 - Resource name
#   $2 - Capability name
# Returns:
#   0 if found, 1 if not
# Outputs:
#   Adapter name if found
#######################################
registry::find_adapter() {
    local resource="${1:?Resource name required}"
    local capability="${2:?Capability name required}"
    local registry_file="${var_ROOT_DIR}/.vrooli/adapter-registry.json"
    
    if [[ ! -f "$registry_file" ]]; then
        return 1
    fi
    
    # Find adapter with best priority for this capability
    local adapter=$(jq -r --arg resource "$resource" \
                          --arg capability "$capability" \
                          '.adapters | 
                           map(select(.resource == $resource and .capability == $capability)) |
                           sort_by(.priority) |
                           .[0].provider // empty' \
                          "$registry_file")
    
    if [[ -n "$adapter" ]]; then
        echo "$adapter"
        return 0
    fi
    
    return 1
}

#######################################
# List all registered capabilities
# Shows what adapters can do
# Returns:
#   0 on success
#######################################
registry::list_capabilities() {
    local registry_file="${var_ROOT_DIR}/.vrooli/adapter-registry.json"
    
    if [[ ! -f "$registry_file" ]]; then
        log::warn "No adapter registry found"
        return 1
    fi
    
    log::header "Registered Adapter Capabilities"
    echo
    
    # Group by resource
    local resources=$(jq -r '.adapters[].resource' "$registry_file" 2>/dev/null | sort -u)
    
    if [[ -z "$resources" ]]; then
        log::info "No capabilities registered yet"
        log::info "Run 'resource-browserless adapter discover' to scan for adapters"
        return 0
    fi
    
    while IFS= read -r resource; do
        echo "Resource: $resource"
        
        # List capabilities for this resource
        jq -r --arg resource "$resource" \
           '.adapters[] | 
            select(.resource == $resource) | 
            "  - \(.capability) (priority: \(.priority)) - \(.description)"' \
           "$registry_file" 2>/dev/null
        
        echo
    done <<< "$resources"
    
    return 0
}

#######################################
# Clean stale entries from registry
# Removes entries for adapters that no longer exist
# Returns:
#   0 on success
#######################################
registry::clean() {
    local registry_file="${var_ROOT_DIR}/.vrooli/adapter-registry.json"
    local adapters_dir="${BROWSERLESS_RESOURCE_DIR}/adapters"
    
    if [[ ! -f "$registry_file" ]]; then
        return 0
    fi
    
    log::info "Cleaning adapter registry..."
    
    # Get list of valid adapter names
    local valid_adapters=()
    for adapter_dir in "$adapters_dir"/*; do
        if [[ -d "$adapter_dir" ]]; then
            local adapter_name=$(basename "$adapter_dir")
            if [[ -f "${adapter_dir}/api.sh" ]]; then
                valid_adapters+=("$adapter_name")
            fi
        fi
    done
    
    # Filter registry to only include valid adapters
    local temp_file=$(mktemp)
    local valid_list=$(printf '"%s",' "${valid_adapters[@]}" | sed 's/,$//')
    
    jq --argjson valid "[$valid_list]" \
       '.adapters = [.adapters[] | select(.resource as $r | $valid | index($r))]' \
       "$registry_file" > "$temp_file"
    
    mv "$temp_file" "$registry_file"
    
    log::info "Registry cleaned"
    return 0
}

# Export registry functions
export -f registry::init
export -f registry::discover
export -f registry::register_from_manifest
export -f registry::auto_discover
export -f registry::find_adapter
export -f registry::list_capabilities
export -f registry::clean