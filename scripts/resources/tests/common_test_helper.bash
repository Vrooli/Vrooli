#!/usr/bin/env bats
# Common test helper for all resource tests
# This file provides shared setup, teardown, and utility functions

# Determine paths relative to the test file that sources this helper
_determine_paths() {
    # The test file that sourced us
    local test_file="${BATS_TEST_FILENAME}"
    local test_dir="$(dirname "$test_file")"
    
    # Navigate up to find the resource directory (e.g., scripts/resources/ai/ollama)
    RESOURCE_DIR="$test_dir"
    while [[ ! -f "$RESOURCE_DIR/manage.sh" ]] && [[ "$RESOURCE_DIR" != "/" ]]; do
        RESOURCE_DIR="$(dirname "$RESOURCE_DIR")"
    done
    
    # Set common paths
    SCRIPT_DIR="$RESOURCE_DIR"
    RESOURCES_DIR="$(dirname "$(dirname "$RESOURCE_DIR")")"
    HELPERS_DIR="$(dirname "$RESOURCES_DIR")/helpers"
    PROJECT_ROOT="$(dirname "$(dirname "$RESOURCES_DIR")")"
}

# Initialize paths
_determine_paths

# Source required dependencies
if [[ -f "$HELPERS_DIR/utils/args.sh" ]]; then
    source "$HELPERS_DIR/utils/args.sh"
fi

if [[ -f "$RESOURCES_DIR/common.sh" ]]; then
    source "$RESOURCES_DIR/common.sh"
fi

# Common setup function that tests can call
common_setup() {
    # Set default environment variables
    export FORCE="${FORCE:-no}"
    export YES="${YES:-no}"
    export ACTION="${ACTION:-status}"
    export REMOVE_DATA="${REMOVE_DATA:-no}"
    export LOG_LINES="${LOG_LINES:-50}"
    export FOLLOW="${FOLLOW:-no}"
    
    # Common mock functions for dependencies
    if ! type -t log::info &>/dev/null; then
        log::info() { echo "[INFO] $*"; }
    fi
    
    if ! type -t log::error &>/dev/null; then
        log::error() { echo "[ERROR] $*" >&2; }
    fi
    
    if ! type -t log::warning &>/dev/null; then
        log::warning() { echo "[WARNING] $*" >&2; }
    fi
    
    if ! type -t log::debug &>/dev/null; then
        log::debug() { [[ "${DEBUG:-}" == "true" ]] && echo "[DEBUG] $*" >&2 || true; }
    fi
    
    if ! type -t log::header &>/dev/null; then
        log::header() { echo "=== $* ==="; }
    fi
    
    if ! type -t log::success &>/dev/null; then
        log::success() { echo "[SUCCESS] $*"; }
    fi
    
    # Mock resources functions if not available
    if ! type -t resources::get_default_port &>/dev/null; then
        resources::get_default_port() {
            case "$1" in
                ollama) echo "11434" ;;
                whisper) echo "8090" ;;
                n8n) echo "5678" ;;
                minio) echo "9000" ;;
                node-red) echo "1880" ;;
                huginn) echo "4111" ;;
                windmill) echo "5681" ;;
                browserless) echo "4110" ;;
                agent-s2) echo "4113" ;;
                searxng) echo "8100" ;;
                qdrant) echo "6333" ;;
                questdb) echo "9000" ;;
                vault) echo "8200" ;;
                judge0) echo "2358" ;;
                unstructured-io) echo "11450" ;;
                comfyui) echo "5679" ;;
                *) echo "8080" ;;
            esac
        }
    fi
    
    # Source configuration files if they exist
    if [[ -f "$SCRIPT_DIR/config/defaults.sh" ]]; then
        source "$SCRIPT_DIR/config/defaults.sh" 2>/dev/null || true
    fi
    
    if [[ -f "$SCRIPT_DIR/config/messages.sh" ]]; then
        source "$SCRIPT_DIR/config/messages.sh" 2>/dev/null || true
    fi
    
    # Source library files if needed (but avoid circular dependencies)
    local lib_dir="$SCRIPT_DIR/lib"
    if [[ -d "$lib_dir" ]]; then
        # Source in specific order to handle dependencies
        for lib_file in common.sh api.sh status.sh install.sh; do
            if [[ -f "$lib_dir/$lib_file" ]] && [[ "$BATS_TEST_FILENAME" != *"$lib_file"* ]]; then
                source "$lib_dir/$lib_file" 2>/dev/null || true
            fi
        done
        
        # Source any remaining lib files
        for lib_file in "$lib_dir"/*.sh; do
            if [[ -f "$lib_file" ]] && [[ "$BATS_TEST_FILENAME" != *"$(basename "$lib_file")"* ]]; then
                # Skip if already sourced
                local base_name=$(basename "$lib_file")
                if [[ "$base_name" != "common.sh" ]] && [[ "$base_name" != "api.sh" ]] && 
                   [[ "$base_name" != "status.sh" ]] && [[ "$base_name" != "install.sh" ]]; then
                    source "$lib_file" 2>/dev/null || true
                fi
            fi
        done
    fi
}

# Common teardown function
common_teardown() {
    # Reset any modified environment variables
    unset FORCE YES ACTION REMOVE_DATA LOG_LINES FOLLOW
    
    # Clean up any temporary files created during tests
    if [[ -n "${BATS_TEST_TMPDIR:-}" ]] && [[ -d "$BATS_TEST_TMPDIR" ]]; then
        rm -rf "$BATS_TEST_TMPDIR"/*
    fi
}

# Helper to check if a service is running (for integration tests)
is_service_running() {
    local service="$1"
    local port="${2:-$(resources::get_default_port "$service")}"
    
    # First check if port is in use
    if command -v lsof &>/dev/null; then
        lsof -i ":$port" &>/dev/null
    elif command -v netstat &>/dev/null; then
        netstat -tuln 2>/dev/null | grep -q ":$port "
    elif command -v ss &>/dev/null; then
        ss -tuln 2>/dev/null | grep -q ":$port "
    else
        # Fallback to curl check
        timeout 2 curl -s "http://localhost:$port" &>/dev/null
    fi
}

# Skip test if service is not running
skip_if_service_not_running() {
    local service="$1"
    local port="${2:-}"
    
    if ! is_service_running "$service" "$port"; then
        skip "$service service is not running on port ${port:-$(resources::get_default_port "$service")}"
    fi
}

# Helper to safely set readonly variables
safe_readonly() {
    local var_name="$1"
    local var_value="$2"
    
    # Check if variable is already readonly
    if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* $var_name="; then
        export "$var_name"="$var_value"
        readonly "$var_name"
    fi
}

# Helper to mock Docker commands for testing
mock_docker() {
    docker() {
        case "$1" in
            ps)
                if [[ "$*" == *"ollama"* ]]; then
                    echo "container_id  ollama:latest  Up 5 minutes"
                fi
                ;;
            images)
                echo "ollama:latest"
                ;;
            *)
                command docker "$@"
                ;;
        esac
    }
    export -f docker
}

# Helper to create temporary test files
create_temp_file() {
    local filename="${1:-test_file}"
    local content="${2:-test content}"
    local temp_file="$BATS_TEST_TMPDIR/$filename"
    
    echo "$content" > "$temp_file"
    echo "$temp_file"
}

# Export all functions for use in tests
export -f common_setup common_teardown is_service_running skip_if_service_not_running
export -f safe_readonly mock_docker create_temp_file