#!/usr/bin/env bash
# PaperMC core functionality

set -euo pipefail

# Health service PID file
readonly HEALTH_PID_FILE="${PAPERMC_DATA_DIR}/.health_service.pid"

# Install PaperMC server
install_papermc() {
    echo "Installing PaperMC server..."
    
    # Create necessary directories
    mkdir -p "${PAPERMC_DATA_DIR}"/{world,plugins,config,backups}
    
    if [[ "${PAPERMC_SERVER_TYPE}" == "docker" ]]; then
        echo "Checking Docker installation..."
        if ! command -v docker &> /dev/null; then
            echo "Error: Docker is not installed" >&2
            echo "Please install Docker first: https://docs.docker.com/engine/install/" >&2
            return 1
        fi
        
        echo "Pulling Docker image: ${PAPERMC_DOCKER_IMAGE}:${PAPERMC_DOCKER_TAG}"
        docker pull "${PAPERMC_DOCKER_IMAGE}:${PAPERMC_DOCKER_TAG}"
        
        # Create docker-compose.yml
        create_docker_compose
        
    else
        echo "Installing native PaperMC server..."
        
        # Check Java installation
        if ! command -v java &> /dev/null; then
            echo "Error: Java is not installed" >&2
            echo "Please install Java 17 or later" >&2
            return 1
        fi
        
        # Download PaperMC JAR
        local paper_jar="${PAPERMC_DATA_DIR}/paper.jar"
        if [[ ! -f "${paper_jar}" ]]; then
            echo "Downloading PaperMC JAR..."
            local version="${PAPERMC_VERSION}"
            if [[ "${version}" == "latest" ]]; then
                version="1.20.4"  # Default to stable version
            fi
            
            # Download from PaperMC API
            local download_url="https://api.papermc.io/v2/projects/paper/versions/${version}/builds/latest/downloads/paper-${version}-latest.jar"
            curl -L -o "${paper_jar}" "${download_url}" || {
                echo "Error: Failed to download PaperMC" >&2
                return 1
            }
        fi
    fi
    
    # Create initial server.properties
    create_server_properties
    
    # Accept EULA if configured
    if [[ "${PAPERMC_EULA}" == "true" ]]; then
        echo "Accepting Minecraft EULA..."
        echo "eula=true" > "${PAPERMC_DATA_DIR}/eula.txt"
    fi
    
    echo "PaperMC installation complete!"
    echo "Data directory: ${PAPERMC_DATA_DIR}"
    return 0
}

# Uninstall PaperMC server
uninstall_papermc() {
    echo "Uninstalling PaperMC server..."
    
    # Stop server if running
    stop_papermc 2>/dev/null || true
    
    if [[ "${PAPERMC_SERVER_TYPE}" == "docker" ]]; then
        # Remove container
        docker rm -f "${PAPERMC_CONTAINER_NAME}" 2>/dev/null || true
    fi
    
    # Optionally remove data
    local keep_data="${1:-}"
    if [[ "${keep_data}" != "--keep-data" ]]; then
        echo "Removing data directory: ${PAPERMC_DATA_DIR}"
        rm -rf "${PAPERMC_DATA_DIR}"
    else
        echo "Keeping data directory: ${PAPERMC_DATA_DIR}"
    fi
    
    echo "PaperMC uninstallation complete!"
    return 0
}

# Start PaperMC server
start_papermc() {
    echo "Starting PaperMC server..."
    
    # Check if already running
    if is_server_running; then
        echo "Server is already running"
        return 0
    fi
    
    # Start health service first
    start_health_service
    
    if [[ "${PAPERMC_SERVER_TYPE}" == "docker" ]]; then
        echo "Starting Docker container..."
        
        # Create docker-compose.yml if it doesn't exist
        if [[ ! -f "${PAPERMC_DATA_DIR}/docker-compose.yml" ]]; then
            create_docker_compose
        fi
        
        cd "${PAPERMC_DATA_DIR}"
        docker-compose up -d
        
    else
        echo "Starting native server..."
        
        cd "${PAPERMC_DATA_DIR}"
        
        # Start server with JVM options
        nohup java \
            -Xms${PAPERMC_MEMORY} \
            -Xmx${PAPERMC_MAX_MEMORY} \
            ${PAPERMC_JVM_OPTS} \
            -jar paper.jar \
            --nogui \
            > "${PAPERMC_LOG_FILE}" 2>&1 &
        
        local server_pid=$!
        echo "${server_pid}" > "${PAPERMC_DATA_DIR}/server.pid"
    fi
    
    # Wait for server to be ready if requested
    local wait_flag="${1:-}"
    if [[ "${wait_flag}" == "--wait" ]]; then
        echo "Waiting for server to be ready..."
        local max_wait="${PAPERMC_STARTUP_TIMEOUT}"
        local waited=0
        
        while [[ ${waited} -lt ${max_wait} ]]; do
            if is_server_ready; then
                echo "Server is ready!"
                return 0
            fi
            sleep 2
            ((waited+=2))
            echo -n "."
        done
        
        echo ""
        echo "Warning: Server startup timeout reached" >&2
    fi
    
    echo "PaperMC server started!"
    return 0
}

