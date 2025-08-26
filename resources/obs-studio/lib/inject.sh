#!/usr/bin/env bash
set -euo pipefail

DESCRIPTION="Inject scenes, sources, and configurations into OBS Studio"

# Source guard to prevent multiple sourcing
[[ -n "${_OBS_INJECT_SOURCED:-}" ]] && return 0
export _OBS_INJECT_SOURCED=1

# Get script directory and source framework
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OBS_INJECT_LIB_DIR="${APP_ROOT}/resources/obs-studio/lib"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/inject_framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/validation_utils.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

# Load OBS configuration and infrastructure
if command -v inject_framework::load_adapter_config &>/dev/null; then
    inject_framework::load_adapter_config "obs-studio" "${OBS_INJECT_LIB_DIR%/*}"
fi

# Source OBS lib functions
for lib_file in "${OBS_INJECT_LIB_DIR}/core.sh"; do
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" || log::warn "Could not load $lib_file"
    fi
done

# OBS-specific configuration with readonly protection
if [[ -z "${OBS_WEBSOCKET_URL:-}" ]]; then
    OBS_WEBSOCKET_URL="ws://localhost:4455"
fi
if ! readonly -p | grep -q "OBS_WEBSOCKET_URL"; then
    readonly OBS_WEBSOCKET_URL
fi

#######################################
# OBS-specific health check
# Returns: 0 if healthy, 1 otherwise
#######################################
obs::check_health() {
    if obs::is_running; then
        log::debug "OBS Studio is healthy and ready for data injection"
        return 0
    else
        log::error "OBS Studio is not accessible for data injection"
        log::info "Ensure OBS Studio is running: vrooli resource obs-studio start"
        return 1
    fi
}

#######################################
# Validate individual scene configuration
# Arguments:
#   $1 - scene configuration object
#   $2 - index
#   $3 - scene name
# Returns:
#   0 if valid, 1 if invalid
#######################################
obs::validate_scene() {
    local scene="$1"
    local index="$2"
    local name="$3"
    
    # Validate required fields
    local file
    file=$(echo "$scene" | jq -r '.file // empty')
    
    if [[ -z "$file" ]]; then
        log::error "Scene $index ($name) missing required field: file"
        return 1
    fi
    
    # Validate file exists
    if [[ ! -f "$file" ]]; then
        log::error "Scene $index ($name) file not found: $file"
        return 1
    fi
    
    # Validate JSON format
    if ! jq empty "$file" 2>/dev/null; then
        log::error "Scene $index ($name) file is not valid JSON: $file"
        return 1
    fi
    
    return 0
}

#######################################
# Inject a single scene into OBS
# Arguments:
#   $1 - scene file path
#   $2 - scene name (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
obs::inject_scene() {
    local scene_file="$1"
    local scene_name="${2:-}"
    
    if [[ -z "$scene_name" ]]; then
        scene_name=$(basename "$scene_file" .json)
    fi
    
    log::info "Injecting scene: $scene_name from $scene_file"
    
    # Read scene configuration
    local scene_config
    scene_config=$(cat "$scene_file")
    
    # Since OBS websocket is already configured and running, we can use the existing
    # WebSocket command function from core.sh
    log::info "Creating scene: $scene_name"
    
    # Create the scene using the WebSocket API
    local create_result
    create_result=$(obs::websocket_command "CreateScene" "{\"sceneName\": \"$scene_name\"}" 2>&1)
    
    if [[ $? -eq 0 ]] || [[ "$create_result" == *"already exists"* ]]; then
        log::success "Scene ready: $scene_name"
        
        # Add sources if specified
        local sources
        sources=$(echo "$scene_config" | jq -c '.sources // []')
        
        if [[ "$sources" != "[]" ]]; then
            echo "$sources" | jq -c '.[]' | while read -r source; do
                local source_name
                source_name=$(echo "$source" | jq -r '.name')
                local source_type
                source_type=$(echo "$source" | jq -r '.type // "ffmpeg_source"')
                local source_settings
                source_settings=$(echo "$source" | jq -c '.settings // {}')
                
                log::info "Adding source: $source_name (type: $source_type)"
                
                # Create input source
                local input_data
                input_data=$(jq -n \
                    --arg scene "$scene_name" \
                    --arg name "$source_name" \
                    --arg kind "$source_type" \
                    --argjson settings "$source_settings" \
                    '{sceneName: $scene, sourceName: $name, sourceKind: $kind, sourceSettings: $settings}')
                
                obs::websocket_command "CreateInput" "$input_data" >/dev/null 2>&1 || {
                    log::warn "Source may already exist or type not supported: $source_name"
                }
            done
        fi
        
        return 0
    else
        log::error "Failed to create scene: $create_result"
        return 1
    fi
}

