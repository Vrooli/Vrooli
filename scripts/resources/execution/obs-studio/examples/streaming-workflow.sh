#!/bin/bash

# OBS Studio Streaming Workflow Example
# This example shows how to set up and manage a streaming session

set -euo pipefail

# Get the OBS Studio CLI
OBS_CLI="resource-obs-studio"

echo "=== OBS Studio Streaming Workflow Example ==="
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

# Show configuration
echo "3. Current configuration:"
$OBS_CLI config

# Start streaming
echo "4. Starting stream..."
echo "   Note: Stream settings must be configured in OBS Studio GUI first"
$OBS_CLI start-streaming

# Stream for 30 seconds
echo "5. Streaming for 30 seconds..."
echo "   Stream is now live!"
sleep 30

# Stop streaming
echo "6. Stopping stream..."
$OBS_CLI stop-streaming

echo
echo "Streaming session complete!"
echo "To configure stream settings (platform, key, etc.), use the OBS Studio GUI."