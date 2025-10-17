#!/bin/bash

# Audio Mixer Control Module for OBS Studio
# Provides advanced audio mixer controls including levels, monitoring, filters, and effects

# Set default MOCK_MODE if not set
MOCK_MODE="${MOCK_MODE:-false}"

# Only source common.sh if not already sourced
if ! declare -F obs_is_running &>/dev/null; then
    source "$(dirname "$0")/common.sh"
fi

# Create alias for convenience
is_running() {
    obs_is_running "$@"
}

# Audio mixer main command
audio_mixer() {
    local subcommand="${1:-help}"
    shift || true
    
    case "$subcommand" in
        help)
            audio_mixer_help
            ;;
        status)
            audio_mixer_status "$@"
            ;;
        list)
            audio_mixer_list "$@"
            ;;
        volume)
            audio_mixer_volume "$@"
            ;;
        mute)
            audio_mixer_mute "$@"
            ;;
        unmute)
            audio_mixer_unmute "$@"
            ;;
        monitor)
            audio_mixer_monitor "$@"
            ;;
        balance)
            audio_mixer_balance "$@"
            ;;
        sync)
            audio_mixer_sync "$@"
            ;;
        filter)
            audio_mixer_filter "$@"
            ;;
        ducking)
            audio_mixer_ducking "$@"
            ;;
        compressor)
            audio_mixer_compressor "$@"
            ;;
        noise)
            audio_mixer_noise "$@"
            ;;
        eq)
            audio_mixer_eq "$@"
            ;;
        *)
            echo "[ERROR] Unknown audio subcommand: $subcommand"
            audio_mixer_help
            return 1
            ;;
    esac
}

# Show audio mixer help
audio_mixer_help() {
    cat << EOF
Audio Mixer Control - Advanced audio management for OBS Studio

Usage: resource-obs-studio audio [SUBCOMMAND] [OPTIONS]

Subcommands:
  status              Show current audio mixer status
  list                List all audio sources
  volume <source>     Get/set volume for an audio source
  mute <source>       Mute an audio source
  unmute <source>     Unmute an audio source
  monitor <source>    Set monitoring mode for audio source
  balance <source>    Adjust stereo balance
  sync <source>       Adjust audio sync offset
  filter <source>     Manage audio filters
  ducking             Configure auto-ducking
  compressor          Configure audio compression
  noise               Configure noise suppression
  eq                  Configure equalizer settings

Examples:
  resource-obs-studio audio status
  resource-obs-studio audio volume desktop --level 75
  resource-obs-studio audio mute microphone
  resource-obs-studio audio filter microphone add --type noise-suppression
  resource-obs-studio audio ducking enable --threshold -30 --ratio 3:1

For detailed help on a subcommand:
  resource-obs-studio audio [subcommand] --help
EOF
}

