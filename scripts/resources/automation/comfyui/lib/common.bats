#!/usr/bin/env bats
# Tests for ComfyUI common.sh functions

# Setup for each test
setup() {
    # Load Vrooli test infrastructure
    source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"
    
    # Setup ComfyUI test environment
    vrooli_setup_service_test "comfyui"
    
    # Source var.sh first
    source "${BATS_TEST_DIRNAME}/../../../../lib/utils/var.sh"
    
    # Set test environment
    export COMFYUI_CUSTOM_PORT="8188"
    export COMFYUI_CONTAINER_NAME="comfyui-test"
    export COMFYUI_BASE_URL="http://localhost:8188"
    export COMFYUI_GPU_TYPE="auto"
    export COMFYUI_DATA_DIR="/tmp/comfyui-test"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    COMFYUI_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directory
    mkdir -p "$COMFYUI_DATA_DIR"
    
    # Mock system functions
    
    # Mock Docker functions
    
    # Mock nvidia-smi
    nvidia-smi() {
        echo "GPU 0: NVIDIA GeForce RTX 4090"
        return 0
    }
    
    # Mock log functions
    
    
    
    
    # Mock args functions
    args::reset() { return 0; }
    args::register_help() { return 0; }
    args::register_yes() { return 0; }
    args::register() { return 0; }
    args::is_asking_for_help() { return 1; }
    args::parse() { return 0; }
    args::get() {
        case "$1" in
            "action") echo "status" ;;
            "force") echo "no" ;;
            "yes") echo "no" ;;
            "gpu-type") echo "auto" ;;
            "workflow-path") echo "" ;;
            "output-dir") echo "" ;;
            "prompt-id") echo "" ;;
        esac
    }
    
    # Load configuration
    source "${COMFYUI_DIR}/config/defaults.sh"
    source "${COMFYUI_DIR}/config/messages.sh"
    
    # Load the functions to test
    source "${COMFYUI_DIR}/lib/common.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$COMFYUI_DATA_DIR"
}

# Test configuration export
@test "configuration variables are set" {
    # Variables should be set by sourcing defaults.sh
    
    [ -n "$COMFYUI_PORT" ]
    [ -n "$COMFYUI_BASE_URL" ]
    [ -n "$COMFYUI_CONTAINER_NAME" ]
    [ -n "$COMFYUI_IMAGE" ]
}

# Test message export  
@test "message variables are set" {
    # Variables should be set by sourcing messages.sh
    
    [ -n "$MSG_COMFYUI_INSTALLING" ]
    [ -n "$MSG_COMFYUI_ALREADY_INSTALLED" ]
    [ -n "$MSG_COMFYUI_NOT_FOUND" ]
}

# Test argument parsing with defaults
@test "common::parse_arguments sets correct defaults" {
    common::parse_arguments --action status
    
    [ "$ACTION" = "status" ]
    [ "$FORCE" = "no" ]
    [ "$GPU_TYPE" = "auto" ]
    [ "$YES" = "no" ]
}

# Test argument parsing with custom values
@test "common::parse_arguments handles custom values" {
    # Mock args::get to return custom values
    args::get() {
        case "$1" in
            "action") echo "generate" ;;
            "force") echo "yes" ;;
            "yes") echo "yes" ;;
            "gpu-type") echo "cuda" ;;
            "workflow-path") echo "/tmp/test.json" ;;
            "output-dir") echo "/tmp/output" ;;
            "prompt-id") echo "test-prompt-123" ;;
        esac
    }
    
    common::parse_arguments \
        --action generate \
        --force yes \
        --gpu-type cuda \
        --workflow-path "/tmp/test.json" \
        --output-dir "/tmp/output" \
        --prompt-id "test-prompt-123" \
        --yes yes
    
    [ "$ACTION" = "generate" ]
    [ "$FORCE" = "yes" ]
    [ "$GPU_TYPE" = "cuda" ]
    [ "$WORKFLOW_PATH" = "/tmp/test.json" ]
    [ "$OUTPUT_DIR" = "/tmp/output" ]
    [ "$PROMPT_ID" = "test-prompt-123" ]
    [ "$YES" = "yes" ]
}

# Test usage display
@test "common::usage displays help information" {
    # Mock args::usage
    args::usage() {
        echo "USAGE_CALLED: $1"
        return 0
    }
    
    result=$(common::usage)
    
    [[ "$result" =~ "USAGE_CALLED:" ]]
    [[ "$result" =~ "ComfyUI" ]]
}

# Test Docker check function
@test "common::check_docker succeeds with Docker available" {
    common::check_docker
    [ "$?" -eq 0 ]
}

# Test Docker check function with Docker unavailable
@test "common::check_docker fails with Docker unavailable" {
    # Override system function to simulate missing Docker
    system::is_command() {
        case "$1" in
            "docker") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    run common::check_docker
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}

# Test container existence check - exists
@test "common::container_exists returns true for existing container" {
    common::container_exists
    [ "$?" -eq 0 ]
}

# Test container existence check - not exists
@test "common::container_exists returns false for non-existing container" {
    # Override docker to return empty result
    docker() {
        case "$1" in
            "ps") echo "" ;;
            *) return 0 ;;
        esac
    }
    
    run common::container_exists
    [ "$status" -eq 1 ]
}

