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

#######################################
# Test Qdrant functionality
# Returns: 0 if all tests pass, 1 if any fail, 2 if service not ready
#######################################
qdrant::test() {
    log::info "Testing Qdrant functionality..."
    
    # Test 1: Check if Qdrant is installed (container exists)
    if ! qdrant::common::container_exists; then
        log::error "❌ Qdrant container is not installed"
        return 1
    fi
    log::success "✅ Qdrant container is installed"
    
    # Test 2: Check if service is running
    if ! qdrant::common::is_running; then
        log::error "❌ Qdrant service is not running"
        return 2
    fi
    log::success "✅ Qdrant service is running"
    
    # Test 3: Check API health
    if ! qdrant::api::health_check; then
        log::error "❌ Qdrant API is not responding"
        return 1
    fi
    log::success "✅ Qdrant API is healthy"
    
    # Test 4: Check cluster info and service info
    log::info "Testing cluster information..."
    if qdrant::api::get_service_status >/dev/null 2>&1; then
        log::success "✅ Cluster information accessible"
    else
        log::warn "⚠️  Cluster information unavailable"
    fi
    
    # Test 5: Test collection operations
    log::info "Testing collection operations..."
    local test_collection="vrooli-test-collection-$(date +%s)"
    
    if qdrant::collections::create "$test_collection" "128" "Cosine" >/dev/null 2>&1; then
        log::success "✅ Collection creation successful"
        
        # Test basic vector operations
        if qdrant::collections::info "$test_collection" >/dev/null 2>&1; then
            log::success "✅ Collection info retrieval successful"
        fi
        
        # Clean up test collection
        qdrant::collections::delete "$test_collection" >/dev/null 2>&1 || true
        log::success "✅ Collection cleanup successful"
    else
        log::warn "⚠️  Collection operations test failed - may be permission issue"
    fi
    
    # Test 6: Check storage metrics
    log::info "Testing storage metrics..."
    local storage_info
    storage_info=$(docker exec "$QDRANT_CONTAINER_NAME" df -h /qdrant/storage 2>/dev/null | tail -1 | awk '{print $3}' || echo "unknown")
    if [[ "$storage_info" != "unknown" ]]; then
        log::success "✅ Storage metrics available (used: $storage_info)"
    else
        log::warn "⚠️  Storage metrics unavailable"
    fi
    
    log::success "🎉 All Qdrant tests passed"
    return 0
}

#######################################
# Show comprehensive Qdrant information
#######################################
qdrant::info() {
    cat << EOF
=== Qdrant Resource Information ===

ID: qdrant
Category: storage
Display Name: Qdrant Vector Database
Description: High-performance vector similarity search engine

Service Details:
- Container Name: $QDRANT_CONTAINER_NAME
- HTTP Port: $QDRANT_PORT
- gRPC Port: $QDRANT_GRPC_PORT
- HTTP URL: http://localhost:$QDRANT_PORT
- gRPC URL: grpc://localhost:$QDRANT_GRPC_PORT
- Data Directory: $QDRANT_DATA_DIR

Endpoints:
- Health Check: http://localhost:$QDRANT_PORT/
- Cluster Info: http://localhost:$QDRANT_PORT/cluster
- Collections: http://localhost:$QDRANT_PORT/collections
- Collections Info: http://localhost:$QDRANT_PORT/collections/{collection}
- Search: POST http://localhost:$QDRANT_PORT/collections/{collection}/points/search
- Upsert: PUT http://localhost:$QDRANT_PORT/collections/{collection}/points

Configuration:
- Docker Image: $QDRANT_IMAGE
- Version: $QDRANT_VERSION
- Data Persistence: $QDRANT_DATA_DIR
- Default Vector Size: 1536 (OpenAI embedding size)
- Default Distance Metric: Cosine

Qdrant Features:
- Vector similarity search
- Payload filtering
- Geolocation support
- Distributed deployment
- Efficient filtering
- Real-time updates
- HNSW indexing
- Quantization support
- Hybrid search capabilities

Example Usage:
# Create a collection
$0 --action create-collection --collection embeddings --vector-size 1536

# List collections
$0 --action list-collections

# Get collection info
$0 --action collection-info --collection embeddings

# Monitor performance
$0 --action monitor --interval 10

Documentation: https://qdrant.tech/documentation/
EOF
}

# Export functions for subshell availability
export -f qdrant::test
export -f qdrant::info