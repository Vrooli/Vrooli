#!/bin/bash
# Resource Monitoring Platform - Startup Script

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="${SCRIPT_DIR}/.."

# Source port registry for dynamic port resolution
# shellcheck disable=SC1091
source "${SCENARIO_DIR}/../../../../resources/common.sh"

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
    "node-red:1880"
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

# Step 3: Setup n8n credentials and Vault secrets
log "Step 3: Setting up credentials and secrets..."

setup_n8n_credentials() {
    local n8n_url="http://localhost:5678"
    local max_retries=10
    local retry_count=0
    
    log "Waiting for n8n to be ready..."
    while ! curl -s "${n8n_url}/healthz" >/dev/null 2>&1; do
        ((retry_count++))
        if [ $retry_count -ge $max_retries ]; then
            log "WARNING: n8n not available after ${max_retries} retries"
            return 1
        fi
        sleep 2
    done
    
    # Create PostgreSQL credential
    log "Creating PostgreSQL credential..."
    curl -s -X POST "${n8n_url}/rest/credentials" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "PostgreSQL Monitoring",
            "type": "postgres",
            "data": {
                "host": "localhost",
                "port": 5433,
                "database": "'"${POSTGRES_DB:-vrooli}"'",
                "user": "'"${POSTGRES_USER:-vrooli}"'",
                "password": "'"${POSTGRES_PASSWORD:-vrooli}"'",
                "ssl": "disable"
            }
        }' >/dev/null 2>&1 || log "WARNING: PostgreSQL credential creation failed"
    
    # Create Redis credential
    log "Creating Redis credential..."
    curl -s -X POST "${n8n_url}/rest/credentials" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "Redis Monitoring",
            "type": "redis",
            "data": {
                "host": "localhost",
                "port": 6380,
                "password": "",
                "database": 0
            }
        }' >/dev/null 2>&1 || log "WARNING: Redis credential creation failed"
    
    # Create Vault credential
    log "Creating Vault credential..."
    curl -s -X POST "${n8n_url}/rest/credentials" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "Vault Monitoring",
            "type": "vault",
            "data": {
                "url": "http://localhost:8200",
                "token": "'"${VAULT_TOKEN:-dev-root-token}"'",
                "version": "v2"
            }
        }' >/dev/null 2>&1 || log "WARNING: Vault credential creation failed"
    
    # Create Twilio credential
    log "Creating Twilio credential..."
    curl -s -X POST "${n8n_url}/rest/credentials" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "Twilio API",
            "type": "twilioApi",
            "data": {
                "accountSid": "'"${TWILIO_ACCOUNT_SID:-ACxxxxxxxxxxxxxxxxxxxxxxxxx}"'",
                "authToken": "'"${TWILIO_AUTH_TOKEN:-dummy_token}"'"
            }
        }' >/dev/null 2>&1 || log "WARNING: Twilio credential creation failed"
    
    # Create SMTP credential
    log "Creating SMTP credential..."
    curl -s -X POST "${n8n_url}/rest/credentials" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "SMTP Monitoring",
            "type": "smtp",
            "data": {
                "host": "'"${SMTP_HOST:-localhost}"'",
                "port": '"${SMTP_PORT:-587}"',
                "secure": false,
                "user": "'"${SMTP_USERNAME:-monitoring@localhost}"'",
                "password": "'"${SMTP_PASSWORD:-dummy_password}"'"
            }
        }' >/dev/null 2>&1 || log "WARNING: SMTP credential creation failed"
    
    log "✓ n8n credentials setup completed"
}

