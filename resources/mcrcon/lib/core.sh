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
    
    # Install Python library if Python is available
    if command -v python3 &>/dev/null; then
        log_info "Installing Python library..."
        if python3 -m pip install -e "${RESOURCE_DIR}/python" &>/dev/null; then
            log_info "Python library installed successfully"
        else
            log_info "Python library installation skipped (pip not available)"
        fi
    fi
    
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

# World Operations
save_world() {
    local server="${1:-default}"
    
    echo "Saving world..."
    execute_command "save-all" "$server"
    if [[ $? -eq 0 ]]; then
        echo "World saved successfully"
        return 0
    else
        log_error "Failed to save world"
        return 1
    fi
}

backup_world() {
    local server="${1:-default}"
    local backup_dir="${MCRCON_DATA_DIR}/backups"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_name="world_backup_${timestamp}"
    
    echo "Creating world backup: $backup_name"
    
    # First, save the world
    if ! save_world "$server"; then
        log_error "Failed to save world before backup"
        return 1
    fi
    
    # Disable auto-save during backup
    execute_command "save-off" "$server"
    
    # Create backup directory if it doesn't exist
    mkdir -p "$backup_dir"
    
    # Get world data path (this would need server config)
    echo "Backup initiated. Note: Full file backup requires access to server files."
    echo "Backup metadata saved to: $backup_dir/${backup_name}.info"
    
    # Create backup info file
    cat > "$backup_dir/${backup_name}.info" << EOF
Backup Time: $(date)
Server: $server
Status: Metadata only (full backup requires server file access)
EOF
    
    # Re-enable auto-save
    execute_command "save-on" "$server"
    
    echo "Backup process completed"
    return 0
}

set_world_property() {
    local property="$1"
    local value="$2"
    local server="${3:-default}"
    
    if [[ -z "$property" ]] || [[ -z "$value" ]]; then
        log_error "Usage: set_world_property <property> <value> [server]"
        return 1
    fi
    
    echo "Setting world property: $property = $value"
    
    # Common world properties
    case "$property" in
        difficulty)
            execute_command "difficulty $value" "$server"
            ;;
        gamemode)
            execute_command "defaultgamemode $value" "$server"
            ;;
        pvp)
            execute_command "gamerule pvp $value" "$server"
            ;;
        spawn-protection)
            execute_command "gamerule spawnRadius $value" "$server"
            ;;
        max-players)
            execute_command "setmaxplayers $value" "$server"
            ;;
        weather)
            execute_command "weather $value" "$server"
            ;;
        time)
            execute_command "time set $value" "$server"
            ;;
        *)
            # Try as a gamerule
            execute_command "gamerule $property $value" "$server"
            ;;
    esac
}

get_world_info() {
    local server="${1:-default}"
    
    echo "World Information"
    echo "================="
    
    # Get various world information
    echo -e "\n[Difficulty]"
    execute_command "difficulty" "$server"
    
    echo -e "\n[Game Time]"
    execute_command "time query gametime" "$server"
    
    echo -e "\n[Day Time]"
    execute_command "time query daytime" "$server"
    
    echo -e "\n[Weather]"
    execute_command "weather query" "$server"
    
    echo -e "\n[Loaded Chunks]"
    execute_command "debug report" "$server" 2>/dev/null || echo "Debug info not available"
}

set_world_spawn() {
    local x="${1:-~}"
    local y="${2:-~}"
    local z="${3:-~}"
    local server="${4:-default}"
    
    echo "Setting world spawn point to ($x, $y, $z)..."
    execute_command "setworldspawn $x $y $z" "$server"
}

# Event Streaming Functions
start_event_stream() {
    local server="${1:-default}"
    local output_file="${2:-${MCRCON_DATA_DIR}/events.log}"
    local filter="${3:-all}"
    
    echo "Starting event stream for server: $server"
    echo "Output: $output_file"
    echo "Filter: $filter"
    
    # Create events directory
    mkdir -p "$(dirname "$output_file")"
    
    # Create streaming script
    cat > "${MCRCON_DATA_DIR}/stream_events.sh" << 'STREAM_EOF'
#!/usr/bin/env bash
set -euo pipefail

source "${RESOURCE_DIR}/config/defaults.sh"

server="$1"
output_file="$2"
filter="$3"

echo "[$(date)] Event stream started" >> "$output_file"

# Stream events in a loop
while true; do
    # Get latest log entries
    if timeout 5 "${MCRCON_BINARY}" -H "${MCRCON_HOST}" -P "${MCRCON_PORT}" -p "${MCRCON_PASSWORD}" "list" &>/dev/null; then
        # Parse server logs if accessible
        echo "[$(date)] Polling server events..." >> "$output_file"
        
        # Try to get recent chat messages
        # Note: This requires server log access or a mod for full functionality
        echo "[$(date)] Event streaming active (requires server log access for full events)" >> "$output_file"
    else
        echo "[$(date)] Connection lost, retrying..." >> "$output_file"
    fi
    
    sleep 5
done
STREAM_EOF
    
    chmod +x "${MCRCON_DATA_DIR}/stream_events.sh"
    
    # Start streaming in background
    nohup bash "${MCRCON_DATA_DIR}/stream_events.sh" "$server" "$output_file" "$filter" > /dev/null 2>&1 &
    local pid=$!
    echo $pid > "${MCRCON_DATA_DIR}/stream.pid"
    
    echo "Event stream started with PID: $pid"
    return 0
}

