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
# Main execution
#######################################
manage::main() {
    local phase="${1:-}"
    
    # Check for help flags ANYWHERE in arguments
    for arg in "$@"; do
        case "$arg" in
            --help|-h)
                help::show_app_help "$0"
                exit 0
                ;;
        esac
    done
    
    # Check for --dry-run early and capture sudo-mode preference
    local dry_run_flag="false"
    local sudo_mode_arg=""
    local expecting_sudo_value="false"
    for arg in "$@"; do
        if [[ "$expecting_sudo_value" == "true" ]]; then
            sudo_mode_arg="$arg"
            expecting_sudo_value="false"
            continue
        fi

        case "$arg" in
            --dry-run)
                dry_run_flag="true"
                export DRY_RUN="true"
                ;;
            --sudo-mode)
                expecting_sudo_value="true"
                ;;
            --sudo-mode=*)
                sudo_mode_arg="${arg#*=}"
                ;;
        esac
    done

    if [[ "$expecting_sudo_value" == "true" ]]; then
        log::error "Missing value for --sudo-mode"
        exit 1
    fi

    if [[ -n "$sudo_mode_arg" ]]; then
        sudo_mode_arg="${sudo_mode_arg,,}"
        case "$sudo_mode_arg" in
            ask|skip|error)
                export SUDO_MODE="$sudo_mode_arg"
                export SUDO_MODE_EXPLICIT="$sudo_mode_arg"
                ;;
            *)
                log::error "Invalid value for --sudo-mode: $sudo_mode_arg"
                exit 1
                ;;
        esac
    fi
    
    # Handle special flags
    case "$phase" in
        "")
            help::show_app_help "$0"
            exit 0
            ;;
        --list-phases|--list)
            help::list_phases
            exit 0
            ;;
    esac
    
    # Scenario detection - check if argument is a scenario name
    local scenario_arg=""
    for arg in "$@"; do
        if [[ "$arg" != --* && "$arg" != "$phase" ]]; then
            scenario_arg="$arg"
            break
        fi
    done
    
    # If we have a potential scenario name argument, try to find it
    if [[ -n "$scenario_arg" ]]; then
        local scenarios_dir="${VROOLI_ROOT}/scenarios"
        local scenario_path="${scenarios_dir}/${scenario_arg}"
        
        if [[ -d "$scenario_path" && -f "${scenario_path}/.vrooli/service.json" ]]; then
            SCENARIO_NAME="$scenario_arg"
            export SCENARIO_NAME
            export SCENARIO_MODE=true
            export SCENARIO_PATH="$scenario_path"
            log::info "Running in scenario mode: $SCENARIO_NAME"
            # Change to scenario directory for lifecycle execution
            cd "$scenario_path"
        fi
    fi
    
    # Fallback: Robust scenario directory detection (current behavior)
    if [[ -z "${SCENARIO_MODE:-}" && -f "${PWD}/.vrooli/service.json" ]]; then
        # Check if we're in a scenario directory by looking at parent paths
        if [[ "${PWD}" == */scenarios/* ]]; then
            SCENARIO_NAME="$(basename "${PWD}")"
            export SCENARIO_NAME
            export SCENARIO_MODE=true
            export SCENARIO_PATH="${PWD}"
            log::info "Running in scenario mode: $SCENARIO_NAME"
        fi
    fi
    
    # Set service.json path BEFORE validation
    if [[ "${SCENARIO_MODE:-false}" == "true" ]]; then
        SERVICE_JSON_PATH="${PWD}/.vrooli/service.json"
        export SERVICE_JSON_PATH
        
        # Scenario-specific process manager paths
        export PM_HOME="${HOME}/.vrooli/processes/scenarios/${SCENARIO_NAME}"
        export PM_LOG_DIR="${HOME}/.vrooli/logs/scenarios/${SCENARIO_NAME}"
        
        # Source port registry for resource ports
        source "${var_ROOT_DIR}/scripts/resources/port_registry.sh" 2>/dev/null || true
    else
        SERVICE_JSON_PATH="${APP_ROOT}/.vrooli/service.json"
        export SERVICE_JSON_PATH
    fi
    
    # Validate service.json exists - now uses correct path
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
    
    # Route develop phase through auto-setup logic
    if [[ "$phase" == "develop" ]]; then
        lifecycle::develop_with_auto_setup "$phase" "$@"
    else
        lifecycle::execute_phase "$phase" "$@"
    fi
}

# Execute
manage::main "$@"
