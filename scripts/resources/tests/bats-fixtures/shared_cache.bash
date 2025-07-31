#!/usr/bin/env bash
# Shared Cache for Test Performance
# Caches commonly sourced files to avoid repeated loading

# Global cache flag
SHARED_CACHE_INITIALIZED="${SHARED_CACHE_INITIALIZED:-false}"

# Initialize shared cache once per test session
init_shared_cache() {
    if [[ "$SHARED_CACHE_INITIALIZED" == "true" ]]; then
        return 0
    fi
    
    # Mark as initialized
    export SHARED_CACHE_INITIALIZED="true"
    
    # Cache frequently used utilities
    local base_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
    local helpers_dir="$base_dir/../helpers"
    
    # Pre-load common utilities if they exist
    if [[ -f "$helpers_dir/utils/log.sh" ]]; then
        source "$helpers_dir/utils/log.sh" 2>/dev/null || true
    fi
    
    if [[ -f "$helpers_dir/utils/system.sh" ]]; then
        source "$helpers_dir/utils/system.sh" 2>/dev/null || true
    fi
    
    if [[ -f "$helpers_dir/utils/flow.sh" ]]; then
        source "$helpers_dir/utils/flow.sh" 2>/dev/null || true
    fi
    
    if [[ -f "$base_dir/common.sh" ]]; then
        source "$base_dir/common.sh" 2>/dev/null || true
    fi
    
    # Cache standard mock definitions
    cache_standard_mocks
    
    echo "# Shared cache initialized for test session" >&2
    return 0
}

# Cache standard mock definitions
cache_standard_mocks() {
    # Mock common functions that are used across many tests
    if ! type -t log::info &>/dev/null; then
        log::info() { echo "[INFO] $*"; }
        export -f log::info
    fi
    
    if ! type -t log::error &>/dev/null; then
        log::error() { echo "[ERROR] $*" >&2; }
        export -f log::error
    fi
    
    if ! type -t log::warning &>/dev/null; then
        log::warning() { echo "[WARNING] $*" >&2; }
        export -f log::warning
    fi
    
    if ! type -t log::success &>/dev/null; then
        log::success() { echo "[SUCCESS] $*"; }
        export -f log::success
    fi
    
    if ! type -t log::header &>/dev/null; then
        log::header() { echo "=== $* ==="; }
        export -f log::header
    fi
    
    if ! type -t system::is_command &>/dev/null; then
        system::is_command() { command -v "$1" >/dev/null 2>&1; }
        export -f system::is_command
    fi
    
    if ! type -t flow::is_yes &>/dev/null; then
        flow::is_yes() { [[ "$1" == "yes" ]]; }
        export -f flow::is_yes
    fi
}

# Fast setup function that uses cached resources
setup_with_cache() {
    # Initialize cache if needed
    init_shared_cache
    
    # Setup standard mocks quickly using cache
    cache_standard_mocks
    
    # Set common test environment variables
    export FORCE="${FORCE:-no}"
    export YES="${YES:-no}"
    export ACTION="${ACTION:-status}"
    export REMOVE_DATA="${REMOVE_DATA:-no}"
    export LOG_LINES="${LOG_LINES:-50}"
    export FOLLOW="${FOLLOW:-no}"
    export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
    export QUIET="${QUIET:-no}"
    export SKIP_MODELS="${SKIP_MODELS:-no}"
    
    # CRITICAL: Source mock files after cache initialization
    # This ensures all mock functions are available
    local mocks_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/mocks" && pwd)"
    if [[ -d "$mocks_dir" ]]; then
        source "$mocks_dir/system_mocks.bash"
        source "$mocks_dir/mock_helpers.bash"
        source "$mocks_dir/resource_mocks.bash"
    fi
}

# Export cache functions
export -f init_shared_cache cache_standard_mocks setup_with_cache