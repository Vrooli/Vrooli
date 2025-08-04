#!/usr/bin/env bash
# PostgreSQL Status Monitoring
# Functions for checking PostgreSQL instances health and status

#######################################
# Get comprehensive PostgreSQL resource status
# Arguments:
#   $1 - verbose flag (optional, default: false)
# Returns: 0 if healthy, 1 if not
#######################################
postgres::status::check() {
    local verbose=${1:-false}
    local all_good=true
    local instances=($(postgres::common::list_instances))
    
    if [[ "$verbose" == "true" ]]; then
        log::info "PostgreSQL Resource Status Check"
        log::info "================================"
    fi
    
    # Check Docker
    if postgres::docker::check_docker >/dev/null 2>&1; then
        [[ "$verbose" == "true" ]] && log::success "Docker: Available"
    else
        [[ "$verbose" == "true" ]] && log::error "Docker: Not available"
        all_good=false
    fi
    
    # Check if resource is installed
    if [[ ${#instances[@]} -eq 0 ]]; then
        [[ "$verbose" == "true" ]] && log::warn "Resource: Not installed (no instances found)"
        all_good=false
    else
        [[ "$verbose" == "true" ]] && log::success "Resource: Installed (${#instances[@]} instance(s))"
        
        # Check each instance
        local running_count=0
        local healthy_count=0
        
        for instance in "${instances[@]}"; do
            local instance_status="stopped"
            local health_status="unknown"
            
            if postgres::common::is_running "$instance"; then
                instance_status="running"
                ((running_count++))
                
                if postgres::common::health_check "$instance"; then
                    health_status="healthy"
                    ((healthy_count++))
                else
                    health_status="unhealthy"
                    all_good=false
                fi
            fi
            
            if [[ "$verbose" == "true" ]]; then
                local port=$(postgres::common::get_instance_config "$instance" "port")
                case "$instance_status" in
                    "running")
                        if [[ "$health_status" == "healthy" ]]; then
                            log::success "Instance '$instance': Running and healthy (port $port)"
                        else
                            log::error "Instance '$instance': Running but unhealthy (port $port)"
                        fi
                        ;;
                    *)
                        log::warn "Instance '$instance': Stopped (port $port)"
                        ;;
                esac
            fi
        done
        
        if [[ "$verbose" == "true" ]]; then
            log::info ""
            log::info "Summary: $running_count/${#instances[@]} running, $healthy_count/${#instances[@]} healthy"
        fi
        
        # Consider resource healthy if at least one instance is healthy
        if [[ $healthy_count -eq 0 && ${#instances[@]} -gt 0 ]]; then
            all_good=false
        fi
    fi
    
    # Check disk space
    if postgres::common::check_disk_space >/dev/null 2>&1; then
        if [[ "$verbose" == "true" ]]; then
            local available_gb=$(df -BG "$(dirname "${POSTGRES_INSTANCES_DIR}")" | awk 'NR==2 {print $4}' | sed 's/G//')
            log::success "Disk Space: ${available_gb}GB available"
        fi
    else
        [[ "$verbose" == "true" ]] && log::error "Disk Space: Insufficient"
        all_good=false
    fi
    
    # Check Vrooli configuration
    if [[ "$verbose" == "true" ]]; then
        if postgres::status::check_vrooli_config; then
            log::success "Vrooli Config: Registered"
        else
            log::warn "Vrooli Config: Not registered"
        fi
    fi
    
    if [[ "$verbose" == "true" ]]; then
        log::info ""
        if [[ "$all_good" == "true" ]]; then
            log::success "Overall Status: ✅ Healthy"
        else
            log::error "Overall Status: ❌ Issues detected"
        fi
        
        # Show usage info
        log::info ""
        log::info "Available port range: ${POSTGRES_INSTANCE_PORT_RANGE_START}-${POSTGRES_INSTANCE_PORT_RANGE_END}"
        log::info "Maximum instances: ${POSTGRES_MAX_INSTANCES}"
        log::info "${MSG_HELP_LIST}"
    fi
    
    [[ "$all_good" == "true" ]] && return 0 || return 1
}

#######################################
# Check if PostgreSQL is registered in Vrooli config
# Returns: 0 if registered, 1 if not
#######################################
postgres::status::check_vrooli_config() {
    local config_file="${HOME}/.vrooli/service.json"
    
    if [[ ! -f "$config_file" ]]; then
        return 1
    fi
    
    # Check if PostgreSQL is in the config
    if grep -q '"postgres"' "$config_file" 2>/dev/null && \
       grep -q '"enabled": true' "$config_file" 2>/dev/null; then
        return 0
    fi
    
    return 1
}

#######################################
# Show detailed status for specific instance
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::status::show_instance() {
    local instance_name="${1:-main}"
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    log::info "PostgreSQL Instance Status: $instance_name"
    log::info "=========================================="
    
    # Basic info
    local port=$(postgres::common::get_instance_config "$instance_name" "port")
    local template=$(postgres::common::get_instance_config "$instance_name" "template")
    local created=$(postgres::common::get_instance_config "$instance_name" "created")
    
    log::info "Port: $port"
    log::info "Template: $template"
    log::info "Created: $created"
    
    # Container status
    if postgres::common::is_running "$instance_name"; then
        log::success "Container: Running"
        
        # Health check
        if postgres::common::health_check "$instance_name"; then
            log::success "Health: Healthy"
        else
            log::error "Health: Unhealthy"
        fi
        
        # Connection info
        log::info ""
        log::info "Connection Information:"
        log::info "  Host: localhost"
        log::info "  Port: $port"
        log::info "  Database: ${POSTGRES_DEFAULT_DB}"
        log::info "  User: ${POSTGRES_DEFAULT_USER}"
        
        local conn_string=$(postgres::instance::get_connection_string "$instance_name")
        log::info "  Connection String: $conn_string"
        
    else
        log::warn "Container: Stopped"
        log::info "Use 'manage.sh --action start --instance $instance_name' to start"
    fi
    
    return 0
}

#######################################
# Show container resource usage for instance
# Arguments:
#   $1 - instance name
#######################################
postgres::status::show_resources() {
    local instance_name="${1:-main}"
    
    if ! postgres::common::is_running "$instance_name"; then
        log::warn "PostgreSQL instance '$instance_name' is not running"
        return 1
    fi
    
    log::info "PostgreSQL Instance Resource Usage: $instance_name"
    log::info "================================================="
    
    # Get container stats
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    local stats=$(postgres::docker::stats "$instance_name")
    
    if [[ -n "$stats" && "$stats" != "{}" ]]; then
        # Parse JSON stats (simplified, real implementation would use jq)
        echo "$stats" | grep -E "(MemUsage|CPUPerc|NetIO|BlockIO)" || {
            # Fallback to docker stats command
            docker stats "$container_name" --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"
        }
    fi
    
    # Show disk usage
    log::info ""
    log::info "Storage Usage:"
    local instance_dir="${POSTGRES_INSTANCES_DIR}/${instance_name}"
    local data_size=$(du -sh "${instance_dir}" 2>/dev/null | cut -f1 || echo "N/A")
    log::info "  Instance Directory: $data_size"
    
    # Show database stats if possible
    if postgres::common::is_running "$instance_name"; then
        log::info ""
        log::info "Database Statistics:"
        local db_count=$(postgres::docker::exec_sql "$instance_name" "SELECT count(*) FROM pg_database;" 2>/dev/null | tail -n +3 | head -n 1 | tr -d ' ' || echo "N/A")
        log::info "  Databases: $db_count"
        
        local conn_count=$(postgres::docker::exec_sql "$instance_name" "SELECT count(*) FROM pg_stat_activity;" 2>/dev/null | tail -n +3 | head -n 1 | tr -d ' ' || echo "N/A")
        log::info "  Active Connections: $conn_count"
    fi
}

#######################################
# Monitor PostgreSQL instances continuously
# Arguments:
#   $1 - Check interval in seconds (default: 5)
#######################################
postgres::status::monitor() {
    local interval=${1:-5}
    
    log::info "Monitoring PostgreSQL instances (Press Ctrl+C to stop)..."
    log::info ""
    
    local iteration=0
    while true; do
        # Clear screen every 10 iterations
        if [[ $((iteration % 10)) -eq 0 ]]; then
            clear
            log::info "PostgreSQL Instance Monitor - $(date)"
            log::info "======================================="
        fi
        
        local instances=($(postgres::common::list_instances))
        local healthy_count=0
        local total_count=${#instances[@]}
        
        if [[ $total_count -eq 0 ]]; then
            printf "\r[$(date +%H:%M:%S)] No instances found    "
        else
            for instance in "${instances[@]}"; do
                if postgres::common::is_running "$instance" && postgres::common::health_check "$instance"; then
                    ((healthy_count++))
                fi
            done
            
            printf "\r[$(date +%H:%M:%S)] Status: $healthy_count/$total_count healthy    "
        fi
        
        sleep "$interval"
        iteration=$((iteration + 1))
    done
}

#######################################
# Run diagnostic checks on PostgreSQL resource
# Returns: 0 if all pass, 1 if any fail
#######################################
postgres::status::diagnose() {
    log::info "Running PostgreSQL Resource Diagnostics..."
    log::info "=========================================="
    
    local issues=0
    local instances=($(postgres::common::list_instances))
    
    # Check Docker
    log::info "1. Checking Docker..."
    if postgres::docker::check_docker; then
        log::success "   Docker is available"
    else
        log::error "   Docker is not available"
        ((issues++))
    fi
    
    # Check if resource is installed
    log::info "2. Checking resource installation..."
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::warn "   No PostgreSQL instances found"
        log::info "   Run 'manage.sh --action install' to install the resource"
    else
        log::success "   Resource installed with ${#instances[@]} instance(s)"
    fi
    
    # Check port availability
    log::info "3. Checking port range availability..."
    local available_ports=0
    for port in $(seq $POSTGRES_INSTANCE_PORT_RANGE_START $POSTGRES_INSTANCE_PORT_RANGE_END); do
        if postgres::common::is_port_available "$port"; then
            ((available_ports++))
        fi
    done
    
    local total_range=$((POSTGRES_INSTANCE_PORT_RANGE_END - POSTGRES_INSTANCE_PORT_RANGE_START + 1))
    if [[ $available_ports -gt $((total_range / 2)) ]]; then
        log::success "   $available_ports/$total_range ports available"
    else
        log::warn "   Only $available_ports/$total_range ports available"
    fi
    
    # Check disk space
    log::info "4. Checking disk space..."
    if postgres::common::check_disk_space; then
        log::success "   Sufficient disk space available"
    else
        log::error "   Insufficient disk space"
        ((issues++))
    fi
    
    # Check individual instances
    if [[ ${#instances[@]} -gt 0 ]]; then
        log::info "5. Checking individual instances..."
        for instance in "${instances[@]}"; do
            if postgres::common::container_exists "$instance"; then
                if postgres::common::is_running "$instance"; then
                    if postgres::common::health_check "$instance"; then
                        log::success "   Instance '$instance': Healthy"
                    else
                        log::error "   Instance '$instance': Unhealthy"
                        ((issues++))
                    fi
                else
                    log::warn "   Instance '$instance': Stopped"
                fi
            else
                log::error "   Instance '$instance': Container missing"
                ((issues++))
            fi
        done
    fi
    
    # Summary
    log::info ""
    if [[ $issues -eq 0 ]]; then
        log::success "Diagnostics passed! No issues found."
        return 0
    else
        log::error "Diagnostics failed: $issues issue(s) found"
        return 1
    fi
}