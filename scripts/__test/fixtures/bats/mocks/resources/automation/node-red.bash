#!/usr/bin/env bash
# Node-RED Resource Mock Implementation
# Provides realistic mock responses for Node-RED flow-based programming service

# Prevent duplicate loading
if [[ "${NODE_RED_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export NODE_RED_MOCK_LOADED="true"

#######################################
# Setup Node-RED mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::node-red::setup() {
    local state="${1:-healthy}"
    
    # Configure Node-RED-specific environment
    export NODE_RED_PORT="${NODE_RED_PORT:-1880}"
    export NODE_RED_BASE_URL="http://localhost:${NODE_RED_PORT}"
    export NODE_RED_CONTAINER_NAME="${TEST_NAMESPACE}_node-red"
    export NODE_RED_FLOW_FILE="${NODE_RED_FLOW_FILE:-flows.json}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$NODE_RED_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::node-red::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::node-red::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::node-red::setup_installing_endpoints
            ;;
        "stopped")
            mock::node-red::setup_stopped_endpoints
            ;;
        *)
            echo "[NODE_RED_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[NODE_RED_MOCK] Node-RED mock configured with state: $state"
}

#######################################
# Setup healthy Node-RED endpoints
#######################################
mock::node-red::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$NODE_RED_BASE_URL/health" \
        '{"status":"ok","version":"3.1.0"}'
    
    # Settings endpoint
    mock::http::set_endpoint_response "$NODE_RED_BASE_URL/settings" \
        '{
            "httpNodeRoot": "/",
            "version": "3.1.0",
            "user": {
                "username": "admin"
            },
            "context": {
                "default": "memory",
                "stores": ["memory", "file"]
            },
            "paletteCategories": [
                "subflows",
                "common",
                "function",
                "network",
                "sequence",
                "parser",
                "storage"
            ],
            "flowFilePretty": true,
            "editorTheme": {
                "projects": {
                    "enabled": true
                }
            }
        }'
    
    # Flows endpoint (GET)
    mock::http::set_endpoint_response "$NODE_RED_BASE_URL/flows" \
        '[
            {
                "id": "f1",
                "type": "tab",
                "label": "Flow 1",
                "disabled": false,
                "info": ""
            },
            {
                "id": "n1",
                "type": "inject",
                "z": "f1",
                "name": "",
                "props": [{"p": "payload"}, {"p": "topic", "vt": "str"}],
                "repeat": "",
                "crontab": "",
                "once": false,
                "x": 150,
                "y": 100,
                "wires": [["n2"]]
            },
            {
                "id": "n2",
                "type": "debug",
                "z": "f1",
                "name": "",
                "active": true,
                "tosidebar": true,
                "console": false,
                "x": 350,
                "y": 100,
                "wires": []
            }
        ]'
    
    # Flows endpoint (POST)
    mock::http::set_endpoint_response "$NODE_RED_BASE_URL/flows" \
        '{"rev":"2-abc123"}' \
        "POST"
    
    # Nodes endpoint
    mock::http::set_endpoint_response "$NODE_RED_BASE_URL/nodes" \
        '[
            {
                "id": "node-red/inject",
                "name": "inject",
                "types": ["inject"],
                "enabled": true,
                "module": "node-red",
                "version": "3.1.0"
            },
            {
                "id": "node-red/debug",
                "name": "debug",
                "types": ["debug"],
                "enabled": true,
                "module": "node-red",
                "version": "3.1.0"
            },
            {
                "id": "node-red/http",
                "name": "http",
                "types": ["http in", "http response"],
                "enabled": true,
                "module": "node-red",
                "version": "3.1.0"
            }
        ]'
    
    # Library endpoint
    mock::http::set_endpoint_response "$NODE_RED_BASE_URL/library/flows" \
        '[
            {
                "name": "Example Flow",
                "description": "A simple example flow"
            }
        ]'
}

#######################################
# Setup unhealthy Node-RED endpoints
#######################################
mock::node-red::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$NODE_RED_BASE_URL/health" \
        '{"status":"unhealthy","error":"Flow runtime error"}' \
        "GET" \
        "503"
    
    # Settings endpoint returns limited info
    mock::http::set_endpoint_response "$NODE_RED_BASE_URL/settings" \
        '{"error":"Runtime not ready"}' \
        "GET" \
        "503"
}

#######################################
# Setup installing Node-RED endpoints
#######################################
mock::node-red::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$NODE_RED_BASE_URL/health" \
        '{"status":"installing","progress":65,"current_step":"Loading nodes"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$NODE_RED_BASE_URL/flows" \
        '{"error":"Node-RED is still initializing"}' \
        "GET" \
        "503"
}

#######################################
# Setup stopped Node-RED endpoints
#######################################
mock::node-red::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$NODE_RED_BASE_URL"
}

#######################################
# Mock Node-RED-specific operations
#######################################

# Mock flow deployment
mock::node-red::deploy_flow() {
    local flow_json="$1"
    local deploy_type="${2:-full}"
    
    echo '{
        "rev": "'$(date +%s)'-'$(echo "$flow_json" | md5sum | cut -c1-6)'",
        "flows": '$(echo "$flow_json" | wc -l)',
        "deploy_type": "'$deploy_type'"
    }'
}

# Mock node installation
mock::node-red::install_node() {
    local node_name="$1"
    
    echo '{
        "name": "'$node_name'",
        "version": "1.0.0",
        "nodes": [
            {
                "id": "'$node_name'/custom-node",
                "name": "custom-node",
                "types": ["custom-input", "custom-output"],
                "enabled": true
            }
        ]
    }'
}

# Mock debug message
mock::node-red::simulate_debug_message() {
    local payload="$1"
    local topic="${2:-test}"
    
    echo '{
        "id": "'$(date +%s%N)'",
        "z": "f1",
        "name": "debug",
        "topic": "'$topic'",
        "property": "payload",
        "propertyType": "msg",
        "_msgid": "'$(uuidgen || echo "msg-$(date +%s)")'",
        "payload": '"$payload"',
        "timestamp": '$(date +%s)'
    }'
}

# Mock flow execution stats
mock::node-red::get_flow_stats() {
    echo '{
        "flows": {
            "f1": {
                "messages_processed": 152,
                "errors": 0,
                "last_message": "'$(date -u +%Y-%m-%dT%H:%M:%S.000Z)'"
            }
        },
        "nodes": {
            "n1": {"count": 76},
            "n2": {"count": 76}
        }
    }'
}

#######################################
# Export mock functions
#######################################
export -f mock::node-red::setup
export -f mock::node-red::setup_healthy_endpoints
export -f mock::node-red::setup_unhealthy_endpoints
export -f mock::node-red::setup_installing_endpoints
export -f mock::node-red::setup_stopped_endpoints
export -f mock::node-red::deploy_flow
export -f mock::node-red::install_node
export -f mock::node-red::simulate_debug_message
export -f mock::node-red::get_flow_stats

echo "[NODE_RED_MOCK] Node-RED mock implementation loaded"