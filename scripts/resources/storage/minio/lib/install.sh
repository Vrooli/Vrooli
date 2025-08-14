#!/usr/bin/env bash
# MinIO Installation Functions
# Handles installation and uninstallation of MinIO

# Source shared secrets management library using var_ variables
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

#######################################
# Install MinIO
# Returns: 0 on success, 1 on failure
#######################################
minio::install() {
    log::info "Installing MinIO Object Storage..."
    
    # Pre-installation checks
    if ! minio::install::pre_checks; then
        return 1
    fi
    
    # Check if already installed
    if minio::common::container_exists; then
        log::warn "${MSG_CONTAINER_EXISTS}"
        
        if minio::common::is_running; then
            log::info "MinIO is already installed and running"
            log::info "${MSG_CONSOLE_AVAILABLE}: ${MINIO_CONSOLE_URL}"
            return 0
        else
            log::info "Starting existing MinIO container..."
            return minio::docker::start
        fi
    fi
    
    # Pull Docker image
    if ! minio::docker::pull_image; then
        log::error "${MSG_INSTALL_FAILED}: Failed to pull Docker image"
        return 1
    fi
    
    # Create and start container
    if ! minio::docker::create_container; then
        log::error "${MSG_INSTALL_FAILED}: Failed to create container"
        return 1
    fi
    
    # Wait for MinIO to be ready
    if ! minio::common::wait_for_ready; then
        log::error "${MSG_INSTALL_FAILED}: Service failed to start"
        minio::common::show_logs 50
        return 1
    fi
    
    # Initialize default buckets
    if ! minio::buckets::init_defaults; then
        log::warn "Failed to initialize some default buckets"
        # Don't fail installation for this
    fi
    
    # Update Vrooli configuration
    if ! minio::install::update_vrooli_config; then
        log::warn "Failed to update Vrooli configuration"
        log::info "You may need to manually add MinIO to ~/.vrooli/service.json"
    fi
    
    # Test functionality
    if minio::buckets::test_upload; then
        log::debug "Upload test passed"
    else
        log::warn "Upload test failed - MinIO may need additional configuration"
    fi
    
    # Show final status
    log::success "${MSG_INSTALL_SUCCESS}"
    log::info ""
    log::info "${MSG_CONSOLE_AVAILABLE}: ${MINIO_CONSOLE_URL}"
    log::info "${MSG_HELP_CREDENTIALS}"
    log::info ""
    
    # Show credentials if generated
    if [[ -f "${MINIO_CONFIG_DIR}/credentials" ]]; then
        minio::status::show_credentials
    fi
    
    # Auto-install CLI if available
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-auto-install.sh" 2>/dev/null || true
    resource_cli::auto_install "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)" || true
    
    return 0
}

#######################################
# Pre-installation checks
# Returns: 0 if all pass, 1 if any fail
#######################################
minio::install::pre_checks() {
    log::info "Running pre-installation checks..."
    
    # Check Docker
    log::info "${MSG_CHECKING_DOCKER}"
    if ! minio::docker::check_docker; then
        log::error "${MSG_DOCKER_NOT_FOUND}"
        return 1
    fi
    
    # Check ports
    log::info "${MSG_CHECKING_PORTS}"
    if ! minio::common::check_ports; then
        log::error "${MSG_PORT_IN_USE}"
        log::info "Please free up ports ${MINIO_PORT} and ${MINIO_CONSOLE_PORT} or set custom ports:"
        log::info "  MINIO_CUSTOM_PORT=9100 MINIO_CUSTOM_CONSOLE_PORT=9101 $0 --action install"
        return 1
    fi
    
    # Check disk space
    if ! minio::common::check_disk_space; then
        return 1
    fi
    
    log::debug "All pre-installation checks passed"
    return 0
}

#######################################
# Update Vrooli resource configuration
# Returns: 0 on success, 1 on failure
#######################################
minio::install::update_vrooli_config() {
    local config_file
    config_file="$(secrets::get_project_config_file)"
    local config_dir=$(dirname "$config_file")
    
    # Create directory if it doesn't exist
    if [[ ! -d "$config_dir" ]]; then
        mkdir -p "$config_dir"
    fi
    
    # Create or update configuration
    if resources::update_config "storage.minio" '{
        "enabled": true,
        "endpoint": "localhost:'"${MINIO_PORT}"'",
        "useSSL": false,
        "accessKey": "${MINIO_ACCESS_KEY}",
        "secretKey": "${MINIO_SECRET_KEY}",
        "region": "'"${MINIO_REGION}"'",
        "buckets": {
            "userUploads": "vrooli-user-uploads",
            "agentArtifacts": "vrooli-agent-artifacts",
            "modelCache": "vrooli-model-cache",
            "tempStorage": "vrooli-temp-storage"
        },
        "healthCheck": {
            "endpoint": "/minio/health/live",
            "intervalMs": 300000,
            "timeoutMs": 5000
        }
    }'; then
        log::debug "Vrooli configuration updated successfully"
        return 0
    else
        log::error "Failed to update Vrooli configuration"
        return 1
    fi
}

