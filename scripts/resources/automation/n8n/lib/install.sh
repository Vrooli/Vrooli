#!/usr/bin/env bash
# n8n Installation Functions - Minimal wrapper using init framework

# Source core and frameworks
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/init-framework.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/docker-utils.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/core.sh"

#######################################
# Install n8n using init framework
#######################################
n8n::install() {
    log::header "🚀 Installing n8n"
    
    # Check Docker
    if ! docker::check_daemon; then
        return 1
    fi
    
    # Check if already installed
    if docker::container_exists "$N8N_CONTAINER_NAME"; then
        log::warn "n8n is already installed"
        log::info "Use --action status to check current state"
        return 0
    fi
    
    # Get password if using basic auth
    local auth_password=""
    if [[ "$BASIC_AUTH" == "yes" ]]; then
        auth_password="${AUTH_PASSWORD:-$(n8n::generate_password)}"
    fi
    
    # Install database if needed
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        n8n::install_postgres
    fi
    
    # Use init framework for installation
    local init_config
    init_config=$(n8n::get_init_config "" "$auth_password")
    
    if init::setup_resource "$init_config"; then
        log::success "✅ n8n installed successfully"
        
        # Show access information
        echo
        log::info "Access n8n at: $N8N_BASE_URL"
        
        if [[ "$BASIC_AUTH" == "yes" ]]; then
            log::info "Login credentials:"
            log::info "  Username: $AUTH_USERNAME"
            log::info "  Password: $auth_password"
        fi
        
        return 0
    else
        log::error "Failed to install n8n"
        return 1
    fi
}

#######################################
# Uninstall n8n
#######################################
n8n::uninstall() {
    log::header "🗑️  Uninstalling n8n"
    
    # Confirm action
    if [[ "${FORCE:-no}" != "yes" ]]; then
        log::warn "This will remove n8n and optionally its data"
        read -p "Are you sure? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    # Stop containers
    if docker::is_running "$N8N_CONTAINER_NAME"; then
        log::info "Stopping n8n..."
        docker stop "$N8N_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    if [[ "$DATABASE_TYPE" == "postgres" ]] && docker::is_running "$N8N_DB_CONTAINER_NAME"; then
        log::info "Stopping PostgreSQL..."
        docker stop "$N8N_DB_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    # Remove containers
    if docker::container_exists "$N8N_CONTAINER_NAME"; then
        log::info "Removing n8n container..."
        docker::remove_container "$N8N_CONTAINER_NAME" "true"
    fi
    
    if [[ "$DATABASE_TYPE" == "postgres" ]] && docker::container_exists "$N8N_DB_CONTAINER_NAME"; then
        log::info "Removing PostgreSQL container..."
        docker::remove_container "$N8N_DB_CONTAINER_NAME" "true"
    fi
    
    # Remove data if requested
    if [[ "${REMOVE_DATA:-no}" == "yes" ]]; then
        log::warn "Removing n8n data directory..."
        rm -rf "$N8N_DATA_DIR"
    else
        log::info "Data preserved at: $N8N_DATA_DIR"
    fi
    
    log::success "✅ n8n uninstalled"
}

#######################################
# Generate secure password
#######################################
n8n::generate_password() {
    openssl rand -base64 16 2>/dev/null || date +%s | sha256sum | head -c 16
}

#######################################
# Build custom n8n image
#######################################
n8n::build_custom_image() {
    log::info "Building custom n8n image..."
    
    local docker_dir="$(dirname "$N8N_LIB_DIR")/docker"
    
    if [[ ! -f "$docker_dir/Dockerfile" ]]; then
        log::error "Dockerfile not found at: $docker_dir/Dockerfile"
        return 1
    fi
    
    if docker build -t "$N8N_CUSTOM_IMAGE" "$docker_dir"; then
        log::success "Custom image built: $N8N_CUSTOM_IMAGE"
        return 0
    else
        log::error "Failed to build custom image"
        return 1
    fi
}