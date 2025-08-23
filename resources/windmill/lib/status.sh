#!/usr/bin/env bash
# Windmill Status Management - Standardized Format
# Functions for checking and displaying Windmill status information

# Source format utilities and config
WINDMILL_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${WINDMILL_STATUS_DIR}/../../../../lib/utils/format.sh"
# shellcheck disable=SC1091
source "${WINDMILL_STATUS_DIR}/../../../lib/status-args.sh"
# shellcheck disable=SC1091
source "${WINDMILL_STATUS_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WINDMILL_STATUS_DIR}/../config/messages.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WINDMILL_STATUS_DIR}/common.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WINDMILL_STATUS_DIR}/docker.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v windmill::export_config &>/dev/null; then
    windmill::export_config 2>/dev/null || true
fi
if command -v windmill::messages::init &>/dev/null; then
    windmill::messages::init 2>/dev/null || true
fi

#######################################
# Collect Windmill status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
windmill::status::collect_data() {
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
    local service_status="unknown"
    
    # Check if Windmill is installed (using docker-compose)
    if windmill::is_installed; then
        installed="true"
        service_status=$(windmill::get_service_status 2>/dev/null || echo "unknown")
        
        case "$service_status" in
            "healthy")
                running="true"
                healthy="true"
                health_message="Healthy - All services operational"
                container_status="running"
                ;;
            "unhealthy")
                running="true"
                healthy="false"
                health_message="Unhealthy - Services running but not responding properly"
                container_status="unhealthy"
                ;;
            "stopped")
                running="false"
                healthy="false"
                health_message="Stopped - Services not running"
                container_status="stopped"
                ;;
            *)
                running="false"
                healthy="false"
                health_message="Unknown - Unable to determine service status"
                container_status="unknown"
                ;;
        esac
    else
        health_message="Not installed - Use ./manage.sh --action install"
    fi
    
    # Basic resource information
    status_data+=("name" "windmill")
    status_data+=("category" "automation")
    status_data+=("description" "Open-source workflow engine and internal tool builder")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_status" "$container_status")
    status_data+=("service_status" "$service_status")
    status_data+=("port" "$WINDMILL_SERVER_PORT")
    
    # Service endpoints
    status_data+=("base_url" "$WINDMILL_BASE_URL")
    status_data+=("api_url" "$WINDMILL_BASE_URL/api")
    status_data+=("admin_email" "$WINDMILL_SUPERADMIN_EMAIL")
    
    # Configuration details
    status_data+=("project_name" "$WINDMILL_PROJECT_NAME")
    status_data+=("data_dir" "$WINDMILL_DATA_DIR")
    status_data+=("worker_replicas" "$WINDMILL_WORKER_REPLICAS")
    status_data+=("db_type" "$WINDMILL_DB_TYPE")
    status_data+=("db_external" "$WINDMILL_DB_EXTERNAL")
    
    # Container names
    status_data+=("server_container" "$WINDMILL_SERVER_CONTAINER")
    status_data+=("worker_container" "$WINDMILL_WORKER_CONTAINER")
    status_data+=("db_container" "$WINDMILL_DB_CONTAINER_NAME")
    
    # Runtime information (only if running and healthy)
    if [[ "$running" == "true" ]]; then
        # Check individual service containers
        local server_running="false"
        local worker_running="false"
        local db_running="false"
        
        if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${WINDMILL_SERVER_CONTAINER}$"; then
            server_running="true"
        fi
        
        if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "${WINDMILL_WORKER_CONTAINER}"; then
            worker_running="true"
        fi
        
        if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${WINDMILL_DB_CONTAINER_NAME}$"; then
            db_running="true"
        fi
        
        status_data+=("server_running" "$server_running")
        status_data+=("worker_running" "$worker_running")
        status_data+=("database_running" "$db_running")
        
        # Get container stats if available (optimized with smart skipping)
        if [[ "$server_running" == "true" ]]; then
            # Skip expensive operations in fast mode
            local skip_stats="$fast_mode"
            
            if [[ "$skip_stats" == "true" ]]; then
                status_data+=("server_cpu" "N/A")
                status_data+=("server_memory" "N/A")
            else
                local server_stats
                server_stats=$(timeout 2s docker stats --no-stream --format "{{.CPUPerc}}|{{.MemUsage}}" "$WINDMILL_SERVER_CONTAINER" 2>/dev/null || echo "")
                if [[ -n "$server_stats" ]]; then
                    local cpu_usage memory_usage
                    cpu_usage=$(echo "$server_stats" | cut -d'|' -f1)
                    memory_usage=$(echo "$server_stats" | cut -d'|' -f2)
                    status_data+=("server_cpu" "$cpu_usage")
                    status_data+=("server_memory" "$memory_usage")
                else
                    status_data+=("server_cpu" "N/A")
                    status_data+=("server_memory" "N/A")
                fi
            fi
        fi
        
        # Worker count
        local worker_count
        worker_count=$(docker ps --filter "name=${WINDMILL_WORKER_CONTAINER}" --format "{{.Names}}" 2>/dev/null | wc -l)
        status_data+=("active_workers" "$worker_count")
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show Windmill status using standardized format
# Args: [--format json|text] [--verbose] [--fast]
#######################################
windmill::status() {
    # Use the comprehensive standard wrapper
    status::run_standard "windmill" "windmill::status::collect_data" "windmill::status::display_text" "$@"
}

