#!/usr/bin/env bats
# Tests for Unstructured.io common.sh functions

# Get script directory first
COMMON_BATS_DIR="${BATS_TEST_DIRNAME}"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${COMMON_BATS_DIR}/../../../../lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "unstructured-io"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    UNSTRUCTURED_IO_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Source library files
    source "${SCRIPT_DIR}/common.sh"
    source "${UNSTRUCTURED_IO_DIR}/config/defaults.sh"
    source "${UNSTRUCTURED_IO_DIR}/config/messages.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_UNSTRUCTURED_IO_DIR="$UNSTRUCTURED_IO_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export UNSTRUCTURED_IO_CUSTOM_PORT="9999"
    export UNSTRUCTURED_IO_CONTAINER_NAME="unstructured-io-test"
    export UNSTRUCTURED_IO_BASE_URL="http://localhost:9999"
    export UNSTRUCTURED_IO_DEFAULT_STRATEGY="hi_res"
    export UNSTRUCTURED_IO_DEFAULT_LANGUAGES="eng"
    export YES="no"
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    UNSTRUCTURED_IO_DIR="${SETUP_FILE_UNSTRUCTURED_IO_DIR}"
    
    # Mock system functions for testing
    
    system::is_port_in_use() {
        # Mock port as available for testing
        return 1
    }
    
    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "unstructured-io") echo "8000" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Mock Docker functions
    
    # Mock log functions
    
    
    
    # Mock args functions
    args::reset() { return 0; }
    args::register_help() { return 0; }
    args::register_yes() { return 0; }
    args::register() { return 0; }
    args::is_asking_for_help() { return 1; }
    args::parse() { return 0; }
    args::get() {
        # Default mock implementation - individual tests can override
        case "$1" in
            "action") echo "install" ;;
            "force") echo "no" ;;
            "yes") echo "no" ;;
            *) echo "" ;;
        esac
    }
    args::usage() {
        echo "USAGE_CALLED"
        return 0
    }
    
    # Export config functions
    unstructured_io::export_config
    unstructured_io::export_messages
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test configuration export
@test "unstructured_io::export_config sets configuration variables" {
    unstructured_io::export_config
    
    [ -n "$UNSTRUCTURED_IO_PORT" ]
    [ -n "$UNSTRUCTURED_IO_BASE_URL" ]
    [ -n "$UNSTRUCTURED_IO_CONTAINER_NAME" ]
    [ -n "$UNSTRUCTURED_IO_IMAGE" ]
}

# Test message export
@test "unstructured_io::export_messages sets message variables" {
    unstructured_io::export_messages
    
    [ -n "$MSG_UNSTRUCTURED_IO_INSTALLING" ]
    [ -n "$MSG_UNSTRUCTURED_IO_ALREADY_INSTALLED" ]
    [ -n "$MSG_UNSTRUCTURED_IO_NOT_FOUND" ]
}

# Test argument parsing with defaults
@test "unstructured_io::parse_arguments sets correct defaults" {
    # Mock args::get to return defaults
    args::get() {
        case "$1" in
            "action") echo "install" ;;
            "force") echo "no" ;;
            "file") echo "" ;;
            "strategy") echo "$UNSTRUCTURED_IO_DEFAULT_STRATEGY" ;;
            "output") echo "json" ;;
            "languages") echo "$UNSTRUCTURED_IO_DEFAULT_LANGUAGES" ;;
            "batch") echo "no" ;;
            "follow") echo "no" ;;
            "yes") echo "no" ;;
        esac
    }
    
    unstructured_io::parse_arguments --action install
    
    [ "$ACTION" = "install" ]
    [ "$FORCE" = "no" ]
    [ "$STRATEGY" = "$UNSTRUCTURED_IO_DEFAULT_STRATEGY" ]
    [ "$OUTPUT" = "json" ]
    [ "$LANGUAGES" = "$UNSTRUCTURED_IO_DEFAULT_LANGUAGES" ]
    [ "$BATCH" = "no" ]
}

# Test usage display
@test "unstructured_io::usage displays help information" {
    result=$(unstructured_io::usage)
    
    [[ "$result" =~ "USAGE_CALLED" ]]
}

