#!/usr/bin/env bash
# n8n Core Functions - Consolidated n8n-specific logic
# All generic operations delegated to shared libraries

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_CORE_SOURCED:-}" ]] && return 0
export _N8N_CORE_SOURCED=1

# Source shared libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
N8N_LIB_DIR="${APP_ROOT}/resources/n8n/lib"
N8N_SCRIPT_DIR="${APP_ROOT}/resources/n8n"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/status-engine.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/health-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/backup-framework.sh"
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

#######################################
# Get n8n initialization configuration
# Returns: JSON configuration for init framework
#######################################
n8n::get_init_config() {
    local webhook_url="${1:-}"
    local auth_password="${2:-}"
    
    # Use custom image if available, otherwise fallback to standard
    local image_to_use="$N8N_IMAGE"
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${N8N_CUSTOM_IMAGE}$"; then
        image_to_use="$N8N_CUSTOM_IMAGE"
        # Log to stderr to avoid contaminating JSON output
        log::info "Using custom n8n image with advanced features" >&2
    fi
    
    # Build environment variables
    local timezone
    timezone=$(timedatectl show -p Timezone --value 2>/dev/null || echo 'UTC')
    
    # Build volumes list - always include advanced mounts for better functionality
    local volumes_array="[\"${N8N_DATA_DIR}:/home/node/.n8n\""
    volumes_array+=",\"/var/run/docker.sock:/var/run/docker.sock:rw\""
    volumes_array+=",\"${PWD}:/workspace:rw\""
    volumes_array+=",\"/usr/bin:/host/usr/bin:ro\""
    volumes_array+=",\"$HOME:/host/home:rw\""
    # Mount Vrooli root for CLI access
    volumes_array+=",\"${VROOLI_ROOT}:/vrooli:ro\""
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
        "networks": ["host"],
        "first_run_check": "n8n::is_first_run",
        "setup_func": "n8n::first_time_setup",
        "wait_for_ready": "n8n::wait_for_ready"
    }'
    
    # Add database configuration
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        # Use localhost for host network mode
        config=$(echo "$config" | jq '.env_vars += {
            "DB_TYPE": "postgresdb",
            "DB_POSTGRESDB_HOST": "localhost",
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
# Create n8n backup using backup framework
# Args: $1 - optional label (default: "auto")
# Returns: 0 on success, 1 on failure
#######################################
n8n::create_backup() {
    local label="${1:-auto}"
    
    if ! n8n::is_running; then
        log::error "n8n must be running to create backup"
        return 1
    fi
    
    log::info "Creating n8n backup..."
    
    # Create temporary backup directory with n8n data
    local temp_dir=$(mktemp -d)
    
    # Copy n8n data directory
    if [[ -d "$N8N_DATA_DIR" ]]; then
        cp -r "$N8N_DATA_DIR"/* "$temp_dir/" 2>/dev/null || true
    fi
    
    # Include current API key from secrets if available
    local api_key
    api_key=$(n8n::resolve_api_key 2>/dev/null)
    if [[ -n "$api_key" ]]; then
        echo "{\"N8N_API_KEY\":\"$api_key\"}" > "$temp_dir/api_key_backup.json"
    fi
    
    # Store backup using framework
    local backup_path
    if backup_path=$(backup::store "n8n" "$temp_dir" "$label"); then
        log::success "n8n backup created: $(basename "$backup_path")"
        rm -rf "$temp_dir"
        echo "$backup_path"
        return 0
    else
        log::error "Failed to create n8n backup"
        rm -rf "$temp_dir"
        return 1
    fi
}

#######################################
# Test if backup contains valid API key
# Args: $1 - backup path
# Returns: 0 if valid API key found, 1 otherwise
#######################################
n8n::backup_has_valid_api_key() {
    local backup_path="$1"
    
    # Check for API key backup file
    local api_key_file="$backup_path/api_key_backup.json"
    [[ -f "$api_key_file" ]] || return 1
    
    # Extract API key
    local stored_key
    stored_key=$(jq -r '.N8N_API_KEY // empty' "$api_key_file" 2>/dev/null)
    [[ -n "$stored_key" ]] || return 1
    
    # Test if key would work (basic validation only - don't actually test against server)
    # Keys are typically long alphanumeric strings
    [[ ${#stored_key} -gt 20 ]] && [[ "$stored_key" =~ ^[A-Za-z0-9_-]+$ ]]
}

#######################################
# Smart API key recovery - find working backup
# Returns: 0 on success, 1 if no valid key found
#######################################
n8n::recover_api_key() {
    log::header "🔑 Smart API Key Recovery"
    
    local valid_backup
    valid_backup=$(backup::find_first "n8n" 'n8n::backup_has_valid_api_key')
    
    if [[ -n "$valid_backup" ]]; then
        local backup_id=$(basename "$valid_backup")
        log::info "Found valid API key in backup: $backup_id"
        
        # Extract and save the API key
        local api_key_file="$valid_backup/api_key_backup.json"
        local recovered_key
        recovered_key=$(jq -r '.N8N_API_KEY' "$api_key_file" 2>/dev/null)
        
        if [[ -n "$recovered_key" ]] && secrets::save_key "N8N_API_KEY" "$recovered_key"; then
            log::success "✅ API key recovered and saved"
            log::info "Key source: backup $backup_id"
            return 0
        else
            log::error "Failed to save recovered API key"
            return 1
        fi
    else
        log::error "❌ No valid API key found in any backup"
        echo ""
        echo "This can happen when:"
        echo "  • No backups exist yet"
        echo "  • All existing backups were created when API key was already invalid"
        echo "  • Backups don't contain API key information"
        echo ""
        echo "To fix this:"
        echo "  1. Access n8n at $N8N_BASE_URL"
        echo "  2. Login with your credentials"  
        echo "  3. Go to Settings → n8n API"
        echo "  4. Create a new API key"
        echo "  5. Save it with: \$0 --action save-api-key --api-key YOUR_NEW_KEY"
        return 1
    fi
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
            "unhealthy": "Service not responding - Try: resource-n8n manage restart"
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
    # wait::for_http expects: (url, expected_code, timeout, headers)
    # n8n health endpoint returns HTTP 200
    wait::for_http "${N8N_BASE_URL}/healthz" 200 60
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
    
    # Skip network creation for host network mode
    # Network is already set to "host" in get_init_config
    
    # Install PostgreSQL if needed
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        n8n::install_postgres
    fi
    
    # Use init framework for container setup
    local init_config
    init_config=$(n8n::get_init_config)
    init::setup_resource "$init_config"
    
    # Auto-install CLI if available
    "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "$N8N_SCRIPT_DIR" 2>/dev/null || true
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
        log::info "   📋 List workflows: resource-n8n content workflows"
        log::info "   ▶️  Execute: resource-n8n content execute ID"
        
        # Display auto-credential status
        log::info ""
        n8n::validate_auto_credentials
        
        # Show credential management commands
        log::info ""
        log::info "🔐 Credential Management:"
        log::info "   🤖 Auto-create: resource-n8n content auto-credentials"
        log::info "   🔄 Refresh all: resource-n8n content auto-credentials"
        log::info "   🔍 List resources: resource-n8n content list-discoverable"
    else
        log::info "   Configure API key to manage workflows and credentials"
    fi
}

n8n::health() {
    # Use health framework
    local config
    config=$(n8n::get_health_config)
    health::tiered_check "$config"
}

n8n::recover() {
    # Warn about API key risk - recovery can wipe database
    if ! n8n::warn_api_key_risk "recovery operation"; then
        return 1
    fi
    
    log::header "🔧 n8n Recovery"
    
    # Find the latest backup
    local latest_backup
    latest_backup=$(backup::get_latest "n8n")
    
    if [[ -z "$latest_backup" ]]; then
        log::error "No backups found for n8n"
        echo ""
        echo "Cannot recover without backups. To create a backup:"
        echo "  \$0 --action create-backup"
        return 1
    fi
    
    local backup_id=$(basename "$latest_backup")
    log::info "Found backup to restore: $backup_id"
    
    # Stop n8n if running
    if n8n::is_running; then
        log::info "Stopping n8n for recovery..."
        n8n::stop
    fi
    
    # Backup current state before recovery
    if [[ -d "$N8N_DATA_DIR" ]]; then
        local emergency_backup="${N8N_DATA_DIR}.corrupted.$(date +%Y%m%d_%H%M%S)"
        log::info "Backing up current state to: $emergency_backup"
        mv "$N8N_DATA_DIR" "$emergency_backup" || true
    fi
    
    # Create fresh data directory
    mkdir -p "$N8N_DATA_DIR"
    
    # Restore from backup
    log::info "Restoring from backup..."
    if cp -r "$latest_backup"/* "$N8N_DATA_DIR/" 2>/dev/null; then
        # Remove backup-specific files that shouldn't be in data dir
        rm -f "$N8N_DATA_DIR/api_key_backup.json" 2>/dev/null || true
        rm -f "$N8N_DATA_DIR/.metadata.json" 2>/dev/null || true
        
        # Fix permissions
        chown -R 1000:1000 "$N8N_DATA_DIR" 2>/dev/null || true
        
        log::success "Data restored from backup"
        
        # Start n8n
        log::info "Starting n8n..."
        if n8n::start; then
            # Try to recover API key from backup
            log::info "Attempting API key recovery..."
            if n8n::recover_api_key; then
                log::success "✅ Recovery completed successfully with API key"
            else
                log::success "✅ Recovery completed (API key recovery failed)"
                log::info "You may need to create a new API key manually"
            fi
        else
            log::error "Failed to start n8n after recovery"
            return 1
        fi
    else
        log::error "Failed to restore from backup"
        return 1
    fi
    
    # Check API key status after recovery
    n8n::check_api_key_after_operation "recovery"
}

n8n::uninstall() {
    log::warn "Uninstalling n8n..."
    
    # Warn about API key risk if data will be removed
    if [[ "${REMOVE_DATA:-no}" == "yes" ]]; then
        if ! n8n::warn_api_key_risk "uninstall with data removal"; then
            return 1
        fi
    fi
    
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
        # Create backup before removal using backup framework
        if n8n::is_running && [[ -d "$N8N_DATA_DIR" ]]; then
            log::info "Creating final backup before data removal..."
            n8n::create_backup "pre-uninstall" || log::warn "Backup creation failed"
        fi
        
        rm -rf "$N8N_DATA_DIR"
        
        # Check API key status after data removal
        n8n::check_api_key_after_operation "data removal"
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
    
    # Create PostgreSQL container with port mapping for host network access
    docker run -d \
        --name "$N8N_DB_CONTAINER_NAME" \
        -p 5432:5432 \
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
    
    # Warn about API key risk
    if ! n8n::warn_api_key_risk "password reset"; then
        return 1
    fi
    
    local new_password="${AUTH_PASSWORD:-$(openssl rand -base64 16 2>/dev/null || date +%s | sha256sum | head -c 16)}"
    
    log::info "Restarting n8n with new password..."
    n8n::stop
    
    # Remove old container
    docker::remove_container "$N8N_CONTAINER_NAME" "true"
    
    # Reinstall with new password
    AUTH_PASSWORD="$new_password" n8n::install
    
    # Check API key status after password reset
    n8n::check_api_key_after_operation "password reset"
    
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
                                       validate-injection, url, auto-credentials,
                                       refresh-credentials, validate-credentials,
                                       list-discoverable

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
    --auto-credentials yes/no Auto-create credentials for resources (default: yes)
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

    # Auto-create credentials for all running resources
    $0 --action auto-credentials

    # Refresh credentials (recreate for all resources)
    $0 --action refresh-credentials

    # List discoverable resources
    $0 --action list-discoverable

    # Validate existing auto-credentials
    $0 --action validate-credentials

    # Install with auto-credentials disabled
    $0 --action install --auto-credentials no

EOF
}

# ============================================================================
# Injection Functions
# ============================================================================

n8n::register_injection_framework() {
    # Source injection framework if not already loaded
    if ! command -v inject_framework::register >/dev/null 2>&1; then
        source "${var_SCRIPTS_RESOURCES_LIB_DIR}/inject_framework.sh"
    fi
    
    inject_framework::register "n8n" \
        --service-host "$N8N_BASE_URL" \
        --health-endpoint "/healthz" \
        --validate-func "n8n::validate_config" \
        --inject-func "n8n::inject_data" \
        --status-func "n8n::check_status" \
        --health-func "n8n::check_health"
}

n8n::inject() {
    local injection_config="${INJECTION_CONFIG:-}"
    local injection_config_file="${INJECTION_CONFIG_FILE:-}"
    
    # Check if we have either direct config or config file
    if [[ -z "$injection_config" && -z "$injection_config_file" ]]; then
        log::error "Missing required --injection-config or --injection-config-file parameter"
        return 1
    fi
    
    # If config file is provided, read it
    if [[ -n "$injection_config_file" ]]; then
        if [[ ! -f "$injection_config_file" ]]; then
            log::error "Injection config file not found: $injection_config_file"
            return 1
        fi
        injection_config=$(cat "$injection_config_file")
    fi
    
    # Load n8n injection functions
    source "${N8N_SCRIPT_DIR}/lib/inject.sh"
    
    # Register with framework and inject
    n8n::register_injection_framework
    inject_framework::main --inject "$injection_config"
}

n8n::validate_injection() {
    local validation_type="${VALIDATION_TYPE:-}"
    local validation_file="${VALIDATION_FILE:-}"
    local injection_config="${INJECTION_CONFIG:-}"
    local injection_config_file="${INJECTION_CONFIG_FILE:-}"
    
    # Support both legacy and new validation interfaces
    if [[ -n "$validation_type" && -n "$validation_file" ]]; then
        # New interface: type + file
        if [[ ! -f "$validation_file" ]]; then
            log::error "Validation file not found: $validation_file"
            return 1
        fi
        
        local content
        content=$(cat "$validation_file")
        
        # Call inject.sh with type and content
        "${N8N_SCRIPT_DIR}/lib/inject.sh" \
            --validate \
            --type "$validation_type" \
            --content "$content"
    elif [[ -n "$injection_config" || -n "$injection_config_file" ]]; then
        # Handle file-based config if provided
        if [[ -n "$injection_config_file" ]]; then
            if [[ ! -f "$injection_config_file" ]]; then
                log::error "Injection config file not found: $injection_config_file"
                return 1
            fi
            injection_config=$(cat "$injection_config_file")
        fi
        
        # Legacy interface: full config JSON
        source "${N8N_SCRIPT_DIR}/lib/inject.sh"
        n8n::register_injection_framework
        inject_framework::main --validate "$injection_config"
    else
        log::error "Required: --validation-type TYPE --validation-file PATH"
        log::info "   or: --injection-config 'JSON_CONFIG' or --injection-config-file PATH"
        return 1
    fi
}