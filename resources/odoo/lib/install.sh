#!/bin/bash
# Install functions for Odoo resource

# Get script directory and source common
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/resources/odoo/lib/common.sh"

odoo_install() {
    local force="${1:-false}"
    
    echo "Installing Odoo Community..."
    
    # Check if already installed
    if odoo_is_installed && [[ "$force" != "--force" ]]; then
        echo "Odoo is already installed. Use --force to reinstall."
        return 0
    fi
    
    # Initialize directories
    odoo_init_dirs
    
    # Pull Docker images
    echo "Pulling Docker images..."
    docker pull "$ODOO_IMAGE" || {
        echo "Failed to pull Odoo image"
        return 1
    }
    
    docker pull "$ODOO_PG_IMAGE" || {
        echo "Failed to pull PostgreSQL image"
        return 1
    }
    
    # Create Docker network
    if ! docker network inspect "$ODOO_NETWORK_NAME" &>/dev/null; then
        echo "Creating Docker network..."
        docker network create "$ODOO_NETWORK_NAME" || {
            echo "Failed to create Docker network"
            return 1
        }
    fi
    
    # Register ports
    odoo_register_ports
    
    # Create Odoo configuration file
    echo "Creating Odoo configuration..."
    cat > "$ODOO_DATA_DIR/config/odoo.conf" <<EOF
[options]
addons_path = /mnt/extra-addons
data_dir = /var/lib/odoo
db_host = $ODOO_PG_CONTAINER_NAME
db_port = 5432
db_user = $ODOO_DB_USER
db_password = $ODOO_DB_PASSWORD
db_name = $ODOO_DB_NAME
admin_passwd = $ODOO_ADMIN_PASSWORD
http_port = 8069
longpolling_port = 8072
log_level = info
log_handler = :INFO
without_demo = False
proxy_mode = True
EOF
    
    # Register CLI
    echo "Registering Odoo CLI..."
    "${APP_ROOT}/scripts/lib/resources/install-resource-cli.sh" \
        "odoo" \
        "$ODOO_BASE_DIR/cli.sh" \
        "resource-odoo"
    
    echo "Odoo installation complete!"
    echo ""
    echo "Configuration:"
    echo "  Web Interface: http://localhost:$ODOO_PORT"
    echo "  Admin Email: $ODOO_ADMIN_EMAIL"
    echo "  Admin Password: $ODOO_ADMIN_PASSWORD (change immediately)"
    echo ""
    echo "Start Odoo with: vrooli resource odoo start"
    
    return 0
}

# Uninstall Odoo - required by Universal Contract
odoo::install::uninstall() {
    log::info "Uninstalling Odoo..."
    
    # Stop containers if running
    if odoo_is_running; then
        log::info "Stopping Odoo services..."
        odoo_stop
    fi
    
    # Remove Docker containers
    if docker ps -a --format "{{.Names}}" | grep -q "^${ODOO_CONTAINER_NAME}$"; then
        log::info "Removing Odoo container..."
        docker rm -f "$ODOO_CONTAINER_NAME" 2>/dev/null || true
    fi
    
    if docker ps -a --format "{{.Names}}" | grep -q "^${ODOO_PG_CONTAINER_NAME}$"; then
        log::info "Removing PostgreSQL container..."
        docker rm -f "$ODOO_PG_CONTAINER_NAME" 2>/dev/null || true
    fi
    
    # Remove Docker network
    if docker network inspect "$ODOO_NETWORK_NAME" &>/dev/null; then
        log::info "Removing Docker network..."
        docker network rm "$ODOO_NETWORK_NAME" 2>/dev/null || true
    fi
    
    # Remove Docker images
    log::info "Removing Docker images..."
    docker rmi "$ODOO_IMAGE" 2>/dev/null || true
    docker rmi "$ODOO_PG_IMAGE" 2>/dev/null || true
    
    # Remove data directory (with confirmation)
    if [[ -d "$ODOO_DATA_DIR" ]]; then
        log::warn "Data directory: $ODOO_DATA_DIR"
        log::warn "This contains your Odoo data and cannot be recovered once deleted."
        read -p "Remove data directory? [y/N]: " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log::info "Removing data directory..."
            rm -rf "$ODOO_DATA_DIR"
        else
            log::info "Data directory preserved at: $ODOO_DATA_DIR"
        fi
    fi
    
    log::success "Odoo uninstalled"
    return 0
}

# Map to v2.0 naming convention
odoo::install::execute() {
    odoo_install "$@"
}

export -f odoo_install
export -f odoo::install::uninstall
export -f odoo::install::execute