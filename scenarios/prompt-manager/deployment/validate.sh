#!/usr/bin/env bash
# Prompt Manager Validation Script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Import logging utilities if available
if [[ -f "${SCRIPT_DIR}/../../../../lib/utils/var.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"
    # shellcheck disable=SC1091
    source "${var_LOG_FILE:-/dev/null}"
else
    # Fallback logging functions
    log::info() { echo "ℹ️  $1"; }
    log::success() { echo "✅ $1"; }
    log::warning() { echo "⚠️  $1"; }
    log::error() { echo "❌ $1"; }
fi

log::info "Validating Prompt Manager deployment..."

VALIDATION_FAILED=false

# Check PostgreSQL database and schema
log::info "Checking PostgreSQL database..."
if psql -U postgres -d prompt_manager -c "SELECT COUNT(*) FROM campaigns; SELECT COUNT(*) FROM prompts; SELECT COUNT(*) FROM tags;" > /dev/null 2>&1; then
    campaign_count=$(psql -U postgres -d prompt_manager -t -c "SELECT COUNT(*) FROM campaigns;" | xargs)
    prompt_count=$(psql -U postgres -d prompt_manager -t -c "SELECT COUNT(*) FROM prompts;" | xargs)
    tag_count=$(psql -U postgres -d prompt_manager -t -c "SELECT COUNT(*) FROM tags;" | xargs)
    
    log::success "PostgreSQL database working - $campaign_count campaigns, $prompt_count prompts, $tag_count tags"
else
    log::error "PostgreSQL database not configured properly"
    VALIDATION_FAILED=true
fi

# Check Qdrant vector database
log::info "Checking Qdrant vector database..."
if curl -s http://localhost:6333/collections 2>/dev/null | grep -q "prompts"; then
    log::success "Qdrant vector store configured with prompts collection"
else
    log::warning "Qdrant collection not found (semantic search will be limited)"
fi

# Check API server
log::info "Checking API server..."
if curl -s http://localhost:8085/health 2>/dev/null | grep -q "healthy"; then
    log::success "API server is healthy"
    
    # Test key endpoints
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/api/campaigns | grep -q "200"; then
        log::success "Campaigns endpoint working"
    else
        log::error "Campaigns endpoint failed"
        VALIDATION_FAILED=true
    fi
    
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/api/prompts | grep -q "200"; then
        log::success "Prompts endpoint working"
    else
        log::error "Prompts endpoint failed"
        VALIDATION_FAILED=true
    fi
    
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/api/tags | grep -q "200"; then
        log::success "Tags endpoint working"
    else
        log::error "Tags endpoint failed"
        VALIDATION_FAILED=true
    fi
    
else
    log::error "API server not healthy or not responding"
    VALIDATION_FAILED=true
fi

# Check React UI
log::info "Checking React UI..."
if curl -s -o /dev/null -w "%{http_code}" http://localhost:3005 2>/dev/null | grep -q "200\|302"; then
    log::success "React UI accessible"
else
    log::error "React UI not accessible"
    VALIDATION_FAILED=true
fi

# Check CLI availability
log::info "Checking CLI tool..."
if command -v prompt-manager >/dev/null 2>&1; then
    log::success "CLI tool available in PATH"
    
    # Test CLI connectivity to API
    if timeout 10s prompt-manager status >/dev/null 2>&1; then
        log::success "CLI can communicate with API"
    else
        log::warning "CLI cannot communicate with API (may need to wait for full startup)"
    fi
else
    if [[ -f "${SCENARIO_DIR}/cli/prompt-manager" ]]; then
        log::success "CLI tool found (not in PATH, use ./cli/prompt-manager)"
    else
        log::warning "CLI tool not found or not installed"
    fi
fi

# Check optional services
log::info "Checking optional services..."

# Ollama
if curl -s http://localhost:11434/api/tags >/dev/null 2>&1; then
    log::success "Ollama LLM available for prompt testing"
else
    log::info "Ollama not available (optional - prompt testing will be limited)"
fi

# Process validation
log::info "Checking running processes..."
if [[ -f "${SCENARIO_DIR}/api/api.pid" ]]; then
    api_pid=$(cat "${SCENARIO_DIR}/api/api.pid")
    if kill -0 "$api_pid" 2>/dev/null; then
        log::success "API server process running (PID: $api_pid)"
    else
        log::warning "API server PID file exists but process not running"
    fi
fi

if [[ -f "${SCENARIO_DIR}/ui/ui.pid" ]]; then
    ui_pid=$(cat "${SCENARIO_DIR}/ui/ui.pid")
    if kill -0 "$ui_pid" 2>/dev/null; then
        log::success "UI server process running (PID: $ui_pid)"
    else
        log::warning "UI server PID file exists but process not running"
    fi
fi

# File structure validation
log::info "Checking file structure..."
required_files=(
    "${SCENARIO_DIR}/api/main.go"
    "${SCENARIO_DIR}/cli/prompt-manager"
    "${SCENARIO_DIR}/ui/src/App.js"
    "${SCENARIO_DIR}/initialization/storage/postgres/schema.sql"
    "${SCENARIO_DIR}/initialization/storage/postgres/seed.sql"
)

for file in "${required_files[@]}"; do
    if [[ -f "$file" ]]; then
        log::success "Required file present: $(basename "$file")"
    else
        log::error "Missing required file: $file"
        VALIDATION_FAILED=true
    fi
done

# Final validation result
echo ""
if [[ "$VALIDATION_FAILED" == "true" ]]; then
    log::error "❌ Validation failed! Please check the errors above."
    echo ""
    echo "🔧 Troubleshooting:"
    echo "   • Check logs: ${SCENARIO_DIR}/api/api.log, ${SCENARIO_DIR}/ui/ui.log"
    echo "   • Restart services: bash ${SCRIPT_DIR}/startup.sh"
    echo "   • Check resource availability: postgres, ollama, qdrant"
    exit 1
else
    log::success "🎉 Validation successful!"
    echo ""
    echo "✅ Prompt Manager is fully operational!"
    echo ""
    echo "📍 Access Points:"
    echo "   • Web UI: http://localhost:3005"
    echo "   • API: http://localhost:8085"
    echo "   • CLI: prompt-manager help"
    echo ""
    echo "🚀 Ready to manage your prompts!"
fi