setup_vault_secrets() {
    local vault_url="http://localhost:8200"
    local vault_token="${VAULT_TOKEN:-dev-root-token}"
    
    if ! curl -s "${vault_url}/v1/sys/health" >/dev/null 2>&1; then
        log "WARNING: Vault not available, skipping secret setup"
        return 1
    fi
    
    # Setup Twilio secrets
    log "Setting up Vault secrets for Twilio..."
    curl -s -X POST "${vault_url}/v1/secret/data/monitoring/twilio" \
        -H "X-Vault-Token: ${vault_token}" \
        -H "Content-Type: application/json" \
        -d '{
            "data": {
                "account_sid": "'"${TWILIO_ACCOUNT_SID:-ACxxxxxxxxxxxxxxxxxxxxxxxxx}"'",
                "auth_token": "'"${TWILIO_AUTH_TOKEN:-dummy_token}"'",
                "from_number": "'"${TWILIO_FROM_NUMBER:-+1234567890}"'",
                "to_number": "'"${ALERT_PHONE_NUMBER:-+1234567890}"'"
            }
        }' >/dev/null 2>&1 || log "WARNING: Twilio secret setup failed"
    
    # Setup SMTP secrets
    log "Setting up Vault secrets for SMTP..."
    curl -s -X POST "${vault_url}/v1/secret/data/monitoring/smtp" \
        -H "X-Vault-Token: ${vault_token}" \
        -H "Content-Type: application/json" \
        -d '{
            "data": {
                "host": "'"${SMTP_HOST:-localhost}"'",
                "port": "'"${SMTP_PORT:-587}"'",
                "username": "'"${SMTP_USERNAME:-monitoring@localhost}"'",
                "password": "'"${SMTP_PASSWORD:-dummy_password}"'",
                "from_email": "'"${ALERT_FROM_EMAIL:-monitoring@localhost}"'",
                "to_email": "'"${ALERT_TO_EMAIL:-admin@localhost}"'"
            }
        }' >/dev/null 2>&1 || log "WARNING: SMTP secret setup failed"
    
    # Setup PostgreSQL secrets
    log "Setting up Vault secrets for PostgreSQL..."
    curl -s -X POST "${vault_url}/v1/secret/data/monitoring/postgres" \
        -H "X-Vault-Token: ${vault_token}" \
        -H "Content-Type: application/json" \
        -d '{
            "data": {
                "host": "localhost",
                "port": "5433",
                "database": "'"${POSTGRES_DB:-vrooli}"'",
                "username": "'"${POSTGRES_USER:-vrooli}"'",
                "password": "'"${POSTGRES_PASSWORD:-vrooli}"'"
            }
        }' >/dev/null 2>&1 || log "WARNING: PostgreSQL secret setup failed"
    
    # Setup Redis secrets
    log "Setting up Vault secrets for Redis..."
    curl -s -X POST "${vault_url}/v1/secret/data/monitoring/redis" \
        -H "X-Vault-Token: ${vault_token}" \
        -H "Content-Type: application/json" \
        -d '{
            "data": {
                "host": "localhost",
                "port": "6380",
                "password": ""
            }
        }' >/dev/null 2>&1 || log "WARNING: Redis secret setup failed"
    
    log "✓ Vault secrets setup completed"
}

