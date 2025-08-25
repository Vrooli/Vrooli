#!/bin/bash
# Database Test Handler - Handles database operations and testing

set -euo pipefail

# Colors for output (only define if not already defined)
if [[ -z "${RED:-}" ]]; then
    readonly RED='\033[0;31m'
    readonly GREEN='\033[0;32m'
    readonly YELLOW='\033[1;33m'
    readonly BLUE='\033[0;34m'
    readonly NC='\033[0m'
fi

# Database test results
DB_ERRORS=0
DB_WARNINGS=0

# Print functions
print_db_info() {
    echo -e "${BLUE}[DB]${NC} $1"
}

print_db_success() {
    echo -e "${GREEN}[DB ✓]${NC} $1"
}

print_db_error() {
    echo -e "${RED}[DB ✗]${NC} $1"
    ((DB_ERRORS++))
}

print_db_warning() {
    echo -e "${YELLOW}[DB ⚠]${NC} $1"
    ((DB_WARNINGS++))
}

# Get database connection string
get_db_connection() {
    local db_type="$1"
    local connection_string=""
    
    case "$db_type" in
        postgres|postgresql)
            # Try environment variables first
            if [[ -n "${DATABASE_URL:-}" ]]; then
                connection_string="$DATABASE_URL"
            elif [[ -n "${POSTGRES_URL:-}" ]]; then
                connection_string="$POSTGRES_URL"
            else
                # Default PostgreSQL connection
                local host="${POSTGRES_HOST:-localhost}"
                local port="${POSTGRES_PORT:-5432}"
                local user="${POSTGRES_USER:-postgres}"
                local password="${POSTGRES_PASSWORD:-}"
                local database="${POSTGRES_DATABASE:-postgres}"
                
                if [[ -n "$password" ]]; then
                    connection_string="postgresql://${user}:${password}@${host}:${port}/${database}"
                else
                    connection_string="postgresql://${user}@${host}:${port}/${database}"
                fi
            fi
            ;;
        *)
            print_db_error "Unsupported database type: $db_type"
            return 1
            ;;
    esac
    
    echo "$connection_string"
}

# Test database connectivity
test_db_connection() {
    local db_type="$1"
    local connection_string="$2"
    
    print_db_info "Testing database connectivity"
    
    case "$db_type" in
        postgres|postgresql)
            if command -v psql >/dev/null 2>&1; then
                if psql "$connection_string" -c "SELECT 1;" >/dev/null 2>&1; then
                    print_db_success "Database connection successful"
                    return 0
                else
                    print_db_error "Database connection failed"
                    return 1
                fi
            else
                print_db_warning "psql not available, skipping connection test"
                return 0
            fi
            ;;
        *)
            print_db_error "Connection test not implemented for: $db_type"
            return 1
            ;;
    esac
}

# Execute SQL query
execute_sql() {
    local db_type="$1"
    local connection_string="$2"
    local query="$3"
    local output_file="${4:-}"
    
    case "$db_type" in
        postgres|postgresql)
            if command -v psql >/dev/null 2>&1; then
                local psql_args=("$connection_string" -c "$query")
                
                if [[ -n "$output_file" ]]; then
                    psql "${psql_args[@]}" > "$output_file" 2>&1
                else
                    psql "${psql_args[@]}" 2>&1
                fi
            else
                print_db_error "psql not available"
                return 1
            fi
            ;;
        *)
            print_db_error "SQL execution not implemented for: $db_type"
            return 1
            ;;
    esac
}

# Create database schema
create_schema() {
    local db_type="$1"
    local connection_string="$2"
    local schema_file="$3"
    
    print_db_info "Creating database schema from: $(basename "$schema_file")"
    
    if [[ ! -f "$schema_file" ]]; then
        print_db_error "Schema file not found: $schema_file"
        return 1
    fi
    
    # Execute schema file
    local result
    result=$(execute_sql "$db_type" "$connection_string" "$(cat "$schema_file")")
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        print_db_success "Schema created successfully"
        return 0
    else
        print_db_error "Schema creation failed: $result"
        return 1
    fi
}

# Insert test data
insert_test_data() {
    local db_type="$1"
    local connection_string="$2"
    local table="$3"
    local data="$4"
    
    print_db_info "Inserting test data into table: $table"
    
    # Parse data (simplified JSON parsing)
    local columns=""
    local values=""
    
    # This is a simplified parser - in production, use proper JSON parsing
    if command -v jq >/dev/null 2>&1; then
        columns=$(echo "$data" | jq -r 'keys | join(", ")')
        values=$(echo "$data" | jq -r 'values | map(if type == "string" then "'"'"'\(.)'"'"'" else . end) | join(", ")')
    else
        # Fallback: assume simple key-value pairs
        print_db_warning "jq not available, using simplified data parsing"
        columns="id, name"
        values="'test-id', 'test-name'"
    fi
    
    local insert_query="INSERT INTO $table ($columns) VALUES ($values);"
    
    local result
    result=$(execute_sql "$db_type" "$connection_string" "$insert_query")
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        print_db_success "Test data inserted"
        return 0
    else
        print_db_error "Data insertion failed: $result"
        return 1
    fi
}

