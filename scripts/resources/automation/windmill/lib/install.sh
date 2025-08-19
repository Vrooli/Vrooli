#!/usr/bin/env bash
# Windmill Installation Functions
# Complete installation workflow for Windmill platform

# Only source if not already loaded via CLI
if [[ -z "${WINDMILL_LIB_DIR:-}" ]]; then
    WINDMILL_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    # shellcheck disable=SC1091
    source "${WINDMILL_LIB_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
    
    # Source necessary windmill lib files
    # shellcheck disable=SC1091
    source "${WINDMILL_LIB_DIR}/state.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${WINDMILL_LIB_DIR}/common.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${WINDMILL_LIB_DIR}/docker.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${WINDMILL_LIB_DIR}/database.sh" 2>/dev/null || true
fi

#######################################
# Main installation function
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::install() {
    log::header "🌪️  Installing Windmill Workflow Automation Platform"
    
    # Initialize FORCE variable if not set
    FORCE="${FORCE:-no}"
    
    # WINDMILL_PROJECT_NAME is already defined in common.sh
    
    # Initialize state management system
    windmill::state_init "$WINDMILL_PROJECT_NAME"
    
    # Acquire lock to prevent concurrent operations
    windmill::acquire_lock
    
    # Get or generate database password from state
    WINDMILL_DB_PASSWORD=$(windmill::get_database_password "$WINDMILL_PROJECT_NAME" "$FORCE")
    if [[ -z "$WINDMILL_DB_PASSWORD" ]]; then
        log::error "Failed to determine database password"
        windmill::release_lock
        return 1
    fi
    export WINDMILL_DB_PASSWORD
    
    # Check if already installed and running
    if windmill::is_installed && windmill::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Windmill is already installed and running"
        log::info "Access it at: $WINDMILL_BASE_URL"
        log::info "Use --force yes to reinstall"
        windmill::release_lock
        return 0
    fi
    
    # Pre-installation validation
    if ! windmill::validate_installation_requirements; then
        windmill::release_lock
        return 1
    fi
    
    # Create necessary directories
    if ! windmill::create_directories; then
        windmill::release_lock
        return 1
    fi
    
    # Generate configuration
    if ! windmill::generate_env_file; then
        windmill::release_lock
        return 1
    fi
    
    # Deploy services
    if ! windmill::deploy; then
        log::error "Failed to deploy Windmill services"
        windmill::release_lock
        return 1
    fi
    
    # Post-installation configuration
    if ! windmill::post_install_setup; then
        log::warn "Post-installation setup had issues, but Windmill should still work"
    fi
    
    # Update state with installation info
    windmill::update_state_config "last_installed" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    windmill::update_state_config "installation_status" "completed"
    
    # Release lock
    windmill::release_lock
    
    # Show success message
    windmill::show_success_message "${WINDMILL_SUPERADMIN_EMAIL:-admin@windmill.dev}" "${WINDMILL_SUPERADMIN_PASSWORD:-changeme}"
    
    # Auto-install CLI if available
    # shellcheck disable=SC1091
    "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)" 2>/dev/null || true
    
    return 0
}

#######################################
# Validate all installation requirements
# Returns: 0 if valid, 1 otherwise
#######################################
windmill::validate_installation_requirements() {
    log::info "Validating installation requirements..."
    
    # Check Docker requirements
    if ! windmill::check_docker_requirements; then
        return 1
    fi
    
    # Check system resources
    if ! windmill::check_system_requirements; then
        return 1
    fi
    
    # Validate port availability
    if ! windmill::validate_port "$WINDMILL_SERVER_PORT"; then
        return 1
    fi
    
    # Validate configuration
    if ! windmill::validate_config; then
        return 1
    fi
    
    # Check for external database connectivity if specified
    if [[ "${EXTERNAL_DB:-no}" == "yes" ]]; then
        if ! windmill::validate_external_database; then
            return 1
        fi
    fi
    
    # Security checks
    if ! windmill::validate_security_config; then
        return 1
    fi
    
    log::success "All installation requirements validated"
    return 0
}

