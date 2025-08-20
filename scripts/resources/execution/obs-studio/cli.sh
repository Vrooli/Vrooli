#!/bin/bash

# OBS Studio CLI Interface
# Direct wrapper for OBS Studio resource management

# Resolve symlinks to get the actual script directory
OBS_CLI_SCRIPT="${BASH_SOURCE[0]}"
while [[ -L "$OBS_CLI_SCRIPT" ]]; do
    OBS_CLI_DIR="$(cd "$(dirname "$OBS_CLI_SCRIPT")" && pwd)"
    OBS_CLI_SCRIPT="$(readlink "$OBS_CLI_SCRIPT")"
    [[ "$OBS_CLI_SCRIPT" != /* ]] && OBS_CLI_SCRIPT="$OBS_CLI_DIR/$OBS_CLI_SCRIPT"
done
OBS_CLI_DIR="$(cd "$(dirname "$OBS_CLI_SCRIPT")" && pwd)"
OBS_LIB_DIR="$OBS_CLI_DIR/lib"

# Source core functions
source "$OBS_LIB_DIR/core.sh" || exit 1

# Display help
show_help() {
    cat <<EOF
OBS Studio Resource CLI

Usage: resource-obs-studio <command> [options]

Commands:
    install                 Install OBS Studio with WebSocket plugin
    start                   Start OBS Studio
    stop                    Stop OBS Studio
    status                  Show OBS Studio status
    restart                 Restart OBS Studio
    
Recording Commands:
    start-recording         Start recording
    stop-recording          Stop recording
    pause-recording         Pause recording
    resume-recording        Resume recording
    
Streaming Commands:
    start-streaming         Start streaming
    stop-streaming          Stop streaming
    
Scene Commands:
    list-scenes             List all scenes
    switch-scene <name>     Switch to a specific scene
    current-scene           Get current scene name
    
Source Commands:
    list-sources            List all sources
    toggle-source <name>    Toggle source visibility
    
Audio Commands:
    mute <source>           Mute audio source
    unmute <source>         Unmute audio source
    set-volume <src> <vol>  Set audio volume (0-100)
    
Configuration Commands:
    config                  Show configuration
    set-port <port>         Set WebSocket port
    reset-password          Generate new WebSocket password
    
Help:
    help                    Show this help message

Environment Variables:
    OBS_PORT               WebSocket port (default: 4455)
    OBS_INSTALL_METHOD     Installation method: native or docker (default: docker)

Examples:
    resource-obs-studio install
    resource-obs-studio start
    resource-obs-studio start-recording
    resource-obs-studio switch-scene "Main"
    resource-obs-studio stop-recording
    resource-obs-studio status
EOF
}

# WebSocket API helper function (delegate to core)
obs_websocket_request() {
    obs::websocket_command "$@"
}

# Recording commands
start_recording() {
    obs::start_recording
}

stop_recording() {
    obs::stop_recording
}

pause_recording() {
    echo "[INFO] Pausing recording..."
    obs_websocket_request "PauseRecord"
    echo "[SUCCESS] Recording paused"
}

resume_recording() {
    echo "[INFO] Resuming recording..."
    obs_websocket_request "ResumeRecord"
    echo "[SUCCESS] Recording resumed"
}

# Streaming commands
start_streaming() {
    echo "[INFO] Starting streaming..."
    obs_websocket_request "StartStream"
    echo "[SUCCESS] Streaming started"
}

stop_streaming() {
    echo "[INFO] Stopping streaming..."
    obs_websocket_request "StopStream"
    echo "[SUCCESS] Streaming stopped"
}

# Scene commands
list_scenes() {
    echo "[INFO] Listing scenes..."
    obs_websocket_request "GetSceneList"
}

switch_scene() {
    local scene_name="$1"
    echo "[INFO] Switching to scene: ${scene_name}"
    obs_websocket_request "SetCurrentProgramScene" '"sceneName":"'${scene_name}'"'
    echo "[SUCCESS] Switched to scene: ${scene_name}"
}

current_scene() {
    echo "[INFO] Getting current scene..."
    obs_websocket_request "GetCurrentProgramScene"
}

# Source commands
list_sources() {
    echo "[INFO] Listing sources..."
    obs_websocket_request "GetInputList"
}

toggle_source() {
    local source_name="$1"
    echo "[INFO] Toggling source: ${source_name}"
    obs_websocket_request "ToggleInputMute" '"inputName":"'${source_name}'"'
    echo "[SUCCESS] Toggled source: ${source_name}"
}

# Audio commands
mute_audio() {
    local source="$1"
    echo "[INFO] Muting audio source: ${source}"
    obs_websocket_request "SetInputMute" '"inputName":"'${source}'","inputMuted":true'
    echo "[SUCCESS] Muted: ${source}"
}

unmute_audio() {
    local source="$1"
    echo "[INFO] Unmuting audio source: ${source}"
    obs_websocket_request "SetInputMute" '"inputName":"'${source}'","inputMuted":false'
    echo "[SUCCESS] Unmuted: ${source}"
}

set_volume() {
    local source="$1"
    local volume="$2"
    echo "[INFO] Setting volume for ${source} to ${volume}%"
    # Convert percentage to dB (0-100% -> -inf to 0dB, roughly)
    local volume_db=$(echo "scale=2; 20 * l(${volume}/100) / l(10)" | bc -l 2>/dev/null || echo "0")
    obs_websocket_request "SetInputVolume" '"inputName":"'${source}'","inputVolumeDb":'${volume_db}
    echo "[SUCCESS] Volume set"
}

# Configuration commands
show_config() {
    echo "[INFO] OBS Studio Configuration:"
    echo "  Config Directory: ${OBS_CONFIG_DIR}"
    echo "  WebSocket Port: ${OBS_PORT}"
    echo "  Recording Path: ${OBS_RECORDINGS_DIR}"
    
    if [[ -f "${OBS_CONFIG_FILE}" ]]; then
        echo ""
        echo "Configuration File:"
        cat "${OBS_CONFIG_FILE}"
    fi
}

set_port() {
    local new_port="$1"
    echo "[INFO] Setting WebSocket port to: ${new_port}"
    
    # Update configuration
    if [[ -f "${OBS_CONFIG_FILE}" ]]; then
        # Update JSON config (using jq if available)
        if command -v jq >/dev/null 2>&1; then
            jq ".websocket.port = ${new_port}" "${OBS_CONFIG_FILE}" > "${OBS_CONFIG_FILE}.tmp"
            mv "${OBS_CONFIG_FILE}.tmp" "${OBS_CONFIG_FILE}"
        else
            sed -i "s/\"port\": [0-9]*/\"port\": ${new_port}/" "${OBS_CONFIG_FILE}"
        fi
    fi
    
    echo "[SUCCESS] Port updated. Restart OBS Studio for changes to take effect"
}

reset_password() {
    echo "[INFO] Generating new WebSocket password..."
    rm -f "${OBS_PASSWORD_FILE}"
    local new_password=$(obs_generate_password)
    echo "[SUCCESS] New password generated and saved"
    echo "[INFO] Password file: ${OBS_PASSWORD_FILE}"
}

# Main command router
main() {
    local command="${1:-help}"
    shift || true
    
    case "${command}" in
        install)
            obs::install "$@"
            ;;
        start)
            obs::start "$@"
            ;;
        stop)
            obs::stop "$@"
            ;;
        status)
            # Handle --format flag
            local format="plain"
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --format)
                        format="${2:-plain}"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            obs::get_status "$format"
            ;;
        restart)
            obs::stop
            sleep 2
            obs::start
            ;;
            
        # Recording commands
        start-recording)
            start_recording
            ;;
        stop-recording)
            stop_recording
            ;;
        pause-recording)
            pause_recording
            ;;
        resume-recording)
            resume_recording
            ;;
            
        # Streaming commands
        start-streaming)
            start_streaming
            ;;
        stop-streaming)
            stop_streaming
            ;;
            
        # Scene commands
        list-scenes)
            list_scenes
            ;;
        switch-scene)
            switch_scene "$@"
            ;;
        current-scene)
            current_scene
            ;;
            
        # Source commands
        list-sources)
            list_sources
            ;;
        toggle-source)
            toggle_source "$@"
            ;;
            
        # Audio commands
        mute)
            mute_audio "$@"
            ;;
        unmute)
            unmute_audio "$@"
            ;;
        set-volume)
            set_volume "$@"
            ;;
            
        # Configuration commands
        config)
            show_config
            ;;
        set-port)
            set_port "$@"
            ;;
        reset-password)
            reset_password
            ;;
            
        # Help
        help|--help|-h)
            show_help
            ;;
            
        *)
            echo "[ERROR] Unknown command: ${command}"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"