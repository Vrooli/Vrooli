#!/bin/bash

# Wiki.js Resource CLI
# Modern wiki platform with Git storage backend

set -euo pipefail

# Get script directory (resolve symlinks)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    WIKIJS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${WIKIJS_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi

WIKIJS_DIR="${APP_ROOT}/resources/wikijs"

# Source dependencies
source "$WIKIJS_DIR/../../../lib/utils/var.sh"
source "$WIKIJS_DIR/../../../lib/utils/format.sh"
source "$WIKIJS_DIR/lib/common.sh"

# Display help
show_help() {
    cat << EOF
Wiki.js Resource Manager

Usage: $(basename "$0") <command> [options]

Commands:
  install          Install Wiki.js
  uninstall        Uninstall Wiki.js (requires --force)
  start            Start Wiki.js
  stop             Stop Wiki.js
  restart          Restart Wiki.js
  status           Show Wiki.js status
  logs             Show Wiki.js logs
  test             Run integration tests
  inject           Inject content into Wiki.js
  api              Direct API access
  search           Search wiki content
  backup           Backup Wiki.js data
  restore          Restore Wiki.js backup
  git-sync         Sync with Git repository
  rebuild-search   Rebuild search index
  help             Show this help message

Options:
  --dry-run        Show what would be done without making changes
  --force          Force operation (required for destructive actions)
  --json           Output in JSON format (for status)
  --verbose        Show detailed output

Examples:
  $(basename "$0") install
  $(basename "$0") status --json
  $(basename "$0") test
  $(basename "$0") api create-page --title "New Page" --content "Content here"
  $(basename "$0") search "query term"

For more information, see: scripts/resources/execution/wikijs/README.md
EOF
}

# Main command handler
main() {
    local cmd="${1:-}"
    shift || true

    case "$cmd" in
        install)
            source "$WIKIJS_DIR/lib/install.sh"
            install_wikijs "$@"
            ;;
        uninstall)
            source "$WIKIJS_DIR/lib/uninstall.sh"
            uninstall_wikijs "$@"
            ;;
        start)
            source "$WIKIJS_DIR/lib/lifecycle.sh"
            start_wikijs "$@"
            ;;
        stop)
            source "$WIKIJS_DIR/lib/lifecycle.sh"
            stop_wikijs "$@"
            ;;
        restart)
            source "$WIKIJS_DIR/lib/lifecycle.sh"
            restart_wikijs "$@"
            ;;
        status)
            source "$WIKIJS_DIR/lib/status.sh"
            status_wikijs "$@"
            ;;
        logs)
            source "$WIKIJS_DIR/lib/logs.sh"
            show_logs "$@"
            ;;
        test)
            source "$WIKIJS_DIR/lib/test.sh"
            run_tests "$@"
            ;;
        inject)
            source "$WIKIJS_DIR/lib/inject.sh"
            inject_content "$@"
            ;;
        api)
            source "$WIKIJS_DIR/lib/api.sh"
            api_command "$@"
            ;;
        search)
            source "$WIKIJS_DIR/lib/search.sh"
            search_content "$@"
            ;;
        backup)
            source "$WIKIJS_DIR/lib/backup.sh"
            backup_wikijs "$@"
            ;;
        restore)
            source "$WIKIJS_DIR/lib/backup.sh"
            restore_wikijs "$@"
            ;;
        git-sync)
            source "$WIKIJS_DIR/lib/git.sh"
            sync_git "$@"
            ;;
        rebuild-search)
            source "$WIKIJS_DIR/lib/search.sh"
            rebuild_search_index "$@"
            ;;
        help|--help|-h)
            show_help
            ;;
        "")
            echo "Error: No command specified"
            show_help
            exit 1
            ;;
        *)
            echo "Error: Unknown command: $cmd"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
