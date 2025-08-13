#!/usr/bin/env bash
################################################################################
# Resource CLI Template
# 
# Base template for creating resource-specific CLI commands.
# Each resource should source this file and implement required functions.
#
# Usage:
#   source resource-cli-template.sh
#   resource_cli::init "resource-name"
#
################################################################################

set -euo pipefail

# Resource CLI framework variables
RESOURCE_CLI_VERSION="1.0.0"
RESOURCE_CLI_REGISTRY="${VROOLI_ROOT:-/root/Vrooli}/.vrooli/resource-registry"

################################################################################
# Core Functions - Must be implemented by each resource
################################################################################

# These functions should be overridden by the specific resource implementation
resource_cli::inject() {
    log::error "inject not implemented for ${RESOURCE_NAME}"
    return 1
}

resource_cli::validate() {
    log::error "validate not implemented for ${RESOURCE_NAME}"
    return 1
}

resource_cli::status() {
    log::error "status not implemented for ${RESOURCE_NAME}"
    return 1
}

resource_cli::start() {
    log::error "start not implemented for ${RESOURCE_NAME}"
    return 1
}

resource_cli::stop() {
    log::error "stop not implemented for ${RESOURCE_NAME}"
    return 1
}

resource_cli::install() {
    log::error "install not implemented for ${RESOURCE_NAME}"
    return 1
}

resource_cli::uninstall() {
    log::error "uninstall not implemented for ${RESOURCE_NAME}"
    return 1
}

################################################################################
# Framework Functions
################################################################################

# Initialize the resource CLI
resource_cli::init() {
    local resource_name="${1:-unknown}"
    export RESOURCE_NAME="$resource_name"
    export RESOURCE_CLI_COMMAND="resource-${resource_name}"
    
    # Determine paths
    export RESOURCE_DIR="${RESOURCE_DIR:-$(pwd)}"
    export RESOURCE_DATA_DIR="${VROOLI_ROOT:-/root/Vrooli}/initialization/${RESOURCE_NAME}"
    
    # Source utilities if not already sourced
    if [[ -z "${var_LOG_FILE:-}" ]]; then
        # shellcheck disable=SC1091
        source "${VROOLI_ROOT:-/root/Vrooli}/scripts/lib/utils/var.sh" 2>/dev/null || true
    fi
    
    # Ensure logging functions are available
    if ! command -v log::info &>/dev/null; then
        # shellcheck disable=SC1091
        source "${var_LOG_FILE:-${VROOLI_ROOT:-/root/Vrooli}/scripts/lib/utils/log.sh}" 2>/dev/null || {
            # Fallback logging functions if sourcing fails
            log::info() { echo "[INFO] $*"; }
            log::error() { echo "[ERROR] $*" >&2; }
            log::warning() { echo "[WARNING] $*"; }
            log::success() { echo "[SUCCESS] $*"; }
            log::header() { echo ""; echo "=== $* ==="; }
        }
    fi
}

# Register this resource with the main CLI
resource_cli::register() {
    local cli_path="${1:-}"
    local resource_name="${2:-$RESOURCE_NAME}"
    
    if [[ -z "$cli_path" ]]; then
        log::error "CLI path required for registration"
        return 1
    fi
    
    # Create registry directory
    mkdir -p "$RESOURCE_CLI_REGISTRY"
    
    # Create registry entry
    local registry_file="${RESOURCE_CLI_REGISTRY}/${resource_name}.json"
    cat > "$registry_file" << EOF
{
    "name": "${resource_name}",
    "command": "resource-${resource_name}",
    "path": "${cli_path}",
    "version": "${RESOURCE_CLI_VERSION}",
    "registered": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    log::success "Registered ${resource_name} CLI"
}

# Show help for the resource
resource_cli::show_help() {
    cat << EOF
🚀 ${RESOURCE_NAME} Resource CLI

USAGE:
    resource-${RESOURCE_NAME} <command> [options]

COMMANDS:
    inject <file>       Inject data into ${RESOURCE_NAME}
    validate            Validate ${RESOURCE_NAME} configuration
    status              Show ${RESOURCE_NAME} status
    start               Start ${RESOURCE_NAME}
    stop                Stop ${RESOURCE_NAME}
    install             Install ${RESOURCE_NAME}
    uninstall           Uninstall ${RESOURCE_NAME}
    help                Show this help message

OPTIONS:
    --verbose, -v       Show detailed output
    --dry-run           Preview actions without executing
    --force             Force operation (skip confirmations)

EXAMPLES:
    resource-${RESOURCE_NAME} status
    resource-${RESOURCE_NAME} inject workflow.json
    resource-${RESOURCE_NAME} start --verbose

For more information: https://docs.vrooli.com/resources/${RESOURCE_NAME}
EOF
}

# Main command router for resource CLI
resource_cli::main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        inject)
            resource_cli::inject "$@"
            ;;
        validate)
            resource_cli::validate "$@"
            ;;
        status)
            resource_cli::status "$@"
            ;;
        start)
            resource_cli::start "$@"
            ;;
        stop)
            resource_cli::stop "$@"
            ;;
        install)
            resource_cli::install "$@"
            ;;
        uninstall)
            resource_cli::uninstall "$@"
            ;;
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Parse common options
resource_cli::parse_options() {
    export VERBOSE=false
    export DRY_RUN=false
    export FORCE=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --force)
                FORCE=true
                shift
                ;;
            *)
                # Return remaining args
                echo "$@"
                return 0
                ;;
        esac
    done
}

# Export functions for use by resource implementations
export -f resource_cli::init
export -f resource_cli::register
export -f resource_cli::show_help
export -f resource_cli::main
export -f resource_cli::parse_options