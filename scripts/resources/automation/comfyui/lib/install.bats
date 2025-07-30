#!/usr/bin/env bats
# Tests for ComfyUI install.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export COMFYUI_CUSTOM_PORT="8188"
    export COMFYUI_CONTAINER_NAME="comfyui-test"
    export COMFYUI_BASE_URL="http://localhost:8188"
    export COMFYUI_IMAGE="comfyanonymous/comfyui:latest"
    export COMFYUI_GPU_TYPE="cuda"
    export COMFYUI_DATA_DIR="/tmp/comfyui-test"
    export COMFYUI_MODELS_DIR="/tmp/comfyui-test/models"
    export COMFYUI_OUTPUT_DIR="/tmp/comfyui-test/output"
    export YES="no"
    export FORCE="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    COMFYUI_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories
    mkdir -p "$COMFYUI_DATA_DIR" "$COMFYUI_MODELS_DIR" "$COMFYUI_OUTPUT_DIR"
    
    # Mock system functions
    
    system::check_port() {
        local port="$1"
        # Mock port as available unless specifically testing conflicts
        if [[ "$port" == "9999" ]]; then
            return 1  # Port busy for conflict tests
        fi
        return 0
    }
    
    system::get_available_port() {
        echo "8189"  # Mock alternative port
    }
    
    # Mock Docker functions
    
    # Mock wget/curl
    wget() {
        echo "WGET: $*"
        return 0
    }
    
    
    # Mock filesystem operations
    mkdir() {
        echo "MKDIR: $*"
        return 0
    }
    
    # Override to avoid actual directory creation in setup
    unset -f mkdir
    
    # Mock log functions
    
    # Mock ComfyUI utility functions
    comfyui::container_exists() { return 1; }  # Default: no existing container
    comfyui::is_running() { return 0; }
    comfyui::is_healthy() { return 0; }
    comfyui::detect_gpu_type() { echo "nvidia"; }
    comfyui::get_docker_image() { echo "comfyanonymous/comfyui:latest-cuda"; }
    
    # Mock resources functions
    resources::add_rollback_action() {
        echo "ROLLBACK_ACTION: $1 -> $2"
        return 0
    }
    
    # Load configuration and messages
    source "${COMFYUI_DIR}/config/defaults.sh"
    source "${COMFYUI_DIR}/config/messages.sh"
    comfyui::export_config
    comfyui::export_messages
    
    # Load the functions to test
    source "${COMFYUI_DIR}/lib/install.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$COMFYUI_DATA_DIR"
}

# Test installation prerequisites check
@test "comfyui::check_prerequisites validates system requirements" {
    result=$(comfyui::check_prerequisites)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Docker" ]]
}

# Test prerequisites failure
@test "comfyui::check_prerequisites handles missing dependencies" {
    # Override to simulate missing Docker
    system::is_command() {
        case "$1" in
            "docker") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    run comfyui::check_prerequisites
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Docker" ]]
}

# Test directory creation
@test "comfyui::create_directories creates required directories" {
    result=$(comfyui::create_directories)
    
    [[ "$result" =~ "Creating" ]] || [[ "$result" =~ "directory" ]]
    [[ "$result" =~ "$COMFYUI_DATA_DIR" ]] || [[ "$result" =~ "data" ]]
}

# Test directory creation with existing directories
@test "comfyui::create_directories handles existing directories" {
    # Create directories first
    mkdir -p "$COMFYUI_DATA_DIR" "$COMFYUI_MODELS_DIR" "$COMFYUI_OUTPUT_DIR"
    
    result=$(comfyui::create_directories)
    
    [[ "$?" -eq 0 ]]
    [[ "$result" =~ "already exists" ]] || [[ "$result" =~ "directory" ]]
}

# Test port availability check
@test "comfyui::check_port_availability validates port availability" {
    result=$(comfyui::check_port_availability)
    
    [[ "$?" -eq 0 ]]
    [[ "$result" =~ "port" ]] || [[ "$result" =~ "available" ]]
}

