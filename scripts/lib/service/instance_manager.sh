#!/usr/bin/env bash
# Generic instance management for development environments
# Detects and manages running application instances across Docker and native deployments
set -euo pipefail

LIB_SERVICE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${LIB_SERVICE_DIR}/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/docker.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"

# Data structures to hold instance information
declare -gA DOCKER_INSTANCES=()
declare -gA NATIVE_INSTANCES=()
declare -g INSTANCE_STATE="none"  # none, docker, native, mixed

# Configuration globals (will be loaded from service.json)
declare -ga APP_SERVICES=()
declare -gA SERVICE_PORTS=()
declare -ga CONTAINER_PATTERNS=()
declare -ga PROCESS_PATTERNS=()
declare -ga DOCKER_COMPOSE_FILES=()
declare -ga DOCKER_COMPOSE_SERVICES=()
declare -g INSTANCE_MANAGEMENT_ENABLED="true"
declare -g INSTANCE_STRATEGY="single"
declare -g CONFLICT_ACTION="prompt"
declare -g CONFLICT_TIMEOUT="30"

# ==============================================================================
# Configuration Loading Functions
# ==============================================================================

# Load instance management configuration from service.json
instance::load_config() {
    # Set defaults first
    APP_SERVICES=("server" "ui" "database")
    SERVICE_PORTS=(
        ["database"]="5432"
        ["server"]="5329"
        ["ui"]="3000"
    )
    CONTAINER_PATTERNS=("app[-_]*" "*[-_]app" "*[-_]server" "*[-_]ui" "*[-_]db")
    PROCESS_PATTERNS=("node.*server" "npm.*dev" "pnpm.*dev" "vite.*" "concurrently.*")
    INSTANCE_MANAGEMENT_ENABLED="true"
    INSTANCE_STRATEGY="single"
    CONFLICT_ACTION="prompt"
    CONFLICT_TIMEOUT="30"
    
    # Try to load from service.json if it exists
    local service_json="${var_SERVICE_JSON_FILE:-}"
    if [[ -f "$service_json" ]] && command -v jq &> /dev/null; then
        log::debug "Loading instance management configuration from: $service_json"
        
        # Load configuration sections
        local config_enabled config_strategy config_action config_timeout
        local config_services config_ports config_containers config_processes
        
        config_enabled=$(jq -r '.instanceManagement.enabled // true' "$service_json" 2>/dev/null || echo "true")
        config_strategy=$(jq -r '.instanceManagement.strategy // "single"' "$service_json" 2>/dev/null || echo "single")
        config_action=$(jq -r '.instanceManagement.conflicts.action // "prompt"' "$service_json" 2>/dev/null || echo "prompt")
        config_timeout=$(jq -r '.instanceManagement.conflicts.timeout // 30' "$service_json" 2>/dev/null || echo "30")
        
        # Update globals
        INSTANCE_MANAGEMENT_ENABLED="$config_enabled"
        INSTANCE_STRATEGY="$config_strategy"
        CONFLICT_ACTION="$config_action"
        CONFLICT_TIMEOUT="$config_timeout"
        
        # Load services array
        config_services=$(jq -r '.instanceManagement.detection.services[]? // empty' "$service_json" 2>/dev/null || echo "")
        if [[ -n "$config_services" ]]; then
            APP_SERVICES=()
            while IFS= read -r service; do
                [[ -n "$service" ]] && APP_SERVICES+=("$service")
            done <<< "$config_services"
        fi
        
        # Load ports object
        local port_keys
        port_keys=$(jq -r '.instanceManagement.detection.ports | keys[]? // empty' "$service_json" 2>/dev/null || echo "")
        if [[ -n "$port_keys" ]]; then
            SERVICE_PORTS=()
            while IFS= read -r service; do
                if [[ -n "$service" ]]; then
                    local port
                    port=$(jq -r ".instanceManagement.detection.ports[\"$service\"]" "$service_json" 2>/dev/null || echo "")
                    [[ -n "$port" && "$port" != "null" ]] && SERVICE_PORTS["$service"]="$port"
                fi
            done <<< "$port_keys"
        fi
        
        # Load container patterns array
        config_containers=$(jq -r '.instanceManagement.detection.containers[]? // empty' "$service_json" 2>/dev/null || echo "")
        if [[ -n "$config_containers" ]]; then
            CONTAINER_PATTERNS=()
            while IFS= read -r pattern; do
                [[ -n "$pattern" ]] && CONTAINER_PATTERNS+=("$pattern")
            done <<< "$config_containers"
        fi
        
        # Load process patterns array
        config_processes=$(jq -r '.instanceManagement.detection.processes[]? // empty' "$service_json" 2>/dev/null || echo "")
        if [[ -n "$config_processes" ]]; then
            PROCESS_PATTERNS=()
            while IFS= read -r pattern; do
                [[ -n "$pattern" ]] && PROCESS_PATTERNS+=("$pattern")
            done <<< "$config_processes"
        fi
        
        # Load docker-compose files array
        local config_compose_files
        config_compose_files=$(jq -r '.instanceManagement.dockerCompose.files[]? // empty' "$service_json" 2>/dev/null || echo "")
        if [[ -n "$config_compose_files" ]]; then
            DOCKER_COMPOSE_FILES=()
            while IFS= read -r file; do
                [[ -n "$file" ]] && DOCKER_COMPOSE_FILES+=("$file")
            done <<< "$config_compose_files"
        else
            DOCKER_COMPOSE_FILES=("docker-compose.yml" "docker-compose.yaml" "compose.yml" "compose.yaml")
        fi
        
        # Load docker-compose services array
        local config_compose_services
        config_compose_services=$(jq -r '.instanceManagement.dockerCompose.services[]? // empty' "$service_json" 2>/dev/null || echo "")
        if [[ -n "$config_compose_services" ]]; then
            DOCKER_COMPOSE_SERVICES=()
            while IFS= read -r service; do
                [[ -n "$service" ]] && DOCKER_COMPOSE_SERVICES+=("$service")
            done <<< "$config_compose_services"
        else
            DOCKER_COMPOSE_SERVICES=("${APP_SERVICES[@]}")  # Default to APP_SERVICES
        fi
        
        log::debug "Instance management config loaded:"
        log::debug "  Enabled: $INSTANCE_MANAGEMENT_ENABLED"
        log::debug "  Strategy: $INSTANCE_STRATEGY"
        log::debug "  Services: ${APP_SERVICES[*]}"
        log::debug "  Containers: ${CONTAINER_PATTERNS[*]}"
        log::debug "  Processes: ${PROCESS_PATTERNS[*]}"
        log::debug "  Compose Files: ${DOCKER_COMPOSE_FILES[*]}"
        log::debug "  Compose Services: ${DOCKER_COMPOSE_SERVICES[*]}"
    else
        log::debug "No service.json found or jq not available, using defaults"
    fi
}