#######################################
# Display status in text format
#######################################
windmill::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "🌪️  Windmill Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Windmill, run: ./manage.sh --action install"
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
    
    # Service components
    log::info "🔧 Service Components:"
    log::info "   📦 Project: ${data[project_name]:-unknown}"
    if [[ "${data[running]:-false}" == "true" ]]; then
        local server_status="${data[server_running]:-false}"
        local worker_status="${data[worker_running]:-false}"
        local db_status="${data[database_running]:-false}"
        
        if [[ "$server_status" == "true" ]]; then
            log::success "   ✅ Server: Running"
        else
            log::error "   ❌ Server: Not running"
        fi
        
        if [[ "$worker_status" == "true" ]]; then
            log::success "   ✅ Workers: ${data[active_workers]:-0} active"
        else
            log::error "   ❌ Workers: Not running"
        fi
        
        if [[ "$db_status" == "true" ]]; then
            log::success "   ✅ Database: Running (${data[db_type]:-unknown})"
        else
            log::error "   ❌ Database: Not running"
        fi
    else
        log::warn "   ⚠️  Server: Stopped"
        log::warn "   ⚠️  Workers: Stopped"
        log::warn "   ⚠️  Database: Stopped"
    fi
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🎨 Web UI: ${data[base_url]:-unknown}"
    log::info "   🔌 API: ${data[api_url]:-unknown}"
    log::info "   👤 Admin: ${data[admin_email]:-unknown}"
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   📁 Data Directory: ${data[data_dir]:-unknown}"
    log::info "   👥 Worker Replicas: ${data[worker_replicas]:-unknown}"
    log::info "   🗄️  Database Type: ${data[db_type]:-unknown}"
    log::info "   🔗 External DB: ${data[db_external]:-unknown}"
    echo
    
    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "📈 Runtime Information:"
        
        # Server performance
        local server_cpu="${data[server_cpu]:-N/A}"
        local server_memory="${data[server_memory]:-N/A}"
        if [[ "$server_cpu" != "N/A" ]]; then
            log::info "   🔥 Server CPU: $server_cpu"
            log::info "   🧠 Server Memory: $server_memory"
        fi
        
        log::info "   👷 Active Workers: ${data[active_workers]:-0}"
        log::info "   📊 Service Status: ${data[service_status]:-unknown}"
    fi
}

# Legacy function - kept for backwards compatibility
windmill::show_connection_info() {
    log::info "🌐 Connection Information:"
    echo "  Web Interface: $WINDMILL_BASE_URL"
    echo "  API Base URL: $WINDMILL_BASE_URL/api"
    echo "  Admin Email: $WINDMILL_SUPERADMIN_EMAIL"
    echo
    echo "  Quick Links:"
    echo "    Dashboard: $WINDMILL_BASE_URL"
    echo "    Workspaces: $WINDMILL_BASE_URL/workspaces"
    echo "    API Docs: $WINDMILL_BASE_URL/openapi.html"
    echo "    Workers: $WINDMILL_BASE_URL/workers"
}

