#!/usr/bin/env bats
# Tests for ComfyUI gpu.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export COMFYUI_CUSTOM_PORT="8188"
    export COMFYUI_CONTAINER_NAME="comfyui-test"
    export COMFYUI_BASE_URL="http://localhost:8188"
    export COMFYUI_IMAGE="comfyanonymous/comfyui:latest"
    export COMFYUI_GPU_TYPE="cuda"
    export COMFYUI_VRAM_LIMIT="8"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    COMFYUI_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock system functions
    
    # Mock nvidia-smi
    nvidia-smi() {
        case "${1:-}" in
            "--query-gpu=name,memory.total,memory.used,memory.free"*)
                echo "name, memory.total [MiB], memory.used [MiB], memory.free [MiB]"
                echo "NVIDIA GeForce RTX 4090, 24564 MiB, 2048 MiB, 22516 MiB"
                echo "NVIDIA GeForce RTX 3080, 10240 MiB, 1024 MiB, 9216 MiB"
                ;;
            "--query-gpu=driver_version"*)
                echo "535.104.05"
                ;;
            "--query-gpu=count"*)
                echo "2"
                ;;
            *)
                echo "GPU 0: NVIDIA GeForce RTX 4090 (UUID: GPU-12345678-1234-1234-1234-123456789abc)"
                echo "GPU 1: NVIDIA GeForce RTX 3080 (UUID: GPU-87654321-4321-4321-4321-cba987654321)"
                ;;
        esac
        return 0
    }
    
    # Mock rocm-smi
    rocm-smi() {
        case "${1:-}" in
            "--showproductname")
                echo "GPU[0] : AMD Radeon RX 7900 XTX"
                echo "GPU[1] : AMD Radeon RX 6800 XT"
                ;;
            "--showmeminfo=vram")
                echo "GPU[0] : vram Total Memory (B): 26843545600"
                echo "GPU[0] : vram Total Used Memory (B): 1073741824"
                echo "GPU[1] : vram Total Memory (B): 17179869184"
                echo "GPU[1] : vram Total Used Memory (B): 536870912"
                ;;
            "--showdriverversion")
                echo "Driver version: 5.7.1"
                ;;
            *)
                echo "GPU[0] : Temperature (Sensor edge) (C): 45"
                echo "GPU[1] : Temperature (Sensor edge) (C): 42"
                ;;
        esac
        return 0
    }
    
    # Mock Docker info
    
    # Mock log functions
    
    # Mock system checks
    system::check_gpu_memory() {
        case "${1:-nvidia}" in
            "nvidia")
                echo "Total VRAM: 34804 MiB"
                echo "Available VRAM: 31732 MiB"
                ;;
            "amd")
                echo "Total VRAM: 42048 MiB"
                echo "Available VRAM: 40474 MiB"
                ;;
        esac
    }
    
    # Load configuration and messages
    source "${COMFYUI_DIR}/config/defaults.sh"
    source "${COMFYUI_DIR}/config/messages.sh"
    comfyui::export_config
    comfyui::export_messages
    
    # Load the functions to test
    source "${COMFYUI_DIR}/lib/gpu.sh"
}

# Test NVIDIA GPU detection
@test "comfyui::detect_gpu_type detects NVIDIA GPU" {
    result=$(comfyui::detect_gpu_type)
    
    [[ "$result" =~ "nvidia" ]] || [[ "$result" =~ "NVIDIA" ]]
}

# Test AMD GPU detection
@test "comfyui::detect_gpu_type detects AMD GPU when NVIDIA not available" {
    # Override nvidia-smi to fail
    nvidia-smi() { return 1; }
    system::is_command() {
        case "$1" in
            "nvidia-smi") return 1 ;;
            "rocm-smi") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    result=$(comfyui::detect_gpu_type)
    
    [[ "$result" =~ "amd" ]] || [[ "$result" =~ "AMD" ]]
}

# Test CPU fallback when no GPU available
@test "comfyui::detect_gpu_type falls back to CPU" {
    # Override both GPU commands to fail
    nvidia-smi() { return 1; }
    rocm-smi() { return 1; }
    system::is_command() { return 1; }
    
    result=$(comfyui::detect_gpu_type)
    
    [[ "$result" =~ "cpu" ]] || [[ "$result" =~ "CPU" ]]
}

