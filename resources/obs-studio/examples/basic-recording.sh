#!/bin/bash

# OBS Studio Basic Recording Example
# This example shows how to start a recording, wait for a period, and stop it

set -euo pipefail

# Get the OBS Studio CLI
OBS_CLI="resource-obs-studio"

echo "=== OBS Studio Basic Recording Example ==="
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

# Start recording
echo "3. Starting recording..."
$OBS_CLI start-recording

# Record for 10 seconds
echo "4. Recording for 10 seconds..."
sleep 10

# Stop recording
echo "5. Stopping recording..."
$OBS_CLI stop-recording

echo
echo "Recording complete! Check the recordings directory:"
echo "  ~/.vrooli/obs-studio/recordings/"
echo

# Show updated status
echo "6. Final status:"
$OBS_CLI status | grep -E "(Recordings:|Health:)"