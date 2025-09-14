#!/bin/bash

# Installation functions for Mail-in-a-Box resource

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
MAILINABOX_INSTALL_LIB_DIR="${APP_ROOT}/resources/mail-in-a-box/lib"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
source "$MAILINABOX_INSTALL_LIB_DIR/core.sh"

# Install Mail-in-a-Box
mailinabox_install() {
    log::header "📧 Installing Mail-in-a-Box"
    
    # Check if already installed
    if mailinabox_is_installed; then
        log::warning "Mail-in-a-Box is already installed"
        return 0
    fi
    
    # Create data directories
    log::info "Creating data directories..."
    mkdir -p "$MAILINABOX_DATA_DIR"/{mail,config,ssl,backup}
    
    # Pull Docker image
    log::info "Pulling Mail-in-a-Box Docker image..."
    if ! docker pull "$MAILINABOX_IMAGE"; then
        log::error "Failed to pull Mail-in-a-Box image"
        return 1
    fi
    
    # Create configuration file for docker-mailserver
    log::info "Creating configuration..."
    cat > "$MAILINABOX_CONFIG_DIR/mailserver.env" <<EOF
# Core settings
OVERRIDE_HOSTNAME=${MAILINABOX_PRIMARY_HOSTNAME}
PERMIT_DOCKER=connected-networks
ENABLE_FAIL2BAN=1
ENABLE_SPAMASSASSIN=1
ENABLE_CLAMAV=0
SPAMASSASSIN_SPAM_TO_INBOX=1
MOVE_SPAM_TO_JUNK=1
VIRUSMAILS_DELETE_DELAY=7
ONE_DIR=0
DMS_DEBUG=0
EOF
    
    # Create and start container for docker-mailserver
    log::info "Creating Mail Server container..."
    docker create \
        --name "$MAILINABOX_CONTAINER_NAME" \
        --hostname "$MAILINABOX_PRIMARY_HOSTNAME" \
        --env-file "$MAILINABOX_CONFIG_DIR/mailserver.env" \
        -p "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_SMTP}:25" \
        -p "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_SUBMISSION}:587" \
        -p "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_IMAPS}:993" \
        -p "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_POP3S}:995" \
        -v "$MAILINABOX_MAIL_DIR:/var/mail" \
        -v "$MAILINABOX_CONFIG_DIR:/tmp/docker-mailserver" \
        -v "$MAILINABOX_SSL_DIR:/etc/letsencrypt" \
        -v /etc/localtime:/etc/localtime:ro \
        --cap-add NET_ADMIN \
        --restart unless-stopped \
        "$MAILINABOX_IMAGE"
    
    if [[ $? -eq 0 ]]; then
        log::success "Mail-in-a-Box installed successfully"
        return 0
    else
        log::error "Failed to create Mail-in-a-Box container"
        return 1
    fi
}

# Uninstall Mail-in-a-Box
mailinabox_uninstall() {
    log::header "🗑️ Uninstalling Mail-in-a-Box"
    
    # Stop and remove container
    if docker inspect "$MAILINABOX_CONTAINER_NAME" >/dev/null 2>&1; then
        log::info "Removing Mail-in-a-Box container..."
        docker stop "$MAILINABOX_CONTAINER_NAME" >/dev/null 2>&1
        docker rm "$MAILINABOX_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    # Optionally remove data
    read -p "Remove Mail-in-a-Box data directory? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log::info "Removing data directory..."
        rm -rf "$MAILINABOX_DATA_DIR"
    fi
    
    log::success "Mail-in-a-Box uninstalled"
    return 0
}