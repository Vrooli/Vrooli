#!/usr/bin/env bash
# Agent Metareasoning Manager - Deployment Startup Script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Source logging utilities
source "${SCRIPT_DIR}/../../../lib/utils/logging.sh"

log::info "Starting Agent Metareasoning Manager deployment..."

# Start required resources in order
start_resources() {
    local resources=("postgres" "redis" "ollama" "n8n" "windmill")
    
    for resource in "${resources[@]}"; do
        log::info "Starting ${resource}..."
        "${SCRIPT_DIR}/../../../resources/${resource}/manage.sh" start || {
            log::error "Failed to start ${resource}"
            return 1
        }
    done
}

# Initialize database
init_database() {
    log::info "Initializing database schema..."
    
    # Wait for PostgreSQL to be ready
    local retries=30
    while ! pg_isready -h localhost -p 5432 >/dev/null 2>&1; do
        ((retries--)) || {
            log::error "PostgreSQL failed to start"
            return 1
        }
        sleep 2
    done
    
    # Create database and user
    psql -h localhost -p 5432 -U postgres <<EOF || true
CREATE DATABASE metareasoning;
CREATE USER meta_user WITH PASSWORD '${POSTGRES_PASSWORD:-metapass}';
GRANT ALL PRIVILEGES ON DATABASE metareasoning TO meta_user;
EOF
    
    # Apply schema
    psql -h localhost -p 5432 -U meta_user -d metareasoning \
        -f "${SCENARIO_DIR}/initialization/storage/postgres/schema.sql" || {
        log::error "Failed to initialize database schema"
        return 1
    }
    
    # Apply seed data if exists
    if [[ -f "${SCENARIO_DIR}/initialization/storage/postgres/seed.sql" ]]; then
        psql -h localhost -p 5432 -U meta_user -d metareasoning \
            -f "${SCENARIO_DIR}/initialization/storage/postgres/seed.sql"
    fi
    
    log::success "Database initialized successfully"
}

# Deploy n8n workflows
deploy_n8n_workflows() {
    log::info "Deploying n8n metareasoning workflows..."
    
    # Wait for n8n to be ready
    local retries=30
    while ! curl -f http://localhost:5678/healthz >/dev/null 2>&1; do
        ((retries--)) || {
            log::error "n8n failed to start"
            return 1
        }
        sleep 2
    done
    
    # Import workflows
    for workflow in "${SCENARIO_DIR}"/initialization/automation/n8n/*.json; do
        [[ -f "$workflow" ]] || continue
        
        local workflow_name=$(basename "$workflow" .json)
        log::info "Importing n8n workflow: ${workflow_name}"
        
        curl -X POST http://localhost:5678/rest/workflows \
            -H "Content-Type: application/json" \
            -d "@${workflow}" || {
            log::warn "Failed to import workflow: ${workflow_name}"
        }
    done
    
    log::success "n8n workflows deployed successfully"
}

# Deploy Windmill flows
deploy_windmill_flows() {
    log::info "Deploying Windmill orchestration flows..."
    
    # Wait for Windmill to be ready
    local retries=30
    while ! curl -f http://localhost:8000/api/version >/dev/null 2>&1; do
        ((retries--)) || {
            log::error "Windmill failed to start"
            return 1
        }
        sleep 2
    done
    
    # Import Windmill flows
    for flow in "${SCENARIO_DIR}"/initialization/automation/windmill/*.json; do
        [[ -f "$flow" ]] || continue
        
        local flow_name=$(basename "$flow" .json)
        log::info "Importing Windmill flow: ${flow_name}"
        
        curl -X POST http://localhost:8000/api/w/main/flows/create \
            -H "Content-Type: application/json" \
            -d "@${flow}" || {
            log::warn "Failed to import flow: ${flow_name}"
        }
    done
    
    log::success "Windmill flows deployed successfully"
}

# Load AI models
load_models() {
    log::info "Loading AI models for metareasoning..."
    
    # Pull required models
    ollama pull llama3.2 || log::warn "Failed to pull llama3.2 (may already exist)"
    ollama pull mistral || log::warn "Failed to pull mistral (may already exist)"
    ollama pull codellama || log::warn "Failed to pull codellama (may already exist)"
    
    log::success "AI models loaded"
}

# Load initial templates and prompts
load_initial_data() {
    log::info "Loading initial templates and prompts..."
    
    # Load reasoning templates
    if [[ -f "${SCENARIO_DIR}/initialization/configuration/reasoning-templates.json" ]]; then
        curl -X POST http://localhost:8080/api/templates/import \
            -H "Content-Type: application/json" \
            -d "@${SCENARIO_DIR}/initialization/configuration/reasoning-templates.json" || {
            log::warn "Failed to import reasoning templates"
        }
    fi
    
    # Load prompt library
    if [[ -f "${SCENARIO_DIR}/initialization/configuration/prompt-library.json" ]]; then
        curl -X POST http://localhost:8080/api/prompts/import \
            -H "Content-Type: application/json" \
            -d "@${SCENARIO_DIR}/initialization/configuration/prompt-library.json" || {
            log::warn "Failed to import prompt library"
        }
    fi
    
    log::success "Initial data loaded"
}

# Main deployment flow
main() {
    log::info "=== Agent Metareasoning Manager Deployment Starting ==="
    
    start_resources || return 1
    init_database || return 1
    deploy_n8n_workflows || return 1
    deploy_windmill_flows || return 1
    load_models || return 1
    
    # Start application servers
    log::info "Starting application servers..."
    cd "${SCENARIO_DIR}/.."
    npm run dev &
    
    # Load initial data after servers are running
    sleep 10
    load_initial_data || log::warn "Initial data loading had issues"
    
    log::success "=== Agent Metareasoning Manager Deployment Complete ==="
    log::info "Dashboard available at http://localhost:3000"
    log::info "n8n workflows at http://localhost:5678"
    log::info "Windmill at http://localhost:8000"
    log::info "API server at http://localhost:8080"
}

main "$@"