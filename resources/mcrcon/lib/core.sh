#!/usr/bin/env bash
# mcrcon core functionality

set -euo pipefail

# Get script directory
CORE_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly CORE_SCRIPT_DIR
if [[ -z "${RESOURCE_DIR:-}" ]]; then
    readonly RESOURCE_DIR="$(dirname "$CORE_SCRIPT_DIR")"
fi

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Logging functions
log_info() {
    if [[ -n "${MCRCON_LOG_FILE:-}" ]] && [[ "${MCRCON_LOG_FILE}" != "/dev/null" ]]; then
        mkdir -p "$(dirname "${MCRCON_LOG_FILE}")"
        echo "[INFO] $*" | tee -a "${MCRCON_LOG_FILE}"
    else
        echo "[INFO] $*"
    fi
}

log_error() {
    if [[ -n "${MCRCON_LOG_FILE:-}" ]] && [[ "${MCRCON_LOG_FILE}" != "/dev/null" ]]; then
        mkdir -p "$(dirname "${MCRCON_LOG_FILE}")"
        echo "[ERROR] $*" >&2 | tee -a "${MCRCON_LOG_FILE}"
    else
        echo "[ERROR] $*" >&2
    fi
}

log_debug() {
    if [[ "${MCRCON_DEBUG}" == "true" ]]; then
        if [[ -n "${MCRCON_LOG_FILE:-}" ]] && [[ "${MCRCON_LOG_FILE}" != "/dev/null" ]]; then
            mkdir -p "$(dirname "${MCRCON_LOG_FILE}")"
            echo "[DEBUG] $*" | tee -a "${MCRCON_LOG_FILE}"
        else
            echo "[DEBUG] $*"
        fi
    fi
}

# Ensure data directory exists
ensure_data_dir() {
    if [[ ! -d "${MCRCON_DATA_DIR}" ]]; then
        mkdir -p "${MCRCON_DATA_DIR}/bin"
        mkdir -p "${MCRCON_DATA_DIR}/logs"
        log_info "Created data directory: ${MCRCON_DATA_DIR}"
    fi
}

# Check if mcrcon binary is installed
is_installed() {
    [[ -f "${MCRCON_BINARY}" ]] && [[ -x "${MCRCON_BINARY}" ]]
}

# Install mcrcon
install_mcrcon() {
    log_info "Installing mcrcon..."
    
    ensure_data_dir
    
    # Download mcrcon binary
    local mcrcon_url="https://github.com/Tiiffi/mcrcon/releases/download/v0.7.2/mcrcon-0.7.2-linux-x86-64.tar.gz"
    local temp_dir=$(mktemp -d)
    
    log_info "Downloading mcrcon from GitHub..."
    if ! wget -q -O "${temp_dir}/mcrcon.tar.gz" "$mcrcon_url"; then
        log_error "Failed to download mcrcon"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Extract binary
    log_info "Extracting mcrcon binary..."
    if ! tar -xzf "${temp_dir}/mcrcon.tar.gz" -C "$temp_dir"; then
        log_error "Failed to extract mcrcon"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Move binary to installation directory
    if [[ -f "${temp_dir}/mcrcon" ]]; then
        # Ensure bin directory exists
        mkdir -p "$(dirname "${MCRCON_BINARY}")"
        mv "${temp_dir}/mcrcon" "${MCRCON_BINARY}"
        chmod +x "${MCRCON_BINARY}"
        log_info "mcrcon installed successfully to ${MCRCON_BINARY}"
    else
        log_error "mcrcon binary not found in archive"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Clean up
    rm -rf "$temp_dir"
    
    # Initialize server configuration file
    if [[ ! -f "${MCRCON_CONFIG_FILE}" ]]; then
        echo '{"servers": []}' > "${MCRCON_CONFIG_FILE}"
        log_info "Initialized server configuration file"
    fi
    
    # Create systemd service for health endpoint
    create_health_service
    
    return 0
}

# Uninstall mcrcon
uninstall_mcrcon() {
    log_info "Uninstalling mcrcon..."
    
    # Stop health service if running
    stop_health_service
    
    # Remove binary and data directory
    if [[ -d "${MCRCON_DATA_DIR}" ]]; then
        rm -rf "${MCRCON_DATA_DIR}"
        log_info "Removed mcrcon data directory"
    fi
    
    return 0
}

