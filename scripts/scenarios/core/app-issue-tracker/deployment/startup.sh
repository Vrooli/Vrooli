#!/usr/bin/env bash
# Issue Tracker Startup Script
# Initializes all services for the centralized issue tracking system

set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# shellcheck disable=SC1091
source "${SCENARIO_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

log::info "🚀 Starting Issue Tracker Services..."

# Get resource ports from registry
POSTGRES_PORT=$(resources::get_default_port "postgres")
QDRANT_PORT=$(resources::get_default_port "qdrant")
REDIS_PORT=$(resources::get_default_port "redis")
MINIO_PORT=$(resources::get_default_port "minio")
WINDMILL_PORT=$(resources::get_default_port "windmill")
N8N_PORT=$(resources::get_default_port "n8n")
OLLAMA_PORT=$(resources::get_default_port "ollama")

# Check required services are running
log::info "✓ Checking service availability..."

startup::check_service() {
    local service=$1
    local port=$2
    if ! nc -z localhost "$port" 2>/dev/null; then
        log::error "❌ $service is not running on port $port"
        log::error "   Please ensure all required resources are started"
        exit 1
    fi
    log::success "  ✓ $service is available on port $port"
}

startup::check_service "PostgreSQL" "$POSTGRES_PORT"
startup::check_service "Qdrant" "$QDRANT_PORT"
startup::check_service "Redis" "$REDIS_PORT"
startup::check_service "MinIO" "$MINIO_PORT"
startup::check_service "Windmill" "$WINDMILL_PORT"
startup::check_service "n8n" "$N8N_PORT"
startup::check_service "Ollama" "$OLLAMA_PORT"

# Initialize PostgreSQL database
log::info "📊 Initializing PostgreSQL database..."
# Create database if it doesn't exist (PostgreSQL doesn't support CREATE DATABASE IF NOT EXISTS)
if ! PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h localhost \
    -p "$POSTGRES_PORT" \
    -U "${POSTGRES_USER:-postgres}" \
    -lqt | cut -d \| -f 1 | grep -qw "issue_tracker"; then
    PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" createdb \
        -h localhost \
        -p "$POSTGRES_PORT" \
        -U "${POSTGRES_USER:-postgres}" \
        issue_tracker
else
    log::info "  Database issue_tracker already exists"
fi

# Run schema and seed scripts
log::info "  Running schema setup..."
PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h localhost \
    -p "$POSTGRES_PORT" \
    -U "${POSTGRES_USER:-postgres}" \
    -d issue_tracker \
    -f "${SCENARIO_DIR}/initialization/storage/postgres/schema.sql"

log::info "  Loading seed data..."
PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h localhost \
    -p "$POSTGRES_PORT" \
    -U "${POSTGRES_USER:-postgres}" \
    -d issue_tracker \
    -f "${SCENARIO_DIR}/initialization/storage/postgres/seed.sql"

# Initialize Qdrant collection
log::info "🔍 Initializing Qdrant vector database..."
curl -X PUT "http://localhost:${QDRANT_PORT}/collections/issues" \
    -H "Content-Type: application/json" \
    -d '{
        "vectors": {
            "size": 1536,
            "distance": "Cosine"
        },
        "optimizers_config": {
            "default_segment_number": 2
        },
        "replication_factor": 1
    }' 2>/dev/null || log::info "  Collection may already exist"

# Initialize MinIO bucket
log::info "📦 Initializing MinIO storage..."
if command -v mc &> /dev/null; then
    mc alias set issue-tracker "http://localhost:${MINIO_PORT}" \
        "${MINIO_ACCESS_KEY:-minioadmin}" \
        "${MINIO_SECRET_KEY:-minioadmin}" 2>/dev/null || true
    
    mc mb issue-tracker/issue-artifacts 2>/dev/null || echo "  Bucket may already exist"
    mc policy set public issue-tracker/issue-artifacts 2>/dev/null || true
else
    log::warning "  MinIO CLI not found, skipping bucket creation"
fi

# Import Windmill apps and scripts
log::info "🌬️ Importing Windmill components..."

# Create workspace if needed
curl -X POST "http://localhost:${WINDMILL_PORT}/api/workspaces" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${WINDMILL_TOKEN:-default_token}" \
    -d '{"id": "issue_tracker", "name": "Issue Tracker"}' 2>/dev/null || \
    log::info "  Workspace may already exist"

# Import apps
for app_file in "${SCENARIO_DIR}"/initialization/automation/windmill/apps/*.json; do
    if [ -f "$app_file" ]; then
        app_name=$(basename "$app_file" .json)
        log::info "  Importing app: $app_name"
        curl -X POST "http://localhost:${WINDMILL_PORT}/api/w/issue_tracker/apps" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer ${WINDMILL_TOKEN:-default_token}" \
            -d "@$app_file" 2>/dev/null || log::info "    App may already exist"
    fi
done

# Import n8n workflows
log::info "🔄 Importing n8n workflows..."
for workflow_file in "${SCENARIO_DIR}"/initialization/automation/n8n/*.json; do
    if [ -f "$workflow_file" ]; then
        workflow_name=$(basename "$workflow_file" .json)
        log::info "  Importing workflow: $workflow_name"
        curl -X POST "http://localhost:${N8N_PORT}/api/v1/workflows" \
            -H "Content-Type: application/json" \
            -H "X-N8N-API-KEY: ${N8N_API_KEY:-default_key}" \
            -d "@$workflow_file" 2>/dev/null || log::info "    Workflow may already exist"
    fi
done

# Check Ollama models
log::info "🤖 Checking Ollama models..."
if ! curl -s "http://localhost:${OLLAMA_PORT}/api/tags" | grep -q "nomic-embed-text"; then
    log::info "  Pulling nomic-embed-text model..."
    curl -X POST "http://localhost:${OLLAMA_PORT}/api/pull" \
        -d '{"name": "nomic-embed-text"}' 2>/dev/null
fi

# Generate API tokens for sample apps
log::info "🔑 Generating API tokens..."
PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h localhost \
    -p "$POSTGRES_PORT" \
    -U "${POSTGRES_USER:-postgres}" \
    -d issue_tracker \
    -c "UPDATE apps SET api_token = generate_api_token(name) WHERE api_token LIKE 'token_%';" 2>/dev/null

# Display access information
log::success ""
log::success "✅ Issue Tracker is ready!"
log::success ""
log::info "📌 Access Points:"
log::info "  • Windmill Dashboard: http://localhost:${WINDMILL_PORT}/apps/issue_tracker_dashboard"
log::info "  • Metrics Dashboard: http://localhost:${WINDMILL_PORT}/apps/issue_tracker_metrics"
log::info "  • Agent Management: http://localhost:${WINDMILL_PORT}/apps/issue_tracker_agents"
log::info "  • n8n Workflows: http://localhost:${N8N_PORT}"
log::success ""
log::info "🔧 API Endpoints:"
log::info "  • Create Issue: POST http://localhost:8091/api/issues"
log::info "  • Get Issues: GET http://localhost:8091/api/issues"
log::info "  • Trigger Investigation: POST http://localhost:8091/api/investigate"
log::success ""
log::info "📝 Sample API Token (Vrooli Core):"
PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h localhost \
    -p "$POSTGRES_PORT" \
    -U "${POSTGRES_USER:-postgres}" \
    -d issue_tracker \
    -t -c "SELECT api_token FROM apps WHERE name = 'vrooli-core' LIMIT 1;" 2>/dev/null

log::success ""
log::info "💡 To report an issue from an app:"
log::info "  vrooli-tracker create --title 'Issue title' --description 'Details' --priority high"
log::success ""