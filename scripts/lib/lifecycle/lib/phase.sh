#!/usr/bin/env bash
# Lifecycle Engine - Phase Orchestration Module
# Orchestrates lifecycle phase execution

set -euo pipefail

# Get script directory
LIB_LIFECYCLE_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "$LIB_LIFECYCLE_LIB_DIR/../../utils/var.sh"

# Guard against re-sourcing
[[ -n "${_PHASE_MODULE_LOADED:-}" ]] && return 0
declare -gr _PHASE_MODULE_LOADED=1

# Source dependencies
# shellcheck source=./config.sh
source "$var_LIB_LIFECYCLE_DIR/lib/config.sh"
# shellcheck source=./targets.sh
source "$var_LIB_LIFECYCLE_DIR/lib/targets.sh"
# shellcheck source=./parallel.sh
source "$var_LIB_LIFECYCLE_DIR/lib/parallel.sh"
# shellcheck source=./condition.sh
source "$var_LIB_LIFECYCLE_DIR/lib/condition.sh"

# Phase execution state
declare -g PHASE_START_TIME=""
declare -g PHASE_END_TIME=""
declare -g PHASE_RESULT=0

#######################################
# Execute a lifecycle phase
# Arguments:
#   $1 - Phase name
#   $2 - Target (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
phase::execute() {
    local phase_name="${1:-$LIFECYCLE_PHASE}"
    local target="${2:-${TARGET:-}}"
    
    # Load phase configuration
    if ! config::get_phase "$phase_name"; then
        phase::log_error "Phase not found: $phase_name"
        phase::log_info "Available phases:"
        config::list_phases | sed 's/^/  - /'
        return 1
    fi
    
    # Display phase header
    phase::show_header "$phase_name" "$target"
    
    # Record start time
    PHASE_START_TIME=$(date +%s)
    
    # Check requirements
    if ! phase::check_requirements; then
        phase::log_error "Requirements not met for phase: $phase_name"
        return 1
    fi
    
    # Prompt for confirmation if needed
    if ! phase::confirm_execution; then
        phase::log_info "Phase execution cancelled by user"
        return 0
    fi
    
    # Set up phase environment
    phase::setup_environment
    
    # Execute phase hooks
    phase::execute_hook "pre" || return 1
    
    # Execute main phase steps
    local result=0
    if ! phase::execute_steps "$target"; then
        result=1
        PHASE_RESULT=1
    fi
    
    # Execute post hooks (always run, even on failure)
    phase::execute_hook "post"
    
    # Record end time
    PHASE_END_TIME=$(date +%s)
    
    # Display phase footer
    phase::show_footer "$phase_name" $result
    
    return $result
}

#######################################
# Display phase execution header
# Arguments:
#   $1 - Phase name
#   $2 - Target (optional)
#######################################
phase::show_header() {
    local phase_name="$1"
    local target="${2:-}"
    
    local description
    description=$(config::get_phase_metadata "description")
    
    echo "═══════════════════════════════════════════════════════════════" >&2
    echo "Lifecycle Phase: $phase_name" >&2
    [[ -n "$description" ]] && echo "Description: $description" >&2
    [[ -n "$target" ]] && echo "Target: $target" >&2
    [[ "${DRY_RUN:-false}" == "true" ]] && echo "Mode: DRY RUN - No actual changes will be made" >&2
    echo "═══════════════════════════════════════════════════════════════" >&2
}

#######################################
# Display phase execution footer
# Arguments:
#   $1 - Phase name
#   $2 - Exit code
#######################################
phase::show_footer() {
    local phase_name="$1"
    local exit_code="$2"
    
    local duration=0
    if [[ -n "$PHASE_START_TIME" ]] && [[ -n "$PHASE_END_TIME" ]]; then
        duration=$((PHASE_END_TIME - PHASE_START_TIME))
    fi
    
    echo "═══════════════════════════════════════════════════════════════" >&2
    
    if [[ $exit_code -eq 0 ]]; then
        echo "✓ Phase completed successfully: $phase_name (${duration}s)" >&2
    else
        echo "✗ Phase failed: $phase_name (${duration}s)" >&2
    fi
    
    echo "═══════════════════════════════════════════════════════════════" >&2
}

