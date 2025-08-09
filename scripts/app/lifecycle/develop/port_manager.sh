#!/usr/bin/env bash
# Port conflict resolution and management for development environments
set -euo pipefail

APP_LIFECYCLE_DEVELOP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEVELOP_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/ports.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"

# Port configuration source of truth
declare -gA PORT_CONFIG=(
    ["api"]="${PORT_API:-${PORT_SERVER:-5329}}"
    ["ui"]="${PORT_UI:-3000}"
    ["db"]="${PORT_DB:-5432}"
    ["redis"]="${PORT_REDIS:-6379}"
    ["jobs_debug"]="9230"
    ["server_debug"]="9229"
)

# Find next available port starting from a given port
develop::find_alternative_port() {
    local start_port="$1"
    local max_tries="${2:-10}"
    
    for ((i=0; i<max_tries; i++)); do
        local try_port=$((start_port + i))
        if ! ports::is_port_in_use "$try_port"; then
            echo "$try_port"
            return 0
        fi
    done
    
    log::error "No available port found in range $start_port-$((start_port + max_tries))"
    exit "$ERROR_PORT_CONFLICT"
}

# Export coordinated ports for all services
develop::update_port_env() {
    local -n ports=$1
    
    # Primary port variables
    export PORT_API="${ports[api]}"
    export PORT_UI="${ports[ui]}"
    export PORT_DB="${ports[db]}"
    export PORT_REDIS="${ports[redis]}"
    
    # Ensure consistency across naming conventions
    export PORT_SERVER="$PORT_API"  # Alias for scripts using PORT_SERVER
    export VITE_PORT_API="$PORT_API"  # For UI to find server
    
    # Update derived URLs if ports changed
    if [[ "${ports[api]}" != "${PORT_CONFIG[api]}" ]] || 
       [[ "${ports[ui]}" != "${PORT_CONFIG[ui]}" ]]; then
        develop::update_derived_urls
    fi
    
    # Log final configuration
    log::info "Port configuration:"
    log::info "  API/Server: $PORT_API"
    log::info "  UI: $PORT_UI"  
    log::info "  Database: $PORT_DB"
    log::info "  Redis: $PORT_REDIS"
}

# Update derived URLs when ports change
develop::update_derived_urls() {
    # Determine hostnames based on target
    local db_host="127.0.0.1"
    local redis_host="127.0.0.1"
    
    # For native targets, always use localhost
    if [[ "${TARGET:-}" =~ ^native ]]; then
        db_host="127.0.0.1"
        redis_host="127.0.0.1"
    fi
    
    # Reconstruct URLs with new ports
    export DB_URL="postgresql://${DB_USER}:${DB_PASSWORD}@${db_host}:${PORT_DB}/${DB_NAME:-postgres}"
    export REDIS_URL="redis://:${REDIS_PASSWORD}@${redis_host}:${PORT_REDIS}"
    
    # Update API URL for consistency
    export API_URL="http://localhost:${PORT_API}/api"
    export SERVER_URL="http://localhost:${PORT_API}"
    
    log::info "Updated service URLs with new ports"
}

# Export all port environment variables without checking
develop::export_port_env() {
    # Export using current values in PORT_CONFIG
    export PORT_API="${PORT_CONFIG[api]}"
    export PORT_UI="${PORT_CONFIG[ui]}"
    export PORT_DB="${PORT_CONFIG[db]}"
    export PORT_REDIS="${PORT_CONFIG[redis]}"
    
    # Ensure consistency
    export PORT_SERVER="$PORT_API"
    export VITE_PORT_API="$PORT_API"
}

# Check and resolve port conflicts
develop::resolve_port_conflicts() {
    local target="${1:-docker}"
    local -A resolved_ports=()
    
    # For Docker, just ensure variables are exported
    if [[ ! "$target" =~ ^native ]]; then
        develop::export_port_env
        return 0
    fi
    
    log::info "Checking port availability for native development..."
    
    # Check each required port
    for service in "${!PORT_CONFIG[@]}"; do
        local port="${PORT_CONFIG[$service]}"
        local final_port="$port"
        
        if ports::is_port_in_use "$port"; then
            log::warning "$service port $port is in use"
            
            # Get PIDs using the port for logging
            local pids
            pids=$(ports::get_listening_pids "$port" || echo "unknown")
            log::info "Process(es) using port $port: $pids"
            
            # Strategy based on service criticality
            case "$service" in
                "api"|"ui")
                    # Critical services - must coordinate
                    if flow::is_yes "${FORCE_KILL_PORTS:-}"; then
                        log::info "Force killing processes on port $port"
                        ports::kill "$port"
                        final_port="$port"
                    else
                        final_port=$(develop::find_alternative_port "$port" 10)
                        log::info "Using alternative port $final_port for $service"
                    fi
                    ;;
                "db"|"redis")
                    # Infrastructure - prompt to free unless forced
                    if flow::is_yes "${FORCE_KILL_PORTS:-}"; then
                        ports::kill "$port"
                        final_port="$port"
                    elif flow::is_yes "${FORCE_PORTS:-}"; then
                        final_port=$(develop::find_alternative_port "$port" 5)
                        log::info "Using alternative port $final_port for $service"
                    else
                        # Interactive prompt
                        ports::check_and_free "$port" "${YES:-}"
                        final_port="$port"
                    fi
                    ;;
                *)
                    # Debug ports - auto-increment
                    final_port=$(develop::find_alternative_port "$port" 10)
                    log::info "Using alternative port $final_port for $service"
                    ;;
            esac
        fi
        
        resolved_ports[$service]="$final_port"
    done
    
    # Update environment with resolved ports
    develop::update_port_env resolved_ports
    
    # Show summary if any ports changed
    local changes=false
    for service in "${!PORT_CONFIG[@]}"; do
        if [[ "${resolved_ports[$service]}" != "${PORT_CONFIG[$service]}" ]]; then
            changes=true
            break
        fi
    done
    
    if [[ "$changes" == "true" ]]; then
        log::warning "⚠️  Port configuration has been adjusted due to conflicts"
        log::warning "⚠️  Make sure to use these ports when connecting to services"
    fi
}

# Utility function to display current port status
develop::show_port_status() {
    log::header "Current Port Status"
    
    for service in "${!PORT_CONFIG[@]}"; do
        local port="${PORT_CONFIG[$service]}"
        local status="✅ Available"
        local pids=""
        
        if ports::is_port_in_use "$port"; then
            status="❌ In Use"
            pids=$(ports::get_listening_pids "$port" || echo "unknown")
            pids=" (PID: $pids)"
        fi
        
        printf "%-15s: %5s %s%s\n" "$service" "$port" "$status" "$pids"
    done
}

# Export functions for use by other scripts
export -f develop::resolve_port_conflicts
export -f develop::export_port_env
export -f develop::show_port_status