# OBS Studio API Documentation

## Overview
The OBS Studio resource provides programmatic control over streaming, recording, and scene management through a WebSocket-based API. This document details all available commands and their usage.

## Command Structure
All commands follow the pattern:
```bash
vrooli resource obs-studio [command] [subcommand] [options]
```

## Core Commands

### Lifecycle Management

#### Install
```bash
vrooli resource obs-studio manage install
```
Installs OBS Studio dependencies and configures the WebSocket server.

#### Start
```bash
vrooli resource obs-studio manage start [--wait]
```
Starts the OBS Studio service with WebSocket server.
- `--wait`: Wait for service to be healthy before returning

#### Stop
```bash
vrooli resource obs-studio manage stop
```
Gracefully stops the OBS Studio service.

#### Restart
```bash
vrooli resource obs-studio manage restart
```
Restarts the OBS Studio service.

#### Status
```bash
vrooli resource obs-studio status [--format json]
```
Shows current status and configuration.
- `--format json`: Output in JSON format for programmatic use

## Content Management

### Add Content
```bash
vrooli resource obs-studio content add --file <path>
```
Adds a scene or profile configuration from a JSON file.

**Example scene.json:**
```json
{
  "name": "Tutorial Scene",
  "sources": [
    {
      "name": "Desktop",
      "type": "monitor_capture",
      "settings": {
        "monitor": 0,
        "capture_cursor": true
      }
    },
    {
      "name": "Webcam",
      "type": "video_capture_device",
      "settings": {
        "device_id": "/dev/video0",
        "resolution": "1920x1080",
        "fps": 30
      }
    }
  ]
}
```

### List Content
```bash
vrooli resource obs-studio content list
```
Lists all available scenes and profiles.

### Get Content
```bash
vrooli resource obs-studio content get --name <scene-name>
```
Retrieves a specific scene configuration.

### Remove Content
```bash
vrooli resource obs-studio content remove --name <scene-name>
```
Removes a scene or profile.

### Execute Content
```bash
vrooli resource obs-studio content execute --name <scene-name>
```
Activates a scene configuration.

## Recording Management

### Start Recording
```bash
vrooli resource obs-studio recording start [--output <path>]
```
Starts recording with current scene.
- `--output`: Custom output file path

### Stop Recording
```bash
vrooli resource obs-studio recording stop
```
Stops current recording.

### Status
```bash
vrooli resource obs-studio recording status
```
Shows recording status and statistics.

### List Recordings
```bash
vrooli resource obs-studio recording list
```
Lists all recorded files.

## Streaming Control

### Start Streaming
```bash
vrooli resource obs-studio streaming start [--profile <name>]
```
Starts streaming with specified profile.
- `--profile`: Streaming profile name (default: twitch)

### Stop Streaming
```bash
vrooli resource obs-studio streaming stop
```
Stops current stream.

### Status
```bash
vrooli resource obs-studio streaming status
```
Shows streaming status and statistics.

### Configure Streaming
```bash
vrooli resource obs-studio streaming configure --server <url> --key <key>
```
Configures streaming settings.
- `--server`: RTMP server URL
- `--key`: Stream key

### Manage Profiles
```bash
# List profiles
vrooli resource obs-studio streaming profiles

# Add profile
vrooli resource obs-studio streaming profiles --add <name> --server <url> --key <key>

# Remove profile
vrooli resource obs-studio streaming profiles --remove <name>
```

### Test Stream
```bash
vrooli resource obs-studio streaming test [--duration <seconds>]
```
Tests streaming configuration.
- `--duration`: Test duration in seconds (default: 10)

## Source Management

### Add Source
```bash
vrooli resource obs-studio sources add --name <name> --type <type> [options]
```
Adds a new source to the current scene.

**Source Types:**
- `camera`: Video capture device
- `screen`: Display capture
- `window`: Window capture
- `image`: Image file
- `media`: Media file
- `browser`: Browser source
- `text`: Text overlay

**Examples:**
```bash
# Add webcam
vrooli resource obs-studio sources add --name webcam --type camera --device /dev/video0

# Add screen capture
vrooli resource obs-studio sources add --name desktop --type screen --monitor 0

# Add image overlay
vrooli resource obs-studio sources add --name logo --type image --file /path/to/logo.png
```

### Remove Source
```bash
vrooli resource obs-studio sources remove --name <name>
```
Removes a source from the current scene.

### List Sources
```bash
vrooli resource obs-studio sources list
```
Lists all sources in the current scene.

### Configure Source
```bash
vrooli resource obs-studio sources configure --name <name> --property <prop> --value <val>
```
Configures source properties.

**Example:**
```bash
# Set webcam resolution
vrooli resource obs-studio sources configure --name webcam --property resolution --value 1920x1080

# Set source position
vrooli resource obs-studio sources configure --name logo --property position --value "100,100"
```

### Device Discovery
```bash
# List available cameras
vrooli resource obs-studio sources cameras

# List audio devices
vrooli resource obs-studio sources audio
```

### Source Visibility
```bash
# Show source
vrooli resource obs-studio sources visibility --name <name> --show

# Hide source
vrooli resource obs-studio sources visibility --name <name> --hide
```

## Scene Management

### Switch Scene
```bash
vrooli resource obs-studio scene switch --name <scene-name>
```
Switches to a different scene.

### List Scenes
```bash
vrooli resource obs-studio scene list
```
Lists all available scenes.

