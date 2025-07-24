#!/usr/bin/env bash
set -euo pipefail

# n8n Workflow Automation Platform Setup and Management
# This script handles installation, configuration, and management of n8n using Docker

DESCRIPTION="Install and manage n8n workflow automation platform using Docker"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# n8n configuration
readonly N8N_PORT="${N8N_CUSTOM_PORT:-$(resources::get_default_port "n8n")}"
readonly N8N_BASE_URL="http://localhost:${N8N_PORT}"
readonly N8N_CONTAINER_NAME="n8n"
readonly N8N_DATA_DIR="${HOME}/.n8n"
readonly N8N_IMAGE="docker.n8n.io/n8nio/n8n:latest"

# Custom image configuration
readonly N8N_CUSTOM_IMAGE="${N8N_CUSTOM_IMAGE:-n8n-vrooli:latest}"
readonly N8N_USE_CUSTOM_IMAGE="${N8N_USE_CUSTOM_IMAGE:-no}"

# Database configuration
readonly N8N_DB_TYPE="${N8N_DB_TYPE:-sqlite}"
readonly N8N_DB_CONTAINER_NAME="n8n-postgres"
readonly N8N_DB_PORT="5432"
readonly N8N_DB_PASSWORD="n8n-secure-password-$(date +%s | sha256sum | head -c 16)"

#######################################
# Parse command line arguments
#######################################
n8n::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|reset-password|logs|info|execute|api-setup|save-api-key" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if n8n appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "webhook-url" \
        --desc "External webhook URL (default: auto-detect)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "workflow-id" \
        --desc "Workflow ID for execution" \
        --type "value" \
        --default ""
    
    args::register \
        --name "api-key" \
        --desc "n8n API key to save" \
        --type "value" \
        --default ""
    
    args::register \
        --name "data" \
        --desc "JSON data to pass to workflow (for webhook workflows)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "basic-auth" \
        --desc "Enable basic authentication" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    args::register \
        --name "username" \
        --desc "Basic auth username (default: admin)" \
        --type "value" \
        --default "admin"
    
    args::register \
        --name "password" \
        --desc "Basic auth password (default: auto-generated)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "database" \
        --desc "Database type (sqlite or postgres)" \
        --type "value" \
        --options "sqlite|postgres" \
        --default "sqlite"
    
    args::register \
        --name "tunnel" \
        --desc "Enable tunnel for webhook testing (dev only)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "build-image" \
        --desc "Build custom n8n image with host access" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    if args::is_asking_for_help "$@"; then
        n8n::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export WEBHOOK_URL=$(args::get "webhook-url")
    export WORKFLOW_ID=$(args::get "workflow-id")
    export API_KEY=$(args::get "api-key")
    export WORKFLOW_DATA=$(args::get "data")
    export BASIC_AUTH=$(args::get "basic-auth")
    export AUTH_USERNAME=$(args::get "username")
    export AUTH_PASSWORD=$(args::get "password")
    export DATABASE_TYPE=$(args::get "database")
    export TUNNEL_ENABLED=$(args::get "tunnel")
    export BUILD_IMAGE=$(args::get "build-image")
}

#######################################
# Display usage information
#######################################
n8n::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                              # Install n8n with basic auth"
    echo "  $0 --action install --basic-auth no              # Install without authentication"
    echo "  $0 --action install --username admin --password secret  # Custom credentials"
    echo "  $0 --action install --database postgres          # Use PostgreSQL instead of SQLite"
    echo "  $0 --action install --tunnel yes                 # Enable webhook tunnel (dev only)"
    echo "  $0 --action install --build-image yes            # Build custom image with host access"
    echo "  $0 --action status                               # Check n8n status"
    echo "  $0 --action logs                                 # View n8n logs"
    echo "  $0 --action reset-password                       # Reset admin password"
    echo "  $0 --action uninstall                           # Remove n8n"
}

#######################################
# Check if Docker is installed
# Returns: 0 if installed, 1 otherwise
#######################################
n8n::check_docker() {
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        log::info "Please install Docker first: https://docs.docker.com/get-docker/"
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        log::info "Start Docker with: sudo systemctl start docker"
        return 1
    fi
    
    # Check if user has permissions
    if ! docker ps >/dev/null 2>&1; then
        log::error "Current user doesn't have Docker permissions"
        log::info "Add user to docker group: sudo usermod -aG docker $USER"
        log::info "Then log out and back in for changes to take effect"
        return 1
    fi
    
    return 0
}

#######################################
# Check if n8n container exists
# Returns: 0 if exists, 1 otherwise
#######################################
n8n::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${N8N_CONTAINER_NAME}$"
}

#######################################
# Check if n8n is running
# Returns: 0 if running, 1 otherwise
#######################################
n8n::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${N8N_CONTAINER_NAME}$"
}

#######################################
# Check if n8n API is responsive
# Returns: 0 if responsive, 1 otherwise
#######################################
n8n::is_healthy() {
    # n8n uses /healthz endpoint for health checks
    if system::is_command "curl"; then
        # Try multiple times as n8n takes time to fully initialize
        local attempts=0
        while [ $attempts -lt 5 ]; do
            if curl -f -s --max-time 5 "$N8N_BASE_URL/healthz" >/dev/null 2>&1; then
                return 0
            fi
            attempts=$((attempts + 1))
            sleep 2
        done
    fi
    return 1
}

