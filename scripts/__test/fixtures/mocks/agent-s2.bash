#!/usr/bin/env bash
# Agent-S2 Resource Mock Implementation
# Provides realistic mock responses for Agent-S2 browser automation service

# Prevent duplicate loading
if [[ "${AGENT_S2_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export AGENT_S2_MOCK_LOADED="true"

#######################################
# Setup Agent-S2 mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::agent-s2::setup() {
    local state="${1:-healthy}"
    
    # Configure Agent-S2-specific environment
    export AGENT_S2_PORT="${AGENT_S2_PORT:-8080}"
    export AGENT_S2_BASE_URL="http://localhost:${AGENT_S2_PORT}"
    export AGENT_S2_CONTAINER_NAME="${TEST_NAMESPACE}_agent-s2"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$AGENT_S2_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::agent-s2::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::agent-s2::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::agent-s2::setup_installing_endpoints
            ;;
        "stopped")
            mock::agent-s2::setup_stopped_endpoints
            ;;
        *)
            echo "[AGENT_S2_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[AGENT_S2_MOCK] Agent-S2 mock configured with state: $state"
}

#######################################
# Setup healthy Agent-S2 endpoints
#######################################
mock::agent-s2::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$AGENT_S2_BASE_URL/health" \
        '{"status":"ok","version":"1.0.0","browser_ready":true}'
    
    # Task execution endpoint
    mock::http::set_endpoint_response "$AGENT_S2_BASE_URL/api/v1/tasks" \
        '{
            "task_id": "task-'$(date +%s)'",
            "status": "accepted",
            "estimated_duration": 30,
            "created_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
        }' \
        "POST"
    
    # Task status endpoint
    mock::http::set_endpoint_response "$AGENT_S2_BASE_URL/api/v1/tasks/task-123" \
        '{
            "task_id": "task-123",
            "status": "completed",
            "progress": 100,
            "result": {
                "success": true,
                "data": {
                    "screenshot": "base64encodedimage...",
                    "extracted_data": {
                        "title": "Example Page",
                        "text_content": "This is the extracted text"
                    }
                }
            },
            "created_at": "2024-01-15T12:00:00Z",
            "completed_at": "2024-01-15T12:00:30Z"
        }'
    
    # Browser sessions endpoint
    mock::http::set_endpoint_response "$AGENT_S2_BASE_URL/api/v1/sessions" \
        '[
            {
                "session_id": "session-abc123",
                "status": "active",
                "created_at": "2024-01-15T12:00:00Z",
                "pages": 3
            }
        ]'
    
    # Capabilities endpoint
    mock::http::set_endpoint_response "$AGENT_S2_BASE_URL/api/v1/capabilities" \
        '{
            "browser_automation": true,
            "screenshot_capture": true,
            "text_extraction": true,
            "form_interaction": true,
            "javascript_execution": true,
            "proxy_support": true,
            "stealth_mode": true
        }'
}

#######################################
# Setup unhealthy Agent-S2 endpoints
#######################################
mock::agent-s2::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$AGENT_S2_BASE_URL/health" \
        '{"status":"error","error":"Browser engine unavailable","browser_ready":false}' \
        "GET" \
        "503"
    
    # Task endpoint returns error
    mock::http::set_endpoint_response "$AGENT_S2_BASE_URL/api/v1/tasks" \
        '{"error":"Service temporarily unavailable"}' \
        "POST" \
        "503"
}

#######################################
# Setup installing Agent-S2 endpoints
#######################################
mock::agent-s2::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$AGENT_S2_BASE_URL/health" \
        '{"status":"installing","progress":95,"current_step":"Starting browser engine","browser_ready":false}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$AGENT_S2_BASE_URL/api/v1/tasks" \
        '{"error":"Agent-S2 is still initializing"}' \
        "POST" \
        "503"
}

#######################################
# Setup stopped Agent-S2 endpoints
#######################################
mock::agent-s2::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$AGENT_S2_BASE_URL"
}

#######################################
# Mock Agent-S2-specific operations
#######################################

# Mock browser task execution
mock::agent-s2::execute_task() {
    local task_type="$1"
    local url="${2:-https://example.com}"
    
    echo '{
        "task_id": "'$(uuidgen || echo "task-$(date +%s)")'",
        "type": "'$task_type'",
        "url": "'$url'",
        "status": "running",
        "progress": 0
    }'
}

# Mock screenshot capture
mock::agent-s2::capture_screenshot() {
    local url="$1"
    
    echo '{
        "screenshot": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
        "format": "png",
        "dimensions": {"width": 1920, "height": 1080},
        "url": "'$url'",
        "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
    }'
}

#######################################
# Export mock functions
#######################################
export -f mock::agent-s2::setup
export -f mock::agent-s2::setup_healthy_endpoints
export -f mock::agent-s2::setup_unhealthy_endpoints
export -f mock::agent-s2::setup_installing_endpoints
export -f mock::agent-s2::setup_stopped_endpoints
export -f mock::agent-s2::execute_task
export -f mock::agent-s2::capture_screenshot

echo "[AGENT_S2_MOCK] Agent-S2 mock implementation loaded"