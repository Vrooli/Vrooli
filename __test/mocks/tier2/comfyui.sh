#!/usr/bin/env bash
# ComfyUI Mock - Tier 2 (Stateful)
# 
# Provides stateful ComfyUI AI image generation mocking for testing:
# - Workflow management (create, execute, delete)
# - Model management (download, list, status)
# - Queue management for prompt execution
# - Image generation simulation
# - HTTP API endpoints
# - Error injection for resilience testing
#
# Coverage: ~80% of common ComfyUI operations in 600 lines

# === Configuration ===
declare -gA COMFYUI_WORKFLOWS=()         # Workflow_name -> "json|status|complexity"
declare -gA COMFYUI_MODELS=()            # Model_name -> "type|size_gb|downloaded"
declare -gA COMFYUI_QUEUE=()             # Prompt_id -> "workflow|status|progress"
declare -gA COMFYUI_OUTPUTS=()           # Prompt_id -> "images|time|nodes"
declare -gA COMFYUI_CONFIG=(             # Service configuration
    [status]="running"
    [port]="8188"
    [gpu_available]="true"
    [gpu_memory_gb]="8"
    [models_path]="/models"
    [error_mode]=""
    [version]="1.0.0"
)

# Debug mode
declare -g COMFYUI_DEBUG="${COMFYUI_DEBUG:-}"

# === Helper Functions ===
comfyui_debug() {
    [[ -n "$COMFYUI_DEBUG" ]] && echo "[MOCK:COMFYUI] $*" >&2
}

comfyui_check_error() {
    case "${COMFYUI_CONFIG[error_mode]}" in
        "service_down")
            echo "Error: ComfyUI service is not running" >&2
            return 1
            ;;
        "gpu_unavailable")
            echo "Error: No GPU available for processing" >&2
            return 1
            ;;
        "model_missing")
            echo "Error: Required model not found" >&2
            return 1
            ;;
        "queue_full")
            echo "Error: Processing queue is full" >&2
            return 1
            ;;
    esac
    return 0
}

comfyui_generate_id() {
    printf "%08x-%04x-%04x-%04x-%012x" \
        $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM
}

# === Main ComfyUI CLI Command ===
comfyui() {
    comfyui_debug "comfyui called with: $*"
    
    if ! comfyui_check_error; then
        return $?
    fi
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        workflow)
            comfyui_cmd_workflow "$@"
            ;;
        execute)
            comfyui_cmd_execute "$@"
            ;;
        models)
            comfyui_cmd_models "$@"
            ;;
        queue)
            comfyui_cmd_queue "$@"
            ;;
        status)
            comfyui_cmd_status "$@"
            ;;
        start|stop|restart)
            comfyui_cmd_service "$command" "$@"
            ;;
        *)
            echo "ComfyUI CLI - AI Image Generation Platform"
            echo "Commands:"
            echo "  workflow  - Manage workflows"
            echo "  execute   - Execute a workflow"
            echo "  models    - Manage models"
            echo "  queue     - View processing queue"
            echo "  status    - Show service status"
            echo "  start     - Start service"
            echo "  stop      - Stop service"
            echo "  restart   - Restart service"
            ;;
    esac
}

# === Workflow Management ===
comfyui_cmd_workflow() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Available workflows:"
            if [[ ${#COMFYUI_WORKFLOWS[@]} -eq 0 ]]; then
                echo "  (none)"
            else
                for name in "${!COMFYUI_WORKFLOWS[@]}"; do
                    local data="${COMFYUI_WORKFLOWS[$name]}"
                    IFS='|' read -r json status complexity <<< "$data"
                    echo "  $name - Status: $status, Complexity: $complexity"
                done
            fi
            ;;
        create)
            local name="${1:-}"
            local complexity="${2:-simple}"
            [[ -z "$name" ]] && { echo "Error: workflow name required" >&2; return 1; }
            
            local workflow_json='{"nodes":[],"links":[],"version":"0.4"}'
            COMFYUI_WORKFLOWS[$name]="$workflow_json|ready|$complexity"
            comfyui_debug "Created workflow: $name"
            echo "Workflow '$name' created successfully"
            ;;
        delete)
            local name="${1:-}"
            [[ -z "$name" ]] && { echo "Error: workflow name required" >&2; return 1; }
            
            if [[ -n "${COMFYUI_WORKFLOWS[$name]}" ]]; then
                unset COMFYUI_WORKFLOWS[$name]
                echo "Workflow '$name' deleted"
            else
                echo "Error: workflow not found: $name" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: comfyui workflow {list|create|delete} [args]"
            return 1
            ;;
    esac
}

