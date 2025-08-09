#!/bin/bash
# Issue Tracker Startup Script
# Initializes all services for the centralized issue tracking system

set -e

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS_DIR="${SCENARIO_DIR}/../../.."

# Source helper functions
source "${SCRIPTS_DIR}/lib/utils/var.sh"
source "${SCRIPTS_DIR}/resources/lib/common.sh"

echo "🚀 Starting Issue Tracker Services..."

# Get resource ports from registry
POSTGRES_PORT=$(resources::get_default_port "postgres")
QDRANT_PORT=$(resources::get_default_port "qdrant")
REDIS_PORT=$(resources::get_default_port "redis")
MINIO_PORT=$(resources::get_default_port "minio")
WINDMILL_PORT=$(resources::get_default_port "windmill")
N8N_PORT=$(resources::get_default_port "n8n")
OLLAMA_PORT=$(resources::get_default_port "ollama")

# Check required services are running
echo "✓ Checking service availability..."

check_service() {
    local service=$1
    local port=$2
    if ! nc -z localhost "$port" 2>/dev/null; then
        echo "❌ $service is not running on port $port"
        echo "   Please ensure all required resources are started"
        exit 1
    fi
    echo "  ✓ $service is available on port $port"
}

check_service "PostgreSQL" "$POSTGRES_PORT"
check_service "Qdrant" "$QDRANT_PORT"
check_service "Redis" "$REDIS_PORT"
check_service "MinIO" "$MINIO_PORT"
check_service "Windmill" "$WINDMILL_PORT"
check_service "n8n" "$N8N_PORT"
check_service "Ollama" "$OLLAMA_PORT"

# Initialize PostgreSQL database
echo "📊 Initializing PostgreSQL database..."
PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h localhost \
    -p "$POSTGRES_PORT" \
    -U "${POSTGRES_USER:-postgres}" \
    -d "${POSTGRES_DB:-postgres}" \
    -c "CREATE DATABASE issue_tracker IF NOT EXISTS;" 2>/dev/null || true

# Run schema and seed scripts
echo "  Running schema setup..."
PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h localhost \
    -p "$POSTGRES_PORT" \
    -U "${POSTGRES_USER:-postgres}" \
    -d issue_tracker \
    -f "${SCENARIO_DIR}/initialization/storage/postgres/schema.sql"

echo "  Loading seed data..."
PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h localhost \
    -p "$POSTGRES_PORT" \
    -U "${POSTGRES_USER:-postgres}" \
    -d issue_tracker \
    -f "${SCENARIO_DIR}/initialization/storage/postgres/seed.sql"

# Initialize Qdrant collection
echo "🔍 Initializing Qdrant vector database..."
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
    }' 2>/dev/null || echo "  Collection may already exist"

# Initialize MinIO bucket
echo "📦 Initializing MinIO storage..."
if command -v mc &> /dev/null; then
    mc alias set issue-tracker "http://localhost:${MINIO_PORT}" \
        "${MINIO_ACCESS_KEY:-minioadmin}" \
        "${MINIO_SECRET_KEY:-minioadmin}" 2>/dev/null || true
    
    mc mb issue-tracker/issue-artifacts 2>/dev/null || echo "  Bucket may already exist"
    mc policy set public issue-tracker/issue-artifacts 2>/dev/null || true
else
    echo "  MinIO CLI not found, skipping bucket creation"
fi

# Import Windmill apps and scripts
echo "🌬️ Importing Windmill components..."

# Create workspace if needed
curl -X POST "http://localhost:${WINDMILL_PORT}/api/workspaces" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${WINDMILL_TOKEN:-default_token}" \
    -d '{"id": "issue_tracker", "name": "Issue Tracker"}' 2>/dev/null || \
    echo "  Workspace may already exist"

# Import apps
for app_file in "${SCENARIO_DIR}"/initialization/automation/windmill/apps/*.json; do
    if [ -f "$app_file" ]; then
        app_name=$(basename "$app_file" .json)
        echo "  Importing app: $app_name"
        curl -X POST "http://localhost:${WINDMILL_PORT}/api/w/issue_tracker/apps" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer ${WINDMILL_TOKEN:-default_token}" \
            -d "@$app_file" 2>/dev/null || echo "    App may already exist"
    fi
done

# Import n8n workflows
echo "🔄 Importing n8n workflows..."
for workflow_file in "${SCENARIO_DIR}"/initialization/automation/n8n/*.json; do
    if [ -f "$workflow_file" ]; then
        workflow_name=$(basename "$workflow_file" .json)
        echo "  Importing workflow: $workflow_name"
        curl -X POST "http://localhost:${N8N_PORT}/api/v1/workflows" \
            -H "Content-Type: application/json" \
            -H "X-N8N-API-KEY: ${N8N_API_KEY:-default_key}" \
            -d "@$workflow_file" 2>/dev/null || echo "    Workflow may already exist"
    fi
done

# Check Ollama models
echo "🤖 Checking Ollama models..."
if ! curl -s "http://localhost:${OLLAMA_PORT}/api/tags" | grep -q "nomic-embed-text"; then
    echo "  Pulling nomic-embed-text model..."
    curl -X POST "http://localhost:${OLLAMA_PORT}/api/pull" \
        -d '{"name": "nomic-embed-text"}' 2>/dev/null
fi

# Generate API tokens for sample apps
echo "🔑 Generating API tokens..."
PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h localhost \
    -p "$POSTGRES_PORT" \
    -U "${POSTGRES_USER:-postgres}" \
    -d issue_tracker \
    -c "UPDATE apps SET api_token = generate_api_token(name) WHERE api_token LIKE 'token_%';" 2>/dev/null

# Display access information
echo ""
echo "✅ Issue Tracker is ready!"
echo ""
echo "📌 Access Points:"
echo "  • Windmill Dashboard: http://localhost:${WINDMILL_PORT}/apps/issue_tracker_dashboard"
echo "  • Metrics Dashboard: http://localhost:${WINDMILL_PORT}/apps/issue_tracker_metrics"
echo "  • Agent Management: http://localhost:${WINDMILL_PORT}/apps/issue_tracker_agents"
echo "  • n8n Workflows: http://localhost:${N8N_PORT}"
echo ""
echo "🔧 API Endpoints:"
echo "  • Create Issue: POST http://localhost:8091/api/issues"
echo "  • Get Issues: GET http://localhost:8091/api/issues"
echo "  • Trigger Investigation: POST http://localhost:8091/api/investigate"
echo ""
echo "📝 Sample API Token (Vrooli Core):"
PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h localhost \
    -p "$POSTGRES_PORT" \
    -U "${POSTGRES_USER:-postgres}" \
    -d issue_tracker \
    -t -c "SELECT api_token FROM apps WHERE name = 'vrooli-core' LIMIT 1;" 2>/dev/null

echo ""
echo "💡 To report an issue from an app:"
echo "  vrooli-tracker create --title 'Issue title' --description 'Details' --priority high"
echo ""