# Setup known services configuration from resource-registry.json
setup_known_services() {
    local registry_file="${SCENARIO_DIR}/initialization/configuration/resource-registry.json"
    
    if [[ ! -f "$registry_file" ]]; then
        log "WARNING: Resource registry file not found: $registry_file"
        return 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        log "WARNING: jq not available, skipping registry setup"
        return 1
    fi
    
    if command -v psql >/dev/null 2>&1; then
        log "Setting up known services from resource registry..."
        
        # Extract port mappings from resource-registry.json
        local port_mappings
        port_mappings=$(jq -c '.portMappings | to_entries | map({key: .value.service, value: (.key | tonumber)}) | from_entries' "$registry_file" 2>/dev/null || echo '{}')
        
        # Insert into database
        PGPASSWORD="${POSTGRES_PASSWORD:-vrooli}" psql \
            -h "${POSTGRES_HOST:-localhost}" \
            -p "${POSTGRES_PORT:-5433}" \
            -U "${POSTGRES_USER:-vrooli}" \
            -d "${POSTGRES_DB:-vrooli}" \
            -c "INSERT INTO resource_monitoring.system_config (key, value, description) VALUES (
                'known_services', 
                '${port_mappings}',
                'Known service port mappings from resource-registry.json'
            ) ON CONFLICT (key) DO UPDATE SET 
                value = EXCLUDED.value, 
                updated_at = NOW();" \
            2>/dev/null || log "WARNING: Known services setup failed"
            
        # Also populate initial resources from registry
        log "Populating initial resources from registry..."
        jq -r '.portMappings | to_entries[] | @base64' "$registry_file" 2>/dev/null | while read -r entry; do
            local decoded_entry
            decoded_entry=$(echo "$entry" | base64 --decode 2>/dev/null || continue)
            
            local port service_name resource_type
            port=$(echo "$decoded_entry" | jq -r '.key' 2>/dev/null || continue)
            service_name=$(echo "$decoded_entry" | jq -r '.value.service' 2>/dev/null || continue)
            resource_type=$(echo "$decoded_entry" | jq -r '.value.type' 2>/dev/null || continue)
            
            # Get custom health endpoint or use default
            local health_endpoint
            health_endpoint=$(jq -r --arg service "$service_name" '.healthEndpoints.custom[$service] // "/health"' "$registry_file" 2>/dev/null || echo '/health')
            
            # Determine display name
            local display_names='{
                "ollama": "Ollama LLM Service",
                "postgres": "PostgreSQL Database", 
                "redis": "Redis Cache",
                "questdb": "QuestDB Time-Series",
                "minio": "MinIO Object Storage",
                "qdrant": "Qdrant Vector Database",
                "vault": "HashiCorp Vault",
                "n8n": "n8n Workflow Automation",
                "node-red": "Node-RED Flow Engine", 
                "windmill": "Windmill Platform",
                "huginn": "Huginn Event Processor",
                "browserless": "Browserless Chrome",
                "agent-s2": "Agent-S2 Desktop Automation",
                "claude-code": "Claude Code Assistant",
                "judge0": "Judge0 Code Executor", 
                "searxng": "SearXNG Search Engine",
                "whisper": "Whisper Speech-to-Text",
                "comfyui": "ComfyUI Image Generation",
                "unstructured-io": "Unstructured.io Document Processing"
            }'
            local display_name
            display_name=$(echo "$display_names" | jq -r --arg service "$service_name" '.[$service] // ($service | ascii_upcase)' 2>/dev/null || echo "$service_name")
            
            # Determine criticality (core services are critical)
            local is_critical=false
            case "$service_name" in
                postgres|redis|questdb|vault|n8n)
                    is_critical=true
                    ;;
            esac
            
            # Insert resource into database
            PGPASSWORD="${POSTGRES_PASSWORD:-vrooli}" psql \
                -h "${POSTGRES_HOST:-localhost}" \
                -p "${POSTGRES_PORT:-5433}" \
                -U "${POSTGRES_USER:-vrooli}" \
                -d "${POSTGRES_DB:-vrooli}" \
                -c "INSERT INTO resource_monitoring.resources (
                    resource_name, resource_type, display_name, description, 
                    port, base_url, health_check_endpoint, is_critical, 
                    auto_discovered, config
                ) VALUES (
                    '$service_name', '$resource_type', '$display_name', 
                    'Resource from static registry: $display_name',
                    $port, 'http://localhost:$port', 
                    $(if [[ "$health_endpoint" == "null" ]]; then echo "NULL"; else echo "'$health_endpoint'"; fi), 
                    $is_critical, false, '{\"source\": \"static_registry\"}'
                ) ON CONFLICT (resource_name) DO UPDATE SET
                    resource_type = EXCLUDED.resource_type,
                    port = EXCLUDED.port,
                    base_url = EXCLUDED.base_url,
                    health_check_endpoint = EXCLUDED.health_check_endpoint,
                    updated_at = NOW();" \
                2>/dev/null || log "WARNING: Failed to insert resource: $service_name"
        done
        
        log "✓ Known services and initial resources configured from registry"
    fi
}

# Execute credential and secret setup
setup_n8n_credentials
setup_vault_secrets
setup_known_services

log "✓ Credentials and secrets setup completed"

# Step 4: Trigger initial discovery
log "Step 4: Running initial resource discovery..."

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
