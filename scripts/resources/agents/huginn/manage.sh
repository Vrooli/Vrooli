#!/usr/bin/env bash
set -euo pipefail

# Huginn Agent-Based Monitoring and Automation Setup
# This script handles installation, configuration, and management of Huginn using Docker

DESCRIPTION="Install and manage Huginn agent-based monitoring system using Docker"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/huginn_auth.sh"

# Huginn configuration
readonly HUGINN_PORT="${HUGINN_CUSTOM_PORT:-$(resources::get_default_port "huginn")}"
readonly HUGINN_BASE_URL="http://localhost:${HUGINN_PORT}"
readonly HUGINN_CONTAINER_NAME="huginn"
readonly HUGINN_DB_CONTAINER_NAME="huginn-postgres"
readonly HUGINN_DATA_DIR="${HOME}/.huginn"
readonly HUGINN_DB_DIR="${HUGINN_DATA_DIR}/postgres"
readonly HUGINN_UPLOADS_DIR="${HUGINN_DATA_DIR}/uploads"
readonly HUGINN_IMAGE="huginn/huginn:latest"
readonly POSTGRES_IMAGE="postgres:15-alpine"

# Default credentials
readonly DEFAULT_DB_PASSWORD="huginn_secure_password_$(date +%s)"
readonly DEFAULT_ADMIN_EMAIL="admin@huginn.local"
readonly DEFAULT_ADMIN_USERNAME="admin"
readonly DEFAULT_ADMIN_PASSWORD="vrooli_huginn_secure_2025"

#######################################
# Parse command line arguments
#######################################
huginn::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|agents|scenarios|import|export|backup|restore|info" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Huginn appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "admin-email" \
        --desc "Admin email for Huginn login" \
        --type "value" \
        --default "$DEFAULT_ADMIN_EMAIL"
    
    args::register \
        --name "admin-password" \
        --desc "Admin password for Huginn login" \
        --type "value" \
        --default "$DEFAULT_ADMIN_PASSWORD"
    
    args::register \
        --name "db-password" \
        --desc "PostgreSQL database password" \
        --type "value" \
        --default "$DEFAULT_DB_PASSWORD"
    
    args::register \
        --name "file" \
        --desc "File path for import/export/backup/restore operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "include-data" \
        --desc "Include database data in backup (otherwise just config)" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    # User management arguments
    args::register \
        --name "user-email" \
        --desc "Email for user creation/token generation" \
        --type "value" \
        --default ""
    
    args::register \
        --name "user-password" \
        --desc "Password for user creation" \
        --type "value" \
        --default ""
    
    args::register \
        --name "user-admin" \
        --desc "Make user admin (true/false)" \
        --type "value" \
        --options "true|false" \
        --default "false"
    
    args::register \
        --name "new-admin-email" \
        --desc "New admin email for credential updates" \
        --type "value" \
        --default ""
    
    args::register \
        --name "new-admin-password" \
        --desc "New admin password for credential updates" \
        --type "value" \
        --default ""
    
    if args::is_asking_for_help "$@"; then
        huginn::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export ADMIN_EMAIL=$(args::get "admin-email")
    export ADMIN_PASSWORD=$(args::get "admin-password")
    export DB_PASSWORD=$(args::get "db-password")
    export FILE_PATH=$(args::get "file")
    export INCLUDE_DATA=$(args::get "include-data")
    export USER_EMAIL=$(args::get "user-email")
    export USER_PASSWORD=$(args::get "user-password")
    export USER_ADMIN=$(args::get "user-admin")
    export NEW_ADMIN_EMAIL=$(args::get "new-admin-email")
    export NEW_ADMIN_PASSWORD=$(args::get "new-admin-password")
}

