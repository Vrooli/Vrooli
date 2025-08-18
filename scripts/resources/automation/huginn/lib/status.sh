#!/usr/bin/env bash
# Huginn Status and Health Monitoring Functions - Standardized Format
# System status, health checks, and information display

# Source format utilities and config
HUGINN_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${HUGINN_STATUS_DIR}/../../../../lib/utils/format.sh"
# shellcheck disable=SC1091
source "${HUGINN_STATUS_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${HUGINN_STATUS_DIR}/common.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v huginn::export_config &>/dev/null; then
    huginn::export_config 2>/dev/null || true
fi

#######################################
# Collect Huginn status data in format-agnostic structure
# Returns: Key-value pairs ready for formatting
#######################################
huginn::status::collect_data() {
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    # Check main container (Huginn can run standalone without separate DB container)
    if huginn::container_exists; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "$CONTAINER_NAME" 2>/dev/null || echo "unknown")
        
        if huginn::is_running; then
            running="true"
            
            if huginn::is_healthy; then
                healthy="true"
                health_message="Healthy - All systems operational - agents ready"
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
    status_data+=("name" "huginn")
    status_data+=("category" "automation")
    status_data+=("description" "Agent-based workflow automation and monitoring platform")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$HUGINN_PORT")
    
    # Service endpoints
    status_data+=("ui_url" "$HUGINN_BASE_URL")
    status_data+=("admin_username" "$DEFAULT_ADMIN_USERNAME")
    status_data+=("admin_password" "$DEFAULT_ADMIN_PASSWORD")
    status_data+=("admin_email" "$DEFAULT_ADMIN_EMAIL")
    
    # Configuration details
    if [[ "$running" == "true" ]]; then
        # Container image
        local image
        image=$(docker inspect --format='{{.Config.Image}}' "$CONTAINER_NAME" 2>/dev/null || echo "unknown")
        status_data+=("image" "$image")
        
        # Huginn version
        local version
        version=$(huginn::get_version 2>/dev/null || echo "unknown")
        status_data+=("version" "$version")
        
        # Database connectivity
        local db_connected="false"
        if huginn::check_database 2>/dev/null; then
            db_connected="true"
        fi
        status_data+=("database_connected" "$db_connected")
        
        # Get system statistics if healthy
        if [[ "$healthy" == "true" ]]; then
            local stats_json
            stats_json=$(huginn::get_system_stats 2>/dev/null)
            
            if [[ -n "$stats_json" ]] && echo "$stats_json" | jq . >/dev/null 2>&1; then
                local users agents scenarios events active_agents
                users=$(echo "$stats_json" | jq -r '.users // 0')
                agents=$(echo "$stats_json" | jq -r '.agents // 0')
                scenarios=$(echo "$stats_json" | jq -r '.scenarios // 0')
                events=$(echo "$stats_json" | jq -r '.events // 0')
                active_agents=$(echo "$stats_json" | jq -r '.active_agents // 0')
                
                status_data+=("users" "$users")
                status_data+=("agents" "$agents")
                status_data+=("scenarios" "$scenarios")
                status_data+=("events" "$events")
                status_data+=("active_agents" "$active_agents")
                
                # Calculate health score
                local health_score=100
                if [[ $agents -gt 0 ]]; then
                    local active_percentage=$((active_agents * 100 / agents))
                    if [[ $active_percentage -lt 50 ]]; then
                        health_score=75
                    elif [[ $active_percentage -lt 25 ]]; then
                        health_score=50
                    fi
                fi
                status_data+=("health_score" "$health_score")
            fi
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show Huginn status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
huginn::status() {
    local format="text"
    local verbose="false"
    
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
            *)
                shift
                ;;
        esac
    done
    
    # Collect status data
    local data_string
    data_string=$(huginn::status::collect_data 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        # Fallback if data collection fails
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect Huginn status data"
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
        huginn::status::display_text "${data_array[@]}"
    fi
    
    # Return appropriate exit code
    local healthy="false"
    local running="false"
    local installed="false"
    for ((i=0; i<${#data_array[@]}; i+=2)); do
        case "${data_array[i]}" in
            "healthy") healthy="${data_array[i+1]}" ;;
            "running") running="${data_array[i+1]}" ;;
            "installed") installed="${data_array[i+1]}" ;;
        esac
    done
    
    if [[ "$healthy" == "true" ]]; then
        return 0
    elif [[ "$running" == "true" ]]; then
        return 1
    elif [[ "$installed" == "true" ]]; then
        return 2
    else
        return 3
    fi
}

