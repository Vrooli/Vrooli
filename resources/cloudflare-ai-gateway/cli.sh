#!/bin/bash
# Cloudflare AI Gateway Resource CLI
# Provides resilient AI traffic proxy with caching, rate limiting, and analytics

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/cloudflare-ai-gateway"
RESOURCE_NAME="cloudflare-ai-gateway"

# Source required libraries
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework.sh"
source "${SCRIPT_DIR}/lib/gateway.sh"
source "${SCRIPT_DIR}/lib/config.sh"
source "${SCRIPT_DIR}/lib/providers.sh"

# Main command handler
main() {
    case "${1:-}" in
        start)
            shift
            gateway_start "$@"
            ;;
        stop)
            shift
            gateway_stop "$@"
            ;;
        status)
            shift
            gateway_status "$@"
            ;;
        info)
            shift
            gateway_info "$@"
            ;;
        logs)
            shift
            gateway_logs "$@"
            ;;
        content)
            shift
            handle_content_command "$@"
            ;;
        inject)
            # Temporary backwards compatibility
            shift
            echo "Warning: 'inject' is deprecated. Use 'content add' instead." >&2
            handle_content_add "$@"
            ;;
        configure)
            shift
            gateway_configure "$@"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo "Error: Unknown command '${1:-}'" >&2
            show_help
            exit 1
            ;;
    esac
}

# Handle content management commands
handle_content_command() {
    case "${1:-}" in
        add)
            shift
            handle_content_add "$@"
            ;;
        list)
            shift
            handle_content_list "$@"
            ;;
        get)
            shift
            handle_content_get "$@"
            ;;
        remove)
            shift
            handle_content_remove "$@"
            ;;
        execute)
            shift
            handle_content_execute "$@"
            ;;
        *)
            echo "Error: Unknown content command '${1:-}'" >&2
            echo "Available commands: add, list, get, remove, execute"
            exit 1
            ;;
    esac
}

# Show help message
show_help() {
    cat <<EOF
Cloudflare AI Gateway Resource CLI

Usage: ${RESOURCE_NAME} <command> [options]

Commands:
    start                    Activate the gateway proxy
    stop                     Deactivate the gateway proxy
    status                   Check gateway health and metrics
    info                     Display gateway configuration
    logs                     View gateway logs and analytics
    configure                Configure gateway settings
    content add              Add gateway configuration/rules
    content list             List gateway configurations
    content get              Get specific configuration
    content remove           Remove configuration
    content execute          Apply configuration changes
    help                     Show this help message

Options:
    --format json           Output in JSON format (for status commands)
    --verbose               Show detailed output

Examples:
    ${RESOURCE_NAME} status
    ${RESOURCE_NAME} configure --provider openrouter --cache-ttl 3600
    ${RESOURCE_NAME} content add --file rules.json
    ${RESOURCE_NAME} logs --tail 100

For more information, see: scripts/resources/execution/cloudflare-ai-gateway/README.md
EOF
}

# Execute main function
main "$@"
