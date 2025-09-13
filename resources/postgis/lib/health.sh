#!/bin/bash
# PostGIS Health Server Functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/*}/../.." && builtin pwd)}"
POSTGIS_HEALTH_LIB_DIR="${APP_ROOT}/resources/postgis/lib"

# Source dependencies
source "${POSTGIS_HEALTH_LIB_DIR}/common.sh"

# PostGIS configuration
POSTGIS_CONTAINER="postgis-main"
POSTGIS_HEALTH_PORT="${POSTGIS_HEALTH_PORT:-5435}"

#######################################
# Start health check HTTP server
# Returns: 0 on success, 1 on failure
#######################################
postgis::health::start_server() {
    local health_pid_file="${POSTGIS_DATA_DIR}/health_server.pid"
    
    # Check if already running
    if [[ -f "$health_pid_file" ]]; then
        local pid
        pid=$(cat "$health_pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            echo "Health server already running with PID $pid"
            return 0
        fi
        rm -f "$health_pid_file"
    fi
    
    # Start simple health server using netcat
    {
        while true; do
            # Check PostGIS health
            if docker exec "${POSTGIS_CONTAINER}" pg_isready -U vrooli -d spatial &>/dev/null; then
                # Get detailed status
                local version
                version=$(docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -t -c "SELECT PostGIS_Version();" 2>/dev/null | xargs | head -c 50)
                
                # Create response
                local response="{\"status\":\"healthy\",\"service\":\"postgis\",\"version\":\"${version:-unknown}\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}"
                local content_length=${#response}
                
                # Send HTTP response
                printf 'HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s' \
                    "$content_length" "$response" | nc -l -p "$POSTGIS_HEALTH_PORT" -q 1
            else
                # Send unhealthy response
                local response="{\"status\":\"unhealthy\",\"service\":\"postgis\",\"error\":\"Database not accessible\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}"
                local content_length=${#response}
                
                printf 'HTTP/1.1 503 Service Unavailable\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s' \
                    "$content_length" "$response" | nc -l -p "$POSTGIS_HEALTH_PORT" -q 1
            fi
            
            sleep 0.1
        done
    } &
    
    local server_pid=$!
    echo "$server_pid" > "$health_pid_file"
    
    # Verify server started
    sleep 1
    if kill -0 "$server_pid" 2>/dev/null; then
        echo "Health server started on port $POSTGIS_HEALTH_PORT (PID: $server_pid)"
        return 0
    else
        rm -f "$health_pid_file"
        echo "Failed to start health server"
        return 1
    fi
}

#######################################
# Stop health check HTTP server
# Returns: 0 on success
#######################################
postgis::health::stop_server() {
    local health_pid_file="${POSTGIS_DATA_DIR}/health_server.pid"
    
    if [[ -f "$health_pid_file" ]]; then
        local pid
        pid=$(cat "$health_pid_file")
        
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null
            sleep 1
            
            # Force kill if still running
            if kill -0 "$pid" 2>/dev/null; then
                kill -9 "$pid" 2>/dev/null
            fi
            
            echo "Health server stopped"
        fi
        
        rm -f "$health_pid_file"
    fi
    
    return 0
}

#######################################
# Check if health server is running
# Returns: 0 if running, 1 if not
#######################################
postgis::health::is_running() {
    local health_pid_file="${POSTGIS_DATA_DIR}/health_server.pid"
    
    if [[ -f "$health_pid_file" ]]; then
        local pid
        pid=$(cat "$health_pid_file")
        
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        fi
    fi
    
    return 1
}

# Export functions
export -f postgis::health::start_server
export -f postgis::health::stop_server
export -f postgis::health::is_running