#!/bin/bash
# Agent S2 Docker Entrypoint

set -e

# Start X virtual framebuffer
echo "Starting Xvfb..."
Xvfb :99 -screen 0 1920x1080x24 &

# Wait for display to be ready
sleep 2

# Start window manager
echo "Starting window manager..."
DISPLAY=:99 fluxbox &

# Set VNC password
echo "Setting VNC password..."
/opt/agent-s2/vnc-password.sh

# Start VNC server
echo "Starting VNC server..."
x11vnc -display :99 -forever -shared -rfbport 5900 -rfbauth /home/agents2/.vnc/passwd &

# Start Agent S2 API server
echo "Starting Agent S2 API server..."
cd /opt/agent-s2
export PYTHONPATH=/opt/agent-s2:$PYTHONPATH
python -m agent_s2.server

# Keep container running
tail -f /dev/null