#######################################
# Generate secure random password
#######################################
n8n::generate_password() {
    # Use multiple sources for randomness
    if system::is_command "openssl"; then
        openssl rand -base64 32 | tr -d "=+/" | cut -c1-16
    elif [[ -r /dev/urandom ]]; then
        tr -dc 'A-Za-z0-9!@#$%^&*' < /dev/urandom | head -c 16
    else
        # Fallback to timestamp-based password
        echo "n8n$(date +%s)$RANDOM" | sha256sum | cut -c1-16
    fi
}

#######################################
# Build custom n8n Docker image
#######################################
n8n::build_custom_image() {
    log::info "Building custom n8n image with host access..."
    
    # Ensure files exist
    if [[ ! -f "$SCRIPT_DIR/Dockerfile" ]] || [[ ! -f "$SCRIPT_DIR/docker-entrypoint.sh" ]]; then
        log::error "Required files missing in $SCRIPT_DIR"
        log::info "Please ensure Dockerfile and docker-entrypoint.sh exist"
        return 1
    fi
    
    # Build image
    if docker build -t "$N8N_CUSTOM_IMAGE" "$SCRIPT_DIR"; then
        log::success "Custom n8n image built successfully"
        return 0
    else
        log::error "Failed to build custom image"
        return 1
    fi
}

#######################################
# Create n8n data directory
#######################################
n8n::create_directories() {
    log::info "Creating n8n data directory..."
    
    mkdir -p "$N8N_DATA_DIR" || {
        log::error "Failed to create n8n data directory"
        return 1
    }
    
    # Add rollback action
    resources::add_rollback_action \
        "Remove n8n data directory" \
        "rm -rf $N8N_DATA_DIR 2>/dev/null || true" \
        10
    
    log::success "n8n directories created"
    return 0
}

#######################################
# Create Docker network for n8n
#######################################
n8n::create_network() {
    local network_name="n8n-network"
    
    if ! docker network ls | grep -q "$network_name"; then
        log::info "Creating Docker network for n8n..."
        
        if docker network create "$network_name" >/dev/null 2>&1; then
            log::success "Docker network created"
            
            # Add rollback action
            resources::add_rollback_action \
                "Remove Docker network" \
                "docker network rm $network_name 2>/dev/null || true" \
                5
        else
            log::warn "Failed to create Docker network (may already exist)"
        fi
    fi
}

