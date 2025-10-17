#!/bin/bash
# Source management for OBS Studio

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OBS_SOURCES_DIR="${APP_ROOT}/resources/obs-studio/lib"
source "${OBS_SOURCES_DIR}/common.sh"

# Source configuration directory
OBS_SOURCES_CONFIG_DIR="${OBS_CONFIG_DIR}/sources"
mkdir -p "${OBS_SOURCES_CONFIG_DIR}"

#######################################
# Add a source to the current scene
# Args:
#   --name: Source name
#   --type: Source type (camera|screen|window|image|media|browser|text)
#   --settings: JSON settings for the source
#   --scene: Scene to add source to (optional, uses current)
#######################################
obs::sources::add() {
    local name=""
    local source_type=""
    local settings="{}"
    local scene=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                source_type="$2"
                shift 2
                ;;
            --settings)
                settings="$2"
                shift 2
                ;;
            --scene)
                scene="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" || -z "$source_type" ]]; then
        echo "[ERROR] --name and --type are required"
        return 1
    fi
    
    # Check if OBS is running
    if ! obs_is_running; then
        echo "[ERROR] OBS Studio is not running"
        return 1
    fi
    
    # Validate source type
    case "$source_type" in
        camera|screen|window|image|media|browser|text)
            ;;
        *)
            echo "[ERROR] Invalid source type: $source_type"
            echo "Valid types: camera, screen, window, image, media, browser, text"
            return 1
            ;;
    esac
    
    # Create source configuration
    local source_config=$(jq -n \
        --arg name "$name" \
        --arg type "$source_type" \
        --argjson settings "$settings" \
        --arg scene "${scene:-current}" \
        '{
            name: $name,
            type: $type,
            settings: $settings,
            scene: $scene,
            created_at: now | todate
        }')
    
    # Save source configuration
    echo "$source_config" > "${OBS_SOURCES_CONFIG_DIR}/${name}.json"
    
    echo "[SUCCESS] Source added: $name (type: $source_type)"
    
    # Apply type-specific defaults
    obs::sources::apply_defaults "$name" "$source_type"
    
    return 0
}

#######################################
# Remove a source
# Args:
#   --name: Source name
#######################################
obs::sources::remove() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "[ERROR] --name is required"
        return 1
    fi
    
    local source_file="${OBS_SOURCES_CONFIG_DIR}/${name}.json"
    if [[ -f "$source_file" ]]; then
        rm "$source_file"
        echo "[SUCCESS] Source removed: $name"
    else
        echo "[ERROR] Source not found: $name"
        return 1
    fi
    
    return 0
}

