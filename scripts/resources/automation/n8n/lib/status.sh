#!/usr/bin/env bash
# n8n Status and Information Functions
# Status checking, health monitoring, and information display

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
# Show detailed container information
#######################################
n8n::inspect() {
    if ! n8n::container_exists; then
        log::error "n8n container does not exist"
        return 1
    fi
    
    log::header "🔍 n8n Container Details"
    
    # Basic info
    local created
    created=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{.Created}}' 2>/dev/null)
    log::info "Created: $created"
    
    local image
    image=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{.Config.Image}}' 2>/dev/null)
    log::info "Image: $image"
    
    # Network info
    echo
    log::info "Network Configuration:"
    docker inspect "$N8N_CONTAINER_NAME" --format='{{range $net, $conf := .NetworkSettings.Networks}}  {{$net}}: {{$conf.IPAddress}}{{end}}' 2>/dev/null
    
    # Volume mounts
    echo
    log::info "Volume Mounts:"
    docker inspect "$N8N_CONTAINER_NAME" --format='{{range .Mounts}}  {{.Source}} -> {{.Destination}}{{println}}{{end}}' 2>/dev/null
    
    # Environment variables (filtered)
    echo
    log::info "Environment Variables (filtered):"
    docker inspect "$N8N_CONTAINER_NAME" --format='{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null | \
        grep -E '^(N8N_|DB_|WEBHOOK_|TZ=)' | \
        grep -v PASSWORD | \
        sort
}

#######################################
# Get n8n version
#######################################
n8n::version() {
    if ! n8n::is_running; then
        log::error "n8n is not running"
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
    if ! n8n::is_running; then
        log::error "n8n is not running"
        return 1
    fi
    
    log::header "📈 n8n Resource Usage"
    
    # Container stats
    docker stats "$N8N_CONTAINER_NAME" --no-stream
    
    # If using PostgreSQL, show its stats too
    if docker ps --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
        echo
        log::info "PostgreSQL Stats:"
        docker stats "$N8N_DB_CONTAINER_NAME" --no-stream
    fi
}

#######################################
# Check all n8n components
#######################################
n8n::check_all() {
    log::header "🔍 n8n Health Check"
    
    local all_good=true
    
    # Docker check
    if n8n::check_docker; then
        log::success "✅ Docker is ready"
    else
        log::error "❌ Docker issues detected"
        all_good=false
    fi
    
    # Container check
    if n8n::container_exists; then
        log::success "✅ n8n container exists"
        
        if n8n::is_running; then
            log::success "✅ n8n container is running"
        else
            log::error "❌ n8n container is not running"
            all_good=false
        fi
    else
        log::error "❌ n8n container does not exist"
        all_good=false
    fi
    
    # API check
    if n8n::is_healthy; then
        log::success "✅ n8n API is healthy"
    else
        log::error "❌ n8n API is not responding"
        all_good=false
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
        if n8n::is_running; then
            log::success "✅ Port $N8N_PORT is in use by n8n"
        else
            log::warn "⚠️  Port $N8N_PORT is available"
        fi
    else
        if ! n8n::is_running; then
            log::error "❌ Port $N8N_PORT is in use by another service"
            all_good=false
        fi
    fi
    
    echo
    if [[ "$all_good" == "true" ]]; then
        log::success "✅ All checks passed!"
    else
        log::error "❌ Some checks failed"
        return 1
    fi
}