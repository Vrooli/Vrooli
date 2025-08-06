#!/usr/bin/env bash
# Windmill Resource Mock Implementation
# Provides realistic mock responses for Windmill developer automation service

# Prevent duplicate loading
if [[ "${WINDMILL_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export WINDMILL_MOCK_LOADED="true"

#######################################
# Setup Windmill mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::windmill::setup() {
    local state="${1:-healthy}"
    
    # Configure Windmill-specific environment
    export WINDMILL_PORT="${WINDMILL_PORT:-8000}"
    export WINDMILL_BASE_URL="http://localhost:${WINDMILL_PORT}"
    export WINDMILL_CONTAINER_NAME="${TEST_NAMESPACE}_windmill"
    export WINDMILL_WORKSPACE="${WINDMILL_WORKSPACE:-starter}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$WINDMILL_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::windmill::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::windmill::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::windmill::setup_installing_endpoints
            ;;
        "stopped")
            mock::windmill::setup_stopped_endpoints
            ;;
        *)
            echo "[WINDMILL_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[WINDMILL_MOCK] Windmill mock configured with state: $state"
}

#######################################
# Setup healthy Windmill endpoints
#######################################
mock::windmill::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$WINDMILL_BASE_URL/api/version" \
        '{"version":"1.250.0","ee_license":false}'
    
    # Scripts endpoint
    mock::http::set_endpoint_response "$WINDMILL_BASE_URL/api/w/$WINDMILL_WORKSPACE/scripts/list" \
        '[
            {
                "path": "f/scripts/hello_world",
                "hash": "abc123def456",
                "summary": "Simple hello world script",
                "description": "Returns a greeting message",
                "created_by": "admin",
                "created_at": "2024-01-15T10:00:00Z",
                "language": "typescript",
                "kind": "script",
                "starred": false,
                "has_draft": false
            },
            {
                "path": "f/scripts/data_processor",
                "hash": "def456ghi789",
                "summary": "Process CSV data",
                "language": "python3",
                "kind": "script"
            }
        ]'
    
    # Single script endpoint
    mock::http::set_endpoint_response "$WINDMILL_BASE_URL/api/w/$WINDMILL_WORKSPACE/scripts/get/f/scripts/hello_world" \
        '{
            "path": "f/scripts/hello_world",
            "content": "export async function main(name: string = \"World\"): Promise<string> {\n  return `Hello ${name}!`;\n}",
            "language": "typescript",
            "schema": {
                "$schema": "https://json-schema.org/draft/2020-12/schema",
                "type": "object",
                "properties": {
                    "name": {
                        "type": "string",
                        "default": "World"
                    }
                }
            }
        }'
    
    # Flows endpoint
    mock::http::set_endpoint_response "$WINDMILL_BASE_URL/api/w/$WINDMILL_WORKSPACE/flows/list" \
        '[
            {
                "path": "f/flows/daily_report",
                "summary": "Generate daily reports",
                "description": "Collects data and sends daily report",
                "value": {
                    "modules": [
                        {
                            "id": "a",
                            "value": {
                                "type": "script",
                                "path": "f/scripts/collect_data"
                            }
                        },
                        {
                            "id": "b",
                            "value": {
                                "type": "script",
                                "path": "f/scripts/generate_report"
                            }
                        }
                    ]
                }
            }
        ]'
    
    # Jobs endpoint
    mock::http::set_endpoint_response "$WINDMILL_BASE_URL/api/w/$WINDMILL_WORKSPACE/jobs/list" \
        '{
            "jobs": [
                {
                    "id": "01234567-89ab-cdef-0123-456789abcdef",
                    "created_at": "2024-01-15T12:00:00Z",
                    "started_at": "2024-01-15T12:00:01Z",
                    "finished_at": "2024-01-15T12:00:05Z",
                    "success": true,
                    "script_path": "f/scripts/hello_world",
                    "args": {"name": "Test"},
                    "result": "Hello Test!",
                    "duration_ms": 4000
                }
            ]
        }'
    
    # Run script endpoint
    mock::http::set_endpoint_response "$WINDMILL_BASE_URL/api/w/$WINDMILL_WORKSPACE/jobs/run/f/scripts/hello_world" \
        '{"job_id":"'$(uuidgen || echo "job-$(date +%s)")'"}' \
        "POST"
    
    # Resources endpoint
    mock::http::set_endpoint_response "$WINDMILL_BASE_URL/api/w/$WINDMILL_WORKSPACE/resources/list" \
        '[
            {
                "path": "f/resources/postgres",
                "resource_type": "postgresql",
                "description": "Main PostgreSQL database"
            },
            {
                "path": "f/resources/s3_bucket",
                "resource_type": "s3",
                "description": "S3 storage bucket"
            }
        ]'
}

