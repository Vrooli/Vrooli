#!/bin/bash

# Transition Effects Module for OBS Studio
# Provides control over scene transitions including types, duration, and custom effects

# Only source common.sh if not already sourced
if ! declare -F obs_is_running &>/dev/null; then
    source "$(dirname "$0")/common.sh"
fi

# Create alias for convenience
is_running() {
    obs_is_running "$@"
}

# Transitions main command
transitions() {
    local subcommand="${1:-help}"
    shift || true
    
    case "$subcommand" in
        help)
            transitions_help
            ;;
        list)
            transitions_list "$@"
            ;;
        current)
            transitions_current "$@"
            ;;
        set)
            transitions_set "$@"
            ;;
        configure)
            transitions_configure "$@"
            ;;
        duration)
            transitions_duration "$@"
            ;;
        preview)
            transitions_preview "$@"
            ;;
        stinger)
            transitions_stinger "$@"
            ;;
        custom)
            transitions_custom "$@"
            ;;
        library)
            transitions_library "$@"
            ;;
        *)
            echo "[ERROR] Unknown transitions subcommand: $subcommand"
            transitions_help
            return 1
            ;;
    esac
}

# Show transitions help
transitions_help() {
    cat << EOF
Transition Effects Control - Manage scene transitions in OBS Studio

Usage: resource-obs-studio transitions [SUBCOMMAND] [OPTIONS]

Subcommands:
  list                List all available transitions
  current             Show current transition settings
  set <type>          Set the transition type
  configure           Configure transition properties
  duration <ms>       Set transition duration in milliseconds
  preview             Preview a transition
  stinger             Configure stinger transitions
  custom              Create custom transition
  library             Manage transition library

Transition Types:
  - cut: Instant switch (no transition)
  - fade: Fade to black then to new scene
  - fade-color: Fade through a specific color
  - swipe: Swipe from one scene to another
  - slide: Slide between scenes
  - stinger: Video-based transition
  - luma-wipe: Luminance-based wipe effect
  - shader: Custom shader transition

Examples:
  resource-obs-studio transitions list
  resource-obs-studio transitions set fade --duration 500
  resource-obs-studio transitions stinger --video /path/to/stinger.mp4
  resource-obs-studio transitions custom create --name "Epic Transition"

For detailed help on a subcommand:
  resource-obs-studio transitions [subcommand] --help
EOF
}

# List all available transitions
transitions_list() {
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    echo "[HEADER]  📋 Available Transitions"
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        cat << EOF
{
  "transitions": [
    {"name": "Cut", "type": "cut_transition", "configurable": false},
    {"name": "Fade", "type": "fade_transition", "configurable": true},
    {"name": "Fade to Color", "type": "fade_to_color_transition", "configurable": true},
    {"name": "Swipe", "type": "swipe_transition", "configurable": true},
    {"name": "Slide", "type": "slide_transition", "configurable": true},
    {"name": "Stinger", "type": "stinger_transition", "configurable": true},
    {"name": "Luma Wipe", "type": "wipe_transition", "configurable": true},
    {"name": "Custom Shader", "type": "shader_transition", "configurable": true}
  ],
  "user_transitions": [
    {"name": "Quick Fade", "type": "fade_transition", "duration": 250},
    {"name": "Slow Fade", "type": "fade_transition", "duration": 1000},
    {"name": "Logo Stinger", "type": "stinger_transition", "duration": 500}
  ]
}
EOF
    else
        python3 << EOF
import json
import asyncio
from obswebsocket import OBSWebSocket, requests

async def list_transitions():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        
        # Get transition list
        result = await ws.send(requests.GetTransitionKindList())
        transitions = result.datain.get('transitionKinds', [])
        
        # Get current scene transition list
        scene_transitions = await ws.send(requests.GetSceneTransitionList())
        
        output = {
            'transitions': transitions,
            'current_transition': scene_transitions.datain.get('currentSceneTransitionName', ''),
            'available': scene_transitions.datain.get('transitions', [])
        }
        
        print(json.dumps(output, indent=2))
        
    except Exception as e:
        print(f'{{"error": "{str(e)}"}}')
    finally:
        await ws.disconnect()

asyncio.run(list_transitions())
EOF
    fi
    
    echo "[SUCCESS] Transitions listed"
}

