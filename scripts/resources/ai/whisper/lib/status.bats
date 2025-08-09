#!/usr/bin/env bats
# Tests for Whisper status.sh functions

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "whisper"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    WHISPER_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Source library files
    source "${WHISPER_DIR}/config/defaults.sh"
    source "${WHISPER_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/status.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_WHISPER_DIR="$WHISPER_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export WHISPER_CUSTOM_PORT="8090"
    export WHISPER_CONTAINER_NAME="whisper-test"
    export WHISPER_BASE_URL="http://localhost:8090"
    export WHISPER_GPU_ENABLED="no"
    export WHISPER_MODEL_SIZE="base"
    export YES="no"
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    WHISPER_DIR="${SETUP_FILE_WHISPER_DIR}""
    
    # Mock system functions
    
    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "whisper") echo "8090" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Mock Docker functions
    
    # Mock curl for health checks
    
    # Mock jq for JSON parsing
    jq() {
        case "$*" in
            *".status"*)
                echo "healthy"
                ;;
            *".model"*)
                echo "base"
                ;;
            *".version"*)
                echo "1.0.0"
                ;;
            *".models"*)
                echo '["base","small","medium","large"]'
                ;;
            *) echo "{}" ;;
        esac
    }
    
    # Mock log functions
    
    
    
    
    
    # Mock common functions
    whisper::check_docker() { return 0; }
    whisper::container_exists() { return 0; }
    whisper::is_running() { return 0; }
    whisper::get_container_port() { echo "8090"; }
    
    # Export config functions
    whisper::export_config
    whisper::export_messages
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test comprehensive status display
@test "whisper::show_status displays comprehensive status information" {
    result=$(whisper::show_status)
    
    [[ "$result" =~ "HEADER: 🎤 Whisper Status" ]]
    [[ "$result" =~ "Docker Status:" ]]
    [[ "$result" =~ "Container Status:" ]]
    [[ "$result" =~ "SUCCESS: ✅ Docker is available" ]]
    [[ "$result" =~ "SUCCESS: ✅ Whisper container is running" ]]
}

# Test status display with Docker unavailable
@test "whisper::show_status handles Docker unavailable" {
    # Override Docker check to fail
    whisper::check_docker() {
        return 1
    }
    
    result=$(whisper::show_status)
    
    [[ "$result" =~ "ERROR: ❌ Docker is not available" ]]
    [[ "$result" =~ "unhealthy" ]]
}

# Test status display with container not running
@test "whisper::show_status handles stopped container" {
    # Override running check to fail
    whisper::is_running() {
        return 1
    }
    
    result=$(whisper::show_status)
    
    [[ "$result" =~ "not running" ]]
}

# Test status display with missing container
@test "whisper::show_status handles missing container" {
    # Override container check to fail
    whisper::container_exists() {
        return 1
    }
    
    result=$(whisper::show_status)
    
    [[ "$result" =~ "not found" ]]
}

# Test quick status check
@test "whisper::quick_status provides brief status" {
    result=$(whisper::quick_status)
    
    [[ "$result" =~ "Whisper" ]]
    [[ "$result" =~ "running" ]] || [[ "$result" =~ "healthy" ]]
}

# Test health check function
@test "whisper::health_check verifies service health" {
    result=$(whisper::health_check)
    
    [[ "$result" =~ "healthy" ]]
}

# Test health check with service unavailable
@test "whisper::health_check handles service unavailable" {
    # Override curl to fail
    curl() {
        return 1
    }
    
    run whisper::health_check
    [ "$status" -eq 1 ]
}

# Test service availability check
@test "whisper::is_service_available checks service accessibility" {
    whisper::is_service_available
    [ "$?" -eq 0 ]
}

# Test service availability with inaccessible service
@test "whisper::is_service_available handles inaccessible service" {
    # Override health check to fail
    whisper::health_check() {
        return 1
    }
    
    run whisper::is_service_available
    [ "$status" -eq 1 ]
}

# Test port status check
@test "whisper::check_port_status verifies port accessibility" {
    result=$(whisper::check_port_status)
    
    [[ "$result" =~ "port" ]] || [[ "$result" =~ "accessible" ]]
}

# Test port status with inaccessible port
@test "whisper::check_port_status handles inaccessible port" {
    # Override curl to fail on port check
    curl() {
        case "$*" in
            *"localhost:8090"*)
                return 1
                ;;
            *) return 0 ;;
        esac
    }
    
    run whisper::check_port_status
    [ "$status" -eq 1 ]
}

# Test container resource usage
@test "whisper::get_resource_usage returns container statistics" {
    result=$(whisper::get_resource_usage)
    
    [[ "$result" =~ "CPU" ]]
    [[ "$result" =~ "MEM" ]]
    [[ "$result" =~ "2.50%" ]]
    [[ "$result" =~ "1.2GiB" ]]
}

# Test container resource usage with missing container
@test "whisper::get_resource_usage handles missing container" {
    # Override container check to fail
    whisper::container_exists() {
        return 1
    }
    
    run whisper::get_resource_usage
    [ "$status" -eq 1 ]
}

