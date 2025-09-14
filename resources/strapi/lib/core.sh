#!/usr/bin/env bash
# Strapi Resource Core Library
# Implements lifecycle management and core functionality

set -euo pipefail

# Prevent multiple sourcing
[[ -n "${STRAPI_CORE_LOADED:-}" ]] && return 0
readonly STRAPI_CORE_LOADED=1

# Resource configuration
readonly STRAPI_CORE_VERSION="5.0"
readonly STRAPI_PORT="${STRAPI_PORT:-1337}"
readonly STRAPI_HOST="${STRAPI_HOST:-0.0.0.0}"
readonly STRAPI_DATA_DIR="${STRAPI_DATA_DIR:-${HOME}/.vrooli/strapi}"
readonly STRAPI_PROJECT_DIR="${STRAPI_DATA_DIR}/app"
readonly STRAPI_LOG_FILE="${STRAPI_DATA_DIR}/strapi.log"

# Database configuration
readonly POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
readonly POSTGRES_PORT="${POSTGRES_PORT:-5433}"
readonly POSTGRES_USER="${POSTGRES_USER:-postgres}"
readonly POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"
readonly STRAPI_DATABASE_NAME="${STRAPI_DATABASE_NAME:-strapi}"

# Admin configuration
readonly STRAPI_ADMIN_EMAIL="${STRAPI_ADMIN_EMAIL:-admin@vrooli.local}"
readonly STRAPI_ADMIN_PASSWORD="${STRAPI_ADMIN_PASSWORD:-$(openssl rand -base64 12)}"
readonly STRAPI_ADMIN_JWT_SECRET="${STRAPI_ADMIN_JWT_SECRET:-$(openssl rand -base64 32)}"
readonly STRAPI_APP_KEYS="${STRAPI_APP_KEYS:-$(openssl rand -base64 32),$(openssl rand -base64 32)}"

#######################################
# Print colored output
#######################################
core::print() {
    local color="$1"
    shift
    echo -e "${color}$*\033[0m"
}

core::info() {
    core::print "\033[0;34m" "[INFO] $*"
}

core::success() {
    core::print "\033[0;32m" "[SUCCESS] $*"
}

core::warning() {
    core::print "\033[0;33m" "[WARNING] $*"
}

core::error() {
    core::print "\033[0;31m" "[ERROR] $*" >&2
}

#######################################
# Check if Strapi is installed
#######################################
core::is_installed() {
    [[ -d "${STRAPI_PROJECT_DIR}" ]] && \
    [[ -f "${STRAPI_PROJECT_DIR}/package.json" ]] && \
    command -v node >/dev/null 2>&1
}

#######################################
# Check if Strapi is running
#######################################
core::is_running() {
    if command -v pm2 >/dev/null 2>&1; then
        pm2 list 2>/dev/null | grep -q "strapi" && return 0
    fi
    
    # Check if port is in use
    timeout 2 bash -c "echo > /dev/tcp/localhost/${STRAPI_PORT}" 2>/dev/null && return 0
    
    return 1
}

#######################################
# Install Strapi and dependencies
#######################################
core::manage_install() {
    local force="${1:-false}"
    
    if core::is_installed; then
        core::warning "Strapi is already installed"
        return 2
    fi
    
    core::info "Installing Strapi v${STRAPI_CORE_VERSION}..."
    
    # Check Node.js
    if ! command -v node >/dev/null 2>&1; then
        core::error "Node.js is required but not installed"
        core::info "Install Node.js 20 LTS first"
        return 1
    fi
    
    # Check Node version
    local node_version=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
    if [[ "$node_version" -lt 18 ]]; then
        core::error "Node.js 18+ required, found v${node_version}"
        return 1
    fi
    
    # Install PM2 globally if not present
    if ! command -v pm2 >/dev/null 2>&1; then
        core::info "Installing PM2 process manager..."
        npm install -g pm2 || {
            core::error "Failed to install PM2"
            return 1
        }
    fi
    
    # Create data directory
    mkdir -p "${STRAPI_DATA_DIR}"
    
    # Create Strapi project
    core::info "Creating Strapi project..."
    cd "${STRAPI_DATA_DIR}"
    
    # Use npx to create Strapi app with PostgreSQL
    npx create-strapi-app@latest app \
        --dbclient=postgres \
        --dbhost="${POSTGRES_HOST}" \
        --dbport="${POSTGRES_PORT}" \
        --dbname="${STRAPI_DATABASE_NAME}" \
        --dbusername="${POSTGRES_USER}" \
        --dbpassword="${POSTGRES_PASSWORD}" \
        --dbssl=false \
        --no-run \
        --quickstart \
        --skip-cloud || {
        core::error "Failed to create Strapi project"
        return 1
    }
    
    # Create environment file
    cat > "${STRAPI_PROJECT_DIR}/.env" << EOF
HOST=${STRAPI_HOST}
PORT=${STRAPI_PORT}
APP_KEYS=${STRAPI_APP_KEYS}
API_TOKEN_SALT=$(openssl rand -base64 32)
ADMIN_JWT_SECRET=${STRAPI_ADMIN_JWT_SECRET}
TRANSFER_TOKEN_SALT=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 32)