# ==============================================================================
# Docker Detection Functions
# ==============================================================================

# Check if Docker is available and running
instance::docker_available() {
    system::is_command "docker" && docker::_is_running
}

# Get all app-related Docker containers
instance::get_docker_containers() {
    if ! instance::docker_available; then
        return 0
    fi
    
    local containers_json
    # Get all containers (running and stopped) with app patterns
    if ! containers_json=$(docker::run ps -a --format json 2>/dev/null); then
        log::debug "Failed to get Docker containers"
        return 0
    fi
    
    # Parse containers and filter for app patterns
    while IFS= read -r container_json; do
        [[ -z "$container_json" ]] && continue
        
        local name state id created_at ports
        name=$(echo "$container_json" | jq -r '.Names' 2>/dev/null || echo "")
        state=$(echo "$container_json" | jq -r '.State' 2>/dev/null || echo "")
        id=$(echo "$container_json" | jq -r '.ID' 2>/dev/null || echo "")
        created_at=$(echo "$container_json" | jq -r '.CreatedAt' 2>/dev/null || echo "")
        ports=$(echo "$container_json" | jq -r '.Ports // ""' 2>/dev/null || echo "")
        
        # Check if this container matches our patterns
        for pattern in "${CONTAINER_PATTERNS[@]}"; do
            if [[ "$name" =~ $pattern ]]; then
                # Determine service type from container name
                local service="unknown"
                for svc in "${APP_SERVICES[@]}"; do
                    if [[ "$name" =~ $svc ]]; then
                        service="$svc"
                        break
                    fi
                done
                
                # Store container info
                DOCKER_INSTANCES["${service}_container"]="$name"
                DOCKER_INSTANCES["${service}_id"]="$id"
                DOCKER_INSTANCES["${service}_state"]="$state"
                DOCKER_INSTANCES["${service}_created"]="$created_at"
                DOCKER_INSTANCES["${service}_ports"]="$ports"
                
                log::debug "Found Docker container: $name ($service) - $state"
                break
            fi
        done
    done <<< "$containers_json"
}

