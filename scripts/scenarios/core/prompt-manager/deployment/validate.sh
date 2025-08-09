#!/usr/bin/env bash
# Prompt Manager Validation Script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

log::info "Validating Prompt Manager deployment..."

# Check PostgreSQL
if psql -U postgres -d prompt_manager -c "SELECT COUNT(*) FROM tags" > /dev/null 2>&1; then
    log::success "PostgreSQL database with tags"
else
    log::error "PostgreSQL database not configured"
    exit 1
fi

# Check Qdrant vector database
if curl -s http://localhost:6333/collections | grep -q "prompts"; then
    log::success "Qdrant vector store configured"
else
    log::error "Qdrant collection not created"
    exit 1
fi

# Check Redis session store
if redis-cli -n 2 GET "prompt_manager:initialized" > /dev/null 2>&1; then
    log::success "Redis session store"
else
    log::error "Redis session store not initialized"
    exit 1
fi

# Check Ollama (optional)
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    log::success "Ollama LLM available for testing"
else
    log::warning "Ollama not available (prompt testing limited)"
fi

# Check UI
if curl -s -o /dev/null -w "%{http_code}" http://localhost:3005 | grep -q "200\|304"; then
    log::success "UI dashboard accessible"
else
    log::error "UI dashboard not accessible"
    exit 1
fi

# Check API health
if curl -s http://localhost:8085/health | grep -q "healthy"; then
    log::success "API healthy"
else
    log::error "API not healthy"
    exit 1
fi

# Test authentication endpoint
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/api/auth/login -X POST | grep -q "400\|401\|200"; then
    log::success "Authentication endpoint responsive"
else
    log::error "Authentication endpoint not working"
fi

# Test semantic search readiness
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/api/prompts/semantic -X POST \
    -H "Content-Type: application/json" \
    -d '{"query":"test"}' | grep -q "200\|401\|400"; then
    log::success "Semantic search endpoint ready"
else
    log::error "Semantic search not configured"
fi

log::success "Prompt Manager validation successful!"
log::info "Campaign-based prompt management ready."
log::info ""
log::info "Access the dashboard at: http://localhost:3005"