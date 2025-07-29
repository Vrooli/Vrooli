#!/usr/bin/env bash
# QuestDB Installation Functions

#######################################
# Run QuestDB installation
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::install::run() {
    echo_header "Installing QuestDB Time-Series Database"
    
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
        echo_warning "Failed to initialize default tables, but QuestDB is running"
    fi
    
    # Show success message
    questdb::install::show_success
    
    return 0
}

#######################################
# Check installation prerequisites
# Returns:
#   0 if all prerequisites met, 1 otherwise
#######################################
questdb::install::check_prerequisites() {
    echo_info "${QUESTDB_INSTALL_MESSAGES["checking_docker"]}"
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        echo_error "${QUESTDB_ERROR_MESSAGES["docker_not_found"]}"
        return 1
    fi
    
    # Check Docker daemon
    if ! docker info &> /dev/null; then
        echo_error "${QUESTDB_ERROR_MESSAGES["docker_not_found"]}"
        echo_info "Make sure Docker daemon is running"
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
        echo_warning "QuestDB is already running"
        if ! args::prompt_yes_no "Reinstall QuestDB?" "n"; then
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
    echo_info "${QUESTDB_INSTALL_MESSAGES["pulling_image"]}"
    
    if ! docker pull "${QUESTDB_IMAGE}"; then
        echo_error "Failed to pull QuestDB image"
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
    echo_info "${QUESTDB_INSTALL_MESSAGES["initializing"]}"
    
    # Wait a bit for QuestDB to fully initialize
    sleep 5
    
    # Create default tables
    local schemas_dir="${SCRIPT_DIR}/schemas"
    local schema_files=(
        "system_metrics.sql"
        "ai_metrics.sql"
        "resource_monitoring.sql"
        "workflow_analytics.sql"
    )
    
    for schema_file in "${schema_files[@]}"; do
        if [[ -f "${schemas_dir}/${schema_file}" ]]; then
            echo_info "Creating table from ${schema_file}..."
            local sql
            sql=$(<"${schemas_dir}/${schema_file}")
            
            if ! questdb::api::query "$sql" 1 &>/dev/null; then
                echo_warning "Failed to create table from ${schema_file}"
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
    echo_success "${QUESTDB_INSTALL_MESSAGES["success"]}"
    echo ""
    echo "🚀 QuestDB is ready to use!"
    echo ""
    echo "${QUESTDB_INFO_MESSAGES["web_console"]}"
    echo "${QUESTDB_INFO_MESSAGES["pg_connection"]}"
    echo "${QUESTDB_INFO_MESSAGES["performance"]}"
    echo ""
    echo "📚 Quick Start:"
    echo "  - View status:    ./manage.sh --action status"
    echo "  - Open console:   ./manage.sh --action console"
    echo "  - Execute query:  ./manage.sh --action query --query 'SELECT * FROM tables()'"
    echo "  - View logs:      ./manage.sh --action logs"
    echo ""
}

#######################################
# Upgrade QuestDB to latest version
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::install::upgrade() {
    echo_header "Upgrading QuestDB"
    
    # Get current version
    local current_version
    current_version=$(questdb::get_version)
    echo_info "Current version: ${current_version:-unknown}"
    
    # Pull latest image
    echo_info "Pulling latest QuestDB image..."
    if ! docker pull "${QUESTDB_IMAGE}"; then
        echo_error "Failed to pull latest image"
        return 1
    fi
    
    # Backup data directory
    echo_info "Backing up data directory..."
    local backup_dir="${QUESTDB_DATA_DIR}.backup.$(date +%Y%m%d_%H%M%S)"
    if ! cp -r "${QUESTDB_DATA_DIR}" "$backup_dir"; then
        echo_error "Failed to backup data directory"
        return 1
    fi
    echo_info "Backup created: $backup_dir"
    
    # Stop current container
    questdb::docker::stop
    
    # Remove old container
    docker rm "${QUESTDB_CONTAINER_NAME}" &>/dev/null || true
    
    # Start with new image
    if ! questdb::docker::start; then
        echo_error "Failed to start upgraded QuestDB"
        echo_info "Restoring from backup..."
        rm -rf "${QUESTDB_DATA_DIR}"
        mv "$backup_dir" "${QUESTDB_DATA_DIR}"
        return 1
    fi
    
    # Verify upgrade
    local new_version
    new_version=$(questdb::get_version)
    echo_success "QuestDB upgraded successfully"
    echo_info "New version: ${new_version:-unknown}"
    
    # Cleanup old backup after successful upgrade
    if args::prompt_yes_no "Remove backup directory?" "y"; then
        rm -rf "$backup_dir"
    fi
    
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