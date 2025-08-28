#!/usr/bin/env bash
# QuestDB Installation Functions

# Source required utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
QUESTDB_LIB_DIR="${APP_ROOT}/resources/questdb/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/docker.sh"

#######################################
# Run QuestDB installation
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::install::run() {
    log::header "Installing QuestDB Time-Series Database"
    
    # Check prerequisites
    if ! questdb::install::check_prerequisites; then
        return 1
    fi
    
    # Create directories
    if ! questdb::create_dirs; then
        return 1
    fi
    
    # Pull Docker image
    if ! questdb::install::pull_image; then
        return 1
    fi
    
    # Start QuestDB
    if ! questdb::docker::start; then
        return 1
    fi
    
    # Initialize default tables
    if ! questdb::install::init_tables; then
        log::warning "Failed to initialize default tables, but QuestDB is running"
    fi
    
    # Show success message
    questdb::install::show_success
    
    # Auto-install CLI if available
    # shellcheck disable=SC1091
    "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "${APP_ROOT}/resources/questdb" 2>/dev/null || true
    
    return 0
}

#######################################
# Check installation prerequisites
# Returns:
#   0 if all prerequisites met, 1 otherwise
#######################################
questdb::install::check_prerequisites() {
    log::info "${QUESTDB_INSTALL_MESSAGES["checking_docker"]}"
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log::error "${QUESTDB_ERROR_MESSAGES["docker_not_found"]}"
        return 1
    fi
    
    # Check Docker daemon
    if ! docker info &> /dev/null; then
        log::error "${QUESTDB_ERROR_MESSAGES["docker_not_found"]}"
        log::info "Make sure Docker daemon is running"
        return 1
    fi
    
    # Check ports
    if ! questdb::check_ports; then
        return 1
    fi
    
    # Check disk space (5GB minimum)
    if ! questdb::check_disk_space 5; then
        return 1
    fi
    
    # Check if already installed
    if questdb::docker::is_running; then
        log::warning "QuestDB is already running"
        if ! flow::confirm "Reinstall QuestDB?"; then
            return 1
        fi
        questdb::docker::stop
    fi
    
    return 0
}

#######################################
# Pull QuestDB Docker image
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::install::pull_image() {
    log::info "${QUESTDB_INSTALL_MESSAGES["pulling_image"]}"
    
    if ! docker pull "${QUESTDB_IMAGE}"; then
        log::error "Failed to pull QuestDB image"
        return 1
    fi
    
    return 0
}

#######################################
# Initialize default tables
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::install::init_tables() {
    log::info "${QUESTDB_INSTALL_MESSAGES["initializing"]}"
    
    # Wait a bit for QuestDB to fully initialize
    sleep 5
    
    # Create default tables
    local schemas_dir="${QUESTDB_LIB_DIR}/schemas"
    local schema_files=(
        "system_metrics.sql"
        "ai_metrics.sql"
        "resource_monitoring.sql"
        "workflow_analytics.sql"
    )
    
    for schema_file in "${schema_files[@]}"; do
        if [[ -f "${schemas_dir}/${schema_file}" ]]; then
            log::info "Creating table from ${schema_file}..."
            local sql
            sql=$(<"${schemas_dir}/${schema_file}")
            
            if ! questdb::api::query "$sql" 1 &>/dev/null; then
                log::warning "Failed to create table from ${schema_file}"
            fi
        fi
    done
    
    return 0
}

#######################################
# Show installation success message
#######################################
questdb::install::show_success() {
    echo ""
    log::success "${QUESTDB_INSTALL_MESSAGES["success"]}"
    echo ""
    echo "🚀 QuestDB is ready to use!"
    echo ""
    echo "${QUESTDB_INFO_MESSAGES["web_console"]}"
    echo "${QUESTDB_INFO_MESSAGES["pg_connection"]}"
    echo "${QUESTDB_INFO_MESSAGES["performance"]}"
    echo ""
    echo "📚 Quick Start:"
    echo "  - View status:    resource-questdb status"
    echo "  - Open console:   resource-questdb content execute --console"
    echo "  - Execute query:  resource-questdb content execute 'SELECT * FROM tables()'"
    echo "  - View logs:      resource-questdb logs"
    echo ""
}

