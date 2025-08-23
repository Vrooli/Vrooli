#!/usr/bin/env bash
# ComfyUI Resource Mock Implementation
# 
# Provides comprehensive mock for ComfyUI operations including:
# - CLI command interception (manage.sh functions)
# - Docker container management simulation
# - HTTP API endpoint mocking
# - GPU detection and NVIDIA validation
# - Model download and management
# - Workflow execution simulation
# - File system operations
#
# This mock follows the same standards as other updated mocks with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions

# Prevent duplicate loading
[[ -n "${COMFYUI_MOCK_LOADED:-}" ]] && return 0
declare -g COMFYUI_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh" &>/dev/null
[[ -f "$MOCK_DIR/http.sh" ]] && source "$MOCK_DIR/http.sh" &>/dev/null
[[ -f "$MOCK_DIR/docker.sh" ]] && source "$MOCK_DIR/docker.sh" &>/dev/null

# Global configuration
declare -g COMFYUI_MOCK_STATE_DIR="${COMFYUI_MOCK_STATE_DIR:-/tmp/comfyui-mock-state}"
declare -g COMFYUI_MOCK_DEBUG="${COMFYUI_MOCK_DEBUG:-}"

# Global state arrays
declare -gA COMFYUI_MOCK_CONFIG=(           # ComfyUI configuration
    [host]="localhost"
    [port]="8188"
    [base_url]="http://localhost:8188"
    [container_name]="comfyui_test"
    [version]="1.0.0"
    [connected]="true"
    [error_mode]=""
    [gpu_available]="true"
    [gpu_type]="cuda"
    [nvidia_runtime]="true"
    [models_downloaded]="true"
    [disk_space_gb]="50"
)

declare -gA COMFYUI_MOCK_WORKFLOWS=()       # Workflow storage: name -> workflow_json
declare -gA COMFYUI_MOCK_EXECUTIONS=()      # Execution storage: prompt_id -> execution_data
declare -gA COMFYUI_MOCK_MODELS=()          # Model storage: model_name -> model_info
declare -gA COMFYUI_MOCK_QUEUE=()           # Queue state: prompt_id -> queue_status
declare -gA COMFYUI_MOCK_CLI_HISTORY=()     # CLI command history tracking

# Initialize state directory
mkdir -p "$COMFYUI_MOCK_STATE_DIR"

# State persistence functions
mock::comfyui::save_state() {
    local state_file="$COMFYUI_MOCK_STATE_DIR/comfyui-state.sh"
    {
        echo "# ComfyUI mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p COMFYUI_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA COMFYUI_MOCK_CONFIG=()"
        declare -p COMFYUI_MOCK_WORKFLOWS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA COMFYUI_MOCK_WORKFLOWS=()"
        declare -p COMFYUI_MOCK_EXECUTIONS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA COMFYUI_MOCK_EXECUTIONS=()"
        declare -p COMFYUI_MOCK_MODELS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA COMFYUI_MOCK_MODELS=()"
        declare -p COMFYUI_MOCK_QUEUE 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA COMFYUI_MOCK_QUEUE=()"
        declare -p COMFYUI_MOCK_CLI_HISTORY 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA COMFYUI_MOCK_CLI_HISTORY=()"
    } > "$state_file"
    
    if command -v mock::log_state &>/dev/null; then
        mock::log_state "comfyui" "Saved ComfyUI state to $state_file"
    fi
}

mock::comfyui::load_state() {
    local state_file="$COMFYUI_MOCK_STATE_DIR/comfyui-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        if command -v mock::log_state &>/dev/null; then
            mock::log_state "comfyui" "Loaded ComfyUI state from $state_file"
        fi
    fi
}

# Automatically load state when sourced
mock::comfyui::load_state

#######################################
# Setup ComfyUI mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::comfyui::setup() {
    local state="${1:-healthy}"
    
    # Configure ComfyUI-specific environment
    COMFYUI_MOCK_CONFIG[port]="${COMFYUI_PORT:-8188}"
    COMFYUI_MOCK_CONFIG[base_url]="http://localhost:${COMFYUI_MOCK_CONFIG[port]}"
    COMFYUI_MOCK_CONFIG[container_name]="${TEST_NAMESPACE:-test}_comfyui"
    
    # Set up Docker mock state if available
    if command -v mock::docker::set_container_state &>/dev/null; then
        mock::docker::set_container_state "${COMFYUI_MOCK_CONFIG[container_name]}" "$state"
    fi
    
    # Update config based on state
    case "$state" in
        "healthy")
            COMFYUI_MOCK_CONFIG[connected]="true"
            COMFYUI_MOCK_CONFIG[error_mode]=""
            mock::comfyui::setup_healthy_endpoints
            ;;
        "unhealthy")
            COMFYUI_MOCK_CONFIG[connected]="true"
            COMFYUI_MOCK_CONFIG[error_mode]="unhealthy"
            mock::comfyui::setup_unhealthy_endpoints
            ;;
        "installing")
            COMFYUI_MOCK_CONFIG[connected]="false"
            COMFYUI_MOCK_CONFIG[error_mode]="installing"
            mock::comfyui::setup_installing_endpoints
            ;;
        "stopped")
            COMFYUI_MOCK_CONFIG[connected]="false"
            COMFYUI_MOCK_CONFIG[error_mode]="stopped"
            mock::comfyui::setup_stopped_endpoints
            ;;
        *)
            echo "[COMFYUI_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    # Save state
    mock::comfyui::save_state
    
    echo "[COMFYUI_MOCK] ComfyUI mock configured with state: $state"
}

#######################################
# Setup healthy ComfyUI endpoints
#######################################
mock::comfyui::setup_healthy_endpoints() {
    local base_url="${COMFYUI_MOCK_CONFIG[base_url]}"
    
    if command -v mock::http::set_endpoint_response &>/dev/null; then
        # Health endpoint
        mock::http::set_endpoint_response "$base_url/health" \
            '{"status":"ok","version":"1.0.0","cuda_available":true}'
        
        # System info endpoint
        mock::http::set_endpoint_response "$base_url/system_stats" \
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
        mock::http::set_endpoint_response "$base_url/prompt" \
            '{
                "prompt_id": "f47b5c3e-4d2e-4a3b-8c1d-9e0f1a2b3c4d",
                "number": 1,
                "node_errors": {}
            }' \
            "POST"
        
        # Queue status endpoint
        mock::http::set_endpoint_response "$base_url/queue" \
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
        mock::http::set_endpoint_response "$base_url/history" \
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
        mock::http::set_endpoint_response "$base_url/object_info" \
            '{
                "checkpoints": ["sd_xl_base_1.0.safetensors", "sd_xl_refiner_1.0.safetensors"],
                "vae": ["sdxl_vae.safetensors"],
                "loras": ["detail_enhancer.safetensors"],
                "embeddings": ["easynegative.pt"]
            }'
    fi
}

#######################################
# Setup unhealthy ComfyUI endpoints
#######################################
mock::comfyui::setup_unhealthy_endpoints() {
    local base_url="${COMFYUI_MOCK_CONFIG[base_url]}"
    
    if command -v mock::http::set_endpoint_response &>/dev/null; then
        # Health endpoint returns error
        mock::http::set_endpoint_response "$base_url/health" \
            '{"status":"unhealthy","error":"GPU memory exhausted"}' \
            "GET" \
            "503"
        
        # Queue endpoint returns error
        mock::http::set_endpoint_response "$base_url/prompt" \
            '{"error":"Cannot process prompt","details":"No available GPU memory"}' \
            "POST" \
            "503"
    fi
}

#######################################
# Setup installing ComfyUI endpoints
#######################################
mock::comfyui::setup_installing_endpoints() {
    local base_url="${COMFYUI_MOCK_CONFIG[base_url]}"
    
    if command -v mock::http::set_endpoint_response &>/dev/null; then
        # Health endpoint returns installing status
        mock::http::set_endpoint_response "$base_url/health" \
            '{"status":"installing","progress":35,"current_step":"Loading models"}'
        
        # Other endpoints return not ready
        mock::http::set_endpoint_response "$base_url/prompt" \
            '{"error":"Service is still initializing","eta_seconds":300}' \
            "POST" \
            "503"
    fi
}

