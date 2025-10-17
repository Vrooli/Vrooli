#!/bin/bash

# OctoPrint Core Library
# Core functionality for OctoPrint resource management

set -euo pipefail

# Installation function
octoprint_install() {
    echo "Installing OctoPrint..."
    
    # Create necessary directories
    mkdir -p "${OCTOPRINT_DATA_DIR}"
    mkdir -p "${OCTOPRINT_CONFIG_DIR}"
    mkdir -p "${OCTOPRINT_GCODE_DIR}"
    mkdir -p "${OCTOPRINT_TIMELAPSE_DIR}"
    mkdir -p "${OCTOPRINT_PLUGIN_DIR}"
    mkdir -p "$(dirname "${OCTOPRINT_LOG_FILE}")"
    
    # Check if Docker is available
    if command -v docker &> /dev/null; then
        echo "Docker detected, pulling OctoPrint image..."
        docker pull "${OCTOPRINT_DOCKER_IMAGE}"
        echo "OctoPrint Docker image ready"
    else
        echo "Docker not available, preparing for native installation..."
        # Create Python virtual environment
        if command -v python3 &> /dev/null; then
            python3 -m venv "${OCTOPRINT_INSTALL_DIR}/venv"
            source "${OCTOPRINT_INSTALL_DIR}/venv/bin/activate"
            pip install --upgrade pip
            pip install OctoPrint
            deactivate
            echo "OctoPrint installed in virtual environment"
        else
            echo "Error: Python 3 is required for native installation"
            exit 1
        fi
    fi
    
    # Generate API key if set to auto
    if [[ "${OCTOPRINT_API_KEY}" == "auto" ]]; then
        OCTOPRINT_API_KEY="$(openssl rand -hex 32 2>/dev/null || date +%s | sha256sum | cut -d' ' -f1)"
        echo "Generated API key: ${OCTOPRINT_API_KEY}"
        echo "${OCTOPRINT_API_KEY}" > "${OCTOPRINT_CONFIG_DIR}/api_key"
        
        # Create OctoPrint config.yaml with API key for Docker container
        # Write to data directory - container will read from mounted volume
        cat > "${OCTOPRINT_DATA_DIR}/config.yaml" <<EOF
api:
  enabled: true
  key: ${OCTOPRINT_API_KEY}
  allowCrossOrigin: true
accessControl:
  enabled: false
  salt: "OqKAR1DDQsfasYNqPeYBzy51Ay5UQZK4"
  userManager: octoprint.access.users.FilebasedUserManager
  userfile: users.yaml
  autologinLocal: true
  autologinAs: admin
  localNetworks:
  - 127.0.0.0/8
  - ::1/128
  - 192.168.0.0/16
  - 172.16.0.0/12
  - 10.0.0.0/8
  - fc00::/7
  - fe80::/10
server:
  host: 0.0.0.0
  port: 5000
  firstRun: false
  commands:
    serverRestartCommand: false
    systemRestartCommand: false
    systemShutdownCommand: false
devel:
  virtualPrinter:
    enabled: ${OCTOPRINT_VIRTUAL_PRINTER}
plugins:
  virtual_printer:
    enabled: ${OCTOPRINT_VIRTUAL_PRINTER}
serial:
  autoconnect: ${OCTOPRINT_VIRTUAL_PRINTER}
  baudrate: 115200
  port: VIRTUAL
  disconnectOnErrors: false
EOF
    fi
    
    echo "OctoPrint installation complete"
}

