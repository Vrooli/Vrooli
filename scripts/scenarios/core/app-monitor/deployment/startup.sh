#!/usr/bin/env bash
# App Monitor Startup Script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# shellcheck disable=SC1091
source "${SCENARIO_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

log::info "Starting App Monitor..."

# Initialize database
log::info "Initializing PostgreSQL database..."
PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h localhost -p "$(resources::get_default_port "postgres")" -U "${POSTGRES_USER:-postgres}" -d app_monitor < "$SCENARIO_DIR/initialization/storage/postgres/schema.sql" || true

# Deploy Node-RED flows
log::info "Deploying Node-RED flows..."
curl -X POST "http://localhost:$(resources::get_default_port "node-red")/flows" \
    -H "Content-Type: application/json" \
    -d @"$SCENARIO_DIR/initialization/automation/node-red/docker-monitor.json" || true

# Start monitoring services
log::info "Starting monitoring services..."
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep "generated-app" || log::info "No generated apps running yet"

# Health check
log::info "Performing health check..."
curl -s "http://localhost:$(resources::get_default_port "api")/health" || log::info "API not yet available"

log::success "App Monitor started successfully!"
log::info "Dashboard available at: http://localhost:$(resources::get_default_port "ui")"