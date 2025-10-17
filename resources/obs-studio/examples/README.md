# OBS Studio Examples

This directory contains example scripts demonstrating common OBS Studio workflows using the Vrooli OBS Studio resource CLI.

## Prerequisites

Before running these examples, ensure:
1. OBS Studio resource is installed: `resource-obs-studio install`
2. OBS Studio is configured with appropriate scenes and sources via the GUI

## Available Examples

### 1. Basic Recording (`basic-recording.sh`)
Demonstrates how to:
- Start OBS Studio
- Begin recording
- Stop recording after a specified duration
- Check recording status

**Usage:**
```bash
./basic-recording.sh
```

### 2. Scene Management (`scene-management.sh`)
Shows how to:
- List available scenes
- Get the current active scene
- Switch between scenes
- List available sources

**Usage:**
```bash
./scene-management.sh
```

### 3. Streaming Workflow (`streaming-workflow.sh`)
Demonstrates:
- Starting a streaming session
- Managing stream state
- Stopping the stream
- Checking configuration

**Usage:**
```bash
./streaming-workflow.sh
```

### 4. Audio Control (`audio-control.sh`)
Shows how to:
- Mute/unmute audio sources
- Adjust volume levels
- Control multiple audio inputs

**Usage:**
```bash
./audio-control.sh
```

## Common Commands Reference

```bash
# Status and management
resource-obs-studio status          # Check OBS Studio status
resource-obs-studio start           # Start OBS Studio
resource-obs-studio stop            # Stop OBS Studio
resource-obs-studio restart         # Restart OBS Studio

# Recording
resource-obs-studio start-recording # Start recording
resource-obs-studio stop-recording  # Stop recording
resource-obs-studio pause-recording # Pause recording
resource-obs-studio resume-recording # Resume recording

# Streaming
resource-obs-studio start-streaming # Start streaming
resource-obs-studio stop-streaming  # Stop streaming

# Scenes
resource-obs-studio list-scenes     # List all scenes
resource-obs-studio current-scene   # Get current scene
resource-obs-studio switch-scene <name> # Switch to scene

# Audio
resource-obs-studio mute <source>   # Mute audio source
resource-obs-studio unmute <source> # Unmute audio source
resource-obs-studio set-volume <source> <0-100> # Set volume
```

## Configuration

OBS Studio configuration is stored in:
- Config directory: `~/.vrooli/obs-studio/config/`
- Scenes directory: `~/.vrooli/obs-studio/scenes/`
- Recordings directory: `~/.vrooli/obs-studio/recordings/`

## WebSocket API

The OBS Studio resource uses the obs-websocket plugin for control. Default settings:
- Port: 4455
- Authentication: Disabled (for local automation)
- Protocol: WebSocket v5

## Troubleshooting

1. **OBS Studio not responding**: Check if WebSocket plugin is enabled in OBS Studio settings
2. **Scenes not found**: Create scenes using the OBS Studio GUI first
3. **Recording fails**: Ensure output directory exists and has write permissions
4. **Stream fails**: Configure stream settings (platform, key) in OBS Studio GUI

## Integration with Scenarios

These examples can be integrated into Vrooli scenarios for:
- Automated content creation
- Scheduled streaming sessions
- Multi-camera production workflows
- Tutorial and demo recording
- Live event broadcasting