### Configure Scene
```bash
vrooli resource obs-studio scene configure --name <name> --transition <type> --duration <ms>
```
Configures scene transition settings.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OBS_PORT` | WebSocket server port | 4455 |
| `OBS_PASSWORD` | WebSocket authentication | auto-generated |
| `OBS_CONFIG_DIR` | Configuration directory | ~/.vrooli/obs-studio |
| `OBS_RECORDINGS_DIR` | Recording output directory | ~/Videos/obs-recordings |
| `OBS_STREAM_KEY` | Default stream key | (none) |
| `OBS_STREAM_SERVER` | Default RTMP server | (none) |

## Error Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Connection failed |
| 3 | Authentication failed |
| 4 | Scene not found |
| 5 | Source not found |
| 6 | Recording already active |
| 7 | Streaming already active |
| 8 | Invalid configuration |
| 9 | Device not found |
| 10 | Permission denied |

## Integration Examples

### Python Integration
```python
import subprocess
import json

# Get status
result = subprocess.run(
    ["vrooli", "resource", "obs-studio", "status", "--format", "json"],
    capture_output=True,
    text=True
)
status = json.loads(result.stdout)

# Start recording
subprocess.run(["vrooli", "resource", "obs-studio", "recording", "start"])

# Add source
subprocess.run([
    "vrooli", "resource", "obs-studio", "sources", "add",
    "--name", "webcam",
    "--type", "camera",
    "--device", "/dev/video0"
])
```

### Node.js Integration
```javascript
const { exec } = require('child_process');
const util = require('util');
const execPromise = util.promisify(exec);

async function startStreaming(profile = 'twitch') {
    try {
        const { stdout } = await execPromise(
            `vrooli resource obs-studio streaming start --profile ${profile}`
        );
        console.log('Streaming started:', stdout);
    } catch (error) {
        console.error('Failed to start streaming:', error);
    }
}

// Start streaming
startStreaming('youtube');
```

### Bash Scripting
```bash
#!/bin/bash

# Setup recording environment
setup_recording() {
    local scene_name="$1"
    
    # Start OBS
    vrooli resource obs-studio manage start --wait
    
    # Load scene
    vrooli resource obs-studio content execute --name "$scene_name"
    
    # Start recording
    vrooli resource obs-studio recording start
    
    echo "Recording started with scene: $scene_name"
}

# Monitor streaming health
monitor_stream() {
    while true; do
        status=$(vrooli resource obs-studio streaming status --format json)
        if [[ $(echo "$status" | jq -r '.streaming') == "false" ]]; then
            echo "Stream offline! Attempting restart..."
            vrooli resource obs-studio streaming start
        fi
        sleep 30
    done
}
```

## WebSocket API (Advanced)

For direct WebSocket communication, connect to `ws://localhost:${OBS_PORT}` with the configured password.

### Authentication
```json
{
    "op": 1,
    "d": {
        "rpcVersion": 1,
        "authentication": "password_hash",
        "eventSubscriptions": 33
    }
}
```

### Request Format
```json
{
    "op": 6,
    "d": {
        "requestType": "GetSceneList",
        "requestId": "unique_id"
    }
}
```

### Response Format
```json
{
    "op": 7,
    "d": {
        "requestType": "GetSceneList",
        "requestId": "unique_id",
        "requestStatus": {
            "result": true,
            "code": 100
        },
        "responseData": {
            "scenes": [...]
        }
    }
}
```

## Troubleshooting

### Common Issues

**WebSocket Connection Failed**
```bash
# Check if service is running
vrooli resource obs-studio status

# Restart service
vrooli resource obs-studio manage restart

# Check logs
vrooli resource obs-studio logs
```

**Recording Not Starting**
```bash
# Check disk space
df -h ~/Videos/obs-recordings

# Check permissions
ls -la ~/Videos/obs-recordings

# Verify scene is active
vrooli resource obs-studio scene list
```

**Stream Key Not Working**
```bash
# Reconfigure streaming
vrooli resource obs-studio streaming configure \
    --server "rtmp://live.twitch.tv/live" \
    --key "YOUR_STREAM_KEY"

# Test configuration
vrooli resource obs-studio streaming test
```

## Performance Optimization

### Recommended Settings
```bash
# CPU-optimized encoding
vrooli resource obs-studio sources configure \
    --name desktop \
    --property encoder \
    --value x264

# GPU-accelerated encoding (if available)
vrooli resource obs-studio sources configure \
    --name desktop \
    --property encoder \
    --value nvenc

# Optimize for streaming
vrooli resource obs-studio streaming configure \
    --bitrate 6000 \
    --keyframe-interval 2 \
    --preset fast
```

### Resource Limits
- Maximum sources per scene: 50
- Maximum scenes: 100
- Maximum simultaneous recordings: 1
- Maximum simultaneous streams: 5 (platform dependent)

## Security Considerations

1. **Stream Keys**: Never log or expose stream keys
2. **WebSocket Password**: Use strong, unique passwords
3. **File Permissions**: Recordings directory should be 755
4. **Network Access**: WebSocket server binds to localhost only
5. **Content Validation**: All JSON configurations are validated

## Support

For issues or questions:
- Check logs: `vrooli resource obs-studio logs`
- Run diagnostics: `vrooli resource obs-studio test all`
- Review examples: `/resources/obs-studio/examples/`