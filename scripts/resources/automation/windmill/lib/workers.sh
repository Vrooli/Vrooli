#!/usr/bin/env bash
# Windmill Workers Management Functions
# Functions for managing and scaling Windmill worker containers

#######################################
# Scale workers to specified count
# Arguments:
#   $1 - Target worker count
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::scale_workers() {
    local target_count="$1"
    
    if [[ -z "$target_count" ]]; then
        log::error "Worker count is required"
        log::info "Usage: $0 --action scale-workers --workers <count>"
        return 1
    fi
    
    if ! [[ "$target_count" =~ ^[1-9][0-9]*$ ]]; then
        log::error "Invalid worker count: $target_count (must be positive integer)"
        return 1
    fi
    
    if ! windmill::is_installed; then
        log::error "Windmill is not installed"
        return 1
    fi
    
    log::header "⚡ Scaling Windmill Workers"
    log::info "Target worker count: $target_count"
    
    # Get current worker count
    local current_count
    current_count=$(windmill::get_worker_count)
    
    if [[ "$current_count" -eq "$target_count" ]]; then
        log::info "Workers already scaled to $target_count"
        return 0
    fi
    
    log::info "Current workers: $current_count"
    log::info "Scaling to: $target_count workers"
    
    # Check system resources before scaling up
    if [[ "$target_count" -gt "$current_count" ]]; then
        if ! windmill::validate_worker_resources "$target_count"; then
            return 1
        fi
    fi
    
    # Scale the workers using Docker Compose
    if ! windmill::compose_cmd up -d --scale "windmill-worker=$target_count"; then
        log::error "Failed to scale workers to $target_count"
        return 1
    fi
    
    # Update environment file with new count
    if [[ -f "$WINDMILL_ENV_FILE" ]]; then
        sed -i "s|^WINDMILL_WORKER_REPLICAS=.*|WINDMILL_WORKER_REPLICAS=$target_count|" "$WINDMILL_ENV_FILE"
    fi
    
    # Wait for workers to be ready
    log::info "Waiting for workers to be ready..."
    sleep 10
    
    # Verify scaling
    local new_count
    new_count=$(windmill::get_worker_count)
    
    if [[ "$new_count" -eq "$target_count" ]]; then
        log::success "✅ Workers scaled successfully to $target_count"
        windmill::show_worker_status
    else
        log::error "❌ Scaling failed - Expected: $target_count, Actual: $new_count"
        return 1
    fi
    
    return 0
}

#######################################
# Get current number of running workers
# Outputs: Number of active worker containers
#######################################
windmill::get_worker_count() {
    docker ps --filter "name=${WINDMILL_PROJECT_NAME}-worker" --format "{{.Names}}" | grep -v native | wc -l
}

#######################################
# Get total number of workers (including native)
# Outputs: Total number of worker containers
#######################################
windmill::get_total_worker_count() {
    docker ps --filter "name=${WINDMILL_PROJECT_NAME}-worker" --format "{{.Names}}" | wc -l
}

