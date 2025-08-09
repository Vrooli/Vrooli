#!/usr/bin/env bash
# Prompt Manager Startup Script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

log::info "Starting Prompt Manager..."

# Initialize database
log::info "Initializing PostgreSQL database..."
if [[ -f "${SCENARIO_DIR}/initialization/storage/postgres/schema.sql" ]]; then
    if psql -U postgres -d prompt_manager < "${SCENARIO_DIR}/initialization/storage/postgres/schema.sql" 2>/dev/null; then
        log::success "Database schema initialized"
    else
        log::warning "Database initialization failed or already exists"
    fi
else
    log::error "Schema file not found: ${SCENARIO_DIR}/initialization/storage/postgres/schema.sql"
fi

# NOTE: Docker operations should be handled by the resource management system
# This is a simplified approach for the scenario
log::info "Checking if Qdrant vector database is available..."
if ! curl -sf "http://localhost:6333/health" >/dev/null 2>&1; then
    log::warning "Qdrant not available - vector operations will be limited"
else
    log::success "Qdrant vector database is available"
    
    # Create Qdrant collection
    log::info "Creating Qdrant collection..."
    if curl -X PUT "http://localhost:6333/collections/prompts" \
        -H "Content-Type: application/json" \
        -d '{
            "vectors": {
                "size": 1536,
                "distance": "Cosine"
            }
        }' >/dev/null 2>&1; then
        log::success "Qdrant collection created"
    else
        log::warning "Failed to create Qdrant collection (may already exist)"
    fi
fi

# Check Ollama availability
log::info "Checking Ollama availability..."
if curl -sf "http://localhost:11434/api/tags" >/dev/null 2>&1; then
    log::success "Ollama is available for prompt testing"
else
    log::warning "Ollama not available - prompt testing will be limited"
fi

# Initialize Redis session store
log::info "Initializing Redis session store..."
if redis-cli -n 2 SET "prompt_manager:initialized" "true" >/dev/null 2>&1; then
    log::success "Redis session store initialized"
else
    log::warning "Redis initialization failed - sessions may not work properly"
fi

# Health check
log::info "Performing health check..."
if curl -sf "http://localhost:8085/health" >/dev/null 2>&1; then
    log::success "API health check passed"
else
    log::warning "API not yet available"
fi

log::success "Prompt Manager startup process completed!"
log::info "Dashboard should be available at: http://localhost:3005"
log::info ""
log::info "Default credentials:"
log::info "  Email: admin@promptmanager.local"
log::info "  Password: ChangeMeNow123!"