#!/usr/bin/env bash
#######################################
# Lifecycle Management Library
# Provides lifecycle phase execution and service orchestration
#######################################

set -euo pipefail

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
    if setup::is_needed; then
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
        
        # Run setup phase with original arguments
        lifecycle::execute_phase "setup" "$@" || {
            log::error "Setup failed, cannot start develop mode"
            return 1
        }
        
        # Mark setup as complete
        setup::mark_complete
        log::success "Setup completed, proceeding with develop"
        
        # Auto-refresh embeddings on git changes
        embeddings::refresh_on_changes
    else
        log::info "✓ Setup is current, proceeding directly to develop"
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
# Returns:
#   0 on success, 1 on failure
#######################################
lifecycle::allocate_service_ports() {
    local ports_config
    ports_config=$(json::get_value '.ports // {}' '{}')
    
    [[ "$ports_config" == '{}' ]] && return 0
    
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
    done < <(echo "$ports_config" | jq -r 'to_entries | .[] | @base64')
    
    return 0
}