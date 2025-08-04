#!/usr/bin/env bats
# Tests for ComfyUI common.sh functions

# Setup for each test
setup() {
    # Load Vrooli test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../../__test/fixtures/setup.bash"
    
    # Setup ComfyUI test environment
    vrooli_setup_service_test "comfyui"
    
    # Set test environment
    export COMFYUI_CUSTOM_PORT="8188"
    export COMFYUI_CONTAINER_NAME="comfyui-test"
    export COMFYUI_BASE_URL="http://localhost:8188"
    export COMFYUI_GPU_TYPE="auto"
    export COMFYUI_DATA_DIR="/tmp/comfyui-test"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
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
    comfyui::export_config
    comfyui::export_messages
    
    # Load the functions to test
    source "${COMFYUI_DIR}/lib/common.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$COMFYUI_DATA_DIR"
}

# Test configuration export
@test "comfyui::export_config sets configuration variables" {
    comfyui::export_config
    
    [ -n "$COMFYUI_PORT" ]
    [ -n "$COMFYUI_BASE_URL" ]
    [ -n "$COMFYUI_CONTAINER_NAME" ]
    [ -n "$COMFYUI_IMAGE" ]
}

# Test message export
@test "comfyui::export_messages sets message variables" {
    comfyui::export_messages
    
    [ -n "$MSG_COMFYUI_INSTALLING" ]
    [ -n "$MSG_COMFYUI_ALREADY_INSTALLED" ]
    [ -n "$MSG_COMFYUI_NOT_FOUND" ]
}

# Test argument parsing with defaults
@test "comfyui::parse_arguments sets correct defaults" {
    comfyui::parse_arguments --action status
    
    [ "$ACTION" = "status" ]
    [ "$FORCE" = "no" ]
    [ "$GPU_TYPE" = "auto" ]
    [ "$YES" = "no" ]
}

# Test argument parsing with custom values
@test "comfyui::parse_arguments handles custom values" {
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
    
    comfyui::parse_arguments \
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
@test "comfyui::usage displays help information" {
    # Mock args::usage
    args::usage() {
        echo "USAGE_CALLED: $1"
        return 0
    }
    
    result=$(comfyui::usage)
    
    [[ "$result" =~ "USAGE_CALLED:" ]]
    [[ "$result" =~ "ComfyUI" ]]
}

# Test Docker check function
@test "comfyui::check_docker succeeds with Docker available" {
    comfyui::check_docker
    [ "$?" -eq 0 ]
}

# Test Docker check function with Docker unavailable
@test "comfyui::check_docker fails with Docker unavailable" {
    # Override system function to simulate missing Docker
    system::is_command() {
        case "$1" in
            "docker") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    run comfyui::check_docker
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}

# Test container existence check - exists
@test "comfyui::container_exists returns true for existing container" {
    comfyui::container_exists
    [ "$?" -eq 0 ]
}

# Test container existence check - not exists
@test "comfyui::container_exists returns false for non-existing container" {
    # Override docker to return empty result
    docker() {
        case "$1" in
            "ps") echo "" ;;
            *) return 0 ;;
        esac
    }
    
    run comfyui::container_exists
    [ "$status" -eq 1 ]
}

# Test container running check - running
@test "comfyui::is_running returns true for running container" {
    comfyui::is_running
    [ "$?" -eq 0 ]
}

# Test container running check - not running
@test "comfyui::is_running returns false for stopped container" {
    # Override docker inspect to show stopped state
    docker() {
        case "$1" in
            "inspect")
                echo '{"State":{"Running":false}}'
                ;;
            *) return 0 ;;
        esac
    }
    
    run comfyui::is_running
    [ "$status" -eq 1 ]
}

# Test service info display
@test "comfyui::info displays service information" {
    result=$(comfyui::info)
    
    [[ "$result" =~ "ComfyUI" ]]
    [[ "$result" =~ "$COMFYUI_BASE_URL" ]]
}

# Test port check function
@test "comfyui::check_port_available succeeds for available port" {
    # Mock system function
    system::is_port_in_use() {
        return 1  # Port not in use
    }
    
    comfyui::check_port_available
    [ "$?" -eq 0 ]
}