# Create health check service
create_health_service() {
    # Simple HTTP server for health checks using Python
    cat > "${MCRCON_DATA_DIR}/health_server.py" << 'EOF'
#!/usr/bin/env python3
import http.server
import socketserver
import json
import os
import subprocess
import sys

PORT = int(os.environ.get('MCRCON_HEALTH_PORT', 8025))

class HealthHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            status = self.check_health()
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps(status).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def check_health(self):
        # Check if mcrcon binary exists
        binary_path = os.environ.get('MCRCON_BINARY', '')
        binary_exists = os.path.isfile(binary_path) and os.access(binary_path, os.X_OK)
        
        return {
            'status': 'healthy' if binary_exists else 'unhealthy',
            'service': 'mcrcon',
            'binary_installed': binary_exists,
            'version': '1.0.0'
        }
    
    def log_message(self, format, *args):
        # Suppress default logging
        pass

with socketserver.TCPServer(("", PORT), HealthHandler) as httpd:
    print(f"Health server running on port {PORT}")
    httpd.serve_forever()
EOF
    chmod +x "${MCRCON_DATA_DIR}/health_server.py"
}

# Start health service
start_health_service() {
    if pgrep -f "health_server.py.*mcrcon" > /dev/null 2>&1; then
        log_info "Health service already running"
        return 0
    fi
    
    log_info "Starting health service on port ${MCRCON_HEALTH_PORT}..."
    mkdir -p "${MCRCON_DATA_DIR}/logs"
    nohup python3 "${MCRCON_DATA_DIR}/health_server.py" > "${MCRCON_DATA_DIR}/logs/health.log" 2>&1 &
    
    # Wait for service to start
    local count=0
    while ! timeout 1 curl -sf "http://localhost:${MCRCON_HEALTH_PORT}/health" > /dev/null 2>&1; do
        sleep 1
        count=$((count + 1))
        if [[ $count -ge 10 ]]; then
            log_error "Health service failed to start"
            return 1
        fi
    done
    
    log_info "Health service started successfully"
    return 0
}

# Stop health service
stop_health_service() {
    local pid=$(pgrep -f "health_server.py.*mcrcon" 2>/dev/null || true)
    if [[ -n "$pid" ]]; then
        kill "$pid" 2>/dev/null || true
        log_info "Health service stopped"
    fi
}

# Execute RCON command
execute_command() {
    local command="$1"
    local server="${2:-default}"
    
    if ! is_installed; then
        log_error "mcrcon is not installed"
        return 1
    fi
    
    local host="${MCRCON_HOST}"
    local port="${MCRCON_PORT}"
    local password="${MCRCON_PASSWORD}"
    
    # Check if using a named server from config
    if [[ "$server" != "default" ]] && [[ -f "${MCRCON_CONFIG_FILE}" ]]; then
        local server_config=$(jq -r --arg name "$server" '.servers[] | select(.name == $name)' "${MCRCON_CONFIG_FILE}")
        if [[ -n "$server_config" ]]; then
            host=$(echo "$server_config" | jq -r '.host')
            port=$(echo "$server_config" | jq -r '.port')
            password=$(echo "$server_config" | jq -r '.password')
            log_debug "Using server config: $server"
        else
            log_error "Server '$server' not found in configuration"
            echo "Available servers:"
            list_servers
            return 1
        fi
    fi
    
    if [[ -z "$password" ]]; then
        log_error "RCON password not configured"
        echo "Set MCRCON_PASSWORD environment variable or configure server"
        return 1
    fi
    
    # Execute command with retry logic
    log_debug "Executing command: $command on $host:$port"
    
    local attempts=0
    local max_attempts="${MCRCON_RETRY_ATTEMPTS:-3}"
    local retry_delay=2
    
    while [[ $attempts -lt $max_attempts ]]; do
        attempts=$((attempts + 1))
        
        if timeout "${MCRCON_TIMEOUT}" "${MCRCON_BINARY}" -H "$host" -P "$port" -p "$password" "$command" 2>/tmp/mcrcon_error.log; then
            return 0
        else
            local error_msg=$(cat /tmp/mcrcon_error.log 2>/dev/null || echo "Unknown error")
            
            # Check for specific error types
            if echo "$error_msg" | grep -q "Connection refused"; then
                log_error "Connection refused - server may not be running or RCON not enabled"
                return 1
            elif echo "$error_msg" | grep -q "Authentication failed"; then
                log_error "Authentication failed - check RCON password"
                return 1
            elif echo "$error_msg" | grep -q "timeout"; then
                log_error "Command timed out (attempt $attempts/$max_attempts)"
                if [[ $attempts -lt $max_attempts ]]; then
                    log_info "Retrying in ${retry_delay} seconds..."
                    sleep $retry_delay
                    retry_delay=$((retry_delay * 2))  # Exponential backoff
                    continue
                fi
            fi
            
            log_error "Failed to execute command: $command (attempt $attempts/$max_attempts)"
            if [[ $attempts -lt $max_attempts ]]; then
                sleep $retry_delay
                continue
            fi
        fi
    done
    
    log_error "Command failed after $max_attempts attempts"
    return 1
}

