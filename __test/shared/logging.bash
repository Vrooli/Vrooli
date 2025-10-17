#!/usr/bin/env bash
# Shared logging utilities for Vrooli testing infrastructure
# Provides consistent logging across all test phases

# Color codes for output
declare -g RED='\033[0;31m'
declare -g GREEN='\033[0;32m'
declare -g YELLOW='\033[1;33m'
declare -g BLUE='\033[0;34m'
declare -g PURPLE='\033[0;35m'
declare -g CYAN='\033[0;36m'
declare -g WHITE='\033[1;37m'
declare -g NC='\033[0m' # No Color

# Log levels
declare -g LOG_LEVEL="${LOG_LEVEL:-INFO}"

# Timestamp function
timestamp() {
    date '+%Y-%m-%d %H:%M:%S'
}

# Core logging functions
log_debug() {
    [[ "$LOG_LEVEL" == "DEBUG" ]] || return 0
    echo -e "${PURPLE}[$(timestamp)] [DEBUG]${NC} $*" >&2
}

log_info() {
    echo -e "${BLUE}[$(timestamp)] [INFO]${NC} $*" >&2
}

log_success() {
    echo -e "${GREEN}[$(timestamp)] [SUCCESS]${NC} $*" >&2
}

log_warning() {
    echo -e "${YELLOW}[$(timestamp)] [WARNING]${NC} $*" >&2
}

log_error() {
    echo -e "${RED}[$(timestamp)] [ERROR]${NC} $*" >&2
}

# Test-specific logging functions
log_test_start() {
    local test_name="$1"
    echo -e "${CYAN}[$(timestamp)] [TEST_START]${NC} $test_name" >&2
}

log_test_pass() {
    local test_name="$1"
    echo -e "${GREEN}[$(timestamp)] [TEST_PASS]${NC} ✅ $test_name" >&2
}

log_test_fail() {
    local test_name="$1"
    local reason="${2:-Unknown error}"
    echo -e "${RED}[$(timestamp)] [TEST_FAIL]${NC} ❌ $test_name - $reason" >&2
}

log_test_skip() {
    local test_name="$1"
    local reason="${2:-Skipped}"
    echo -e "${YELLOW}[$(timestamp)] [TEST_SKIP]${NC} ⏭️  $test_name - $reason" >&2
}

# Progress reporting
log_progress() {
    local current="$1"
    local total="$2"
    local task="${3:-Processing}"
    local percentage=$((current * 100 / total))
    echo -e "${WHITE}[$(timestamp)] [PROGRESS]${NC} $task [$current/$total] ($percentage%)" >&2
}

# Section headers
log_section() {
    local section_name="$1"
    local separator_line=""
    for ((i=1; i<=${#section_name}+4; i++)); do
        separator_line+="="
    done
    
    echo -e "\n${WHITE}$separator_line${NC}" >&2
    echo -e "${WHITE}  $section_name${NC}" >&2
    echo -e "${WHITE}$separator_line${NC}\n" >&2
}

# File/directory reporting
log_file_found() {
    local file_path="$1"
    local file_type="${2:-file}"
    [[ "$LOG_LEVEL" == "DEBUG" ]] && echo -e "${PURPLE}[$(timestamp)] [FILE_FOUND]${NC} $file_type: $file_path" >&2
}

log_file_missing() {
    local file_path="$1"
    local file_type="${2:-file}"
    echo -e "${YELLOW}[$(timestamp)] [FILE_MISSING]${NC} $file_type: $file_path" >&2
}

log_file_error() {
    local file_path="$1"
    local error_message="$2"
    echo -e "${RED}[$(timestamp)] [FILE_ERROR]${NC} $file_path - $error_message" >&2
}

# Command execution logging  
log_command() {
    local command="$1"
    [[ "$LOG_LEVEL" == "DEBUG" ]] && echo -e "${PURPLE}[$(timestamp)] [COMMAND]${NC} Executing: $command" >&2
}

log_command_success() {
    local command="$1"
    [[ "$LOG_LEVEL" == "DEBUG" ]] && echo -e "${GREEN}[$(timestamp)] [COMMAND_SUCCESS]${NC} $command" >&2
}

log_command_failure() {
    local command="$1"
    local exit_code="${2:-1}"
    echo -e "${RED}[$(timestamp)] [COMMAND_FAILURE]${NC} $command (exit code: $exit_code)" >&2
}

# Cache logging
log_cache_hit() {
    local cache_key="$1"
    [[ "$LOG_LEVEL" == "DEBUG" ]] && echo -e "${CYAN}[$(timestamp)] [CACHE_HIT]${NC} $cache_key" >&2
}

log_cache_miss() {
    local cache_key="$1"
    [[ "$LOG_LEVEL" == "DEBUG" ]] && echo -e "${YELLOW}[$(timestamp)] [CACHE_MISS]${NC} $cache_key" >&2
}

# Summary reporting
log_summary() {
    local total="$1"
    local passed="$2"
    local failed="$3"
    local skipped="${4:-0}"
    
    log_section "Test Summary"
    log_info "Total: $total"
    [[ $passed -gt 0 ]] && log_success "Passed: $passed"
    [[ $failed -gt 0 ]] && log_error "Failed: $failed"
    [[ $skipped -gt 0 ]] && log_warning "Skipped: $skipped"
    
    local success_rate=0
    if [[ $total -gt 0 ]]; then
        success_rate=$((passed * 100 / total))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log_success "🎉 All tests passed! (Success rate: ${success_rate}%)"
    else
        log_error "💥 Some tests failed. (Success rate: ${success_rate}%)"
    fi
}

# Utility functions
is_verbose() {
    [[ -n "${VERBOSE:-}" ]]
}

is_dry_run() {
    [[ -n "${DRY_RUN:-}" ]]
}

is_parallel() {
    [[ -n "${PARALLEL:-}" ]]
}

# Export functions for use in subshells
export -f timestamp log_debug log_info log_success log_warning log_error
export -f log_test_start log_test_pass log_test_fail log_test_skip
export -f log_progress log_section log_file_found log_file_missing log_file_error
export -f log_command log_command_success log_command_failure
export -f log_cache_hit log_cache_miss log_summary
export -f is_verbose is_dry_run is_parallel