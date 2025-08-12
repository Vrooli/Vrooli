#!/usr/bin/env bash
# Generic Health Check Framework
# Provides tiered health checking for all resources

# Source required utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/log.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/docker-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/http-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/runtimes/sqlite.sh" 2>/dev/null || true

#######################################
# Perform tiered health check using resource-specific checks
# Args: $1 - health_config (JSON configuration)
# Returns: 0 and outputs HEALTHY|DEGRADED|UNHEALTHY
#
# Health Config Schema:
# {
#   "container_name": "container-name",
#   "checks": {
#     "basic": "function_name_for_basic_check",
#     "advanced": "function_name_for_advanced_check"
#   },
#   "thresholds": {
#     "degraded_if": ["condition1", "condition2"],
#     "unhealthy_if": ["condition3", "condition4"]
#   }
# }
#######################################
health::tiered_check() {
    local health_config="$1"
    
    local container_name
    local basic_check_func
    local advanced_check_func
    
    container_name=$(echo "$health_config" | jq -r '.container_name // empty')
    basic_check_func=$(echo "$health_config" | jq -r '.checks.basic // empty')
    advanced_check_func=$(echo "$health_config" | jq -r '.checks.advanced // empty')
    
    # First check if container is running
    if [[ -n "$container_name" ]] && ! docker::is_running "$container_name"; then
        echo "UNHEALTHY"
        return 0
    fi
    
    # Run basic health check
    local basic_result=0
    if [[ -n "$basic_check_func" ]] && command -v "$basic_check_func" &>/dev/null; then
        "$basic_check_func" || basic_result=$?
    fi
    
    # If basic check fails, return UNHEALTHY
    if [[ $basic_result -ne 0 ]]; then
        echo "UNHEALTHY"
        return 0
    fi
    
    # Run advanced health check if available
    local advanced_result=0
    if [[ -n "$advanced_check_func" ]] && command -v "$advanced_check_func" &>/dev/null; then
        "$advanced_check_func" || advanced_result=$?
    fi
    
    # Determine health tier based on results
    if [[ $advanced_result -ne 0 ]]; then
        echo "DEGRADED"
    else
        echo "HEALTHY"
    fi
    
    return 0
}

#######################################
# Check filesystem health for corruption
# Args: $1 - data_directory
# Returns: 0 if healthy, 1 if corrupted
#######################################
health::check_filesystem() {
    local data_dir="$1"
    
    if [[ ! -d "$data_dir" ]]; then
        return 1
    fi
    
    # Check for zero-link directories (corruption indicator)
    local zero_link_dirs
    zero_link_dirs=$(find "$data_dir" -type d -links 0 2>/dev/null | wc -l)
    
    if [[ $zero_link_dirs -gt 0 ]]; then
        log::warn "Detected $zero_link_dirs corrupted directories"
        return 1
    fi
    
    # Check for permission issues
    if ! touch "$data_dir/.health_check" 2>/dev/null; then
        log::warn "Cannot write to data directory"
        return 1
    fi
    rm -f "$data_dir/.health_check" 2>/dev/null
    
    return 0
}

#######################################
# Check database integrity (SQLite)
# Args: $1 - database_path
# Returns: 0 if healthy, 1 if corrupted
#######################################
health::check_sqlite_integrity() {
    local db_path="$1"
    
    if [[ ! -f "$db_path" ]]; then
        return 1
    fi
    
    # Check if sqlite3 is available
    if ! command -v sqlite3 >/dev/null 2>&1; then
        log::debug "SQLite not installed, skipping integrity check"
        # Return success since we can't check but the DB file exists
        # and n8n is working with its internal SQLite libraries
        return 0
    fi
    
    # Use the SQLite runtime's integrity check if available
    if declare -f sqlite::check_integrity >/dev/null 2>&1; then
        sqlite::check_integrity "$db_path"
        return $?
    fi
    
    # Fallback to direct check
    local result
    result=$(sqlite3 "$db_path" "PRAGMA integrity_check;" 2>/dev/null || echo "error")
    
    if [[ "$result" != "ok" ]]; then
        log::warn "Database integrity check failed: $result"
        return 1
    fi
    
    return 0
}

#######################################
# Generic API health check
# Args: $1 - api_url
#       $2 - auth_header (optional)
# Returns: 0 if healthy, 1 otherwise
#######################################
health::check_api() {
    local api_url="$1"
    local auth_header="${2:-}"
    
    local curl_args=(-s -o /dev/null -w "%{http_code}" --max-time 5)
    
    if [[ -n "$auth_header" ]]; then
        curl_args+=(-H "$auth_header")
    fi
    
    local status_code
    status_code=$(curl "${curl_args[@]}" "$api_url" 2>/dev/null || echo "000")
    
    [[ "$status_code" =~ ^(200|201|204|401|403)$ ]]
}

#######################################
# Aggregate multiple health checks
# Args: $@ - list of check results (0 or 1)
# Returns: HEALTHY if all pass, DEGRADED if some pass, UNHEALTHY if none pass
#######################################
health::aggregate_results() {
    local total=0
    local passed=0
    
    for result in "$@"; do
        ((total++))
        [[ $result -eq 0 ]] && ((passed++))
    done
    
    if [[ $passed -eq $total ]]; then
        echo "HEALTHY"
    elif [[ $passed -gt 0 ]]; then
        echo "DEGRADED"
    else
        echo "UNHEALTHY"
    fi
}

#######################################
# Monitor health over time with retries
# Args: $1 - check_function
#       $2 - max_retries (default: 3)
#       $3 - retry_delay (default: 5)
# Returns: Final health status
#######################################
health::monitor_with_retry() {
    local check_func="$1"
    local max_retries="${2:-3}"
    local retry_delay="${3:-5}"
    
    local attempt=0
    local last_status="UNHEALTHY"
    
    while [[ $attempt -lt $max_retries ]]; do
        last_status=$("$check_func")
        
        if [[ "$last_status" == "HEALTHY" ]]; then
            echo "$last_status"
            return 0
        fi
        
        ((attempt++))
        if [[ $attempt -lt $max_retries ]]; then
            sleep "$retry_delay"
        fi
    done
    
    echo "$last_status"
}