#######################################
# Show detailed status of individual services
#######################################
windmill::show_detailed_service_status() {
    log::info "🔧 Service Status:"
    
    if ! windmill::is_installed; then
        echo "  Services are not installed"
        return 1
    fi
    
    # Get service information
    local services=("windmill-app" "windmill-worker" "windmill-db")
    
    # Add optional services if enabled
    if [[ "$DISABLE_NATIVE_WORKER" != "true" ]]; then
        services+=("windmill-worker-native")
    fi
    
    if [[ "$DISABLE_LSP" != "true" ]]; then
        services+=("windmill-lsp")
    fi
    
    if [[ "$ENABLE_MULTIPLAYER" == "true" ]]; then
        services+=("windmill-multiplayer")
    fi
    
    echo
    printf "  %-25s %-15s %-15s %s\n" "SERVICE" "STATUS" "HEALTH" "UPTIME"
    printf "  %-25s %-15s %-15s %s\n" "-------" "------" "------" "------"
    
    for service in "${services[@]}"; do
        local status="stopped"
        local health="unknown"
        local uptime="--"
        
        # Check if container exists and is running
        if docker ps --format "{{.Names}}" | grep -q "^${service}$"; then
            status="running"
            
            # Get uptime
            uptime=$(docker ps --format "{{.Status}}" --filter "name=^${service}$" | sed 's/Up //')
            
            # Check health based on service type
            case "$service" in
                "windmill-app")
                    if resources::check_http_health "$WINDMILL_BASE_URL" "/api/version"; then
                        health="healthy"
                    else
                        health="unhealthy"
                    fi
                    ;;
                "windmill-db")
                    if windmill::compose_cmd exec -T windmill-db pg_isready -U postgres >/dev/null 2>&1; then
                        health="healthy"
                    else
                        health="unhealthy"
                    fi
                    ;;
                *)
                    # For workers and other services, assume healthy if running
                    health="healthy"
                    ;;
            esac
        elif docker ps -a --format "{{.Names}}" | grep -q "^${service}$"; then
            status="exited"
        fi
        
        # Format status with colors
        local status_display health_display
        case "$status" in
            "running") status_display="✅ running" ;;
            "exited") status_display="❌ exited" ;;
            *) status_display="⚪ stopped" ;;
        esac
        
        case "$health" in
            "healthy") health_display="✅ healthy" ;;
            "unhealthy") health_display="❌ unhealthy" ;;
            *) health_display="⚪ unknown" ;;
        esac
        
        printf "  %-25s %-23s %-23s %s\n" "$service" "$status_display" "$health_display" "$uptime"
    done
}

#######################################
# Show resource usage information
#######################################
windmill::show_resource_usage() {
    log::info "📊 Resource Usage:"
    
    # Get container stats
    local container_ids
    container_ids=$(windmill::compose_cmd ps -q 2>/dev/null)
    
    if [[ -n "$container_ids" ]]; then
        echo
        printf "  %-25s %-12s %-20s %-12s\n" "CONTAINER" "CPU %" "MEMORY USAGE" "MEM %"
        printf "  %-25s %-12s %-20s %-12s\n" "---------" "-----" "------------" "-----"
        
        # Get stats for each container
        docker stats --no-stream --format "{{.Container}}\t{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}" $container_ids 2>/dev/null | \
        while IFS=$'\t' read -r container name cpu mem_usage mem_perc; do
            printf "  %-25s %-12s %-20s %-12s\n" "$(echo "$name" | cut -c1-24)" "$cpu" "$mem_usage" "$mem_perc"
        done
        
        echo
        
        # Show worker count and scaling info
        local worker_count
        worker_count=$(docker ps --filter "name=${WINDMILL_PROJECT_NAME}-worker" --format "{{.Names}}" | wc -l)
        echo "  Active Workers: $worker_count"
        echo "  Worker Memory Limit: $WORKER_MEMORY_LIMIT per worker"
        
        # Show volume usage
        echo
        local volumes
        volumes=$(docker volume ls --filter "name=${WINDMILL_PROJECT_NAME}" --format "{{.Name}}" 2>/dev/null)
        if [[ -n "$volumes" ]]; then
            echo "  Docker Volumes:"
            for volume in $volumes; do
                local size
                size=$(docker run --rm -v "$volume:/volume:ro" alpine:latest du -sh /volume 2>/dev/null | cut -f1 || echo "unknown")
                printf "    %-30s %s\n" "$(echo "$volume" | sed "s/${WINDMILL_PROJECT_NAME}_//")" "$size"
            done
        fi
    else
        echo "  No running containers found"
    fi
}