#######################################
# Uninstall MinIO
# Arguments:
#   $1 - Keep data (optional, default: true)
# Returns: 0 on success, 1 on failure
#######################################
minio::uninstall() {
    local keep_data=${1:-true}
    
    log::info "Uninstalling MinIO..."
    
    # Stop and remove container
    if minio::common::container_exists; then
        if ! minio::docker::remove true; then
            log::error "Failed to remove MinIO container"
            return 1
        fi
    fi
    
    # Remove from Vrooli configuration
    if resources::remove_from_config "storage.minio"; then
        log::debug "Removed from Vrooli configuration"
    fi
    
    # Handle data directories
    if [[ "$keep_data" == "false" ]]; then
        log::warn "Removing all MinIO data..."
        if [[ -d "${MINIO_DATA_DIR}" ]]; then
            trash::safe_remove "${MINIO_DATA_DIR}" --no-confirm
            log::info "Removed data directory: ${MINIO_DATA_DIR}"
        fi
        if [[ -d "${MINIO_CONFIG_DIR}" ]]; then
            trash::safe_remove "${MINIO_CONFIG_DIR}" --no-confirm
            log::info "Removed config directory: ${MINIO_CONFIG_DIR}"
        fi
    else
        log::info "Data preserved at: ${MINIO_DATA_DIR}"
        log::info "Config preserved at: ${MINIO_CONFIG_DIR}"
        log::info "To completely remove, run: $0 --action uninstall --remove-data"
    fi
    
    # Clean up Docker resources
    minio::common::cleanup
    
    log::success "${MSG_UNINSTALL_SUCCESS}"
    return 0
}

#######################################
# Reset MinIO credentials
# Returns: 0 on success, 1 on failure
#######################################
minio::install::reset_credentials() {
    log::info "Resetting MinIO credentials..."
    
    if ! minio::common::is_running; then
        log::error "MinIO is not running. Please start it first."
        return 1
    fi
    
    # Generate new credentials
    local new_user="minio$(openssl rand -hex 4)"
    local new_password=$(openssl rand -base64 24 | tr -d "=+/" | cut -c1-20)
    
    # Stop MinIO
    log::info "Stopping MinIO to apply new credentials..."
    minio::docker::stop || return 1
    
    # Update environment variables
    export MINIO_ROOT_USER="$new_user"
    export MINIO_ROOT_PASSWORD="$new_password"
    
    # Save to credentials file
    local creds_file="${MINIO_CONFIG_DIR}/credentials"
    cat > "$creds_file" << EOF
# MinIO Credentials - Reset on $(date)
# Keep this file secure!
export MINIO_ROOT_USER="${MINIO_ROOT_USER}"
export MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD}"
EOF
    chmod 600 "$creds_file"
    
    # Remove old container
    docker rm "$MINIO_CONTAINER_NAME" >/dev/null 2>&1
    
    # Create new container with new credentials
    log::info "Creating new container with updated credentials..."
    if ! minio::docker::create_container; then
        log::error "Failed to create container with new credentials"
        return 1
    fi
    
    # Wait for startup
    if ! minio::common::wait_for_ready; then
        log::error "MinIO failed to start with new credentials"
        return 1
    fi
    
    # Reconfigure buckets
    log::info "Reconfiguring buckets..."
    minio::buckets::init_defaults >/dev/null 2>&1
    
    # Update Vrooli config
    minio::install::update_vrooli_config >/dev/null 2>&1
    
    log::success "Credentials reset successfully!"
    log::info ""
    minio::status::show_credentials
    
    return 0
}

#######################################
# Upgrade MinIO to latest version
# Returns: 0 on success, 1 on failure
#######################################
minio::install::upgrade() {
    log::info "Upgrading MinIO to latest version..."
    
    if ! minio::common::container_exists; then
        log::error "MinIO is not installed. Run install first."
        return 1
    fi
    
    # Pull latest image
    log::info "Pulling latest MinIO image..."
    if ! docker pull "${MINIO_IMAGE}"; then
        log::error "Failed to pull latest image"
        return 1
    fi
    
    # Stop current container
    if minio::common::is_running; then
        log::info "Stopping current MinIO instance..."
        minio::docker::stop || return 1
    fi
    
    # Remove old container (keep data)
    docker rm "$MINIO_CONTAINER_NAME" >/dev/null 2>&1
    
    # Create new container with latest image
    log::info "Creating new container with latest image..."
    if ! minio::docker::create_container; then
        log::error "Failed to create new container"
        return 1
    fi
    
    # Wait for startup
    if ! minio::common::wait_for_ready; then
        log::error "Upgraded MinIO failed to start"
        return 1
    fi
    
    log::success "MinIO upgraded successfully!"
    
    # Show version info
    if minio::docker::exec minio --version 2>/dev/null; then
        :
    fi
    
    return 0
}