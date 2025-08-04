#!/usr/bin/env bats
# Tests for Unstructured.io status.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export UNSTRUCTURED_IO_CUSTOM_PORT="9999"
    export UNSTRUCTURED_IO_CONTAINER_NAME="unstructured-io-test"
    export UNSTRUCTURED_IO_BASE_URL="http://localhost:9999"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    UNSTRUCTURED_IO_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock system functions
    
    # Mock Docker functions
    
    # Mock curl for health checks
    
    # Mock jq for JSON parsing
    jq() {
        case "$*" in
            *".status"*)
                echo "healthy"
                ;;
            *".version"*)
                echo "0.10.0"
                ;;
            *".uptime"*)
                echo "120"
                ;;
            *".supported_types"*)
                echo '["pdf","docx","txt"]'
                ;;
            *".max_file_size"*)
                echo "50MB"
                ;;
            *) echo "{}" ;;
        esac
    }
    
    # Mock log functions
    
    
    
    
    # Mock common functions
    unstructured_io::container_exists() { return 0; }  # Container exists by default
    unstructured_io::container_running() { return 0; }  # Container running by default
    
    # Load configuration and messages
    source "${UNSTRUCTURED_IO_DIR}/config/defaults.sh"
    source "${UNSTRUCTURED_IO_DIR}/config/messages.sh"
    unstructured_io::export_config
    unstructured_io::export_messages
    
    # Load the functions to test
    source "${UNSTRUCTURED_IO_DIR}/lib/status.sh"
}

# Test status check with healthy service (verbose)
@test "unstructured_io::status shows healthy service status verbosely" {
    result=$(unstructured_io::status "yes")
    
    [[ "$result" =~ "INFO: " ]]
    [[ "$result" =~ "container" ]]
    [[ "$result" =~ "running" ]]
    [[ "$result" =~ "healthy" ]]
    [[ "$result" =~ "9999" ]]
}

# Test status check with healthy service (silent)
@test "unstructured_io::status returns success silently" {
    unstructured_io::status "no"
    [ "$?" -eq 0 ]
}

# Test status check with missing container
@test "unstructured_io::status fails when container missing" {
    # Override container check to return no container
    unstructured_io::container_exists() { return 1; }
    
    run unstructured_io::status "yes"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]]
}

# Test status check with stopped container
@test "unstructured_io::status fails when container stopped" {
    # Override running check to return stopped
    unstructured_io::container_running() { return 1; }
    
    run unstructured_io::status "yes"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running" ]]
}

# Test status check with unhealthy service
@test "unstructured_io::status fails when service unhealthy" {
    # Override curl to return unhealthy status
    curl() {
        case "$*" in
            *"/health"*)
                return 1  # Health check fails
                ;;
            *) return 0 ;;
        esac
    }
    
    run unstructured_io::status "yes"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "health check failed" ]]
}

# Test detailed status display
@test "unstructured_io::detailed_status shows comprehensive information" {
    result=$(unstructured_io::detailed_status)
    
    [[ "$result" =~ "Unstructured.io Status" ]]
    [[ "$result" =~ "Container:" ]]
    [[ "$result" =~ "Image:" ]]
    [[ "$result" =~ "Port:" ]]
    [[ "$result" =~ "Health:" ]]
    [[ "$result" =~ "Version:" ]]
    [[ "$result" =~ "Uptime:" ]]
    [[ "$result" =~ "Resource Usage:" ]]
}

# Test health check function
@test "unstructured_io::health_check returns service health status" {
    result=$(unstructured_io::health_check)
    
    [[ "$result" =~ "healthy" ]]
}

# Test health check with service unavailable
@test "unstructured_io::health_check handles service unavailable" {
    # Override curl to fail
    curl() {
        return 1
    }
    
    run unstructured_io::health_check
    [ "$status" -eq 1 ]
}

# Test version check function
@test "unstructured_io::get_version returns service version" {
    result=$(unstructured_io::get_version)
    
    [[ "$result" =~ "0.10.0" ]]
}

