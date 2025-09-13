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
            if timeout 2 curl -sf "http://localhost:${CNCJS_PORT}/" &>/dev/null; then
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