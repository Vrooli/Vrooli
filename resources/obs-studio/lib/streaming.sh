#!/bin/bash
# Streaming control for OBS Studio

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OBS_STREAMING_DIR="${APP_ROOT}/resources/obs-studio/lib"
source "${OBS_STREAMING_DIR}/common.sh"

# Streaming state management
OBS_STREAMING_STATE_FILE="${OBS_DATA_DIR}/.streaming_state"
OBS_STREAMING_CONFIG_FILE="${OBS_DATA_DIR}/streaming_config.json"

#######################################
# Start streaming with specified profile
# Args:
#   --profile: Streaming profile name
#   --platform: Platform (twitch|youtube|custom)
#   --key: Stream key (optional, uses profile if not provided)
#   --server: Server URL (optional, uses profile if not provided)
#######################################
obs::streaming::start() {
    local profile=""
    local platform=""
    local stream_key=""
    local server_url=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --profile)
                profile="$2"
                shift 2
                ;;
            --platform)
                platform="$2"
                shift 2
                ;;
            --key)
                stream_key="$2"
                shift 2
                ;;
            --server)
                server_url="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check if OBS is running
    if ! obs_is_running; then
        echo "[ERROR] OBS Studio is not running"
        return 1
    fi
    
    # Check if already streaming
    if obs::streaming::is_active; then
        echo "[WARNING] Streaming is already active"
        return 1
    fi
    
    # Load profile if specified
    if [[ -n "$profile" ]]; then
        local profile_file="${OBS_STREAMING_PROFILES_DIR}/${profile}.json"
        if [[ -f "$profile_file" ]]; then
            # Extract settings from profile
            if [[ -z "$platform" ]]; then
                platform=$(jq -r '.platform // "custom"' "$profile_file" 2>/dev/null || echo "custom")
            fi
            if [[ -z "$stream_key" ]]; then
                stream_key=$(jq -r '.stream_key // ""' "$profile_file" 2>/dev/null || echo "")
            fi
            if [[ -z "$server_url" ]]; then
                server_url=$(jq -r '.server_url // ""' "$profile_file" 2>/dev/null || echo "")
            fi
        else
            echo "[ERROR] Streaming profile not found: ${profile}"
            return 1
        fi
    fi
    
    # Set default server URLs based on platform
    if [[ -z "$server_url" ]]; then
        case "$platform" in
            twitch)
                server_url="rtmp://live.twitch.tv/live"
                ;;
            youtube)
                server_url="rtmp://a.rtmp.youtube.com/live2"
                ;;
            custom)
                if [[ -z "$server_url" ]]; then
                    echo "[ERROR] Server URL required for custom platform"
                    return 1
                fi
                ;;
            *)
                echo "[ERROR] Unknown platform: $platform"
                return 1
                ;;
        esac
    fi
    
    # Create streaming configuration
    cat > "$OBS_STREAMING_CONFIG_FILE" <<EOF
{
    "platform": "$platform",
    "server_url": "$server_url",
    "stream_key": "$stream_key",
    "started_at": "$(date -Iseconds)",
    "profile": "$profile"
}
EOF
    
    # In production, this would use WebSocket to start streaming
    # For now, we simulate it
    echo "active" > "$OBS_STREAMING_STATE_FILE"
    
    echo "[SUCCESS] Streaming started"
    echo "[INFO]    Platform: $platform"
    echo "[INFO]    Server: $server_url"
    [[ -n "$profile" ]] && echo "[INFO]    Profile: $profile"
    
    return 0
}

#######################################
# Stop streaming
#######################################
obs::streaming::stop() {
    if ! obs::streaming::is_active; then
        echo "[WARNING] Streaming is not active"
        return 1
    fi
    
    # In production, this would use WebSocket to stop streaming
    echo "stopped" > "$OBS_STREAMING_STATE_FILE"
    
    # Get streaming duration
    if [[ -f "$OBS_STREAMING_CONFIG_FILE" ]]; then
        local started_at=$(jq -r '.started_at' "$OBS_STREAMING_CONFIG_FILE" 2>/dev/null)
        if [[ -n "$started_at" ]]; then
            local duration=$(( $(date +%s) - $(date -d "$started_at" +%s) ))
            local hours=$((duration / 3600))
            local minutes=$(( (duration % 3600) / 60 ))
            local seconds=$((duration % 60))
            echo "[INFO] Stream duration: ${hours}h ${minutes}m ${seconds}s"
        fi
    fi
    
    echo "[SUCCESS] Streaming stopped"
    return 0
}

#######################################
# Get streaming status
#######################################
obs::streaming::status() {
    local format="${1:-text}"
    
    if ! obs_is_running; then
        if [[ "$format" == "json" ]]; then
            echo '{"status": "obs_not_running"}'
        else
            echo "[INFO] OBS Studio is not running"
        fi
        return 0
    fi
    
    if obs::streaming::is_active; then
        if [[ -f "$OBS_STREAMING_CONFIG_FILE" ]]; then
            local config=$(cat "$OBS_STREAMING_CONFIG_FILE")
            local platform=$(echo "$config" | jq -r '.platform')
            local server=$(echo "$config" | jq -r '.server_url')
            local started_at=$(echo "$config" | jq -r '.started_at')
            local profile=$(echo "$config" | jq -r '.profile // "none"')
            
            if [[ "$format" == "json" ]]; then
                echo "$config" | jq '. + {status: "streaming"}'
            else
                echo "[INFO] Streaming Status: Active"
                echo "[INFO]   Platform: $platform"
                echo "[INFO]   Server: $server"
                echo "[INFO]   Profile: $profile"
                echo "[INFO]   Started: $started_at"
                
                # Calculate duration
                if [[ -n "$started_at" ]]; then
                    local duration=$(( $(date +%s) - $(date -d "$started_at" +%s) ))
                    local hours=$((duration / 3600))
                    local minutes=$(( (duration % 3600) / 60 ))
                    local seconds=$((duration % 60))
                    echo "[INFO]   Duration: ${hours}h ${minutes}m ${seconds}s"
                fi
            fi
        else
            if [[ "$format" == "json" ]]; then
                echo '{"status": "streaming", "details": "unknown"}'
            else
                echo "[INFO] Streaming Status: Active (details unknown)"
            fi
        fi
    else
        if [[ "$format" == "json" ]]; then
            echo '{"status": "not_streaming"}'
        else
            echo "[INFO] Streaming Status: Not streaming"
        fi
    fi
    
    return 0
}

