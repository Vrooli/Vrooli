#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL Client Database Setup Example
# Demonstrates complete client database environment setup for Vrooli's platform factory

# This script shows how to:
# - Create a dedicated PostgreSQL instance for a client
# - Configure the instance with appropriate template
# - Create client-specific databases and users
# - Run migrations and seed initial data
# - Configure backups and monitoring
# - Package the configuration for deployment

#######################################
# Configuration
#######################################

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/*/*/*}" && builtin pwd)}"
POSTGRES_DIR="${APP_ROOT}/resources/postgres"

# Source the PostgreSQL management script
# shellcheck disable=SC1091
source "${POSTGRES_DIR}/manage.sh"

#######################################
# Parse command line arguments
#######################################
usage() {
    cat << EOF
Client Database Setup Example

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --client-name <name>    Client name (required)
    --template <template>   PostgreSQL template (development|production|testing|minimal)
    --port <port>          Custom port (optional, auto-assigned if not specified)
    --migrations <path>    Path to migrations directory (optional)
    --seeds <path>         Path to seeds directory (optional)
    --create-users         Create additional client users (optional)
    --setup-backup         Setup automatic backups (optional)
    --help                 Show this help message

EXAMPLES:
    # Basic client setup
    $0 --client-name real-estate --template production

    # Full setup with migrations and seeds
    $0 --client-name ecommerce --template production \\
       --migrations ./clients/ecommerce/migrations \\
       --seeds ./clients/ecommerce/seeds \\
       --create-users --setup-backup

    # Development setup
    $0 --client-name testing-client --template development --port 5440

EOF
}

# Parse arguments
CLIENT_NAME=""
TEMPLATE="production"
PORT=""
MIGRATIONS_DIR=""
SEEDS_DIR=""
CREATE_USERS=false
SETUP_BACKUP=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --client-name)
            CLIENT_NAME="$2"
            shift 2
            ;;
        --template)
            TEMPLATE="$2"
            shift 2
            ;;
        --port)
            PORT="$2"
            shift 2
            ;;
        --migrations)
            MIGRATIONS_DIR="$2"
            shift 2
            ;;
        --seeds)
            SEEDS_DIR="$2"
            shift 2
            ;;
        --create-users)
            CREATE_USERS=true
            shift
            ;;
        --setup-backup)
            SETUP_BACKUP=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate required arguments
if [[ -z "$CLIENT_NAME" ]]; then
    echo "Error: --client-name is required"
    usage
    exit 1
fi

