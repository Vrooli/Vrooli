#!/bin/bash

# Status functions for Mail-in-a-Box resource

MAILINABOX_STATUS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "$MAILINABOX_STATUS_LIB_DIR/core.sh"

# Get Mail-in-a-Box status
mailinabox_status() {
    local format="${1:-text}"
    
    # Gather status information
    local installed=$(mailinabox_is_installed && echo "true" || echo "false")
    local running=$(mailinabox_is_running && echo "true" || echo "false")
    local health=$(mailinabox_get_health)
    local version=$(mailinabox_get_version)
    local details=$(mailinabox_get_status_details)
    local healthy=$([[ "$health" == "healthy" ]] && echo "true" || echo "false")
    
    # Prepare data for formatting
    local status_data=(
        "name" "$MAILINABOX_NAME"
        "category" "$MAILINABOX_CATEGORY"
        "description" "$MAILINABOX_DESCRIPTION"
        "installed" "$installed"
        "running" "$running"
        "healthy" "$healthy"
        "health_message" "$health"
        "version" "$version"
        "port" "$MAILINABOX_PORT_ADMIN"
        "admin_url" "https://${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_ADMIN}/admin"
        "webmail_url" "https://${MAILINABOX_BIND_ADDRESS}/mail"
        "details" "$details"
    )
    
    # Output based on format
    if [[ "$format" == "json" ]]; then
        format::output "json" "kv" "${status_data[@]}"
    else
        # Text format - simple output for now
        echo "Status: $([[ "$installed" == "false" ]] && echo "not_installed" || ([[ "$running" == "false" ]] && echo "stopped" || echo "running"))"
        echo "Health: $health"
        echo "Message: Mail-in-a-Box email server"
        echo "Api Base: https://${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_ADMIN}"
        echo "Port: $MAILINABOX_PORT_ADMIN"
    fi
    
    # Return appropriate exit code
    if [[ "$healthy" == "true" ]]; then
        return 0
    elif [[ "$running" == "true" ]]; then
        return 1
    else
        return 2
    fi
}