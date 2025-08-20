#!/bin/bash
# Stop functions for Odoo resource

odoo_stop() {
    echo "Stopping Odoo Community..."
    
    # Stop Odoo container
    if docker ps --format "{{.Names}}" | grep -q "^${ODOO_CONTAINER_NAME}$"; then
        echo "Stopping Odoo container..."
        docker stop "$ODOO_CONTAINER_NAME" || {
            echo "Warning: Failed to stop Odoo gracefully"
        }
        docker rm "$ODOO_CONTAINER_NAME" 2>/dev/null
    fi
    
    # Stop PostgreSQL container
    if docker ps --format "{{.Names}}" | grep -q "^${ODOO_PG_CONTAINER_NAME}$"; then
        echo "Stopping PostgreSQL container..."
        docker stop "$ODOO_PG_CONTAINER_NAME" || {
            echo "Warning: Failed to stop PostgreSQL gracefully"
        }
        docker rm "$ODOO_PG_CONTAINER_NAME" 2>/dev/null
    fi
    
    echo "Odoo stopped"
    return 0
}

export -f odoo_stop