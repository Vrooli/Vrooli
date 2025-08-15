#!/usr/bin/env bash
set -euo pipefail

#######################################
# Application Management Script
# 
# Single entry point for all lifecycle operations.
# All behavior is controlled by .vrooli/service.json
#######################################

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source utilities
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/utils/json.sh"
# shellcheck disable=SC1091
source "${var_PORT_REGISTRY_FILE}" 2>/dev/null || true
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/service/secrets.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/process-manager.sh" 2>/dev/null || true

#######################################
# Execute lifecycle phase
# Simple function that runs steps from service.json
#######################################
manage::execute_phase() {
    local phase="$1"
    shift  # Remove phase from arguments
    
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
            *) ;; # Other args available via $@
        esac
    done
    
    # Export phase info for scripts
    export LIFECYCLE_PHASE="$phase"
    export LIFECYCLE_ARGS="$*"
    
    # Allocate and export dynamic ports from service.json
    manage::allocate_service_ports
    
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
    
    # Use a temporary file to avoid pipeline issues
    local steps_file=$(mktemp)
    if ! echo "$steps" | jq -c '.[]' > "$steps_file" 2>/dev/null; then
        log::error "Failed to parse steps JSON"
        rm -f "$steps_file"
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
            
            # Execute in project root for consistency
            # Export SERVICE_PORT explicitly if it exists
            if [[ "$is_background" == "true" ]]; then
                # Use process manager for background processes
                local process_name="vrooli.${phase}.${name}"
                if command -v pm::start >/dev/null 2>&1; then
                    if pm::start "$process_name" "export SERVICE_PORT='${SERVICE_PORT:-}' && $processed_run" "$var_ROOT_DIR"; then
                        bg_processes+=("$process_name")
                        log::info "  Started background process: $process_name"
                    else
                        log::error "Failed to start background process: $process_name"
                        rm -f "$steps_file"
                        return 1
                    fi
                else
                    log::warning "Process manager not available, falling back to manual process management"
                    (cd "$var_ROOT_DIR" && export SERVICE_PORT="${SERVICE_PORT:-}" && exec bash -c "$processed_run") &
                    local bg_pid=$!
                    log::info "  Started background process (PID: $bg_pid) - manual management"
                fi
            else
                # Run without timeout wrapper to preserve terminal control for sudo
                (cd "$var_ROOT_DIR" && export SERVICE_PORT="${SERVICE_PORT:-}" && bash -c "$processed_run") || {
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
        log::info "Use 'vrooli app stop <name>' to stop services"
        log::info "Use 'vrooli app logs <name>' to view logs"
    fi
    
    log::success "Phase '$phase' completed successfully"
    return 0
}

#######################################
# Allocate dynamic ports from service.json
# Reads .ports config and allocates/exports ports
#######################################
manage::allocate_service_ports() {
    local ports_config
    ports_config=$(json::get_value '.ports // {}' '{}')
    
    [[ "$ports_config" == '{}' ]] && return 0
    
    log::info "Allocating service ports..."
    
    # Process each port configuration
    echo "$ports_config" | jq -r 'to_entries | .[] | @base64' | while IFS= read -r port_entry; do
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
                # Fallback to simple allocation if port registry not available
                for ((port=start_port; port<=end_port; port++)); do
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
    done
    
    return 0
}

#######################################
# Display help with dynamic app info
#######################################
manage::show_help() {
    # Get app info from service.json
    local app_name app_version app_desc
    
    if json::validate_config 2>/dev/null; then
        app_name=$(json::get_value '.service.displayName // .service.name' 'Application')
        app_version=$(json::get_value '.version // .service.version' 'unknown')
        app_desc=$(json::get_value '.service.description' '')
    else
        app_name="Application"
        app_version="unknown"
        app_desc=""
    fi
    
    cat << EOF
$app_name v$app_version
$([ -n "$app_desc" ] && echo "$app_desc" && echo)

USAGE:
    $0 <phase> [options]
    $0 --help | -h
    $0 --list-phases

PHASES:
    Phases are defined in .vrooli/service.json under 'lifecycle'.
    Common phases include:
    
    setup       Prepare the environment for development/production
    develop     Start the development environment  
    build       Build the application artifacts
    test        Run tests
    deploy      Deploy the application
    
    Custom phases can be defined in service.json.

OPTIONS:
    Options vary by phase. Common options include:
    
    --target <target>        Deployment target (native-linux, docker, k8s, etc.)
    --environment <env>      Environment (development, production, staging)
    --dry-run               Preview what would be executed without making changes
    --yes                   Skip confirmation prompts
    
    Run '$0 <phase> --help' for phase-specific options.

EXAMPLES:
    $0 setup --target native-linux
    $0 develop --detached yes
    $0 build --environment production --dry-run
    $0 test --coverage yes
    $0 deploy --source k8s --version 2.0.0

CONFIGURATION:
    Lifecycle phases and steps: $var_ROOT_DIR/.vrooli/service.json
    Project root: $var_ROOT_DIR

EOF
}

#######################################
# List available phases
#######################################
manage::list_phases() {
    if ! json::validate_config; then
        log::error "Cannot list phases - invalid or missing service.json"
        return 1
    fi
    
    echo "Available lifecycle phases:"
    echo
    
    local phases
    phases=$(json::list_lifecycle_phases)
    
    if [[ -z "$phases" ]]; then
        echo "  (No lifecycle phases configured)"
        return 0
    fi
    
    while IFS= read -r phase; do
        [[ -z "$phase" ]] && continue
        local description
        description=$(json::get_value ".lifecycle.${phase}.description" "")
        printf "  %-12s  %s\n" "$phase" "$description"
    done <<< "$phases"
}

#######################################
# Main execution
#######################################
manage::main() {
    local phase="${1:-}"
    
    # Check for --dry-run early
    local dry_run_flag="false"
    for arg in "$@"; do
        [[ "$arg" == "--dry-run" ]] && {
            dry_run_flag="true"
            export DRY_RUN="true"
            break
        }
    done
    
    # Handle special flags
    case "$phase" in
        --help|-h|"")
            manage::show_help
            exit 0
            ;;
        --list-phases|--list)
            manage::list_phases
            exit 0
            ;;
    esac
    
    # Validate service.json exists
    if ! json::validate_config; then
        log::error "No valid service.json found in this directory"
        echo "Create .vrooli/service.json with lifecycle configuration"
        exit 1
    fi
    
    # Validate phase exists
    if ! json::path_exists ".lifecycle.${phase}"; then
        log::error "Phase '$phase' not found in service.json"
        echo
        manage::list_phases
        echo
        echo "Run '$0 --help' for usage information"
        exit 1
    fi
    
    # Show dry-run banner
    if [[ "$dry_run_flag" == "true" ]]; then
        echo "═══════════════════════════════════════════════════════" >&2
        echo "DRY RUN MODE: No actual changes will be made" >&2
        echo "═══════════════════════════════════════════════════════" >&2
        echo >&2
    fi
    
    # Export common environment variables
    export var_ROOT_DIR
    export ENVIRONMENT="${ENVIRONMENT:-development}"
    export LOCATION="${LOCATION:-Local}"
    export TARGET="${TARGET:-docker}"
    
    # Remove phase from arguments
    shift
    
    # Execute phase directly (no more external executor!)
    [[ "$dry_run_flag" == "true" ]] && \
        log::info "[DRY RUN] Executing phase '$phase'..." || \
        log::info "Executing phase '$phase'..."
    
    manage::execute_phase "$phase" "$@"
}

# Execute
manage::main "$@"