# Test GPU information gathering
@test "comfyui::get_gpu_info returns detailed GPU information" {
    result=$(comfyui::get_gpu_info)
    
    [[ "$result" =~ "GPU" ]]
    [[ "$result" =~ "NVIDIA" ]] || [[ "$result" =~ "AMD" ]]
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "Memory" ]]
}

# Test VRAM detection
@test "comfyui::get_vram_info returns memory information" {
    result=$(comfyui::get_vram_info)
    
    [[ "$result" =~ "VRAM" ]] || [[ "$result" =~ "Memory" ]]
    [[ "$result" =~ "MiB" ]] || [[ "$result" =~ "GB" ]]
}

# Test VRAM limit validation
@test "comfyui::validate_vram_limit validates memory limits" {
    # Valid limit
    result=$(comfyui::validate_vram_limit "8")
    [[ "$?" -eq 0 ]]
    
    # Invalid limit (too high)
    run comfyui::validate_vram_limit "64"
    [ "$status" -eq 1 ]
    
    # Invalid limit (non-numeric)
    run comfyui::validate_vram_limit "invalid"
    [ "$status" -eq 1 ]
}

# Test NVIDIA driver compatibility
@test "comfyui::check_nvidia_compatibility checks driver version" {
    result=$(comfyui::check_nvidia_compatibility)
    
    [[ "$result" =~ "driver" ]] || [[ "$result" =~ "compatible" ]]
}

# Test AMD ROCm compatibility
@test "comfyui::check_amd_compatibility checks ROCm support" {
    # Override to use AMD GPU
    nvidia-smi() { return 1; }
    system::is_command() {
        case "$1" in
            "nvidia-smi") return 1 ;;
            "rocm-smi") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    result=$(comfyui::check_amd_compatibility)
    
    [[ "$result" =~ "ROCm" ]] || [[ "$result" =~ "AMD" ]] || [[ "$result" =~ "compatible" ]]
}

# Test GPU configuration generation
@test "comfyui::generate_gpu_config creates proper Docker args" {
    export GPU_TYPE="nvidia"
    
    result=$(comfyui::generate_gpu_config)
    
    [[ "$result" =~ "--gpus" ]] || [[ "$result" =~ "NVIDIA" ]]
}

# Test AMD GPU configuration
@test "comfyui::generate_gpu_config handles AMD GPU" {
    export GPU_TYPE="amd"
    
    result=$(comfyui::generate_gpu_config)
    
    [[ "$result" =~ "--device" ]] && [[ "$result" =~ "/dev/kfd" ]]
}

# Test CPU configuration
@test "comfyui::generate_gpu_config handles CPU mode" {
    export GPU_TYPE="cpu"
    
    result=$(comfyui::generate_gpu_config)
    
    [[ "$result" =~ "CUDA_VISIBLE_DEVICES=" ]]
}

# Test GPU memory optimization
@test "comfyui::optimize_gpu_memory configures memory settings" {
    export COMFYUI_VRAM_LIMIT="8"
    export GPU_TYPE="nvidia"
    
    result=$(comfyui::optimize_gpu_memory)
    
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "VRAM" ]]
}

# Test GPU monitoring setup
@test "comfyui::setup_gpu_monitoring configures monitoring" {
    result=$(comfyui::setup_gpu_monitoring)
    
    [[ "$result" =~ "monitoring" ]] || [[ "$result" =~ "setup" ]]
}

# Test GPU temperature monitoring
@test "comfyui::get_gpu_temperature returns temperature data" {
    result=$(comfyui::get_gpu_temperature)
    
    [[ "$result" =~ "temperature" ]] || [[ "$result" =~ "°C" ]] || [[ "$result" =~ "C" ]]
}

# Test GPU utilization monitoring
@test "comfyui::get_gpu_utilization returns utilization data" {
    result=$(comfyui::get_gpu_utilization)
    
    [[ "$result" =~ "utilization" ]] || [[ "$result" =~ "%" ]] || [[ "$result" =~ "usage" ]]
}

