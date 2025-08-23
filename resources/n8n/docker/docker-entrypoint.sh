#!/bin/bash
# Custom entrypoint that sets up PATH for seamless host command execution

# Preserve original PATH
export ORIGINAL_PATH="$PATH"

# Add host directories to PATH (after container paths)
export PATH="$PATH:/host/usr/bin:/host/bin:/host/usr/local/bin"

# Create command wrapper function
command_not_found_handle() {
    local cmd="$1"
    shift
    
    # Try to find command in host paths
    for dir in /host/usr/bin /host/bin /host/usr/local/bin; do
        if [ -x "$dir/$cmd" ]; then
            exec "$dir/$cmd" "$@"
        fi
    done
    
    # Command not found
    echo "bash: $cmd: command not found" >&2
    return 127
}

# Export the function
export -f command_not_found_handle

# Run the original n8n entrypoint
exec tini -- /docker-entrypoint.sh "$@"