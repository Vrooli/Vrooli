#!/bin/bash
# Discover funnel-builder API port from running processes
# Returns the port number or empty if not found

# Method 1: ss command (most reliable)
PORT=$(ss -tlnp 2>/dev/null | grep "funnel-builder-" | grep -oP ':\K[0-9]+' | head -1)

# Method 2: netstat fallback
if [ -z "$PORT" ]; then
    PORT=$(netstat -tlnp 2>/dev/null | grep "funnel-builder-" | awk '{print $4}' | cut -d':' -f2 | head -1)
fi

# Method 3: lsof fallback
if [ -z "$PORT" ]; then
    PORT=$(lsof -i -P -n 2>/dev/null | grep "funnel-builder-" | grep LISTEN | awk '{print $9}' | cut -d':' -f2 | head -1)
fi

echo "$PORT"
