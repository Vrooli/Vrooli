#!/bin/bash
# ====================================================================
# Generic HTTP Health Check
# ====================================================================
#
# Fallback health check for resources that expose standard HTTP endpoints
#
# ====================================================================

check_generic_health() {
    local resource="$1"
    local port="$2"
    local response
    
    # Try common health endpoints
    for endpoint in "/" "/health" "/healthz" "/ping" "/api/health" "/status"; do
        response=$(curl -s --max-time 10 "http://localhost:${port}${endpoint}" 2>/dev/null)
        if [[ $? -eq 0 && -n "$response" ]]; then
            echo "healthy"
            return
        fi
    done
    
    echo "unreachable"
}

# Specific implementations using generic pattern

check_whisper_health() {
    local port="${1:-8090}"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/health" 2>/dev/null)
    if [[ $? -eq 0 ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_n8n_health() {
    local port="${1:-5678}"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/healthz" 2>/dev/null)
    if [[ $? -eq 0 ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_browserless_health() {
    local port="${1:-4110}"
    local response
    
    # Browserless provides a /pressure endpoint for health checks
    response=$(curl -s --max-time 10 "http://localhost:${port}/pressure" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        # Check if response contains expected fields
        if echo "$response" | jq -e '.pressure' >/dev/null 2>&1; then
            echo "healthy"
        else
            echo "unhealthy"
        fi
    else
        echo "unreachable"
    fi
}

check_minio_health() {
    local port="${1:-9000}"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/minio/health/live" 2>/dev/null)
    if [[ $? -eq 0 ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_qdrant_health() {
    local port="${1:-6333}"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_vault_health() {
    local port="${1:-8200}"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/v1/sys/health" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_node_red_health() {
    local port="${1:-1880}"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_huginn_health() {
    local port="${1:-4111}"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_comfyui_health() {
    local port="${1:-8188}"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_searxng_health() {
    local port="${1:-9200}"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/healthz" 2>/dev/null)
    if [[ $? -eq 0 ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}