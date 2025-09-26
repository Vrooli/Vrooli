#!/usr/bin/env bash
# PaperMC Minecraft Server Resource CLI
# Provides lifecycle management for PaperMC server instances

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

# Source configurations and libraries
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Resource metadata
readonly RESOURCE_VERSION="1.0.0"

# Show help information
show_help() {
    cat << EOF
PaperMC Minecraft Server Resource v${RESOURCE_VERSION}

USAGE:
    resource-papermc <command> [options]

COMMANDS:
    help                    Show this help message
    info                    Display resource information
    manage <subcommand>     Lifecycle management
        install            Install PaperMC server
        uninstall          Remove PaperMC server
        start              Start the server
        stop               Stop the server
        restart            Restart the server
    test <type>            Run tests
        smoke              Quick health check
        integration        Full functionality test
        unit               Library function tests
        all                Run all tests
    content <subcommand>    Content management
        execute <cmd>      Execute RCON command
        backup             Backup world and config
        configure          Update server configuration
        list-plugins       List installed plugins
        add-plugin <src>   Add plugin from URL or name
        remove-plugin      Remove installed plugin
        health             Show server health metrics
        analyze-logs       Analyze server logs
    status                 Show server status
    logs                   View server logs

EXAMPLES:
    # Install and start the server
    resource-papermc manage install
    resource-papermc manage start --wait

    # Execute RCON commands
    resource-papermc content execute "say Hello World!"
    resource-papermc content execute "list"

    # Backup the world
    resource-papermc content backup

    # Run tests
    resource-papermc test smoke
    resource-papermc test all

CONFIGURATION:
    Server Type: ${PAPERMC_SERVER_TYPE}
    Memory: ${PAPERMC_MEMORY} - ${PAPERMC_MAX_MEMORY}
    RCON Port: ${PAPERMC_RCON_PORT}
    Game Port: ${PAPERMC_SERVER_PORT}
    Data Directory: ${PAPERMC_DATA_DIR}

EOF
}

# Show resource information
show_info() {
    local json_flag="${1:-}"
    
    if [[ "${json_flag}" == "--json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "PaperMC Resource Information:"
        echo "=============================="
        jq -r '
            "Startup Order: \(.startup_order)",
            "Dependencies: \(.dependencies | join(", "))",
            "Optional Dependencies: \(.optional_dependencies | join(", "))",
            "Startup Timeout: \(.startup_timeout)s",
            "Startup Time Estimate: \(.startup_time_estimate)",
            "Recovery Attempts: \(.recovery_attempts)",
            "Priority: \(.priority)",
            "Ports:",
            "  - Game: \(.ports.game)",
            "  - RCON: \(.ports.rcon)",
            "  - Health: \(.ports.health)"
        ' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

# Main command router
main() {
    local command="${1:-}"
    shift || true

    case "${command}" in
        help|--help|-h|"")
            show_help
            ;;
        info)
            show_info "$@"
            ;;
        manage)
            local subcommand="${1:-}"
            shift || true
            case "${subcommand}" in
                install)
                    install_papermc "$@"
                    ;;
                uninstall)
                    uninstall_papermc "$@"
                    ;;
                start)
                    start_papermc "$@"
                    ;;
                stop)
                    stop_papermc "$@"
                    ;;
                restart)
                    restart_papermc "$@"
                    ;;
                *)
                    echo "Error: Unknown manage subcommand: ${subcommand}" >&2
                    echo "Run 'resource-papermc help' for usage" >&2
                    exit 1
                    ;;
            esac
            ;;
        test)
            local test_type="${1:-all}"
            run_tests "${test_type}"
            ;;
        content)
            local subcommand="${1:-}"
            shift || true
            case "${subcommand}" in
                execute)
                    execute_rcon_command "$@"
                    ;;
                backup)
                    backup_world "$@"
                    ;;
                configure)
                    configure_server "$@"
                    ;;
                list-plugins)
                    list_plugins "$@"
                    ;;
                add-plugin)
                    add_plugin "$@"
                    ;;
                remove-plugin)
                    remove_plugin "$@"
                    ;;
                health)
                    get_health_metrics "$@"
                    ;;
                analyze-logs)
                    analyze_logs "$@"
                    ;;
                *)
                    echo "Error: Unknown content subcommand: ${subcommand}" >&2
                    exit 1
                    ;;
            esac
            ;;
        status)
            show_status "$@"
            ;;
        logs)
            show_logs "$@"
            ;;
        *)
            echo "Error: Unknown command: ${command}" >&2
            echo "Run 'resource-papermc help' for usage" >&2
            exit 1
            ;;
    esac
}

# Run main function
main "$@"