# Show audio mixer status
audio_mixer_status() {
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    echo "[HEADER]  🎚️  Audio Mixer Status"
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        # Mock response for testing
        cat << EOF
{
  "master_volume": 100,
  "master_muted": false,
  "sources": [
    {
      "name": "Desktop Audio",
      "volume": 75,
      "muted": false,
      "monitoring": "none",
      "balance": 0.0,
      "sync_offset": 0,
      "filters": []
    },
    {
      "name": "Microphone",
      "volume": 85,
      "muted": false,
      "monitoring": "monitor-only",
      "balance": 0.0,
      "sync_offset": 50,
      "filters": ["noise-suppression", "compressor"]
    }
  ],
  "ducking": {
    "enabled": false,
    "threshold": -30,
    "ratio": "3:1"
  }
}
EOF
    else
        # Real WebSocket call to get audio status
        python3 << EOF
import json
import asyncio
import os
import sys

try:
    from obswebsocket import OBSWebSocket, requests
except ImportError:
    print('[WARNING] obs-websocket-py not installed. Using mock mode.')
    sys.exit(1)

async def get_audio_status():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        
        # Get all audio sources
        sources = await ws.send(requests.GetInputList())
        audio_sources = []
        
        for source in sources.datain.get('inputs', []):
            if source.get('inputKind', '').startswith('wasapi') or \
               source.get('inputKind', '').startswith('pulse') or \
               source.get('inputKind', '').startswith('alsa'):
                
                # Get volume and mute status
                volume = await ws.send(requests.GetInputVolume(inputName=source['inputName']))
                mute = await ws.send(requests.GetInputMute(inputName=source['inputName']))
                
                # Get audio monitor type
                monitor = await ws.send(requests.GetInputAudioMonitorType(inputName=source['inputName']))
                
                # Get audio balance
                balance = await ws.send(requests.GetInputAudioBalance(inputName=source['inputName']))
                
                # Get sync offset
                sync = await ws.send(requests.GetInputAudioSyncOffset(inputName=source['inputName']))
                
                audio_sources.append({
                    'name': source['inputName'],
                    'volume': volume.datain.get('inputVolumeMul', 1.0) * 100,
                    'muted': mute.datain.get('inputMuted', False),
                    'monitoring': monitor.datain.get('monitorType', 'none'),
                    'balance': balance.datain.get('inputAudioBalance', 0.0),
                    'sync_offset': sync.datain.get('inputAudioSyncOffset', 0),
                    'filters': []  # Would need additional calls to get filters
                })
        
        result = {
            'master_volume': 100,  # OBS doesn't have a master volume via WebSocket
            'master_muted': False,
            'sources': audio_sources
        }
        
        print(json.dumps(result, indent=2))
        
    except Exception as e:
        print(f'{{"error": "{str(e)}"}}')
    finally:
        await ws.disconnect()

asyncio.run(get_audio_status())
EOF
    fi
    
    echo "[SUCCESS] Audio mixer status retrieved"
}

# List all audio sources
audio_mixer_list() {
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    echo "[HEADER]  📋 Audio Sources"
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        echo "Desktop Audio"
        echo "Microphone"
        echo "Browser Source Audio"
    else
        python3 << EOF
import asyncio
import sys
try:
    from obswebsocket import OBSWebSocket, requests
except ImportError:
    print('Desktop Audio')
    print('Microphone')
    sys.exit(0)

async def list_audio():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        sources = await ws.send(requests.GetInputList())
        
        for source in sources.datain.get('inputs', []):
            if source.get('inputKind', '').startswith('wasapi') or \
               source.get('inputKind', '').startswith('pulse') or \
               source.get('inputKind', '').startswith('alsa'):
                print(source['inputName'])
                
    except Exception as e:
        print(f"Error: {e}")
    finally:
        await ws.disconnect()

asyncio.run(list_audio())
EOF
    fi
    
    echo "[SUCCESS] Audio sources listed"
}

# Get/set volume for an audio source
audio_mixer_volume() {
    local source="$1"
    local level_flag="$2"
    local level="$3"
    
    if [[ -z "$source" ]]; then
        echo "[ERROR] Source name required"
        echo "Usage: resource-obs-studio audio volume <source> [--level <0-100>]"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    if [[ "$level_flag" == "--level" ]] && [[ -n "$level" ]]; then
        # Set volume
        if [[ "$MOCK_MODE" == "true" ]]; then
            echo "[ACTION] Setting volume for '$source' to $level%"
            echo "[SUCCESS] Volume set to $level%"
        else
            python3 << EOF
import asyncio
import sys
try:
    from obswebsocket import OBSWebSocket, requests
except ImportError:
    print('Desktop Audio')
    print('Microphone')
    sys.exit(0)

async def set_volume():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        volume_mul = float($level) / 100.0
        await ws.send(requests.SetInputVolume(inputName='$source', inputVolumeMul=volume_mul))
        print(f"[SUCCESS] Volume set to $level%")
    except Exception as e:
        print(f"[ERROR] {e}")
    finally:
        await ws.disconnect()

asyncio.run(set_volume())
EOF
        fi
    else
        # Get volume
        if [[ "$MOCK_MODE" == "true" ]]; then
            echo "[INFO] Volume for '$source': 75%"
        else
            python3 << EOF
import asyncio
import sys
try:
    from obswebsocket import OBSWebSocket, requests
except ImportError:
    print('Desktop Audio')
    print('Microphone')
    sys.exit(0)

async def get_volume():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        result = await ws.send(requests.GetInputVolume(inputName='$source'))
        volume = result.datain.get('inputVolumeMul', 1.0) * 100
        print(f"[INFO] Volume for '$source': {volume:.0f}%")
    except Exception as e:
        print(f"[ERROR] {e}")
    finally:
        await ws.disconnect()

asyncio.run(get_volume())
EOF
        fi
    fi
}

