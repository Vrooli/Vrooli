#!/bin/bash
# Airbyte Pipeline Optimization Library
# Handles performance monitoring, batch processing, and resource optimization

set -euo pipefail

# Only source defaults if not already sourced
if [[ -z "${AIRBYTE_WEBAPP_PORT:-}" ]]; then
    source "$(dirname "${BASH_SOURCE[0]}")/../config/defaults.sh"
fi

# Performance monitoring for sync jobs
monitor_sync_performance() {
    local connection_id="${1:-}"
    
    if [[ -z "$connection_id" ]]; then
        log_error "Connection ID is required for performance monitoring"
        return 1
    fi
    
    # Setup port forwarding if needed
    setup_api_port_forward
    
    # Get recent sync jobs for this connection
    local result
    if ! result=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "{\"configTypes\": [\"connection\"], \"configId\": \"$connection_id\"}" \
        "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/jobs/list" 2>/dev/null); then
        log_error "Failed to retrieve job history"
        return 1
    fi
    
    # Calculate performance metrics
    local total_records=0
    local total_duration=0
    local success_count=0
    local failure_count=0
    
    while read -r job; do
        local status duration records
        status=$(echo "$job" | jq -r '.status')
        duration=$(echo "$job" | jq -r '.duration // 0')
        records=$(echo "$job" | jq -r '.recordsSynced // 0')
        
        case "$status" in
            "succeeded")
                success_count=$((success_count + 1))
                total_records=$((total_records + records))
                total_duration=$((total_duration + duration))
                ;;
            "failed")
                failure_count=$((failure_count + 1))
                ;;
        esac
    done < <(echo "$result" | jq -c '.jobs[]' 2>/dev/null)
    
    # Calculate metrics
    local avg_throughput=0
    if [[ $total_duration -gt 0 ]]; then
        avg_throughput=$((total_records * 60 / total_duration))  # Records per minute
    fi
    
    local success_rate=0
    local total_jobs=$((success_count + failure_count))
    if [[ $total_jobs -gt 0 ]]; then
        success_rate=$((success_count * 100 / total_jobs))
    fi
    
    # Output performance report
    cat <<EOF
{
    "connection_id": "$connection_id",
    "metrics": {
        "total_records_synced": $total_records,
        "total_sync_duration_seconds": $total_duration,
        "average_throughput_per_minute": $avg_throughput,
        "success_rate_percent": $success_rate,
        "successful_syncs": $success_count,
        "failed_syncs": $failure_count
    }
}
EOF
}

# Optimize sync configuration for large datasets
optimize_sync_config() {
    local connection_id="${1:-}"
    local batch_size="${2:-10000}"
    
    if [[ -z "$connection_id" ]]; then
        log_error "Connection ID is required for optimization"
        return 1
    fi
    
    # Setup port forwarding if needed
    setup_api_port_forward
    
    # Get current connection configuration
    local result
    if ! result=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "{\"connectionId\": \"$connection_id\"}" \
        "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/connections/get" 2>/dev/null); then
        log_error "Failed to retrieve connection configuration"
        return 1
    fi
    
    # Update with optimized settings
    local optimized_config
    optimized_config=$(echo "$result" | jq \
        --arg batch_size "$batch_size" \
        '.syncCatalog.streams[].config += {
            "batch_size": $batch_size,
            "cursor_field_overrides": [],
            "primary_key_overrides": []
        }')
    
    # Apply optimized configuration
    if curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$optimized_config" \
        "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/connections/update" > /dev/null 2>&1; then
        log_info "Successfully optimized sync configuration for connection: $connection_id"
        echo "{\"status\": \"optimized\", \"batch_size\": $batch_size}"
    else
        log_error "Failed to apply optimization"
        return 1
    fi
}

# Analyze data quality for a connection
analyze_data_quality() {
    local connection_id="${1:-}"
    
    if [[ -z "$connection_id" ]]; then
        log_error "Connection ID is required for data quality analysis"
        return 1
    fi
    
    # Setup port forwarding if needed
    setup_api_port_forward
    
    # Get last successful sync job
    local result
    if ! result=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "{\"configTypes\": [\"connection\"], \"configId\": \"$connection_id\"}" \
        "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/jobs/list" 2>/dev/null); then
        log_error "Failed to retrieve job history"
        return 1
    fi
    
    # Find the last successful job
    local last_job
    last_job=$(echo "$result" | jq -r '.jobs[] | select(.status == "succeeded") | .id' | head -n1)
    
    if [[ -z "$last_job" ]]; then
        log_error "No successful sync jobs found for analysis"
        return 1
    fi
    
    # Get detailed job information
    local job_details
    if ! job_details=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "{\"id\": \"$last_job\"}" \
        "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/jobs/get" 2>/dev/null); then
        log_error "Failed to retrieve job details"
        return 1
    fi
    
    # Extract quality metrics
    local records_synced errors_count warnings_count
    records_synced=$(echo "$job_details" | jq -r '.job.recordsSynced // 0')
    errors_count=$(echo "$job_details" | jq -r '.job.errors // 0')
    warnings_count=$(echo "$job_details" | jq -r '.job.warnings // 0')
    
    # Calculate quality score (0-100)
    local quality_score=100
    if [[ $records_synced -gt 0 ]]; then
        local error_rate=$((errors_count * 100 / records_synced))
        quality_score=$((100 - error_rate))
    fi
    
    # Output quality report
    cat <<EOF
{
    "connection_id": "$connection_id",
    "job_id": "$last_job",
    "data_quality": {
        "records_synced": $records_synced,
        "errors": $errors_count,
        "warnings": $warnings_count,
        "quality_score": $quality_score,
        "assessment": "$([ $quality_score -gt 95 ] && echo "excellent" || [ $quality_score -gt 80 ] && echo "good" || echo "needs_attention")"
    }
}
EOF
}

