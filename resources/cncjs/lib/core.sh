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
        log::error "CNCjs image not found"
        log::info "Recovery hint: Run 'vrooli resource cncjs manage install' to download the image"
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
        --allow-remote-access 2>&1; then
        local error_msg=$(docker logs "${CNCJS_CONTAINER_NAME}" 2>&1 | tail -n 5)
        log::error "Failed to start CNCjs container"
        
        # Check for common issues
        if echo "${error_msg}" | grep -q "port is already allocated"; then
            log::info "Recovery hint: Port ${CNCJS_PORT} is in use. Stop the conflicting service or change the port"
        elif echo "${error_msg}" | grep -q "permission denied"; then
            log::info "Recovery hint: Docker requires elevated permissions for serial port access"
        else
            log::info "Recovery hint: Check Docker logs with 'docker logs ${CNCJS_CONTAINER_NAME}'"
        fi
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
    
    # Check HTTP endpoint with proper timeout and response validation
    local response
    if ! response=$(timeout 5 curl -sf "http://localhost:${CNCJS_PORT}/" 2>/dev/null); then
        return 1
    fi
    
    # Verify response contains CNCjs content
    if ! echo "$response" | grep -q "CNCjs"; then
        return 1
    fi
    
    return 0
}

#######################################
# Provide health endpoint response
#######################################
cncjs::health() {
    local start_time=$(date +%s%3N)
    local status="unhealthy"
    local controller_state="unknown"
    local message="Service not available"
    local uptime="0s"
    
    # Check if container is running
    if docker ps --format "{{.Names}}" | grep -q "^${CNCJS_CONTAINER_NAME}$"; then
        # Get container uptime
        uptime=$(docker ps --format "table {{.Status}}" --filter "name=${CNCJS_CONTAINER_NAME}" | tail -n 1)
        
        # Check web interface (using v2.0 standard timeout)
        if timeout 5 curl -sf "http://localhost:${CNCJS_PORT}/" &>/dev/null; then
            status="healthy"
            message="CNCjs web interface is accessible"
            
            # Check for controller connection (simulated since we don't have hardware)
            if [[ -e "${CNCJS_SERIAL_PORT}" ]]; then
                controller_state="connected"
            else
                controller_state="disconnected (no hardware)"
            fi
        else
            status="degraded"
            message="Container running but web interface not responding"
        fi
    fi
    
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    # Output JSON health response
    cat <<EOF
{
  "status": "${status}",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "response_time_ms": ${response_time},
  "service": {
    "name": "cncjs",
    "version": "latest",
    "uptime": "${uptime}",
    "port": ${CNCJS_PORT}
  },
  "controller": {
    "type": "${CNCJS_CONTROLLER}",
    "state": "${controller_state}",
    "port": "${CNCJS_SERIAL_PORT}"
  },
  "message": "${message}"
}
EOF
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
            # Note: API requires authentication, so we check if service is responding
            if timeout 5 curl -sf "http://localhost:${CNCJS_PORT}/" &>/dev/null; then
                # Controller state requires actual hardware connection
                controller_state="service ready (controller not connected)"
            fi
        else
            health="unhealthy"
        fi
    fi
    
    if [[ "$json_output" == "--json" ]]; then
        cat << EOF
{
  "service": "cncjs",
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
    
    if [[ "$follow" == "--follow" || "$follow" == "-f" ]]; then
        log::info "Following CNCjs logs (Ctrl+C to exit)..."
        docker logs -f "${CNCJS_CONTAINER_NAME}"
    elif [[ "$follow" == "--tail" ]]; then
        local lines="${2:-50}"
        docker logs --tail "$lines" "${CNCJS_CONTAINER_NAME}"
    else
        log::info "Showing last 50 lines of CNCjs logs (use --follow for live logs)..."
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

#######################################
# Macro management for automation
#######################################
cncjs::macro() {
    local action="${1:-list}"
    shift || true
    
    local macro_dir="${CNCJS_DATA_DIR}/macros"
    mkdir -p "$macro_dir"
    
    case "$action" in
        list)
            log::info "Available macros:"
            if [[ -d "$macro_dir" ]]; then
                for macro in "$macro_dir"/*.macro; do
                    if [[ -f "$macro" ]]; then
                        local name=$(basename "$macro" .macro)
                        local desc=$(head -n 1 "$macro" | sed 's/^; *//')
                        echo "  - $name: $desc"
                    fi
                done 2>/dev/null
                if ! ls "$macro_dir"/*.macro &>/dev/null; then
                    echo "  No macros found"
                fi
            else
                echo "  No macros found"
            fi
            ;;
        add)
            local name="${1:-}"
            local gcode="${2:-}"
            if [[ -z "$name" ]] || [[ -z "$gcode" ]]; then
                log::error "Usage: macro add <name> <gcode_or_file>"
                return 1
            fi
            
            local macro_file="$macro_dir/${name}.macro"
            if [[ -f "$gcode" ]]; then
                # Copy file as macro
                cp "$gcode" "$macro_file"
            else
                # Save as direct G-code command with description
                echo "; $name - User-defined macro" > "$macro_file"
                echo "; Created: $(date)" >> "$macro_file"
                echo "" >> "$macro_file"
                echo "$gcode" >> "$macro_file"
            fi
            log::success "Macro '$name' saved"
            ;;
        run)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: macro run <name>"
                return 1
            fi
            
            local macro_file="$macro_dir/${name}.macro"
            if [[ ! -f "$macro_file" ]]; then
                log::error "Macro not found: $name"
                return 1
            fi
            
            # Copy macro to watch directory for execution
            cp "$macro_file" "${CNCJS_WATCH_DIR}/${name}_$(date +%s).gcode"
            log::info "Macro '$name' queued for execution"
            log::warning "Note: Actual execution requires CNC controller connection"
            ;;
        remove)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: macro remove <name>"
                return 1
            fi
            
            local macro_file="$macro_dir/${name}.macro"
            if [[ ! -f "$macro_file" ]]; then
                log::error "Macro not found: $name"
                return 1
            fi
            
            rm "$macro_file"
            log::success "Macro '$name' removed"
            ;;
        *)
            log::error "Unknown macro action: $action"
            echo "Valid actions: list, add, run, remove"
            return 1
            ;;
    esac
}

#######################################
# Workflow management for job sequences
#######################################
cncjs::workflow() {
    local action="${1:-list}"
    shift || true
    
    local workflow_dir="${CNCJS_DATA_DIR}/workflows"
    mkdir -p "$workflow_dir"
    
    case "$action" in
        list)
            log::info "Available workflows:"
            if [[ -d "$workflow_dir" ]]; then
                for workflow in "$workflow_dir"/*.workflow; do
                    if [[ -f "$workflow" ]]; then
                        local name=$(basename "$workflow" .workflow)
                        local desc=$(grep "^description:" "$workflow" 2>/dev/null | cut -d: -f2- | xargs)
                        local steps=$(grep -c "^  - " "$workflow" 2>/dev/null || echo "0")
                        echo "  - $name: $desc (${steps} steps)"
                    fi
                done 2>/dev/null
                if ! ls "$workflow_dir"/*.workflow &>/dev/null; then
                    echo "  No workflows found"
                fi
            else
                echo "  No workflows found"
            fi
            ;;
        create)
            local name="${1:-}"
            local description="${2:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: workflow create <name> [description]"
                return 1
            fi
            
            local workflow_file="$workflow_dir/${name}.workflow"
            cat > "$workflow_file" << EOF
name: $name
description: ${description:-CNC workflow for $name}
created: $(date --iso-8601=seconds)
version: 1.0
steps: []
settings:
  controller: ${CNCJS_CONTROLLER}
  baud_rate: ${CNCJS_BAUD_RATE}
  serial_port: ${CNCJS_SERIAL_PORT}
EOF
            log::success "Workflow '$name' created"
            ;;
        add-step)
            local workflow="${1:-}"
            local gcode_file="${2:-}"
            local description="${3:-}"
            if [[ -z "$workflow" ]] || [[ -z "$gcode_file" ]]; then
                log::error "Usage: workflow add-step <workflow> <gcode_file> [description]"
                return 1
            fi
            
            local workflow_file="$workflow_dir/${workflow}.workflow"
            if [[ ! -f "$workflow_file" ]]; then
                log::error "Workflow not found: $workflow"
                return 1
            fi
            
            # Add step to workflow
            if ! grep -q "^steps:$" "$workflow_file"; then
                echo "steps:" >> "$workflow_file"
            fi
            
            cat >> "$workflow_file" << EOF
  - file: $(basename "$gcode_file")
    description: ${description:-Step $(grep -c "^  - " "$workflow_file" 2>/dev/null || echo "1")}
    added: $(date --iso-8601=seconds)
EOF
            
            # Copy G-code file to workflow directory
            if [[ -f "$gcode_file" ]]; then
                cp "$gcode_file" "$workflow_dir/$(basename "$gcode_file")"
            fi
            
            log::success "Step added to workflow '$workflow'"
            ;;
        show)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: workflow show <name>"
                return 1
            fi
            
            local workflow_file="$workflow_dir/${name}.workflow"
            if [[ ! -f "$workflow_file" ]]; then
                log::error "Workflow not found: $name"
                return 1
            fi
            
            cat "$workflow_file"
            ;;
        execute)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: workflow execute <name>"
                return 1
            fi
            
            local workflow_file="$workflow_dir/${name}.workflow"
            if [[ ! -f "$workflow_file" ]]; then
                log::error "Workflow not found: $name"
                return 1
            fi
            
            log::info "Executing workflow: $name"
            
            # Extract and queue all steps
            local step_count=0
            while IFS= read -r line; do
                if [[ "$line" =~ ^[[:space:]]*-[[:space:]]file:[[:space:]](.+)$ ]]; then
                    local file="${BASH_REMATCH[1]}"
                    local src_file="$workflow_dir/$file"
                    if [[ -f "$src_file" ]]; then
                        ((step_count++))
                        cp "$src_file" "${CNCJS_WATCH_DIR}/workflow_${name}_step${step_count}_$(date +%s).gcode"
                        log::info "  Queued step $step_count: $file"
                    else
                        log::warning "  Step file not found: $file"
                    fi
                fi
            done < "$workflow_file"
            
            log::success "Workflow '$name' queued ($step_count steps)"
            log::warning "Note: Actual execution requires CNC controller connection"
            ;;
        remove)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: workflow remove <name>"
                return 1
            fi
            
            local workflow_file="$workflow_dir/${name}.workflow"
            if [[ ! -f "$workflow_file" ]]; then
                log::error "Workflow not found: $name"
                return 1
            fi
            
            rm "$workflow_file"
            log::success "Workflow '$name' removed"
            ;;
        export)
            local name="${1:-}"
            local output="${2:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: workflow export <name> [output.tar.gz]"
                return 1
            fi
            
            local workflow_file="$workflow_dir/${name}.workflow"
            if [[ ! -f "$workflow_file" ]]; then
                log::error "Workflow not found: $name"
                return 1
            fi
            
            output="${output:-${name}_workflow.tar.gz}"
            
            # Create temporary directory for export
            local temp_dir=$(mktemp -d)
            cp "$workflow_file" "$temp_dir/"
            
            # Copy all referenced files
            while IFS= read -r line; do
                if [[ "$line" =~ ^[[:space:]]*-[[:space:]]file:[[:space:]](.+)$ ]]; then
                    local file="${BASH_REMATCH[1]}"
                    if [[ -f "$workflow_dir/$file" ]]; then
                        cp "$workflow_dir/$file" "$temp_dir/"
                    fi
                fi
            done < "$workflow_file"
            
            # Create archive
            tar -czf "$output" -C "$temp_dir" .
            rm -rf "$temp_dir"
            
            log::success "Workflow exported to $output"
            ;;
        import)
            local archive="${1:-}"
            if [[ -z "$archive" ]] || [[ ! -f "$archive" ]]; then
                log::error "Usage: workflow import <archive.tar.gz>"
                return 1
            fi
            
            # Extract to workflow directory
            tar -xzf "$archive" -C "$workflow_dir"
            log::success "Workflow imported from $archive"
            ;;
        *)
            log::error "Unknown workflow action: $action"
            echo "Valid actions: list, create, add-step, show, execute, remove, export, import"
            return 1
            ;;
    esac
}

#######################################
# Controller configuration management
#######################################
cncjs::controller() {
    local action="${1:-list}"
    shift || true
    
    local controllers_dir="${CNCJS_DATA_DIR}/controllers"
    mkdir -p "$controllers_dir"
    
    case "$action" in
        list)
            log::info "Supported controllers:"
            echo "  - grbl (0.9, 1.1) - Default 3-axis CNC controller"
            echo "  - marlin (1.x, 2.x) - 3D printer firmware with CNC support"
            echo "  - smoothieware - Modern 32-bit controller"
            echo "  - tinyg - High-performance 6-axis controller"
            echo "  - g2core - ARM-based evolution of TinyG"
            echo ""
            log::info "Configured controller profiles:"
            if [[ -d "$controllers_dir" ]]; then
                for profile in "$controllers_dir"/*.profile; do
                    if [[ -f "$profile" ]]; then
                        local name=$(basename "$profile" .profile)
                        local type=$(grep "^type:" "$profile" 2>/dev/null | cut -d: -f2 | xargs)
                        local port=$(grep "^port:" "$profile" 2>/dev/null | cut -d: -f2 | xargs)
                        echo "  - $name: $type on $port"
                    fi
                done 2>/dev/null
                if ! ls "$controllers_dir"/*.profile &>/dev/null; then
                    echo "  No custom profiles configured"
                fi
            else
                echo "  No custom profiles configured"
            fi
            ;;
        configure)
            local name="${1:-}"
            local controller_type="${2:-}"
            local serial_port="${3:-}"
            local baud_rate="${4:-115200}"
            
            if [[ -z "$name" ]] || [[ -z "$controller_type" ]]; then
                log::error "Usage: controller configure <name> <type> [serial_port] [baud_rate]"
                echo "Types: grbl, marlin, smoothieware, tinyg, g2core"
                return 1
            fi
            
            # Validate controller type
            case "$controller_type" in
                grbl|marlin|smoothieware|tinyg|g2core)
                    ;;
                *)
                    log::error "Invalid controller type: $controller_type"
                    echo "Valid types: grbl, marlin, smoothieware, tinyg, g2core"
                    return 1
                    ;;
            esac
            
            # Set defaults based on controller type
            local default_settings=""
            case "$controller_type" in
                grbl)
                    default_settings="work_offsets: G54-G59
acceleration: 100
max_feed_rate: 2000
spindle_max: 10000"
                    ;;
                marlin)
                    default_settings="work_offsets: G54-G59
acceleration: 500
max_feed_rate: 3000
extruder_enabled: false"
                    ;;
                smoothieware)
                    default_settings="work_offsets: G54-G59
acceleration: 1000
max_feed_rate: 5000
laser_mode: false"
                    ;;
                tinyg)
                    default_settings="work_offsets: G54-G59
acceleration: 2000
max_feed_rate: 10000
axes: 6"
                    ;;
                g2core)
                    default_settings="work_offsets: G54-G59
acceleration: 3000
max_feed_rate: 15000
axes: 9"
                    ;;
            esac
            
            # Create controller profile
            local profile_file="$controllers_dir/${name}.profile"
            cat > "$profile_file" << EOF
name: $name
type: $controller_type
port: ${serial_port:-/dev/ttyUSB0}
baud_rate: $baud_rate
created: $(date --iso-8601=seconds)
$default_settings
EOF
            
            log::success "Controller profile '$name' created"
            echo "To use this profile, update CNCJS_CONTROLLER in config/defaults.sh"
            ;;
        show)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: controller show <name>"
                return 1
            fi
            
            local profile_file="$controllers_dir/${name}.profile"
            if [[ ! -f "$profile_file" ]]; then
                log::error "Profile not found: $name"
                return 1
            fi
            
            cat "$profile_file"
            ;;
        apply)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: controller apply <name>"
                return 1
            fi
            
            local profile_file="$controllers_dir/${name}.profile"
            if [[ ! -f "$profile_file" ]]; then
                log::error "Profile not found: $name"
                return 1
            fi
            
            # Extract settings from profile
            local controller_type=$(grep "^type:" "$profile_file" | cut -d: -f2 | xargs)
            local serial_port=$(grep "^port:" "$profile_file" | cut -d: -f2 | xargs)
            local baud_rate=$(grep "^baud_rate:" "$profile_file" | cut -d: -f2 | xargs)
            
            # Update CNCjs configuration
            local temp_config=$(mktemp)
            jq --arg type "$controller_type" \
               --arg port "$serial_port" \
               --arg baud "$baud_rate" \
               '.controller = $type | 
                .ports[0].comName = $port | 
                .baudRate = ($baud | tonumber)' \
               "${CNCJS_CONFIG_FILE}" > "$temp_config"
            
            mv "$temp_config" "${CNCJS_CONFIG_FILE}"
            
            log::success "Applied controller profile '$name'"
            log::info "Restart CNCjs for changes to take effect"
            ;;
        remove)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log::error "Usage: controller remove <name>"
                return 1
            fi
            
            local profile_file="$controllers_dir/${name}.profile"
            if [[ ! -f "$profile_file" ]]; then
                log::error "Profile not found: $name"
                return 1
            fi
            
            rm "$profile_file"
            log::success "Controller profile '$name' removed"
            ;;
        test)
            log::info "Testing controller connectivity..."
            
            # Check serial ports
            log::info "Available serial ports:"
            ls -la /dev/tty{USB,ACM}* 2>/dev/null || echo "  No USB serial ports found"
            
            # Check current configuration
            echo ""
            log::info "Current configuration:"
            echo "  Controller: ${CNCJS_CONTROLLER}"
            echo "  Serial Port: ${CNCJS_SERIAL_PORT}"
            echo "  Baud Rate: ${CNCJS_BAUD_RATE}"
            
            # Test if port exists
            if [[ -e "${CNCJS_SERIAL_PORT}" ]]; then
                log::success "Serial port exists"
            else
                log::warning "Serial port not found: ${CNCJS_SERIAL_PORT}"
            fi
            ;;
        *)
            log::error "Unknown controller action: $action"
            echo "Valid actions: list, configure, show, apply, remove, test"
            return 1
            ;;
    esac
}

#######################################
# G-code Visualization Functions
# WebGL-based 3D path preview
#######################################
cncjs::visualization() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        preview)
            cncjs::visualization::preview "$@"
            ;;
        analyze)
            cncjs::visualization::analyze "$@"
            ;;
        render)
            cncjs::visualization::render "$@"
            ;;
        export)
            cncjs::visualization::export "$@"
            ;;
        server)
            cncjs::visualization::server "$@"
            ;;
        *)
            echo "Usage: visualization [preview|analyze|render|export|server]"
            echo ""
            echo "Actions:"
            echo "  preview <file>     Generate 3D preview of G-code file"
            echo "  analyze <file>     Analyze G-code paths and statistics"
            echo "  render <file>      Render static visualization image"
            echo "  export <file>      Export visualization as HTML"
            echo "  server             Start visualization server"
            return 1
            ;;
    esac
}

#######################################
# Generate 3D preview of G-code file
#######################################
cncjs::visualization::preview() {
    local gcode_file="${1:-}"
    
    if [[ -z "$gcode_file" ]]; then
        log::error "G-code file required"
        echo "Usage: visualization preview <file>"
        return 1
    fi
    
    # Check if file exists
    local file_path="${CNCJS_WATCH_DIR}/${gcode_file}"
    if [[ ! -f "$file_path" ]]; then
        log::error "G-code file not found: $gcode_file"
        return 1
    fi
    
    log::info "Generating 3D preview for ${gcode_file}..."
    
    # Create visualization HTML with embedded Three.js
    local viz_file="${CNCJS_DATA_DIR}/visualizations/${gcode_file%.gcode}.html"
    mkdir -p "${CNCJS_DATA_DIR}/visualizations"
    
    # Generate visualization HTML
    cat > "$viz_file" << 'VIZEOF'
<!DOCTYPE html>
<html>
<head>
    <title>G-code Visualization</title>
    <style>
        body { margin: 0; overflow: hidden; }
        #container { width: 100vw; height: 100vh; }
        #info {
            position: absolute;
            top: 10px;
            left: 10px;
            color: white;
            background: rgba(0,0,0,0.7);
            padding: 10px;
            border-radius: 5px;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div id="container"></div>
    <div id="info">
        <div>File: GCODE_FILE</div>
        <div id="stats"></div>
        <div>Controls: Mouse to rotate, Scroll to zoom</div>
    </div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/three@0.128.0/examples/js/controls/OrbitControls.js"></script>
    <script>
        // G-code parser and visualization
        const gcodeContent = `GCODE_CONTENT`;
        
        // Initialize Three.js scene
        const scene = new THREE.Scene();
        scene.background = new THREE.Color(0x1a1a1a);
        
        const camera = new THREE.PerspectiveCamera(
            75, window.innerWidth / window.innerHeight, 0.1, 10000
        );
        camera.position.set(100, 100, 200);
        
        const renderer = new THREE.WebGLRenderer({ antialias: true });
        renderer.setSize(window.innerWidth, window.innerHeight);
        document.getElementById('container').appendChild(renderer.domElement);
        
        // Add controls
        const controls = new THREE.OrbitControls(camera, renderer.domElement);
        controls.enableDamping = true;
        
        // Add lights
        const ambientLight = new THREE.AmbientLight(0xffffff, 0.6);
        scene.add(ambientLight);
        const directionalLight = new THREE.DirectionalLight(0xffffff, 0.4);
        directionalLight.position.set(50, 50, 50);
        scene.add(directionalLight);
        
        // Add grid
        const gridHelper = new THREE.GridHelper(200, 20, 0x444444, 0x222222);
        scene.add(gridHelper);
        
        // Parse G-code and create path
        const points = [];
        let currentX = 0, currentY = 0, currentZ = 0;
        let minX = 0, maxX = 0, minY = 0, maxY = 0, minZ = 0, maxZ = 0;
        let totalDistance = 0;
        let rapidMoves = 0, feedMoves = 0;
        
        const lines = gcodeContent.split('\n');
        lines.forEach(line => {
            line = line.trim();
            if (!line || line.startsWith(';') || line.startsWith('(')) return;
            
            const parts = line.split(' ');
            const command = parts[0];
            
            if (command === 'G0' || command === 'G00' || command === 'G1' || command === 'G01') {
                let newX = currentX, newY = currentY, newZ = currentZ;
                
                parts.forEach(part => {
                    if (part.startsWith('X')) newX = parseFloat(part.slice(1));
                    if (part.startsWith('Y')) newY = parseFloat(part.slice(1));
                    if (part.startsWith('Z')) newZ = parseFloat(part.slice(1));
                });
                
                // Track statistics
                const distance = Math.sqrt(
                    Math.pow(newX - currentX, 2) +
                    Math.pow(newY - currentY, 2) +
                    Math.pow(newZ - currentZ, 2)
                );
                totalDistance += distance;
                
                if (command === 'G0' || command === 'G00') {
                    rapidMoves++;
                } else {
                    feedMoves++;
                    points.push(new THREE.Vector3(newX, newZ, newY));
                }
                
                // Update bounds
                minX = Math.min(minX, newX);
                maxX = Math.max(maxX, newX);
                minY = Math.min(minY, newY);
                maxY = Math.max(maxY, newY);
                minZ = Math.min(minZ, newZ);
                maxZ = Math.max(maxZ, newZ);
                
                currentX = newX;
                currentY = newY;
                currentZ = newZ;
            }
        });
        
        // Create line geometry from points
        if (points.length > 0) {
            const geometry = new THREE.BufferGeometry().setFromPoints(points);
            const material = new THREE.LineBasicMaterial({ 
                color: 0x00ff00,
                linewidth: 2
            });
            const line = new THREE.Line(geometry, material);
            scene.add(line);
            
            // Center camera on object
            const centerX = (minX + maxX) / 2;
            const centerY = (minY + maxY) / 2;
            const centerZ = (minZ + maxZ) / 2;
            controls.target.set(centerX, centerZ, centerY);
        }
        
        // Display statistics
        document.getElementById('stats').innerHTML = `
            <div>Total Distance: ${totalDistance.toFixed(2)} mm</div>
            <div>Feed Moves: ${feedMoves}</div>
            <div>Rapid Moves: ${rapidMoves}</div>
            <div>Bounds: X[${minX.toFixed(1)}, ${maxX.toFixed(1)}] Y[${minY.toFixed(1)}, ${maxY.toFixed(1)}] Z[${minZ.toFixed(1)}, ${maxZ.toFixed(1)}]</div>
        `;
        
        // Animation loop
        function animate() {
            requestAnimationFrame(animate);
            controls.update();
            renderer.render(scene, camera);
        }
        animate();
        
        // Handle window resize
        window.addEventListener('resize', () => {
            camera.aspect = window.innerWidth / window.innerHeight;
            camera.updateProjectionMatrix();
            renderer.setSize(window.innerWidth, window.innerHeight);
        });
    </script>
</body>
</html>
VIZEOF
    
    # Read G-code content and inject into HTML
    local gcode_content=$(cat "$file_path" | sed 's/`/\\`/g' | sed 's/\$/\\$/g' | sed ':a;N;$!ba;s/\n/\\n/g')
    sed -i "s|GCODE_FILE|${gcode_file}|g" "$viz_file"
    # Use a different approach for content replacement to avoid sed issues
    local temp_file="${viz_file}.tmp"
    awk -v content="$gcode_content" '{gsub(/GCODE_CONTENT/, content); print}' "$viz_file" > "$temp_file"
    mv "$temp_file" "$viz_file"
    
    log::success "3D preview generated: ${viz_file}"
    echo "Preview saved to: ${viz_file}"
    echo "Open in browser: http://localhost:${CNCJS_PORT}/visualizations/${gcode_file%.gcode}.html"
    
    return 0
}

#######################################
# Analyze G-code paths and statistics
#######################################
cncjs::visualization::analyze() {
    local gcode_file="${1:-}"
    
    if [[ -z "$gcode_file" ]]; then
        log::error "G-code file required"
        return 1
    fi
    
    local file_path="${CNCJS_WATCH_DIR}/${gcode_file}"
    if [[ ! -f "$file_path" ]]; then
        log::error "G-code file not found: $gcode_file"
        return 1
    fi
    
    log::info "Analyzing G-code: ${gcode_file}"
    
    # Parse G-code for statistics  
    local total_lines=$(wc -l < "$file_path")
    local g0_moves=$(grep -c "^G0\|^G00" "$file_path" 2>/dev/null || echo "0")
    local g1_moves=$(grep -c "^G1\|^G01" "$file_path" 2>/dev/null || echo "0")
    local g2_moves=$(grep -c "^G2\|^G02" "$file_path" 2>/dev/null || echo "0")
    local g3_moves=$(grep -c "^G3\|^G03" "$file_path" 2>/dev/null || echo "0")
    local tool_changes=$(grep -c "^M6\|^T[0-9]" "$file_path" 2>/dev/null || echo "0")
    local spindle_commands=$(grep -c "^M3\|^M4\|^M5" "$file_path" 2>/dev/null || echo "0")
    
    # Extract bounds
    local x_values=$(grep -oE "X-?[0-9]+\.?[0-9]*" "$file_path" | cut -c2- | sort -n)
    local y_values=$(grep -oE "Y-?[0-9]+\.?[0-9]*" "$file_path" | cut -c2- | sort -n)
    local z_values=$(grep -oE "Z-?[0-9]+\.?[0-9]*" "$file_path" | cut -c2- | sort -n)
    
    local x_min=$(echo "$x_values" | head -1)
    local x_max=$(echo "$x_values" | tail -1)
    local y_min=$(echo "$y_values" | head -1)
    local y_max=$(echo "$y_values" | tail -1)
    local z_min=$(echo "$z_values" | head -1)
    local z_max=$(echo "$z_values" | tail -1)
    
    # Calculate work envelope (using awk for safer arithmetic)
    local x_range=$(awk -v max="${x_max:-0}" -v min="${x_min:-0}" 'BEGIN {print max - min}')
    local y_range=$(awk -v max="${y_max:-0}" -v min="${y_min:-0}" 'BEGIN {print max - min}')
    local z_range=$(awk -v max="${z_max:-0}" -v min="${z_min:-0}" 'BEGIN {print max - min}')
    
    # Output analysis
    cat << EOF
G-code Analysis: ${gcode_file}
========================================
File Statistics:
  Total Lines:      ${total_lines}
  Rapid Moves (G0): ${g0_moves}
  Feed Moves (G1):  ${g1_moves}
  Arc CW (G2):      ${g2_moves}
  Arc CCW (G3):     ${g3_moves}
  Tool Changes:     ${tool_changes}
  Spindle Commands: ${spindle_commands}

Work Envelope:
  X: [${x_min:-0}, ${x_max:-0}] (range: ${x_range} mm)
  Y: [${y_min:-0}, ${y_max:-0}] (range: ${y_range} mm)
  Z: [${z_min:-0}, ${z_max:-0}] (range: ${z_range} mm)

Estimated Statistics:
  Total Moves:      $(awk -v g0="$g0_moves" -v g1="$g1_moves" -v g2="$g2_moves" -v g3="$g3_moves" 'BEGIN {print g0 + g1 + g2 + g3}')
  Cut Percentage:   $(awk -v g0="$g0_moves" -v g1="$g1_moves" 'BEGIN {if(g0 + g1 > 0) printf "%.0f", g1 * 100 / (g0 + g1); else print "0"}')%
EOF
    
    return 0
}

#######################################
# Render static visualization image
#######################################
cncjs::visualization::render() {
    local gcode_file="${1:-}"
    local output_file="${2:-}"
    
    if [[ -z "$gcode_file" ]]; then
        log::error "G-code file required"
        return 1
    fi
    
    local file_path="${CNCJS_WATCH_DIR}/${gcode_file}"
    if [[ ! -f "$file_path" ]]; then
        log::error "G-code file not found: $gcode_file"
        return 1
    fi
    
    # Default output file
    if [[ -z "$output_file" ]]; then
        output_file="${CNCJS_DATA_DIR}/visualizations/${gcode_file%.gcode}.png"
    fi
    
    mkdir -p "$(dirname "$output_file")"
    
    log::info "Rendering visualization to ${output_file}..."
    
    # First generate HTML preview
    cncjs::visualization::preview "$gcode_file" &>/dev/null
    
    # Use browserless if available to capture screenshot
    if command -v resource-browserless &>/dev/null; then
        local html_file="${CNCJS_DATA_DIR}/visualizations/${gcode_file%.gcode}.html"
        resource-browserless screenshot "file://${html_file}" --output "$output_file" 2>/dev/null && {
            log::success "Visualization rendered to ${output_file}"
            return 0
        }
    fi
    
    # Fallback: Create a simple SVG representation
    log::info "Creating SVG visualization..."
    # If output file already ends with .svg, use it as is
    local svg_file="$output_file"
    if [[ "$output_file" != *.svg ]]; then
        svg_file="${output_file%.png}.svg"
    fi
    
    # Parse G-code and create SVG paths
    cat > "$svg_file" << 'SVGEOF'
<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="800" height="600" viewBox="0 0 800 600">
    <rect width="800" height="600" fill="#1a1a1a"/>
    <g transform="translate(400,300)">
SVGEOF
    
    # Add G-code paths as SVG path elements
    awk '
    BEGIN { x=0; y=0; path="M 0 0" }
    /^G0|^G00|^G1|^G01/ {
        for(i=1; i<=NF; i++) {
            if(substr($i,1,1)=="X") x=substr($i,2)
            if(substr($i,1,1)=="Y") y=substr($i,2)
        }
        path = path " L " x " " (-y)
    }
    END {
        print "<path d=\"" path "\" stroke=\"#00ff00\" stroke-width=\"1\" fill=\"none\"/>"
    }
    ' "$file_path" >> "$svg_file"
    
    echo '    </g>' >> "$svg_file"
    echo '</svg>' >> "$svg_file"
    
    log::success "SVG visualization saved to ${svg_file}"
    echo "Output: ${svg_file}"
    
    return 0
}

#######################################
# Export visualization as standalone HTML
#######################################
cncjs::visualization::export() {
    local gcode_file="${1:-}"
    local output_file="${2:-}"
    
    if [[ -z "$gcode_file" ]]; then
        log::error "G-code file required"
        return 1
    fi
    
    # Generate preview first
    cncjs::visualization::preview "$gcode_file" || return 1
    
    # Set output file
    if [[ -z "$output_file" ]]; then
        output_file="${CNCJS_DATA_DIR}/exports/${gcode_file%.gcode}_viz.html"
    fi
    
    mkdir -p "$(dirname "$output_file")"
    
    # Copy visualization to export location
    local viz_file="${CNCJS_DATA_DIR}/visualizations/${gcode_file%.gcode}.html"
    cp "$viz_file" "$output_file"
    
    log::success "Visualization exported to ${output_file}"
    echo "Exported to: ${output_file}"
    
    return 0
}

#######################################
# Start visualization server
#######################################
cncjs::visualization::server() {
    local port="${1:-8195}"  # Use port adjacent to CNCjs
    
    log::info "Starting visualization server on port ${port}..."
    
    # Create index page for visualizations
    local index_file="${CNCJS_DATA_DIR}/visualizations/index.html"
    cat > "$index_file" << 'INDEXEOF'
<!DOCTYPE html>
<html>
<head>
    <title>CNCjs G-code Visualizations</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 20px;
            background: #1a1a1a;
            color: #fff;
        }
        h1 { color: #00ff00; }
        .viz-list {
            list-style: none;
            padding: 0;
        }
        .viz-item {
            background: #2a2a2a;
            margin: 10px 0;
            padding: 15px;
            border-radius: 5px;
            border-left: 3px solid #00ff00;
        }
        .viz-item a {
            color: #00ff00;
            text-decoration: none;
            font-size: 18px;
        }
        .viz-item a:hover {
            text-decoration: underline;
        }
        .meta {
            color: #888;
            font-size: 14px;
            margin-top: 5px;
        }
    </style>
</head>
<body>
    <h1>CNCjs G-code Visualizations</h1>
    <ul class="viz-list">
INDEXEOF
    
    # List all visualization files
    for viz in "${CNCJS_DATA_DIR}"/visualizations/*.html; do
        if [[ -f "$viz" ]] && [[ "$(basename "$viz")" != "index.html" ]]; then
            local name=$(basename "$viz" .html)
            local modified=$(stat -c %y "$viz" 2>/dev/null | cut -d' ' -f1,2 | cut -d'.' -f1)
            cat >> "$index_file" << ITEMEOF
        <li class="viz-item">
            <a href="$(basename "$viz")">${name}</a>
            <div class="meta">Modified: ${modified}</div>
        </li>
ITEMEOF
        fi
    done
    
    cat >> "$index_file" << 'INDEXEOF'
    </ul>
    <p style="color: #666; margin-top: 40px;">
        Generate new visualizations using: <code>cncjs visualization preview &lt;gcode-file&gt;</code>
    </p>
</body>
</html>
INDEXEOF
    
    # Start simple HTTP server
    cd "${CNCJS_DATA_DIR}/visualizations"
    python3 -m http.server "$port" &>/dev/null &
    local server_pid=$!
    
    log::success "Visualization server started on port ${port}"
    echo "Access visualizations at: http://localhost:${port}/"
    echo "Server PID: ${server_pid}"
    echo "Stop with: kill ${server_pid}"
    
    return 0
}

#######################################
# Camera integration for real-time monitoring
#######################################
cncjs::camera() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        list)
            log::info "Available camera devices:"
            
            # Check for video devices
            if ls /dev/video* &>/dev/null; then
                for device in /dev/video*; do
                    if [[ -c "$device" ]]; then
                        local name=$(v4l2-ctl --device="$device" --info 2>/dev/null | grep "Card" | cut -d: -f2 | xargs || echo "Unknown")
                        echo "  $device: $name"
                    fi
                done
            else
                echo "  No camera devices found"
            fi
            
            # Check if ffmpeg is available
            echo ""
            if command -v ffmpeg &>/dev/null; then
                log::success "FFmpeg is available for video processing"
            else
                log::warning "FFmpeg not installed - required for camera streaming"
            fi
            ;;
            
        enable)
            local device="${1:-/dev/video0}"
            local output_dir="${CNCJS_DATA_DIR}/camera"
            
            # Create camera directory
            mkdir -p "$output_dir"
            
            # Test camera device
            if [[ ! -c "$device" ]]; then
                log::error "Camera device not found: $device"
                return 1
            fi
            
            log::info "Enabling camera monitoring on $device..."
            
            # Create camera configuration
            cat > "${output_dir}/camera.conf" << EOF
# CNCjs Camera Configuration
CAMERA_DEVICE=$device
CAMERA_RESOLUTION=1280x720
CAMERA_FPS=30
CAMERA_FORMAT=mjpeg
STREAM_PORT=$((CNCJS_PORT + 1))
SNAPSHOT_INTERVAL=5
OUTPUT_DIR=$output_dir
EOF
            
            log::success "Camera configuration created"
            log::info "Stream will be available at http://localhost:$((CNCJS_PORT + 1))/stream"
            ;;
            
        disable)
            local camera_dir="${CNCJS_DATA_DIR}/camera"
            
            # Stop any running camera processes
            pkill -f "ffmpeg.*cncjs.*camera" || true
            
            # Remove configuration
            rm -f "${camera_dir}/camera.conf"
            
            log::success "Camera monitoring disabled"
            ;;
            
        snapshot)
            local output="${1:-${CNCJS_DATA_DIR}/camera/snapshot.jpg}"
            local device="${2:-/dev/video0}"
            
            if [[ ! -c "$device" ]]; then
                log::error "Camera device not found: $device"
                return 1
            fi
            
            if ! command -v ffmpeg &>/dev/null; then
                log::error "FFmpeg is required for snapshots"
                return 1
            fi
            
            log::info "Capturing snapshot from $device..."
            
            # Create output directory
            mkdir -p "$(dirname "$output")"
            
            # Capture single frame
            if ffmpeg -f v4l2 -i "$device" -frames:v 1 "$output" -y &>/dev/null; then
                log::success "Snapshot saved to $output"
                echo "$output"
            else
                log::error "Failed to capture snapshot"
                return 1
            fi
            ;;
            
        stream)
            local action="${1:-start}"
            local device="${2:-/dev/video0}"
            local port=$((CNCJS_PORT + 1))
            
            case "$action" in
                start)
                    if [[ ! -c "$device" ]]; then
                        log::error "Camera device not found: $device"
                        return 1
                    fi
                    
                    if ! command -v ffmpeg &>/dev/null; then
                        log::error "FFmpeg is required for streaming"
                        return 1
                    fi
                    
                    log::info "Starting camera stream on port $port..."
                    
                    # Start MJPEG stream
                    nohup ffmpeg -f v4l2 -i "$device" \
                        -r 30 -s 1280x720 \
                        -f mjpeg -q:v 10 \
                        "http://localhost:${port}/feed.mjpg" \
                        &> "${CNCJS_DATA_DIR}/camera/stream.log" &
                    
                    local pid=$!
                    echo $pid > "${CNCJS_DATA_DIR}/camera/stream.pid"
                    
                    log::success "Camera stream started (PID: $pid)"
                    log::info "View stream at http://localhost:${port}/feed.mjpg"
                    ;;
                    
                stop)
                    local pid_file="${CNCJS_DATA_DIR}/camera/stream.pid"
                    
                    if [[ -f "$pid_file" ]]; then
                        local pid=$(cat "$pid_file")
                        if kill -0 "$pid" 2>/dev/null; then
                            kill "$pid"
                            rm "$pid_file"
                            log::success "Camera stream stopped"
                        else
                            rm "$pid_file"
                            log::warning "Stream process not running"
                        fi
                    else
                        log::warning "No stream process found"
                    fi
                    ;;
                    
                status)
                    local pid_file="${CNCJS_DATA_DIR}/camera/stream.pid"
                    
                    if [[ -f "$pid_file" ]]; then
                        local pid=$(cat "$pid_file")
                        if kill -0 "$pid" 2>/dev/null; then
                            log::success "Camera stream is running (PID: $pid)"
                            echo "Stream URL: http://localhost:${port}/feed.mjpg"
                        else
                            log::warning "Stream process not running (stale PID file)"
                        fi
                    else
                        log::info "Camera stream is not running"
                    fi
                    ;;
                    
                *)
                    log::error "Unknown stream action: $action"
                    echo "Available actions: start, stop, status"
                    return 1
                    ;;
            esac
            ;;
            
        timelapse)
            local action="${1:-start}"
            local interval="${2:-10}"
            local output_dir="${CNCJS_DATA_DIR}/camera/timelapse"
            
            case "$action" in
                start)
                    mkdir -p "$output_dir"
                    
                    log::info "Starting timelapse capture (interval: ${interval}s)..."
                    
                    # Create timelapse script
                    cat > "${output_dir}/timelapse.sh" << 'EOF'
#!/usr/bin/env bash
INTERVAL=$1
OUTPUT_DIR=$2
DEVICE=${3:-/dev/video0}
COUNTER=0

while true; do
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    FILENAME="${OUTPUT_DIR}/frame_${TIMESTAMP}.jpg"
    
    ffmpeg -f v4l2 -i "$DEVICE" -frames:v 1 "$FILENAME" -y &>/dev/null
    
    if [[ $? -eq 0 ]]; then
        ((COUNTER++))
        echo "Captured frame $COUNTER: $FILENAME"
    fi
    
    sleep "$INTERVAL"
done
EOF
                    chmod +x "${output_dir}/timelapse.sh"
                    
                    # Start timelapse capture
                    nohup "${output_dir}/timelapse.sh" "$interval" "$output_dir" \
                        &> "${output_dir}/timelapse.log" &
                    
                    local pid=$!
                    echo $pid > "${output_dir}/timelapse.pid"
                    
                    log::success "Timelapse started (PID: $pid)"
                    log::info "Frames saved to $output_dir"
                    ;;
                    
                stop)
                    local pid_file="${output_dir}/timelapse.pid"
                    
                    if [[ -f "$pid_file" ]]; then
                        local pid=$(cat "$pid_file")
                        if kill -0 "$pid" 2>/dev/null; then
                            kill "$pid"
                            rm "$pid_file"
                            log::success "Timelapse stopped"
                        else
                            rm "$pid_file"
                            log::warning "Timelapse process not running"
                        fi
                    else
                        log::warning "No timelapse process found"
                    fi
                    ;;
                    
                compile)
                    if [[ ! -d "$output_dir" ]]; then
                        log::error "No timelapse frames found"
                        return 1
                    fi
                    
                    local frame_count=$(ls -1 "${output_dir}"/frame_*.jpg 2>/dev/null | wc -l)
                    if [[ $frame_count -eq 0 ]]; then
                        log::error "No frames to compile"
                        return 1
                    fi
                    
                    log::info "Compiling $frame_count frames into video..."
                    
                    local output_video="${output_dir}/timelapse_$(date +%Y%m%d_%H%M%S).mp4"
                    
                    ffmpeg -framerate 30 -pattern_type glob -i "${output_dir}/frame_*.jpg" \
                        -c:v libx264 -pix_fmt yuv420p "$output_video" &>/dev/null
                    
                    if [[ $? -eq 0 ]]; then
                        log::success "Timelapse video created: $output_video"
                        echo "$output_video"
                    else
                        log::error "Failed to compile timelapse"
                        return 1
                    fi
                    ;;
                    
                *)
                    log::error "Unknown timelapse action: $action"
                    echo "Available actions: start, stop, compile"
                    return 1
                    ;;
            esac
            ;;
            
        *)
            log::error "Unknown camera subcommand: $subcommand"
            echo "Available subcommands: list, enable, disable, snapshot, stream, timelapse"
            return 1
            ;;
    esac
}

#######################################
# Custom widget management system
#######################################
cncjs::widget() {
    local subcommand="${1:-list}"
    shift || true
    
    local widgets_dir="${CNCJS_DATA_DIR}/widgets"
    mkdir -p "$widgets_dir"
    
    case "$subcommand" in
        list)
            log::info "Available custom widgets:"
            
            if ls "${widgets_dir}"/*.widget &>/dev/null; then
                for widget_file in "${widgets_dir}"/*.widget; do
                    local widget_name=$(basename "$widget_file" .widget)
                    local widget_type=$(grep "^type:" "$widget_file" | cut -d: -f2 | xargs)
                    local widget_desc=$(grep "^description:" "$widget_file" | cut -d: -f2- | xargs)
                    echo "  • $widget_name ($widget_type): $widget_desc"
                done
            else
                echo "  No custom widgets installed"
            fi
            
            echo ""
            log::info "Built-in widget types:"
            echo "  • gauge: Display numeric values"
            echo "  • button: Trigger commands"
            echo "  • chart: Display time series data"
            echo "  • terminal: Command output display"
            echo "  • status: State indicator"
            ;;
            
        create)
            local name="${1:-}"
            local type="${2:-gauge}"
            
            if [[ -z "$name" ]]; then
                log::error "Usage: widget create <name> [type]"
                echo "Available types: gauge, button, chart, terminal, status"
                return 1
            fi
            
            local widget_file="${widgets_dir}/${name}.widget"
            
            log::info "Creating widget '$name' of type '$type'..."
            
            case "$type" in
                gauge)
                    cat > "$widget_file" << EOF
# CNCjs Custom Widget Definition
name: $name
type: gauge
description: Custom gauge widget
created: $(date -Iseconds)
config:
  min: 0
  max: 100
  unit: "%"
  color_ranges:
    - range: [0, 30]
      color: green
    - range: [30, 70]
      color: yellow
    - range: [70, 100]
      color: red
  refresh_interval: 1000
  data_source: "api/widgets/data/$name"
EOF
                    ;;
                    
                button)
                    cat > "$widget_file" << EOF
# CNCjs Custom Widget Definition
name: $name
type: button
description: Custom button widget
created: $(date -Iseconds)
config:
  label: "Custom Action"
  icon: "play"
  color: "primary"
  action:
    type: "gcode"
    commands:
      - "G28 ; Home all axes"
  confirmation: true
  confirmation_message: "Execute custom action?"
EOF
                    ;;
                    
                chart)
                    cat > "$widget_file" << EOF
# CNCjs Custom Widget Definition
name: $name
type: chart
description: Custom chart widget
created: $(date -Iseconds)
config:
  chart_type: "line"
  x_axis:
    label: "Time"
    type: "datetime"
  y_axis:
    label: "Value"
    min: 0
    max: 100
  series:
    - name: "Temperature"
      color: "#FF6B6B"
    - name: "Speed"
      color: "#4ECDC4"
  refresh_interval: 5000
  max_points: 100
  data_source: "api/widgets/data/$name"
EOF
                    ;;
                    
                terminal)
                    cat > "$widget_file" << EOF
# CNCjs Custom Widget Definition
name: $name
type: terminal
description: Custom terminal widget
created: $(date -Iseconds)
config:
  height: 200
  max_lines: 100
  font_size: 12
  background_color: "#1e1e1e"
  text_color: "#00ff00"
  auto_scroll: true
  command_history: true
  data_source: "api/widgets/terminal/$name"
EOF
                    ;;
                    
                status)
                    cat > "$widget_file" << EOF
# CNCjs Custom Widget Definition
name: $name
type: status
description: Custom status widget
created: $(date -Iseconds)
config:
  indicators:
    - name: "Machine State"
      states:
        - value: "idle"
          color: "green"
          icon: "check"
        - value: "running"
          color: "blue"
          icon: "play"
        - value: "error"
          color: "red"
          icon: "alert"
    - name: "Connection"
      states:
        - value: "connected"
          color: "green"
        - value: "disconnected"
          color: "gray"
  refresh_interval: 2000
  data_source: "api/widgets/status/$name"
EOF
                    ;;
                    
                *)
                    log::error "Unknown widget type: $type"
                    echo "Available types: gauge, button, chart, terminal, status"
                    return 1
                    ;;
            esac
            
            log::success "Widget '$name' created successfully"
            echo "Widget file: $widget_file"
            ;;
            
        show)
            local name="${1:-}"
            
            if [[ -z "$name" ]]; then
                log::error "Usage: widget show <name>"
                return 1
            fi
            
            local widget_file="${widgets_dir}/${name}.widget"
            
            if [[ ! -f "$widget_file" ]]; then
                log::error "Widget not found: $name"
                return 1
            fi
            
            log::info "Widget: $name"
            cat "$widget_file"
            ;;
            
        install)
            local name="${1:-}"
            
            if [[ -z "$name" ]]; then
                log::error "Usage: widget install <name>"
                return 1
            fi
            
            local widget_file="${widgets_dir}/${name}.widget"
            
            if [[ ! -f "$widget_file" ]]; then
                log::error "Widget not found: $name"
                return 1
            fi
            
            # Create installation package
            local install_dir="${CNCJS_DATA_DIR}/installed-widgets"
            mkdir -p "$install_dir"
            
            cp "$widget_file" "$install_dir/"
            
            # Register widget with CNCjs
            local registry_file="${install_dir}/registry.json"
            if [[ ! -f "$registry_file" ]]; then
                echo '{"widgets": []}' > "$registry_file"
            fi
            
            # Add widget to registry
            local temp_registry=$(mktemp)
            jq --arg name "$name" \
               --arg file "${name}.widget" \
               '.widgets += [{"name": $name, "definition": $file, "installed": now}]' \
               "$registry_file" > "$temp_registry"
            
            mv "$temp_registry" "$registry_file"
            
            log::success "Widget '$name' installed successfully"
            log::info "Restart CNCjs to load the widget"
            ;;
            
        uninstall)
            local name="${1:-}"
            
            if [[ -z "$name" ]]; then
                log::error "Usage: widget uninstall <name>"
                return 1
            fi
            
            local install_dir="${CNCJS_DATA_DIR}/installed-widgets"
            local registry_file="${install_dir}/registry.json"
            
            if [[ ! -f "$registry_file" ]]; then
                log::error "No widgets installed"
                return 1
            fi
            
            # Remove from registry
            local temp_registry=$(mktemp)
            jq --arg name "$name" \
               '.widgets = [.widgets[] | select(.name != $name)]' \
               "$registry_file" > "$temp_registry"
            
            mv "$temp_registry" "$registry_file"
            
            # Remove files
            rm -f "${install_dir}/${name}.widget"
            
            log::success "Widget '$name' uninstalled"
            ;;
            
        export)
            local name="${1:-}"
            local output="${2:-${name}.tar.gz}"
            
            if [[ -z "$name" ]]; then
                log::error "Usage: widget export <name> [output-file]"
                return 1
            fi
            
            local widget_file="${widgets_dir}/${name}.widget"
            
            if [[ ! -f "$widget_file" ]]; then
                log::error "Widget not found: $name"
                return 1
            fi
            
            log::info "Exporting widget '$name'..."
            
            # Create temporary directory
            local temp_dir=$(mktemp -d)
            mkdir -p "${temp_dir}/${name}"
            
            cp "$widget_file" "${temp_dir}/${name}/"
            
            # Create metadata
            cat > "${temp_dir}/${name}/metadata.json" << EOF
{
    "name": "$name",
    "exported": "$(date -Iseconds)",
    "version": "1.0.0",
    "cncjs_version": "4.0.0"
}
EOF
            
            # Create archive
            tar -czf "$output" -C "$temp_dir" "$name"
            rm -rf "$temp_dir"
            
            log::success "Widget exported to $output"
            ;;
            
        import)
            local archive="${1:-}"
            
            if [[ -z "$archive" ]]; then
                log::error "Usage: widget import <archive-file>"
                return 1
            fi
            
            if [[ ! -f "$archive" ]]; then
                log::error "Archive not found: $archive"
                return 1
            fi
            
            log::info "Importing widget from $archive..."
            
            # Extract to temporary directory
            local temp_dir=$(mktemp -d)
            tar -xzf "$archive" -C "$temp_dir"
            
            # Find widget directory
            local widget_dir=$(find "$temp_dir" -maxdepth 1 -type d | tail -n 1)
            local widget_name=$(basename "$widget_dir")
            
            # Copy files
            cp "${widget_dir}"/*.widget "$widgets_dir/" 2>/dev/null || true
            
            rm -rf "$temp_dir"
            
            log::success "Widget '$widget_name' imported successfully"
            echo "Use 'widget install $widget_name' to activate it"
            ;;
            
        remove)
            local name="${1:-}"
            
            if [[ -z "$name" ]]; then
                log::error "Usage: widget remove <name>"
                return 1
            fi
            
            local widget_file="${widgets_dir}/${name}.widget"
            
            if [[ ! -f "$widget_file" ]]; then
                log::error "Widget not found: $name"
                return 1
            fi
            
            rm -f "$widget_file"
            
            log::success "Widget '$name' removed"
            ;;
            
        *)
            log::error "Unknown widget subcommand: $subcommand"
            echo "Available subcommands: list, create, show, install, uninstall, export, import, remove"
            return 1
            ;;
    esac
}

#######################################
# Job Queue Management System
# Automated job scheduling and execution
#######################################
cncjs::jobqueue() {
    local subcommand="${1:-list}"
    shift || true
    
    local queue_dir="${CNCJS_DATA_DIR}/queue"
    local queue_status_file="${queue_dir}/status.json"
    local queue_lock_file="${queue_dir}/.lock"
    
    # Initialize queue directory
    mkdir -p "${queue_dir}/pending"
    mkdir -p "${queue_dir}/running"
    mkdir -p "${queue_dir}/completed"
    mkdir -p "${queue_dir}/failed"
    
    # Initialize status file if not exists
    if [[ ! -f "$queue_status_file" ]]; then
        echo '{"active": false, "current_job": null, "stats": {"total": 0, "completed": 0, "failed": 0}}' > "$queue_status_file"
    fi
    
    case "$subcommand" in
        list)
            log::info "Job Queue Status:"
            
            # Show queue state
            local active=$(jq -r '.active' "$queue_status_file" 2>/dev/null || echo "false")
            local current=$(jq -r '.current_job // "none"' "$queue_status_file" 2>/dev/null || echo "none")
            echo "  Queue Active: $active"
            echo "  Current Job: $current"
            echo ""
            
            # Show jobs by status
            echo "  Pending Jobs:"
            local pending_count=0
            for job in "${queue_dir}/pending"/*.job; do
                if [[ -f "$job" ]]; then
                    local job_name=$(basename "$job" .job)
                    local priority=$(jq -r '.priority // 5' "$job" 2>/dev/null || echo "5")
                    local file=$(jq -r '.file' "$job" 2>/dev/null || echo "unknown")
                    echo "    [$priority] $job_name: $file"
                    ((pending_count++))
                fi
            done
            [[ $pending_count -eq 0 ]] && echo "    (none)"
            
            echo ""
            echo "  Running Jobs:"
            local running_count=0
            for job in "${queue_dir}/running"/*.job; do
                if [[ -f "$job" ]]; then
                    local job_name=$(basename "$job" .job)
                    local start_time=$(jq -r '.start_time' "$job" 2>/dev/null || echo "unknown")
                    echo "    $job_name (started: $start_time)"
                    ((running_count++))
                fi
            done
            [[ $running_count -eq 0 ]] && echo "    (none)"
            
            # Show statistics
            echo ""
            echo "  Statistics:"
            local total=$(jq -r '.stats.total' "$queue_status_file" 2>/dev/null || echo "0")
            local completed=$(jq -r '.stats.completed' "$queue_status_file" 2>/dev/null || echo "0")
            local failed=$(jq -r '.stats.failed' "$queue_status_file" 2>/dev/null || echo "0")
            echo "    Total Jobs: $total"
            echo "    Completed: $completed"
            echo "    Failed: $failed"
            ;;
            
        add)
            local file="${1:-}"
            local priority="${2:-5}"
            local name="${3:-}"
            
            if [[ -z "$file" ]]; then
                log::error "Usage: jobqueue add <file> [priority] [name]"
                echo "  Priority: 1-10 (1=highest, default=5)"
                return 1
            fi
            
            # Check if file exists in watch directory
            local file_path="${CNCJS_WATCH_DIR}/${file}"
            if [[ ! -f "$file_path" ]]; then
                log::error "G-code file not found: $file"
                echo "Available files:"
                ls -1 "${CNCJS_WATCH_DIR}/" 2>/dev/null | sed 's/^/  /'
                return 1
            fi
            
            # Generate job ID if no name provided
            if [[ -z "$name" ]]; then
                name="job_$(date +%Y%m%d_%H%M%S)_${file%.gcode}"
            fi
            
            # Create job file
            local job_file="${queue_dir}/pending/${name}.job"
            cat > "$job_file" << EOF
{
  "id": "$name",
  "file": "$file",
  "priority": $priority,
  "created": "$(date -Iseconds)",
  "status": "pending",
  "estimated_time": null,
  "metadata": {
    "file_size": $(stat -c%s "$file_path" 2>/dev/null || echo 0),
    "line_count": $(wc -l < "$file_path" 2>/dev/null || echo 0)
  }
}
EOF
            
            # Update statistics
            local total=$(jq -r '.stats.total' "$queue_status_file")
            jq ".stats.total = $((total + 1))" "$queue_status_file" > "${queue_status_file}.tmp" && \
                mv "${queue_status_file}.tmp" "$queue_status_file"
            
            log::success "Job '$name' added to queue with priority $priority"
            ;;
            
        remove)
            local job_id="${1:-}"
            
            if [[ -z "$job_id" ]]; then
                log::error "Usage: jobqueue remove <job_id>"
                return 1
            fi
            
            # Find and remove job from any state
            local removed=false
            for state in pending running completed failed; do
                local job_file="${queue_dir}/${state}/${job_id}.job"
                if [[ -f "$job_file" ]]; then
                    rm "$job_file"
                    log::success "Removed job '$job_id' from $state queue"
                    removed=true
                    break
                fi
            done
            
            if [[ "$removed" == "false" ]]; then
                log::error "Job '$job_id' not found in queue"
                return 1
            fi
            ;;
            
        start)
            log::info "Starting job queue processor..."
            
            # Check if already running
            if [[ -f "$queue_lock_file" ]]; then
                local pid=$(cat "$queue_lock_file")
                if kill -0 "$pid" 2>/dev/null; then
                    log::warning "Queue processor already running (PID: $pid)"
                    return 0
                fi
                rm "$queue_lock_file"
            fi
            
            # Start queue processor in background
            (
                echo $$ > "$queue_lock_file"
                jq '.active = true' "$queue_status_file" > "${queue_status_file}.tmp" && \
                    mv "${queue_status_file}.tmp" "$queue_status_file"
                
                # Redirect output to avoid subshell issues
                exec &>/dev/null
                
                while [[ -f "$queue_lock_file" ]]; do
                    # Find next job by priority
                    local next_job=""
                    local highest_priority=11
                    
                    for job in "${queue_dir}/pending"/*.job; do
                        if [[ -f "$job" ]]; then
                            local priority=$(jq -r '.priority // 5' "$job")
                            if [[ $priority -lt $highest_priority ]]; then
                                highest_priority=$priority
                                next_job="$job"
                            fi
                        fi
                    done
                    
                    if [[ -n "$next_job" ]]; then
                        local job_name=$(basename "$next_job" .job)
                        local file=$(jq -r '.file' "$next_job")
                        
                        log::info "Processing job: $job_name (file: $file)"
                        
                        # Move to running state
                        mv "$next_job" "${queue_dir}/running/"
                        jq ".current_job = \"$job_name\" | .start_time = \"$(date -Iseconds)\"" \
                            "${queue_dir}/running/${job_name}.job" > "${queue_dir}/running/${job_name}.job.tmp" && \
                            mv "${queue_dir}/running/${job_name}.job.tmp" "${queue_dir}/running/${job_name}.job"
                        
                        jq ".current_job = \"$job_name\"" "$queue_status_file" > "${queue_status_file}.tmp" && \
                            mv "${queue_status_file}.tmp" "$queue_status_file"
                        
                        # Simulate job execution (in real scenario, would send to CNC)
                        log::warning "Simulating execution of $file (no controller connected)"
                        sleep 5  # Simulate processing time
                        
                        # Mark as completed
                        jq ".status = \"completed\" | .end_time = \"$(date -Iseconds)\"" \
                            "${queue_dir}/running/${job_name}.job" > "${queue_dir}/completed/${job_name}.job"
                        rm "${queue_dir}/running/${job_name}.job"
                        
                        # Update statistics
                        local completed=$(jq -r '.stats.completed // 0' "$queue_status_file")
                        local total=$(jq -r '.stats.total // 0' "$queue_status_file")
                        jq ".stats.completed = $((completed + 1)) | .stats.total = $((total + 1)) | .current_job = null" \
                            "$queue_status_file" > "${queue_status_file}.tmp" && \
                            mv "${queue_status_file}.tmp" "$queue_status_file"
                    else
                        # No jobs to process, wait
                        sleep 2
                    fi
                done
                
                jq '.active = false | .current_job = null' "$queue_status_file" > "${queue_status_file}.tmp" && \
                    mv "${queue_status_file}.tmp" "$queue_status_file"
                log::info "Queue processor stopped"
            ) &
            
            log::success "Queue processor started in background"
            ;;
            
        stop)
            log::info "Stopping job queue processor..."
            
            if [[ -f "$queue_lock_file" ]]; then
                local pid=$(cat "$queue_lock_file")
                if kill -0 "$pid" 2>/dev/null; then
                    # Send termination signal
                    kill "$pid" 2>/dev/null || true
                    
                    # Wait briefly for graceful shutdown
                    local wait_count=0
                    while [[ $wait_count -lt 3 ]] && kill -0 "$pid" 2>/dev/null; do
                        sleep 0.5
                        ((wait_count++))
                    done
                    
                    # Force kill if still running
                    if kill -0 "$pid" 2>/dev/null; then
                        kill -9 "$pid" 2>/dev/null || true
                    fi
                    
                    rm -f "$queue_lock_file"
                    
                    # Update status
                    jq '.active = false' "$queue_status_file" > "${queue_status_file}.tmp" && \
                        mv "${queue_status_file}.tmp" "$queue_status_file"
                    
                    log::success "Queue processor stopped"
                else
                    rm -f "$queue_lock_file"
                    log::warning "Queue processor was not running"
                fi
            else
                log::warning "Queue processor is not running"
            fi
            ;;
            
        clear)
            local state="${1:-pending}"
            
            if [[ "$state" == "all" ]]; then
                log::info "Clearing all queues..."
                rm -f "${queue_dir}/pending"/*.job 2>/dev/null
                rm -f "${queue_dir}/completed"/*.job 2>/dev/null
                rm -f "${queue_dir}/failed"/*.job 2>/dev/null
                
                # Reset statistics
                jq '.stats = {"total": 0, "completed": 0, "failed": 0}' \
                    "$queue_status_file" > "${queue_status_file}.tmp" && \
                    mv "${queue_status_file}.tmp" "$queue_status_file"
                
                log::success "All queues cleared"
            else
                case "$state" in
                    pending|completed|failed)
                        rm -f "${queue_dir}/${state}"/*.job 2>/dev/null
                        log::success "Cleared $state queue"
                        ;;
                    *)
                        log::error "Invalid queue state: $state"
                        echo "Valid states: pending, completed, failed, all"
                        return 1
                        ;;
                esac
            fi
            ;;
            
        status)
            local job_id="${1:-}"
            
            if [[ -z "$job_id" ]]; then
                # Show overall status
                cncjs::jobqueue list
            else
                # Show specific job status
                local found=false
                for state in pending running completed failed; do
                    local job_file="${queue_dir}/${state}/${job_id}.job"
                    if [[ -f "$job_file" ]]; then
                        log::info "Job: $job_id"
                        echo "  State: $state"
                        jq '.' "$job_file" 2>/dev/null | sed 's/^/  /'
                        found=true
                        break
                    fi
                done
                
                if [[ "$found" == "false" ]]; then
                    log::error "Job '$job_id' not found"
                    return 1
                fi
            fi
            ;;
            
        *)
            log::error "Unknown jobqueue subcommand: $subcommand"
            echo "Available subcommands: list, add, remove, start, stop, clear, status"
            return 1
            ;;
    esac
}

#######################################
# Real-time position tracking
#######################################
cncjs::position() {
    local subcommand="${1:-current}"
    shift || true
    
    local position_file="${CNCJS_DATA_DIR}/position.json"
    local tracking_file="${CNCJS_DATA_DIR}/tracking.pid"
    
    # Initialize position file if not exists
    if [[ ! -f "$position_file" ]]; then
        cat > "$position_file" << EOF
{
    "x": 0.0,
    "y": 0.0,
    "z": 0.0,
    "feedrate": 0,
    "spindle": 0,
    "status": "idle",
    "work_offset": {
        "x": 0.0,
        "y": 0.0,
        "z": 0.0
    },
    "machine_position": {
        "x": 0.0,
        "y": 0.0,
        "z": 0.0
    },
    "controller_state": "disconnected",
    "last_update": "$(date -Iseconds)"
}
EOF
    fi
    
    case "$subcommand" in
        current|get)
            # Return current position
            if [[ "${1:-}" == "--json" ]]; then
                cat "$position_file"
            else
                local x=$(jq -r '.x' "$position_file")
                local y=$(jq -r '.y' "$position_file")
                local z=$(jq -r '.z' "$position_file")
                local status=$(jq -r '.status' "$position_file")
                local controller=$(jq -r '.controller_state' "$position_file")
                
                echo "Machine Position:"
                echo "  X: ${x}mm"
                echo "  Y: ${y}mm"
                echo "  Z: ${z}mm"
                echo "  Status: $status"
                echo "  Controller: $controller"
            fi
            ;;
            
        track|start)
            # Start position tracking
            if [[ -f "$tracking_file" ]]; then
                local pid=$(cat "$tracking_file")
                if kill -0 "$pid" 2>/dev/null; then
                    log::warning "Position tracking already running (PID: $pid)"
                    return 0
                fi
                rm "$tracking_file"
            fi
            
            log::info "Starting position tracking..."
            
            # Start tracking in background (with error handling disabled for subshell)
            (
                set +e  # Disable error exit in subshell
                echo $$ > "$tracking_file"
                local history_file="${CNCJS_DATA_DIR}/position_history.jsonl"
                local history_counter=0
                
                while [[ -f "$tracking_file" ]]; do
                    # In a real implementation, this would query the controller
                    # For now, simulate movement if a job is running
                    local current_job=$(jq -r '.current_job // "none"' "${CNCJS_DATA_DIR}/queue/status.json" 2>/dev/null || echo "none")
                    
                    if [[ "$current_job" != "none" && "$current_job" != "null" && -n "$current_job" ]]; then
                        # Simulate movement during job
                        local x=$(jq -r '.x' "$position_file" 2>/dev/null || echo "0")
                        local y=$(jq -r '.y' "$position_file" 2>/dev/null || echo "0")
                        local z=$(jq -r '.z' "$position_file" 2>/dev/null || echo "0")
                        
                        # Small simulated movements
                        x=$(awk "BEGIN {print $x + 0.1}")
                        y=$(awk "BEGIN {print $y + 0.05}")
                        
                        jq ".x = $x | .y = $y | .z = $z | .status = \"running\" | .last_update = \"$(date -Iseconds)\"" \
                            "$position_file" > "${position_file}.tmp" 2>/dev/null && \
                            mv "${position_file}.tmp" "$position_file" 2>/dev/null
                    else
                        # Update idle status
                        jq ".status = \"idle\" | .last_update = \"$(date -Iseconds)\"" \
                            "$position_file" > "${position_file}.tmp" 2>/dev/null && \
                            mv "${position_file}.tmp" "$position_file" 2>/dev/null
                    fi
                    
                    # Record to history every 10 updates (1 second at 10Hz)
                    ((history_counter++))
                    if [[ $((history_counter % 10)) -eq 0 ]]; then
                        local current_data=$(cat "$position_file" 2>/dev/null)
                        if [[ -n "$current_data" ]]; then
                            echo "$current_data" | jq -c '.timestamp = .last_update' >> "$history_file" 2>/dev/null
                            
                            # Keep history file size manageable (max 10000 entries)
                            local line_count=$(wc -l < "$history_file" 2>/dev/null || echo "0")
                            if [[ $line_count -gt 10000 ]]; then
                                tail -n 9000 "$history_file" > "${history_file}.tmp" 2>/dev/null && \
                                    mv "${history_file}.tmp" "$history_file" 2>/dev/null
                            fi
                        fi
                    fi
                    
                    sleep 0.1  # 10Hz update rate
                done
            ) > /dev/null 2>&1 &
            
            log::success "Position tracking started"
            ;;
            
        stop)
            # Stop position tracking
            if [[ -f "$tracking_file" ]]; then
                local pid=$(cat "$tracking_file")
                if kill -0 "$pid" 2>/dev/null; then
                    kill "$pid"
                    rm "$tracking_file"
                    log::success "Position tracking stopped"
                else
                    rm "$tracking_file"
                    log::warning "Position tracking was not running"
                fi
            else
                log::warning "Position tracking is not active"
            fi
            ;;
            
        zero|home)
            # Zero/home position
            log::info "Setting position to zero..."
            jq '.x = 0 | .y = 0 | .z = 0 | .work_offset.x = 0 | .work_offset.y = 0 | .work_offset.z = 0' \
                "$position_file" > "${position_file}.tmp" && \
                mv "${position_file}.tmp" "$position_file"
            log::success "Position zeroed"
            ;;
            
        set)
            # Set position manually
            local axis="${2:-}"
            local value="${3:-0}"
            
            if [[ -z "$axis" ]]; then
                log::error "Usage: position set <axis> <value>"
                return 1
            fi
            
            case "$axis" in
                x|X) axis="x" ;;
                y|Y) axis="y" ;;
                z|Z) axis="z" ;;
                *)
                    log::error "Invalid axis: $axis (use x, y, or z)"
                    return 1
                    ;;
            esac
            
            jq ".${axis} = $value | .last_update = \"$(date -Iseconds)\"" \
                "$position_file" > "${position_file}.tmp" && \
                mv "${position_file}.tmp" "$position_file"
            
            log::success "Position $axis set to $value"
            ;;
            
        history)
            # Show position history
            local history_file="${CNCJS_DATA_DIR}/position_history.jsonl"
            local limit="${2:-100}"
            local format="${3:-text}"
            
            if [[ ! -f "$history_file" ]]; then
                log::warning "No position history available"
                return 0
            fi
            
            if [[ "$format" == "--json" || "$format" == "json" ]]; then
                # Output last N entries as JSON array
                tail -n "$limit" "$history_file" | jq -s '.'
            else
                echo "Position History (last $limit entries):"
                echo "========================================="
                tail -n "$limit" "$history_file" | while IFS= read -r line; do
                    local timestamp=$(echo "$line" | jq -r '.timestamp')
                    local x=$(echo "$line" | jq -r '.x')
                    local y=$(echo "$line" | jq -r '.y')
                    local z=$(echo "$line" | jq -r '.z')
                    local status=$(echo "$line" | jq -r '.status')
                    printf "[%s] X:%.3f Y:%.3f Z:%.3f Status:%s\n" "$timestamp" "$x" "$y" "$z" "$status"
                done
            fi
            ;;
            
        clear-history)
            # Clear position history
            local history_file="${CNCJS_DATA_DIR}/position_history.jsonl"
            if [[ -f "$history_file" ]]; then
                > "$history_file"
                log::success "Position history cleared"
            else
                log::warning "No position history to clear"
            fi
            ;;
            
        export-history)
            # Export position history as CSV or JSON
            local history_file="${CNCJS_DATA_DIR}/position_history.jsonl"
            local output="${2:-position_history.csv}"
            local format="${3:-csv}"
            
            if [[ ! -f "$history_file" ]]; then
                log::error "No position history to export"
                return 1
            fi
            
            if [[ "$format" == "json" ]]; then
                cat "$history_file" | jq -s '.' > "$output"
                log::success "Position history exported to $output (JSON)"
            else
                # Export as CSV
                echo "timestamp,x,y,z,status,feedrate,spindle" > "$output"
                while IFS= read -r line; do
                    echo "$line" | jq -r '[.timestamp, .x, .y, .z, .status, .feedrate // 0, .spindle // 0] | @csv' >> "$output"
                done < "$history_file"
                log::success "Position history exported to $output (CSV)"
            fi
            ;;
            
        *)
            log::error "Unknown position subcommand: $subcommand"
            echo "Available subcommands: current, track, stop, zero, set, history, clear-history, export-history"
            return 1
            ;;
    esac
}

#######################################
# Emergency stop functionality
#######################################
cncjs::emergency_stop() {
    log::error "EMERGENCY STOP ACTIVATED!"
    
    local queue_lock="${CNCJS_DATA_DIR}/queue/processor.lock"
    local position_tracking="${CNCJS_DATA_DIR}/tracking.pid"
    local position_file="${CNCJS_DATA_DIR}/position.json"
    local queue_status="${CNCJS_DATA_DIR}/queue/status.json"
    
    # Stop queue processor immediately
    if [[ -f "$queue_lock" ]]; then
        local pid=$(cat "$queue_lock")
        if kill -0 "$pid" 2>/dev/null; then
            kill -9 "$pid"
            log::warning "Queue processor killed (PID: $pid)"
        fi
        rm "$queue_lock"
    fi
    
    # Stop position tracking
    if [[ -f "$position_tracking" ]]; then
        local pid=$(cat "$position_tracking")
        if kill -0 "$pid" 2>/dev/null; then
            kill -9 "$pid"
            log::warning "Position tracking killed (PID: $pid)"
        fi
        rm "$position_tracking"
    fi
    
    # Update status files
    if [[ -f "$position_file" ]]; then
        jq '.status = "emergency_stop" | .controller_state = "halted"' \
            "$position_file" > "${position_file}.tmp" && \
            mv "${position_file}.tmp" "$position_file"
    fi
    
    if [[ -f "$queue_status" ]]; then
        jq '.active = false | .current_job = null | .emergency_stop = true' \
            "$queue_status" > "${queue_status}.tmp" && \
            mv "${queue_status}.tmp" "$queue_status"
    fi
    
    # Move any running jobs to failed state
    local running_dir="${CNCJS_DATA_DIR}/queue/running"
    local failed_dir="${CNCJS_DATA_DIR}/queue/failed"
    
    if [[ -d "$running_dir" ]]; then
        for job in "$running_dir"/*.job; do
            if [[ -f "$job" ]]; then
                local job_name=$(basename "$job")
                jq '.status = "failed" | .error = "Emergency stop" | .end_time = "'"$(date -Iseconds)"'"' \
                    "$job" > "$failed_dir/$job_name"
                rm "$job"
                log::warning "Job moved to failed: $job_name"
            fi
        done
    fi
    
    # In a real implementation, would send immediate halt command to controller
    # For simulation, just log the action
    log::warning "Controller halt command sent (simulated)"
    
    log::error "Emergency stop complete. System halted."
    echo ""
    echo "To resume operations:"
    echo "  1. Clear the emergency condition"
    echo "  2. Run: vrooli resource cncjs reset"
    echo "  3. Re-home the machine if needed"
    echo "  4. Restart queue processing if desired"
    
    return 0
}

#######################################
# Safety zones management
#######################################
cncjs::safety_zones() {
    local subcommand="${1:-list}"
    shift || true
    
    local zones_file="${CNCJS_DATA_DIR}/safety_zones.json"
    
    # Initialize zones file if not exists
    if [[ ! -f "$zones_file" ]]; then
        cat > "$zones_file" << 'EOF'
{
    "soft_limits": {
        "x_min": -500,
        "x_max": 500,
        "y_min": -500,
        "y_max": 500,
        "z_min": -100,
        "z_max": 100,
        "enabled": true
    },
    "no_go_zones": [],
    "safe_retract_height": 10,
    "boundary_buffer": 5
}
EOF
    fi
    
    case "$subcommand" in
        list|show)
            # Show current safety zones
            if [[ "${1:-}" == "--json" ]]; then
                cat "$zones_file"
            else
                echo "Safety Zones Configuration:"
                echo "==========================="
                echo ""
                echo "Soft Limits:"
                local x_min=$(jq -r '.soft_limits.x_min' "$zones_file")
                local x_max=$(jq -r '.soft_limits.x_max' "$zones_file")
                local y_min=$(jq -r '.soft_limits.y_min' "$zones_file")
                local y_max=$(jq -r '.soft_limits.y_max' "$zones_file")
                local z_min=$(jq -r '.soft_limits.z_min' "$zones_file")
                local z_max=$(jq -r '.soft_limits.z_max' "$zones_file")
                local enabled=$(jq -r '.soft_limits.enabled' "$zones_file")
                
                echo "  X: $x_min to $x_max mm"
                echo "  Y: $y_min to $y_max mm"
                echo "  Z: $z_min to $z_max mm"
                echo "  Enabled: $enabled"
                echo ""
                
                echo "No-Go Zones:"
                local zone_count=$(jq '.no_go_zones | length' "$zones_file")
                if [[ $zone_count -eq 0 ]]; then
                    echo "  None defined"
                else
                    jq -r '.no_go_zones[] | "  [\(.name)]: X:\(.x_min)-\(.x_max) Y:\(.y_min)-\(.y_max) Z:\(.z_min)-\(.z_max)"' "$zones_file"
                fi
                echo ""
                
                local retract=$(jq -r '.safe_retract_height' "$zones_file")
                local buffer=$(jq -r '.boundary_buffer' "$zones_file")
                echo "Safe Retract Height: ${retract}mm"
                echo "Boundary Buffer: ${buffer}mm"
            fi
            ;;
            
        set-limits)
            # Set soft limits
            local axis="${1:-}"
            local min="${2:-}"
            local max="${3:-}"
            
            if [[ -z "$axis" || -z "$min" || -z "$max" ]]; then
                log::error "Usage: safety-zones set-limits <axis> <min> <max>"
                return 1
            fi
            
            case "$axis" in
                x|X) axis="x" ;;
                y|Y) axis="y" ;;
                z|Z) axis="z" ;;
                *)
                    log::error "Invalid axis: $axis (use x, y, or z)"
                    return 1
                    ;;
            esac
            
            jq ".soft_limits.${axis}_min = $min | .soft_limits.${axis}_max = $max" \
                "$zones_file" > "${zones_file}.tmp" && \
                mv "${zones_file}.tmp" "$zones_file"
            
            log::success "Soft limits for $axis axis set to [$min, $max]"
            ;;
            
        enable-limits)
            # Enable soft limits
            jq '.soft_limits.enabled = true' "$zones_file" > "${zones_file}.tmp" && \
                mv "${zones_file}.tmp" "$zones_file"
            log::success "Soft limits enabled"
            ;;
            
        disable-limits)
            # Disable soft limits
            jq '.soft_limits.enabled = false' "$zones_file" > "${zones_file}.tmp" && \
                mv "${zones_file}.tmp" "$zones_file"
            log::warning "Soft limits disabled - machine can move outside safe boundaries"
            ;;
            
        add-zone)
            # Add a no-go zone
            local name="${1:-}"
            local x_min="${2:-}"
            local x_max="${3:-}"
            local y_min="${4:-}"
            local y_max="${5:-}"
            local z_min="${6:-}"
            local z_max="${7:-}"
            
            if [[ -z "$name" || -z "$x_min" || -z "$x_max" || -z "$y_min" || -z "$y_max" || -z "$z_min" || -z "$z_max" ]]; then
                log::error "Usage: safety-zones add-zone <name> <x_min> <x_max> <y_min> <y_max> <z_min> <z_max>"
                return 1
            fi
            
            local new_zone=$(jq -n \
                --arg name "$name" \
                --argjson x_min "$x_min" \
                --argjson x_max "$x_max" \
                --argjson y_min "$y_min" \
                --argjson y_max "$y_max" \
                --argjson z_min "$z_min" \
                --argjson z_max "$z_max" \
                '{name: $name, x_min: $x_min, x_max: $x_max, y_min: $y_min, y_max: $y_max, z_min: $z_min, z_max: $z_max, enabled: true}')
            
            jq ".no_go_zones += [$new_zone]" "$zones_file" > "${zones_file}.tmp" && \
                mv "${zones_file}.tmp" "$zones_file"
            
            log::success "No-go zone '$name' added"
            ;;
            
        remove-zone)
            # Remove a no-go zone
            local name="${1:-}"
            
            if [[ -z "$name" ]]; then
                log::error "Usage: safety-zones remove-zone <name>"
                return 1
            fi
            
            jq "del(.no_go_zones[] | select(.name == \"$name\"))" "$zones_file" > "${zones_file}.tmp" && \
                mv "${zones_file}.tmp" "$zones_file"
            
            log::success "No-go zone '$name' removed"
            ;;
            
        check)
            # Check if a position is safe
            local x="${1:-0}"
            local y="${2:-0}"
            local z="${3:-0}"
            
            local limits_enabled=$(jq -r '.soft_limits.enabled' "$zones_file")
            local is_safe=true
            local violations=()
            
            # Check soft limits
            if [[ "$limits_enabled" == "true" ]]; then
                local x_min=$(jq -r '.soft_limits.x_min' "$zones_file")
                local x_max=$(jq -r '.soft_limits.x_max' "$zones_file")
                local y_min=$(jq -r '.soft_limits.y_min' "$zones_file")
                local y_max=$(jq -r '.soft_limits.y_max' "$zones_file")
                local z_min=$(jq -r '.soft_limits.z_min' "$zones_file")
                local z_max=$(jq -r '.soft_limits.z_max' "$zones_file")
                
                if (( $(echo "$x < $x_min || $x > $x_max" | bc -l) )); then
                    violations+=("X position $x outside limits [$x_min, $x_max]")
                    is_safe=false
                fi
                if (( $(echo "$y < $y_min || $y > $y_max" | bc -l) )); then
                    violations+=("Y position $y outside limits [$y_min, $y_max]")
                    is_safe=false
                fi
                if (( $(echo "$z < $z_min || $z > $z_max" | bc -l) )); then
                    violations+=("Z position $z outside limits [$z_min, $z_max]")
                    is_safe=false
                fi
            fi
            
            # Check no-go zones
            while IFS= read -r zone; do
                local zone_name=$(echo "$zone" | jq -r '.name')
                local zone_enabled=$(echo "$zone" | jq -r '.enabled')
                
                if [[ "$zone_enabled" == "true" ]]; then
                    local zone_x_min=$(echo "$zone" | jq -r '.x_min')
                    local zone_x_max=$(echo "$zone" | jq -r '.x_max')
                    local zone_y_min=$(echo "$zone" | jq -r '.y_min')
                    local zone_y_max=$(echo "$zone" | jq -r '.y_max')
                    local zone_z_min=$(echo "$zone" | jq -r '.z_min')
                    local zone_z_max=$(echo "$zone" | jq -r '.z_max')
                    
                    if (( $(echo "$x >= $zone_x_min && $x <= $zone_x_max && $y >= $zone_y_min && $y <= $zone_y_max && $z >= $zone_z_min && $z <= $zone_z_max" | bc -l) )); then
                        violations+=("Position inside no-go zone '$zone_name'")
                        is_safe=false
                    fi
                fi
            done < <(jq -c '.no_go_zones[]' "$zones_file" 2>/dev/null)
            
            if [[ "$is_safe" == "true" ]]; then
                log::success "Position (X:$x, Y:$y, Z:$z) is SAFE"
            else
                log::error "Position (X:$x, Y:$y, Z:$z) is UNSAFE"
                for violation in "${violations[@]}"; do
                    echo "  - $violation"
                done
            fi
            
            return $([ "$is_safe" == "true" ] && echo 0 || echo 1)
            ;;
            
        *)
            log::error "Unknown safety-zones subcommand: $subcommand"
            echo "Available subcommands: list, set-limits, enable-limits, disable-limits, add-zone, remove-zone, check"
            return 1
            ;;
    esac
}

#######################################
# Reset after emergency stop
#######################################
cncjs::reset() {
    log::info "Resetting CNCjs after emergency stop..."
    
    local position_file="${CNCJS_DATA_DIR}/position.json"
    local queue_status="${CNCJS_DATA_DIR}/queue/status.json"
    
    # Clear emergency stop flag
    if [[ -f "$queue_status" ]]; then
        jq 'del(.emergency_stop)' "$queue_status" > "${queue_status}.tmp" && \
            mv "${queue_status}.tmp" "$queue_status"
    fi
    
    # Reset position status
    if [[ -f "$position_file" ]]; then
        jq '.status = "idle" | .controller_state = "ready"' \
            "$position_file" > "${position_file}.tmp" && \
            mv "${position_file}.tmp" "$position_file"
    fi
    
    log::success "System reset complete. Ready for operation."
}