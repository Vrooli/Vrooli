#\!/bin/bash
# Resource Monitoring Platform - Startup Script

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="${SCRIPT_DIR}/.."

# Source port registry for dynamic port resolution
# shellcheck disable=SC1091
source "${SCENARIO_DIR}/../../../resources/common.sh"

echo "============================================"
echo "Resource Monitoring Platform - Startup"
echo "$(date)"
echo "============================================"

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check if service is running
check_service() {
    local service_name="$1"
    local port="$2"
    
    if nc -z localhost "${port}" 2>/dev/null; then
        log "✓ ${service_name} is running on port ${port}"
        return 0
    else
        log "✗ ${service_name} is not running on port ${port}"
        return 1
    fi
}

# Step 1: Verify required resources
log "Step 1: Verifying required resources..."

required_services=(
    "postgres:5433"
    "redis:6380" 
    "questdb:$(resources::get_default_port "questdb")"
    "vault:8200"
    "n8n:5678"
    "windmill:5681"
)

missing_services=()

for service_info in "${required_services[@]}"; do
    IFS=':' read -r service port <<< "${service_info}"
    if \! check_service "${service}" "${port}"; then
        missing_services+=("${service}")
    fi
done

if [ ${#missing_services[@]} -gt 0 ]; then
    log "ERROR: Missing required services: ${missing_services[*]}"
    exit 1
fi

log "✓ All required resources are available"

# Step 2: Initialize databases
log "Step 2: Initializing databases..."

# PostgreSQL
if command -v psql >/dev/null 2>&1; then
    PGPASSWORD="${POSTGRES_PASSWORD:-vrooli}" psql \
        -h "${POSTGRES_HOST:-localhost}" \
        -p "${POSTGRES_PORT:-5433}" \
        -U "${POSTGRES_USER:-vrooli}" \
        -d "${POSTGRES_DB:-vrooli}" \
        -f "${SCENARIO_DIR}/initialization/storage/postgres/schema.sql" \
        -f "${SCENARIO_DIR}/initialization/storage/postgres/seed.sql" \
        2>/dev/null || log "WARNING: PostgreSQL setup failed"
fi

# QuestDB
curl -s "http://localhost:$(resources::get_default_port "questdb")/exec" \
    --data-urlencode "query=$(cat "${SCENARIO_DIR}/initialization/storage/questdb/tables.sql")" \
    >/dev/null 2>&1 || log "WARNING: QuestDB setup failed"

log "✓ Database initialization completed"

# Step 3: Trigger initial discovery
log "Step 3: Running initial resource discovery..."

curl -s -X POST "http://localhost:5678/webhook/discover-resources" >/dev/null 2>&1 || \
    log "WARNING: Could not trigger discovery"

log "✓ Resource monitoring platform startup completed"

echo ""
echo "🚀 Resource Monitoring Platform is READY\!"
echo ""
echo "Dashboard: http://localhost:5681/f/monitoring/dashboard"
echo "n8n Workflows: http://localhost:5678"
echo "QuestDB Console: http://localhost:$(resources::get_default_port "questdb")"
echo ""
