#!/usr/bin/env bash
# n8n Core Functions - Consolidated n8n-specific logic
# All generic operations delegated to shared libraries

# Source shared libraries
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/status-engine.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/health-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/recovery-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/init-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/wait-utils.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

#######################################
# n8n Configuration Constants
# These are set in config/defaults.sh as readonly
# Only set non-readonly variables here
#######################################
# Variables that aren't set as readonly in defaults.sh
: "${DATABASE_TYPE:=sqlite}"
: "${BASIC_AUTH:=yes}"
: "${AUTH_USERNAME:=admin}"
: "${TUNNEL_ENABLED:=no}"
: "${BUILD_IMAGE:=no}"

#######################################
# Get n8n initialization configuration
# Returns: JSON configuration for init framework
#######################################
n8n::get_init_config() {
    local webhook_url="${1:-}"
    local auth_password="${2:-}"
    
    # Select image (custom if available)
    local image_to_use="$N8N_IMAGE"
    if [[ "$BUILD_IMAGE" == "yes" ]] && docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${N8N_CUSTOM_IMAGE}$"; then
        image_to_use="$N8N_CUSTOM_IMAGE"
    fi
    
    # Build environment variables
    local timezone
    timezone=$(timedatectl show -p Timezone --value 2>/dev/null || echo 'UTC')
    
    # Build volumes list
    local volumes_array="[\"${N8N_DATA_DIR}:/home/node/.n8n\""
    if [[ "$BUILD_IMAGE" == "yes" ]]; then
        volumes_array+=",\"/var/run/docker.sock:/var/run/docker.sock:rw\""
        volumes_array+=",\"${PWD}:/workspace:rw\""
        volumes_array+=",\"/usr/bin:/host/usr/bin:ro\""
        volumes_array+=",\"$HOME:/host/home:rw\""
    fi
    volumes_array+="]"
    
    # Build init config
    local config='{
        "resource_name": "n8n",
        "container_name": "'$N8N_CONTAINER_NAME'",
        "data_dir": "'$N8N_DATA_DIR'",
        "port": '$N8N_PORT',
        "image": "'$image_to_use'",
        "env_vars": {
            "GENERIC_TIMEZONE": "'$timezone'",
            "TZ": "'$timezone'",
            "N8N_DIAGNOSTICS_ENABLED": "false",
            "N8N_TEMPLATES_ENABLED": "true",
            "N8N_PERSONALIZATION_ENABLED": "false",
            "N8N_PUBLIC_API_DISABLED": "false"
        },
        "volumes": '$volumes_array',
        "networks": ["'$N8N_NETWORK_NAME'"],
        "first_run_check": "n8n::is_first_run",
        "setup_func": "n8n::first_time_setup",
        "wait_for_ready": "n8n::wait_for_ready"
    }'
    
    # Add database configuration
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        config=$(echo "$config" | jq '.env_vars += {
            "DB_TYPE": "postgresdb",
            "DB_POSTGRESDB_HOST": "'$N8N_DB_CONTAINER_NAME'",
            "DB_POSTGRESDB_PORT": "5432",
            "DB_POSTGRESDB_DATABASE": "n8n"
        }')
    else
        config=$(echo "$config" | jq '.env_vars += {
            "DB_TYPE": "sqlite",
            "DB_SQLITE_VACUUM_ON_STARTUP": "true"
        }')
    fi
    
    # Add authentication if enabled
    if [[ "$BASIC_AUTH" == "yes" ]] && [[ -n "$auth_password" ]]; then
        config=$(echo "$config" | jq '.env_vars += {
            "N8N_BASIC_AUTH_ACTIVE": "true",
            "N8N_BASIC_AUTH_USER": "'$AUTH_USERNAME'",
            "N8N_BASIC_AUTH_PASSWORD": "'$auth_password'"
        }')
    fi
    
    echo "$config"
}

#######################################
# Get health check configuration
# Returns: JSON configuration for health framework
#######################################
n8n::get_health_config() {
    echo '{
        "container_name": "'$N8N_CONTAINER_NAME'",
        "checks": {
            "basic": "n8n::check_basic_health",
            "advanced": "n8n::check_api_functionality"
        }
    }'
}

#######################################
# Get recovery configuration
# Returns: JSON configuration for recovery framework
#######################################
n8n::get_recovery_config() {
    echo '{
        "container_name": "'$N8N_CONTAINER_NAME'",
        "data_dir": "'$N8N_DATA_DIR'",
        "backup_dir": "'$N8N_DATA_DIR'/backups",
        "backup_pattern": "*.db",
        "stop_func": "n8n::stop",
        "start_func": "n8n::start",
        "owner": "1000:1000"
    }'
}

