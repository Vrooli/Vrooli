#!/usr/bin/env bash
# Qdrant Status and Health Monitoring
# Functions for checking Qdrant health and status

#######################################
# Check overall Qdrant status
# Arguments:
#   $1 - Show detailed information (true/false, default: false)
# Returns: 0 if healthy, 1 if not
#######################################
qdrant::status::check() {
    local show_details="${1:-false}"
    local overall_status=0
    
    echo "=== Qdrant Status Check ==="
    echo
    
    # Check if Docker is available
    if ! qdrant::docker::check_docker; then
        return 1
    fi
    
    # Check container status
    local container_status
    container_status=$(qdrant::common::get_container_status)
    echo "Container Status: $container_status"
    
    if [[ "$container_status" == "Container does not exist" ]]; then
        echo "Status: ❌ Not installed"
        echo
        echo "Run '$0 --action install' to install Qdrant"
        return 1
    elif [[ "$container_status" == "Stopped" ]]; then
        echo "Status: ⚠️  Stopped"
        overall_status=1
    elif [[ "$container_status" == "Running" ]]; then
        # Check API health
        local api_health
        api_health=$(qdrant::api::detailed_health_check)
        echo "API Health: $api_health"
        
        if [[ "$api_health" == "✅ Healthy" ]]; then
            echo "Status: ✅ Running and healthy"
            
            # Show connection information
            echo
            echo "Connection Information:"
            echo "  REST API: ${QDRANT_BASE_URL}"
            echo "  gRPC API: ${QDRANT_GRPC_URL}"
            echo "  Web UI: ${QDRANT_BASE_URL}/dashboard"
            
            if [[ -n "${QDRANT_API_KEY:-}" ]]; then
                echo "  Authentication: API key configured"
            else
                echo "  Authentication: None (unauthenticated access)"
            fi
        else
            echo "Status: ⚠️  Running but unhealthy"
            overall_status=1
        fi
    fi
    
    # Show detailed information if requested
    if [[ "$show_details" == "true" && "$container_status" == "Running" ]]; then
        echo
        qdrant::status::show_detailed_info
    fi
    
    # Check port availability
    echo
    echo "Port Status:"
    if qdrant::common::is_port_available "${QDRANT_PORT}"; then
        echo "  REST Port ${QDRANT_PORT}: Available"
    else
        echo "  REST Port ${QDRANT_PORT}: In use"
    fi
    
    if qdrant::common::is_port_available "${QDRANT_GRPC_PORT}"; then
        echo "  gRPC Port ${QDRANT_GRPC_PORT}: Available"
    else
        echo "  gRPC Port ${QDRANT_GRPC_PORT}: In use"
    fi
    
    # Check disk space
    echo
    echo "Storage Status:"
    echo "  Data Directory: ${QDRANT_DATA_DIR}"
    
    if qdrant::common::check_disk_space; then
        local available_kb
        available_kb=$(df "${QDRANT_DATA_DIR}" | awk 'NR==2 {print $4}')
        local available_gb=$((available_kb / 1024 / 1024))
        echo "  Available Space: ${available_gb}GB ✅"
    else
        echo "  Available Space: Low ⚠️"
        overall_status=1
    fi
    
    echo
    return $overall_status
}

#######################################
# Show detailed service information
#######################################
qdrant::status::show_detailed_info() {
    if ! qdrant::common::is_running; then
        echo "Qdrant is not running - cannot show detailed information"
        return 1
    fi
    
    echo "=== Detailed Service Information ==="
    echo
    
    # Version and cluster info
    qdrant::api::get_service_status
    
    # Resource usage
    echo "Resource Usage:"
    qdrant::docker::get_resource_usage
    
    # Collections summary
    echo
    echo "Collections Summary:"
    local collections_count
    collections_count=$(qdrant::collections::count 2>/dev/null || echo "unknown")
    echo "  Total Collections: $collections_count"
    
    if [[ "$collections_count" != "unknown" && "$collections_count" -gt 0 ]]; then
        echo "  Collection List:"
        qdrant::collections::list_simple | sed 's/^/    /'
    fi
    
    echo
}

