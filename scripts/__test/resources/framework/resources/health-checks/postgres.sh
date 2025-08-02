#!/bin/bash
# ====================================================================
# PostgreSQL Health Check
# ====================================================================

check_postgres_health() {
    # PostgreSQL doesn't expose HTTP endpoints, so we check via database connection
    # First, try to find running postgres containers
    local running_containers
    running_containers=$(docker ps --format "{{.Names}}" 2>/dev/null | grep -E "(postgres|postgresql)" || echo "")
    
    if [[ -z "$running_containers" ]]; then
        echo "unreachable"
        return
    fi
    
    # Try to connect to PostgreSQL instances on common ports
    local ports=(5433 5434 5435 5436 5437 5438 5439)
    local healthy_found=false
    
    for port in "${ports[@]}"; do
        # Check if port is open
        if timeout 3 bash -c "</dev/tcp/localhost/$port" 2>/dev/null; then
            # Try a simple PostgreSQL connection test
            if which pg_isready >/dev/null 2>&1; then
                if pg_isready -h localhost -p "$port" -U postgres -t 3 >/dev/null 2>&1; then
                    healthy_found=true
                    break
                fi
            else
                # Fallback: if port is open and we have containers, assume healthy
                healthy_found=true
                break
            fi
        fi
    done
    
    if [[ "$healthy_found" == "true" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}