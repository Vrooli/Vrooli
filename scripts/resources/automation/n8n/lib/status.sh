#!/usr/bin/env bash
# n8n Status and Information Functions
# Status checking, health monitoring, and information display

# Source specialized modules
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/docker-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/http-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/health.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/utils.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/constants.sh" 2>/dev/null || true

#######################################
# Display container status and stats
# Args: none
# Returns: 0 if running, 2 if not
#######################################
n8n::display_container_status() {
    if ! n8n::container_exists_any; then
        log::error "❌ Container: Not found"
        return 2
    elif ! n8n::container_running; then
        log::error "❌ Container: Exists but not running"
        return 2
    fi
    local stats=$(docker stats "$N8N_CONTAINER_NAME" --no-stream --format "{{.CPUPerc}}|{{.MemUsage}}" 2>/dev/null || echo "")
    log::success "✅ Container: Running ($N8N_CONTAINER_NAME) [$stats]"
    return 0
}

#######################################
# Display health tier with actionable feedback
# Args: $1 - health tier (HEALTHY/DEGRADED/UNHEALTHY)
#######################################
n8n::display_health_tier_feedback() {
    local health_tier="$1"
    case "$health_tier" in
        "HEALTHY")
            log::success "✅ Health: HEALTHY - All systems operational"
            ;;
        "DEGRADED")
            log::warn "⚠️  Health: DEGRADED - API key missing"
            log::info "Fix: Go to $N8N_BASE_URL → Settings → n8n API → Create API key"
            log::info "Then: $0 --action save-api-key --api-key YOUR_KEY"
            ;;
        "UNHEALTHY") 
            log::error "❌ Health: UNHEALTHY - Service not responding. Try: $0 --action restart"
            ;;
    esac
}

#######################################
# Display service endpoint details
#######################################
n8n::display_service_details() {
    log::info "Service: UI=$N8N_BASE_URL | Health=$N8N_BASE_URL/healthz | API=$N8N_BASE_URL/api/v1"
}

#######################################
# Display authentication information
#######################################
n8n::display_authentication_info() {
    # Basic auth status using shared utility
    local auth_active
    auth_active=$(n8n::extract_container_env "N8N_BASIC_AUTH_ACTIVE")
    if [[ "$auth_active" == "true" ]]; then
        local auth_user
        auth_user=$(n8n::extract_container_env "N8N_BASIC_AUTH_USER")
        log::info "  Web Auth: Enabled (user: ${auth_user:-admin})"
    else
        log::warn "  Web Auth: Disabled"
    fi
    # API key status using shared utility
    local api_key_status="Not configured"
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -n "$api_key" ]]; then
        api_key_status="Configured (${#api_key} chars)"
    fi
    log::info "  API Key: $api_key_status"
}

#######################################
# Enhanced status with tiered health reporting
# REFACTORED: Main orchestrator function (moved from common.sh)
# Provides clear feedback about what works vs what's missing
#######################################
n8n::enhanced_status() {
    log::header "📊 n8n Enhanced Status Check"
    # Get tiered health status
    local health_tier
    local health_code
    health_tier=$(n8n::tiered_health_check)
    health_code=$?
    # Display container status
    if ! n8n::display_container_status; then
        return 2
    fi
    # Display health tier feedback
    n8n::display_health_tier_feedback "$health_tier"
    # Display service and authentication details
    n8n::display_service_details
    n8n::display_authentication_info
    return $health_code
}

