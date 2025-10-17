#!/bin/bash

# OBS Studio Audio Control Example
# This example shows how to control audio sources

set -euo pipefail

# Get the OBS Studio CLI
OBS_CLI="resource-obs-studio"

echo "=== OBS Studio Audio Control Example ==="
echo

# Check OBS Studio status
echo "1. Checking OBS Studio status..."
$OBS_CLI status

# Start OBS Studio if not running
if ! $OBS_CLI status | grep -q "Running: true"; then
    echo "2. Starting OBS Studio..."
    $OBS_CLI start
    sleep 3
fi

# Example audio source name (typically "Desktop Audio" or "Mic/Aux")
AUDIO_SOURCE="Desktop Audio"

echo "3. Audio control examples:"
echo

# Mute audio
echo "   Muting '$AUDIO_SOURCE'..."
$OBS_CLI mute "$AUDIO_SOURCE"
sleep 2

# Unmute audio
echo "   Unmuting '$AUDIO_SOURCE'..."
$OBS_CLI unmute "$AUDIO_SOURCE"
sleep 2

# Set volume to 50%
echo "   Setting volume to 50%..."
$OBS_CLI set-volume "$AUDIO_SOURCE" 50
sleep 2

# Set volume to 75%
echo "   Setting volume to 75%..."
$OBS_CLI set-volume "$AUDIO_SOURCE" 75

echo
echo "Audio control demonstration complete!"
echo "Note: Audio source names must match those configured in OBS Studio."