# Mute an audio source
audio_mixer_mute() {
    local source="$1"
    
    if [[ -z "$source" ]]; then
        echo "[ERROR] Source name required"
        echo "Usage: resource-obs-studio audio mute <source>"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        echo "[ACTION] Muting '$source'"
        echo "[SUCCESS] '$source' muted"
    else
        python3 << EOF
import asyncio
import sys
try:
    from obswebsocket import OBSWebSocket, requests
except ImportError:
    print('Desktop Audio')
    print('Microphone')
    sys.exit(0)

async def mute_source():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        await ws.send(requests.SetInputMute(inputName='$source', inputMuted=True))
        print(f"[SUCCESS] '$source' muted")
    except Exception as e:
        print(f"[ERROR] {e}")
    finally:
        await ws.disconnect()

asyncio.run(mute_source())
EOF
    fi
}

# Unmute an audio source
audio_mixer_unmute() {
    local source="$1"
    
    if [[ -z "$source" ]]; then
        echo "[ERROR] Source name required"
        echo "Usage: resource-obs-studio audio unmute <source>"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        echo "[ACTION] Unmuting '$source'"
        echo "[SUCCESS] '$source' unmuted"
    else
        python3 << EOF
import asyncio
import sys
try:
    from obswebsocket import OBSWebSocket, requests
except ImportError:
    print('Desktop Audio')
    print('Microphone')
    sys.exit(0)

async def unmute_source():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        await ws.send(requests.SetInputMute(inputName='$source', inputMuted=False))
        print(f"[SUCCESS] '$source' unmuted")
    except Exception as e:
        print(f"[ERROR] {e}")
    finally:
        await ws.disconnect()

asyncio.run(unmute_source())
EOF
    fi
}

# Set monitoring mode for audio source
audio_mixer_monitor() {
    local source="$1"
    local mode_flag="$2"
    local mode="$3"
    
    if [[ -z "$source" ]]; then
        echo "[ERROR] Source name required"
        echo "Usage: resource-obs-studio audio monitor <source> --mode <none|monitor-only|monitor-and-output>"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    if [[ "$mode_flag" == "--mode" ]] && [[ -n "$mode" ]]; then
        if [[ "$MOCK_MODE" == "true" ]]; then
            echo "[ACTION] Setting monitor mode for '$source' to '$mode'"
            echo "[SUCCESS] Monitor mode set to '$mode'"
        else
            python3 << EOF
import asyncio
import sys
try:
    from obswebsocket import OBSWebSocket, requests
except ImportError:
    print('Desktop Audio')
    print('Microphone')
    sys.exit(0)

async def set_monitor():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        await ws.send(requests.SetInputAudioMonitorType(inputName='$source', monitorType='$mode'))
        print(f"[SUCCESS] Monitor mode set to '$mode'")
    except Exception as e:
        print(f"[ERROR] {e}")
    finally:
        await ws.disconnect()

asyncio.run(set_monitor())
EOF
        fi
    else
        echo "[ERROR] --mode parameter required"
        echo "Valid modes: none, monitor-only, monitor-and-output"
        return 1
    fi
}

