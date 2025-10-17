#!/bin/bash
# Airbyte Test Library

set -euo pipefail

# Test: smoke (quick health check)
test_smoke() {
    log_info "Running smoke tests..."
    
    local method
    method=$(detect_deployment_method)
    
    if [[ "$method" == "abctl" ]]; then
        # For abctl deployment
        local ABCTL="${DATA_DIR}/abctl"
        
        # Check if abctl exists
        if [[ ! -f "$ABCTL" ]]; then
            log_error "abctl CLI not found"
            return 1
        fi
        
        # Check cluster status
        if ! "$ABCTL" local status &> /dev/null; then
            log_error "Airbyte cluster is not running"
            return 1
        fi
        
        # Check if control plane container exists
        if ! docker ps --format "{{.Names}}" | grep -q "airbyte-abctl-control-plane"; then
            log_error "Airbyte control plane container is not running"
            return 1
        fi
        
        # Check pod health
        local pod_status
        pod_status=$(docker exec airbyte-abctl-control-plane sh -c 'kubectl -n airbyte-abctl get pods --no-headers' 2>&1 || echo "ERROR")
        
        if [[ "$pod_status" == "ERROR" ]] || [[ "$pod_status" == *"Error"* ]]; then
            log_error "Failed to check pod status"
            return 1
        fi
        
        # Check if we have running pods (at least server, worker, temporal, db should be running)
        local running_pods
        running_pods=$(echo "$pod_status" | grep -c "Running" || echo "0")
        
        if [[ "$running_pods" -lt "4" ]]; then
            log_error "Not enough Airbyte pods are running (found $running_pods, expected at least 4)"
            return 1
        fi
        
        # Check webapp accessibility
        if ! timeout 5 curl -sf "http://localhost:${AIRBYTE_WEBAPP_PORT}" > /dev/null; then
            log_error "Webapp not accessible on port ${AIRBYTE_WEBAPP_PORT}"
            return 1
        fi
        
        log_info "Airbyte cluster is healthy"
    else
        # For docker-compose deployment
        if ! docker ps | grep -q "airbyte-server"; then
            log_error "Airbyte server is not running"
            return 1
        fi
        
        # Check health endpoint
        if ! timeout 5 curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/health" > /dev/null; then
            log_error "Health check failed"
            return 1
        fi
    fi
    
    log_info "Smoke tests passed"
    return 0
}

# Test: integration
test_integration() {
    log_info "Running integration tests..."
    
    local method
    method=$(detect_deployment_method)
    
    # Test webapp accessibility first (works for both methods)
    log_info "Testing webapp..."
    if ! timeout 5 curl -sf "http://localhost:${AIRBYTE_WEBAPP_PORT}" > /dev/null; then
        log_error "Webapp is not accessible"
        return 1
    fi
    
    if [[ "$method" == "abctl" ]]; then
        # For abctl, test through kubectl port-forward or ingress
        log_info "Testing Airbyte services in Kubernetes cluster..."
        
        # Check all critical pods are running
        local critical_pods=("server" "worker" "temporal" "db")
        for pod in "${critical_pods[@]}"; do
            if ! docker exec airbyte-abctl-control-plane sh -c "kubectl -n airbyte-abctl get pods 2>/dev/null" | grep -q "$pod.*Running"; then
                log_error "Critical pod '$pod' is not running"
                return 1
            fi
        done
        
        # Test abctl credentials command
        local ABCTL="${DATA_DIR}/abctl"
        if ! "$ABCTL" local credentials &> /dev/null; then
            log_error "Failed to retrieve credentials"
            return 1
        fi
        
        log_info "Kubernetes cluster tests passed"
    else
        # For docker-compose, test API endpoints directly
        log_info "Testing API endpoints..."
        
        log_info "Testing source definitions endpoint..."
        if ! curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/source_definitions/list" > /dev/null; then
            log_error "Failed to list source definitions"
            return 1
        fi
        
        log_info "Testing destination definitions endpoint..."
        if ! curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/destination_definitions/list" > /dev/null; then
            log_error "Failed to list destination definitions"
            return 1
        fi
        
        log_info "Testing workspace endpoint..."
        if ! curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/workspaces/list" > /dev/null; then
            log_error "Failed to list workspaces"
            return 1
        fi
    fi
    
    log_info "Integration tests passed"
    return 0
}

# Test: unit
test_unit() {
    log_info "Running unit tests..."
    
    # Test configuration loading
    if [[ -z "${AIRBYTE_VERSION:-}" ]]; then
        log_error "AIRBYTE_VERSION not set"
        return 1
    fi
    
    if [[ -z "${AIRBYTE_WEBAPP_PORT:-}" ]]; then
        log_error "AIRBYTE_WEBAPP_PORT not set"
        return 1
    fi
    
    # Test directory structure (resource dir, not data dir which is created on install)
    if [[ ! -d "${RESOURCE_DIR}/lib" ]]; then
        log_error "lib directory not found"
        return 1
    fi
    
    if [[ ! -d "${RESOURCE_DIR}/config" ]]; then
        log_error "config directory not found"
        return 1
    fi
    
    log_info "Unit tests passed"
    return 0
}

# Test: all
test_all() {
    local failed=0
    
    test_smoke || failed=$((failed + 1))
    test_integration || failed=$((failed + 1))
    test_unit || failed=$((failed + 1))
    
    if [[ $failed -gt 0 ]]; then
        log_error "$failed test suite(s) failed"
        return 1
    fi
    
    log_info "All tests passed"
    return 0
}