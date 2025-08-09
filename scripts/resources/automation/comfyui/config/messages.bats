#!/usr/bin/env bats
# Tests for ComfyUI messages.sh configuration

# Setup for each test
setup() {
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    COMFYUI_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Set up test variables that messages might reference
    export COMFYUI_PORT="8188"
    export COMFYUI_BASE_URL="http://localhost:8188"
    export COMFYUI_CONTAINER_NAME="vrooli-comfyui"
    export COMFYUI_SERVICE_NAME="comfyui"
    export WORKFLOW_ID="test-workflow-123"
    export MODEL_NAME="test-model.safetensors"
    export GPU_TYPE="nvidia"
    
    # Load the messages configuration to test
    source "${COMFYUI_DIR}/config/messages.sh"
}

# Test installation messages
@test "MSG_COMFYUI_INSTALLING is defined and meaningful" {
    [ -n "$MSG_COMFYUI_INSTALLING" ]
    [[ "$MSG_COMFYUI_INSTALLING" =~ "Installing" ]]
    [[ "$MSG_COMFYUI_INSTALLING" =~ "ComfyUI" ]]
}

@test "MSG_COMFYUI_ALREADY_INSTALLED is defined and meaningful" {
    [ -n "$MSG_COMFYUI_ALREADY_INSTALLED" ]
    [[ "$MSG_COMFYUI_ALREADY_INSTALLED" =~ "already" ]]
    [[ "$MSG_COMFYUI_ALREADY_INSTALLED" =~ "installed" ]]
}

# Test status messages
@test "MSG_COMFYUI_NOT_FOUND is defined and meaningful" {
    [ -n "$MSG_COMFYUI_NOT_FOUND" ]
    [[ "$MSG_COMFYUI_NOT_FOUND" =~ "not found" ]]
}

@test "MSG_COMFYUI_NOT_RUNNING is defined and meaningful" {
    [ -n "$MSG_COMFYUI_NOT_RUNNING" ]
    [[ "$MSG_COMFYUI_NOT_RUNNING" =~ "not running" ]]
}

@test "MSG_STATUS_CONTAINER_OK is defined and meaningful" {
    [ -n "$MSG_STATUS_CONTAINER_OK" ]
    [[ "$MSG_STATUS_CONTAINER_OK" =~ "container" ]]
    [[ "$MSG_STATUS_CONTAINER_OK" =~ "OK" || "$MSG_STATUS_CONTAINER_OK" =~ "found" ]]
}

@test "MSG_STATUS_CONTAINER_RUNNING is defined and meaningful" {
    [ -n "$MSG_STATUS_CONTAINER_RUNNING" ]
    [[ "$MSG_STATUS_CONTAINER_RUNNING" =~ "running" ]]
}

# Test workflow messages
@test "MSG_WORKFLOW_EXECUTING is defined and supports variable substitution" {
    [ -n "$MSG_WORKFLOW_EXECUTING" ]
    [[ "$MSG_WORKFLOW_EXECUTING" =~ "Executing" || "$MSG_WORKFLOW_EXECUTING" =~ "workflow" ]]
    
    # Test variable substitution
    workflow_id="test-workflow-123"
    expanded_msg=$(eval "echo \"$MSG_WORKFLOW_EXECUTING\"")
    [[ "$expanded_msg" =~ "$workflow_id" ]]
}

@test "MSG_WORKFLOW_COMPLETED is defined and meaningful" {
    [ -n "$MSG_WORKFLOW_COMPLETED" ]
    [[ "$MSG_WORKFLOW_COMPLETED" =~ "complete" || "$MSG_WORKFLOW_COMPLETED" =~ "finished" ]]
}

@test "MSG_WORKFLOW_FAILED is defined and meaningful" {
    [ -n "$MSG_WORKFLOW_FAILED" ]
    [[ "$MSG_WORKFLOW_FAILED" =~ "failed" || "$MSG_WORKFLOW_FAILED" =~ "error" ]]
}

