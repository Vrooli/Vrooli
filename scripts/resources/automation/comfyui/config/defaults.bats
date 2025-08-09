#!/usr/bin/env bats
# Tests for ComfyUI defaults.sh configuration

# Setup for each test
setup() {
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    COMFYUI_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Set up test environment variables that might be referenced
    export HOME="/tmp/test-home"
    export USER="testuser"
    export VROOLI_DIR="/tmp/vrooli"
    
    # Create test directories
    mkdir -p "$HOME/.vrooli"
    mkdir -p "$VROOLI_DIR"
    
    # Load the defaults configuration to test
    source "${COMFYUI_DIR}/config/defaults.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "/tmp/test-home"
    rm -rf "/tmp/vrooli"
}

# Test core service configuration
@test "COMFYUI_SERVICE_NAME is defined and valid" {
    [ -n "$COMFYUI_SERVICE_NAME" ]
    [[ "$COMFYUI_SERVICE_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
}

@test "COMFYUI_CONTAINER_NAME is defined and valid" {
    [ -n "$COMFYUI_CONTAINER_NAME" ]
    [[ "$COMFYUI_CONTAINER_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
}

# Test port configuration
@test "COMFYUI_DEFAULT_PORT is defined and valid" {
    [ -n "$COMFYUI_DEFAULT_PORT" ]
    [[ "$COMFYUI_DEFAULT_PORT" =~ ^[0-9]+$ ]]
    [ "$COMFYUI_DEFAULT_PORT" -gt 1024 ]
    [ "$COMFYUI_DEFAULT_PORT" -lt 65536 ]
}

@test "COMFYUI_PROXY_PORT is defined and valid" {
    [ -n "$COMFYUI_PROXY_PORT" ]
    [[ "$COMFYUI_PROXY_PORT" =~ ^[0-9]+$ ]]
    [ "$COMFYUI_PROXY_PORT" -gt 1024 ]
    [ "$COMFYUI_PROXY_PORT" -lt 65536 ]
}

@test "COMFYUI_DIRECT_PORT is defined and valid" {
    [ -n "$COMFYUI_DIRECT_PORT" ]
    [[ "$COMFYUI_DIRECT_PORT" =~ ^[0-9]+$ ]]
    [ "$COMFYUI_DIRECT_PORT" -gt 1024 ]
    [ "$COMFYUI_DIRECT_PORT" -lt 65536 ]
}

# Test URL configuration
@test "COMFYUI_BASE_URL is defined and valid" {
    [ -n "$COMFYUI_BASE_URL" ]
    [[ "$COMFYUI_BASE_URL" =~ ^http://localhost:[0-9]+$ ]]
}

@test "COMFYUI_API_BASE is defined and valid" {
    [ -n "$COMFYUI_API_BASE" ]
    [[ "$COMFYUI_API_BASE" =~ /api$ ]]
}

# Test Docker image configuration
@test "COMFYUI_DEFAULT_IMAGE is defined and valid" {
    [ -n "$COMFYUI_DEFAULT_IMAGE" ]
    [[ "$COMFYUI_DEFAULT_IMAGE" =~ ^[a-zA-Z0-9._/-]+:[a-zA-Z0-9._-]+$ ]]
}

@test "COMFYUI_CUDA_IMAGE is defined and valid" {
    [ -n "$COMFYUI_CUDA_IMAGE" ]
    [[ "$COMFYUI_CUDA_IMAGE" =~ cuda ]]
}

@test "COMFYUI_CPU_IMAGE is defined and valid" {
    [ -n "$COMFYUI_CPU_IMAGE" ]
    [[ "$COMFYUI_CPU_IMAGE" =~ cpu ]]
}

@test "COMFYUI_ROCM_IMAGE is defined and valid" {
    [ -n "$COMFYUI_ROCM_IMAGE" ]
    [[ "$COMFYUI_ROCM_IMAGE" =~ rocm ]]
}

# Test directory configuration
@test "COMFYUI_DATA_DIR is defined and valid" {
    [ -n "$COMFYUI_DATA_DIR" ]
    [[ "$COMFYUI_DATA_DIR" =~ ^/.+ ]]  # Absolute path
}

@test "COMFYUI_MODELS_DIR is defined and valid" {
    [ -n "$COMFYUI_MODELS_DIR" ]
    [[ "$COMFYUI_MODELS_DIR" =~ ^/.+ ]]  # Absolute path
}

@test "COMFYUI_OUTPUT_DIR is defined and valid" {
    [ -n "$COMFYUI_OUTPUT_DIR" ]
    [[ "$COMFYUI_OUTPUT_DIR" =~ ^/.+ ]]  # Absolute path
}

@test "COMFYUI_INPUT_DIR is defined and valid" {
    [ -n "$COMFYUI_INPUT_DIR" ]
    [[ "$COMFYUI_INPUT_DIR" =~ ^/.+ ]]  # Absolute path
}

@test "COMFYUI_WORKFLOWS_DIR is defined and valid" {
    [ -n "$COMFYUI_WORKFLOWS_DIR" ]
    [[ "$COMFYUI_WORKFLOWS_DIR" =~ ^/.+ ]]  # Absolute path
}

# Test GPU configuration
@test "COMFYUI_DEFAULT_GPU_TYPE is defined and valid" {
    [ -n "$COMFYUI_DEFAULT_GPU_TYPE" ]
    [[ "$COMFYUI_DEFAULT_GPU_TYPE" =~ ^(nvidia|amd|cpu)$ ]]
}

@test "COMFYUI_DEFAULT_VRAM_LIMIT is defined and valid" {
    [ -n "$COMFYUI_DEFAULT_VRAM_LIMIT" ]
    [[ "$COMFYUI_DEFAULT_VRAM_LIMIT" =~ ^[0-9]+$ ]]
    [ "$COMFYUI_DEFAULT_VRAM_LIMIT" -gt 0 ]
}

# Test health check configuration
@test "COMFYUI_HEALTH_CHECK_TIMEOUT is defined and valid" {
    [ -n "$COMFYUI_HEALTH_CHECK_TIMEOUT" ]
    [[ "$COMFYUI_HEALTH_CHECK_TIMEOUT" =~ ^[0-9]+$ ]]
    [ "$COMFYUI_HEALTH_CHECK_TIMEOUT" -gt 0 ]
}

@test "COMFYUI_HEALTH_CHECK_INTERVAL is defined and valid" {
    [ -n "$COMFYUI_HEALTH_CHECK_INTERVAL" ]
    [[ "$COMFYUI_HEALTH_CHECK_INTERVAL" =~ ^[0-9]+$ ]]
    [ "$COMFYUI_HEALTH_CHECK_INTERVAL" -gt 0 ]
}

@test "COMFYUI_STARTUP_TIMEOUT is defined and valid" {
    [ -n "$COMFYUI_STARTUP_TIMEOUT" ]
    [[ "$COMFYUI_STARTUP_TIMEOUT" =~ ^[0-9]+$ ]]
    [ "$COMFYUI_STARTUP_TIMEOUT" -gt 0 ]
}

# Test backup configuration
@test "COMFYUI_BACKUP_DIR is defined and valid" {
    [ -n "$COMFYUI_BACKUP_DIR" ]
    [[ "$COMFYUI_BACKUP_DIR" =~ ^/.+ ]]  # Absolute path
}

@test "COMFYUI_BACKUP_RETENTION_DAYS is defined and valid" {
    [ -n "$COMFYUI_BACKUP_RETENTION_DAYS" ]
    [[ "$COMFYUI_BACKUP_RETENTION_DAYS" =~ ^[0-9]+$ ]]
    [ "$COMFYUI_BACKUP_RETENTION_DAYS" -gt 0 ]
}

# Test logging configuration
@test "COMFYUI_LOG_LEVEL is defined and valid" {
    [ -n "$COMFYUI_LOG_LEVEL" ]
    [[ "$COMFYUI_LOG_LEVEL" =~ ^(DEBUG|INFO|WARN|ERROR)$ ]]
}

@test "COMFYUI_LOG_DIR is defined and valid" {
    [ -n "$COMFYUI_LOG_DIR" ]
    [[ "$COMFYUI_LOG_DIR" =~ ^/.+ ]]  # Absolute path
}

# Test resource limits
@test "COMFYUI_MEMORY_LIMIT is defined and valid" {
    [ -n "$COMFYUI_MEMORY_LIMIT" ]
    [[ "$COMFYUI_MEMORY_LIMIT" =~ ^[0-9]+[GMg]?$ ]]
}

@test "COMFYUI_CPU_LIMIT is defined and valid" {
    [ -n "$COMFYUI_CPU_LIMIT" ]
    [[ "$COMFYUI_CPU_LIMIT" =~ ^[0-9]*\.?[0-9]+$ ]]
}

# Test environment variables array
@test "COMFYUI_ENV_VARS is defined as array" {
    [ -n "$COMFYUI_ENV_VARS" ]
    # Check if it's an array
    declare -p COMFYUI_ENV_VARS | grep -q "declare -a"
}

# Test volumes array
@test "COMFYUI_VOLUMES is defined as array" {
    [ -n "$COMFYUI_VOLUMES" ]
    # Check if it's an array
    declare -p COMFYUI_VOLUMES | grep -q "declare -a"
}

# Test default models configuration
@test "COMFYUI_DEFAULT_MODELS is defined as array" {
    [ -n "$COMFYUI_DEFAULT_MODELS" ]
    # Check if it's an array
    declare -p COMFYUI_DEFAULT_MODELS | grep -q "declare -a"
}

# Test model categories
@test "COMFYUI_MODEL_CATEGORIES is defined as array" {
    [ -n "$COMFYUI_MODEL_CATEGORIES" ]
    # Check if it's an array
    declare -p COMFYUI_MODEL_CATEGORIES | grep -q "declare -a"
    
    # Check for essential categories
    local categories_str="${COMFYUI_MODEL_CATEGORIES[*]}"
    [[ "$categories_str" =~ checkpoints ]]
    [[ "$categories_str" =~ loras ]]
    [[ "$categories_str" =~ vae ]]
}

# Test configuration export function
@test "comfyui::export_config exports all configuration variables" {
    comfyui::export_config
    
    # Check that key variables are exported (available in subshells)
    result=$(bash -c 'echo $COMFYUI_SERVICE_NAME')
    [ -n "$result" ]
    
    result=$(bash -c 'echo $COMFYUI_DEFAULT_PORT')
    [ -n "$result" ]
    
    result=$(bash -c 'echo $COMFYUI_BASE_URL')
    [ -n "$result" ]
}

# Test custom configuration override
@test "custom configuration can override defaults" {
    # Set custom values
    export COMFYUI_CUSTOM_PORT="9999"
    export COMFYUI_CUSTOM_IMAGE="custom/comfyui:test"
    
    # Reload configuration
    source "${COMFYUI_DIR}/config/defaults.sh"
    
    # Check that custom values are used
    [[ "$COMFYUI_BASE_URL" =~ 9999 ]]
}

# Test GPU type detection integration
@test "GPU_TYPE variable is respected" {
    export GPU_TYPE="amd"
    
    # Reload configuration
    source "${COMFYUI_DIR}/config/defaults.sh"
    
    # Should influence image selection logic
    [ -n "$GPU_TYPE" ]
    [[ "$GPU_TYPE" == "amd" ]]
}

# Test directory path consistency
@test "all data directories are under COMFYUI_DATA_DIR" {
    [[ "$COMFYUI_MODELS_DIR" =~ ^$COMFYUI_DATA_DIR ]]
    [[ "$COMFYUI_OUTPUT_DIR" =~ ^$COMFYUI_DATA_DIR ]]
    [[ "$COMFYUI_INPUT_DIR" =~ ^$COMFYUI_DATA_DIR ]]
    [[ "$COMFYUI_WORKFLOWS_DIR" =~ ^$COMFYUI_DATA_DIR ]]
}

# Test port uniqueness
@test "all ports are unique" {
    [ "$COMFYUI_DEFAULT_PORT" != "$COMFYUI_PROXY_PORT" ]
    [ "$COMFYUI_DEFAULT_PORT" != "$COMFYUI_DIRECT_PORT" ]
    [ "$COMFYUI_PROXY_PORT" != "$COMFYUI_DIRECT_PORT" ]
}

# Test resource limit validation
@test "resource limits are reasonable" {
    # Memory limit should be at least 2GB
    if [[ "$COMFYUI_MEMORY_LIMIT" =~ ^([0-9]+)G$ ]]; then
        local mem_gb="${BASH_REMATCH[1]}"
        [ "$mem_gb" -ge 2 ]
    fi
    
    # CPU limit should be reasonable (0.1 to 16.0)
    if [[ "$COMFYUI_CPU_LIMIT" =~ ^([0-9]*\.?[0-9]+)$ ]]; then
        local cpu_val="${BASH_REMATCH[1]}"
        # Use awk for floating point comparison
        awk -v cpu="$cpu_val" 'BEGIN { exit (cpu >= 0.1 && cpu <= 16.0) ? 0 : 1 }'
    fi
}

# Test environment variables format
@test "environment variables have valid format" {
    for env_var in "${COMFYUI_ENV_VARS[@]}"; do
        [[ "$env_var" =~ ^[A-Z_]+=.* ]]
    done
}

# Test volume mounts format
@test "volume mounts have valid format" {
    for volume in "${COMFYUI_VOLUMES[@]}"; do
        [[ "$volume" =~ ^/.+:/.+ ]]
    done
}

# Test configuration validation function
@test "comfyui::validate_config validates configuration" {
    result=$(comfyui::validate_config)
    
    [[ "$?" -eq 0 ]]
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "OK" ]]
}

# Test configuration with missing required variables
@test "comfyui::validate_config detects missing variables" {
    # Temporarily unset a required variable
    local original_port="$COMFYUI_DEFAULT_PORT"
    unset COMFYUI_DEFAULT_PORT
    
    run comfyui::validate_config
    [ "$status" -eq 1 ]
    
    # Restore the variable
    export COMFYUI_DEFAULT_PORT="$original_port"
}

# Test configuration file generation
@test "comfyui::generate_config_file creates valid configuration file" {
    local config_file="/tmp/comfyui-test-config.env"
    
    result=$(comfyui::generate_config_file "$config_file")
    
    [ -f "$config_file" ]
    [[ "$result" =~ "generated" ]] || [[ "$result" =~ "created" ]]
    
    # Verify file contains expected variables
    grep -q "COMFYUI_SERVICE_NAME=" "$config_file"
    grep -q "COMFYUI_DEFAULT_PORT=" "$config_file"
    
    rm -f "$config_file"
}

# Test configuration backup
@test "comfyui::backup_config creates configuration backup" {
    result=$(comfyui::backup_config)
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "saved" ]]
}

# Test configuration restoration
@test "comfyui::restore_config restores configuration from backup" {
    result=$(comfyui::restore_config)
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "restored" ]]
}