stop_event_stream() {
    if [[ -f "${MCRCON_DATA_DIR}/stream.pid" ]]; then
        local pid=$(cat "${MCRCON_DATA_DIR}/stream.pid")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            echo "Event stream stopped (PID: $pid)"
            rm -f "${MCRCON_DATA_DIR}/stream.pid"
        else
            echo "Event stream not running"
            rm -f "${MCRCON_DATA_DIR}/stream.pid"
        fi
    else
        echo "No event stream found"
    fi
}

stream_chat() {
    local server="${1:-default}"
    local duration="${2:-60}"
    
    echo "Streaming chat messages for ${duration} seconds..."
    echo "Note: Full chat streaming requires server log access or a supporting mod"
    
    local end_time=$(($(date +%s) + duration))
    
    while [[ $(date +%s) -lt $end_time ]]; do
        # Try to capture chat activity through player list changes
        local players=$(execute_command "list" "$server" 2>/dev/null | grep -oP '\d+ of \d+ players' || echo "")
        if [[ -n "$players" ]]; then
            echo "[$(date '+%H:%M:%S')] Online: $players"
        fi
        sleep 2
    done
    
    echo "Chat stream ended"
}

monitor_events() {
    local server="${1:-default}"
    local event_type="${2:-all}"
    
    echo "Monitoring events: $event_type"
    echo "Press Ctrl+C to stop"
    
    case "$event_type" in
        joins)
            echo "Monitoring player joins..."
            local prev_count=0
            while true; do
                local curr_count=$(execute_command "list" "$server" 2>/dev/null | grep -oP '\d+(?= of)' || echo "0")
                if [[ $curr_count -gt $prev_count ]]; then
                    echo "[$(date '+%H:%M:%S')] Player joined (total: $curr_count)"
                fi
                prev_count=$curr_count
                sleep 2
            done
            ;;
        leaves)
            echo "Monitoring player leaves..."
            local prev_count=0
            while true; do
                local curr_count=$(execute_command "list" "$server" 2>/dev/null | grep -oP '\d+(?= of)' || echo "0")
                if [[ $curr_count -lt $prev_count ]] && [[ $prev_count -ne 0 ]]; then
                    echo "[$(date '+%H:%M:%S')] Player left (total: $curr_count)"
                fi
                prev_count=$curr_count
                sleep 2
            done
            ;;
        deaths)
            echo "Monitoring deaths (requires server log access)..."
            echo "Note: Full death monitoring requires server log access"
            while true; do
                echo "[$(date '+%H:%M:%S')] Monitoring active..."
                sleep 10
            done
            ;;
        achievements)
            echo "Monitoring achievements (requires server log access)..."
            echo "Note: Full achievement monitoring requires server log access"
            while true; do
                echo "[$(date '+%H:%M:%S')] Monitoring active..."
                sleep 10
            done
            ;;
        all|*)
            echo "Monitoring all events..."
            local prev_count=0
            while true; do
                local curr_count=$(execute_command "list" "$server" 2>/dev/null | grep -oP '\d+(?= of)' || echo "0")
                local curr_time=$(date '+%H:%M:%S')
                
                if [[ $curr_count -gt $prev_count ]]; then
                    echo "[$curr_time] EVENT: Player joined (total: $curr_count)"
                elif [[ $curr_count -lt $prev_count ]] && [[ $prev_count -ne 0 ]]; then
                    echo "[$curr_time] EVENT: Player left (total: $curr_count)"
                fi
                
                # Periodic status
                if [[ $((SECONDS % 30)) -eq 0 ]]; then
                    echo "[$curr_time] STATUS: $curr_count players online"
                fi
                
                prev_count=$curr_count
                sleep 2
            done
            ;;
    esac
}

