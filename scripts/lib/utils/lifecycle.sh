#!/usr/bin/env bash
#######################################
# Lifecycle Management Library
# Provides lifecycle phase execution and service orchestration
# 
# Can be used as a library or called directly:
#   lifecycle.sh <scenario_name> <phase> [options]
#######################################

set -euo pipefail

# Source resource verification functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/resource-verification.sh"

#######################################
# Enhanced develop lifecycle with conditional setup
# Arguments:
#   $1 - Phase name
#   $@ - Additional arguments (parsed for --fast flag)
# Returns:
#   0 on success, 1 on failure
#######################################
lifecycle::develop_with_auto_setup() {
    local phase="$1"
    shift
    
    # Parse fast mode flag
    # When running within Vrooli context (vrooli develop), fast mode is always enabled
    # for generated apps since the main Vrooli setup handles system-level checks.
    # Individual app runs (vrooli app start) use normal mode for thorough standalone setup.
    local fast_mode=false
    for arg in "$@"; do
        [[ "$arg" == "--fast" ]] && fast_mode=true
    done
    
    # Export fast mode for child processes
    export FAST_MODE="$fast_mode"
    
    # Use parameter expansion instead of basename subshell
    local app_name="${PWD##*/}"
    log::info "Starting develop lifecycle for $app_name"
    
    # Check if setup is needed
    log::debug "Checking if setup is needed (VROOLI_ROOT=${VROOLI_ROOT:-not set}, APP_ROOT=${APP_ROOT:-not set})"
    if setup::is_needed "$PWD"; then
        # Display specific reasons for setup
        log::info "Setup required for the following reason(s):"
        if [[ ${#SETUP_REASONS[@]} -gt 0 ]]; then
            for reason in "${SETUP_REASONS[@]}"; do
                echo "  • $reason" >&2
            done
        else
            echo "  • General setup required" >&2
        fi
        echo "" >&2
        
        # Determine setup mode
        if [[ "$fast_mode" == "true" ]]; then
            log::info "Running setup in fast mode..."
        else
            log::info "Running setup..."
        fi
        
        # Set managed resources for generated apps to skip Vrooli's resources
        if [[ -n "${VROOLI_ROOT:-}" ]] && [[ "$VROOLI_ROOT" != "$PWD" ]]; then
            local main_service_json="${VROOLI_ROOT}/.vrooli/service.json"
            if [[ -f "$main_service_json" ]]; then
                VROOLI_MANAGED_RESOURCES=$(jq -r '.resources | to_entries[] | .value | to_entries[] | select(.value | if type == "boolean" then . elif type == "object" then .enabled // false else false end) | .key' "$main_service_json" | paste -sd,)
                export VROOLI_MANAGED_RESOURCES
                log::debug "Set VROOLI_MANAGED_RESOURCES for generated app: $(echo "$VROOLI_MANAGED_RESOURCES" | tr ',' ' ' | wc -w) resources"
            fi
        fi
        
        # Run setup phase with original arguments
        lifecycle::execute_phase "setup" "$@" || {
            log::error "Setup failed, cannot start develop mode"
            return 1
        }
        
        # Mark setup as complete
        setup::mark_complete
        log::success "Setup completed, proceeding with develop"
    else
        log::info "✓ Setup is current, proceeding directly to develop"
    fi
    
    # Verify required resources are running before starting develop phase
    log::info "Verifying required resources are running..."
    if ! resource_verify::verify_with_priority; then
        log::warning "Some required resources failed to start, but continuing with develop phase"
        log::info "Scenarios may fail if they depend on unavailable resources"
    fi
    
    # Now run the actual develop phase
    log::info "Starting develop phase..."
    lifecycle::execute_phase "$phase" "$@"
}

#######################################
# Execute lifecycle phase
# Simple function that runs steps from service.json
# Arguments:
#   $1 - Phase name
#   $@ - Additional arguments passed to phase
# Returns:
#   0 on success, 1 on failure
#######################################
lifecycle::execute_phase() {
    local phase="$1"
    shift  # Remove phase from arguments
    
    # Extract app name from working directory to avoid PID collisions
    local app_name="${PWD##*/}"
    
    # Get phase description
    local description
    description=$(json::get_value ".lifecycle.${phase}.description" "")
    
    log::header "Starting phase: $phase"
    [[ -n "$description" ]] && log::info "$description"
    
    # Parse common arguments into environment variables
    for arg in "$@"; do
        case "$arg" in
            --target=*) export TARGET="${arg#*=}" ;;
            --environment=*) export ENVIRONMENT="${arg#*=}" ;;
            --dry-run) export DRY_RUN="true" ;;
            --fast) export FAST_MODE="true" ;;
            *) ;; # Other args available via $@
        esac
    done
    
    # Export phase info for scripts
    export LIFECYCLE_PHASE="$phase"
    export LIFECYCLE_ARGS="$*"
    
    # Allocate and export dynamic ports from service.json
    lifecycle::allocate_service_ports
    
    # Get and execute steps
    local steps
    steps=$(json::get_value ".lifecycle.${phase}.steps" "[]")
    
    if [[ "$steps" == "[]" ]]; then
        log::warning "No steps defined for phase: $phase"
        return 0
    fi
    
    # Execute each step
    local step_count=0
    local total_steps
    total_steps=$(echo "$steps" | jq 'length')
    
    # Use a predictable temp file to avoid mktemp subshell
    local steps_file="/tmp/vrooli-steps-$$-$RANDOM.tmp"
    if ! echo "$steps" | jq -c '.[]' > "$steps_file" 2>/dev/null; then
        log::error "Failed to parse steps JSON"
        rm -f "$steps_file"
        return 1
    fi
    
    # DEBUG: Check if file was created
    if [[ ! -f "$steps_file" ]]; then
        log::error "Steps file was not created at $steps_file"
        return 1
    fi
    
    # Ensure file ends with newline to prevent read issues
    echo "" >> "$steps_file"
    
    # Track background processes for cleanup
    local bg_processes=()
    trap 'for process in "${bg_processes[@]}"; do pm::stop "$process" 2>/dev/null || true; done; rm -f "${steps_file:-}"' EXIT INT TERM
    
    while IFS= read -r step; do
        step_count=$((step_count + 1))
        
        # Validate step is not empty
        if [[ -z "$step" ]]; then
            continue
        fi
        
        local name
        name=$(echo "$step" | jq -r '.name // "unnamed"' 2>/dev/null) || name="unnamed"
        local run
        run=$(echo "$step" | jq -r '.run // ""' 2>/dev/null) || run=""
        local desc
        desc=$(echo "$step" | jq -r '.description // ""' 2>/dev/null) || desc=""
        local is_background
        is_background=$(echo "$step" | jq -r '.background // false' 2>/dev/null) || is_background="false"
        
        [[ -z "$run" ]] && continue
        
        log::info "[$step_count/$total_steps] Executing: $name"
        [[ -n "$desc" ]] && echo "  → $desc" >&2
        
        if [[ "${DRY_RUN:-}" == "true" ]]; then
            echo "[DRY RUN] Would execute: $run" >&2
        else
            # Process template variables in the command before execution
            local processed_run
            if command -v secrets::process_bash_templates >/dev/null 2>&1; then
                processed_run=$(secrets::process_bash_templates "$run")
            else
                processed_run="$run"
            fi
            
            # Remove any trailing & from command if background flag is set
            if [[ "$is_background" == "true" ]]; then
                processed_run="${processed_run% &}"
                processed_run="${processed_run%&}"
            fi
            
            # Execute in appropriate directory (scenario dir for scenarios, root for main)
            # Port variables are already exported by lifecycle::allocate_service_ports
            local exec_dir="${SCENARIO_PATH:-$var_ROOT_DIR}"
            if [[ "$is_background" == "true" ]]; then
                # Use process manager for background processes
                local process_name="vrooli.${phase}.${app_name}.${name}"
                log::info "[DEBUG] Creating process: phase=$phase app_name=$app_name name=$name -> $process_name"
                if command -v pm::start >/dev/null 2>&1; then
                    if pm::start "$process_name" "$processed_run" "$exec_dir"; then
                        bg_processes+=("$process_name")
                        log::info "  Started background process: $process_name"
                    else
                        log::error "Failed to start background process: $process_name"
                        rm -f "$steps_file"
                        return 1
                    fi
                else
                    log::warning "Process manager not available, falling back to manual process management"
                    (cd "$exec_dir" && export API_PORT="${API_PORT:-}" && exec bash -c "$processed_run") &
                    local bg_pid=$!
                    log::info "  Started background process (PID: $bg_pid) - manual management"
                fi
            else
                # Run without timeout wrapper to preserve terminal control for sudo
                (cd "$exec_dir" && export API_PORT="${API_PORT:-}" && bash -c "$processed_run") || {
                    local exit_code=$?
                    log::error "Step '$name' failed with exit code $exit_code"
                    rm -f "$steps_file"
                    return $exit_code
                }
            fi
        fi
    done < "$steps_file"
    
    rm -f "$steps_file"
    
    # For develop phase with background processes, show status and exit
    # Process manager handles all lifecycle management
    if [[ "$phase" == "develop" ]] && [[ ${#bg_processes[@]} -gt 0 ]]; then
        log::info "Development services started successfully:"
        for process in "${bg_processes[@]}"; do
            echo "  • $process"
        done
        echo ""
        
        # Show next steps with app status
        show_develop_next_steps
    fi
    
    log::success "Phase '$phase' completed successfully"
    return 0
}

#######################################
# Show next steps after starting develop phase
# Displays app status and useful commands
#######################################
show_develop_next_steps() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log::header "🎯 NEXT STEPS"
    echo ""
    
    # Brief wait for services to initialize
    sleep 2
    
    # Get app status from API if available
    local api_available=false
    local app_count=0
    local running_count=0
    
    if command -v curl >/dev/null 2>&1 && curl -s --connect-timeout 1 --max-time 2 "http://localhost:${VROOLI_API_PORT:-8092}/health" >/dev/null 2>&1; then
        api_available=true
        local response
        response=$(curl -s --connect-timeout 2 --max-time 3 "http://localhost:${VROOLI_API_PORT:-8092}/apps" 2>/dev/null || echo '{"success": false}')
        
        if echo "$response" | jq -e '.success' >/dev/null 2>&1 2>/dev/null; then
            local apps_data
            apps_data=$(echo "$response" | jq -r '.data' 2>/dev/null)
            if [[ -n "$apps_data" ]]; then
                app_count=$(echo "$apps_data" | jq 'length' 2>/dev/null || echo "0")
                running_count=$(echo "$apps_data" | jq '[.[] | select(.runtime_status == "running")] | length' 2>/dev/null || echo "0")
            fi
        fi
    fi
    
    # Show app status
    if [[ "$api_available" == "true" ]] && [[ "$app_count" -gt 0 ]]; then
        log::info "📦 App Status: $running_count/$app_count apps running"
        echo ""
        
        if [[ "$running_count" -gt 0 ]]; then
            echo "🟢 Running apps can be accessed via their individual URLs"
            echo "   Run 'vrooli scenario list' to see all scenarios"
        else
            echo "🔴 Apps are starting up (may take 30-60 seconds)"
            echo "   Run 'vrooli status' to check their status"
        fi
    else
        echo "📦 Apps are starting up in the background..."
    fi
    
    echo ""
    log::info "🔍 Useful Commands:"
    echo "  • 'vrooli status'           - Check system health and scenario status"
    echo "  • 'vrooli scenario list'    - List all scenarios"
    echo "  • 'vrooli scenario run <name>' - Run a specific scenario"
    echo "  • 'vrooli stop scenarios'   - Stop all scenarios"
    
    echo ""
    log::info "🌐 Main Services:"
    echo "  • Unified API: http://localhost:${VROOLI_API_PORT:-8092}"
    
    echo ""
    log::info "💡 Troubleshooting:"
    echo "  • If scenarios fail to start: Check logs in ~/.vrooli/logs/scenarios/"
    echo "  • If setup fails: Try 'vrooli scenario run <name>' to run individual scenarios"
    echo "  • For system issues: Run 'vrooli status --verbose' for detailed health check"
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

#######################################
# Allocate dynamic ports from service.json
# Reads .ports config and allocates/exports ports
# Returns:
#   0 on success, 1 on failure
#######################################
lifecycle::allocate_service_ports() {
    local ports_config
    ports_config=$(json::get_value '.ports // {}' '{}')
    
    [[ "$ports_config" == '{}' ]] && return 0
    
    # Source port registry if available for conflict prevention
    local port_registry="${var_SCRIPTS_RESOURCES_DIR:-${var_ROOT_DIR}/scripts/resources}/port_registry.sh"
    if [[ -f "$port_registry" ]]; then
        # shellcheck disable=SC1090
        source "$port_registry" 2>/dev/null || log::warning "Could not load port registry"
        
        # Export standard database URLs using postgres container credentials
        local postgres_port="${RESOURCE_PORTS[postgres]:-5433}"
        if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' | grep -q "vrooli-postgres-main"; then
            # Get credentials from postgres container environment
            local container_env
            container_env=$(docker inspect vrooli-postgres-main --format '{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null)
            
            local db_user db_pass db_name
            db_user=$(echo "$container_env" | grep '^POSTGRES_USER=' | cut -d'=' -f2)
            db_pass=$(echo "$container_env" | grep '^POSTGRES_PASSWORD=' | cut -d'=' -f2)
            db_name=$(echo "$container_env" | grep '^POSTGRES_DB=' | cut -d'=' -f2)
            
            if [[ -n "$db_user" && -n "$db_pass" && -n "$db_name" ]]; then
                export POSTGRES_URL="postgres://${db_user}:${db_pass}@localhost:${postgres_port}/${db_name}?sslmode=disable"
                export DATABASE_URL="$POSTGRES_URL"  # Some scenarios use DATABASE_URL
                export POSTGRES_USER="$db_user"
                export POSTGRES_PASSWORD="$db_pass"
                export POSTGRES_DB="$db_name"
                log::debug "Exported postgres credentials from container"
            fi
        fi
    fi
    
    # Build list of reserved ports from project-level service.json if available
    local -a reserved_ports=()
    local project_service_json="${var_ROOT_DIR}/.vrooli/service.json"
    if [[ -f "$project_service_json" ]]; then
        # Extract fixed ports from project service.json
        while IFS= read -r port; do
            [[ -n "$port" ]] && reserved_ports+=("$port")
        done < <(jq -r '.ports | to_entries[] | select(.value.fixed != null) | .value.fixed' "$project_service_json" 2>/dev/null)
    fi
    
    # Add resource ports from port registry to reserved list
    if declare -p RESOURCE_PORTS &>/dev/null 2>&1; then
        for port in "${RESOURCE_PORTS[@]}"; do
            reserved_ports+=("$port")
        done
    fi
    
    log::info "Allocating service ports..."
    
    # Process each port configuration - using process substitution to avoid subshell
    while IFS= read -r port_entry; do
        # Decode the port configuration
        local port_name port_config
        port_name=$(echo "$port_entry" | base64 -d | jq -r '.key')
        port_config=$(echo "$port_entry" | base64 -d | jq -r '.value')
        
        local env_var range fixed fallback description
        env_var=$(echo "$port_config" | jq -r '.env_var // ""')
        range=$(echo "$port_config" | jq -r '.range // ""')
        fixed=$(echo "$port_config" | jq -r '.fixed // ""')
        fallback=$(echo "$port_config" | jq -r '.fallback // "auto"')
        description=$(echo "$port_config" | jq -r '.description // ""')
        
        [[ -z "$env_var" ]] && continue
        
        # Check if already set
        if [[ -n "${!env_var:-}" ]]; then
            log::info "  • $port_name: Using existing $env_var=${!env_var}"
            continue
        fi
        
        local allocated_port=""
        
        # Try fixed port first
        if [[ -n "$fixed" ]]; then
            allocated_port="$fixed"
        # Try dynamic range allocation
        elif [[ -n "$range" ]]; then
            local start_port end_port
            start_port=$(echo "$range" | cut -d'-' -f1)
            end_port=$(echo "$range" | cut -d'-' -f2)
            
            if command -v ports::find_available_in_range &>/dev/null; then
                allocated_port=$(ports::find_available_in_range "$start_port" "$end_port")
            else
                # Enhanced allocation checking against reserved ports
                for ((port=start_port; port<=end_port; port++)); do
                    # Check if port is in reserved list
                    local is_reserved=false
                    for reserved in "${reserved_ports[@]}"; do
                        if [[ "$port" == "$reserved" ]]; then
                            is_reserved=true
                            break
                        fi
                    done
                    
                    # Skip if reserved or already in use
                    if [[ "$is_reserved" == "true" ]]; then
                        continue
                    fi
                    
                    if ! nc -z localhost "$port" 2>/dev/null; then
                        allocated_port="$port"
                        break
                    fi
                done
            fi
        fi
        
        # Handle allocation result
        if [[ -n "$allocated_port" ]]; then
            export "$env_var=$allocated_port"
            log::info "  • $port_name: Allocated $env_var=$allocated_port"
            [[ -n "$description" ]] && log::info "    $description"
        elif [[ "$fallback" == "error" ]]; then
            log::error "Failed to allocate port for $port_name"
            return 1
        else
            log::warning "Could not allocate port for $port_name (fallback: auto)"
        fi
    done < <(echo "$ports_config" | jq -r 'to_entries | .[] | @base64')
    
    return 0
}

#######################################
# Main entry point for direct execution
# Usage: lifecycle.sh <scenario_name> <phase> [options]
#######################################
lifecycle::main() {
    # Check if being sourced or executed
    if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
        # Being sourced, dont execute main
        return 0
    fi
    
    # Parse arguments
    local scenario_name="${1:-}"
    local phase="${2:-}"
    shift 2 || true
    
    if [[ -z "$scenario_name" ]] || [[ -z "$phase" ]]; then
        echo "Usage: $0 <scenario_name> <phase> [options]" >&2
        echo "  scenario_name: Name of scenario in scenarios/ directory" >&2
        echo "  phase: Lifecycle phase (develop, test, stop, etc.)" >&2
        echo "  options: --fast, --dry-run, etc." >&2
        exit 1
    fi
    
    # Source required libraries if not already sourced
    SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
    
    # Check if var.sh is already sourced
    if [[ -z "${var_ROOT_DIR:-}" ]]; then
        source "${SCRIPT_DIR}/var.sh"
    fi
    
    # Check if other required libs are sourced
    if ! declare -f log::info &>/dev/null; then
        source "${SCRIPT_DIR}/log.sh"
    fi
    
    if ! declare -f json::get &>/dev/null; then
        source "${SCRIPT_DIR}/json.sh"
    fi
    
    if ! declare -f setup::is_needed &>/dev/null; then
        source "${SCRIPT_DIR}/setup.sh"
    fi
    
    if ! declare -f process_manager::start &>/dev/null; then
        source "${SCRIPT_DIR}/../process-manager.sh"
    fi
    
    # Source port registry
    if [[ -f "${var_ROOT_DIR}/scripts/resources/port_registry.sh" ]]; then
        source "${var_ROOT_DIR}/scripts/resources/port_registry.sh"
    fi
    
    # Set up scenario context
    local scenario_path="${var_ROOT_DIR}/scenarios/${scenario_name}"
    
    if [[ ! -d "$scenario_path" ]]; then
        log::error "Scenario not found: $scenario_name"
        exit 1
    fi
    
    # Set service.json path for scenario
    export SERVICE_JSON_PATH="${scenario_path}/.vrooli/service.json"
    
    if [[ ! -f "$SERVICE_JSON_PATH" ]]; then
        log::error "service.json not found for scenario: $scenario_name"
        exit 1
    fi
    
    # Set scenario environment
    export SCENARIO_NAME="$scenario_name"
    export SCENARIO_PATH="$scenario_path"
    export SCENARIO_MODE=true
    export VROOLI_ROOT="${var_ROOT_DIR}"
    
    # Set PM2 paths for scenario isolation
    export PM_HOME="${HOME}/.vrooli/processes/scenarios/${scenario_name}"
    export PM_LOG_DIR="${HOME}/.vrooli/logs/scenarios/${scenario_name}"
    
    # Change to scenario directory for execution
    cd "$scenario_path" || exit 1
    
    # Execute the phase
    log::info "Executing phase '$phase' for scenario '$scenario_name'"
    
    # Route to appropriate handler based on phase
    if [[ "$phase" == "develop" ]]; then
        lifecycle::develop_with_auto_setup "$phase" "$@"
    else
        lifecycle::execute_phase "$phase" "$@"
    fi
}

# Execute main if running directly
lifecycle::main "$@"
