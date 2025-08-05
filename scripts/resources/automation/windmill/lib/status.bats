#!/usr/bin/env bats

# Load Vrooli test infrastructure (REQUIRED)
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "windmill"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    export SETUP_FILE_WINDMILL_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")" 
    export SETUP_FILE_CONFIG_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")/config"
    export SETUP_FILE_LIB_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")/lib"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    WINDMILL_DIR="${SETUP_FILE_WINDMILL_DIR}"
    CONFIG_DIR="${SETUP_FILE_CONFIG_DIR}"
    LIB_DIR="${SETUP_FILE_LIB_DIR}"
    
    # Set test environment BEFORE sourcing config files to avoid readonly conflicts
    export WINDMILL_PORT="5681"
    export WINDMILL_CONTAINER_NAME="windmill-test"
    export WINDMILL_DB_CONTAINER_NAME="windmill-db-test"
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_API_TOKEN="wm_abc123def456"
    export WORKSPACE_NAME="demo"
    export YES="no"
    
    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "windmill") echo "5681" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Now source the config files
    source "${WINDMILL_DIR}/config/defaults.sh"
    source "${WINDMILL_DIR}/config/messages.sh"
    
    # Export config and messages
    windmill::export_config
    windmill::export_messages
    
    # Load the functions to test
    source "${WINDMILL_DIR}/lib/status.sh"
    
    # Mock curl for health checks and API calls
    curl() {
        case "$*" in
            *"localhost:5681/api/version"*)
                echo '{"version":"1.0.0"}'
                ;;
            *"localhost:5681/api/w/demo/jobs/completed"*)
                cat <<EOF
[
  {
    "id": "job_123456",
    "type": "CompletedJob",
    "workspace_id": "demo",
    "created_at": "2024-01-15T10:30:00Z",
    "success": true,
    "result": "Hello from Windmill!"
  }
]
EOF
                ;;
            *"localhost:5681/api/w/demo/workers/list"*)
                cat <<EOF
[
  {
    "worker_instance": "worker-1",
    "last_ping": "2024-01-15T11:00:00Z",
    "jobs_executed": 150,
    "worker_group": "default"
  },
  {
    "worker_instance": "worker-2",
    "last_ping": "2024-01-15T11:00:00Z",
    "jobs_executed": 200,
    "worker_group": "default"
  }
]
EOF
                ;;
            *) echo "CURL: $*" ;;
        esac
        return 0
    }
    
    # Mock Docker operations
    docker() {
        case "$1" in
            "ps")
                if [[ "$*" =~ "windmill-test" ]]; then
                    echo "windmill-test   running   5681->8000/tcp"
                elif [[ "$*" =~ "windmill-db-test" ]]; then
                    echo "windmill-db-test   running   5432/tcp"
                else
                    echo "windmill-test   running   5681->8000/tcp"
                    echo "windmill-db-test   running   5432/tcp"
                fi
                ;;
            "inspect")
                if [[ "$*" =~ "windmill-test" ]]; then
                    cat <<EOF
{
  "State": {
    "Running": true,
    "Status": "running",
    "StartedAt": "2024-01-15T10:30:00Z",
    "Health": {
      "Status": "healthy",
      "LastCheck": "2024-01-15T11:00:00Z"
    }
  },
  "Config": {
    "Image": "windmillhq/windmill:latest"
  }
}
EOF
                elif [[ "$*" =~ "windmill-db-test" ]]; then
                    cat <<EOF
{
  "State": {
    "Running": true,
    "Status": "running",
    "StartedAt": "2024-01-15T10:25:00Z",
    "Health": {
      "Status": "healthy",
      "LastCheck": "2024-01-15T11:00:00Z"
    }
  },
  "Config": {
    "Image": "postgres:14"
  }
}
EOF
                fi
                ;;
            "stats")
                echo "CONTAINER       CPU %     MEM USAGE / LIMIT     MEM %     NET I/O             BLOCK I/O           PIDS"
                echo "windmill-test   8.50%     1.2GiB / 4GiB         30.00%    2.1MB / 1.5MB       75MB / 50MB         45"
                echo "windmill-db-test 2.30%    256MiB / 2GiB         12.50%    500KB / 300KB       25MB / 15MB         12"
                ;;
            "logs")
                if [[ "$*" =~ "windmill-test" ]]; then
                    echo "Windmill server startup complete"
                    echo "Server listening on 0.0.0.0:8000"
                    echo "Database connection established"
                    echo "Worker pool initialized with 4 workers"
                elif [[ "$*" =~ "windmill-db-test" ]]; then
                    echo "PostgreSQL init process complete; ready for start up"
                    echo "database system is ready to accept connections"
                fi
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
        if [[ "$*" =~ "5681" ]]; then
            echo "tcp 0 0 0.0.0.0:5681 0.0.0.0:* LISTEN"
        fi
    }
    
    lsof() {
        if [[ "$*" =~ "5681" ]]; then
            echo "windmill 12345 root 5u IPv4 123456 0t0 TCP *:5681 (LISTEN)"
        fi
    }
    
    # Mock ps for process checking
    ps() {
        echo "  PID  PPID COMMAND"
        echo "12345     1 windmill"
        echo "12346 12345 windmill-worker"
        echo "12347 12345 windmill-worker"
    }
    
    # Mock date for timestamps
    date() {
        echo "2024-01-15 11:00:00"
    }
    
    # Mock utility functions
    system::check_port() {
        local port="$1"
        if [[ "$port" == "5681" ]]; then
            return 0  # Port is in use
        fi
        return 1
    }
    
    system::get_memory_usage() {
        echo "Total: 16GB, Used: 8GB, Available: 8GB"
    }
}