#######################################
# Display status in text format
#######################################
huginn::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "🤖 Huginn Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Huginn, run: ./manage.sh --action install"
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
    if [[ -n "${data[image]:-}" ]]; then
        log::info "   🖼️  Image: ${data[image]}"
    fi
    if [[ -n "${data[version]:-}" ]]; then
        log::info "   🔖 Version: ${data[version]}"
    fi
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🎨 UI: ${data[ui_url]:-unknown}"
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::info "   👤 Username: ${data[admin_username]:-unknown}"
        log::info "   🔑 Password: ${data[admin_password]:-unknown}"
    fi
    echo
    
    # Database status
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::info "🗄️  Database:"
        if [[ "${data[database_connected]:-false}" == "true" ]]; then
            log::success "   ✅ Connection: Working"
        else
            log::error "   ❌ Connection: Failed"
        fi
        echo
    fi
    
    # Statistics (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "📈 Statistics:"
        log::info "   👥 Users: ${data[users]:-0}"
        log::info "   🤖 Agents: ${data[agents]:-0} (Active: ${data[active_agents]:-0})"
        log::info "   📂 Scenarios: ${data[scenarios]:-0}"
        log::info "   📊 Events: ${data[events]:-0}"
        
        local health_score="${data[health_score]:-100}"
        if [[ $health_score -ge 90 ]]; then
            log::success "   💚 Health Score: $health_score% (Excellent)"
        elif [[ $health_score -ge 75 ]]; then
            log::info "   💛 Health Score: $health_score% (Good)"
        else
            log::warn "   🧡 Health Score: $health_score% (Needs Attention)"
        fi
    fi
}

#######################################
# Legacy function for backward compatibility
# Returns: 0 always
#######################################
huginn::show_status() {
    huginn::status "$@"
}

#######################################
# Show basic installation and running status
# Returns: 0 always
#######################################
huginn::show_basic_status() {
    log::info "📊 Basic Status:"
    
    # Installation status
    if huginn::is_installed; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        huginn::show_not_installed
        return 0
    fi
    
    # Running status
    if huginn::is_running; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
        huginn::show_not_running
    fi
    
    # Health status
    if huginn::is_healthy; then
        log::success "   ✅ Healthy: Yes"
    else
        log::warn "   ⚠️  Healthy: No"
    fi
    
    # Web interface
    log::info "   🌐 Web Interface: $HUGINN_BASE_URL"
    
    if huginn::is_running; then
        log::info "   👤 Username: $DEFAULT_ADMIN_USERNAME"
        log::info "   🔑 Password: $DEFAULT_ADMIN_PASSWORD"
    fi
}

#######################################
# Show detailed system information
# Returns: 0 always
#######################################
huginn::show_system_info() {
    if ! huginn::is_running; then
        log::info "📋 System Info: (Service not running)"
        return 0
    fi
    
    log::info "📋 System Information:"
    
    # Version information
    local version
    version=$(huginn::get_version)
    log::info "   📦 Version: $version"
    
    # Database status
    if huginn::check_database; then
        log::success "   ✅ Database: Connected"
    else
        log::error "   ❌ Database: Connection failed"
    fi
    
    # Container information
    huginn::show_container_info
    
    # Resource usage
    huginn::show_resource_summary
}

#######################################
# Show container information
# Returns: 0 always
#######################################
huginn::show_container_info() {
    log::info "   🐳 Containers:"
    
    # Huginn container
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        local status="Running"
        local uptime
        uptime=$(docker inspect --format='{{.State.StartedAt}}' "$CONTAINER_NAME" 2>/dev/null)
        if [[ -n "$uptime" ]]; then
            local start_date
            start_date=$(date -d "$uptime" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "Unknown")
            status="Running since $start_date"
        fi
        log::success "      ✅ $CONTAINER_NAME: $status"
    else
        log::error "      ❌ $CONTAINER_NAME: Stopped"
    fi
    
    # Database container
    if docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER_NAME}$"; then
        log::success "      ✅ $DB_CONTAINER_NAME: Running"
    else
        log::error "      ❌ $DB_CONTAINER_NAME: Stopped"
    fi
}

#######################################
# Show resource usage summary
# Returns: 0 always
#######################################
huginn::show_resource_summary() {
    if ! huginn::is_running; then
        return 0
    fi
    
    log::info "   💾 Resource Usage:"
    
    # Get container stats (lightweight version)
    local huginn_stats
    local db_stats
    
    huginn_stats=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemUsage}}" "$CONTAINER_NAME" 2>/dev/null || echo "N/A\tN/A")
    db_stats=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemUsage}}" "$DB_CONTAINER_NAME" 2>/dev/null || echo "N/A\tN/A")
    
    local huginn_cpu huginn_mem db_cpu db_mem
    read -r huginn_cpu huginn_mem <<< "$huginn_stats"
    read -r db_cpu db_mem <<< "$db_stats"
    
    log::info "      App: CPU $huginn_cpu, Memory $huginn_mem"
    log::info "      DB:  CPU $db_cpu, Memory $db_mem"
}