# Start function
octoprint_start() {
    local wait_flag=false
    if [[ "${1:-}" == "--wait" ]]; then
        wait_flag=true
    fi
    
    echo "Starting OctoPrint on port ${OCTOPRINT_PORT}..."
    
    # Check if already running
    if octoprint_is_running; then
        echo "OctoPrint is already running"
        return 0
    fi
    
    # Ensure config is in place before starting
    if [[ "${OCTOPRINT_VIRTUAL_PRINTER}" == "true" ]] && [[ -f "${OCTOPRINT_DATA_DIR}/config.yaml" ]]; then
        # Copy our config to the container's expected location
        if [[ -d "${OCTOPRINT_DATA_DIR}/octoprint" ]]; then
            cp "${OCTOPRINT_DATA_DIR}/config.yaml" "${OCTOPRINT_DATA_DIR}/octoprint/config.yaml" 2>/dev/null || true
        fi
    fi
    
    # Start with Docker if available
    if command -v docker &> /dev/null; then
        local docker_args=(
            "--name" "${OCTOPRINT_CONTAINER_NAME}"
            "-d"
            "-p" "${OCTOPRINT_PORT}:5000"
            "-v" "${OCTOPRINT_DATA_DIR}:/octoprint"
            "-v" "${OCTOPRINT_GCODE_DIR}:/octoprint/uploads"
            "--restart" "unless-stopped"
        )
        
        # Add serial port access if not using virtual printer
        if [[ "${OCTOPRINT_VIRTUAL_PRINTER}" != "true" ]]; then
            docker_args+=("--device" "${OCTOPRINT_PRINTER_PORT}:${OCTOPRINT_PRINTER_PORT}")
            docker_args+=("--privileged")
        fi
        
        # Add environment variables
        docker_args+=(
            "-e" "OCTOPRINT_PORT=5000"
            "-e" "ENABLE_MJPG_STREAMER=${OCTOPRINT_CAMERA_ENABLED}"
            "-e" "HOME=/octoprint"
        )
        
        docker run "${docker_args[@]}" "${OCTOPRINT_DOCKER_IMAGE}"
        
    else
        # Native installation
        source "${OCTOPRINT_INSTALL_DIR}/venv/bin/activate"
        octoprint serve \
            --port "${OCTOPRINT_PORT}" \
            --host "${OCTOPRINT_HOST}" \
            --basedir "${OCTOPRINT_DATA_DIR}" \
            > "${OCTOPRINT_LOG_FILE}" 2>&1 &
        
        local pid=$!
        echo "${pid}" > "${OCTOPRINT_DATA_DIR}/octoprint.pid"
        deactivate
    fi
    
    if [[ "${wait_flag}" == true ]]; then
        echo "Waiting for OctoPrint to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ ${attempt} -lt ${max_attempts} ]]; do
            if timeout 5 curl -sf "http://localhost:${OCTOPRINT_PORT}/api/version" &>/dev/null; then
                echo "OctoPrint is ready"
                
                # Auto-connect virtual printer if enabled
                if [[ "${OCTOPRINT_VIRTUAL_PRINTER}" == "true" ]]; then
                    echo "Attempting to auto-connect virtual printer..."
                    octoprint_connect_virtual_printer
                fi
                
                return 0
            fi
            sleep 2
            ((attempt++))
        done
        
        echo "Warning: OctoPrint did not become ready within timeout"
        return 1
    fi
    
    echo "OctoPrint started"
}

# Stop function
octoprint_stop() {
    echo "Stopping OctoPrint..."
    
    if command -v docker &> /dev/null; then
        if docker ps -q -f name="${OCTOPRINT_CONTAINER_NAME}" | grep -q .; then
            docker stop "${OCTOPRINT_CONTAINER_NAME}"
            docker rm "${OCTOPRINT_CONTAINER_NAME}"
            echo "OctoPrint container stopped"
        else
            echo "OctoPrint container not running"
        fi
    else
        if [[ -f "${OCTOPRINT_DATA_DIR}/octoprint.pid" ]]; then
            local pid=$(cat "${OCTOPRINT_DATA_DIR}/octoprint.pid")
            if kill -0 "${pid}" 2>/dev/null; then
                kill "${pid}"
                rm -f "${OCTOPRINT_DATA_DIR}/octoprint.pid"
                echo "OctoPrint process stopped"
            else
                echo "OctoPrint process not running"
                rm -f "${OCTOPRINT_DATA_DIR}/octoprint.pid"
            fi
        else
            echo "OctoPrint not running"
        fi
    fi
}

