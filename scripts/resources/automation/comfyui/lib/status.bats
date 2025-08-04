#!/usr/bin/env bats
# Tests for ComfyUI status.sh functions

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
    export COMFYUI_DATA_DIR="/tmp/comfyui-test"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    COMFYUI_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock system functions
    
    # Mock curl for health checks
    
    # Mock Docker operations
}
EOF
                fi
                ;;
            "stats")
                echo "CONTAINER       CPU %     MEM USAGE / LIMIT     MEM %     NET I/O             BLOCK I/O           PIDS"
                echo "comfyui-test    15.50%    2.5GiB / 8GiB         31.25%    1.2MB / 800KB       50MB / 25MB         35"
                ;;
            "logs")
                echo "ComfyUI startup complete"
                echo "Total VRAM 24564 MB, total RAM 32768 MB"
                echo "Starting server on 0.0.0.0:8188"
                echo "Server started successfully"
                ;;
            "exec")
                echo "DOCKER_EXEC: $*"
                return 0
                ;;
            *) echo "DOCKER: $*" ;;
        esac
        return 0
    }
    
    # Mock netstat/lsof for port checking
    netstat() {
        if [[ "$*" =~ "8188" ]]; then
            echo "tcp 0 0 0.0.0.0:8188 0.0.0.0:* LISTEN"
        fi
    }
    
    lsof() {
        if [[ "$*" =~ "8188" ]]; then
            echo "python 12345 root 5u IPv4 123456 0t0 TCP *:8188 (LISTEN)"
        fi
    }
    
    # Mock date for timestamps
    date() {
        echo "2024-01-15 11:00:00"
    }
    
    # Mock log functions
    log::info() { echo "INFO: $1"; }
    log::error() { echo "ERROR: $1"; }
    log::warn() { echo "WARN: $1"; }
    log::success() { echo "SUCCESS: $1"; }
    log::debug() { echo "DEBUG: $1"; }
    log::header() { echo "=== $1 ==="; }
    
    # Mock utility functions
    system::check_port() {
        local port="$1"
        if [[ "$port" == "8188" ]]; then
            return 0  # Port is in use
        fi
        return 1
    }
    
    system::get_memory_usage() {
        echo "Total: 32GB, Used: 8GB, Available: 24GB"
    }
    
    # Load configuration and messages
    source "${COMFYUI_DIR}/config/defaults.sh"
    source "${COMFYUI_DIR}/config/messages.sh"
    comfyui::export_config
    comfyui::export_messages
    
    # Load the functions to test
    source "${COMFYUI_DIR}/lib/status.sh"
}

# Test container existence check
@test "comfyui::container_exists detects existing container" {
    result=$(comfyui::container_exists && echo "exists" || echo "not found")
    
    [[ "$result" == "exists" ]]
}

# Test container existence check with missing container
@test "comfyui::container_exists handles missing container" {
    # Override docker ps to return empty
    docker() {
        case "$1" in
            "ps") echo "" ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(comfyui::container_exists && echo "exists" || echo "not found")
    
    [[ "$result" == "not found" ]]
}

# Test container running check
@test "comfyui::is_running detects running container" {
    result=$(comfyui::is_running && echo "running" || echo "stopped")
    
    [[ "$result" == "running" ]]
}

# Test container running check with stopped container
@test "comfyui::is_running handles stopped container" {
    # Override docker inspect to show stopped state
    docker() {
        case "$1" in
            "inspect")
                echo '{"State":{"Running":false,"Status":"exited"}}'
                ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(comfyui::is_running && echo "running" || echo "stopped")
    
    [[ "$result" == "stopped" ]]
}

# Test health check
@test "comfyui::is_healthy performs health check" {
    result=$(comfyui::is_healthy && echo "healthy" || echo "unhealthy")
    
    [[ "$result" == "healthy" ]]
}

