# OBS Studio Resource

Professional streaming and recording software with programmatic control via obs-websocket plugin.

## Overview

OBS Studio enables automated content production through:
- Scene and source management
- Automated recording and streaming
- Real-time composition and effects
- Programmatic control via WebSocket API

## Features

- **Scene Control**: Switch between scenes programmatically
- **Source Management**: Add/remove/modify sources dynamically
- **Recording**: Start/stop recording with custom settings
- **Streaming**: Control streaming to multiple platforms
- **Effects**: Apply filters and transitions via API
- **Audio Mixing**: Control audio sources and levels
- **Configuration Injection**: Programmatically inject scenes and sources from JSON

## Use Cases

### Content Production
- Automated video recording for tutorials
- Multi-camera streaming setups
- Screen recording automation

### Live Streaming
- Scheduled streaming sessions
- Dynamic scene switching based on events
- Automated highlight creation

### Integration Scenarios
- Combine with Whisper for live transcription
- Use with ComfyUI for dynamic overlays
- Integrate with n8n for event-driven streaming

## Architecture

```
┌─────────────────┐
│   OBS Studio    │
│   (Main App)    │
└────────┬────────┘
         │
┌────────▼────────┐
│  obs-websocket  │
│    (Plugin)     │
└────────┬────────┘
         │
┌────────▼────────┐
│  WebSocket API  │
│   (Port 4455)   │
└────────┬────────┘
         │
┌────────▼────────┐
│  Vrooli CLI     │
│   Interface     │
└─────────────────┘
```

## Quick Start

```bash
# Install OBS Studio
vrooli resource obs-studio manage install

# Start OBS Studio
vrooli resource obs-studio manage start

# Check status
vrooli resource obs-studio status

# Content Management
# Add a scene configuration
vrooli resource obs-studio content add --file scene.json --type scene

# List all content
vrooli resource obs-studio content list

# Get specific content
vrooli resource obs-studio content get --name my-scene

# Activate a scene
vrooli resource obs-studio content execute --name my-scene

# Control recording
vrooli resource obs-studio content record start
vrooli resource obs-studio content record stop

# Remove content
vrooli resource obs-studio content remove --name my-scene
```

## Configuration

Default configuration at `~/.vrooli/obs-studio/config.json`:
- WebSocket port: 4455
- WebSocket password: Auto-generated
- Default scene: "Main"
- Recording path: `~/Videos/obs-recordings`

## Documentation

- [Scene Injection Guide](docs/INJECTION.md) - Programmatic scene and source injection
- [API Reference](docs/api.md) - Complete WebSocket API documentation
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions

## Examples

See the [examples/](examples/) directory for:
- Basic scene configurations
- Multi-scene streaming setups  
- Recording configurations
- Source management patterns