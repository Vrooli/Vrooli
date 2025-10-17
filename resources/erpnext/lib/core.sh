#!/usr/bin/env bash
################################################################################
# ERPNext Resource - Core Library
# 
# Core functionality for ERPNext ERP system management
# Implements v2.0 universal contract requirements
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
ERPNEXT_RESOURCE_DIR="${APP_ROOT}/resources/erpnext"
ERPNEXT_LIB_DIR="${ERPNEXT_RESOURCE_DIR}/lib"
ERPNEXT_CONFIG_DIR="${ERPNEXT_RESOURCE_DIR}/config"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh" || return 1
# shellcheck disable=SC2154  # var_LIB_UTILS_DIR is set by var.sh
source "${var_LIB_UTILS_DIR}/format.sh" || return 1
source "${var_LIB_UTILS_DIR}/log.sh" || return 1
source "${APP_ROOT}/scripts/resources/port_registry.sh" || return 1
source "${ERPNEXT_CONFIG_DIR}/defaults.sh" || return 1

# Get ERPNext port from registry
ERPNEXT_PORT="${RESOURCE_PORTS["erpnext"]:-8020}"
export ERPNEXT_PORT

################################################################################
# Health Check Functions
################################################################################

erpnext::health_check() {
    local timeout_seconds="${1:-5}"
    
    # Check if service is running
    if ! erpnext::is_running; then
        log::error "ERPNext is not running"
        return 1
    fi
    
    # ERPNext requires proper Host header for multi-tenant setup
    # Check if the service responds with the correct Host header
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    if timeout "$timeout_seconds" curl -sf -o /dev/null -w "%{http_code}" \
        -H "Host: ${site_name}" \
        "http://localhost:${ERPNEXT_PORT}/" | grep -q "^[234]"; then
        return 0
    else
        log::error "Health check failed for ERPNext on port ${ERPNEXT_PORT}"
        return 1
    fi
}

erpnext::is_running() {
    # Check if ERPNext containers are running
    if command -v docker &>/dev/null; then
        local running_containers
        running_containers=$(docker ps --filter "name=erpnext" --format "{{.Names}}" 2>/dev/null | wc -l)
        [[ "$running_containers" -gt 0 ]]
    else
        # Fallback to process check
        pgrep -f "erpnext" &>/dev/null
    fi
}

erpnext::is_healthy() {
    erpnext::health_check 5
}

erpnext::is_installed() {
    # Check if ERPNext data directory exists
    [[ -d "${HOME}/.erpnext" ]]
}

################################################################################
# Lifecycle Management Functions
################################################################################

erpnext::start() {
    log::info "Starting ERPNext..."
    
    if erpnext::is_running; then
        log::warn "ERPNext is already running"
        return 2  # Already running exit code per v2.0
    fi
    
    # Ensure dependencies are running
    erpnext::check_dependencies || {
        log::error "Required dependencies are not available"
        return 1
    }
    
    # Start using Docker Compose
    if [[ -f "${ERPNEXT_LIB_DIR}/docker.sh" ]]; then
        source "${ERPNEXT_LIB_DIR}/docker.sh"
        erpnext::docker::start || {
            log::error "Failed to start ERPNext containers"
            return 1
        }
    else
        log::error "Docker library not found"
        return 1
    fi
    
    # Wait for health with timeout
    local max_wait="${ERPNEXT_STARTUP_TIMEOUT:-90}"
    local waited=0
    
    log::info "Waiting for ERPNext to become healthy (timeout: ${max_wait}s)..."
    
    while [[ $waited -lt $max_wait ]]; do
        if erpnext::is_healthy; then
            log::success "ERPNext started successfully on port ${ERPNEXT_PORT}"
            
            # Initialize site if needed
            if ! erpnext::site_exists; then
                log::info "No site found, initializing..."
                erpnext::initialize_site || {
                    log::warn "Site initialization had issues, but service is running"
                }
            fi
            
            return 0
        fi
        sleep 2
        ((waited+=2))
        echo -n "."
    done
    
    log::error "ERPNext failed to become healthy within ${max_wait} seconds"
    return 1
}

erpnext::stop() {
    log::info "Stopping ERPNext..."
    
    if ! erpnext::is_running; then
        log::warn "ERPNext is not running"
        return 2  # Not running exit code per v2.0
    fi
    
    # Stop using Docker Compose
    if [[ -f "${ERPNEXT_LIB_DIR}/docker.sh" ]]; then
        source "${ERPNEXT_LIB_DIR}/docker.sh"
        erpnext::docker::stop || {
            log::error "Failed to stop ERPNext containers"
            return 1
        }
    else
        log::error "Docker library not found"
        return 1
    fi
    
    log::success "ERPNext stopped successfully"
    return 0
}

