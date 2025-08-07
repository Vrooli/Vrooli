#!/usr/bin/env bats
# Tests for ComfyUI manage.sh script

# Load Vrooli test infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "comfyui"
    
    # Load resource specific configuration once per file
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    COMFYUI_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and manage script once
    source "${COMFYUI_DIR}/config/defaults.sh"
    source "${COMFYUI_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/manage.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment variables (lightweight per-test)
    export COMFYUI_CUSTOM_PORT="9999"
    export COMFYUI_CONTAINER_NAME="comfyui-test"
    export COMFYUI_BASE_URL="http://localhost:9999"
    export FORCE="no"
    export YES="no"
    export GPU_TYPE="auto"
    export WORKFLOW_PATH=""
    export OUTPUT_DIR=""
    export PROMPT_ID=""
    
    # Export config functions
    comfyui::export_config
    comfyui::export_messages
    
    # Mock log functions
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    export -f log::header log::info log::error log::success log::warning
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test script loading
@test "manage.sh loads without errors" {
    # Check that essential functions are available
    declare -f comfyui::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
}

# Test argument parsing
@test "comfyui::parse_arguments sets defaults correctly" {
    comfyui::parse_arguments --action status
    
    [ "$ACTION" = "status" ]
    [ "$FORCE" = "no" ]
    [ "$GPU_TYPE" = "auto" ]
    [ "$WORKFLOW_PATH" = "" ]
    [ "$OUTPUT_DIR" = "" ]
    [ "$PROMPT_ID" = "" ]
}

# Test argument parsing with custom values
@test "comfyui::parse_arguments handles custom values" {
    comfyui::parse_arguments \
        --action install \
        --force yes \
        --gpu nvidia \
        --workflow "/path/to/workflow.json" \
        --output "/path/to/output" \
        --prompt-id "test-123"
    
    [ "$ACTION" = "install" ]
    [ "$FORCE" = "yes" ]
    [ "$GPU_TYPE" = "nvidia" ]
    [ "$WORKFLOW_PATH" = "/path/to/workflow.json" ]
    [ "$OUTPUT_DIR" = "/path/to/output" ]
    [ "$PROMPT_ID" = "test-123" ]
}

# Test GPU type options
@test "comfyui::parse_arguments validates GPU types" {
    comfyui::parse_arguments --action install --gpu nvidia
    [ "$GPU_TYPE" = "nvidia" ]
    
    comfyui::parse_arguments --action install --gpu amd
    [ "$GPU_TYPE" = "amd" ]
    
    comfyui::parse_arguments --action install --gpu cpu
    [ "$GPU_TYPE" = "cpu" ]
    
    comfyui::parse_arguments --action install --gpu auto
    [ "$GPU_TYPE" = "auto" ]
}

# Test action validation
@test "comfyui::parse_arguments handles all valid actions" {
    local actions=(
        "install" "uninstall" "start" "stop" "restart" 
        "status" "logs" "info" "download-models" "list-models"
        "execute-workflow" "import-workflow" "gpu-info" 
        "validate-nvidia" "check-ready" "cleanup-help"
    )
    
    for action in "${actions[@]}"; do
        comfyui::parse_arguments --action "$action"
        [ "$ACTION" = "$action" ]
    done
}

# Test environment variable override
@test "comfyui::parse_arguments respects COMFYUI_GPU_TYPE override" {
    export COMFYUI_GPU_TYPE="nvidia"
    
    comfyui::parse_arguments --action install --gpu cpu
    
    # Environment variable should override command line
    [ "$GPU_TYPE" = "nvidia" ]
}

# Test workflow-related arguments
@test "comfyui::parse_arguments handles workflow execution arguments" {
    comfyui::parse_arguments \
        --action execute-workflow \
        --workflow "/workflows/test.json" \
        --output "/output/images"
    
    [ "$ACTION" = "execute-workflow" ]
    [ "$WORKFLOW_PATH" = "/workflows/test.json" ]
    [ "$OUTPUT_DIR" = "/output/images" ]
}

# Test import workflow arguments
@test "comfyui::parse_arguments handles import workflow arguments" {
    comfyui::parse_arguments \
        --action import-workflow \
        --workflow "/workflows/new.json"
    
    [ "$ACTION" = "import-workflow" ]
    [ "$WORKFLOW_PATH" = "/workflows/new.json" ]
}

# Test prompt ID argument
@test "comfyui::parse_arguments handles prompt-id correctly" {
    comfyui::parse_arguments \
        --action execute-workflow \
        --prompt-id "unique-12345"
    
    [ "$PROMPT_ID" = "unique-12345" ]
}

# Test configuration export
@test "configuration values are set correctly" {
    # Configuration should be available after sourcing
    [ -n "$COMFYUI_RESOURCE_NAME" ]
    [ "$COMFYUI_RESOURCE_NAME" = "comfyui" ]
    [ -n "$COMFYUI_CONTAINER_NAME" ]
    [ "$COMFYUI_CONTAINER_NAME" = "comfyui" ]
    [ -n "$COMFYUI_DEFAULT_PORT" ]
    [ "$COMFYUI_DEFAULT_PORT" = "5679" ]
    [ -n "$COMFYUI_DIRECT_PORT" ]
    [ "$COMFYUI_DIRECT_PORT" = "8188" ]
}