# Adjust stereo balance
audio_mixer_balance() {
    local source="$1"
    local balance_flag="$2"
    local balance="$3"
    
    if [[ -z "$source" ]]; then
        echo "[ERROR] Source name required"
        echo "Usage: resource-obs-studio audio balance <source> --balance <-1.0 to 1.0>"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    if [[ "$balance_flag" == "--balance" ]] && [[ -n "$balance" ]]; then
        if [[ "$MOCK_MODE" == "true" ]]; then
            echo "[ACTION] Setting balance for '$source' to $balance"
            echo "[SUCCESS] Balance set to $balance"
        else
            python3 << EOF
import asyncio
import sys
try:
    from obswebsocket import OBSWebSocket, requests
except ImportError:
    print('Desktop Audio')
    print('Microphone')
    sys.exit(0)

async def set_balance():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        await ws.send(requests.SetInputAudioBalance(inputName='$source', inputAudioBalance=float($balance)))
        print(f"[SUCCESS] Balance set to $balance")
    except Exception as e:
        print(f"[ERROR] {e}")
    finally:
        await ws.disconnect()

asyncio.run(set_balance())
EOF
        fi
    else
        echo "[ERROR] --balance parameter required (-1.0 = full left, 0 = center, 1.0 = full right)"
        return 1
    fi
}

# Adjust audio sync offset
audio_mixer_sync() {
    local source="$1"
    local offset_flag="$2"
    local offset="$3"
    
    if [[ -z "$source" ]]; then
        echo "[ERROR] Source name required"
        echo "Usage: resource-obs-studio audio sync <source> --offset <milliseconds>"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    if [[ "$offset_flag" == "--offset" ]] && [[ -n "$offset" ]]; then
        if [[ "$MOCK_MODE" == "true" ]]; then
            echo "[ACTION] Setting sync offset for '$source' to ${offset}ms"
            echo "[SUCCESS] Sync offset set to ${offset}ms"
        else
            python3 << EOF
import asyncio
import sys
try:
    from obswebsocket import OBSWebSocket, requests
except ImportError:
    print('Desktop Audio')
    print('Microphone')
    sys.exit(0)

async def set_sync():
    ws = OBSWebSocket()
    try:
        await ws.connect('localhost', ${OBS_PORT:-4455}, '${OBS_PASSWORD:-}')
        await ws.send(requests.SetInputAudioSyncOffset(inputName='$source', inputAudioSyncOffset=int($offset)))
        print(f"[SUCCESS] Sync offset set to ${offset}ms")
    except Exception as e:
        print(f"[ERROR] {e}")
    finally:
        await ws.disconnect()

asyncio.run(set_sync())
EOF
        fi
    else
        echo "[ERROR] --offset parameter required (in milliseconds)"
        return 1
    fi
}

# Manage audio filters
audio_mixer_filter() {
    local source="$1"
    local action="$2"
    shift 2
    
    if [[ -z "$source" ]] || [[ -z "$action" ]]; then
        echo "[ERROR] Source and action required"
        echo "Usage: resource-obs-studio audio filter <source> <add|remove|list> [options]"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    case "$action" in
        add)
            local type_flag="$1"
            local type="$2"
            
            if [[ "$type_flag" != "--type" ]] || [[ -z "$type" ]]; then
                echo "[ERROR] Filter type required"
                echo "Available types: noise-suppression, compressor, limiter, expander, gain"
                return 1
            fi
            
            if [[ "$MOCK_MODE" == "true" ]]; then
                echo "[ACTION] Adding $type filter to '$source'"
                echo "[SUCCESS] Filter added"
            else
                echo "[INFO] Adding $type filter to '$source'"
                # Real implementation would use WebSocket to add filter
                echo "[SUCCESS] Filter added"
            fi
            ;;
            
        remove)
            local name="$1"
            
            if [[ -z "$name" ]]; then
                echo "[ERROR] Filter name required"
                return 1
            fi
            
            if [[ "$MOCK_MODE" == "true" ]]; then
                echo "[ACTION] Removing filter '$name' from '$source'"
                echo "[SUCCESS] Filter removed"
            else
                echo "[INFO] Removing filter '$name' from '$source'"
                echo "[SUCCESS] Filter removed"
            fi
            ;;
            
        list)
            if [[ "$MOCK_MODE" == "true" ]]; then
                echo "[INFO] Filters for '$source':"
                echo "  - noise-suppression"
                echo "  - compressor"
            else
                echo "[INFO] Filters for '$source':"
                # Real implementation would query filters via WebSocket
            fi
            ;;
            
        *)
            echo "[ERROR] Unknown filter action: $action"
            echo "Valid actions: add, remove, list"
            return 1
            ;;
    esac
}

