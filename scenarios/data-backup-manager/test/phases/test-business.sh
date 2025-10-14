#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"


# Source scenario-specific testing helpers
SCENARIO_ROOT="$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)"
source "${SCENARIO_ROOT}/test/utils/testing-helpers.sh"
testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Running business logic tests..."

# Get API port
API_PORT=$(vrooli scenario status data-backup-manager --json 2>/dev/null | jq -r '.scenario_data.allocated_ports.API_PORT // .raw_response.data.allocated_ports.API_PORT // "20010"')
testing::phase::log "Using API port: $API_PORT"

# Test: Backup Job Creation Workflow
testing::phase::log "Testing backup job creation workflow..."

backup_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/backup/create" \
    -H "Content-Type: application/json" \
    -d '{"type":"full","targets":["postgres"],"description":"Business test backup"}')

if echo "$backup_response" | jq -e '.job_id' >/dev/null 2>&1; then
    job_id=$(echo "$backup_response" | jq -r '.job_id')
    testing::phase::success "Backup job created successfully: $job_id"

    # Verify job has expected status
    if echo "$backup_response" | jq -e '.status == "pending"' >/dev/null 2>&1; then
        testing::phase::success "New backup job has correct initial status"
    fi
else
    testing::phase::error "Failed to create backup job"
fi

# Test: Invalid Backup Type Rejection
testing::phase::log "Testing invalid backup type rejection..."

invalid_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/backup/create" \
    -H "Content-Type: application/json" \
    -d '{"type":"invalid_type","targets":["postgres"]}' \
    -w "\n%{http_code}")

http_code=$(echo "$invalid_response" | tail -1)
if [ "$http_code" = "400" ]; then
    testing::phase::success "Invalid backup type correctly rejected"
else
    testing::phase::warn "Invalid backup type not properly validated (got HTTP $http_code)"
fi

# Test: Empty Targets Rejection
testing::phase::log "Testing empty targets rejection..."

empty_targets_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/backup/create" \
    -H "Content-Type: application/json" \
    -d '{"type":"full","targets":[]}' \
    -w "\n%{http_code}")

http_code=$(echo "$empty_targets_response" | tail -1)
if [ "$http_code" = "400" ]; then
    testing::phase::success "Empty targets correctly rejected"
else
    testing::phase::warn "Empty targets not properly validated (got HTTP $http_code)"
fi

# Test: Backup List Retrieval
testing::phase::log "Testing backup list retrieval..."

list_response=$(curl -s "http://localhost:$API_PORT/api/v1/backup/list")

if echo "$list_response" | jq -e '.backups' >/dev/null 2>&1; then
    backup_count=$(echo "$list_response" | jq '.backups | length')
    testing::phase::success "Backup list retrieved successfully ($backup_count backups)"
else
    testing::phase::warn "Backup list format unexpected"
fi

# Test: Schedule Management
testing::phase::log "Testing schedule management..."

schedule_list=$(curl -s "http://localhost:$API_PORT/api/v1/schedules")

if echo "$schedule_list" | jq -e '.schedules' >/dev/null 2>&1; then
    schedule_count=$(echo "$schedule_list" | jq '.schedules | length')
    testing::phase::success "Schedule list retrieved successfully ($schedule_count schedules)"

    # Verify schedules have required fields
    if echo "$schedule_list" | jq -e '.schedules[0].cron_expression' >/dev/null 2>&1; then
        testing::phase::success "Schedules contain cron expressions"
    fi
else
    testing::phase::warn "Schedule list format unexpected"
fi

# Test: Restore Workflow Validation
testing::phase::log "Testing restore workflow validation..."

# Try restore without job ID or restore point (should fail)
restore_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/restore/create" \
    -H "Content-Type: application/json" \
    -d '{"targets":["postgres"],"verify_before_restore":true}' \
    -w "\n%{http_code}")

http_code=$(echo "$restore_response" | tail -1)
if [ "$http_code" = "400" ]; then
    testing::phase::success "Restore validation requires backup_job_id or restore_point_id"
else
    testing::phase::warn "Restore validation not working as expected (got HTTP $http_code)"
fi

# Test: Backup Verification
if [ -n "$job_id" ]; then
    testing::phase::log "Testing backup verification..."

    verify_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/backup/verify/$job_id")

    if echo "$verify_response" | jq -e '.backup_id' >/dev/null 2>&1; then
        testing::phase::success "Backup verification endpoint functional"
    else
        testing::phase::warn "Backup verification response unexpected"
    fi
fi

# Test: Storage Status Reporting
testing::phase::log "Testing storage status reporting..."

status_response=$(curl -s "http://localhost:$API_PORT/api/v1/backup/status")

if echo "$status_response" | jq -e '.storage_usage.used_gb' >/dev/null 2>&1; then
    used_gb=$(echo "$status_response" | jq -r '.storage_usage.used_gb')
    available_gb=$(echo "$status_response" | jq -r '.storage_usage.available_gb')
    testing::phase::success "Storage usage reported: ${used_gb}GB used, ${available_gb}GB available"
fi

if echo "$status_response" | jq -e '.storage_usage.compression_ratio' >/dev/null 2>&1; then
    ratio=$(echo "$status_response" | jq -r '.storage_usage.compression_ratio')
    testing::phase::success "Compression ratio reported: $ratio"
fi

# Test: Resource Health Monitoring
testing::phase::log "Testing resource health monitoring..."

if echo "$status_response" | jq -e '.resource_health.postgres.status' >/dev/null 2>&1; then
    postgres_status=$(echo "$status_response" | jq -r '.resource_health.postgres.status')
    testing::phase::log "PostgreSQL health: $postgres_status"
fi

if echo "$status_response" | jq -e '.resource_health.minio.status' >/dev/null 2>&1; then
    minio_status=$(echo "$status_response" | jq -r '.resource_health.minio.status')
    testing::phase::log "MinIO health: $minio_status"
fi

if echo "$status_response" | jq -e '.resource_health.n8n.status' >/dev/null 2>&1; then
    n8n_status=$(echo "$status_response" | jq -r '.resource_health.n8n.status')
    testing::phase::log "N8n health: $n8n_status"
fi

# Test: System Status Determination
if echo "$status_response" | jq -e '.system_status == "healthy"' >/dev/null 2>&1; then
    testing::phase::success "System status correctly reported as healthy"
else
    system_status=$(echo "$status_response" | jq -r '.system_status')
    testing::phase::warn "System status: $system_status"
fi

testing::phase::end_with_summary "Business logic tests completed"
