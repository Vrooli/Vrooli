#!/bin/bash
# Startup script for Agent S2 container

# Ensure runtime directory exists
mkdir -p /tmp/runtime-agents2
chmod 700 /tmp/runtime-agents2

# Set environment variables
export DISPLAY=:99
export XDG_RUNTIME_DIR=/tmp/runtime-agents2

# Create .Xauthority file if it doesn't exist
touch /home/agents2/.Xauthority
chmod 600 /home/agents2/.Xauthority

# Wait for X server to be ready
echo "Waiting for X server..."
for i in {1..30}; do
    if xdpyinfo -display :99 >/dev/null 2>&1; then
        echo "X server is ready"
        break
    fi
    sleep 1
done

# Start the application or any other initialization
echo "Agent S2 container initialized successfully"