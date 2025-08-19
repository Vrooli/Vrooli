#!/bin/bash
# ERPNext Status Functions

# Status function
erpnext::status() {
    local format="text"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Gather status information
    local installed="false"
    local running="false"
    local healthy="false"
    local message="ERPNext not installed"
    
    if erpnext::is_installed; then
        installed="true"
        if erpnext::is_running; then
            running="true"
            if erpnext::is_healthy; then
                healthy="true"
                message="ERPNext is running and healthy on port ${ERPNEXT_PORT}"
            else
                message="ERPNext is running but not responding to health checks"
            fi
        else
            message="ERPNext is installed but not running"
        fi
    fi
    
    # Build status data array
    local status_data=(
        "name" "erpnext"
        "status" "$([ "$healthy" = "true" ] && echo "healthy" || echo "unhealthy")"
        "installed" "$installed"
        "running" "$running"
        "health" "$healthy"
        "healthy" "$healthy"
        "version" "${ERPNEXT_VERSION:-unknown}"
        "port" "${ERPNEXT_PORT:-8020}"
        "site" "${ERPNEXT_SITE_NAME:-vrooli.local}"
        "message" "$message"
        "description" "Complete open-source ERP suite"
        "category" "automation"
    )
    
    # Format output
    if [[ "$format" == "json" ]]; then
        format::output "json" "kv" "${status_data[@]}"
    else
        # Text output
        echo "Name: erpnext"
        echo "Status: $([ "$healthy" = "true" ] && echo "healthy" || echo "unhealthy")"
        echo "Installed: $installed"
        echo "Running: $running"
        echo "Health: $healthy"
        echo "Healthy: $healthy"
        echo "Version: ${ERPNEXT_VERSION:-unknown}"
        echo "Port: ${ERPNEXT_PORT:-8020}"
        echo "Site: ${ERPNEXT_SITE_NAME:-vrooli.local}"
        echo "Message: $message"
        echo "Description: Complete open-source ERP suite"
        echo "Category: automation"
    fi
    
    return 0
}