#######################################
# Show n8n status with enhanced tiered health check
#######################################
n8n::status() {
    # Pre-flight checks
    if ! system::is_command "docker"; then
        log::error "$N8N_ERR_DOCKER_NOT_INSTALLED"
        log::info "$N8N_FIX_INSTALL_DOCKER"
        return 1
    fi
    if ! docker info >/dev/null 2>&1; then
        log::error "$N8N_ERR_DOCKER_NOT_RUNNING"
        log::info "$N8N_FIX_START_DOCKER"
        return 1
    fi
    # Use enhanced status with tiered health check
    n8n::enhanced_status
    local result=$?
    # Show PostgreSQL status if relevant
    if n8n::postgres_exists; then
        log::info "\nPostgreSQL Database:"
        if n8n::postgres_running; then
            log::success "  Status: ✅ Running"
        else
            log::warn "  Status: ⚠️  Stopped"
        fi
    fi
    # Show helpful commands
    log::info "\nAvailable Commands:"
    log::info "  View logs: $0 --action logs"
    log::info "  Restart: $0 --action restart"
    log::info "  Full test: $0 --action test"
    return $result
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
# Show detailed container information
#######################################
n8n::inspect() {
    if ! n8n::container_exists_any; then
        log::error "$N8N_ERR_CONTAINER_NOT_EXISTS"
        return 1
    fi
    log::header "🔍 n8n Container Details"
    
    # Gather all details using standardized Docker utilities
    local created image networks volumes env_vars
    created=$(docker::get_created "$N8N_CONTAINER_NAME")
    image=$(docker::get_image "$N8N_CONTAINER_NAME")
    networks=$(docker::get_networks "$N8N_CONTAINER_NAME")
    volumes=$(docker::get_mounts "$N8N_CONTAINER_NAME")
    env_vars=$(docker::get_all_env "$N8N_CONTAINER_NAME" | \
        grep -E '^(N8N_|DB_|WEBHOOK_|TZ=)' | grep -v PASSWORD | sort)
    
    # Output all at once
    cat << EOF
Created: $created
Image: $image

Network Configuration:
$networks

Volume Mounts:
$volumes

Environment Variables (filtered):
$env_vars
EOF
}

#######################################
# Get n8n version
#######################################
n8n::version() {
    if ! n8n::container_running; then
        log::error "$N8N_ERR_CONTAINER_NOT_RUNNING"
        return 1
    fi
    local version
    version=$(docker exec "$N8N_CONTAINER_NAME" n8n --version 2>/dev/null)
    if [[ -n "$version" ]]; then
        log::info "n8n version: $version"
    else
        log::error "Failed to get n8n version"
        return 1
    fi
}

#######################################
# Show resource usage statistics
#######################################
n8n::stats() {
    if ! n8n::container_running; then
        log::error "$N8N_ERR_CONTAINER_NOT_RUNNING"
        return 1
    fi
    log::header "📈 n8n Resource Usage"
    # Container stats
    docker stats "$N8N_CONTAINER_NAME" --no-stream
    # If using PostgreSQL, show its stats too
    if n8n::postgres_running; then
        echo
        log::info "PostgreSQL Stats:"
        docker stats "$N8N_DB_CONTAINER_NAME" --no-stream
    fi
}

#######################################
# Check all n8n components with enhanced diagnostics
#######################################
n8n::check_all() {
    log::header "🔍 n8n Comprehensive Health Check"
    local all_good=true
    local recovery_attempted=false
    # Docker check
    if n8n::check_docker; then
        log::success "✅ Docker is ready"
    else
        log::error "❌ Docker issues detected"
        all_good=false
    fi
    # Filesystem corruption check
    if ! n8n::detect_filesystem_corruption; then
        log::error "❌ Filesystem corruption detected"
        if [[ "${AUTO_RECOVER:-yes}" == "yes" ]] && [[ "$recovery_attempted" == "false" ]]; then
            log::info "Attempting automatic recovery..."
            if n8n::auto_recover; then
                log::success "✅ Automatic recovery completed"
                recovery_attempted=true
            else
                log::error "❌ Automatic recovery failed"
                all_good=false
            fi
        else
            all_good=false
        fi
    else
        log::success "✅ Filesystem is healthy"
    fi
    # Database health check
    if n8n::check_database_health; then
        log::success "✅ Database is healthy"
    else
        log::error "❌ Database issues detected"
        all_good=false
    fi
    # Container check
    if n8n::container_exists_any; then
        log::success "✅ n8n container exists"
        if n8n::container_running; then
            log::success "✅ n8n container is running"
        else
            log::error "❌ n8n container is not running"
            all_good=false
        fi
    else
        log::error "❌ $N8N_ERR_CONTAINER_NOT_EXISTS"
        all_good=false
    fi
    # API check with enhanced diagnostics
    if n8n::is_healthy; then
        log::success "✅ n8n API is healthy"
    else
        log::error "❌ n8n API is not responding"
        all_good=false
        # Check for common issues
        if n8n::container_running; then
            local recent_logs
            recent_logs=$(docker logs "$N8N_CONTAINER_NAME" --tail 20 2>&1 || echo "")
            if echo "$recent_logs" | grep -qi "SQLITE_READONLY"; then
                log::info "💡 Detected SQLite readonly errors - restart may help"
            fi
            if echo "$recent_logs" | grep -qi "EACCES\|permission denied"; then
                log::info "💡 Detected permission errors - check data directory permissions"
            fi
        fi
    fi
    # Database check
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        if n8n::postgres_is_healthy; then
            log::success "✅ PostgreSQL is healthy"
        else
            log::error "❌ PostgreSQL issues detected"
            all_good=false
        fi
    fi
    # Port check
    if n8n::is_port_available "$N8N_PORT"; then
        if n8n::container_running; then
            log::success "✅ Port $N8N_PORT is in use by n8n"
        else
            log::warn "⚠️  Port $N8N_PORT is available"
        fi
    else
        if ! n8n::container_running; then
            log::error "❌ Port $N8N_PORT is in use by another service"
            all_good=false
        fi
    fi
    echo
    if [[ "$all_good" == "true" ]]; then
        log::success "✅ All checks passed!"
        if [[ "$recovery_attempted" == "true" ]]; then
            log::info "💡 Recovery was performed - consider running a backup"
        fi
    else
        log::error "❌ Some checks failed"
        if [[ "$recovery_attempted" == "false" ]]; then
            log::info "💡 Try running: $0 --action restart (with automatic recovery)"
        fi
        return 1
    fi
}

#######################################
# Test n8n functionality
# Returns: 0 if tests pass, 1 if tests fail, 2 if skip
#######################################
n8n::test() {
    log::info "Testing n8n functionality..."
    # Test 1: Check if Docker is available
    if ! system::is_command "docker"; then
        log::error "❌ $N8N_ERR_DOCKER_NOT_INSTALLED"
        return 1
    fi
    log::success "✅ Docker is available"
    # Test 2: Check if n8n container exists
    if ! n8n::container_exists_any; then
        log::error "❌ $N8N_ERR_CONTAINER_NOT_EXISTS"
        return 1
    fi
    log::success "✅ n8n container exists"
    # Test 3: Check if n8n is running
    if ! n8n::container_running; then
        log::error "❌ n8n container is not running"
        return 1
    fi
    log::success "✅ n8n container is running"
    # Test 4: Check API health
    if ! n8n::is_healthy; then
        log::error "❌ n8n API is not responding"
        return 1
    fi
    log::success "✅ n8n API is healthy"
    # Test 5: Run comprehensive health check
    log::info "Running comprehensive health check..."
    if n8n::check_all; then
        log::success "✅ Comprehensive health check passed"
    else
        log::error "❌ Comprehensive health check failed"
        return 1
    fi
    # Test 6: Check workflows endpoint (automation-specific test)
    if n8n::is_healthy; then
        log::info "Testing workflows API endpoint..."
        local workflows_response
        # Use standardized HTTP utility  
        http::request "GET" "http://localhost:${N8N_PORT}/api/v1/workflows" "" "Accept: application/json" >/dev/null 2>&1
        workflows_response=$?
        if [[ "$workflows_response" == "200" ]] || [[ "$workflows_response" == "401" ]]; then
            log::success "✅ Workflows API endpoint is accessible"
        else
            log::warn "⚠️  Workflows API endpoint test failed (HTTP: $workflows_response)"
        fi
    fi
    log::success "🎉 All n8n tests passed"
    return 0
}