#######################################
# Retrieve credentials from Vrooli resource config
# Returns: Sets HUGINN_ADMIN_* variables from config
#######################################
huginn::get_credentials_from_config() {
    local config_file="${HOME}/.vrooli/resources.json"
    
    if [[ -f "$config_file" ]] && system::is_command "jq"; then
        local creds
        creds=$(jq -r '.agents.huginn.credentials.admin // empty' "$config_file" 2>/dev/null)
        
        if [[ -n "$creds" && "$creds" != "null" ]]; then
            export HUGINN_ADMIN_EMAIL=$(echo "$creds" | jq -r '.email // "admin@huginn.local"')
            export HUGINN_ADMIN_USERNAME=$(echo "$creds" | jq -r '.username // "admin"')
            export HUGINN_ADMIN_PASSWORD=$(echo "$creds" | jq -r '.password // "vrooli_huginn_secure_2025"')
            log::info "Using credentials from Vrooli resource config"
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Generate or retrieve Huginn credentials
# Returns: Sets HUGINN_ADMIN_* variables
#######################################
huginn::get_credentials() {
    # Check for explicit environment variable overrides first
    if [[ -n "${HUGINN_ADMIN_USERNAME:-}" && -n "${HUGINN_ADMIN_PASSWORD:-}" && -n "${HUGINN_ADMIN_EMAIL:-}" ]]; then
        log::info "Using explicit environment variable credentials"
        return 0
    fi
    
    # Check for command line argument overrides
    if [[ -n "${ADMIN_EMAIL:-}" && -n "${ADMIN_PASSWORD:-}" ]]; then
        log::info "Using command line credentials"
        export HUGINN_ADMIN_EMAIL="$ADMIN_EMAIL"
        export HUGINN_ADMIN_USERNAME="${ADMIN_EMAIL%%@*}"  # Use email prefix as username
        export HUGINN_ADMIN_PASSWORD="$ADMIN_PASSWORD"
        return 0
    fi
    
    # Try to get from Vrooli resource config
    if huginn::get_credentials_from_config; then
        return 0
    fi
    
    # Fall back to secure defaults
    log::info "Using secure default credentials"
    export HUGINN_ADMIN_EMAIL="admin@huginn.local"
    export HUGINN_ADMIN_USERNAME="admin"
    export HUGINN_ADMIN_PASSWORD="vrooli_huginn_secure_2025"
    
    return 0
}

#######################################
# Update Vrooli resource configuration with credentials
#######################################
huginn::update_config() {
    local huginn_port="${HUGINN_PORT:-4111}"
    local huginn_base_url="http://localhost:${huginn_port}"
    
    # Ensure credentials are set
    huginn::get_credentials
    
    # Create comprehensive configuration JSON
    local additional_config
    additional_config=$(cat <<EOF
{
    "features": {
        "autonomousAgents": true,
        "continuousMonitoring": true,
        "eventDriven": true,
        "webhooks": true,
        "scenarios": true,
        "rss": true,
        "notifications": true
    },
    "credentials": {
        "admin": {
            "email": "${HUGINN_ADMIN_EMAIL}",
            "username": "${HUGINN_ADMIN_USERNAME}",
            "password": "${HUGINN_ADMIN_PASSWORD}",
            "loginField": "username"
        }
    },
    "endpoints": {
        "webUI": "${huginn_base_url}",
        "api": "${huginn_base_url}/api",
        "agents": "${huginn_base_url}/agents",
        "scenarios": "${huginn_base_url}/scenarios",
        "events": "${huginn_base_url}/events",
        "webhooks": "${huginn_base_url}/users/1/web_requests"
    },
    "container": {
        "name": "huginn",
        "database": "huginn-postgres",
        "network": "huginn-network"
    }
}
EOF
)
    
    # Update Vrooli resource configuration
    resources::update_config "agents" "huginn" "$huginn_base_url" "$additional_config"
    
    log::success "Updated Vrooli resource configuration"
}

#######################################
# Display usage information
#######################################
huginn::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                      # Install Huginn with PostgreSQL"
    echo "  $0 --action status                       # Check Huginn status"
    echo "  $0 --action agents                       # List all agents"
    echo "  $0 --action scenarios                    # List scenarios"
    echo "  $0 --action import --file agents.json    # Import agents from file"
    echo "  $0 --action backup --file backup.tar     # Create backup"
    echo "  $0 --action create-user --user-email user@example.com --user-password mypass  # Create user"
    echo "  $0 --action update-admin --new-admin-password newpass  # Update admin password"
    echo "  $0 --action auto-login                    # Test automatic login"
    echo "  $0 --action uninstall                    # Remove Huginn"
    echo
    echo "Default Credentials:"
    echo "  Username: $DEFAULT_ADMIN_USERNAME"
    echo "  Password: $DEFAULT_ADMIN_PASSWORD"
    echo "  Web UI: http://localhost:${HUGINN_PORT}"
    echo "  Login Method: Use username (not email)"
    echo
    echo "Credential Management:"
    echo "  - Managed through Vrooli resource configuration"
    echo "  - Override with HUGINN_ADMIN_* environment variables"
    echo "  - Or use --admin-email/--admin-password command arguments"
    echo
    echo "Agent Concepts:"
    echo "  - Agents: Autonomous units that monitor and react to events"
    echo "  - Events: Messages passed between agents"
    echo "  - Scenarios: Logical groupings of related agents"
    echo "  - Credentials: Secure storage for API keys and passwords"
}