tail_events() {
    local lines="${1:-20}"
    local output_file="${MCRCON_DATA_DIR}/events.log"
    
    if [[ -f "$output_file" ]]; then
        echo "Last $lines events:"
        tail -n "$lines" "$output_file"
    else
        echo "No event log found. Start event streaming first."
        return 1
    fi
}

# Webhook Support Functions
configure_webhook() {
    local url="$1"
    local events="${2:-all}"
    local server="${3:-default}"
    
    if [[ -z "$url" ]]; then
        log_error "Webhook URL required"
        return 1
    fi
    
    # Validate URL format
    if ! echo "$url" | grep -qE '^https?://'; then
        log_error "Invalid webhook URL format (must start with http:// or https://)"
        return 1
    fi
    
    # Save webhook configuration
    local webhook_config="${MCRCON_DATA_DIR}/webhooks.json"
    
    # Load existing config or create new
    local config="{}"
    if [[ -f "$webhook_config" ]]; then
        config=$(cat "$webhook_config")
    fi
    
    # Add/update webhook
    config=$(echo "$config" | jq --arg url "$url" --arg events "$events" --arg server "$server" \
        '.webhooks += [{"url": $url, "events": $events, "server": $server, "enabled": true}]')
    
    echo "$config" > "$webhook_config"
    echo "Webhook configured: $url"
    echo "Events: $events"
    echo "Server: $server"
    
    # Restart webhook service if running
    if pgrep -f "webhook_service.py.*mcrcon" > /dev/null 2>&1; then
        stop_webhook_service
        start_webhook_service
    fi
    
    return 0
}

list_webhooks() {
    local webhook_config="${MCRCON_DATA_DIR}/webhooks.json"
    
    if [[ ! -f "$webhook_config" ]]; then
        echo "No webhooks configured"
        return 0
    fi
    
    echo "Configured Webhooks:"
    echo "===================="
    jq -r '.webhooks[] | "URL: \(.url)\nEvents: \(.events)\nServer: \(.server)\nEnabled: \(.enabled)\n"' "$webhook_config"
}

remove_webhook() {
    local url="$1"
    local webhook_config="${MCRCON_DATA_DIR}/webhooks.json"
    
    if [[ -z "$url" ]]; then
        log_error "Webhook URL required"
        return 1
    fi
    
    if [[ ! -f "$webhook_config" ]]; then
        echo "No webhooks configured"
        return 1
    fi
    
    # Remove webhook from config
    local config=$(cat "$webhook_config")
    config=$(echo "$config" | jq --arg url "$url" '.webhooks |= map(select(.url != $url))')
    
    echo "$config" > "$webhook_config"
    echo "Webhook removed: $url"
    
    # Restart webhook service if running
    if pgrep -f "webhook_service.py.*mcrcon" > /dev/null 2>&1; then
        stop_webhook_service
        start_webhook_service
    fi
    
    return 0
}

start_webhook_service() {
    if pgrep -f "webhook_service.py.*mcrcon" > /dev/null 2>&1; then
        log_info "Webhook service already running"
        return 0
    fi
    
    # Create webhook service script
    cat > "${MCRCON_DATA_DIR}/webhook_service.py" << 'WEBHOOK_EOF'
#!/usr/bin/env python3
import json
import os
import time
import urllib.request
import urllib.parse
import subprocess
from pathlib import Path

WEBHOOK_CONFIG = Path(os.environ.get('MCRCON_DATA_DIR', Path.home() / '.mcrcon')) / 'webhooks.json'
MCRCON_BINARY = Path(os.environ.get('MCRCON_DATA_DIR', Path.home() / '.mcrcon')) / 'bin' / 'mcrcon'

def load_webhooks():
    if not WEBHOOK_CONFIG.exists():
        return []
    
    with open(WEBHOOK_CONFIG, 'r') as f:
        config = json.load(f)
        return config.get('webhooks', [])

def send_webhook(url, event_data):
    try:
        data = json.dumps(event_data).encode('utf-8')
        req = urllib.request.Request(url, data=data, headers={'Content-Type': 'application/json'})
        with urllib.request.urlopen(req, timeout=10) as response:
            return response.status == 200
    except Exception as e:
        print(f"Webhook error: {e}")
        return False

def monitor_events():
    webhooks = load_webhooks()
    prev_player_count = 0
    
    while True:
        try:
            # Get current player count (simplified monitoring)
            # In production, this would monitor actual server logs
            
            for webhook in webhooks:
                if not webhook.get('enabled', True):
                    continue
                
                # Example event (in production, would be actual events)
                event = {
                    'type': 'heartbeat',
                    'server': webhook['server'],
                    'timestamp': time.time(),
                    'message': 'Webhook service active'
                }
                
                # Only send events that match webhook filter
                if webhook['events'] == 'all' or event['type'] in webhook['events']:
                    send_webhook(webhook['url'], event)
            
        except Exception as e:
            print(f"Monitor error: {e}")
        
        # Reload config periodically
        if int(time.time()) % 60 == 0:
            webhooks = load_webhooks()
        
        time.sleep(10)

if __name__ == "__main__":
    print("Webhook service started")
    monitor_events()
WEBHOOK_EOF
    
    chmod +x "${MCRCON_DATA_DIR}/webhook_service.py"
    
    log_info "Starting webhook service..."
    nohup python3 "${MCRCON_DATA_DIR}/webhook_service.py" > "${MCRCON_DATA_DIR}/logs/webhook.log" 2>&1 &
    
    echo "Webhook service started"
    return 0
}

