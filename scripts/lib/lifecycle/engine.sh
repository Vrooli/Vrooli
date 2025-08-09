#!/usr/bin/env bash
################################################################################
# Vrooli Lifecycle Engine
# 
# Main orchestrator for service.json lifecycle configurations.
# Uses modular components for maintainability and testability.
#
# Usage:
#   ./engine.sh <phase> [options]
#   ./engine.sh setup --target native-linux
#   ./engine.sh develop --target docker --detached yes
#   ./engine.sh build --target k8s --version 2.0.0
#
# See ./engine.sh --help for full options
################################################################################

set -euo pipefail

# Source var.sh with relative path first
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_LIFECYCLE_DIR}/lib/parser.sh"
# shellcheck disable=SC1091
source "${var_LIB_LIFECYCLE_DIR}/lib/config.sh"
# shellcheck disable=SC1091
source "${var_LIB_LIFECYCLE_DIR}/lib/condition.sh"
# shellcheck disable=SC1091
source "${var_LIB_LIFECYCLE_DIR}/lib/output.sh"
# shellcheck disable=SC1091
source "${var_LIB_LIFECYCLE_DIR}/lib/executor.sh"
# shellcheck disable=SC1091
source "${var_LIB_LIFECYCLE_DIR}/lib/parallel.sh"
# shellcheck disable=SC1091
source "${var_LIB_LIFECYCLE_DIR}/lib/targets.sh"
# shellcheck disable=SC1091
source "${var_LIB_LIFECYCLE_DIR}/lib/phase.sh"

# Global exit code
declare -g EXIT_CODE=0

################################################################################
# Main Functions
################################################################################

#######################################
# Initialize the lifecycle engine
#######################################
engine::init() {
    log::debug "Initializing lifecycle engine..."
    
    # Initialize output management
    output::init
    
    # Set up signal handlers
    trap engine::cleanup EXIT INT TERM
    
    log::debug "Engine initialized"
}

#######################################
# Load and validate configuration
# Returns:
#   0 on success, 1 on error
#######################################
engine::load_config() {
    log::debug "Loading configuration..."
    
    # Find service.json
    local config_path
    if ! config_path=$(config::find_service_json); then
        log::error "No service.json found"
        log::info "Searched locations:"
        for path in "${CONFIG_SEARCH_PATHS[@]}"; do
            echo "  - $path" >&2
        done
        return 1
    fi
    
    log::debug "Found configuration: $config_path"
    
    # Load configuration file
    if ! config::load_file "$config_path"; then
        return 1
    fi
    
    # Extract lifecycle configuration
    if ! config::extract_lifecycle; then
        return 1
    fi
    
    # Validate schema
    if ! config::validate_schema; then
        return 1
    fi
    
    log::success "Configuration loaded successfully"
    return 0
}

#######################################
# Execute the lifecycle phase
# Returns:
#   0 on success, 1 on failure
#######################################
engine::execute() {
    log::debug "Executing phase: $LIFECYCLE_PHASE"
    
    # Load phase configuration first (needed for target validation)
    if ! config::get_phase "$LIFECYCLE_PHASE"; then
        log::error "Phase not found: $LIFECYCLE_PHASE"
        log::info "Available phases:"
        config::list_phases | sed 's/^/  - /'
        return 1
    fi
    
    # Validate target if specified (now that PHASE_CONFIG is loaded)
    if [[ -n "${TARGET:-}" ]]; then
        if ! targets::validate "$TARGET"; then
            return 1
        fi
    fi
    
    # Execute the phase
    if ! phase::execute "$LIFECYCLE_PHASE" "$TARGET"; then
        EXIT_CODE=1
        return 1
    fi
    
    return 0
}

#######################################
# Cleanup on exit
#######################################
engine::cleanup() {
    log::debug "Cleaning up..."
    
    # Kill any remaining background processes
    parallel::kill_all
    
    # Clean up output management
    output::cleanup
    
    # Clear caches
    targets::clear_cache
    
    log::debug "Cleanup complete"
}

#######################################
# Display available phases
#######################################
engine::list_phases() {
    echo "Available lifecycle phases:" >&2
    config::list_phases | sed 's/^/  - /' >&2
}

#######################################
# Display engine version
#######################################
engine::version() {
    echo "Vrooli Lifecycle Engine v2.0.0" >&2
    echo "Modular architecture for improved maintainability" >&2
}

################################################################################
# Main Execution
################################################################################

main() {
    # Initialize engine
    engine::init
    
    # Parse command line arguments
    if ! parser::parse_args "$@"; then
        exit 1
    fi
    
    # Validate arguments
    if ! parser::validate; then
        exit 1
    fi
    
    # Export parsed values
    parser::export
    
    # Handle special commands
    case "$LIFECYCLE_PHASE" in
        --version|version)
            engine::version
            exit 0
            ;;
        --list|list)
            engine::load_config || exit 1
            engine::list_phases
            exit 0
            ;;
    esac
    
    # Load configuration
    if ! engine::load_config; then
        exit 1
    fi
    
    # Export configuration
    config::export
    
    # Execute lifecycle phase
    if ! engine::execute; then
        exit 1
    fi
    
    exit $EXIT_CODE
}

# Only run main if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi