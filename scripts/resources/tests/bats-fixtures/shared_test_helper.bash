#!/usr/bin/env bash
# Shared Test Helper Pattern for Resource Tests
# Use this pattern to optimize test performance by minimizing setup/teardown operations

# ===========================================
# PERFORMANCE-OPTIMIZED TEST PATTERN
# ===========================================
# Use this pattern in your .bats files:
#
# load path/to/shared_test_helper
#
# # Heavy operations run once per file
# setup_file() {
#     setup_resource_test_environment "$RESOURCE_NAME"
# }
#
# # Light operations run per test
# setup() {
#     reset_mocks
# }
#
# # Cleanup once per file
# teardown_file() {
#     cleanup_resource_test_environment
# }

# Global test environment variables
export SHARED_TEST_CACHE_DIR="/tmp/vrooli-bats-cache-$$"
export SHARED_TEST_ACTIVE=false

#######################################
# Setup a resource test environment efficiently
# Arguments:
#   $1 - resource name (e.g., "node-red", "ollama", "whisper")
#######################################
setup_resource_test_environment() {
    local resource_name="$1"
    
    if [[ -z "$resource_name" ]]; then
        echo "ERROR: Resource name required for setup_resource_test_environment" >&2
        return 1
    fi
    
    # Create shared cache directory
    mkdir -p "$SHARED_TEST_CACHE_DIR"
    
    # Set resource-specific test directory
    export RESOURCE_TEST_DIR="$SHARED_TEST_CACHE_DIR/$resource_name"
    mkdir -p "$RESOURCE_TEST_DIR"
    
    # Create minimal directory structure (much faster than copying files)
    create_minimal_resource_structure "$resource_name"
    
    # Create shared mock environment (reusable across tests)
    setup_shared_mocks
    
    export SHARED_TEST_ACTIVE=true
    echo "[SETUP] Initialized test environment for $resource_name in $RESOURCE_TEST_DIR"
}

#######################################
# Create minimal directory structure without file copying
#######################################
create_minimal_resource_structure() {
    local resource_name="$1"
    
    # Create standard directories
    mkdir -p "$RESOURCE_TEST_DIR"/{config,lib,docker,data}
    
    # Create in-memory configuration instead of copying files
    create_inmemory_config "$resource_name"
    
    # Create mock common functions
    create_mock_common_functions
}

#######################################
# Create in-memory configuration (faster than file operations)
#######################################
create_inmemory_config() {
    local resource_name="$1"
    
    # Set common environment variables instead of sourcing config files
    export RESOURCE_NAME="$resource_name"
    export RESOURCE_PORT="${RESOURCE_PORT:-8080}"
    export CONTAINER_NAME="${resource_name}-test"
    export RESOURCE_DATA_DIR="$RESOURCE_TEST_DIR/data"
    
    # Resource-specific configurations
    case "$resource_name" in
        "node-red")
            export RESOURCE_PORT="1880"
            export NODE_RED_DATA_DIR="$RESOURCE_DATA_DIR"
            ;;
        "ollama")
            export RESOURCE_PORT="11434"
            export OLLAMA_DATA_DIR="$RESOURCE_DATA_DIR"
            ;;
        "whisper")
            export RESOURCE_PORT="9000"
            export WHISPER_DATA_DIR="$RESOURCE_DATA_DIR"
            ;;
    esac
}

#######################################
# Setup shared mock functions (created once, reused)
#######################################
setup_shared_mocks() {
    # Create mock functions in memory instead of files
    
    # Mock logging functions
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::debug() { echo "[DEBUG] $*"; }
    log::header() { echo "=== $* ==="; }
    
    # Mock system functions
    system::is_command() { command -v "$1" >/dev/null 2>&1; }
    
    # Mock flow control functions
    flow::confirm() { return 0; }  # Always confirm in tests
    flow::is_yes() { return 0; }
    
    # Export all functions
    export -f log::info log::success log::warning log::error log::debug log::header
    export -f system::is_command flow::confirm flow::is_yes
}

