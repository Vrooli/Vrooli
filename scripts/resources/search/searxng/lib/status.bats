#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Load the functions we are testing (required for bats isolation)
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/status.sh"
    
    # Path to the script under test
    SCRIPT_PATH="$BATS_TEST_DIRNAME/status.sh"
    SEARXNG_DIR="$BATS_TEST_DIRNAME/.."
    
    # Source dependencies
    local resources_dir="$SEARXNG_DIR/../.."
    local helpers_dir="$resources_dir/../helpers"
    
    # Source utilities first
    source "$helpers_dir/utils/log.sh"
    source "$helpers_dir/utils/system.sh"
    source "$helpers_dir/utils/ports.sh"
    source "$helpers_dir/utils/flow.sh"
    source "$resources_dir/port-registry.sh"
    source "$resources_dir/common.sh"
    
    # Source config and messages
    source "$SEARXNG_DIR/config/defaults.sh"
    source "$SEARXNG_DIR/config/messages.sh"
    searxng::export_config
    
    # Source dependencies
    source "$SEARXNG_DIR/lib/common.sh"
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Mock message function
    searxng::message() {
        local type="$1"
        local msg_var="$2"
        echo "$type: ${!msg_var}"
    }
    
    # Mock status functions
    searxng::get_status() {
        echo "$MOCK_STATUS"
    }
    
    searxng::is_installed() { [[ "$MOCK_SEARXNG_INSTALLED" == "yes" ]]; }
    searxng::is_running() { [[ "$MOCK_SEARXNG_RUNNING" == "yes" ]]; }
    searxng::is_healthy() { [[ "$MOCK_SEARXNG_HEALTHY" == "yes" ]]; }
    
    # Mock Docker functions
    docker() {
        case "$1" in
            "container")
                if [[ "$2" == "inspect" ]]; then
                    case "$3" in
                        "--format={{.State.Status}}")
                            echo "$MOCK_CONTAINER_STATUS"
                            ;;
                        "--format={{.State.StartedAt}}")
                            echo "2024-01-01T10:00:00Z"
                            ;;
                        "--format={{.State.Health.Status}}")
                            echo "$MOCK_HEALTH_STATUS"
                            ;;
                        "--format={{.Config.Image}}")
                            echo "searxng/searxng:latest"
                            ;;
                    esac
                    return 0
                fi
                ;;
            "network")
                if [[ "$2" == "inspect" ]]; then
                    if [[ "$MOCK_NETWORK_EXISTS" == "yes" ]]; then
                        return 0
                    else
                        return 1
                    fi
                fi
                ;;
            "image")
                if [[ "$2" == "inspect" ]]; then
                    if [[ "$MOCK_IMAGE_EXISTS" == "yes" ]]; then
                        return 0
                    else
                        return 1
                    fi
                fi
                ;;
            "logs")
                echo "Mock log line 1"
                echo "Mock error occurred"
                echo "Mock warning message"
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    # Mock system functions
    resources::is_service_running() {
        if [[ "$MOCK_PORT_AVAILABLE" == "yes" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    ports::validate_port() {
        local port="$1"
        if [[ "$port" =~ ^[0-9]+$ ]] && [[ "$port" -ge 1024 ]] && [[ "$port" -le 65535 ]]; then
            return 0
        else
            return 1
        fi
    }
    
    # Mock curl for API checks (override shared mock for SearXNG-specific behavior)
    curl() {
        if [[ "$MOCK_API_RESPONDS" == "yes" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    # Mock command availability
    command() {
        case "$*" in
            "-v docker")
                if [[ "$MOCK_DOCKER_AVAILABLE" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            "-v curl")
                if [[ "$MOCK_CURL_AVAILABLE" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    # Set default mocks
    export MOCK_STATUS="healthy"
    export MOCK_SEARXNG_INSTALLED="yes"
    export MOCK_SEARXNG_RUNNING="yes"
    export MOCK_SEARXNG_HEALTHY="yes"
    export MOCK_CONTAINER_STATUS="running"
    export MOCK_HEALTH_STATUS="healthy"
    export MOCK_NETWORK_EXISTS="yes"
    export MOCK_IMAGE_EXISTS="yes"
    export MOCK_PORT_AVAILABLE="yes"
    export MOCK_API_RESPONDS="yes"
    export MOCK_DOCKER_AVAILABLE="yes"
    export MOCK_CURL_AVAILABLE="yes"
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "sourcing status.sh defines required functions" {
    local required_functions=(
        "searxng::show_status"
        "searxng::show_detailed_status"
        "searxng::check_api_endpoints"
        "searxng::diagnose"
        "searxng::analyze_logs"
        "searxng::show_troubleshooting"
        "searxng::monitor"
    )
    
    for func in "${required_functions[@]}"; do
        run bash -c "declare -f $func"
        [ "$status" -eq 0 ]
    done
}

# ============================================================================
# Status Display Tests
# ============================================================================

@test "searxng::show_status displays healthy status correctly" {
    export MOCK_STATUS="healthy"
    
    run searxng::show_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HEADER: SearXNG Status Report" ]]
    [[ "$output" =~ "Status: healthy" ]]
    [[ "$output" =~ "SUCCESS:" ]]
    [[ "$output" =~ "running and healthy" ]]
}

@test "searxng::show_status displays not_installed status" {
    export MOCK_STATUS="not_installed"
    
    run searxng::show_status
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Status: not_installed" ]]
    [[ "$output" =~ "INFO:" ]]
    [[ "$output" =~ "not installed" ]]
    [[ "$output" =~ "./manage.sh --action install" ]]
}

@test "searxng::show_status displays stopped status" {
    export MOCK_STATUS="stopped"
    
    run searxng::show_status
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Status: stopped" ]]
    [[ "$output" =~ "WARNING:" ]]
    [[ "$output" =~ "not running" ]]
    [[ "$output" =~ "./manage.sh --action start" ]]
}

@test "searxng::show_status displays unhealthy status" {
    export MOCK_STATUS="unhealthy"
    export MOCK_SEARXNG_INSTALLED="yes"
    
    run searxng::show_status
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Status: unhealthy" ]]
    [[ "$output" =~ "WARNING:" ]]
    [[ "$output" =~ "running but not healthy" ]]
}

@test "searxng::show_status includes basic information" {
    run searxng::show_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Port: $SEARXNG_PORT" ]]
    [[ "$output" =~ "Base URL: $SEARXNG_BASE_URL" ]]
    [[ "$output" =~ "Container: $SEARXNG_CONTAINER_NAME" ]]
    [[ "$output" =~ "Data Directory: $SEARXNG_DATA_DIR" ]]
}

# ============================================================================
# Detailed Status Tests
# ============================================================================

@test "searxng::show_detailed_status shows container information" {
    run searxng::show_detailed_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Container Details:" ]]
    [[ "$output" =~ "Docker Status: running" ]]
    [[ "$output" =~ "Started At: 2024-01-01T10:00:00Z" ]]
    [[ "$output" =~ "Health Status: healthy" ]]
    [[ "$output" =~ "Image: searxng/searxng:latest" ]]
}

@test "searxng::show_detailed_status shows network status" {
    run searxng::show_detailed_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Network Status:" ]]
    [[ "$output" =~ "Port $SEARXNG_PORT: ✅ Open" ]]
}

@test "searxng::show_detailed_status shows port conflicts" {
    export MOCK_PORT_AVAILABLE="no"
    
    run searxng::show_detailed_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Port $SEARXNG_PORT: ❌ Not responding" ]]
}

@test "searxng::show_detailed_status shows configuration summary" {
    run searxng::show_detailed_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Configuration:" ]]
    [[ "$output" =~ "Bind Address: $SEARXNG_BIND_ADDRESS" ]]
    [[ "$output" =~ "Default Engines: $SEARXNG_DEFAULT_ENGINES" ]]
    [[ "$output" =~ "Safe Search: $SEARXNG_SAFE_SEARCH" ]]
    [[ "$output" =~ "Rate Limiting: $SEARXNG_LIMITER_ENABLED" ]]
    [[ "$output" =~ "Redis Caching: $SEARXNG_ENABLE_REDIS" ]]
}

# ============================================================================
# API Endpoints Check Tests
# ============================================================================

@test "searxng::check_api_endpoints shows all endpoints healthy" {
    export MOCK_API_RESPONDS="yes"
    
    run searxng::check_api_endpoints
    [ "$status" -eq 0 ]
    [[ "$output" =~ "API Endpoints:" ]]
    [[ "$output" =~ "/stats: ✅ Responding" ]]
    [[ "$output" =~ "/search: ✅ Responding" ]]
    [[ "$output" =~ "/config: ✅ Responding" ]]
}

@test "searxng::check_api_endpoints shows failing endpoints" {
    export MOCK_API_RESPONDS="no"
    
    run searxng::check_api_endpoints
    [ "$status" -eq 0 ]
    [[ "$output" =~ "/stats: ❌ Not responding" ]]
    [[ "$output" =~ "/search: ❌ Not responding" ]]
    [[ "$output" =~ "/config: ❌ Not responding" ]]
}

@test "searxng::check_api_endpoints fails when container not running" {
    export MOCK_SEARXNG_RUNNING="no"
    
    run searxng::check_api_endpoints
    [ "$status" -eq 1 ]
    [[ "$output" =~ "❌ Container not running" ]]
}

# ============================================================================
# Diagnostic Tests
# ============================================================================

@test "searxng::diagnose performs comprehensive system check" {
    run searxng::diagnose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HEADER: SearXNG Diagnostic Report" ]]
    [[ "$output" =~ "System Requirements:" ]]
    [[ "$output" =~ "Docker: ✅ Installed" ]]
    [[ "$output" =~ "Docker Daemon: ✅ Running" ]]
    [[ "$output" =~ "curl: ✅ Available" ]]
}

@test "searxng::diagnose detects missing dependencies" {
    export MOCK_DOCKER_AVAILABLE="no"
    export MOCK_CURL_AVAILABLE="no"
    
    run searxng::diagnose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Docker: ❌ Not found" ]]
    [[ "$output" =~ "curl: ❌ Not found" ]]
}

@test "searxng::diagnose validates configuration" {
    run searxng::diagnose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Configuration Validation:" ]]
    [[ "$output" =~ "Port: $SEARXNG_PORT" ]]
    [[ "$output" =~ "✅ Valid port number" ]]
    [[ "$output" =~ "✅ No port conflicts" ]]
    [[ "$output" =~ "Secret Key:" ]]
    [[ "$output" =~ "✅ Adequate length" ]]
}

@test "searxng::diagnose checks Docker environment" {
    run searxng::diagnose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Docker Environment:" ]]
    [[ "$output" =~ "Network: ✅ $SEARXNG_NETWORK_NAME exists" ]]
    [[ "$output" =~ "Image: ✅ $SEARXNG_IMAGE available" ]]
}

@test "searxng::diagnose detects missing Docker resources" {
    export MOCK_NETWORK_EXISTS="no"
    export MOCK_IMAGE_EXISTS="no"
    
    run searxng::diagnose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Network: ❌ $SEARXNG_NETWORK_NAME not found" ]]
    [[ "$output" =~ "Image: ❌ $SEARXNG_IMAGE not found" ]]
}

@test "searxng::diagnose shows troubleshooting suggestions" {
    run searxng::diagnose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HEADER: Troubleshooting Suggestions" ]]
}

# ============================================================================
# Log Analysis Tests
# ============================================================================

@test "searxng::analyze_logs analyzes container logs" {
    export MOCK_SEARXNG_INSTALLED="yes"
    
    run searxng::analyze_logs
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Error messages:" ]]
    [[ "$output" =~ "Warning messages:" ]]
}

@test "searxng::analyze_logs fails when container not available" {
    export MOCK_SEARXNG_INSTALLED="no"
    
    run searxng::analyze_logs
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Container not available for log analysis" ]]
}

@test "searxng::analyze_logs detects specific issues" {
    export MOCK_SEARXNG_INSTALLED="yes"
    
    # Override docker logs to include specific issues
    docker() {
        if [[ "$1" == "logs" ]]; then
            echo "permission denied error"
            echo "address already in use"
            echo "connection refused"
        fi
    }
    
    run searxng::analyze_logs
    [ "$status" -eq 0 ]
    [[ "$output" =~ "⚠️  Permission issues detected" ]]
    [[ "$output" =~ "⚠️  Port conflict detected" ]]
    [[ "$output" =~ "⚠️  Connection issues detected" ]]
}

# ============================================================================
# Troubleshooting Tests
# ============================================================================

@test "searxng::show_troubleshooting provides status-specific guidance" {
    export MOCK_STATUS="not_installed"
    
    run searxng::show_troubleshooting
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Install SearXNG: ./manage.sh --action install" ]]
    [[ "$output" =~ "Check Docker is installed" ]]
}

@test "searxng::show_troubleshooting handles stopped status" {
    export MOCK_STATUS="stopped"
    
    run searxng::show_troubleshooting
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Start SearXNG: ./manage.sh --action start" ]]
}

@test "searxng::show_troubleshooting handles unhealthy status" {
    export MOCK_STATUS="unhealthy"
    
    run searxng::show_troubleshooting
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Check logs: ./manage.sh --action logs" ]]
    [[ "$output" =~ "Restart service: ./manage.sh --action restart" ]]
}

@test "searxng::show_troubleshooting includes general guidance" {
    run searxng::show_troubleshooting
    [ "$status" -eq 0 ]
    [[ "$output" =~ "General troubleshooting:" ]]
}

# ============================================================================
# Monitoring Tests
# ============================================================================

@test "searxng::monitor displays status continuously" {
    # Mock date and sleep to control the loop
    local call_count=0
    date() {
        echo "2024-01-01 10:00:0$((call_count++))"
    }
    
    sleep() {
        # Exit after first iteration to avoid infinite loop
        if [[ $call_count -gt 1 ]]; then
            exit 0
        fi
    }
    
    run /usr/bin/timeout 1s searxng::monitor 1
    # Status may be 0 (success) or 124 (timeout) - both are acceptable for this test
    [[ "$status" -eq 0 || "$status" -eq 124 ]]
    [[ "$output" =~ "Starting SearXNG monitoring" ]]
    [[ "$output" =~ "Status: healthy" ]]
}

@test "searxng::monitor shows response time for healthy status" {
    # Mock curl to return timing
    curl() {
        if [[ "$*" =~ "-w" ]]; then
            echo "0.123"
            return 0
        fi
        return 0
    }
    
    # Control the monitoring loop
    local call_count=0
    sleep() {
        if [[ $call_count -gt 0 ]]; then
            exit 0
        fi
        ((call_count++))
    }
    
    run /usr/bin/timeout 1s searxng::monitor 1
    [[ "$status" -eq 0 || "$status" -eq 124 ]]
    [[ "$output" =~ "Response:" ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "status functions handle all status combinations correctly" {
    local statuses=("not_installed" "stopped" "unhealthy" "healthy")
    
    for status in "${statuses[@]}"; do
        export MOCK_STATUS="$status"
        
        run searxng::show_status
        # Check that each status is handled (returns appropriate exit code)
        if [[ "$status" == "healthy" ]]; then
            [ "$status" -eq 0 ]
        else
            [ "$status" -eq 1 ]
        fi
    done
}

@test "diagnostic checks are comprehensive and accurate" {
    run searxng::diagnose
    [ "$status" -eq 0 ]
    
    # Verify all major diagnostic sections are present
    [[ "$output" =~ "System Requirements:" ]]
    [[ "$output" =~ "Configuration Validation:" ]]
    [[ "$output" =~ "Docker Environment:" ]]
    [[ "$output" =~ "Recent Log Analysis:" ]]
    [[ "$output" =~ "Troubleshooting Suggestions:" ]]
}
