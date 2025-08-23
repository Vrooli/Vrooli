#!/usr/bin/env bash
# Prompt Manager Startup Script
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

log::info "Starting Prompt Manager..."

# Create database if it doesn't exist
log::info "Creating prompt_manager database..."
if psql -U postgres -lqt | cut -d \| -f 1 | grep -qw prompt_manager; then
    log::success "Database prompt_manager already exists"
else
    if createdb -U postgres prompt_manager; then
        log::success "Database prompt_manager created"
    else
        log::error "Failed to create database"
        exit 1
    fi
fi

# Initialize database schema
log::info "Initializing PostgreSQL schema..."
if [[ -f "${SCENARIO_DIR}/initialization/storage/postgres/schema.sql" ]]; then
    if psql -U postgres -d prompt_manager < "${SCENARIO_DIR}/initialization/storage/postgres/schema.sql" 2>/dev/null; then
        log::success "Database schema initialized"
    else
        log::warning "Schema initialization failed (may already exist)"
    fi
else
    log::error "Schema file not found: ${SCENARIO_DIR}/initialization/storage/postgres/schema.sql"
fi

# Load seed data
log::info "Loading seed data..."
if [[ -f "${SCENARIO_DIR}/initialization/storage/postgres/seed.sql" ]]; then
    if psql -U postgres -d prompt_manager < "${SCENARIO_DIR}/initialization/storage/postgres/seed.sql" 2>/dev/null; then
        log::success "Seed data loaded"
    else
        log::warning "Seed data loading failed (may already exist)"
    fi
fi

# Set up Qdrant vector database
log::info "Setting up Qdrant vector database..."
if curl -sf "http://localhost:6333/health" >/dev/null 2>&1; then
    log::success "Qdrant is available"
    
    # Create Qdrant collection
    log::info "Creating Qdrant collection for prompts..."
    if curl -X PUT "http://localhost:6333/collections/prompts" \
        -H "Content-Type: application/json" \
        -d '{
            "vectors": {
                "size": 384,
                "distance": "Cosine"
            },
            "optimizers_config": {
                "default_segment_number": 2
            },
            "replication_factor": 1
        }' >/dev/null 2>&1; then
        log::success "Qdrant collection created"
    else
        log::warning "Failed to create Qdrant collection (may already exist)"
    fi
else
    log::warning "Qdrant not available - semantic search will be limited"
fi

# Check Ollama availability
log::info "Checking Ollama LLM service..."
if curl -sf "http://localhost:11434/api/tags" >/dev/null 2>&1; then
    log::success "Ollama is available for prompt testing"
    
    # Check if a small model is available for embeddings
    if curl -s "http://localhost:11434/api/tags" | grep -q "nomic-embed-text"; then
        log::success "Embedding model available"
    else
        log::info "Consider pulling a small embedding model: ollama pull nomic-embed-text"
    fi
else
    log::warning "Ollama not available - prompt testing will be limited"
fi

# Start Go API server
log::info "Starting API server..."
cd "${SCENARIO_DIR}/api"

# Set environment variables
export PORT=8085
export POSTGRES_URL="postgres://postgres:postgres@localhost:5433/prompt_manager?sslmode=disable"
export QDRANT_URL="http://localhost:6333"
export OLLAMA_URL="http://localhost:11434"

# Build and start Go server in background
if command -v go >/dev/null 2>&1; then
    log::info "Building Go API server..."
    if go mod tidy && go build -o prompt-manager-api main.go; then
        log::success "API server built successfully"
        
        # Start server in background
        nohup ./prompt-manager-api > api.log 2>&1 &
        API_PID=$!
        echo $API_PID > api.pid
        log::success "API server started (PID: $API_PID)"
    else
        log::error "Failed to build API server"
        exit 1
    fi
else
    log::error "Go compiler not found. Please install Go to run the API server."
    exit 1
fi

# Wait for API to be ready
log::info "Waiting for API server to be ready..."
for i in {1..30}; do
    if curl -sf "http://localhost:8085/health" >/dev/null 2>&1; then
        log::success "API server is responding"
        break
    fi
    sleep 1
    if [[ $i -eq 30 ]]; then
        log::error "API server failed to start within 30 seconds"
        exit 1
    fi
done

# Start React UI
log::info "Starting React UI..."
cd "${SCENARIO_DIR}/ui"

# Install dependencies if needed
if [[ ! -d "node_modules" ]]; then
    log::info "Installing React dependencies..."
    if command -v npm >/dev/null 2>&1; then
        if npm install; then
            log::success "Dependencies installed"
        else
            log::error "Failed to install dependencies"
            exit 1
        fi
    else
        log::error "npm not found. Please install Node.js and npm."
        exit 1
    fi
fi

# Start React dev server in background
log::info "Starting React development server..."
export PORT=3005
export REACT_APP_API_URL="http://localhost:8085"

nohup npm start > ui.log 2>&1 &
UI_PID=$!
echo $UI_PID > ui.pid
log::success "React UI server started (PID: $UI_PID)"

# Wait for UI to be ready
log::info "Waiting for UI server to be ready..."
for i in {1..60}; do
    if curl -sf "http://localhost:3005" >/dev/null 2>&1; then
        log::success "UI server is responding"
        break
    fi
    sleep 2
    if [[ $i -eq 60 ]]; then
        log::warning "UI server took longer than expected to start"
        break
    fi
done

# Install CLI tool
log::info "Installing CLI tool..."
if [[ -f "${SCENARIO_DIR}/cli/install.sh" ]]; then
    bash "${SCENARIO_DIR}/cli/install.sh" || log::warning "CLI installation failed"
fi

# Final status check
log::info "Performing final health checks..."

# Check database
if psql -U postgres -d prompt_manager -c "SELECT COUNT(*) FROM campaigns;" >/dev/null 2>&1; then
    log::success "Database is working"
else
    log::error "Database health check failed"
fi

# Check API
if curl -sf "http://localhost:8085/health" >/dev/null 2>&1; then
    log::success "API server is healthy"
else
    log::error "API server health check failed"
fi

# Check UI
if curl -sf "http://localhost:3005" >/dev/null 2>&1; then
    log::success "UI server is accessible"
else
    log::warning "UI server may not be fully ready yet"
fi

echo ""
log::success "🎉 Prompt Manager startup completed!"
echo ""
echo "📍 Access Points:"
echo "   • Web UI: http://localhost:3005"
echo "   • API: http://localhost:8085"
echo "   • CLI: prompt-manager help"
echo ""
echo "💡 Quick Start:"
echo "   1. Open http://localhost:3005 in your browser"
echo "   2. Create your first campaign"
echo "   3. Add prompts to organize your AI interactions"
echo ""
echo "🔧 Process Management:"
echo "   • API Server PID: $(cat ${SCENARIO_DIR}/api/api.pid 2>/dev/null || echo 'Not found')"
echo "   • UI Server PID: $(cat ${SCENARIO_DIR}/ui/ui.pid 2>/dev/null || echo 'Not found')"
echo "   • Logs: ${SCENARIO_DIR}/api/api.log, ${SCENARIO_DIR}/ui/ui.log"
echo ""