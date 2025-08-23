# OBS Studio Injection Documentation

## Overview

The OBS Studio resource supports programmatic injection of scenes, sources, and configurations through its injection API. This allows scenarios to automatically configure OBS for streaming, recording, and content production workflows.

## Features

- **Scene Creation**: Automatically create and configure OBS scenes
- **Source Management**: Add video, audio, and image sources to scenes
- **Configuration Injection**: Apply recording and streaming settings
- **Batch Operations**: Process multiple scenes from configuration files

## Usage

### Basic Scene Injection

Inject a single scene configuration:

```bash
vrooli resource obs-studio inject /path/to/scene.json
```

### Multi-Scene Configuration

Inject multiple scenes and settings:

```bash
vrooli resource obs-studio inject /path/to/config.json
```

## Configuration Format

### Single Scene Format

```json
{
  "name": "Scene Name",
  "sources": [
    {
      "name": "Source Name",
      "type": "source_type",
      "settings": {
        // Source-specific settings
      }
    }
  ]
}
```

### Multi-Scene Configuration Format

```json
{
  "scenes": [
    {
      "name": "Scene 1",
      "file": "/path/to/scene1.json"
    },
    {
      "name": "Scene 2",
      "file": "/path/to/scene2.json"
    }
  ],
  "recordings": {
    "output_path": "/path/to/recordings",
    "format": "mp4",
    "quality": "high"
  },
  "streaming": {
    "service": "twitch",
    "server": "auto",
    "bitrate": 2500
  }
}
```

## Source Types

### Video Sources

- **monitor_capture**: Desktop/screen capture
- **window_capture**: Specific window capture
- **v4l2_input**: Webcam/video device (Linux)
- **av_capture_input**: Video capture (macOS)
- **dshow_input**: DirectShow video (Windows)

### Audio Sources

- **pulse_input_capture**: PulseAudio input
- **pulse_output_capture**: PulseAudio output
- **alsa_input_capture**: ALSA input (Linux)
- **wasapi_input_capture**: Windows audio input
- **wasapi_output_capture**: Windows audio output

### Media Sources

- **ffmpeg_source**: Media file playback
- **image_source**: Static image
- **slideshow**: Image slideshow

### Other Sources

- **text_gdiplus**: Text overlay (Windows)
- **text_ft2_source**: FreeType2 text (Linux/macOS)
- **color_source**: Solid color background
- **browser_source**: Web page/HTML

## Examples

### Stream Setup Example

```json
{
  "scenes": [
    {
      "name": "Main Stream",
      "file": "scenes/main.json"
    },
    {
      "name": "BRB Screen",
      "file": "scenes/brb.json"
    },
    {
      "name": "Ending Screen",
      "file": "scenes/ending.json"
    }
  ],
  "streaming": {
    "service": "twitch",
    "bitrate": 3500,
    "encoder": "x264"
  }
}
```

### Podcast Recording Example

```json
{
  "name": "Podcast Setup",
  "sources": [
    {
      "name": "Host Mic",
      "type": "pulse_input_capture",
      "settings": {
        "device_id": "alsa_input.usb-Blue_Microphones"
      }
    },
    {
      "name": "Guest Audio",
      "type": "pulse_output_capture",
      "settings": {
        "device_id": "default"
      }
    },
    {
      "name": "Webcam",
      "type": "v4l2_input",
      "settings": {
        "device": "/dev/video0",
        "resolution": "1920x1080",
        "fps": 30
      }
    }
  ]
}
```

### Screen Recording Example

```json
{
  "name": "Tutorial Recording",
  "sources": [
    {
      "name": "Screen",
      "type": "monitor_capture",
      "settings": {
        "monitor": 0,
        "capture_cursor": true
      }
    },
    {
      "name": "Microphone",
      "type": "pulse_input_capture",
      "settings": {
        "device_id": "default"
      }
    },
    {
      "name": "Facecam",
      "type": "v4l2_input",
      "settings": {
        "device": "/dev/video0",
        "resolution": "320x240"
      }
    }
  ]
}
```

## Integration with Scenarios

Scenarios can use the OBS injection API to:

1. **Automated Streaming**: Configure and start streams programmatically
2. **Content Production**: Set up recording environments for tutorials, demos
3. **Event Broadcasting**: Switch between scenes during live events
4. **Podcast Production**: Configure multi-source audio/video setups
5. **Screen Recording**: Capture application demos and walkthroughs

## Best Practices

1. **Validate Configurations**: Always validate JSON before injection
2. **Check Resource Status**: Ensure OBS is running before injection
3. **Handle Existing Scenes**: Check if scenes exist to avoid duplicates
4. **Source Compatibility**: Verify source types work on your platform
5. **Path Management**: Use absolute paths for media files
6. **Error Handling**: Implement proper error handling for failed injections

## Troubleshooting

### Common Issues

1. **WebSocket Connection Failed**
   - Ensure OBS is running: `vrooli resource obs-studio status`
   - Check WebSocket port: `vrooli resource obs-studio config`

2. **Scene Creation Failed**
   - Verify JSON syntax is correct
   - Check scene name doesn't contain invalid characters

3. **Source Not Added**
   - Confirm source type is supported on your platform
   - Check device paths and permissions

4. **Invalid Configuration**
   - Validate JSON with `jq`: `jq empty < config.json`
   - Ensure all file paths are absolute

## API Reference

### CLI Commands

```bash
# Inject a configuration
vrooli resource obs-studio inject <config-file>

# List current scenes
vrooli resource obs-studio list-scenes

# Switch to a scene
vrooli resource obs-studio switch-scene <scene-name>

# List sources
vrooli resource obs-studio list-sources
```

### Library Functions

```bash
# Source the injection library
source scripts/resources/execution/obs-studio/lib/inject.sh

# Check health
obs::check_health

# Inject configuration
obs::inject "/path/to/config.json"

# Inject single scene
obs::inject_scene "/path/to/scene.json" "Scene Name"

# Process multiple scenes
obs::process_scenes "$config_object"
```

## Security Considerations

- Scene configurations may contain paths to local files
- Streaming configurations may include API keys (store in Vault)
- WebSocket is configured without authentication by default
- Consider network isolation for production use

## Future Enhancements

- [ ] Support for scene transitions
- [ ] Audio mixing configurations
- [ ] Filter and effect injection
- [ ] Scene collection management
- [ ] Profile switching support
- [ ] Automated scene scheduling