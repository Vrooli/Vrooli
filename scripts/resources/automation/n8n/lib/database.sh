#!/usr/bin/env bash
# n8n Database Management Functions
# PostgreSQL support for n8n

# Source required utilities
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../lib/wait-utils.sh" 2>/dev/null || true

#######################################
# Start PostgreSQL container for n8n
#######################################
n8n::start_postgres() {
    if [[ "$DATABASE_TYPE" != "postgres" ]]; then
        return 0
    fi
    log::info "Starting PostgreSQL container for n8n..."
    # Check if postgres container already exists
    if n8n::postgres_exists; then
        if n8n::postgres_running; then
            log::info "PostgreSQL container is already running"
            return 0
        else
            log::info "Starting existing PostgreSQL container..."
            docker start "$N8N_DB_CONTAINER_NAME" >/dev/null 2>&1
            return 0
        fi
    fi
    # Create postgres data directory
    local pg_data_dir="${N8N_DATA_DIR}/postgres"
    mkdir -p "$pg_data_dir"
    # Run PostgreSQL container
    if docker run -d \
        --name "$N8N_DB_CONTAINER_NAME" \
        --network "$N8N_NETWORK_NAME" \
        -e POSTGRES_USER=n8n \
        -e POSTGRES_PASSWORD="$N8N_DB_PASSWORD" \
        -e POSTGRES_DB=n8n \
        -v "$pg_data_dir:/var/lib/postgresql/data" \
        --restart unless-stopped \
        postgres:14-alpine >/dev/null 2>&1; then
        log::success "PostgreSQL container started"
        # Add rollback action
        resources::add_rollback_action \
            "Stop and remove PostgreSQL container" \
            "docker stop $N8N_DB_CONTAINER_NAME 2>/dev/null; docker rm $N8N_DB_CONTAINER_NAME 2>/dev/null || true" \
            20
        # Wait for PostgreSQL to be ready
        log::info "Waiting for PostgreSQL to be ready..."
        sleep 5
        return 0
    else
        log::error "Failed to start PostgreSQL container"
        return 1
    fi
}

#######################################
# Check if PostgreSQL is healthy
# Returns: 0 if healthy, 1 otherwise
#######################################
n8n::postgres_is_healthy() {
    if [[ "$DATABASE_TYPE" != "postgres" ]]; then
        return 0  # Not using postgres, so "healthy"
    fi
    # Check if container is running
    if ! n8n::postgres_running; then
        return 1
    fi
    # Check if postgres is accepting connections
    if docker exec "$N8N_DB_CONTAINER_NAME" pg_isready -U n8n >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

#######################################
# Wait for PostgreSQL to be ready
# Args: max_attempts (optional, default 30)
# Returns: 0 if ready, 1 if timeout
#######################################
n8n::wait_for_postgres() {
    local max_attempts=${1:-30}
    local attempt=0
    if [[ "$DATABASE_TYPE" != "postgres" ]]; then
        return 0  # Not using postgres
    fi
    # Use standardized wait utility
    wait::for_condition \
        "n8n::postgres_is_healthy" \
        "$max_attempts" \
        "PostgreSQL to be ready"
    return $?
}

#######################################
# Get PostgreSQL connection info
# Returns: Connection string for display
#######################################
n8n::get_postgres_info() {
    if [[ "$DATABASE_TYPE" != "postgres" ]]; then
        echo "Using SQLite database"
        return
    fi
    cat << EOF
PostgreSQL Connection Info:
  Host: $N8N_DB_CONTAINER_NAME
  Port: 5432
  Database: n8n
  User: n8n
  Password: [hidden]
EOF
}

#######################################
# Backup PostgreSQL database
# Args: backup_dir (optional)
# Returns: 0 on success, 1 on failure
#######################################
n8n::backup_postgres() {
    local backup_dir="${1:-$N8N_BACKUP_DIR}"
    if [[ "$DATABASE_TYPE" != "postgres" ]]; then
        return 0  # Nothing to backup for SQLite
    fi
    if ! n8n::postgres_running; then
        log::warn "PostgreSQL container is not running, skipping backup"
        return 0
    fi
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${backup_dir}/n8n_postgres_backup_${timestamp}.sql"
    log::info "Backing up PostgreSQL database..."
    mkdir -p "$backup_dir"
    if docker exec "$N8N_DB_CONTAINER_NAME" pg_dump -U n8n n8n > "$backup_file" 2>/dev/null; then
        log::success "PostgreSQL backup saved to: $backup_file"
        return 0
    else
        log::error "Failed to backup PostgreSQL database"
        return 1
    fi
}

#######################################
# Restore PostgreSQL database
# Args: backup_file
# Returns: 0 on success, 1 on failure
#######################################
n8n::restore_postgres() {
    local backup_file="$1"
    if [[ "$DATABASE_TYPE" != "postgres" ]]; then
        log::error "Cannot restore PostgreSQL backup when using SQLite"
        return 1
    fi
    if [[ ! -f "$backup_file" ]]; then
        log::error "Backup file not found: $backup_file"
        return 1
    fi
    if ! n8n::postgres_running; then
        log::error "PostgreSQL container is not running"
        return 1
    fi
    log::info "Restoring PostgreSQL database from: $backup_file"
    if docker exec -i "$N8N_DB_CONTAINER_NAME" psql -U n8n n8n < "$backup_file" 2>/dev/null; then
        log::success "PostgreSQL database restored successfully"
        return 0
    else
        log::error "Failed to restore PostgreSQL database"
        return 1
    fi
}

#######################################
# Get database type and status
# Returns: Database info string
#######################################
n8n::get_database_status() {
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        if n8n::postgres_running; then
            if n8n::postgres_is_healthy; then
                echo "PostgreSQL (healthy)"
            else
                echo "PostgreSQL (unhealthy)"
            fi
        else
            echo "PostgreSQL (not running)"
        fi
    else
        echo "SQLite (embedded)"
    fi
}

#######################################
# Clean up database resources
#######################################
n8n::cleanup_database() {
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        # Backup before cleanup
        n8n::backup_postgres
        # Remove PostgreSQL container
        n8n::remove_postgres_container
    fi
}