#######################################
# Reset mocks between tests (lightweight operation)
#######################################
reset_mocks() {
    # Reset mock states without recreating everything
    export DOCKER_MOCK_MODE="success"
    export CURL_MOCK_MODE="success"
    export JQ_MOCK_MODE="success"
    
    # Reset any test-specific environment variables
    unset TEST_OUTPUT TEST_ERROR TEST_STATUS
}

#######################################
# Mock Docker with different behaviors
#######################################
mock_docker() {
    local mode="${1:-success}"
    export DOCKER_MOCK_MODE="$mode"
    
    docker() {
        case "$DOCKER_MOCK_MODE" in
            "success")
                case "$1" in
                    "ps") echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES" ;;
                    "container") 
                        case "$2" in
                            "inspect") echo "true" ;;
                            *) return 0 ;;
                        esac ;;
                    "stats") echo "CONTAINER     CPU %     MEM USAGE / LIMIT     MEM %     NET I/O     BLOCK I/O" ;;
                    "logs") echo "Mock log output" ;;
                    *) return 0 ;;
                esac ;;
            "not_installed")
                case "$1" in
                    "ps"|"container") return 1 ;;
                    *) return 0 ;;
                esac ;;
            "not_running")
                case "$1" in
                    "ps") echo "" ;;  # No running containers
                    "container") 
                        case "$2" in
                            "inspect") echo "false" ;;  # Not running
                            *) return 0 ;;
                        esac ;;
                    *) return 1 ;;
                esac ;;
            "failure") return 1 ;;
            *) return 0 ;;
        esac
    }
    export -f docker
}

#######################################
# Mock curl with different behaviors
#######################################
mock_curl() {
    local mode="${1:-success}"
    export CURL_MOCK_MODE="$mode"
    
    curl() {
        case "$CURL_MOCK_MODE" in
            "success") echo '{"status": "ok", "version": "1.0.0"}' ;;
            "failure") return 1 ;;
            *) echo '{"status": "unknown"}' ;;
        esac
    }
    export -f curl
}

#######################################
# Mock jq
#######################################
mock_jq() {
    local mode="${1:-success}"
    export JQ_MOCK_MODE="$mode"
    
    jq() {
        case "$JQ_MOCK_MODE" in
            "success") 
                # Simple jq simulation for basic queries
                case "$1" in
                    "-r") shift; echo "mock-value" ;;
                    ".status") echo "ok" ;;
                    ".version") echo "1.0.0" ;;
                    *) echo "null" ;;
                esac ;;
            "failure") return 1 ;;
            *) echo "null" ;;
        esac
    }
    export -f jq
}

#######################################
# Cleanup test environment
#######################################
cleanup_resource_test_environment() {
    if [[ "$SHARED_TEST_ACTIVE" == "true" ]]; then
        rm -rf "$SHARED_TEST_CACHE_DIR"
        echo "[CLEANUP] Removed test environment"
        export SHARED_TEST_ACTIVE=false
    fi
}

#######################################
# Assertion helpers for consistent testing
#######################################
assert_success() {
    if [[ $status -ne 0 ]]; then
        echo "Expected success but got exit code: $status"
        echo "Output: $output"
        return 1
    fi
}

assert_failure() {
    if [[ $status -eq 0 ]]; then
        echo "Expected failure but got success"
        echo "Output: $output"
        return 1
    fi
}

assert_output_contains() {
    local expected="$1"
    if [[ "$output" != *"$expected"* ]]; then
        echo "Expected output to contain: $expected"
        echo "Actual output: $output"
        return 1
    fi
}

assert_output_matches() {
    local pattern="$1"
    if [[ ! "$output" =~ $pattern ]]; then
        echo "Expected output to match pattern: $pattern"
        echo "Actual output: $output"
        return 1
    fi
}

assert_file_exists() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        echo "Expected file to exist: $file"
        return 1
    fi
}

# Export assertion functions
export -f assert_success assert_failure assert_output_contains assert_output_matches assert_file_exists

echo "[SHARED_TEST_HELPER] Loaded performance-optimized test helpers"