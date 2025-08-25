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

export -f odoo_install