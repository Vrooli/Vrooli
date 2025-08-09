#!/usr/bin/env bash
################################################################################
# Universal Develop Phase Handler
# 
# Handles ONLY universal development environment concerns:
# - Instance conflict resolution (via instance_manager.sh)
# - Port availability checking
# - Environment variable loading
#
# ALL service startup logic MUST be defined in service.json
# This handler does NOT look for app-specific scripts
################################################################################

set -euo pipefail

# Get script directory
LIB_LIFECYCLE_PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${LIB_LIFECYCLE_PHASES_DIR}/../../utils/var.sh"

# Source common utilities
# shellcheck disable=SC1091
source "${LIB_LIFECYCLE_PHASES_DIR}/common.sh"

# Source required libraries
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/target_matcher.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/ports.sh"

################################################################################
# Universal Development Functions
################################################################################

#######################################
# Auto-convert scenarios to standalone apps
# This function provides universal scenario auto-conversion for any app
# Returns:
#   0 on success or if no scenarios, 1 on failure
#######################################
develop::auto_convert_scenarios() {
    local scenarios_dir="${var_ROOT_DIR}/scripts/scenarios"
    local auto_converter="${scenarios_dir}/auto-converter.sh"
    
    # Check if app has scenarios
    if [[ ! -d "$scenarios_dir" ]]; then
        log::debug "No scenarios directory found, skipping auto-conversion"
        return 0
    fi
    
    # Check if auto-converter exists
    if [[ ! -f "$auto_converter" ]]; then
        log::debug "No auto-converter found at $auto_converter, skipping auto-conversion"
        return 0
    fi
    
    # Check if auto-converter is executable
    if [[ ! -x "$auto_converter" ]]; then
        log::warning "Auto-converter found but not executable: $auto_converter"
        return 0
    fi
    
    log::info "Auto-converting enabled scenarios to standalone apps..."
    
    # Run with verbose output if in debug mode
    local converter_opts=""
    if [[ "${DEBUG:-}" == "true" ]] || [[ "${VERBOSE:-}" == "true" ]]; then
        converter_opts="--verbose"
    fi
    
    # Run the auto-converter
    if "$auto_converter" $converter_opts; then
        log::success "✅ Scenario auto-conversion completed"
    else
        log::warning "⚠️  Some scenarios failed to convert (check logs above)"
        # Don't fail the develop phase for scenario conversion failures
    fi
    
    return 0
}

#######################################
# Check for port conflicts
# Arguments:
#   $1 - Service name
#   $2 - Port number
# Returns:
#   0 if port is free, 1 if occupied
#######################################
develop::check_port() {
    local service="$1"
    local port="$2"
    
    if ports::is_port_in_use "$port"; then
        log::warning "Port $port for $service is already in use"
        return 1
    fi
    return 0
}

#######################################
# Resolve port conflicts for services
# Returns:
#   0 on success
#######################################
develop::resolve_port_conflicts() {
    log::info "Checking for port conflicts..."
    
    # Define default ports (can be overridden by env vars)
    local ports=(
        "UI:${PORT_UI:-3000}"
        "Server:${PORT_SERVER:-5329}"
        "Database:${PORT_DB:-5432}"
        "Redis:${PORT_REDIS:-6379}"
        "Jobs:${PORT_JOBS:-4001}"
    )
    
    local has_conflicts=false
    for port_def in "${ports[@]}"; do
        IFS=':' read -r service port <<< "$port_def"
        if ! develop::check_port "$service" "$port"; then
            has_conflicts=true
        fi
    done
    
    if [[ "$has_conflicts" == "true" ]]; then
        log::warning "Port conflicts detected. Services may fail to start."
        log::info "You can either:"
        log::info "  1. Stop conflicting services"
        log::info "  2. Change ports in your .env file"
        log::info "  3. Use --skip-port-check flag (not recommended)"
        
        if [[ "${SKIP_PORT_CHECK:-no}" != "yes" ]]; then
            return 1
        fi
    else
        log::success "✅ All ports are available"
    fi
    
    return 0
}

#######################################
# Check for running instances
# Returns:
#   0 if no conflicts, 1 if conflicts exist
#######################################
develop::check_instances() {
    local instance_check_script="${var_LIB_SERVICE_DIR:-}/instance_manager.sh"
    
    if [[ -f "$instance_check_script" ]]; then
        # shellcheck disable=SC1090
        source "$instance_check_script"
        
        if function_exists "instance::detect_all"; then
            instance::detect_all
            
            if [[ "${INSTANCE_STATE:-none}" != "none" ]]; then
                log::warning "Detected running instances"
                return 1
            fi
        fi
    fi
    
    return 0
}

################################################################################
# Main Development Logic
################################################################################

