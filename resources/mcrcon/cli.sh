#!/usr/bin/env bash
# mcrcon CLI - Main entry point for Minecraft Remote Console resource
# Implements v2.0 universal resource contract

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

# Source configuration and libraries
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"

# Resource information
readonly RESOURCE_NAME="mcrcon"
readonly RESOURCE_VERSION="1.0.0"

# Help function
show_help() {
    cat << EOF
mcrcon - Minecraft Remote Console Resource

USAGE:
    resource-mcrcon <command> [options]

COMMANDS:
    help                    Show this help message
    info                    Show resource runtime information
    manage <subcommand>     Lifecycle management
        install             Install mcrcon and dependencies
        uninstall           Remove mcrcon
        start               Start mcrcon service
        stop                Stop mcrcon service
        restart             Restart mcrcon service
    test <type>             Run tests
        smoke               Quick health check (<30s)
        integration         Full functionality test (<120s)
        unit                Library function tests (<60s)
        all                 Run all test suites
    content <subcommand>    Server and command management
        execute <cmd>       Execute RCON command
        execute-all <cmd>   Execute on all servers
        list                List configured servers
        add <server>        Add server configuration
        remove <server>     Remove server configuration
        discover            Auto-discover Minecraft servers
        test <host> <port>  Test RCON connection
    player <subcommand>     Player management
        list [server]       List online players
        info <player>       Get player information
        teleport <p> <x> <y> <z>  Teleport player
        kick <player>       Kick player from server
        ban <player>        Ban player from server
        give <p> <item> [n] Give items to player
    world <subcommand>      World management operations
        save [server]       Save world data
        backup [server]     Create world backup
        info [server]       Display world information
        set-property <prop> <val>  Set world property
        set-spawn <x> <y> <z>      Set world spawn point
    event <subcommand>      Event streaming and monitoring
        start-stream        Start background event streaming
        stop-stream         Stop event streaming
        stream-chat [secs]  Stream chat for duration
        monitor [type]      Monitor specific events (joins/leaves/deaths/all)
        tail [lines]        Show recent events
    webhook <subcommand>    Webhook integration
        configure <url>     Configure webhook endpoint
        list                List configured webhooks
        remove <url>        Remove webhook
        start               Start webhook service
        stop                Stop webhook service
        test <url>          Test webhook connectivity
    mod <subcommand>        Mod/Plugin integration
        list [server]       List installed mods/plugins
        execute <mod> <cmd> Execute mod-specific command
        register-commands   Register custom mod commands
        show-commands [mod] Show registered mod commands
        test-support        Test mod support on server
    status                  Show resource status
    logs                    View resource logs
    credentials             Display connection credentials
    integration <sub>       Integration with other resources
        auto-configure      Auto-detect Minecraft servers
        quick-start         Quick setup and test
        detect-papermc      Detect PaperMC installation
        check-papermc       Check PaperMC status

EXAMPLES:
    # Install and start mcrcon
    resource-mcrcon manage install
    resource-mcrcon manage start --wait

    # Execute Minecraft commands
    resource-mcrcon content execute "list"
    resource-mcrcon content execute "say Hello from Vrooli!"

    # Run tests
    resource-mcrcon test smoke
    resource-mcrcon test all

DEFAULT CONFIGURATION:
    Server Host: ${MCRCON_HOST:-localhost}
    Server Port: ${MCRCON_PORT:-25575}
    Timeout: ${MCRCON_TIMEOUT:-30}s

For more information, see the README.md file.
EOF
}

# Info function - displays runtime configuration
show_info() {
    local json_flag="${1:-}"
    local runtime_file="${SCRIPT_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        echo "Error: runtime.json not found" >&2
        return 1
    fi
    
    if [[ "$json_flag" == "--json" ]]; then
        cat "$runtime_file"
    else
        echo "mcrcon Resource Information:"
        echo "----------------------------"
        jq -r '
            "Startup Order: \(.startup_order)",
            "Dependencies: \(.dependencies | join(", "))",
            "Startup Time: \(.startup_time_estimate)",
            "Startup Timeout: \(.startup_timeout)s",
            "Recovery Attempts: \(.recovery_attempts)",
            "Priority: \(.priority)"
        ' "$runtime_file"
    fi
}

# Main command router
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        help|--help|-h|"")
            show_help
            ;;
        info)
            show_info "$@"
            ;;
        manage)
            "${SCRIPT_DIR}/lib/core.sh" manage "$@"
            ;;
        test)
            "${SCRIPT_DIR}/lib/test.sh" "$@"
            ;;
        content)
            "${SCRIPT_DIR}/lib/core.sh" content "$@"
            ;;
        player)
            "${SCRIPT_DIR}/lib/core.sh" player "$@"
            ;;
        world)
            "${SCRIPT_DIR}/lib/core.sh" world "$@"
            ;;
        event)
            "${SCRIPT_DIR}/lib/core.sh" event "$@"
            ;;
        webhook)
            "${SCRIPT_DIR}/lib/core.sh" webhook "$@"
            ;;
        mod)
            "${SCRIPT_DIR}/lib/core.sh" mod "$@"
            ;;
        status)
            "${SCRIPT_DIR}/lib/core.sh" status
            ;;
        logs)
            "${SCRIPT_DIR}/lib/core.sh" logs
            ;;
        credentials)
            "${SCRIPT_DIR}/lib/core.sh" credentials
            ;;
        integration)
            # Integration helpers for working with other Vrooli resources
            source "${SCRIPT_DIR}/lib/integration.sh"
            case "${1:-}" in
                auto-configure)
                    auto_configure
                    ;;
                quick-start)
                    quick_start
                    ;;
                detect-papermc)
                    detect_papermc
                    ;;
                check-papermc)
                    check_papermc_status
                    ;;
                *)
                    echo "Integration commands:"
                    echo "  auto-configure  - Auto-detect and configure Minecraft servers"
                    echo "  quick-start     - Quick setup and connection test"
                    echo "  detect-papermc  - Detect PaperMC installation"
                    echo "  check-papermc   - Check if PaperMC is running"
                    ;;
            esac
            ;;
        *)
            echo "Error: Unknown command: $command" >&2
            echo "Run 'resource-mcrcon help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"