#!/usr/bin/env bash
# SearXNG Status Management - Standardized Format
# Functions for checking and displaying SearXNG status information

# Source format utilities and config
SEARXNG_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SEARXNG_STATUS_DIR}/../../../lib/status-args.sh"
# shellcheck disable=SC1091
source "${SEARXNG_STATUS_DIR}/../../../../lib/utils/format.sh"
# shellcheck disable=SC1091
source "${SEARXNG_STATUS_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SEARXNG_STATUS_DIR}/../config/messages.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SEARXNG_STATUS_DIR}/common.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v searxng::export_config &>/dev/null; then
    searxng::export_config 2>/dev/null || true
fi

#######################################
# Collect SearXNG status data in format-agnostic structure
# Returns: Key-value pairs ready for formatting
#######################################
searxng::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    if searxng::is_installed; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "${SEARXNG_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
        
        if searxng::is_running; then
            running="true"
            
            if searxng::is_healthy; then
                healthy="true"
                health_message="Healthy - Search service ready"
            else
                health_message="Unhealthy - Service not responding"
            fi
        else
            health_message="Stopped - Container not running"
        fi
    else
        health_message="Not installed - Container does not exist"
    fi
    
    # Basic resource information
    status_data+=("name" "searxng")
    status_data+=("category" "search")
    status_data+=("description" "Privacy-respecting metasearch engine")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$SEARXNG_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$SEARXNG_PORT")
    
    # Service endpoints
    status_data+=("ui_url" "$SEARXNG_BASE_URL")
    status_data+=("api_url" "${SEARXNG_BASE_URL}/search")
    status_data+=("stats_url" "${SEARXNG_BASE_URL}/stats")
    status_data+=("config_url" "${SEARXNG_BASE_URL}/config")
    
    # Configuration details
    status_data+=("image" "$SEARXNG_IMAGE")
    status_data+=("data_dir" "$SEARXNG_DATA_DIR")
    status_data+=("bind_address" "$SEARXNG_BIND_ADDRESS")
    status_data+=("default_engines" "$SEARXNG_DEFAULT_ENGINES")
    status_data+=("safe_search" "$SEARXNG_SAFE_SEARCH")
    status_data+=("rate_limiting" "$SEARXNG_LIMITER_ENABLED")
    status_data+=("rate_limit" "$SEARXNG_RATE_LIMIT")
    status_data+=("redis_caching" "$SEARXNG_ENABLE_REDIS")
    status_data+=("public_access" "$SEARXNG_ENABLE_PUBLIC_ACCESS")
    
    # Runtime information (only if running and healthy)
    if [[ "$running" == "true" && "$healthy" == "true" ]]; then
        # Skip expensive operations in fast mode
        local skip_expensive_ops="$fast_mode"
        
        if [[ "$skip_expensive_ops" == "false" ]]; then
            # Get search engine stats if available (with optimized timeout)
            local stats_data
            stats_data=$(timeout 2s curl -sf --max-time 2 "${SEARXNG_BASE_URL}/stats" 2>/dev/null || echo "{}")
            
            if [[ -n "$stats_data" ]] && [[ "$stats_data" != "{}" ]]; then
                # Extract engine count from stats page (simple approach)
                local engine_count
                engine_count=$(echo "$stats_data" | grep -o "'[^']*'" | wc -l 2>/dev/null || echo "unknown")
                status_data+=("active_engines" "$engine_count")
            fi
            
            # Get version if available (with optimized execution)
            local version
            version=$(timeout 2s docker exec "$SEARXNG_CONTAINER_NAME" python -c "import searx; print(searx.__version__)" 2>/dev/null || echo "unknown")
            if [[ -n "$version" && "$version" != "unknown" ]]; then
                status_data+=("version" "$version")
            fi
        fi
        
        # Container uptime (fast operation, always include)
        local started_at
        started_at=$(docker inspect --format='{{.State.StartedAt}}' "$SEARXNG_CONTAINER_NAME" 2>/dev/null)
        if [[ -n "$started_at" ]]; then
            status_data+=("started_at" "$started_at")
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show SearXNG status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
searxng::status::show() {
    local format="text"
    local verbose="false"
    local fast="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            --verbose|-v)
                verbose="true"
                shift
                ;;
            --fast)
                fast="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Collect status data (pass fast flag if set)
    local data_string
    local collect_args=""
    if [[ "$fast" == "true" ]]; then
        collect_args="--fast"
    fi
    data_string=$(searxng::status::collect_data $collect_args 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        # Fallback if data collection fails
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect SearXNG status data"
        fi
        return 1
    fi
    
    # Convert string to array
    local data_array
    mapfile -t data_array <<< "$data_string"
    
    # Output based on format
    if [[ "$format" == "json" ]]; then
        format::output "json" "kv" "${data_array[@]}"
    else
        # Text format with standardized structure
        searxng::status::display_text "${data_array[@]}"
    fi
    
    # Return appropriate exit code
    local healthy="false"
    local running="false"
    for ((i=0; i<${#data_array[@]}; i+=2)); do
        case "${data_array[i]}" in
            "healthy") healthy="${data_array[i+1]}" ;;
            "running") running="${data_array[i+1]}" ;;
        esac
    done
    
    if [[ "$healthy" == "true" ]]; then
        return 0
    elif [[ "$running" == "true" ]]; then
        return 1
    else
        return 2
    fi
}

#######################################
# Display status in text format
#######################################
searxng::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "🔍 SearXNG Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install SearXNG, run: ./manage.sh --action install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Container information
    log::info "🐳 Container Info:"
    log::info "   📦 Name: ${data[container_name]:-unknown}"
    log::info "   📊 Status: ${data[container_status]:-unknown}"
    log::info "   🖼️  Image: ${data[image]:-unknown}"
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🔍 UI: ${data[ui_url]:-unknown}"
    log::info "   🔌 Search API: ${data[api_url]:-unknown}"
    log::info "   📊 Stats: ${data[stats_url]:-unknown}"
    log::info "   ⚙️  Config: ${data[config_url]:-unknown}"
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   🌐 Bind Address: ${data[bind_address]:-unknown}"
    log::info "   🔍 Default Engines: ${data[default_engines]:-unknown}"
    log::info "   🛡️  Safe Search: ${data[safe_search]:-unknown}"
    log::info "   🚦 Rate Limiting: ${data[rate_limiting]:-unknown} (${data[rate_limit]:-unknown}/min)"
    log::info "   💾 Redis Caching: ${data[redis_caching]:-unknown}"
    log::info "   🌍 Public Access: ${data[public_access]:-unknown}"
    echo
    
    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "📈 Runtime Information:"
        if [[ -n "${data[version]:-}" ]]; then
            log::info "   🔖 Version: ${data[version]}"
        fi
        if [[ -n "${data[active_engines]:-}" ]]; then
            log::info "   🔍 Active Engines: ${data[active_engines]}"
        fi
        if [[ -n "${data[started_at]:-}" ]]; then
            log::info "   ⏱️  Started: ${data[started_at]}"
        fi
        echo
        
        # Quick access info
        log::info "🎯 Quick Actions:"
        log::info "   🌐 Access UI: ${data[ui_url]:-http://localhost:9200}"
        log::info "   🔍 Search: ${data[api_url]:-http://localhost:9200/search}?q=test"
        log::info "   📊 View Stats: ${data[stats_url]:-http://localhost:9200/stats}"
        log::info "   📄 View logs: ./manage.sh --action logs"
        log::info "   🛑 Stop service: ./manage.sh --action stop"
    fi
}

