#!/usr/bin/env bash
# n8n Status Functions - Minimal wrapper using status engine

# Source core and frameworks
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/status-engine.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/docker-utils.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/core.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/health.sh"

#######################################
# Show n8n status using unified engine
#######################################
n8n::status() {
    # Check Docker daemon
    if ! docker::check_daemon; then
        return 1
    fi
    
    # Use status engine
    local config
    config=$(n8n::get_status_config)
    status::display_unified_status "$config" "n8n::display_workflow_info"
}

#######################################
# Enhanced status (alias for compatibility)
#######################################
n8n::enhanced_status() {
    n8n::status
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
- Docker Image: $N8N_IMAGE
- Data Directory: $N8N_DATA_DIR

Configuration:
- Authentication: ${BASIC_AUTH:-yes}
- Database: ${DATABASE_TYPE:-sqlite}
- Tunnel Mode: ${TUNNEL_ENABLED:-no}

For more information, visit: https://docs.n8n.io
EOF
}

#######################################
# Show container details
#######################################
n8n::inspect() {
    if ! docker::container_exists "$N8N_CONTAINER_NAME"; then
        log::error "n8n container does not exist"
        return 1
    fi
    
    log::header "🔍 n8n Container Details"
    
    # Use Docker utilities
    local created image networks volumes
    created=$(docker inspect "$N8N_CONTAINER_NAME" --format '{{.Created}}' 2>/dev/null || echo "unknown")
    image=$(docker inspect "$N8N_CONTAINER_NAME" --format '{{.Config.Image}}' 2>/dev/null || echo "unknown")
    networks=$(docker inspect "$N8N_CONTAINER_NAME" --format '{{range $k, $v := .NetworkSettings.Networks}}{{$k}} {{end}}' 2>/dev/null || echo "none")
    volumes=$(docker inspect "$N8N_CONTAINER_NAME" --format '{{range .Mounts}}{{.Source}} -> {{.Destination}}{{println}}{{end}}' 2>/dev/null || echo "none")
    
    cat << EOF
Created: $created
Image: $image
Networks: $networks

Volume Mounts:
$volumes
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
    version=$(docker exec "$N8N_CONTAINER_NAME" n8n --version 2>/dev/null || echo "unknown")
    log::info "n8n version: $version"
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
    docker stats --no-stream "$N8N_CONTAINER_NAME"
    
    # Show PostgreSQL stats if used
    if n8n::postgres_running; then
        echo
        log::info "PostgreSQL Stats:"
        docker stats --no-stream "$N8N_DB_CONTAINER_NAME"
    fi
}

#######################################
# Comprehensive check of all components
#######################################
n8n::check_all() {
    log::header "🔍 n8n Comprehensive Health Check"
    
    local all_good=true
    
    # Docker check
    if docker::check_daemon; then
        log::success "✅ Docker is ready"
    else
        log::error "❌ Docker issues detected"
        all_good=false
    fi
    
    # Container check
    if n8n::is_installed; then
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
    
    # Health check
    local health_tier
    health_tier=$(n8n::tiered_health_check)
    case "$health_tier" in
        "HEALTHY")
            log::success "✅ n8n is healthy"
            ;;
        "DEGRADED")
            log::warn "⚠️  n8n is degraded"
            ;;
        "UNHEALTHY")
            log::error "❌ n8n is unhealthy"
            all_good=false
            ;;
    esac
    
    # Database check
    if n8n::check_database_health; then
        log::success "✅ Database is healthy"
    else
        log::error "❌ Database issues detected"
        all_good=false
    fi
    
    # Port check
    if docker::is_port_available "$N8N_PORT"; then
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
    else
        log::error "❌ Some checks failed"
        log::info "💡 Try running: ./manage.sh --action restart"
        return 1
    fi
}

#######################################
# Test n8n functionality
#######################################
n8n::test() {
    log::info "Testing n8n functionality..."
    
    # Run comprehensive health check
    if n8n::check_all; then
        log::success "🎉 All n8n tests passed"
        return 0
    else
        log::error "❌ Some tests failed"
        return 1
    fi
}