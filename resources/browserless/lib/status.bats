#!/usr/bin/env bats
# Tests for Browserless status.sh functions
bats_require_minimum_version 1.5.0

# Setup paths and source var.sh first
SCRIPT_DIR="$(builtin cd "${BATS_TEST_FILENAME%/*}" && builtin pwd)"
# shellcheck disable=SC1091
source "$(builtin cd "${SCRIPT_DIR%/*/*/*/*/*}" && builtin pwd)/lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "browserless"
    
    # Set up directories and paths once
    export BROWSERLESS_DIR="${SCRIPT_DIR}/.."
    export MOCK_DIR="${var_SCRIPTS_TEST_DIR}/fixtures/mocks"
    
    # Load configuration and messages once
    # shellcheck disable=SC1091
    source "${BROWSERLESS_DIR}/config/defaults.sh"
    # shellcheck disable=SC1091
    source "${BROWSERLESS_DIR}/config/messages.sh"
    
    # Load status functions once
    source "${SCRIPT_DIR}/status.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment (lightweight per-test)
    export BROWSERLESS_CUSTOM_PORT="9999"
    export BROWSERLESS_CONTAINER_NAME="browserless-test"
    export BROWSERLESS_BASE_URL="http://localhost:9999"
    export BROWSERLESS_HEALTH_CHECK_MAX_ATTEMPTS="3"
    export BROWSERLESS_API_TIMEOUT="10"
    export BROWSERLESS_HEALTH_CHECK_INTERVAL="1"
    export BROWSERLESS_MAX_BROWSERS="5"
    export BROWSERLESS_HEADLESS="yes"
    export BROWSERLESS_TIMEOUT="30000"
    
    # Export config functions (lightweight)
    browserless::export_config
    browserless::export_messages
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test health check with healthy service
@test "browserless::is_healthy succeeds when service is responsive" {
    # Mock successful curl
    curl() {
        if [[ "$*" == *"/pressure"* ]]; then
            return 0
        fi
        return 1
    }
    
    run browserless::is_healthy
    [ "$status" -eq 0 ]
}

# Test health check with unresponsive service
@test "browserless::is_healthy fails when service is unresponsive" {
    # Mock failed curl
    curl() {
        return 1
    }
    
    # Mock sleep to speed up test
    sleep() { return 0; }
    
    run browserless::is_healthy
    [ "$status" -eq 1 ]
}

# Test health check without curl
@test "browserless::is_healthy fails when curl not available" {
    # Override system command check
    system::is_command() {
        return 1
    }
    
    run browserless::is_healthy
    [ "$status" -eq 1 ]
}

# Test pressure data retrieval
@test "browserless::get_pressure returns JSON when service responds" {
    # Mock successful curl with JSON response
    curl() {
        if [[ "$*" == *"/pressure"* ]]; then
            echo '{"running":2,"queued":0,"maxConcurrent":5,"isAvailable":true}'
            return 0
        fi
        return 1
    }
    
    run browserless::get_pressure
    
    [ "$status" -eq 0 ]
    [[ "$output" == *'"running":2'* ]]
    [[ "$output" == *'"maxConcurrent":5'* ]]
}

# Test pressure data with service down
@test "browserless::get_pressure returns empty JSON when service down" {
    # Mock failed curl
    curl() {
        return 1
    }
    
    run browserless::get_pressure
    
    [ "$status" -eq 0 ]
    [ "$output" = "{}" ]
}

# Test detailed status display with valid data
@test "browserless::show_detailed_status displays parsed metrics" {
    # Mock get_pressure function
    browserless::get_pressure() {
        echo '{"running":3,"queued":1,"maxConcurrent":5,"isAvailable":true,"cpu":0.25,"memory":0.4}'
    }
    
    # Mock jq command - need to handle the complex format string
    jq() {
        if [[ "$1" == "-r" ]]; then
            # Handle the multi-line format string
            case "$2" in
                *"Running browsers:"*) cat << 'EOF'
  Running browsers: 3/5
  Queued requests: 1
  Recently rejected: 0
  Available: true
  CPU usage: 25%
  Memory usage: 40%
EOF
                    ;;
                *) echo "parse error" ;;
            esac
        else
            echo '{"running":3,"queued":1,"maxConcurrent":5,"isAvailable":true,"cpu":0.25,"memory":0.4}'
        fi
    }
    
    # Mock command check for jq
    command() {
        if [[ "$1" == "-v" && "$2" == "jq" ]]; then
            return 0
        fi
        return 1
    }
    
    run browserless::show_detailed_status
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Running browsers: 3"* ]]
    [[ "$output" == *"Queued requests: 1"* ]]
    [[ "$output" == *"CPU usage: 25%"* ]]
    [[ "$output" == *"Memory usage: 40%"* ]]
}