# Database
DATABASE_CLIENT=postgres
DATABASE_HOST=${POSTGRES_HOST}
DATABASE_PORT=${POSTGRES_PORT}
DATABASE_NAME=${STRAPI_DATABASE_NAME}
DATABASE_USERNAME=${POSTGRES_USER}
DATABASE_PASSWORD=${POSTGRES_PASSWORD}
DATABASE_SSL=false

# Admin
STRAPI_ADMIN_EMAIL=${STRAPI_ADMIN_EMAIL}
STRAPI_ADMIN_PASSWORD=${STRAPI_ADMIN_PASSWORD}
EOF
    
    # Build Strapi
    core::info "Building Strapi (this may take a few minutes)..."
    cd "${STRAPI_PROJECT_DIR}"
    npm run build || {
        core::error "Failed to build Strapi"
        return 1
    }
    
    # Create PM2 ecosystem file
    cat > "${STRAPI_DATA_DIR}/ecosystem.config.js" << EOF
module.exports = {
  apps: [{
    name: 'strapi',
    script: 'npm',
    args: 'run start',
    cwd: '${STRAPI_PROJECT_DIR}',
    env: {
      NODE_ENV: 'production',
      HOST: '${STRAPI_HOST}',
      PORT: '${STRAPI_PORT}'
    },
    max_memory_restart: '512M',
    error_file: '${STRAPI_DATA_DIR}/error.log',
    out_file: '${STRAPI_DATA_DIR}/out.log',
    merge_logs: true,
    time: true
  }]
};
EOF
    
    # Save credentials
    cat > "${STRAPI_DATA_DIR}/credentials.json" << EOF
{
  "admin_email": "${STRAPI_ADMIN_EMAIL}",
  "admin_password": "${STRAPI_ADMIN_PASSWORD}",
  "admin_url": "http://localhost:${STRAPI_PORT}/admin",
  "api_url": "http://localhost:${STRAPI_PORT}/api",
  "graphql_url": "http://localhost:${STRAPI_PORT}/graphql"
}
EOF
    
    core::success "Strapi installed successfully"
    core::info "Admin credentials saved to: ${STRAPI_DATA_DIR}/credentials.json"
    return 0
}

#######################################
# Uninstall Strapi
#######################################
core::manage_uninstall() {
    local keep_data="${1:-false}"
    
    if ! core::is_installed; then
        core::warning "Strapi is not installed"
        return 2
    fi
    
    # Stop if running
    if core::is_running; then
        core::info "Stopping Strapi service..."
        core::manage_stop
    fi
    
    if [[ "$keep_data" != "--keep-data" ]]; then
        core::info "Removing Strapi data..."
        rm -rf "${STRAPI_DATA_DIR}"
    else
        core::info "Keeping user data as requested"
    fi
    
    core::success "Strapi uninstalled successfully"
    return 0
}

