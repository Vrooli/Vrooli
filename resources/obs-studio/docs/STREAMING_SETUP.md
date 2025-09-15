# OBS Studio Streaming Setup Guide

## Overview
This guide walks you through setting up streaming with OBS Studio for various platforms including Twitch, YouTube, Facebook, and custom RTMP servers.

## Prerequisites

### System Requirements
- **CPU**: Intel i5-4xxx or AMD equivalent (minimum)
- **GPU**: DirectX 10.1 compatible (for hardware encoding)
- **RAM**: 8GB minimum, 16GB recommended
- **Network**: 5 Mbps upload minimum, 10+ Mbps recommended
- **Storage**: 10GB free space for cache and recordings

### Network Requirements
```bash
# Test your upload speed
curl -s https://raw.githubusercontent.com/sivel/speedtest-cli/master/speedtest.py | python - --simple

# Check network stability
ping -c 100 stream.platform.com
```

## Quick Start

### 1. Initialize OBS Studio
```bash
# Install and start OBS
vrooli resource obs-studio manage install
vrooli resource obs-studio manage start --wait

# Verify it's running
vrooli resource obs-studio status
```

### 2. Configure Your First Stream
```bash
# Set up Twitch streaming
vrooli resource obs-studio streaming configure \
  --server "rtmp://live.twitch.tv/live" \
  --key "YOUR_STREAM_KEY"

# Test the configuration
vrooli resource obs-studio streaming test --duration 10
```

### 3. Start Streaming
```bash
# Start with default settings
vrooli resource obs-studio streaming start

# Check stream status
vrooli resource obs-studio streaming status
```

## Platform-Specific Setup

### Twitch

