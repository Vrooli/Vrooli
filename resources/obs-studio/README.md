# OBS Studio Resource

Professional streaming and recording software with programmatic control via obs-websocket plugin.

## Overview

OBS Studio enables automated content production through:
- Scene and source management
- Automated recording and streaming
- Real-time composition and effects
- Programmatic control via WebSocket API

## Features

### Core Capabilities
- **Scene Control**: Switch between scenes programmatically
- **Source Management**: Add/remove/modify video, audio, and media sources
- **Recording**: Start/stop recording with custom settings
- **Streaming**: Full streaming control with platform profiles
- **Effects**: Apply filters and transitions via API
- **Audio Mixing**: Control audio sources and levels

### New v2.0 Features
- **Streaming Control**: Start/stop/configure streaming with profiles
- **Source Management**: Complete control over cameras, screens, windows, images, media, browser sources, and text overlays
- **Profile Management**: Save and reuse streaming/recording configurations
- **Device Discovery**: List available cameras and audio devices
- **Source Properties**: Configure detailed source settings
- **Visibility Control**: Show/hide sources dynamically
- **Docker Deployment**: Run OBS in containers with specific version tags (no 'latest')
- **VNC/NoVNC Access**: Remote control via VNC or web browser
- **Advanced Audio Mixer**: Full control over audio levels, filters, monitoring, ducking, compression, and EQ
- **Transition Effects**: Complete transition management including fade, swipe, stinger, and custom transitions
- **Dependency Management**: Automatic installation of Python dependencies with graceful fallback
- **Enhanced Error Handling**: Improved resilience when dependencies are missing

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

# Streaming Control
# Start streaming with a profile
vrooli resource obs-studio streaming start --profile twitch-main

# Check streaming status
vrooli resource obs-studio streaming status

# Stop streaming
vrooli resource obs-studio streaming stop

# Configure streaming settings
vrooli resource obs-studio streaming configure --bitrate 3500 --fps 30

# List streaming profiles
vrooli resource obs-studio streaming profiles

# Source Management
# Add a camera source
vrooli resource obs-studio sources add --name webcam --type camera

# List all sources
vrooli resource obs-studio sources list

# Configure source properties
vrooli resource obs-studio sources configure --name webcam --property fps --value 60

# Set source visibility
vrooli resource obs-studio sources visibility --name webcam --visible true

# List available cameras
vrooli resource obs-studio sources cameras

# List audio devices
vrooli resource obs-studio sources audio

# Advanced Audio Mixer Control
# Show audio mixer status with all sources
vrooli resource obs-studio audio status

# Adjust volume for specific source
vrooli resource obs-studio audio volume Microphone --level 85

# Configure noise suppression
vrooli resource obs-studio audio noise Microphone enable

# Set up auto-ducking for background music
vrooli resource obs-studio audio ducking enable --threshold -30 --ratio 3:1

# Configure audio compression
vrooli resource obs-studio audio compressor Microphone --threshold -20 --ratio 4:1

# Transition Effects Management
# List available transitions
vrooli resource obs-studio transitions list

# Set transition type and duration
vrooli resource obs-studio transitions set fade --duration 500

# Configure stinger transition with video
vrooli resource obs-studio transitions stinger --video /path/to/stinger.mp4 --point 250

# Create custom transition
vrooli resource obs-studio transitions custom create --name "Epic Transition" --base fade
```

## Configuration

Default configuration at `~/.vrooli/obs-studio/config.json`:
- WebSocket port: 4455
- WebSocket password: Auto-generated
- Default scene: "Main"
- Recording path: `~/Videos/obs-recordings`

### Docker Deployment

Run OBS Studio in Docker with specific versions:
```bash
# Build and run with version 30.2.3 (no 'latest' tags)
vrooli resource obs-studio docker build
vrooli resource obs-studio docker run

# Access via VNC or web browser
# VNC: vnc://localhost:5900
# Web: http://localhost:6080
```

## Documentation

- [Scene Injection Guide](docs/INJECTION.md) - Programmatic scene and source injection
- [API Reference](docs/API.md) - Complete WebSocket API documentation
- [Scene Configuration Guide](docs/SCENE_CONFIGURATION.md) - Templates and best practices
- [Streaming Setup Guide](docs/STREAMING_SETUP.md) - Platform-specific configurations
- [Docker Deployment Guide](docs/DOCKER_DEPLOYMENT.md) - Container deployment with version management
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions

## Examples

See the [examples/](examples/) directory for:
- Basic scene configurations
- Multi-scene streaming setups  
- Recording configurations
- Source management patterns
- Streaming setup with profiles
- Complete source management workflows

### Key Examples:
- `streaming-setup.sh` - Complete streaming configuration with Twitch profile
- `source-management.sh` - Managing cameras, screens, and overlays
- `basic-recording.sh` - Simple recording workflow
- `streaming-workflow.sh` - Advanced streaming with scene switching