#######################################
# Check if streaming is active
#######################################
obs::streaming::is_active() {
    if [[ -f "$OBS_STREAMING_STATE_FILE" ]]; then
        local state=$(cat "$OBS_STREAMING_STATE_FILE" 2>/dev/null)
        [[ "$state" == "active" ]] && return 0
    fi
    return 1
}

#######################################
# Configure streaming settings
# Args:
#   --bitrate: Video bitrate in kbps
#   --audio-bitrate: Audio bitrate in kbps
#   --resolution: Output resolution (e.g., 1920x1080)
#   --fps: Frames per second
#   --encoder: Video encoder (x264|nvenc|qsv)
#######################################
obs::streaming::configure() {
    local bitrate=""
    local audio_bitrate=""
    local resolution=""
    local fps=""
    local encoder=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --bitrate)
                bitrate="$2"
                shift 2
                ;;
            --audio-bitrate)
                audio_bitrate="$2"
                shift 2
                ;;
            --resolution)
                resolution="$2"
                shift 2
                ;;
            --fps)
                fps="$2"
                shift 2
                ;;
            --encoder)
                encoder="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Load existing settings
    local settings_file="${OBS_CONFIG_DIR}/streaming_settings.json"
    local current_settings="{}"
    if [[ -f "$settings_file" ]]; then
        current_settings=$(cat "$settings_file")
    fi
    
    # Update settings
    local updated_settings=$(echo "$current_settings" | jq \
        --arg bitrate "${bitrate:-}" \
        --arg audio_bitrate "${audio_bitrate:-}" \
        --arg resolution "${resolution:-}" \
        --arg fps "${fps:-}" \
        --arg encoder "${encoder:-}" \
        '. + {
            video_bitrate: (if $bitrate != "" then ($bitrate | tonumber) else .video_bitrate end),
            audio_bitrate: (if $audio_bitrate != "" then ($audio_bitrate | tonumber) else .audio_bitrate end),
            resolution: (if $resolution != "" then $resolution else .resolution end),
            fps: (if $fps != "" then ($fps | tonumber) else .fps end),
            encoder: (if $encoder != "" then $encoder else .encoder end)
        }')
    
    # Save settings
    echo "$updated_settings" > "$settings_file"
    
    echo "[SUCCESS] Streaming settings updated"
    echo "$updated_settings" | jq .
    
    return 0
}

#######################################
# List available streaming profiles
#######################################
obs::streaming::profiles() {
    echo "[INFO] Available Streaming Profiles:"
    
    local profile_count=0
    for profile_file in "${OBS_STREAMING_PROFILES_DIR}"/*.json; do
        if [[ -f "$profile_file" ]]; then
            local profile_name=$(basename "$profile_file" .json)
            local platform=$(jq -r '.platform // "unknown"' "$profile_file" 2>/dev/null)
            local server=$(jq -r '.server_url // "not set"' "$profile_file" 2>/dev/null)
            
            echo "  - $profile_name"
            echo "      Platform: $platform"
            echo "      Server: $server"
            
            profile_count=$((profile_count + 1))
        fi
    done
    
    if [[ $profile_count -eq 0 ]]; then
        echo "  (No profiles configured)"
    fi
    
    return 0
}

#######################################
# Test streaming connectivity
# Args:
#   --profile: Profile to test
#   --server: Server URL to test
#######################################
obs::streaming::test() {
    local profile=""
    local server_url=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --profile)
                profile="$2"
                shift 2
                ;;
            --server)
                server_url="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Load profile if specified
    if [[ -n "$profile" ]]; then
        local profile_file="${OBS_STREAMING_PROFILES_DIR}/${profile}.json"
        if [[ -f "$profile_file" ]]; then
            server_url=$(jq -r '.server_url' "$profile_file" 2>/dev/null)
        else
            echo "[ERROR] Profile not found: $profile"
            return 1
        fi
    fi
    
    if [[ -z "$server_url" ]]; then
        echo "[ERROR] No server URL to test"
        return 1
    fi
    
    echo "[INFO] Testing connection to: $server_url"
    
    # Extract host and port from RTMP URL
    local host=$(echo "$server_url" | sed -E 's|rtmp://([^:/]+).*|\1|')
    local port=1935  # Default RTMP port
    
    # Test connectivity
    if timeout 5 bash -c "echo > /dev/tcp/$host/$port" 2>/dev/null; then
        echo "[SUCCESS] Connection test successful"
        return 0
    else
        echo "[ERROR] Connection test failed"
        return 1
    fi
}

# Export functions
export -f obs::streaming::start
export -f obs::streaming::stop
export -f obs::streaming::status
export -f obs::streaming::is_active
export -f obs::streaming::configure
export -f obs::streaming::profiles
export -f obs::streaming::test