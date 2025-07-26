#!/usr/bin/env bash
# Instance management for Vrooli development environments
# Detects and manages running Vrooli instances across Docker and native deployments
set -euo pipefail

INSTANCE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${INSTANCE_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${INSTANCE_DIR}/../utils/flow.sh"
# shellcheck disable=SC1091
source "${INSTANCE_DIR}/../utils/system.sh"
# shellcheck disable=SC1091
source "${INSTANCE_DIR}/../utils/docker.sh"
# shellcheck disable=SC1091
source "${INSTANCE_DIR}/../utils/exit_codes.sh"

# Data structures to hold instance information
declare -gA DOCKER_INSTANCES=()
declare -gA NATIVE_INSTANCES=()
declare -g INSTANCE_STATE="none"  # none, docker, native, mixed

# Core Vrooli services
declare -ga VROOLI_SERVICES=("postgres" "redis" "server" "jobs" "ui")
declare -gA SERVICE_PORTS=(
    ["postgres"]="5432"
    ["redis"]="6379"
    ["server"]="5329"
    ["jobs"]="4001"
    ["ui"]="3000"
)

# Container name patterns for Vrooli
declare -ga CONTAINER_PATTERNS=(
    "vrooli[-_]postgres"
    "vrooli[-_]redis"
    "vrooli[-_]server"
    "vrooli[-_]jobs"
    "vrooli[-_]ui"
    "vrooli[-_]app"
)

# Process patterns for native Vrooli instances
declare -ga PROCESS_PATTERNS=(
    "node.*vrooli.*server"
    "node.*vrooli.*jobs"
    "npm.*vrooli.*ui"
    "pnpm.*vrooli.*ui"
    "node.*@vrooli/server"
    "node.*@vrooli/jobs"
    "node.*@vrooli/ui"
    "vite.*vrooli"
    "concurrently.*SERVER,JOBS,UI"
    "pnpm.*filter.*@vrooli"
    "bash.*package.*start.sh"
    "node.*--inspect.*dist/index\.js"
    "node.*--experimental-modules.*dist/index\.js"
)

# ==============================================================================
# Docker Detection Functions
# ==============================================================================

# Check if Docker is available and running
instance::docker_available() {
    system::is_command "docker" && docker::_is_running
}

# Get all Vrooli-related Docker containers
instance::get_docker_containers() {
    if ! instance::docker_available; then
        return 0
    fi
    
    local containers_json
    # Get all containers (running and stopped) with Vrooli in the name
    if ! containers_json=$(docker::run ps -a --format json 2>/dev/null); then
        log::debug "Failed to get Docker containers"
        return 0
    fi
    
    # Parse containers and filter for Vrooli
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
                for svc in "${VROOLI_SERVICES[@]}"; do
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

# Check if a process is a Vrooli process
instance::is_vrooli_process() {
    local cmd="$1"
    local cwd="$2"
    
    # Check command patterns
    for pattern in "${PROCESS_PATTERNS[@]}"; do
        if [[ "$cmd" =~ $pattern ]]; then
            return 0
        fi
    done
    
    # Check if working directory contains Vrooli
    if [[ -n "$cwd" && "$cwd" =~ [Vv]rooli ]]; then
        return 0
    fi
    
    return 1
}

# Get all native Vrooli processes
instance::get_native_processes() {
    # First, check for processes listening on service ports
    for service in "${!SERVICE_PORTS[@]}"; do
        local port="${SERVICE_PORTS[$service]}"
        local pids
        
        # Skip infrastructure services for native detection (they run in Docker)
        if [[ "$service" == "postgres" || "$service" == "redis" ]]; then
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
    
    # Also search for Vrooli processes not bound to expected ports
    local all_pids
    all_pids=$(pgrep -f "vrooli" 2>/dev/null || echo "")
    
    for pid in $all_pids; do
        # Skip if we already tracked this PID
        local already_tracked=false
        for service in "${VROOLI_SERVICES[@]}"; do
            if [[ "${NATIVE_INSTANCES[${service}_pid]:-}" == "$pid" ]]; then
                already_tracked=true
                break
            fi
        done
        
        if [[ "$already_tracked" == "false" ]]; then
            local info cmd start_time cwd
            info=$(instance::get_process_info "$pid")
            IFS='|' read -r cmd start_time cwd <<< "$info"
            
            if instance::is_vrooli_process "$cmd" "$cwd"; then
                # Try to determine service type from command
                local service="unknown"
                if [[ "$cmd" =~ server ]]; then
                    service="server"
                elif [[ "$cmd" =~ jobs ]]; then
                    service="jobs"
                elif [[ "$cmd" =~ ui|vite ]]; then
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

# Detect all running Vrooli instances
instance::detect_all() {
    # Reset state
    DOCKER_INSTANCES=()
    NATIVE_INSTANCES=()
    INSTANCE_STATE="none"
    
    log::debug "Detecting Vrooli instances..."
    
    # Detect Docker containers
    instance::get_docker_containers
    
    # Detect native processes
    instance::get_native_processes
    
    # Determine overall state
    local has_docker=false
    local has_native=false
    
    # Check for any running Docker containers
    for service in "${VROOLI_SERVICES[@]}"; do
        if [[ -n "${DOCKER_INSTANCES[${service}_container]:-}" ]] && 
           [[ "${DOCKER_INSTANCES[${service}_state]:-}" == "running" ]]; then
            has_docker=true
            break
        fi
    done
    
    # Check for any native processes
    for service in "${VROOLI_SERVICES[@]}"; do
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
    for service in "${VROOLI_SERVICES[@]}"; do
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
        echo "✨ No Vrooli instances currently running"
        echo ""
        return
    fi
    
    # Show simple summary
    echo ""
    echo "🔍 Found running Vrooli instances:"
    echo ""
    
    if [[ $docker_count -gt 0 ]]; then
        echo "  📦 Docker: $docker_count container$([ $docker_count -ne 1 ] && echo 's') running"
        for service in "${VROOLI_SERVICES[@]}"; do
            if [[ -n "${DOCKER_INSTANCES[${service}_container]:-}" ]] && 
               [[ "${DOCKER_INSTANCES[${service}_state]:-}" == "running" ]]; then
                echo "     • $service"
            fi
        done
    fi
    
    if [[ $native_count -gt 0 ]]; then
        echo "  🖥️  Services: $native_count process$([ $native_count -ne 1 ] && echo 'es') running"
        for service in "${VROOLI_SERVICES[@]}"; do
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
    local compose_file="${var_ROOT_DIR:-/home/matthalloran8/Vrooli}/docker-compose.yml"
    
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
    else
        # Fallback: stop containers individually
        log::info "Stopping Docker containers individually..."
        local stopped=0
        local failed=0
        
        for service in "${VROOLI_SERVICES[@]}"; do
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
    fi
}

# Shutdown native processes
instance::shutdown_native() {
    log::info "Stopping native processes..."
    local stopped=0
    local failed=0
    
    # Stop processes in reverse order (UI, jobs, server)
    local services=("ui" "jobs" "server")
    
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
    
    # Also kill any orphaned Vrooli processes using pgrep
    local orphaned_pids
    orphaned_pids=$(pgrep -f "vrooli|@vrooli|concurrently.*SERVER,JOBS,UI" 2>/dev/null || echo "")
    
    if [[ -n "$orphaned_pids" ]]; then
        log::info "Found orphaned Vrooli processes, cleaning up..."
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

# Shutdown all Vrooli instances
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
            # For native targets, stop both native and docker (since native uses docker for DB)
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
    echo "⚠️  Vrooli is already running!" >&2
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
    
    # Display current status
    instance::display_status
    
    # Determine action
    local action
    if [[ -n "$force_action" ]]; then
        action="$force_action"
        log::info "Using pre-selected action: $action"
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
    local vrooli_dir="${HOME}/.vrooli"
    mkdir -p "$vrooli_dir"
    echo "$vrooli_dir/.instance_preferences"
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
    for service in "${VROOLI_SERVICES[@]}"; do
        if [[ -n "${DOCKER_INSTANCES[${service}_container]:-}" ]] && 
           [[ "${DOCKER_INSTANCES[${service}_state]:-}" == "running" ]]; then
            ((docker_count++))
        fi
    done
    
    # Count native processes
    for service in "${VROOLI_SERVICES[@]}"; do
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