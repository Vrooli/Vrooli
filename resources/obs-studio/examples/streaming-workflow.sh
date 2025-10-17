#!/bin/bash

# OBS Studio Streaming Workflow Example
# This example shows how to set up and manage a streaming session

set -euo pipefail

# Get the OBS Studio CLI
OBS_CLI="vrooli resource obs-studio"

echo "=== OBS Studio Streaming Workflow Example ==="
echo

# Check OBS Studio status
echo "1. Checking OBS Studio status..."
$OBS_CLI status

# Start OBS Studio if not running
if ! $OBS_CLI status | grep -q "Running: Yes"; then
    echo "2. Starting OBS Studio..."
    $OBS_CLI manage start
    sleep 3
fi

# Show configuration
echo "3. Current configuration:"
$OBS_CLI info

# Add a streaming profile if not exists
echo "4. Setting up streaming profile..."
cat > /tmp/streaming-profile.json <<EOF
{
  "name": "example-stream",
  "server": "rtmp://example.twitch.tv/live",
  "key": "YOUR_STREAM_KEY_HERE",
  "bitrate": 2500,
  "resolution": "1920x1080",
  "fps": 30
}
EOF
$OBS_CLI content add --file /tmp/streaming-profile.json --type streaming

# Start streaming using the profile
echo "5. Starting stream with profile..."
echo "   Note: In production, configure actual stream settings"
$OBS_CLI content stream start

# Stream for 30 seconds (demonstration only)
echo "6. Streaming for 30 seconds..."
echo "   Stream would be live with proper configuration!"
sleep 30

# Stop streaming
echo "7. Stopping stream..."
$OBS_CLI content stream stop

# Clean up
echo "8. Cleaning up profile..."
$OBS_CLI content remove --name example-stream --type streaming

echo
echo "Streaming session example complete!"
echo "To use in production:"
echo "  1. Replace server and key with actual streaming platform credentials"
echo "  2. Configure scenes and sources in OBS Studio"
echo "  3. Use 'content execute' to activate streaming profiles"