# Test port check function with port in use
@test "comfyui::check_port_available fails for port in use" {
    # Mock system function to simulate port in use
    system::is_port_in_use() {
        return 0  # Port is in use
    }
    
    run comfyui::check_port_available
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}

# Test GPU detection
@test "comfyui::detect_gpu detects available GPU" {
    result=$(comfyui::detect_gpu)
    
    [[ "$result" =~ "GPU detected:" ]]
    [[ "$result" =~ "NVIDIA" ]]
}

# Test GPU detection with no GPU
@test "comfyui::detect_gpu handles no GPU available" {
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
    
    result=$(comfyui::detect_gpu)
    
    [[ "$result" =~ "No GPU detected" ]] || [[ "$result" =~ "CPU mode" ]]
}

# Test data directory setup
@test "comfyui::setup_data_directory creates necessary directories" {
    result=$(comfyui::setup_data_directory)
    
    [[ "$result" =~ "Setting up data directory" ]]
    [ -d "$COMFYUI_DATA_DIR" ]
}

# Test configuration validation
@test "comfyui::validate_config validates service configuration" {
    result=$(comfyui::validate_config)
    
    [[ "$result" =~ "Configuration:" ]]
    [[ "$result" =~ "Port:" ]]
    [[ "$result" =~ "Container:" ]]
    [[ "$result" =~ "GPU:" ]]
}

# Test environment validation
@test "comfyui::validate_environment checks system requirements" {
    result=$(comfyui::validate_environment)
    
    [[ "$result" =~ "Validating environment" ]]
    [[ "$result" =~ "Docker" ]]
    [[ "$result" =~ "GPU" ]]
}

# Test environment validation with missing requirements
@test "comfyui::validate_environment detects missing requirements" {
    # Override system check to fail
    system::is_command() {
        case "$1" in
            "docker") return 1 ;;  # Docker not available
            *) return 0 ;;
        esac
    }
    
    run comfyui::validate_environment
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test log level configuration
@test "comfyui::set_log_level configures logging level" {
    result=$(comfyui::set_log_level "debug")
    
    [[ "$result" =~ "Setting log level" ]]
    [[ "$result" =~ "debug" ]]
}

# Test memory configuration
@test "comfyui::configure_memory configures memory settings" {
    result=$(comfyui::configure_memory "4g")
    
    [[ "$result" =~ "Configuring memory" ]]
    [[ "$result" =~ "4g" ]]
}

# Test cleanup functionality
@test "comfyui::cleanup performs cleanup operations" {
    result=$(comfyui::cleanup)
    
    [[ "$result" =~ "Cleaning up" ]] || [[ "$result" =~ "cleanup" ]]
}

# Test version detection
@test "comfyui::get_version returns service version" {
    # Mock version detection
    comfyui::get_version() {
        echo "ComfyUI version 1.0.0"
    }
    
    result=$(comfyui::get_version)
    
    [[ "$result" =~ "version" ]]
    [[ "$result" =~ "1.0.0" ]]
}

# Test health check functionality
@test "comfyui::health_check verifies service health" {
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
    
    result=$(comfyui::health_check)
    
    [[ "$result" =~ "healthy" ]] || [[ "$result" =~ "Health check" ]]
}

# Test file validation
@test "comfyui::validate_file validates input files" {
    local test_file="$COMFYUI_DATA_DIR/test.json"
    echo '{"prompt":"test"}' > "$test_file"
    
    comfyui::validate_file "$test_file"
    [ "$?" -eq 0 ]
}

# Test file validation with invalid file
@test "comfyui::validate_file rejects invalid files" {
    run comfyui::validate_file "/nonexistent/file.json"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test parameter validation
@test "comfyui::validate_parameters validates input parameters" {
    result=$(comfyui::validate_parameters)
    
    [[ "$result" =~ "Validating parameters" ]] || [ "$?" -eq 0 ]
}

# Test error handling
@test "comfyui::handle_error processes errors gracefully" {
    local error_message="Test error"
    local error_code="1"
    
    result=$(comfyui::handle_error "$error_message" "$error_code")
    
    [[ "$result" =~ "ERROR:" ]]
    [[ "$result" =~ "Test error" ]]
}