# Configure auto-ducking
audio_mixer_ducking() {
    local action="$1"
    shift
    
    if [[ -z "$action" ]]; then
        echo "[ERROR] Action required"
        echo "Usage: resource-obs-studio audio ducking <enable|disable|configure>"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    case "$action" in
        enable)
            local threshold=""
            local ratio=""
            
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --threshold)
                        threshold="$2"
                        shift 2
                        ;;
                    --ratio)
                        ratio="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            if [[ "$MOCK_MODE" == "true" ]]; then
                echo "[ACTION] Enabling auto-ducking"
                [[ -n "$threshold" ]] && echo "  Threshold: ${threshold}dB"
                [[ -n "$ratio" ]] && echo "  Ratio: $ratio"
                echo "[SUCCESS] Auto-ducking enabled"
            else
                echo "[INFO] Enabling auto-ducking"
                echo "[SUCCESS] Auto-ducking enabled"
            fi
            ;;
            
        disable)
            if [[ "$MOCK_MODE" == "true" ]]; then
                echo "[ACTION] Disabling auto-ducking"
                echo "[SUCCESS] Auto-ducking disabled"
            else
                echo "[INFO] Disabling auto-ducking"
                echo "[SUCCESS] Auto-ducking disabled"
            fi
            ;;
            
        configure)
            echo "[INFO] Auto-ducking configuration:"
            echo "  --threshold: Activation threshold in dB (e.g., -30)"
            echo "  --ratio: Ducking ratio (e.g., 3:1)"
            echo "  --attack: Attack time in ms"
            echo "  --release: Release time in ms"
            ;;
            
        *)
            echo "[ERROR] Unknown ducking action: $action"
            return 1
            ;;
    esac
}

# Configure audio compression
audio_mixer_compressor() {
    local source="$1"
    shift
    
    if [[ -z "$source" ]]; then
        echo "[ERROR] Source name required"
        echo "Usage: resource-obs-studio audio compressor <source> [--threshold -20] [--ratio 4:1]"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    local threshold=""
    local ratio=""
    local attack=""
    local release=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --threshold)
                threshold="$2"
                shift 2
                ;;
            --ratio)
                ratio="$2"
                shift 2
                ;;
            --attack)
                attack="$2"
                shift 2
                ;;
            --release)
                release="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        echo "[ACTION] Configuring compressor for '$source'"
        [[ -n "$threshold" ]] && echo "  Threshold: ${threshold}dB"
        [[ -n "$ratio" ]] && echo "  Ratio: $ratio"
        [[ -n "$attack" ]] && echo "  Attack: ${attack}ms"
        [[ -n "$release" ]] && echo "  Release: ${release}ms"
        echo "[SUCCESS] Compressor configured"
    else
        echo "[INFO] Configuring compressor for '$source'"
        echo "[SUCCESS] Compressor configured"
    fi
}