# Test model management messages
@test "MSG_MODEL_DOWNLOADING is defined and supports variable substitution" {
    [ -n "$MSG_MODEL_DOWNLOADING" ]
    [[ "$MSG_MODEL_DOWNLOADING" =~ "Downloading" || "$MSG_MODEL_DOWNLOADING" =~ "model" ]]
    
    # Test variable substitution
    model_name="test-model.safetensors"
    expanded_msg=$(eval "echo \"$MSG_MODEL_DOWNLOADING\"")
    [[ "$expanded_msg" =~ "$model_name" ]]
}

@test "MSG_MODEL_INSTALLED is defined and meaningful" {
    [ -n "$MSG_MODEL_INSTALLED" ]
    [[ "$MSG_MODEL_INSTALLED" =~ "installed" || "$MSG_MODEL_INSTALLED" =~ "ready" ]]
}

@test "MSG_MODEL_NOT_FOUND is defined and meaningful" {
    [ -n "$MSG_MODEL_NOT_FOUND" ]
    [[ "$MSG_MODEL_NOT_FOUND" =~ "not found" || "$MSG_MODEL_NOT_FOUND" =~ "missing" ]]
}

# Test GPU messages
@test "MSG_GPU_DETECTED is defined and supports variable substitution" {
    [ -n "$MSG_GPU_DETECTED" ]
    [[ "$MSG_GPU_DETECTED" =~ "GPU" || "$MSG_GPU_DETECTED" =~ "detected" ]]
    
    # Test variable substitution
    gpu_type="nvidia"
    expanded_msg=$(eval "echo \"$MSG_GPU_DETECTED\"")
    [[ "$expanded_msg" =~ "$gpu_type" ]]
}

@test "MSG_GPU_NOT_AVAILABLE is defined and meaningful" {
    [ -n "$MSG_GPU_NOT_AVAILABLE" ]
    [[ "$MSG_GPU_NOT_AVAILABLE" =~ "GPU" ]]
    [[ "$MSG_GPU_NOT_AVAILABLE" =~ "not available" || "$MSG_GPU_NOT_AVAILABLE" =~ "not found" ]]
}

@test "MSG_FALLBACK_CPU_MODE is defined and meaningful" {
    [ -n "$MSG_FALLBACK_CPU_MODE" ]
    [[ "$MSG_FALLBACK_CPU_MODE" =~ "CPU" ]]
    [[ "$MSG_FALLBACK_CPU_MODE" =~ "fallback" || "$MSG_FALLBACK_CPU_MODE" =~ "mode" ]]
}

# Test error messages
@test "MSG_DOCKER_NOT_AVAILABLE is defined and meaningful" {
    [ -n "$MSG_DOCKER_NOT_AVAILABLE" ]
    [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ "Docker" ]]
    [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ "not available" || "$MSG_DOCKER_NOT_AVAILABLE" =~ "not found" ]]
}

@test "MSG_PORT_IN_USE is defined and supports variable substitution" {
    [ -n "$MSG_PORT_IN_USE" ]
    [[ "$MSG_PORT_IN_USE" =~ "port" ]]
    [[ "$MSG_PORT_IN_USE" =~ "in use" || "$MSG_PORT_IN_USE" =~ "busy" ]]
    
    # Test variable substitution
    port="8188"
    expanded_msg=$(eval "echo \"$MSG_PORT_IN_USE\"")
    [[ "$expanded_msg" =~ "$port" ]]
}

@test "MSG_INSUFFICIENT_MEMORY is defined and meaningful" {
    [ -n "$MSG_INSUFFICIENT_MEMORY" ]
    [[ "$MSG_INSUFFICIENT_MEMORY" =~ "memory" ]]
    [[ "$MSG_INSUFFICIENT_MEMORY" =~ "insufficient" || "$MSG_INSUFFICIENT_MEMORY" =~ "not enough" ]]
}

@test "MSG_IMAGE_PULL_FAILED is defined and meaningful" {
    [ -n "$MSG_IMAGE_PULL_FAILED" ]
    [[ "$MSG_IMAGE_PULL_FAILED" =~ "image" ]]
    [[ "$MSG_IMAGE_PULL_FAILED" =~ "pull" ]]
    [[ "$MSG_IMAGE_PULL_FAILED" =~ "failed" ]]
}

