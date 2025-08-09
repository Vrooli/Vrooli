#!/bin/bash
# PostgreSQL setup for Smart File Photo Manager
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VROOLI_ROOT="$(cd "$SCENARIO_ROOT/../../.." && pwd)"

# Load environment variables
source "$VROOLI_ROOT/scripts/resources/lib/resource-helper.sh"

# Configuration
DB_NAME="file_manager"
DB_USER="filemanager_user"
DB_PASSWORD="${POSTGRES_PASSWORD:-filemanager_pass123}"
POSTGRES_HOST="localhost"
POSTGRES_PORT="5435"

# Get default postgres port if available
if command -v "resources::get_default_port" &> /dev/null; then
    POSTGRES_PORT=$(resources::get_default_port "postgres" || echo "5435")
fi

log_info() {
    echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

log_success() {
    echo "[SUCCESS] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Wait for PostgreSQL to be ready
wait_for_postgres() {
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for PostgreSQL on port $POSTGRES_PORT..."
    
    while [ $attempt -lt $max_attempts ]; do
        if pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U postgres >/dev/null 2>&1; then
            log_success "PostgreSQL is ready"
            return 0
        fi
        
        sleep 2
        ((attempt++))
    done
    
    log_error "PostgreSQL failed to become ready"
    return 1
}

# Create database and user
create_database() {
    log_info "Creating database and user..."
    
    # Create database if it doesn't exist
    PGPASSWORD="postgres" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1 || \
    PGPASSWORD="postgres" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U postgres -c "CREATE DATABASE $DB_NAME;"
    
    # Create user if it doesn't exist
    PGPASSWORD="postgres" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U postgres -tc "SELECT 1 FROM pg_roles WHERE rolname = '$DB_USER'" | grep -q 1 || \
    PGPASSWORD="postgres" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U postgres -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';"
    
    # Grant privileges
    PGPASSWORD="postgres" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
    PGPASSWORD="postgres" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U postgres -c "ALTER USER $DB_USER CREATEDB;"
    
    log_success "Database and user created successfully"
}

# Apply schema
apply_schema() {
    log_info "Applying database schema..."
    
    local schema_file="$SCENARIO_ROOT/initialization/storage/postgres/schema.sql"
    
    if [ ! -f "$schema_file" ]; then
        log_error "Schema file not found: $schema_file"
        return 1
    fi
    
    # Apply schema
    PGPASSWORD="$DB_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$schema_file"
    
    log_success "Database schema applied successfully"
}

# Apply seed data
apply_seed_data() {
    local seed_file="$SCENARIO_ROOT/initialization/storage/postgres/seed.sql"
    
    if [ -f "$seed_file" ]; then
        log_info "Applying seed data..."
        PGPASSWORD="$DB_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$seed_file"
        log_success "Seed data applied successfully"
    else
        log_info "No seed file found, skipping seed data"
    fi
}

# Verify database setup
verify_setup() {
    log_info "Verifying database setup..."
    
    local table_count
    table_count=$(PGPASSWORD="$DB_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")
    
    if [ "$table_count" -gt 0 ]; then
        log_success "Database verification passed ($table_count tables created)"
    else
        log_error "Database verification failed (no tables found)"
        return 1
    fi
}

# Create database connection test
test_connection() {
    log_info "Testing database connection..."
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null
    
    log_success "Database connection test passed"
}

# Main execution
main() {
    log_info "Setting up PostgreSQL database for File Manager..."
    
    # Wait for PostgreSQL to be ready
    wait_for_postgres
    
    # Create database and user
    create_database
    
    # Apply schema
    apply_schema
    
    # Apply seed data if available
    apply_seed_data
    
    # Verify setup
    verify_setup
    
    # Test connection
    test_connection
    
    log_success "PostgreSQL setup completed successfully"
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi