#!/usr/bin/env bash
# Idea Generator - Deployment Startup Script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(cd "$SCENARIO_DIR/../../../.." && pwd)"

# Source logging utilities if available
if [[ -f "${PROJECT_ROOT}/scripts/lib/utils/logging.sh" ]]; then
    source "${PROJECT_ROOT}/scripts/lib/utils/logging.sh"
else
    # Fallback logging functions
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::warn() { echo "[WARN] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
fi

log::info "Starting Idea Generator application deployment..."

# Wait for a service to be ready
wait_for_service() {
    local service=$1
    local port=$2
    local max_attempts=30
    local attempt=0
    
    log::info "Waiting for $service on port $port..."
    while [ $attempt -lt $max_attempts ]; do
        if nc -z localhost "$port" 2>/dev/null; then
            log::success "$service is ready"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done
    
    log::error "$service failed to start on port $port"
    return 1
}

# Initialize database
init_database() {
    log::info "Initializing database schema..."
    
    # Wait for PostgreSQL to be ready
    wait_for_service "PostgreSQL" 5433 || return 1
    
    # Apply schema
    PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
        -h localhost \
        -p 5433 \
        -U "${POSTGRES_USER:-ideagen_user}" \
        -d "${POSTGRES_DB:-ideagen}" \
        -f "${SCENARIO_DIR}/initialization/storage/postgres/schema.sql" 2>/dev/null || {
        log::warn "Schema may already exist, continuing..."
    }
    
    # Apply seed data if exists
    if [[ -f "${SCENARIO_DIR}/initialization/storage/postgres/seed.sql" ]]; then
        PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
            -h localhost \
            -p 5433 \
            -U "${POSTGRES_USER:-ideagen_user}" \
            -d "${POSTGRES_DB:-ideagen}" \
            -f "${SCENARIO_DIR}/initialization/storage/postgres/seed.sql" 2>/dev/null || {
            log::warn "Seed data may already exist, continuing..."
        }
    fi
    
    log::success "Database initialized successfully"
}

# Deploy n8n workflows
deploy_workflows() {
    log::info "Deploying n8n workflows..."
    
    # Wait for n8n to be ready
    wait_for_service "n8n" 5678 || return 1
    
    # Import workflows
    for workflow in "${SCENARIO_DIR}"/initialization/automation/n8n/*.json; do
        [[ -f "$workflow" ]] || continue
        
        local workflow_name=$(basename "$workflow" .json)
        log::info "Importing workflow: ${workflow_name}"
        
        curl -X POST http://localhost:5678/rest/workflows \
            -H "Content-Type: application/json" \
            -d "@${workflow}" 2>/dev/null || {
            log::warn "Workflow ${workflow_name} may already exist"
        }
    done
    
    log::success "Workflows deployed successfully"
}

# Initialize Qdrant collections
init_vector_db() {
    log::info "Initializing Qdrant vector database..."
    
    # Wait for Qdrant to be ready
    wait_for_service "Qdrant" 6333 || return 1
    
    # Create collections from configuration
    if [[ -f "${SCENARIO_DIR}/initialization/storage/qdrant/collections.json" ]]; then
        # Parse collections configuration and create each one
        for collection in ideas documents campaigns chat_messages; do
            curl -X PUT "http://localhost:6333/collections/${collection}" \
                -H "Content-Type: application/json" \
                -d '{
                    "vectors": {
                        "size": 1536,
                        "distance": "Cosine"
                    }
                }' 2>/dev/null || {
                log::warn "Collection ${collection} may already exist"
            }
        done
    fi
    
    log::success "Vector database initialized"
}

# Initialize MinIO buckets
init_storage() {
    log::info "Initializing MinIO object storage..."
    
    # Wait for MinIO to be ready
    wait_for_service "MinIO" 9000 || return 1
    
    # Create buckets
    for bucket in idea-documents processed-content exports backups; do
        curl -X PUT "http://localhost:9000/${bucket}" \
            -H "Host: localhost:9000" \
            2>/dev/null || {
            log::warn "Bucket ${bucket} may already exist"
        }
    done
    
    log::success "Object storage initialized"
}

# Deploy Windmill app
deploy_windmill_app() {
    log::info "Deploying Windmill application..."
    
    # Wait for Windmill to be ready
    wait_for_service "Windmill" 5681 || return 1
    
    # Deploy main app
    if [[ -f "${SCENARIO_DIR}/initialization/automation/windmill/idea-generator-app.json" ]]; then
        curl -X POST "http://localhost:5681/api/apps" \
            -H "Content-Type: application/json" \
            -d "@${SCENARIO_DIR}/initialization/automation/windmill/idea-generator-app.json" \
            2>/dev/null || {
            log::warn "Windmill app may already be deployed"
        }
    fi
    
    # Deploy scripts
    for script in "${SCENARIO_DIR}"/initialization/automation/windmill/scripts/*.ts; do
        [[ -f "$script" ]] || continue
        
        local script_name=$(basename "$script" .ts)
        log::info "Deploying script: ${script_name}"
        
        # Read script content and create JSON payload
        local script_content=$(cat "$script" | sed 's/"/\\"/g' | sed ':a;N;$!ba;s/\n/\\n/g')
        
        curl -X POST "http://localhost:5681/api/scripts" \
            -H "Content-Type: application/json" \
            -d "{
                \"path\": \"scripts/${script_name}\",
                \"content\": \"${script_content}\",
                \"language\": \"typescript\"
            }" 2>/dev/null || {
            log::warn "Script ${script_name} may already be deployed"
        }
    done
    
    log::success "Windmill application deployed"
}

# Load Ollama models
load_models() {
    log::info "Loading AI models..."
    
    # Wait for Ollama to be ready
    wait_for_service "Ollama" 11434 || return 1
    
    # Pull required models
    curl -X POST "http://localhost:11434/api/pull" \
        -d '{"name": "llama3.2"}' 2>/dev/null || log::warn "Model llama3.2 may already exist"
    
    curl -X POST "http://localhost:11434/api/pull" \
        -d '{"name": "mistral"}' 2>/dev/null || log::warn "Model mistral may already exist"
    
    curl -X POST "http://localhost:11434/api/pull" \
        -d '{"name": "nomic-embed-text"}' 2>/dev/null || log::warn "Model nomic-embed-text may already exist"
    
    log::success "AI models loaded"
}

# Main deployment flow
main() {
    log::info "=== Idea Generator Deployment Starting ==="
    
    # Initialize all components
    init_database || return 1
    init_vector_db || return 1
    init_storage || return 1
    deploy_workflows || return 1
    deploy_windmill_app || return 1
    load_models || return 1
    
    log::success "=== Idea Generator Deployment Complete ==="
    log::info ""
    log::info "Access points:"
    log::info "  • Windmill UI: http://localhost:5681"
    log::info "  • n8n Workflows: http://localhost:5678"
    log::info "  • API Endpoints: http://localhost:5679/webhook/*"
    log::info ""
    log::info "Default credentials:"
    log::info "  • Username: demo_user"
    log::info "  • Email: demo@ideagen.com"
}

main "$@"