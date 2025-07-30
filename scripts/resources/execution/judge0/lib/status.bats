#!/usr/bin/env bats
# Tests for Judge0 status.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export JUDGE0_PORT="2358"
    export JUDGE0_CONTAINER_NAME="judge0-test"
    export JUDGE0_WORKERS_NAME="judge0-workers-test"
    export JUDGE0_BASE_URL="http://localhost:2358"
    export JUDGE0_API_KEY="test_api_key_12345"
    export JUDGE0_HEALTH_ENDPOINT="/system_info"
    export JUDGE0_STARTUP_WAIT="30"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock system functions
    
    # Mock curl for API calls
    
    # Mock docker commands
}
EOF
                fi
                ;;
            "stats")
                echo "CONTAINER       CPU %     MEM USAGE / LIMIT     MEM %     NET I/O             BLOCK I/O           PIDS"
                echo "judge0-test     12.50%    1.5GiB / 2GiB         75.00%    5.2MB / 3.1MB       125MB / 85MB        35"
                echo "judge0-workers-test-1 8.30%  512MiB / 1GiB     50.00%    2.1MB / 1.5MB       45MB / 25MB         15"
                echo "judge0-workers-test-2 6.20%  384MiB / 1GiB     37.50%    1.8MB / 1.2MB       38MB / 22MB         12"
                ;;
            "logs")
                if [[ "$*" =~ "judge0-test" ]]; then
                    echo "Judge0 API server startup complete"
                    echo "Server listening on 0.0.0.0:2358"
                    echo "Database connection established"
                    echo "Worker pool initialized with 2 workers"
                    echo "Authentication enabled with API key"
                elif [[ "$*" =~ "judge0-workers-test" ]]; then
                    echo "Worker startup complete"
                    echo "Connected to queue: default"
                    echo "Worker ready for job processing"
                fi
                ;;
            *) echo "DOCKER: $*" ;;
        esac
        return 0
    }
    
    # Mock netstat/lsof for port checking
    netstat() {
        if [[ "$*" =~ "2358" ]]; then
            echo "tcp 0 0 0.0.0.0:2358 0.0.0.0:* LISTEN"
        fi
    }
    
    lsof() {
        if [[ "$*" =~ "2358" ]]; then
            echo "judge0    12345 root    5u  IPv4  123456      0t0  TCP *:2358 (LISTEN)"
        fi
    }
    
    # Mock ps for process checking
    ps() {
        echo "  PID  PPID COMMAND"
        echo "12345     1 judge0-server"
        echo "12346 12345 judge0-worker-1"
        echo "12347 12345 judge0-worker-2"
    }
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".version"*) echo "1.13.1" ;;
            *".hostname"*) echo "judge0-server" ;;
            *".worker_count"*) echo "2" ;;
            *".cpu_count"*) echo "4" ;;
            *".memory_limit"*) echo "2GB" ;;
            *".State.Running"*) echo "true" ;;
            *".State.Status"*) echo "running" ;;
            *"length"*) echo "2" ;;
            *) echo "JQ: $*" ;;
        esac
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
        if [[ "$port" == "2358" ]]; then
            return 0  # Port is in use
        fi
        return 1
    }
    
    system::get_memory_usage() {
        echo "Total: 16GB, Used: 8GB, Available: 8GB"
    }
    
    # Mock Judge0 functions
    judge0::docker::container_exists() { return 0; }
    judge0::docker::is_running() { return 0; }
    judge0::api::health_check() { return 0; }
    judge0::get_api_key() { echo "$JUDGE0_API_KEY"; }
    
    # Load configuration and messages
    source "${JUDGE0_DIR}/config/defaults.sh"
    source "${JUDGE0_DIR}/config/messages.sh"
    judge0::export_config
    judge0::export_messages
    
    # Load the functions to test
    source "${JUDGE0_DIR}/lib/status.sh"
}

# Test container existence check
@test "judge0::status::container_exists detects existing container" {
    result=$(judge0::status::container_exists && echo "exists" || echo "not found")
    
    [[ "$result" == "exists" ]]
}

# Test container existence check with missing container
@test "judge0::status::container_exists handles missing container" {
    # Override docker ps to return empty
    docker() {
        case "$1" in
            "ps") echo "" ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(judge0::status::container_exists && echo "exists" || echo "not found")
    
    [[ "$result" == "not found" ]]
}

# Test container running check
@test "judge0::status::is_running detects running container" {
    result=$(judge0::status::is_running && echo "running" || echo "stopped")
    
    [[ "$result" == "running" ]]
}

# Test container running check with stopped container
@test "judge0::status::is_running handles stopped container" {
    # Override docker inspect to show stopped state
    docker() {
        case "$1" in
            "inspect")
                echo '{"State":{"Running":false,"Status":"exited"}}'
                ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(judge0::status::is_running && echo "running" || echo "stopped")
    
    [[ "$result" == "stopped" ]]
}