# Cleanup after each test
teardown() {
    vrooli_cleanup_test
}

# Test container existence check
@test "windmill::container_exists detects existing container" {
    result=$(windmill::container_exists && echo "exists" || echo "not found")
    
    [[ "$result" == "exists" ]]
}

# Test container existence check with missing container
@test "windmill::container_exists handles missing container" {
    # Override docker ps to return empty
    docker() {
        case "$1" in
            "ps") echo "" ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(windmill::container_exists && echo "exists" || echo "not found")
    
    [[ "$result" == "not found" ]]
}

# Test container running check
@test "windmill::is_running detects running container" {
    result=$(windmill::is_running && echo "running" || echo "stopped")
    
    [[ "$result" == "running" ]]
}

# Test container running check with stopped container
@test "windmill::is_running handles stopped container" {
    # Override docker inspect to show stopped state
    docker() {
        case "$1" in
            "inspect")
                echo '{"State":{"Running":false,"Status":"exited"}}'
                ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(windmill::is_running && echo "running" || echo "stopped")
    
    [[ "$result" == "stopped" ]]
}

# Test health check
@test "windmill::is_healthy performs health check" {
    result=$(windmill::is_healthy && echo "healthy" || echo "unhealthy")
    
    [[ "$result" == "healthy" ]]
}

# Test health check failure
@test "windmill::is_healthy detects unhealthy service" {
    # Override curl to fail
    curl() {
        case "$*" in
            *"localhost:5681/api/version"*) return 1 ;;
            *) echo "CURL: $*" ;;
        esac
    }
    
    result=$(windmill::is_healthy && echo "healthy" || echo "unhealthy")
    
    [[ "$result" == "unhealthy" ]]
}

# Test comprehensive status check
@test "windmill::status provides comprehensive status information" {
    result=$(windmill::status)
    
    [[ "$result" =~ "=== Windmill Status ===" ]]
    [[ "$result" =~ "Container" ]]
    [[ "$result" =~ "running" ]]
    [[ "$result" =~ "healthy" ]]
}

# Test quick status check
@test "windmill::quick_status provides brief status" {
    result=$(windmill::quick_status)
    
    [[ "$result" =~ "Windmill" ]]
    [[ "$result" =~ "running" ]] || [[ "$result" =~ "healthy" ]]
}

# Test port status check
@test "windmill::check_port_status verifies port availability" {
    result=$(windmill::check_port_status)
    
    [[ "$result" =~ "port" ]]
    [[ "$result" =~ "5681" ]]
    [[ "$result" =~ "listening" ]] || [[ "$result" =~ "open" ]]
}

# Test API connectivity check
@test "windmill::check_api_connectivity tests API endpoints" {
    result=$(windmill::check_api_connectivity)
    
    [[ "$result" =~ "API" ]]
    [[ "$result" =~ "accessible" ]] || [[ "$result" =~ "responding" ]]
}

