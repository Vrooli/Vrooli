#\!/bin/bash

# Stop functions for AutoGPT

autogpt_stop() {
    echo "[HEADER]  Stopping AutoGPT..."
    
    if \! autogpt_container_running; then
        echo "[INFO]    AutoGPT is not running"
        return 0
    fi
    
    echo "[INFO]    Stopping AutoGPT container..."
    docker stop "$AUTOGPT_CONTAINER_NAME" >/dev/null 2>&1
    
    echo "[SUCCESS] AutoGPT stopped"
    return 0
}