#######################################
# Validate system resources for worker scaling
# Arguments:
#   $1 - Target worker count
# Returns: 0 if resources sufficient, 1 otherwise
#######################################
windmill::validate_worker_resources() {
    local target_count="$1"
    local memory_per_worker_mb=2048  # Default memory limit in MB
    local errors=()
    
    # Extract numeric value from memory limit
    if [[ "$WORKER_MEMORY_LIMIT" =~ ^([0-9]+)([MG])$ ]]; then
        local memory_value="${BASH_REMATCH[1]}"
        local memory_unit="${BASH_REMATCH[2]}"
        
        if [[ "$memory_unit" == "G" ]]; then
            memory_per_worker_mb=$((memory_value * 1024))
        else
            memory_per_worker_mb="$memory_value"
        fi
    fi
    
    # Calculate total memory required for workers
    local total_memory_required_mb=$((target_count * memory_per_worker_mb))
    local total_memory_required_gb=$((total_memory_required_mb / 1024))
    
    # Check available system memory
    local available_memory_gb
    if available_memory_gb=$(system::get_memory_gb 2>/dev/null); then
        # Reserve 2GB for system and other services
        local usable_memory_gb=$((available_memory_gb - 2))
        
        if [[ $total_memory_required_gb -gt $usable_memory_gb ]]; then
            errors+=("Insufficient memory: need ${total_memory_required_gb}GB for workers, have ${usable_memory_gb}GB available")
        fi
    fi
    
    # Check CPU cores
    local cpu_cores
    if cpu_cores=$(nproc 2>/dev/null); then
        if [[ $target_count -gt $((cpu_cores * 2)) ]]; then
            log::warn "Worker count ($target_count) exceeds 2x CPU cores ($cpu_cores)"
            log::warn "This may cause performance issues"
        fi
    fi
    
    # Check available disk space for container overhead
    local disk_gb
    if disk_gb=$(df "$HOME" --output=avail --block-size=1G | tail -n1 | tr -d ' ' 2>/dev/null); then
        local disk_needed_gb=$((target_count / 10 + 1))  # Rough estimate
        if [[ $disk_gb -lt $disk_needed_gb ]]; then
            log::warn "Low disk space: ${disk_gb}GB available"
        fi
    fi
    
    if [[ ${#errors[@]} -gt 0 ]]; then
        log::error "Resource validation failed:"
        for error in "${errors[@]}"; do
            log::error "  • $error"
        done
        
        if [[ "$FORCE" != "yes" ]]; then
            log::info "Use --force yes to proceed anyway (not recommended)"
            return 1
        else
            log::warn "Proceeding with insufficient resources (--force specified)"
        fi
    fi
    
    return 0
}

#######################################
# Show detailed worker status
#######################################
windmill::show_worker_status() {
    log::info "👷 Worker Status:"
    
    echo
    printf "  %-30s %-15s %-15s %-20s\n" "WORKER CONTAINER" "STATUS" "UPTIME" "MEMORY USAGE"
    printf "  %-30s %-15s %-15s %-20s\n" "----------------" "------" "------" "------------"
    
    # Get worker containers
    local workers
    workers=$(docker ps --filter "name=${WINDMILL_PROJECT_NAME}-worker" --format "{{.Names}}")
    
    if [[ -n "$workers" ]]; then
        while IFS= read -r worker; do
            local status="stopped"
            local uptime="--"
            local memory="--"
            
            # Get container info
            if docker ps --format "{{.Names}}" | grep -q "^${worker}$"; then
                status="✅ running"
                uptime=$(docker ps --format "{{.Status}}" --filter "name=^${worker}$" | sed 's/Up //')
                memory=$(docker stats --no-stream --format "{{.MemUsage}}" "$worker" 2>/dev/null || echo "--")
            else
                status="❌ stopped"
            fi
            
            printf "  %-30s %-15s %-15s %-20s\n" "$worker" "$status" "$uptime" "$memory"
        done <<< "$workers"
    else
        echo "  No worker containers found"
    fi
    
    echo
    
    # Summary
    local running_workers total_workers
    running_workers=$(windmill::get_worker_count)
    total_workers=$(windmill::get_total_worker_count)
    
    echo "  Summary:"
    echo "    Default Workers: $running_workers running"
    echo "    Total Workers: $total_workers (including native/specialized)"
    echo "    Memory Limit: $WORKER_MEMORY_LIMIT per worker"
    echo "    Worker Group: $WINDMILL_WORKER_GROUP"
    
    # Show resource recommendations
    echo
    echo "  Scaling Recommendations:"
    local cpu_cores
    if cpu_cores=$(nproc 2>/dev/null); then
        echo "    CPU Cores: $cpu_cores (recommend 1 worker per core)"
        echo "    Suggested Range: 1-$cpu_cores workers"
    fi
    
    local memory_gb
    if memory_gb=$(system::get_memory_gb 2>/dev/null); then
        local max_workers=$((memory_gb / 2))  # 2GB per worker
        echo "    Available Memory: ${memory_gb}GB (${max_workers} workers max)"
    fi
}

#######################################
# Restart all worker containers
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::restart_workers() {
    if ! windmill::is_installed; then
        log::error "Windmill is not installed"
        return 1
    fi
    
    log::header "🔄 Restarting Windmill Workers"
    
    # Get current worker count to maintain after restart
    local current_count
    current_count=$(windmill::get_worker_count)
    
    log::info "Restarting $current_count worker containers..."
    
    # Restart default workers
    if ! windmill::compose_cmd restart windmill-worker; then
        log::error "Failed to restart default workers"
        return 1
    fi
    
    # Restart native worker if enabled
    if [[ "$DISABLE_NATIVE_WORKER" != "true" ]]; then
        if docker ps --format "{{.Names}}" | grep -q "${WINDMILL_PROJECT_NAME}-worker-native"; then
            if ! windmill::compose_cmd restart windmill-worker-native; then
                log::error "Failed to restart native worker"
                return 1
            fi
        fi
    fi
    
    # Wait for workers to be ready
    log::info "Waiting for workers to restart..."
    sleep 10
    
    # Verify workers are running
    local new_count
    new_count=$(windmill::get_worker_count)
    
    if [[ "$new_count" -eq "$current_count" ]]; then
        log::success "✅ Workers restarted successfully"
        windmill::show_worker_status
    else
        log::error "❌ Some workers failed to restart - Expected: $current_count, Running: $new_count"
        return 1
    fi
    
    return 0
}

#######################################
# Monitor worker performance and health
# Arguments:
#   $1 - Duration in seconds (optional, default: 30)
#######################################
windmill::monitor_workers() {
    local duration="${1:-30}"
    
    if ! windmill::is_running; then
        log::error "Windmill is not running"
        return 1
    fi
    
    log::info "📊 Monitoring Workers for ${duration}s (Press Ctrl+C to stop early)"
    echo
    
    local end_time=$(($(date +%s) + duration))
    
    while [[ $(date +%s) -lt $end_time ]]; do
        clear
        echo "🌪️  Windmill Worker Monitor - $(date)"
        echo "================================================"
        echo
        
        # Show current worker status
        windmill::show_worker_status
        
        echo
        echo "Real-time Resource Usage:"
        
        # Get worker container IDs
        local worker_ids
        worker_ids=$(docker ps --filter "name=${WINDMILL_PROJECT_NAME}-worker" -q)
        
        if [[ -n "$worker_ids" ]]; then
            docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" $worker_ids
        else
            echo "No workers running"
        fi
        
        echo
        echo "Press Ctrl+C to stop monitoring..."
        
        sleep 5
    done
    
    log::success "Monitoring completed"
}

#######################################
# Auto-scale workers based on system resources
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::auto_scale_workers() {
    if ! windmill::is_installed; then
        log::error "Windmill is not installed"
        return 1
    fi
    
    log::info "🤖 Auto-scaling workers based on system resources..."
    
    # Get system information
    local cpu_cores memory_gb
    cpu_cores=$(nproc 2>/dev/null || echo "2")
    memory_gb=$(system::get_memory_gb 2>/dev/null || echo "4")
    
    # Calculate optimal worker count
    local optimal_workers
    
    # Base on CPU cores (1 worker per core, max 10)
    local cpu_based_workers=$cpu_cores
    if [[ $cpu_based_workers -gt 10 ]]; then
        cpu_based_workers=10
    fi
    
    # Base on memory (2GB per worker, reserve 2GB for system)
    local memory_based_workers=$((memory_gb / 2 - 1))
    if [[ $memory_based_workers -lt 1 ]]; then
        memory_based_workers=1
    fi
    
    # Use the smaller of the two
    if [[ $cpu_based_workers -lt $memory_based_workers ]]; then
        optimal_workers=$cpu_based_workers
    else
        optimal_workers=$memory_based_workers
    fi
    
    # Ensure minimum of 1 worker
    if [[ $optimal_workers -lt 1 ]]; then
        optimal_workers=1
    fi
    
    log::info "System Analysis:"
    log::info "  CPU Cores: $cpu_cores"
    log::info "  Memory: ${memory_gb}GB"
    log::info "  CPU-based workers: $cpu_based_workers"
    log::info "  Memory-based workers: $memory_based_workers"
    log::info "  Optimal workers: $optimal_workers"
    
    # Scale to optimal count
    windmill::scale_workers "$optimal_workers"
}

#######################################
# Show worker logs with filtering
# Arguments:
#   $1 - Worker name pattern (optional, default: all workers)
#   $2 - Follow flag (optional, default: false)
#######################################
windmill::worker_logs() {
    local pattern="${1:-}"
    local follow="${2:-false}"
    
    if ! windmill::is_installed; then
        log::error "Windmill is not installed"
        return 1
    fi
    
    local compose_args=("logs")
    
    if [[ "$follow" == "true" ]]; then
        compose_args+=("--follow")
    fi
    
    compose_args+=("--timestamps" "--tail=100")
    
    # Add worker services
    if [[ -n "$pattern" ]]; then
        # Filter workers by pattern
        local workers
        workers=$(docker ps --filter "name=${WINDMILL_PROJECT_NAME}-worker" --format "{{.Names}}" | grep "$pattern" || true)
        
        if [[ -z "$workers" ]]; then
            log::error "No workers found matching pattern: $pattern"
            return 1
        fi
        
        log::info "Showing logs for workers matching: $pattern"
    else
        # Show all worker logs
        compose_args+=("windmill-worker")
        if [[ "$DISABLE_NATIVE_WORKER" != "true" ]]; then
            compose_args+=("windmill-worker-native")
        fi
        
        log::info "Showing logs for all workers"
    fi
    
    windmill::compose_cmd "${compose_args[@]}"
}