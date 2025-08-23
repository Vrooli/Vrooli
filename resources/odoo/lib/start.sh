#!/bin/bash
# Start functions for Odoo resource

odoo_start() {
    echo "Starting Odoo Community..."
    
    # Check if already running
    if odoo_is_running; then
        echo "Odoo is already running"
        return 0
    fi
    
    # Check if installed
    if ! odoo_is_installed; then
        echo "Odoo is not installed. Run 'vrooli resource odoo install' first."
        return 1
    fi
    
    # Initialize directories
    odoo_init_dirs
    
    # Start PostgreSQL container
    if ! docker ps --format "{{.Names}}" | grep -q "^${ODOO_PG_CONTAINER_NAME}$"; then
        echo "Starting PostgreSQL for Odoo..."
        docker run -d \
            --name "$ODOO_PG_CONTAINER_NAME" \
            --network "$ODOO_NETWORK_NAME" \
            -e POSTGRES_DB="$ODOO_DB_NAME" \
            -e POSTGRES_USER="$ODOO_DB_USER" \
            -e POSTGRES_PASSWORD="$ODOO_DB_PASSWORD" \
            -v "$ODOO_DATA_DIR/postgres:/var/lib/postgresql/data" \
            -p "$ODOO_PG_PORT:5432" \
            --restart unless-stopped \
            "$ODOO_PG_IMAGE" || {
                echo "Failed to start PostgreSQL"
                return 1
            }
        
        # Wait for PostgreSQL to be ready
        echo "Waiting for PostgreSQL to be ready..."
        sleep 5
    fi
    
    # Start Odoo container
    echo "Starting Odoo container..."
    docker run -d \
        --name "$ODOO_CONTAINER_NAME" \
        --network "$ODOO_NETWORK_NAME" \
        -p "$ODOO_PORT:8069" \
        -p "$ODOO_LONGPOLLING_PORT:8072" \
        -v "$ODOO_DATA_DIR/config:/etc/odoo" \
        -v "$ODOO_DATA_DIR/addons:/mnt/extra-addons" \
        -v "$ODOO_DATA_DIR/filestore:/var/lib/odoo" \
        -v "$ODOO_DATA_DIR/sessions:/var/lib/odoo/sessions" \
        --restart unless-stopped \
        "$ODOO_IMAGE" || {
            echo "Failed to start Odoo"
            docker rm -f "$ODOO_PG_CONTAINER_NAME" 2>/dev/null
            return 1
        }
    
    # Wait for Odoo to be ready
    echo "Waiting for Odoo to initialize..."
    local max_attempts=30
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if curl -s "http://localhost:$ODOO_PORT/web/database/selector" | grep -q "Odoo"; then
            echo "Odoo is ready!"
            echo ""
            echo "Access Odoo at: http://localhost:$ODOO_PORT"
            echo "Default credentials:"
            echo "  Email: $ODOO_ADMIN_EMAIL"
            echo "  Password: $ODOO_ADMIN_PASSWORD"
            return 0
        fi
        sleep 2
        ((attempt++))
    done
    
    echo "Warning: Odoo may still be initializing. Check logs with 'vrooli resource odoo logs'"
    return 0
}

export -f odoo_start