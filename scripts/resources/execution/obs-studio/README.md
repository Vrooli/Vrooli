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
vrooli resource obs-studio install

# Start OBS Studio
vrooli resource obs-studio start

# Check status
vrooli resource obs-studio status

# Control scenes
resource-obs-studio switch-scene "Main"

# Start recording
resource-obs-studio start-recording

# Stop recording
resource-obs-studio stop-recording
```

## Configuration

Default configuration at `~/.vrooli/obs-studio/config.json`:
- WebSocket port: 4455
- WebSocket password: Auto-generated
- Default scene: "Main"
- Recording path: `~/Videos/obs-recordings`

## API Reference

See [docs/api.md](docs/api.md) for complete WebSocket API documentation.

## Troubleshooting

See [docs/troubleshooting.md](docs/troubleshooting.md) for common issues and solutions.