#######################################
# Setup unhealthy Windmill endpoints
#######################################
mock::windmill::setup_unhealthy_endpoints() {
    # Version endpoint returns error
    mock::http::set_endpoint_response "$WINDMILL_BASE_URL/api/version" \
        '{"error":"Database unavailable"}' \
        "GET" \
        "503"
    
    # API endpoints return errors
    mock::http::set_endpoint_response "$WINDMILL_BASE_URL/api/w/$WINDMILL_WORKSPACE/scripts/list" \
        '{"error":"Service temporarily unavailable"}' \
        "GET" \
        "503"
}

#######################################
# Setup installing Windmill endpoints
#######################################
mock::windmill::setup_installing_endpoints() {
    # Version endpoint returns installing status
    mock::http::set_endpoint_response "$WINDMILL_BASE_URL/api/version" \
        '{"status":"installing","progress":55,"current_step":"Setting up workers"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$WINDMILL_BASE_URL/api/w/$WINDMILL_WORKSPACE/scripts/list" \
        '{"error":"Windmill is still initializing"}' \
        "GET" \
        "503"
}

#######################################
# Setup stopped Windmill endpoints
#######################################
mock::windmill::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$WINDMILL_BASE_URL"
}

#######################################
# Mock Windmill-specific operations
#######################################

# Mock script execution
mock::windmill::run_script() {
    local script_path="$1"
    local args="${2:-{}}"
    
    local job_id=$(uuidgen || echo "job-$(date +%s)")
    echo '{
        "job_id": "'$job_id'",
        "created_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
        "status": "running",
        "script_path": "'$script_path'",
        "args": '$args'
    }'
}

# Mock job result
mock::windmill::get_job_result() {
    local job_id="$1"
    local success="${2:-true}"
    
    if [[ "$success" == "true" ]]; then
        echo '{
            "id": "'$job_id'",
            "success": true,
            "result": {"output": "Task completed successfully"},
            "duration_ms": 1234
        }'
    else
        echo '{
            "id": "'$job_id'",
            "success": false,
            "error": {"message": "Task failed", "name": "ExecutionError"},
            "duration_ms": 567
        }'
    fi
}

# Mock flow execution
mock::windmill::run_flow() {
    local flow_path="$1"
    
    echo '{
        "job_id": "'$(uuidgen || echo "flow-$(date +%s)")'",
        "flow_path": "'$flow_path'",
        "status": "running",
        "modules_to_run": 3,
        "modules_completed": 0
    }'
}

# Mock schedule creation
mock::windmill::create_schedule() {
    local script_path="$1"
    local cron="${2:-0 9 * * *}"
    
    echo '{
        "path": "f/schedules/'$(basename "$script_path")'_schedule",
        "script_path": "'$script_path'",
        "schedule": "'$cron'",
        "timezone": "UTC",
        "enabled": true,
        "created_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
    }'
}

#######################################
# Export mock functions
#######################################
export -f mock::windmill::setup
export -f mock::windmill::setup_healthy_endpoints
export -f mock::windmill::setup_unhealthy_endpoints
export -f mock::windmill::setup_installing_endpoints
export -f mock::windmill::setup_stopped_endpoints
export -f mock::windmill::run_script
export -f mock::windmill::get_job_result
export -f mock::windmill::run_flow
export -f mock::windmill::create_schedule

echo "[WINDMILL_MOCK] Windmill mock implementation loaded"