# === Workflow Execution ===
comfyui_cmd_execute() {
    local workflow="${1:-}"
    [[ -z "$workflow" ]] && { echo "Error: workflow name required" >&2; return 1; }
    
    if [[ -z "${COMFYUI_WORKFLOWS[$workflow]}" ]]; then
        echo "Error: workflow not found: $workflow" >&2
        return 1
    fi
    
    local data="${COMFYUI_WORKFLOWS[$workflow]}"
    IFS='|' read -r json status complexity <<< "$data"
    
    local prompt_id=$(comfyui_generate_id)
    COMFYUI_QUEUE[$prompt_id]="$workflow|queued|0"
    
    comfyui_debug "Queued workflow execution: $workflow -> $prompt_id"
    
    # Simulate execution based on complexity
    local exec_time nodes images
    case "$complexity" in
        simple)
            exec_time="5.2"
            nodes="5"
            images="1"
            ;;
        complex)
            exec_time="25.8"
            nodes="15"
            images="3"
            ;;
        heavy)
            exec_time="120.5"
            nodes="30"
            images="5"
            ;;
        *)
            exec_time="10.0"
            nodes="8"
            images="2"
            ;;
    esac
    
    # Update queue status
    COMFYUI_QUEUE[$prompt_id]="$workflow|processing|50"
    COMFYUI_QUEUE[$prompt_id]="$workflow|completed|100"
    
    # Store output
    COMFYUI_OUTPUTS[$prompt_id]="$images|$exec_time|$nodes"
    
    echo "Workflow execution started"
    echo "Prompt ID: $prompt_id"
    echo "Execution time: ${exec_time}s"
    echo "Nodes executed: $nodes"
    echo "Images generated: $images"
}

# === Model Management ===
comfyui_cmd_models() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Available models:"
            echo "Checkpoints:"
            for model in "${!COMFYUI_MODELS[@]}"; do
                local data="${COMFYUI_MODELS[$model]}"
                IFS='|' read -r type size downloaded <<< "$data"
                [[ "$type" == "checkpoint" ]] && echo "  $model (${size}GB) - $downloaded"
            done
            echo "VAE:"
            for model in "${!COMFYUI_MODELS[@]}"; do
                local data="${COMFYUI_MODELS[$model]}"
                IFS='|' read -r type size downloaded <<< "$data"
                [[ "$type" == "vae" ]] && echo "  $model (${size}GB) - $downloaded"
            done
            echo "LoRA:"
            for model in "${!COMFYUI_MODELS[@]}"; do
                local data="${COMFYUI_MODELS[$model]}"
                IFS='|' read -r type size downloaded <<< "$data"
                [[ "$type" == "lora" ]] && echo "  $model (${size}GB) - $downloaded"
            done
            ;;
        download)
            local model="${1:-}"
            [[ -z "$model" ]] && { echo "Error: model name required" >&2; return 1; }
            
            # Simulate download
            echo "Downloading model: $model"
            echo "Progress: [##########] 100%"
            
            # Add to models based on name patterns
            local type="checkpoint"
            [[ "$model" =~ vae ]] && type="vae"
            [[ "$model" =~ lora ]] && type="lora"
            
            COMFYUI_MODELS[$model]="$type|2.5|downloaded"
            echo "Model '$model' downloaded successfully"
            ;;
        remove)
            local model="${1:-}"
            [[ -z "$model" ]] && { echo "Error: model name required" >&2; return 1; }
            
            if [[ -n "${COMFYUI_MODELS[$model]}" ]]; then
                unset COMFYUI_MODELS[$model]
                echo "Model '$model' removed"
            else
                echo "Error: model not found: $model" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: comfyui models {list|download|remove} [args]"
            return 1
            ;;
    esac
}

