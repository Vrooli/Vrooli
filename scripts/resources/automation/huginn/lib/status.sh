#!/usr/bin/env bash
# Huginn Status and Health Monitoring Functions
# System status, health checks, and information display

#######################################
# Show comprehensive system status
# Returns: 0 always
#######################################
huginn::show_status() {
    huginn::show_status_header
    echo
    
    # Basic status
    huginn::show_basic_status
    echo
    
    # Detailed system information
    huginn::show_system_info
    echo
    
    # Health metrics
    huginn::show_health_metrics
    echo
    
    # Recent activity
    huginn::show_recent_activity
    
    return 0
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