#!/usr/bin/env bash
# Lifecycle Engine - Target Management Module
# Handles target-specific configurations and inheritance

set -euo pipefail

# Get script directory
LIB_LIFECYCLE_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source common environment variables
source "$LIB_LIFECYCLE_LIB_DIR/../../utils/var.sh"

# Guard against re-sourcing
[[ -n "${_TARGETS_MODULE_LOADED:-}" ]] && return 0
declare -gr _TARGETS_MODULE_LOADED=1

# Source dependencies
# shellcheck source=./config.sh
source "$var_LIB_LIFECYCLE_DIR/lib/config.sh"

# Target resolution cache
declare -gA TARGET_CACHE=()

#######################################
# Get target configuration with inheritance
# Arguments:
#   $1 - Target name
#   $2 - Phase configuration JSON (optional)
# Returns:
#   Merged target configuration JSON
#######################################
targets::get_config() {
    local target="${1:-default}"
    local phase_config="${2:-$PHASE_CONFIG}"
    
    # Check cache
    local cache_key="${LIFECYCLE_PHASE}.${target}"
    if [[ -n "${TARGET_CACHE[$cache_key]:-}" ]]; then
        echo "${TARGET_CACHE[$cache_key]}"
        return 0
    fi
    
    # Get targets section
    local targets
    targets=$(echo "$phase_config" | jq -r '.targets // {}')
    
    if [[ "$targets" == "{}" ]]; then
        targets::log_debug "No target-specific configuration found"
        echo "{}"
        return 0
    fi
    
    # Get specific target config
    local target_config
    target_config=$(echo "$targets" | jq ".\"$target\" // empty")
    
    # If target not found, try default
    if [[ -z "$target_config" ]] || [[ "$target_config" == "null" ]]; then
        if [[ "$target" != "default" ]]; then
            targets::log_debug "Target '$target' not found, trying default"
            target_config=$(echo "$targets" | jq '.default // empty')
        fi
        
        if [[ -z "$target_config" ]] || [[ "$target_config" == "null" ]]; then
            targets::log_debug "No configuration for target: $target"
            echo "{}"
            return 0
        fi
    fi
    
    # Resolve inheritance
    local resolved_config
    resolved_config=$(targets::resolve_inheritance "$target_config" "$phase_config")
    
    # Cache result
    TARGET_CACHE["$cache_key"]="$resolved_config"
    
    echo "$resolved_config"
}

#######################################
# Resolve target inheritance chain
# Arguments:
#   $1 - Target configuration JSON
#   $2 - Phase configuration JSON
# Returns:
#   Resolved configuration with inheritance applied
#######################################
targets::resolve_inheritance() {
    local target_config="$1"
    local phase_config="$2"
    
    # Check for extends field
    local extends
    extends=$(echo "$target_config" | jq -r '.extends // empty')
    
    if [[ -z "$extends" ]] || [[ "$extends" == "null" ]]; then
        # No inheritance
        echo "$target_config"
        return 0
    fi
    
    targets::log_debug "Target extends: $extends"
    
    # Get parent configuration
    local parent_config
    parent_config=$(targets::get_parent_config "$extends" "$phase_config")
    
    # Check override strategy
    local override
    override=$(echo "$target_config" | jq -r '.override // false')
    
    if [[ "$override" == "true" ]]; then
        # Complete override - ignore parent
        echo "$target_config"
    else
        # Merge configurations
        targets::merge_configs "$parent_config" "$target_config"
    fi
}

#######################################
# Get parent target configuration
# Arguments:
#   $1 - Parent target name(s) - can be comma-separated
#   $2 - Phase configuration JSON
# Returns:
#   Merged parent configuration
#######################################
targets::get_parent_config() {
    local extends="$1"
    local phase_config="$2"
    
    # Handle multiple inheritance (comma-separated)
    if [[ "$extends" == *","* ]]; then
        local merged="{}"
        IFS=',' read -ra parents <<< "$extends"
        
        for parent in "${parents[@]}"; do
            parent="${parent// /}"  # Trim whitespace
            local parent_config
            parent_config=$(targets::get_single_parent "$parent" "$phase_config")
            merged=$(targets::merge_configs "$merged" "$parent_config")
        done
        
        echo "$merged"
    else
        # Single inheritance
        targets::get_single_parent "$extends" "$phase_config"
    fi
}

