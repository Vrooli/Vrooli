#!/usr/bin/env bats
# Tests for Qdrant status.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export QDRANT_PORT="6333"
    export QDRANT_GRPC_PORT="6334"
    export QDRANT_CONTAINER_NAME="qdrant-test"
    export QDRANT_BASE_URL="http://localhost:6333"
    export QDRANT_GRPC_URL="grpc://localhost:6334"
    export QDRANT_IMAGE="qdrant/qdrant:latest"
    export QDRANT_DATA_DIR="/tmp/qdrant-test/data"
    export QDRANT_CONFIG_DIR="/tmp/qdrant-test/config"
    export QDRANT_SNAPSHOTS_DIR="/tmp/qdrant-test/snapshots"
    export QDRANT_API_KEY="test_qdrant_api_key_123"
    export QDRANT_NETWORK_NAME="qdrant-network-test"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    QDRANT_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories
    mkdir -p "$QDRANT_DATA_DIR"
    mkdir -p "$QDRANT_CONFIG_DIR"
    mkdir -p "$QDRANT_SNAPSHOTS_DIR"
    
    # Mock system functions
    
    # Mock docker commands
}
EOF
                fi
                ;;
            "logs")
                if [[ "$*" =~ "qdrant-test" ]]; then
                    echo "Qdrant server startup complete"
                    echo "REST API listening on 0.0.0.0:6333"
                    echo "gRPC API listening on 0.0.0.0:6334"
                    echo "Storage initialization complete"
                fi
                ;;
            "stats")
                echo "CONTAINER       CPU %     MEM USAGE / LIMIT     MEM %     NET I/O             BLOCK I/O           PIDS"
                echo "qdrant-test     8.50%     1.2GiB / 4GiB         30.00%    2.1MB / 1.5MB       125MB / 85MB        25"
                ;;
            "exec")
                echo "DOCKER_EXEC: $*"
                ;;
            *) echo "DOCKER: $*" ;;
        esac
        return 0
    }
    
    # Mock curl for API calls
    curl() {
        case "$*" in
            *"/health"*)
                echo '{"status":"ok","version":"1.7.4"}'
                ;;
            *"/cluster"*)
                echo '{"result":{"status":"enabled","peer_id":"12345","peers":{"known":[],"total":1}}}'
                ;;
            *"/collections"*)
                echo '{"result":{"collections":[{"name":"test_collection","vectors_count":1000,"indexed_vectors_count":1000,"points_count":1000,"segments_count":4}]}}'
                ;;
            *"/metrics"*)
                echo '{"result":{"collections_total":5,"points_total":10000,"vectors_total":10000,"memory_usage":{"vectors":52428800,"payload":10485760,"index":20971520},"requests":{"total":1000,"per_second":10.5}}}'
                ;;
            *"/telemetry"*)
                echo '{"result":{"id":"qdrant-instance-123","version":"1.7.4","system":{"cpu_count":8,"memory_total":16777216},"collections":[{"name":"test_collection","vectors":1000}]}}'
                ;;
            *)
                echo "CURL: $*"
                ;;
        esac
        return 0
    }
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".status"*) echo "ok" ;;
            *".version"*) echo "1.7.4" ;;
            *".State.Running"*) echo "true" ;;
            *".State.Status"*) echo "running" ;;
            *".State.Health.Status"*) echo "healthy" ;;
            *".result.collections_total"*) echo "5" ;;
            *".result.points_total"*) echo "10000" ;;
            *".result.memory_usage.vectors"*) echo "52428800" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock netstat for port checking
    netstat() {
        case "$*" in
            *":6333"*) echo "tcp        0      0 0.0.0.0:6333            0.0.0.0:*               LISTEN" ;;
            *":6334"*) echo "tcp        0      0 0.0.0.0:6334            0.0.0.0:*               LISTEN" ;;
            *) echo "NETSTAT: $*" ;;
        esac
        return 0
    }
    
    # Mock ps for process checking
    ps() {
        echo "  PID TTY          TIME CMD"
        echo " 1234 ?        00:05:30 qdrant"
        return 0
    }
    
    # Mock top for system monitoring
    top() {
        echo "top - 11:00:00 up 1 day,  2:30,  1 user,  load average: 0.50, 0.45, 0.40"
        echo "Tasks: 150 total,   1 running, 149 sleeping"
        echo "KiB Mem : 16777216 total,  8388608 free,  4194304 used,  4194304 buff/cache"
        echo " 1234 qdrant    20   0 1228800 1228800  12288 S  8.5  7.3   5:30.23 qdrant"
    }
    
    # Mock log functions
    log::info() { echo "INFO: $1"; }
    log::error() { echo "ERROR: $1"; }
    log::warn() { echo "WARN: $1"; }
    log::success() { echo "SUCCESS: $1"; }
    log::debug() { echo "DEBUG: $1"; }
    log::header() { echo "=== $1 ==="; }
    
    # Load configuration and messages
    source "${QDRANT_DIR}/config/defaults.sh"
    source "${QDRANT_DIR}/config/messages.sh"
    qdrant::export_config
    qdrant::messages::init
    
    # Load the functions to test
    source "${QDRANT_DIR}/lib/status.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "/tmp/qdrant-test"
}