#######################################
# Show health metrics and statistics
# Returns: 0 always
#######################################
huginn::show_health_metrics() {
    if ! huginn::is_running; then
        log::info "📈 Health Metrics: (Service not running)"
        return 0
    fi
    
    log::info "📈 Health Metrics:"
    
    # Get system statistics
    local stats_json
    stats_json=$(huginn::get_system_stats 2>/dev/null)
    
    if [[ -n "$stats_json" ]] && echo "$stats_json" | jq . >/dev/null 2>&1; then
        # Parse statistics
        local users agents scenarios events links active_agents recent_events
        users=$(echo "$stats_json" | jq -r '.users // 0')
        agents=$(echo "$stats_json" | jq -r '.agents // 0')
        scenarios=$(echo "$stats_json" | jq -r '.scenarios // 0')
        events=$(echo "$stats_json" | jq -r '.events // 0')
        links=$(echo "$stats_json" | jq -r '.links // 0')
        active_agents=$(echo "$stats_json" | jq -r '.active_agents // 0')
        recent_events=$(echo "$stats_json" | jq -r '.recent_events // 0')
        
        log::info "   👥 Users: $users"
        log::info "   🤖 Agents: $agents (Active: $active_agents)"
        log::info "   📂 Scenarios: $scenarios"
        log::info "   📊 Events: $events (Recent: $recent_events)"
        log::info "   🔗 Agent Links: $links"
        
        # Calculate health score
        local health_score=100
        if [[ $agents -gt 0 ]]; then
            local active_percentage=$((active_agents * 100 / agents))
            if [[ $active_percentage -lt 50 ]]; then
                health_score=75
            elif [[ $active_percentage -lt 25 ]]; then
                health_score=50
            fi
        fi
        
        if [[ $health_score -ge 90 ]]; then
            log::success "   💚 Health Score: $health_score% (Excellent)"
        elif [[ $health_score -ge 75 ]]; then
            log::info "   💛 Health Score: $health_score% (Good)"
        else
            log::warn "   🧡 Health Score: $health_score% (Needs Attention)"
        fi
    else
        log::warn "   ⚠️  Unable to retrieve system statistics"
    fi
}

#######################################
# Show recent system activity
# Returns: 0 always
#######################################
huginn::show_recent_activity() {
    if ! huginn::is_running; then
        log::info "📰 Recent Activity: (Service not running)"
        return 0
    fi
    
    log::info "📰 Recent Activity (last 5 events):"
    
    local activity_code='
    begin
      events = Event.includes(:agent)
                   .order(created_at: :desc)
                   .limit(5)
      
      if events.any?
        events.each do |event|
          agent_name = event.agent&.name || "Unknown Agent"
          timestamp = event.created_at.strftime("%m/%d %H:%M")
          payload_preview = event.payload.to_s[0..60] + "..."
          puts "   📄 #{timestamp} - #{agent_name}: #{payload_preview}"
        end
      else
        puts "   📭 No recent events found"
      end
    rescue => e
      puts "   ❌ Error retrieving events: #{e.message}"
    end
    '
    
    huginn::rails_runner "$activity_code" 2>/dev/null || log::warn "   ⚠️  Unable to retrieve recent activity"
}

#######################################
# Show detailed system information
# Returns: 0 always
#######################################
huginn::show_info() {
    log::header "ℹ️  Huginn System Information"
    echo
    
    # Basic information
    log::info "🤖 Huginn Automation Platform"
    log::info "   Web Interface: $HUGINN_BASE_URL"
    log::info "   Container: $CONTAINER_NAME"
    log::info "   Database: $DB_CONTAINER_NAME"
    log::info "   Network: $NETWORK_NAME"
    echo
    
    # Data locations
    log::info "📁 Data Locations:"
    log::info "   Config: $HUGINN_DATA_DIR"
    log::info "   Database: $HUGINN_DB_DIR"
    log::info "   Uploads: $HUGINN_UPLOADS_DIR"
    echo
    
    # Docker resources
    log::info "🐳 Docker Resources:"
    log::info "   Huginn Image: $HUGINN_IMAGE"
    log::info "   Database Image: $POSTGRES_IMAGE"
    log::info "   Data Volume: $VOLUME_NAME"
    log::info "   DB Volume: $DB_VOLUME_NAME"
    echo
    
    # Authentication
    log::info "🔐 Authentication:"
    log::info "   Username: $DEFAULT_ADMIN_USERNAME"
    log::info "   Password: $DEFAULT_ADMIN_PASSWORD"
    log::info "   Email: $DEFAULT_ADMIN_EMAIL"
    echo
    
    # Management commands
    log::info "🛠️  Management Commands:"
    log::info "   Status: $0 --action status"
    log::info "   Agents: $0 --action agents --operation list"
    log::info "   Scenarios: $0 --action scenarios --operation list"
    log::info "   Events: $0 --action events --operation recent"
    log::info "   Logs: $0 --action logs"
    log::info "   Backup: $0 --action backup"
}