#######################################
# Get single parent configuration
# Arguments:
#   $1 - Parent target name
#   $2 - Phase configuration JSON
# Returns:
#   Parent configuration JSON
#######################################
targets::get_single_parent() {
    local parent="$1"
    local phase_config="$2"
    
    local targets
    targets=$(echo "$phase_config" | jq -r '.targets // {}')
    
    local parent_config
    parent_config=$(echo "$targets" | jq ".\"$parent\" // empty")
    
    if [[ -z "$parent_config" ]] || [[ "$parent_config" == "null" ]]; then
        targets::log_warning "Parent target not found: $parent"
        echo "{}"
        return 0
    fi
    
    # Recursively resolve parent's inheritance
    targets::resolve_inheritance "$parent_config" "$phase_config"
}

#######################################
# Merge two target configurations
# Arguments:
#   $1 - Base configuration JSON
#   $2 - Override configuration JSON
# Returns:
#   Merged configuration
#######################################
targets::merge_configs() {
    local base="$1"
    local override="$2"
    
    # Get merge strategy
    local merge_strategy
    merge_strategy=$(echo "$override" | jq -r '.merge // "append"')
    
    case "$merge_strategy" in
        append)
            targets::merge_append "$base" "$override"
            ;;
        prepend)
            targets::merge_prepend "$base" "$override"
            ;;
        replace)
            echo "$override"
            ;;
        deep)
            targets::merge_deep "$base" "$override"
            ;;
        *)
            targets::log_warning "Unknown merge strategy: $merge_strategy"
            targets::merge_append "$base" "$override"
            ;;
    esac
}