# Test detailed status with no jq available
@test "browserless::show_detailed_status handles missing jq gracefully" {
    # Mock get_pressure function
    browserless::get_pressure() {
        echo '{"running":2,"queued":0}'
    }
    
    # Mock command check to return false for jq
    command() {
        return 1
    }
    
    run browserless::show_detailed_status
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Status: Available"* ]]
    [[ "$output" == *'{"running":2,"queued":0}'* ]]
}

# Test comprehensive status display
@test "browserless::show_status shows complete service information" {
    # Mock container functions
    browserless::container_exists() { return 0; }
    browserless::is_running() { return 0; }
    browserless::is_healthy() { return 0; }
    browserless::show_resource_usage() { echo "CPU: 15% | Memory: 512MB"; }
    browserless::show_detailed_status() { echo "Browser sessions: 2/5"; }
    
    # Mock docker info
    docker() {
        case "$1" in
            "info") return 0 ;;
            *) return 0 ;;
        esac
    }
    
    run browserless::show_status
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Browserless Status"* ]]
    [[ "$output" == *"Browserless container is running"* ]]
    [[ "$output" == *"Browserless API is healthy"* ]]
    [[ "$output" == *"Max Browsers: 5"* ]]
}

# Test status when Docker not installed
@test "browserless::show_status handles missing Docker" {
    # Override system command check
    system::is_command() {
        return 1
    }
    
    run browserless::show_status
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Docker is not installed"* ]]
}

# Test status when container exists but not running
@test "browserless::show_status handles stopped container" {
    # Mock container functions
    browserless::container_exists() { return 0; }
    browserless::is_running() { return 1; }
    
    # Mock docker commands
    docker() {
        case "$1" in
            "info") return 0 ;;
            *) return 0 ;;
        esac
    }
    
    run browserless::show_status
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"container exists but is not running"* ]]
    [[ "$output" == *"Start with: $0 --action start"* ]]
}

# Test status when not installed
@test "browserless::show_status handles missing installation" {
    # Mock container functions
    browserless::container_exists() { return 1; }
    
    # Mock docker commands
    docker() {
        case "$1" in
            "info") return 0 ;;
            *) return 0 ;;
        esac
    }
    
    run browserless::show_status
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Browserless is not installed"* ]]
    [[ "$output" == *"Install with: $0 --action install"* ]]
}

# Test wait for healthy with quick success
@test "browserless::wait_for_healthy succeeds when service becomes healthy" {
    local attempt=0
    
    # Mock health check that succeeds on second attempt
    browserless::is_healthy() {
        attempt=$((attempt + 1))
        if [ $attempt -ge 2 ]; then
            return 0
        fi
        return 1
    }
    
    # Mock sleep
    sleep() { return 0; }
    
    run browserless::wait_for_healthy 10
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Browserless is healthy and responsive"* ]]
}

# Test wait for healthy with timeout
@test "browserless::wait_for_healthy times out when service never becomes healthy" {
    # Mock health check that always fails
    browserless::is_healthy() {
        return 1
    }
    
    # Mock sleep
    sleep() { return 0; }
    
    run browserless::wait_for_healthy 5
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Browserless started but health check failed"* ]]
}

# Test service info display
@test "browserless::show_info displays complete service information" {
    run browserless::show_info
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Browserless Resource Information"* ]]
    [[ "$output" == *"ID: browserless"* ]]
    [[ "$output" == *"Category: agents"* ]]
    [[ "$output" == *"Service URL: $BROWSERLESS_BASE_URL"* ]]
    [[ "$output" == *"POST $BROWSERLESS_BASE_URL/chrome/screenshot"* ]]
    [[ "$output" == *"For more information, visit: https://www.browserless.io/docs/"* ]]
}