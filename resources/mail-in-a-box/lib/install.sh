#!/bin/bash

# Installation functions for Mail-in-a-Box resource

MAILINABOX_INSTALL_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "$MAILINABOX_INSTALL_LIB_DIR/core.sh"

# Install Mail-in-a-Box
mailinabox_install() {
    format_header "📧 Installing Mail-in-a-Box"
    
    # Check if already installed
    if mailinabox_is_installed; then
        format_warning "Mail-in-a-Box is already installed"
        return 0
    fi
    
    # Create data directories
    format_info "Creating data directories..."
    mkdir -p "$MAILINABOX_DATA_DIR"/{mail,config,ssl,backup}
    
    # Pull Docker image
    format_info "Pulling Mail-in-a-Box Docker image..."
    if ! docker pull "$MAILINABOX_IMAGE"; then
        format_error "Failed to pull Mail-in-a-Box image"
        return 1
    fi
    
    # Create configuration file
    format_info "Creating configuration..."
    cat > "$MAILINABOX_CONFIG_DIR/mailinabox.env" <<EOF
PRIMARY_HOSTNAME=${MAILINABOX_PRIMARY_HOSTNAME}
ADMIN_EMAIL=${MAILINABOX_ADMIN_EMAIL}
ADMIN_PASSWORD=${MAILINABOX_ADMIN_PASSWORD}
ENABLE_WEBMAIL=${MAILINABOX_ENABLE_WEBMAIL}
ENABLE_CALDAV=${MAILINABOX_ENABLE_CALDAV}
ENABLE_CARDDAV=${MAILINABOX_ENABLE_CARDDAV}
ENABLE_AUTOCONFIG=${MAILINABOX_ENABLE_AUTOCONFIG}
ENABLE_GREYLISTING=${MAILINABOX_ENABLE_GREYLISTING}
ENABLE_SPAMASSASSIN=${MAILINABOX_ENABLE_SPAMASSASSIN}
ENABLE_FAIL2BAN=${MAILINABOX_ENABLE_FAIL2BAN}
EOF
    
    # Create and start container
    format_info "Creating Mail-in-a-Box container..."
    docker create \
        --name "$MAILINABOX_CONTAINER_NAME" \
        --hostname "$MAILINABOX_PRIMARY_HOSTNAME" \
        --env-file "$MAILINABOX_CONFIG_DIR/mailinabox.env" \
        -p "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_SMTP}:25" \
        -p "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_SUBMISSION}:587" \
        -p "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_IMAP}:143" \
        -p "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_IMAPS}:993" \
        -p "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_POP3}:110" \
        -p "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_POP3S}:995" \
        -p "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_ADMIN}:443" \
        -v "$MAILINABOX_MAIL_DIR:/home/user-data/mail" \
        -v "$MAILINABOX_CONFIG_DIR:/home/user-data/config" \
        -v "$MAILINABOX_SSL_DIR:/home/user-data/ssl" \
        --restart unless-stopped \
        "$MAILINABOX_IMAGE"
    
    if [[ $? -eq 0 ]]; then
        format_success "Mail-in-a-Box installed successfully"
        return 0
    else
        format_error "Failed to create Mail-in-a-Box container"
        return 1
    fi
}

# Uninstall Mail-in-a-Box
mailinabox_uninstall() {
    format_header "🗑️ Uninstalling Mail-in-a-Box"
    
    # Stop and remove container
    if docker inspect "$MAILINABOX_CONTAINER_NAME" >/dev/null 2>&1; then
        format_info "Removing Mail-in-a-Box container..."
        docker stop "$MAILINABOX_CONTAINER_NAME" >/dev/null 2>&1
        docker rm "$MAILINABOX_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    # Optionally remove data
    read -p "Remove Mail-in-a-Box data directory? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        format_info "Removing data directory..."
        rm -rf "$MAILINABOX_DATA_DIR"
    fi
    
    format_success "Mail-in-a-Box uninstalled"
    return 0
}