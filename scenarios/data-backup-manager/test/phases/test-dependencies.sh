#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"


# Source scenario-specific testing helpers
SCENARIO_ROOT="$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)"
source "${SCENARIO_ROOT}/test/utils/testing-helpers.sh"
testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Testing resource dependencies..."

# Test PostgreSQL availability
testing::phase::log "Checking PostgreSQL dependency..."

if command -v psql &>/dev/null; then
    if PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h "${POSTGRES_HOST:-localhost}" -p "${POSTGRES_PORT:-5432}" -U "${POSTGRES_USER:-postgres}" -c "SELECT 1" &>/dev/null 2>&1; then
        testing::phase::success "PostgreSQL is available and accessible"

        # Check for required database
        if PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h "${POSTGRES_HOST:-localhost}" -p "${POSTGRES_PORT:-5432}" -U "${POSTGRES_USER:-postgres}" -lqt | grep -qw "${POSTGRES_DB:-vrooli}"; then
            testing::phase::success "Required database exists"
        else
            testing::phase::warn "Required database may not exist"
        fi

        # Check for schema tables
        table_check=$(PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h "${POSTGRES_HOST:-localhost}" -p "${POSTGRES_PORT:-5432}" -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-vrooli}" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('backup_jobs', 'backup_schedules', 'restore_points');" 2>/dev/null | tr -d ' ')

        if [ "$table_check" = "3" ]; then
            testing::phase::success "All backup schema tables exist"
        else
            testing::phase::warn "Some backup schema tables missing (found: $table_check/3)"
        fi
    else
        testing::phase::error "PostgreSQL not accessible"
    fi
else
    testing::phase::warn "psql command not available, skipping PostgreSQL checks"
fi

# Test MinIO availability
testing::phase::log "Checking MinIO dependency..."

if command -v mc &>/dev/null; then
    testing::phase::success "MinIO client (mc) is installed"
else
    testing::phase::warn "MinIO client (mc) not installed - MinIO backups will fail"
    testing::phase::log "Install with: wget https://dl.min.io/client/mc/release/linux-amd64/mc && chmod +x mc && sudo mv mc /usr/local/bin/"
fi

# Check MinIO service availability via HTTP
if command -v curl &>/dev/null; then
    minio_host="${MINIO_HOST:-localhost}"
    minio_port="${MINIO_PORT:-9000}"

    if curl -sf "http://$minio_host:$minio_port/minio/health/live" >/dev/null 2>&1; then
        testing::phase::success "MinIO service is accessible"
    else
        testing::phase::warn "MinIO service not accessible at $minio_host:$minio_port"
    fi
fi

# Test N8n availability
testing::phase::log "Checking N8n dependency..."

n8n_host="${N8N_HOST:-localhost}"
n8n_port="${N8N_PORT:-5678}"

if command -v curl &>/dev/null; then
    if curl -sf "http://$n8n_host:$n8n_port/" >/dev/null 2>&1; then
        testing::phase::success "N8n service is accessible"
    else
        testing::phase::warn "N8n service not accessible at $n8n_host:$n8n_port"
    fi
else
    testing::phase::warn "curl not available for N8n check"
fi

# Test backup tool dependencies
testing::phase::log "Checking backup tool dependencies..."

# Check for pg_dump (PostgreSQL backups)
if command -v pg_dump &>/dev/null; then
    testing::phase::success "pg_dump available for PostgreSQL backups"
else
    testing::phase::warn "pg_dump not available - PostgreSQL backups may use docker exec fallback"
fi

# Check for tar (file backups)
if command -v tar &>/dev/null; then
    testing::phase::success "tar available for file backups"
else
    testing::phase::error "tar not available - file backups will fail"
fi

# Check for gzip (compression)
if command -v gzip &>/dev/null; then
    testing::phase::success "gzip available for compression"
else
    testing::phase::warn "gzip not available - compression may be limited"
fi

# Check Go dependencies
testing::phase::log "Checking Go dependencies..."

if [ -f "api/go.mod" ]; then
    cd api

    # Check if dependencies are downloaded
    if go list -m all >/dev/null 2>&1; then
        testing::phase::success "Go dependencies are available"

        # Check for specific required dependencies
        required_deps=("github.com/gorilla/mux" "github.com/lib/pq" "github.com/rs/cors")
        for dep in "${required_deps[@]}"; do
            if go list -m "$dep" >/dev/null 2>&1; then
                testing::phase::success "Dependency present: $dep"
            else
                testing::phase::error "Missing dependency: $dep"
            fi
        done
    else
        testing::phase::error "Go dependencies not properly downloaded"
    fi

    cd ..
else
    testing::phase::error "go.mod not found"
fi

# Check directory permissions
testing::phase::log "Checking directory permissions..."

required_dirs=("data/backups" "data/backups/postgres" "data/backups/files" "data/backups/scenarios" "data/backups/metadata")

for dir in "${required_dirs[@]}"; do
    if [ -d "$dir" ]; then
        if [ -w "$dir" ]; then
            testing::phase::success "Directory writable: $dir"
        else
            testing::phase::error "Directory not writable: $dir"
        fi
    else
        testing::phase::warn "Directory missing: $dir"
    fi
done

# Check environment variables
testing::phase::log "Checking environment configuration..."

env_vars=("POSTGRES_HOST" "POSTGRES_PORT" "POSTGRES_USER" "MINIO_HOST" "MINIO_PORT" "N8N_HOST" "N8N_PORT")

for var in "${env_vars[@]}"; do
    if [ -n "${!var:-}" ]; then
        testing::phase::log "$var is set"
    else
        testing::phase::log "$var not set (using defaults)"
    fi
done

testing::phase::end_with_summary "Dependency tests completed"
