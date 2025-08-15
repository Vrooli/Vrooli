#!/usr/bin/env bash
################################################################################
# PostgreSQL Resource CLI
# 
# Lightweight CLI interface for PostgreSQL that delegates to existing lib functions.
#
# Usage:
#   resource-postgres <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (handle symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    POSTGRES_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    POSTGRES_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
POSTGRES_CLI_DIR="$(cd "$(dirname "$POSTGRES_CLI_SCRIPT")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$POSTGRES_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$POSTGRES_CLI_DIR"
export POSTGRES_SCRIPT_DIR="$POSTGRES_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

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

# Initialize with resource name
resource_cli::init "postgres"

################################################################################
# Delegate to existing PostgreSQL functions
################################################################################

# Inject SQL or configuration into PostgreSQL
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-postgres inject <file.sql>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would inject: $file"
        return 0
    fi
    
    # Use existing injection function
    if command -v postgres::inject_file &>/dev/null; then
        postgres::inject_file "$file"
    elif command -v postgres::inject_sql &>/dev/null; then
        postgres::inject_sql "$file"
    else
        log::error "PostgreSQL injection functions not available"
        return 1
    fi
}

# Validate PostgreSQL configuration
resource_cli::validate() {
    if command -v postgres::validate &>/dev/null; then
        postgres::validate
    elif command -v postgres::check_health &>/dev/null; then
        postgres::check_health
    else
        # Basic validation
        log::header "Validating PostgreSQL"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "postgres" || {
            log::error "PostgreSQL container not running"
            return 1
        }
        log::success "PostgreSQL is running"
    fi
}

# Show PostgreSQL status
resource_cli::status() {
    if command -v postgres::status &>/dev/null; then
        postgres::status
    else
        # Basic status
        log::header "PostgreSQL Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "postgres"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=postgres" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start PostgreSQL
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start PostgreSQL"
        return 0
    fi
    
    if command -v postgres::start &>/dev/null; then
        postgres::start
    elif command -v postgres::docker::start &>/dev/null; then
        postgres::docker::start
    else
        docker start postgres || log::error "Failed to start PostgreSQL"
    fi
}

# Stop PostgreSQL
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop PostgreSQL"
        return 0
    fi
    
    if command -v postgres::stop &>/dev/null; then
        postgres::stop
    elif command -v postgres::docker::stop &>/dev/null; then
        postgres::docker::stop
    else
        docker stop postgres || log::error "Failed to stop PostgreSQL"
    fi
}

# Install PostgreSQL
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install PostgreSQL"
        return 0
    fi
    
    if command -v postgres::install &>/dev/null; then
        postgres::install
    else
        log::error "postgres::install not available"
        return 1
    fi
}

# Uninstall PostgreSQL
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove PostgreSQL and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall PostgreSQL"
        return 0
    fi
    
    if command -v postgres::uninstall &>/dev/null; then
        postgres::uninstall
    else
        docker stop postgres 2>/dev/null || true
        docker rm postgres 2>/dev/null || true
        log::success "PostgreSQL uninstalled"
    fi
}

################################################################################
# PostgreSQL-specific commands (if functions exist)
################################################################################

# List instances
postgres_list_instances() {
    # Ensure messages are initialized
    postgres::messages::init 2>/dev/null || true
    
    if command -v postgres::instance::list &>/dev/null; then
        postgres::instance::list
    elif command -v postgres::list_instances &>/dev/null; then
        postgres::list_instances
    else
        # Basic instance listing using Docker
        log::header "PostgreSQL Instances"
        docker ps --filter "name=postgres" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || {
            echo "No PostgreSQL instances running"
        }
    fi
}

