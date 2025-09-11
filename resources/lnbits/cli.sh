#!/usr/bin/env bash
# LNbits Resource CLI Interface
# Implements v2.0 universal contract for Lightning Network payments

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="lnbits"

# Source core libraries
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Main command routing
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help)
            show_help
            ;;
        info)
            show_info "$@"
            ;;
        manage)
            handle_manage "$@"
            ;;
        test)
            handle_test "$@"
            ;;
        content)
            handle_content "$@"
            ;;
        status)
            show_status "$@"
            ;;
        logs)
            show_logs "$@"
            ;;
        credentials)
            show_credentials "$@"
            ;;
        *)
            echo "Error: Unknown command: $command" >&2
            echo "Run '${RESOURCE_NAME} help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Show comprehensive help
show_help() {
    cat << EOF
LNbits - Lightning Network Bitcoin Payments Resource

USAGE:
    ${RESOURCE_NAME} <command> [options]

COMMANDS:
    help                Show this help message
    info [--json]       Show resource information and configuration
    manage <action>     Manage resource lifecycle
    test <phase>        Run validation tests
    content <action>    Manage wallets and payments
    status [--json]     Show detailed resource status
    logs [--tail N]     View resource logs
    credentials         Display API credentials

MANAGE ACTIONS:
    install            Install LNbits and dependencies
    start [--wait]     Start the LNbits service
    stop               Stop the LNbits service
    restart            Restart the LNbits service
    uninstall          Remove LNbits installation

TEST PHASES:
    smoke              Quick health check (<30s)
    integration        Full functionality test (<120s)
    unit               Library function tests (<60s)
    all                Run all test phases

CONTENT ACTIONS:
    add                Create new wallet or payment
    list               List wallets or payments
    get <id>           Get specific wallet or payment
    remove <id>        Delete wallet
    execute            Execute payment operations

EXAMPLES:
    # Install and start LNbits
    ${RESOURCE_NAME} manage install
    ${RESOURCE_NAME} manage start --wait
    
    # Create a new wallet
    ${RESOURCE_NAME} content add --type wallet --name "Store Wallet"
    
    # Generate an invoice
    ${RESOURCE_NAME} content execute --action invoice --amount 1000 --memo "Coffee"
    
    # Check payment status
    ${RESOURCE_NAME} content get --type payment --id <payment_hash>
    
    # Run health check
    ${RESOURCE_NAME} test smoke
    
    # View logs
    ${RESOURCE_NAME} logs --tail 50

DEFAULT CONFIGURATION:
    Port: 5001 (API)
    Database: PostgreSQL (required)
    Backend: LND/CLN (optional, can use custodial)
    Data Directory: \${DATA_DIR}/resources/lnbits

For more information, see: resources/lnbits/README.md
EOF
    exit 0
}

# Show resource information
show_info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "LNbits Resource Information"
        echo "=========================="
        echo "Version: 0.12.1"
        echo "Port: ${LNBITS_PORT:-5001}"
        echo "Backend: ${LNBITS_BACKEND_WALLET:-FakeWallet}"
        echo ""
        echo "Runtime Configuration:"
        jq -r 'to_entries[] | "  \(.key): \(.value)"' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

# Handle lifecycle management
handle_manage() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        install)
            manage_install "$@"
            ;;
        start)
            manage_start "$@"
            ;;
        stop)
            manage_stop
            ;;
        restart)
            manage_restart "$@"
            ;;
        uninstall)
            manage_uninstall "$@"
            ;;
        *)
            echo "Error: Unknown manage action: $action" >&2
            echo "Valid actions: install, start, stop, restart, uninstall" >&2
            exit 1
            ;;
    esac
}

# Handle test execution
handle_test() {
    local phase="${1:-all}"
    
    case "$phase" in
        smoke)
            test_smoke
            ;;
        integration)
            test_integration
            ;;
        unit)
            test_unit
            ;;
        all)
            test_all
            ;;
        *)
            echo "Error: Unknown test phase: $phase" >&2
            echo "Valid phases: smoke, integration, unit, all" >&2
            exit 1
            ;;
    esac
}

# Handle content management
handle_content() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        add)
            content_add "$@"
            ;;
        list)
            content_list "$@"
            ;;
        get)
            content_get "$@"
            ;;
        remove)
            content_remove "$@"
            ;;
        execute)
            content_execute "$@"
            ;;
        *)
            echo "Error: Unknown content action: $action" >&2
            echo "Valid actions: add, list, get, remove, execute" >&2
            exit 1
            ;;
    esac
}

# Show resource status
show_status() {
    local format="${1:-text}"
    
    if ! is_installed; then
        if [[ "$format" == "--json" ]]; then
            echo '{"status":"not_installed","health":"unknown"}'
        else
            echo "Status: Not installed"
        fi
        exit 0
    fi
    
    local health_status="unhealthy"
    local container_status="stopped"
    
    if is_running; then
        container_status="running"
        if check_health; then
            health_status="healthy"
        fi
    fi
    
    if [[ "$format" == "--json" ]]; then
        cat << EOF
{
    "status": "${container_status}",
    "health": "${health_status}",
    "port": ${LNBITS_PORT:-5001},
    "backend": "${LNBITS_BACKEND_WALLET:-FakeWallet}",
    "uptime": "$(get_uptime)"
}
EOF
    else
        echo "LNbits Status"
        echo "============="
        echo "Container: ${container_status}"
        echo "Health: ${health_status}"
        echo "Port: ${LNBITS_PORT:-5001}"
        echo "Backend: ${LNBITS_BACKEND_WALLET:-FakeWallet}"
        echo "Uptime: $(get_uptime)"
    fi
}

# Show logs
show_logs() {
    local tail_lines=50
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tail)
                tail_lines="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if ! is_installed; then
        echo "Error: LNbits is not installed" >&2
        exit 1
    fi
    
    docker logs --tail "$tail_lines" lnbits-server 2>&1
}

# Show credentials
show_credentials() {
    if ! is_running; then
        echo "Error: LNbits is not running" >&2
        exit 1
    fi
    
    echo "LNbits Credentials"
    echo "=================="
    echo "API URL: http://localhost:${LNBITS_PORT:-5001}"
    echo "Admin UI: http://localhost:${LNBITS_PORT:-5001}/wallet"
    echo ""
    echo "Note: Wallet API keys are generated per wallet."
    echo "Create a wallet first with: ${RESOURCE_NAME} content add --type wallet"
}

# Run main function
main "$@"