#######################################
# Append merge strategy (default)
# Arguments:
#   $1 - Base configuration JSON
#   $2 - Override configuration JSON
# Returns:
#   Merged configuration with override steps appended
#######################################
targets::merge_append() {
    local base="$1"
    local override="$2"
    
    # Extract steps
    local base_steps override_steps
    base_steps=$(echo "$base" | jq '.steps // []')
    override_steps=$(echo "$override" | jq '.steps // []')
    
    # Combine steps (base + override)
    local combined_steps
    combined_steps=$(jq -n --argjson b "$base_steps" --argjson o "$override_steps" '$b + $o')
    
    # Merge other fields
    local merged
    merged=$(jq -n --argjson base "$base" --argjson override "$override" --argjson steps "$combined_steps" '
        $base * $override | .steps = $steps
    ')
    
    echo "$merged"
}

#######################################
# Prepend merge strategy
# Arguments:
#   $1 - Base configuration JSON
#   $2 - Override configuration JSON
# Returns:
#   Merged configuration with override steps prepended
#######################################
targets::merge_prepend() {
    local base="$1"
    local override="$2"
    
    # Extract steps
    local base_steps override_steps
    base_steps=$(echo "$base" | jq '.steps // []')
    override_steps=$(echo "$override" | jq '.steps // []')
    
    # Combine steps (override + base)
    local combined_steps
    combined_steps=$(jq -n --argjson b "$base_steps" --argjson o "$override_steps" '$o + $b')
    
    # Merge other fields
    local merged
    merged=$(jq -n --argjson base "$base" --argjson override "$override" --argjson steps "$combined_steps" '
        $base * $override | .steps = $steps
    ')
    
    echo "$merged"
}

#######################################
# Deep merge strategy
# Arguments:
#   $1 - Base configuration JSON
#   $2 - Override configuration JSON
# Returns:
#   Deep merged configuration
#######################################
targets::merge_deep() {
    local base="$1"
    local override="$2"
    
    # Deep merge using jq
    jq -n --argjson base "$base" --argjson override "$override" '
        def deepmerge(a; b):
            a as $a | b as $b |
            if ($a|type) == "object" and ($b|type) == "object" then
                reduce ($b|keys[]) as $key (
                    $a;
                    if $a|has($key) then
                        .[$key] = deepmerge($a[$key]; $b[$key])
                    else
                        .[$key] = $b[$key]
                    end
                )
            elif ($a|type) == "array" and ($b|type) == "array" then
                $a + $b
            else
                $b
            end;
        deepmerge($base; $override)
    '
}

#######################################
# List available targets for phase
# Arguments:
#   $1 - Phase configuration JSON (optional)
# Returns:
#   List of target names
#######################################
targets::list() {
    local phase_config="${1:-$PHASE_CONFIG}"
    
    local targets
    targets=$(echo "$phase_config" | jq -r '.targets // {}')
    
    if [[ "$targets" == "{}" ]]; then
        return 0
    fi
    
    echo "$targets" | jq -r 'keys[]' | sort
}

#######################################
# Validate target configuration
# Arguments:
#   $1 - Target name
# Returns:
#   0 if valid, 1 if invalid
#######################################
targets::validate() {
    local target="$1"
    
    if [[ -z "$target" ]]; then
        targets::log_error "Target name required"
        return 1
    fi
    
    # Check if ANY targets are defined in the phase
    local all_targets
    all_targets=$(config::get_targets)
    
    # If no targets are defined at all (only universal steps), any target is valid
    if [[ "$all_targets" == "{}" ]] || [[ -z "$all_targets" ]]; then
        targets::log_debug "No targets defined in phase config - accepting any target for universal steps"
        return 0
    fi
    
    # Check if target exists
    if ! config::target_exists "$target"; then
        # Check for default
        if config::target_exists "default"; then
            targets::log_info "Using default target configuration"
            return 0
        else
            targets::log_error "Target not found: $target"
            targets::log_info "Available targets:"
            targets::list | sed 's/^/  - /'
            return 1
        fi
    fi
    
    # Get and validate configuration
    local target_config
    target_config=$(targets::get_config "$target")
    
    if [[ "$target_config" == "{}" ]]; then
        targets::log_warning "Empty target configuration: $target"
    fi
    
    return 0
}

#######################################
# Get target-specific environment variables
# Arguments:
#   $1 - Target name
# Returns:
#   Environment variables as key=value pairs
#######################################
targets::get_env() {
    local target="$1"
    
    local target_config
    target_config=$(targets::get_config "$target")
    
    local env
    env=$(echo "$target_config" | jq -r '.env // {}')
    
    if [[ "$env" != "{}" ]] && [[ "$env" != "null" ]]; then
        echo "$env" | jq -r 'to_entries[] | "\(.key)=\(.value)"'
    fi
}

#######################################
# Get target-specific defaults
# Arguments:
#   $1 - Target name
#   $2 - Default key (timeout, shell, etc.)
# Returns:
#   Default value or empty
#######################################
targets::get_default() {
    local target="$1"
    local key="$2"
    
    local target_config
    target_config=$(targets::get_config "$target")
    
    echo "$target_config" | jq -r ".defaults.\"$key\" // empty"
}

#######################################
# Clear target cache
#######################################
targets::clear_cache() {
    TARGET_CACHE=()
}

#######################################
# Logging functions
#######################################
targets::log_debug() {
    [[ "${DEBUG:-false}" == "true" ]] && echo "[TARGETS-DEBUG] $*" >&2 || true
}

targets::log_info() {
    echo "[TARGETS] $*" >&2
}

targets::log_warning() {
    echo "[TARGETS-WARNING] $*" >&2
}

targets::log_error() {
    echo "[TARGETS-ERROR] $*" >&2
}

# If sourced for testing, don't auto-execute
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script should be sourced, not executed directly" >&2
    exit 1
fi