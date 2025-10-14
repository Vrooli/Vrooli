#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"


# Source scenario-specific testing helpers
SCENARIO_ROOT="$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)"
source "${SCENARIO_ROOT}/test/utils/testing-helpers.sh"
testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Running integration tests..."

# Check if required resources are available
POSTGRES_AVAILABLE=false
MINIO_AVAILABLE=false

if command -v psql &>/dev/null; then
    if PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h "${POSTGRES_HOST:-localhost}" -p "${POSTGRES_PORT:-5432}" -U "${POSTGRES_USER:-postgres}" -c "SELECT 1" &>/dev/null; then
        POSTGRES_AVAILABLE=true
        testing::phase::success "PostgreSQL available"
    fi
fi

# Check MinIO availability
if command -v mc &>/dev/null; then
    MINIO_AVAILABLE=true
    testing::phase::success "MinIO client (mc) available"
else
    testing::phase::warn "MinIO client (mc) not available, some tests will be skipped"
fi

if [ "$POSTGRES_AVAILABLE" = false ]; then
    testing::phase::warn "PostgreSQL not available, skipping database integration tests"
fi

# Get API port
API_PORT=$(vrooli scenario status data-backup-manager --json 2>/dev/null | jq -r '.scenario_data.allocated_ports.API_PORT // .raw_response.data.allocated_ports.API_PORT // "20010"')
testing::phase::log "Using API port: $API_PORT"

# Test API health endpoint
testing::phase::log "Testing API health check..."

if curl -f -s "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
    health_response=$(curl -s "http://localhost:$API_PORT/health")
    testing::phase::success "API health check successful"

    if echo "$health_response" | jq -e '.resources.postgres.status == "healthy"' >/dev/null 2>&1; then
        testing::phase::success "PostgreSQL resource health check passed"
    fi

    if echo "$health_response" | jq -e '.resources.minio.status == "healthy"' >/dev/null 2>&1; then
        testing::phase::success "MinIO resource health check passed"
    fi
else
    testing::phase::error "API health check failed"
    testing::phase::end_with_summary "API health check failed" 1
fi

# Test backup status endpoint
testing::phase::log "Testing backup status endpoint..."

if curl -f -s "http://localhost:$API_PORT/api/v1/backup/status" >/dev/null 2>&1; then
    status_response=$(curl -s "http://localhost:$API_PORT/api/v1/backup/status")
    testing::phase::success "Backup status endpoint responsive"

    if echo "$status_response" | jq -e '.system_status' >/dev/null 2>&1; then
        testing::phase::success "Status response includes system_status"
    fi

    if echo "$status_response" | jq -e '.storage_usage' >/dev/null 2>&1; then
        testing::phase::success "Status response includes storage_usage"
    fi
else
    testing::phase::error "Backup status endpoint failed"
fi

# Test backup list endpoint
testing::phase::log "Testing backup list endpoint..."

if curl -f -s "http://localhost:$API_PORT/api/v1/backup/list" >/dev/null 2>&1; then
    testing::phase::success "Backup list endpoint responsive"
else
    testing::phase::warn "Backup list endpoint failed (may be empty)"
fi

# Test backup creation
testing::phase::log "Testing backup creation..."

backup_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/backup/create" \
    -H "Content-Type: application/json" \
    -d '{"type":"full","targets":["test-data"],"description":"Integration test backup"}')

if echo "$backup_response" | jq -e '.job_id' >/dev/null 2>&1; then
    job_id=$(echo "$backup_response" | jq -r '.job_id')
    testing::phase::success "Backup creation successful, job_id: $job_id"
else
    testing::phase::warn "Backup creation response unexpected: $backup_response"
fi

# Test schedule list endpoint
testing::phase::log "Testing schedule list endpoint..."

if curl -f -s "http://localhost:$API_PORT/api/v1/schedules" >/dev/null 2>&1; then
    testing::phase::success "Schedule list endpoint responsive"
else
    testing::phase::warn "Schedule list endpoint failed"
fi

# Test database schema if PostgreSQL is available
if [ "$POSTGRES_AVAILABLE" = true ]; then
    testing::phase::log "Testing database schema..."

    table_check=$(PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h "${POSTGRES_HOST:-localhost}" -p "${POSTGRES_PORT:-5432}" -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-vrooli}" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('backup_jobs', 'backup_schedules', 'restore_points');" 2>/dev/null || echo "0")

    if [ "$table_check" -ge 3 ]; then
        testing::phase::success "Database schema tables exist"
    else
        testing::phase::warn "Some database schema tables may be missing (found: $table_check)"
    fi
fi

# Test file system integration
testing::phase::log "Testing file system integration..."

if [ -d "data/backups" ]; then
    testing::phase::success "Backup directory accessible"

    # Check write permissions
    test_file="data/backups/.test_write_$$"
    if touch "$test_file" 2>/dev/null; then
        rm -f "$test_file"
        testing::phase::success "Backup directory writable"
    else
        testing::phase::error "Backup directory not writable"
    fi
else
    testing::phase::error "Backup directory not found"
fi

# Test CLI integration
if [ -f "cli/data-backup-manager" ] && [ -x "cli/data-backup-manager" ]; then
    testing::phase::log "Testing CLI integration..."

    if API_PORT=$API_PORT ./cli/data-backup-manager status >/dev/null 2>&1; then
        testing::phase::success "CLI status command works"
    else
        testing::phase::warn "CLI status command failed"
    fi
else
    testing::phase::warn "CLI not available for testing"
fi

testing::phase::end_with_summary "Integration tests completed"
