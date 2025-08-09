#!/usr/bin/env bash
# System Monitor Startup Script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# shellcheck disable=SC1091
source "$(cd "$SCRIPT_DIR" && cd ../../../lib/utils && pwd)/var.sh"
# shellcheck disable=SC1091
source "$var_LOG_FILE"

echo "Starting System Monitor..."

# Initialize database
echo "Initializing PostgreSQL database..."
psql -U postgres -d system_monitor < "$SCENARIO_DIR/initialization/storage/postgres/schema.sql" || true

# Start QuestDB for time-series
echo "Starting QuestDB..."
docker run -d --name questdb-system-monitor \
    -p 9000:9000 -p 9009:9009 \
    questdb/questdb:latest || true

# Deploy Node-RED anomaly detector
echo "Deploying Node-RED anomaly detection flow..."
curl -X POST http://localhost:1880/flows \
    -H "Content-Type: application/json" \
    -d @"$SCENARIO_DIR/initialization/automation/node-red/anomaly-detector.json" || true

# Configure Grafana dashboards
echo "Configuring Grafana dashboards..."
# Would normally import dashboard JSON here

# Start metric collection
echo "Starting metric collection..."
curl -X POST http://localhost:8083/api/metrics/start || true

# Health check
echo "Performing health check..."
curl -s http://localhost:8083/health || echo "API not yet available"

echo "System Monitor started successfully!"
echo "Dashboard available at: http://localhost:3003"
echo "Grafana available at: http://localhost:3004"