#######################################
# Main status function for CLI registration
#######################################
searxng::status() {
    status::run_standard "searxng" "searxng::status::collect_data" "searxng::status::display_text" "$@"
}

#######################################
# Legacy function compatibility - Show detailed SearXNG status
#######################################
searxng::show_status() {
    searxng::status::show "$@"
}

#######################################
# Legacy function compatibility - Show detailed status when running
#######################################
searxng::show_detailed_status() {
    # Use new standardized text display
    searxng::status::display_text
}

#######################################
# Legacy function compatibility - Check API endpoints health
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
    
    # Enhanced health check with permission diagnostics
    searxng::health_check_detailed
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
    
    # Check for permission issues and provide specific guidance
    local current_uid current_gid
    current_uid=$(id -u)
    current_gid=$(id -g)
    
    if [[ -d "$SEARXNG_DATA_DIR" ]]; then
        local incorrect_files
        incorrect_files=$(find "$SEARXNG_DATA_DIR" -not -user "$current_uid" -o -not -group "$current_gid" 2>/dev/null || true)
        
        if [[ -n "$incorrect_files" ]]; then
            echo "• Fix file permissions: sudo chown -R \$(whoami):\$(whoami) $SEARXNG_DATA_DIR"
            echo "• Or try: ./manage.sh --action reset-config"
        fi
    fi
    
    # Check for Docker permission issues
    if ! docker info >/dev/null 2>&1; then
        echo "• Fix Docker permissions: sudo usermod -aG docker \$USER && newgrp docker"
        echo "• Or restart Docker service: sudo systemctl restart docker"
    fi
    
    echo
    echo "General troubleshooting:"
    searxng::message "info" "MSG_SEARXNG_TROUBLESHOOT_LOGS"
    searxng::message "info" "MSG_SEARXNG_TROUBLESHOOT_CONFIG"
    searxng::message "info" "MSG_SEARXNG_TROUBLESHOOT_PORT"
    searxng::message "info" "MSG_SEARXNG_TROUBLESHOOT_RESTART"
    
    echo
    echo "Permission troubleshooting:"
    echo "• If container fails to start with permission errors:"
    echo "  - Check data directory ownership: ls -la $SEARXNG_DATA_DIR"
    echo "  - Fix permissions: ./manage.sh --action reset-config"
    echo "  - Clean install: ./manage.sh --action uninstall && ./manage.sh --action install"
}

#######################################
# Legacy function compatibility - Monitor SearXNG status continuously
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

#######################################
# Legacy function compatibility - Get resource usage information
#######################################
searxng::get_resource_usage() {
    if ! searxng::is_running; then
        log::info "Resource usage unavailable - container not running"
        return 1
    fi
    
    log::info "Resource Usage:"
    
    # Get container stats
    local stats
    stats=$(timeout 2s docker stats --no-stream --format "{{.CPUPerc}};{{.MemUsage}}" "$SEARXNG_CONTAINER_NAME" 2>/dev/null || echo "N/A;N/A")
    
    if [[ "$stats" != "N/A;N/A" ]]; then
        local cpu_usage memory_usage
        cpu_usage=$(echo "$stats" | cut -d';' -f1)
        memory_usage=$(echo "$stats" | cut -d';' -f2)
        log::info "  CPU Usage: $cpu_usage"
        log::info "  Memory Usage: $memory_usage"
    else
        log::info "  Unable to retrieve resource usage"
    fi
}