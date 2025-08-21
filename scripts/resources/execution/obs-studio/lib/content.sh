#!/bin/bash
# Content management for OBS Studio

set -euo pipefail

# Get script directory
OBS_CONTENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${OBS_CONTENT_DIR}/common.sh"

# Content storage directories
OBS_SCENES_DIR="${OBS_CONFIG_DIR}/scenes"
OBS_PROFILES_DIR="${OBS_CONFIG_DIR}/profiles"
OBS_RECORDINGS_PROFILES_DIR="${OBS_PROFILES_DIR}/recording"
OBS_STREAMING_PROFILES_DIR="${OBS_PROFILES_DIR}/streaming"

# Ensure directories exist
mkdir -p "${OBS_SCENES_DIR}"
mkdir -p "${OBS_RECORDINGS_PROFILES_DIR}"
mkdir -p "${OBS_STREAMING_PROFILES_DIR}"

#######################################
# Add content to OBS Studio
# Args:
#   --file: Path to content file
#   --type: Content type (scene|recording|streaming)
#   --name: Optional name override
#######################################
obs_content_add() {
    local file=""
    local content_type="scene"
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --type)
                content_type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                echo "[ERROR] Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        echo "[ERROR] --file is required"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        echo "[ERROR] File not found: $file"
        return 1
    fi
    
    # Determine name from file if not provided
    if [[ -z "$name" ]]; then
        name=$(basename "$file" .json)
    fi
    
    # Copy file to appropriate directory
    case "$content_type" in
        scene)
            cp "$file" "${OBS_SCENES_DIR}/${name}.json"
            echo "[SUCCESS] Added scene: ${name}"
            ;;
        recording)
            cp "$file" "${OBS_RECORDINGS_PROFILES_DIR}/${name}.json"
            echo "[SUCCESS] Added recording profile: ${name}"
            ;;
        streaming)
            cp "$file" "${OBS_STREAMING_PROFILES_DIR}/${name}.json"
            echo "[SUCCESS] Added streaming profile: ${name}"
            ;;
        *)
            echo "[ERROR] Unknown content type: $content_type"
            return 1
            ;;
    esac
}