#######################################
# Setup stopped ComfyUI endpoints
#######################################
mock::comfyui::setup_stopped_endpoints() {
    local base_url="${COMFYUI_MOCK_CONFIG[base_url]}"
    
    if command -v mock::http::set_endpoint_unreachable &>/dev/null; then
        # All endpoints fail to connect
        mock::http::set_endpoint_unreachable "$base_url"
    fi
}

#######################################
# Mock ComfyUI-specific operations
#######################################

# Mock workflow execution
mock::comfyui::simulate_workflow_execution() {
    local workflow_complexity="${1:-simple}"
    local prompt_id="mock-$(date +%s)"
    
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

# Mock workflow execution function
mock::comfyui::execute_workflow() {
    local workflow_file="$1"
    local complexity="${2:-simple}"
    
    # Check if workflow file is specified
    if [[ -z "$workflow_file" ]]; then
        echo "Error: No workflow file specified" >&2
        return 1
    fi
    
    # Check if ComfyUI is running
    if [[ "${COMFYUI_MOCK_CONFIG[connected]}" != "true" ]]; then
        echo "Error: ComfyUI is not running" >&2
        return 1
    fi
    
    local prompt_id="mock-prompt-$(date +%s)"
    
    echo "Executing workflow: $workflow_file"
    echo "Prompt ID: $prompt_id"
    
    # Simulate progress based on complexity
    case "$complexity" in
        "simple")
            echo "Progress: 50%"
            sleep 0.1
            echo "Progress: 100%"
            ;;
        "complex")
            echo "Progress: 25%"
            sleep 0.1
            echo "Progress: 75%"
            sleep 0.1
            echo "Progress: 100%"
            ;;
        *)
            echo "Progress: 100%"
            ;;
    esac
    
    echo "Workflow execution completed successfully"
    echo "Output: ComfyUI_00001_.png"
    
    return 0
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
export -f mock::comfyui::execute_workflow

echo "[COMFYUI_MOCK] ComfyUI mock implementation loaded" >/dev/null
# Mock nvidia-smi command for GPU detection
nvidia-smi() {
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "nvidia-smi" "$@"
    fi
    
    case "${COMFYUI_MOCK_CONFIG[gpu_type]}" in
        "cuda")
            if [[ "${COMFYUI_MOCK_CONFIG[gpu_available]}" == "true" ]]; then
                cat <<NVIDIA_EOF
+-----------------------------------------------------------------------------+
| NVIDIA-SMI 535.104.05             Driver Version: 535.104.05   CUDA Version: 12.2     |
|-------------------------------+----------------------+----------------------+
| GPU  Name                 Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |
| Fan  Temp  Perf          Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |
|                                      |                      |               MIG M. |
|===============================+======================+======================|
|   0  NVIDIA GeForce RTX 4090    Off  | 00000000:01:00.0  On |                  Off |
| 30%   45C    P2               150W / 450W |   2048MiB / 24564MiB |      5%      Default |
|                                      |                      |                  N/A |
+-------------------------------+----------------------+----------------------+
NVIDIA_EOF
                return 0
            else
                echo "NVIDIA-SMI has failed because it couldn't communicate with the NVIDIA driver." >&2
                return 1
            fi
            ;;
        "none")
            echo "nvidia-smi: command not found" >&2
            return 127
            ;;
        *)
            echo "nvidia-smi: unknown error" >&2
            return 1
            ;;
    esac
}

# Mock curl for model downloads and API calls
curl() {
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "curl" "$@"
    fi
    
    # Handle model downloads (simulate)
    local url="$*"
    if [[ "$url" == *".safetensors"* ]] || [[ "$url" == *".ckpt"* ]] || [[ "$url" == *".pt"* ]]; then
        if [[ "${COMFYUI_MOCK_CONFIG[models_downloaded]}" == "true" ]]; then
            echo "Mock model file content"
            return 0
        else
            echo "curl: (7) Failed to connect" >&2
            return 7
        fi
    fi
    
    # Default response for other URLs
    echo "curl: mocked response"
    return 0
}

