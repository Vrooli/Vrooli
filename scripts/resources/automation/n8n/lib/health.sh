#!/usr/bin/env bash
# n8n Health Checking Functions
# All health monitoring, validation, and diagnostic functions

# Source required utilities
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/wait-utils.sh" 2>/dev/null || true

#######################################
# Check if n8n API is responsive with enhanced diagnostics
# Returns: 0 if responsive, 1 otherwise
#######################################
n8n::is_healthy() {
    # Check if container is running first
    if ! n8n::container_running; then
        return 1
    fi
    # Check for database corruption indicators in logs
    local recent_logs
    recent_logs=$(docker::get_logs "$N8N_CONTAINER_NAME" 50 2>&1 || echo "")
    if echo "$recent_logs" | grep -qi "SQLITE_READONLY\|database.*locked\|database.*corrupted"; then
        log::warn "Database corruption detected in logs"
        return 1
    fi
    # n8n uses /healthz endpoint for health checks
    if system::is_command "curl"; then
        # Use standardized wait utility
        wait::for_condition \
            "http::request 'GET' '$N8N_BASE_URL/healthz' >/dev/null 2>&1" \
            10 \
            "n8n healthz endpoint"
        return $?
    fi
    return 1
}

#######################################
# Detect filesystem corruption in n8n data directory
# Returns: 0 if healthy, 1 if corrupted
#######################################
n8n::detect_filesystem_corruption() {
    if [[ ! -d "$N8N_DATA_DIR" ]]; then
        log::warn "n8n data directory does not exist: $N8N_DATA_DIR"
        return 1
    fi
    # Check if directory has zero links (corruption indicator)
    local links
    links=$(stat -c '%h' "$N8N_DATA_DIR" 2>/dev/null || echo "0")
    if [[ "$links" == "0" ]]; then
        log::error "Filesystem corruption detected: directory has 0 links"
        return 1
    fi
    # Check for deleted but open database files (if container is running)
    if n8n::container_running; then
        local n8n_pid
        n8n_pid=$(docker::exec "$N8N_CONTAINER_NAME" pgrep -f 'node.*n8n' 2>/dev/null | head -1)
        if [[ -n "$n8n_pid" ]]; then
            local deleted_files
            deleted_files=$(docker::exec "$N8N_CONTAINER_NAME" lsof -p "$n8n_pid" 2>/dev/null | grep -c '(deleted)' 2>/dev/null || echo "0")
            if [[ "$deleted_files" =~ ^[0-9]+$ ]] && [[ "$deleted_files" -gt 0 ]]; then
                log::error "Database corruption detected: $deleted_files deleted files still open"
                return 1
            fi
        fi
    fi
    return 0
}

#######################################
# Check database health and integrity
# Returns: 0 if healthy, 1 if corrupted
#######################################
n8n::check_database_health() {
    local db_file="$N8N_DATA_DIR/database.sqlite"
    if [[ ! -f "$db_file" ]]; then
        log::warn "Database file does not exist: $db_file"
        return 1
    fi
    # Check file permissions
    if [[ ! -r "$db_file" ]] || [[ ! -w "$db_file" ]]; then
        log::error "Database file has incorrect permissions"
        return 1
    fi
    # Check if SQLite can open the database
    if system::is_command "sqlite3"; then
        if ! echo "PRAGMA integrity_check;" | sqlite3 "$db_file" >/dev/null 2>&1; then
            log::error "Database integrity check failed"
            return 1
        fi
    fi
    return 0
}

#######################################
# Comprehensive health check with automatic recovery
# Returns: 0 if healthy, 1 if issues detected
#######################################
n8n::comprehensive_health_check() {
    log::info "Running comprehensive n8n health check..."
    local issues_found=0
    # Check filesystem corruption
    if ! n8n::detect_filesystem_corruption; then
        log::error "Filesystem corruption detected"
        if [[ "${AUTO_RECOVER:-yes}" == "yes" ]]; then
            if n8n::auto_recover; then
                log::success "Automatic recovery completed"
            else
                log::error "Automatic recovery failed"
                issues_found=1
            fi
        else
            issues_found=1
        fi
    fi
    # Check database health
    if ! n8n::check_database_health; then
        log::warn "Database health issues detected"
        issues_found=1
    fi
    return $issues_found
}

#######################################
# Check basic container and endpoint health
# Returns: 0 if healthy, 2 if unhealthy
#######################################
n8n::check_basic_health() {
    # Basic container health
    if ! n8n::container_running; then
        return 2
    fi
    # Check basic healthz endpoint
    if ! n8n::is_healthy >/dev/null 2>&1; then
        return 2
    fi
    return 0
}

#######################################
# Test API functionality with current key
# Returns: 0 (HEALTHY), 1 (DEGRADED), 2 (UNHEALTHY)
#######################################
n8n::test_api_functionality() {
    # Get API key using shared utility
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        # No API key configured - container works but automation unusable
        return 1  # DEGRADED
    fi
    # Test API authentication using shared utility
    local response
    local http_code
    response=$(n8n::safe_curl_call "GET" "${N8N_BASE_URL}/api/v1/workflows?limit=1" "" "X-N8N-API-KEY: $api_key\nAccept: application/json")
    http_code=$?
    case "$http_code" in
        200)
            return 0  # HEALTHY
            ;;
        401)
            return 1  # DEGRADED
            ;;
        *)
            return 2  # UNHEALTHY
            ;;
    esac
}

#######################################
# Tiered health check with API functionality validation
# REFACTORED: Main orchestrator (was 45 lines, now 20 lines)
# Returns: 
#   0 = HEALTHY (fully functional with API auth)
#   1 = DEGRADED (container healthy, but API unusable) 
#   2 = UNHEALTHY (container or basic health issues)
# Outputs: Health tier (HEALTHY/DEGRADED/UNHEALTHY)
#######################################
n8n::tiered_health_check() {
    # Tier 1: Basic container and endpoint health
    if ! n8n::check_basic_health; then
        echo "UNHEALTHY"
        return 2
    fi
    # Tier 2: API functionality validation
    local api_result
    n8n::test_api_functionality
    api_result=$?
    case "$api_result" in
        0)
            echo "HEALTHY"
            return 0
            ;;
        1)
            echo "DEGRADED"
            return 1
            ;;
        *)
            echo "UNHEALTHY"
            return 2
            ;;
    esac
}
