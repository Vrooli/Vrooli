#!/usr/bin/env bash
# n8n Common Utility Functions
# Shared utilities used across all modules

# Source required utilities
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../../lib/docker-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../../lib/http-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../../lib/wait-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/constants.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/health.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/recovery.sh" 2>/dev/null || true

# Docker checking now handled by docker::check_daemon() from shared utilities

# Container check functions are in utils.sh
# Direct usage: n8n::container_exists_any and n8n::container_running

#######################################
# Generate secure random password
#######################################
n8n::generate_password() {
    # Use multiple sources for randomness
    if system::is_command "openssl"; then
        openssl rand -base64 32 | tr -d "=+/" | cut -c1-16
    elif [[ -r /dev/urandom ]]; then
        tr -dc 'A-Za-z0-9!@#$%^&*' < /dev/urandom | head -c 16
    else
        # Fallback to timestamp-based password
        echo "n8n$(date +%s)$RANDOM" | sha256sum | cut -c1-16
    fi
}

#######################################
# Create n8n data directory with enhanced validation
#######################################
n8n::create_directories() {
    n8n::log_with_context "info" "setup" "Creating n8n data directory..."
    # Check if directory already exists and is corrupted
    if [[ -d "$N8N_DATA_DIR" ]] && ! n8n::detect_filesystem_corruption; then
        n8n::log_with_context "info" "setup" "Data directory already exists and appears healthy"
        return 0
    fi
    # Remove any corrupted directory
    if [[ -d "$N8N_DATA_DIR" ]]; then
        n8n::log_with_context "warn" "setup" "Removing potentially corrupted directory"
        trash::safe_remove "$N8N_DATA_DIR" --no-confirm 2>/dev/null || true
    fi
    # Create fresh directory
    mkdir -p "$N8N_DATA_DIR" || {
        n8n::log_with_context "error" "setup" "$N8N_ERR_CREATE_DIR_FAILED"
        return 1
    }
    # Set proper ownership and permissions
    if command -v sudo::restore_owner &>/dev/null; then
        sudo::restore_owner "$N8N_DATA_DIR"
    else
        local current_user
        current_user=$(whoami)
        chown "$current_user:$current_user" "$N8N_DATA_DIR" 2>/dev/null || true
    fi
    chmod 755 "$N8N_DATA_DIR" || true
    # Verify directory is healthy
    if ! n8n::detect_filesystem_corruption; then
        n8n::log_with_context "error" "setup" "Created directory is still corrupted"
        return 1
    fi
    # Add rollback action
    resources::add_rollback_action \
        "Remove n8n data directory" \
        "trash::safe_remove '$N8N_DATA_DIR' --no-confirm 2>/dev/null || true" \
        10
    n8n::log_with_context "success" "setup" "n8n directories created with proper permissions"
    return 0
}

#######################################
# Create Docker network for n8n
#######################################
n8n::create_network() {
    if docker::create_network "$N8N_NETWORK_NAME"; then
        n8n::log_with_context "success" "network" "Docker network created"
        resources::add_rollback_action \
            "Remove Docker network" \
            "docker network rm $N8N_NETWORK_NAME 2>/dev/null || true" \
            5
    fi
}

#######################################
# Check if port is available
# Args: port number
# Returns: 0 if available, 1 if in use
#######################################
n8n::is_port_available() {
    local port=$1
    docker::is_port_available "$port"
}

#######################################
# Wait for service to be ready
# Args: max_attempts (optional, default 30)
# Returns: 0 if ready, 1 if timeout
#######################################
n8n::wait_for_ready() {
    local max_attempts=${1:-$N8N_HEALTH_CHECK_MAX_ATTEMPTS}
    local timeout=$((max_attempts * N8N_HEALTH_CHECK_INTERVAL))
    
    wait::for_condition "n8n::is_healthy" "$timeout" "n8n to be ready"
}

#######################################
# Check if basic auth is enabled
# Returns: 0 if enabled, 1 if disabled
#######################################
n8n::is_basic_auth_enabled() {
    local auth_active=$(n8n::extract_container_env "N8N_BASIC_AUTH_ACTIVE")
    [[ "$auth_active" == "true" ]]
}

#######################################
# Validate workflow ID format
# Args: workflow_id
# Returns: 0 if valid, 1 if invalid
#######################################
n8n::validate_workflow_id() {
    local workflow_id=$1
    # n8n workflow IDs are typically numeric
    if [[ ! "$workflow_id" =~ ^[0-9]+$ ]]; then
        log::error "Invalid workflow ID format: $workflow_id"
        log::info "Workflow IDs should be numeric (e.g., 1, 42, 123)"
        return 1
    fi
    return 0
}

#######################################
# Check for required commands
# Returns: 0 if all present, 1 if any missing
#######################################
n8n::check_requirements() {
    local missing=0
    # Required commands
    local required_commands=("docker" "curl")
    for cmd in "${required_commands[@]}"; do
        if ! system::is_command "$cmd"; then
            log::error "Required command not found: $cmd"
            missing=1
        fi
    done
    # Optional but recommended commands
    local optional_commands=("jq" "openssl" "sqlite3")
    for cmd in "${optional_commands[@]}"; do
        if ! system::is_command "$cmd"; then
            log::warn "Optional command not found: $cmd"
            log::info "Some features may be limited without $cmd"
        fi
    done
    return $missing
}

#######################################
# Simplified injection wrapper with built-in validation
# Arguments:
#   $1 - injection configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n::inject_data() {
    local config="$1"
    if [[ -z "$config" ]]; then
        log::error "Injection configuration required"
        log::info "Use: --injection-config 'JSON_CONFIG'"
        return 1
    fi
    # Framework handles all validation and error handling
    "${N8N_LIB_DIR}/inject.sh" --inject "$config"
}

#######################################
# Simplified injection validation wrapper
# Arguments:
#   $1 - injection configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
n8n::validate_injection() {
    local config="$1"
    if [[ -z "$config" ]]; then
        log::error "Injection configuration required for validation"
        log::info "Use: --injection-config 'JSON_CONFIG'"
        return 1
    fi
    # Framework handles all validation logic
    "${N8N_LIB_DIR}/inject.sh" --validate "$config"
}

