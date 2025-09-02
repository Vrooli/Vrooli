#!/usr/bin/env bash
################################################################################
# Population System v2.0 - Main Entry Point
# Ultra-simple scenario content population for Vrooli resources
################################################################################
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
POPULATE_DIR="${APP_ROOT}/scripts/resources/populate"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Source population libraries
for lib in core content validate scenario; do
    # shellcheck disable=SC1090
    source "${POPULATE_DIR}/lib/${lib}.sh"
done

# Show usage
show_usage() {
    cat << EOF
Vrooli Population System v2.0

USAGE:
    populate.sh <path|command> [options]

PATH-BASED COMMANDS (Primary Interface):
    populate.sh .                        # Populate from current directory
    populate.sh /path/to/scenario       # Populate from specific path
    populate.sh --status                # Show population status  
    populate.sh --list                  # List available scenarios

LEGACY COMMANDS (Backwards Compatibility):
    add <scenario>      Populate resources with content from scenario
    validate <scenario> Validate scenario configuration
    list               List available scenarios
    status             Show population status
    help               Show this help message

OPTIONS:
    --dry-run          Show what would be done without executing
    --parallel         Enable parallel processing (default)
    --verbose          Show detailed output

EXAMPLES:
    populate.sh .                            # Populate from current directory (like old injection/engine.sh .)
    populate.sh . --dry-run                  # Preview population from current directory
    populate.sh --status                     # Show resource status
    populate.sh add my-scenario              # Legacy: populate from scenario name
    populate.sh validate my-scenario         # Legacy: validate scenario config

EOF
}

# Main dispatcher
main() {
    local first_arg="${1:-help}"
    
    case "$first_arg" in
        # Flag-style commands
        --status)
            shift
            populate::status "$@"
            ;;
        --list)
            shift
            populate::list "$@"
            ;;
        --help|-h|help)
            show_usage
            ;;
        # Legacy command-style (backwards compatibility)
        add)
            shift
            populate::add "$@"
            ;;
        validate)
            shift
            populate::validate "$@"
            ;;
        list)
            shift
            populate::list "$@"
            ;;
        status)
            shift
            populate::status "$@"
            ;;
        # Path-based population (new primary interface)
        ./*|/*|.)
            # This is a path - populate from it
            local path="$first_arg"
            shift
            populate::add_from_path "$path" "$@"
            ;;
        *)
            # Check if it's a valid path or scenario name
            if [[ -d "$first_arg" ]] || [[ -f "$first_arg/.vrooli/service.json" ]]; then
                local path="$first_arg"
                shift
                populate::add_from_path "$path" "$@"
            else
                # Try as scenario name for backwards compatibility
                local scenario="$first_arg"
                shift
                populate::add "$scenario" "$@"
            fi
            ;;
    esac
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi