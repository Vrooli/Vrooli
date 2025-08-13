#!/bin/bash

# Task Planner - PostgreSQL Database Setup Script
# Creates database, applies schema, and loads seed data

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $*${NC}"; }
success() { echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] ✅ $*${NC}"; }
warn() { echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] ⚠️  $*${NC}"; }
error() { echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ❌ $*${NC}" >&2; }

# Database configuration
DB_HOST="${POSTGRES_HOST:-localhost}"
DB_PORT="${POSTGRES_PORT:-5434}"
DB_NAME="task_planner"
DB_USER="taskplanner_user"
DB_PASSWORD="${POSTGRES_PASSWORD:-$(openssl rand -base64 32)}"
ADMIN_DB="${POSTGRES_DB:-postgres}"
ADMIN_USER="${POSTGRES_USER:-postgres}"
ADMIN_PASSWORD="${POSTGRES_PASSWORD:-postgres}"

# File paths
SCHEMA_FILE="${SCENARIO_DIR}/initialization/storage/postgres/schema.sql"
SEED_FILE="${SCENARIO_DIR}/initialization/storage/postgres/seed.sql"

# Wait for PostgreSQL to be ready
wait_for_postgres() {
    log "Waiting for PostgreSQL to be ready..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if PGPASSWORD="$ADMIN_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -d "$ADMIN_DB" -c "SELECT 1;" > /dev/null 2>&1; then
            success "PostgreSQL is ready"
            return 0
        fi
        
        warn "Waiting for PostgreSQL... (attempt $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    error "PostgreSQL did not become ready within expected time"
    return 1
}

# Create database and user
create_database() {
    log "Creating database and user..."
    
    # Create user if it doesn't exist
    PGPASSWORD="$ADMIN_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -d "$ADMIN_DB" -c "
        DO \$\$
        BEGIN
            IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '$DB_USER') THEN
                CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
                ALTER USER $DB_USER CREATEDB;
            END IF;
        END
        \$\$;
    " || {
        error "Failed to create database user"
        return 1
    }
    
    # Create database if it doesn't exist
    PGPASSWORD="$ADMIN_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -d "$ADMIN_DB" -c "
        SELECT 'CREATE DATABASE $DB_NAME OWNER $DB_USER;'
        WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$DB_NAME')\gexec
    " || {
        error "Failed to create database"
        return 1
    }
    
    # Grant permissions
    PGPASSWORD="$ADMIN_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -d "$DB_NAME" -c "
        GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;
        GRANT ALL ON SCHEMA public TO $DB_USER;
        GRANT CREATE ON SCHEMA public TO $DB_USER;
    " || {
        error "Failed to grant permissions"
        return 1
    }
    
    success "Database and user created successfully"
}

# Apply database schema
apply_schema() {
    log "Applying database schema..."
    
    if [[ ! -f "$SCHEMA_FILE" ]]; then
        error "Schema file not found: $SCHEMA_FILE"
        return 1
    fi
    
    # Check if schema is already applied
    local table_count
    table_count=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT count(*) 
        FROM information_schema.tables 
        WHERE table_schema = 'public' 
        AND table_type = 'BASE TABLE'
        AND table_name IN ('apps', 'tasks', 'unstructured_sessions', 'agent_runs');
    " | xargs) || {
        error "Failed to check existing schema"
        return 1
    }
    
    if [[ "$table_count" -eq "4" ]]; then
        warn "Schema already exists, skipping schema application"
    else
        log "Applying schema from $SCHEMA_FILE"
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$SCHEMA_FILE" || {
            error "Failed to apply schema"
            return 1
        }
        success "Schema applied successfully"
    fi
}

# Load seed data
load_seed_data() {
    log "Loading seed data..."
    
    if [[ ! -f "$SEED_FILE" ]]; then
        warn "Seed file not found: $SEED_FILE, skipping seed data"
        return 0
    fi
    
    # Check if seed data already loaded
    local app_count
    app_count=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT count(*) FROM apps;
    " | xargs) || {
        error "Failed to check seed data"
        return 1
    }
    
    if [[ "$app_count" -gt "0" ]]; then
        warn "Seed data already exists ($app_count apps found), skipping seed load"
    else
        log "Loading seed data from $SEED_FILE"
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$SEED_FILE" || {
            error "Failed to load seed data"
            return 1
        }
        success "Seed data loaded successfully"
    fi
}

