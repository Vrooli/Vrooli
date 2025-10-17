#!/usr/bin/env bash
################################################################################
# n8n Smoke Test - Quick Health Validation
# 
# Validates n8n service health and basic connectivity (30s max)
################################################################################

set -euo pipefail

# Get directory paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
N8N_DIR="${APP_ROOT}/resources/n8n"

# Source utilities and configuration
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/resources/port_registry.sh"
source "${N8N_DIR}/config/defaults.sh"

# Get n8n port from registry
N8N_PORT=$(get_port "n8n" 2>/dev/null || echo "5678")

log::info "Starting n8n smoke tests..."

# Test 1: Check if n8n container is running
log::info "Test 1: Checking n8n container status..."
if docker ps --format "table {{.Names}}" | grep -q "n8n"; then
    log::success "✓ n8n container is running"
else
    log::error "✗ n8n container is not running"
    exit 1
fi

# Test 2: Health endpoint check
log::info "Test 2: Checking n8n health endpoint..."
if timeout 5 curl -sf "http://localhost:${N8N_PORT}/health" > /dev/null 2>&1; then
    log::success "✓ n8n health endpoint is responding"
else
    log::error "✗ n8n health endpoint is not responding"
    exit 1
fi

# Test 3: API endpoint check
log::info "Test 3: Checking n8n API availability..."
response_code=$(timeout 5 curl -s -o /dev/null -w "%{http_code}" "http://localhost:${N8N_PORT}/api/workflows" 2>/dev/null || echo "000")
if [[ "$response_code" == "200" ]] || [[ "$response_code" == "401" ]] || [[ "$response_code" == "403" ]]; then
    if [[ "$response_code" == "200" ]]; then
        log::success "✓ n8n API is accessible"
    else
        log::success "✓ n8n API is accessible (authentication required - HTTP $response_code)"
    fi
else
    log::error "✗ n8n API is not accessible (HTTP $response_code)"
    exit 1
fi

# Test 4: Database connectivity check
log::info "Test 4: Checking database connectivity..."
if docker exec n8n sh -c 'echo "SELECT 1;" | psql "$DB_POSTGRESDB_DATABASE" -h "$DB_POSTGRESDB_HOST" -U "$DB_POSTGRESDB_USER" -t' > /dev/null 2>&1; then
    log::success "✓ n8n can connect to PostgreSQL"
else
    log::warn "⚠ n8n database connectivity could not be verified (may be using SQLite)"
fi

# Test 5: Redis connectivity check
log::info "Test 5: Checking Redis connectivity..."
if docker exec n8n sh -c 'redis-cli -h "$QUEUE_BULL_REDIS_HOST" -p "$QUEUE_BULL_REDIS_PORT" ping' 2>/dev/null | grep -q "PONG"; then
    log::success "✓ n8n can connect to Redis"
else
    log::warn "⚠ n8n Redis connectivity could not be verified (may not be configured)"
fi

log::success "All smoke tests passed"
exit 0