#!/usr/bin/env bash
################################################################################
# SageMath Health Check Implementation
# 
# Provides health endpoint functionality for v2.0 compliance
################################################################################

# Health check endpoint implementation
sagemath::health::check() {
    local format="${1:-json}"
    local start_time=$(date +%s%N)
    
    # Check container status
    local container_status="stopped"
    local jupyter_status="unavailable"
    local sage_status="unavailable"
    local overall_status="unhealthy"
    
    if sagemath_container_running; then
        container_status="running"
        
        # Check Jupyter API
        if timeout 2 curl -sf "http://localhost:${SAGEMATH_PORT_JUPYTER}/api" > /dev/null 2>&1; then
            jupyter_status="healthy"
        fi
        
        # Check SageMath kernel
        if docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "print('ok')" > /dev/null 2>&1; then
            sage_status="healthy"
        fi
        
        if [[ "$jupyter_status" == "healthy" ]] && [[ "$sage_status" == "healthy" ]]; then
            overall_status="healthy"
        fi
    fi
    
    local end_time=$(date +%s%N)
    local response_time=$((($end_time - $start_time) / 1000000))
    
    if [[ "$format" == "json" ]]; then
        cat <<EOF
{
  "status": "$overall_status",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "response_time_ms": $response_time,
  "components": {
    "container": "$container_status",
    "jupyter": "$jupyter_status",
    "sage_kernel": "$sage_status"
  },
  "version": "10.7"
}
EOF
    else
        echo "Health Status: $overall_status"
        echo "Container: $container_status"
        echo "Jupyter: $jupyter_status"
        echo "SageMath: $sage_status"
        echo "Response Time: ${response_time}ms"
    fi
    
    # Return appropriate exit code
    [[ "$overall_status" == "healthy" ]] && return 0 || return 1
}

# Simple health endpoint for HTTP requests
sagemath::health::endpoint() {
    # This would typically be served by a small HTTP server
    # For now, we'll just provide the function that returns health data
    sagemath::health::check json
}