#######################################
# Check phase requirements
# Returns:
#   0 if requirements met, 1 otherwise
#######################################
phase::check_requirements() {
    local requires
    requires=$(config::get_phase_metadata "requires")
    
    if [[ -z "$requires" ]] || [[ "$requires" == "null" ]]; then
        return 0
    fi
    
    phase::log_info "Checking requirements..."
    
    # Parse requirements array
    local req_count
    req_count=$(echo "$requires" | jq 'length' 2>/dev/null || echo 0)
    
    if [[ $req_count -eq 0 ]]; then
        # Single requirement as string
        phase::check_single_requirement "$requires"
    else
        # Multiple requirements
        local i=0
        while [[ $i -lt $req_count ]]; do
            local req
            req=$(echo "$requires" | jq -r ".[$i]")
            if ! phase::check_single_requirement "$req"; then
                return 1
            fi
            ((i++))
        done
    fi
    
    return 0
}

#######################################
# Check single requirement
# Arguments:
#   $1 - Requirement specification
# Returns:
#   0 if met, 1 otherwise
#######################################
phase::check_single_requirement() {
    local req="$1"
    
    case "$req" in
        command:*)
            local cmd="${req#command:}"
            if ! command -v "$cmd" >/dev/null 2>&1; then
                phase::log_error "Required command not found: $cmd"
                return 1
            fi
            ;;
        file:*)
            local file="${req#file:}"
            if [[ ! -f "$file" ]]; then
                phase::log_error "Required file not found: $file"
                return 1
            fi
            ;;
        dir:*)
            local dir="${req#dir:}"
            if [[ ! -d "$dir" ]]; then
                phase::log_error "Required directory not found: $dir"
                return 1
            fi
            ;;
        phase:*)
            local phase="${req#phase:}"
            # Check if phase was previously completed (would need state tracking)
            phase::log_info "Requires prior phase: $phase"
            ;;
        *)
            # Treat as command by default
            if ! command -v "$req" >/dev/null 2>&1; then
                phase::log_error "Required command not found: $req"
                return 1
            fi
            ;;
    esac
    
    return 0
}