# Validate client name
if [[ ! "$CLIENT_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]; then
    echo "Error: Client name can only contain letters, numbers, hyphens, and underscores"
    exit 1
fi

#######################################
# Client Database Setup Functions
#######################################

log_step() {
    echo "===================="
    echo "STEP: $1"
    echo "===================="
}

log_info() {
    echo "INFO: $1"
}

log_success() {
    echo "SUCCESS: $1"
}

log_error() {
    echo "ERROR: $1" >&2
}

#######################################
# Setup client database environment
#######################################
setup_client_database() {
    local client_name="$1"
    local template="$2"
    local port="$3"
    
    log_step "Creating PostgreSQL instance for client: $client_name"
    
    # Create PostgreSQL instance
    local create_args=(
        "--action" "create"
        "--instance" "$client_name"
        "--template" "$template"
    )
    
    if [[ -n "$port" ]]; then
        create_args+=("--port" "$port")
    fi
    
    if "${POSTGRES_DIR}/manage.sh" "${create_args[@]}"; then
        log_success "PostgreSQL instance '$client_name' created successfully"
    else
        log_error "Failed to create PostgreSQL instance"
        return 1
    fi
    
    # Wait for instance to be ready
    log_info "Waiting for instance to be ready..."
    sleep 5
    
    # Verify instance is running and healthy
    if "${POSTGRES_DIR}/manage.sh" --action status --instance "$client_name"; then
        log_success "Instance is running and healthy"
    else
        log_error "Instance is not healthy"
        return 1
    fi
}

#######################################
# Create client-specific databases
#######################################
create_client_databases() {
    local client_name="$1"
    
    log_step "Creating client databases"
    
    # Create main application database
    local app_db="${client_name}_app"
    log_info "Creating application database: $app_db"
    
    if "${POSTGRES_DIR}/manage.sh" --action create-db --instance "$client_name" --database "$app_db"; then
        log_success "Application database created: $app_db"
    else
        log_error "Failed to create application database"
        return 1
    fi
    
    # Create analytics database (optional)
    local analytics_db="${client_name}_analytics"
    log_info "Creating analytics database: $analytics_db"
    
    if "${POSTGRES_DIR}/manage.sh" --action create-db --instance "$client_name" --database "$analytics_db"; then
        log_success "Analytics database created: $analytics_db"
    else
        log_error "Failed to create analytics database"
        return 1
    fi
    
    # Create audit database (optional)
    local audit_db="${client_name}_audit"
    log_info "Creating audit database: $audit_db"
    
    if "${POSTGRES_DIR}/manage.sh" --action create-db --instance "$client_name" --database "$audit_db"; then
        log_success "Audit database created: $audit_db"
    else
        log_error "Failed to create audit database"
        return 1
    fi
}

#######################################
# Create client users
#######################################
create_client_users() {
    local client_name="$1"
    
    if [[ "$CREATE_USERS" != "true" ]]; then
        log_info "Skipping user creation (not requested)"
        return 0
    fi
    
    log_step "Creating client users"
    
    # Create application user
    local app_user="${client_name}_app"
    local app_password=$(openssl rand -base64 32)
    
    log_info "Creating application user: $app_user"
    
    if "${POSTGRES_DIR}/manage.sh" --action create-user --instance "$client_name" --username "$app_user" --password "$app_password"; then
        log_success "Application user created: $app_user"
        echo "Application user password: $app_password"
    else
        log_error "Failed to create application user"
        return 1
    fi
    
    # Create read-only user
    local readonly_user="${client_name}_readonly"
    local readonly_password=$(openssl rand -base64 32)
    
    log_info "Creating read-only user: $readonly_user"
    
    if "${POSTGRES_DIR}/manage.sh" --action create-user --instance "$client_name" --username "$readonly_user" --password "$readonly_password"; then
        log_success "Read-only user created: $readonly_user"
        echo "Read-only user password: $readonly_password"
        
        # Grant read-only permissions (this would need to be done after tables are created)
        log_info "Note: Grant read-only permissions after schema creation"
    else
        log_error "Failed to create read-only user"
        return 1
    fi
}

#######################################
# Run migrations
#######################################
run_client_migrations() {
    local client_name="$1"
    local migrations_dir="$2"
    
    if [[ -z "$migrations_dir" ]]; then
        log_info "No migrations directory specified, skipping migrations"
        return 0
    fi
    
    if [[ ! -d "$migrations_dir" ]]; then
        log_error "Migrations directory not found: $migrations_dir"
        return 1
    fi
    
    log_step "Running client migrations"
    
    # Initialize migration system
    log_info "Initializing migration system"
    if "${POSTGRES_DIR}/manage.sh" --action migrate-init --instance "$client_name" --database "${client_name}_app"; then
        log_success "Migration system initialized"
    else
        log_error "Failed to initialize migration system"
        return 1
    fi
    
    # Run migrations
    log_info "Running migrations from: $migrations_dir"
    if "${POSTGRES_DIR}/manage.sh" --action migrate --instance "$client_name" --migrations-dir "$migrations_dir" --database "${client_name}_app"; then
        log_success "Migrations completed successfully"
    else
        log_error "Failed to run migrations"
        return 1
    fi
    
    # Show migration status
    log_info "Migration status:"
    "${POSTGRES_DIR}/manage.sh" --action migrate-status --instance "$client_name" --database "${client_name}_app"
}

#######################################
# Seed initial data
#######################################
seed_client_data() {
    local client_name="$1"
    local seeds_dir="$2"
    
    if [[ -z "$seeds_dir" ]]; then
        log_info "No seeds directory specified, skipping data seeding"
        return 0
    fi
    
    if [[ ! -d "$seeds_dir" ]]; then
        log_error "Seeds directory not found: $seeds_dir"
        return 1
    fi
    
    log_step "Seeding initial data"
    
    log_info "Seeding data from: $seeds_dir"
    if "${POSTGRES_DIR}/manage.sh" --action seed --instance "$client_name" --seed-path "$seeds_dir" --database "${client_name}_app"; then
        log_success "Data seeding completed successfully"
    else
        log_error "Failed to seed data"
        return 1
    fi
}

#######################################
# Setup automated backups
#######################################
setup_client_backups() {
    local client_name="$1"
    
    if [[ "$SETUP_BACKUP" != "true" ]]; then
        log_info "Skipping backup setup (not requested)"
        return 0
    fi
    
    log_step "Setting up automated backups"
    
    # Create initial backup
    local backup_name="initial_setup_$(date +%Y%m%d_%H%M%S)"
    log_info "Creating initial backup: $backup_name"
    
    if "${POSTGRES_DIR}/manage.sh" --action backup --instance "$client_name" --backup-name "$backup_name" --backup-type "full"; then
        log_success "Initial backup created: $backup_name"
    else
        log_error "Failed to create initial backup"
        return 1
    fi
    
    # Show backup information
    log_info "Backup information:"
    "${POSTGRES_DIR}/manage.sh" --action list-backups --instance "$client_name"
    
    # Create backup script for client
    local backup_script="/tmp/${client_name}_backup.sh"
    cat > "$backup_script" << EOF
#!/usr/bin/env bash
# Automated backup script for client: $client_name
# Generated on: $(date)

POSTGRES_DIR="\$(builtin cd "\${BASH_SOURCE[0]%/*/*}" && builtin pwd)"

# Create daily backup
BACKUP_NAME="daily_\$(date +%Y%m%d_%H%M%S)"

if "\${POSTGRES_DIR}/manage.sh" --action backup --instance "$client_name" --backup-name "\$BACKUP_NAME" --backup-type "full"; then
    echo "Backup created successfully: \$BACKUP_NAME"
    
    # Cleanup old backups (keep last 7 days)
    "\${POSTGRES_DIR}/manage.sh" --action cleanup-backups --instance "$client_name" --retention-days 7
else
    echo "Backup failed" >&2
    exit 1
fi
EOF
    
    chmod +x "$backup_script"
    log_success "Backup script created: $backup_script"
    log_info "Add this to cron for daily backups: 0 2 * * * $backup_script"
}

#######################################
# Generate client configuration
#######################################
generate_client_config() {
    local client_name="$1"
    
    log_step "Generating client configuration"
    
    # Get connection information
    local conn_info=$("${POSTGRES_DIR}/manage.sh" --action connect --instance "$client_name" 2>/dev/null || echo "")
    
    if [[ -z "$conn_info" ]]; then
        log_error "Failed to get connection information"
        return 1
    fi
    
    # Extract connection details
    local host="localhost"
    local port=$(echo "$conn_info" | grep "Port:" | awk '{print $2}')
    local database="${client_name}_app"
    local username="vrooli"
    local password=$(echo "$conn_info" | grep "Password:" | awk '{print $2}')
    
    # Generate configuration file
    local config_file="/tmp/${client_name}_config.json"
    cat > "$config_file" << EOF
{
  "client": {
    "name": "$client_name",
    "created": "$(date -Iseconds)",
    "template": "$template"
  },
  "database": {
    "instance": "$client_name",
    "host": "$host",
    "port": $port,
    "databases": {
      "application": "${client_name}_app",
      "analytics": "${client_name}_analytics",
      "audit": "${client_name}_audit"
    },
    "connection": {
      "username": "$username",
      "password": "$password",
      "ssl": false
    },
    "connection_string": "postgresql://$username:$password@$host:$port/${client_name}_app"
  },
  "backup": {
    "enabled": $SETUP_BACKUP,
    "retention_days": 7,
    "backup_types": ["full", "schema", "data"]
  },
  "monitoring": {
    "health_check_url": "http://$host:$port/health",
    "metrics_enabled": true
  }
}
EOF
    
    log_success "Client configuration generated: $config_file"
    
    # Generate environment file
    local env_file="/tmp/${client_name}.env"
    cat > "$env_file" << EOF
# Environment configuration for client: $client_name
# Generated on: $(date)

# Database Configuration
DATABASE_HOST=$host
DATABASE_PORT=$port
DATABASE_NAME=${client_name}_app
DATABASE_USER=$username
DATABASE_PASSWORD=$password
DATABASE_URL=postgresql://$username:$password@$host:$port/${client_name}_app

# Additional Databases
ANALYTICS_DATABASE=${client_name}_analytics
AUDIT_DATABASE=${client_name}_audit

# Instance Information
POSTGRES_INSTANCE=$client_name
POSTGRES_TEMPLATE=$template

# Backup Configuration
BACKUP_ENABLED=$SETUP_BACKUP
BACKUP_RETENTION_DAYS=7
EOF
    
    log_success "Environment file generated: $env_file"
    
    return 0
}

#######################################
# Display setup summary
#######################################
display_setup_summary() {
    local client_name="$1"
    
    echo ""
    echo "===================="
    echo "SETUP COMPLETE"
    echo "===================="
    echo ""
    echo "Client: $client_name"
    echo "Template: $template"
    echo ""
    
    # Show instance status
    echo "Instance Status:"
    "${POSTGRES_DIR}/manage.sh" --action status --instance "$client_name"
    echo ""
    
    # Show connection information
    echo "Connection Information:"
    "${POSTGRES_DIR}/manage.sh" --action connect --instance "$client_name"
    echo ""
    
    # Show databases
    echo "Databases Created:"
    echo "  - ${client_name}_app (main application)"
    echo "  - ${client_name}_analytics (analytics data)"
    echo "  - ${client_name}_audit (audit logs)"
    echo ""
    
    if [[ "$CREATE_USERS" == "true" ]]; then
        echo "Additional Users Created:"
        echo "  - ${client_name}_app (application user)"
        echo "  - ${client_name}_readonly (read-only user)"
        echo ""
    fi
    
    if [[ -n "$MIGRATIONS_DIR" ]]; then
        echo "Migrations Applied: Yes"
        echo "Migration Status:"
        "${POSTGRES_DIR}/manage.sh" --action migrate-status --instance "$client_name" --database "${client_name}_app"
        echo ""
    fi
    
    if [[ -n "$SEEDS_DIR" ]]; then
        echo "Data Seeded: Yes"
        echo ""
    fi
    
    if [[ "$SETUP_BACKUP" == "true" ]]; then
        echo "Backups Configured: Yes"
        echo "Backup Location: ~/.vrooli/backups/postgres/$client_name"
        echo ""
    fi
    
    echo "Configuration Files:"
    echo "  - Client Config: /tmp/${client_name}_config.json"
    echo "  - Environment: /tmp/${client_name}.env"
    
    if [[ "$SETUP_BACKUP" == "true" ]]; then
        echo "  - Backup Script: /tmp/${client_name}_backup.sh"
    fi
    
    echo ""
    echo "Next Steps:"
    echo "1. Copy configuration files to your application"
    echo "2. Test database connectivity"
    
    if [[ "$CREATE_USERS" == "true" ]]; then
        echo "3. Configure application to use dedicated users"
    fi
    
    if [[ "$SETUP_BACKUP" == "true" ]]; then
        echo "4. Schedule backup script in cron"
    fi
    
    echo "5. Monitor instance health regularly"
    echo ""
    echo "Useful Commands:"
    echo "  Status:    ${POSTGRES_DIR}/manage.sh --action status --instance $client_name"
    echo "  Logs:      ${POSTGRES_DIR}/manage.sh --action logs --instance $client_name"
    echo "  Backup:    ${POSTGRES_DIR}/manage.sh --action backup --instance $client_name"
    echo "  Connect:   ${POSTGRES_DIR}/manage.sh --action connect --instance $client_name"
}

#######################################
# Main execution
#######################################
main() {
    echo "PostgreSQL Client Database Setup"
    echo "================================"
    echo "Client: $CLIENT_NAME"
    echo "Template: $TEMPLATE"
    echo "Port: ${PORT:-auto-assigned}"
    echo "Migrations: ${MIGRATIONS_DIR:-none}"
    echo "Seeds: ${SEEDS_DIR:-none}"
    echo "Create Users: $CREATE_USERS"
    echo "Setup Backup: $SETUP_BACKUP"
    echo ""
    
    # Confirm setup
    read -p "Proceed with client database setup? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Setup cancelled"
        exit 0
    fi
    
    # Execute setup steps
    setup_client_database "$CLIENT_NAME" "$TEMPLATE" "$PORT" || exit 1
    create_client_databases "$CLIENT_NAME" || exit 1
    create_client_users "$CLIENT_NAME" || exit 1
    run_client_migrations "$CLIENT_NAME" "$MIGRATIONS_DIR" || exit 1
    seed_client_data "$CLIENT_NAME" "$SEEDS_DIR" || exit 1
    setup_client_backups "$CLIENT_NAME" || exit 1
    generate_client_config "$CLIENT_NAME" || exit 1
    
    # Display summary
    display_setup_summary "$CLIENT_NAME"
    
    echo "Client database setup completed successfully!"
}

# Execute main function
main "$@"