# ==============================================================================
# Native Process Detection Functions
# ==============================================================================

# Get process info by PID
instance::get_process_info() {
    local pid="$1"
    local cmd start_time cwd
    
    # Get command
    if [[ "$OSTYPE" == "darwin"* ]]; then
        cmd=$(ps -p "$pid" -o comm= 2>/dev/null || echo "")
        start_time=$(ps -p "$pid" -o lstart= 2>/dev/null || echo "")
    else
        cmd=$(ps -p "$pid" -o cmd= 2>/dev/null || echo "")
        start_time=$(ps -p "$pid" -o lstart= 2>/dev/null || echo "")
    fi
    
    # Try to get working directory
    if [[ -e "/proc/$pid/cwd" ]]; then
        cwd=$(readlink "/proc/$pid/cwd" 2>/dev/null || echo "")
    else
        cwd=""
    fi
    
    echo "$cmd|$start_time|$cwd"
}

# Check if a process matches our app patterns
instance::is_app_process() {
    local cmd="$1"
    local cwd="$2"
    
    # Exclude shell scripts - we only want actual application processes
    if [[ "$cmd" =~ ^(/bin/)?(ba)?sh ]] || [[ "$cmd" =~ \.sh$ ]]; then
        return 1
    fi
    
    # Check command patterns
    for pattern in "${PROCESS_PATTERNS[@]}"; do
        if [[ "$cmd" =~ $pattern ]]; then
            # Additional check: ensure it's not a shell script running the pattern
            if ! [[ "$cmd" =~ ^(/bin/)?(ba)?sh ]]; then
                return 0
            fi
        fi
    done
    
    # Only check working directory for specific application processes
    # (node, npm, pnpm, python, etc. - not shell scripts)
    if [[ -n "$cwd" ]] && [[ "$cwd" =~ ${var_ROOT_DIR} ]]; then
        # Only consider it an app process if it's a known runtime
        if [[ "$cmd" =~ ^(node|npm|pnpm|python|java|ruby|php) ]]; then
            return 0
        fi
    fi
    
    return 1
}

# Get all native app processes
instance::get_native_processes() {
    # First, check for processes listening on service ports
    for service in "${!SERVICE_PORTS[@]}"; do
        local port="${SERVICE_PORTS[$service]}"
        local pids
        
        # Skip infrastructure services for native detection (they typically run in containers)
        if [[ "$service" == "database" || "$service" == "postgres" || "$service" == "redis" ]]; then
            continue
        fi
        
        # Get PIDs listening on this port
        pids=$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || echo "")
        
        if [[ -n "$pids" ]]; then
            for pid in $pids; do
                local info cmd start_time cwd
                info=$(instance::get_process_info "$pid")
                IFS='|' read -r cmd start_time cwd <<< "$info"
                
                # For port-based detection, we're more confident this is the right process
                NATIVE_INSTANCES["${service}_pid"]="$pid"
                NATIVE_INSTANCES["${service}_cmd"]="$cmd"
                NATIVE_INSTANCES["${service}_port"]="$port"
                NATIVE_INSTANCES["${service}_start"]="$start_time"
                NATIVE_INSTANCES["${service}_cwd"]="$cwd"
                
                log::debug "Found native process: $service (PID: $pid) on port $port"
            done
        fi
    done
    
    # Also search for app processes using pattern matching
    local pattern_search=""
    for pattern in "${PROCESS_PATTERNS[@]}"; do
        # Extract key terms from patterns for pgrep
        local key_term
        if [[ "$pattern" =~ node\..*server ]]; then
            key_term="server"
        elif [[ "$pattern" =~ npm.*dev ]]; then
            key_term="dev"
        elif [[ "$pattern" =~ pnpm.*dev ]]; then
            key_term="pnpm"
        elif [[ "$pattern" =~ vite ]]; then
            key_term="vite"
        elif [[ "$pattern" =~ concurrently ]]; then
            key_term="concurrently"
        fi
        
        if [[ -n "$key_term" && -z "$pattern_search" ]]; then
            pattern_search="$key_term"
        fi
    done
    
    # Only search if we have specific patterns, don't use project name as fallback
    # (that would match shell scripts and other non-application processes)
    local all_pids=""
    if [[ -n "$pattern_search" ]]; then
        all_pids=$(pgrep -f "$pattern_search" 2>/dev/null || echo "")
    fi
    
    for pid in $all_pids; do
        # Skip if we already tracked this PID
        local already_tracked=false
        for service in "${APP_SERVICES[@]}"; do
            if [[ "${NATIVE_INSTANCES[${service}_pid]:-}" == "$pid" ]]; then
                already_tracked=true
                break
            fi
        done
        
        if [[ "$already_tracked" == "false" ]]; then
            local info cmd start_time cwd
            info=$(instance::get_process_info "$pid")
            IFS='|' read -r cmd start_time cwd <<< "$info"
            
            if instance::is_app_process "$cmd" "$cwd"; then
                # Try to determine service type from command
                local service="unknown"
                if [[ "$cmd" =~ server ]]; then
                    service="server"
                elif [[ "$cmd" =~ ui|vite|dev ]]; then
                    service="ui"
                fi
                
                # Store as additional instance
                NATIVE_INSTANCES["${service}_extra_${pid}_pid"]="$pid"
                NATIVE_INSTANCES["${service}_extra_${pid}_cmd"]="$cmd"
                
                log::debug "Found additional native process: $cmd (PID: $pid)"
            fi
        fi
    done
}