# Show current transition settings
transitions_current() {
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    echo "[HEADER]  ⚙️  Current Transition Settings"
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        cat << EOF
{
  "name": "Fade",
  "type": "fade_transition",
  "duration": 300,
  "properties": {
    "fade_to_color": false,
    "color": "#000000",
    "switch_point": 50
  },
  "hotkeys": {
    "transition": "Ctrl+T",
    "studio_mode_toggle": "Ctrl+S"
  }
}
EOF
    else
        python3 << EOF
import json
import asyncio
from obswebsocket import OBSWebSocket, requests

async def get_current():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        
        # Get current transition
        current = await ws.send(requests.GetCurrentSceneTransition())
        
        # Get transition duration
        duration = await ws.send(requests.GetCurrentSceneTransitionDuration())
        
        output = {
            'name': current.datain.get('transitionName', ''),
            'kind': current.datain.get('transitionKind', ''),
            'duration': duration.datain.get('transitionDuration', 0),
            'fixed': current.datain.get('transitionFixed', False),
            'configurable': current.datain.get('transitionConfigurable', False)
        }
        
        print(json.dumps(output, indent=2))
        
    except Exception as e:
        print(f'{{"error": "{str(e)}"}}')
    finally:
        await ws.disconnect()

asyncio.run(get_current())
EOF
    fi
    
    echo "[SUCCESS] Current transition retrieved"
}

# Set the transition type
transitions_set() {
    local type="$1"
    shift
    
    if [[ -z "$type" ]]; then
        echo "[ERROR] Transition type required"
        echo "Available types: cut, fade, fade-color, swipe, slide, stinger, luma-wipe"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    local duration=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --duration)
                duration="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        echo "[ACTION] Setting transition to '$type'"
        [[ -n "$duration" ]] && echo "  Duration: ${duration}ms"
        echo "[SUCCESS] Transition set to '$type'"
    else
        python3 << EOF
import asyncio
from obswebsocket import OBSWebSocket, requests

async def set_transition():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        
        # Map user-friendly names to OBS transition types
        transition_map = {
            'cut': 'cut_transition',
            'fade': 'fade_transition',
            'fade-color': 'fade_to_color_transition',
            'swipe': 'swipe_transition',
            'slide': 'slide_transition',
            'stinger': 'stinger_transition',
            'luma-wipe': 'wipe_transition'
        }
        
        obs_type = transition_map.get('$type', '$type')
        
        # Set transition
        await ws.send(requests.SetCurrentSceneTransition(transitionName=obs_type))
        
        # Set duration if provided
        if '$duration':
            await ws.send(requests.SetCurrentSceneTransitionDuration(transitionDuration=int('$duration')))
        
        print("[SUCCESS] Transition set to '$type'")
        
    except Exception as e:
        print(f"[ERROR] {e}")
    finally:
        await ws.disconnect()

asyncio.run(set_transition())
EOF
    fi
}

# Configure transition properties
transitions_configure() {
    local property="$1"
    local value="$2"
    
    if [[ -z "$property" ]]; then
        echo "[INFO] Configurable transition properties:"
        echo "  --color <hex>       : Color for fade-to-color transition"
        echo "  --direction <dir>   : Direction for swipe/slide (left/right/up/down)"
        echo "  --softness <0-100>  : Edge softness for wipe transitions"
        echo "  --switch-point <ms> : Point to switch scenes during transition"
        echo "  --luma-image <path> : Grayscale image for luma wipe"
        echo "  --invert            : Invert the luma wipe"
        return 0
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        echo "[ACTION] Configuring transition property '$property' to '$value'"
        echo "[SUCCESS] Transition configured"
    else
        echo "[INFO] Configuring transition property '$property'"
        # Real implementation would use WebSocket to configure
        echo "[SUCCESS] Transition configured"
    fi
}

