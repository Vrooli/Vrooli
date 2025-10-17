#!/usr/bin/env bash
# Strapi Resource CLI - Headless CMS for content management
# Implements v2.0 universal contract for resource management

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

# Source core library
source "${SCRIPT_DIR}/lib/core.sh"

# Source additional libraries
if [[ -f "${SCRIPT_DIR}/lib/docker.sh" ]]; then
    source "${SCRIPT_DIR}/lib/docker.sh"
fi
if [[ -f "${SCRIPT_DIR}/lib/storage.sh" ]]; then
    source "${SCRIPT_DIR}/lib/storage.sh"
fi
if [[ -f "${SCRIPT_DIR}/lib/backup.sh" ]]; then
    source "${SCRIPT_DIR}/lib/backup.sh"
fi

# Resource metadata
readonly RESOURCE_NAME="strapi"
readonly RESOURCE_VERSION="5.0"
readonly RESOURCE_PORT="${STRAPI_PORT:-1337}"

#######################################
# Show help information
#######################################
show_help() {
    cat << EOF
Strapi Resource - Headless CMS v${RESOURCE_VERSION}

USAGE:
    resource-strapi <command> [options]

COMMANDS:
    help                Show this help message
    info                Show resource runtime information
    manage              Lifecycle management commands
      install           Install Strapi and dependencies
      uninstall         Remove Strapi installation
      start             Start Strapi service
      stop              Stop Strapi service
      restart           Restart Strapi service
    test                Run validation tests
      smoke             Quick health check (<30s)
      integration       Full functionality test (<120s)
      unit              Library function tests (<60s)
      all               Run all test phases
    content             Content management operations
      list              List content types
      add <type> <data> Create new content
      get <type> <id>   Retrieve content by ID
      remove <type> <id> Delete content
      execute <query>   Execute custom query
    docker              Docker deployment commands
      start             Start with Docker Compose
      stop              Stop Docker containers
      status            Show Docker status
      logs              View Docker logs
    admin               Admin management
      create            Create admin user
    storage             S3/MinIO storage management
      enable            Configure S3 storage provider
      disable           Revert to local storage
      test              Test S3 integration
      list              List uploaded files
    backup              Backup and restore operations
      create [name]     Create full backup
      restore <file>    Restore from backup
      list              List available backups
      schedule <freq>   Schedule automatic backups (daily/weekly/monthly)
    status              Show service status
    logs                View service logs
    credentials         Display admin credentials

OPTIONS:
    --json              Output in JSON format
    --verbose           Show detailed output
    --force             Skip confirmation prompts
    --wait              Wait for service to be ready
    --timeout <sec>     Operation timeout (default: 60)

ENVIRONMENT:
    STRAPI_PORT         Service port (default: 1337)
    STRAPI_HOST         Bind address (default: 0.0.0.0)
    POSTGRES_HOST       Database host (required)
    POSTGRES_PORT       Database port (required)
    POSTGRES_USER       Database user (required)
    POSTGRES_PASSWORD   Database password (required)
    STRAPI_DATABASE_NAME Database name (default: strapi)
    STRAPI_ADMIN_EMAIL  Admin email (default: admin@vrooli.local)
    STRAPI_ADMIN_PASSWORD Admin password (auto-generated if not set)

EXAMPLES:
    # Install and start Strapi
    resource-strapi manage install
    resource-strapi manage start --wait

    # Check service health
    resource-strapi test smoke
    resource-strapi status --json

    # View logs
    resource-strapi logs --tail 50

    # List content types
    resource-strapi content list

DEFAULT CONFIGURATION:
    Port: ${RESOURCE_PORT}
    Admin URL: http://localhost:${RESOURCE_PORT}/admin
    API URL: http://localhost:${RESOURCE_PORT}/api
    GraphQL: http://localhost:${RESOURCE_PORT}/graphql
    Health: http://localhost:${RESOURCE_PORT}/health

For more information, see resources/strapi/README.md
EOF
}

