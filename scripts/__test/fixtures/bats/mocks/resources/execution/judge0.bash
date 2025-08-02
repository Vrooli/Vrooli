#!/usr/bin/env bash
# Judge0 Resource Mock Implementation
# Provides realistic mock responses for Judge0 code execution service

# Prevent duplicate loading
if [[ "${JUDGE0_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export JUDGE0_MOCK_LOADED="true"

#######################################
# Setup Judge0 mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::judge0::setup() {
    local state="${1:-healthy}"
    
    # Configure Judge0-specific environment
    export JUDGE0_PORT="${JUDGE0_PORT:-2358}"
    export JUDGE0_BASE_URL="http://localhost:${JUDGE0_PORT}"
    export JUDGE0_CONTAINER_NAME="${TEST_NAMESPACE}_judge0"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$JUDGE0_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::judge0::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::judge0::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::judge0::setup_installing_endpoints
            ;;
        "stopped")
            mock::judge0::setup_stopped_endpoints
            ;;
        *)
            echo "[JUDGE0_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[JUDGE0_MOCK] Judge0 mock configured with state: $state"
}

#######################################
# Setup healthy Judge0 endpoints
#######################################
mock::judge0::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/health" \
        '{"status":"ok","version":"1.13.0"}'
    
    # Languages endpoint
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/languages" \
        '[
            {"id": 50, "name": "C (GCC 9.2.0)"},
            {"id": 54, "name": "C++ (GCC 9.2.0)"},
            {"id": 62, "name": "Java (OpenJDK 13.0.1)"},
            {"id": 71, "name": "Python (3.8.1)"},
            {"id": 63, "name": "JavaScript (Node.js 12.14.0)"},
            {"id": 78, "name": "Kotlin (1.3.70)"},
            {"id": 72, "name": "Ruby (2.7.0)"},
            {"id": 73, "name": "Rust (1.40.0)"}
        ]'
    
    # Submit submission endpoint
    mock::judge0::setup_submission_endpoint
    
    # Get submission endpoint
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/submissions/1" \
        '{
            "token": "test-submission-1",
            "source_code": "print(\"Hello, World!\")",
            "language_id": 71,
            "stdin": "",
            "expected_output": null,
            "stdout": "Hello, World!\n",
            "stderr": null,
            "status": {
                "id": 3,
                "description": "Accepted"
            },
            "created_at": "2024-01-15T12:00:00.000Z",
            "finished_at": "2024-01-15T12:00:01.234Z",
            "time": "0.02",
            "memory": 3584,
            "compile_output": null,
            "message": null,
            "wall_time": 0.1
        }'
    
    # Statistics endpoint
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/statistics" \
        '{
            "submissions": {
                "total": 12345,
                "today": 156,
                "this_month": 4567
            },
            "languages": {
                "most_popular": {"id": 71, "name": "Python (3.8.1)", "submissions": 3456},
                "total_supported": 8
            }
        }'
    
    # System info endpoint
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/system_info" \
        '{
            "version": "1.13.0",
            "hostname": "judge0-container",
            "architecture": "x86_64",
            "cpu_count": 4,
            "memory": "4096 MB",
            "kernel": "Linux 5.15.0"
        }'
}

#######################################
# Setup submission endpoint with different responses
#######################################
mock::judge0::setup_submission_endpoint() {
    # Successful submission
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/submissions" \
        '{"token":"test-submission-'$(date +%s)'"}' \
        "POST"
    
    # Multiple submissions
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/submissions/batch" \
        '{"tokens":["batch-1","batch-2","batch-3"]}' \
        "POST"
}

#######################################
# Setup unhealthy Judge0 endpoints
#######################################
mock::judge0::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/health" \
        '{"status":"error","error":"Execution environment unavailable"}' \
        "GET" \
        "503"
    
    # Submission endpoint returns error
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/submissions" \
        '{"error":"Service temporarily unavailable"}' \
        "POST" \
        "503"
}

#######################################
# Setup installing Judge0 endpoints
#######################################
mock::judge0::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/health" \
        '{"status":"installing","progress":90,"current_step":"Preparing execution environments"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/submissions" \
        '{"error":"Judge0 is still initializing"}' \
        "POST" \
        "503"
}

#######################################
# Setup stopped Judge0 endpoints
#######################################
mock::judge0::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$JUDGE0_BASE_URL"
}

#######################################
# Mock Judge0-specific operations
#######################################

# Mock code execution
mock::judge0::execute_code() {
    local language_id="$1"
    local source_code="$2"
    local stdin="${3:-}"
    
    echo '{
        "token": "'$(echo -n "$source_code" | md5sum | cut -c1-16)'",
        "source_code": "'"$(echo "$source_code" | sed 's/"/\\"/g')"'",
        "language_id": '$language_id',
        "stdin": "'$stdin'",
        "stdout": "Execution successful\n",
        "stderr": null,
        "status": {"id": 3, "description": "Accepted"},
        "time": "0.01",
        "memory": 2048
    }'
}

# Mock batch execution
mock::judge0::execute_batch() {
    local count="${1:-3}"
    local tokens=()
    
    for i in $(seq 1 "$count"); do
        tokens+=("\"batch-token-$i\"")
    done
    
    echo '{"tokens":['$(IFS=,; echo "${tokens[*]}")']}'
}

#######################################
# Export mock functions
#######################################
export -f mock::judge0::setup
export -f mock::judge0::setup_healthy_endpoints
export -f mock::judge0::setup_submission_endpoint
export -f mock::judge0::setup_unhealthy_endpoints
export -f mock::judge0::setup_installing_endpoints
export -f mock::judge0::setup_stopped_endpoints
export -f mock::judge0::execute_code
export -f mock::judge0::execute_batch

echo "[JUDGE0_MOCK] Judge0 mock implementation loaded"