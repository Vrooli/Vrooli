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
    populate.sh <command> [options]

COMMANDS:
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
    populate.sh add my-scenario              # Populate from scenario
    populate.sh add my-scenario --dry-run    # Preview population
    populate.sh validate my-scenario         # Validate scenario config
    populate.sh list                         # Show available scenarios

EOF
}

# Main dispatcher
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        add)
            populate::add "$@"
            ;;
        validate)
            populate::validate "$@"
            ;;
        list)
            populate::list "$@"
            ;;
        status)
            populate::status "$@"
            ;;
        help|--help|-h)
            show_usage
            ;;
        *)
            log::error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi