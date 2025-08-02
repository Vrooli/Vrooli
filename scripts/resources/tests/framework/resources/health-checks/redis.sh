#!/bin/bash
# ====================================================================
# Redis Health Check
# ====================================================================

check_redis_health() {
    local port="${1:-6380}"
    local response
    
    # If no port provided, try to find Redis container and get its port
    if [[ "$port" == "6380" ]]; then
        # Find Redis container (might be named differently like vrooli-redis-resource)
        local redis_container
        redis_container=$(docker ps --format "{{.Names}}" 2>/dev/null | grep -E "(redis|^redis$)" | head -1)
        
        if [[ -n "$redis_container" ]]; then
            # Get the port mapping
            local port_mapping
            port_mapping=$(docker port "$redis_container" 2>/dev/null | head -1)
            if [[ -n "$port_mapping" ]]; then
                # Extract host port from format: 6379/tcp -> 0.0.0.0:6380
                port=$(echo "$port_mapping" | sed 's/.*://g')
            fi
        fi
    fi
    
    # Redis uses its own protocol, not HTTP. Test with Redis PING command
    # Using netcat to send Redis protocol PING command
    response=$(timeout 5 bash -c 'echo -e "*1\r\n\$4\r\nPING\r\n" | nc localhost '"$port"' 2>/dev/null' | head -1 | tr -d '\r\n')
    if [[ "$response" == "+PONG" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}