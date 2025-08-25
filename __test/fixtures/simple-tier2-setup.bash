#!/usr/bin/env bash
# Simple Tier 2 Mock Setup for BATS tests
# Minimal setup without the full Vrooli test infrastructure

# Determine root directory
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../.." && builtin pwd)}"
MOCK_BASE_DIR="${APP_ROOT}/__test/mocks"
MOCK_TIER2_DIR="${MOCK_BASE_DIR}/tier2"
export APP_ROOT MOCK_BASE_DIR MOCK_TIER2_DIR

# Load the test helper
source "${MOCK_BASE_DIR}/test_helper.sh"

# Load a mock with proper state initialization
load_mock_with_state() {
    local mock_name="$1"
    local mock_file="${MOCK_TIER2_DIR}/${mock_name}.sh"
    
    if [[ -f "$mock_file" ]]; then
        # Source the mock file
        source "$mock_file"
        
        # Initialize mock state if reset function exists
        if declare -F "${mock_name}_mock_reset" >/dev/null 2>&1; then
            "${mock_name}_mock_reset"
        fi
        
        echo "[DEBUG] Loaded mock: $mock_name" >&2
        return 0
    else
        echo "[ERROR] Mock file not found: $mock_file" >&2
        return 1
    fi
}

# Setup function for BATS tests - loads mocks and initializes state
vrooli_setup_unit_test() {
    echo "[SETUP] Simple Tier 2 setup for BATS tests" >&2
    
    # Set environment for proper mock loading
    export MOCK_MODE="tier2"
    export MOCK_ADAPTER_MODE="tier2"
    
    # Load commonly used mocks with state initialization
    # These will be available in all tests
    if [[ "${LOAD_DEFAULT_MOCKS:-true}" == "true" ]]; then
        load_mock_with_state "redis" || true
        load_mock_with_state "postgres" || true
        load_mock_with_state "docker" || true
    fi
    
    return 0
}

# Cleanup function - resets mock states
vrooli_cleanup_test() {
    echo "[CLEANUP] Simple cleanup for BATS tests" >&2
    
    # Reset all loaded mocks
    for mock in redis postgres docker; do
        if declare -F "${mock}_mock_reset" >/dev/null 2>&1; then
            "${mock}_mock_reset"
        fi
    done
    
    return 0
}

# Export functions and variables
export -f vrooli_setup_unit_test
export -f vrooli_cleanup_test
export -f load_mock_with_state

# Export mock-related variables for BATS subshells
export REDIS_STATE REDIS_LISTS REDIS_EXPIRES REDIS_PUBSUB
export POSTGRES_STATE POSTGRES_TABLES POSTGRES_SEQUENCES
export DOCKER_CONTAINERS DOCKER_IMAGES DOCKER_NETWORKS

echo "[SETUP] Simple Tier 2 setup loaded successfully" >&2