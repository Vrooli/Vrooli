#!/usr/bin/env bash
# ComfyUI Resource Mock Implementation
# Provides realistic mock responses for ComfyUI image generation service

# Prevent duplicate loading
if [[ "${COMFYUI_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export COMFYUI_MOCK_LOADED="true"

#######################################
# Setup ComfyUI mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::comfyui::setup() {
    local state="${1:-healthy}"
    
    # Configure ComfyUI-specific environment
    export COMFYUI_PORT="${COMFYUI_PORT:-8188}"
    export COMFYUI_BASE_URL="http://localhost:${COMFYUI_PORT}"
    export COMFYUI_CONTAINER_NAME="${TEST_NAMESPACE}_comfyui"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$COMFYUI_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::comfyui::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::comfyui::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::comfyui::setup_installing_endpoints
            ;;
        "stopped")
            mock::comfyui::setup_stopped_endpoints
            ;;
        *)
            echo "[COMFYUI_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[COMFYUI_MOCK] ComfyUI mock configured with state: $state"
}

#######################################
# Setup healthy ComfyUI endpoints
#######################################
mock::comfyui::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$COMFYUI_BASE_URL/health" \
        '{"status":"ok","version":"1.0.0","cuda_available":true}'
    
    # System info endpoint
    mock::http::set_endpoint_response "$COMFYUI_BASE_URL/system_stats" \
        '{
            "system": {
                "os": "linux",
                "python_version": "3.10.0",
                "embedded_python": false
            },
            "devices": [
                {
                    "name": "cuda:0",
                    "type": "cuda",
                    "vram_total": 8589934592,
                    "vram_free": 7516192768
                }
            ]
        }'
    
    # Queue prompt endpoint
    mock::http::set_endpoint_response "$COMFYUI_BASE_URL/api/prompt" \
        '{
            "prompt_id": "f47b5c3e-4d2e-4a3b-8c1d-9e0f1a2b3c4d",
            "number": 1,
            "node_errors": {}
        }' \
        "POST"
    
    # Queue status endpoint
    mock::http::set_endpoint_response "$COMFYUI_BASE_URL/api/queue" \
        '{
            "queue_running": [],
            "queue_pending": [],
            "queue_completed": [
                {
                    "prompt_id": "f47b5c3e-4d2e-4a3b-8c1d-9e0f1a2b3c4d",
                    "outputs": {
                        "images": [
                            {
                                "filename": "ComfyUI_00001_.png",
                                "subfolder": "",
                                "type": "output"
                            }
                        ]
                    }
                }
            ]
        }'
    
    # History endpoint
    mock::http::set_endpoint_response "$COMFYUI_BASE_URL/api/history" \
        '{
            "f47b5c3e-4d2e-4a3b-8c1d-9e0f1a2b3c4d": {
                "prompt": {},
                "outputs": {
                    "9": {
                        "images": [
                            {
                                "filename": "ComfyUI_00001_.png",
                                "subfolder": "",
                                "type": "output"
                            }
                        ]
                    }
                },
                "status": {
                    "status_str": "success",
                    "completed": true
                }
            }
        }'
    
    # Models endpoint
    mock::http::set_endpoint_response "$COMFYUI_BASE_URL/api/models" \
        '{
            "checkpoints": ["sd_xl_base_1.0.safetensors", "sd_xl_refiner_1.0.safetensors"],
            "vae": ["sdxl_vae.safetensors"],
            "loras": ["detail_enhancer.safetensors"],
            "embeddings": ["easynegative.pt"]
        }'
}

#######################################
# Setup unhealthy ComfyUI endpoints
#######################################
mock::comfyui::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$COMFYUI_BASE_URL/health" \
        '{"status":"unhealthy","error":"GPU memory exhausted"}' \
        "GET" \
        "503"
    
    # Queue endpoint returns error
    mock::http::set_endpoint_response "$COMFYUI_BASE_URL/api/prompt" \
        '{"error":"Cannot process prompt","details":"No available GPU memory"}' \
        "POST" \
        "503"
}

#######################################
# Setup installing ComfyUI endpoints
#######################################
mock::comfyui::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$COMFYUI_BASE_URL/health" \
        '{"status":"installing","progress":35,"current_step":"Loading models"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$COMFYUI_BASE_URL/api/prompt" \
        '{"error":"Service is still initializing","eta_seconds":300}' \
        "POST" \
        "503"
}

#######################################
# Setup stopped ComfyUI endpoints
#######################################
mock::comfyui::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$COMFYUI_BASE_URL"
}

#######################################
# Mock ComfyUI-specific operations
#######################################

# Mock workflow execution
mock::comfyui::simulate_workflow_execution() {
    local workflow_complexity="${1:-simple}"
    local prompt_id="$(uuidgen || echo "mock-$(date +%s)")"
    
    case "$workflow_complexity" in
        "simple")
            echo "{\"prompt_id\":\"$prompt_id\",\"execution_time\":5.2,\"nodes_executed\":5}"
            ;;
        "complex")
            echo "{\"prompt_id\":\"$prompt_id\",\"execution_time\":25.8,\"nodes_executed\":15}"
            ;;
        "heavy")
            echo "{\"prompt_id\":\"$prompt_id\",\"execution_time\":120.5,\"nodes_executed\":30}"
            ;;
    esac
}

# Mock image generation progress
mock::comfyui::simulate_generation_progress() {
    local prompt_id="$1"
    local steps=(
        '{"value":0,"max":20,"prompt_id":"'$prompt_id'","node":"3"}'
        '{"value":5,"max":20,"prompt_id":"'$prompt_id'","node":"3"}'
        '{"value":10,"max":20,"prompt_id":"'$prompt_id'","node":"3"}'
        '{"value":15,"max":20,"prompt_id":"'$prompt_id'","node":"3"}'
        '{"value":20,"max":20,"prompt_id":"'$prompt_id'","node":"3"}'
    )
    
    # In real implementation, this would be websocket events
    # For testing, we return the final state
    echo "${steps[-1]}"
}

#######################################
# Export mock functions
#######################################
export -f mock::comfyui::setup
export -f mock::comfyui::setup_healthy_endpoints
export -f mock::comfyui::setup_unhealthy_endpoints
export -f mock::comfyui::setup_installing_endpoints
export -f mock::comfyui::setup_stopped_endpoints
export -f mock::comfyui::simulate_workflow_execution
export -f mock::comfyui::simulate_generation_progress

echo "[COMFYUI_MOCK] ComfyUI mock implementation loaded"