# Test Docker check function
@test "unstructured_io::check_docker succeeds with Docker available" {
    unstructured_io::check_docker
    [ "$?" -eq 0 ]
}

# Test Docker check function with Docker unavailable
@test "unstructured_io::check_docker fails with Docker unavailable" {
    # Override system function to simulate missing Docker
    system::is_command() {
        case "$1" in
            "docker") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    run unstructured_io::check_docker
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}

# Test container existence check - exists
@test "unstructured_io::container_exists returns true for existing container" {
    unstructured_io::container_exists
    [ "$?" -eq 0 ]
}

# Test container existence check - not exists
@test "unstructured_io::container_exists returns false for non-existing container" {
    # Override docker to return empty result
    docker() {
        case "$1" in
            "ps") echo "" ;;
            *) return 0 ;;
        esac
    }
    
    run unstructured_io::container_exists
    [ "$status" -eq 1 ]
}

# Test container running check - running
@test "unstructured_io::container_running returns true for running container" {
    unstructured_io::container_running
    [ "$?" -eq 0 ]
}

# Test container running check - not running
@test "unstructured_io::container_running returns false for stopped container" {
    # Override docker inspect to show stopped state
    docker() {
        case "$1" in
            "inspect")
                echo '{"State":{"Running":false}}'
                ;;
            *) return 0 ;;
        esac
    }
    
    run unstructured_io::container_running
    [ "$status" -eq 1 ]
}

# Test log display function
@test "unstructured_io::logs displays container logs" {
    result=$(unstructured_io::logs "no")
    
    [[ "$result" =~ "Mock container logs" ]]
}

# Test log display with follow option
@test "unstructured_io::logs handles follow option" {
    # Mock docker logs with follow
    docker() {
        case "$1" in
            "logs")
                if [[ "$*" =~ "-f" ]]; then
                    echo "Following logs..."
                else
                    echo "Static logs"
                fi
                ;;
            *) return 0 ;;
        esac
    }
    
    result=$(unstructured_io::logs "yes")
    [[ "$result" =~ "Following logs" ]]
    
    result=$(unstructured_io::logs "no")
    [[ "$result" =~ "Static logs" ]]
}

# Test info display function
@test "unstructured_io::info displays service information" {
    result=$(unstructured_io::info)
    
    [[ "$result" =~ "Unstructured.io" ]]
    [[ "$result" =~ "$UNSTRUCTURED_IO_BASE_URL" ]]
}

# Test port check function
@test "unstructured_io::check_port_available succeeds for available port" {
    unstructured_io::check_port_available
    [ "$?" -eq 0 ]
}

# Test port check function with port in use
@test "unstructured_io::check_port_available fails for port in use" {
    # Override system function to simulate port in use
    system::is_port_in_use() {
        return 0  # Port is in use
    }
    
    run unstructured_io::check_port_available
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}

# Test parameter validation - valid strategy
@test "unstructured_io::validate_strategy accepts valid strategies" {
    unstructured_io::validate_strategy "hi_res"
    [ "$?" -eq 0 ]
    
    unstructured_io::validate_strategy "fast"
    [ "$?" -eq 0 ]
    
    unstructured_io::validate_strategy "auto"
    [ "$?" -eq 0 ]
}

# Test parameter validation - invalid strategy
@test "unstructured_io::validate_strategy rejects invalid strategy" {
    run unstructured_io::validate_strategy "invalid"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}

# Test parameter validation - valid output format
@test "unstructured_io::validate_output_format accepts valid formats" {
    unstructured_io::validate_output_format "json"
    [ "$?" -eq 0 ]
    
    unstructured_io::validate_output_format "markdown"
    [ "$?" -eq 0 ]
    
    unstructured_io::validate_output_format "text"
    [ "$?" -eq 0 ]
    
    unstructured_io::validate_output_format "elements"
    [ "$?" -eq 0 ]
}

# Test parameter validation - invalid output format
@test "unstructured_io::validate_output_format rejects invalid format" {
    run unstructured_io::validate_output_format "invalid"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}