# Test helper functions
mock::comfyui::reset() {
    # Clear all data
    COMFYUI_MOCK_WORKFLOWS=()
    COMFYUI_MOCK_EXECUTIONS=()
    COMFYUI_MOCK_MODELS=()
    COMFYUI_MOCK_QUEUE=()
    COMFYUI_MOCK_CLI_HISTORY=()
    
    # Reset configuration to defaults
    COMFYUI_MOCK_CONFIG=(
        [host]="localhost"
        [port]="8188"
        [base_url]="http://localhost:8188"
        [container_name]="comfyui_test"
        [version]="1.0.0"
        [connected]="true"
        [error_mode]=""
        [gpu_available]="true"
        [gpu_type]="cuda"
        [nvidia_runtime]="true"
        [models_downloaded]="true"
        [disk_space_gb]="50"
    )
    
    # Save state
    mock::comfyui::save_state
    
    if command -v mock::log_state &>/dev/null; then
        mock::log_state "comfyui" "ComfyUI mock reset to initial state"
    fi
}

mock::comfyui::set_gpu_available() {
    local available="$1"
    COMFYUI_MOCK_CONFIG[gpu_available]="$available"
    if [[ "$available" == "false" ]]; then
        COMFYUI_MOCK_CONFIG[nvidia_runtime]="false"
    fi
    mock::comfyui::save_state
}

mock::comfyui::set_models_downloaded() {
    local downloaded="$1"
    COMFYUI_MOCK_CONFIG[models_downloaded]="$downloaded"
    mock::comfyui::save_state
}

mock::comfyui::set_connected() {
    local connected="$1"
    COMFYUI_MOCK_CONFIG[connected]="$connected"
    mock::comfyui::save_state
}

mock::comfyui::set_error_mode() {
    local error_mode="$1"
    COMFYUI_MOCK_CONFIG[error_mode]="$error_mode"
    mock::comfyui::save_state
}

# Debug function
mock::comfyui::dump_state() {
    echo "=== ComfyUI Mock State ==="
    echo "Configuration:"
    for key in "${!COMFYUI_MOCK_CONFIG[@]}"; do
        echo "  $key: ${COMFYUI_MOCK_CONFIG[$key]}"
    done
    
    # Show workflows if any exist
    if [[ ${#COMFYUI_MOCK_WORKFLOWS[@]} -gt 0 ]]; then
        echo "Workflows:"
        for key in "${!COMFYUI_MOCK_WORKFLOWS[@]}"; do
            echo "  $key: ${COMFYUI_MOCK_WORKFLOWS[$key]}"
        done
    fi
    
    # Show executions if any exist
    if [[ ${#COMFYUI_MOCK_EXECUTIONS[@]} -gt 0 ]]; then
        echo "Executions:"
        for key in "${!COMFYUI_MOCK_EXECUTIONS[@]}"; do
            echo "  $key: ${COMFYUI_MOCK_EXECUTIONS[$key]}"
        done
    fi
    
    echo "=========================="
}

# Export new functions
export -f nvidia-smi
export -f curl  
export -f mock::comfyui::reset
export -f mock::comfyui::set_gpu_available
export -f mock::comfyui::set_models_downloaded
export -f mock::comfyui::set_connected
export -f mock::comfyui::set_error_mode
export -f mock::comfyui::dump_state
export -f mock::comfyui::save_state
export -f mock::comfyui::load_state

# Initialize models on load
COMFYUI_MOCK_MODELS["sd_xl_base_1.0.safetensors"]="{\"size\":\"6.62GB\",\"type\":\"checkpoint\"}"
COMFYUI_MOCK_MODELS["sd_xl_refiner_1.0.safetensors"]="{\"size\":\"6.08GB\",\"type\":\"checkpoint\"}"
COMFYUI_MOCK_MODELS["sdxl_vae.safetensors"]="{\"size\":\"334.6MB\",\"type\":\"vae\"}"