#######################################
# Process all scenes in a configuration
# Arguments:
#   $1 - configuration object
# Returns:
#   0 on success, 1 on failure
#######################################
obs::process_scenes() {
    local config="$1"
    local scenes
    scenes=$(echo "$config" | jq -c '.scenes // []')
    
    if [[ "$scenes" == "[]" ]]; then
        log::info "No scenes to inject"
        return 0
    fi
    
    local total_scenes
    total_scenes=$(echo "$scenes" | jq 'length')
    log::info "Processing $total_scenes scene(s)"
    
    local failed=0
    local index=0
    
    echo "$scenes" | jq -c '.[]' | while read -r scene; do
        local name
        name=$(echo "$scene" | jq -r '.name // "scene_'$index'"')
        
        if obs::validate_scene "$scene" "$index" "$name"; then
            local file
            file=$(echo "$scene" | jq -r '.file')
            
            if obs::inject_scene "$file" "$name"; then
                log::success "Scene $index ($name) injected successfully"
            else
                log::error "Failed to inject scene $index ($name)"
                ((failed++))
            fi
        else
            ((failed++))
        fi
        
        ((index++))
    done
    
    if [[ $failed -gt 0 ]]; then
        log::error "$failed scene(s) failed to inject"
        return 1
    fi
    
    return 0
}

#######################################
# Main injection entry point
# Arguments:
#   $1 - configuration file or object
# Returns:
#   0 on success, 1 on failure
#######################################
obs::inject() {
    local config="$1"
    
    # Check if OBS is healthy first
    if ! obs::check_health; then
        return 1
    fi
    
    # Parse configuration
    if [[ -f "$config" ]]; then
        # If it's a single scene file, wrap it in a scenes array
        local file_content
        file_content=$(cat "$config")
        
        # Check if it's a single scene (has sources but not scenes)
        if echo "$file_content" | jq -e '.sources' >/dev/null 2>&1 && \
           ! echo "$file_content" | jq -e '.scenes' >/dev/null 2>&1; then
            # It's a single scene, inject it directly
            obs::inject_scene "$config"
            return $?
        else
            config="$file_content"
        fi
    fi
    
    # Validate JSON
    if ! echo "$config" | jq empty 2>/dev/null; then
        log::error "Invalid JSON configuration"
        return 1
    fi
    
    # Process scenes
    if ! obs::process_scenes "$config"; then
        log::error "Failed to process scenes"
        return 1
    fi
    
    # Process recordings configuration if present
    local recordings
    recordings=$(echo "$config" | jq -c '.recordings // {}')
    if [[ "$recordings" != "{}" ]]; then
        log::info "Configuring recording settings"
        # TODO: Implement recording configuration
    fi
    
    # Process streaming configuration if present
    local streaming
    streaming=$(echo "$config" | jq -c '.streaming // {}')
    if [[ "$streaming" != "{}" ]]; then
        log::info "Configuring streaming settings"
        # TODO: Implement streaming configuration
    fi
    
    log::success "OBS Studio injection completed successfully"
    return 0
}

# Export functions for external use
export -f obs::check_health
export -f obs::validate_scene
export -f obs::inject_scene
export -f obs::process_scenes
export -f obs::inject