# Test health check failure
@test "comfyui::is_healthy detects unhealthy service" {
    # Override curl to fail
    curl() {
        case "$*" in
            *"localhost:8188"*) return 1 ;;
            *) echo "CURL: $*" ;;
        esac
    }
    
    result=$(comfyui::is_healthy && echo "healthy" || echo "unhealthy")
    
    [[ "$result" == "unhealthy" ]]
}

# Test comprehensive status check
@test "comfyui::status provides comprehensive status information" {
    result=$(comfyui::status)
    
    [[ "$result" =~ "=== ComfyUI Status ===" ]]
    [[ "$result" =~ "Container" ]]
    [[ "$result" =~ "running" ]]
    [[ "$result" =~ "healthy" ]]
}

# Test quick status check
@test "comfyui::quick_status provides brief status" {
    result=$(comfyui::quick_status)
    
    [[ "$result" =~ "ComfyUI" ]]
    [[ "$result" =~ "running" ]] || [[ "$result" =~ "healthy" ]]
}

# Test port status check
@test "comfyui::check_port_status verifies port availability" {
    result=$(comfyui::check_port_status)
    
    [[ "$result" =~ "port" ]]
    [[ "$result" =~ "8188" ]]
    [[ "$result" =~ "listening" ]] || [[ "$result" =~ "open" ]]
}

# Test API connectivity check
@test "comfyui::check_api_connectivity tests API endpoints" {
    result=$(comfyui::check_api_connectivity)
    
    [[ "$result" =~ "API" ]]
    [[ "$result" =~ "accessible" ]] || [[ "$result" =~ "responding" ]]
}

# Test API connectivity failure
@test "comfyui::check_api_connectivity handles API failure" {
    # Override curl to fail for API endpoints
    curl() {
        case "$*" in
            *"localhost:8188"*) return 1 ;;
            *) echo "CURL: $*" ;;
        esac
    }
    
    result=$(comfyui::check_api_connectivity)
    
    [[ "$result" =~ "API" ]]
    [[ "$result" =~ "not accessible" ]] || [[ "$result" =~ "failed" ]]
}

# Test queue status
@test "comfyui::get_queue_status shows queue information" {
    result=$(comfyui::get_queue_status)
    
    [[ "$result" =~ "queue" ]]
    [[ "$result" =~ "pending" ]] || [[ "$result" =~ "running" ]]
}

# Test system resource status
@test "comfyui::get_system_status shows resource usage" {
    result=$(comfyui::get_system_status)
    
    [[ "$result" =~ "System" ]] || [[ "$result" =~ "Resource" ]]
    [[ "$result" =~ "RAM" ]] || [[ "$result" =~ "VRAM" ]]
}

# Test container statistics
@test "comfyui::get_container_stats shows container metrics" {
    result=$(comfyui::get_container_stats)
    
    [[ "$result" =~ "CPU" ]]
    [[ "$result" =~ "MEM" ]]
    [[ "$result" =~ "15.50%" ]]
}

# Test log status check
@test "comfyui::check_logs examines recent logs" {
    result=$(comfyui::check_logs)
    
    [[ "$result" =~ "startup complete" ]] || [[ "$result" =~ "Server started" ]]
}

# Test error detection in logs
@test "comfyui::detect_errors_in_logs finds error patterns" {
    # Override docker logs to include errors
    docker() {
        case "$1" in
            "logs")
                echo "ComfyUI startup complete"
                echo "ERROR: CUDA out of memory"
                echo "WARNING: Model not found"
                ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(comfyui::detect_errors_in_logs)
    
    [[ "$result" =~ "ERROR" ]] || [[ "$result" =~ "error" ]]
}

# Test performance metrics
@test "comfyui::get_performance_metrics provides performance data" {
    result=$(comfyui::get_performance_metrics)
    
    [[ "$result" =~ "performance" ]] || [[ "$result" =~ "metrics" ]]
}

# Test uptime calculation
@test "comfyui::get_uptime calculates service uptime" {
    result=$(comfyui::get_uptime)
    
    [[ "$result" =~ "uptime" ]] || [[ "$result" =~ "running" ]]
}

