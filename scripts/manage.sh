#!/usr/bin/env bash
set -euo pipefail

#######################################
# Application Management Script
# 
# Single entry point for all lifecycle operations.
# All behavior is controlled by .vrooli/service.json
#######################################

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)}"
MAIN_SCRIPT_DIR="${APP_ROOT}/scripts"
export MAIN_SCRIPT_DIR

# Source utilities
# shellcheck disable=SC1091
source "$MAIN_SCRIPT_DIR/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$MAIN_SCRIPT_DIR/lib/utils/json.sh"
# shellcheck disable=SC1091
source "${var_PORT_REGISTRY_FILE}" 2>/dev/null || true
# shellcheck disable=SC1091
source "$MAIN_SCRIPT_DIR/lib/service/secrets.sh"
# shellcheck disable=SC1091
source "$MAIN_SCRIPT_DIR/lib/process-manager.sh"

# Source new library modules
# shellcheck disable=SC1091
source "$MAIN_SCRIPT_DIR/lib/utils/git.sh"
# shellcheck disable=SC1091
source "$MAIN_SCRIPT_DIR/lib/utils/setup.sh"
# shellcheck disable=SC1091
source "$MAIN_SCRIPT_DIR/lib/utils/lifecycle.sh"
# shellcheck disable=SC1091
source "$MAIN_SCRIPT_DIR/lib/utils/help.sh"

#######################################
# Setup State Management Infrastructure
# Provides git-aware setup tracking and fast mode support
#######################################

#######################################
# Get current git commit hash (thin wrapper)
#######################################
manage::get_git_commit() {
    git::get_commit
}


#######################################
# Check if setup artifacts exist and are valid (thin wrapper)
#######################################
manage::verify_setup_artifacts() {
    setup::verify_artifacts
}

#######################################
# Check if app needs setup (thin wrapper)
#######################################
manage::needs_setup() {
    setup::is_needed
}

#######################################
# Get completed setup steps from service.json (thin wrapper)
#######################################
manage::get_setup_steps_list() {
    setup::get_steps_list
}

#######################################
# Mark setup as complete with current git state (thin wrapper)
#######################################
manage::mark_setup_complete() {
    setup::mark_complete
}

#######################################
# Enhanced develop lifecycle with conditional setup (thin wrapper)
#######################################
manage::develop_with_auto_setup() {
    lifecycle::develop_with_auto_setup "$@"
}

#######################################
# Execute lifecycle phase (thin wrapper)
#######################################
manage::execute_phase() {
    lifecycle::execute_phase "$@"
}

#######################################
# Allocate dynamic ports from service.json (thin wrapper)
#######################################
manage::allocate_service_ports() {
    lifecycle::allocate_service_ports
}

#######################################
# Display help with dynamic app info (thin wrapper)
#######################################
manage::show_help() {
    help::show_app_help "$0"
}

#######################################
# List available phases (thin wrapper)
#######################################
manage::list_phases() {
    help::list_phases
}

#######################################
# Main execution
#######################################
manage::main() {
    local phase="${1:-}"
    
    # Check for help flags ANYWHERE in arguments
    for arg in "$@"; do
        case "$arg" in
            --help|-h)
                manage::show_help
                exit 0
                ;;
        esac
    done
    
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
        "")
            manage::show_help
            exit 0
            ;;
        --list-phases|--list)
            manage::list_phases
            exit 0
            ;;
    esac
    
    # Robust scenario directory detection
    if [[ -f "${PWD}/.vrooli/service.json" ]]; then
        # Check if we're in a scenario directory by looking at parent paths
        if [[ "${PWD}" == */scenarios/* ]]; then
            SCENARIO_NAME="$(basename "${PWD}")"
            export SCENARIO_NAME
            export SCENARIO_MODE=true
            export SCENARIO_PATH="${PWD}"
            log::info "Running in scenario mode: $SCENARIO_NAME"
        fi
    fi
    
    # Validate service.json exists
    if ! json::validate_config; then
        log::error "No valid service.json found in this directory"
        echo "Create .vrooli/service.json with lifecycle configuration"
        exit 1
    fi
    
    # Adjust service.json path for scenarios
    if [[ "${SCENARIO_MODE:-false}" == "true" ]]; then
        SERVICE_JSON_PATH="${PWD}/.vrooli/service.json"
        
        # Scenario-specific process manager paths
        export PM_HOME="${HOME}/.vrooli/processes/scenarios/${SCENARIO_NAME}"
        export PM_LOG_DIR="${HOME}/.vrooli/logs/scenarios/${SCENARIO_NAME}"
        
        # Source port registry for resource ports
        source "${var_ROOT_DIR}/scripts/resources/port-registry.sh" 2>/dev/null || true
    else
        SERVICE_JSON_PATH="${APP_ROOT}/.vrooli/service.json"
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
    
    # Route develop phase through auto-setup logic
    if [[ "$phase" == "develop" ]]; then
        manage::develop_with_auto_setup "$phase" "$@"
    else
        manage::execute_phase "$phase" "$@"
    fi
}

# Execute
manage::main "$@"