# Test basic status check
@test "qdrant::status::check returns overall service status" {
    result=$(qdrant::status::check)
    
    [[ "$result" =~ "status" ]] || [[ "$result" =~ "running" ]]
}

# Test service status check with stopped service
@test "qdrant::status::check handles stopped service" {
    # Override docker ps to return empty
    docker() {
        case "$1" in
            "ps") echo "" ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(qdrant::status::check)
    
    [[ "$result" =~ "stopped" ]] || [[ "$result" =~ "not running" ]]
}

# Test container status
@test "qdrant::status::container_status checks Docker container status" {
    result=$(qdrant::status::container_status)
    
    [[ "$result" =~ "container" ]] || [[ "$result" =~ "running" ]]
}

# Test API health status
@test "qdrant::status::api_health checks API health" {
    result=$(qdrant::status::api_health)
    
    [[ "$result" =~ "API" ]] || [[ "$result" =~ "healthy" ]]
}

# Test API health status with failure
@test "qdrant::status::api_health handles API failure" {
    # Override curl to simulate failure
    curl() {
        return 1
    }
    
    result=$(qdrant::status::api_health)
    
    [[ "$result" =~ "API" ]] && [[ "$result" =~ "unhealthy" ]]
}

# Test port status
@test "qdrant::status::port_status checks port accessibility" {
    result=$(qdrant::status::port_status)
    
    [[ "$result" =~ "port" ]] || [[ "$result" =~ "listening" ]]
}

# Test cluster status
@test "qdrant::status::cluster_status checks cluster health" {
    result=$(qdrant::status::cluster_status)
    
    [[ "$result" =~ "cluster" ]] || [[ "$result" =~ "status" ]]
}

# Test collections status
@test "qdrant::status::collections_status checks collections health" {
    result=$(qdrant::status::collections_status)
    
    [[ "$result" =~ "collections" ]] || [[ "$result" =~ "total" ]]
}

# Test memory usage status
@test "qdrant::status::memory_usage checks memory consumption" {
    result=$(qdrant::status::memory_usage)
    
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "usage" ]]
}

# Test disk usage status
@test "qdrant::status::disk_usage checks disk space usage" {
    result=$(qdrant::status::disk_usage)
    
    [[ "$result" =~ "disk" ]] || [[ "$result" =~ "usage" ]]
}

# Test CPU usage status
@test "qdrant::status::cpu_usage checks CPU utilization" {
    result=$(qdrant::status::cpu_usage)
    
    [[ "$result" =~ "CPU" ]] || [[ "$result" =~ "usage" ]]
}

# Test network status
@test "qdrant::status::network_status checks network connectivity" {
    result=$(qdrant::status::network_status)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "connectivity" ]]
}

# Test performance metrics
@test "qdrant::status::performance_metrics retrieves performance data" {
    result=$(qdrant::status::performance_metrics)
    
    [[ "$result" =~ "performance" ]] || [[ "$result" =~ "metrics" ]]
}

