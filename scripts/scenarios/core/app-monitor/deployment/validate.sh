#!/usr/bin/env bash
# App Monitor Validation Script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

log::info "Validating App Monitor deployment..."

# Check PostgreSQL
if PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h localhost -p "$(resources::get_default_port "postgres")" -U "${POSTGRES_USER:-postgres}" -d app_monitor -c "SELECT 1" > /dev/null 2>&1; then
    log::success "✓ PostgreSQL database connected"
else
    log::error "✗ PostgreSQL database not accessible"
    exit 1
fi

# Check Redis
if redis-cli -h localhost -p "$(resources::get_default_port "redis")" ping > /dev/null 2>&1; then
    log::success "✓ Redis connected"
else
    log::error "✗ Redis not accessible"
    exit 1
fi

# Check Docker socket
if docker ps > /dev/null 2>&1; then
    log::success "✓ Docker API accessible"
else
    log::error "✗ Docker API not accessible"
    exit 1
fi

# Check UI
if curl -s -o /dev/null -w "%{http_code}" "http://localhost:$(resources::get_default_port "ui")" | grep -q "200\|304"; then
    log::success "✓ UI dashboard accessible"
else
    log::error "✗ UI dashboard not accessible"
    exit 1
fi

# Check API
if curl -s "http://localhost:$(resources::get_default_port "api")/health" | grep -q "healthy"; then
    log::success "✓ API healthy"
else
    log::error "✗ API not healthy"
    exit 1
fi

log::success ""
log::success "App Monitor validation successful!"
log::info "All components are operational."