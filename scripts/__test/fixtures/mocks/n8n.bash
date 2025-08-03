#!/usr/bin/env bash
# N8N Resource Mock Implementation
# Provides realistic mock responses for N8N workflow automation service

# Prevent duplicate loading
if [[ "${N8N_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export N8N_MOCK_LOADED="true"

#######################################
# Setup N8N mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::n8n::setup() {
    local state="${1:-healthy}"
    
    # Configure N8N-specific environment
    export N8N_PORT="${N8N_PORT:-5678}"
    export N8N_BASE_URL="http://localhost:${N8N_PORT}"
    export N8N_CONTAINER_NAME="${TEST_NAMESPACE}_n8n"
    export N8N_BASIC_AUTH_USER="${N8N_BASIC_AUTH_USER:-admin}"
    export N8N_BASIC_AUTH_PASSWORD="${N8N_BASIC_AUTH_PASSWORD:-password}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$N8N_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::n8n::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::n8n::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::n8n::setup_installing_endpoints
            ;;
        "stopped")
            mock::n8n::setup_stopped_endpoints
            ;;
        *)
            echo "[N8N_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[N8N_MOCK] N8N mock configured with state: $state"
}

#######################################
# Setup healthy N8N endpoints
#######################################
mock::n8n::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$N8N_BASE_URL/healthz" \
        '{"status":"ok","version":"1.25.0"}'
    
    # Workflows endpoint
    mock::http::set_endpoint_response "$N8N_BASE_URL/api/v1/workflows" \
        '{
            "data": [
                {
                    "id": "1",
                    "name": "Welcome Workflow",
                    "active": true,
                    "createdAt": "2024-01-15T10:00:00.000Z",
                    "updatedAt": "2024-01-15T10:00:00.000Z",
                    "nodes": [
                        {
                            "id": "node1",
                            "name": "Start",
                            "type": "n8n-nodes-base.start",
                            "position": [100, 100]
                        },
                        {
                            "id": "node2",
                            "name": "Set",
                            "type": "n8n-nodes-base.set",
                            "position": [300, 100]
                        }
                    ],
                    "connections": {
                        "Start": {
                            "main": [[{"node": "Set", "type": "main", "index": 0}]]
                        }
                    }
                },
                {
                    "id": "2",
                    "name": "Data Processing Pipeline",
                    "active": false,
                    "createdAt": "2024-01-16T11:00:00.000Z",
                    "updatedAt": "2024-01-16T11:00:00.000Z"
                }
            ],
            "nextCursor": null
        }'
    
    # Single workflow endpoint
    mock::http::set_endpoint_response "$N8N_BASE_URL/api/v1/workflows/1" \
        '{
            "data": {
                "id": "1",
                "name": "Welcome Workflow",
                "active": true,
                "nodes": [],
                "connections": {},
                "settings": {
                    "executionOrder": "v1"
                }
            }
        }'
    
    # Workflow activation endpoint
    mock::http::set_endpoint_response "$N8N_BASE_URL/api/v1/workflows/1/activate" \
        '{"data":{"id":"1","active":true}}' \
        "PATCH"
    
    # Executions endpoint
    mock::http::set_endpoint_response "$N8N_BASE_URL/api/v1/executions" \
        '{
            "data": [
                {
                    "id": "100",
                    "finished": true,
                    "mode": "manual",
                    "startedAt": "2024-01-15T12:00:00.000Z",
                    "stoppedAt": "2024-01-15T12:00:05.000Z",
                    "workflowId": "1",
                    "status": "success"
                }
            ],
            "nextCursor": null
        }'
    
    # Execute workflow endpoint
    mock::http::set_endpoint_response "$N8N_BASE_URL/api/v1/workflows/1/execute" \
        '{
            "data": {
                "executionId": "101",
                "status": "running"
            }
        }' \
        "POST"
    
    # Credentials endpoint
    mock::http::set_endpoint_response "$N8N_BASE_URL/api/v1/credentials" \
        '{
            "data": [
                {
                    "id": "1",
                    "name": "HTTP Header Auth",
                    "type": "httpHeaderAuth",
                    "createdAt": "2024-01-15T10:00:00.000Z"
                }
            ]
        }'
}

#######################################
# Setup unhealthy N8N endpoints
#######################################
mock::n8n::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$N8N_BASE_URL/healthz" \
        '{"status":"unhealthy","error":"Database connection failed"}' \
        "GET" \
        "503"
    
    # API endpoints return errors
    mock::http::set_endpoint_response "$N8N_BASE_URL/api/v1/workflows" \
        '{"code":503,"message":"Service temporarily unavailable"}' \
        "GET" \
        "503"
}

#######################################
# Setup installing N8N endpoints
#######################################
mock::n8n::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$N8N_BASE_URL/healthz" \
        '{"status":"installing","progress":80,"current_step":"Initializing database"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$N8N_BASE_URL/api/v1/workflows" \
        '{"code":503,"message":"N8N is still initializing"}' \
        "GET" \
        "503"
}

#######################################
# Setup stopped N8N endpoints
#######################################
mock::n8n::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$N8N_BASE_URL"
}

#######################################
# Mock N8N-specific operations
#######################################

# Mock workflow creation
mock::n8n::create_workflow() {
    local workflow_name="$1"
    local workflow_type="${2:-simple}"
    
    case "$workflow_type" in
        "simple")
            echo '{
                "data": {
                    "id": "'$(date +%s)'",
                    "name": "'$workflow_name'",
                    "active": false,
                    "nodes": [
                        {"id": "start", "type": "n8n-nodes-base.start"},
                        {"id": "set", "type": "n8n-nodes-base.set"}
                    ],
                    "connections": {
                        "start": {"main": [[{"node": "set", "type": "main", "index": 0}]]}
                    }
                }
            }'
            ;;
        "webhook")
            echo '{
                "data": {
                    "id": "'$(date +%s)'",
                    "name": "'$workflow_name'",
                    "active": true,
                    "nodes": [
                        {"id": "webhook", "type": "n8n-nodes-base.webhook"},
                        {"id": "respond", "type": "n8n-nodes-base.respondToWebhook"}
                    ]
                }
            }'
            ;;
    esac
}

# Mock workflow execution
mock::n8n::simulate_workflow_execution() {
    local workflow_id="$1"
    local execution_time="${2:-5}"
    
    echo '{
        "data": {
            "executionId": "'$(date +%s)'",
            "workflowId": "'$workflow_id'",
            "status": "success",
            "mode": "manual",
            "startedAt": "'$(date -u +%Y-%m-%dT%H:%M:%S.000Z)'",
            "executionTime": '$execution_time',
            "data": {
                "resultData": {
                    "runData": {
                        "Set": [
                            {
                                "data": {
                                    "main": [[{"json": {"result": "success"}}]]
                                }
                            }
                        ]
                    }
                }
            }
        }
    }'
}

# Mock community nodes
mock::n8n::get_community_nodes() {
    echo '{
        "data": [
            {
                "name": "n8n-nodes-custom",
                "version": "1.0.0",
                "description": "Custom nodes for N8N"
            }
        ]
    }'
}

#######################################
# Export mock functions
#######################################
export -f mock::n8n::setup
export -f mock::n8n::setup_healthy_endpoints
export -f mock::n8n::setup_unhealthy_endpoints
export -f mock::n8n::setup_installing_endpoints
export -f mock::n8n::setup_stopped_endpoints
export -f mock::n8n::create_workflow
export -f mock::n8n::simulate_workflow_execution
export -f mock::n8n::get_community_nodes

echo "[N8N_MOCK] N8N mock implementation loaded"