# Test success messages
@test "MSG_INSTALLATION_SUCCESS is defined and meaningful" {
    [ -n "$MSG_INSTALLATION_SUCCESS" ]
    [[ "$MSG_INSTALLATION_SUCCESS" =~ "success" || "$MSG_INSTALLATION_SUCCESS" =~ "installed" ]]
    [[ "$MSG_INSTALLATION_SUCCESS" =~ "ComfyUI" ]]
}

@test "MSG_SERVICE_HEALTHY is defined and meaningful" {
    [ -n "$MSG_SERVICE_HEALTHY" ]
    [[ "$MSG_SERVICE_HEALTHY" =~ "healthy" || "$MSG_SERVICE_HEALTHY" =~ "running" ]]
}

@test "MSG_CONTAINER_STARTED is defined and meaningful" {
    [ -n "$MSG_CONTAINER_STARTED" ]
    [[ "$MSG_CONTAINER_STARTED" =~ "started" || "$MSG_CONTAINER_STARTED" =~ "running" ]]
}

# Test information messages
@test "MSG_SERVICE_INFO is defined and supports variable substitution" {
    [ -n "$MSG_SERVICE_INFO" ]
    [[ "$MSG_SERVICE_INFO" =~ "ComfyUI" ]]
    
    # Test that it can include URL
    expanded_msg=$(eval "echo \"$MSG_SERVICE_INFO\"")
    [[ "$expanded_msg" =~ "http://" || "$expanded_msg" =~ "localhost" ]]
}

@test "MSG_AVAILABLE_AT is defined and supports variable substitution" {
    [ -n "$MSG_AVAILABLE_AT" ]
    [[ "$MSG_AVAILABLE_AT" =~ "available" ]]
    
    # Test URL substitution
    expanded_msg=$(eval "echo \"$MSG_AVAILABLE_AT\"")
    [[ "$expanded_msg" =~ "http://localhost:8188" ]]
}

@test "MSG_API_ENDPOINT is defined and supports variable substitution" {
    [ -n "$MSG_API_ENDPOINT" ]
    [[ "$MSG_API_ENDPOINT" =~ "API" ]]
    
    # Test API URL substitution
    expanded_msg=$(eval "echo \"$MSG_API_ENDPOINT\"")
    [[ "$expanded_msg" =~ "http://localhost:8188/api" ]]
}

# Test queue and execution messages
@test "MSG_QUEUE_EMPTY is defined and meaningful" {
    [ -n "$MSG_QUEUE_EMPTY" ]
    [[ "$MSG_QUEUE_EMPTY" =~ "queue" ]]
    [[ "$MSG_QUEUE_EMPTY" =~ "empty" || "$MSG_QUEUE_EMPTY" =~ "no" ]]
}

@test "MSG_QUEUE_PROCESSING is defined and meaningful" {
    [ -n "$MSG_QUEUE_PROCESSING" ]
    [[ "$MSG_QUEUE_PROCESSING" =~ "queue" ]]
    [[ "$MSG_QUEUE_PROCESSING" =~ "processing" || "$MSG_QUEUE_PROCESSING" =~ "running" ]]
}

@test "MSG_EXECUTION_INTERRUPTED is defined and meaningful" {
    [ -n "$MSG_EXECUTION_INTERRUPTED" ]
    [[ "$MSG_EXECUTION_INTERRUPTED" =~ "interrupt" || "$MSG_EXECUTION_INTERRUPTED" =~ "stopped" ]]
}

# Test uninstallation messages
@test "MSG_UNINSTALLING is defined and meaningful" {
    [ -n "$MSG_UNINSTALLING" ]
    [[ "$MSG_UNINSTALLING" =~ "Uninstalling" || "$MSG_UNINSTALLING" =~ "Removing" ]]
}

@test "MSG_UNINSTALL_SUCCESS is defined and meaningful" {
    [ -n "$MSG_UNINSTALL_SUCCESS" ]
    [[ "$MSG_UNINSTALL_SUCCESS" =~ "success" || "$MSG_UNINSTALL_SUCCESS" =~ "removed" ]]
}

# Test startup/shutdown messages
@test "MSG_STARTING_SERVICE is defined and meaningful" {
    [ -n "$MSG_STARTING_SERVICE" ]
    [[ "$MSG_STARTING_SERVICE" =~ "Starting" ]]
}

