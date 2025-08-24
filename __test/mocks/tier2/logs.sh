#!/usr/bin/env bash
# Logs Mock - Tier 2 (Stateful)
# 
# Provides centralized logging functionality for Tier 2 mocks:
# - Command logging with timestamps
# - System-specific log categorization
# - State tracking for mock usage
# - Debug output control
# - Error injection for testing
#
# Coverage: ~80% of common logging operations in 250 lines

# === Configuration ===
declare -gA LOGS_CONFIG=(
    [status]="active"
    [dir]="/tmp/mock-logs-$$"
    [verbose]="${MOCK_UTILS_VERBOSE:-false}"
    [strict]="${MOCK_LOG_CALL_STRICT:-false}"
    [error_mode]=""
    [version]="2.0.0"
)

declare -gA LOGS_FILES=(
    [command]="command_calls.log"
    [docker]="docker_calls.log"
    [http]="http_calls.log"
    [state]="used_mocks.log"
)

declare -ga LOGS_HISTORY=()
declare -gi LOGS_CALL_COUNT=0

# Debug mode
declare -g LOGS_DEBUG="${LOGS_DEBUG:-}"

# === Helper Functions ===
logs_debug() {
    [[ -n "$LOGS_DEBUG" ]] && echo "[MOCK:LOGS] $*" >&2
}

logs_check_error() {
    case "${LOGS_CONFIG[error_mode]}" in
        "disk_full")
            echo "Error: No space left on device" >&2
            return 1
            ;;
        "permission_denied")
            echo "Error: Permission denied" >&2
            return 1
            ;;
        "io_error")
            echo "Error: I/O error" >&2
            return 1
            ;;
    esac
    return 0
}

# === Initialization ===
logs_init() {
    local custom_dir="${1:-}"
    
    # Use custom directory if provided
    [[ -n "$custom_dir" ]] && LOGS_CONFIG[dir]="$custom_dir"
    
    # Create log directory
    if ! mkdir -p "${LOGS_CONFIG[dir]}" 2>/dev/null; then
        LOGS_CONFIG[dir]="/tmp/mock-logs-fallback-$$"
        mkdir -p "${LOGS_CONFIG[dir]}" 2>/dev/null || return 1
    fi
    
    # Initialize log files
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
    for file in "${LOGS_FILES[@]}"; do
        echo "# Log started at $timestamp" > "${LOGS_CONFIG[dir]}/$file"
    done
    
    # Export for compatibility
    export MOCK_LOG_DIR="${LOGS_CONFIG[dir]}"
    export MOCK_RESPONSES_DIR="${LOGS_CONFIG[dir]}"
    
    logs_debug "Logging initialized at ${LOGS_CONFIG[dir]}"
    return 0
}

# === Main Logging Function ===
mock::log_call() {
    local system="$1"
    shift
    local call_details="$*"
    
    logs_debug "log_call: $system - $call_details"
    
    if ! logs_check_error; then
        return $?
    fi
    
    # Validate inputs
    [[ -z "$system" ]] && return 1
    
    # Initialize if needed
    [[ ! -d "${LOGS_CONFIG[dir]}" ]] && logs_init
    
    # Increment counter
    ((LOGS_CALL_COUNT++))
    
    # Create timestamp
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
    
    # Determine log file
    local log_file
    case "$system" in
        docker|docker-compose)
            log_file="${LOGS_CONFIG[dir]}/${LOGS_FILES[docker]}"
            ;;
        http|curl|wget)
            log_file="${LOGS_CONFIG[dir]}/${LOGS_FILES[http]}"
            ;;
        *)
            log_file="${LOGS_CONFIG[dir]}/${LOGS_FILES[command]}"
            ;;
    esac
    
    # Log entry
    local log_entry="[$timestamp] $system: $call_details"
    LOGS_HISTORY+=("$log_entry")
    
    # Write to file
    if ! echo "$log_entry" >> "$log_file" 2>/dev/null; then
        if [[ "${LOGS_CONFIG[strict]}" == "true" ]]; then
            return 1
        fi
    fi
    
    return 0
}

