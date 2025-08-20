#!/bin/bash
# Status functions for Odoo resource

# Simple format_status function if not already available
if ! declare -f format_status &>/dev/null; then
    format_status() {
        local format="${1:-text}"
        local name="$2"
        local status="$3"
        local health="$4"
        local message="$5"
        local details="${6:-{}}"
        
        if [[ "$format" == "json" ]]; then
            echo "{\"name\":\"$name\",\"status\":\"$status\",\"health\":\"$health\",\"message\":\"$message\",\"details\":$details}"
        else
            echo "Resource: $name"
            echo "Status: $status"
            echo "Health: $health"
            echo "Message: $message"
        fi
    }
fi

odoo_status() {
    local format="${1:-text}"
    local status="stopped"
    local message="Odoo is not running"
    local health="unhealthy"
    local details=""
    local test_status="not_run"
    local test_timestamp=""
    
    # Check if installed
    if ! odoo_is_installed; then
        message="Odoo is not installed"
        format_status "$format" "$ODOO_RESOURCE_NAME" "$status" "$health" "$message" "{}"
        return 1
    fi
    
    # Check if running
    if odoo_is_running; then
        status="running"
        
        # Check health via HTTP
        if curl -s -f "http://localhost:$ODOO_PORT/web/database/selector" | grep -q "Odoo" 2>/dev/null; then
            health="healthy"
            message="Odoo is running and responding"
            
            # Get container stats
            local cpu_usage=$(docker stats --no-stream --format "{{.CPUPerc}}" "$ODOO_CONTAINER_NAME" 2>/dev/null | tr -d '%')
            local mem_usage=$(docker stats --no-stream --format "{{.MemUsage}}" "$ODOO_CONTAINER_NAME" 2>/dev/null)
            
            # Check test results
            if [[ -f "$ODOO_DATA_DIR/.last_test" ]]; then
                test_timestamp=$(stat -c %Y "$ODOO_DATA_DIR/.last_test" 2>/dev/null || echo "0")
                test_status=$(cat "$ODOO_DATA_DIR/.last_test" 2>/dev/null || echo "unknown")
            fi
            
            details=$(cat <<EOF
{
  "url": "http://localhost:$ODOO_PORT",
  "database": "$ODOO_DB_NAME",
  "containers": {
    "odoo": "$ODOO_CONTAINER_NAME",
    "postgres": "$ODOO_PG_CONTAINER_NAME"
  },
  "resources": {
    "cpu_usage": "$cpu_usage%",
    "memory": "$mem_usage"
  },
  "test": {
    "status": "$test_status",
    "last_run": "$test_timestamp"
  }
}
EOF
)
        else
            health="degraded"
            message="Odoo is running but not responding properly"
            details='{"error": "HTTP check failed"}'
        fi
    else
        # Check if PostgreSQL is running
        if docker ps --format "{{.Names}}" | grep -q "^${ODOO_PG_CONTAINER_NAME}$"; then
            health="degraded"
            message="PostgreSQL is running but Odoo is stopped"
            details='{"postgres": "running", "odoo": "stopped"}'
        fi
    fi
    
    # Format output
    format_status "$format" "$ODOO_RESOURCE_NAME" "$status" "$health" "$message" "$details"
    
    [[ "$health" == "healthy" ]] && return 0 || return 1
}

export -f odoo_status