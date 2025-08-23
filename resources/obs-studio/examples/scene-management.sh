#!/bin/bash

# OBS Studio Scene Management Example
# This example shows how to list, create, and switch between scenes

set -euo pipefail

# Get the OBS Studio CLI
OBS_CLI="resource-obs-studio"

echo "=== OBS Studio Scene Management Example ==="
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

# List current scenes
echo "3. Listing current scenes..."
$OBS_CLI list-scenes

# Get current scene
echo "4. Getting current scene..."
$OBS_CLI current-scene

# Example of switching scenes (if multiple scenes exist)
echo "5. Scene switching example:"
echo "   To switch to a scene named 'Main':"
echo "   $OBS_CLI switch-scene Main"

# List sources
echo "6. Listing available sources..."
$OBS_CLI list-sources

echo
echo "Scene management commands demonstrated!"
echo "You can create scenes using the OBS Studio GUI and control them via CLI."