#######################################
# Validate external database connection
# Returns: 0 if valid, 1 otherwise
#######################################
windmill::validate_external_database() {
    if [[ -z "${DB_URL:-}" ]]; then
        log::error "External database URL is required when using external database"
        log::info "Set --db-url 'postgresql://user:pass@host:port/database'"
        return 1
    fi
    
    log::info "Validating external database connection..."
    
    # Extract components from database URL
    local db_host db_port db_name db_user
    if [[ "$DB_URL" =~ postgresql://([^:]+):([^@]+)@([^:]+):([0-9]+)/(.+) ]]; then
        db_user="${BASH_REMATCH[1]}"
        db_host="${BASH_REMATCH[3]}"
        db_port="${BASH_REMATCH[4]}"
        db_name="${BASH_REMATCH[5]}"
    else
        log::error "Invalid database URL format"
        log::info "Expected format: postgresql://user:password@host:port/database"
        return 1
    fi
    
    # Test database connectivity using Docker
    log::info "Testing database connectivity to $db_host:$db_port..."
    
    if ! docker run --rm postgres:16 pg_isready -h "$db_host" -p "$db_port" -U "$db_user" >/dev/null 2>&1; then
        log::error "Cannot connect to external database"
        log::info "Please verify:"
        log::info "  • Database server is running"
        log::info "  • Host and port are correct: $db_host:$db_port"
        log::info "  • User has access: $db_user"
        log::info "  • Database exists: $db_name"
        log::info "  • Network connectivity is available"
        return 1
    fi
    
    log::success "External database connection validated"
    return 0
}

#######################################
# Validate security configuration
# Returns: 0 if valid, 1 otherwise
#######################################
windmill::validate_security_config() {
    log::info "Validating security configuration..."
    
    local warnings=()
    
    # Check for default password
    if [[ "${WINDMILL_SUPERADMIN_PASSWORD:-changeme}" == "changeme" ]]; then
        warnings+=("Using default superadmin password - change this immediately after installation!")
    fi
    
    # Check JWT secret strength
    if [[ ${#WINDMILL_JWT_SECRET} -lt 32 ]]; then
        warnings+=("JWT secret is too short (minimum 32 characters recommended)")
    fi
    
    # Check if JWT secret looks like default
    if [[ "$WINDMILL_JWT_SECRET" == *"fallback-secret"* ]]; then
        warnings+=("Using fallback JWT secret - generate a secure random secret")
    fi
    
    # Check superadmin email format
    if [[ "${WINDMILL_SUPERADMIN_EMAIL:-admin@windmill.dev}" == "admin@windmill.dev" ]]; then
        warnings+=("Using default superadmin email - consider using a real email address")
    fi
    
    if [[ ${#warnings[@]} -gt 0 ]]; then
        log::warn "Security configuration warnings:"
        for warning in "${warnings[@]}"; do
            log::warn "  • $warning"
        done
        
        if [[ "$FORCE" != "yes" ]] && ! windmill::prompt_password_change; then
            log::info "Installation cancelled. Use --force yes to proceed anyway."
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Post-installation setup and configuration
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::post_install_setup() {
    log::info "Performing post-installation setup..."
    
    # Wait for API to be ready
    if ! windmill::wait_for_health; then
        log::error "Windmill API is not responding"
        return 1
    fi
    
    # Update Vrooli configuration
    if ! windmill::update_vrooli_config; then
        log::warn "Failed to update Vrooli configuration"
    fi
    
    # Initialize default workspace (if needed)
    if ! windmill::initialize_default_workspace; then
        log::warn "Failed to initialize default workspace"
    fi
    
    # Create example scripts (optional)
    if ! windmill::create_example_scripts; then
        log::warn "Failed to create example scripts"
    fi
    
    return 0
}

#######################################
# Initialize default workspace
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::initialize_default_workspace() {
    log::info "Initializing default workspace..."
    
    # This would typically involve API calls to create workspace
    # For now, just log that it should be done manually
    log::info "Default workspace should be created via web interface"
    log::info "Visit $WINDMILL_BASE_URL and create your first workspace"
    
    return 0
}

#######################################
# Create example scripts for new users
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::create_example_scripts() {
    log::info "Example scripts will be available in the examples/ directory"
    log::info "Import them via the Windmill web interface after creating a workspace"
    
    return 0
}

#######################################
# Uninstall Windmill completely
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::uninstall() {
    log::header "🗑️  Uninstalling Windmill"
    
    # Confirm uninstallation
    if ! windmill::prompt_uninstall_confirmation; then
        return 0
    fi
    
    local backup_created=false
    
    # Create backup before uninstalling
    if windmill::is_running; then
        log::info "Creating backup before uninstalling..."
        
        local backup_file
        if backup_file=$(windmill::backup_data "$WINDMILL_BACKUP_DIR"); then
            log::success "Backup created: $backup_file"
            backup_created=true
        else
            log::warn "Failed to create backup"
        fi
    fi
    
    # Stop and remove all services
    if ! windmill::cleanup true; then
        log::error "Failed to clean up all Windmill resources"
        return 1
    fi
    
    # Remove volumes
    log::info "Removing Docker volumes..."
    
    local volumes
    volumes=$(docker volume ls --filter "name=${WINDMILL_PROJECT_NAME}" --format "{{.Name}}" 2>/dev/null || true)
    if [[ -n "$volumes" ]]; then
        for volume in $volumes; do
            docker volume rm "$volume" 2>/dev/null || true
        done
    fi
    
    # Remove configuration files (with confirmation)
    if [[ -f "$WINDMILL_ENV_FILE" ]]; then
        log::info "Removing configuration files..."
        trash::safe_remove "$WINDMILL_ENV_FILE" --temp
    fi
    
    # Remove from Vrooli configuration
    windmill::remove_vrooli_config
    
    # Show completion message
    log::success "✅ Windmill uninstalled successfully"
    
    if [[ "$backup_created" == "true" ]]; then
        echo
        log::info "💾 Your data has been backed up to: $WINDMILL_BACKUP_DIR"
        log::info "To restore later, use: $0 --action restore --backup-path <backup-file>"
    fi
    
    echo
    log::info "To reinstall Windmill:"
    log::info "  $0 --action install"
    
    return 0
}

#######################################
# Start Windmill services
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::start() {
    # Initialize state management system
    windmill::state_init "$WINDMILL_PROJECT_NAME"
    
    # Acquire lock to prevent concurrent operations
    windmill::acquire_lock
    
    # Get database password from state
    WINDMILL_DB_PASSWORD=$(windmill::get_database_password "$WINDMILL_PROJECT_NAME" "false")
    if [[ -z "$WINDMILL_DB_PASSWORD" ]]; then
        log::error "Failed to determine database password"
        windmill::release_lock
        return 1
    fi
    export WINDMILL_DB_PASSWORD
    
    if ! windmill::is_installed; then
        log::error "Windmill is not installed"
        log::info "Install it first with: $0 --action install"
        windmill::release_lock
        return 1
    fi
    
    if windmill::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Windmill is already running"
        log::info "Access it at: $WINDMILL_BASE_URL"
        windmill::release_lock
        return 0
    fi
    
    log::info "Starting Windmill services..."
    
    # Run preflight checks before starting
    if ! windmill::preflight_checks; then
        log::error "Preflight checks failed"
        log::info "Run with WINDMILL_AUTO_RECOVER=true to attempt automatic recovery"
        windmill::release_lock
        return 1
    fi
    
    # Update .env file with password from state
    if [[ -f "$WINDMILL_ENV_FILE" ]]; then
        sed -i "s|^WINDMILL_DB_PASSWORD=.*|WINDMILL_DB_PASSWORD=${WINDMILL_DB_PASSWORD}|" "$WINDMILL_ENV_FILE"
    fi
    
    if ! windmill::compose_up; then
        windmill::release_lock
        return 1
    fi
    
    if ! windmill::wait_for_health; then
        log::error "Windmill failed to start properly"
        windmill::release_lock
        return 1
    fi
    
    # Update state with last start time
    windmill::update_state_config "last_started" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    
    # Release lock
    windmill::release_lock
    
    log::success "✅ Windmill started successfully"
    log::info "Access it at: $WINDMILL_BASE_URL"
    
    return 0
}

#######################################
# Stop Windmill services
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::stop() {
    if ! windmill::is_installed; then
        log::error "Windmill is not installed"
        return 1
    fi
    
    if ! windmill::is_running; then
        log::info "Windmill is not running"
        return 0
    fi
    
    log::info "Stopping Windmill services..."
    
    if ! windmill::compose_down; then
        return 1
    fi
    
    log::success "✅ Windmill stopped successfully"
    return 0
}

#######################################
# Restart Windmill services
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::restart() {
    if ! windmill::is_installed; then
        log::error "Windmill is not installed"
        log::info "Install it first with: $0 --action install"
        return 1
    fi
    
    log::info "Restarting Windmill services..."
    
    # Stop services
    if windmill::is_running; then
        if ! windmill::compose_down; then
            log::warn "Failed to gracefully stop services, forcing restart..."
        fi
    fi
    
    # Start services
    if ! windmill::compose_up; then
        return 1
    fi
    
    if ! windmill::wait_for_health; then
        log::error "Windmill failed to restart properly"
        return 1
    fi
    
    log::success "✅ Windmill restarted successfully"
    log::info "Access it at: $WINDMILL_BASE_URL"
    
    return 0
}

#######################################
# Backup Windmill data
# Arguments:
#   $1 - Backup path (optional)
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::backup() {
    local backup_path="${1:-$WINDMILL_BACKUP_DIR}"
    
    if ! windmill::is_running; then
        log::error "Windmill is not running - cannot create backup"
        log::info "Start Windmill first with: $0 --action start"
        return 1
    fi
    
    log::header "💾 Creating Windmill Backup"
    
    local backup_file
    if backup_file=$(windmill::backup_data "$backup_path"); then
        log::success "✅ Backup completed successfully"
        log::info "Backup file: $backup_file"
        
        # Show backup information
        echo
        log::info "📋 Backup Information:"
        log::info "  File: $backup_file"
        log::info "  Size: $(du -h "$backup_file" | cut -f1)"
        log::info "  Created: $(date)"
        
        echo
        log::info "To restore this backup:"
        log::info "  $0 --action restore --backup-path \"$backup_file\""
        
        return 0
    else
        log::error "❌ Backup failed"
        return 1
    fi
}

#######################################
# Restore Windmill data from backup
# Arguments:
#   $1 - Backup file path
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::restore() {
    local backup_file="$1"
    
    if [[ -z "$backup_file" ]]; then
        log::error "Backup file path is required"
        log::info "Usage: $0 --action restore --backup-path /path/to/backup.tar.gz"
        return 1
    fi
    
    if [[ ! -f "$backup_file" ]]; then
        log::error "Backup file not found: $backup_file"
        return 1
    fi
    
    log::header "🔄 Restoring Windmill from Backup"
    log::warn "This will overwrite all current Windmill data!"
    
    if ! flow::is_yes "$YES"; then
        read -p "Continue with restore? [y/N]: " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Restore cancelled"
            return 0
        fi
    fi
    
    if windmill::restore_data "$backup_file"; then
        log::success "✅ Restore completed successfully"
        log::info "Access Windmill at: $WINDMILL_BASE_URL"
        return 0
    else
        log::error "❌ Restore failed"
        return 1
    fi
}