#######################################
# Run comprehensive diagnostics
# Returns: 0 if all checks pass, 1 if any issues found
#######################################
qdrant::status::diagnose() {
    local issues_found=0
    
    echo "=== Qdrant Diagnostic Report ==="
    echo "Generated: $(date)"
    echo
    
    # System checks
    echo "1. System Requirements:"
    if command -v docker >/dev/null 2>&1; then
        echo "   ✅ Docker is installed"
        local docker_version
        docker_version=$(docker --version | cut -d' ' -f3 | cut -d',' -f1)
        echo "      Version: $docker_version"
    else
        echo "   ❌ Docker is not installed"
        issues_found=1
    fi
    
    if docker info >/dev/null 2>&1; then
        echo "   ✅ Docker daemon is running"
    else
        echo "   ❌ Docker daemon is not running"
        issues_found=1
    fi
    
    # Check required tools
    echo
    echo "2. Required Tools:"
    local tools=("curl" "jq" "nc")
    for tool in "${tools[@]}"; do
        if command -v "$tool" >/dev/null 2>&1; then
            echo "   ✅ $tool is available"
        else
            echo "   ⚠️  $tool is not available (some features may not work)"
        fi
    done
    
    # Container checks
    echo
    echo "3. Container Status:"
    if qdrant::common::container_exists; then
        echo "   ✅ Container exists"
        
        if qdrant::common::is_running; then
            echo "   ✅ Container is running"
            
            # Check container health
            local health_status
            health_status=$(docker inspect "${QDRANT_CONTAINER_NAME}" --format='{{.State.Health.Status}}' 2>/dev/null || echo "unknown")
            echo "   Container Health: $health_status"
            
            if [[ "$health_status" != "healthy" && "$health_status" != "unknown" ]]; then
                issues_found=1
            fi
        else
            echo "   ⚠️  Container exists but is not running"
            issues_found=1
        fi
    else
        echo "   ❌ Container does not exist"
        issues_found=1
    fi
    
    # Port checks
    echo
    echo "4. Network Configuration:"
    if qdrant::common::check_ports; then
        echo "   ✅ All ports are available"
    else
        echo "   ⚠️  Port conflicts detected"
        issues_found=1
    fi
    
    # Storage checks
    echo
    echo "5. Storage Configuration:"
    if [[ -d "${QDRANT_DATA_DIR}" ]]; then
        echo "   ✅ Data directory exists: ${QDRANT_DATA_DIR}"
        
        if [[ -r "${QDRANT_DATA_DIR}" && -w "${QDRANT_DATA_DIR}" ]]; then
            echo "   ✅ Data directory is readable and writable"
        else
            echo "   ❌ Data directory permission issues"
            issues_found=1
        fi
    else
        echo "   ⚠️  Data directory does not exist: ${QDRANT_DATA_DIR}"
    fi
    
    if qdrant::common::check_disk_space; then
        echo "   ✅ Sufficient disk space available"
    else
        echo "   ⚠️  Low disk space"
        issues_found=1
    fi
    
    # API checks (only if container is running)
    if qdrant::common::is_running; then
        echo
        echo "6. API Connectivity:"
        qdrant::api::diagnose_connectivity || issues_found=1
    fi
    
    # Configuration checks
    echo
    echo "7. Configuration:"
    echo "   REST Port: ${QDRANT_PORT}"
    echo "   gRPC Port: ${QDRANT_GRPC_PORT}"
    echo "   Data Directory: ${QDRANT_DATA_DIR}"
    echo "   Container Name: ${QDRANT_CONTAINER_NAME}"
    
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        echo "   Authentication: API key configured ✅"
    else
        echo "   Authentication: None (development mode) ⚠️"
    fi
    
    # Summary
    echo
    echo "=== Diagnostic Summary ==="
    if [[ $issues_found -eq 0 ]]; then
        echo "✅ All checks passed - Qdrant appears to be configured correctly"
        return 0
    else
        echo "⚠️  $issues_found issue(s) found - see details above"
        return 1
    fi
}

#######################################
# Monitor Qdrant health continuously
# Arguments:
#   $1 - Interval in seconds between checks
#######################################
qdrant::status::monitor() {
    local interval="${1:-5}"
    
    echo "=== Qdrant Health Monitor ==="
    echo "Checking every ${interval} seconds. Press Ctrl+C to stop."
    echo
    
    local check_count=0
    
    while true; do
        check_count=$((check_count + 1))
        local timestamp
        timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        
        echo "[$timestamp] Check #$check_count"
        
        if qdrant::common::is_running; then
            local api_health
            api_health=$(qdrant::api::detailed_health_check)
            echo "  Container: Running"
            echo "  API: $api_health"
            
            if qdrant::api::health_check; then
                # Show basic metrics
                local collections_count
                collections_count=$(qdrant::collections::count 2>/dev/null || echo "N/A")
                echo "  Collections: $collections_count"
                
                # Show resource usage
                local cpu_usage mem_usage
                cpu_usage=$(docker stats "${QDRANT_CONTAINER_NAME}" --no-stream --format="{{.CPUPerc}}" 2>/dev/null || echo "N/A")
                mem_usage=$(docker stats "${QDRANT_CONTAINER_NAME}" --no-stream --format="{{.MemUsage}}" 2>/dev/null || echo "N/A")
                echo "  CPU: $cpu_usage, Memory: $mem_usage"
            fi
        else
            echo "  Status: Not running"
        fi
        
        echo
        sleep "$interval"
    done
}

#######################################
# Get a simple health status for automation
# Outputs: "healthy", "unhealthy", or "not-running"
# Returns: 0 if healthy, 1 otherwise
#######################################
qdrant::status::simple_health() {
    if ! qdrant::common::is_running; then
        echo "not-running"
        return 1
    fi
    
    if qdrant::api::health_check; then
        echo "healthy"
        return 0
    else
        echo "unhealthy"
        return 1
    fi
}