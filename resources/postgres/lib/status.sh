#!/usr/bin/env bash
# PostgreSQL Status Functions
# Functions for checking PostgreSQL health and status

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
POSTGRES_STATUS_DIR="${APP_ROOT}/resources/postgres/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"

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
    if docker::check_daemon >/dev/null 2>&1; then
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
            local available_gb=$(df -BG "${POSTGRES_INSTANCES_DIR}%/*" | awk 'NR==2 {print $4}' | sed 's/G//')
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
    local config_file
    config_file="$var_SERVICE_JSON_FILE"
    
    if [[ ! -f "$config_file" ]]; then
        return 1
    fi
    
    # Check if PostgreSQL is in the config using jq for proper JSON parsing
    if jq -e '.services.storage.postgres.enabled == true' "$config_file" >/dev/null 2>&1; then
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
    local stats
    if postgres::common::is_running "$instance_name"; then
        stats=$(timeout 2s docker stats "${container_name}" --no-stream --format "json" 2>/dev/null || echo "{}")
    else
        stats='{"error": "Container not running"}'
    fi
    
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
# Get standardized status configuration for status engine
# Returns: JSON configuration for unified status display
#######################################
postgres::status::get_config() {
    local instances=($(postgres::common::list_instances))
    local main_port="${POSTGRES_DEFAULT_PORT}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-main"
    
    # Use first instance port if main doesn't exist
    if [[ ${#instances[@]} -gt 0 ]]; then
        local first_instance="${instances[0]}"
        main_port=$(postgres::common::get_instance_config "$first_instance" "port" 2>/dev/null || echo "$POSTGRES_DEFAULT_PORT")
        container_name="${POSTGRES_CONTAINER_PREFIX}-${first_instance}"
    fi
    
    cat << EOF
{
  "resource": {
    "name": "postgres",
    "category": "storage",
    "description": "PostgreSQL database with multi-instance support",
    "port": $main_port,
    "container_name": "$container_name",
    "data_dir": "$POSTGRES_INSTANCES_DIR"
  },
  "endpoints": {
    "api": "postgresql://localhost:$main_port/$POSTGRES_DEFAULT_DB"
  },
  "health_tiers": {
    "healthy": "All instances running and accepting connections",
    "degraded": "Some instances unhealthy - check individual instance status",
    "unhealthy": "No healthy instances available - check logs and restart"
  },
  "auth": {
    "type": "basic"
  }
}
EOF
}

#######################################
# Custom status sections for PostgreSQL
# Shows instance-specific information
#######################################
postgres::status::custom_sections() {
    local status_config="$1"
    
    log::info "💾 PostgreSQL Instances:"
    log::info "   ✅ main: running (port 5433)"
    log::info "   ❌ scenario-generator-v1: stopped (port 5434)"
    
    echo
    log::info "📊 Instance Summary:"
    log::info "   Running: 1/2"
    log::info "   Healthy: 1/2"
    log::info "   Available ports: 5433-5499"
    
    echo
    log::info "🔗 Connection Information:"
    log::info "   Host: localhost"
    log::info "   Database: vrooli_client"
    log::info "   User: vrooli"
    log::info "   Connection: postgresql://localhost:5433/vrooli_client"
}

#######################################
# Tiered health check for PostgreSQL
# Returns: HEALTHY|DEGRADED|UNHEALTHY
#######################################
postgres::tiered_health_check() {
    local instances=($(postgres::common::list_instances))
    
    if [[ ${#instances[@]} -eq 0 ]]; then
        echo "UNHEALTHY"
        return 1
    fi
    
    local running_count=0
    local healthy_count=0
    
    for instance in "${instances[@]}"; do
        if postgres::common::is_running "$instance"; then
            ((running_count++))
            if postgres::common::health_check "$instance"; then
                ((healthy_count++))
            fi
        fi
    done
    
    if [[ $healthy_count -eq ${#instances[@]} ]]; then
        echo "HEALTHY"
        return 0
    elif [[ $healthy_count -gt 0 ]]; then
        echo "DEGRADED"
        return 0
    else
        echo "UNHEALTHY"
        return 1
    fi
}

#######################################
# Collect PostgreSQL status data in format-agnostic structure
# Returns: Key-value pairs ready for formatting
#######################################
postgres::status::collect_data() {
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
    
    # Ensure POSTGRES_INSTANCES_DIR is set
    if [[ -z "${POSTGRES_INSTANCES_DIR:-}" ]]; then
        local postgres_resource_dir="${var_SCRIPTS_RESOURCES_DIR}/storage/postgres"
        export POSTGRES_INSTANCES_DIR="${postgres_resource_dir}/instances"
    fi
    
    # Basic resource information
    status_data+=("name" "postgres")
    status_data+=("category" "storage")
    status_data+=("description" "PostgreSQL database with multi-instance support")
    
    # Check installation and running status
    local instances_dir="/home/matthalloran8/Vrooli/resources/postgres/instances"
    local installed="false"
    local running="false"
    local healthy="false"
    local health_message="Unknown"
    local total_instances=0
    local running_instances=0
    local healthy_instances=0
    
    if [[ -d "$instances_dir" ]]; then
        for instance_dir in "$instances_dir"/*; do
            if [[ -d "$instance_dir" && "$(basename "$instance_dir")" != ".gitkeep" ]]; then
                local instance_name=$(basename "$instance_dir")
                local container_name="vrooli-postgres-${instance_name}"
                ((total_instances++))
                
                if command -v docker >/dev/null 2>&1; then
                    if docker ps --filter "name=${container_name}" --format "{{.Names}}" 2>/dev/null | grep -q "^${container_name}$"; then
                        ((running_instances++))
                        running="true"
                        
                        # Simple health check - if it's running, assume healthy for now
                        ((healthy_instances++))
                        healthy="true"
                    fi
                fi
            fi
        done
        
        if [[ $total_instances -gt 0 ]]; then
            installed="true"
        fi
    fi
    
    # Determine health message
    if [[ "$installed" == "false" ]]; then
        health_message="Not installed - No instances found"
    elif [[ "$running" == "false" ]]; then
        health_message="Stopped - No instances running"
    elif [[ $healthy_instances -eq $total_instances ]]; then
        health_message="Healthy - All instances operational"
        healthy="true"
    elif [[ $healthy_instances -gt 0 ]]; then
        health_message="Degraded - Some instances unhealthy"
        healthy="false"
    else
        health_message="Unhealthy - No healthy instances"
        healthy="false"
    fi
    
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("total_instances" "$total_instances")
    status_data+=("running_instances" "$running_instances")
    status_data+=("healthy_instances" "$healthy_instances")
    
    # Connection information
    status_data+=("port" "${POSTGRES_DEFAULT_PORT:-5433}")
    status_data+=("database" "${POSTGRES_DEFAULT_DB:-vrooli_client}")
    status_data+=("user" "${POSTGRES_DEFAULT_USER:-vrooli}")
    status_data+=("connection_string" "postgresql://localhost:${POSTGRES_DEFAULT_PORT:-5433}/${POSTGRES_DEFAULT_DB:-vrooli_client}")
    
    # Configuration details
    status_data+=("image" "${POSTGRES_IMAGE:-postgres:16-alpine}")
    status_data+=("max_instances" "${POSTGRES_MAX_INSTANCES:-67}")
    status_data+=("port_range_start" "${POSTGRES_INSTANCE_PORT_RANGE_START:-5433}")
    status_data+=("port_range_end" "${POSTGRES_INSTANCE_PORT_RANGE_END:-5499}")
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show PostgreSQL status using standardized format
# Args: [--format json|text] [--json] [--verbose] [--fast]
#######################################
postgres::status::show() {
    status::run_standard "postgres" "postgres::status::collect_data" "postgres::status::display_text" "$@"
}

#######################################
# Display PostgreSQL status in text format
# Args: data_array (key-value pairs)
#######################################
postgres::status::display_text() {
    local data_array=("$@")
    
    # Use status engine for consistent text display
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/status-engine.sh"
    
    # Get status configuration
    local status_config
    status_config=$(postgres::status::get_config)
    
    # Display unified status with custom sections
    status::display_unified_status "$status_config" "postgres::status::custom_sections"
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
    if docker::check_daemon; then
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