# Test version information
@test "whisper::get_version returns service version" {
    result=$(whisper::get_version)
    
    [[ "$result" =~ "1.0.0" ]]
}

# Test version with service unavailable
@test "whisper::get_version handles service unavailable" {
    # Override curl to fail
    curl() {
        return 1
    }
    
    run whisper::get_version
    [ "$status" -eq 1 ]
}

# Test model information
@test "whisper::get_model_info returns model information" {
    result=$(whisper::get_model_info)
    
    [[ "$result" =~ "base" ]]
}

# Test model information with service unavailable
@test "whisper::get_model_info handles service unavailable" {
    # Override curl to fail
    curl() {
        return 1
    }
    
    run whisper::get_model_info
    [ "$status" -eq 1 ]
}

# Test available models
@test "whisper::get_available_models lists available models" {
    result=$(whisper::get_available_models)
    
    [[ "$result" =~ "base" ]]
    [[ "$result" =~ "small" ]]
    [[ "$result" =~ "medium" ]]
    [[ "$result" =~ "large" ]]
}

# Test container network information
@test "whisper::get_network_info returns network details" {
    result=$(whisper::get_network_info)
    
    [[ "$result" =~ "Port:" ]]
    [[ "$result" =~ "8090" ]]
}

# Test container network info with missing container
@test "whisper::get_network_info handles missing container" {
    # Override container check to fail
    whisper::container_exists() {
        return 1
    }
    
    run whisper::get_network_info
    [ "$status" -eq 1 ]
}

# Test service configuration validation
@test "whisper::validate_configuration checks service configuration" {
    result=$(whisper::validate_configuration)
    
    [[ "$result" =~ "Configuration:" ]]
    [[ "$result" =~ "Port:" ]]
    [[ "$result" =~ "Container:" ]]
    [[ "$result" =~ "Model:" ]]
}

# Test uptime calculation
@test "whisper::get_uptime returns service uptime" {
    # Mock container start time
    docker() {
        case "$1" in
            "inspect")
                echo '{"State":{"StartedAt":"2024-01-01T12:00:00Z"}}'
                ;;
            *) return 0 ;;
        esac
    }
    
    # Mock date command
    date() {
        case "$*" in
            *"+%s"*) echo "1704110400" ;;  # Mock current timestamp
            *) echo "2024-01-01 12:02:00" ;;
        esac
    }
    
    result=$(whisper::get_uptime)
    
    [[ "$result" =~ "uptime" ]] || [[ "$result" =~ "minutes" ]] || [[ "$result" =~ "seconds" ]]
}

# Test detailed system information
@test "whisper::get_system_info returns system details" {
    result=$(whisper::get_system_info)
    
    [[ "$result" =~ "System Information:" ]]
    [[ "$result" =~ "Docker:" ]]
    [[ "$result" =~ "GPU:" ]]
}

# Test performance metrics
@test "whisper::get_performance_metrics collects performance data" {
    result=$(whisper::get_performance_metrics)
    
    [[ "$result" =~ "Performance Metrics:" ]]
    [[ "$result" =~ "CPU" ]]
    [[ "$result" =~ "Memory" ]]
}

# Test service endpoints check
@test "whisper::check_endpoints verifies API endpoints" {
    result=$(whisper::check_endpoints)
    
    [[ "$result" =~ "Endpoints:" ]]
    [[ "$result" =~ "/health" ]]
    [[ "$result" =~ "/models" ]]
}

# Test service endpoints with unavailable service
@test "whisper::check_endpoints handles unavailable service" {
    # Override curl to fail
    curl() {
        return 1
    }
    
    run whisper::check_endpoints
    [ "$status" -eq 1 ]
}

# Test log analysis
@test "whisper::analyze_logs analyzes container logs for issues" {
    # Mock docker logs with some issues
    docker() {
        case "$1" in
            "logs")
                echo "INFO: Whisper service started"
                echo "WARNING: Model loading took longer than expected"
                echo "ERROR: Failed to process audio file"
                echo "INFO: Service ready"
                ;;
            *) return 0 ;;
        esac
    }
    
    result=$(whisper::analyze_logs)
    
    [[ "$result" =~ "Log Analysis:" ]]
    [[ "$result" =~ "INFO" ]]
    [[ "$result" =~ "WARNING" ]]
    [[ "$result" =~ "ERROR" ]]
}

# Test troubleshooting suggestions
@test "whisper::get_troubleshooting_tips provides helpful suggestions" {
    # Simulate unhealthy status
    whisper::health_check() {
        return 1
    }
    
    result=$(whisper::get_troubleshooting_tips)
    
    [[ "$result" =~ "Troubleshooting:" ]]
    [[ "$result" =~ "suggestions" ]] || [[ "$result" =~ "tips" ]]
}

# Test metrics collection
@test "whisper::collect_metrics gathers comprehensive metrics" {
    result=$(whisper::collect_metrics)
    
    [[ "$result" =~ "status" ]]
    [[ "$result" =~ "version" ]]
    [[ "$result" =~ "uptime" ]]
    [[ "$result" =~ "cpu_usage" ]]
    [[ "$result" =~ "memory_usage" ]]
    [[ "$result" =~ "model" ]]
}