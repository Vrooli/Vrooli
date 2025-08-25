#!/usr/bin/env bash
#
# Browserless status functions

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSERLESS_STATUS_DIR="${APP_ROOT}/resources/browserless/lib"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "$BROWSERLESS_STATUS_DIR/common.sh"

function status() {
    local format_type="text"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format_type="${2:-text}"
                shift 2
                ;;
            json)
                format_type="json"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Collect status information
    local installed="No"
    local running="No"
    local health="Unknown"
    local container_status="not_found"
    local metrics="{}"
    
    # Check if Docker image exists
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "$BROWSERLESS_IMAGE"; then
        installed="Yes"
    fi
    
    # Check container status
    if is_running; then
        running="Yes"
        container_status=$(get_container_status)
        
        # Check health
        health=$(check_health)
        if [[ "$health" == "healthy" ]]; then
            health="Healthy"
            metrics=$(get_metrics)
        else
            health="Unhealthy"
        fi
    fi
    
    # Format output based on requested format
    if [[ "$format_type" == "json" ]]; then
        cat <<EOF
{
  "name": "browserless",
  "description": "Headless Chrome automation service",
  "category": "agents",
  "installed": $([ "$installed" = "Yes" ] && echo "true" || echo "false"),
  "running": $([ "$running" = "Yes" ] && echo "true" || echo "false"),
  "healthy": $([ "$health" = "Healthy" ] && echo "true" || echo "false"),
  "status": "$health",
  "container": {
    "name": "$BROWSERLESS_CONTAINER_NAME",
    "status": "$container_status",
    "image": "$BROWSERLESS_IMAGE"
  },
  "endpoints": {
    "base": "http://localhost:${BROWSERLESS_PORT}",
    "docs": "http://localhost:${BROWSERLESS_PORT}/docs",
    "pressure": "http://localhost:${BROWSERLESS_PORT}/pressure",
    "screenshot": "http://localhost:${BROWSERLESS_PORT}/screenshot",
    "pdf": "http://localhost:${BROWSERLESS_PORT}/pdf",
    "content": "http://localhost:${BROWSERLESS_PORT}/content"
  },
  "configuration": {
    "port": ${BROWSERLESS_PORT},
    "max_concurrent_sessions": ${BROWSERLESS_MAX_CONCURRENT_SESSIONS},
    "workspace_dir": "$BROWSERLESS_WORKSPACE_DIR"
  },
  "metrics": $metrics
}
EOF
    else
        echo "🌐 Browserless Status"
        echo "─────────────────────"
        echo "Installed: $installed"
        echo "Running: $running"
        echo "Health: $health"
        echo "Port: $BROWSERLESS_PORT"
        echo "Container: $BROWSERLESS_CONTAINER_NAME"
        echo "Status: $container_status"
    fi
}