# === State Logging ===
mock::log_state() {
    local mock_name="$1"
    local state="$2"
    
    logs_debug "log_state: $mock_name - $state"
    
    [[ ! -d "${LOGS_CONFIG[dir]}" ]] && logs_init
    
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
    local log_file="${LOGS_CONFIG[dir]}/${LOGS_FILES[state]}"
    
    echo "[$timestamp] $mock_name: $state" >> "$log_file" 2>/dev/null
    return 0
}

# === Verification Logging ===
mock::log_and_verify() {
    local command="$1"
    shift
    local args="$*"
    
    # Log the call
    mock::log_call "$command" "$args"
    
    # Verification hook for tests
    if declare -F mock::verify_call >/dev/null 2>&1; then
        mock::verify_call "$command" "$args"
    fi
    
    return 0
}

# === Query Functions ===
logs_get_calls() {
    local system="${1:-}"
    
    if [[ -z "$system" ]]; then
        printf '%s\n' "${LOGS_HISTORY[@]}"
    else
        printf '%s\n' "${LOGS_HISTORY[@]}" | grep "\[$system:"
    fi
}

logs_count_calls() {
    local system="${1:-}"
    
    if [[ -z "$system" ]]; then
        echo "$LOGS_CALL_COUNT"
    else
        logs_get_calls "$system" | wc -l
    fi
}

# === Mock Control Functions ===
logs_mock_reset() {
    logs_debug "Resetting mock state"
    
    LOGS_HISTORY=()
    LOGS_CALL_COUNT=0
    LOGS_CONFIG[error_mode]=""
    
    # Re-initialize files
    logs_init "${LOGS_CONFIG[dir]}"
}

logs_mock_set_error() {
    LOGS_CONFIG[error_mode]="$1"
    logs_debug "Set error mode: $1"
}

logs_mock_dump_state() {
    echo "=== Logs Mock State ==="
    echo "Status: ${LOGS_CONFIG[status]}"
    echo "Directory: ${LOGS_CONFIG[dir]}"
    echo "Call Count: $LOGS_CALL_COUNT"
    echo "History Size: ${#LOGS_HISTORY[@]}"
    echo "Error Mode: ${LOGS_CONFIG[error_mode]:-none}"
    echo "==================="
}

# === Cleanup Functions ===
logs_cleanup() {
    logs_debug "Cleaning up log directory"
    
    if [[ -d "${LOGS_CONFIG[dir]}" ]] && [[ "${LOGS_CONFIG[dir]}" == /tmp/* ]]; then
        rm -rf "${LOGS_CONFIG[dir]}"
    fi
}

# === Convention-based Test Functions ===
test_logs_connection() {
    logs_debug "Testing connection..."
    
    # Check if we can write to log directory
    if [[ -w "${LOGS_CONFIG[dir]}" ]]; then
        logs_debug "Connection test passed"
        return 0
    else
        logs_debug "Connection test failed"
        return 1
    fi
}

test_logs_health() {
    logs_debug "Testing health..."
    
    test_logs_connection || return 1
    
    # Check if log files exist
    for file in "${LOGS_FILES[@]}"; do
        [[ -f "${LOGS_CONFIG[dir]}/$file" ]] || return 1
    done
    
    logs_debug "Health test passed"
    return 0
}

test_logs_basic() {
    logs_debug "Testing basic operations..."
    
    # Test logging
    mock::log_call "test" "sample call" || return 1
    
    # Verify it was logged
    [[ $LOGS_CALL_COUNT -gt 0 ]] || return 1
    
    logs_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f mock::log_call
export -f mock::log_state
export -f mock::log_and_verify
export -f logs_get_calls
export -f logs_count_calls
export -f test_logs_connection
export -f test_logs_health
export -f test_logs_basic
export -f logs_mock_reset
export -f logs_mock_set_error
export -f logs_mock_dump_state
export -f logs_debug
export -f logs_init
export -f logs_cleanup

# Auto-initialize if enabled (skip in BATS environment)
if [[ "${MOCK_UTILS_AUTO_INIT:-true}" == "true" ]] && [[ -z "${BATS_TEST_FILENAME:-}" ]]; then
    logs_init
fi

# Compatibility exports
export MOCK_UTILS_LOADED="true"

# Compatibility wrapper
mock::init_logging() { logs_init "$@"; }
export -f mock::init_logging

# logs_debug "Logs Tier 2 mock initialized" # Disabled in BATS environment
# Ensure we return success when sourced
return 0 2>/dev/null || true
