#!/usr/bin/env bats
# Tests for Ollama status and service management functions

# Get script directory first
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
STATUS_BATS_DIR="$APP_ROOT/resources/ollama/lib"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "ollama"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    OLLAMA_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and messages once
    source "${OLLAMA_DIR}/config/defaults.sh"
    source "${OLLAMA_DIR}/config/messages.sh"
    
    # Load status functions once
    source "${SCRIPT_DIR}/status.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_OLLAMA_DIR="$OLLAMA_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    OLLAMA_DIR="${SETUP_FILE_OLLAMA_DIR}"
    
    # Set test environment
    export OLLAMA_PORT="11434"
    export OLLAMA_BASE_URL="http://localhost:11434"
    export OLLAMA_SERVICE_NAME="ollama"
    export FORCE="no"
    
    # Mock message variables
    export MSG_OLLAMA_RUNNING="Ollama is running"
    export MSG_OLLAMA_STARTED_NO_API="Ollama started but API not responding"
    
    # Mock functions
    resources::binary_exists() { return 0; }  # Default: binary exists
    resources::is_service_running() { return 0; }  # Default: service running
    resources::check_http_health() { return 0; }  # Default: healthy
    resources::can_sudo() { return 0; }  # Default: can sudo
    resources::is_service_active() { return 0; }  # Default: service active
    resources::start_service() { return 0; }  # Default: start succeeds
    resources::stop_service() { return 0; }  # Default: stop succeeds
    resources::wait_for_service() { return 0; }  # Default: wait succeeds
    resources::handle_error() { return 1; }
    resources::print_status() { echo "Status for $1"; }
    
    
    
    # Mock system commands
    systemctl() {
        case "$1" in
            "list-unit-files") echo "ollama.service" ;;
        esac
        return 0
    }
    
    ollama() {
        case "$1" in
            "list") 
                echo "NAME                ID              SIZE      MODIFIED"
                echo "llama3.1:8b         abc123          4.9 GB    1 hour ago"
                echo "deepseek-r1:8b      def456          4.7 GB    2 hours ago"
                ;;
        esac
    }
    
    sleep() { return 0; }  # Mock sleep for faster tests
    wc() { /usr/bin/wc "$@"; }
    tail() { /usr/bin/tail "$@"; }
    
    # Export config functions
    ollama::export_config
    ollama::export_messages
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

@test "ollama::is_installed returns 0 when binary exists" {
    resources::binary_exists() { return 0; }
    
    ollama::is_installed
    [ "$?" -eq 0 ]
}

@test "ollama::is_installed returns 1 when binary doesn't exist" {
    resources::binary_exists() { return 1; }
    
    ollama::is_installed
    [ "$?" -eq 1 ]
}

@test "ollama::is_running returns 0 when service is running" {
    resources::is_service_running() { return 0; }
    
    ollama::is_running
    [ "$?" -eq 0 ]
}

@test "ollama::is_running returns 1 when service is not running" {
    resources::is_service_running() { return 1; }
    
    ollama::is_running
    [ "$?" -eq 1 ]
}

@test "ollama::is_healthy returns 0 when API is responsive" {
    resources::check_http_health() { return 0; }
    
    ollama::is_healthy
    [ "$?" -eq 0 ]
}

@test "ollama::is_healthy returns 1 when API is not responsive" {
    resources::check_http_health() { return 1; }
    
    ollama::is_healthy
    [ "$?" -eq 1 ]
}

@test "ollama::start succeeds when service starts normally" {
    # Mock service not currently running
    ollama::is_running() { return 1; }
    
    run ollama::start
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting Ollama service" ]]
    [[ "$output" =~ "Ollama is running" ]]
}

@test "ollama::start skips when already running" {
    # Mock service already running
    ollama::is_running() { return 0; }
    
    run ollama::start
    [ "$status" -eq 0 ]
    [[ "$output" =~ "already running on port" ]]
}

@test "ollama::start fails when service not found" {
    ollama::is_running() { return 1; }
    systemctl() {
        case "$1" in
            "list-unit-files") return 1 ;;  # Service not found
        esac
    }
    
    run ollama::start
    [ "$status" -eq 1 ]
}