# Execute command on all configured servers
execute_all() {
    local command="$1"
    
    if ! is_installed; then
        log_error "mcrcon is not installed"
        return 1
    fi
    
    if [[ ! -f "${MCRCON_CONFIG_FILE}" ]]; then
        log_error "No servers configured"
        return 1
    fi
    
    local server_count=$(jq -r '.servers | length' "${MCRCON_CONFIG_FILE}")
    if [[ "$server_count" -eq 0 ]]; then
        log_error "No servers configured"
        return 1
    fi
    
    echo "Executing command on all servers..."
    echo "================================"
    
    local success_count=0
    local fail_count=0
    
    # Execute on each server
    while IFS= read -r server_name; do
        echo ""
        echo "Server: $server_name"
        if execute_command "$command" "$server_name"; then
            success_count=$((success_count + 1))
        else
            fail_count=$((fail_count + 1))
        fi
    done < <(jq -r '.servers[].name' "${MCRCON_CONFIG_FILE}")
    
    echo ""
    echo "================================"
    echo "Results: $success_count succeeded, $fail_count failed"
    
    return 0
}

# List configured servers
list_servers() {
    if [[ -f "${MCRCON_CONFIG_FILE}" ]]; then
        jq -r '.servers[] | "\(.name)\t\(.host):\(.port)\t\(.description // "")"' "${MCRCON_CONFIG_FILE}"
    else
        echo "No servers configured"
    fi
}

# Add server configuration
add_server() {
    local name="$1"
    local host="${2:-localhost}"
    local port="${3:-25575}"
    local password="${4:-}"
    local description="${5:-}"
    
    ensure_data_dir
    
    if [[ ! -f "${MCRCON_CONFIG_FILE}" ]]; then
        echo '{"servers": []}' > "${MCRCON_CONFIG_FILE}"
    fi
    
    # Add server to configuration
    local temp_file=$(mktemp)
    jq --arg name "$name" \
       --arg host "$host" \
       --arg port "$port" \
       --arg password "$password" \
       --arg description "$description" \
       '.servers += [{
           name: $name,
           host: $host,
           port: ($port | tonumber),
           password: $password,
           description: $description
       }]' "${MCRCON_CONFIG_FILE}" > "$temp_file"
    
    mv "$temp_file" "${MCRCON_CONFIG_FILE}"
    log_info "Added server: $name"
}

# Remove server configuration
remove_server() {
    local name="$1"
    
    if [[ ! -f "${MCRCON_CONFIG_FILE}" ]]; then
        log_error "No servers configured"
        return 1
    fi
    
    local temp_file=$(mktemp)
    jq --arg name "$name" '.servers |= map(select(.name != $name))' "${MCRCON_CONFIG_FILE}" > "$temp_file"
    mv "$temp_file" "${MCRCON_CONFIG_FILE}"
    log_info "Removed server: $name"
}