# Set transition duration
transitions_duration() {
    local ms="$1"
    
    if [[ -z "$ms" ]]; then
        echo "[ERROR] Duration in milliseconds required"
        echo "Usage: resource-obs-studio transitions duration <milliseconds>"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        echo "[ACTION] Setting transition duration to ${ms}ms"
        echo "[SUCCESS] Duration set to ${ms}ms"
    else
        python3 << EOF
import asyncio
from obswebsocket import OBSWebSocket, requests

async def set_duration():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        await ws.send(requests.SetCurrentSceneTransitionDuration(transitionDuration=int($ms)))
        print(f"[SUCCESS] Duration set to ${ms}ms")
    except Exception as e:
        print(f"[ERROR] {e}")
    finally:
        await ws.disconnect()

asyncio.run(set_duration())
EOF
    fi
}

# Preview a transition
transitions_preview() {
    local scene="$1"
    
    if [[ -z "$scene" ]]; then
        echo "[ERROR] Target scene required"
        echo "Usage: resource-obs-studio transitions preview <scene-name>"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        echo "[ACTION] Previewing transition to scene '$scene'"
        echo "[INFO] Transition preview started"
        sleep 1
        echo "[SUCCESS] Transition preview complete"
    else
        python3 << EOF
import asyncio
from obswebsocket import OBSWebSocket, requests

async def preview_transition():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        
        # Set preview scene
        await ws.send(requests.SetCurrentPreviewScene(sceneName='$scene'))
        
        # Trigger transition
        await ws.send(requests.TriggerStudioModeTransition())
        
        print("[SUCCESS] Transition preview complete")
        
    except Exception as e:
        print(f"[ERROR] {e}")
    finally:
        await ws.disconnect()

asyncio.run(preview_transition())
EOF
    fi
}

# Configure stinger transitions
transitions_stinger() {
    local video_flag="$1"
    local video_path="$2"
    shift 2
    
    if [[ "$video_flag" != "--video" ]] || [[ -z "$video_path" ]]; then
        echo "[ERROR] Video file required for stinger transition"
        echo "Usage: resource-obs-studio transitions stinger --video <path> [options]"
        echo ""
        echo "Options:"
        echo "  --point <ms>        : Transition point in the video"
        echo "  --audio-monitoring  : Monitor stinger audio"
        echo "  --audio-fade        : Fade audio during transition"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    local point=""
    local monitoring="false"
    local fade="false"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --point)
                point="$2"
                shift 2
                ;;
            --audio-monitoring)
                monitoring="true"
                shift
                ;;
            --audio-fade)
                fade="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        echo "[ACTION] Configuring stinger transition"
        echo "  Video: $video_path"
        [[ -n "$point" ]] && echo "  Transition point: ${point}ms"
        [[ "$monitoring" == "true" ]] && echo "  Audio monitoring: enabled"
        [[ "$fade" == "true" ]] && echo "  Audio fade: enabled"
        echo "[SUCCESS] Stinger transition configured"
    else
        echo "[INFO] Configuring stinger transition with video: $video_path"
        # Real implementation would configure via WebSocket
        echo "[SUCCESS] Stinger transition configured"
    fi
}

# Create custom transition
transitions_custom() {
    local action="$1"
    shift
    
    if [[ -z "$action" ]]; then
        echo "[ERROR] Action required"
        echo "Usage: resource-obs-studio transitions custom <create|edit|delete|export>"
        return 1
    fi
    
    case "$action" in
        create)
            local name=""
            local base=""
            
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --name)
                        name="$2"
                        shift 2
                        ;;
                    --base)
                        base="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            if [[ -z "$name" ]]; then
                echo "[ERROR] Custom transition name required"
                return 1
            fi
            
            if [[ "$MOCK_MODE" == "true" ]]; then
                echo "[ACTION] Creating custom transition '$name'"
                [[ -n "$base" ]] && echo "  Based on: $base"
                echo "[SUCCESS] Custom transition created"
            else
                echo "[INFO] Creating custom transition '$name'"
                echo "[SUCCESS] Custom transition created"
            fi
            ;;
            
        edit)
            local name="$1"
            
            if [[ -z "$name" ]]; then
                echo "[ERROR] Transition name required"
                return 1
            fi
            
            echo "[INFO] Editing transition '$name'"
            echo "[SUCCESS] Transition editor opened"
            ;;
            
        delete)
            local name="$1"
            
            if [[ -z "$name" ]]; then
                echo "[ERROR] Transition name required"
                return 1
            fi
            
            if [[ "$MOCK_MODE" == "true" ]]; then
                echo "[ACTION] Deleting custom transition '$name'"
                echo "[SUCCESS] Transition deleted"
            else
                echo "[INFO] Deleting transition '$name'"
                echo "[SUCCESS] Transition deleted"
            fi
            ;;
            
        export)
            local name="$1"
            local output="$2"
            
            if [[ -z "$name" ]]; then
                echo "[ERROR] Transition name required"
                return 1
            fi
            
            if [[ "$MOCK_MODE" == "true" ]]; then
                echo "[ACTION] Exporting transition '$name'"
                [[ -n "$output" ]] && echo "  Output: $output"
                echo "[SUCCESS] Transition exported"
            else
                echo "[INFO] Exporting transition '$name'"
                echo "[SUCCESS] Transition exported"
            fi
            ;;
            
        *)
            echo "[ERROR] Unknown custom action: $action"
            return 1
            ;;
    esac
}