# Test health check
@test "judge0::status::is_healthy performs health check" {
    result=$(judge0::status::is_healthy && echo "healthy" || echo "unhealthy")
    
    [[ "$result" == "healthy" ]]
}

# Test health check failure
@test "judge0::status::is_healthy detects unhealthy service" {
    # Override API health check to fail
    judge0::api::health_check() { return 1; }
    
    result=$(judge0::status::is_healthy && echo "healthy" || echo "unhealthy")
    
    [[ "$result" == "unhealthy" ]]
}

# Test comprehensive status check
@test "judge0::status::status provides comprehensive status information" {
    result=$(judge0::status::status)
    
    [[ "$result" =~ "=== Judge0 Status ===" ]]
    [[ "$result" =~ "Container" ]]
    [[ "$result" =~ "running" ]]
    [[ "$result" =~ "healthy" ]]
}

# Test quick status check
@test "judge0::status::quick_status provides brief status" {
    result=$(judge0::status::quick_status)
    
    [[ "$result" =~ "Judge0" ]]
    [[ "$result" =~ "running" ]] || [[ "$result" =~ "healthy" ]]
}

# Test port status check
@test "judge0::status::check_port_status verifies port availability" {
    result=$(judge0::status::check_port_status)
    
    [[ "$result" =~ "port" ]]
    [[ "$result" =~ "2358" ]]
    [[ "$result" =~ "listening" ]] || [[ "$result" =~ "open" ]]
}

# Test API connectivity check
@test "judge0::status::check_api_connectivity tests API endpoints" {
    result=$(judge0::status::check_api_connectivity)
    
    [[ "$result" =~ "API" ]]
    [[ "$result" =~ "accessible" ]] || [[ "$result" =~ "responding" ]]
}

# Test API connectivity failure
@test "judge0::status::check_api_connectivity handles API failure" {
    # Override curl to fail for API endpoints
    curl() {
        case "$*" in
            *"localhost:2358/system_info"*) return 1 ;;
            *) echo "CURL: $*" ;;
        esac
    }
    
    result=$(judge0::status::check_api_connectivity)
    
    [[ "$result" =~ "API" ]]
    [[ "$result" =~ "not accessible" ]] || [[ "$result" =~ "failed" ]]
}

# Test worker status
@test "judge0::status::get_worker_status shows worker information" {
    result=$(judge0::status::get_worker_status)
    
    [[ "$result" =~ "worker" ]]
    [[ "$result" =~ "worker-1" ]]
    [[ "$result" =~ "worker-2" ]]
    [[ "$result" =~ "150" ]] # jobs processed
}

# Test system resource status
@test "judge0::status::get_system_status shows resource usage" {
    result=$(judge0::status::get_system_status)
    
    [[ "$result" =~ "System" ]] || [[ "$result" =~ "Resource" ]]
    [[ "$result" =~ "CPU" ]] || [[ "$result" =~ "Memory" ]]
}

# Test container statistics
@test "judge0::status::get_container_stats shows container metrics" {
    result=$(judge0::status::get_container_stats)
    
    [[ "$result" =~ "CPU" ]]
    [[ "$result" =~ "MEM" ]]
    [[ "$result" =~ "12.50%" ]]
    [[ "$result" =~ "judge0-test" ]]
}

# Test queue status
@test "judge0::status::get_queue_status shows queue information" {
    result=$(judge0::status::get_queue_status)
    
    [[ "$result" =~ "queue" ]] || [[ "$result" =~ "submission" ]]
    [[ "$result" =~ "Accepted" ]] || [[ "$result" =~ "In Queue" ]]
}

# Test language availability
@test "judge0::status::get_language_status shows supported languages" {
    result=$(judge0::status::get_language_status)
    
    [[ "$result" =~ "language" ]]
    [[ "$result" =~ "Python" ]] || [[ "$result" =~ "JavaScript" ]]
}

# Test log status check
@test "judge0::status::check_logs examines recent logs" {
    result=$(judge0::status::check_logs)
    
    [[ "$result" =~ "startup complete" ]] || [[ "$result" =~ "Server listening" ]]
}

# Test error detection in logs
@test "judge0::status::detect_errors_in_logs finds error patterns" {
    # Override docker logs to include errors
    docker() {
        case "$1" in
            "logs")
                echo "Judge0 API server startup complete"
                echo "ERROR: Database connection failed"
                echo "WARNING: High memory usage detected"
                ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(judge0::status::detect_errors_in_logs)
    
    [[ "$result" =~ "ERROR" ]] || [[ "$result" =~ "error" ]]
}

# Test performance metrics
@test "judge0::status::get_performance_metrics provides performance data" {
    result=$(judge0::status::get_performance_metrics)
    
    [[ "$result" =~ "performance" ]] || [[ "$result" =~ "metrics" ]]
}

# Test uptime calculation
@test "judge0::status::get_uptime calculates service uptime" {
    result=$(judge0::status::get_uptime)
    
    [[ "$result" =~ "uptime" ]] || [[ "$result" =~ "running" ]]
}