# Test port conflict handling
@test "comfyui::check_port_availability handles port conflicts" {
    export COMFYUI_CUSTOM_PORT="9999"  # Mock busy port
    
    run comfyui::check_port_availability
    [ "$status" -eq 1 ]
    [[ "$output" =~ "port" ]]
    [[ "$output" =~ "in use" ]] || [[ "$output" =~ "busy" ]]
}

# Test image selection for NVIDIA
@test "comfyui::select_image chooses correct NVIDIA image" {
    export COMFYUI_GPU_TYPE="nvidia"
    
    result=$(comfyui::select_image)
    
    [[ "$result" =~ "cuda" ]] || [[ "$result" =~ "nvidia" ]]
}

# Test image selection for AMD
@test "comfyui::select_image chooses correct AMD image" {
    export COMFYUI_GPU_TYPE="amd"
    
    result=$(comfyui::select_image)
    
    [[ "$result" =~ "rocm" ]] || [[ "$result" =~ "amd" ]]
}

# Test image selection for CPU
@test "comfyui::select_image chooses correct CPU image" {
    export COMFYUI_GPU_TYPE="cpu"
    
    result=$(comfyui::select_image)
    
    [[ "$result" =~ "cpu" ]] || [[ "$result" =~ "latest" ]]
}

# Test Docker image pull
@test "comfyui::pull_docker_image downloads container image" {
    result=$(comfyui::pull_docker_image)
    
    [[ "$result" =~ "Pulling" ]] || [[ "$result" =~ "DOCKER_PULL" ]]
    [[ "$result" =~ "comfyanonymous/comfyui" ]]
}

# Test Docker image pull failure
@test "comfyui::pull_docker_image handles pull failure" {
    # Override docker to fail on pull
    docker() {
        case "$1" in
            "pull") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    run comfyui::pull_docker_image
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test container configuration
@test "comfyui::configure_container sets up container configuration" {
    result=$(comfyui::configure_container)
    
    [[ "$result" =~ "config" ]] || [[ "$result" =~ "container" ]]
    [[ "$result" =~ "8188" ]]  # Port configuration
}

# Test volume configuration
@test "comfyui::setup_volumes configures data volumes" {
    result=$(comfyui::setup_volumes)
    
    [[ "$result" =~ "volume" ]] || [[ "$result" =~ "mount" ]]
    [[ "$result" =~ "$COMFYUI_DATA_DIR" ]]
}

# Test environment variables setup
@test "comfyui::setup_environment configures environment variables" {
    result=$(comfyui::setup_environment)
    
    [[ "$result" =~ "environment" ]] || [[ "$result" =~ "variable" ]]
}

# Test full installation process
@test "comfyui::install performs complete installation" {
    result=$(comfyui::install)
    
    [[ "$result" =~ "Installing" ]]
    [[ "$result" =~ "ComfyUI" ]]
    [[ "$result" =~ "SUCCESS:" ]] || [[ "$result" =~ "successful" ]]
}

# Test installation with existing container
@test "comfyui::install handles existing container" {
    # Override to simulate existing container
    comfyui::container_exists() { return 0; }
    
    result=$(comfyui::install)
    
    [[ "$result" =~ "already" ]] || [[ "$result" =~ "exists" ]]
}

# Test force installation
@test "comfyui::install performs force installation" {
    export FORCE="yes"
    comfyui::container_exists() { return 0; }
    
    result=$(comfyui::install)
    
    [[ "$result" =~ "force" ]] || [[ "$result" =~ "Installing" ]]
}

# Test default model download
@test "comfyui::download_default_models downloads essential models" {
    result=$(comfyui::download_default_models)
    
    [[ "$result" =~ "model" ]] || [[ "$result" =~ "download" ]]
    [[ "$result" =~ "WGET:" ]] || [[ "$result" =~ "CURL:" ]]
}

# Test model download failure handling
@test "comfyui::download_default_models handles download failures" {
    # Override wget to fail
    wget() { return 1; }
    curl() { return 1; }
    
    run comfyui::download_default_models
    # Should not fail the installation, just warn
    [ "$status" -eq 0 ]
    [[ "$output" =~ "WARN:" ]] || [[ "$output" =~ "failed" ]]
}

# Test custom node installation
@test "comfyui::install_custom_nodes installs additional nodes" {
    result=$(comfyui::install_custom_nodes)
    
    [[ "$result" =~ "custom" ]] || [[ "$result" =~ "node" ]]
}

# Test health check after installation
@test "comfyui::verify_installation checks installation health" {
    result=$(comfyui::verify_installation)
    
    [[ "$result" =~ "verify" ]] || [[ "$result" =~ "health" ]]
    [[ "$result" =~ "healthy" ]] || [[ "$result" =~ "ready" ]]
}

# Test installation rollback
@test "comfyui::rollback_installation cleans up failed installation" {
    result=$(comfyui::rollback_installation)
    
    [[ "$result" =~ "rollback" ]] || [[ "$result" =~ "cleanup" ]]
}

# Test installation status reporting
@test "comfyui::report_installation_status provides status information" {
    result=$(comfyui::report_installation_status)
    
    [[ "$result" =~ "ComfyUI" ]]
    [[ "$result" =~ "http://localhost:8188" ]]
    [[ "$result" =~ "Web UI" ]] || [[ "$result" =~ "access" ]]
}

# Test configuration file creation
@test "comfyui::create_config_files generates configuration files" {
    result=$(comfyui::create_config_files)
    
    [[ "$result" =~ "config" ]] || [[ "$result" =~ "file" ]]
}

# Test backup creation during installation
@test "comfyui::create_installation_backup creates backup before install" {
    result=$(comfyui::create_installation_backup)
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "created" ]]
}