#######################################
# Start Strapi service
#######################################
core::manage_start() {
    local wait_flag="false"
    local timeout=60
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait)
                wait_flag="true"
                ;;
            --timeout)
                timeout="${2:-60}"
                shift
                ;;
        esac
        shift
    done
    
    if ! core::is_installed; then
        core::error "Strapi is not installed. Run 'manage install' first"
        return 1
    fi
    
    if core::is_running; then
        core::warning "Strapi is already running"
        return 2
    fi
    
    core::info "Starting Strapi service..."
    
    # Start with PM2
    cd "${STRAPI_DATA_DIR}"
    pm2 start ecosystem.config.js || {
        core::error "Failed to start Strapi with PM2"
        return 1
    }
    
    if [[ "$wait_flag" == "true" ]]; then
        core::info "Waiting for Strapi to be ready (timeout: ${timeout}s)..."
        local elapsed=0
        
        while [[ $elapsed -lt $timeout ]]; do
            if timeout 5 curl -sf "http://localhost:${STRAPI_PORT}/health" >/dev/null 2>&1; then
                core::success "Strapi is ready"
                return 0
            fi
            sleep 2
            elapsed=$((elapsed + 2))
        done
        
        core::error "Strapi failed to become ready within ${timeout} seconds"
        return 1
    fi
    
    core::success "Strapi started successfully"
    return 0
}

#######################################
# Stop Strapi service
#######################################
core::manage_stop() {
    local force="${1:-false}"
    local timeout=30
    
    if ! core::is_running; then
        core::warning "Strapi is not running"
        return 2
    fi
    
    core::info "Stopping Strapi service..."
    
    if command -v pm2 >/dev/null 2>&1; then
        pm2 stop strapi 2>/dev/null || true
        pm2 delete strapi 2>/dev/null || true
    fi
    
    # Wait for shutdown
    local elapsed=0
    while [[ $elapsed -lt $timeout ]] && core::is_running; do
        sleep 1
        elapsed=$((elapsed + 1))
    done
    
    if core::is_running; then
        if [[ "$force" == "--force" ]]; then
            core::warning "Force stopping Strapi..."
            pkill -f "strapi" || true
        else
            core::error "Failed to stop Strapi gracefully"
            return 1
        fi
    fi
    
    core::success "Strapi stopped successfully"
    return 0
}

#######################################
# Restart Strapi service
#######################################
core::manage_restart() {
    core::info "Restarting Strapi service..."
    
    if core::is_running; then
        core::manage_stop || return 1
    fi
    
    core::manage_start "$@"
}

#######################################
# Show service status
#######################################
core::show_status() {
    local json_mode="${1:-false}"
    
    local status="stopped"
    local health="unhealthy"
    local uptime="0"
    
    if core::is_running; then
        status="running"
        
        if timeout 5 curl -sf "http://localhost:${STRAPI_PORT}/health" >/dev/null 2>&1; then
            health="healthy"
        fi
        
        if command -v pm2 >/dev/null 2>&1; then
            uptime=$(pm2 list 2>/dev/null | grep "strapi" | awk '{print $10}' || echo "unknown")
        fi
    fi
    
    if [[ "$json_mode" == "--json" ]]; then
        cat << EOF
{
  "status": "${status}",
  "health": "${health}",
  "port": ${STRAPI_PORT},
  "uptime": "${uptime}",
  "admin_url": "http://localhost:${STRAPI_PORT}/admin",
  "api_url": "http://localhost:${STRAPI_PORT}/api"
}
EOF
    else
        echo "Strapi Service Status"
        echo "===================="
        echo "  Status: ${status}"
        echo "  Health: ${health}"
        echo "  Port: ${STRAPI_PORT}"
        echo "  Uptime: ${uptime}"
        echo "  Admin URL: http://localhost:${STRAPI_PORT}/admin"
        echo "  API URL: http://localhost:${STRAPI_PORT}/api"
    fi
}

#######################################
# Show service logs
#######################################
core::show_logs() {
    local tail_lines=50
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tail)
                tail_lines="${2:-50}"
                shift
                ;;
        esac
        shift
    done
    
    if command -v pm2 >/dev/null 2>&1; then
        pm2 logs strapi --lines "$tail_lines"
    elif [[ -f "${STRAPI_LOG_FILE}" ]]; then
        tail -n "$tail_lines" "${STRAPI_LOG_FILE}"
    else
        core::error "No logs available"
        return 1
    fi
}