#######################################
# Start PostgreSQL container for n8n
#######################################
n8n::start_postgres() {
    if [[ "$DATABASE_TYPE" != "postgres" ]]; then
        return 0
    fi
    
    log::info "Starting PostgreSQL container for n8n..."
    
    # Check if postgres container already exists
    if docker ps -a --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
        if docker ps --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
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
        --network "n8n-network" \
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
# Build n8n Docker command
#######################################
n8n::build_docker_command() {
    local webhook_url="$1"
    local auth_password="$2"
    
    local docker_cmd="docker run -d"
    docker_cmd+=" --name $N8N_CONTAINER_NAME"
    docker_cmd+=" --network n8n-network"
    docker_cmd+=" -p ${N8N_PORT}:5678"
    docker_cmd+=" -v ${N8N_DATA_DIR}:/home/node/.n8n"
    docker_cmd+=" --restart unless-stopped"
    
    # Add volume mounts for host access
    # Mount only if directories exist on host
    if [[ -d /usr/bin ]]; then
        docker_cmd+=" -v /usr/bin:/host/usr/bin:ro"
    fi
    if [[ -d /bin ]]; then
        docker_cmd+=" -v /bin:/host/bin:ro"
    fi
    if [[ -d /usr/local/bin ]]; then
        docker_cmd+=" -v /usr/local/bin:/host/usr/local/bin:ro"
    fi
    
    # User home and workspace
    docker_cmd+=" -v $HOME:/host/home:rw"
    docker_cmd+=" -v $HOME/Vrooli:/workspace:rw"
    
    # Docker socket for container control
    if [[ -S /var/run/docker.sock ]]; then
        docker_cmd+=" -v /var/run/docker.sock:/var/run/docker.sock:rw"
    fi
    
    # Environment variables
    docker_cmd+=" -e GENERIC_TIMEZONE=$(timedatectl show -p Timezone --value 2>/dev/null || echo 'UTC')"
    docker_cmd+=" -e TZ=$(timedatectl show -p Timezone --value 2>/dev/null || echo 'UTC')"
    
    # Webhook configuration
    if [[ -n "$webhook_url" ]]; then
        docker_cmd+=" -e WEBHOOK_URL=$webhook_url"
        docker_cmd+=" -e N8N_PROTOCOL=https"
        docker_cmd+=" -e N8N_HOST=$(echo "$webhook_url" | sed 's|https://||' | sed 's|/.*||')"
    fi
    
    # Basic authentication
    if [[ "$BASIC_AUTH" == "yes" ]]; then
        docker_cmd+=" -e N8N_BASIC_AUTH_ACTIVE=true"
        docker_cmd+=" -e N8N_BASIC_AUTH_USER=$AUTH_USERNAME"
        docker_cmd+=" -e N8N_BASIC_AUTH_PASSWORD=$auth_password"
    fi
    
    # Database configuration
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        docker_cmd+=" -e DB_TYPE=postgresdb"
        docker_cmd+=" -e DB_POSTGRESDB_HOST=$N8N_DB_CONTAINER_NAME"
        docker_cmd+=" -e DB_POSTGRESDB_PORT=5432"
        docker_cmd+=" -e DB_POSTGRESDB_DATABASE=n8n"
        docker_cmd+=" -e DB_POSTGRESDB_USER=n8n"
        docker_cmd+=" -e DB_POSTGRESDB_PASSWORD=$N8N_DB_PASSWORD"
    else
        docker_cmd+=" -e DB_TYPE=sqlite"
        docker_cmd+=" -e DB_SQLITE_VACUUM_ON_STARTUP=true"
    fi
    
    # Additional settings
    docker_cmd+=" -e N8N_DIAGNOSTICS_ENABLED=false"
    docker_cmd+=" -e N8N_TEMPLATES_ENABLED=true"
    docker_cmd+=" -e N8N_PERSONALIZATION_ENABLED=false"
    docker_cmd+=" -e N8N_RUNNERS_ENABLED=true"
    docker_cmd+=" -e N8N_ENFORCE_SETTINGS_FILE_PERMISSIONS=true"
    docker_cmd+=" -e EXECUTIONS_DATA_SAVE_ON_ERROR=all"
    docker_cmd+=" -e EXECUTIONS_DATA_SAVE_ON_SUCCESS=all"
    docker_cmd+=" -e EXECUTIONS_DATA_SAVE_ON_PROGRESS=true"
    docker_cmd+=" -e EXECUTIONS_DATA_SAVE_MANUAL_EXECUTIONS=true"
    
    # Use custom image if available
    local image_to_use="$N8N_IMAGE"
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${N8N_CUSTOM_IMAGE}$"; then
        image_to_use="$N8N_CUSTOM_IMAGE"
        log::info "Using custom n8n image: $N8N_CUSTOM_IMAGE"
    fi
    
    # Image and command
    docker_cmd+=" $image_to_use"
    
    # Add tunnel flag if enabled (development only)
    if [[ "$TUNNEL_ENABLED" == "yes" ]]; then
        docker_cmd+=" n8n start --tunnel"
        log::warn "⚠️  Tunnel mode enabled - for development only!"
    fi
    
    echo "$docker_cmd"
}

#######################################
# Start n8n container
#######################################
n8n::start_container() {
    local webhook_url="$1"
    local auth_password="$2"
    
    log::info "Starting n8n container..."
    
    # Build and execute Docker command
    local docker_cmd
    docker_cmd=$(n8n::build_docker_command "$webhook_url" "$auth_password")
    
    if eval "$docker_cmd" >/dev/null 2>&1; then
        log::success "n8n container started"
        
        # Add rollback action
        resources::add_rollback_action \
            "Stop and remove n8n container" \
            "docker stop $N8N_CONTAINER_NAME 2>/dev/null; docker rm $N8N_CONTAINER_NAME 2>/dev/null || true" \
            25
        
        return 0
    else
        log::error "Failed to start n8n container"
        return 1
    fi
}

#######################################
# Update Vrooli configuration
#######################################
n8n::update_config() {
    # Create JSON with proper escaping
    local additional_config
    additional_config=$(cat <<EOF
{
    "features": {
        "workflows": true,
        "webhooks": true,
        "api": true,
        "templates": true
    },
    "ui": {
        "endpoint": "/",
        "port": "$N8N_PORT"
    },
    "webhook": {
        "url": "$WEBHOOK_URL"
    },
    "api": {
        "version": "v1",
        "restEndpoint": "/rest",
        "webhookEndpoint": "/webhook",
        "webhookTestEndpoint": "/webhook-test"
    },
    "container": {
        "name": "$N8N_CONTAINER_NAME",
        "image": "$N8N_IMAGE"
    }
}
EOF
)
    
    resources::update_config "automation" "n8n" "$N8N_BASE_URL" "$additional_config"
}

#######################################
# Complete n8n installation
#######################################
n8n::install() {
    log::header "🤖 Installing n8n Workflow Automation (Docker)"
    
    # Start rollback context
    resources::start_rollback_context "install_n8n_docker"
    
    # Check if already installed
    if n8n::container_exists && n8n::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "n8n is already installed and running"
        log::info "Use --force yes to reinstall"
        return 0
    fi
    
    # Check Docker
    if ! n8n::check_docker; then
        return 1
    fi
    
    # Validate port assignment
    if ! resources::validate_port "n8n" "$N8N_PORT"; then
        log::error "Port validation failed for n8n"
        log::info "You can set a custom port with: export N8N_CUSTOM_PORT=<port>"
        return 1
    fi
    
    # Build custom image if requested
    if [[ "$BUILD_IMAGE" == "yes" ]]; then
        if ! n8n::build_custom_image; then
            resources::handle_error \
                "Failed to build custom n8n image" \
                "system" \
                "Check Docker logs and Dockerfile"
            return 1
        fi
    fi
    
    # Generate password if needed
    if [[ "$BASIC_AUTH" == "yes" && -z "$AUTH_PASSWORD" ]]; then
        AUTH_PASSWORD=$(n8n::generate_password)
        log::info "Generated password for user '$AUTH_USERNAME'"
    fi
    
    # Detect webhook URL if not provided
    if [[ -z "$WEBHOOK_URL" ]]; then
        # Try to detect public IP or hostname
        if system::is_command "curl"; then
            local public_ip
            public_ip=$(curl -s -4 https://api.ipify.org 2>/dev/null || echo "")
            if [[ -n "$public_ip" ]]; then
                WEBHOOK_URL="http://$public_ip:$N8N_PORT"
                log::info "Auto-detected webhook URL: $WEBHOOK_URL"
            fi
        fi
        
        # Fallback to localhost
        if [[ -z "$WEBHOOK_URL" ]]; then
            WEBHOOK_URL="$N8N_BASE_URL"
            log::warn "Using localhost for webhook URL - external webhooks may not work"
        fi
    fi
    
    # Create directories
    if ! n8n::create_directories; then
        resources::handle_error \
            "Failed to create n8n directories" \
            "system" \
            "Check directory permissions"
        return 1
    fi
    
    # Create Docker network
    n8n::create_network
    
    # Start PostgreSQL if needed
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        if ! n8n::start_postgres; then
            resources::handle_error \
                "Failed to start PostgreSQL" \
                "system" \
                "Check Docker logs"
            return 1
        fi
    fi
    
    # Start n8n container
    if ! n8n::start_container "$WEBHOOK_URL" "$AUTH_PASSWORD"; then
        resources::handle_error \
            "Failed to start n8n container" \
            "system" \
            "Check Docker logs: docker logs $N8N_CONTAINER_NAME"
        return 1
    fi
    
    # Wait for service to be ready
    log::info "Waiting for n8n to start..."
    
    # Wait for container to be running and port to be available
    local wait_time=0
    local max_wait=60
    while [ $wait_time -lt $max_wait ]; do
        if n8n::is_running && ss -tlnp 2>/dev/null | grep -q ":$N8N_PORT"; then
            log::info "Container is running and port is bound"
            break
        fi
        sleep 2
        wait_time=$((wait_time + 2))
        echo -n "."
    done
    echo
    
    if [ $wait_time -ge $max_wait ]; then
        resources::handle_error \
            "n8n failed to start within timeout" \
            "system" \
            "Check container logs for errors"
        return 1
    fi
    
    # Give n8n time to initialize and run migrations
    log::info "Waiting for n8n to complete initialization..."
    sleep 10
    
    if n8n::is_healthy; then
            log::success "✅ n8n is running and healthy on port $N8N_PORT"
            
            # Display access information
            echo
            log::header "🌐 n8n Access Information"
            log::info "URL: $N8N_BASE_URL"
            log::info "Webhook URL: $WEBHOOK_URL"
            
            if [[ "$BASIC_AUTH" == "yes" ]]; then
                log::info "Username: $AUTH_USERNAME"
                if [[ -n "$AUTH_PASSWORD" ]]; then
                    log::warn "Password: $AUTH_PASSWORD (save this password!)"
                fi
            else
                log::warn "⚠️  No authentication enabled - n8n is publicly accessible"
            fi
            
            if [[ "$DATABASE_TYPE" == "postgres" ]]; then
                log::info "Database: PostgreSQL (persistent)"
            else
                log::info "Database: SQLite (file-based)"
            fi
            
            # Update Vrooli configuration
            if ! n8n::update_config; then
                log::warn "Failed to update Vrooli configuration"
                log::info "n8n is installed but may need manual configuration in Vrooli"
            fi
            
            # Clear rollback context on success
            ROLLBACK_ACTIONS=()
            OPERATION_ID=""
            
            echo
            log::header "🎯 Next Steps"
            log::info "1. Access n8n at: $N8N_BASE_URL"
            log::info "2. Create your first workflow"
            log::info "3. Configure webhook integrations using: $WEBHOOK_URL"
            log::info "4. Check the docs: https://docs.n8n.io"
            
            return 0
        else
            log::warn "n8n started but health check failed"
            log::info "Check logs: docker logs $N8N_CONTAINER_NAME"
            return 0
        fi
}

#######################################
# Stop n8n
#######################################
n8n::stop() {
    if ! n8n::is_running; then
        log::info "n8n is not running"
        return 0
    fi
    
    log::info "Stopping n8n..."
    
    # Stop n8n container
    if docker stop "$N8N_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "n8n stopped"
    else
        log::error "Failed to stop n8n"
        return 1
    fi
    
    # Stop PostgreSQL if used
    if [[ "$DATABASE_TYPE" == "postgres" ]] && docker ps --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
        log::info "Stopping PostgreSQL..."
        docker stop "$N8N_DB_CONTAINER_NAME" >/dev/null 2>&1
    fi
}

#######################################
# Start n8n
#######################################
n8n::start() {
    if n8n::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "n8n is already running on port $N8N_PORT"
        return 0
    fi
    
    log::info "Starting n8n..."
    
    # Check if container exists
    if ! n8n::container_exists; then
        log::error "n8n container does not exist. Run install first."
        return 1
    fi
    
    # Start PostgreSQL first if needed
    if docker ps -a --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
        if ! docker ps --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
            log::info "Starting PostgreSQL..."
            docker start "$N8N_DB_CONTAINER_NAME" >/dev/null 2>&1
            sleep 3
        fi
    fi
    
    # Start n8n container
    if docker start "$N8N_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "n8n started"
        
        # Wait for service to be ready
        if resources::wait_for_service "n8n" "$N8N_PORT" 30; then
            log::success "✅ n8n is running on port $N8N_PORT"
            log::info "Access n8n at: $N8N_BASE_URL"
        else
            log::warn "n8n started but may not be fully ready yet"
        fi
    else
        log::error "Failed to start n8n"
        return 1
    fi
}

#######################################
# Restart n8n
#######################################
n8n::restart() {
    log::info "Restarting n8n..."
    n8n::stop
    sleep 2
    n8n::start
}

#######################################
# Show n8n logs
#######################################
n8n::logs() {
    if ! n8n::container_exists; then
        log::error "n8n container does not exist"
        return 1
    fi
    
    log::info "Showing n8n logs (Ctrl+C to exit)..."
    docker logs -f "$N8N_CONTAINER_NAME"
}

#######################################
# Reset admin password
#######################################
n8n::reset_password() {
    log::header "🔐 Reset n8n Password"
    
    if ! n8n::container_exists; then
        log::error "n8n is not installed"
        return 1
    fi
    
    # Generate new password
    local new_password
    new_password=$(n8n::generate_password)
    
    log::info "Generating new password..."
    
    # Get current environment variables
    local current_env
    current_env=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null)
    
    # Update password in environment
    local new_env=()
    local username="admin"
    while IFS= read -r env_var; do
        if [[ "$env_var" =~ ^N8N_BASIC_AUTH_PASSWORD= ]]; then
            new_env+=("-e" "N8N_BASIC_AUTH_PASSWORD=$new_password")
        elif [[ "$env_var" =~ ^N8N_BASIC_AUTH_USER= ]]; then
            username=$(echo "$env_var" | cut -d'=' -f2)
            new_env+=("-e" "$env_var")
        elif [[ -n "$env_var" ]]; then
            new_env+=("-e" "$env_var")
        fi
    done <<< "$current_env"
    
    # Ensure basic auth is enabled
    if ! echo "$current_env" | grep -q "N8N_BASIC_AUTH_ACTIVE=true"; then
        new_env+=("-e" "N8N_BASIC_AUTH_ACTIVE=true")
    fi
    
    # Get container configuration
    local image
    image=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{.Config.Image}}')
    
    local volumes
    volumes=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{range .Mounts}}{{if eq .Type "bind"}}-v {{.Source}}:{{.Destination}} {{end}}{{end}}')
    
    local ports
    ports=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{range $p, $conf := .NetworkSettings.Ports}}{{if $conf}}-p {{(index $conf 0).HostPort}}:{{$p}} {{end}}{{end}}' | sed 's|/tcp||g')
    
    # Stop and remove old container
    log::info "Stopping n8n container..."
    docker stop "$N8N_CONTAINER_NAME" >/dev/null 2>&1
    docker rm "$N8N_CONTAINER_NAME" >/dev/null 2>&1
    
    # Start new container with updated password
    log::info "Starting n8n with new password..."
    if docker run -d \
        --name "$N8N_CONTAINER_NAME" \
        --network "n8n-network" \
        $ports \
        $volumes \
        --restart unless-stopped \
        "${new_env[@]}" \
        "$image" >/dev/null 2>&1; then
        
        log::success "Password updated successfully"
        
        # Wait for service to start
        if resources::wait_for_service "n8n" "$N8N_PORT" 30; then
            echo
            log::header "🔑 New n8n Credentials"
            log::info "Username: $username"
            log::warn "Password: $new_password (save this password!)"
            log::info "URL: $N8N_BASE_URL"
        fi
        
        return 0
    else
        log::error "Failed to restart n8n with new password"
        return 1
    fi
}

