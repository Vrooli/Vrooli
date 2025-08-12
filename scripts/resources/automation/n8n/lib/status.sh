#!/usr/bin/env bash
# n8n Status and Information Functions
# Status checking, health monitoring, and information display

# Source specialized modules
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/docker-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/http-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/status-engine.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/health.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/utils.sh" 2>/dev/null || true

# NOTE: Old display functions removed - replaced by unified status engine
# All status display now handled by n8n::enhanced_status() using status::display_unified_status()

#######################################
# Create n8n status configuration for the unified status engine
# Returns: JSON configuration via stdout
#######################################
n8n::_build_status_config() {
    local config='{
        "resource": {
            "name": "n8n",
            "category": "automation",
            "description": "Business workflow automation platform",
            "port": '${N8N_PORT}',
            "container_name": "'${N8N_CONTAINER_NAME}'",
            "data_dir": "'${N8N_DATA_DIR}'"
        },
        "endpoints": {
            "ui": "'${N8N_BASE_URL}'",
            "api": "'${N8N_BASE_URL}'/api/v1",
            "health": "'${N8N_BASE_URL}'/healthz"
        },
        "health_tiers": {
            "healthy": "All systems operational",
            "degraded": "API key missing - Go to '${N8N_BASE_URL}' → Settings → n8n API → Create API key, then save with: ./manage.sh --action save-api-key --api-key YOUR_KEY",
            "unhealthy": "Service not responding - Try: ./manage.sh --action restart"
        },
        "auth": {
            "type": "api-key",
            "status_func": "n8n::_display_auth_status"
        }
    }'
    echo "$config"
}

#######################################
# Custom authentication status display for n8n
# Called by the status engine when auth.status_func is specified
#######################################
n8n::_display_auth_status() {
    # Basic auth status
    if docker::is_running "$N8N_CONTAINER_NAME"; then
        local auth_active
        auth_active=$(n8n::extract_container_env "N8N_BASIC_AUTH_ACTIVE")
        if [[ "$auth_active" == "true" ]]; then
            local auth_user
            auth_user=$(n8n::extract_container_env "N8N_BASIC_AUTH_USER")
            log::info "   ✅ Basic Auth: Enabled (user: ${auth_user:-admin})"
        else
            log::warn "   ⚠️  Basic Auth: Disabled"
        fi
    else
        log::warn "   ❓ Basic Auth: Cannot determine (container not running)"
    fi
    
    # API key status
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -n "$api_key" ]]; then
        # Validate the API key
        if n8n::validate_api_key_setup >/dev/null 2>&1; then
            log::success "   ✅ API Key: Valid and configured (${#api_key} chars)"
        else
            log::error "   ❌ API Key: Configured but invalid (${#api_key} chars)"
        fi
    else
        log::warn "   ⚠️  API Key: Not configured"
        log::info "      Set up at: $N8N_BASE_URL → Settings → n8n API"
    fi
}

#######################################
# Enhanced status with unified status engine
# REFACTORED: Now uses the unified status display system
#######################################
n8n::enhanced_status() {
    local status_config
    status_config=$(n8n::_build_status_config)
    
    # Use the unified status engine with custom workflow section
    status::display_unified_status "$status_config" "n8n::_display_workflow_section"
}

#######################################
# Custom workflow section for n8n status display
# Args: $1 - status_config (JSON configuration)
#######################################
n8n::_display_workflow_section() {
    local status_config="$1"
    
    local container_name
    container_name=$(echo "$status_config" | jq -r '.resource.container_name // empty')
    
    if [[ -z "$container_name" ]] || ! docker::is_running "$N8N_CONTAINER_NAME"; then
        return 0
    fi
    
    log::info "⚡ Workflow Management:"
    
    # Try to get workflow count via API if available
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -n "$api_key" ]] && n8n::validate_api_key_setup >/dev/null 2>&1; then
        local workflow_count
        workflow_count=$(http::request "GET" "${N8N_BASE_URL}/api/v1/workflows?limit=1" "" "X-N8N-API-KEY: $api_key" 2>/dev/null | jq -r '.count // "unknown"' 2>/dev/null || echo "unknown")
        log::info "   💼 Total Workflows: $workflow_count"
        log::info "   📋 Management: ./manage.sh --action workflow-list"
        log::info "   ▶️  Execution: ./manage.sh --action execute --workflow-id ID"
    else
        log::info "   💼 Workflows: Configure API key to see workflow details"
        log::info "   📋 Management: Set up API access first"
    fi
    
    # Show example workflow creation
    log::info "   ➕ Create: Access $N8N_BASE_URL to build workflows"
}

#######################################
# Show n8n status with enhanced tiered health check
#######################################
n8n::status() {
    # Use shared Docker validation
    if ! docker::check_daemon; then
        return 1
    fi
    
    # Delegate to unified status engine
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
    if ! docker::container_exists "$N8N_CONTAINER_NAME"; then
        log::error "n8n container does not exist"
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
    if ! docker::is_running "$N8N_CONTAINER_NAME"; then
        log::error "n8n is not running"
        return 1
    fi
    local version
    version=$(docker::exec "$N8N_CONTAINER_NAME" n8n --version 2>/dev/null)
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
    if ! docker::is_running "$N8N_CONTAINER_NAME"; then
        log::error "n8n is not running"
        return 1
    fi
    log::header "📈 n8n Resource Usage"
    # Container stats
    docker::get_stats "$N8N_CONTAINER_NAME"
    # If using PostgreSQL, show its stats too
    if n8n::postgres_running; then
        echo
        log::info "PostgreSQL Stats:"
        docker::get_stats "$N8N_DB_CONTAINER_NAME"
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
    if docker::check_daemon; then
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
    if docker::container_exists "$N8N_CONTAINER_NAME"; then
        log::success "✅ n8n container exists"
        if docker::is_running "$N8N_CONTAINER_NAME"; then
            log::success "✅ n8n container is running"
        else
            log::error "❌ n8n container is not running"
            all_good=false
        fi
    else
        log::error "❌ n8n container does not exist"
        all_good=false
    fi
    # API check with enhanced diagnostics
    if n8n::is_healthy; then
        log::success "✅ n8n API is healthy"
    else
        log::error "❌ n8n API is not responding"
        all_good=false
        # Check for common issues
        if docker::is_running "$N8N_CONTAINER_NAME"; then
            local recent_logs
            recent_logs=$(docker::get_logs "$N8N_CONTAINER_NAME" 20 2>&1 || echo "")
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
        if docker::is_running "$N8N_CONTAINER_NAME"; then
            log::success "✅ Port $N8N_PORT is in use by n8n"
        else
            log::warn "⚠️  Port $N8N_PORT is available"
        fi
    else
        if ! docker::is_running "$N8N_CONTAINER_NAME"; then
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
        log::error "❌ Docker is not installed"
        return 1
    fi
    log::success "✅ Docker is available"
    # Test 2: Check if n8n container exists
    if ! docker::container_exists "$N8N_CONTAINER_NAME"; then
        log::error "❌ n8n container does not exist"
        return 1
    fi
    log::success "✅ n8n container exists"
    # Test 3: Check if n8n is running
    if ! docker::is_running "$N8N_CONTAINER_NAME"; then
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
