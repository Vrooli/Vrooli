#!/bin/bash
# Traccar Core Library Functions

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"
CONFIG_DIR="${RESOURCE_DIR}/config"

# Source configuration
source "${CONFIG_DIR}/defaults.sh"

# Source common libraries if available
if [[ -f "/home/matthalloran8/Vrooli/scripts/common/logging.sh" ]]; then
    source "/home/matthalloran8/Vrooli/scripts/common/logging.sh"
else
    # Fallback logging functions
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
fi

# ==================== Help Command ====================
cmd_help() {
    cat << EOF
Traccar Fleet & Telematics Platform

USAGE:
    resource-traccar <command> [options]

COMMANDS:
    help                Show this help message
    info                Display runtime configuration
    manage              Lifecycle management commands
        install         Install Traccar and dependencies
        start           Start Traccar server
        stop            Stop Traccar server  
        restart         Restart Traccar server
        uninstall       Remove Traccar completely
    test                Run validation tests
        smoke           Quick health check
        integration     Full functionality test
        unit            Library function tests
        all             Run all tests
    content             Manage devices and data
        add             Add a device or GPS data
        list            List devices or tracks
        get             Get device details or track
        remove          Remove device or data
        execute         Process GPS data batch
    status              Show Traccar server status
    logs                View Traccar logs
    credentials         Display API credentials
    device              Device management commands
        create          Register new device
        list            List all devices
        update          Update device info
        delete          Remove device
    track               GPS tracking commands
        push            Send GPS position
        history         Get position history
        live            Start live tracking

EXAMPLES:
    # Install and start Traccar
    resource-traccar manage install
    resource-traccar manage start --wait
    
    # Create a demo device
    resource-traccar device create --name "Fleet-001" --type "truck"
    
    # Push GPS position
    resource-traccar track push --device "Fleet-001" --lat 37.7749 --lon -122.4194
    
    # Get device history
    resource-traccar track history --device "Fleet-001" --days 7

CONFIGURATION:
    Port:       ${TRACCAR_PORT}
    Host:       ${TRACCAR_HOST}
    Admin:      ${TRACCAR_ADMIN_EMAIL}
    Data Dir:   ${TRACCAR_DATA_DIR}

For more information, see: resources/traccar/README.md
EOF
}

# ==================== Info Command ====================
cmd_info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${CONFIG_DIR}/runtime.json"
    else
        echo "Traccar Runtime Information:"
        echo "  Startup Order:    500"
        echo "  Dependencies:     postgres"
        echo "  Startup Timeout:  60s"
        echo "  Startup Estimate: 30-60s"
        echo "  Recovery Attempts: 3"
        echo "  Priority:         high"
        echo "  Category:         mobility"
    fi
}

# ==================== Management Commands ====================
cmd_manage() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        install)
            manage_install "$@"
            ;;
        uninstall)
            manage_uninstall "$@"
            ;;
        start)
            manage_start "$@"
            ;;
        stop)
            manage_stop "$@"
            ;;
        restart)
            manage_restart "$@"
            ;;
        *)
            echo "Error: Unknown manage action: $action" >&2
            echo "Valid actions: install, uninstall, start, stop, restart" >&2
            exit 1
            ;;
    esac
}

# Install Traccar
manage_install() {
    log::info "Installing Traccar GPS tracking server..."
    
    # Create directories
    mkdir -p "${TRACCAR_DATA_DIR}" "${TRACCAR_CONFIG_DIR}" "${TRACCAR_LOGS_DIR}" "${TRACCAR_MEDIA_DIR}"
    
    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        log::error "Docker is required but not installed"
        exit 1
    fi
    
    # Pull Traccar Docker image
    log::info "Pulling Traccar Docker image ${TRACCAR_DOCKER_IMAGE}..."
    docker pull "${TRACCAR_DOCKER_IMAGE}"
    
    # Create Traccar configuration file with proper XML structure
    cat > "${TRACCAR_CONFIG_DIR}/traccar.xml" << 'EOF'
<?xml version='1.0' encoding='UTF-8'?>
<!DOCTYPE properties SYSTEM 'http://java.sun.com/dtd/properties.dtd'>
<properties>
    <entry key='config.default'>./conf/default.xml</entry>
    <entry key='web.enable'>true</entry>
    <entry key='web.port'>8082</entry>
    <entry key='web.path'>./web</entry>
    <entry key='geocoder.enable'>false</entry>
    
    <!-- In-memory H2 database for demo purposes -->
    <entry key='database.driver'>org.h2.Driver</entry>
    <entry key='database.url'>jdbc:h2:./data/database</entry>
    <entry key='database.user'>sa</entry>
    <entry key='database.password'></entry>
    
    <entry key='server.statistics'>false</entry>
    <entry key='notificator.types'>web,mail</entry>
</properties>
EOF
    
    log::success "Traccar installed successfully"
}