stop_webhook_service() {
    local pid=$(pgrep -f "webhook_service.py.*mcrcon" 2>/dev/null || true)
    if [[ -n "$pid" ]]; then
        kill "$pid" 2>/dev/null || true
        log_info "Webhook service stopped"
    fi
}

test_webhook() {
    local url="$1"
    
    if [[ -z "$url" ]]; then
        log_error "Webhook URL required"
        return 1
    fi
    
    echo "Testing webhook: $url"
    
    # Send test event
    local test_event='{
        "type": "test",
        "message": "Test webhook from mcrcon",
        "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
    }'
    
    if curl -X POST "$url" \
        -H "Content-Type: application/json" \
        -d "$test_event" \
        --connect-timeout 5 \
        -w "\nHTTP Status: %{http_code}\n" \
        2>/dev/null; then
        echo "Webhook test successful"
        return 0
    else
        echo "Webhook test failed"
        return 1
    fi
}

# Mod Integration Functions
list_mods() {
    local server="${1:-default}"
    
    echo "Checking for installed mods..."
    
    # Try common mod list commands
    local result=$(execute_command "mods" "$server" 2>/dev/null || true)
    if [[ -n "$result" ]]; then
        echo "$result"
        return 0
    fi
    
    # Try Forge command
    result=$(execute_command "forge mods" "$server" 2>/dev/null || true)
    if [[ -n "$result" ]]; then
        echo "Forge Mods:"
        echo "$result"
        return 0
    fi
    
    # Try Fabric command
    result=$(execute_command "fabric:mods" "$server" 2>/dev/null || true)
    if [[ -n "$result" ]]; then
        echo "Fabric Mods:"
        echo "$result"
        return 0
    fi
    
    # Try Bukkit/Spigot/Paper plugins
    result=$(execute_command "plugins" "$server" 2>/dev/null || true)
    if [[ -n "$result" ]]; then
        echo "Plugins:"
        echo "$result"
        return 0
    fi
    
    echo "No mod list command available (server may be vanilla or mod commands not exposed)"
    echo "Note: Full mod support requires compatible server mods/plugins"
    return 1
}

execute_mod_command() {
    local mod="$1"
    local command="$2"
    local server="${3:-default}"
    
    if [[ -z "$mod" ]] || [[ -z "$command" ]]; then
        log_error "Usage: execute_mod_command <mod> <command> [server]"
        return 1
    fi
    
    echo "Executing mod command: $mod:$command"
    
    # Try various mod command formats
    local formats=(
        "${mod}:${command}"
        "${mod} ${command}"
        "/${mod}:${command}"
        "/${mod} ${command}"
    )
    
    for format in "${formats[@]}"; do
        local result=$(execute_command "$format" "$server" 2>/dev/null || true)
        if [[ -n "$result" ]] && [[ "$result" != *"Unknown command"* ]]; then
            echo "$result"
            return 0
        fi
    done
    
    log_error "Failed to execute mod command (mod may not be installed or command not available)"
    return 1
}

register_mod_commands() {
    local config_file="${1:-${MCRCON_DATA_DIR}/mod_commands.json}"
    local server="${2:-default}"
    
    echo "Registering custom mod commands..."
    
    # Create or load mod commands configuration
    if [[ ! -f "$config_file" ]]; then
        cat > "$config_file" << 'MOD_CONFIG_EOF'
{
    "mod_commands": [
        {
            "mod": "worldedit",
            "commands": [
                {"name": "set", "usage": "//set <block>", "description": "Set selection to block"},
                {"name": "copy", "usage": "//copy", "description": "Copy selection"},
                {"name": "paste", "usage": "//paste", "description": "Paste clipboard"}
            ]
        },
        {
            "mod": "essentials",
            "commands": [
                {"name": "home", "usage": "/home [name]", "description": "Teleport to home"},
                {"name": "warp", "usage": "/warp <name>", "description": "Teleport to warp"},
                {"name": "kit", "usage": "/kit <name>", "description": "Get a kit"}
            ]
        },
        {
            "mod": "permissions",
            "commands": [
                {"name": "group", "usage": "/perm group <action>", "description": "Manage groups"},
                {"name": "user", "usage": "/perm user <action>", "description": "Manage users"}
            ]
        }
    ]
}
MOD_CONFIG_EOF
        echo "Created default mod commands configuration"
    fi
    
    echo "Mod commands registered at: $config_file"
    echo "Edit this file to add custom mod commands for your server"
    return 0
}

