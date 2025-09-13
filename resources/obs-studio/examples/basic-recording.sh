#!/bin/bash

# OBS Studio Basic Recording Example
# This example shows how to start a recording, wait for a period, and stop it

set -euo pipefail

# Get the OBS Studio CLI
OBS_CLI="vrooli resource obs-studio"

echo "=== OBS Studio Basic Recording Example ==="
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

# Add a recording profile
echo "3. Setting up recording profile..."
cat > /tmp/recording-profile.json <<EOF
{
  "name": "basic-recording",
  "format": "mkv",
  "encoder": "x264",
  "quality": "high",
  "fps": 30,
  "resolution": "1920x1080"
}
EOF
$OBS_CLI content add --file /tmp/recording-profile.json --type recording

# Start recording
echo "4. Starting recording..."
$OBS_CLI content record start

# Record for 10 seconds
echo "5. Recording for 10 seconds..."
sleep 10

# Stop recording
echo "6. Stopping recording..."
$OBS_CLI content record stop

# Clean up profile
echo "7. Cleaning up profile..."
$OBS_CLI content remove --name basic-recording --type recording

echo
echo "Recording complete! Check the recordings directory:"
echo "  ~/Videos/obs-recordings/"
echo

# Show updated status
echo "8. Final status:"
$OBS_CLI status | grep -E "(Recordings:|Health:)"