# Test multiple GPU detection
@test "comfyui::detect_multiple_gpus handles multiple GPUs" {
    result=$(comfyui::detect_multiple_gpus)
    
    [[ "$result" =~ "GPU" ]]
    # Should detect 2 mock GPUs
    [[ "$result" =~ "2" ]] || [[ "$result" =~ "multiple" ]]
}

# Test GPU selection
@test "comfyui::select_gpu allows GPU selection" {
    result=$(comfyui::select_gpu "0")
    
    [[ "$?" -eq 0 ]]
    [[ "$result" =~ "GPU 0" ]] || [[ "$result" =~ "selected" ]]
}

# Test GPU benchmark
@test "comfyui::benchmark_gpu performs basic GPU test" {
    result=$(comfyui::benchmark_gpu)
    
    [[ "$result" =~ "benchmark" ]] || [[ "$result" =~ "test" ]] || [[ "$result" =~ "performance" ]]
}

# Test Docker runtime detection
@test "comfyui::check_docker_gpu_support checks Docker GPU support" {
    export GPU_RUNTIME_AVAILABLE="yes"
    
    result=$(comfyui::check_docker_gpu_support)
    
    [[ "$result" =~ "GPU support" ]] || [[ "$result" =~ "nvidia runtime" ]]
}

# Test Docker runtime missing
@test "comfyui::check_docker_gpu_support handles missing GPU runtime" {
    export GPU_RUNTIME_AVAILABLE="no"
    
    result=$(comfyui::check_docker_gpu_support)
    
    [[ "$result" =~ "runtime not available" ]] || [[ "$result" =~ "CPU mode" ]]
}

# Test GPU environment setup
@test "comfyui::setup_gpu_environment configures GPU environment" {
    export GPU_TYPE="nvidia"
    
    result=$(comfyui::setup_gpu_environment)
    
    [[ "$result" =~ "NVIDIA" ]] || [[ "$result" =~ "environment" ]]
}

# Test GPU cleanup
@test "comfyui::cleanup_gpu_resources cleans up GPU resources" {
    result=$(comfyui::cleanup_gpu_resources)
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "released" ]]
}

# Test GPU error recovery
@test "comfyui::recover_gpu_error handles GPU errors" {
    result=$(comfyui::recover_gpu_error)
    
    [[ "$result" =~ "recovery" ]] || [[ "$result" =~ "reset" ]]
}

# Test comprehensive GPU status
@test "comfyui::get_comprehensive_gpu_status returns complete status" {
    result=$(comfyui::get_comprehensive_gpu_status)
    
    [[ "$result" =~ "GPU" ]]
    [[ "$result" =~ "driver" ]] || [[ "$result" =~ "memory" ]]
    [[ "$result" =~ "status" ]] || [[ "$result" =~ "available" ]]
}

# Test GPU configuration validation
@test "comfyui::validate_gpu_config validates GPU configuration" {
    export GPU_TYPE="nvidia"
    export COMFYUI_VRAM_LIMIT="8"
    
    result=$(comfyui::validate_gpu_config)
    
    [[ "$?" -eq 0 ]]
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "configuration" ]]
}

# Test GPU diagnostic report
@test "comfyui::generate_gpu_diagnostic creates diagnostic report" {
    result=$(comfyui::generate_gpu_diagnostic)
    
    [[ "$result" =~ "diagnostic" ]] || [[ "$result" =~ "report" ]]
    [[ "$result" =~ "GPU" ]]
}

# Test VRAM optimization suggestions
@test "comfyui::suggest_vram_optimization provides optimization tips" {
    export COMFYUI_VRAM_LIMIT="4"
    
    result=$(comfyui::suggest_vram_optimization)
    
    [[ "$result" =~ "optimization" ]] || [[ "$result" =~ "suggestion" ]]
    [[ "$result" =~ "VRAM" ]] || [[ "$result" =~ "memory" ]]
}

# Test GPU power management
@test "comfyui::configure_gpu_power_management sets power settings" {
    result=$(comfyui::configure_gpu_power_management)
    
    [[ "$result" =~ "power" ]] || [[ "$result" =~ "performance" ]]
}