# Test request statistics
@test "qdrant::status::request_stats shows API request statistics" {
    result=$(qdrant::status::request_stats)
    
    [[ "$result" =~ "request" ]] || [[ "$result" =~ "statistics" ]]
}

# Test version status
@test "qdrant::status::version_info shows version information" {
    result=$(qdrant::status::version_info)
    
    [[ "$result" =~ "version" ]] || [[ "$result" =~ "1.7.4" ]]
}

# Test uptime status
@test "qdrant::status::uptime shows service uptime" {
    result=$(qdrant::status::uptime)
    
    [[ "$result" =~ "uptime" ]] || [[ "$result" =~ "running" ]]
}

# Test process status
@test "qdrant::status::process_status checks process information" {
    result=$(qdrant::status::process_status)
    
    [[ "$result" =~ "process" ]] || [[ "$result" =~ "PID" ]]
}

# Test log status
@test "qdrant::status::log_status checks log health" {
    result=$(qdrant::status::log_status)
    
    [[ "$result" =~ "log" ]] || [[ "$result" =~ "status" ]]
}

# Test error detection
@test "qdrant::status::detect_errors detects service errors" {
    result=$(qdrant::status::detect_errors)
    
    [[ "$result" =~ "error" ]] || [[ "$result" =~ "none" ]]
}

# Test warning detection
@test "qdrant::status::detect_warnings detects service warnings" {
    result=$(qdrant::status::detect_warnings)
    
    [[ "$result" =~ "warning" ]] || [[ "$result" =~ "none" ]]
}

# Test health score calculation
@test "qdrant::status::health_score calculates overall health score" {
    result=$(qdrant::status::health_score)
    
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "score" ]]
}

# Test detailed status report
@test "qdrant::status::detailed_report generates comprehensive status report" {
    result=$(qdrant::status::detailed_report)
    
    [[ "$result" =~ "report" ]] || [[ "$result" =~ "status" ]]
}

# Test quick status check
@test "qdrant::status::quick_check performs rapid status check" {
    result=$(qdrant::status::quick_check)
    
    [[ "$result" =~ "quick" ]] || [[ "$result" =~ "status" ]]
}

# Test status monitoring
@test "qdrant::status::monitor starts continuous monitoring" {
    result=$(qdrant::status::monitor 1)
    
    [[ "$result" =~ "monitor" ]] || [[ "$result" =~ "status" ]]
}

# Test status alerting
@test "qdrant::status::check_alerts checks for status alerts" {
    result=$(qdrant::status::check_alerts)
    
    [[ "$result" =~ "alert" ]] || [[ "$result" =~ "status" ]]
}

# Test threshold monitoring
@test "qdrant::status::check_thresholds monitors performance thresholds" {
    result=$(qdrant::status::check_thresholds)
    
    [[ "$result" =~ "threshold" ]] || [[ "$result" =~ "monitor" ]]
}

# Test dependency status
@test "qdrant::status::dependency_check checks dependency status" {
    result=$(qdrant::status::dependency_check)
    
    [[ "$result" =~ "dependency" ]] || [[ "$result" =~ "status" ]]
}

# Test configuration status
@test "qdrant::status::config_status checks configuration health" {
    result=$(qdrant::status::config_status)
    
    [[ "$result" =~ "config" ]] || [[ "$result" =~ "status" ]]
}

# Test security status
@test "qdrant::status::security_status checks security configuration" {
    result=$(qdrant::status::security_status)
    
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "status" ]]
}

# Test backup status
@test "qdrant::status::backup_status checks backup health" {
    result=$(qdrant::status::backup_status)
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "status" ]]
}

# Test connectivity status
@test "qdrant::status::connectivity_test tests external connectivity" {
    result=$(qdrant::status::connectivity_test)
    
    [[ "$result" =~ "connectivity" ]] || [[ "$result" =~ "test" ]]
}