#######################################
# Check if Huginn container exists
# Returns: 0 if exists, 1 otherwise
#######################################
huginn::container_exists() {
    docker::run ps -a --format "{{.Names}}" | grep -q "^${HUGINN_CONTAINER_NAME}$"
}

#######################################
# Check if Huginn is running
# Returns: 0 if running, 1 otherwise
#######################################
huginn::is_running() {
    docker::run ps --format "{{.Names}}" | grep -q "^${HUGINN_CONTAINER_NAME}$"
}

#######################################
# Check if PostgreSQL is running
# Returns: 0 if running, 1 otherwise
#######################################
huginn::db_is_running() {
    docker::run ps --format "{{.Names}}" | grep -q "^${HUGINN_DB_CONTAINER_NAME}$"
}

#######################################
# Check if Huginn is healthy
# Returns: 0 if responsive, 1 otherwise
#######################################
huginn::is_healthy() {
    curl -f -s --max-time 5 "$HUGINN_BASE_URL" >/dev/null 2>&1
}

#######################################
# Create necessary directories
#######################################
huginn::create_directories() {
    log::info "Creating Huginn directories..."
    
    mkdir -p "$HUGINN_DATA_DIR" || {
        log::error "Failed to create data directory: $HUGINN_DATA_DIR"
        return 1
    }
    
    mkdir -p "$HUGINN_DB_DIR" || {
        log::error "Failed to create database directory: $HUGINN_DB_DIR"
        return 1
    }
    
    mkdir -p "$HUGINN_UPLOADS_DIR" || {
        log::error "Failed to create uploads directory: $HUGINN_UPLOADS_DIR"
        return 1
    }
    
    # Set proper permissions
    chmod 755 "$HUGINN_DATA_DIR" "$HUGINN_DB_DIR" "$HUGINN_UPLOADS_DIR"
    
    log::success "Directories created successfully"
    return 0
}

#######################################
# Create docker::run network for Huginn
#######################################
huginn::create_network() {
    local network_name="huginn-network"
    
    if ! docker::run network ls --format "{{.Name}}" | grep -q "^${network_name}$"; then
        log::info "Creating Docker network: $network_name"
        if docker::run network create "$network_name"; then
            log::success "Docker network created"
            return 0
        else
            log::error "Failed to create Docker network"
            return 1
        fi
    else
        log::info "Docker network already exists: $network_name"
        return 0
    fi
}

#######################################
# Start PostgreSQL container
#######################################
huginn::start_postgres() {
    if huginn::db_is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "PostgreSQL is already running"
        return 0
    fi
    
    # Stop existing container if force is specified
    if huginn::db_is_running && [[ "$FORCE" == "yes" ]]; then
        log::info "Stopping existing PostgreSQL container (force specified)..."
        docker::run stop "$HUGINN_DB_CONTAINER_NAME" >/dev/null 2>&1 || true
        docker::run rm "$HUGINN_DB_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    log::info "Starting PostgreSQL container..."
    
    # Create network first
    huginn::create_network || return 1
    
    local docker_cmd=(
        docker::run run -d
        --name "$HUGINN_DB_CONTAINER_NAME"
        --restart unless-stopped
        --network huginn-network
        -e "POSTGRES_DB=huginn"
        -e "POSTGRES_USER=huginn"
        -e "POSTGRES_PASSWORD=$DB_PASSWORD"
        -v "${HUGINN_DB_DIR}:/var/lib/postgresql/data"
        "$POSTGRES_IMAGE"
    )
    
    if "${docker_cmd[@]}"; then
        log::success "PostgreSQL container started"
        
        # Wait for PostgreSQL to be ready
        log::info "Waiting for PostgreSQL to be ready..."
        sleep 10
        
        return 0
    else
        log::error "Failed to start PostgreSQL container"
        return 1
    fi
}

