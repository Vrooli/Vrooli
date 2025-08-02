#!/bin/bash
# ====================================================================
# Ollama Health Check
# ====================================================================

check_ollama_health() {
    local port="${1:-11434}"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/api/tags" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        if echo "$response" | jq -e '.models' >/dev/null 2>&1; then
            echo "healthy"
        else
            echo "unhealthy"
        fi
    else
        echo "unreachable"
    fi
}