# Stop PaperMC server
stop_papermc() {
    echo "Stopping PaperMC server..."
    
    # Try graceful shutdown via RCON if mcrcon is available
    if command -v mcrcon &> /dev/null && is_server_ready; then
        echo "Attempting graceful shutdown via RCON..."
        mcrcon -H localhost -P "${PAPERMC_RCON_PORT}" -p "${PAPERMC_RCON_PASSWORD}" "save-all" 2>/dev/null || true
        sleep 2
        mcrcon -H localhost -P "${PAPERMC_RCON_PORT}" -p "${PAPERMC_RCON_PASSWORD}" "stop" 2>/dev/null || true
        sleep 5
    fi
    
    if [[ "${PAPERMC_SERVER_TYPE}" == "docker" ]]; then
        echo "Stopping Docker container..."
        docker stop "${PAPERMC_CONTAINER_NAME}" 2>/dev/null || true
    else
        # Stop native server
        if [[ -f "${PAPERMC_DATA_DIR}/server.pid" ]]; then
            local pid
            pid=$(cat "${PAPERMC_DATA_DIR}/server.pid")
            if kill -0 "${pid}" 2>/dev/null; then
                echo "Stopping server process ${pid}..."
                kill "${pid}" 2>/dev/null || true
                
                # Wait for process to stop
                local waited=0
                while kill -0 "${pid}" 2>/dev/null && [[ ${waited} -lt ${PAPERMC_STOP_TIMEOUT} ]]; do
                    sleep 1
                    ((waited++))
                done
                
                # Force kill if still running
                if kill -0 "${pid}" 2>/dev/null; then
                    echo "Force stopping server..."
                    kill -9 "${pid}" 2>/dev/null || true
                fi
            fi
            rm -f "${PAPERMC_DATA_DIR}/server.pid"
        fi
    fi
    
    # Stop health service
    stop_health_service
    
    echo "PaperMC server stopped!"
    return 0
}

# Restart PaperMC server
restart_papermc() {
    echo "Restarting PaperMC server..."
    stop_papermc
    sleep 2
    start_papermc "$@"
    return 0
}

# Check if server is running
is_server_running() {
    if [[ "${PAPERMC_SERVER_TYPE}" == "docker" ]]; then
        docker ps --format "{{.Names}}" | grep -q "^${PAPERMC_CONTAINER_NAME}$"
    else
        if [[ -f "${PAPERMC_DATA_DIR}/server.pid" ]]; then
            local pid
            pid=$(cat "${PAPERMC_DATA_DIR}/server.pid")
            kill -0 "${pid}" 2>/dev/null
        else
            return 1
        fi
    fi
}

# Check if server is ready (RCON responsive)
is_server_ready() {
    # Simple check: try to connect to RCON port
    nc -z -w2 localhost "${PAPERMC_RCON_PORT}" 2>/dev/null
}

# Execute RCON command
execute_rcon_command() {
    local command="$*"
    
    if [[ -z "${command}" ]]; then
        echo "Error: No command specified" >&2
        return 1
    fi
    
    # Check if server is running
    if ! is_server_running; then
        echo "Error: Server is not running" >&2
        return 1
    fi
    
    # Check if mcrcon is available
    if ! command -v mcrcon &> /dev/null; then
        # Try using the mcrcon resource if available
        if command -v vrooli &> /dev/null; then
            vrooli resource mcrcon content execute "${command}"
        else
            echo "Error: mcrcon is not available" >&2
            echo "Please install the mcrcon resource: vrooli resource mcrcon manage install" >&2
            return 1
        fi
    else
        mcrcon -H localhost -P "${PAPERMC_RCON_PORT}" -p "${PAPERMC_RCON_PASSWORD}" "${command}"
    fi
}