#######################################
# Show configuration summary
#######################################
windmill::show_configuration_summary() {
    log::info "⚙️  Configuration Summary:"
    echo "  Project Name: $WINDMILL_PROJECT_NAME"
    echo "  Server Port: $WINDMILL_SERVER_PORT"
    echo "  Worker Count: $WORKER_COUNT"
    echo "  Worker Memory: $WORKER_MEMORY_LIMIT"
    echo "  Database Type: ${WINDMILL_DB_TYPE} (${EXTERNAL_DB:-internal})"
    echo "  LSP Enabled: $([ "$DISABLE_LSP" != "true" ] && echo "yes" || echo "no")"
    echo "  Native Worker: $([ "$DISABLE_NATIVE_WORKER" != "true" ] && echo "yes" || echo "no")"
    echo "  Multiplayer: $([ "$ENABLE_MULTIPLAYER" == "true" ] && echo "yes" || echo "no")"
    echo "  Log Level: $WINDMILL_LOG_LEVEL"
    
    # Show file locations
    echo
    echo "  Configuration Files:"
    echo "    Environment: $WINDMILL_ENV_FILE"
    echo "    Compose File: $WINDMILL_COMPOSE_FILE"
    echo "    Data Directory: $WINDMILL_DATA_DIR"
    echo "    Backup Directory: $WINDMILL_BACKUP_DIR"
}

#######################################
# Show health check results
#######################################
windmill::show_health_checks() {
    log::info "🏥 Health Checks:"
    
    local checks_passed=0
    local total_checks=0
    
    # API Health Check
    total_checks=$((total_checks + 1))
    echo -n "  API Endpoint (/api/version): "
    if resources::check_http_health "$WINDMILL_BASE_URL" "/api/version"; then
        echo "✅ responsive"
        checks_passed=$((checks_passed + 1))
    else
        echo "❌ not responding"
    fi
    
    # Database Health Check
    if windmill::compose_cmd ps --services | grep -q "windmill-db"; then
        total_checks=$((total_checks + 1))
        echo -n "  Database (PostgreSQL): "
        if windmill::compose_cmd exec -T windmill-db pg_isready -U postgres >/dev/null 2>&1; then
            echo "✅ ready"
            checks_passed=$((checks_passed + 1))
        else
            echo "❌ not ready"
        fi
    fi
    
    # Port accessibility
    total_checks=$((total_checks + 1))
    echo -n "  Port $WINDMILL_SERVER_PORT: "
    if resources::is_service_running "$WINDMILL_SERVER_PORT"; then
        echo "✅ listening"
        checks_passed=$((checks_passed + 1))
    else
        echo "❌ not listening"
    fi
    
    # Worker availability
    total_checks=$((total_checks + 1))
    echo -n "  Workers: "
    local worker_count
    worker_count=$(docker ps --filter "name=${WINDMILL_PROJECT_NAME}-worker" --format "{{.Names}}" | wc -l)
    if [[ $worker_count -gt 0 ]]; then
        echo "✅ $worker_count active"
        checks_passed=$((checks_passed + 1))
    else
        echo "❌ none active"
    fi
    
    # Docker socket (for worker containers)
    if [[ "$DISABLE_NATIVE_WORKER" != "true" ]]; then
        total_checks=$((total_checks + 1))
        echo -n "  Docker Socket Access: "
        if docker ps >/dev/null 2>&1; then
            echo "✅ available"
            checks_passed=$((checks_passed + 1))
        else
            echo "❌ not available"
        fi
    fi
    
    echo
    echo "  Health Score: $checks_passed/$total_checks checks passed"
    
    if [[ $checks_passed -eq $total_checks ]]; then
        echo "  Overall Health: ✅ Excellent"
    elif [[ $checks_passed -ge $((total_checks * 3 / 4)) ]]; then
        echo "  Overall Health: ⚠️  Good (minor issues)"
    elif [[ $checks_passed -ge $((total_checks / 2)) ]]; then
        echo "  Overall Health: 🔶 Fair (significant issues)"
    else
        echo "  Overall Health: ❌ Poor (major issues)"
    fi
}