# Test API connectivity failure
@test "windmill::check_api_connectivity handles API failure" {
    # Override curl to fail for API endpoints
    curl() {
        case "$*" in
            *"localhost:5681/api/version"*) return 1 ;;
            *) echo "CURL: $*" ;;
        esac
    }
    
    result=$(windmill::check_api_connectivity)
    
    [[ "$result" =~ "API" ]]
    [[ "$result" =~ "not accessible" ]] || [[ "$result" =~ "failed" ]]
}

# Test database status
@test "windmill::check_database_status shows database information" {
    result=$(windmill::check_database_status)
    
    [[ "$result" =~ "database" ]] || [[ "$result" =~ "PostgreSQL" ]]
    [[ "$result" =~ "healthy" ]] || [[ "$result" =~ "connected" ]]
}

# Test worker status
@test "windmill::get_worker_status shows worker information" {
    result=$(windmill::get_worker_status)
    
    [[ "$result" =~ "worker" ]]
    [[ "$result" =~ "worker-1" ]]
    [[ "$result" =~ "worker-2" ]]
    [[ "$result" =~ "150" ]] # jobs executed
}

# Test system resource status
@test "windmill::get_system_status shows resource usage" {
    result=$(windmill::get_system_status)
    
    [[ "$result" =~ "System" ]] || [[ "$result" =~ "Resource" ]]
    [[ "$result" =~ "CPU" ]] || [[ "$result" =~ "Memory" ]]
}

# Test container statistics
@test "windmill::get_container_stats shows container metrics" {
    result=$(windmill::get_container_stats)
    
    [[ "$result" =~ "CPU" ]]
    [[ "$result" =~ "MEM" ]]
    [[ "$result" =~ "8.50%" ]]
    [[ "$result" =~ "windmill-test" ]]
}

# Test job queue status
@test "windmill::get_queue_status shows job queue information" {
    result=$(windmill::get_queue_status)
    
    [[ "$result" =~ "queue" ]] || [[ "$result" =~ "job" ]]
    [[ "$result" =~ "QueuedJob" ]] || [[ "$result" =~ "CompletedJob" ]]
}

# Test workspace statistics
@test "windmill::get_workspace_stats shows workspace statistics" {
    result=$(windmill::get_workspace_stats)
    
    [[ "$result" =~ "workspace" ]] || [[ "$result" =~ "stats" ]]
    [[ "$result" =~ "scripts" ]] || [[ "$result" =~ "jobs" ]]
}

# Test log status check
@test "windmill::check_logs examines recent logs" {
    result=$(windmill::check_logs)
    
    [[ "$result" =~ "startup complete" ]] || [[ "$result" =~ "Server listening" ]]
}

# Test error detection in logs
@test "windmill::detect_errors_in_logs finds error patterns" {
    # Override docker logs to include errors
    docker() {
        case "$1" in
            "logs")
                echo "Windmill server startup complete"
                echo "ERROR: Database connection failed"
                echo "WARNING: Worker timeout detected"
                ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(windmill::detect_errors_in_logs)
    
    [[ "$result" =~ "ERROR" ]] || [[ "$result" =~ "error" ]]
}

# Test performance metrics
@test "windmill::get_performance_metrics provides performance data" {
    result=$(windmill::get_performance_metrics)
    
    [[ "$result" =~ "performance" ]] || [[ "$result" =~ "metrics" ]]
}

# Test uptime calculation
@test "windmill::get_uptime calculates service uptime" {
    result=$(windmill::get_uptime)
    
    [[ "$result" =~ "uptime" ]] || [[ "$result" =~ "running" ]]
}

# Test memory usage check
@test "windmill::check_memory_usage monitors memory consumption" {
    result=$(windmill::check_memory_usage)
    
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "RAM" ]]
    [[ "$result" =~ "GB" ]] || [[ "$result" =~ "MB" ]]
}

# Test disk usage check
@test "windmill::check_disk_usage monitors storage consumption" {
    result=$(windmill::check_disk_usage)
    
    [[ "$result" =~ "disk" ]] || [[ "$result" =~ "storage" ]]
}

# Test network status check
@test "windmill::check_network_status monitors network connectivity" {
    result=$(windmill::check_network_status)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "connectivity" ]]
}

# Test process monitoring
@test "windmill::monitor_processes checks running processes" {
    result=$(windmill::monitor_processes)
    
    [[ "$result" =~ "process" ]] || [[ "$result" =~ "windmill" ]]
    [[ "$result" =~ "12345" ]]
}