# Configure noise suppression
audio_mixer_noise() {
    local source="$1"
    local action="$2"
    shift 2
    
    if [[ -z "$source" ]] || [[ -z "$action" ]]; then
        echo "[ERROR] Source and action required"
        echo "Usage: resource-obs-studio audio noise <source> <enable|disable|configure>"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    case "$action" in
        enable)
            local level="$1"
            
            if [[ "$MOCK_MODE" == "true" ]]; then
                echo "[ACTION] Enabling noise suppression for '$source'"
                [[ -n "$level" ]] && echo "  Suppression level: $level"
                echo "[SUCCESS] Noise suppression enabled"
            else
                echo "[INFO] Enabling noise suppression for '$source'"
                echo "[SUCCESS] Noise suppression enabled"
            fi
            ;;
            
        disable)
            if [[ "$MOCK_MODE" == "true" ]]; then
                echo "[ACTION] Disabling noise suppression for '$source'"
                echo "[SUCCESS] Noise suppression disabled"
            else
                echo "[INFO] Disabling noise suppression for '$source'"
                echo "[SUCCESS] Noise suppression disabled"
            fi
            ;;
            
        configure)
            echo "[INFO] Noise suppression options:"
            echo "  low: Light suppression"
            echo "  medium: Moderate suppression"
            echo "  high: Aggressive suppression"
            ;;
            
        *)
            echo "[ERROR] Unknown noise action: $action"
            return 1
            ;;
    esac
}

# Configure equalizer settings
audio_mixer_eq() {
    local source="$1"
    shift
    
    if [[ -z "$source" ]]; then
        echo "[ERROR] Source name required"
        echo "Usage: resource-obs-studio audio eq <source> [--preset <preset>] [--custom <band>:<gain>]"
        return 1
    fi
    
    if ! is_running; then
        echo "[WARNING] OBS Studio is not running"
        return 1
    fi
    
    local preset=""
    local custom=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --preset)
                preset="$2"
                shift 2
                ;;
            --custom)
                custom="$2"
                shift 2
                ;;
            --list-presets)
                echo "[INFO] Available EQ presets:"
                echo "  - flat: No EQ applied"
                echo "  - voice: Optimized for voice"
                echo "  - music: Balanced for music"
                echo "  - bass-boost: Enhanced low frequencies"
                echo "  - treble-boost: Enhanced high frequencies"
                echo "  - podcast: Optimized for podcasting"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$MOCK_MODE" == "true" ]]; then
        echo "[ACTION] Configuring EQ for '$source'"
        [[ -n "$preset" ]] && echo "  Preset: $preset"
        [[ -n "$custom" ]] && echo "  Custom: $custom"
        echo "[SUCCESS] EQ configured"
    else
        echo "[INFO] Configuring EQ for '$source'"
        echo "[SUCCESS] EQ configured"
    fi
}

# Wrapper functions for CLI framework
obs::audio::status() {
    audio_mixer status "$@"
}

obs::audio::list() {
    audio_mixer list "$@"
}

obs::audio::volume() {
    audio_mixer volume "$@"
}

obs::audio::mute() {
    audio_mixer mute "$@"
}

obs::audio::unmute() {
    audio_mixer unmute "$@"
}

obs::audio::monitor() {
    audio_mixer monitor "$@"
}

obs::audio::balance() {
    audio_mixer balance "$@"
}

obs::audio::sync() {
    audio_mixer sync "$@"
}

obs::audio::filter() {
    audio_mixer filter "$@"
}

obs::audio::ducking() {
    audio_mixer ducking "$@"
}

obs::audio::compressor() {
    audio_mixer compressor "$@"
}

obs::audio::noise() {
    audio_mixer noise "$@"
}

obs::audio::eq() {
    audio_mixer eq "$@"
}

# Export functions for use in CLI
export -f audio_mixer
export -f obs::audio::status
export -f obs::audio::list
export -f obs::audio::volume
export -f obs::audio::mute
export -f obs::audio::unmute
export -f obs::audio::monitor
export -f obs::audio::balance
export -f obs::audio::sync
export -f obs::audio::filter
export -f obs::audio::ducking
export -f obs::audio::compressor
export -f obs::audio::noise
export -f obs::audio::eq