#######################################
# Uninstall n8n
#######################################
n8n::uninstall() {
    log::header "🗑️  Uninstalling n8n"
    
    if ! flow::is_yes "$YES"; then
        log::warn "This will remove n8n and all workflow data"
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    # Stop and remove n8n container
    if n8n::container_exists; then
        log::info "Removing n8n container..."
        docker stop "$N8N_CONTAINER_NAME" 2>/dev/null || true
        docker rm "$N8N_CONTAINER_NAME" 2>/dev/null || true
        log::success "n8n container removed"
    fi
    
    # Stop and remove PostgreSQL container
    if docker ps -a --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
        log::info "Removing PostgreSQL container..."
        docker stop "$N8N_DB_CONTAINER_NAME" 2>/dev/null || true
        docker rm "$N8N_DB_CONTAINER_NAME" 2>/dev/null || true
        log::success "PostgreSQL container removed"
    fi
    
    # Remove Docker network
    if docker network ls | grep -q "n8n-network"; then
        log::info "Removing Docker network..."
        docker network rm "n8n-network" 2>/dev/null || true
    fi
    
    # Backup data before removal
    if [[ -d "$N8N_DATA_DIR" ]]; then
        local backup_dir="$HOME/n8n-backup-$(date +%Y%m%d-%H%M%S)"
        log::info "Backing up n8n data to: $backup_dir"
        cp -r "$N8N_DATA_DIR" "$backup_dir" 2>/dev/null || true
    fi
    
    # Remove data directory
    read -p "Remove n8n data directory? (y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$N8N_DATA_DIR" 2>/dev/null || true
        log::info "Data directory removed"
    fi
    
    # Remove from Vrooli config
    resources::remove_config "automation" "n8n"
    
    log::success "✅ n8n uninstalled successfully"
}