# Test API configuration
@test "API configuration is set correctly" {
    [ -n "$COMFYUI_API_BASE" ]
    [ "$COMFYUI_API_BASE" = "http://localhost:8188" ]
    [ -n "$COMFYUI_HEALTH_ENDPOINT" ]
    [ "$COMFYUI_HEALTH_ENDPOINT" = "/system_stats" ]
}

# Test model configuration arrays
@test "model configuration arrays are set correctly" {
    [ -n "$COMFYUI_DEFAULT_MODELS" ]
    [ "${#COMFYUI_DEFAULT_MODELS[@]}" -eq 2 ]
    [ -n "$COMFYUI_MODEL_NAMES" ]
    [ "${#COMFYUI_MODEL_NAMES[@]}" -eq 2 ]
    [ "${COMFYUI_MODEL_NAMES[0]}" = "sd_xl_base_1.0.safetensors" ]
    [ "${COMFYUI_MODEL_NAMES[1]}" = "sdxl_vae.safetensors" ]
}

# Test GPU configuration constants
@test "GPU configuration constants are set correctly" {
    [ -n "$COMFYUI_GPU_TYPES" ]
    [ "${#COMFYUI_GPU_TYPES[@]}" -eq 4 ]
    [[ " ${COMFYUI_GPU_TYPES[@]} " =~ " nvidia " ]]
    [[ " ${COMFYUI_GPU_TYPES[@]} " =~ " amd " ]]
    [[ " ${COMFYUI_GPU_TYPES[@]} " =~ " cpu " ]]
    [[ " ${COMFYUI_GPU_TYPES[@]} " =~ " auto " ]]
}

# Test resource requirements
@test "resource requirements are set correctly" {
    [ -n "$COMFYUI_MIN_RAM_GB" ]
    [ "$COMFYUI_MIN_RAM_GB" -eq 16 ]
    [ -n "$COMFYUI_MIN_DISK_GB" ]
    [ "$COMFYUI_MIN_DISK_GB" -eq 50 ]
}

# Test container existence check (mocked)
@test "comfyui::container_exists function exists" {
    declare -f comfyui::container_exists > /dev/null
    [ "$?" -eq 0 ]
}

# Test running check (mocked)
@test "comfyui::is_running function exists" {
    declare -f comfyui::is_running > /dev/null
    [ "$?" -eq 0 ]
}

# Test health check (mocked)
@test "comfyui::is_healthy function exists" {
    declare -f comfyui::is_healthy > /dev/null
    [ "$?" -eq 0 ]
}

# Test edge cases
@test "comfyui::parse_arguments handles empty arguments" {
    comfyui::parse_arguments
    
    # Should use all defaults
    [ "$ACTION" = "install" ]
    [ "$FORCE" = "no" ]
    [ "$GPU_TYPE" = "auto" ]
}

# Test combined options
@test "comfyui::parse_arguments handles combined options" {
    comfyui::parse_arguments \
        --action download-models \
        --force yes \
        --yes yes
    
    [ "$ACTION" = "download-models" ]
    [ "$FORCE" = "yes" ]
    [ "$YES" = "yes" ]
}

# Test invalid action handling (verify it's in the options)
@test "comfyui::parse_arguments action options are comprehensive" {
    # All actions listed in the script should be valid
    local valid_actions=(
        "install" "uninstall" "start" "stop" "restart"
        "status" "logs" "info" "download-models" "list-models"
        "execute-workflow" "import-workflow" "gpu-info"
        "validate-nvidia" "check-ready" "cleanup-help"
    )
    
    for action in "${valid_actions[@]}"; do
        comfyui::parse_arguments --action "$action"
        [ "$ACTION" = "$action" ]
    done
}

# Test workflow path with spaces
@test "comfyui::parse_arguments handles paths with spaces" {
    comfyui::parse_arguments \
        --action execute-workflow \
        --workflow "/path with spaces/workflow.json" \
        --output "/output with spaces/"
    
    [ "$WORKFLOW_PATH" = "/path with spaces/workflow.json" ]
    [ "$OUTPUT_DIR" = "/output with spaces/" ]
}

# Test model metadata arrays consistency
@test "model metadata arrays have consistent lengths" {
    [ "${#COMFYUI_DEFAULT_MODELS[@]}" -eq "${#COMFYUI_MODEL_NAMES[@]}" ]
    [ "${#COMFYUI_MODEL_NAMES[@]}" -eq "${#COMFYUI_MODEL_SIZES[@]}" ]
    [ "${#COMFYUI_MODEL_SIZES[@]}" -eq "${#COMFYUI_MODEL_SHA256[@]}" ]
}

# Test data directory configuration
@test "data directory is configured correctly" {
    [ -n "$COMFYUI_DATA_DIR" ]
    [ "$COMFYUI_DATA_DIR" = "${HOME}/.comfyui" ]
}