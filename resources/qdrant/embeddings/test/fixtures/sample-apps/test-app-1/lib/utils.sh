#!/usr/bin/env bash
# Simple utility functions for test app

#######################################
# Send test notification
# Arguments:
#   $1 - Message to send
# Returns: 0 on success
#######################################
notify::send() {
    local message="$1"
    
    if [[ -z "$message" ]]; then
        echo "Error: Message required" >&2
        return 1
    fi
    
    # Send webhook notification
    curl -X POST \
        -H "Content-Type: application/json" \
        -d "{\"message\":\"$message\"}" \
        "http://localhost:5678/webhook/notify" || {
            echo "Error: Failed to send notification" >&2
            return 1
        }
    
    echo "Notification sent: $message"
}

#######################################
# Validate webhook endpoint
# Returns: 0 if accessible
#######################################
notify::health_check() {
    if curl -s -f "http://localhost:5678/health" >/dev/null; then
        echo "Webhook endpoint healthy"
        return 0
    else
        echo "Webhook endpoint unavailable"
        return 1
    fi
}