#######################################
# Show n8n status
#######################################
n8n::status() {
    log::header "📊 n8n Status"
    
    # Check Docker
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        return 1
    fi
    
    # Check container status
    if n8n::container_exists; then
        if n8n::is_running; then
            log::success "✅ n8n container is running"
            
            # Get container stats
            local stats
            stats=$(docker stats "$N8N_CONTAINER_NAME" --no-stream --format "CPU: {{.CPUPerc}} | Memory: {{.MemUsage}}" 2>/dev/null || echo "")
            if [[ -n "$stats" ]]; then
                log::info "Resource usage: $stats"
            fi
            
            # Check health
            if n8n::is_healthy; then
                log::success "✅ n8n API is healthy"
            else
                log::warn "⚠️  n8n API health check failed"
            fi
            
            # Additional details
            echo
            log::info "n8n Details:"
            log::info "  Web UI: $N8N_BASE_URL"
            log::info "  Container: $N8N_CONTAINER_NAME"
            
            # Get environment info
            local auth_active
            auth_active=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{range .Config.Env}}{{if eq (index (split . "=") 0) "N8N_BASIC_AUTH_ACTIVE"}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
            
            if [[ "$auth_active" == "true" ]]; then
                local auth_user
                auth_user=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{range .Config.Env}}{{if eq (index (split . "=") 0) "N8N_BASIC_AUTH_USER"}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
                log::info "  Authentication: Enabled (user: ${auth_user:-unknown})"
            else
                log::warn "  Authentication: Disabled"
            fi
            
            # Database info
            if docker ps --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
                log::info "  Database: PostgreSQL (running)"
            else
                local db_type
                db_type=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{range .Config.Env}}{{if eq (index (split . "=") 0) "DB_TYPE"}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
                log::info "  Database: ${db_type:-sqlite}"
            fi
            
            # Show logs command
            echo
            log::info "View logs: $0 --action logs"
        else
            log::warn "⚠️  n8n container exists but is not running"
            log::info "Start with: $0 --action start"
        fi
    else
        log::error "❌ n8n is not installed"
        log::info "Install with: $0 --action install"
    fi
    
    # Check for PostgreSQL
    if docker ps -a --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
        echo
        if docker ps --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
            log::info "PostgreSQL Status: ✅ Running"
        else
            log::warn "PostgreSQL Status: ⚠️  Stopped"
        fi
    fi
}

