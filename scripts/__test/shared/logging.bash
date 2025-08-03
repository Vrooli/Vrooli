#!/usr/bin/env bash
# Vrooli Test Logging - Centralized Logging Functions
# Standardized logging across the testing infrastructure

# Prevent duplicate loading
if [[ "${VROOLI_LOGGING_LOADED:-}" == "true" ]]; then
    return 0
fi
export VROOLI_LOGGING_LOADED="true"

# Logging configuration
VROOLI_LOG_LEVEL="${VROOLI_LOG_LEVEL:-INFO}"
VROOLI_LOG_FILE="${VROOLI_LOG_FILE:-}"
VROOLI_LOG_TO_CONSOLE="${VROOLI_LOG_TO_CONSOLE:-true}"

# Log levels (numeric values for comparison)
declare -A VROOLI_LOG_LEVELS=(
    ["DEBUG"]=0
    ["INFO"]=1
    ["WARN"]=2
    ["ERROR"]=3
)

# Color codes for terminal output
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    MAGENTA='\033[0;35m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    MAGENTA=''
    CYAN=''
    BOLD=''
    NC=''
fi

echo "[LOGGING] Loading Vrooli test logging (level: $VROOLI_LOG_LEVEL)" >&2

#######################################
# Initialize logging system
# Arguments: None
# Returns: 0 on success
#######################################
vrooli_logging_init() {
    # Get log configuration if config system is available
    if command -v vrooli_config_get >/dev/null 2>&1; then
        VROOLI_LOG_LEVEL=$(vrooli_config_get "logging.level" "INFO")
        VROOLI_LOG_FILE=$(vrooli_config_get "derived.log_file" "")
        VROOLI_LOG_TO_CONSOLE=$(vrooli_config_get_bool "logging.console" "true")
    fi
    
    # Create log file directory if needed
    if [[ -n "$VROOLI_LOG_FILE" ]]; then
        local log_dir
        log_dir=$(dirname "$VROOLI_LOG_FILE")
        mkdir -p "$log_dir" 2>/dev/null || true
    fi
    
    echo "[LOGGING] Logging initialized (level: $VROOLI_LOG_LEVEL, file: ${VROOLI_LOG_FILE:-none})"
    return 0
}

#######################################
# Check if a log level should be output
# Arguments: $1 - log level to check
# Returns: 0 if should log, 1 if should not
#######################################
_vrooli_should_log() {
    local level="$1"
    local current_level_num="${VROOLI_LOG_LEVELS[$VROOLI_LOG_LEVEL]:-1}"
    local check_level_num="${VROOLI_LOG_LEVELS[$level]:-0}"
    
    [[ $check_level_num -ge $current_level_num ]]
}

#######################################
# Format log message
# Arguments: $1 - level, $2 - component, $3 - message
# Returns: formatted log message
#######################################
_vrooli_format_log() {
    local level="$1"
    local component="$2"
    local message="$3"
    local timestamp
    
    timestamp=$(vrooli_timestamp "human" 2>/dev/null || date +"%Y-%m-%d %H:%M:%S")
    
    printf "[%s] [%s] [%s] %s\n" "$timestamp" "$level" "$component" "$message"
}

#######################################
# Write log message
# Arguments: $1 - level, $2 - component, $3 - message
# Returns: 0
#######################################
_vrooli_write_log() {
    local level="$1"
    local component="$2"
    local message="$3"
    
    if ! _vrooli_should_log "$level"; then
        return 0
    fi
    
    local formatted_msg
    formatted_msg=$(_vrooli_format_log "$level" "$component" "$message")
    
    # Write to console if enabled
    if [[ "$VROOLI_LOG_TO_CONSOLE" == "true" ]]; then
        case "$level" in
            "ERROR")
                echo "$formatted_msg" >&2
                ;;
            "WARN")
                echo "$formatted_msg" >&2
                ;;
            *)
                echo "$formatted_msg" >&2
                ;;
        esac
    fi
    
    # Write to file if configured
    if [[ -n "$VROOLI_LOG_FILE" ]]; then
        echo "$formatted_msg" >> "$VROOLI_LOG_FILE" 2>/dev/null || true
    fi
    
    return 0
}

#######################################
# Debug log message
# Arguments: $1 - component, $2 - message
# Returns: 0
#######################################
vrooli_log_debug() {
    local component="${1:-GENERAL}"
    local message="${2:-$1}"
    _vrooli_write_log "DEBUG" "$component" "$message"
}

#######################################
# Info log message
# Arguments: $1 - component, $2 - message
# Returns: 0
#######################################
vrooli_log_info() {
    local component="${1:-GENERAL}"
    local message="${2:-$1}"
    _vrooli_write_log "INFO" "$component" "$message"
}

#######################################
# Warning log message
# Arguments: $1 - component, $2 - message
# Returns: 0
#######################################
vrooli_log_warn() {
    local component="${1:-GENERAL}"
    local message="${2:-$1}"
    _vrooli_write_log "WARN" "$component" "$message"
}

#######################################
# Error log message
# Arguments: $1 - component, $2 - message
# Returns: 0
#######################################
vrooli_log_error() {
    local component="${1:-GENERAL}"
    local message="${2:-$1}"
    _vrooli_write_log "ERROR" "$component" "$message"
}

#######################################
# Log test start
# Arguments: $1 - test name
# Returns: 0
#######################################
vrooli_log_test_start() {
    local test_name="$1"
    vrooli_log_info "TEST" "Starting test: $test_name"
}