#######################################
# Get status configuration
# Returns: JSON configuration for status engine
#######################################
n8n::get_status_config() {
    echo '{
        "resource": {
            "name": "n8n",
            "category": "automation",
            "description": "Business workflow automation platform",
            "port": '$N8N_PORT',
            "container_name": "'$N8N_CONTAINER_NAME'",
            "data_dir": "'$N8N_DATA_DIR'"
        },
        "endpoints": {
            "ui": "'$N8N_BASE_URL'",
            "api": "'$N8N_BASE_URL'/api/v1",
            "health": "'$N8N_BASE_URL'/healthz"
        },
        "health_tiers": {
            "healthy": "All systems operational",
            "degraded": "API key missing - Configure at '$N8N_BASE_URL' → Settings",
            "unhealthy": "Service not responding - Try: ./manage.sh --action restart"
        },
        "auth": {
            "type": "api-key",
            "status_func": "n8n::display_auth_status"
        }
    }'
}

#######################################
# n8n-specific health checks
#######################################
n8n::check_basic_health() {
    # Check if container is running
    docker::is_running "$N8N_CONTAINER_NAME" || return 1
    
    # Check health endpoint
    health::check_api "${N8N_BASE_URL}/healthz" || return 1
    
    return 0
}

n8n::check_api_functionality() {
    local api_key
    api_key=$(n8n::resolve_api_key)
    
    if [[ -z "$api_key" ]]; then
        return 1  # Degraded - no API key
    fi
    
    # Test API with authentication
    health::check_api "${N8N_BASE_URL}/api/v1/workflows?limit=1" "X-N8N-API-KEY: $api_key"
}

#######################################
# Resolve API key from multiple sources
# Returns: API key or empty string
#######################################
n8n::resolve_api_key() {
    # 1. Environment variable
    if [[ -n "${N8N_API_KEY:-}" ]]; then
        echo "$N8N_API_KEY"
        return 0
    fi
    
    # 2. Project secrets
    local secret_key
    secret_key=$(secrets::get_key "N8N_API_KEY" 2>/dev/null || echo "")
    if [[ -n "$secret_key" ]]; then
        echo "$secret_key"
        return 0
    fi
    
    # 3. Container environment (if running)
    if docker::is_running "$N8N_CONTAINER_NAME"; then
        local container_key
        container_key=$(docker exec "$N8N_CONTAINER_NAME" printenv N8N_API_KEY 2>/dev/null || echo "")
        if [[ -n "$container_key" ]]; then
            echo "$container_key"
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Check if first run
# Returns: 0 if first run, 1 otherwise
#######################################
n8n::is_first_run() {
    # Check if database exists
    if [[ "$DATABASE_TYPE" == "sqlite" ]]; then
        [[ ! -f "${N8N_DATA_DIR}/database.sqlite" ]]
    else
        # For PostgreSQL, check if tables exist
        docker exec "$N8N_DB_CONTAINER_NAME" psql -U n8n -d n8n -c "SELECT 1 FROM workflow LIMIT 1" &>/dev/null
        [[ $? -ne 0 ]]
    fi
}

#######################################
# First time setup
#######################################
n8n::first_time_setup() {
    log::info "Running first-time n8n setup..."
    
    # Create initial workflow example
    if docker::is_running "$N8N_CONTAINER_NAME"; then
        log::info "n8n is ready for initial configuration"
        log::info "Access n8n at: $N8N_BASE_URL"
        log::info "Create your first workflow in the web interface"
    fi
    
    return 0
}

#######################################
# Wait for n8n to be ready
#######################################
n8n::wait_for_ready() {
    wait::for_http "${N8N_BASE_URL}/healthz" 60
}

#######################################
# Display authentication status
#######################################
n8n::display_auth_status() {
    # Basic auth status
    if docker::is_running "$N8N_CONTAINER_NAME"; then
        local auth_active
        auth_active=$(docker exec "$N8N_CONTAINER_NAME" printenv N8N_BASIC_AUTH_ACTIVE 2>/dev/null || echo "false")
        if [[ "$auth_active" == "true" ]]; then
            log::info "   ✅ Basic Auth: Enabled"
        else
            log::warn "   ⚠️  Basic Auth: Disabled"
        fi
    fi
    
    # API key status
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -n "$api_key" ]]; then
        log::success "   ✅ API Key: Configured"
    else
        log::warn "   ⚠️  API Key: Not configured"
    fi
}

#######################################
# Core n8n operations using frameworks
#######################################
n8n::install() {
    log::info "Installing n8n..."
    
    # Create data directory
    init::create_data_dir "$N8N_DATA_DIR"
    
    # Create network
    docker::create_network "$N8N_NETWORK_NAME"
    
    # Install PostgreSQL if needed
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        n8n::install_postgres
    fi
    
    # Use init framework for container setup
    local init_config
    init_config=$(n8n::get_init_config)
    init::setup_resource "$init_config"
}