# Show status
show_status() {
    echo "mcrcon Resource Status"
    echo "====================="
    
    if is_installed; then
        echo "Binary: Installed (${MCRCON_BINARY})"
    else
        echo "Binary: Not installed"
    fi
    
    # Check health service
    if timeout 1 curl -sf "http://localhost:${MCRCON_HEALTH_PORT}/health" > /dev/null 2>&1; then
        echo "Health Service: Running (port ${MCRCON_HEALTH_PORT})"
    else
        echo "Health Service: Not running"
    fi
    
    # Show configured servers
    echo ""
    echo "Configured Servers:"
    list_servers
}

# Show logs
show_logs() {
    if [[ -f "${MCRCON_LOG_FILE}" ]]; then
        tail -n 50 "${MCRCON_LOG_FILE}"
    else
        echo "No logs available"
    fi
}

# Show credentials
show_credentials() {
    echo "mcrcon Connection Credentials"
    echo "============================"
    echo "Default Server:"
    echo "  Host: ${MCRCON_HOST}"
    echo "  Port: ${MCRCON_PORT}"
    echo "  Password: ${MCRCON_PASSWORD:-(not set)}"
    echo ""
    echo "To set password:"
    echo "  export MCRCON_PASSWORD='your_password'"
}

# Auto-discover Minecraft servers on local network
discover_servers() {
    log_info "Scanning for Minecraft servers..."
    
    local discovered_count=0
    local scan_ports="${1:-25565,25575}"  # Default MC port and RCON port
    local scan_range="${2:-127.0.0.1}"    # Default to localhost
    
    echo "Discovering Minecraft servers..."
    echo "================================"
    
    # Check common ports on localhost
    for port in $(echo "$scan_ports" | tr ',' ' '); do
        if timeout 1 bash -c "echo > /dev/tcp/127.0.0.1/$port" 2>/dev/null; then
            echo "Found server on 127.0.0.1:$port"
            discovered_count=$((discovered_count + 1))
            
            # Check if it's RCON port (usually 25575)
            if [[ "$port" == "25575" ]]; then
                echo "  Type: RCON port (likely)"
            elif [[ "$port" == "25565" ]]; then
                echo "  Type: Game port (likely)"
            fi
        fi
    done
    
    # Check Docker containers for Minecraft servers
    if command -v docker &> /dev/null; then
        log_debug "Checking Docker containers..."
        local containers=$(docker ps --format "table {{.Names}}\t{{.Ports}}" 2>/dev/null | grep -E "25565|25575" || true)
        if [[ -n "$containers" ]]; then
            echo ""
            echo "Docker Minecraft Containers:"
            echo "$containers"
            discovered_count=$((discovered_count + 1))
        fi
    fi
    
    # Check common Minecraft server process names
    local mc_processes=$(ps aux 2>/dev/null | grep -E "minecraft|spigot|paper|forge|fabric" | grep -v grep || true)
    if [[ -n "$mc_processes" ]]; then
        echo ""
        echo "Running Minecraft Processes:"
        echo "$mc_processes" | awk '{print $11}' | head -5
        discovered_count=$((discovered_count + 1))
    fi
    
    echo ""
    echo "================================"
    echo "Discovered $discovered_count potential server(s)"
    
    if [[ $discovered_count -eq 0 ]]; then
        echo ""
        echo "No Minecraft servers found. To start a test server:"
        echo "  docker run -d -p 25565:25565 -p 25575:25575 -e EULA=TRUE -e ENABLE_RCON=true -e RCON_PASSWORD=minecraft itzg/minecraft-server"
    else
        echo ""
        echo "To add a discovered server:"
        echo "  vrooli resource mcrcon content add <name> <host> <port> <password>"
    fi
    
    return 0
}

# Test RCON connection to a server
test_connection() {
    local host="${1:-$MCRCON_HOST}"
    local port="${2:-$MCRCON_PORT}"
    local password="${3:-$MCRCON_PASSWORD}"
    
    if ! is_installed; then
        log_error "mcrcon is not installed"
        return 1
    fi
    
    if [[ -z "$password" ]]; then
        log_error "Password required for connection test"
        return 1
    fi
    
    echo "Testing connection to $host:$port..."
    
    # Try to execute a simple command
    if timeout 5 "${MCRCON_BINARY}" -H "$host" -P "$port" -p "$password" "list" &>/dev/null; then
        echo "✓ Connection successful"
        return 0
    else
        echo "✗ Connection failed"
        echo "  Verify server is running and RCON is enabled"
        echo "  Check password is correct"
        return 1
    fi
}