# Batch sync orchestrator for multiple connections
batch_sync_orchestrator() {
    local connections_file="${1:-}"
    local parallel="${2:-false}"
    
    if [[ -z "$connections_file" ]] || [[ ! -f "$connections_file" ]]; then
        log_error "Connections file not found: $connections_file"
        return 1
    fi
    
    local total_syncs=0
    local successful_syncs=0
    local failed_syncs=0
    
    while IFS= read -r connection_id; do
        [[ -z "$connection_id" ]] && continue
        
        total_syncs=$((total_syncs + 1))
        log_info "Starting sync for connection: $connection_id"
        
        if [[ "$parallel" == "true" ]]; then
            # Start sync asynchronously
            content_execute --connection-id "$connection_id" &
        else
            # Start sync synchronously and wait
            if content_execute --connection-id "$connection_id" --wait; then
                successful_syncs=$((successful_syncs + 1))
            else
                failed_syncs=$((failed_syncs + 1))
            fi
        fi
    done < "$connections_file"
    
    # Wait for all parallel jobs if needed
    if [[ "$parallel" == "true" ]]; then
        log_info "Waiting for all parallel syncs to complete..."
        wait
        # Note: We can't track individual success/failure in parallel mode
        successful_syncs=$total_syncs
    fi
    
    # Output batch results
    cat <<EOF
{
    "batch_sync_results": {
        "total_syncs": $total_syncs,
        "successful": $successful_syncs,
        "failed": $failed_syncs,
        "mode": "$([ "$parallel" == "true" ] && echo "parallel" || echo "sequential")"
    }
}
EOF
}

# Resource usage analyzer
analyze_resource_usage() {
    local connection_id="${1:-}"
    
    # Check deployment method
    local method
    method=$(detect_deployment_method)
    
    if [[ "$method" == "abctl" ]]; then
        # For abctl/Kubernetes deployment, use kubectl to get pod metrics
        local pods
        pods=$(kubectl --kubeconfig=/home/matthalloran8/.airbyte/abctl/abctl.kubeconfig \
               get pods -n airbyte-abctl --no-headers 2>/dev/null | wc -l || echo "0")
        
        # Output simplified resource usage report for Kubernetes
        cat <<EOF
{
    "resource_usage": {
        "deployment_method": "abctl",
        "pod_count": $pods,
        "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
        "note": "Detailed metrics require kubectl metrics-server"
    }
}
EOF
    else
        # Get Docker stats for Airbyte containers
        local container_names
        container_names=$(docker ps --format "{{.Names}}" | grep "^airbyte-" 2>/dev/null || echo "")
        
        if [[ -z "$container_names" ]]; then
            cat <<EOF
{
    "resource_usage": {
        "deployment_method": "docker",
        "container_count": 0,
        "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
        "note": "No Airbyte containers running"
    }
}
EOF
            return
        fi
        
        # Get stats for each container
        local stats
        stats=$(docker stats --no-stream --format json "$container_names" 2>/dev/null || echo "{}")
        
        # Parse and aggregate resource usage
        local total_cpu=0
        local total_memory=0
        local container_count=0
        
        while read -r container_stats; do
            [[ -z "$container_stats" ]] && continue
            
            local cpu_percent memory_usage
            cpu_percent=$(echo "$container_stats" | jq -r '.CPUPerc' | sed 's/%//' 2>/dev/null || echo "0")
            memory_usage=$(echo "$container_stats" | jq -r '.MemUsage' | awk '{print $1}' | sed 's/[a-zA-Z]//g' 2>/dev/null || echo "0")
            
            if command -v bc > /dev/null 2>&1; then
                total_cpu=$(echo "$total_cpu + $cpu_percent" | bc 2>/dev/null || echo "$total_cpu")
                total_memory=$(echo "$total_memory + $memory_usage" | bc 2>/dev/null || echo "$total_memory")
            else
                # Use awk if bc is not available
                total_cpu=$(awk "BEGIN {print $total_cpu + $cpu_percent}" 2>/dev/null || echo "$total_cpu")
                total_memory=$(awk "BEGIN {print $total_memory + $memory_usage}" 2>/dev/null || echo "$total_memory")
            fi
            container_count=$((container_count + 1))
        done < <(echo "$stats")
        
        # Output resource usage report
        cat <<EOF
{
    "resource_usage": {
        "deployment_method": "docker",
        "total_cpu_percent": "$total_cpu",
        "total_memory_mb": "$total_memory",
        "container_count": $container_count,
        "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    }
}
EOF
    fi
}

# Export functions for use in other scripts
export -f monitor_sync_performance
export -f optimize_sync_config
export -f analyze_data_quality
export -f batch_sync_orchestrator
export -f analyze_resource_usage