#######################################
# Perform comprehensive health check
# Returns: 0 if healthy, 1 if issues found
#######################################
huginn::health_check() {
    log::header "🏥 Huginn Health Check"
    echo
    
    local issues_found=0
    
    # Check installation
    log::info "1. Checking installation..."
    if huginn::is_installed; then
        log::success "   ✅ Huginn is installed"
    else
        log::error "   ❌ Huginn is not installed"
        issues_found=1
    fi
    
    # Check if running
    log::info "2. Checking service status..."
    if huginn::is_running; then
        log::success "   ✅ Containers are running"
    else
        log::error "   ❌ Containers are not running"
        issues_found=1
    fi
    
    # Check web interface
    log::info "3. Checking web interface..."
    if huginn::is_healthy; then
        log::success "   ✅ Web interface is accessible"
    else
        log::error "   ❌ Web interface is not responding"
        issues_found=1
    fi
    
    # Check database connection
    log::info "4. Checking database connection..."
    if huginn::check_database; then
        log::success "   ✅ Database connection is working"
    else
        log::error "   ❌ Database connection failed"
        issues_found=1
    fi
    
    # Check disk space
    log::info "5. Checking disk space..."
    local available_space
    available_space=$(df "${HOME}" | awk 'NR==2 {print $4}')
    if [[ $available_space -gt 1048576 ]]; then  # 1GB in KB
        log::success "   ✅ Sufficient disk space available"
    else
        log::warn "   ⚠️  Low disk space (less than 1GB available)"
        issues_found=1
    fi
    
    # Check Docker resources
    log::info "6. Checking Docker resources..."
    if docker system df >/dev/null 2>&1; then
        log::success "   ✅ Docker is healthy"
    else
        log::error "   ❌ Docker system issues detected"
        issues_found=1
    fi
    
    echo
    if [[ $issues_found -eq 0 ]]; then
        log::success "🎉 All health checks passed!"
        return 0
    else
        log::error "⚠️  $issues_found issue(s) found"
        log::info "Run '$0 --action logs' to investigate issues"
        return 1
    fi
}

#######################################
# Monitor system in real-time
# Arguments:
#   $1 - interval in seconds (optional, defaults to 30)
# Returns: 0 on interrupt, 1 on error
#######################################
huginn::monitor() {
    local interval="${1:-30}"
    
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    log::header "📊 Huginn Real-time Monitor (${interval}s intervals)"
    log::info "Press Ctrl+C to stop monitoring"
    echo
    
    # Monitor loop
    while true; do
        clear
        echo "🤖 Huginn Monitor - $(date '+%Y-%m-%d %H:%M:%S')"
        echo "=" | tr '=' '=' | head -c 50; echo
        
        # Basic status
        huginn::show_basic_status
        echo
        
        # Resource usage
        if huginn::is_running; then
            huginn::get_resource_usage
        fi
        
        # Recent events count
        local recent_events
        recent_events=$(huginn::rails_runner 'puts Event.where("created_at > ?", 5.minutes.ago).count' 2>/dev/null || echo "N/A")
        echo "📊 Events (last 5 min): $recent_events"
        
        echo
        echo "Next update in ${interval}s... (Ctrl+C to stop)"
        
        sleep "$interval"
    done
}

#######################################
# Show system metrics for automation
# Returns: 0 always
#######################################
huginn::show_metrics() {
    if ! huginn::is_running; then
        echo "status=stopped"
        return 0
    fi
    
    # Output metrics in key=value format for automation
    echo "status=running"
    echo "health=$(huginn::is_healthy && echo "healthy" || echo "unhealthy")"
    echo "url=$HUGINN_BASE_URL"
    echo "port=$HUGINN_PORT"
    
    # Get system stats
    local stats_json
    stats_json=$(huginn::get_system_stats 2>/dev/null)
    
    if [[ -n "$stats_json" ]] && echo "$stats_json" | jq . >/dev/null 2>&1; then
        echo "users=$(echo "$stats_json" | jq -r '.users // 0')"
        echo "agents=$(echo "$stats_json" | jq -r '.agents // 0')"
        echo "scenarios=$(echo "$stats_json" | jq -r '.scenarios // 0')"
        echo "events=$(echo "$stats_json" | jq -r '.events // 0')"
        echo "active_agents=$(echo "$stats_json" | jq -r '.active_agents // 0')"
        echo "recent_events=$(echo "$stats_json" | jq -r '.recent_events // 0')"
    fi
}