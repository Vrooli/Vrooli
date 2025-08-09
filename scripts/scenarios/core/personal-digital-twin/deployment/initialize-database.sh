#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Personal Digital Twin - Database Initialization
# Initializes PostgreSQL database schema and seed data
################################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

SCENARIO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5433}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-digital_twin}"

# Wait for PostgreSQL to be ready
log::info "Waiting for PostgreSQL to be ready..."
until pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" >/dev/null 2>&1; do
    echo "Waiting for PostgreSQL..."
    sleep 2
done

log::success "PostgreSQL is ready!"

# Execute schema creation
log::info "Creating database schema..."
PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    -f "${SCENARIO_DIR}/initialization/storage/postgres/schema.sql"

if [[ $? -eq 0 ]]; then
    log::success "Database schema initialized successfully"
else
    log::error "Failed to initialize database schema"
    exit 1
fi

# Create seed data if it exists
SEED_FILE="${SCENARIO_DIR}/initialization/storage/postgres/seed.sql"
if [[ -f "$SEED_FILE" ]]; then
    log::info "Loading seed data..."
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
        -f "$SEED_FILE"
    
    if [[ $? -eq 0 ]]; then
        log::success "Seed data loaded successfully"
    else
        log::error "Failed to load seed data"
        exit 1
    fi
fi

log::success "Database initialization complete!"