#######################################
# List all sources
# Args:
#   --format: Output format (text|json)
#   --scene: Filter by scene (optional)
#######################################
obs::sources::list() {
    local format="text"
    local scene_filter=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            --scene)
                scene_filter="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$format" == "json" ]]; then
        echo "["
        local first=true
        for source_file in "${OBS_SOURCES_CONFIG_DIR}"/*.json; do
            if [[ -f "$source_file" ]]; then
                local source=$(cat "$source_file")
                local scene=$(echo "$source" | jq -r '.scene')
                
                # Apply scene filter if specified
                if [[ -n "$scene_filter" && "$scene" != "$scene_filter" ]]; then
                    continue
                fi
                
                if [[ "$first" != "true" ]]; then echo ","; fi
                echo "$source" | jq -c .
                first=false
            fi
        done
        echo "]"
    else
        echo "[INFO] Available Sources:"
        
        local source_count=0
        for source_file in "${OBS_SOURCES_CONFIG_DIR}"/*.json; do
            if [[ -f "$source_file" ]]; then
                local name=$(basename "$source_file" .json)
                local type=$(jq -r '.type' "$source_file" 2>/dev/null)
                local scene=$(jq -r '.scene' "$source_file" 2>/dev/null)
                
                # Apply scene filter if specified
                if [[ -n "$scene_filter" && "$scene" != "$scene_filter" ]]; then
                    continue
                fi
                
                echo "  - $name"
                echo "      Type: $type"
                echo "      Scene: $scene"
                
                source_count=$((source_count + 1))
            fi
        done
        
        if [[ $source_count -eq 0 ]]; then
            echo "  (No sources configured)"
        fi
    fi
    
    return 0
}

#######################################
# Configure source properties
# Args:
#   --name: Source name
#   --property: Property to set
#   --value: Property value
#######################################
obs::sources::configure() {
    local name=""
    local property=""
    local value=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --property)
                property="$2"
                shift 2
                ;;
            --value)
                value="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" || -z "$property" ]]; then
        echo "[ERROR] --name and --property are required"
        return 1
    fi
    
    local source_file="${OBS_SOURCES_CONFIG_DIR}/${name}.json"
    if [[ ! -f "$source_file" ]]; then
        echo "[ERROR] Source not found: $name"
        return 1
    fi
    
    # Update source configuration
    local updated=$(jq \
        --arg prop "$property" \
        --arg val "$value" \
        '.settings[$prop] = $val' \
        "$source_file")
    
    echo "$updated" > "$source_file"
    
    echo "[SUCCESS] Source updated: $name"
    echo "[INFO] Set $property = $value"
    
    return 0
}

#######################################
# Apply default settings based on source type
#######################################
obs::sources::apply_defaults() {
    local name="$1"
    local source_type="$2"
    local source_file="${OBS_SOURCES_CONFIG_DIR}/${name}.json"
    
    case "$source_type" in
        camera)
            # Default camera settings
            jq '.settings += {
                device_id: "default",
                resolution: "1920x1080",
                fps: 30,
                format: "YUY2"
            }' "$source_file" > "${source_file}.tmp" && mv "${source_file}.tmp" "$source_file"
            ;;
        screen)
            # Default screen capture settings
            jq '.settings += {
                monitor: 0,
                capture_cursor: true,
                compatibility_mode: false
            }' "$source_file" > "${source_file}.tmp" && mv "${source_file}.tmp" "$source_file"
            ;;
        window)
            # Default window capture settings
            jq '.settings += {
                window: "",
                priority: 0,
                capture_cursor: true
            }' "$source_file" > "${source_file}.tmp" && mv "${source_file}.tmp" "$source_file"
            ;;
        image)
            # Default image settings
            jq '.settings += {
                file: "",
                unload_when_not_showing: false
            }' "$source_file" > "${source_file}.tmp" && mv "${source_file}.tmp" "$source_file"
            ;;
        media)
            # Default media source settings
            jq '.settings += {
                local_file: "",
                looping: false,
                restart_on_activate: true,
                hw_decode: true
            }' "$source_file" > "${source_file}.tmp" && mv "${source_file}.tmp" "$source_file"
            ;;
        browser)
            # Default browser source settings
            jq '.settings += {
                url: "https://example.com",
                width: 1920,
                height: 1080,
                fps: 30,
                css: ""
            }' "$source_file" > "${source_file}.tmp" && mv "${source_file}.tmp" "$source_file"
            ;;
        text)
            # Default text source settings
            jq '.settings += {
                text: "Sample Text",
                font: {
                    face: "Arial",
                    size: 72,
                    style: "Regular"
                },
                color: "#FFFFFF",
                outline: true,
                outline_color: "#000000",
                outline_size: 2
            }' "$source_file" > "${source_file}.tmp" && mv "${source_file}.tmp" "$source_file"
            ;;
    esac
}

#######################################
# Get available cameras
#######################################
obs::sources::cameras() {
    echo "[INFO] Available Cameras:"
    
    # In production, this would query actual devices
    # For mock mode, return simulated devices
    if [[ -f "${OBS_DATA_DIR}/.mock_mode" ]]; then
        echo "  - Built-in Webcam (device_id: webcam0)"
        echo "  - USB Camera (device_id: usb_cam1)"
        echo "  - Virtual Camera (device_id: virtual0)"
    else
        # Try to list actual video devices
        if [[ -d /dev ]]; then
            local found=false
            for device in /dev/video*; do
                if [[ -e "$device" ]]; then
                    echo "  - $device"
                    found=true
                fi
            done
            
            if [[ "$found" == "false" ]]; then
                echo "  (No cameras detected)"
            fi
        else
            echo "  (Cannot detect cameras)"
        fi
    fi
    
    return 0
}

#######################################
# Get available audio devices
#######################################
obs::sources::audio_devices() {
    echo "[INFO] Available Audio Devices:"
    
    # In production, this would query actual devices
    # For mock mode, return simulated devices
    if [[ -f "${OBS_DATA_DIR}/.mock_mode" ]]; then
        echo "  Input Devices:"
        echo "    - Built-in Microphone (device_id: mic0)"
        echo "    - USB Microphone (device_id: usb_mic1)"
        echo "  Output Devices:"
        echo "    - Built-in Speakers (device_id: speakers0)"
        echo "    - Headphones (device_id: headphones1)"
    else
        # Try to list actual audio devices using pactl if available
        if command -v pactl &>/dev/null; then
            echo "  Input Devices:"
            pactl list sources short 2>/dev/null | awk '{print "    - " $2}' || echo "    (Cannot list devices)"
            echo "  Output Devices:"
            pactl list sinks short 2>/dev/null | awk '{print "    - " $2}' || echo "    (Cannot list devices)"
        else
            echo "  (PulseAudio not available)"
        fi
    fi
    
    return 0
}

#######################################
# Preview a source
# Args:
#   --name: Source name
#######################################
obs::sources::preview() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "[ERROR] --name is required"
        return 1
    fi
    
    local source_file="${OBS_SOURCES_CONFIG_DIR}/${name}.json"
    if [[ ! -f "$source_file" ]]; then
        echo "[ERROR] Source not found: $name"
        return 1
    fi
    
    local type=$(jq -r '.type' "$source_file")
    
    echo "[INFO] Previewing source: $name (type: $type)"
    
    # In production, this would open a preview window
    # For now, we just show the configuration
    jq '.settings' "$source_file"
    
    return 0
}

#######################################
# Set source visibility
# Args:
#   --name: Source name
#   --visible: true|false
#######################################
obs::sources::visibility() {
    local name=""
    local visible=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --visible)
                visible="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" || -z "$visible" ]]; then
        echo "[ERROR] --name and --visible are required"
        return 1
    fi
    
    local source_file="${OBS_SOURCES_CONFIG_DIR}/${name}.json"
    if [[ ! -f "$source_file" ]]; then
        echo "[ERROR] Source not found: $name"
        return 1
    fi
    
    # Update visibility
    jq --arg vis "$visible" '.visible = ($vis == "true")' "$source_file" > "${source_file}.tmp" && \
        mv "${source_file}.tmp" "$source_file"
    
    if [[ "$visible" == "true" ]]; then
        echo "[SUCCESS] Source $name is now visible"
    else
        echo "[SUCCESS] Source $name is now hidden"
    fi
    
    return 0
}

# Export functions
export -f obs::sources::add
export -f obs::sources::remove
export -f obs::sources::list
export -f obs::sources::configure
export -f obs::sources::apply_defaults
export -f obs::sources::cameras
export -f obs::sources::audio_devices
export -f obs::sources::preview
export -f obs::sources::visibility