# Restart function
octoprint_restart() {
    echo "Restarting OctoPrint..."
    octoprint_stop
    sleep 2
    octoprint_start --wait
}

# Uninstall function
octoprint_uninstall() {
    echo "Uninstalling OctoPrint..."
    
    # Stop if running
    octoprint_stop
    
    # Remove Docker image if used
    if command -v docker &> /dev/null; then
        docker rmi "${OCTOPRINT_DOCKER_IMAGE}" 2>/dev/null || true
    fi
    
    # Clean up directories (preserve user data by default)
    echo "Note: User data preserved in ${OCTOPRINT_DATA_DIR}"
    echo "Remove manually if no longer needed"
    
    echo "OctoPrint uninstalled"
}

# Check if running
octoprint_is_running() {
    if command -v docker &> /dev/null; then
        docker ps -q -f name="${OCTOPRINT_CONTAINER_NAME}" | grep -q .
    else
        [[ -f "${OCTOPRINT_DATA_DIR}/octoprint.pid" ]] && \
        kill -0 "$(cat "${OCTOPRINT_DATA_DIR}/octoprint.pid")" 2>/dev/null
    fi
}

# Status function
octoprint_status() {
    echo "OctoPrint Status"
    echo "================"
    
    if octoprint_is_running; then
        echo "Status: Running"
        echo "Port: ${OCTOPRINT_PORT}"
        echo "URL: http://localhost:${OCTOPRINT_PORT}"
        
        # Try to get printer status via API
        local api_key="${OCTOPRINT_API_KEY}"
        if [[ -f "${OCTOPRINT_CONFIG_DIR}/api_key" ]]; then
            api_key=$(cat "${OCTOPRINT_CONFIG_DIR}/api_key")
        fi
        
        if [[ -n "${api_key}" ]] && [[ "${api_key}" != "auto" ]]; then
            local printer_status=$(timeout 5 curl -sf -H "X-Api-Key: ${api_key}" \
                "http://localhost:${OCTOPRINT_PORT}/api/printer" 2>/dev/null || echo "{}")
            
            if [[ "${printer_status}" != "{}" ]]; then
                echo ""
                echo "Printer Status:"
                echo "${printer_status}" | python3 -m json.tool 2>/dev/null || echo "${printer_status}"
            fi
        fi
    else
        echo "Status: Not running"
    fi
    
    # Show Docker container status if applicable
    if command -v docker &> /dev/null; then
        echo ""
        echo "Container Status:"
        docker ps -a --filter name="${OCTOPRINT_CONTAINER_NAME}" --format "table {{.Status}}\t{{.Ports}}" 2>/dev/null || echo "No container"
    fi
}

# Logs function
octoprint_logs() {
    local lines="${1:-50}"
    
    echo "OctoPrint Logs (last ${lines} lines)"
    echo "===================================="
    
    if command -v docker &> /dev/null && docker ps -q -f name="${OCTOPRINT_CONTAINER_NAME}" | grep -q .; then
        docker logs --tail "${lines}" "${OCTOPRINT_CONTAINER_NAME}"
    elif [[ -f "${OCTOPRINT_LOG_FILE}" ]]; then
        tail -n "${lines}" "${OCTOPRINT_LOG_FILE}"
    else
        echo "No logs available"
    fi
}

# Credentials function
octoprint_credentials() {
    echo "OctoPrint Credentials"
    echo "===================="
    echo "URL: http://localhost:${OCTOPRINT_PORT}"
    
    local api_key=""
    
    # Try to get API key from running container
    if command -v docker &> /dev/null && docker ps -q -f name="${OCTOPRINT_CONTAINER_NAME}" | grep -q .; then
        api_key=$(docker exec "${OCTOPRINT_CONTAINER_NAME}" grep "key:" /octoprint/octoprint/config.yaml 2>/dev/null | head -1 | sed 's/.*key: *//')
    fi
    
    # Fall back to stored key if container not running
    if [[ -z "${api_key}" ]] && [[ -f "${OCTOPRINT_CONFIG_DIR}/api_key" ]]; then
        api_key=$(cat "${OCTOPRINT_CONFIG_DIR}/api_key")
    fi
    
    if [[ -n "${api_key}" ]] && [[ "${api_key}" != "auto" ]]; then
        echo "API Key: ${api_key}"
        echo ""
        echo "Example API Usage:"
        echo "  curl -H \"X-Api-Key: ${api_key}\" http://localhost:${OCTOPRINT_PORT}/api/printer"
        
        # Store the active key for future use
        echo "${api_key}" > "${OCTOPRINT_CONFIG_DIR}/api_key"
    else
        echo "API Key: Not configured"
    fi
}

# Content management functions
content_list() {
    echo "G-Code Files:"
    echo "============="
    
    if [[ -d "${OCTOPRINT_GCODE_DIR}" ]]; then
        find "${OCTOPRINT_GCODE_DIR}" -name "*.gcode" -o -name "*.g" | while read -r file; do
            basename "${file}"
        done
    else
        echo "No G-code files found"
    fi
}

content_add() {
    local file="${1:-}"
    
    if [[ -z "${file}" ]]; then
        echo "Error: File path required"
        echo "Usage: content add <file>"
        exit 1
    fi
    
    if [[ ! -f "${file}" ]]; then
        echo "Error: File not found: ${file}"
        exit 1
    fi
    
    echo "Uploading ${file}..."
    cp "${file}" "${OCTOPRINT_GCODE_DIR}/"
    echo "File uploaded: $(basename "${file}")"
}

content_get() {
    local file="${1:-}"
    
    if [[ -z "${file}" ]]; then
        echo "Error: Filename required"
        echo "Usage: content get <file>"
        exit 1
    fi
    
    local filepath="${OCTOPRINT_GCODE_DIR}/${file}"
    if [[ ! -f "${filepath}" ]]; then
        echo "Error: File not found: ${file}"
        exit 1
    fi
    
    cat "${filepath}"
}

content_remove() {
    local file="${1:-}"
    
    if [[ -z "${file}" ]]; then
        echo "Error: Filename required"
        echo "Usage: content remove <file>"
        exit 1
    fi
    
    local filepath="${OCTOPRINT_GCODE_DIR}/${file}"
    if [[ ! -f "${filepath}" ]]; then
        echo "Error: File not found: ${file}"
        exit 1
    fi
    
    rm "${filepath}"
    echo "File removed: ${file}"
}

content_execute() {
    local file="${1:-}"
    
    if [[ -z "${file}" ]]; then
        echo "Error: Filename required"
        echo "Usage: content execute <file>"
        exit 1
    fi
    
    echo "Starting print job: ${file}"
    
    local api_key="${OCTOPRINT_API_KEY}"
    if [[ -f "${OCTOPRINT_CONFIG_DIR}/api_key" ]]; then
        api_key=$(cat "${OCTOPRINT_CONFIG_DIR}/api_key")
    fi
    
    if [[ -n "${api_key}" ]] && [[ "${api_key}" != "auto" ]]; then
        # Select file
        curl -X POST -H "X-Api-Key: ${api_key}" \
            "http://localhost:${OCTOPRINT_PORT}/api/files/local/${file}" \
            -d '{"command":"select","print":true}'
        
        echo "Print job started"
    else
        echo "Error: API key not configured"
        exit 1
    fi
}

# Helper function to get API key
get_api_key() {
    local api_key=""
    
    # Try to get API key from running container first
    if command -v docker &> /dev/null && docker ps -q -f name="${OCTOPRINT_CONTAINER_NAME}" | grep -q .; then
        api_key=$(docker exec "${OCTOPRINT_CONTAINER_NAME}" grep "key:" /octoprint/octoprint/config.yaml 2>/dev/null | head -1 | sed 's/.*key: *//')
    fi
    
    # Fall back to stored key if container not running
    if [[ -z "${api_key}" ]] && [[ -f "${OCTOPRINT_CONFIG_DIR}/api_key" ]]; then
        api_key=$(cat "${OCTOPRINT_CONFIG_DIR}/api_key")
    fi
    
    # Store the active key for future use if we found it
    if [[ -n "${api_key}" ]] && [[ "${api_key}" != "auto" ]] && [[ -n "${OCTOPRINT_CONFIG_DIR}" ]]; then
        mkdir -p "${OCTOPRINT_CONFIG_DIR}"
        echo "${api_key}" > "${OCTOPRINT_CONFIG_DIR}/api_key"
    fi
    
    echo "${api_key}"
}