#######################################
# Upgrade QuestDB to latest version
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::install::upgrade() {
    log::header "Upgrading QuestDB"
    
    # Get current version
    local current_version
    current_version=$(questdb::get_version)
    log::info "Current version: ${current_version:-unknown}"
    
    # Pull latest image
    log::info "Pulling latest QuestDB image..."
    if ! docker pull "${QUESTDB_IMAGE}"; then
        log::error "Failed to pull latest image"
        return 1
    fi
    
    # Backup data directory
    log::info "Backing up data directory..."
    local backup_dir="${QUESTDB_DATA_DIR}.backup.$(date +%Y%m%d_%H%M%S)"
    if ! cp -r "${QUESTDB_DATA_DIR}" "$backup_dir"; then
        log::error "Failed to backup data directory"
        return 1
    fi
    log::info "Backup created: $backup_dir"
    
    # Stop current container
    questdb::docker::stop
    
    # Remove old container
    docker rm "${QUESTDB_CONTAINER_NAME}" &>/dev/null || true
    
    # Start with new image
    if ! questdb::docker::start; then
        log::error "Failed to start upgraded QuestDB"
        log::info "Restoring from backup..."
        trash::safe_remove "${QUESTDB_DATA_DIR}" --production
        mv "$backup_dir" "${QUESTDB_DATA_DIR}"
        return 1
    fi
    
    # Verify upgrade
    local new_version
    new_version=$(questdb::get_version)
    log::success "QuestDB upgraded successfully"
    log::info "New version: ${new_version:-unknown}"
    
    # Cleanup old backup after successful upgrade
    if flow::confirm "Remove backup directory?"; then
        trash::safe_remove "$backup_dir" --temp
    fi
    
    return 0
}

#######################################
# Uninstall QuestDB completely
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::install::uninstall() {
    log::header "Uninstalling QuestDB"
    
    if ! flow::confirm "Are you sure you want to uninstall QuestDB? This will remove all data."; then
        log::info "Uninstall cancelled"
        return 0
    fi
    
    # Stop container if running
    questdb::docker::stop || true
    
    # Remove container
    if docker ps -a --format '{{.Names}}' | grep -q "^${QUESTDB_CONTAINER_NAME}$"; then
        log::info "Removing QuestDB container..."
        docker rm -f "${QUESTDB_CONTAINER_NAME}" || true
    fi
    
    # Remove network if it exists
    if docker network ls --format '{{.Name}}' | grep -q "^${QUESTDB_NETWORK_NAME}$"; then
        log::info "Removing Docker network..."
        docker network rm "${QUESTDB_NETWORK_NAME}" || true
    fi
    
    # Remove data directories
    if flow::confirm "Remove all QuestDB data directories?"; then
        log::info "Removing data directories..."
        trash::safe_remove "${QUESTDB_DATA_DIR}" --no-confirm 2>/dev/null || true
        trash::safe_remove "${QUESTDB_CONFIG_DIR}" --no-confirm 2>/dev/null || true
        trash::safe_remove "${QUESTDB_LOG_DIR}" --no-confirm 2>/dev/null || true
        trash::safe_remove "${HOME}/.questdb" --no-confirm 2>/dev/null || true
    fi
    
    log::success "QuestDB uninstalled successfully"
    return 0
}

#######################################
# Export install functions
#######################################
export -f questdb::install::run
export -f questdb::install::check_prerequisites
export -f questdb::install::pull_image
export -f questdb::install::init_tables
export -f questdb::install::show_success
export -f questdb::install::upgrade
export -f questdb::install::uninstall