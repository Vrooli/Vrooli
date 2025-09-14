#!/bin/bash
# Example: Set up streaming with OBS Studio

set -euo pipefail

echo "📺 OBS Studio Streaming Setup Example"
echo "======================================"
echo ""

# Check if OBS Studio is running
if ! vrooli resource obs-studio status | grep -q "Running: Yes"; then
    echo "Starting OBS Studio..."
    vrooli resource obs-studio manage start --wait
fi

echo "1. Creating Twitch streaming profile..."
cat > /tmp/twitch-profile.json <<EOF
{
    "platform": "twitch",
    "server_url": "rtmp://live.twitch.tv/live",
    "stream_key": "YOUR_STREAM_KEY_HERE",
    "bitrate": 3500,
    "audio_bitrate": 160,
    "resolution": "1920x1080",
    "fps": 30
}
EOF

# Add the streaming profile
vrooli resource obs-studio content add --file /tmp/twitch-profile.json --type streaming --name twitch-main
echo "✅ Twitch profile created"

echo ""
echo "2. Configuring streaming settings..."
vrooli resource obs-studio streaming configure \
    --bitrate 3500 \
    --audio-bitrate 160 \
    --resolution "1920x1080" \
    --fps 30 \
    --encoder x264
echo "✅ Streaming settings configured"

echo ""
echo "3. Setting up sources..."

# Add a screen capture source
vrooli resource obs-studio sources add \
    --name "main-screen" \
    --type screen \
    --settings '{"monitor": 0, "capture_cursor": true}'
echo "✅ Screen capture source added"

# Add a webcam source
vrooli resource obs-studio sources add \
    --name "webcam" \
    --type camera \
    --settings '{"device_id": "default", "resolution": "1280x720", "fps": 30}'
echo "✅ Webcam source added"

# Add a text overlay
vrooli resource obs-studio sources add \
    --name "stream-title" \
    --type text \
    --settings '{"text": "Live Stream - Vrooli Demo", "font": {"size": 48}}'
echo "✅ Text overlay added"

echo ""
echo "4. Creating scene configuration..."
cat > /tmp/main-scene.json <<EOF
{
    "name": "Main Stream Scene",
    "sources": [
        {
            "name": "main-screen",
            "position": {"x": 0, "y": 0},
            "scale": {"x": 1.0, "y": 1.0}
        },
        {
            "name": "webcam",
            "position": {"x": 1520, "y": 820},
            "scale": {"x": 0.2, "y": 0.2}
        },
        {
            "name": "stream-title",
            "position": {"x": 10, "y": 10}
        }
    ]
}
EOF

vrooli resource obs-studio content add --file /tmp/main-scene.json --name main-scene
echo "✅ Scene created"

echo ""
echo "5. Testing streaming connectivity..."
if vrooli resource obs-studio streaming test --profile twitch-main; then
    echo "✅ Streaming server is reachable"
else
    echo "⚠️  Could not reach streaming server - check your network settings"
fi

echo ""
echo "6. Checking streaming profiles..."
vrooli resource obs-studio streaming profiles

echo ""
echo "7. Viewing all configured sources..."
vrooli resource obs-studio sources list

echo ""
echo "============================================"
echo "✅ Streaming setup complete!"
echo ""
echo "To start streaming:"
echo "  vrooli resource obs-studio streaming start --profile twitch-main"
echo ""
echo "To stop streaming:"
echo "  vrooli resource obs-studio streaming stop"
echo ""
echo "To check streaming status:"
echo "  vrooli resource obs-studio streaming status"
echo ""
echo "⚠️  Remember to update the stream key in the profile before going live!"

# Clean up temp files
rm -f /tmp/twitch-profile.json /tmp/main-scene.json