# ==============================================================================
# Unified Detection and Status
# ==============================================================================

# Detect all running app instances
instance::detect_all() {
    # Load configuration first
    instance::load_config
    
    # Reset state
    DOCKER_INSTANCES=()
    NATIVE_INSTANCES=()
    INSTANCE_STATE="none"
    
    log::debug "Detecting application instances..."
    
    # Skip detection if disabled
    if ! flow::is_yes "$INSTANCE_MANAGEMENT_ENABLED"; then
        log::debug "Instance management disabled in configuration"
        return 0
    fi
    
    # Detect Docker containers
    instance::get_docker_containers
    
    # Detect native processes
    instance::get_native_processes
    
    # Determine overall state
    local has_docker=false
    local has_native=false
    
    # Check for any running Docker containers
    for service in "${APP_SERVICES[@]}"; do
        if [[ -n "${DOCKER_INSTANCES[${service}_container]:-}" ]] && 
           [[ "${DOCKER_INSTANCES[${service}_state]:-}" == "running" ]]; then
            has_docker=true
            break
        fi
    done
    
    # Check for any native processes
    for service in "${APP_SERVICES[@]}"; do
        if [[ -n "${NATIVE_INSTANCES[${service}_pid]:-}" ]]; then
            has_native=true
            break
        fi
    done
    
    # Also check for additional/unknown processes
    if [[ "$has_native" == "false" ]]; then
        for key in "${!NATIVE_INSTANCES[@]}"; do
            if [[ "$key" =~ _extra_.*_pid$ ]] && [[ -n "${NATIVE_INSTANCES[$key]}" ]]; then
                has_native=true
                break
            fi
        done
    fi
    
    # Set instance state
    if [[ "$has_docker" == "true" && "$has_native" == "true" ]]; then
        INSTANCE_STATE="mixed"
    elif [[ "$has_docker" == "true" ]]; then
        INSTANCE_STATE="docker"
    elif [[ "$has_native" == "true" ]]; then
        INSTANCE_STATE="native"
    else
        INSTANCE_STATE="none"
    fi
    
    log::debug "Instance state: $INSTANCE_STATE"
}