@test "MSG_STOPPING_SERVICE is defined and meaningful" {
    [ -n "$MSG_STOPPING_SERVICE" ]
    [[ "$MSG_STOPPING_SERVICE" =~ "Stopping" ]]
}

@test "MSG_RESTARTING_SERVICE is defined and meaningful" {
    [ -n "$MSG_RESTARTING_SERVICE" ]
    [[ "$MSG_RESTARTING_SERVICE" =~ "Restarting" ]]
}

# Test backup/restore messages
@test "MSG_BACKUP_CREATED is defined and meaningful" {
    [ -n "$MSG_BACKUP_CREATED" ]
    [[ "$MSG_BACKUP_CREATED" =~ "backup" ]]
    [[ "$MSG_BACKUP_CREATED" =~ "created" ]]
}

@test "MSG_BACKUP_RESTORED is defined and meaningful" {
    [ -n "$MSG_BACKUP_RESTORED" ]
    [[ "$MSG_BACKUP_RESTORED" =~ "backup" ]]
    [[ "$MSG_BACKUP_RESTORED" =~ "restored" ]]
}

# Test configuration messages
@test "MSG_CONFIG_UPDATED is defined and meaningful" {
    [ -n "$MSG_CONFIG_UPDATED" ]
    [[ "$MSG_CONFIG_UPDATED" =~ "config" ]]
    [[ "$MSG_CONFIG_UPDATED" =~ "updated" ]]
}

@test "MSG_CONFIG_INVALID is defined and meaningful" {
    [ -n "$MSG_CONFIG_INVALID" ]
    [[ "$MSG_CONFIG_INVALID" =~ "config" ]]
    [[ "$MSG_CONFIG_INVALID" =~ "invalid" ]]
}

# Test message export function
@test "comfyui::export_messages exports all message variables" {
    comfyui::export_messages
    
    # Check that variables are exported (available in subshells)
    result=$(bash -c 'echo $MSG_COMFYUI_INSTALLING')
    [ -n "$result" ]
    
    result=$(bash -c 'echo $MSG_SERVICE_INFO')
    [ -n "$result" ]
}

# Test that all required messages are defined
@test "all essential message variables are defined" {
    [ -n "$MSG_COMFYUI_INSTALLING" ]
    [ -n "$MSG_COMFYUI_ALREADY_INSTALLED" ]
    [ -n "$MSG_COMFYUI_NOT_FOUND" ]
    [ -n "$MSG_COMFYUI_NOT_RUNNING" ]
    [ -n "$MSG_STATUS_CONTAINER_OK" ]
    [ -n "$MSG_STATUS_CONTAINER_RUNNING" ]
    [ -n "$MSG_WORKFLOW_EXECUTING" ]
    [ -n "$MSG_MODEL_DOWNLOADING" ]
    [ -n "$MSG_GPU_DETECTED" ]
    [ -n "$MSG_DOCKER_NOT_AVAILABLE" ]
    [ -n "$MSG_PORT_IN_USE" ]
    [ -n "$MSG_INSTALLATION_SUCCESS" ]
    [ -n "$MSG_SERVICE_INFO" ]
    [ -n "$MSG_AVAILABLE_AT" ]
}

# Test message content quality
@test "messages contain appropriate emoji or formatting" {
    # Check that some messages have visual indicators
    local has_visual_indicator=false
    
    if [[ "$MSG_INSTALLATION_SUCCESS" =~ [✅🎉✨] ]] || \
       [[ "$MSG_COMFYUI_INSTALLING" =~ [🔧📦⚙️] ]] || \
       [[ "$MSG_SERVICE_HEALTHY" =~ [✅💚🟢] ]]; then
        has_visual_indicator=true
    fi
    
    [ "$has_visual_indicator" = true ]
}

@test "error messages are appropriately formatted" {
    # Error messages should have warning indicators or clear language
    local has_error_indicator=false
    
    if [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ [❌⚠️🔴] ]] || \
       [[ "$MSG_WORKFLOW_FAILED" =~ [❌⚠️] ]] || \
       [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ "ERROR" ]]; then
        has_error_indicator=true
    fi
    
    [ "$has_error_indicator" = true ]
}