#######################################
# Show admin credentials
#######################################
core::show_credentials() {
    if [[ -f "${STRAPI_DATA_DIR}/credentials.json" ]]; then
        cat "${STRAPI_DATA_DIR}/credentials.json"
    else
        core::error "Credentials file not found"
        return 1
    fi
}

#######################################
# Content management functions
#######################################
core::content_list() {
    if ! core::is_running; then
        core::error "Strapi is not running"
        return 1
    fi
    
    core::info "Fetching content types..."
    curl -sf "http://localhost:${STRAPI_PORT}/api/content-type-builder/content-types" | jq '.'
}

core::content_add() {
    local content_type="$1"
    local data="$2"
    
    if ! core::is_running; then
        core::error "Strapi is not running"
        return 1
    fi
    
    curl -X POST \
        -H "Content-Type: application/json" \
        -d "$data" \
        "http://localhost:${STRAPI_PORT}/api/${content_type}"
}

core::content_get() {
    local content_type="$1"
    local id="${2:-}"
    
    if ! core::is_running; then
        core::error "Strapi is not running"
        return 1
    fi
    
    local url="http://localhost:${STRAPI_PORT}/api/${content_type}"
    [[ -n "$id" ]] && url="${url}/${id}"
    
    curl -sf "$url" | jq '.'
}

core::content_remove() {
    local content_type="$1"
    local id="$2"
    
    if ! core::is_running; then
        core::error "Strapi is not running"
        return 1
    fi
    
    curl -X DELETE "http://localhost:${STRAPI_PORT}/api/${content_type}/${id}"
}

core::content_execute() {
    local query="$1"
    
    if ! core::is_running; then
        core::error "Strapi is not running"
        return 1
    fi
    
    # Execute GraphQL query
    curl -X POST \
        -H "Content-Type: application/json" \
        -d "{\"query\":\"$query\"}" \
        "http://localhost:${STRAPI_PORT}/graphql" | jq '.'
}

#######################################
# Create admin user automatically
#######################################
core::create_admin() {
    local email="${1:-${STRAPI_ADMIN_EMAIL}}"
    local password="${2:-${STRAPI_ADMIN_PASSWORD}}"
    local firstname="${3:-Admin}"
    local lastname="${4:-User}"
    
    if ! core::is_running; then
        core::error "Strapi must be running to create admin user"
        return 1
    fi
    
    core::info "Creating admin user..."
    
    # Create admin user via Strapi's admin registration endpoint
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"${email}\",
            \"password\": \"${password}\",
            \"firstname\": \"${firstname}\",
            \"lastname\": \"${lastname}\"
        }" \
        "http://localhost:${STRAPI_PORT}/admin/register-admin")
    
    if echo "$response" | grep -q "error"; then
        # Check if admin already exists
        if echo "$response" | grep -q "already exists"; then
            core::info "Admin user already exists"
            return 0
        else
            core::error "Failed to create admin user: $response"
            return 1
        fi
    fi
    
    # Save updated credentials
    cat > "${STRAPI_DATA_DIR}/credentials.json" << EOF
{
  "admin_email": "${email}",
  "admin_password": "${password}",
  "admin_url": "http://localhost:${STRAPI_PORT}/admin",
  "api_url": "http://localhost:${STRAPI_PORT}/api",
  "graphql_url": "http://localhost:${STRAPI_PORT}/graphql",
  "admin_firstname": "${firstname}",
  "admin_lastname": "${lastname}"
}
EOF
    
    core::success "Admin user created successfully"
    core::info "Email: ${email}"
    core::info "Password has been saved to credentials.json"
    return 0
}

#######################################
# Setup initial content types
#######################################
core::setup_content_types() {
    if ! core::is_running; then
        core::error "Strapi must be running to setup content types"
        return 1
    fi
    
    core::info "Setting up example content types..."
    
    # This would typically be done through the admin panel
    # or by adding content type definitions to the project
    core::info "Please use the admin panel to create content types"
    core::info "Admin URL: http://localhost:${STRAPI_PORT}/admin"
    
    return 0
}