# Create database
postgres_create_database() {
    local database_name="${1:-}"
    local instance_name="${2:-main}"
    
    if [[ -z "$database_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-postgres create-database <name> [instance]"
        return 1
    fi
    
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would create database: $database_name on instance: $instance_name"
        return 0
    fi
    
    if command -v postgres::database::create &>/dev/null; then
        postgres::database::create "$instance_name" "$database_name"
    else
        log::error "Database creation not available"
        return 1
    fi
}

# Execute SQL
postgres_execute_sql() {
    local sql_command="${1:-}"
    local instance_name="${2:-main}"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    
    # Ensure messages are initialized
    postgres::messages::init 2>/dev/null || true
    
    if [[ -z "$sql_command" ]]; then
        log::error "SQL command required"
        echo "Usage: resource-postgres execute-sql '<command>' [instance] [database]"
        return 1
    fi
    
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would execute SQL on instance: $instance_name"
        log::info "[DRY RUN] SQL: $sql_command"
        return 0
    fi
    
    if command -v postgres::database::execute &>/dev/null; then
        postgres::database::execute "$instance_name" "$sql_command" "$database"
    else
        log::error "SQL execution not available"
        return 1
    fi
}

# List databases
postgres_list_databases() {
    local instance_name="${1:-main}"
    
    if command -v postgres::database::list &>/dev/null; then
        postgres::database::list "$instance_name"
    else
        # Use SQL execution to list databases
        if command -v postgres::database::execute &>/dev/null; then
            postgres::database::execute "$instance_name" "\\l" "postgres"
        else
            log::error "Database listing not available"
            return 1
        fi
    fi
}

# Create backup
postgres_create_backup() {
    local database_name="${1:-}"
    local instance_name="${2:-main}"
    local backup_file="${3:-}"
    
    if [[ -z "$database_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-postgres create-backup <database> [instance] [backup-file]"
        return 1
    fi
    
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would create backup of database: $database_name from instance: $instance_name"
        return 0
    fi
    
    if command -v postgres::backup::create &>/dev/null; then
        postgres::backup::create "$instance_name" "$database_name" "$backup_file"
    else
        log::error "Backup creation not available"
        return 1
    fi
}

# Create instance
postgres_create_instance() {
    local instance_name="${1:-}"
    local template="${2:-development}"
    
    if [[ -z "$instance_name" ]]; then
        log::error "Instance name required"
        echo "Usage: resource-postgres create-instance <name> [template]"
        return 1
    fi
    
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would create instance: $instance_name with template: $template"
        return 0
    fi
    
    if command -v postgres::instance::create &>/dev/null; then
        # Pass empty string for port to let it auto-assign
        postgres::instance::create "$instance_name" "" "$template"
    else
        log::error "Instance creation not available"
        return 1
    fi
}

# Get credentials for n8n integration
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    # Parse arguments
    credentials::parse_args "$@"
    local parse_result=$?
    if [[ $parse_result -eq 2 ]]; then
        credentials::show_help "postgres"
        return 0
    elif [[ $parse_result -ne 0 ]]; then
        return 1
    fi
    
    # Get all PostgreSQL containers (instances)
    local connections=()
    local overall_status="stopped"
    
    # Check for PostgreSQL containers
    if command -v docker &>/dev/null; then
        # Get all containers with postgres prefix
        local containers
        containers=$(docker ps -a --filter "name=^/${POSTGRES_CONTAINER_PREFIX}-" --format "{{.Names}}:{{.Status}}")
        
        if [[ -n "$containers" ]]; then
            overall_status="running"
            
            while IFS=: read -r container_name container_status; do
                # Skip if container is not running
                if [[ ! "$container_status" =~ ^Up ]]; then
                    continue
                fi
                
                # Extract instance info from container name
                local instance_name="${container_name#$POSTGRES_CONTAINER_PREFIX-}"
                
                # Get container port mapping
                local port_mapping
                port_mapping=$(docker port "$container_name" 5432 2>/dev/null | head -1)
                local host_port="${port_mapping##*:}"
                host_port="${host_port:-5432}"
                
                # Get environment variables for database info
                local db_name db_user db_password
                db_name=$(docker inspect "$container_name" --format '{{range .Config.Env}}{{if (index (split . "=") 0 | eq "POSTGRES_DB")}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
                db_name="${db_name:-${POSTGRES_DEFAULT_DB}}"
                
                db_user=$(docker inspect "$container_name" --format '{{range .Config.Env}}{{if (index (split . "=") 0 | eq "POSTGRES_USER")}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
                db_user="${db_user:-${POSTGRES_DEFAULT_USER}}"
                
                db_password=$(docker inspect "$container_name" --format '{{range .Config.Env}}{{if (index (split . "=") 0 | eq "POSTGRES_PASSWORD")}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
                db_password="${db_password:-postgres}"
                
                # Create connection object (compact JSON to avoid multiline issues)
                local connection_obj
                connection_obj=$(jq -n -c --arg host "localhost" --argjson port "$host_port" --arg database "$db_name" '{host: $host, port: $port, database: $database, ssl: false}')
                
                local auth_obj
                auth_obj=$(jq -n -c --arg username "$db_user" --arg password "$db_password" '{username: $username, password: $password}')
                # Remove any trailing extra braces that might be added by bash parsing
                auth_obj="${auth_obj%\}}"
                
                local metadata_obj
                metadata_obj=$(jq -n -c --arg description "PostgreSQL instance: $instance_name" --argjson capabilities '["sql", "transactions", "acid"]' --arg version "16" '{description: $description, capabilities: $capabilities, version: $version}')
                # Remove any trailing extra braces that might be added by bash parsing
                metadata_obj="${metadata_obj%\}}"
                
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
        overall_status="running"
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "postgres" "$overall_status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 PostgreSQL Resource CLI

USAGE:
    resource-postgres <command> [options]

CORE COMMANDS:
    inject <file>       Inject SQL/config into PostgreSQL
    validate            Validate PostgreSQL configuration
    status              Show PostgreSQL status
    start               Start PostgreSQL container
    stop                Stop PostgreSQL container
    install             Install PostgreSQL
    uninstall           Uninstall PostgreSQL (requires --force)
    credentials         Get connection credentials for n8n integration
    
POSTGRES COMMANDS:
    list-instances      List all PostgreSQL instances
    create-instance <name> [template]  Create new instance
    list-databases [instance]          List databases in instance
    create-database <name> [instance]  Create new database
    execute-sql '<sql>' [instance] [db] Execute SQL command
    create-backup <db> [instance] [file] Create database backup

OPTIONS:
    --verbose, -v       Show detailed output
    --dry-run           Preview actions without executing
    --force             Force operation (skip confirmations)

EXAMPLES:
    resource-postgres status
    resource-postgres list-instances
    resource-postgres credentials --format pretty
    resource-postgres create-database myapp main
    resource-postgres execute-sql 'SELECT version();' main postgres
    resource-postgres create-backup myapp main /backups/myapp.sql
    resource-postgres inject shared:initialization/storage/postgres/schema.sql

For more information: https://docs.vrooli.com/resources/postgres
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first but preserve arguments properly
    export VERBOSE=false
    export DRY_RUN=false
    export FORCE=false
    
    # Process options while preserving remaining args
    local args=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --force)
                FORCE=true
                shift
                ;;
            *)
                args+=("$1")
                shift
                ;;
        esac
    done
    
    # Reset args to non-option arguments
    set -- "${args[@]}"
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall|credentials)
            resource_cli::$command "$@"
            ;;
            
        # PostgreSQL-specific commands
        list-instances)
            postgres_list_instances "$@"
            ;;
        create-instance)
            postgres_create_instance "$@"
            ;;
        list-databases)
            postgres_list_databases "$@"
            ;;
        create-database)
            postgres_create_database "$@"
            ;;
        execute-sql)
            postgres_execute_sql "$@"
            ;;
        create-backup)
            postgres_create_backup "$@"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi