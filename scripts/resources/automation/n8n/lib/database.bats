#!/usr/bin/env bats
# Tests for n8n database.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export N8N_CUSTOM_PORT="5678"
    export N8N_CONTAINER_NAME="n8n-test"
    export N8N_DB_CONTAINER_NAME="n8n-postgres-test"
    export N8N_DATA_DIR="/tmp/n8n-test"
    export DATABASE_TYPE="postgres"
    export N8N_DB_POSTGRESDB_HOST="localhost"
    export N8N_DB_POSTGRESDB_PORT="5432"
    export N8N_DB_POSTGRESDB_DATABASE="n8n"
    export N8N_DB_POSTGRESDB_USER="n8n"
    export N8N_DB_POSTGRESDB_PASSWORD="n8n"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    N8N_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directory
    mkdir -p "$N8N_DATA_DIR"
    
    # Mock system functions
    
    # Mock Docker functions
    
    # Mock psql command
    psql() {
        case "$*" in
            *"-c"*)
                echo "PSQL_COMMAND: $*"
                return 0
                ;;
            *)
                echo "PSQL_CONNECT: $*"
                return 0
                ;;
        esac
    }
    
    # Mock log functions
    
    
    
    
    # Load configuration and messages
    source "${N8N_DIR}/config/defaults.sh"
    source "${N8N_DIR}/config/messages.sh"
    n8n::export_config
    n8n::export_messages
    
    # Load the functions to test
    source "${N8N_DIR}/lib/database.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$N8N_DATA_DIR"
}

# Test PostgreSQL container start
@test "n8n::start_postgres starts PostgreSQL container successfully" {
    export DATABASE_TYPE="postgres"
    
    # Mock as non-existing container
    docker() {
        case "$1" in
            "ps")
                echo ""  # No existing container
                ;;
            "run")
                echo "DOCKER_RUN: $*"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    
    result=$(n8n::start_postgres)
    
    [[ "$result" =~ "INFO: Starting PostgreSQL container" ]]
    [[ "$result" =~ "DOCKER_RUN:" ]]
    [[ "$result" =~ "postgres" ]]
}

# Test PostgreSQL container start with existing running container
@test "n8n::start_postgres handles existing running container" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::start_postgres)
    
    [[ "$result" =~ "INFO: PostgreSQL container is already running" ]]
    [[ ! "$result" =~ "DOCKER_RUN:" ]]
}

# Test PostgreSQL container start with existing stopped container
@test "n8n::start_postgres starts existing stopped container" {
    export DATABASE_TYPE="postgres"
    
    # Mock existing but stopped container
    docker() {
        case "$1" in
            "ps")
                if [[ "$*" =~ "-a" ]]; then
                    echo "n8n-postgres-test"  # Exists but not running
                else
                    echo ""  # Not running
                fi
                ;;
            "start")
                echo "DOCKER_START: $*"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    
    result=$(n8n::start_postgres)
    
    [[ "$result" =~ "INFO: Starting existing PostgreSQL container" ]]
    [[ "$result" =~ "DOCKER_START:" ]]
}

# Test PostgreSQL start with non-postgres database type
@test "n8n::start_postgres skips for non-postgres database" {
    export DATABASE_TYPE="sqlite"
    
    result=$(n8n::start_postgres)
    
    # Should return success but do nothing
    [ "$?" -eq 0 ]
    [[ ! "$result" =~ "PostgreSQL" ]]
}

# Test PostgreSQL container stop
@test "n8n::stop_postgres stops PostgreSQL container" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::stop_postgres)
    
    [[ "$result" =~ "INFO: Stopping PostgreSQL container" ]]
    [[ "$result" =~ "DOCKER_STOP:" ]]
}

# Test PostgreSQL stop with non-postgres database type
@test "n8n::stop_postgres skips for non-postgres database" {
    export DATABASE_TYPE="sqlite"
    
    result=$(n8n::stop_postgres)
    
    # Should return success but do nothing
    [ "$?" -eq 0 ]
    [[ ! "$result" =~ "PostgreSQL" ]]
}

# Test PostgreSQL container removal
@test "n8n::remove_postgres removes PostgreSQL container" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::remove_postgres)
    
    [[ "$result" =~ "INFO: Removing PostgreSQL container" ]]
    [[ "$result" =~ "DOCKER_STOP:" ]]
    [[ "$result" =~ "DOCKER_RM:" ]]
}

# Test database initialization
@test "n8n::init_database initializes database schema" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::init_database)
    
    [[ "$result" =~ "INFO: Initializing n8n database" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
}

# Test database backup
@test "n8n::backup_database creates database backup" {
    export DATABASE_TYPE="postgres"
    local backup_file="/tmp/n8n_backup.sql"
    
    result=$(n8n::backup_database "$backup_file")
    
    [[ "$result" =~ "INFO: Creating database backup" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
    [[ "$result" =~ "pg_dump" ]]
}

# Test database restore
@test "n8n::restore_database restores database from backup" {
    export DATABASE_TYPE="postgres"
    local backup_file="/tmp/n8n_backup.sql"
    echo "-- Test backup" > "$backup_file"
    
    result=$(n8n::restore_database "$backup_file")
    
    [[ "$result" =~ "INFO: Restoring database from backup" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
    [[ "$result" =~ "psql" ]]
    
    rm -f "$backup_file"
}

# Test database restore with missing file
@test "n8n::restore_database handles missing backup file" {
    export DATABASE_TYPE="postgres"
    
    run n8n::restore_database "/nonexistent/backup.sql"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Backup file not found" ]]
}

# Test database connection test
@test "n8n::test_database_connection tests database connectivity" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::test_database_connection)
    
    [[ "$result" =~ "INFO: Testing database connection" ]]
    [[ "$result" =~ "PSQL_CONNECT:" ]]
}

# Test database connection with connection failure
@test "n8n::test_database_connection handles connection failure" {
    export DATABASE_TYPE="postgres"
    
    # Mock psql to fail
    psql() {
        echo "ERROR: connection failed"
        return 1
    }
    
    run n8n::test_database_connection
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test database migration
@test "n8n::migrate_database runs database migrations" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::migrate_database)
    
    [[ "$result" =~ "INFO: Running database migrations" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
}

# Test database status check
@test "n8n::check_database_status checks database health" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::check_database_status)
    
    [[ "$result" =~ "Database Status:" ]]
    [[ "$result" =~ "Connection:" ]]
}

# Test database cleanup
@test "n8n::cleanup_database performs database cleanup" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::cleanup_database)
    
    [[ "$result" =~ "INFO: Cleaning up database" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
}

# Test database user creation
@test "n8n::create_database_user creates database user" {
    export DATABASE_TYPE="postgres"
    local username="testuser"
    local password="testpass"
    
    result=$(n8n::create_database_user "$username" "$password")
    
    [[ "$result" =~ "INFO: Creating database user" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
    [[ "$result" =~ "CREATE USER" ]]
}

# Test database user removal
@test "n8n::remove_database_user removes database user" {
    export DATABASE_TYPE="postgres"
    local username="testuser"
    
    result=$(n8n::remove_database_user "$username")
    
    [[ "$result" =~ "INFO: Removing database user" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
    [[ "$result" =~ "DROP USER" ]]
}

# Test database configuration validation
@test "n8n::validate_database_config validates database configuration" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::validate_database_config)
    
    [[ "$result" =~ "Database configuration:" ]]
    [[ "$result" =~ "Type: postgres" ]]
    [[ "$result" =~ "Host:" ]]
    [[ "$result" =~ "Port:" ]]
    [[ "$result" =~ "Database:" ]]
}

# Test database configuration validation with invalid config
@test "n8n::validate_database_config detects invalid configuration" {
    export DATABASE_TYPE="postgres"
    export N8N_DB_POSTGRESDB_HOST=""  # Invalid empty host
    
    run n8n::validate_database_config
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test database size check
@test "n8n::get_database_size returns database size information" {
    export DATABASE_TYPE="postgres"
    
    # Mock psql to return size info
    psql() {
        echo "50 MB"
        return 0
    }
    
    result=$(n8n::get_database_size)
    
    [[ "$result" =~ "50 MB" ]]
}

# Test database performance metrics
@test "n8n::get_database_metrics collects database performance metrics" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::get_database_metrics)
    
    [[ "$result" =~ "Database metrics:" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
}

# Test database environment setup
@test "n8n::setup_database_environment configures database environment" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::setup_database_environment)
    
    [[ "$result" =~ "Setting up database environment" ]]
    [[ "$result" =~ "PGHOST" ]] || [[ "$result" =~ "postgres" ]]
}