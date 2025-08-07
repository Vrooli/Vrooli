#!/bin/bash
# Resource Discovery Script - Triggers manual resource discovery

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

echo "============================================"
echo "Resource Discovery - Manual Trigger"
echo "$(date)"
echo "============================================"

# Check if n8n is running
if ! curl -s "http://localhost:5678/healthz" >/dev/null 2>&1; then
    log "ERROR: n8n is not running on port 5678"
    exit 1
fi

# Trigger discovery via webhook
log "Triggering resource discovery..."
response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"trigger": "manual", "timestamp": "'$(date -Iseconds)'"}' \
    "http://localhost:5678/webhook/discover-resources" 2>&1)

if [ $? -eq 0 ]; then
    log "✓ Discovery triggered successfully"
    
    # Parse response if it contains JSON
    if echo "$response" | grep -q "{"; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        echo "$response"
    fi
    
    # Wait a moment for discovery to process
    sleep 3
    
    # Check discovery summary in Redis
    log "Checking discovery results..."
    if command -v redis-cli >/dev/null 2>&1; then
        summary=$(redis-cli -p 6380 GET discovery_summary 2>/dev/null || echo "{}")
        if [ "$summary" != "{}" ] && [ -n "$summary" ]; then
            log "Discovery Summary:"
            echo "$summary" | jq '.' 2>/dev/null || echo "$summary"
        fi
    fi
    
    # Query database for discovered resources
    if command -v psql >/dev/null 2>&1; then
        log "Discovered resources:"
        PGPASSWORD="${POSTGRES_PASSWORD:-vrooli}" psql \
            -h "${POSTGRES_HOST:-localhost}" \
            -p "${POSTGRES_PORT:-5433}" \
            -U "${POSTGRES_USER:-vrooli}" \
            -d "${POSTGRES_DB:-vrooli}" \
            -t -c "SELECT resource_name, resource_type, current_status, port 
                   FROM resource_monitoring.resources 
                   WHERE is_enabled = true 
                   ORDER BY is_critical DESC, resource_name" 2>/dev/null | \
            column -t -s '|' || log "Could not query database"
    fi
    
    exit 0
else
    log "✗ Failed to trigger discovery"
    echo "Error: $response"
    exit 1
fi