#######################################
# Show n8n information
#######################################
n8n::info() {
    cat << EOF
=== n8n Resource Information ===

ID: n8n
Category: automation
Display Name: n8n
Description: Workflow automation platform

Service Details:
- Container Name: $N8N_CONTAINER_NAME
- Service Port: $N8N_PORT
- Service URL: $N8N_BASE_URL
- Webhook URL: ${WEBHOOK_URL:-$N8N_BASE_URL}
- Docker Image: $N8N_IMAGE
- Data Directory: $N8N_DATA_DIR

Endpoints:
- Health Check: $N8N_BASE_URL/healthz
- Web UI: $N8N_BASE_URL
- REST API: $N8N_BASE_URL/rest
- Webhooks: ${WEBHOOK_URL:-$N8N_BASE_URL}/webhook
- Webhook Test: ${WEBHOOK_URL:-$N8N_BASE_URL}/webhook-test

Configuration:
- Authentication: ${BASIC_AUTH:-yes}
- Database: ${DATABASE_TYPE:-sqlite}
- Tunnel Mode: ${TUNNEL_ENABLED:-no}

n8n Features:
- Visual workflow builder
- 400+ integrations
- Webhook triggers
- Schedule triggers
- Custom code nodes
- Error handling
- Version control
- Team collaboration
- API access
- Custom node creation

Example Usage:
# Access the web UI
Open $N8N_BASE_URL in your browser

# Create a webhook workflow
1. Create a new workflow
2. Add a Webhook node as trigger
3. Use the webhook URL: ${WEBHOOK_URL:-$N8N_BASE_URL}/webhook/<path>

# Test a webhook
curl -X POST ${WEBHOOK_URL:-$N8N_BASE_URL}/webhook-test/your-path \\
  -H "Content-Type: application/json" \\
  -d '{"message": "Hello n8n!"}'

# Access REST API (requires authentication)
curl -u $AUTH_USERNAME:<password> \\
  $N8N_BASE_URL/rest/workflows

For more information, visit: https://docs.n8n.io
EOF
}