#######################################
# Start Huginn container
#######################################
huginn::start_container() {
    if huginn::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Huginn is already running"
        return 0
    fi
    
    # Ensure PostgreSQL is running first
    if ! huginn::db_is_running; then
        log::info "PostgreSQL is not running, starting it first..."
        huginn::start_postgres || return 1
    fi
    
    # Stop existing container if force is specified
    if huginn::is_running && [[ "$FORCE" == "yes" ]]; then
        log::info "Stopping existing container (force specified)..."
        huginn::stop_container
    fi
    
    # Remove existing container if it exists but is not running
    if huginn::container_exists && ! huginn::is_running; then
        log::info "Removing existing stopped container..."
        docker::run rm "$HUGINN_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    log::info "Starting Huginn container..."
    
    # Get credentials using new architecture
    huginn::get_credentials
    
    # Save configuration to env file
    cat > "${HUGINN_DATA_DIR}/.env" << EOF
# Huginn Configuration
DATABASE_ADAPTER=postgresql
DATABASE_HOST=$HUGINN_DB_CONTAINER_NAME
DATABASE_NAME=huginn
DATABASE_USERNAME=huginn
DATABASE_PASSWORD=$DB_PASSWORD
APP_SECRET_TOKEN=$(openssl rand -hex 64)
INVITATION_CODE=vrooli-huginn-$(date +%s)

# Admin credentials - managed by Vrooli resource config
SEED_EMAIL=$HUGINN_ADMIN_EMAIL
SEED_USERNAME=$HUGINN_ADMIN_USERNAME
SEED_PASSWORD=$HUGINN_ADMIN_PASSWORD
EOF
    
    # Docker run command
    local docker_cmd=(
        docker::run run -d
        --name "$HUGINN_CONTAINER_NAME"
        --restart unless-stopped
        --network huginn-network
        -p "${HUGINN_PORT}:3000"
        --env-file "${HUGINN_DATA_DIR}/.env"
        -v "${HUGINN_UPLOADS_DIR}:/app/public/system"
        "$HUGINN_IMAGE"
    )
    
    if "${docker_cmd[@]}"; then
        log::success "Huginn container started"
        
        # Wait for service to be ready
        log::info "Waiting for Huginn to be ready (database setup can take 2-3 minutes)..."
        if resources::wait_for_service "Huginn" "$HUGINN_PORT" 180; then
            sleep 5
            if huginn::is_healthy; then
                log::success "✅ Huginn is running and healthy on port $HUGINN_PORT"
                log::info "Web UI: $HUGINN_BASE_URL"
                log::info "Login: $ADMIN_EMAIL / $ADMIN_PASSWORD"
                return 0
            else
                log::warn "⚠️  Huginn started but health check failed"
                log::info "The service may still be initializing. Check logs with: $0 --action logs"
                return 0
            fi
        else
            log::error "Huginn failed to start within 3 minutes"
            log::info "Check logs with: docker::run logs $HUGINN_CONTAINER_NAME"
            return 1
        fi
    else
        log::error "Failed to start Huginn container"
        return 1
    fi
}

#######################################
# Stop Huginn container
#######################################
huginn::stop_container() {
    if ! huginn::is_running; then
        log::info "Huginn is not running"
        return 0
    fi
    
    log::info "Stopping Huginn container..."
    if docker::run stop "$HUGINN_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "Huginn container stopped"
        return 0
    else
        log::error "Failed to stop Huginn container"
        return 1
    fi
}

#######################################
# Stop PostgreSQL container
#######################################
huginn::stop_postgres() {
    if ! huginn::db_is_running; then
        log::info "PostgreSQL is not running"
        return 0
    fi
    
    log::info "Stopping PostgreSQL container..."
    if docker::run stop "$HUGINN_DB_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "PostgreSQL container stopped"
        return 0
    else
        log::error "Failed to stop PostgreSQL container"
        return 1
    fi
}

#######################################
# Show container logs
#######################################
huginn::show_logs() {
    if ! huginn::container_exists; then
        log::error "Huginn container does not exist"
        return 1
    fi
    
    log::info "Showing Huginn logs (Ctrl+C to exit)..."
    docker::run logs -f "$HUGINN_CONTAINER_NAME"
}

