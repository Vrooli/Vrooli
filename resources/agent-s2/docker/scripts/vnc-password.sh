#!/bin/bash
# Set VNC password for x11vnc

VNC_PASSWORD="${1:-agents2vnc}"
VNC_PASSWORD_FILE="/home/agents2/.vnc/passwd"

# Create VNC directory
mkdir -p /home/agents2/.vnc

# Set password
x11vnc -storepasswd "$VNC_PASSWORD" "$VNC_PASSWORD_FILE" 2>/dev/null || {
    # If x11vnc is not available during build, create the file manually
    echo "$VNC_PASSWORD" | vncpasswd -f > "$VNC_PASSWORD_FILE" 2>/dev/null || {
        # Fallback: create a simple password file
        echo "$VNC_PASSWORD" > "$VNC_PASSWORD_FILE"
    }
}

# Set permissions
chmod 600 "$VNC_PASSWORD_FILE"
chown agents2:agents2 "$VNC_PASSWORD_FILE"