#######################################
# Run universal development tasks
# This function handles ONLY universal concerns.
# Service startup is handled by service.json.
# Globals:
#   TARGET
#   DETACHED
#   ENVIRONMENT
#   SKIP_INSTANCE_CHECK
#   CLEAN_INSTANCES
# Returns:
#   0 on success, 1 on failure
#######################################
develop::universal::main() {
    # Initialize phase
    phase::init "Develop"
    
    # Get parameters from environment or defaults
    local target="${TARGET:-native-linux}"
    local detached="${DETACHED:-no}"
    local environment="${ENVIRONMENT:-development}"
    local skip_instance_check="${SKIP_INSTANCE_CHECK:-no}"
    local clean_instances="${CLEAN_INSTANCES:-no}"
    
    log::info "Universal develop phase starting..."
    log::debug "Parameters:"
    log::debug "  Target: $target"
    log::debug "  Detached: $detached"
    log::debug "  Environment: $environment"
    log::debug "  Skip instance check: $skip_instance_check"
    log::debug "  Clean instances: $clean_instances"
    
    # Validate and canonicalize target
    if ! canonical=$(target_matcher::match_target "$target"); then
        log::error "Invalid target: $target"
        return "$ERROR_USAGE"
    fi
    export TARGET="$canonical"
    
    # Handle clean instances request
    if flow::is_yes "$clean_instances"; then
        log::header "🧹 Cleaning up instances"
        
        local instance_script="${var_LIB_SERVICE_DIR:-}/instance_manager.sh"
        if [[ -f "$instance_script" ]]; then
            # shellcheck disable=SC1090
            source "$instance_script"
            if function_exists "instance::shutdown_all"; then
                if instance::shutdown_all; then
                    log::success "✅ All instances stopped"
                else
                    log::error "Failed to stop some instances"
                    return 1
                fi
            fi
        else
            log::info "No instance management available"
        fi
        
        return 0
    fi
    
    # Step 1: Check for running instances
    if ! flow::is_yes "$skip_instance_check"; then
        log::header "🔍 Checking for Running Instances"
        
        # Load instance manager if available
        local instance_script="${var_LIB_SERVICE_DIR:-}/instance_manager.sh"
        if [[ -f "$instance_script" ]]; then
            # shellcheck disable=SC1090
            source "$instance_script"
            
            # Use the handle_conflicts function which includes prompting
            if function_exists "instance::handle_conflicts"; then
                if ! instance::handle_conflicts "$target"; then
                    log::info "Development cancelled or conflict resolution failed"
                    return "${ERROR_INSTANCE_CONFLICT:-1}"
                fi
            else
                # Fallback to simple check
                if ! develop::check_instances; then
                    log::error "Instance conflicts detected"
                    log::info "Use --skip-instance-check to bypass this check"
                    log::info "Or use --clean-instances yes to stop all instances"
                    return "${ERROR_INSTANCE_CONFLICT:-1}"
                fi
            fi
        else
            # No instance manager available, skip check
            log::debug "Instance manager not available, skipping check"
        fi
    fi
    
    # Step 2: Resolve port conflicts
    log::header "🔌 Port Management"
    if ! develop::resolve_port_conflicts; then
        if [[ "${SKIP_PORT_CHECK:-}" != "yes" ]]; then
            log::error "Port conflicts must be resolved before continuing"
            return 1
        fi
    fi
    
    # Step 3: Run pre-develop hook
    phase::run_hook "preDevelop"
    
    # Step 4: Setup development environment
    log::header "🚀 Development Environment Setup"
    
    # Source environment file if it exists
    local env_file="${var_ENV_DEV_FILE:-}"
    if [[ -f "$env_file" ]]; then
        log::info "Loading development environment variables..."
        # shellcheck disable=SC1090
        source "$env_file"
    fi
    
    # Export parameters for service.json steps to use
    export DETACHED="$detached"
    export TARGET="$target"
    export ENVIRONMENT="$environment"
    
    # Step 5: Service startup is handled by service.json
    log::info "Development services will be started according to service.json configuration"
    log::info "The lifecycle engine will now execute the configured steps..."
    
    # The lifecycle engine (phase.sh) will execute service.json steps after this handler returns
    
    # Step 6: Universal scenario auto-conversion
    develop::auto_convert_scenarios
    
    # Step 7: Run post-develop hook
    phase::run_hook "postDevelop"
    
    # Complete phase
    phase::complete
    
    log::success "✅ Development phase universal tasks completed"
    log::info "Service startup will proceed via service.json configuration"
    
    return 0
}

#######################################
# Cleanup development environment
# Note: Actual cleanup should be defined in service.json
#######################################
develop::cleanup() {
    log::info "Cleaning up development environment..."
    
    # Run cleanup hook if defined
    phase::run_hook "cleanupDevelop"
    
    # Use instance manager for cleanup if available
    local instance_script="${var_LIB_SERVICE_DIR:-}/instance_manager.sh"
    if [[ -f "$instance_script" ]]; then
        # shellcheck disable=SC1090
        source "$instance_script"
        if function_exists "instance::shutdown_all"; then
            instance::shutdown_all
        fi
    fi
    
    log::success "✅ Cleanup complete"
}

#######################################
# Utility function check
#######################################
function_exists() {
    declare -f "$1" > /dev/null
    return $?
}

#######################################
# Entry point for direct execution
#######################################
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Check if being called by lifecycle engine
    if [[ "${LIFECYCLE_PHASE:-}" == "develop" ]]; then
        develop::universal::main "$@"
    else
        log::error "This script should be called through the lifecycle engine"
        log::info "Use: ./scripts/manage.sh develop [options]"
        exit 1
    fi
fi