#######################################
# List OBS Studio content
# Args:
#   --type: Content type (all|scene|recording|streaming)
#   --format: Output format (text|json)
#######################################
obs_content_list() {
    local content_type="all"
    local format="text"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                content_type="$2"
                shift 2
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$format" == "json" ]]; then
        echo "{"
        
        if [[ "$content_type" == "all" || "$content_type" == "scene" ]]; then
            echo '  "scenes": ['
            local first=true
            for scene in "${OBS_SCENES_DIR}"/*.json; do
                if [[ -f "$scene" ]]; then
                    if [[ "$first" != "true" ]]; then echo ","; fi
                    echo -n "    \"$(basename "$scene" .json)\""
                    first=false
                fi
            done
            echo ""
            echo "  ],"
        fi
        
        if [[ "$content_type" == "all" || "$content_type" == "recording" ]]; then
            echo '  "recording_profiles": ['
            local first=true
            for profile in "${OBS_RECORDINGS_PROFILES_DIR}"/*.json; do
                if [[ -f "$profile" ]]; then
                    if [[ "$first" != "true" ]]; then echo ","; fi
                    echo -n "    \"$(basename "$profile" .json)\""
                    first=false
                fi
            done
            echo ""
            echo "  ],"
        fi
        
        if [[ "$content_type" == "all" || "$content_type" == "streaming" ]]; then
            echo '  "streaming_profiles": ['
            local first=true
            for profile in "${OBS_STREAMING_PROFILES_DIR}"/*.json; do
                if [[ -f "$profile" ]]; then
                    if [[ "$first" != "true" ]]; then echo ","; fi
                    echo -n "    \"$(basename "$profile" .json)\""
                    first=false
                fi
            done
            echo ""
            echo "  ]"
        fi
        
        echo "}"
    else
        # Text format
        if [[ "$content_type" == "all" || "$content_type" == "scene" ]]; then
            echo "[INFO] Scenes:"
            for scene in "${OBS_SCENES_DIR}"/*.json; do
                if [[ -f "$scene" ]]; then
                    echo "  - $(basename "$scene" .json)"
                fi
            done
        fi
        
        if [[ "$content_type" == "all" || "$content_type" == "recording" ]]; then
            echo "[INFO] Recording Profiles:"
            for profile in "${OBS_RECORDINGS_PROFILES_DIR}"/*.json; do
                if [[ -f "$profile" ]]; then
                    echo "  - $(basename "$profile" .json)"
                fi
            done
        fi
        
        if [[ "$content_type" == "all" || "$content_type" == "streaming" ]]; then
            echo "[INFO] Streaming Profiles:"
            for profile in "${OBS_STREAMING_PROFILES_DIR}"/*.json; do
                if [[ -f "$profile" ]]; then
                    echo "  - $(basename "$profile" .json)"
                fi
            done
        fi
    fi
}

#######################################
# Get OBS Studio content
# Args:
#   --name: Content name
#   --type: Content type (scene|recording|streaming)
#######################################
obs_content_get() {
    local name=""
    local content_type="scene"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                content_type="$2"
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
    
    case "$content_type" in
        scene)
            local file="${OBS_SCENES_DIR}/${name}.json"
            ;;
        recording)
            local file="${OBS_RECORDINGS_PROFILES_DIR}/${name}.json"
            ;;
        streaming)
            local file="${OBS_STREAMING_PROFILES_DIR}/${name}.json"
            ;;
        *)
            echo "[ERROR] Unknown content type: $content_type"
            return 1
            ;;
    esac
    
    if [[ -f "$file" ]]; then
        cat "$file"
    else
        echo "[ERROR] Content not found: ${name}"
        return 1
    fi
}

#######################################
# Remove OBS Studio content
# Args:
#   --name: Content name
#   --type: Content type (scene|recording|streaming)
#######################################
obs_content_remove() {
    local name=""
    local content_type="scene"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                content_type="$2"
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
    
    case "$content_type" in
        scene)
            local file="${OBS_SCENES_DIR}/${name}.json"
            ;;
        recording)
            local file="${OBS_RECORDINGS_PROFILES_DIR}/${name}.json"
            ;;
        streaming)
            local file="${OBS_STREAMING_PROFILES_DIR}/${name}.json"
            ;;
        *)
            echo "[ERROR] Unknown content type: $content_type"
            return 1
            ;;
    esac
    
    if [[ -f "$file" ]]; then
        rm "$file"
        echo "[SUCCESS] Removed ${content_type}: ${name}"
    else
        echo "[ERROR] Content not found: ${name}"
        return 1
    fi
}

#######################################
# Execute OBS Studio content
# Args:
#   --name: Content name
#   --type: Content type (scene|recording|streaming)
#######################################
obs_content_execute() {
    local name=""
    local content_type="scene"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                content_type="$2"
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
    
    # Check if OBS is running
    if ! obs_is_running; then
        echo "[ERROR] OBS Studio is not running"
        return 1
    fi
    
    case "$content_type" in
        scene)
            echo "[INFO] Activating scene: ${name}"
            # In mock mode, just validate the scene exists
            if [[ -f "${OBS_SCENES_DIR}/${name}.json" ]]; then
                echo "[SUCCESS] Scene activated: ${name}"
            else
                echo "[ERROR] Scene not found: ${name}"
                return 1
            fi
            ;;
        recording)
            echo "[INFO] Starting recording with profile: ${name}"
            if [[ -f "${OBS_RECORDINGS_PROFILES_DIR}/${name}.json" ]]; then
                # In mock mode, create a dummy recording file
                touch "${OBS_RECORDINGS_DIR}/recording_${name}_$(date +%Y%m%d_%H%M%S).mkv"
                echo "[SUCCESS] Recording started with profile: ${name}"
            else
                echo "[ERROR] Recording profile not found: ${name}"
                return 1
            fi
            ;;
        streaming)
            echo "[INFO] Starting stream with profile: ${name}"
            if [[ -f "${OBS_STREAMING_PROFILES_DIR}/${name}.json" ]]; then
                echo "[SUCCESS] Stream started with profile: ${name}"
            else
                echo "[ERROR] Streaming profile not found: ${name}"
                return 1
            fi
            ;;
        *)
            echo "[ERROR] Unknown content type: $content_type"
            return 1
            ;;
    esac
}

# Export functions
export -f obs_content_add
export -f obs_content_list
export -f obs_content_get
export -f obs_content_remove
export -f obs_content_execute