# Uninstall Traccar
manage_uninstall() {
    local keep_data=false
    [[ "${1:-}" == "--keep-data" ]] && keep_data=true
    
    log::info "Uninstalling Traccar..."
    
    # Stop container if running
    if docker ps -q -f name="${TRACCAR_CONTAINER_NAME}" 2>/dev/null; then
        docker stop "${TRACCAR_CONTAINER_NAME}"
        docker rm "${TRACCAR_CONTAINER_NAME}"
    fi
    
    # Remove data if requested
    if [[ "$keep_data" == false ]]; then
        rm -rf "${TRACCAR_DATA_DIR}"
        log::info "Removed Traccar data directory"
    fi
    
    log::success "Traccar uninstalled successfully"
}

# Start Traccar
manage_start() {
    local wait_ready=false
    local timeout="${TRACCAR_API_TIMEOUT}"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait)
                wait_ready=true
                ;;
            --timeout)
                timeout="$2"
                shift
                ;;
        esac
        shift
    done
    
    # Check if already running
    if docker ps -q -f name="${TRACCAR_CONTAINER_NAME}" 2>/dev/null | grep -q .; then
        log::info "Traccar is already running"
        exit 2
    fi
    
    log::info "Starting Traccar server..."
    
    # Start Traccar container
    docker run -d \
        --name "${TRACCAR_CONTAINER_NAME}" \
        -p "${TRACCAR_PORT}:8082" \
        -v "${TRACCAR_CONFIG_DIR}/traccar.xml:/opt/traccar/conf/traccar.xml:ro" \
        -v "${TRACCAR_DATA_DIR}:/opt/traccar/data:rw" \
        -v "${TRACCAR_LOGS_DIR}:/opt/traccar/logs:rw" \
        -v "${TRACCAR_MEDIA_DIR}:/opt/traccar/media:rw" \
        --restart unless-stopped \
        "${TRACCAR_DOCKER_IMAGE}"
    
    if [[ "$wait_ready" == true ]]; then
        log::info "Waiting for Traccar to be ready..."
        local elapsed=0
        while [[ $elapsed -lt $timeout ]]; do
            if timeout 5 curl -sf "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/server" &>/dev/null; then
                log::success "Traccar is ready"
                return 0
            fi
            sleep 2
            elapsed=$((elapsed + 2))
        done
        log::error "Traccar failed to become ready within ${timeout} seconds"
        exit 1
    fi
    
    log::success "Traccar started successfully"
}

# Stop Traccar
manage_stop() {
    local force=false
    local timeout=30
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force=true
                ;;
            --timeout)
                timeout="$2"
                shift
                ;;
        esac
        shift
    done
    
    if ! docker ps -q -f name="${TRACCAR_CONTAINER_NAME}" 2>/dev/null | grep -q .; then
        log::info "Traccar is not running"
        exit 2
    fi
    
    log::info "Stopping Traccar server..."
    
    if [[ "$force" == true ]]; then
        docker kill "${TRACCAR_CONTAINER_NAME}"
    else
        docker stop -t "${timeout}" "${TRACCAR_CONTAINER_NAME}"
    fi
    
    docker rm "${TRACCAR_CONTAINER_NAME}"
    
    log::success "Traccar stopped successfully"
}

# Restart Traccar
manage_restart() {
    manage_stop "$@"
    manage_start "$@"
}

# ==================== Test Commands ====================
cmd_test() {
    local test_type="${1:-all}"
    
    case "$test_type" in
        smoke|integration|unit|all)
            source "${SCRIPT_DIR}/test.sh"
            test_run "$test_type"
            ;;
        *)
            echo "Error: Unknown test type: $test_type" >&2
            echo "Valid types: smoke, integration, unit, all" >&2
            exit 1
            ;;
    esac
}

# ==================== Content Commands ====================
cmd_content() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        add|list|get|remove|execute)
            source "${SCRIPT_DIR}/content.sh"
            content_"${action}" "$@"
            ;;
        *)
            echo "Error: Unknown content action: $action" >&2
            echo "Valid actions: add, list, get, remove, execute" >&2
            exit 1
            ;;
    esac
}

# ==================== Status Command ====================
cmd_status() {
    local format="${1:-text}"
    
    if docker ps -q -f name="${TRACCAR_CONTAINER_NAME}" 2>/dev/null | grep -q .; then
        if timeout 5 curl -sf "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/server" &>/dev/null; then
            if [[ "$format" == "--json" ]]; then
                echo '{"status":"running","health":"healthy","port":'${TRACCAR_PORT}',"url":"http://'${TRACCAR_HOST}':'${TRACCAR_PORT}'"}'
            else
                echo "Traccar Status: Running"
                echo "  Health:     Healthy"
                echo "  Port:       ${TRACCAR_PORT}"
                echo "  URL:        http://${TRACCAR_HOST}:${TRACCAR_PORT}"
                echo "  Container:  ${TRACCAR_CONTAINER_NAME}"
            fi
            exit 0
        else
            if [[ "$format" == "--json" ]]; then
                echo '{"status":"running","health":"unhealthy","port":'${TRACCAR_PORT}'}'
            else
                echo "Traccar Status: Running but unhealthy"
            fi
            exit 1
        fi
    else
        if [[ "$format" == "--json" ]]; then
            echo '{"status":"stopped","health":"n/a"}'
        else
            echo "Traccar Status: Not running"
        fi
        exit 2
    fi
}

# ==================== Logs Command ====================
cmd_logs() {
    local tail_lines="${1:-50}"
    local follow=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tail)
                tail_lines="$2"
                shift
                ;;
            --follow)
                follow=true
                ;;
        esac
        shift
    done
    
    if docker ps -q -f name="${TRACCAR_CONTAINER_NAME}" 2>/dev/null | grep -q .; then
        if [[ "$follow" == true ]]; then
            docker logs -f --tail "${tail_lines}" "${TRACCAR_CONTAINER_NAME}"
        else
            docker logs --tail "${tail_lines}" "${TRACCAR_CONTAINER_NAME}"
        fi
    else
        if [[ -f "${TRACCAR_LOGS_DIR}/tracker-server.log" ]]; then
            tail -n "${tail_lines}" "${TRACCAR_LOGS_DIR}/tracker-server.log"
        else
            echo "No logs available - Traccar is not running"
            exit 2
        fi
    fi
}

# ==================== Credentials Command ====================
cmd_credentials() {
    local format="${1:-text}"
    local show_secrets="${2:-false}"
    
    if [[ "$format" == "--json" ]]; then
        cat << EOF
{
  "admin_email": "${TRACCAR_ADMIN_EMAIL}",
  "admin_password": ${show_secrets:+"\"${TRACCAR_ADMIN_PASSWORD}\""},
  "api_url": "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api",
  "web_url": "http://${TRACCAR_HOST}:${TRACCAR_PORT}"
}
EOF
    else
        echo "Traccar Credentials:"
        echo "  Admin Email:    ${TRACCAR_ADMIN_EMAIL}"
        [[ "$show_secrets" == "--show-secrets" ]] && echo "  Admin Password: ${TRACCAR_ADMIN_PASSWORD}"
        echo "  API URL:        http://${TRACCAR_HOST}:${TRACCAR_PORT}/api"
        echo "  Web URL:        http://${TRACCAR_HOST}:${TRACCAR_PORT}"
    fi
}

# ==================== Device Commands ====================
cmd_device() {
    local action="${1:-}"
    shift || true
    
    source "${SCRIPT_DIR}/device.sh"
    
    case "$action" in
        create|list|update|delete)
            device_"${action}" "$@"
            ;;
        *)
            echo "Error: Unknown device action: $action" >&2
            echo "Valid actions: create, list, update, delete" >&2
            exit 1
            ;;
    esac
}

# ==================== Track Commands ====================
cmd_track() {
    local action="${1:-}"
    shift || true
    
    source "${SCRIPT_DIR}/track.sh"
    
    case "$action" in
        push|history|live)
            track_"${action}" "$@"
            ;;
        *)
            echo "Error: Unknown track action: $action" >&2
            echo "Valid actions: push, history, live" >&2
            exit 1
            ;;
    esac
}