# Helper function to connect virtual printer
octoprint_connect_virtual_printer() {
    local api_key=$(get_api_key)
    if [[ -z "${api_key}" ]] || [[ "${api_key}" == "auto" ]]; then
        echo "Warning: Cannot auto-connect virtual printer - API key not available"
        return 1
    fi
    
    # Check if already connected
    local connection_status=$(timeout 5 curl -sf -H "X-Api-Key: ${api_key}" \
        "http://localhost:${OCTOPRINT_PORT}/api/connection" 2>/dev/null | \
        python3 -c "import sys, json; print(json.load(sys.stdin).get('current', {}).get('state', 'Unknown'))" 2>/dev/null || echo "Unknown")
    
    if [[ "${connection_status}" != "Closed" ]] && [[ "${connection_status}" != "Unknown" ]]; then
        echo "Virtual printer already connected (state: ${connection_status})"
        return 0
    fi
    
    # Attempt to connect
    echo "Connecting virtual printer..."
    local response=$(timeout 5 curl -X POST -H "X-Api-Key: ${api_key}" \
        -H "Content-Type: application/json" \
        "http://localhost:${OCTOPRINT_PORT}/api/connection" \
        -d '{"command":"connect","port":"VIRTUAL","baudrate":115200,"save":true,"autoconnect":true}' \
        -w "\n%{http_code}" 2>/dev/null)
    
    local http_code=$(echo "${response}" | tail -1)
    
    if [[ "${http_code}" == "204" ]] || [[ "${http_code}" == "200" ]]; then
        echo "Virtual printer connected successfully"
        return 0
    else
        echo "Note: Virtual printer connection may require manual setup via web interface"
        echo "Access http://localhost:${OCTOPRINT_PORT} to complete setup"
        return 0  # Don't fail - just notify
    fi
}

# Temperature monitoring
octoprint_temperature() {
    echo "Temperature Status"
    echo "=================="
    
    local api_key=$(get_api_key)
    if [[ -z "${api_key}" ]] || [[ "${api_key}" == "auto" ]]; then
        echo "Error: API key not configured"
        return 1
    fi
    
    # First check printer connection status
    local connection_status=$(timeout 5 curl -sf -H "X-Api-Key: ${api_key}" \
        "http://localhost:${OCTOPRINT_PORT}/api/connection" 2>/dev/null | \
        python3 -c "import sys, json; print(json.load(sys.stdin).get('current', {}).get('state', 'Unknown'))" 2>/dev/null)
    
    if [[ "${connection_status}" == "Closed" ]] || [[ "${connection_status}" == "Unknown" ]]; then
        echo "Printer Status: Not connected"
        if [[ "${OCTOPRINT_VIRTUAL_PRINTER}" == "true" ]]; then
            echo ""
            echo "Virtual Printer Mode: Simulated temperatures"
            echo "  Hotend: 22.0°C / 0.0°C (ambient)"
            echo "  Bed: 22.0°C / 0.0°C (ambient)"
            echo ""
            echo "Note: Connect virtual printer via web interface for live data"
        else
            echo "Note: Connect printer to see temperature data"
        fi
        return 0
    fi
    
    # Get actual temperature data if connected
    local temp_data=$(timeout 5 curl -sf -H "X-Api-Key: ${api_key}" \
        "http://localhost:${OCTOPRINT_PORT}/api/printer" 2>/dev/null)
    
    if [[ -n "${temp_data}" ]]; then
        # Parse and display temperature data
        echo "${temp_data}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if 'temperature' in data:
        temps = data['temperature']
        if 'tool0' in temps:
            print(f\"  Hotend: {temps['tool0']['actual']}°C / {temps['tool0']['target']}°C\")
        if 'bed' in temps:
            print(f\"  Bed: {temps['bed']['actual']}°C / {temps['bed']['target']}°C\")
    else:
        print('  No temperature data available')
except:
    print('  Error parsing temperature data')
" 2>/dev/null || echo "  Unable to parse temperature data"
    else
        echo "Unable to fetch temperature data"
    fi
}

# Print control commands
octoprint_print_control() {
    local command="${1:-}"
    
    if [[ -z "${command}" ]]; then
        echo "Error: Command required"
        echo "Usage: print-control <start|pause|cancel|resume>"
        return 1
    fi
    
    local api_key=$(get_api_key)
    if [[ -z "${api_key}" ]] || [[ "${api_key}" == "auto" ]]; then
        echo "Error: API key not configured"
        return 1
    fi
    
    case "${command}" in
        start)
            echo "Starting print..."
            curl -X POST -H "X-Api-Key: ${api_key}" \
                "http://localhost:${OCTOPRINT_PORT}/api/job" \
                -d '{"command":"start"}'
            ;;
        pause)
            echo "Pausing print..."
            curl -X POST -H "X-Api-Key: ${api_key}" \
                "http://localhost:${OCTOPRINT_PORT}/api/job" \
                -d '{"command":"pause","action":"pause"}'
            ;;
        resume)
            echo "Resuming print..."
            curl -X POST -H "X-Api-Key: ${api_key}" \
                "http://localhost:${OCTOPRINT_PORT}/api/job" \
                -d '{"command":"pause","action":"resume"}'
            ;;
        cancel)
            echo "Cancelling print..."
            curl -X POST -H "X-Api-Key: ${api_key}" \
                "http://localhost:${OCTOPRINT_PORT}/api/job" \
                -d '{"command":"cancel"}'
            ;;
        *)
            echo "Error: Invalid command: ${command}"
            echo "Valid commands: start, pause, resume, cancel"
            return 1
            ;;
    esac
    
    echo ""
    echo "Command sent: ${command}"
}

# Job status
octoprint_job_status() {
    echo "Print Job Status"
    echo "================"
    
    local api_key=$(get_api_key)
    if [[ -z "${api_key}" ]] || [[ "${api_key}" == "auto" ]]; then
        echo "Error: API key not configured"
        return 1
    fi
    
    local job_data=$(timeout 5 curl -sf -H "X-Api-Key: ${api_key}" \
        "http://localhost:${OCTOPRINT_PORT}/api/job" 2>/dev/null)
    
    if [[ -n "${job_data}" ]]; then
        echo "${job_data}" | python3 -m json.tool 2>/dev/null || echo "${job_data}"
    else
        echo "Unable to fetch job data"
    fi
}

# Webcam configuration (P1 requirement)
octoprint_webcam_setup() {
    echo "Webcam Configuration"
    echo "===================="
    
    local api_key=$(get_api_key)
    if [[ -z "${api_key}" ]] || [[ "${api_key}" == "auto" ]]; then
        echo "Error: API key not configured"
        return 1
    fi
    
    # Check current webcam settings
    local webcam_status=$(timeout 5 curl -sf -H "X-Api-Key: ${api_key}" \
        "http://localhost:${OCTOPRINT_PORT}/api/settings" 2>/dev/null | \
        python3 -c "import sys, json; print(json.load(sys.stdin).get('webcam', {}))" 2>/dev/null || echo "{}")
    
    if [[ "${webcam_status}" != "{}" ]]; then
        echo "Current webcam settings:"
        echo "${webcam_status}" | python3 -m json.tool
    else
        echo "Webcam not configured"
        echo ""
        echo "To enable webcam:"
        echo "1. Install mjpg-streamer on your system"
        echo "2. Set OCTOPRINT_CAMERA_ENABLED=true"
        echo "3. Configure OCTOPRINT_CAMERA_URL (default: http://localhost:8080/?action=stream)"
        echo "4. Restart OctoPrint"
    fi
}