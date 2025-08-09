#!/usr/bin/env bash
# Web Scraper Manager Startup Script
# Initializes unified dashboard for Huginn, Browserless, and Agent-S2

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

echo "Starting Web Scraper Manager..."

# Initialize database
echo "Initializing scraper database..."
psql -h localhost -p 5433 -U postgres -d scraper_manager < "$SCRIPT_DIR/../initialization/storage/postgres/schema.sql"

# Verify scraping platforms are running
echo "Checking scraping platforms..."
curl -s http://localhost:4111/agents || echo "Warning: Huginn not responding"
curl -s http://localhost:4110/stats || echo "Warning: Browserless not responding"  
curl -s http://localhost:4113/health || echo "Warning: Agent-S2 not responding"

# Deploy orchestration workflow
echo "Deploying scraper orchestrator..."
curl -X POST http://localhost:5678/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @"$SCRIPT_DIR/../initialization/automation/n8n/scraper-orchestrator.json"

# Load platform configurations
echo "Loading platform configurations..."
psql -h localhost -p 5433 -U postgres -d scraper_manager \
  -c "INSERT INTO platform_configs (data) VALUES ('$(cat "$SCRIPT_DIR/../initialization/configuration/platform-configs.json")')" || echo "Config already exists"

# Setup MinIO for scraped content
echo "Creating storage buckets..."
mc alias set minio http://localhost:9000 minioadmin minioadmin || true
mc mb minio/scraper-assets || true
mc mb minio/screenshots || true
mc mb minio/exports || true

# Initialize Qdrant for similarity detection
echo "Setting up vector search..."
curl -X PUT http://localhost:6333/collections/scraped_content \
  -H "Content-Type: application/json" \
  -d '{"vectors": {"size": 768, "distance": "Cosine"}}'

# Setup Redis queues
echo "Initializing job queues..."
redis-cli -p 6380 LPUSH scraper:queue:init "ready" > /dev/null

# Create default proxies
echo "Setting up proxy pool..."
psql -h localhost -p 5433 -U postgres -d scraper_manager \
  -c "INSERT INTO proxy_pool (proxy_url, proxy_type, location) VALUES 
      ('http://proxy1.example.com:8080', 'http', 'US'),
      ('http://proxy2.example.com:8080', 'http', 'EU')" || echo "Proxy data already exists"

# Verify Windmill is running for the UI
echo "Checking Windmill UI..."
curl -s http://localhost:5681/api/version || echo "Warning: Windmill not responding"

echo "Web Scraper Manager started successfully!"
echo "Dashboard: http://localhost:5681"
echo "N8N Orchestrator: http://localhost:5678"
echo "Vrooli API: http://localhost:5329/api"
echo ""
echo "Available platforms:"
echo "  - Huginn: http://localhost:4111"
echo "  - Browserless: http://localhost:4110"
echo "  - Agent-S2: http://localhost:4113"
echo "  - Qdrant Vector DB: http://localhost:6333"
echo "  - MinIO Storage: http://localhost:9000"