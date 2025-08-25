#!/bin/bash

# BTCPay Server Installation Functions

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BTCPAY_INSTALL_DIR="${APP_ROOT}/resources/btcpay/lib"

# Source common functions
source "${BTCPAY_INSTALL_DIR}/common.sh"

# Main install function
btcpay::install() {
    log::header "Installing BTCPay Server"
    
    # Check if already installed
    if btcpay::is_installed; then
        log::info "BTCPay Server is already installed"
        local version=$(docker image inspect "${BTCPAY_IMAGE}" --format '{{index .RepoDigests 0}}' 2>/dev/null | cut -d@ -f2 | cut -c1-12)
        log::success "Version: ${version:-unknown}"
        return 0
    fi
    
    # Create data directories
    log::info "Creating data directories..."
    mkdir -p "${BTCPAY_CONFIG_DIR}"
    mkdir -p "${BTCPAY_POSTGRES_DATA}"
    mkdir -p "${BTCPAY_LOGS_DIR}"
    
    # Create Docker network
    log::info "Creating Docker network..."
    docker network create "${BTCPAY_NETWORK}" 2>/dev/null || true
    
    # Pull Docker images
    log::info "Pulling BTCPay Server image..."
    if ! docker pull "${BTCPAY_IMAGE}"; then
        log::error "Failed to pull BTCPay Server image"
        return 1
    fi
    
    log::info "Pulling PostgreSQL image..."
    if ! docker pull "${BTCPAY_POSTGRES_IMAGE}"; then
        log::error "Failed to pull PostgreSQL image"
        return 1
    fi
    
    # Register port
    if command -v scripts/resources/port_registry.sh &>/dev/null; then
        scripts/resources/port_registry.sh register btcpay "${BTCPAY_PORT}" || true
    fi
    
    # Create default configuration
    btcpay::create_default_config
    
    log::success "BTCPay Server installed successfully"
    log::info "Use 'start' command to launch BTCPay Server"
    
    return 0
}

# Create default configuration
btcpay::create_default_config() {
    log::info "Creating default configuration..."
    
    # Create BTCPay settings file
    cat > "${BTCPAY_CONFIG_DIR}/settings.json" <<EOF
{
  "ConnectionStrings": {
    "postgres": "Host=btcpay-postgres;Database=btcpayserver;Username=btcpay;Password=btcpay123"
  },
  "BTCPayServer": {
    "Port": ${BTCPAY_PORT},
    "Bind": "0.0.0.0",
    "RootPath": "/",
    "Protocol": "http"
  }
}
EOF
    
    log::success "Configuration created at ${BTCPAY_CONFIG_DIR}/settings.json"
}

# Uninstall function
btcpay::uninstall() {
    log::header "Uninstalling BTCPay Server"
    
    # Stop if running
    if btcpay::is_running; then
        log::info "Stopping BTCPay Server..."
        btcpay::stop
    fi
    
    # Remove containers
    log::info "Removing containers..."
    docker rm -f "${BTCPAY_CONTAINER_NAME}" 2>/dev/null || true
    docker rm -f "${BTCPAY_POSTGRES_CONTAINER}" 2>/dev/null || true
    
    # Remove network
    log::info "Removing network..."
    docker network rm "${BTCPAY_NETWORK}" 2>/dev/null || true
    
    # Remove images (optional)
    read -p "Remove Docker images? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker rmi "${BTCPAY_IMAGE}" 2>/dev/null || true
        docker rmi "${BTCPAY_POSTGRES_IMAGE}" 2>/dev/null || true
    fi
    
    # Keep data directories (for safety)
    log::warning "Data directories preserved at ${BTCPAY_DATA_DIR}"
    log::info "Remove manually if no longer needed"
    
    log::success "BTCPay Server uninstalled"
}

# Export functions
export -f btcpay::install
export -f btcpay::uninstall
export -f btcpay::create_default_config