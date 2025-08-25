#!/bin/bash

# Core functions for Mail-in-a-Box resource

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
MAILINABOX_LIB_DIR="${APP_ROOT}/resources/mail-in-a-box/lib"
MAILINABOX_CONFIG_DIR="${APP_ROOT}/resources/mail-in-a-box/config"

# Source dependencies
source "$MAILINABOX_CONFIG_DIR/defaults.sh"
source "/home/matthalloran8/Vrooli/scripts/lib/utils/format.sh"

# Check if Mail-in-a-Box is installed
mailinabox_is_installed() {
    if [[ -d "$MAILINABOX_DATA_DIR" ]] && docker inspect "$MAILINABOX_CONTAINER_NAME" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Check if Mail-in-a-Box is running
mailinabox_is_running() {
    local status=$(docker inspect -f '{{.State.Running}}' "$MAILINABOX_CONTAINER_NAME" 2>/dev/null)
    [[ "$status" == "true" ]]
}

# Get Mail-in-a-Box health status
mailinabox_get_health() {
    if ! mailinabox_is_running; then
        echo "unhealthy"
        return 1
    fi
    
    # Check admin panel availability
    if timeout 5 curl -sk "https://${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_ADMIN}/admin" >/dev/null 2>&1; then
        echo "healthy"
        return 0
    else
        echo "unhealthy"
        return 1
    fi
}

# Get Mail-in-a-Box version
mailinabox_get_version() {
    if mailinabox_is_installed; then
        docker inspect -f '{{.Config.Image}}' "$MAILINABOX_CONTAINER_NAME" 2>/dev/null | cut -d: -f2 || echo "unknown"
    else
        echo "not_installed"
    fi
}

# Get Mail-in-a-Box status details
mailinabox_get_status_details() {
    local details=""
    
    if mailinabox_is_running; then
        # Get mailbox count
        local mailbox_count=$(docker exec "$MAILINABOX_CONTAINER_NAME" sqlite3 /home/user-data/mail/users.sqlite "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "0")
        details="Mailboxes: ${mailbox_count}"
        
        # Get domain count
        local domain_count=$(docker exec "$MAILINABOX_CONTAINER_NAME" sqlite3 /home/user-data/mail/users.sqlite "SELECT COUNT(DISTINCT domain) FROM users;" 2>/dev/null || echo "0")
        details="${details}, Domains: ${domain_count}"
    else
        details="Service not running"
    fi
    
    echo "$details"
}

# Get resource URLs
mailinabox_get_urls() {
    local urls=""
    urls="Admin Panel: https://${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_ADMIN}/admin"
    urls="${urls}, Webmail: https://${MAILINABOX_BIND_ADDRESS}/mail"
    echo "$urls"
}