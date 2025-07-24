#!/bin/bash
# Enhanced entrypoint for Node-RED with host system access

# Set up PATH to include host directories
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/host/usr/bin:/host/bin:$PATH"

# Create command wrapper function for host access
command_not_found_handle() {
    local cmd="$1"
    shift
    
    # Check if command exists in host directories
    if [[ -x "/host/usr/bin/$cmd" ]]; then
        "/host/usr/bin/$cmd" "$@"
    elif [[ -x "/host/bin/$cmd" ]]; then
        "/host/bin/$cmd" "$@"
    else
        echo "bash: $cmd: command not found" >&2
        return 127
    fi
}

# Export the function so it's available in subshells
export -f command_not_found_handle

# Set up environment for Node-RED
export NODE_PATH=/usr/src/node-red/node_modules:/data/node_modules:$NODE_PATH
export NODE_RED_HOME=/usr/src/node-red

# Ensure data directory has correct permissions
if [[ "$(id -u)" = "0" ]]; then
    # Running as root, fix permissions
    chown -R node-red:node-red /data 2>/dev/null || true
    
    # Drop to node-red user
    exec su-exec node-red "$0" "$@"
fi

# Create required directories if they don't exist
mkdir -p /data/flows /data/nodes /data/lib/flows

# Copy default flows if none exist
if [[ ! -f "/data/flows.json" ]] && [[ -f "/data/flows/default-flows.json" ]]; then
    cp /data/flows/default-flows.json /data/flows.json
fi

# Set default flow file if not specified
export NODE_RED_FLOW_FILE="${NODE_RED_FLOW_FILE:-flows.json}"

# Start Node-RED with the original npm start command
cd /usr/src/node-red
exec npm start -- --userDir /data $NODE_RED_OPTIONS "$@"