#######################################
# Prompt for execution confirmation
# Returns:
#   0 to continue, 1 to abort
#######################################
phase::confirm_execution() {
    local confirm
    confirm=$(config::get_phase_metadata "confirm")
    
    if [[ -z "$confirm" ]] || [[ "$confirm" == "null" ]]; then
        return 0
    fi
    
    # Evaluate condition
    if ! condition::evaluate "$confirm"; then
        # No confirmation needed
        return 0
    fi
    
    # Skip confirmation in dry-run mode
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        return 0
    fi
    
    # Skip confirmation if non-interactive
    if [[ ! -t 0 ]] || [[ "${CI:-false}" == "true" ]]; then
        phase::log_warning "Non-interactive mode, skipping confirmation"
        return 0
    fi
    
    phase::log_warning "This action requires confirmation"
    read -rp "Continue? (y/N): " response
    
    if [[ "$response" =~ ^[Yy]$ ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Set up phase environment
#######################################
phase::setup_environment() {
    # Export default values
    local timeout shell error
    timeout=$(config::get_default "timeout")
    shell=$(config::get_default "shell")
    error=$(config::get_default "error")
    
    [[ -n "$timeout" ]] && export LIFECYCLE_TIMEOUT="${LIFECYCLE_TIMEOUT:-$timeout}"
    [[ -n "$shell" ]] && export LIFECYCLE_SHELL="${LIFECYCLE_SHELL:-$shell}"
    [[ -n "$error" ]] && export LIFECYCLE_ERROR="${LIFECYCLE_ERROR:-$error}"
    
    # Export phase-specific environment
    local phase_env
    phase_env=$(config::get_phase_metadata "env")
    
    if [[ -n "$phase_env" ]] && [[ "$phase_env" != "{}" ]] && [[ "$phase_env" != "null" ]]; then
        while IFS= read -r key && IFS= read -r value; do
            export "$key=$value"
        done < <(echo "$phase_env" | jq -r 'to_entries[] | .key, .value')
    fi
    
    # Export target-specific environment if target is set
    if [[ -n "${TARGET:-}" ]]; then
        local target_env
        target_env=$(targets::get_env "$TARGET")
        
        if [[ -n "$target_env" ]]; then
            while IFS= read -r env_pair; do
                export "$env_pair"
            done <<< "$target_env"
        fi
    fi
}

#######################################
# Execute phase hook
# Arguments:
#   $1 - Hook type (pre, post)
# Returns:
#   0 on success, 1 on failure
#######################################
phase::execute_hook() {
    local hook_type="$1"
    
    local hook_steps
    hook_steps=$(config::get_phase_steps "${hook_type}_steps")
    
    if [[ "$hook_steps" == "[]" ]] || [[ "$hook_steps" == "null" ]]; then
        return 0
    fi
    
    phase::log_info "Executing ${hook_type}-phase hooks..."
    
    if ! parallel::execute_sequential "$hook_steps"; then
        phase::log_error "${hook_type^}-phase hooks failed"
        return 1
    fi
    
    return 0
}

#######################################
# Execute phase steps
# Arguments:
#   $1 - Target (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
phase::execute_steps() {
    local target="${1:-}"
    local phase_name="${LIFECYCLE_PHASE}"
    
    # Check for universal phase handler first
    local universal_handler="$var_LIB_LIFECYCLE_DIR/phases/${phase_name}.sh"
    if [[ -f "$universal_handler" ]]; then
        phase::log_info "Executing universal phase handler: $phase_name"
        
        # Export phase name for handler to use
        export LIFECYCLE_PHASE="$phase_name"
        
        # Execute the universal handler
        if bash "$universal_handler"; then
            phase::log_success "Universal phase handler completed successfully"
        else
            phase::log_error "Universal phase handler failed"
            return 1
        fi
        
        # Universal handler may have already executed app-specific logic
        # Check if we should continue with service.json steps
        if [[ "${SKIP_SERVICE_JSON_STEPS:-no}" == "yes" ]]; then
            return 0
        fi
    fi
    
    # Execute steps from service.json if defined
    local universal_steps
    universal_steps=$(config::get_phase_steps "steps")
    
    if [[ "$universal_steps" != "[]" ]] && [[ "$universal_steps" != "null" ]]; then
        phase::log_info "Executing service.json defined steps..."
        
        if ! parallel::execute_sequential "$universal_steps"; then
            phase::log_error "Service.json steps failed"
            return 1
        fi
    fi
    
    # Execute target-specific steps if target is specified
    if [[ -n "$target" ]]; then
        local target_config
        target_config=$(targets::get_config "$target")
        
        if [[ "$target_config" != "{}" ]]; then
            local target_steps
            target_steps=$(echo "$target_config" | jq '.steps // []')
            
            if [[ "$target_steps" != "[]" ]]; then
                phase::log_info "Executing target-specific steps for: $target"
                
                if ! parallel::execute_sequential "$target_steps"; then
                    phase::log_error "Target steps failed"
                    return 1
                fi
            fi
        fi
    elif [[ $(config::get_targets) != "{}" ]]; then
        # No target specified but targets exist
        phase::log_warning "No target specified. Available targets:"
        targets::list | sed 's/^/  - /'
        
        # Check for default target
        if config::target_exists "default"; then
            phase::log_info "Using default target configuration"
            phase::execute_steps "default"
            return $?
        else
            phase::log_info "Use --target <name> to specify target"
        fi
    fi
    
    return 0
}

#######################################
# Execute sub-phase
# Arguments:
#   $1 - Sub-phase name
# Returns:
#   0 on success, 1 on failure
#######################################
phase::execute_subphase() {
    local subphase="$1"
    
    phase::log_info "Executing sub-phase: $subphase"
    
    # Temporarily save current phase
    local saved_phase="$LIFECYCLE_PHASE"
    
    # Execute sub-phase
    LIFECYCLE_PHASE="$subphase"
    phase::execute "$subphase"
    local result=$?
    
    # Restore phase
    LIFECYCLE_PHASE="$saved_phase"
    
    return $result
}

#######################################
# Get phase execution statistics
#######################################
phase::get_stats() {
    echo "Phase Execution Statistics:" >&2
    echo "  Phase: ${LIFECYCLE_PHASE:-unknown}" >&2
    echo "  Target: ${TARGET:-none}" >&2
    
    if [[ -n "$PHASE_START_TIME" ]] && [[ -n "$PHASE_END_TIME" ]]; then
        local duration=$((PHASE_END_TIME - PHASE_START_TIME))
        echo "  Duration: ${duration}s" >&2
    fi
    
    echo "  Result: ${PHASE_RESULT}" >&2
}

#######################################
# Logging functions
#######################################
phase::log_info() {
    echo "[INFO] $*" >&2
}

phase::log_success() {
    echo "[SUCCESS] $*" >&2
}

phase::log_warning() {
    echo "[WARNING] $*" >&2
}

phase::log_error() {
    echo "[ERROR] $*" >&2
}

# If sourced for testing, don't auto-execute
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script should be sourced, not executed directly" >&2
    exit 1
fi