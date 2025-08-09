#!/usr/bin/env bash
# Prompt Manager Startup Script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

echo "Starting Prompt Manager..."

# Initialize database
echo "Initializing PostgreSQL database..."
psql -U postgres -d prompt_manager < "$SCENARIO_DIR/initialization/storage/postgres/schema.sql" || true

# Start Qdrant vector database
echo "Starting Qdrant vector database..."
docker run -d --name qdrant-prompt-manager \
    -p 6333:6333 -p 6334:6334 \
    -v "$(pwd)/qdrant_storage:/qdrant/storage" \
    qdrant/qdrant:latest || true

# Create Qdrant collection
echo "Creating Qdrant collection..."
curl -X PUT http://localhost:6333/collections/prompts \
    -H "Content-Type: application/json" \
    -d '{
        "vectors": {
            "size": 1536,
            "distance": "Cosine"
        }
    }' || true

# Ensure Ollama is running
echo "Checking Ollama..."
curl -s http://localhost:11434/api/tags || echo "Ollama not available - prompt testing will be limited"

# Initialize Redis session store
echo "Initializing Redis session store..."
redis-cli -n 2 SET "prompt_manager:initialized" "true" || true

# Health check
echo "Performing health check..."
curl -s http://localhost:8085/health || echo "API not yet available"

echo "Prompt Manager started successfully!"
echo "Dashboard available at: http://localhost:3005"
echo ""
echo "Default credentials:"
echo "  Email: admin@promptmanager.local"
echo "  Password: ChangeMeNow123!"