# Display detected instances in a user-friendly format
instance::display_status() {
    local has_instances=false
    local docker_count=0
    local native_count=0
    local orphan_count=0
    
    # Count everything first
    for service in "${APP_SERVICES[@]}"; do
        if [[ -n "${DOCKER_INSTANCES[${service}_container]:-}" ]] && 
           [[ "${DOCKER_INSTANCES[${service}_state]:-}" == "running" ]]; then
            docker_count=$((docker_count + 1))
            has_instances=true
        fi
        if [[ -n "${NATIVE_INSTANCES[${service}_pid]:-}" ]]; then
            native_count=$((native_count + 1))
            has_instances=true
        fi
    done
    
    # Count orphaned processes
    for key in "${!NATIVE_INSTANCES[@]}"; do
        if [[ "$key" =~ _extra_ ]]; then
            orphan_count=$((orphan_count + 1))
            has_instances=true
        fi
    done
    
    if [[ "$has_instances" == "false" ]]; then
        echo ""
        echo "✨ No application instances currently running"
        echo ""
        return
    fi
    
    # Show simple summary
    echo ""
    echo "🔍 Found running application instances:"
    echo ""
    
    if [[ $docker_count -gt 0 ]]; then
        echo "  📦 Docker: $docker_count container$([ $docker_count -ne 1 ] && echo 's') running"
        for service in "${APP_SERVICES[@]}"; do
            if [[ -n "${DOCKER_INSTANCES[${service}_container]:-}" ]] && 
               [[ "${DOCKER_INSTANCES[${service}_state]:-}" == "running" ]]; then
                echo "     • $service"
            fi
        done
    fi
    
    if [[ $native_count -gt 0 ]]; then
        echo "  🖥️  Services: $native_count process$([ $native_count -ne 1 ] && echo 'es') running"
        for service in "${APP_SERVICES[@]}"; do
            if [[ -n "${NATIVE_INSTANCES[${service}_pid]:-}" ]]; then
                local port="${NATIVE_INSTANCES[${service}_port]:-}"
                echo "     • $service (port $port)"
            fi
        done
    fi
    
    if [[ $orphan_count -gt 0 ]]; then
        echo "  ⚠️  Other: $orphan_count background process$([ $orphan_count -ne 1 ] && echo 'es')"
    fi
    
    echo ""
}

# ==============================================================================
# Shutdown Functions
# ==============================================================================

# Shutdown Docker containers
instance::shutdown_docker() {
    local compose_file="${var_DOCKER_COMPOSE_DEV_FILE:-}"
    
    # Build list of compose files to try
    local compose_files=()
    
    # Add configured compose file if it exists
    [[ -n "$compose_file" ]] && compose_files+=("$compose_file")
    
    # Add configured compose files from service.json
    for file in "${DOCKER_COMPOSE_FILES[@]}"; do
        compose_files+=("${var_ROOT_DIR}/$file")
    done
    
    # Try each compose file
    for compose_file in "${compose_files[@]}"; do
        if [[ -f "$compose_file" ]]; then
            log::info "Stopping Docker containers via docker-compose..."
            cd "$(dirname "$compose_file")"
            if docker::compose down; then
                log::success "Docker containers stopped successfully"
                return 0
            else
                log::error "Failed to stop Docker containers"
                return 1
            fi
        fi
    done
    
    # Fallback: stop containers individually
    log::info "Stopping Docker containers individually..."
    local stopped=0
    local failed=0
    
    for service in "${APP_SERVICES[@]}"; do
        if [[ -n "${DOCKER_INSTANCES[${service}_container]:-}" ]]; then
            local container="${DOCKER_INSTANCES[${service}_container]}"
            if docker::run stop "$container" >/dev/null 2>&1; then
                log::debug "Stopped container: $container"
                ((stopped++))
            else
                log::warning "Failed to stop container: $container"
                ((failed++))
            fi
        fi
    done
    
    if [[ $stopped -gt 0 ]]; then
        log::success "Stopped $stopped Docker container(s)"
    fi
    
    return $failed
}

# Shutdown native processes
instance::shutdown_native() {
    log::info "Stopping native processes..."
    local stopped=0
    local failed=0
    
    # Stop processes in reverse order (ui, server, database)
    local services=()
    for service in "${APP_SERVICES[@]}"; do
        services=("$service" "${services[@]}")  # prepend to reverse order
    done
    
    for service in "${services[@]}"; do
        if [[ -n "${NATIVE_INSTANCES[${service}_pid]:-}" ]]; then
            local pid="${NATIVE_INSTANCES[${service}_pid]}"
            
            # Try graceful shutdown first
            if kill -TERM "$pid" 2>/dev/null; then
                log::debug "Sent TERM signal to $service (PID: $pid)"
                
                # Wait up to 5 seconds for process to exit
                local count=0
                while [[ $count -lt 50 ]] && kill -0 "$pid" 2>/dev/null; do
                    sleep 0.1
                    ((count++))
                done
                
                # If still running, force kill
                if kill -0 "$pid" 2>/dev/null; then
                    log::warning "$service didn't stop gracefully, forcing..."
                    kill -KILL "$pid" 2>/dev/null || true
                fi
                
                ((stopped++))
            else
                log::warning "Process already gone: $service (PID: $pid)"
            fi
        fi
    done
    
    # Stop any additional processes
    for key in "${!NATIVE_INSTANCES[@]}"; do
        if [[ "$key" =~ _extra_.*_pid$ ]]; then
            local pid="${NATIVE_INSTANCES[$key]}"
            if kill -TERM "$pid" 2>/dev/null; then
                log::debug "Stopped additional process (PID: $pid)"
                ((stopped++))
            fi
        fi
    done
    
    # Also kill any orphaned APPLICATION processes (not shell scripts)
    # Build a list of PIDs to exclude (current process and parents)
    local exclude_pids="$$"  # Current process
    local parent_pid=$PPID
    while [[ "$parent_pid" -gt 1 ]]; do
        exclude_pids="$exclude_pids|$parent_pid"
        # Get parent of parent (if we can)
        if [[ -e "/proc/$parent_pid/stat" ]]; then
            parent_pid=$(awk '{print $4}' "/proc/$parent_pid/stat" 2>/dev/null || echo "1")
        else
            # macOS or no /proc - try ps
            parent_pid=$(ps -o ppid= -p "$parent_pid" 2>/dev/null | tr -d ' ' || echo "1")
        fi
    done
    
    # Search for actual application processes only (node, npm, pnpm running from our directory)
    local orphaned_pids=""
    
    # Look for node/npm/pnpm processes running from our project directory
    for cmd in node npm pnpm; do
        local cmd_pids
        cmd_pids=$(pgrep -f "$cmd.*${var_ROOT_DIR}" 2>/dev/null || echo "")
        
        for pid in $cmd_pids; do
            # Skip if it's in our exclude list
            if echo "$pid" | grep -qE "^($exclude_pids)$"; then
                continue
            fi
            
            # Skip if already tracked by instance manager
            local already_tracked=false
            for key in "${!NATIVE_INSTANCES[@]}"; do
                if [[ "${NATIVE_INSTANCES[$key]}" == "$pid" ]]; then
                    already_tracked=true
                    break
                fi
            done
            
            if [[ "$already_tracked" == "false" ]]; then
                orphaned_pids="$orphaned_pids $pid"
            fi
        done
    done
    
    if [[ -n "$orphaned_pids" ]]; then
        log::info "Found orphaned application processes, cleaning up..."
        for pid in $orphaned_pids; do
            if kill -TERM "$pid" 2>/dev/null; then
                log::debug "Stopped orphaned process (PID: $pid)"
                ((stopped++))
                
                # Give it a moment to exit
                sleep 0.5
                
                # Force kill if still running
                if kill -0 "$pid" 2>/dev/null; then
                    kill -KILL "$pid" 2>/dev/null || true
                fi
            fi
        done
    fi
    
    if [[ $stopped -gt 0 ]]; then
        log::success "Stopped $stopped native process(es)"
    fi
    
    return $failed
}

# Shutdown all app instances
instance::shutdown_all() {
    local docker_result=0
    local native_result=0
    
    case "$INSTANCE_STATE" in
        "docker")
            instance::shutdown_docker
            docker_result=$?
            ;;
        "native")
            instance::shutdown_native
            native_result=$?
            ;;
        "mixed")
            instance::shutdown_native
            native_result=$?
            instance::shutdown_docker
            docker_result=$?
            ;;
        "none")
            log::info "No instances to shut down"
            return 0
            ;;
    esac
    
    # Return non-zero if either failed
    return $((docker_result + native_result))
}

# Shutdown instances based on target
instance::shutdown_target() {
    local target="${1:-}"
    
    case "$target" in
        "native-linux"|"native-mac"|"native-win")
            # For native targets, stop both native and docker (since native may use docker for infrastructure)
            instance::shutdown_native
            instance::shutdown_docker
            ;;
        "docker"|"docker-only")
            # For docker target, only stop docker
            instance::shutdown_docker
            ;;
        *)
            # Unknown target, stop everything to be safe
            instance::shutdown_all
            ;;
    esac
}

# ==============================================================================
# User Interaction Functions
# ==============================================================================