#######################################
# Show troubleshooting tips for common issues
#######################################
windmill::show_troubleshooting_tips() {
    log::info "🔧 Troubleshooting Tips:"
    
    echo "  Common Solutions:"
    echo "    • Check logs: $0 --action logs"
    echo "    • Restart services: $0 --action restart"
    echo "    • Verify port availability: sudo lsof -i :$WINDMILL_SERVER_PORT"
    echo "    • Check Docker status: docker ps -a"
    echo "    • Verify resources: docker stats"
    echo
    echo "  If problems persist:"
    echo "    • Review installation: $0 --action install --force yes"
    echo "    • Check system requirements (4GB+ RAM, 5GB+ disk)"
    echo "    • Verify Docker and Docker Compose versions"
    echo "    • Check firewall and network connectivity"
    echo
    echo "  Get Help:"
    echo "    • Windmill docs: https://docs.windmill.dev"
    echo "    • GitHub issues: https://github.com/windmill-labs/windmill/issues"
    echo "    • Discord community: https://discord.gg/V7PM2YHsPB"
}

#######################################
# Quick status check for scripting
# Outputs: healthy|unhealthy|stopped|not_installed
#######################################
windmill::quick_status() {
    windmill::get_service_status
}

#######################################
# Check if all services are healthy
# Returns: 0 if all healthy, 1 otherwise
#######################################
windmill::is_all_healthy() {
    [[ "$(windmill::get_service_status)" == "healthy" ]]
}

#######################################
# Show logs with smart formatting
# Arguments:
#   $1 - Service name (optional, default: all)
#   $2 - Follow flag (optional, default: false)
#######################################
windmill::logs() {
    local service="${1:-all}"
    local follow="${2:-false}"
    
    if ! windmill::is_installed; then
        log::error "Windmill is not installed"
        return 1
    fi
    
    log::info "📋 Windmill Logs - Service: $service"
    
    if [[ "$follow" == "true" ]]; then
        log::info "Following logs (Press Ctrl+C to stop)..."
    fi
    
    echo
    windmill::show_logs "$service" "$follow"
}

#######################################
# Show information about Windmill
#######################################
windmill::info() {
    cat << EOF
=== Windmill Workflow Automation Platform ===

Overview:
Windmill is a developer-centric workflow automation platform that allows you to build 
workflows and UIs using code (TypeScript, Python, Go, Bash) instead of drag-and-drop.

Key Features:
• Code-first approach with multi-language support
• Built-in web IDE with autocomplete and debugging
• Scalable worker architecture for high-performance execution
• REST API for integration and automation
• Webhook triggers and scheduled executions
• Secret management and resource sharing
• Real-time collaboration and version control

Architecture:
• Windmill Server: Web interface + API (Port: $WINDMILL_SERVER_PORT)
• PostgreSQL Database: Persistent storage
• Worker Containers: Script execution (Current: $WORKER_COUNT workers)
• Language Server: IDE features and autocomplete
• Native Worker: System command execution

Installation Status:
$(if windmill::is_installed; then
    echo "✅ Installed at: $WINDMILL_BASE_URL"
    echo "Configuration: $WINDMILL_ENV_FILE"
else
    echo "❌ Not installed"
    echo "Install with: $0 --action install"
fi)

Management Commands:
• Install: $0 --action install
• Start: $0 --action start
• Stop: $0 --action stop
• Status: $0 --action status
• Logs: $0 --action logs
• Scale workers: $0 --action scale-workers <count>
• Backup: $0 --action backup
• Uninstall: $0 --action uninstall

Learn More:
• Documentation: https://docs.windmill.dev
• GitHub: https://github.com/windmill-labs/windmill
• Discord: https://discord.gg/V7PM2YHsPB

EOF
}