# Test service dependencies check
@test "windmill::check_dependencies verifies service dependencies" {
    result=$(windmill::check_dependencies)
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "requirement" ]]
}

# Test configuration validation
@test "windmill::validate_configuration checks configuration validity" {
    result=$(windmill::validate_configuration)
    
    [[ "$result" =~ "configuration" ]] || [[ "$result" =~ "config" ]]
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "OK" ]]
}

# Test detailed status report
@test "windmill::generate_status_report creates comprehensive report" {
    result=$(windmill::generate_status_report)
    
    [[ "$result" =~ "Status Report" ]]
    [[ "$result" =~ "Windmill" ]]
    [[ "$result" =~ "Container" ]]
    [[ "$result" =~ "System" ]]
}

# Test status monitoring with alerts
@test "windmill::monitor_with_alerts checks status with alerting" {
    result=$(windmill::monitor_with_alerts)
    
    [[ "$result" =~ "monitor" ]] || [[ "$result" =~ "alert" ]]
}

# Test health history tracking
@test "windmill::track_health_history maintains health records" {
    result=$(windmill::track_health_history)
    
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "history" ]]
}

# Test status dashboard
@test "windmill::display_status_dashboard shows interactive dashboard" {
    result=$(windmill::display_status_dashboard)
    
    [[ "$result" =~ "dashboard" ]] || [[ "$result" =~ "status" ]]
}

# Test automated health checks
@test "windmill::run_health_checks performs automated health verification" {
    result=$(windmill::run_health_checks)
    
    [[ "$result" =~ "health" ]]
    [[ "$result" =~ "check" ]]
}

# Test service readiness check
@test "windmill::check_readiness verifies service readiness" {
    result=$(windmill::check_readiness)
    
    [[ "$result" =~ "ready" ]] || [[ "$result" =~ "readiness" ]]
}

# Test service liveness check
@test "windmill::check_liveness verifies service liveness" {
    result=$(windmill::check_liveness)
    
    [[ "$result" =~ "alive" ]] || [[ "$result" =~ "liveness" ]]
}

# Test status export
@test "windmill::export_status_data exports status information" {
    result=$(windmill::export_status_data)
    
    [[ "$result" =~ "export" ]] || [[ "$result" =~ "status" ]]
}

# Test historical status analysis
@test "windmill::analyze_status_trends analyzes status patterns" {
    result=$(windmill::analyze_status_trends)
    
    [[ "$result" =~ "analyze" ]] || [[ "$result" =~ "trend" ]]
}

# Test status notifications
@test "windmill::send_status_notifications handles status alerts" {
    result=$(windmill::send_status_notifications)
    
    [[ "$result" =~ "notification" ]] || [[ "$result" =~ "alert" ]]
}

# Test component health checks
@test "windmill::check_component_health checks individual components" {
    result=$(windmill::check_component_health "database")
    
    [[ "$result" =~ "database" ]]
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "component" ]]
}

# Test API endpoint status
@test "windmill::check_api_endpoints verifies API endpoint health" {
    result=$(windmill::check_api_endpoints)
    
    [[ "$result" =~ "API" ]] || [[ "$result" =~ "endpoint" ]]
    [[ "$result" =~ "/api/version" ]]
}

# Test load balancer status
@test "windmill::check_load_balancer verifies load balancer health" {
    result=$(windmill::check_load_balancer)
    
    [[ "$result" =~ "load balancer" ]] || [[ "$result" =~ "balance" ]]
}

# Test SSL certificate status
@test "windmill::check_ssl_certificates verifies SSL certificate validity" {
    result=$(windmill::check_ssl_certificates)
    
    [[ "$result" =~ "SSL" ]] || [[ "$result" =~ "certificate" ]]
}

# Test backup status
@test "windmill::check_backup_status verifies backup system health" {
    result=$(windmill::check_backup_status)
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "status" ]]
}

# Test log rotation status
@test "windmill::check_log_rotation verifies log rotation health" {
    result=$(windmill::check_log_rotation)
    
    [[ "$result" =~ "log" ]] || [[ "$result" =~ "rotation" ]]
}

# Test monitoring system status
@test "windmill::check_monitoring_system verifies monitoring health" {
    result=$(windmill::check_monitoring_system)
    
    [[ "$result" =~ "monitoring" ]] || [[ "$result" =~ "system" ]]
}