#######################################
# Execute workflow via API
#######################################
n8n::execute() {
    local workflow_id="${WORKFLOW_ID:-}"
    
    log::header "🚀 n8n Workflow Execution"
    
    # Check if workflow ID is provided
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo ""
        echo "Usage: $0 --action execute --workflow-id YOUR_WORKFLOW_ID"
        echo ""
        echo "To find workflow IDs:"
        echo "  docker exec $N8N_CONTAINER_NAME n8n list:workflow"
        return 1
    fi
    
    # Check for API key - first from env, then from config
    local api_key="${N8N_API_KEY:-}"
    
    # If not in env, try to load from resources config
    if [[ -z "$api_key" ]]; then
        local config_file="${HOME}/.vrooli/resources.local.json"
        if [[ -f "$config_file" ]]; then
            api_key=$(jq -r '.services.automation.n8n.apiKey // empty' "$config_file" 2>/dev/null)
        fi
    fi
    
    if [[ -z "$api_key" ]]; then
        log::warn "No API key found"
        echo ""
        n8n::show_api_setup_instructions
        return 1
    fi
    
    # First, get workflow details to check if it has a webhook
    log::info "Fetching workflow details: $workflow_id"
    
    local workflow_details
    workflow_details=$(curl -s -H "X-N8N-API-KEY: $api_key" \
        "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}" 2>&1)
    
    # Check if workflow exists
    if echo "$workflow_details" | grep -q '"message".*not found'; then
        log::error "Workflow not found"
        echo ""
        echo "Available workflows:"
        docker exec "$N8N_CONTAINER_NAME" n8n list:workflow
        return 1
    fi
    
    # Check if workflow has a webhook node
    local webhook_path
    webhook_path=$(echo "$workflow_details" | jq -r '.nodes[]? | select(.type == "n8n-nodes-base.webhook") | .parameters.path // empty' 2>/dev/null | head -1)
    
    if [[ -n "$webhook_path" ]]; then
        # Workflow has webhook - check if it's active
        local is_active
        is_active=$(echo "$workflow_details" | jq -r '.active' 2>/dev/null)
        
        if [[ "$is_active" != "true" ]]; then
            log::warn "Webhook workflow is not active. Activating it..."
            
            # Activate the workflow
            local activate_response
            activate_response=$(curl -s -X POST \
                -H "X-N8N-API-KEY: $api_key" \
                -H "Content-Type: application/json" \
                -d '{}' \
                "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}/activate" 2>&1)
            
            if echo "$activate_response" | grep -q '"active":true'; then
                log::success "✅ Workflow activated"
            else
                log::error "Failed to activate workflow"
                echo "Response: $activate_response"
                return 1
            fi
        fi
        
        # Execute via webhook
        log::info "Executing webhook workflow at path: /$webhook_path"
        
        # Get webhook method from node parameters
        local webhook_method
        webhook_method=$(echo "$workflow_details" | jq -r '.nodes[]? | select(.type == "n8n-nodes-base.webhook") | .parameters.method // empty' 2>/dev/null | head -1)
        webhook_method="${webhook_method:-GET}"
        
        local data="${WORKFLOW_DATA:-{}}"
        local response
        local http_code
        
        if [[ "$webhook_method" == "GET" ]] || [[ -z "$webhook_method" ]]; then
            # For GET requests, append data as query parameters if provided
            local query_string=""
            if [[ "$data" != "{}" ]]; then
                query_string="?data=$(echo "$data" | jq -c . | sed 's/ /%20/g')"
            fi
            response=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" -X GET \
                "${N8N_BASE_URL}/webhook/${webhook_path}${query_string}" 2>&1)
        else
            # For POST/PUT/etc, send data in body
            response=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" -X "$webhook_method" \
                -H "Content-Type: application/json" \
                -d "$data" \
                "${N8N_BASE_URL}/webhook/${webhook_path}" 2>&1)
        fi
        
        # Extract HTTP code
        http_code=$(echo "$response" | grep "__HTTP_CODE__:" | cut -d':' -f2)
        response=$(echo "$response" | grep -v "__HTTP_CODE__:")
        
        if [[ "$http_code" == "200" ]] || [[ "$http_code" == "201" ]]; then
            log::success "✅ Webhook workflow executed successfully"
            if [[ -n "$response" ]] && [[ "$response" != "{}" ]] && [[ "$response" != "null" ]]; then
                echo ""
                echo "Response:"
                # Try to parse as JSON, fallback to raw output
                if echo "$response" | jq empty 2>/dev/null; then
                    echo "$response" | jq '.' 2>/dev/null
                else
                    echo "$response"
                fi
            fi
        else
            log::error "Webhook execution failed (HTTP $http_code)"
            echo "Response: $response"
            return 1
        fi
    else
        # No webhook - workflow has manual trigger only
        log::error "This workflow uses a Manual Trigger and cannot be executed via API"
        echo ""
        echo "n8n's public API does not support executing manual trigger workflows."
        echo ""
        echo "Options:"
        echo "  1. Replace the Manual Trigger with a Webhook node in your workflow"
        echo "  2. Use the n8n web interface to test manual workflows"
        echo "  3. Use the n8n CLI (currently broken in v1.93.0+)"
        echo ""
        echo "To create an API-executable workflow:"
        echo "  - Add a Webhook node as the trigger"
        echo "  - Set a unique path (e.g., 'my-workflow')"
        echo "  - In Webhook settings, set:"
        echo "    • Respond: 'When Last Node Finishes'"
        echo "    • Response Data: 'Last Node' (to get workflow output)"
        echo "  - Save and activate the workflow"
        echo "  - Execute with: $0 --action execute --workflow-id $workflow_id"
        return 1
    fi
}

