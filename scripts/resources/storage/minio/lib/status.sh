#!/usr/bin/env bash
# MinIO Status Monitoring
# Functions for checking MinIO health and status

#######################################
# Get comprehensive MinIO status
# Returns: 0 if healthy, 1 if not
#######################################
minio::status::check() {
    local verbose=${1:-false}
    local all_good=true
    
    if [[ "$verbose" == "true" ]]; then
        log::info "MinIO Status Check"
        log::info "=================="
    fi
    
    # Check Docker
    if minio::docker::check_docker >/dev/null 2>&1; then
        [[ "$verbose" == "true" ]] && log::success "Docker: Available"
    else
        [[ "$verbose" == "true" ]] && log::error "Docker: Not available"
        all_good=false
    fi
    
    # Check container exists
    if minio::common::container_exists; then
        [[ "$verbose" == "true" ]] && log::success "Container: Exists"
        
        # Check if running
        if minio::common::is_running; then
            [[ "$verbose" == "true" ]] && log::success "${MSG_RUNNING}"
            
            # Check API health
            if minio::common::health_check; then
                [[ "$verbose" == "true" ]] && log::success "${MSG_HEALTHY}"
                
                # Check ports
                if ! minio::common::is_port_available "${MINIO_PORT}"; then
                    [[ "$verbose" == "true" ]] && log::success "API Port ${MINIO_PORT}: In use by MinIO"
                else
                    [[ "$verbose" == "true" ]] && log::warn "API Port ${MINIO_PORT}: Not listening"
                    all_good=false
                fi
                
                if ! minio::common::is_port_available "${MINIO_CONSOLE_PORT}"; then
                    [[ "$verbose" == "true" ]] && log::success "Console Port ${MINIO_CONSOLE_PORT}: In use by MinIO"
                else
                    [[ "$verbose" == "true" ]] && log::warn "Console Port ${MINIO_CONSOLE_PORT}: Not listening"
                fi
                
                # Show access info
                if [[ "$verbose" == "true" ]]; then
                    log::info ""
                    log::info "Access Information:"
                    log::info "  API URL: ${MINIO_BASE_URL}"
                    log::info "  Console URL: ${MINIO_CONSOLE_URL}"
                    log::info "  ${MSG_HELP_CREDENTIALS}"
                fi
            else
                [[ "$verbose" == "true" ]] && log::error "${MSG_HEALTH_CHECK_FAILED}"
                all_good=false
            fi
        else
            [[ "$verbose" == "true" ]] && log::warn "Container: Stopped"
            all_good=false
        fi
    else
        [[ "$verbose" == "true" ]] && log::warn "Container: Not found"
        all_good=false
    fi
    
    # Check disk space
    if minio::common::check_disk_space >/dev/null 2>&1; then
        if [[ "$verbose" == "true" ]]; then
            local available_gb=$(df -BG "$(dirname "${MINIO_DATA_DIR}")" | awk 'NR==2 {print $4}' | sed 's/G//')
            log::success "Disk Space: ${available_gb}GB available"
        fi
    else
        [[ "$verbose" == "true" ]] && log::error "Disk Space: Insufficient"
        all_good=false
    fi
    
    # Check Vrooli configuration
    if [[ "$verbose" == "true" ]]; then
        if minio::status::check_vrooli_config; then
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
    fi
    
    [[ "$all_good" == "true" ]] && return 0 || return 1
}

#######################################
# Check if MinIO is registered in Vrooli config
# Returns: 0 if registered, 1 if not
#######################################
minio::status::check_vrooli_config() {
    local config_file="${HOME}/.vrooli/resources.local.json"
    
    if [[ ! -f "$config_file" ]]; then
        return 1
    fi
    
    # Check if MinIO is in the config
    if grep -q '"minio"' "$config_file" 2>/dev/null && \
       grep -q '"enabled": true' "$config_file" 2>/dev/null; then
        return 0
    fi
    
    return 1
}

#######################################
# Show container resource usage
#######################################
minio::status::show_resources() {
    if ! minio::common::is_running; then
        log::warn "MinIO is not running"
        return 1
    fi
    
    log::info "MinIO Resource Usage:"
    log::info "===================="
    
    # Get container stats
    local stats=$(minio::docker::stats)
    
    if [[ -n "$stats" && "$stats" != "{}" ]]; then
        # Parse JSON stats (this is simplified, real implementation would use jq)
        echo "$stats" | grep -E "(MemUsage|CPUPerc|NetIO|BlockIO)" || {
            # Fallback to docker stats command
            docker stats "$MINIO_CONTAINER_NAME" --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"
        }
    fi
    
    # Show disk usage
    log::info ""
    log::info "Storage Usage:"
    local data_size=$(du -sh "${MINIO_DATA_DIR}" 2>/dev/null | cut -f1 || echo "N/A")
    log::info "  Data Directory: $data_size"
    
    # Show bucket stats
    if minio::common::is_running; then
        log::info ""
        minio::buckets::show_stats
    fi
}

