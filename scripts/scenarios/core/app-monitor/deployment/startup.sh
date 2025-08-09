#!/usr/bin/env bash
# App Monitor Startup Script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

echo "Starting App Monitor..."

# Initialize database
echo "Initializing PostgreSQL database..."
psql -U postgres -d app_monitor < "$SCENARIO_DIR/initialization/storage/postgres/schema.sql" || true

# Deploy Node-RED flows
echo "Deploying Node-RED flows..."
curl -X POST http://localhost:1880/flows \
    -H "Content-Type: application/json" \
    -d @"$SCENARIO_DIR/initialization/automation/node-red/docker-monitor.json" || true

# Start monitoring services
echo "Starting monitoring services..."
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep "generated-app" || echo "No generated apps running yet"

# Health check
echo "Performing health check..."
curl -s http://localhost:8081/health || echo "API not yet available"

echo "App Monitor started successfully!"
echo "Dashboard available at: http://localhost:3001"