#### Getting Your Stream Key
1. Go to [dashboard.twitch.tv](https://dashboard.twitch.tv)
2. Navigate to Settings → Stream
3. Copy your Primary Stream key

#### Configuration
```bash
# Add Twitch profile
vrooli resource obs-studio streaming profiles \
  --add twitch \
  --server "rtmp://live.twitch.tv/live" \
  --key "live_XXXXXXXXX_XXXXXXXXXXXXXXXXXXXXXXXXX"

# Recommended Twitch settings
vrooli resource obs-studio streaming configure \
  --bitrate 6000 \
  --keyframe-interval 2 \
  --preset "fast" \
  --encoder "x264" \
  --audio-bitrate 160
```

#### Twitch Recommended Settings
| Quality | Resolution | Bitrate | FPS |
|---------|------------|---------|-----|
| 1080p60 | 1920x1080 | 6000 | 60 |
| 1080p30 | 1920x1080 | 4500 | 30 |
| 720p60 | 1280x720 | 4500 | 60 |
| 720p30 | 1280x720 | 3000 | 30 |

### YouTube

#### Getting Your Stream Key
1. Go to [YouTube Studio](https://studio.youtube.com)
2. Click "Create" → "Go Live"
3. Copy the Stream key from Stream Settings

#### Configuration
```bash
# Add YouTube profile
vrooli resource obs-studio streaming profiles \
  --add youtube \
  --server "rtmp://a.rtmp.youtube.com/live2" \
  --key "xxxx-xxxx-xxxx-xxxx-xxxx"

# Recommended YouTube settings
vrooli resource obs-studio streaming configure \
  --bitrate 9000 \
  --keyframe-interval 2 \
  --preset "medium" \
  --encoder "x264" \
  --audio-bitrate 256
```

#### YouTube Recommended Settings
| Quality | Resolution | Bitrate | FPS |
|---------|------------|---------|-----|
| 4K60 | 3840x2160 | 20000-51000 | 60 |
| 1440p60 | 2560x1440 | 9000-18000 | 60 |
| 1080p60 | 1920x1080 | 4500-9000 | 60 |
| 720p60 | 1280x720 | 2250-6000 | 60 |

### Facebook Live

#### Configuration
```bash
# Add Facebook profile
vrooli resource obs-studio streaming profiles \
  --add facebook \
  --server "rtmps://live-api-s.facebook.com:443/rtmp/" \
  --key "FB-STREAM-KEY"

# Facebook recommended settings
vrooli resource obs-studio streaming configure \
  --bitrate 4000 \
  --keyframe-interval 2 \
  --preset "fast" \
  --audio-bitrate 128
```

### Custom RTMP Server

#### Configuration
```bash
# Add custom RTMP server
vrooli resource obs-studio streaming profiles \
  --add custom \
  --server "rtmp://your-server.com/live" \
  --key "your-stream-key"

# Configure for custom server
vrooli resource obs-studio streaming configure \
  --bitrate 5000 \
  --keyframe-interval 2 \
  --preset "medium"
```

## Encoding Settings

### Software Encoding (x264)
Best for quality, higher CPU usage.

```bash
# Configure x264 encoder
vrooli resource obs-studio streaming configure \
  --encoder "x264" \
  --preset "medium" \
  --profile "high" \
  --tune "zerolatency" \
  --x264-params "bframes=2:keyint=120:min-keyint=60"
```

#### x264 Presets (Fastest to Slowest)
- `ultrafast`: Lowest quality, minimal CPU
- `superfast`: Low quality, low CPU
- `veryfast`: Decent quality, moderate CPU
- `faster`: Good quality, moderate CPU
- `fast`: Good quality, moderate-high CPU
- `medium`: Better quality, high CPU (default)
- `slow`: Best quality, very high CPU
- `slower`: Best quality, extreme CPU
- `veryslow`: Maximum quality, extreme CPU

### Hardware Encoding

#### NVIDIA NVENC
```bash
# Configure NVENC (NVIDIA GPUs)
vrooli resource obs-studio streaming configure \
  --encoder "nvenc" \
  --preset "quality" \
  --profile "high" \
  --look-ahead true \
  --psycho-visual-tuning true \
  --gpu 0
```

#### AMD AMF
```bash
# Configure AMF (AMD GPUs)
vrooli resource obs-studio streaming configure \
  --encoder "amf" \
  --preset "quality" \
  --profile "high" \
  --quality-preset "quality"
```

#### Intel Quick Sync
```bash
# Configure Quick Sync (Intel CPUs with iGPU)
vrooli resource obs-studio streaming configure \
  --encoder "qsv" \
  --preset "quality" \
  --profile "high" \
  --async-depth 4
```

## Audio Configuration

### Basic Audio Setup
```bash
# Configure audio settings
vrooli resource obs-studio streaming configure \
  --audio-bitrate 160 \
  --audio-sample-rate 48000 \
  --audio-channels "stereo"
```

### Advanced Audio Processing
```bash
# Add noise suppression
vrooli resource obs-studio sources configure \
  --name "Microphone" \
  --filter "noise_suppress" \
  --suppress-level -30

# Add compressor
vrooli resource obs-studio sources configure \
  --name "Microphone" \
  --filter "compressor" \
  --ratio 4.0 \
  --threshold -20 \
  --attack 1 \
  --release 60

# Add limiter
vrooli resource obs-studio sources configure \
  --name "Microphone" \
  --filter "limiter" \
  --threshold -3 \
  --release 60
```

## Multi-Platform Streaming

### Simultaneous Streaming
Stream to multiple platforms at once using restream.io or similar services.

```bash
# Configure for restream.io
vrooli resource obs-studio streaming profiles \
  --add restream \
  --server "rtmp://live.restream.io/live" \
  --key "YOUR_RESTREAM_KEY"

# Start multi-platform stream
vrooli resource obs-studio streaming start --profile restream
```

### Platform Switching
```bash
#!/bin/bash
# Script to switch between platforms

switch_platform() {
    local platform="$1"
    
    # Stop current stream
    vrooli resource obs-studio streaming stop
    
    # Wait for stream to fully stop
    sleep 5
    
    # Start with new platform
    vrooli resource obs-studio streaming start --profile "$platform"
    
    echo "Switched to $platform"
}

# Usage
switch_platform "youtube"
```

## Optimization Tips

### Network Optimization

#### 1. Bitrate Calculator
```bash
# Calculate optimal bitrate
calculate_bitrate() {
    local upload_speed="$1"
    local safety_margin=0.75
    
    # Use 75% of upload speed for stability
    echo "scale=0; $upload_speed * 1000 * $safety_margin" | bc
}

# Example: 10 Mbps upload
optimal_bitrate=$(calculate_bitrate 10)
echo "Optimal bitrate: ${optimal_bitrate} kbps"
```

#### 2. Dynamic Bitrate
```bash
# Adjust bitrate based on network conditions
monitor_and_adjust() {
    while true; do
        # Get current dropped frames
        status=$(vrooli resource obs-studio streaming status --format json)
        dropped=$(echo "$status" | jq -r '.dropped_frames')
        
        if [ "$dropped" -gt 100 ]; then
            # Reduce bitrate by 500
            current_bitrate=$(echo "$status" | jq -r '.bitrate')
            new_bitrate=$((current_bitrate - 500))
            vrooli resource obs-studio streaming configure --bitrate "$new_bitrate"
            echo "Reduced bitrate to ${new_bitrate} due to dropped frames"
        fi
        
        sleep 30
    done
}
```

### CPU Optimization

#### Process Priority
```bash
# Set OBS to high priority
obs_pid=$(pgrep -f obs-studio)
sudo renice -n -5 -p "$obs_pid"
```

#### CPU Affinity
```bash
# Dedicate CPU cores to OBS
taskset -cp 4-7 "$obs_pid"  # Use cores 4-7
```

### Scene Optimization

#### 1. Reduce Source Complexity
```bash
# Disable unused sources
vrooli resource obs-studio sources visibility --name "overlay1" --hide
vrooli resource obs-studio sources visibility --name "overlay2" --hide
```

#### 2. Lower Source Resolution
```bash
# Reduce webcam resolution for PiP
vrooli resource obs-studio sources configure \
  --name "Webcam" \
  --property resolution \
  --value "640x480"
```

## Stream Health Monitoring

### Real-Time Monitoring Script
```bash
#!/bin/bash

monitor_stream_health() {
    echo "Starting stream health monitor..."
    
    while true; do
        # Get stream status
        status=$(vrooli resource obs-studio streaming status --format json)
        
        if [ $? -eq 0 ]; then
            # Extract metrics
            fps=$(echo "$status" | jq -r '.fps')
            bitrate=$(echo "$status" | jq -r '.bitrate')
            dropped=$(echo "$status" | jq -r '.dropped_frames')
            duration=$(echo "$status" | jq -r '.duration')
            
            # Display health
            clear
            echo "=== Stream Health Monitor ==="
            echo "Duration: $duration"
            echo "FPS: $fps"
            echo "Bitrate: ${bitrate} kbps"
            echo "Dropped Frames: $dropped"
            
            # Alert on issues
            if [ "$dropped" -gt 500 ]; then
                echo "⚠️  WARNING: High dropped frames!"
                # Could send notification here
            fi
            
            if [ "$fps" -lt 25 ]; then
                echo "⚠️  WARNING: Low FPS detected!"
            fi
        else
            echo "Stream is offline"
        fi
        
        sleep 5
    done
}

monitor_stream_health
```

### Automated Recovery
```bash
#!/bin/bash

auto_recover_stream() {
    local max_retries=3
    local retry_count=0
    
    while true; do
        # Check if stream is live
        status=$(vrooli resource obs-studio streaming status --format json)
        is_live=$(echo "$status" | jq -r '.streaming')
        
        if [ "$is_live" = "false" ] && [ $retry_count -lt $max_retries ]; then
            echo "Stream offline! Attempting recovery (${retry_count}/${max_retries})..."
            
            # Restart stream
            vrooli resource obs-studio streaming stop
            sleep 5
            vrooli resource obs-studio streaming start
            
            retry_count=$((retry_count + 1))
            sleep 10
        elif [ "$is_live" = "true" ]; then
            retry_count=0
        fi
        
        sleep 30
    done
}
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Stream Not Starting
```bash
# Check configuration
vrooli resource obs-studio streaming test

# Verify stream key
vrooli resource obs-studio streaming profiles

# Check network connectivity
ping -c 4 live.twitch.tv
```

#### 2. Dropped Frames
```bash
# Reduce bitrate
vrooli resource obs-studio streaming configure --bitrate 3000

# Change encoder preset
vrooli resource obs-studio streaming configure --preset "veryfast"

# Check CPU usage
top -p $(pgrep -f obs-studio)
```

#### 3. Audio Sync Issues
```bash
# Adjust audio sync offset
vrooli resource obs-studio sources configure \
  --name "Microphone" \
  --property sync_offset \
  --value 50  # milliseconds
```

#### 4. Black Screen
```bash
# Check source visibility
vrooli resource obs-studio sources list

# Verify capture permissions
ls -la /dev/video*

# Restart with clean configuration
vrooli resource obs-studio manage restart
```

### Diagnostic Commands
```bash
# Full diagnostic report
diagnose_streaming() {
    echo "=== OBS Streaming Diagnostics ==="
    
    # Check service status
    echo -e "\n1. Service Status:"
    vrooli resource obs-studio status
    
    # Check streaming status
    echo -e "\n2. Stream Status:"
    vrooli resource obs-studio streaming status
    
    # List active sources
    echo -e "\n3. Active Sources:"
    vrooli resource obs-studio sources list
    
    # Check network
    echo -e "\n4. Network Test:"
    ping -c 4 live.twitch.tv
    
    # Check resources
    echo -e "\n5. System Resources:"
    free -h
    df -h
    
    # Check logs
    echo -e "\n6. Recent Logs:"
    vrooli resource obs-studio logs | tail -20
}

diagnose_streaming
```

## Advanced Workflows

### Scheduled Streaming
```bash
#!/bin/bash
# Schedule streams using cron

# Add to crontab:
# 0 20 * * 1-5 /path/to/start_stream.sh  # Weekdays at 8 PM
# 0 22 * * 1-5 /path/to/stop_stream.sh   # Stop at 10 PM

# start_stream.sh
vrooli resource obs-studio manage start --wait
vrooli resource obs-studio content execute --name "Stream Scene"
vrooli resource obs-studio streaming start --profile twitch

# stop_stream.sh
vrooli resource obs-studio streaming stop
vrooli resource obs-studio manage stop
```

### Stream Recording
```bash
# Record while streaming
start_stream_with_recording() {
    # Start streaming
    vrooli resource obs-studio streaming start
    
    # Start recording
    vrooli resource obs-studio recording start \
      --output "/recordings/stream_$(date +%Y%m%d_%H%M%S).mp4"
    
    echo "Streaming and recording started"
}

# Stop both
stop_stream_and_recording() {
    vrooli resource obs-studio recording stop
    vrooli resource obs-studio streaming stop
    echo "Streaming and recording stopped"
}
```

### Scene Automation
```bash
#!/bin/bash
# Automated scene switching during stream

stream_with_scenes() {
    # Start with intro
    vrooli resource obs-studio content execute --name "Starting Soon"
    vrooli resource obs-studio streaming start
    
    # Wait 5 minutes
    sleep 300
    
    # Switch to main content
    vrooli resource obs-studio content execute --name "Main Content"
    
    # Stream main content
    sleep 3600  # 1 hour
    
    # Switch to outro
    vrooli resource obs-studio content execute --name "Ending Screen"
    
    # Wait 2 minutes
    sleep 120
    
    # Stop stream
    vrooli resource obs-studio streaming stop
}
```

## Best Practices

### 1. Pre-Stream Checklist
- [ ] Test internet connection
- [ ] Verify stream key is correct
- [ ] Check audio levels
- [ ] Test all sources
- [ ] Clear disk space for recordings
- [ ] Close unnecessary applications
- [ ] Set streaming schedule/title

### 2. Stream Quality Guidelines
- Use consistent frame rates (30 or 60 FPS)
- Match output resolution to content
- Leave 20% headroom on bitrate
- Monitor CPU usage (<80%)
- Keep local recordings as backup

### 3. Professional Tips
- Use a stream deck for quick scene switching
- Set up automatic tweets/posts when going live
- Have a backup streaming method ready
- Test everything before going live
- Keep stream keys secure and rotate regularly

## Integration Examples

### Discord Webhook Notifications
```bash
#!/bin/bash

WEBHOOK_URL="https://discord.com/api/webhooks/..."

notify_discord() {
    local message="$1"
    curl -H "Content-Type: application/json" \
         -X POST \
         -d "{\"content\":\"$message\"}" \
         "$WEBHOOK_URL"
}

# Start stream with notification
vrooli resource obs-studio streaming start
notify_discord "🔴 Stream is now LIVE on Twitch!"
```

### Twitter Integration
```bash
# Post when going live (requires Twitter API setup)
post_going_live() {
    local platform="$1"
    local url="$2"
    
    tweet="🔴 We're LIVE on $platform! Join us: $url"
    # Use your Twitter posting method here
    echo "$tweet"
}

post_going_live "Twitch" "https://twitch.tv/yourchannel"
```

## Conclusion

This guide covers the essential aspects of streaming with OBS Studio. Remember to:
1. Start with conservative settings and increase quality gradually
2. Monitor your stream health continuously
3. Have backup plans for technical issues
4. Engage with your audience consistently
5. Keep learning and improving your setup

For more advanced topics, refer to the API documentation and scene configuration guide.