# Backup world and configuration
backup_world() {
    echo "Creating backup..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${PAPERMC_BACKUP_DIR}/backup_${timestamp}.tar.gz"
    
    mkdir -p "${PAPERMC_BACKUP_DIR}"
    
    # Save world if server is running
    if is_server_running && is_server_ready; then
        execute_rcon_command "save-all" || true
        sleep 2
    fi
    
    # Create backup
    tar -czf "${backup_file}" \
        -C "${PAPERMC_DATA_DIR}" \
        world \
        server.properties \
        plugins \
        2>/dev/null || true
    
    echo "Backup created: ${backup_file}"
    
    # Clean old backups (keep last 10)
    ls -t "${PAPERMC_BACKUP_DIR}"/backup_*.tar.gz 2>/dev/null | tail -n +11 | xargs rm -f 2>/dev/null || true
    
    return 0
}

# Configure server
configure_server() {
    echo "Server configuration update not yet implemented"
    return 0
}

# List plugins
list_plugins() {
    echo "Installed plugins:"
    ls -la "${PAPERMC_PLUGINS_DIR}" 2>/dev/null || echo "No plugins installed"
    return 0
}

# Show server status
show_status() {
    echo "PaperMC Server Status"
    echo "====================="
    
    if is_server_running; then
        echo "Status: Running"
        
        if is_server_ready; then
            echo "RCON: Ready"
            
            # Try to get player count if possible
            if command -v mcrcon &> /dev/null; then
                echo ""
                echo "Players online:"
                execute_rcon_command "list" 2>/dev/null || echo "Unable to query players"
            fi
        else
            echo "RCON: Not ready"
        fi
    else
        echo "Status: Stopped"
    fi
    
    echo ""
    echo "Configuration:"
    echo "  Type: ${PAPERMC_SERVER_TYPE}"
    echo "  Memory: ${PAPERMC_MEMORY} - ${PAPERMC_MAX_MEMORY}"
    echo "  Game Port: ${PAPERMC_SERVER_PORT}"
    echo "  RCON Port: ${PAPERMC_RCON_PORT}"
    echo "  Data Directory: ${PAPERMC_DATA_DIR}"
    
    return 0
}

# Show server logs
show_logs() {
    local lines="${1:-50}"
    
    if [[ "${PAPERMC_SERVER_TYPE}" == "docker" ]]; then
        docker logs --tail "${lines}" "${PAPERMC_CONTAINER_NAME}" 2>/dev/null || echo "No logs available"
    else
        if [[ -f "${PAPERMC_LOG_FILE}" ]]; then
            tail -n "${lines}" "${PAPERMC_LOG_FILE}"
        else
            echo "No logs available"
        fi
    fi
    
    return 0
}

# Create docker-compose.yml
create_docker_compose() {
    cat > "${PAPERMC_DATA_DIR}/docker-compose.yml" << EOF
version: '3.8'

services:
  minecraft:
    image: ${PAPERMC_DOCKER_IMAGE}:${PAPERMC_DOCKER_TAG}
    container_name: ${PAPERMC_CONTAINER_NAME}
    restart: unless-stopped
    environment:
      - TYPE=PAPER
      - VERSION=${PAPERMC_VERSION}
      - EULA=${PAPERMC_EULA}
      - MEMORY=${PAPERMC_MEMORY}
      - MAX_MEMORY=${PAPERMC_MAX_MEMORY}
      - ENABLE_RCON=true
      - RCON_PORT=${PAPERMC_RCON_PORT}
      - RCON_PASSWORD=${PAPERMC_RCON_PASSWORD}
      - MODE=${PAPERMC_GAMEMODE}
      - DIFFICULTY=${PAPERMC_DIFFICULTY}
      - MAX_PLAYERS=${PAPERMC_MAX_PLAYERS}
      - LEVEL=${PAPERMC_WORLD_NAME}
      - SEED=${PAPERMC_WORLD_SEED}
      - JVM_OPTS=${PAPERMC_JVM_OPTS}
    ports:
      - "127.0.0.1:${PAPERMC_SERVER_PORT}:25565"
      - "127.0.0.1:${PAPERMC_RCON_PORT}:25575"
    volumes:
      - ${PAPERMC_DATA_DIR}:/data
    stdin_open: true
    tty: true
EOF
}

# Create server.properties
create_server_properties() {
    cat > "${PAPERMC_DATA_DIR}/server.properties" << EOF
# Minecraft server properties
# Generated by PaperMC resource

# Network
server-port=${PAPERMC_SERVER_PORT}
enable-rcon=true
rcon.port=${PAPERMC_RCON_PORT}
rcon.password=${PAPERMC_RCON_PASSWORD}

# World
level-name=${PAPERMC_WORLD_NAME}
level-seed=${PAPERMC_WORLD_SEED}
gamemode=${PAPERMC_GAMEMODE}
difficulty=${PAPERMC_DIFFICULTY}
max-players=${PAPERMC_MAX_PLAYERS}

# Performance
view-distance=10
simulation-distance=10

# Security
online-mode=false
white-list=false

# Other
motd=PaperMC Server - Managed by Vrooli
EOF
}

# Start health service
start_health_service() {
    # Stop any existing health service
    stop_health_service 2>/dev/null || true
    
    echo "Starting health service on port ${PAPERMC_HEALTH_PORT}..."
    
    # Create directory if it doesn't exist
    mkdir -p "${PAPERMC_DATA_DIR}"
    
    # Create a simple Python health server that responds with JSON
    cat > "${PAPERMC_DATA_DIR}/health_server.py" << 'EOF'
#!/usr/bin/env python3
import http.server
import json
import os
import socketserver
import sys

PORT = int(os.environ.get('PAPERMC_HEALTH_PORT', '11459'))

class HealthHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            # Check if server is running by testing RCON port
            import socket
            rcon_port = int(os.environ.get('PAPERMC_RCON_PORT', '25575'))
            server_running = False
            try:
                s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                s.settimeout(1)
                result = s.connect_ex(('localhost', rcon_port))
                s.close()
                server_running = (result == 0)
            except:
                pass
            
            status = "healthy" if server_running else "starting"
            response = {
                "status": status,
                "service": "papermc",
                "server_running": server_running,
                "rcon_port": rcon_port
            }
            
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        # Suppress request logging
        pass

with socketserver.TCPServer(("127.0.0.1", PORT), HealthHandler) as httpd:
    httpd.serve_forever()
EOF
    
    # Start the health server
    nohup python3 "${PAPERMC_DATA_DIR}/health_server.py" > "${PAPERMC_DATA_DIR}/health.log" 2>&1 &
    
    local health_pid=$!
    echo "${health_pid}" > "${HEALTH_PID_FILE}"
    
    # Give it a moment to start
    sleep 2
    
    # Verify it's running
    if ! kill -0 "${health_pid}" 2>/dev/null; then
        echo "Warning: Health service failed to start" >&2
        rm -f "${HEALTH_PID_FILE}"
        return 1
    fi
    
    # Verify health endpoint responds
    if ! timeout 5 curl -sf "http://localhost:${PAPERMC_HEALTH_PORT}/health" > /dev/null; then
        echo "Warning: Health endpoint not responding" >&2
        return 1
    fi
    
    return 0
}

# Stop health service
stop_health_service() {
    if [[ -f "${HEALTH_PID_FILE}" ]]; then
        local pid
        pid=$(cat "${HEALTH_PID_FILE}")
        if kill -0 "${pid}" 2>/dev/null; then
            echo "Stopping health service..."
            kill "${pid}" 2>/dev/null || true
            sleep 1
            kill -9 "${pid}" 2>/dev/null || true
        fi
        rm -f "${HEALTH_PID_FILE}"
    fi
    
    # Clean up any processes using the health port
    local pids
    pids=$(lsof -t -i:${PAPERMC_HEALTH_PORT} 2>/dev/null || true)
    if [[ -n "${pids}" ]]; then
        echo "Cleaning up processes on port ${PAPERMC_HEALTH_PORT}..."
        echo "${pids}" | xargs -r kill -9 2>/dev/null || true
    fi
    
    # Clean up any orphaned nc processes
    pkill -f "nc.*${PAPERMC_HEALTH_PORT}" 2>/dev/null || true
    
    # Clean up any orphaned Python health servers
    pkill -f "health_server.py" 2>/dev/null || true
    
    return 0
}