n8n::display_workflow_info() {
    # Custom workflow information section
    if ! docker::is_running "$N8N_CONTAINER_NAME"; then
        return 0
    fi
    
    log::info "⚡ Workflow Management:"
    
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -n "$api_key" ]]; then
        log::info "   📋 List workflows: ./manage.sh --action list-workflows"
        log::info "   ▶️  Execute: ./manage.sh --action execute --workflow-id ID"
    else
        log::info "   Configure API key to manage workflows"
    fi
}

n8n::health() {
    # Use health framework
    local config
    config=$(n8n::get_health_config)
    health::tiered_check "$config"
}

n8n::recover() {
    # Use recovery framework
    local config
    config=$(n8n::get_recovery_config)
    recovery::auto_recover "$config"
}

n8n::uninstall() {
    log::warn "Uninstalling n8n..."
    
    # Stop containers
    n8n::stop
    
    # Remove containers
    docker::remove_container "$N8N_CONTAINER_NAME" "true"
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        docker::remove_container "$N8N_DB_CONTAINER_NAME" "true"
    fi
    
    # Optionally remove data
    if [[ "${REMOVE_DATA:-no}" == "yes" ]]; then
        log::warn "Removing n8n data directory..."
        rm -rf "$N8N_DATA_DIR"
    fi
    
    log::info "n8n uninstalled"
}

#######################################
# PostgreSQL support for n8n
#######################################
n8n::install_postgres() {
    log::info "Setting up PostgreSQL for n8n..."
    
    # Create PostgreSQL data directory
    local pg_data_dir="${N8N_DATA_DIR}/postgres"
    init::create_data_dir "$pg_data_dir"
    
    # Create PostgreSQL container
    docker run -d \
        --name "$N8N_DB_CONTAINER_NAME" \
        --network "$N8N_NETWORK_NAME" \
        -e POSTGRES_USER=n8n \
        -e POSTGRES_PASSWORD="${N8N_DB_PASSWORD:-n8n_password}" \
        -e POSTGRES_DB=n8n \
        -v "$pg_data_dir:/var/lib/postgresql/data" \
        --restart unless-stopped \
        postgres:14-alpine >/dev/null 2>&1
    
    # Wait for PostgreSQL to be ready
    wait::for_condition "docker exec $N8N_DB_CONTAINER_NAME pg_isready -U n8n" 30
}


#######################################
# Show API setup instructions
#######################################
n8n::show_api_setup_instructions() {
    log::info "To create an n8n API key:"
    log::info "  1. Access n8n at $N8N_BASE_URL"
    log::info "  2. Go to Settings → n8n API"
    log::info "  3. Create and copy your API key"
    log::info "  4. Save it with: ./manage.sh --action save-api-key --api-key YOUR_KEY"
}

#######################################
# Reset password for basic auth
#######################################
n8n::reset_password() {
    log::header "🔐 Reset n8n Password"
    
    if ! docker::is_running "$N8N_CONTAINER_NAME"; then
        log::error "n8n is not running"
        return 1
    fi
    
    local new_password="${AUTH_PASSWORD:-$(openssl rand -base64 16 2>/dev/null || date +%s | sha256sum | head -c 16)}"
    
    log::info "Restarting n8n with new password..."
    n8n::stop
    
    # Remove old container
    docker::remove_container "$N8N_CONTAINER_NAME" "true"
    
    # Reinstall with new password
    AUTH_PASSWORD="$new_password" n8n::install
    
    log::success "Password reset successfully"
    log::info "New login credentials:"
    log::info "  Username: $AUTH_USERNAME"
    log::info "  Password: $new_password"
}




#######################################
# Usage information
#######################################
n8n::usage() {
    cat << EOF
n8n Workflow Automation Platform Manager

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --action, -a ACTION       Action to perform (default: install)
                             Available: install, uninstall, start, stop, restart,
                                       status, reset-password, logs, info, test,
                                       execute, api-setup, save-api-key, inject,
                                       validate-injection, url

    --force, -f              Force action even if already installed/running
    --lines, -n NUMBER       Number of log lines to show (default: 50)
    --webhook-url URL        External webhook URL for testing
    --workflow-id ID         Workflow ID for execution
    --api-key KEY           n8n API key to save
    --data JSON             JSON data for workflow execution
    --basic-auth yes/no     Enable basic authentication (default: yes)
    --username NAME         Basic auth username (default: admin)
    --password PASS         Basic auth password (default: auto-generated)
    --database TYPE         Database type: sqlite or postgres (default: sqlite)
    --tunnel yes/no         Enable tunnel for webhook testing (default: no)
    --build-image yes/no    Build custom n8n image (default: no)
    --help, -h              Show this help message

EXAMPLES:
    # Install n8n
    $0 --action install

    # Check status
    $0 --action status

    # Execute workflow
    $0 --action execute --workflow-id YOUR_WORKFLOW_ID

    # Save API key
    $0 --action save-api-key --api-key YOUR_API_KEY

EOF
}