#######################################
# Log test end
# Arguments: $1 - test name, $2 - result (success/failure), $3 - duration (optional)
# Returns: 0
#######################################
vrooli_log_test_end() {
    local test_name="$1"
    local result="$2"
    local duration="${3:-unknown}"
    
    local level="INFO"
    if [[ "$result" != "success" ]]; then
        level="ERROR"
    fi
    
    _vrooli_write_log "$level" "TEST" "Test $result: $test_name (duration: ${duration}s)"
}

#######################################
# Log command execution
# Arguments: $1 - command, $2 - exit code, $3 - duration (optional)
# Returns: 0
#######################################
vrooli_log_command() {
    local command="$1"
    local exit_code="$2"
    local duration="${3:-unknown}"
    
    local level="DEBUG"
    local result="success"
    
    if [[ "$exit_code" -ne 0 ]]; then
        level="WARN"
        result="failure"
    fi
    
    _vrooli_write_log "$level" "COMMAND" "Command $result: $command (exit: $exit_code, duration: ${duration}s)"
}

#######################################
# Log setup/teardown operations
# Arguments: $1 - operation (setup/teardown), $2 - component, $3 - status
# Returns: 0
#######################################
vrooli_log_operation() {
    local operation="$1"
    local component="$2"
    local status="$3"
    
    local level="INFO"
    if [[ "$status" != "success" && "$status" != "complete" ]]; then
        level="ERROR"
    fi
    
    _vrooli_write_log "$level" "OPERATION" "${operation^} $status: $component"
}

#######################################
# Log performance metrics
# Arguments: $1 - test name, $2 - metric name, $3 - value, $4 - unit
# Returns: 0
#######################################
vrooli_log_metric() {
    local test_name="$1"
    local metric_name="$2"
    local value="$3"
    local unit="$4"
    
    vrooli_log_info "METRIC" "$test_name - $metric_name: $value $unit"
}

#######################################
# Log with custom level and component
# Arguments: $1 - level, $2 - component, $3 - message
# Returns: 0
#######################################
vrooli_log() {
    local level="$1"
    local component="$2"
    local message="$3"
    
    # Validate level
    if [[ -z "${VROOLI_LOG_LEVELS[$level]:-}" ]]; then
        level="INFO"
    fi
    
    _vrooli_write_log "$level" "$component" "$message"
}

#######################################
# Enable debug logging temporarily
# Arguments: None
# Returns: 0
#######################################
vrooli_log_enable_debug() {
    export VROOLI_LOG_LEVEL_BACKUP="$VROOLI_LOG_LEVEL"
    export VROOLI_LOG_LEVEL="DEBUG"
    vrooli_log_info "LOGGING" "Debug logging enabled"
}

#######################################
# Restore previous log level
# Arguments: None
# Returns: 0
#######################################
vrooli_log_restore_level() {
    if [[ -n "${VROOLI_LOG_LEVEL_BACKUP:-}" ]]; then
        export VROOLI_LOG_LEVEL="$VROOLI_LOG_LEVEL_BACKUP"
        unset VROOLI_LOG_LEVEL_BACKUP
        vrooli_log_info "LOGGING" "Log level restored to: $VROOLI_LOG_LEVEL"
    fi
}

#######################################
# Show current log configuration
# Arguments: None
# Returns: 0
#######################################
vrooli_log_show_config() {
    echo "[LOGGING] Configuration:"
    echo "  Level: $VROOLI_LOG_LEVEL"
    echo "  File: ${VROOLI_LOG_FILE:-none}"
    echo "  Console: $VROOLI_LOG_TO_CONSOLE"
}

#######################################
# Clear log file
# Arguments: None
# Returns: 0
#######################################
vrooli_log_clear() {
    if [[ -n "$VROOLI_LOG_FILE" && -f "$VROOLI_LOG_FILE" ]]; then
        > "$VROOLI_LOG_FILE"
        vrooli_log_info "LOGGING" "Log file cleared: $VROOLI_LOG_FILE"
    fi
}

#######################################
# Success log message (convenience function)
# Arguments: $1 - message
# Returns: 0
#######################################
vrooli_log_success() {
    local message="$1"
    echo -e "${GREEN}✅ $message${NC}" >&2
    if [[ -n "$VROOLI_LOG_FILE" ]]; then
        echo "[$(date +'%Y-%m-%d %H:%M:%S')] [SUCCESS] $message" >> "$VROOLI_LOG_FILE"
    fi
    return 0
}

#######################################
# Header log message (for section headers)
# Arguments: $1 - message
# Returns: 0
#######################################
vrooli_log_header() {
    local message="$1"
    echo -e "${BOLD}${CYAN}$message${NC}" >&2
    if [[ -n "$VROOLI_LOG_FILE" ]]; then
        echo "[$(date +'%Y-%m-%d %H:%M:%S')] [HEADER] $message" >> "$VROOLI_LOG_FILE"
    fi
    return 0
}

#######################################
# Export logging functions
#######################################
export -f vrooli_logging_init
export -f _vrooli_should_log _vrooli_format_log _vrooli_write_log
export -f vrooli_log_debug vrooli_log_info vrooli_log_warn vrooli_log_error
export -f vrooli_log_test_start vrooli_log_test_end vrooli_log_command vrooli_log_operation
export -f vrooli_log_metric vrooli_log vrooli_log_enable_debug vrooli_log_restore_level
export -f vrooli_log_show_config vrooli_log_clear vrooli_log_success vrooli_log_header

echo "[LOGGING] Vrooli test logging functions loaded successfully" >&2