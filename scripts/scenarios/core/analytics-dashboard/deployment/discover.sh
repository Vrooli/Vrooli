#\!/bin/bash
# Resource Monitoring Platform - Manual Discovery Script

set -euo pipefail

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"; }

log "Running resource discovery..."

# Trigger n8n discovery workflow
if curl -s -X POST "http://localhost:5678/webhook/discover-resources" | grep -q "success\|discovered"; then
    log "✓ n8n discovery workflow triggered"
else
    log "✗ Failed to trigger n8n discovery workflow"
fi

# Manual port scan for common services
declare -A known_services=(
    [5433]="postgres"
    [6380]="redis"
    [9009]="questdb"
    [8200]="vault"
    [5678]="n8n"
    [5681]="windmill"
    [1880]="node-red"
    [11434]="ollama"
    [9000]="minio"
)

log "Scanning for known services..."

for port in "${\!known_services[@]}"; do
    service="${known_services[$port]}"
    if nc -z localhost "$port" 2>/dev/null; then
        log "✓ Found ${service} on port ${port}"
    else
        log "- ${service} not found on port ${port}"
    fi
done

log "Discovery completed"