# Test memory usage check
@test "judge0::status::check_memory_usage monitors memory consumption" {
    result=$(judge0::status::check_memory_usage)
    
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "RAM" ]]
    [[ "$result" =~ "GB" ]] || [[ "$result" =~ "MB" ]]
}

# Test disk usage check
@test "judge0::status::check_disk_usage monitors storage consumption" {
    result=$(judge0::status::check_disk_usage)
    
    [[ "$result" =~ "disk" ]] || [[ "$result" =~ "storage" ]]
}

# Test network status check
@test "judge0::status::check_network_status monitors network connectivity" {
    result=$(judge0::status::check_network_status)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "connectivity" ]]
}

# Test process monitoring
@test "judge0::status::monitor_processes checks running processes" {
    result=$(judge0::status::monitor_processes)
    
    [[ "$result" =~ "process" ]] || [[ "$result" =~ "judge0" ]]
    [[ "$result" =~ "12345" ]]
}

# Test service dependencies check
@test "judge0::status::check_dependencies verifies service dependencies" {
    result=$(judge0::status::check_dependencies)
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "requirement" ]]
}

# Test configuration validation
@test "judge0::status::validate_configuration checks configuration validity" {
    result=$(judge0::status::validate_configuration)
    
    [[ "$result" =~ "configuration" ]] || [[ "$result" =~ "config" ]]
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "OK" ]]
}

# Test detailed status report
@test "judge0::status::generate_status_report creates comprehensive report" {
    result=$(judge0::status::generate_status_report)
    
    [[ "$result" =~ "Status Report" ]]
    [[ "$result" =~ "Judge0" ]]
    [[ "$result" =~ "Container" ]]
    [[ "$result" =~ "System" ]]
}

# Test status monitoring with alerts
@test "judge0::status::monitor_with_alerts checks status with alerting" {
    result=$(judge0::status::monitor_with_alerts)
    
    [[ "$result" =~ "monitor" ]] || [[ "$result" =~ "alert" ]]
}

# Test health history tracking
@test "judge0::status::track_health_history maintains health records" {
    result=$(judge0::status::track_health_history)
    
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "history" ]]
}

# Test JSON health status
@test "judge0::status::get_health_json returns health status in JSON format" {
    result=$(judge0::status::get_health_json)
    
    [[ "$result" =~ "status" ]]
    [[ "$result" =~ "api" ]]
    [[ "$result" =~ "workers" ]]
    
    # Verify JSON structure
    echo "$result" | jq -e '.status' >/dev/null
    echo "$result" | jq -e '.api' >/dev/null
    echo "$result" | jq -e '.workers' >/dev/null
}

# Test submission statistics
@test "judge0::status::get_submission_stats shows submission statistics" {
    result=$(judge0::status::get_submission_stats)
    
    [[ "$result" =~ "submission" ]] || [[ "$result" =~ "stats" ]]
}

# Test service readiness check
@test "judge0::status::check_readiness verifies service readiness" {
    result=$(judge0::status::check_readiness)
    
    [[ "$result" =~ "ready" ]] || [[ "$result" =~ "readiness" ]]
}

# Test service liveness check
@test "judge0::status::check_liveness verifies service liveness" {
    result=$(judge0::status::check_liveness)
    
    [[ "$result" =~ "alive" ]] || [[ "$result" =~ "liveness" ]]
}

# Test status export
@test "judge0::status::export_status_data exports status information" {
    result=$(judge0::status::export_status_data)
    
    [[ "$result" =~ "export" ]] || [[ "$result" =~ "status" ]]
}

# Test historical status analysis
@test "judge0::status::analyze_status_trends analyzes status patterns" {
    result=$(judge0::status::analyze_status_trends)
    
    [[ "$result" =~ "analyze" ]] || [[ "$result" =~ "trend" ]]
}

# Test status notifications
@test "judge0::status::send_status_notifications handles status alerts" {
    result=$(judge0::status::send_status_notifications)
    
    [[ "$result" =~ "notification" ]] || [[ "$result" =~ "alert" ]]
}

# Test component health checks
@test "judge0::status::check_component_health checks individual components" {
    result=$(judge0::status::check_component_health "server")
    
    [[ "$result" =~ "server" ]]
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "component" ]]
}

# Test load balancer status
@test "judge0::status::check_load_balancer verifies load balancer health" {
    result=$(judge0::status::check_load_balancer)
    
    [[ "$result" =~ "load balancer" ]] || [[ "$result" =~ "balance" ]]
}

# Test security status
@test "judge0::status::check_security_status verifies security configuration" {
    result=$(judge0::status::check_security_status)
    
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "configuration" ]]
}

# Test backup status
@test "judge0::status::check_backup_status verifies backup system health" {
    result=$(judge0::status::check_backup_status)
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "status" ]]
}

# Test monitoring system status
@test "judge0::status::check_monitoring_system verifies monitoring health" {
    result=$(judge0::status::check_monitoring_system)
    
    [[ "$result" =~ "monitoring" ]] || [[ "$result" =~ "system" ]]
}