erpnext::restart() {
    log::info "Restarting ERPNext..."
    
    # Stop if running (ignore exit code 2 for not running)
    erpnext::stop
    local stop_code=$?
    if [[ $stop_code -ne 0 && $stop_code -ne 2 ]]; then
        log::error "Failed to stop ERPNext"
        return 1
    fi
    
    sleep 2
    
    # Start
    erpnext::start || {
        log::error "Failed to start ERPNext after restart"
        return 1
    }
    
    log::success "ERPNext restarted successfully"
    return 0
}

################################################################################
# Dependency Management
################################################################################

erpnext::check_dependencies() {
    local all_deps_ok=true
    
    # Check PostgreSQL
    if ! timeout 5 nc -zv localhost "${RESOURCE_PORTS["postgres"]:-5432}" &>/dev/null; then
        log::warn "PostgreSQL is not available"
        all_deps_ok=false
    fi
    
    # Check Redis
    if ! timeout 5 nc -zv localhost "${RESOURCE_PORTS["redis"]:-6379}" &>/dev/null; then
        log::warn "Redis is not available"
        all_deps_ok=false
    fi
    
    $all_deps_ok
}

################################################################################
# Information Functions
################################################################################

erpnext::info() {
    local json_output="${1:-false}"
    
    if [[ ! -f "${ERPNEXT_CONFIG_DIR}/runtime.json" ]]; then
        log::error "Runtime configuration not found"
        return 1
    fi
    
    if [[ "$json_output" == "true" ]] || [[ "$json_output" == "--json" ]]; then
        cat "${ERPNEXT_CONFIG_DIR}/runtime.json"
    else
        log::info "ERPNext Resource Information:"
        jq -r 'to_entries[] | "  \(.key): \(.value)"' "${ERPNEXT_CONFIG_DIR}/runtime.json"
    fi
}

erpnext::status() {
    log::info "ERPNext Status:"
    
    # Running status
    if erpnext::is_running; then
        log::success "  Service: Running"
        
        # Health status
        if erpnext::is_healthy; then
            log::success "  Health: Healthy"
        else
            log::warn "  Health: Unhealthy"
        fi
        
        # Port status
        log::info "  Port: ${ERPNEXT_PORT}"
        log::info "  URL: http://localhost:${ERPNEXT_PORT}"
        
        # Site status
        local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
        if erpnext::site_exists "$site_name"; then
            log::success "  Site: ${site_name} (initialized)"
        else
            log::warn "  Site: Not initialized"
        fi
        
        # API status
        local api_status
        api_status=$(erpnext::get_api_status 5 2>/dev/null || echo "Unknown")
        if [[ "$api_status" == *"operational"* ]]; then
            log::success "  API: $api_status"
        else
            log::warn "  API: $api_status"
        fi
        
        # Dependencies
        log::info "  Dependencies:"
        if timeout 5 nc -zv localhost "${RESOURCE_PORTS["postgres"]:-5432}" &>/dev/null; then
            log::success "    PostgreSQL: Connected"
        else
            log::error "    PostgreSQL: Not available"
        fi
        
        if timeout 5 nc -zv localhost "${RESOURCE_PORTS["redis"]:-6379}" &>/dev/null; then
            log::success "    Redis: Connected"
        else
            log::error "    Redis: Not available"
        fi
    else
        log::warn "  Service: Not running"
    fi
    
    # Installation status
    if erpnext::is_installed; then
        log::info "  Installation: Complete"
    else
        log::warn "  Installation: Not installed"
    fi
}

################################################################################
# Site Management Functions
################################################################################

erpnext::site_exists() {
    local site_name="${1:-${ERPNEXT_SITE_NAME:-vrooli.local}}"
    
    if ! erpnext::is_running; then
        return 1
    fi
    
    # Check if site exists in container
    docker exec erpnext-app test -d "/home/frappe/frappe-bench/sites/${site_name}" 2>/dev/null
}

