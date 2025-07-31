#!/usr/bin/env bash
# Fast setup helper for BATS tests
# Provides lightweight alternatives to heavy setup operations

# Cache for sourced files (persists within test file execution)
declare -gA SOURCED_CACHE

# Fast source function that caches file content
fast_source() {
    local file="$1"
    local cache_key="${file//\//_}"
    
    # If already sourced in this session, skip
    if [[ -n "${SOURCED_CACHE[$cache_key]:-}" ]]; then
        return 0
    fi
    
    # Source the file and mark as cached
    source "$file"
    SOURCED_CACHE[$cache_key]=1
}

# Lightweight mock setup (no external dependencies)
fast_setup_mocks() {
    # Essential environment variables
    export FORCE="${FORCE:-no}"
    export YES="${YES:-no}"
    export SKIP_MODELS="${SKIP_MODELS:-no}"
    export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
    export QUIET="${QUIET:-no}"
    
    # Essential mock functions
    if ! type -t mock::network::set_online &>/dev/null; then
        mock::network::set_online() { return 0; }
        mock::network::set_offline() { return 1; }
        export -f mock::network::set_online mock::network::set_offline
    fi
    
    # Essential logging functions
    if ! type -t log::info &>/dev/null; then
        log::info() { echo "[INFO] $*"; }
        log::error() { echo "[ERROR] $*" >&2; }
        log::warning() { echo "[WARNING] $*" >&2; }
        log::success() { echo "[SUCCESS] $*"; }
        log::header() { echo "=== $* ==="; }
        log::debug() { [[ "${DEBUG:-}" == "true" ]] && echo "[DEBUG] $*" >&2 || true; }
        export -f log::info log::error log::warning log::success log::header log::debug
    fi
    
    # Essential system functions
    if ! type -t system::is_command &>/dev/null; then
        system::is_command() { command -v "$1" >/dev/null 2>&1; }
        flow::is_yes() { [[ "$1" == "yes" ]]; }
        export -f system::is_command flow::is_yes
    fi
    
    # Cleanup function
    if ! type -t cleanup_mocks &>/dev/null; then
        cleanup_mocks() {
            [[ -d "${MOCK_RESPONSES_DIR:-}" ]] && rm -rf "$MOCK_RESPONSES_DIR" || true
        }
        export -f cleanup_mocks
    fi
}

# Fast path resolution (cached)
declare -gA PATH_CACHE

fast_resolve_path() {
    local key="$1"
    local default_path="$2"
    
    # Return cached path if available
    if [[ -n "${PATH_CACHE[$key]:-}" ]]; then
        echo "${PATH_CACHE[$key]}"
        return 0
    fi
    
    # Resolve and cache the path
    PATH_CACHE[$key]="$default_path"
    echo "$default_path"
}

# Export fast functions
export -f fast_source fast_setup_mocks fast_resolve_path