# Test GPU driver validation
@test "comfyui::validate_gpu_drivers checks GPU driver compatibility" {
    result=$(comfyui::validate_gpu_drivers)
    
    [[ "$result" =~ "driver" ]] || [[ "$result" =~ "GPU" ]]
}

# Test disk space check
@test "comfyui::check_disk_space validates available disk space" {
    result=$(comfyui::check_disk_space)
    
    [[ "$result" =~ "disk" ]] || [[ "$result" =~ "space" ]]
}

# Test memory requirements check
@test "comfyui::check_memory_requirements validates system memory" {
    result=$(comfyui::check_memory_requirements)
    
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "RAM" ]]
}

# Test network connectivity check
@test "comfyui::check_network_connectivity validates internet access" {
    result=$(comfyui::check_network_connectivity)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "connectivity" ]]
}

# Test Docker daemon check
@test "comfyui::check_docker_daemon validates Docker service" {
    result=$(comfyui::check_docker_daemon)
    
    [[ "$result" =~ "Docker" ]] || [[ "$result" =~ "daemon" ]]
}

# Test installation preparation
@test "comfyui::prepare_installation sets up installation environment" {
    result=$(comfyui::prepare_installation)
    
    [[ "$result" =~ "prepare" ]] || [[ "$result" =~ "setup" ]]
}

# Test installation cleanup
@test "comfyui::cleanup_installation removes temporary files" {
    result=$(comfyui::cleanup_installation)
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "clean" ]]
}

# Test installation progress tracking
@test "comfyui::track_installation_progress monitors installation steps" {
    result=$(comfyui::track_installation_progress "Downloading models")
    
    [[ "$result" =~ "progress" ]] || [[ "$result" =~ "Downloading" ]]
}

# Test installation error recovery
@test "comfyui::recover_from_installation_error handles installation failures" {
    result=$(comfyui::recover_from_installation_error "Docker pull failed")
    
    [[ "$result" =~ "recover" ]] || [[ "$result" =~ "error" ]]
}

# Test post-installation configuration
@test "comfyui::post_install_configuration configures installed instance" {
    result=$(comfyui::post_install_configuration)
    
    [[ "$result" =~ "post" ]] || [[ "$result" =~ "configuration" ]]
}

# Test installation summary
@test "comfyui::generate_installation_summary creates installation report" {
    result=$(comfyui::generate_installation_summary)
    
    [[ "$result" =~ "summary" ]] || [[ "$result" =~ "report" ]]
    [[ "$result" =~ "ComfyUI" ]]
}