# Test memory usage check
@test "comfyui::check_memory_usage monitors memory consumption" {
    result=$(comfyui::check_memory_usage)
    
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "RAM" ]]
    [[ "$result" =~ "GB" ]] || [[ "$result" =~ "MB" ]]
}

# Test GPU status check
@test "comfyui::check_gpu_status monitors GPU usage" {
    result=$(comfyui::check_gpu_status)
    
    [[ "$result" =~ "GPU" ]] || [[ "$result" =~ "VRAM" ]]
}

# Test disk usage check
@test "comfyui::check_disk_usage monitors storage consumption" {
    result=$(comfyui::check_disk_usage)
    
    [[ "$result" =~ "disk" ]] || [[ "$result" =~ "storage" ]]
}

# Test network status check
@test "comfyui::check_network_status monitors network connectivity" {
    result=$(comfyui::check_network_status)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "connectivity" ]]
}

# Test process monitoring
@test "comfyui::monitor_processes checks running processes" {
    result=$(comfyui::monitor_processes)
    
    [[ "$result" =~ "process" ]] || [[ "$result" =~ "running" ]]
}

# Test service dependencies check
@test "comfyui::check_dependencies verifies service dependencies" {
    result=$(comfyui::check_dependencies)
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "requirement" ]]
}

# Test configuration validation
@test "comfyui::validate_configuration checks configuration validity" {
    result=$(comfyui::validate_configuration)
    
    [[ "$result" =~ "configuration" ]] || [[ "$result" =~ "config" ]]
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "OK" ]]
}

# Test detailed status report
@test "comfyui::generate_status_report creates comprehensive report" {
    result=$(comfyui::generate_status_report)
    
    [[ "$result" =~ "Status Report" ]]
    [[ "$result" =~ "ComfyUI" ]]
    [[ "$result" =~ "Container" ]]
    [[ "$result" =~ "System" ]]
}

# Test status monitoring with alerts
@test "comfyui::monitor_with_alerts checks status with alerting" {
    result=$(comfyui::monitor_with_alerts)
    
    [[ "$result" =~ "monitor" ]] || [[ "$result" =~ "alert" ]]
}

# Test health history tracking
@test "comfyui::track_health_history maintains health records" {
    result=$(comfyui::track_health_history)
    
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "history" ]]
}

# Test status dashboard
@test "comfyui::display_status_dashboard shows interactive dashboard" {
    result=$(comfyui::display_status_dashboard)
    
    [[ "$result" =~ "dashboard" ]] || [[ "$result" =~ "status" ]]
}

# Test automated health checks
@test "comfyui::run_health_checks performs automated health verification" {
    result=$(comfyui::run_health_checks)
    
    [[ "$result" =~ "health" ]]
    [[ "$result" =~ "check" ]]
}

# Test service readiness check
@test "comfyui::check_readiness verifies service readiness" {
    result=$(comfyui::check_readiness)
    
    [[ "$result" =~ "ready" ]] || [[ "$result" =~ "readiness" ]]
}

# Test service liveness check
@test "comfyui::check_liveness verifies service liveness" {
    result=$(comfyui::check_liveness)
    
    [[ "$result" =~ "alive" ]] || [[ "$result" =~ "liveness" ]]
}

# Test status export
@test "comfyui::export_status_data exports status information" {
    result=$(comfyui::export_status_data)
    
    [[ "$result" =~ "export" ]] || [[ "$result" =~ "status" ]]
}

# Test historical status analysis
@test "comfyui::analyze_status_trends analyzes status patterns" {
    result=$(comfyui::analyze_status_trends)
    
    [[ "$result" =~ "analyze" ]] || [[ "$result" =~ "trend" ]]
}

# Test status notifications
@test "comfyui::send_status_notifications handles status alerts" {
    result=$(comfyui::send_status_notifications)
    
    [[ "$result" =~ "notification" ]] || [[ "$result" =~ "alert" ]]
}