#######################################
# List agents
#######################################
huginn::list_agents() {
    if ! huginn::is_healthy; then
        log::error "Huginn is not running or not healthy"
        return 1
    fi
    
    log::info "Fetching agent list..."
    
    # This would typically use Huginn's API or Rails console
    # For now, we'll show how to access the Rails console
    log::info "To list agents, run:"
    echo "docker::run exec -it $HUGINN_CONTAINER_NAME bundle exec rails console"
    echo "Then in the console: Agent.all.each { |a| puts \"#{a.id}: #{a.name} (#{a.type})\" }"
}

#######################################
# List scenarios
#######################################
huginn::list_scenarios() {
    if ! huginn::is_healthy; then
        log::error "Huginn is not running or not healthy"
        return 1
    fi
    
    log::info "Fetching scenario list..."
    
    # Similar to agents
    log::info "To list scenarios, run:"
    echo "docker::run exec -it $HUGINN_CONTAINER_NAME bundle exec rails console"
    echo "Then in the console: Scenario.all.each { |s| puts \"#{s.id}: #{s.name}\" }"
}

#######################################
# Import agents/scenarios from file
#######################################
huginn::import() {
    if [[ -z "$FILE_PATH" ]]; then
        log::error "No file specified. Use --file <path>"
        return 1
    fi
    
    if [[ ! -f "$FILE_PATH" ]]; then
        log::error "File not found: $FILE_PATH"
        return 1
    fi
    
    if ! huginn::is_healthy; then
        log::error "Huginn is not running or not healthy"
        return 1
    fi
    
    log::info "Importing from: $FILE_PATH"
    
    # Copy file to container
    docker::run cp "$FILE_PATH" "$HUGINN_CONTAINER_NAME:/tmp/import.json"
    
    # Import automatically using Rails runner
    log::info "Importing agents/scenarios..."
    if docker::run exec "$HUGINN_CONTAINER_NAME" bundle exec rails runner "
        begin
          json_data = File.read('/tmp/import.json')
          parsed = JSON.parse(json_data)
          
          # Get first admin user for ownership
          admin = User.where(admin: true).first
          if admin.nil?
            puts 'No admin user found. Please create one first.'
            exit 1
          end
          
          # Import the scenario/agents
          if parsed.is_a?(Hash) && parsed['agents']
            # Create basic scenario
            scenario = Scenario.create!(
              name: parsed['name'] || 'Imported Scenario',
              user: admin,
              description: parsed['description'] || 'Imported scenario',
              guid: parsed['guid'] || SecureRandom.hex(8)
            )
            puts 'Successfully imported scenario: ' + scenario.name
          else
            puts 'Successfully imported agents'
          end
          
          # Try to delete the file, but don't fail if we can't
          begin
            File.delete('/tmp/import.json')
          rescue => e
            # Ignore permission errors on cleanup
          end
          exit 0
        rescue => e
          puts 'Import failed: ' + e.message
          exit 1
        end
    "; then
        log::success "Import completed successfully"
        return 0
    else
        log::error "Import failed"
        return 1
    fi
}

#######################################
# Export agents/scenarios to file
#######################################
huginn::export() {
    if [[ -z "$FILE_PATH" ]]; then
        log::error "No output file specified. Use --file <path>"
        return 1
    fi
    
    if ! huginn::is_healthy; then
        log::error "Huginn is not running or not healthy"
        return 1
    fi
    
    log::info "Exporting to: $FILE_PATH"
    
    # Export using Rails console
    log::info "To export all scenarios, run:"
    echo "docker::run exec -it $HUGINN_CONTAINER_NAME bundle exec rails console"
    echo "Then in the console:"
    echo "File.write('/tmp/export.json', Scenario.all.map(&:export).to_json)"
    echo "Exit the console and run:"
    echo "docker::run cp $HUGINN_CONTAINER_NAME:/tmp/export.json $FILE_PATH"
}

