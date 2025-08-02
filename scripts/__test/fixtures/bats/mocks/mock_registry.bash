#!/usr/bin/env bash
# Unified Mock Registry System
# Central management for all test mocks with lazy loading and performance optimization

# Prevent duplicate loading
if [[ "${MOCK_REGISTRY_LOADED:-}" == "true" ]]; then
    return 0
fi
export MOCK_REGISTRY_LOADED="true"

# Mock registry state
declare -A LOADED_MOCKS
declare -A MOCK_STATES
declare -A MOCK_CONFIGS

# Global configuration
export VROOLI_TEST_FIXTURES_DIR="${VROOLI_TEST_FIXTURES_DIR:-/home/matthalloran8/Vrooli/scripts/__test/fixtures/bats}"

#######################################
# Load a specific mock module
# Globals: LOADED_MOCKS, VROOLI_TEST_FIXTURES_DIR
# Arguments:
#   $1 - category (system, resource)
#   $2 - name (docker, ollama, http)
# Returns: 0 on success, 1 on failure
#######################################
mock::load() {
    local category="$1"
    local name="$2"
    local mock_key="${category}_${name}"
    
    # Check if already loaded
    if [[ -n "${LOADED_MOCKS[$mock_key]}" ]]; then
        return 0
    fi
    
    # Determine file path
    local mock_file
    case "$category" in
        "system")
            mock_file="${VROOLI_TEST_FIXTURES_DIR}/mocks/system/${name}.bash"
            ;;
        "resource")
            # Auto-detect resource category
            local resource_category
            resource_category=$(mock::detect_resource_category "$name")
            mock_file="${VROOLI_TEST_FIXTURES_DIR}/mocks/resources/${resource_category}/${name}.bash"
            ;;
        *)
            echo "ERROR: Unknown mock category: $category" >&2
            return 1
            ;;
    esac
    
    # Load the mock file if it exists
    if [[ -f "$mock_file" ]]; then
        source "$mock_file"
        LOADED_MOCKS["$mock_key"]="true"
        echo "[MOCK_REGISTRY] Loaded $category:$name from $mock_file"
        return 0
    else
        # Fallback to legacy mock system
        mock::load_legacy_mock "$category" "$name"
    fi
}

#######################################
# Detect which category a resource belongs to
# Arguments: $1 - resource name
# Returns: category name
#######################################
mock::detect_resource_category() {
    local resource="$1"
    
    case "$resource" in
        "ollama"|"whisper"|"comfyui"|"unstructured-io")
            echo "ai"
            ;;
        "n8n"|"node-red"|"huginn"|"windmill")
            echo "automation"
            ;;
        "agent-s2"|"browserless"|"claude-code")
            echo "agents"
            ;;
        "minio"|"qdrant"|"questdb"|"vault"|"postgres"|"redis")
            echo "storage"
            ;;
        "searxng")
            echo "search"
            ;;
        "judge0")
            echo "execution"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

#######################################
# Load legacy mock (fallback for existing mocks)
# Arguments: $1 - category, $2 - name
#######################################
mock::load_legacy_mock() {
    local category="$1"
    local name="$2"
    
    # Try to load from old location
    local legacy_file
    if [[ "$category" == "system" ]]; then
        legacy_file="${VROOLI_TEST_FIXTURES_DIR}/system_mocks.bash"
    else
        legacy_file="${VROOLI_TEST_FIXTURES_DIR}/resource_mocks.bash"
    fi
    
    if [[ -f "$legacy_file" && -z "${LOADED_MOCKS[legacy_$legacy_file]}" ]]; then
        source "$legacy_file"
        LOADED_MOCKS["legacy_$legacy_file"]="true"
        LOADED_MOCKS["${category}_${name}"]="true"
        echo "[MOCK_REGISTRY] Loaded legacy mock for $category:$name"
        return 0
    fi
    
    echo "[MOCK_REGISTRY] WARNING: Mock not found for $category:$name" >&2
    return 1
}

#######################################
# Setup minimal test environment with basic mocks
# Fast loading for simple tests
#######################################
mock::setup_minimal() {
    echo "[MOCK_REGISTRY] Setting up minimal test environment"
    
    # Load essential system mocks
    mock::load system docker
    mock::load system http
    mock::load system commands
    
    # Set up basic environment
    export FORCE="${FORCE:-no}"
    export YES="${YES:-no}"
    export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
    export QUIET="${QUIET:-no}"
    
    # Create minimal test directories
    export TEST_TMPDIR="${TEST_TMPDIR:-/tmp/vrooli-test-$$}"
    mkdir -p "$TEST_TMPDIR"
}