#######################################
# Display credentials
# Returns: 0 on success, 1 on failure
#######################################
minio::status::show_credentials() {
    local creds_file="${MINIO_CONFIG_DIR}/credentials"
    
    log::info "MinIO Access Credentials:"
    log::info "========================"
    
    # Try to load from file first
    if [[ -f "$creds_file" ]]; then
        source "$creds_file"
    fi
    
    if [[ -n "${MINIO_ROOT_USER}" && -n "${MINIO_ROOT_PASSWORD}" ]]; then
        log::info "Username: ${MINIO_ROOT_USER}"
        log::info "Password: ${MINIO_ROOT_PASSWORD}"
        log::info ""
        log::info "Console URL: ${MINIO_CONSOLE_URL}"
        log::info "API Endpoint: ${MINIO_BASE_URL}"
        
        if [[ "${MINIO_ROOT_USER}" == "minioadmin" && "${MINIO_ROOT_PASSWORD}" == "minioadmin" ]]; then
            log::warn ""
            log::warn "⚠️  Using default credentials - not secure for production!"
            log::info "Run '$0 --action reset-credentials' to generate secure credentials"
        fi
        
        return 0
    else
        log::error "Credentials not found"
        log::info "MinIO may not be installed or credentials were not saved"
        return 1
    fi
}

#######################################
# Monitor MinIO health continuously
# Arguments:
#   $1 - Check interval in seconds (default: 5)
#######################################
minio::status::monitor() {
    local interval=${1:-5}
    
    log::info "Monitoring MinIO health (Press Ctrl+C to stop)..."
    log::info ""
    
    local iteration=0
    while true; do
        # Clear screen every 10 iterations
        if [[ $((iteration % 10)) -eq 0 ]]; then
            clear
            log::info "MinIO Health Monitor - $(date)"
            log::info "========================================"
        fi
        
        # Quick health check
        if minio::common::health_check; then
            printf "\r[$(date +%H:%M:%S)] Status: ✅ Healthy    "
        else
            printf "\r[$(date +%H:%M:%S)] Status: ❌ Unhealthy  "
        fi
        
        sleep "$interval"
        iteration=$((iteration + 1))
    done
}

#######################################
# Run diagnostic checks
# Returns: 0 if all pass, 1 if any fail
#######################################
minio::status::diagnose() {
    log::info "Running MinIO Diagnostics..."
    log::info "==========================="
    
    local issues=0
    
    # Check Docker
    log::info "1. Checking Docker..."
    if minio::docker::check_docker; then
        log::success "   Docker is available"
    else
        log::error "   Docker is not available"
        ((issues++))
    fi
    
    # Check ports
    log::info "2. Checking port availability..."
    if minio::common::is_port_available "${MINIO_PORT}"; then
        if minio::common::is_running; then
            log::error "   Port ${MINIO_PORT} appears free but MinIO is running"
            ((issues++))
        else
            log::success "   Port ${MINIO_PORT} is available"
        fi
    else
        if minio::common::is_running; then
            log::success "   Port ${MINIO_PORT} is in use by MinIO"
        else
            log::error "   Port ${MINIO_PORT} is in use by another process"
            ((issues++))
        fi
    fi
    
    # Check disk space
    log::info "3. Checking disk space..."
    if minio::common::check_disk_space; then
        log::success "   Sufficient disk space available"
    else
        log::error "   Insufficient disk space"
        ((issues++))
    fi
    
    # Check container
    log::info "4. Checking container..."
    if minio::common::container_exists; then
        log::success "   Container exists"
        
        if minio::common::is_running; then
            log::success "   Container is running"
            
            # Check logs for errors
            log::info "5. Checking container logs..."
            local error_count=$(docker logs "$MINIO_CONTAINER_NAME" 2>&1 | grep -c -i "error" || echo "0")
            if [[ $error_count -gt 0 ]]; then
                log::warn "   Found $error_count error(s) in logs"
                log::info "   Run '$0 --action logs' to view full logs"
            else
                log::success "   No errors in recent logs"
            fi
        else
            log::warn "   Container is stopped"
            ((issues++))
        fi
    else
        log::warn "   Container not found"
        ((issues++))
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