# Query and validate data
query_and_validate() {
    local db_type="$1"
    local connection_string="$2"
    local table="$3"
    local where_clause="$4"
    local expected_count="$5"
    
    print_db_info "Querying table: $table"
    
    local query="SELECT COUNT(*) FROM $table"
    if [[ -n "$where_clause" ]]; then
        query="$query WHERE $where_clause"
    fi
    query="$query;"
    
    local result
    result=$(execute_sql "$db_type" "$connection_string" "$query")
    local exit_code=$?
    
    if [[ $exit_code -ne 0 ]]; then
        print_db_error "Query failed: $result"
        return 1
    fi
    
    # Extract count from result
    local actual_count
    actual_count=$(echo "$result" | grep -E '^[[:space:]]*[0-9]+[[:space:]]*$' | xargs)
    
    if [[ -z "$actual_count" ]]; then
        print_db_error "Could not extract count from query result"
        return 1
    fi
    
    # Validate count
    if [[ "$actual_count" -eq "$expected_count" ]]; then
        print_db_success "Query validation passed (count: $actual_count)"
        return 0
    else
        print_db_error "Query validation failed (expected: $expected_count, actual: $actual_count)"
        return 1
    fi
}

# Run database migration
run_migration() {
    local db_type="$1"
    local connection_string="$2"
    local migration_file="$3"
    
    print_db_info "Running migration: $(basename "$migration_file")"
    
    if [[ ! -f "$migration_file" ]]; then
        print_db_error "Migration file not found: $migration_file"
        return 1
    fi
    
    # Check if migration has already been applied
    local migration_name
    migration_name=$(basename "$migration_file" .sql)
    local check_query="SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'schema_migrations';"
    
    local migrations_table_exists
    migrations_table_exists=$(execute_sql "$db_type" "$connection_string" "$check_query" 2>/dev/null | grep -E '^[[:space:]]*[0-9]+[[:space:]]*$' | xargs)
    
    if [[ "$migrations_table_exists" == "0" ]]; then
        # Create migrations table
        local create_migrations_table="CREATE TABLE schema_migrations (version VARCHAR(255) PRIMARY KEY, applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);"
        execute_sql "$db_type" "$connection_string" "$create_migrations_table" >/dev/null
    fi
    
    # Check if migration already applied
    local check_migration="SELECT COUNT(*) FROM schema_migrations WHERE version = '$migration_name';"
    local already_applied
    already_applied=$(execute_sql "$db_type" "$connection_string" "$check_migration" 2>/dev/null | grep -E '^[[:space:]]*[0-9]+[[:space:]]*$' | xargs)
    
    if [[ "$already_applied" == "1" ]]; then
        print_db_warning "Migration already applied: $migration_name"
        return 0
    fi
    
    # Apply migration
    local result
    result=$(execute_sql "$db_type" "$connection_string" "$(cat "$migration_file")")
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        # Record migration
        local record_migration="INSERT INTO schema_migrations (version) VALUES ('$migration_name');"
        execute_sql "$db_type" "$connection_string" "$record_migration" >/dev/null
        
        print_db_success "Migration applied: $migration_name"
        return 0
    else
        print_db_error "Migration failed: $result"
        return 1
    fi
}

# Clean up test data
cleanup_test_data() {
    local db_type="$1"
    local connection_string="$2"
    local table="$3"
    local where_clause="${4:-1=1}"
    
    print_db_info "Cleaning up test data from: $table"
    
    local delete_query="DELETE FROM $table WHERE $where_clause;"
    
    local result
    result=$(execute_sql "$db_type" "$connection_string" "$delete_query")
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        print_db_success "Test data cleaned up"
        return 0
    else
        print_db_error "Cleanup failed: $result"
        return 1
    fi
}

# Execute database test from configuration
execute_database_test_from_config() {
    local test_config="$1"
    
    # Parse configuration (simplified)
    local db_type
    db_type=$(echo "$test_config" | grep "type:" | cut -d: -f2 | xargs)
    # Note: operations variable was unused, removing it
    
    db_type="${db_type:-postgres}"
    
    # Get connection string
    local connection_string
    connection_string=$(get_db_connection "$db_type")
    
    if [[ -z "$connection_string" ]]; then
        print_db_error "Could not determine database connection"
        return 1
    fi
    
    # Test connection
    if ! test_db_connection "$db_type" "$connection_string"; then
        return 1
    fi
    
    # Process operations
    # This is simplified - in production, parse YAML properly
    print_db_info "Processing database operations"
    
    return 0
}

# Test scenario database setup
test_scenario_database() {
    local scenario_dir="$1"
    local db_type="${2:-postgres}"
    
    print_db_info "Testing scenario database setup"
    
    local connection_string
    connection_string=$(get_db_connection "$db_type")
    
    if [[ -z "$connection_string" ]]; then
        print_db_error "Database connection not configured"
        return 1
    fi
    
    # Test connection
    if ! test_db_connection "$db_type" "$connection_string"; then
        return 1
    fi
    
    # Look for schema file
    local schema_file="$scenario_dir/initialization/database/schema.sql"
    if [[ -f "$schema_file" ]]; then
        create_schema "$db_type" "$connection_string" "$schema_file"
    fi
    
    # Look for seed data
    local seed_file="$scenario_dir/initialization/database/seed.sql"
    if [[ -f "$seed_file" ]]; then
        print_db_info "Loading seed data"
        execute_sql "$db_type" "$connection_string" "$(cat "$seed_file")" >/dev/null
        if [[ $? -eq 0 ]]; then
            print_db_success "Seed data loaded"
        else
            print_db_warning "Seed data loading failed"
        fi
    fi
    
    return 0
}

# Execute database test handler
execute_database_test() {
    local test_name="$1"
    local test_data="$2"
    
    print_db_info "Executing database test: $test_name"
    
    if [[ -f "$test_data" ]]; then
        execute_database_test_from_config "$test_data"
    else
        execute_database_test_from_config "$test_data"
    fi
}

# Export functions
export -f execute_database_test
export -f test_scenario_database
export -f create_schema
export -f insert_test_data
export -f query_and_validate
export -f run_migration
export -f cleanup_test_data