# Test API endpoint status
@test "qdrant::status::endpoint_status checks all API endpoints" {
    result=$(qdrant::status::endpoint_status)
    
    [[ "$result" =~ "endpoint" ]] || [[ "$result" =~ "status" ]]
}

# Test data integrity status
@test "qdrant::status::data_integrity checks data integrity" {
    result=$(qdrant::status::data_integrity)
    
    [[ "$result" =~ "data" ]] || [[ "$result" =~ "integrity" ]]
}

# Test index status
@test "qdrant::status::index_status checks index health" {
    result=$(qdrant::status::index_status)
    
    [[ "$result" =~ "index" ]] || [[ "$result" =~ "status" ]]
}

# Test replication status
@test "qdrant::status::replication_status checks replication health" {
    result=$(qdrant::status::replication_status)
    
    [[ "$result" =~ "replication" ]] || [[ "$result" =~ "status" ]]
}

# Test shard status
@test "qdrant::status::shard_status checks shard distribution" {
    result=$(qdrant::status::shard_status)
    
    [[ "$result" =~ "shard" ]] || [[ "$result" =~ "status" ]]
}

# Test consistency status
@test "qdrant::status::consistency_check checks data consistency" {
    result=$(qdrant::status::consistency_check)
    
    [[ "$result" =~ "consistency" ]] || [[ "$result" =~ "check" ]]
}

# Test load balancing status
@test "qdrant::status::load_balance_status checks load balancing" {
    result=$(qdrant::status::load_balance_status)
    
    [[ "$result" =~ "load" ]] || [[ "$result" =~ "balance" ]]
}

# Test failover status
@test "qdrant::status::failover_status checks failover readiness" {
    result=$(qdrant::status::failover_status)
    
    [[ "$result" =~ "failover" ]] || [[ "$result" =~ "status" ]]
}

# Test maintenance status
@test "qdrant::status::maintenance_status checks maintenance mode" {
    result=$(qdrant::status::maintenance_status)
    
    [[ "$result" =~ "maintenance" ]] || [[ "$result" =~ "status" ]]
}

# Test upgrade status
@test "qdrant::status::upgrade_status checks upgrade readiness" {
    result=$(qdrant::status::upgrade_status)
    
    [[ "$result" =~ "upgrade" ]] || [[ "$result" =~ "status" ]]
}

# Test rollback status
@test "qdrant::status::rollback_status checks rollback capability" {
    result=$(qdrant::status::rollback_status)
    
    [[ "$result" =~ "rollback" ]] || [[ "$result" =~ "status" ]]
}

# Test migration status
@test "qdrant::status::migration_status checks migration progress" {
    result=$(qdrant::status::migration_status)
    
    [[ "$result" =~ "migration" ]] || [[ "$result" =~ "status" ]]
}

# Test recovery status
@test "qdrant::status::recovery_status checks recovery readiness" {
    result=$(qdrant::status::recovery_status)
    
    [[ "$result" =~ "recovery" ]] || [[ "$result" =~ "status" ]]
}

# Test disaster recovery status
@test "qdrant::status::disaster_recovery_status checks disaster recovery" {
    result=$(qdrant::status::disaster_recovery_status)
    
    [[ "$result" =~ "disaster" ]] || [[ "$result" =~ "recovery" ]]
}

# Test compliance status
@test "qdrant::status::compliance_check checks compliance status" {
    result=$(qdrant::status::compliance_check)
    
    [[ "$result" =~ "compliance" ]] || [[ "$result" =~ "check" ]]
}

# Test audit status
@test "qdrant::status::audit_status checks audit trail health" {
    result=$(qdrant::status::audit_status)
    
    [[ "$result" =~ "audit" ]] || [[ "$result" =~ "status" ]]
}

# Test licensing status
@test "qdrant::status::license_status checks license compliance" {
    result=$(qdrant::status::license_status)
    
    [[ "$result" =~ "license" ]] || [[ "$result" =~ "status" ]]
}

# Test service discovery status
@test "qdrant::status::service_discovery checks service discovery" {
    result=$(qdrant::status::service_discovery)
    
    [[ "$result" =~ "service" ]] || [[ "$result" =~ "discovery" ]]
}