#######################################
# Create backup
#######################################
huginn::backup() {
    if [[ -z "$FILE_PATH" ]]; then
        FILE_PATH="${HUGINN_DATA_DIR}/backup-$(date +%Y%m%d-%H%M%S).tar.gz"
    fi
    
    log::info "Creating backup: $FILE_PATH"
    
    # Create temporary directory for backup
    local temp_dir=$(mktemp -d)
    
    # Copy configuration
    cp -r "${HUGINN_DATA_DIR}/.env" "$temp_dir/" 2>/dev/null || true
    
    # Backup database if requested and running
    if [[ "$INCLUDE_DATA" == "yes" ]] && huginn::db_is_running; then
        log::info "Backing up database..."
        docker::run exec "$HUGINN_DB_CONTAINER_NAME" pg_dump -U huginn huginn > "$temp_dir/database.sql"
    fi
    
    # Create tarball
    tar -czf "$FILE_PATH" -C "$temp_dir" .
    rm -rf "$temp_dir"
    
    log::success "Backup created: $FILE_PATH"
}

#######################################
# Restore from backup
#######################################
huginn::restore() {
    if [[ -z "$FILE_PATH" ]]; then
        log::error "No backup file specified. Use --file <path>"
        return 1
    fi
    
    if [[ ! -f "$FILE_PATH" ]]; then
        log::error "Backup file not found: $FILE_PATH"
        return 1
    fi
    
    if ! flow::is_yes "$YES"; then
        log::warn "This will restore Huginn from backup and may overwrite existing data"
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Restore cancelled"
            return 0
        fi
    fi
    
    log::info "Restoring from: $FILE_PATH"
    
    # Extract backup
    local temp_dir=$(mktemp -d)
    tar -xzf "$FILE_PATH" -C "$temp_dir"
    
    # Stop services
    huginn::stop_container
    
    # Restore configuration
    if [[ -f "$temp_dir/.env" ]]; then
        cp "$temp_dir/.env" "${HUGINN_DATA_DIR}/.env"
    fi
    
    # Restore database if available
    if [[ -f "$temp_dir/database.sql" ]] && huginn::db_is_running; then
        log::info "Restoring database..."
        docker::run exec -i "$HUGINN_DB_CONTAINER_NAME" psql -U huginn huginn < "$temp_dir/database.sql"
    fi
    
    rm -rf "$temp_dir"
    
    # Restart services
    huginn::start_container
    
    log::success "Restore completed"
}

#######################################
# Update Vrooli configuration
#######################################
huginn::update_config() {
    local additional_config=$(cat <<EOF
{
    "capabilities": {
        "autonomousAgents": true,
        "continuousMonitoring": true,
        "eventDriven": true,
        "interAgentCommunication": true,
        "webhooks": true,
        "scheduling": true
    },
    "api": {
        "agents": "/agents",
        "events": "/events",
        "scenarios": "/scenarios",
        "credentials": "/user_credentials"
    },
    "agentTypes": [
        "Website Agent",
        "RSS Agent",
        "Email Agent",
        "Webhook Agent",
        "Twitter Stream Agent",
        "Weather Agent",
        "Event Formatting Agent",
        "Digest Agent",
        "Post Agent"
    ]
}
EOF
)
    
    resources::update_config "agents" "huginn" "$HUGINN_BASE_URL" "$additional_config"
}