# Get user choice for handling conflicts
instance::prompt_action() {
    local target="${1:-}"
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
    echo "⚠️  Application is already running!" >&2
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
    echo "" >&2
    echo "You need to choose what to do with the existing instances" >&2
    echo "before starting new ones:" >&2
    echo "" >&2
    echo "  [s] Stop all  - Shut down everything and start fresh ✨" >&2
    echo "  [k] Keep all  - Exit without making changes 👋" >&2
    echo "  [f] Force     - Try to start anyway (⚠️  risky!)" >&2
    echo "" >&2
    
    # Ensure prompt is visible
    echo -n "What do you want to do? [s/k/f] (default: s): " >&2
    
    local choice
    # Read from stdin
    read -r choice
    choice="${choice:-s}"
    
    # Normalize input
    choice=$(echo "$choice" | tr '[:upper:]' '[:lower:]' | tr -d '[:space:]')
    
    case "$choice" in
        stop|s|1|"")
            echo "stop_all"
            ;;
        keep|k|exit|e|2|3)
            echo "keep_exit"
            ;;
        force|f|4)
            echo "force"
            ;;
        *)
            log::warning "Invalid choice: '$choice'. Defaulting to 'stop all'."
            echo "stop_all"
            ;;
    esac
}

# Handle instance conflicts based on user preference
instance::handle_conflicts() {
    local target="${1:-}"
    local force_action="${2:-}"  # Optional: pre-selected action
    
    # Detect current instances
    instance::detect_all
    
    # If no instances found, nothing to do
    if [[ "$INSTANCE_STATE" == "none" ]]; then
        log::debug "No existing instances found"
        return 0
    fi
    
    # Check strategy - if multiple instances allowed, just warn and continue
    if [[ "$INSTANCE_STRATEGY" == "multiple" ]]; then
        log::info "Multiple instances allowed by configuration"
        instance::display_status
        return 0
    fi
    
    # Display current status
    instance::display_status
    
    # Determine action
    local action
    if [[ -n "$force_action" ]]; then
        action="$force_action"
        log::info "Using pre-selected action: $action"
    elif [[ "$CONFLICT_ACTION" == "stop" ]]; then
        action="stop_all"
        log::info "Auto-stopping instances (configured action: stop)"
    elif [[ "$CONFLICT_ACTION" == "force" ]]; then
        action="force"
        log::info "Force-starting with existing instances (configured action: force)"
    else
        # Check for saved preference
        action=$(instance::load_preference)
        
        # If no saved preference or interactive mode, prompt user
        if [[ -z "$action" ]] || [[ "${INTERACTIVE:-true}" == "true" ]]; then
            action=$(instance::prompt_action "$target")
        fi
    fi
    
    # Execute action
    case "$action" in
        "stop_all"|"stop_target"|"stop")
            log::info "Stopping all existing instances..."
            if instance::shutdown_all; then
                log::success "✅ All instances stopped successfully"
                sleep 1  # Brief pause to let user see the success message
                return 0
            else
                log::error "Failed to stop some instances"
                return 1
            fi
            ;;
        "keep_exit"|"keep")
            log::info "Keeping existing instances running. Exiting..."
            return 2  # Special exit code for "user chose to exit"
            ;;
        "force")
            log::warning "⚠️  Forcing start with existing instances - conflicts may occur!"
            log::warning "⚠️  You may experience port conflicts or duplicate services!"
            sleep 2  # Give user time to see the warning
            return 0
            ;;
        *)
            log::error "Unknown action: $action"
            return 1
            ;;
    esac
}

# ==============================================================================
# Preference Management
# ==============================================================================

# Get preference file path
instance::get_preference_file() {
    local app_dir="${HOME}/.${var_APP_NAME:-app}"
    mkdir -p "$app_dir"
    echo "$app_dir/.instance_preferences"
}

# Save user preference
instance::save_preference() {
    local action="$1"
    local duration="${2:-86400}"  # Default: 24 hours
    
    local pref_file
    pref_file=$(instance::get_preference_file)
    
    local expiry=$(($(date +%s) + duration))
    
    cat > "$pref_file" << EOF
{
    "default_action": "$action",
    "expires_at": $expiry,
    "saved_at": $(date +%s)
}
EOF
    
    log::debug "Saved preference: $action (expires in ${duration}s)"
}