# Test message variable interpolation
@test "messages with variables interpolate correctly" {
    # Test workflow ID variable
    workflow_id="test-workflow-123"
    expanded=$(eval "echo \"$MSG_WORKFLOW_EXECUTING\"")
    [[ "$expanded" =~ "$workflow_id" ]]
    
    # Test model name variable  
    model_name="test-model.safetensors"
    expanded=$(eval "echo \"$MSG_MODEL_DOWNLOADING\"")
    [[ "$expanded" =~ "$model_name" ]]
    
    # Test port variable
    expanded=$(eval "echo \"$MSG_PORT_IN_USE\"")
    [[ "$expanded" =~ "8188" ]]
    
    # Test URL variable
    expanded=$(eval "echo \"$MSG_AVAILABLE_AT\"")
    [[ "$expanded" =~ "http://localhost:8188" ]]
    
    # Test API endpoint variable
    expanded=$(eval "echo \"$MSG_API_ENDPOINT\"")
    [[ "$expanded" =~ "http://localhost:8188/api" ]]
}

# Test message consistency
@test "messages use consistent service naming" {
    # Check that service is consistently referred to
    local service_refs=0
    
    if [[ "$MSG_COMFYUI_INSTALLING" =~ "ComfyUI" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$MSG_INSTALLATION_SUCCESS" =~ "ComfyUI" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$MSG_SERVICE_INFO" =~ "ComfyUI" ]]; then
        ((service_refs++))
    fi
    
    # At least 2 messages should consistently name the service
    [ "$service_refs" -ge 2 ]
}

# Test workflow-specific message quality
@test "workflow messages are specific and informative" {
    # Workflow execution message should be specific
    [[ "$MSG_WORKFLOW_EXECUTING" =~ "workflow" ]]
    [[ "$MSG_WORKFLOW_COMPLETED" =~ "workflow" ]] || [[ "$MSG_WORKFLOW_COMPLETED" =~ "execution" ]]
    [[ "$MSG_WORKFLOW_FAILED" =~ "workflow" ]] || [[ "$MSG_WORKFLOW_FAILED" =~ "execution" ]]
}

# Test model-specific message quality
@test "model messages are specific and informative" {
    # Model messages should mention models explicitly
    [[ "$MSG_MODEL_DOWNLOADING" =~ "model" ]]
    [[ "$MSG_MODEL_INSTALLED" =~ "model" ]]
    [[ "$MSG_MODEL_NOT_FOUND" =~ "model" ]]
}

# Test GPU-specific message quality
@test "GPU messages are specific and informative" {
    # GPU messages should mention GPU explicitly
    [[ "$MSG_GPU_DETECTED" =~ "GPU" ]]
    [[ "$MSG_GPU_NOT_AVAILABLE" =~ "GPU" ]]
    [[ "$MSG_FALLBACK_CPU_MODE" =~ "CPU" ]]
}

# Test queue message quality
@test "queue messages are clear and actionable" {
    # Queue messages should be clear about queue state
    [[ "$MSG_QUEUE_EMPTY" =~ "queue" ]]
    [[ "$MSG_QUEUE_PROCESSING" =~ "queue" ]]
    [[ "$MSG_EXECUTION_INTERRUPTED" =~ "interrupt" ]] || [[ "$MSG_EXECUTION_INTERRUPTED" =~ "stop" ]]
}

# Test technical message appropriateness
@test "technical messages provide helpful context" {
    # Technical messages should be informative
    [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ "Docker" ]]
    [[ "$MSG_IMAGE_PULL_FAILED" =~ "image" ]]
    [[ "$MSG_INSUFFICIENT_MEMORY" =~ "memory" ]]
}

# Test user-facing message quality
@test "user-facing messages are welcoming and informative" {
    # Service info should be welcoming
    [[ "$MSG_SERVICE_INFO" =~ "ComfyUI" ]]
    [[ "$MSG_AVAILABLE_AT" =~ "available" ]]
    [[ "$MSG_API_ENDPOINT" =~ "API" ]]
}
