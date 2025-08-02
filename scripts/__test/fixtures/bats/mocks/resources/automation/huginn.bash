#!/usr/bin/env bash
# Huginn Resource Mock Implementation
# Provides realistic mock responses for Huginn agent automation service

# Prevent duplicate loading
if [[ "${HUGINN_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export HUGINN_MOCK_LOADED="true"

#######################################
# Setup Huginn mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::huginn::setup() {
    local state="${1:-healthy}"
    
    # Configure Huginn-specific environment
    export HUGINN_PORT="${HUGINN_PORT:-3000}"
    export HUGINN_BASE_URL="http://localhost:${HUGINN_PORT}"
    export HUGINN_CONTAINER_NAME="${TEST_NAMESPACE}_huginn"
    export HUGINN_USERNAME="${HUGINN_USERNAME:-admin}"
    export HUGINN_PASSWORD="${HUGINN_PASSWORD:-password}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$HUGINN_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::huginn::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::huginn::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::huginn::setup_installing_endpoints
            ;;
        "stopped")
            mock::huginn::setup_stopped_endpoints
            ;;
        *)
            echo "[HUGINN_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[HUGINN_MOCK] Huginn mock configured with state: $state"
}

#######################################
# Setup healthy Huginn endpoints
#######################################
mock::huginn::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$HUGINN_BASE_URL/health" \
        '{"status":"ok","version":"2023.03.20"}'
    
    # Agents endpoint
    mock::http::set_endpoint_response "$HUGINN_BASE_URL/agents" \
        '[
            {
                "id": 1,
                "name": "Weather Agent",
                "type": "Agents::WeatherAgent",
                "schedule": "every_1h",
                "disabled": false,
                "guid": "weather-agent-001",
                "options": {
                    "location": "San Francisco",
                    "api_key": "****"
                },
                "created_at": "2024-01-15T10:00:00Z",
                "updated_at": "2024-01-15T10:00:00Z",
                "last_checked_at": "2024-01-15T15:00:00Z",
                "last_check_status": "ok"
            },
            {
                "id": 2,
                "name": "Email Digest",
                "type": "Agents::EmailDigestAgent",
                "schedule": "9am",
                "disabled": false,
                "guid": "email-digest-001",
                "options": {
                    "subject": "Daily Digest",
                    "expected_receive_period_in_days": 1
                }
            }
        ]'
    
    # Single agent endpoint
    mock::http::set_endpoint_response "$HUGINN_BASE_URL/agents/1" \
        '{
            "id": 1,
            "name": "Weather Agent",
            "type": "Agents::WeatherAgent",
            "memory": {
                "last_weather": {
                    "temperature": 68,
                    "conditions": "sunny"
                }
            },
            "options": {
                "location": "San Francisco"
            }
        }'
    
    # Events endpoint
    mock::http::set_endpoint_response "$HUGINN_BASE_URL/events" \
        '[
            {
                "id": 100,
                "agent_id": 1,
                "created_at": "2024-01-15T15:00:00Z",
                "expires_at": null,
                "payload": {
                    "temperature": 68,
                    "conditions": "sunny",
                    "humidity": 65
                }
            },
            {
                "id": 101,
                "agent_id": 2,
                "created_at": "2024-01-15T09:00:00Z",
                "payload": {
                    "subject": "Daily Digest",
                    "items": ["Weather update", "News summary"]
                }
            }
        ]'
    
    # Scenarios endpoint
    mock::http::set_endpoint_response "$HUGINN_BASE_URL/scenarios" \
        '[
            {
                "id": 1,
                "name": "Weather Monitoring",
                "description": "Monitor weather and send alerts",
                "agents": [1, 2],
                "enabled": true
            }
        ]'
}

#######################################
# Setup unhealthy Huginn endpoints
#######################################
mock::huginn::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$HUGINN_BASE_URL/health" \
        '{"status":"unhealthy","error":"Database connection failed"}' \
        "GET" \
        "503"
    
    # API endpoints return errors
    mock::http::set_endpoint_response "$HUGINN_BASE_URL/agents" \
        '{"error":"Service unavailable"}' \
        "GET" \
        "503"
}

#######################################
# Setup installing Huginn endpoints
#######################################
mock::huginn::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$HUGINN_BASE_URL/health" \
        '{"status":"installing","progress":40,"current_step":"Running database migrations"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$HUGINN_BASE_URL/agents" \
        '{"error":"Huginn is still initializing"}' \
        "GET" \
        "503"
}

#######################################
# Setup stopped Huginn endpoints
#######################################
mock::huginn::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$HUGINN_BASE_URL"
}

#######################################
# Mock Huginn-specific operations
#######################################

# Mock agent creation
mock::huginn::create_agent() {
    local agent_name="$1"
    local agent_type="${2:-Agents::WebsiteAgent}"
    
    echo '{
        "id": '$(date +%s)',
        "name": "'$agent_name'",
        "type": "'$agent_type'",
        "guid": "'$(uuidgen || echo "$agent_name-$(date +%s)")'",
        "options": {},
        "schedule": "never",
        "disabled": false,
        "created_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
    }'
}

# Mock agent run
mock::huginn::run_agent() {
    local agent_id="$1"
    
    echo '{
        "status": "success",
        "agent_id": '$agent_id',
        "events_created": 1,
        "memory_updated": true,
        "runtime_seconds": 0.5,
        "logs": [
            "Agent started",
            "Fetching data",
            "Created 1 event",
            "Agent completed"
        ]
    }'
}

# Mock event propagation
mock::huginn::simulate_event_flow() {
    local source_agent="$1"
    local target_agents="${2:-2,3}"
    
    echo '{
        "source_agent": '$source_agent',
        "event_id": '$(date +%s)',
        "propagated_to": ['$target_agents'],
        "propagation_time": 0.1,
        "status": "completed"
    }'
}

#######################################
# Export mock functions
#######################################
export -f mock::huginn::setup
export -f mock::huginn::setup_healthy_endpoints
export -f mock::huginn::setup_unhealthy_endpoints
export -f mock::huginn::setup_installing_endpoints
export -f mock::huginn::setup_stopped_endpoints
export -f mock::huginn::create_agent
export -f mock::huginn::run_agent
export -f mock::huginn::simulate_event_flow

echo "[HUGINN_MOCK] Huginn mock implementation loaded"