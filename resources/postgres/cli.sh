#!/usr/bin/env bash
################################################################################
# PostgreSQL Resource CLI
# 
# Lightweight CLI interface for PostgreSQL using the CLI Command Framework
#
# Usage:
#   resource-postgres <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    POSTGRES_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    POSTGRES_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
POSTGRES_CLI_DIR="$(cd "$(dirname "$POSTGRES_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${POSTGRES_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source PostgreSQL configuration
# shellcheck disable=SC1091
source "${POSTGRES_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${POSTGRES_CLI_DIR}/config/messages.sh" 2>/dev/null || true
postgres::export_config 2>/dev/null || true
postgres::messages::init 2>/dev/null || true

# Source PostgreSQL libraries
for lib in common docker status database instance backup; do
    lib_file="${POSTGRES_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "postgres" "PostgreSQL relational database management"

# Override help to provide PostgreSQL-specific examples
cli::register_command "help" "Show this help message with PostgreSQL examples" "postgres_show_help"

# Override status to use standardized format support
cli::register_command "status" "Show PostgreSQL status" "postgres::status::show"

# Register additional PostgreSQL-specific commands
cli::register_command "inject" "Inject SQL/config into PostgreSQL" "postgres_inject" "modifies-system"
cli::register_command "list-instances" "List all PostgreSQL instances" "postgres_list_instances"
cli::register_command "create-instance" "Create new instance" "postgres_create_instance" "modifies-system"
cli::register_command "list-databases" "List databases in instance" "postgres_list_databases"
cli::register_command "create-database" "Create new database" "postgres_create_database" "modifies-system"
cli::register_command "execute-sql" "Execute SQL command" "postgres_execute_sql"
cli::register_command "create-backup" "Create database backup" "postgres_create_backup" "modifies-system"
cli::register_command "credentials" "Show n8n credentials for PostgreSQL" "postgres_credentials"
cli::register_command "uninstall" "Uninstall PostgreSQL (requires --force)" "postgres_uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject SQL or configuration into PostgreSQL
postgres_inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-postgres inject <file.sql>"
        echo ""
        echo "Examples:"
        echo "  resource-postgres inject schema.sql"
        echo "  resource-postgres inject shared:initialization/storage/postgres/init.sql"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${var_ROOT_DIR}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    postgres::inject_file "$file"
}

# Uninstall PostgreSQL
postgres_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove PostgreSQL and all its data. Use --force to confirm."
        return 1
    fi
    
    postgres::uninstall
}

# List instances
postgres_list_instances() {
    # Ensure messages are initialized
    postgres::messages::init 2>/dev/null || true
    
    postgres::instance::list
}

