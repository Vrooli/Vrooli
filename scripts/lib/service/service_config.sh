#!/usr/bin/env bash
# Service configuration inheritance and template resolution
# Provides runtime resolution of service.json inheritance and template substitution

set -euo pipefail

# Source var.sh with relative path first
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_SYSTEM_COMMANDS_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

#######################################
# Check if service.json has inheritance defined
# Arguments:
#   $1 - Path to service.json file
# Returns:
#   0 if inheritance exists, 1 otherwise
#######################################
service_config::has_inheritance() {
    local service_json="$1"
    
    if [[ ! -f "$service_json" ]]; then
        return 1
    fi
    
    local extends
    extends=$(jq -r '.inheritance.extends // empty' "$service_json" 2>/dev/null || echo "")
    
    [[ -n "$extends" ]]
}

#######################################
# Resolve inheritance chain for service.json
# Arguments:
#   $1 - Path to service.json file
#   $2 - Visited files (for circular dependency detection)
# Outputs:
#   Merged JSON configuration with inheritance resolved
#######################################
service_config::resolve_inheritance() {
    local service_json="$1"
    local visited="${2:-}"
    
    if [[ ! -f "$service_json" ]]; then
        log::error "Service configuration not found: $service_json"
        return 1
    fi
    
    # Check for circular dependencies
    if [[ " $visited " == *" $service_json "* ]]; then
        log::error "Circular inheritance detected: $service_json"
        return 1
    fi
    visited="$visited $service_json"
    
    # Load current configuration
    local current_config
    current_config=$(jq '.' "$service_json")
    
    # Check for inheritance
    local extends
    extends=$(echo "$current_config" | jq -r '.inheritance.extends // empty')
    
    if [[ -z "$extends" ]]; then
        # No inheritance, return current config
        echo "$current_config"
        return 0
    fi
    
    # Handle single or multiple inheritance
    local parent_configs=()
    if echo "$extends" | jq -e 'type == "array"' >/dev/null 2>&1; then
        # Multiple inheritance
        while IFS= read -r parent_path; do
            parent_configs+=("$parent_path")
        done < <(echo "$extends" | jq -r '.[]')
    else
        # Single inheritance
        parent_configs=("$extends")
    fi
    
    # Start with empty base for merging
    local merged_config='{}'
    
    # Process each parent
    for parent_path in "${parent_configs[@]}"; do
        # Resolve relative path
        local resolved_parent_path
        if [[ "$parent_path" == /* ]]; then
            # Absolute path
            resolved_parent_path="$parent_path"
        else
            # Relative path - resolve from current service.json directory
            local current_dir
            current_dir=$(dirname "$service_json")
            resolved_parent_path="$(cd "$current_dir" && realpath "$parent_path")"
        fi
        
        if [[ ! -f "$resolved_parent_path" ]]; then
            log::error "Parent configuration not found: $resolved_parent_path (referenced from $service_json)"
            return 1
        fi
        
        log::debug "Resolving parent: $resolved_parent_path"
        
        # Recursively resolve parent
        local parent_config
        if ! parent_config=$(service_config::resolve_inheritance "$resolved_parent_path" "$visited"); then
            return 1
        fi
        
        # Merge parent into result
        merged_config=$(service_config::merge_configs "$merged_config" "$parent_config")
    done
    
    # Get merge strategy
    local merge_strategy
    merge_strategy=$(echo "$current_config" | jq -r '.inheritance.merge.strategy // "merge"')
    
    # Apply current config based on strategy
    case "$merge_strategy" in
        "replace")
            # Current completely replaces parent
            echo "$current_config"
            ;;
        "merge"|*)
            # Default: deep merge current over parent
            service_config::merge_configs "$merged_config" "$current_config"
            ;;
    esac
}

#######################################
# Deep merge two JSON configurations
# Arguments:
#   $1 - Base JSON configuration
#   $2 - Override JSON configuration  
# Outputs:
#   Merged JSON configuration
#######################################
service_config::merge_configs() {
    local base="$1"
    local override="$2"
    
    # Use jq to perform deep merge
    jq -s '
        def deepmerge(a; b):
            a as $a | b as $b |
            if ($a | type) == "object" and ($b | type) == "object" then
                $a | with_entries(
                    .value = deepmerge(.value; $b[.key])
                ) + ($b | with_entries(select(.key | in($a) | not)))
            elif ($a | type) == "array" and ($b | type) == "array" then
                $a + $b | unique
            else
                $b
            end;
        deepmerge(.[0]; .[1])
    ' <(echo "$base") <(echo "$override")
}

#######################################
# Load and process service configuration
# Arguments:
#   $1 - Path to service.json file
#   $2 - Whether to substitute templates (yes/no, default: auto)
# Outputs:
#   Processed JSON configuration
#######################################
service_config::load_and_process() {
    local service_json="$1"
    local substitute="${2:-auto}"
    
    if [[ ! -f "$service_json" ]]; then
        log::error "Service configuration not found: $service_json"
        return 1
    fi
    
    log::debug "Loading service configuration: $service_json"
    
    # Resolve inheritance
    local resolved_config
    if ! resolved_config=$(service_config::resolve_inheritance "$service_json"); then
        log::error "Failed to resolve inheritance for: $service_json"
        return 1
    fi
    
    # Determine whether to substitute templates
    local should_substitute="no"
    case "$substitute" in
        "yes")
            should_substitute="yes"
            ;;
        "no")
            should_substitute="no"
            ;;
        "auto"|*)
            # Auto: substitute if no inheritance is defined
            if ! service_config::has_inheritance "$service_json"; then
                should_substitute="yes"
                log::debug "No inheritance found, will substitute templates"
            else
                should_substitute="no"
                log::debug "Inheritance found, preserving templates for runtime resolution"
            fi
            ;;
    esac
    
    # Apply template substitution if needed
    if [[ "$should_substitute" == "yes" ]]; then
        log::debug "Applying template substitution"
        resolved_config=$(echo "$resolved_config" | secrets::substitute_all_templates)
    fi
    
    echo "$resolved_config"
}

#######################################
# Extract resource URLs from service configuration
# Arguments:
#   $1 - Path to service.json file
# Outputs:
#   Export environment variables for each resource URL
#######################################
service_config::export_resource_urls() {
    local service_json="$1"
    
    # Load and process configuration with template substitution
    local config
    if ! config=$(service_config::load_and_process "$service_json" "yes"); then
        log::error "Failed to load service configuration"
        return 1
    fi
    
    # Source port registry for resource port information
    local port_registry="${var_SCRIPTS_RESOURCES_DIR:-${LIB_SERVICE_DIR}/../../resources}/port_registry.sh"
    if [[ -f "$port_registry" ]]; then
        # shellcheck disable=SC1090
        source "$port_registry"
    fi
    
    # Export URLs for each known resource
    if [[ -n "${RESOURCE_PORTS[*]:-}" ]]; then
        for resource in "${!RESOURCE_PORTS[@]}"; do
            local port="${RESOURCE_PORTS[$resource]}"
            local var_name="${resource^^}_URL"
            var_name="${var_name//-/_}"  # Replace hyphens with underscores
            export "${var_name}=http://localhost:${port}"
            log::debug "Exported ${var_name}=http://localhost:${port}"
        done
    fi
}

#######################################
# Validate service configuration structure
# Arguments:
#   $1 - Path to service.json file
# Returns:
#   0 if valid, 1 if invalid
#######################################
service_config::validate() {
    local service_json="$1"
    
    if [[ ! -f "$service_json" ]]; then
        log::error "Service configuration not found: $service_json"
        return 1
    fi
    
    # Check if valid JSON
    if ! jq empty "$service_json" 2>/dev/null; then
        log::error "Invalid JSON in service configuration: $service_json"
        return 1
    fi
    
    # Check required fields
    local has_version
    has_version=$(jq -r '.version // empty' "$service_json")
    if [[ -z "$has_version" ]]; then
        log::error "Missing required field 'version' in service configuration"
        return 1
    fi
    
    local has_service
    has_service=$(jq -r '.service // empty' "$service_json")
    if [[ -z "$has_service" ]]; then
        log::error "Missing required field 'service' in service configuration"
        return 1
    fi
    
    log::debug "Service configuration is valid: $service_json"
    return 0
}

# Export functions if sourced
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    export -f service_config::has_inheritance
    export -f service_config::resolve_inheritance
    export -f service_config::merge_configs
    export -f service_config::load_and_process
    export -f service_config::export_resource_urls
    export -f service_config::validate
fi