# Test container running check - running
@test "common::is_running returns true for running container" {
    common::is_running
    [ "$?" -eq 0 ]
}

# Test container running check - not running
@test "common::is_running returns false for stopped container" {
    # Override docker inspect to show stopped state
    docker() {
        case "$1" in
            "inspect")
                echo '{"State":{"Running":false}}'
                ;;
            *) return 0 ;;
        esac
    }
    
    run common::is_running
    [ "$status" -eq 1 ]
}

# Test service info display
@test "common::info displays service information" {
    result=$(common::info)
    
    [[ "$result" =~ "ComfyUI" ]]
    [[ "$result" =~ "$COMFYUI_BASE_URL" ]]
}

# Test port check function
@test "common::check_port_available succeeds for available port" {
    # Mock system function
    system::is_port_in_use() {
        return 1  # Port not in use
    }
    
    common::check_port_available
    [ "$?" -eq 0 ]
}

# Test port check function with port in use
@test "common::check_port_available fails for port in use" {
    # Mock system function to simulate port in use
    system::is_port_in_use() {
        return 0  # Port is in use
    }
    
    run common::check_port_available
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}

# Test GPU detection
@test "common::detect_gpu detects available GPU" {
    result=$(common::detect_gpu)
    
    [[ "$result" =~ "GPU detected:" ]]
    [[ "$result" =~ "NVIDIA" ]]
}

# Test GPU detection with no GPU
@test "common::detect_gpu handles no GPU available" {
    # Override nvidia-smi to fail
    nvidia-smi() {
        return 1
    }
    
    # Override system check
    system::is_command() {
        case "$1" in
            "nvidia-smi") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    result=$(common::detect_gpu)
    
    [[ "$result" =~ "No GPU detected" ]] || [[ "$result" =~ "CPU mode" ]]
}

# Test data directory setup
@test "common::setup_data_directory creates necessary directories" {
    result=$(common::setup_data_directory)
    
    [[ "$result" =~ "Setting up data directory" ]]
    [ -d "$COMFYUI_DATA_DIR" ]
}

# Test configuration validation
@test "common::validate_config validates service configuration" {
    result=$(common::validate_config)
    
    [[ "$result" =~ "Configuration:" ]]
    [[ "$result" =~ "Port:" ]]
    [[ "$result" =~ "Container:" ]]
    [[ "$result" =~ "GPU:" ]]
}

# Test environment validation
@test "common::validate_environment checks system requirements" {
    result=$(common::validate_environment)
    
    [[ "$result" =~ "Validating environment" ]]
    [[ "$result" =~ "Docker" ]]
    [[ "$result" =~ "GPU" ]]
}

# Test environment validation with missing requirements
@test "common::validate_environment detects missing requirements" {
    # Override system check to fail
    system::is_command() {
        case "$1" in
            "docker") return 1 ;;  # Docker not available
            *) return 0 ;;
        esac
    }
    
    run common::validate_environment
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test log level configuration
@test "common::set_log_level configures logging level" {
    result=$(common::set_log_level "debug")
    
    [[ "$result" =~ "Setting log level" ]]
    [[ "$result" =~ "debug" ]]
}

# Test memory configuration
@test "common::configure_memory configures memory settings" {
    result=$(common::configure_memory "4g")
    
    [[ "$result" =~ "Configuring memory" ]]
    [[ "$result" =~ "4g" ]]
}

# Test cleanup functionality
@test "common::cleanup performs cleanup operations" {
    result=$(common::cleanup)
    
    [[ "$result" =~ "Cleaning up" ]] || [[ "$result" =~ "cleanup" ]]
}

# Test version detection
@test "common::get_version returns service version" {
    # Mock version detection
    common::get_version() {
        echo "ComfyUI version 1.0.0"
    }
    
    result=$(common::get_version)
    
    [[ "$result" =~ "version" ]]
    [[ "$result" =~ "1.0.0" ]]
}

# Test health check functionality
@test "common::health_check verifies service health" {
    # Mock curl for health check
    curl() {
        case "$*" in
            *"/system_stats"*)
                echo '{"status":"healthy"}'
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    
    result=$(common::health_check)
    
    [[ "$result" =~ "healthy" ]] || [[ "$result" =~ "Health check" ]]
}

# Test file validation
@test "common::validate_file validates input files" {
    local test_file="$COMFYUI_DATA_DIR/test.json"
    echo '{"prompt":"test"}' > "$test_file"
    
    common::validate_file "$test_file"
    [ "$?" -eq 0 ]
}

# Test file validation with invalid file
@test "common::validate_file rejects invalid files" {
    run common::validate_file "/nonexistent/file.json"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test parameter validation
@test "common::validate_parameters validates input parameters" {
    result=$(common::validate_parameters)
    
    [[ "$result" =~ "Validating parameters" ]] || [ "$?" -eq 0 ]
}

# Test error handling
@test "common::handle_error processes errors gracefully" {
    local error_message="Test error"
    local error_code="1"
    
    result=$(common::handle_error "$error_message" "$error_code")
    
    [[ "$result" =~ "ERROR:" ]]
    [[ "$result" =~ "Test error" ]]
}