# Test version check with service unavailable
@test "unstructured_io::get_version handles service unavailable" {
    # Override curl to fail
    curl() {
        return 1
    }
    
    run unstructured_io::get_version
    [ "$status" -eq 1 ]
}

# Test service info retrieval
@test "unstructured_io::get_service_info returns service information" {
    result=$(unstructured_io::get_service_info)
    
    [[ "$result" =~ "version" ]]
    [[ "$result" =~ "supported_types" ]]
    [[ "$result" =~ "max_file_size" ]]
}

# Test service info with service unavailable
@test "unstructured_io::get_service_info handles service unavailable" {
    # Override curl to fail
    curl() {
        return 1
    }
    
    run unstructured_io::get_service_info
    [ "$status" -eq 1 ]
}

# Test container stats retrieval
@test "unstructured_io::get_container_stats returns resource usage" {
    result=$(unstructured_io::get_container_stats)
    
    [[ "$result" =~ "CPU" ]]
    [[ "$result" =~ "MEM" ]]
    [[ "$result" =~ "0.50%" ]]
    [[ "$result" =~ "512MiB" ]]
}

# Test container stats with missing container
@test "unstructured_io::get_container_stats handles missing container" {
    # Override container check to return no container
    unstructured_io::container_exists() { return 1; }
    
    run unstructured_io::get_container_stats
    [ "$status" -eq 1 ]
}

# Test port status check
@test "unstructured_io::check_port_status verifies port accessibility" {
    unstructured_io::check_port_status
    [ "$?" -eq 0 ]
}

# Test port status check with inaccessible port
@test "unstructured_io::check_port_status fails with inaccessible port" {
    # Override curl to fail on port check
    curl() {
        case "$*" in
            *"localhost:$UNSTRUCTURED_IO_PORT"*)
                return 1
                ;;
            *) return 0 ;;
        esac
    }
    
    run unstructured_io::check_port_status
    [ "$status" -eq 1 ]
}

# Test container network information
@test "unstructured_io::get_network_info returns container network details" {
    result=$(unstructured_io::get_network_info)
    
    [[ "$result" =~ "Port:" ]]
    [[ "$result" =~ "9999" ]]
}

# Test container network info with missing container
@test "unstructured_io::get_network_info handles missing container" {
    # Override container check to return no container
    unstructured_io::container_exists() { return 1; }
    
    run unstructured_io::get_network_info
    [ "$status" -eq 1 ]
}

# Test uptime calculation
@test "unstructured_io::get_uptime returns service uptime" {
    result=$(unstructured_io::get_uptime)
    
    [[ "$result" =~ "120" ]]
}

# Test uptime with service unavailable
@test "unstructured_io::get_uptime handles service unavailable" {
    # Override curl to fail
    curl() {
        return 1
    }
    
    run unstructured_io::get_uptime
    [ "$status" -eq 1 ]
}

# Test service availability check
@test "unstructured_io::is_service_available returns true for available service" {
    unstructured_io::is_service_available
    [ "$?" -eq 0 ]
}

# Test service availability check with unavailable service
@test "unstructured_io::is_service_available returns false for unavailable service" {
    # Override health check to fail
    unstructured_io::health_check() {
        return 1
    }
    
    run unstructured_io::is_service_available
    [ "$status" -eq 1 ]
}

# Test configuration validation
@test "unstructured_io::validate_configuration checks service configuration" {
    result=$(unstructured_io::validate_configuration)
    
    [[ "$result" =~ "Configuration:" ]]
    [[ "$result" =~ "Port:" ]]
    [[ "$result" =~ "Container:" ]]
    [[ "$result" =~ "Image:" ]]
}

# Test service metrics collection
@test "unstructured_io::collect_metrics gathers service metrics" {
    result=$(unstructured_io::collect_metrics)
    
    [[ "$result" =~ "status" ]]
    [[ "$result" =~ "version" ]]
    [[ "$result" =~ "uptime" ]]
    [[ "$result" =~ "cpu_usage" ]]
    [[ "$result" =~ "memory_usage" ]]
}