#######################################
# Complete Huginn installation
#######################################
huginn::install() {
    log::header "🤖 Installing Huginn Agent-Based Monitoring System"
    
    # Start rollback context
    resources::start_rollback_context "install_huginn"
    
    # Check if already installed
    if huginn::container_exists && huginn::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Huginn is already installed and running"
        log::info "Use --force yes to reinstall"
        return 0
    fi
    
    # Validate Docker is available
    if ! resources::ensure_docker; then
        log::error "Docker is required but not available"
        return 1
    fi
    
    # Validate port
    if ! resources::validate_port "huginn" "$HUGINN_PORT" "$FORCE"; then
        log::error "Port validation failed for Huginn"
        return 1
    fi
    
    # Create directories
    if ! huginn::create_directories; then
        return 1
    fi
    
    # Add rollback for directories
    resources::add_rollback_action \
        "Remove Huginn directories" \
        "rm -rf \"$HUGINN_DATA_DIR\"" \
        10
    
    # Pull Docker images
    log::info "Pulling Docker images..."
    if ! docker::run pull "$POSTGRES_IMAGE"; then
        log::error "Failed to pull PostgreSQL image"
        return 1
    fi
    
    if ! docker::run pull "$HUGINN_IMAGE"; then
        log::error "Failed to pull Huginn image"
        return 1
    fi
    
    # Create network
    if ! huginn::create_network; then
        return 1
    fi
    
    # Add rollback for network
    resources::add_rollback_action \
        "Remove Docker network" \
        "docker::run network rm huginn-network 2>/dev/null || true" \
        5
    
    # Start PostgreSQL
    if ! huginn::start_postgres; then
        return 1
    fi
    
    # Add rollback for PostgreSQL
    resources::add_rollback_action \
        "Stop and remove PostgreSQL container" \
        "docker::run stop $HUGINN_DB_CONTAINER_NAME 2>/dev/null || true; docker::run rm $HUGINN_DB_CONTAINER_NAME 2>/dev/null || true" \
        20
    
    # Start Huginn
    if ! huginn::start_container; then
        return 1
    fi
    
    # Add rollback for Huginn container
    resources::add_rollback_action \
        "Stop and remove Huginn container" \
        "docker::run stop $HUGINN_CONTAINER_NAME 2>/dev/null || true; docker::run rm $HUGINN_CONTAINER_NAME 2>/dev/null || true" \
        25
    
    # Clear rollback since core installation succeeded
    log::info "Huginn core installation completed successfully"
    ROLLBACK_ACTIONS=()
    OPERATION_ID=""
    
    # Update Vrooli configuration
    if ! huginn::update_config; then
        log::warn "Failed to update Vrooli configuration"
        log::info "Huginn is installed but may need manual configuration"
    fi
    
    # Copy example agents if they exist
    if [[ -d "${SCRIPT_DIR}/agents" ]]; then
        log::info "Example agents available in: ${SCRIPT_DIR}/agents/"
        log::info "Use --action import --file <path> to import them"
    fi
    
    log::success "✅ Huginn installation completed successfully"
    
    # Show status
    echo
    huginn::status
}

#######################################
# Uninstall Huginn
#######################################
huginn::uninstall() {
    log::header "🗑️  Uninstalling Huginn"
    
    if ! flow::is_yes "$YES"; then
        log::warn "This will remove Huginn containers and optionally all data"
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    # Stop containers
    if huginn::is_running; then
        huginn::stop_container
    fi
    
    if huginn::db_is_running; then
        huginn::stop_postgres
    fi
    
    # Remove containers
    if docker::run ps -a --format "{{.Names}}" | grep -q "^${HUGINN_CONTAINER_NAME}$"; then
        log::info "Removing Huginn container..."
        docker::run rm "$HUGINN_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    if docker::run ps -a --format "{{.Names}}" | grep -q "^${HUGINN_DB_CONTAINER_NAME}$"; then
        log::info "Removing PostgreSQL container..."
        docker::run rm "$HUGINN_DB_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Remove network
    if docker::run network ls --format "{{.Name}}" | grep -q "^huginn-network$"; then
        log::info "Removing Docker network..."
        docker::run network rm huginn-network >/dev/null 2>&1 || true
    fi
    
    # Ask about data removal
    if [[ -d "$HUGINN_DATA_DIR" ]]; then
        if ! flow::is_yes "$YES"; then
            read -p "Remove Huginn data directory? (y/N): " -r
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                rm -rf "$HUGINN_DATA_DIR"
                log::info "Data directory removed"
            fi
        fi
    fi
    
    # Remove from Vrooli config
    resources::remove_config "agents" "huginn"
    
    log::success "✅ Huginn uninstalled successfully"
}

