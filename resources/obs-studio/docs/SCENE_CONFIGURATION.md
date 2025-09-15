# OBS Studio Scene Configuration Guide

## Introduction
Scenes in OBS Studio are collections of sources (video, audio, images, text) arranged in a specific layout. This guide explains how to create, configure, and manage scenes for various use cases.

## Scene Concepts

### What is a Scene?
A scene is a complete composition containing:
- **Sources**: Individual media elements (camera, screen, images, etc.)
- **Layout**: Position and size of each source
- **Properties**: Settings like filters, transitions, and effects
- **Audio Mix**: Audio levels and routing

### Scene vs Profile
- **Scene**: Visual and audio composition (what you see)
- **Profile**: Encoding and output settings (how it's recorded/streamed)

## Creating Scenes

### Basic Scene Structure
```json
{
  "name": "Scene Name",
  "sources": [
    {
      "name": "Source Name",
      "type": "source_type",
      "settings": {},
      "position": {"x": 0, "y": 0},
      "scale": {"x": 1.0, "y": 1.0},
      "visible": true
    }
  ],
  "transitions": {
    "type": "fade",
    "duration": 300
  }
}
```

### Source Types Reference

#### Video Capture Device (Webcam)
```json
{
  "name": "Webcam",
  "type": "video_capture_device",
  "settings": {
    "device_id": "/dev/video0",
    "resolution": "1920x1080",
    "fps": 30,
    "video_format": "YUY2",
    "buffering": false
  }
}
```

#### Display Capture (Screen)
```json
{
  "name": "Desktop",
  "type": "monitor_capture",
  "settings": {
    "monitor": 0,
    "capture_cursor": true,
    "compatibility_mode": false
  }
}
```

#### Window Capture
```json
{
  "name": "Browser Window",
  "type": "window_capture",
  "settings": {
    "window": "Mozilla Firefox",
    "priority": 2,
    "capture_cursor": true
  }
}
```

#### Image Source
```json
{
  "name": "Logo",
  "type": "image_source",
  "settings": {
    "file": "/path/to/logo.png",
    "unload_when_not_showing": false
  },
  "position": {"x": 1700, "y": 50},
  "scale": {"x": 0.5, "y": 0.5}
}
```

#### Text Source
```json
{
  "name": "Title",
  "type": "text_gdiplus",
  "settings": {
    "text": "Live Stream",
    "font": {
      "face": "Arial",
      "size": 48,
      "style": "Bold"
    },
    "color": "#FFFFFF",
    "outline": true,
    "outline_color": "#000000",
    "outline_size": 2
  },
  "position": {"x": 100, "y": 100}
}
```

#### Browser Source
```json
{
  "name": "Alert Box",
  "type": "browser_source",
  "settings": {
    "url": "https://streamlabs.com/alert-box/v3/...",
    "width": 800,
    "height": 600,
    "fps": 30,
    "css": "body { background-color: transparent; }"
  }
}
```

#### Media Source
```json
{
  "name": "Intro Video",
  "type": "ffmpeg_source",
  "settings": {
    "local_file": "/path/to/intro.mp4",
    "looping": false,
    "restart_on_activate": true,
    "hw_decode": true
  }
}
```

## Common Scene Templates

### 1. Presentation Scene
Perfect for tutorials, webinars, and educational content.

```json
{
  "name": "Presentation",
  "sources": [
    {
      "name": "Desktop",
      "type": "monitor_capture",
      "settings": {
        "monitor": 0,
        "capture_cursor": true
      },
      "position": {"x": 0, "y": 0},
      "scale": {"x": 1.0, "y": 1.0}
    },
    {
      "name": "Presenter",
      "type": "video_capture_device",
      "settings": {
        "device_id": "/dev/video0",
        "resolution": "640x480",
        "fps": 30
      },
      "position": {"x": 1400, "y": 700},
      "scale": {"x": 0.3, "y": 0.3},
      "crop": {
        "top": 10,
        "bottom": 10,
        "left": 10,
        "right": 10
      }
    },
    {
      "name": "Logo",
      "type": "image_source",
      "settings": {
        "file": "/assets/logo.png"
      },
      "position": {"x": 50, "y": 50},
      "scale": {"x": 0.2, "y": 0.2}
    }
  ]
}
```

### 2. Gaming Scene
Optimized for game streaming with webcam overlay.

```json
{
  "name": "Gaming",
  "sources": [
    {
      "name": "Game Capture",
      "type": "game_capture",
      "settings": {
        "capture_mode": "auto",
        "window": "",
        "priority": 1,
        "capture_cursor": false,
        "allow_transparency": false,
        "force_scaling": false,
        "limit_framerate": false
      }
    },
    {
      "name": "Webcam",
      "type": "video_capture_device",
      "settings": {
        "device_id": "/dev/video0",
        "resolution": "1280x720",
        "fps": 60
      },
      "position": {"x": 1500, "y": 800},
      "scale": {"x": 0.25, "y": 0.25},
      "filters": [
        {
          "name": "Background Removal",
          "type": "chroma_key",
          "settings": {
            "key_color": "#00FF00",
            "similarity": 400,
            "smoothness": 80
          }
        }
      ]
    },
    {
      "name": "Chat Overlay",
      "type": "browser_source",
      "settings": {
        "url": "https://streamlabs.com/chat/...",
        "width": 400,
        "height": 600
      },
      "position": {"x": 20, "y": 400}
    }
  ]
}
```

### 3. Interview Scene
Two-person interview or podcast setup.

```json
{
  "name": "Interview",
  "sources": [
    {
      "name": "Background",
      "type": "image_source",
      "settings": {
        "file": "/assets/studio-background.jpg"
      },
      "scale": {"x": 1.0, "y": 1.0}
    },
    {
      "name": "Host",
      "type": "video_capture_device",
      "settings": {
        "device_id": "/dev/video0",
        "resolution": "1920x1080",
        "fps": 30
      },
      "position": {"x": 100, "y": 200},
      "scale": {"x": 0.4, "y": 0.4}
    },
    {
      "name": "Guest",
      "type": "video_capture_device",
      "settings": {
        "device_id": "/dev/video1",
        "resolution": "1920x1080",
        "fps": 30
      },
      "position": {"x": 1000, "y": 200},
      "scale": {"x": 0.4, "y": 0.4}
    },
    {
      "name": "Lower Third",
      "type": "image_source",
      "settings": {
        "file": "/assets/lower-third.png"
      },
      "position": {"x": 0, "y": 800}
    }
  ]
}
```

### 4. Starting Soon Scene
Pre-stream countdown scene.

```json
{
  "name": "Starting Soon",
  "sources": [
    {
      "name": "Background Video",
      "type": "ffmpeg_source",
      "settings": {
        "local_file": "/assets/animated-bg.mp4",
        "looping": true
      }
    },
    {
      "name": "Countdown Timer",
      "type": "text_gdiplus",
      "settings": {
        "text": "Starting in 5:00",
        "font": {
          "face": "Roboto",
          "size": 72,
          "style": "Bold"
        },
        "color": "#FFFFFF"
      },
      "position": {"x": 700, "y": 500}
    },
    {
      "name": "Background Music",
      "type": "ffmpeg_source",
      "settings": {
        "local_file": "/assets/intro-music.mp3",
        "looping": true
      },
      "audio_monitoring": "monitor_and_output"
    }
  ]
}
```

## Advanced Configuration

### Filters
Add visual and audio effects to sources.

```json
{
  "name": "Webcam",
  "type": "video_capture_device",
  "filters": [
    {
      "name": "Color Correction",
      "type": "color_filter",
      "settings": {
        "brightness": 0.1,
        "contrast": 0.05,
        "saturation": 0.1,
        "hue_shift": 0
      }
    },
    {
      "name": "Sharpen",
      "type": "sharpness_filter",
      "settings": {
        "sharpness": 0.2
      }
    },
    {
      "name": "Noise Suppression",
      "type": "noise_suppress_filter",
      "settings": {
        "suppress_level": -30
      }
    }
  ]
}
```

### Transitions
Configure scene transitions.

```json
{
  "transitions": {
    "type": "stinger",
    "settings": {
      "path": "/assets/transition.webm",
      "transition_point": 500,
      "audio_monitoring": "none",
      "audio_fade_style": "fade_out_fade_in"
    }
  }
}
```

### Audio Configuration
```json
{
  "audio": {
    "desktop_audio": {
      "enabled": true,
      "volume": 0.8,
      "sync_offset": 0,
      "monitoring": "monitor_off"
    },
    "mic_audio": {
      "enabled": true,
      "device": "default",
      "volume": 1.0,
      "sync_offset": 0,
      "filters": [
        {
          "type": "noise_gate",
          "settings": {
            "open_threshold": -30,
            "close_threshold": -35,
            "attack_time": 5,
            "hold_time": 10,
            "release_time": 50
          }
        }
      ]
    }
  }
}
```

## Working with Scenes

### Creating a Scene
```bash
# Create scene from JSON file
vrooli resource obs-studio content add --file presentation-scene.json

# Activate the scene
vrooli resource obs-studio content execute --name "Presentation"
```

### Modifying Scenes
```bash
# Add a source to current scene
vrooli resource obs-studio sources add \
  --name "Watermark" \
  --type image \
  --file /assets/watermark.png

# Configure source properties
vrooli resource obs-studio sources configure \
  --name "Watermark" \
  --property position \
  --value "1700,1000"

# Toggle source visibility
vrooli resource obs-studio sources visibility \
  --name "Watermark" \
  --hide
```

### Scene Management
```bash
# List all scenes
vrooli resource obs-studio scene list

# Switch to a different scene
vrooli resource obs-studio scene switch --name "Gaming"

# Export scene configuration
vrooli resource obs-studio content get --name "Gaming" > gaming-scene.json
```

## Best Practices

### 1. Resolution Consistency
- Keep all sources at the same base resolution
- Use canvas resolution of 1920x1080 for compatibility
- Scale sources proportionally

### 2. Performance Optimization
- Use hardware encoding when available
- Limit browser sources to essential elements
- Disable sources when not visible
- Use appropriate video formats (YUY2, NV12)

### 3. Audio Management
- Always use noise suppression on microphones
- Set appropriate noise gate thresholds
- Monitor audio levels during setup
- Use compressor for consistent volume

### 4. Scene Organization
- Name sources descriptively
- Group related sources
- Create template scenes for reuse
- Document custom settings

### 5. Backup and Version Control
```bash
# Export all scenes
for scene in $(vrooli resource obs-studio scene list); do
  vrooli resource obs-studio content get --name "$scene" > "backup/$scene.json"
done

# Version control scenes
git add scenes/*.json
git commit -m "Updated streaming scenes"
```

## Troubleshooting

### Source Not Showing
1. Check source visibility setting
2. Verify source is above others in layer order
3. Confirm source dimensions and position
4. Check if source is cropped or scaled to 0

### Audio Issues
1. Verify audio device is selected correctly
2. Check audio monitoring settings
3. Ensure filters aren't suppressing all audio
4. Confirm sync offset is appropriate

### Performance Problems
1. Lower source resolutions
2. Reduce FPS to 30
3. Disable unnecessary filters
4. Use hardware acceleration
5. Close other applications

### Scene Switching Lag
1. Reduce transition duration
2. Preload media sources
3. Use simpler transitions (cut/fade)
4. Optimize source count

## Example Workflow

### Complete Stream Setup
```bash
#!/bin/bash

# 1. Start OBS
vrooli resource obs-studio manage start --wait

# 2. Load presentation scene
vrooli resource obs-studio content add --file scenes/presentation.json

# 3. Configure webcam
vrooli resource obs-studio sources add \
  --name "Webcam" \
  --type camera \
  --device /dev/video0

# 4. Position webcam
vrooli resource obs-studio sources configure \
  --name "Webcam" \
  --property position \
  --value "1400,700"

# 5. Add background music
vrooli resource obs-studio sources add \
  --name "BGM" \
  --type media \
  --file /assets/background-music.mp3

# 6. Set audio levels
vrooli resource obs-studio sources configure \
  --name "BGM" \
  --property volume \
  --value "0.3"

# 7. Activate scene
vrooli resource obs-studio content execute --name "Presentation"

# 8. Start streaming
vrooli resource obs-studio streaming start --profile youtube

echo "Stream setup complete!"
```

## Additional Resources

- [OBS Studio Scene Collection Format](https://obsproject.com/wiki/Scene-Collection-Format)
- [Source Types Reference](https://obsproject.com/wiki/Sources)
- [Filter Documentation](https://obsproject.com/wiki/Filters)
- [Audio Configuration Guide](https://obsproject.com/wiki/Audio)