@test "ollama::start fails without sudo privileges" {
    ollama::is_running() { return 1; }
    resources::can_sudo() { return 1; }
    
    run ollama::start
    [ "$status" -eq 1 ]
}

@test "ollama::start fails when port is in use by another process" {
    ollama::is_running() { return 1; }
    resources::is_service_running() { return 0; }  # Port in use
    resources::is_service_active() { return 1; }   # But not by our service
    
    run ollama::start
    [ "$status" -eq 1 ]
}

@test "ollama::start fails when service fails to start" {
    ollama::is_running() { return 1; }
    resources::start_service() { return 1; }  # Service start fails
    
    run ollama::start
    [ "$status" -eq 1 ]
}

@test "ollama::start warns when API doesn't respond after start" {
    ollama::is_running() { return 1; }
    resources::check_http_health() { return 1; }  # API not healthy
    
    run ollama::start
    [ "$status" -eq 0 ]  # Still succeeds but with warning
    [[ "$output" =~ "Ollama started but API not responding" ]]
}

@test "ollama::start fails when service doesn't come up in time" {
    ollama::is_running() { return 1; }
    resources::wait_for_service() { return 1; }  # Service doesn't start in time
    
    run ollama::start
    [ "$status" -eq 1 ]
}

@test "ollama::stop calls resources::stop_service" {
    resources::stop_service() {
        echo "Stopping $1"
        return 0
    }
    
    run ollama::stop
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Stopping $OLLAMA_SERVICE_NAME" ]]
}

@test "ollama::restart stops then starts service" {
    local call_sequence=""
    
    ollama::stop() {
        call_sequence+="stop "
        return 0
    }
    
    ollama::start() {
        call_sequence+="start"
        return 0
    }
    
    run ollama::restart
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Restarting Ollama service" ]]
    [[ "$call_sequence" == "stop start" ]]
}

@test "ollama::status displays comprehensive status" {
    run ollama::status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status for ollama" ]]
    [[ "$output" =~ "API Endpoints:" ]]
    [[ "$output" =~ "Base URL: $OLLAMA_BASE_URL" ]]
    [[ "$output" =~ "Models: $OLLAMA_BASE_URL/api/tags" ]]
    [[ "$output" =~ "Chat: $OLLAMA_BASE_URL/api/chat" ]]
    [[ "$output" =~ "Generate: $OLLAMA_BASE_URL/api/generate" ]]
    [[ "$output" =~ "Installed models:" ]]
}

@test "ollama::status skips API details when not healthy" {
    resources::check_http_health() { return 1; }  # API not healthy
    
    run ollama::status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status for ollama" ]]
    [[ ! "$output" =~ "API Endpoints:" ]]  # Should not show API details
}

@test "ollama::status handles missing ollama command gracefully" {
    system::is_command() { return 1; }  # ollama command not found
    
    run ollama::status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status for ollama" ]]
    # Should still work, just without model count
}

@test "all status functions are defined" {
    # Test that all expected functions exist
    type ollama::is_installed >/dev/null
    type ollama::is_running >/dev/null
    type ollama::is_healthy >/dev/null
    type ollama::start >/dev/null
    type ollama::stop >/dev/null
    type ollama::restart >/dev/null
    type ollama::status >/dev/null
}

@test "status functions use correct parameters" {
    # Test that functions call dependencies with expected parameters
    local binary_check=""
    local port_check=""
    local health_url=""
    local health_endpoint=""
    
    resources::binary_exists() {
        binary_check="$1"
        return 0
    }
    
    resources::is_service_running() {
        port_check="$1"
        return 0
    }
    
    resources::check_http_health() {
        health_url="$1"
        health_endpoint="$2"
        return 0
    }
    
    ollama::is_installed
    [ "$binary_check" = "ollama" ]
    
    ollama::is_running
    [ "$port_check" = "$OLLAMA_PORT" ]
    
    ollama::is_healthy
    [ "$health_url" = "$OLLAMA_BASE_URL" ]
    [ "$health_endpoint" = "/api/tags" ]
}