# Create database indexes for performance
optimize_database() {
    log "Optimizing database performance..."
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        -- Additional performance optimizations
        
        -- Update table statistics
        ANALYZE;
        
        -- Create additional indexes if they don't exist
        DO \$\$
        BEGIN
            -- Index for task search by app and status
            IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_tasks_app_status') THEN
                CREATE INDEX CONCURRENTLY idx_tasks_app_status ON tasks(app_id, status);
            END IF;
            
            -- Index for recent activity queries
            IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_transitions_created_app') THEN
                CREATE INDEX CONCURRENTLY idx_transitions_created_app ON task_transitions(created_at DESC, task_id);
            END IF;
            
            -- Index for agent performance queries  
            IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_agent_runs_perf') THEN
                CREATE INDEX CONCURRENTLY idx_agent_runs_perf ON agent_runs(action, successful, completed_at DESC);
            END IF;
        END
        \$\$;
    " || {
        warn "Some optimizations failed - database will still work correctly"
    }
    
    success "Database optimization completed"
}

# Verify database setup
verify_setup() {
    log "Verifying database setup..."
    
    # Test basic connectivity and permissions
    local result
    result=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT 
            (SELECT count(*) FROM apps) as app_count,
            (SELECT count(*) FROM tasks) as task_count,
            (SELECT count(*) FROM unstructured_sessions) as session_count,
            (SELECT count(*) FROM task_transitions) as transition_count;
    " | xargs) || {
        error "Failed to verify database setup"
        return 1
    }
    
    read -r app_count task_count session_count transition_count <<< "$result"
    
    success "Database verification completed:"
    log "  📊 Apps: $app_count"
    log "  📋 Tasks: $task_count" 
    log "  📝 Sessions: $session_count"
    log "  🔄 Transitions: $transition_count"
    
    # Test vector extension
    local vector_test
    vector_test=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT count(*) FROM pg_extension WHERE extname = 'vector';
    " | xargs) || {
        warn "Could not verify vector extension"
        return 0
    }
    
    if [[ "$vector_test" -eq "1" ]]; then
        success "Vector extension is available for embeddings"
    else
        warn "Vector extension not found - semantic search may be limited"
    fi
}

# Display connection information
show_connection_info() {
    log "PostgreSQL Connection Information:"
    echo ""
    echo "📍 Host: $DB_HOST:$DB_PORT"
    echo "🗄️  Database: $DB_NAME"
    echo "👤 User: $DB_USER"
    echo "🔐 Password: [stored in environment variable POSTGRES_PASSWORD]"
    echo ""
    echo "📞 Connection String: postgresql://$DB_USER:[password]@$DB_HOST:$DB_PORT/$DB_NAME"
    echo ""
    echo "🔧 Management Commands:"
    echo "  psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME"
    echo "  pg_dump -h $DB_HOST -p $DB_PORT -U $DB_USER $DB_NAME > backup.sql"
    echo ""
}

# Main execution
main() {
    log "Setting up PostgreSQL for Task Planner..."
    
    # Ensure we have required variables
    if [[ -z "${POSTGRES_PASSWORD:-}" ]]; then
        POSTGRES_PASSWORD="$(openssl rand -base64 32)"
        export POSTGRES_PASSWORD
        warn "Generated random PostgreSQL password"
    fi
    
    # Setup steps
    wait_for_postgres
    create_database
    apply_schema
    load_seed_data
    optimize_database
    verify_setup
    show_connection_info
    
    success "PostgreSQL setup completed successfully!"
    
    # Export connection details for other scripts
    cat > "${SCRIPT_DIR}/.postgres_connection" << EOF
export POSTGRES_HOST="$DB_HOST"
export POSTGRES_PORT="$DB_PORT"
export POSTGRES_DB="$DB_NAME"
export POSTGRES_USER="$DB_USER"
export POSTGRES_PASSWORD="$DB_PASSWORD"
export DATABASE_URL="postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME"
EOF
    
    success "Connection details saved to ${SCRIPT_DIR}/.postgres_connection"
}

# Execute main function
main "$@"