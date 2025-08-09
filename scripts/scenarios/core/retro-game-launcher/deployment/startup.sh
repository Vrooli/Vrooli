#!/usr/bin/env bash
# Retro Game Launcher Startup Script
# Initializes and starts the AI-powered retro game platform

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

echo "Starting Retro Game Launcher..."

# Initialize database
echo "Initializing game database..."
psql -h localhost -U postgres -d retro_games < "$SCRIPT_DIR/../initialization/storage/postgres/schema.sql"

# Setup MinIO buckets for assets
echo "Creating asset storage buckets..."
mc mb minio/game-assets || true
mc mb minio/user-uploads || true
mc policy set public minio/game-assets

# Deploy n8n game generation workflow
echo "Deploying game generation workflow..."
curl -X POST http://localhost:5678/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @"$SCRIPT_DIR/../initialization/automation/n8n/game-generation-workflow.json"

# Load game templates
echo "Loading game templates..."
psql -h localhost -U postgres -d retro_games \
  -c "INSERT INTO game_templates (data) VALUES ('$(cat "$SCRIPT_DIR/../initialization/configuration/game-templates.json")')"

# Start Judge0 for code execution
echo "Verifying Judge0 code execution service..."
curl -X GET http://localhost:2358/system_info || echo "Warning: Judge0 not responding"

# Preload Ollama models
echo "Loading AI models..."
ollama pull codellama || echo "Warning: Could not pull codellama model"
ollama pull llama3.2 || echo "Warning: Could not pull llama3.2 model"

# Start UI with retro theme
echo "Starting Retro UI..."
cd "$PROJECT_ROOT/packages/ui" && THEME=retro npm run dev &

# Start game API service
echo "Starting Game API..."
cd "$PROJECT_ROOT/packages/server" && npm run dev &

echo "Retro Game Launcher started successfully!"
echo "Access the launcher at: http://localhost:3000"
echo "Start creating games with AI prompts!"