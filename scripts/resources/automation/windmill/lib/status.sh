#!/usr/bin/env bash
# Windmill Status and Health Check Functions
# Comprehensive status reporting for Windmill services

#######################################
# Show comprehensive Windmill status
# Returns: 0 if healthy, 1 if issues found
#######################################
windmill::status() {
    log::header "🌪️  Windmill Status Report"
    
    local overall_status="healthy"
    local issues=()
    
    # Basic installation check
    if ! windmill::is_installed; then
        echo "❌ Status: Not Installed"
        echo
        echo "To install Windmill:"
        echo "  $0 --action install"
        return 1
    fi
    
    # Service status overview
    local service_status
    service_status=$(windmill::get_service_status)
    
    case "$service_status" in
        "healthy")
            echo "✅ Status: Running and Healthy"
            ;;
        "unhealthy")
            echo "⚠️  Status: Running but Unhealthy"
            overall_status="unhealthy"
            issues+=("Services are running but not responding properly")
            ;;
        "stopped")
            echo "⏹️  Status: Stopped"
            overall_status="stopped"
            issues+=("Services are not running")
            ;;
        *)
            echo "❓ Status: Unknown"
            overall_status="unknown"
            issues+=("Unable to determine service status")
            ;;
    esac
    
    echo
    
    # Connection information
    if [[ "$service_status" == "healthy" ]]; then
        windmill::show_connection_info
    fi
    
    # Detailed service status
    windmill::show_detailed_service_status
    
    # Resource usage
    if windmill::is_running; then
        echo
        windmill::show_resource_usage
    fi
    
    # Configuration summary
    echo
    windmill::show_configuration_summary
    
    # Health checks
    echo
    windmill::show_health_checks
    
    # Show issues if any
    if [[ ${#issues[@]} -gt 0 ]]; then
        echo
        log::error "Issues found:"
        for issue in "${issues[@]}"; do
            log::error "  • $issue"
        done
        
        echo
        windmill::show_troubleshooting_tips
    fi
    
    # Return appropriate exit code
    case "$overall_status" in
        "healthy") return 0 ;;
        *) return 1 ;;
    esac
}

#######################################
# Show connection information
#######################################
windmill::show_connection_info() {
    log::info "🌐 Connection Information:"
    echo "  Web Interface: $WINDMILL_BASE_URL"
    echo "  API Base URL: $WINDMILL_BASE_URL/api"
    echo "  Admin Email: $SUPERADMIN_EMAIL"
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
    local services=("windmill-server" "windmill-worker" "windmill-db")
    
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
                "windmill-server")
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