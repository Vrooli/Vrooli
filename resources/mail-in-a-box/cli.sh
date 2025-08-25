#!/bin/bash

# CLI interface for Mail-in-a-Box resource

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
MAILINABOX_CLI_DIR="${APP_ROOT}/resources/mail-in-a-box"
MAILINABOX_LIB_DIR="${APP_ROOT}/resources/mail-in-a-box/lib"

# Source all library functions
source "$MAILINABOX_LIB_DIR/core.sh"
source "$MAILINABOX_LIB_DIR/install.sh"
source "$MAILINABOX_LIB_DIR/start.sh"
source "$MAILINABOX_LIB_DIR/stop.sh"
source "$MAILINABOX_LIB_DIR/status.sh"
source "$MAILINABOX_LIB_DIR/inject.sh"

# Main CLI handler
main() {
    local command="${1:-help}"
    shift
    
    case "$command" in
        install)
            mailinabox_install "$@"
            ;;
        uninstall)
            mailinabox_uninstall "$@"
            ;;
        start)
            mailinabox_start "$@"
            ;;
        stop)
            mailinabox_stop "$@"
            ;;
        restart)
            mailinabox_stop && mailinabox_start "$@"
            ;;
        status)
            mailinabox_status "$@"
            ;;
        add-account|add-user)
            mailinabox_add_account "$@"
            ;;
        add-alias)
            mailinabox_add_alias "$@"
            ;;
        add-domain)
            mailinabox_add_domain "$@"
            ;;
        inject)
            mailinabox_inject_file "$@"
            ;;
        health)
            mailinabox_get_health
            ;;
        version)
            mailinabox_get_version
            ;;
        help)
            cat <<EOF
Mail-in-a-Box Resource CLI

Usage: $(basename "$0") [command] [options]

Commands:
  install              Install Mail-in-a-Box
  uninstall            Uninstall Mail-in-a-Box
  start                Start Mail-in-a-Box service
  stop                 Stop Mail-in-a-Box service
  restart              Restart Mail-in-a-Box service
  status [--json]      Show Mail-in-a-Box status
  add-account EMAIL PASSWORD   Add email account
  add-alias ALIAS TARGET       Add email alias
  add-domain DOMAIN           Add custom domain
  inject FILE                 Import email configuration from file
  health               Check service health
  version              Show version
  help                 Show this help message

Examples:
  $(basename "$0") install
  $(basename "$0") start
  $(basename "$0") add-account user@example.com SecurePass123
  $(basename "$0") add-alias info@example.com user@example.com
  $(basename "$0") status --json
  $(basename "$0") inject accounts.csv
EOF
            ;;
        *)
            format_error "Unknown command: $command"
            echo "Run '$(basename "$0") help' for usage information"
            return 1
            ;;
    esac
}

# Run main function
main "$@"