# Player management functions
list_players() {
    local server="${1:-default}"
    
    echo "Online Players:"
    echo "==============="
    
    local response=$(execute_command "list" "$server")
    if [[ $? -eq 0 ]]; then
        echo "$response"
    else
        log_error "Failed to list players"
        return 1
    fi
}

player_info() {
    local player="$1"
    local server="${2:-default}"
    
    if [[ -z "$player" ]]; then
        log_error "Player name required"
        return 1
    fi
    
    echo "Player Information: $player"
    echo "========================="
    
    # Get player data
    execute_command "data get entity $player" "$server"
}

teleport_player() {
    local player="$1"
    local x="$2"
    local y="$3"
    local z="$4"
    local server="${5:-default}"
    
    if [[ -z "$player" ]] || [[ -z "$x" ]] || [[ -z "$y" ]] || [[ -z "$z" ]]; then
        log_error "Usage: teleport_player <player> <x> <y> <z> [server]"
        return 1
    fi
    
    echo "Teleporting $player to ($x, $y, $z)..."
    execute_command "tp $player $x $y $z" "$server"
}

kick_player() {
    local player="$1"
    local reason="${2:-Kicked by administrator}"
    local server="${3:-default}"
    
    if [[ -z "$player" ]]; then
        log_error "Player name required"
        return 1
    fi
    
    echo "Kicking player: $player"
    execute_command "kick $player $reason" "$server"
}

ban_player() {
    local player="$1"
    local reason="${2:-Banned by administrator}"
    local server="${3:-default}"
    
    if [[ -z "$player" ]]; then
        log_error "Player name required"
        return 1
    fi
    
    echo "Banning player: $player"
    execute_command "ban $player $reason" "$server"
}

give_item() {
    local player="$1"
    local item="$2"
    local count="${3:-1}"
    local server="${4:-default}"
    
    if [[ -z "$player" ]] || [[ -z "$item" ]]; then
        log_error "Usage: give_item <player> <item> [count] [server]"
        return 1
    fi
    
    echo "Giving $count x $item to $player..."
    execute_command "give $player $item $count" "$server"
}

# Main command handler
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        manage)
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                install)
                    install_mcrcon
                    ;;
                uninstall)
                    uninstall_mcrcon
                    ;;
                start)
                    start_health_service
                    # Wait flag support
                    if [[ "${1:-}" == "--wait" ]]; then
                        log_info "Waiting for service to be ready..."
                        sleep 2
                    fi
                    ;;
                stop)
                    stop_health_service
                    ;;
                restart)
                    stop_health_service
                    sleep 1
                    start_health_service
                    ;;
                *)
                    echo "Unknown manage subcommand: $subcommand" >&2
                    exit 1
                    ;;
            esac
            ;;
        content)
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                execute)
                    execute_command "$@"
                    ;;
                execute-all)
                    execute_all "$@"
                    ;;
                list)
                    list_servers
                    ;;
                add)
                    add_server "$@"
                    ;;
                remove)
                    remove_server "$@"
                    ;;
                discover)
                    discover_servers "$@"
                    ;;
                test)
                    test_connection "$@"
                    ;;
                *)
                    echo "Unknown content subcommand: $subcommand" >&2
                    exit 1
                    ;;
            esac
            ;;
        player)
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                list)
                    list_players "$@"
                    ;;
                info)
                    player_info "$@"
                    ;;
                teleport)
                    teleport_player "$@"
                    ;;
                kick)
                    kick_player "$@"
                    ;;
                ban)
                    ban_player "$@"
                    ;;
                give)
                    give_item "$@"
                    ;;
                *)
                    echo "Unknown player subcommand: $subcommand" >&2
                    exit 1
                    ;;
            esac
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs
            ;;
        credentials)
            show_credentials
            ;;
        *)
            echo "Unknown command: $command" >&2
            exit 1
            ;;
    esac
}

# Execute if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi