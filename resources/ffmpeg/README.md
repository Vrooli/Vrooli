# FFmpeg Resource

Universal media processing framework for video, audio, and image manipulation.

## Overview

FFmpeg is a comprehensive multimedia processing tool that enables:
- Format conversion between 100+ media formats
- Audio/video extraction and manipulation
- Stream processing and encoding
- Frame extraction and thumbnail generation
- Media analysis and metadata extraction

## Quick Start

```bash
# Install and start the resource
vrooli resource ffmpeg manage install
vrooli resource ffmpeg manage start

# Check status
vrooli resource ffmpeg status

# Run tests
vrooli resource ffmpeg test all
```

## Core Capabilities

### Media Transcoding
Convert between any supported formats:
```bash
resource-ffmpeg transcode --input video.avi --output video.mp4
resource-ffmpeg transcode --input audio.wav --output audio.mp3 --bitrate 192k
```

### Audio Extraction
Extract audio tracks from video:
```bash
resource-ffmpeg extract --type audio --input video.mp4 --output audio.mp3
```

### Frame Extraction
Extract frames or thumbnails:
```bash
resource-ffmpeg extract --type frames --input video.mp4 --output frame_%04d.jpg
resource-ffmpeg extract --type thumbnail --input video.mp4 --output thumb.jpg --time 00:00:05
```

### Media Information
Get detailed media file information:
```bash
resource-ffmpeg media-info media.mp4
```

### Resource Information
Display resource runtime configuration:
```bash
resource-ffmpeg info         # Human-readable format
resource-ffmpeg info --json  # JSON format
```

### Preset Library
Apply optimized presets for common use cases:
```bash
# List available presets
resource-ffmpeg preset list

# Apply presets
resource-ffmpeg preset apply web-720p input.mp4
resource-ffmpeg preset apply podcast audio.wav
resource-ffmpeg preset apply social-square video.mp4
resource-ffmpeg preset apply gif-from-video clip.mp4
```

Available presets include:
- **Video**: web-1080p, web-720p, mobile-high, mobile-low, social-square, social-vertical
- **Audio**: podcast, music-high, music-standard, audiobook
- **Conversion**: gif-from-video, extract-audio, remove-audio, compress-50

### Stream Processing
Process live streams and real-time media:
```bash
# Get stream information
resource-ffmpeg stream info https://stream.example.com/live.m3u8

# Capture stream to file (duration in seconds)
resource-ffmpeg stream capture https://stream.example.com/live.m3u8 output.mp4 60

# Transcode live stream
resource-ffmpeg stream transcode rtmp://input.com/live rtmp://output.com/live web-720p
```

## Configuration

Configuration is stored in `config/defaults.sh` and can be customized via environment variables:

```bash
# Processing settings
export FFMPEG_MAX_CONCURRENT_JOBS=2
export FFMPEG_DEFAULT_QUALITY=high
export FFMPEG_ENABLE_GPU=true

# Resource limits
export FFMPEG_MAX_FILE_SIZE_MB=5000
export FFMPEG_MAX_DURATION_SECONDS=3600
export FFMPEG_CPU_THREADS=0  # 0=auto

# Codec preferences
export FFMPEG_VIDEO_CODEC=h264
export FFMPEG_AUDIO_CODEC=aac
```

## Presets

Built-in conversion presets for common use cases:

### Web Video
Optimized for web streaming:
```bash
resource-ffmpeg transcode --preset web_video --input source.mov --output web.mp4
```

### Podcast Audio
Optimized for podcast distribution:
```bash
resource-ffmpeg transcode --preset podcast_audio --input raw.wav --output podcast.mp3
```

### Thumbnails
Extract video thumbnails:
```bash
resource-ffmpeg extract --preset thumbnail --input video.mp4 --output thumb.jpg
```

## Content Management

Manage processing templates and workflows:

```bash
# List available presets
resource-ffmpeg content list

# Add custom preset
resource-ffmpeg content add --name custom_preset --file preset.json

# Get preset details
resource-ffmpeg content get --name web_video

# Remove preset
resource-ffmpeg content remove --name old_preset
```

## Testing

Run comprehensive tests:

```bash
# Quick smoke test (< 30s)
vrooli resource ffmpeg test smoke

# Unit tests for library functions
vrooli resource ffmpeg test unit

# Integration tests (full workflow)
vrooli resource ffmpeg test integration

# All tests
vrooli resource ffmpeg test all
```

## API Integration

FFmpeg can be integrated with other Vrooli resources:

- **minio**: Store processed media files
- **postgres**: Track processing jobs
- **redis**: Cache processing status
- **qdrant**: Index media metadata

## Performance

### Benchmarks
- **1080p → 720p transcode**: ~30s for 1min video
- **Audio extraction**: <5s for typical song
- **Thumbnail generation**: <1s per image
- **Media info query**: <1s response time

### Hardware Acceleration
GPU acceleration is automatically detected and used when available:
- NVIDIA NVENC for H.264/H.265 encoding
- Intel Quick Sync Video
- AMD VCE/VCN
- VAAPI generic acceleration

Test hardware acceleration:
```bash
# Benchmark hardware vs software encoding
vrooli resource ffmpeg content benchmark
```

### Batch Processing
Process multiple files in parallel:
```bash
# Process all videos in a directory (4 parallel jobs)
vrooli resource ffmpeg content process /path/to/videos transcode "*.mp4" 4

# Batch extract audio from videos (2 parallel jobs)
vrooli resource ffmpeg content process /path/to/videos extract "*.avi" 2
```

## Troubleshooting

### Common Issues

**Issue**: Codec not found
```bash
# Check available codecs
ffmpeg -codecs | grep h264

# Install missing codecs
sudo apt-get install libx264-dev
```

**Issue**: Out of memory
```bash
# Reduce concurrent jobs
export FFMPEG_MAX_CONCURRENT_JOBS=1

# Limit resolution
resource-ffmpeg transcode --max-width 1280 --max-height 720
```

**Issue**: Slow processing
```bash
# Enable hardware acceleration
export FFMPEG_ENABLE_GPU=true

# Use faster preset
resource-ffmpeg transcode --preset fast
```

## Security

- Input validation on all media files
- Sandboxed execution environment
- Resource limits enforced
- No execution of embedded scripts
- Temporary files cleaned automatically

## Dependencies

- **Required**: ffmpeg binary (6.0+)
- **Optional**: GPU drivers for hardware acceleration
- **Recommended**: 4GB+ RAM, multi-core CPU

## License

FFmpeg is licensed under LGPL/GPL. See FFmpeg documentation for details.

## Support

For issues or questions:
- Check logs: `vrooli resource ffmpeg logs`
- Run diagnostics: `vrooli resource ffmpeg test smoke`
- View configuration: `vrooli resource ffmpeg info`