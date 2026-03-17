#!/usr/bin/env bats
# Tests for Kokoro status.sh functions

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "kokoro"

    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    KOKORO_DIR="$(dirname "$SCRIPT_DIR")"

    # Source library files
    source "${KOKORO_DIR}/config/defaults.sh"
    source "${KOKORO_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/status.sh"

    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_KOKORO_DIR="$KOKORO_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup

    # Set test environment
    export KOKORO_CUSTOM_PORT="8880"
    export KOKORO_CONTAINER_NAME="kokoro-test"
    export KOKORO_BASE_URL="http://localhost:8880"
    export KOKORO_GPU_ENABLED="no"
    export YES="no"

    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    KOKORO_DIR="${SETUP_FILE_KOKORO_DIR}"

    # Mock system functions

    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "kokoro") echo "8880" ;;
            *) echo "8080" ;;
        esac
    }

    # Mock jq for JSON parsing
    jq() {
        case "$*" in
            *".status"*)
                echo "healthy"
                ;;
            *".voices"*)
                echo '["af_heart","af_bella","am_adam"]'
                ;;
            *"length"*)
                echo "3"
                ;;
            *) echo "{}" ;;
        esac
    }

    # Mock common functions
    common::check_docker() { return 0; }
    common::container_exists() { return 0; }
    common::is_running() { return 0; }

    # Load config and messages from config files
    if [[ -f "${KOKORO_DIR}/config/defaults.sh" ]]; then
        source "${KOKORO_DIR}/config/defaults.sh"
        defaults::export_config 2>/dev/null || true
    fi
    if [[ -f "${KOKORO_DIR}/config/messages.sh" ]]; then
        source "${KOKORO_DIR}/config/messages.sh"
        messages::export_messages 2>/dev/null || true
    fi
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test comprehensive status display
@test "kokoro::show_status displays comprehensive status information" {
    result=$(kokoro::status::show)

    [[ "$result" =~ "Kokoro Status" ]]
    [[ "$result" =~ "Container" ]] || [[ "$result" =~ "container" ]]
}

# Test status display with Docker unavailable
@test "kokoro::show_status handles Docker unavailable" {
    common::check_docker() {
        return 1
    }

    result=$(kokoro::status::show)

    [[ "$result" =~ "error" ]] || [[ "$result" =~ "Error" ]] || [[ "$result" =~ "not" ]]
}

# Test status display with container not running
@test "kokoro::show_status handles stopped container" {
    common::is_running() {
        return 1
    }

    result=$(kokoro::status::show)

    [[ "$result" =~ "not running" ]] || [[ "$result" =~ "Stopped" ]] || [[ "$result" =~ "Running: No" ]]
}

# Test status display with missing container
@test "kokoro::show_status handles missing container" {
    common::container_exists() {
        return 1
    }

    result=$(kokoro::status::show)

    [[ "$result" =~ "not found" ]] || [[ "$result" =~ "Not installed" ]] || [[ "$result" =~ "Installed: No" ]]
}

# Test quick status check
@test "kokoro::quick_status provides brief status" {
    result=$(status::quick_status)

    [[ "$result" =~ "running" ]] || [[ "$result" =~ "healthy" ]] || [[ "$result" =~ "stopped" ]]
}