# === Queue Management ===
comfyui_cmd_queue() {
    echo "Processing queue:"
    if [[ ${#COMFYUI_QUEUE[@]} -eq 0 ]] && [[ ${#COMFYUI_OUTPUTS[@]} -eq 0 ]]; then
        echo "  (empty)"
    else
        # Show active queue items
        for prompt_id in "${!COMFYUI_QUEUE[@]}"; do
            local data="${COMFYUI_QUEUE[$prompt_id]}"
            IFS='|' read -r workflow status progress <<< "$data"
            echo "  $prompt_id - Workflow: $workflow, Status: $status, Progress: ${progress}%"
        done
        # Also show completed items
        if [[ ${#COMFYUI_OUTPUTS[@]} -gt 0 ]]; then
            echo "Completed:"
            for prompt_id in "${!COMFYUI_OUTPUTS[@]}"; do
                echo "  $prompt_id - Status: completed"
            done
        fi
    fi
}

# === Status Command ===
comfyui_cmd_status() {
    echo "ComfyUI Status"
    echo "=============="
    echo "Service: ${COMFYUI_CONFIG[status]}"
    echo "Port: ${COMFYUI_CONFIG[port]}"
    echo "GPU Available: ${COMFYUI_CONFIG[gpu_available]}"
    echo "GPU Memory: ${COMFYUI_CONFIG[gpu_memory_gb]}GB"
    echo "Version: ${COMFYUI_CONFIG[version]}"
    echo ""
    echo "Workflows: ${#COMFYUI_WORKFLOWS[@]}"
    echo "Models: ${#COMFYUI_MODELS[@]}"
    echo "Queue: ${#COMFYUI_QUEUE[@]} items"
    echo "Completed: ${#COMFYUI_OUTPUTS[@]} prompts"
}

# === Service Management ===
comfyui_cmd_service() {
    local action="$1"
    
    case "$action" in
        start)
            if [[ "${COMFYUI_CONFIG[status]}" == "running" ]]; then
                echo "ComfyUI is already running"
            else
                COMFYUI_CONFIG[status]="running"
                echo "ComfyUI started on port ${COMFYUI_CONFIG[port]}"
            fi
            ;;
        stop)
            COMFYUI_CONFIG[status]="stopped"
            echo "ComfyUI stopped"
            ;;
        restart)
            COMFYUI_CONFIG[status]="stopped"
            COMFYUI_CONFIG[status]="running"
            echo "ComfyUI restarted"
            ;;
    esac
}

# === HTTP API Mock (via curl interceptor) ===
curl() {
    comfyui_debug "curl called with: $*"
    
    local url="" method="GET" data="" output_file="" silent=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X) method="$2"; shift 2 ;;
            -d|--data|--data-raw) data="$2"; shift 2 ;;
            -o|--output) output_file="$2"; shift 2 ;;
            -s|--silent) silent=true; shift ;;
            -H|--header) shift 2 ;; # Skip headers
            http*) url="$1"; shift ;;
            *) shift ;;
        esac
    done
    
    # Check if this is a ComfyUI API call
    if [[ "$url" =~ localhost:8188 || "$url" =~ 127.0.0.1:8188 ]]; then
        comfyui_handle_api "$method" "$url" "$data" "$output_file"
        return $?
    fi
    
    echo "curl: Not a ComfyUI endpoint"
    return 0
}

comfyui_handle_api() {
    local method="$1" url="$2" data="$3" output_file="$4"
    local response=""
    
    case "$url" in
        */api/prompt)
            if [[ "$method" == "POST" ]]; then
                local prompt_id=$(comfyui_generate_id)
                response='{"prompt_id":"'$prompt_id'","number":1,"node_errors":{}}'
                COMFYUI_QUEUE[$prompt_id]="api|queued|0"
            else
                response='{"error":"Method not allowed"}'
            fi
            ;;
        */api/queue)
            local queue_items=()
            for pid in "${!COMFYUI_QUEUE[@]}"; do
                queue_items+=("\"$pid\"")
            done
            response='{"queue_running":['$(IFS=,; echo "${queue_items[*]}")'],"queue_pending":[]}'
            ;;
        */api/history*)
            response='{'
            local first=true
            for pid in "${!COMFYUI_OUTPUTS[@]}"; do
                [[ "$first" != "true" ]] && response+=","
                local output="${COMFYUI_OUTPUTS[$pid]}"
                IFS='|' read -r images time nodes <<< "$output"
                response+='"'$pid'":{"outputs":{"images":['
                for ((i=1; i<=images; i++)); do
                    [[ $i -gt 1 ]] && response+=","
                    response+='{"filename":"output_'$i'.png","type":"output"}'
                done
                response+=']},"status":{"completed":true}}'
                first=false
            done
            response+='}'
            ;;
        */api/object_info)
            response='{"checkpoints":["sd_xl_base.safetensors"],"vae":["vae.safetensors"],"loras":[]}'
            ;;
        */health|*/api/health)
            if [[ "${COMFYUI_CONFIG[status]}" == "running" ]]; then
                response='{"status":"healthy","gpu_available":'${COMFYUI_CONFIG[gpu_available]}'}'
            else
                response='{"status":"unhealthy","error":"Service not running"}'
            fi
            ;;
        */api/interrupt)
            response='{"interrupted":true}'
            ;;
        *)
            response='{"status":"ok","version":"'${COMFYUI_CONFIG[version]}'"}'
            ;;
    esac
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
}

