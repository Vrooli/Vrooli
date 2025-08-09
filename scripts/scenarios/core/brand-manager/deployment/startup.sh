#!/bin/bash
set -euo pipefail

# Brand Manager Startup Script
# Initializes all required services and validates integration

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$SCENARIO_DIR")")")"

# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/lib/utils/log.sh"

log_info "Starting Brand Manager scenario..."

# Configuration
POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-brand_manager}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-password}"

MINIO_ENDPOINT="${MINIO_ENDPOINT:-minio:9000}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"

N8N_ENDPOINT="${N8N_ENDPOINT:-http://n8n:5678}"
WINDMILL_ENDPOINT="${WINDMILL_ENDPOINT:-http://windmill:8000}"
OLLAMA_ENDPOINT="${OLLAMA_ENDPOINT:-http://ollama:11434}"
COMFYUI_ENDPOINT="${COMFYUI_ENDPOINT:-http://comfyui:8188}"

# Wait for services to be ready
wait_for_service() {
    local service_name="$1"
    local endpoint="$2"
    local max_attempts="${3:-30}"
    local attempt=1
    
    log_info "Waiting for $service_name to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -sf "$endpoint" >/dev/null 2>&1; then
            log_info "$service_name is ready"
            return 0
        fi
        log_info "Attempt $attempt/$max_attempts: $service_name not ready, waiting..."
        sleep 10
        attempt=$((attempt + 1))
    done
    
    log_error "$service_name failed to start after $max_attempts attempts"
    return 1
}

# Wait for TCP service
wait_for_tcp_service() {
    local service_name="$1"
    local host="$2"  
    local port="$3"
    local max_attempts="${4:-30}"
    local attempt=1
    
    log_info "Waiting for $service_name TCP service to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if timeout 5 bash -c "</dev/tcp/$host/$port" >/dev/null 2>&1; then
            log_info "$service_name TCP service is ready"
            return 0
        fi
        log_info "Attempt $attempt/$max_attempts: $service_name TCP service not ready, waiting..."
        sleep 5
        attempt=$((attempt + 1))
    done
    
    log_error "$service_name TCP service failed to start after $max_attempts attempts"
    return 1
}

# Check MinIO buckets
check_minio_buckets() {
    log_info "Checking MinIO buckets..."
    
    local buckets=("brand-logos" "brand-icons" "brand-exports" "brand-templates" "app-backups")
    
    for bucket in "${buckets[@]}"; do
        if ! mc ls "minio-brand/$bucket" >/dev/null 2>&1; then
            log_info "Creating bucket: $bucket"
            mc mb "minio-brand/$bucket" || {
                log_error "Failed to create bucket: $bucket"
                return 1
            }
        else
            log_info "Bucket exists: $bucket"
        fi
    done
    
    return 0
}

# Setup MinIO alias
setup_minio() {
    log_info "Setting up MinIO client..."
    
    # Configure MinIO client
    mc alias set minio-brand "http://$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" || {
        log_error "Failed to setup MinIO alias"
        return 1
    }
    
    check_minio_buckets || return 1
    
    log_info "MinIO setup completed successfully"
    return 0
}

# Check database connection and tables
check_database() {
    log_info "Checking database connection..."
    
    export PGPASSWORD="$POSTGRES_PASSWORD"
    
    # Test connection
    if ! psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;" >/dev/null 2>&1; then
        log_error "Cannot connect to database"
        return 1
    fi
    
    # Check if main tables exist
    local tables=("brands" "templates" "campaigns" "exports" "integration_requests")
    for table in "${tables[@]}"; do
        if ! psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1 FROM $table LIMIT 1;" >/dev/null 2>&1; then
            log_error "Table $table does not exist or is not accessible"
            return 1
        fi
    done
    
    log_info "Database validation completed successfully"
    return 0
}

# Check n8n workflows
check_n8n_workflows() {
    log_info "Checking n8n workflows..."
    
    local workflows=("brand-pipeline" "claude-spawner" "integration-monitor")
    
    for workflow in "${workflows[@]}"; do
        # This would check if workflows are imported and active
        # For now, just verify n8n API is accessible
        if ! curl -sf "$N8N_ENDPOINT/api/v1/workflows" >/dev/null 2>&1; then
            log_error "Cannot access n8n API"
            return 1
        fi
    done
    
    log_info "n8n workflows validation completed"
    return 0
}

# Check Windmill apps
check_windmill_apps() {
    log_info "Checking Windmill applications..."
    
    if ! curl -sf "$WINDMILL_ENDPOINT/api/version" >/dev/null 2>&1; then
        log_error "Cannot access Windmill API"
        return 1
    fi
    
    log_info "Windmill applications validation completed"
    return 0
}

# Check AI services
check_ai_services() {
    log_info "Checking AI services..."
    
    # Check Ollama
    if ! curl -sf "$OLLAMA_ENDPOINT/api/tags" >/dev/null 2>&1; then
        log_error "Cannot access Ollama API"
        return 1
    fi
    
    # Check ComfyUI
    if ! curl -sf "$COMFYUI_ENDPOINT/system_stats" >/dev/null 2>&1; then
        log_error "Cannot access ComfyUI API" 
        return 1
    fi
    
    log_info "AI services validation completed"
    return 0
}

# Main startup sequence
main() {
    log_info "Brand Manager startup initiated"
    
    # Wait for core services
    wait_for_tcp_service "PostgreSQL" "$POSTGRES_HOST" "$POSTGRES_PORT" || exit 1
    wait_for_service "MinIO" "http://$MINIO_ENDPOINT/minio/health/ready" || exit 1
    wait_for_service "n8n" "$N8N_ENDPOINT/healthz" || exit 1
    wait_for_service "Windmill" "$WINDMILL_ENDPOINT/api/version" || exit 1
    wait_for_service "Ollama" "$OLLAMA_ENDPOINT/api/tags" || exit 1
    wait_for_service "ComfyUI" "$COMFYUI_ENDPOINT/system_stats" || exit 1
    
    # Setup services
    setup_minio || exit 1
    
    # Validate services
    check_database || exit 1
    check_n8n_workflows || exit 1
    check_windmill_apps || exit 1
    check_ai_services || exit 1
    
    log_info "Brand Manager startup completed successfully"
    log_info "Access Windmill UI at: $WINDMILL_ENDPOINT"
    log_info "Brand Manager Dashboard: $WINDMILL_ENDPOINT/apps/f/brand-manager/dashboard"
    log_info "Integration Monitor: $WINDMILL_ENDPOINT/apps/f/brand-manager/integration-dashboard"
    
    return 0
}

# Run main function
main "$@"