show_mod_commands() {
    local mod="${1:-all}"
    local config_file="${MCRCON_DATA_DIR}/mod_commands.json"
    
    if [[ ! -f "$config_file" ]]; then
        echo "No mod commands registered. Run 'register-commands' first."
        return 1
    fi
    
    if [[ "$mod" == "all" ]]; then
        echo "Registered Mod Commands:"
        echo "========================"
        jq -r '.mod_commands[] | "\n[\(.mod)]\n" + (.commands[] | "  \(.name): \(.usage)\n    \(.description)")' "$config_file"
    else
        echo "Commands for mod: $mod"
        echo "===================="
        jq -r --arg mod "$mod" '.mod_commands[] | select(.mod == $mod) | .commands[] | "\(.name): \(.usage)\n  \(.description)"' "$config_file"
    fi
}

test_mod_support() {
    local server="${1:-default}"
    
    echo "Testing mod support on server..."
    echo "================================"
    
    # Test for common mod loaders
    echo -n "Forge: "
    if execute_command "forge" "$server" &>/dev/null; then
        echo "✓ Detected"
    else
        echo "✗ Not found"
    fi
    
    echo -n "Fabric: "
    if execute_command "fabric" "$server" &>/dev/null; then
        echo "✓ Detected"
    else
        echo "✗ Not found"
    fi
    
    echo -n "Bukkit/Spigot/Paper: "
    if execute_command "version" "$server" 2>/dev/null | grep -qE "(Bukkit|Spigot|Paper)"; then
        echo "✓ Detected"
    else
        echo "✗ Not found"
    fi
    
    echo -n "Plugins: "
    if execute_command "plugins" "$server" &>/dev/null; then
        echo "✓ Available"
    else
        echo "✗ Not available"
    fi
    
    echo ""
    echo "Attempting to list installed mods/plugins..."
    list_mods "$server"
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
        world)
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                save)
                    save_world "$@"
                    ;;
                backup)
                    backup_world "$@"
                    ;;
                info)
                    get_world_info "$@"
                    ;;
                set-property)
                    set_world_property "$@"
                    ;;
                set-spawn)
                    set_world_spawn "$@"
                    ;;
                *)
                    echo "Unknown world subcommand: $subcommand" >&2
                    echo "Available: save, backup, info, set-property, set-spawn"
                    exit 1
                    ;;
            esac
            ;;
        event)
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                start-stream)
                    start_event_stream "$@"
                    ;;
                stop-stream)
                    stop_event_stream
                    ;;
                stream-chat)
                    stream_chat "$@"
                    ;;
                monitor)
                    monitor_events "$@"
                    ;;
                tail)
                    tail_events "$@"
                    ;;
                *)
                    echo "Unknown event subcommand: $subcommand" >&2
                    echo "Available: start-stream, stop-stream, stream-chat, monitor, tail"
                    exit 1
                    ;;
            esac
            ;;
        webhook)
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                configure)
                    configure_webhook "$@"
                    ;;
                list)
                    list_webhooks
                    ;;
                remove)
                    remove_webhook "$@"
                    ;;
                start)
                    start_webhook_service
                    ;;
                stop)
                    stop_webhook_service
                    ;;
                test)
                    test_webhook "$@"
                    ;;
                *)
                    echo "Unknown webhook subcommand: $subcommand" >&2
                    echo "Available: configure, list, remove, start, stop, test"
                    exit 1
                    ;;
            esac
            ;;
        mod)
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                list)
                    list_mods "$@"
                    ;;
                execute)
                    execute_mod_command "$@"
                    ;;
                register-commands)
                    register_mod_commands "$@"
                    ;;
                show-commands)
                    show_mod_commands "$@"
                    ;;
                test-support)
                    test_mod_support "$@"
                    ;;
                *)
                    echo "Unknown mod subcommand: $subcommand" >&2
                    echo "Available: list, execute, register-commands, show-commands, test-support"
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