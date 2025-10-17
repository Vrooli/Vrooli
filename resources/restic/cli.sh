#!/usr/bin/env bash
# Restic CLI - Main entrypoint for the restic resource

set -euo pipefail

# Get the directory of this script
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly RESOURCE_DIR
export RESOURCE_DIR

# Source core functionality
source "${RESOURCE_DIR}/lib/core.sh"

# Main function
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help)
            restic::show_help
            ;;
        info)
            restic::show_info "$@"
            ;;
        manage)
            restic::manage "$@"
            ;;
        test)
            restic::test "$@"
            ;;
        content)
            restic::content "$@"
            ;;
        status)
            restic::status "$@"
            ;;
        logs)
            restic::logs "$@"
            ;;
        backup)
            restic::backup "$@"
            ;;
        backup-with-hooks|backup-hooks)
            restic::backup_with_hooks "$@"
            ;;
        backup-postgres)
            restic::backup_postgres "$@"
            ;;
        backup-redis)
            restic::backup_redis "$@"
            ;;
        backup-minio)
            restic::backup_minio "$@"
            ;;
        restore)
            restic::restore "$@"
            ;;
        snapshots)
            restic::snapshots "$@"
            ;;
        prune)
            restic::prune "$@"
            ;;
        verify)
            restic::verify "$@"
            ;;
        *)
            echo "Error: Unknown command: $command"
            echo "Run 'resource-restic help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"