#######################################
# Show runtime information
#######################################
show_info() {
    local json_mode="${1:-false}"
    
    if [[ ! -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        core::error "Runtime configuration not found"
        return 1
    fi
    
    if [[ "$json_mode" == "true" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "Strapi Resource Runtime Information:"
        echo "====================================="
        jq -r 'to_entries | .[] | "  \(.key): \(.value)"' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

#######################################
# Main CLI router
#######################################
main() {
    # Check for no arguments
    if [[ $# -eq 0 ]]; then
        show_help
        exit 0
    fi
    
    local command="${1:-}"
    shift || true
    
    case "$command" in
        help|--help|-h)
            show_help
            ;;
            
        info)
            local json_mode="false"
            [[ "${1:-}" == "--json" ]] && json_mode="true"
            show_info "$json_mode"
            ;;
            
        manage)
            local subcommand="${1:-}"
            shift || true
            
            case "$subcommand" in
                install)
                    core::manage_install "$@"
                    ;;
                uninstall)
                    core::manage_uninstall "$@"
                    ;;
                start)
                    core::manage_start "$@"
                    ;;
                stop)
                    core::manage_stop "$@"
                    ;;
                restart)
                    core::manage_restart "$@"
                    ;;
                *)
                    core::error "Unknown manage subcommand: $subcommand"
                    echo "Available: install, uninstall, start, stop, restart"
                    exit 1
                    ;;
            esac
            ;;
            
        test)
            source "${SCRIPT_DIR}/lib/test.sh"
            local phase="${1:-all}"
            shift || true
            
            case "$phase" in
                smoke)
                    test::run_smoke
                    ;;
                integration)
                    test::run_integration
                    ;;
                unit)
                    test::run_unit
                    ;;
                all)
                    test::run_all
                    ;;
                *)
                    core::error "Unknown test phase: $phase"
                    echo "Available: smoke, integration, unit, all"
                    exit 1
                    ;;
            esac
            ;;
            
        content)
            local operation="${1:-}"
            shift || true
            
            case "$operation" in
                list)
                    core::content_list "$@"
                    ;;
                add)
                    core::content_add "$@"
                    ;;
                get)
                    core::content_get "$@"
                    ;;
                remove)
                    core::content_remove "$@"
                    ;;
                execute)
                    core::content_execute "$@"
                    ;;
                *)
                    core::error "Unknown content operation: $operation"
                    echo "Available: list, add, get, remove, execute"
                    exit 1
                    ;;
            esac
            ;;
            
        docker)
            local subcommand="${1:-}"
            shift || true
            
            case "$subcommand" in
                start)
                    docker::start "$@"
                    ;;
                stop)
                    docker::stop "$@"
                    ;;
                status)
                    docker::status "$@"
                    ;;
                logs)
                    docker::logs "$@"
                    ;;
                *)
                    core::error "Unknown docker subcommand: $subcommand"
                    echo "Available: start, stop, status, logs"
                    exit 1
                    ;;
            esac
            ;;
            
        admin)
            local subcommand="${1:-}"
            shift || true
            
            case "$subcommand" in
                create)
                    core::create_admin "$@"
                    ;;
                *)
                    core::error "Unknown admin subcommand: $subcommand"
                    echo "Available: create"
                    exit 1
                    ;;
            esac
            ;;
            
        storage)
            local subcommand="${1:-}"
            shift || true
            
            case "$subcommand" in
                enable)
                    storage::enable "$@"
                    ;;
                disable)
                    storage::disable "$@"
                    ;;
                test)
                    storage::test "$@"
                    ;;
                list)
                    storage::list_files "$@"
                    ;;
                *)
                    core::error "Unknown storage subcommand: $subcommand"
                    echo "Available: enable, disable, test, list"
                    exit 1
                    ;;
            esac
            ;;
            
        backup)
            local subcommand="${1:-}"
            shift || true
            
            case "$subcommand" in
                create)
                    backup::create "$@"
                    ;;
                restore)
                    backup::restore "$@"
                    ;;
                list)
                    backup::list "$@"
                    ;;
                schedule)
                    backup::schedule "$@"
                    ;;
                *)
                    core::error "Unknown backup subcommand: $subcommand"
                    echo "Available: create, restore, list, schedule"
                    exit 1
                    ;;
            esac
            ;;
            
        status)
            core::show_status "$@"
            ;;
            
        logs)
            core::show_logs "$@"
            ;;
            
        credentials)
            core::show_credentials "$@"
            ;;
            
        *)
            core::error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"