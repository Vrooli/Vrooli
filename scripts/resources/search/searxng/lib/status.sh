#!/usr/bin/env bash
# SearXNG Status and Health Checking
# Comprehensive status monitoring and diagnostics

#######################################
# Show detailed SearXNG status
#######################################
searxng::show_status() {
    local status
    status=$(searxng::get_status)
    
    log::header "SearXNG Status Report"
    
    # Basic status
    echo "Status: $status"
    echo "Port: $SEARXNG_PORT"
    echo "Base URL: $SEARXNG_BASE_URL"
    echo "Container: $SEARXNG_CONTAINER_NAME"
    echo "Data Directory: $SEARXNG_DATA_DIR"
    echo
    
    case "$status" in
        "not_installed")
            log::info "SearXNG is not installed"
            log::info "Run: ./manage.sh --action install"
            return 1
            ;;
        "stopped")
            log::warn "SearXNG is installed but not running"
            log::info "Run: ./manage.sh --action start"
            return 1
            ;;
        "unhealthy")
            log::warn "SearXNG is running but not healthy"
            searxng::show_detailed_status
            return 1
            ;;
        "healthy")
            log::success "SearXNG is running and healthy"
            searxng::show_detailed_status
            return 0
            ;;
    esac
}

#######################################
# Show detailed status when running
#######################################
searxng::show_detailed_status() {
    # Container information
    echo "Container Details:"
    if searxng::is_installed; then
        local container_status
        container_status=$(docker container inspect --format='{{.State.Status}}' "$SEARXNG_CONTAINER_NAME" 2>/dev/null)
        echo "  Docker Status: $container_status"
        
        local started_at
        started_at=$(docker container inspect --format='{{.State.StartedAt}}' "$SEARXNG_CONTAINER_NAME" 2>/dev/null)
        echo "  Started At: $started_at"
        
        # Health check status
        local health_status
        health_status=$(docker container inspect --format='{{.State.Health.Status}}' "$SEARXNG_CONTAINER_NAME" 2>/dev/null)
        if [[ -n "$health_status" ]]; then
            echo "  Health Status: $health_status"
        fi
        
        # Image information
        local image
        image=$(docker container inspect --format='{{.Config.Image}}' "$SEARXNG_CONTAINER_NAME" 2>/dev/null)
        echo "  Image: $image"
    fi
    echo
    
    # Network connectivity
    echo "Network Status:"
    if resources::is_service_running "$SEARXNG_PORT"; then
        echo "  Port $SEARXNG_PORT: ✅ Open"
    else
        echo "  Port $SEARXNG_PORT: ❌ Not responding"
    fi
    
    # API endpoint checks
    searxng::check_api_endpoints
    
    # Resource usage
    if searxng::is_running; then
        echo
        searxng::get_resource_usage
    fi
    
    # Configuration summary
    echo
    echo "Configuration:"
    echo "  Bind Address: $SEARXNG_BIND_ADDRESS"
    echo "  Default Engines: $SEARXNG_DEFAULT_ENGINES"
    echo "  Safe Search: $SEARXNG_SAFE_SEARCH"
    echo "  Rate Limiting: $SEARXNG_LIMITER_ENABLED ($SEARXNG_RATE_LIMIT/min)"
    echo "  Redis Caching: $SEARXNG_ENABLE_REDIS"
    echo "  Public Access: $SEARXNG_ENABLE_PUBLIC_ACCESS"
}

#######################################
# Check API endpoints health
#######################################
searxng::check_api_endpoints() {
    if ! searxng::is_running; then
        echo "  API Endpoints: ❌ Container not running"
        return 1
    fi
    
    echo "API Endpoints:"
    
    # Health endpoint
    local health_url="${SEARXNG_BASE_URL}/stats"
    if curl -sf "$health_url" >/dev/null 2>&1; then
        echo "  /stats: ✅ Responding"
    else
        echo "  /stats: ❌ Not responding"
    fi
    
    # Search endpoint
    local search_url="${SEARXNG_BASE_URL}/search?q=test&format=json"
    if curl -sf "$search_url" >/dev/null 2>&1; then
        echo "  /search: ✅ Responding"
    else
        echo "  /search: ❌ Not responding"
    fi
    
    # Configuration endpoint
    local config_url="${SEARXNG_BASE_URL}/config"
    if curl -sf "$config_url" >/dev/null 2>&1; then
        echo "  /config: ✅ Responding"
    else
        echo "  /config: ❌ Not responding"
    fi
}

#######################################
# Run comprehensive diagnostics
#######################################
searxng::diagnose() {
    log::header "SearXNG Diagnostic Report"
    
    # Basic system checks
    echo "System Requirements:"
    echo -n "  Docker: "
    if command -v docker >/dev/null 2>&1; then
        echo "✅ Installed"
        echo -n "  Docker Daemon: "
        if docker info >/dev/null 2>&1; then
            echo "✅ Running"
        else
            echo "❌ Not accessible"
        fi
    else
        echo "❌ Not found"
    fi
    
    echo -n "  curl: "
    if command -v curl >/dev/null 2>&1; then
        echo "✅ Available"
    else
        echo "❌ Not found"
    fi
    echo
    
    # Configuration validation
    echo "Configuration Validation:"
    echo "  Port: $SEARXNG_PORT"
    if ports::validate_port "$SEARXNG_PORT"; then
        echo "    ✅ Valid port number"
    else
        echo "    ❌ Invalid port number"
    fi
    
    if resources::is_service_running "$SEARXNG_PORT" && ! searxng::is_running; then
        echo "    ❌ Port conflict detected"
    else
        echo "    ✅ No port conflicts"
    fi
    
    echo "  Secret Key: ${#SEARXNG_SECRET_KEY} characters"
    if [[ ${#SEARXNG_SECRET_KEY} -ge 32 ]]; then
        echo "    ✅ Adequate length"
    else
        echo "    ❌ Too short (minimum 32 characters)"
    fi
    
    echo "  Data Directory: $SEARXNG_DATA_DIR"
    if [[ -d "$SEARXNG_DATA_DIR" ]]; then
        echo "    ✅ Exists"
        if [[ -r "$SEARXNG_DATA_DIR" ]] && [[ -w "$SEARXNG_DATA_DIR" ]]; then
            echo "    ✅ Readable and writable"
        else
            echo "    ❌ Permission issues"
        fi
    else
        echo "    ❌ Does not exist"
    fi
    echo
    
    # Docker environment
    echo "Docker Environment:"
    if docker network inspect "$SEARXNG_NETWORK_NAME" >/dev/null 2>&1; then
        echo "  Network: ✅ $SEARXNG_NETWORK_NAME exists"
    else
        echo "  Network: ❌ $SEARXNG_NETWORK_NAME not found"
    fi
    
    if docker image inspect "$SEARXNG_IMAGE" >/dev/null 2>&1; then
        echo "  Image: ✅ $SEARXNG_IMAGE available"
    else
        echo "  Image: ❌ $SEARXNG_IMAGE not found"
    fi
    echo
    
    # Container status
    searxng::show_status
    echo
    
    # Log analysis
    if searxng::is_installed; then
        echo "Recent Log Analysis:"
        searxng::analyze_logs
    fi
    
    # Troubleshooting suggestions
    echo
    log::header "Troubleshooting Suggestions"
    searxng::show_troubleshooting
}

#######################################
# Analyze recent logs for issues
#######################################
searxng::analyze_logs() {
    if ! searxng::is_installed; then
        echo "  Container not available for log analysis"
        return 1
    fi
    
    local logs
    logs=$(docker logs --tail 50 "$SEARXNG_CONTAINER_NAME" 2>&1)
    
    # Check for common error patterns
    local error_count
    error_count=$(echo "$logs" | grep -ci "error\|exception\|failed\|traceback" || true)
    
    local warning_count
    warning_count=$(echo "$logs" | grep -ci "warning\|warn" || true)
    
    echo "  Error messages: $error_count"
    echo "  Warning messages: $warning_count"
    
    # Show recent errors if any
    if [[ $error_count -gt 0 ]]; then
        echo "  Recent errors:"
        echo "$logs" | grep -i "error\|exception\|failed" | tail -3 | sed 's/^/    /'
    fi
    
    # Check for specific known issues
    if echo "$logs" | grep -qi "permission denied"; then
        echo "  ⚠️  Permission issues detected"
    fi
    
    if echo "$logs" | grep -qi "address already in use"; then
        echo "  ⚠️  Port conflict detected"
    fi
    
    if echo "$logs" | grep -qi "connection refused"; then
        echo "  ⚠️  Connection issues detected"
    fi
}

#######################################
# Show troubleshooting suggestions
#######################################
searxng::show_troubleshooting() {
    local status
    status=$(searxng::get_status)
    
    case "$status" in
        "not_installed")
            echo "• Install SearXNG: ./manage.sh --action install"
            echo "• Check Docker is installed and running"
            ;;
        "stopped")
            echo "• Start SearXNG: ./manage.sh --action start"
            searxng::message "info" "MSG_SEARXNG_TROUBLESHOOT_LOGS"
            ;;
        "unhealthy")
            echo "• Check logs: ./manage.sh --action logs"
            echo "• Restart service: ./manage.sh --action restart"
            searxng::message "info" "MSG_SEARXNG_TROUBLESHOOT_PORT"
            searxng::message "info" "MSG_SEARXNG_TROUBLESHOOT_CONFIG"
            ;;
    esac
    
    echo
    echo "General troubleshooting:"
    searxng::message "info" "MSG_SEARXNG_TROUBLESHOOT_LOGS"
    searxng::message "info" "MSG_SEARXNG_TROUBLESHOOT_CONFIG"
    searxng::message "info" "MSG_SEARXNG_TROUBLESHOOT_PORT"
    searxng::message "info" "MSG_SEARXNG_TROUBLESHOOT_RESTART"
}

#######################################
# Monitor SearXNG status continuously
# Arguments:
#   $1 - interval in seconds (default: 30)
#######################################
searxng::monitor() {
    local interval="${1:-30}"
    
    log::info "Starting SearXNG monitoring (interval: ${interval}s)"
    log::info "Press Ctrl+C to stop"
    echo
    
    while true; do
        local timestamp
        timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        
        local status
        status=$(searxng::get_status)
        
        printf "[%s] Status: %s" "$timestamp" "$status"
        
        if [[ "$status" == "healthy" ]]; then
            # Check API response time
            local response_time
            response_time=$(curl -sf -w "@%{time_total}" -o /dev/null "${SEARXNG_BASE_URL}/stats" 2>/dev/null || echo "timeout")
            printf " | Response: %ss" "$response_time"
        fi
        
        echo
        sleep "$interval"
    done
}