#######################################
# Setup test environment for a specific resource
# Arguments: $1 - resource name
#######################################
mock::setup_resource() {
    local resource="$1"
    
    echo "[MOCK_REGISTRY] Setting up test environment for resource: $resource"
    
    # Load minimal setup first
    mock::setup_minimal
    
    # Load resource-specific mocks
    mock::load resource "$resource"
    
    # Set resource-specific environment
    mock::configure_resource_environment "$resource"
}

#######################################
# Configure environment variables for a specific resource
# Arguments: $1 - resource name
#######################################
mock::configure_resource_environment() {
    local resource="$1"
    local port_base=$((8000 + (RANDOM % 1000)))
    
    # Set common environment variables
    export RESOURCE_NAME="$resource"
    export TEST_NAMESPACE="test_$$_${RANDOM}"
    export CONTAINER_NAME_PREFIX="${TEST_NAMESPACE}_"
    
    # Set resource-specific variables
    case "$resource" in
        "ollama")
            export OLLAMA_PORT="${OLLAMA_PORT:-11434}"
            export OLLAMA_BASE_URL="http://localhost:${OLLAMA_PORT}"
            export OLLAMA_CONTAINER_NAME="${CONTAINER_NAME_PREFIX}ollama"
            ;;
        "whisper")
            export WHISPER_PORT="${WHISPER_PORT:-8090}"
            export WHISPER_BASE_URL="http://localhost:${WHISPER_PORT}"
            export WHISPER_CONTAINER_NAME="${CONTAINER_NAME_PREFIX}whisper"
            ;;
        "n8n")
            export N8N_PORT="${N8N_PORT:-5678}"
            export N8N_BASE_URL="http://localhost:${N8N_PORT}"
            export N8N_CONTAINER_NAME="${CONTAINER_NAME_PREFIX}n8n"
            ;;
        "qdrant")
            export QDRANT_PORT="${QDRANT_PORT:-6333}"
            export QDRANT_BASE_URL="http://localhost:${QDRANT_PORT}"
            export QDRANT_CONTAINER_NAME="${CONTAINER_NAME_PREFIX}qdrant"
            ;;
        *)
            # Generic configuration
            export RESOURCE_PORT="${RESOURCE_PORT:-$port_base}"
            export RESOURCE_BASE_URL="http://localhost:${RESOURCE_PORT}"
            export RESOURCE_CONTAINER_NAME="${CONTAINER_NAME_PREFIX}${resource}"
            ;;
    esac
}

#######################################
# Setup integration test environment for multiple resources
# Arguments: $@ - list of resource names
#######################################
mock::setup_integration() {
    local resources=("$@")
    
    echo "[MOCK_REGISTRY] Setting up integration test environment for: ${resources[*]}"
    
    # Load minimal setup
    mock::setup_minimal
    
    # Load all resource mocks
    for resource in "${resources[@]}"; do
        mock::load resource "$resource"
        mock::configure_resource_environment "$resource"
    done
}

#######################################
# Cleanup test environment
#######################################
mock::cleanup() {
    echo "[MOCK_REGISTRY] Cleaning up test environment"
    
    # Clean up temporary directories
    if [[ -n "${TEST_TMPDIR:-}" && -d "$TEST_TMPDIR" ]]; then
        rm -rf "$TEST_TMPDIR"
    fi
    
    # Reset mock state
    unset LOADED_MOCKS
    unset MOCK_STATES
    unset MOCK_CONFIGS
    
    # Kill any background processes
    jobs -p | xargs -r kill 2>/dev/null || true
}

#######################################
# List loaded mocks (for debugging)
#######################################
mock::list_loaded() {
    echo "[MOCK_REGISTRY] Currently loaded mocks:"
    for mock_key in "${!LOADED_MOCKS[@]}"; do
        echo "  - $mock_key"
    done
}

#######################################
# Check if a mock is loaded
# Arguments: $1 - category, $2 - name
# Returns: 0 if loaded, 1 if not
#######################################
mock::is_loaded() {
    local category="$1"
    local name="$2"
    local mock_key="${category}_${name}"
    
    [[ -n "${LOADED_MOCKS[$mock_key]}" ]]
}

# Export all functions
export -f mock::load mock::detect_resource_category mock::load_legacy_mock
export -f mock::setup_minimal mock::setup_resource mock::setup_integration
export -f mock::configure_resource_environment mock::cleanup
export -f mock::list_loaded mock::is_loaded

echo "[MOCK_REGISTRY] Mock registry system loaded"