# Load user preference
instance::load_preference() {
    local pref_file
    pref_file=$(instance::get_preference_file)
    
    if [[ ! -f "$pref_file" ]]; then
        return 0
    fi
    
    local action expires_at
    action=$(jq -r '.default_action // ""' "$pref_file" 2>/dev/null || echo "")
    expires_at=$(jq -r '.expires_at // 0' "$pref_file" 2>/dev/null || echo "0")
    
    # Check if preference is still valid
    local now
    now=$(date +%s)
    
    if [[ -n "$action" ]] && [[ $expires_at -gt $now ]]; then
        log::debug "Using saved preference: $action"
        echo "$action"
    else
        # Preference expired or invalid
        rm -f "$pref_file"
        echo ""
    fi
}

# Clear saved preferences
instance::clear_preferences() {
    local pref_file
    pref_file=$(instance::get_preference_file)
    rm -f "$pref_file"
    log::info "Cleared instance preferences"
}

# ==============================================================================
# Helper Functions
# ==============================================================================

# Check if we should skip instance checks
instance::should_skip_check() {
    # Check if instance management is disabled in config
    if ! flow::is_yes "$INSTANCE_MANAGEMENT_ENABLED"; then
        return 0
    fi
    
    # Check environment variable (accept yes/true/1)
    local skip_check="${SKIP_INSTANCE_CHECK:-no}"
    if [[ "$skip_check" == "yes" ]] || [[ "$skip_check" == "true" ]] || [[ "$skip_check" == "1" ]]; then
        return 0
    fi
    
    # Check if running in CI/CD
    if [[ "${CI:-}" == "true" ]] || [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
        return 0
    fi
    
    return 1
}

# Get summary of what's running
instance::get_summary() {
    local docker_count=0
    local native_count=0
    local extra_count=0
    
    # Count Docker containers
    for service in "${APP_SERVICES[@]}"; do
        if [[ -n "${DOCKER_INSTANCES[${service}_container]:-}" ]] && 
           [[ "${DOCKER_INSTANCES[${service}_state]:-}" == "running" ]]; then
            ((docker_count++))
        fi
    done
    
    # Count native processes
    for service in "${APP_SERVICES[@]}"; do
        if [[ -n "${NATIVE_INSTANCES[${service}_pid]:-}" ]]; then
            ((native_count++))
        fi
    done
    
    # Count extra/orphaned processes
    for key in "${!NATIVE_INSTANCES[@]}"; do
        if [[ "$key" =~ _extra_.*_pid$ ]]; then
            ((extra_count++))
        fi
    done
    
    # Build human-readable summary
    local parts=()
    if [[ $docker_count -gt 0 ]]; then
        parts+=("$docker_count Docker container$([ $docker_count -ne 1 ] && echo 's')")
    fi
    if [[ $native_count -gt 0 ]]; then
        parts+=("$native_count service$([ $native_count -ne 1 ] && echo 's')")
    fi
    if [[ $extra_count -gt 0 ]]; then
        parts+=("$extra_count orphaned process$([ $extra_count -ne 1 ] && echo 'es')")
    fi
    
    # Join with commas and "and"
    local result=""
    if [[ ${#parts[@]} -eq 0 ]]; then
        result="none"
    elif [[ ${#parts[@]} -eq 1 ]]; then
        result="${parts[0]}"
    elif [[ ${#parts[@]} -eq 2 ]]; then
        result="${parts[0]} and ${parts[1]}"
    else
        # Join all but last with commas, then add "and" before last
        local all_but_last=""
        local i
        for ((i=0; i<${#parts[@]}-1; i++)); do
            if [[ $i -eq 0 ]]; then
                all_but_last="${parts[i]}"
            else
                all_but_last="${all_but_last}, ${parts[i]}"
            fi
        done
        result="${all_but_last}, and ${parts[-1]}"
    fi
    
    echo "$result"
}

# ==============================================================================
# Export Functions
# ==============================================================================

export -f instance::detect_all
export -f instance::display_status
export -f instance::shutdown_all
export -f instance::shutdown_docker
export -f instance::shutdown_native
export -f instance::shutdown_target
export -f instance::handle_conflicts
export -f instance::save_preference
export -f instance::load_preference
export -f instance::clear_preferences
export -f instance::should_skip_check
export -f instance::get_summary
export -f instance::load_config