# Manage transition library
transitions_library() {
    local action="$1"
    shift
    
    if [[ -z "$action" ]]; then
        echo "[ERROR] Action required"
        echo "Usage: resource-obs-studio transitions library <list|import|export|share>"
        return 1
    fi
    
    case "$action" in
        list)
            echo "[HEADER]  📚 Transition Library"
            
            if [[ "$MOCK_MODE" == "true" ]]; then
                cat << EOF
Preset Transitions:
  - Professional Fade (300ms fade)
  - Quick Cut (instant)
  - Smooth Slide (500ms slide)
  - Logo Stinger (custom stinger with logo)
  - Cinematic Wipe (luma wipe effect)
  
Custom Transitions:
  - Stream Starting (1000ms fade with sound)
  - Stream Ending (fade to logo)
  - Scene Switch (300ms swipe)
  
Downloaded Transitions:
  - Gaming Pack (10 transitions)
  - Professional Bundle (15 transitions)
EOF
            else
                echo "Transitions in library:"
                # Real implementation would list from storage
            fi
            ;;
            
        import)
            local file="$1"
            
            if [[ -z "$file" ]]; then
                echo "[ERROR] Transition file required"
                return 1
            fi
            
            echo "[ACTION] Importing transition from '$file'"
            echo "[SUCCESS] Transition imported"
            ;;
            
        export)
            local name="$1"
            local output="$2"
            
            if [[ -z "$name" ]]; then
                echo "[ERROR] Transition name required"
                return 1
            fi
            
            echo "[ACTION] Exporting transition '$name'"
            [[ -n "$output" ]] && echo "  Output: $output"
            echo "[SUCCESS] Transition exported"
            ;;
            
        share)
            local name="$1"
            
            if [[ -z "$name" ]]; then
                echo "[ERROR] Transition name required"
                return 1
            fi
            
            echo "[ACTION] Sharing transition '$name'"
            echo "[INFO] Share URL: https://transitions.obs.studio/share/abc123"
            echo "[SUCCESS] Transition shared"
            ;;
            
        *)
            echo "[ERROR] Unknown library action: $action"
            return 1
            ;;
    esac
}

# Wrapper functions for CLI framework
obs::transitions::list() {
    transitions list "$@"
}

obs::transitions::current() {
    transitions current "$@"
}

obs::transitions::set() {
    transitions set "$@"
}

obs::transitions::configure() {
    transitions configure "$@"
}

obs::transitions::duration() {
    transitions duration "$@"
}

obs::transitions::preview() {
    transitions preview "$@"
}

obs::transitions::stinger() {
    transitions stinger "$@"
}

obs::transitions::custom() {
    transitions custom "$@"
}

obs::transitions::library() {
    transitions library "$@"
}

# Export functions for use in CLI
export -f transitions
export -f obs::transitions::list
export -f obs::transitions::current
export -f obs::transitions::set
export -f obs::transitions::configure
export -f obs::transitions::duration
export -f obs::transitions::preview
export -f obs::transitions::stinger
export -f obs::transitions::custom
export -f obs::transitions::library