erpnext::initialize_site() {
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    local admin_password="${ERPNEXT_ADMIN_PASSWORD:-admin}"
    
    log::info "Initializing ERPNext site: ${site_name}"
    
    if ! erpnext::is_running; then
        log::error "ERPNext must be running to initialize site"
        return 1
    fi
    
    # Check if site already exists
    if erpnext::site_exists "$site_name"; then
        log::info "Site ${site_name} already exists"
        
        # Set as default site if not already
        docker exec erpnext-app bash -c "echo '${site_name}' > /home/frappe/frappe-bench/sites/currentsite.txt" 2>/dev/null || true
        return 0
    fi
    
    log::info "Creating new site ${site_name}..."
    
    # Wait for database to be ready
    local db_ready=false
    local wait_count=0
    while [[ $wait_count -lt 30 ]]; do
        if docker exec erpnext-app bash -c "mysql -h erpnext-db -u root -p${admin_password} -e 'SELECT 1'" &>/dev/null; then
            db_ready=true
            break
        fi
        sleep 2
        ((wait_count++))
        echo -n "."
    done
    
    if [[ "$db_ready" != "true" ]]; then
        log::error "Database not ready after 60 seconds"
        return 1
    fi
    
    # Create the site with correct db-host
    local create_cmd="bench new-site ${site_name} --db-host erpnext-db --admin-password '${admin_password}' --mariadb-root-password '${admin_password}' --mariadb-user-host-login-scope='%'"
    
    if ! docker exec erpnext-app bash -c "$create_cmd" 2>&1; then
        log::error "Failed to create site ${site_name}"
        return 1
    fi
    
    # Install ERPNext app on the site
    log::info "Installing ERPNext app on site..."
    if ! docker exec erpnext-app bench --site "${site_name}" install-app erpnext 2>&1; then
        log::warn "ERPNext app installation had issues, but continuing..."
    fi
    
    # Set as default site
    docker exec erpnext-app bench use "${site_name}" 2>&1 || true
    docker exec erpnext-app bash -c "echo '${site_name}' > /home/frappe/frappe-bench/sites/currentsite.txt" 2>/dev/null || true
    
    log::success "Site ${site_name} initialized successfully"
    return 0
}

erpnext::get_api_status() {
    local timeout_seconds="${1:-5}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if ! erpnext::is_running; then
        echo "Service not running"
        return 1
    fi
    
    # Check if API responds with proper Host header
    local api_response
    api_response=$(timeout "$timeout_seconds" curl -sf -o /dev/null -w "%{http_code}" \
        -H "Host: ${site_name}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.auth.get_logged_user" 2>/dev/null || echo "000")
    
    case "$api_response" in
        200|401|403)
            echo "API operational (HTTP $api_response)"
            return 0
            ;;
        404)
            echo "API endpoint not found - site may not be initialized"
            return 1
            ;;
        000)
            echo "API not responding"
            return 1
            ;;
        *)
            echo "API returned HTTP $api_response"
            return 1
            ;;
    esac
}

################################################################################
# Credentials and Access Functions
################################################################################

erpnext::show_credentials() {
    log::info "=== ERPNext Access Information ==="
    echo ""
    
    if erpnext::is_running; then
        log::success "Service Status: Running"
        echo ""
        log::info "Access Options:"
        echo ""
        
        # Option 1: Direct access (may need hosts file update)
        log::info "Option 1: Direct Access"
        echo "  URL: http://localhost:${ERPNEXT_PORT}"
        echo "  Note: If you see 'localhost does not exist', use Option 2"
        echo ""
        
        # Option 2: Hosts file modification
        log::info "Option 2: With Hosts File Update"
        echo "  1. Add to /etc/hosts (requires sudo):"
        echo "     127.0.0.1 vrooli.local"
        echo "  2. Access at: http://vrooli.local:${ERPNEXT_PORT}"
        echo ""
        
        # Login credentials
        log::info "Login Credentials:"
        echo "  Username: Administrator"
        echo "  Password: ${ERPNEXT_ADMIN_PASSWORD:-admin}"
        echo ""
        
        # API access
        log::info "API Access:"
        echo "  Endpoint: http://localhost:${ERPNEXT_PORT}/api"
        echo "  Auth: Use login endpoint with above credentials"
        echo "  Example:"
        echo "    curl -X POST http://localhost:${ERPNEXT_PORT}/api/method/login \\"
        echo "      -H 'Host: vrooli.local' \\"
        echo "      -H 'Content-Type: application/json' \\"
        echo "      -d '{\"usr\":\"Administrator\",\"pwd\":\"${ERPNEXT_ADMIN_PASSWORD:-admin}\"}'"
        echo ""
        
        # Current site info
        local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
        if erpnext::site_exists "$site_name"; then
            log::success "Site: ${site_name} (initialized)"
        else
            log::warn "Site: Not initialized - run 'vrooli resource erpnext manage start' to initialize"
        fi
    else
        log::warn "Service Status: Not Running"
        echo ""
        echo "Start ERPNext first:"
        echo "  vrooli resource erpnext manage start"
    fi
}

################################################################################
# Export functions for use by other scripts
################################################################################

export -f erpnext::health_check
export -f erpnext::is_running
export -f erpnext::is_healthy
export -f erpnext::is_installed
export -f erpnext::start
export -f erpnext::stop
export -f erpnext::restart
export -f erpnext::check_dependencies
export -f erpnext::info
export -f erpnext::status
export -f erpnext::site_exists
export -f erpnext::initialize_site
export -f erpnext::get_api_status
export -f erpnext::show_credentials