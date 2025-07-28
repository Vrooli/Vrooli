#!/usr/bin/env bash
# SearXNG Installation Logic
# Complete installation, uninstallation, and setup procedures

#######################################
# Check installation prerequisites
#######################################
searxng::check_prerequisites() {
    local missing_deps=()
    
    # Check Docker
    if ! command -v docker >/dev/null 2>&1; then
        missing_deps+=("docker")
    fi
    
    # Check Docker daemon
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running or accessible"
        log::info "Please start Docker and ensure current user has Docker access"
        return 1
    fi
    
    # Check curl (for health checks)
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    # Check openssl (for secret generation)
    if ! command -v openssl >/dev/null 2>&1; then
        log::warn "OpenSSL not found - will use alternative for secret generation"
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log::error "Missing required dependencies: ${missing_deps[*]}"
        log::info "Please install the missing dependencies and try again"
        return 1
    fi
    
    return 0
}

#######################################
# Install SearXNG
#######################################
searxng::install() {
    searxng::message "info" "MSG_SEARXNG_SETUP_START"
    
    # Check if already installed
    if searxng::is_installed && [[ "$FORCE" != "yes" ]]; then
        log::warn "SearXNG is already installed"
        log::info "Use --force yes to reinstall"
        return 0
    fi
    
    # Check prerequisites
    if ! searxng::check_prerequisites; then
        return 1
    fi
    
    # Validate configuration
    if ! searxng::validate_config; then
        searxng::message "error" "MSG_SEARXNG_CONFIG_FAILED"
        return 1
    fi
    
    # Setup data directory
    searxng::message "info" "MSG_SEARXNG_SETUP_CONFIG"
    if ! searxng::ensure_data_dir; then
        return 1
    fi
    
    # Generate configuration files
    if ! searxng::generate_config; then
        searxng::message "error" "MSG_SEARXNG_CONFIG_FAILED"
        return 1
    fi
    
    # Pull Docker image
    if ! searxng::pull_image; then
        return 1
    fi
    
    # Create Docker network
    searxng::message "info" "MSG_SEARXNG_SETUP_NETWORK"
    if ! searxng::create_network; then
        searxng::message "error" "MSG_SEARXNG_NETWORK_FAILED"
        return 1
    fi
    
    # Start SearXNG container
    searxng::message "info" "MSG_SEARXNG_SETUP_CONTAINER"
    if ! searxng::start_container; then
        return 1
    fi
    
    # Wait for health check
    searxng::message "info" "MSG_SEARXNG_SETUP_HEALTH"
    if ! searxng::wait_for_health; then
        log::warn "SearXNG may not be fully ready yet, but installation completed"
    fi
    
    # Show success information
    searxng::message "success" "MSG_SEARXNG_INSTALL_SUCCESS"
    echo
    searxng::message "info" "MSG_SEARXNG_ACCESS_INFO"
    searxng::message "info" "MSG_SEARXNG_API_INFO"
    searxng::message "info" "MSG_SEARXNG_STATS_INFO"
    echo
    searxng::message "info" "MSG_SEARXNG_ENGINES_INFO"
    searxng::message "info" "MSG_SEARXNG_SECURITY_INFO"
    searxng::message "info" "MSG_SEARXNG_RATE_LIMIT_INFO"
    
    # Integration information
    echo
    searxng::message "info" "MSG_SEARXNG_N8N_INTEGRATION"
    searxng::message "info" "MSG_SEARXNG_DISCOVERY_INFO"
    
    # Show warning if public access is enabled
    if [[ "$SEARXNG_ENABLE_PUBLIC_ACCESS" == "yes" ]]; then
        echo
        searxng::message "warning" "MSG_SEARXNG_PUBLIC_ACCESS_WARNING"
    fi
    
    return 0
}

