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
    
    # Get server configuration
    local host="${MCRCON_HOST}"
    local port="${MCRCON_PORT}"
    local password="${MCRCON_PASSWORD}"
    
    if [[ -z "$password" ]]; then
        log_error "RCON password not configured"
        echo "Set MCRCON_PASSWORD environment variable or configure server"
        return 1
    fi
    
    # Execute command
    log_debug "Executing command: $command on $host:$port"
    
    if timeout "${MCRCON_TIMEOUT}" "${MCRCON_BINARY}" -H "$host" -P "$port" -p "$password" "$command"; then
        return 0
    else
        log_error "Failed to execute command: $command"
        return 1
    fi
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
                list)
                    list_servers
                    ;;
                add)
                    add_server "$@"
                    ;;
                remove)
                    remove_server "$@"
                    ;;
                *)
                    echo "Unknown content subcommand: $subcommand" >&2
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