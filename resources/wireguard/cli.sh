#!/usr/bin/env bash
# WireGuard Resource CLI - Secure VPN networking for Vrooli
# Implements v2.0 universal contract

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="wireguard"

# Load core libraries
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"
source "${SCRIPT_DIR}/config/defaults.sh"

# ====================
# Help Command
# ====================
show_help() {
    cat << EOF
WireGuard Resource - Modern VPN Networking for Vrooli

USAGE:
    resource-wireguard <command> [options]

COMMANDS:
    help                Show this help message
    info                Show resource runtime information
    manage              Lifecycle management commands
      install           Install WireGuard and dependencies
      uninstall         Remove WireGuard completely
      start             Start WireGuard service
      stop              Stop WireGuard service
      restart           Restart WireGuard service
    test                Run validation tests
      smoke             Quick health check (<30s)
      integration       Full functionality test (<120s)
      unit              Library function tests (<60s)
      all               Run all test phases
    content             Manage tunnel configurations
      add <name>        Add a new tunnel configuration
      list              List all tunnel configurations
      get <name>        Get a specific tunnel configuration
      remove <name>     Remove a tunnel configuration
      execute <name>    Activate a tunnel configuration
    status              Show WireGuard service status
    stats               Show detailed traffic statistics
    logs                View WireGuard logs
    credentials         Display connection information
    rotate              Key rotation management
      keys <name>       Rotate keys for a tunnel immediately
      schedule <name>   Schedule automatic key rotation
      status            Show rotation status and history
    namespace           Network isolation management
      create <name>     Create isolated Docker network
      list              List all isolated networks
      delete <name>     Delete an isolated network
      connect <container> <network> Connect container to network
      status <name>     Show network status
    nat                 NAT traversal management
      enable <name>     Enable NAT traversal for tunnel
      disable <name>    Disable NAT traversal for tunnel
      status            Show NAT traversal status
      test <name>       Test NAT connectivity

EXAMPLES:
    # Install and start WireGuard
    resource-wireguard manage install
    resource-wireguard manage start --wait

    # Create a new tunnel
    resource-wireguard content add site-to-site

    # Check service health
    resource-wireguard test smoke
    resource-wireguard status

    # View logs
    resource-wireguard logs --tail 50

DEFAULT CONFIGURATION:
    Port: ${WIREGUARD_PORT:-51820}/udp
    Network: ${WIREGUARD_NETWORK:-10.13.13.0/24}
    Container: ${CONTAINER_NAME:-vrooli-wireguard}

EOF
}

# ====================
# Info Command
# ====================
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
        echo "WireGuard Resource Information:"
        echo "==============================="
        jq -r '
            "Startup Order: \(.startup_order)
Dependencies: \(.dependencies | join(", "))
Startup Time: \(.startup_time_estimate)
Timeout: \(.startup_timeout)s
Recovery Attempts: \(.recovery_attempts)
Priority: \(.priority)"
        ' "$runtime_file"
    fi
}

# ====================
# Main Command Router
# ====================
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
            handle_manage_command "$@"
            ;;
        test)
            handle_test_command "$@"
            ;;
        content)
            handle_content_command "$@"
            ;;
        status)
            show_status "$@"
            ;;
        stats)
            show_traffic_stats "$@"
            ;;
        logs)
            show_logs "$@"
            ;;
        credentials)
            show_credentials "$@"
            ;;
        rotate)
            handle_rotate_command "$@"
            ;;
        namespace)
            handle_namespace_command "$@"
            ;;
        nat)
            handle_nat_command "$@"
            ;;
        *)
            echo "Error: Unknown command: $command" >&2
            echo "Run 'resource-wireguard help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"