# Test integration status
@test "qdrant::status::integration_status checks external integrations" {
    result=$(qdrant::status::integration_status)
    
    [[ "$result" =~ "integration" ]] || [[ "$result" =~ "status" ]]
}

# Test API compatibility status
@test "qdrant::status::api_compatibility checks API version compatibility" {
    result=$(qdrant::status::api_compatibility)
    
    [[ "$result" =~ "API" ]] || [[ "$result" =~ "compatibility" ]]
}

# Test client status
@test "qdrant::status::client_status checks connected clients" {
    result=$(qdrant::status::client_status)
    
    [[ "$result" =~ "client" ]] || [[ "$result" =~ "status" ]]
}

# Test session status
@test "qdrant::status::session_status checks active sessions" {
    result=$(qdrant::status::session_status)
    
    [[ "$result" =~ "session" ]] || [[ "$result" =~ "status" ]]
}

# Test cache status
@test "qdrant::status::cache_status checks cache performance" {
    result=$(qdrant::status::cache_status)
    
    [[ "$result" =~ "cache" ]] || [[ "$result" =~ "status" ]]
}

# Test queue status
@test "qdrant::status::queue_status checks operation queues" {
    result=$(qdrant::status::queue_status)
    
    [[ "$result" =~ "queue" ]] || [[ "$result" =~ "status" ]]
}

# Test worker status
@test "qdrant::status::worker_status checks worker processes" {
    result=$(qdrant::status::worker_status)
    
    [[ "$result" =~ "worker" ]] || [[ "$result" =~ "status" ]]
}

# Test thread status
@test "qdrant::status::thread_status checks thread utilization" {
    result=$(qdrant::status::thread_status)
    
    [[ "$result" =~ "thread" ]] || [[ "$result" =~ "status" ]]
}

# Test resource utilization
@test "qdrant::status::resource_utilization shows comprehensive resource usage" {
    result=$(qdrant::status::resource_utilization)
    
    [[ "$result" =~ "resource" ]] || [[ "$result" =~ "utilization" ]]
}

# Test system limits
@test "qdrant::status::system_limits checks system resource limits" {
    result=$(qdrant::status::system_limits)
    
    [[ "$result" =~ "system" ]] || [[ "$result" =~ "limits" ]]
}

# Test capacity planning
@test "qdrant::status::capacity_planning provides capacity insights" {
    result=$(qdrant::status::capacity_planning)
    
    [[ "$result" =~ "capacity" ]] || [[ "$result" =~ "planning" ]]
}

# Test trend analysis
@test "qdrant::status::trend_analysis analyzes performance trends" {
    result=$(qdrant::status::trend_analysis)
    
    [[ "$result" =~ "trend" ]] || [[ "$result" =~ "analysis" ]]
}

# Test status export
@test "qdrant::status::export_status exports status information" {
    result=$(qdrant::status::export_status "/tmp/status.json")
    
    [[ "$result" =~ "export" ]] || [[ "$result" =~ "status" ]]
}

# Test status import
@test "qdrant::status::import_status imports status configuration" {
    result=$(qdrant::status::import_status "/tmp/status.json")
    
    [[ "$result" =~ "import" ]] || [[ "$result" =~ "status" ]]
}

# Test status comparison
@test "qdrant::status::compare_status compares status snapshots" {
    result=$(qdrant::status::compare_status "/tmp/status1.json" "/tmp/status2.json")
    
    [[ "$result" =~ "compare" ]] || [[ "$result" =~ "status" ]]
}

# Test status visualization
@test "qdrant::status::visualize_status creates status visualization" {
    result=$(qdrant::status::visualize_status)
    
    [[ "$result" =~ "visualize" ]] || [[ "$result" =~ "status" ]]
}

# Test status dashboard
@test "qdrant::status::dashboard generates status dashboard" {
    result=$(qdrant::status::dashboard)
    
    [[ "$result" =~ "dashboard" ]] || [[ "$result" =~ "status" ]]
}