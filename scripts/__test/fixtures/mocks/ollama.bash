#!/usr/bin/env bash
# Ollama Resource Mock Implementation
# Provides realistic mock responses for Ollama LLM service

# Prevent duplicate loading
if [[ "${OLLAMA_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export OLLAMA_MOCK_LOADED="true"

#######################################
# Setup Ollama mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::ollama::setup() {
    local state="${1:-healthy}"
    
    # Configure Ollama-specific environment
    export OLLAMA_PORT="${OLLAMA_PORT:-11434}"
    export OLLAMA_BASE_URL="http://localhost:${OLLAMA_PORT}"
    export OLLAMA_CONTAINER_NAME="${TEST_NAMESPACE}_ollama"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$OLLAMA_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::ollama::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::ollama::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::ollama::setup_installing_endpoints
            ;;
        "stopped")
            mock::ollama::setup_stopped_endpoints
            ;;
        *)
            echo "[OLLAMA_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[OLLAMA_MOCK] Ollama mock configured with state: $state"
}

#######################################
# Setup healthy Ollama endpoints
#######################################
mock::ollama::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/health" \
        '{"status":"ok","version":"0.1.38"}'
    
    # Tags/models endpoint
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" \
        '{
            "models": [
                {
                    "name": "llama3.1:8b",
                    "modified_at": "2024-01-15T10:30:00Z",
                    "size": 4900000000,
                    "digest": "sha256:abc123",
                    "details": {
                        "format": "gguf",
                        "family": "llama",
                        "parameter_size": "8B"
                    }
                },
                {
                    "name": "mistral:7b",
                    "modified_at": "2024-01-14T09:00:00Z",
                    "size": 4100000000,
                    "digest": "sha256:def456",
                    "details": {
                        "format": "gguf",
                        "family": "mistral",
                        "parameter_size": "7B"
                    }
                }
            ]
        }'
    
    # Generate endpoint (non-streaming)
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/generate" \
        '{
            "model": "llama3.1:8b",
            "created_at": "2024-01-15T10:35:00Z",
            "response": "Hello! I am a helpful AI assistant. How can I help you today?",
            "done": true,
            "context": [1, 2, 3, 4, 5],
            "total_duration": 1234567890,
            "load_duration": 234567890,
            "prompt_eval_count": 10,
            "prompt_eval_duration": 123456789,
            "eval_count": 25,
            "eval_duration": 987654321
        }' \
        "POST"
    
    # Pull endpoint
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/pull" \
        '{"status":"success","digest":"sha256:abc123"}' \
        "POST"
    
    # Show endpoint
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/show" \
        '{
            "modelfile": "FROM llama3.1:8b\\nPARAMETER temperature 0.7",
            "parameters": "temperature 0.7",
            "template": "{{ .Prompt }}",
            "details": {
                "format": "gguf",
                "family": "llama",
                "parameter_size": "8B"
            }
        }' \
        "POST"
}

#######################################
# Setup unhealthy Ollama endpoints
#######################################
mock::ollama::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/health" \
        '{"status":"unhealthy","error":"Service unavailable"}' \
        "GET" \
        "503"
    
    # Other endpoints return errors
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" \
        '{"error":"Service temporarily unavailable"}' \
        "GET" \
        "503"
}

#######################################
# Setup installing Ollama endpoints
#######################################
mock::ollama::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/health" \
        '{"status":"installing","progress":45}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" \
        '{"error":"Service is still initializing"}' \
        "GET" \
        "503"
}

#######################################
# Setup stopped Ollama endpoints
#######################################
mock::ollama::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$OLLAMA_BASE_URL"
}

#######################################
# Mock Ollama-specific operations
#######################################

# Mock model download progress
mock::ollama::simulate_model_download() {
    local model_name="$1"
    local progress_steps=(10 25 50 75 90 100)
    
    for progress in "${progress_steps[@]}"; do
        mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/pull" \
            "{\"status\":\"downloading\",\"digest\":\"sha256:abc123\",\"completed\":$progress,\"total\":100}" \
            "POST"
        sleep 0.1  # Simulate download time
    done
    
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/pull" \
        '{"status":"success","digest":"sha256:abc123"}' \
        "POST"
}

# Mock streaming response
mock::ollama::simulate_streaming_response() {
    local prompt="$1"
    local chunks=(
        '{"model":"llama3.1:8b","created_at":"2024-01-15T10:35:00Z","response":"Hello","done":false}'
        '{"model":"llama3.1:8b","created_at":"2024-01-15T10:35:01Z","response":" there","done":false}'
        '{"model":"llama3.1:8b","created_at":"2024-01-15T10:35:02Z","response":"!","done":false}'
        '{"model":"llama3.1:8b","created_at":"2024-01-15T10:35:03Z","response":"","done":true,"context":[1,2,3]}'
    )
    
    # In real implementation, this would stream responses
    # For testing, we just return the final result
    echo "${chunks[-1]}"
}

#######################################
# Export mock functions
#######################################
export -f mock::ollama::setup
export -f mock::ollama::setup_healthy_endpoints
export -f mock::ollama::setup_unhealthy_endpoints
export -f mock::ollama::setup_installing_endpoints
export -f mock::ollama::setup_stopped_endpoints
export -f mock::ollama::simulate_model_download
export -f mock::ollama::simulate_streaming_response

echo "[OLLAMA_MOCK] Ollama mock implementation loaded"