#######################################
# Show API setup instructions
#######################################
n8n::show_api_setup_instructions() {
    local auth_user="${N8N_BASIC_AUTH_USER:-admin}"
    local auth_pass
    
    # Try to get password from running container
    if docker ps --format "{{.Names}}" | grep -q "^${N8N_CONTAINER_NAME}$"; then
        auth_pass=$(docker exec "$N8N_CONTAINER_NAME" env | grep N8N_BASIC_AUTH_PASSWORD | cut -d'=' -f2)
    fi
    
    cat << EOF
=== n8n API Setup Instructions ===

The n8n CLI execute command has a known bug in versions 1.93.0+.
You need to use the REST API instead. Here's how:

1. Access n8n Web Interface:
   URL: $N8N_BASE_URL
   Username: $auth_user
   Password: ${auth_pass:-[check container logs or env]}

2. Create API Key:
   - Go to Settings → n8n API
   - Click "Create an API key"
   - Set a label (e.g., "CLI Access")
   - Set expiration (optional)
   - Copy the API key (shown only once!)

3. Save API Key (Choose One):
   Option A - Save to configuration file (recommended):
     $0 --action save-api-key --api-key YOUR_API_KEY
   
   Option B - Set environment variable (temporary):
     export N8N_API_KEY="your-api-key-here"

4. Execute Workflows:
   $0 --action execute --workflow-id YOUR_WORKFLOW_ID

Alternative: Direct API Call:
   curl -X POST -H "X-N8N-API-KEY: \$N8N_API_KEY" \\
        -H "Content-Type: application/json" \\
        ${N8N_BASE_URL}/api/v1/workflows/WORKFLOW_ID/execute

Note: This is a workaround for GitHub issue #15567.
The standard CLI command 'n8n execute --id' is broken in current versions.
EOF
}

#######################################
# API setup helper
#######################################
n8n::api_setup() {
    log::header "🔑 n8n API Setup Guide"
    n8n::show_api_setup_instructions
}

#######################################
# Save API key to configuration
#######################################
n8n::save_api_key() {
    local api_key="${API_KEY:-}"
    
    log::header "💾 Save n8n API Key"
    
    # Check if API key is provided
    if [[ -z "$api_key" ]]; then
        log::error "API key is required"
        echo ""
        echo "Usage: $0 --action save-api-key --api-key YOUR_API_KEY"
        echo ""
        echo "To create an API key:"
        echo "  1. Access n8n at $N8N_BASE_URL"
        echo "  2. Go to Settings → n8n API"
        echo "  3. Create and copy your API key"
        return 1
    fi
    
    # Ensure config directory exists
    local config_dir="${HOME}/.vrooli"
    mkdir -p "$config_dir"
    
    # Load existing config or create new
    local config_file="${config_dir}/resources.local.json"
    local config
    
    if [[ -f "$config_file" ]]; then
        # Backup existing config
        cp "$config_file" "${config_file}.backup" 2>/dev/null || true
        config=$(cat "$config_file")
    else
        # Create default config structure
        config='{
  "version": "1.0.0",
  "enabled": true,
  "services": {
    "ai": {},
    "automation": {},
    "storage": {},
    "agents": {}
  }
}'
    fi
    
    # Update config with API key
    local updated_config
    updated_config=$(echo "$config" | jq --arg key "$api_key" '
        .services.automation.n8n = (.services.automation.n8n // {}) |
        .services.automation.n8n.apiKey = $key |
        .services.automation.n8n.enabled = true |
        .services.automation.n8n.baseUrl = "http://localhost:5678" |
        .services.automation.n8n.healthCheck = {
            "intervalMs": 60000,
            "timeoutMs": 5000
        }
    ')
    
    # Write updated config
    echo "$updated_config" | jq '.' > "$config_file"
    
    # Set secure permissions
    chmod 600 "$config_file"
    
    log::success "✅ API key saved to $config_file"
    echo ""
    echo "You can now execute workflows without setting N8N_API_KEY:"
    echo "  $0 --action execute --workflow-id YOUR_WORKFLOW_ID"
    echo ""
    echo "The API key will be loaded automatically from the configuration."
}

#######################################
# Main execution function
#######################################
n8n::main() {
    n8n::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            n8n::install
            ;;
        "uninstall")
            n8n::uninstall
            ;;
        "start")
            n8n::start
            ;;
        "stop")
            n8n::stop
            ;;
        "restart")
            n8n::restart
            ;;
        "status")
            n8n::status
            ;;
        "reset-password")
            n8n::reset_password
            ;;
        "logs")
            n8n::logs
            ;;
        "info")
            n8n::info
            ;;
        "execute")
            n8n::execute
            ;;
        "api-setup")
            n8n::api_setup
            ;;
        "save-api-key")
            n8n::save_api_key
            ;;
        *)
            log::error "Unknown action: $ACTION"
            n8n::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    n8n::main "$@"
fi