# Create database
postgres_create_database() {
    local database_name="${1:-}"
    local instance_name="${2:-main}"
    
    if [[ -z "$database_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-postgres create-database <name> [instance]"
        echo ""
        echo "Examples:"
        echo "  resource-postgres create-database myapp"
        echo "  resource-postgres create-database testdb main"
        return 1
    fi
    
    postgres::database::create "$instance_name" "$database_name"
}

# Execute SQL
postgres_execute_sql() {
    local sql_command="${1:-}"
    local instance_name="${2:-main}"
    local database="${3:-${POSTGRES_DEFAULT_DB:-postgres}}"
    
    # Ensure messages are initialized
    postgres::messages::init 2>/dev/null || true
    
    if [[ -z "$sql_command" ]]; then
        log::error "SQL command required"
        echo "Usage: resource-postgres execute-sql '<command>' [instance] [database]"
        echo ""
        echo "Examples:"
        echo "  resource-postgres execute-sql 'SELECT version();'"
        echo "  resource-postgres execute-sql '\\l' main postgres"
        echo "  resource-postgres execute-sql 'SELECT * FROM users LIMIT 10;' main myapp"
        return 1
    fi
    
    postgres::database::execute "$instance_name" "$sql_command" "$database"
}

# List databases
postgres_list_databases() {
    local instance_name="${1:-main}"
    
    postgres::database::list "$instance_name"
}

# Create backup
postgres_create_backup() {
    local database_name="${1:-}"
    local instance_name="${2:-main}"
    local backup_file="${3:-}"
    
    if [[ -z "$database_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-postgres create-backup <database> [instance] [backup-file]"
        echo ""
        echo "Examples:"
        echo "  resource-postgres create-backup myapp"
        echo "  resource-postgres create-backup myapp main /backups/myapp.sql"
        return 1
    fi
    
    postgres::backup::create "$instance_name" "$database_name" "$backup_file"
}

# Create instance
postgres_create_instance() {
    local instance_name="${1:-}"
    local template="${2:-development}"
    
    if [[ -z "$instance_name" ]]; then
        log::error "Instance name required"
        echo "Usage: resource-postgres create-instance <name> [template]"
        echo ""
        echo "Examples:"
        echo "  resource-postgres create-instance testing"
        echo "  resource-postgres create-instance prod production"
        echo ""
        echo "Templates: development, testing, production"
        return 1
    fi
    
    # Pass empty string for port to let it auto-assign
    postgres::instance::create "$instance_name" "" "$template"
}

# Show credentials for n8n integration
postgres_credentials() {
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "postgres"; return 0; }
        return 1
    fi
    
    # Get all PostgreSQL containers (instances)
    local connections=()
    local overall_status="stopped"
    
    # Check for PostgreSQL containers
    if command -v docker &>/dev/null; then
        local container_prefix="${POSTGRES_CONTAINER_PREFIX:-vrooli-postgres}"
        local containers
        containers=$(docker ps -a --filter "name=^/${container_prefix}-" --format "{{.Names}}:{{.Status}}")
        
        if [[ -n "$containers" ]]; then
            while IFS=: read -r container_name container_status; do
                # Skip if container is not running
                if [[ ! "$container_status" =~ ^Up ]]; then
                    continue
                fi
                
                overall_status="running"
                
                # Extract instance info from container name
                local instance_name="${container_name#$container_prefix-}"
                
                # Get container port mapping
                local port_mapping
                port_mapping=$(docker port "$container_name" 5432 2>/dev/null | head -1)
                local host_port="${port_mapping##*:}"
                host_port="${host_port:-5432}"
                
                # Get environment variables for database info
                local db_name db_user db_password
                db_name=$(docker inspect "$container_name" --format '{{range .Config.Env}}{{if (index (split . "=") 0 | eq "POSTGRES_DB")}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
                db_name="${db_name:-${POSTGRES_DEFAULT_DB:-postgres}}"
                
                db_user=$(docker inspect "$container_name" --format '{{range .Config.Env}}{{if (index (split . "=") 0 | eq "POSTGRES_USER")}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
                db_user="${db_user:-${POSTGRES_DEFAULT_USER:-postgres}}"
                
                db_password=$(docker inspect "$container_name" --format '{{range .Config.Env}}{{if (index (split . "=") 0 | eq "POSTGRES_PASSWORD")}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
                db_password="${db_password:-postgres}"
                
                # Create connection object
                local connection_obj
                connection_obj=$(jq -n -c \
                    --arg host "localhost" \
                    --arg port "$host_port" \
                    --arg database "$db_name" \
                    '{host: $host, port: ($port | tonumber), database: $database, ssl: false}')
                
                local auth_obj
                auth_obj=$(jq -n -c \
                    --arg username "$db_user" \
                    --arg password "$db_password" \
                    '{username: $username, password: $password}')
                
                local metadata_obj
                metadata_obj=$(jq -n -c \
                    --arg description "PostgreSQL instance: $instance_name" \
                    --argjson capabilities '["sql", "transactions", "acid"]' \
                    --arg version "16" \
                    '{description: $description, capabilities: $capabilities, version: $version}')
                
                local connection
                connection=$(credentials::build_connection \
                    "$instance_name" \
                    "PostgreSQL ($instance_name)" \
                    "postgres" \
                    "$connection_obj" \
                    "$auth_obj" \
                    "$metadata_obj")
                
                connections+=("$connection")
                
            done <<< "$containers"
        fi
    fi
    
    # Convert connections array to JSON
    local connections_array="[]"
    if [[ ${#connections[@]} -gt 0 ]]; then
        connections_array=$(printf '%s\n' "${connections[@]}" | jq -s '.')
    fi
    
    local response
    response=$(credentials::build_response "postgres" "$overall_status" "$connections_array")
    credentials::format_output "$response"
}

# Custom help function with PostgreSQL-specific examples
postgres_show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add PostgreSQL-specific examples
    echo ""
    echo "🐘 PostgreSQL Database Examples:"
    echo ""
    echo "Instance Management:"
    echo "  resource-postgres list-instances               # List all instances"
    echo "  resource-postgres create-instance testing     # Create test instance"
    echo "  resource-postgres create-instance prod production  # Create production instance"
    echo ""
    echo "Database Operations:"
    echo "  resource-postgres list-databases main          # List databases"
    echo "  resource-postgres create-database myapp        # Create database"
    echo "  resource-postgres execute-sql 'SELECT version();'  # Run SQL query"
    echo "  resource-postgres execute-sql '\\l' main postgres  # List databases via SQL"
    echo ""
    echo "Data Management:"
    echo "  resource-postgres inject schema.sql            # Import SQL schema"
    echo "  resource-postgres inject shared:init/postgres/data.sql  # Import shared data"
    echo "  resource-postgres create-backup myapp main /backups/myapp.sql  # Backup database"
    echo ""
    echo "Integration:"
    echo "  resource-postgres credentials --format pretty  # Show credentials"
    echo "  resource-postgres status                       # Check all instances"
    echo ""
    echo "SQL Features:"
    echo "  • ACID transactions and full SQL compliance"
    echo "  • Multi-instance support for development/staging/production"
    echo "  • Advanced indexing and query optimization"
    echo "  • JSON/JSONB support for document storage"
    echo ""
    echo "Default Port: 5432"
    echo "Templates: development, testing, production"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi