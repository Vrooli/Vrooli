#!/usr/bin/env bash
################################################################################
# Injection System v2.0 - Main Entry Point
# Ultra-simple scenario content injection for Vrooli resources
################################################################################
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
INJECTION_DIR="${APP_ROOT}/scripts/resources/injection"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Source injection libraries
for lib in core content validate scenario; do
    # shellcheck disable=SC1090
    source "${INJECTION_DIR}/lib/${lib}.sh"
done

# Show usage
show_usage() {
    cat << EOF
Vrooli Injection System v2.0

USAGE:
    inject.sh <command> [options]

COMMANDS:
    add <scenario>      Add content from scenario to resources
    validate <scenario> Validate scenario configuration
    list               List available scenarios
    status             Show injection status
    help               Show this help message

OPTIONS:
    --dry-run          Show what would be done without executing
    --parallel         Enable parallel processing (default)
    --verbose          Show detailed output

EXAMPLES:
    inject.sh add my-scenario              # Inject content from scenario
    inject.sh add my-scenario --dry-run    # Preview injection
    inject.sh validate my-scenario         # Validate scenario config
    inject.sh list                         # Show available scenarios

EOF
}

# Main dispatcher
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        add)
            inject::add "$@"
            ;;
        validate)
            inject::validate "$@"
            ;;
        list)
            inject::list "$@"
            ;;
        status)
            inject::status "$@"
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