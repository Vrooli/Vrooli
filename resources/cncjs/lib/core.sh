#!/usr/bin/env bash
################################################################################
# CNCjs Core Functions Library
# Implements lifecycle management and core functionality
################################################################################

set -euo pipefail

# Source guard
[[ -n "${_CNCJS_CORE_SOURCED:-}" ]] && return 0
export _CNCJS_CORE_SOURCED=1

# Source shared libraries
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || {
    # Fallback logging functions if library not available
    log::info() { echo "[INFO] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
}
source "${APP_ROOT}/scripts/lib/utils/validation.sh" 2>/dev/null || true

#######################################
# Install CNCjs and dependencies
#######################################
cncjs::install() {
    local force="${1:-false}"
    
    log::info "Installing CNCjs resource..."
    
    # Check if already installed
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${CNCJS_IMAGE}$"; then
        log::warning "CNCjs image already installed"
        [[ "$force" != "--force" ]] && return 2
    fi
    
    # Create data directories
    mkdir -p "${CNCJS_DATA_DIR}"
    mkdir -p "${CNCJS_WATCH_DIR}"
    
    # Create default configuration if not exists
    if [[ ! -f "${CNCJS_CONFIG_FILE}" ]]; then
        cat > "${CNCJS_CONFIG_FILE}" << EOF
{
  "state": {
    "checkForUpdates": false
  },
  "watchDirectory": "${CNCJS_WATCH_DIR}",
  "controller": "${CNCJS_CONTROLLER}",
  "ports": [
    {
      "comName": "${CNCJS_SERIAL_PORT}",
      "manufacturer": ""
    }
  ],
  "baudRate": ${CNCJS_BAUD_RATE},
  "accessTokenLifetime": "${CNCJS_ACCESS_TOKEN_LIFETIME}",
  "allowRemoteAccess": ${CNCJS_ALLOW_REMOTE}
}
EOF
        log::info "Created default configuration at ${CNCJS_CONFIG_FILE}"
    fi
    
    # Pull Docker image
    log::info "Pulling CNCjs Docker image..."
    if ! docker pull "${CNCJS_IMAGE}"; then
        log::error "Failed to pull CNCjs image"
        return 1
    fi
    
    log::success "CNCjs installed successfully"
    return 0
}

#######################################
# Uninstall CNCjs
#######################################
cncjs::uninstall() {
    local keep_data="${1:-false}"
    
    log::info "Uninstalling CNCjs resource..."
    
    # Stop container if running
    cncjs::stop || true
    
    # Remove container
    if docker ps -a --format "{{.Names}}" | grep -q "^${CNCJS_CONTAINER_NAME}$"; then
        docker rm -f "${CNCJS_CONTAINER_NAME}" &>/dev/null || true
    fi
    
    # Remove image
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${CNCJS_IMAGE}$"; then
        docker rmi "${CNCJS_IMAGE}" || true
    fi
    
    # Remove data if requested
    if [[ "$keep_data" != "--keep-data" ]]; then
        log::info "Removing CNCjs data directory..."
        rm -rf "${CNCJS_DATA_DIR}"
    fi
    
    log::success "CNCjs uninstalled successfully"
    return 0
}

#######################################
# Start CNCjs service
#######################################
cncjs::start() {
    local wait_flag="${1:-}"
    
    log::info "Starting CNCjs service..."
    
    # Check if already running
    if docker ps --format "{{.Names}}" | grep -q "^${CNCJS_CONTAINER_NAME}$"; then
        log::warning "CNCjs is already running"
        return 0
    fi
    
    # Ensure image exists
    if ! docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${CNCJS_IMAGE}$"; then
        log::error "CNCjs image not found. Run 'manage install' first"
        return 1
    fi
    
    # Start container with serial port access
    log::info "Starting CNCjs container on port ${CNCJS_PORT}..."
    if ! docker run -d \
        --name "${CNCJS_CONTAINER_NAME}" \
        --restart unless-stopped \
        --privileged \
        -p "${CNCJS_PORT}:8000" \
        -v /dev:/dev \
        -v "${CNCJS_CONFIG_FILE}:/root/.cncrc" \
        -v "${CNCJS_WATCH_DIR}:/root/watch" \
        -v "${CNCJS_DATA_DIR}:/root/.cncjs" \
        "${CNCJS_IMAGE}" \
        --port 8000 \
        --host 0.0.0.0 \
        --allow-remote-access; then
        log::error "Failed to start CNCjs container"
        return 1
    fi
    
    # Wait for service to be ready if requested
    if [[ "$wait_flag" == "--wait" ]]; then
        log::info "Waiting for CNCjs to be ready..."
        local attempts=0
        local max_attempts=$((CNCJS_STARTUP_TIMEOUT / 5))
        
        while [[ $attempts -lt $max_attempts ]]; do
            if cncjs::health_check; then
                log::success "CNCjs is ready"
                return 0
            fi
            sleep 5
            ((attempts++))
        done
        
        log::error "CNCjs failed to start within ${CNCJS_STARTUP_TIMEOUT} seconds"
        return 1
    fi
    
    log::success "CNCjs started successfully"
    return 0
}

#######################################
# Stop CNCjs service
#######################################
cncjs::stop() {
    log::info "Stopping CNCjs service..."
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${CNCJS_CONTAINER_NAME}$"; then
        log::warning "CNCjs is not running"
        return 0
    fi
    
    # Graceful shutdown
    if ! timeout "${CNCJS_SHUTDOWN_TIMEOUT}" docker stop "${CNCJS_CONTAINER_NAME}" &>/dev/null; then
        log::warning "Graceful shutdown timed out, forcing stop..."
        docker kill "${CNCJS_CONTAINER_NAME}" &>/dev/null || true
    fi
    
    # Remove container
    docker rm -f "${CNCJS_CONTAINER_NAME}" &>/dev/null || true
    
    log::success "CNCjs stopped successfully"
    return 0
}

#######################################
# Restart CNCjs service
#######################################
cncjs::restart() {
    log::info "Restarting CNCjs service..."
    cncjs::stop
    sleep 2
    cncjs::start "$@"
}

#######################################
# Check CNCjs health
#######################################
cncjs::health_check() {
    # Check container is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${CNCJS_CONTAINER_NAME}$"; then
        return 1
    fi
    
    # Check HTTP endpoint
    if ! timeout 5 curl -sf "http://localhost:${CNCJS_PORT}/" &>/dev/null; then
        return 1
    fi
    
    return 0
}

#######################################
# Show CNCjs status
#######################################
cncjs::status() {
    local json_output="${1:-false}"
    
    local status="stopped"
    local health="unknown"
    local uptime="N/A"
    local controller_state="disconnected"
    
    # Check if running
    if docker ps --format "{{.Names}}" | grep -q "^${CNCJS_CONTAINER_NAME}$"; then
        status="running"
        
        # Get uptime
        uptime=$(docker ps --format "table {{.Status}}" --filter "name=${CNCJS_CONTAINER_NAME}" | tail -n 1)
        
        # Check health
        if cncjs::health_check; then
            health="healthy"
            
            # Try to get controller state
            local api_response
            if api_response=$(timeout 5 curl -sf "http://localhost:${CNCJS_PORT}/api/state" 2>/dev/null); then
                controller_state="connected"
            fi
        else
            health="unhealthy"
        fi
    fi
    
    if [[ "$json_output" == "--json" ]]; then
        cat << EOF
{
  "service": "${CLI_NAME}",
  "status": "${status}",
  "health": "${health}",
  "uptime": "${uptime}",
  "controller_state": "${controller_state}",
  "port": ${CNCJS_PORT},
  "controller": "${CNCJS_CONTROLLER}",
  "serial_port": "${CNCJS_SERIAL_PORT}"
}
EOF
    else
        echo "CNCjs Status"
        echo "============"
        echo "Service:         ${status}"
        echo "Health:          ${health}"
        echo "Uptime:          ${uptime}"
        echo "Controller:      ${CNCJS_CONTROLLER}"
        echo "Serial Port:     ${CNCJS_SERIAL_PORT}"
        echo "Controller State: ${controller_state}"
        echo "Web Interface:   http://localhost:${CNCJS_PORT}"
    fi
}

#######################################
# View CNCjs logs
#######################################
cncjs::logs() {
    local follow="${1:-}"
    
    if ! docker ps -a --format "{{.Names}}" | grep -q "^${CNCJS_CONTAINER_NAME}$"; then
        log::error "CNCjs container not found"
        return 1
    fi
    
    if [[ "$follow" == "--follow" ]]; then
        docker logs -f "${CNCJS_CONTAINER_NAME}"
    else
        docker logs --tail 50 "${CNCJS_CONTAINER_NAME}"
    fi
}

#######################################
# Show connection credentials
#######################################
cncjs::credentials() {
    echo "CNCjs Connection Information"
    echo "============================"
    echo "Web Interface:   http://localhost:${CNCJS_PORT}"
    echo "Controller Type: ${CNCJS_CONTROLLER}"
    echo "Serial Port:     ${CNCJS_SERIAL_PORT}"
    echo "Baud Rate:       ${CNCJS_BAUD_RATE}"
    echo ""
    echo "Default Credentials:"
    echo "  No authentication required by default"
    echo "  Configure in ${CNCJS_CONFIG_FILE} if needed"
    echo ""
    echo "G-code Directory: ${CNCJS_WATCH_DIR}"
}

#######################################
# Content management
#######################################
cncjs::content() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            log::info "G-code files in ${CNCJS_WATCH_DIR}:"
            ls -la "${CNCJS_WATCH_DIR}" 2>/dev/null || echo "No files found"
            ;;
        add)
            local file="${1:-}"
            if [[ -z "$file" ]]; then
                log::error "Usage: content add <file>"
                return 1
            fi
            if [[ ! -f "$file" ]]; then
                log::error "File not found: $file"
                return 1
            fi
            cp "$file" "${CNCJS_WATCH_DIR}/"
            log::success "Added $(basename "$file") to CNCjs"
            ;;
        get)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: content get <name>"
                return 1
            fi
            local src="${CNCJS_WATCH_DIR}/${name}"
            if [[ ! -f "$src" ]]; then
                log::error "File not found: $name"
                return 1
            fi
            cat "$src"
            ;;
        remove)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: content remove <name>"
                return 1
            fi
            local target="${CNCJS_WATCH_DIR}/${name}"
            if [[ ! -f "$target" ]]; then
                log::error "File not found: $name"
                return 1
            fi
            rm "$target"
            log::success "Removed $name"
            ;;
        execute)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: content execute <name>"
                return 1
            fi
            log::info "Executing G-code: $name"
            log::warning "Note: Actual execution requires CNC controller connection"
            log::info "Use the web interface at http://localhost:${CNCJS_PORT} to monitor execution"
            ;;
        *)
            log::error "Unknown content action: $action"
            return 1
            ;;
    esac
}