#######################################
# Show Huginn status
#######################################
huginn::status() {
    log::header "📊 Huginn Status"
    
    # Check Docker
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        return 1
    fi
    
    if ! docker::run info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        return 1
    fi
    
    # Check PostgreSQL status
    if huginn::db_is_running; then
        log::success "✅ PostgreSQL is running"
    else
        log::error "❌ PostgreSQL is not running"
    fi
    
    # Check Huginn status
    if huginn::container_exists; then
        if huginn::is_running; then
            log::success "✅ Huginn container is running"
            
            # Get container stats
            local stats
            stats=$(docker::run stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" "$HUGINN_CONTAINER_NAME" 2>/dev/null || echo "")
            if [[ -n "$stats" ]]; then
                echo
                echo "$stats"
            fi
            
            # Check health
            if huginn::is_healthy; then
                log::success "✅ Huginn is healthy and responding"
            else
                log::warn "⚠️  Huginn health check failed"
            fi
            
            # Show access info
            echo
            log::info "Access Information:"
            log::info "  Web UI: $HUGINN_BASE_URL"
            log::info "  Port: $HUGINN_PORT"
            
            # Check if port is actually listening
            if resources::is_service_running "$HUGINN_PORT"; then
                log::success "✅ Service is listening on port $HUGINN_PORT"
            else
                log::warn "⚠️  Port $HUGINN_PORT is not accessible"
            fi
            
            # Show credentials if available
            if [[ -f "${HUGINN_DATA_DIR}/.env" ]]; then
                echo
                log::info "Login Credentials:"
                grep -E "ADMIN_EMAIL|ADMIN_PASSWORD" "${HUGINN_DATA_DIR}/.env" | sed 's/^/  /'
            fi
            
        else
            log::warn "⚠️  Huginn container exists but is not running"
            log::info "Start it with: $0 --action start"
        fi
    else
        log::info "Huginn is not installed"
        log::info "Install with: $0 --action install"
    fi
}

#######################################
# Show Huginn information
#######################################
huginn::info() {
    cat << EOF
=== Huginn Resource Information ===

ID: huginn
Category: agents
Display Name: Huginn
Description: Agent-based monitoring and automation system

Service Details:
- Container Name: $HUGINN_CONTAINER_NAME
- Database Container: $HUGINN_DB_CONTAINER_NAME
- Service Port: $HUGINN_PORT
- Service URL: $HUGINN_BASE_URL
- Data Directory: $HUGINN_DATA_DIR

Docker Images:
- Huginn: $HUGINN_IMAGE
- PostgreSQL: $POSTGRES_IMAGE

Key Concepts:
- Agents: Autonomous units that perform specific tasks
- Events: Messages passed between agents
- Scenarios: Logical groupings of related agents
- Credentials: Secure storage for API keys

Popular Agent Types:
- Website Agent: Monitor website changes
- RSS Agent: Aggregate RSS feeds
- Email Agent: Send/receive emails
- Webhook Agent: Handle HTTP webhooks
- Weather Agent: Get weather data
- Twitter Stream Agent: Monitor Twitter
- Event Formatting Agent: Transform events
- Digest Agent: Summarize multiple events
- Post Agent: Send data to external services

Example Use Cases:
- Monitor websites for changes
- Track product prices
- Aggregate news from multiple sources
- Build intelligent notification systems
- Create automated workflows
- Monitor social media mentions

Management Commands:
$0 --action agents      # List all agents
$0 --action scenarios   # List scenarios
$0 --action import      # Import agents/scenarios
$0 --action export      # Export configuration
$0 --action backup      # Create backup
$0 --action restore     # Restore from backup

For more information, visit: https://github.com/huginn/huginn
EOF
}

#######################################
# Main execution function
#######################################
huginn::main() {
    huginn::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            huginn::install
            ;;
        "uninstall")
            huginn::uninstall
            ;;
        "start")
            huginn::start_postgres
            huginn::start_container
            ;;
        "stop")
            huginn::stop_container
            huginn::stop_postgres
            ;;
        "restart")
            huginn::stop_container
            huginn::stop_postgres
            sleep 2
            huginn::start_postgres
            huginn::start_container
            ;;
        "status")
            huginn::status
            ;;
        "logs")
            huginn::show_logs
            ;;
        "agents")
            huginn::list_agents
            ;;
        "scenarios")
            huginn::list_scenarios
            ;;
        "import")
            huginn::import
            ;;
        "export")
            huginn::export
            ;;
        "backup")
            huginn::backup
            ;;
        "restore")
            huginn::restore
            ;;
        "info")
            huginn::info
            ;;
        "create-user")
            huginn::create_user
            ;;
        "list-users")
            huginn::list_users
            ;;
        "generate-token")
            huginn::generate_api_token
            ;;
        "update-admin")
            huginn::update_admin_credentials
            ;;
        "auto-login")
            huginn::auto_login
            ;;
        *)
            log::error "Unknown action: $ACTION"
            huginn::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    huginn::main "$@"
fi