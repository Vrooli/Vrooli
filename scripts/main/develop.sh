#!/usr/bin/env bash
set -euo pipefail
DESCRIPTION="Starts the development environment."

MAIN_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/args.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/flow.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/log.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/targetMatcher.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/var.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/exit_codes.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/develop/index.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/develop/port_manager.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/develop/instance_manager.sh"

develop::parse_arguments() {
    args::reset

    args::register_help
    args::register_sudo_mode
    args::register_yes
    args::register_location
    args::register_environment
    args::register_target
    args::register_detached
    
    # Register instance management arguments
    args::register \
        --name "skip-instance-check" \
        --flag "" \
        --desc "Skip checking for running instances" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "instance-action" \
        --flag "" \
        --desc "Pre-select instance conflict action" \
        --type "value" \
        --options "stop|keep|force" \
        --default ""
    
    args::register \
        --name "clean-instances" \
        --flag "" \
        --desc "Stop all Vrooli instances and exit" \
        --type "value" \
        --options "yes|no" \
        --default "no"

    if args::is_asking_for_help "$@"; then
        args::usage "$DESCRIPTION"
        exit_codes::print
        exit 0
    fi

    args::parse "$@"

    export TARGET=$(args::get "target")
    export SUDO_MODE=$(args::get "sudo-mode")
    export YES=$(args::get "yes")
    export LOCATION=$(args::get "location")
    export ENVIRONMENT=$(args::get "environment")
    export DETACHED=$(args::get "detached")
    export SKIP_INSTANCE_CHECK=$(args::get "skip-instance-check")
    export INSTANCE_ACTION=$(args::get "instance-action")
    export CLEAN_INSTANCES=$(args::get "clean-instances")
}

develop::main() {
    develop::parse_arguments "$@"
    
    # Export parsed arguments so they survive the subshell
    export TARGET SUDO_MODE YES LOCATION ENVIRONMENT DETACHED SKIP_INSTANCE_CHECK INSTANCE_ACTION CLEAN_INSTANCES
    
    # Handle --clean-instances flag
    if flow::is_yes "$CLEAN_INSTANCES"; then
        log::header "🧹 Cleaning up Vrooli instances"
        instance::detect_all
        instance::display_status
        
        if [[ "$INSTANCE_STATE" != "none" ]]; then
            log::info "Stopping all Vrooli instances..."
            if instance::shutdown_all; then
                log::success "All instances stopped successfully"
            else
                log::error "Failed to stop some instances"
                exit 1
            fi
        else
            log::info "No instances to clean up"
        fi
        exit 0
    fi
    
    # Validate target and canonicalize
    if ! canonical=$(match_target "$TARGET"); then
        args::usage "$DESCRIPTION"
        exit "$ERROR_USAGE"
    fi
    export TARGET="$canonical"
    log::header "🏃 Starting development environment for $(match_target "$TARGET")"

    # Save the target before setup overwrites it
    local saved_target="$TARGET"
    
    # Run setup in a subshell to prevent it from exiting our script
    log::info "Running setup..."
    if ! bash "${MAIN_DIR}/setup.sh" "$@"; then
        log::error "Setup failed"
        exit 1
    fi
    
    # Restore the correct target
    export TARGET="$saved_target"
    
    # Re-source all necessary utilities after setup completes
    # (setup was run in a subshell, so we lost our functions)
    source "${MAIN_DIR}/../helpers/utils/var.sh"
    source "${MAIN_DIR}/../helpers/utils/log.sh"
    source "${MAIN_DIR}/../helpers/utils/flow.sh"
    source "${MAIN_DIR}/../helpers/develop/index.sh"
    source "${MAIN_DIR}/../helpers/develop/port_manager.sh"
    source "${MAIN_DIR}/../helpers/develop/instance_manager.sh"
    
    # Source environment file directly to get variables
    if [[ -f "${var_ROOT_DIR}/.env-dev" ]]; then
        source "${var_ROOT_DIR}/.env-dev"
    fi
    
    # Source environment utilities
    if [[ -f "${MAIN_DIR}/../helpers/utils/env.sh" ]]; then
        source "${MAIN_DIR}/../helpers/utils/env.sh"
    fi
    
    # Source proxy utilities if needed for remote locations
    if [[ -f "${MAIN_DIR}/../helpers/utils/proxy.sh" ]]; then
        source "${MAIN_DIR}/../helpers/utils/proxy.sh"
    fi
    
    # Check for existing instances and handle conflicts
    if ! instance::should_skip_check; then
        local instance_result
        instance::handle_conflicts "$TARGET" "${INSTANCE_ACTION:-}"
        instance_result=$?
        log::debug "Instance conflict check returned: $instance_result"
        
        if [[ $instance_result -eq 2 ]]; then
            # User chose to keep existing instances and exit
            exit 0
        elif [[ $instance_result -ne 0 ]]; then
            log::error "Failed to handle instance conflicts"
            exit "${ERROR_INSTANCE_CONFLICT:-1}"
        fi
    else
        log::debug "Skipping instance check"
    fi
    
    # Resolve port conflicts before starting services
    develop::resolve_port_conflicts "$TARGET"

    if env::is_location_remote; then
        proxy::setup
        if ! flow::is_yes "$DETACHED"; then
            trap 'info "Tearing down Caddy reverse proxy..."; stop_reverse_proxy' EXIT INT TERM
        fi
    fi

    # Run the target-specific develop script
    TARGET_SCRIPT="${MAIN_DIR}/../helpers/develop/target/${TARGET//-/_}.sh"
    if [[ ! -f "$TARGET_SCRIPT" ]]; then
        log::error "Target develop script not found: $TARGET_SCRIPT"
        exit "${ERROR_FUNCTION_NOT_FOUND}"
    fi
    
    # Execute the target script and capture its exit code
    bash "$TARGET_SCRIPT" "$@"
    local exit_code=$?
    
    # Only show success message if the script completed successfully
    if [[ $exit_code -eq 0 ]]; then
        log::success "✅ Development environment started."
    elif [[ $exit_code -eq "$EXIT_USER_INTERRUPT" ]]; then
        log::info "Development environment stopped by user."
    else
        log::error "Development environment exited with error code: $exit_code"
    fi
    
    exit $exit_code 
}

develop::main "$@"