# === Mock Control Functions ===
comfyui_mock_reset() {
    comfyui_debug "Resetting mock state"
    
    COMFYUI_WORKFLOWS=()
    COMFYUI_MODELS=()
    COMFYUI_QUEUE=()
    COMFYUI_OUTPUTS=()
    COMFYUI_CONFIG[error_mode]=""
    COMFYUI_CONFIG[status]="running"
    
    # Initialize defaults
    comfyui_mock_init_defaults
}

comfyui_mock_init_defaults() {
    # Default models
    COMFYUI_MODELS["sd_xl_base_1.0.safetensors"]="checkpoint|6.5|downloaded"
    COMFYUI_MODELS["sdxl_vae.safetensors"]="vae|0.3|downloaded"
    COMFYUI_MODELS["detail_enhancer.safetensors"]="lora|0.1|downloaded"
    
    # Default workflow
    COMFYUI_WORKFLOWS["default"]='{"nodes":[],"links":[]}|ready|simple'
}

comfyui_mock_set_error() {
    COMFYUI_CONFIG[error_mode]="$1"
    comfyui_debug "Set error mode: $1"
}

comfyui_mock_create_workflow() {
    local name="${1:-test-workflow}"
    local complexity="${2:-simple}"
    COMFYUI_WORKFLOWS[$name]='{"nodes":[],"links":[]}|ready|'$complexity
    comfyui_debug "Created workflow: $name"
    echo "$name"
}

comfyui_mock_dump_state() {
    echo "=== ComfyUI Mock State ==="
    echo "Status: ${COMFYUI_CONFIG[status]}"
    echo "Port: ${COMFYUI_CONFIG[port]}"
    echo "GPU: ${COMFYUI_CONFIG[gpu_available]} (${COMFYUI_CONFIG[gpu_memory_gb]}GB)"
    echo "Workflows: ${#COMFYUI_WORKFLOWS[@]}"
    for name in "${!COMFYUI_WORKFLOWS[@]}"; do
        echo "  $name: ${COMFYUI_WORKFLOWS[$name]}"
    done
    echo "Models: ${#COMFYUI_MODELS[@]}"
    for model in "${!COMFYUI_MODELS[@]}"; do
        echo "  $model: ${COMFYUI_MODELS[$model]}"
    done
    echo "Queue: ${#COMFYUI_QUEUE[@]}"
    echo "Outputs: ${#COMFYUI_OUTPUTS[@]}"
    echo "Error Mode: ${COMFYUI_CONFIG[error_mode]:-none}"
    echo "====================="
}

# === Convention-based Test Functions ===
test_comfyui_connection() {
    comfyui_debug "Testing connection..."
    
    local result
    result=$(curl -s http://localhost:8188/health 2>&1)
    
    if [[ "$result" =~ "status" ]]; then
        comfyui_debug "Connection test passed"
        return 0
    else
        comfyui_debug "Connection test failed"
        return 1
    fi
}

test_comfyui_health() {
    comfyui_debug "Testing health..."
    
    test_comfyui_connection || return 1
    
    # Test workflow operations
    comfyui workflow create test-health simple >/dev/null 2>&1 || return 1
    comfyui workflow list | grep -q "test-health" || return 1
    comfyui workflow delete test-health >/dev/null 2>&1 || return 1
    
    comfyui_debug "Health test passed"
    return 0
}

test_comfyui_basic() {
    comfyui_debug "Testing basic operations..."
    
    # Test model management
    comfyui models download test-model.safetensors >/dev/null 2>&1 || return 1
    comfyui models list | grep -q "test-model" || return 1
    
    # Test workflow execution
    local workflow=$(comfyui_mock_create_workflow "test-basic" "simple")
    comfyui execute "$workflow" >/dev/null 2>&1 || return 1
    
    # Test queue
    local result
    result=$(comfyui queue 2>&1)
    [[ "$result" =~ "completed" ]] || [[ "$result" =~ "empty" ]] || return 1
    
    comfyui_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f comfyui curl
export -f test_comfyui_connection test_comfyui_health test_comfyui_basic
export -f comfyui_mock_reset comfyui_mock_set_error comfyui_mock_create_workflow
export -f comfyui_mock_dump_state
export -f comfyui_debug comfyui_check_error

# Initialize with defaults
comfyui_mock_reset
comfyui_debug "ComfyUI Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