#######################################
# Uninstall SearXNG
#######################################
searxng::uninstall() {
    log::info "Uninstalling SearXNG..."
    
    if ! searxng::is_installed; then
        log::info "SearXNG is not installed"
        return 0
    fi
    
    # Confirm uninstallation if not forced or auto-confirmed
    if [[ "$FORCE" != "yes" ]] && [[ "$YES" != "yes" ]]; then
        if ! flow::prompt_yes_no "Are you sure you want to uninstall SearXNG? This will remove all containers and optionally data."; then
            log::info "Uninstallation cancelled"
            return 0
        fi
        
        # Ask about data removal
        local remove_data="no"
        if flow::prompt_yes_no "Do you also want to remove SearXNG data directory ($SEARXNG_DATA_DIR)?"; then
            remove_data="yes"
        fi
    else
        # In force/auto mode, remove data too
        local remove_data="yes"
    fi
    
    # Stop and remove containers
    searxng::cleanup "containers"
    
    # Remove data if requested
    if [[ "$remove_data" == "yes" ]]; then
        searxng::cleanup "all"
    fi
    
    searxng::message "success" "MSG_SEARXNG_UNINSTALL_SUCCESS"
    return 0
}

#######################################
# Upgrade SearXNG
#######################################
searxng::upgrade() {
    log::info "Upgrading SearXNG..."
    
    if ! searxng::is_installed; then
        log::error "SearXNG is not installed. Use install action instead."
        return 1
    fi
    
    # Pull latest image
    if ! searxng::pull_image; then
        return 1
    fi
    
    # Restart container with new image
    if searxng::restart_container; then
        log::success "SearXNG upgraded successfully"
        searxng::show_info
        return 0
    else
        log::error "Failed to upgrade SearXNG"
        return 1
    fi
}

#######################################
# Reset SearXNG installation
#######################################
searxng::reset() {
    log::info "Resetting SearXNG installation..."
    
    # Stop container
    if searxng::is_running; then
        searxng::stop_container
    fi
    
    # Regenerate configuration
    searxng::message "info" "MSG_SEARXNG_SETUP_CONFIG"
    if ! searxng::generate_config; then
        searxng::message "error" "MSG_SEARXNG_CONFIG_FAILED"
        return 1
    fi
    
    # Restart container
    if searxng::start_container; then
        log::success "SearXNG reset successfully"
        return 0
    else
        log::error "Failed to reset SearXNG"
        return 1
    fi
}

#######################################
# Backup SearXNG configuration
#######################################
searxng::backup() {
    local backup_dir="${HOME}/.searxng-backup-$(date +%Y%m%d-%H%M%S)"
    
    if [[ ! -d "$SEARXNG_DATA_DIR" ]]; then
        log::error "SearXNG data directory not found: $SEARXNG_DATA_DIR"
        return 1
    fi
    
    log::info "Creating SearXNG backup..."
    log::info "Backup location: $backup_dir"
    
    if mkdir -p "$backup_dir" && cp -r "$SEARXNG_DATA_DIR"/* "$backup_dir"/; then
        log::success "SearXNG backup created successfully"
        echo "Backup location: $backup_dir"
        return 0
    else
        log::error "Failed to create SearXNG backup"
        return 1
    fi
}

#######################################
# Restore SearXNG configuration
# Arguments:
#   $1 - backup directory path
#######################################
searxng::restore() {
    local backup_dir="$1"
    
    if [[ -z "$backup_dir" ]]; then
        log::error "Backup directory path is required"
        return 1
    fi
    
    if [[ ! -d "$backup_dir" ]]; then
        log::error "Backup directory not found: $backup_dir"
        return 1
    fi
    
    log::info "Restoring SearXNG from backup..."
    log::info "Backup source: $backup_dir"
    
    # Stop container
    if searxng::is_running; then
        searxng::stop_container
    fi
    
    # Ensure data directory exists
    searxng::ensure_data_dir
    
    # Restore configuration
    if cp -r "$backup_dir"/* "$SEARXNG_DATA_DIR"/; then
        log::success "SearXNG configuration restored"
        
        # Restart container
        if searxng::start_container; then
            log::success "SearXNG restore completed successfully"
            return 0
        else
            log::error "SearXNG restored but failed to start"
            return 1
        fi
    else
        log::error "Failed to restore SearXNG configuration"
        return 1
    fi
}

#######################################
# Update SearXNG to use Docker Compose
#######################################
searxng::migrate_to_compose() {
    log::info "Migrating SearXNG to Docker Compose setup..."
    
    # Stop existing container
    if searxng::is_running; then
        searxng::stop_container
    fi
    
    # Remove standalone container
    searxng::remove_container
    
    # Start with Docker Compose
    if searxng::compose_up; then
        log::success "SearXNG migrated to Docker Compose successfully"
        return 0
    else
        log::error "Failed to migrate SearXNG to Docker Compose"
        return 1
    fi
}