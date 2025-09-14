#!/usr/bin/env bash
# Pi-hole Resource CLI - Network-level ad blocking and DNS management
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="${SCRIPT_DIR}"  # Store original resource directory
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Main command handler
main() {
    local cmd="${1:-help}"
    shift || true
    
    case "$cmd" in
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
            echo "Error: Unknown command: $cmd" >&2
            echo "Run 'vrooli resource pihole help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Show comprehensive help
show_help() {
    cat << EOF
Pi-hole Resource - Network-level DNS Ad Blocking

USAGE:
    vrooli resource pihole <command> [options]

COMMANDS:
    help                Show this help message
    info                Show resource configuration
    manage              Lifecycle management
        install         Install Pi-hole
        start           Start Pi-hole service
        stop            Stop Pi-hole service
        restart         Restart Pi-hole service
        uninstall       Remove Pi-hole completely
    test                Run tests
        smoke           Quick health check (30s)
        integration     Full integration tests (120s)
        unit            Test library functions (60s)
        all             Run all tests (180s)
    content             Manage DNS filtering
        stats           Show blocking statistics
        update          Update blocklists
        blacklist       Manage blacklist
        whitelist       Manage whitelist
        logs            Query DNS logs
        disable         Temporarily disable blocking
        enable          Re-enable blocking
    status              Show service status
    logs                View service logs
    credentials         Display API credentials

EXAMPLES:
    # Basic setup
    vrooli resource pihole manage install
    vrooli resource pihole manage start --wait
    vrooli resource pihole status
    
    # Manage blocking
    vrooli resource pihole content stats
    vrooli resource pihole content blacklist add ads.example.com
    vrooli resource pihole content whitelist add safe.example.com
    
    # Testing
    vrooli resource pihole test smoke
    vrooli resource pihole test all

CONFIGURATION:
    DNS Port:     53 (TCP/UDP)
    API Port:     8087
    DHCP Port:    67 (UDP, optional)
    
DEFAULT BLOCKLISTS:
    - Steven Black's Unified Hosts
    - EasyList
    - EasyPrivacy
    - Malware domains

For more information, see: /home/matthalloran8/Vrooli/resources/pihole/README.md
EOF
    exit 0
}

# Show resource info from runtime.json
show_info() {
    local format="${1:-text}"
    local runtime_file="${RESOURCE_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        echo "Error: runtime.json not found" >&2
        exit 1
    fi
    
    if [[ "$format" == "--json" ]]; then
        cat "$runtime_file"
    else
        echo "Pi-hole Resource Information:"
        echo "=============================="
        jq -r '
            "Startup Order: \(.startup_order)
Dependencies: \(.dependencies | join(", "))
Startup Timeout: \(.startup_timeout)s
Startup Time: \(.startup_time_estimate)
Recovery Attempts: \(.recovery_attempts)
Priority: \(.priority)"
        ' "$runtime_file"
    fi
}

# Handle lifecycle management
handle_manage() {
    local subcmd="${1:-}"
    shift || true
    
    case "$subcmd" in
        install)
            install_pihole "$@"
            ;;
        start)
            start_pihole "$@"
            ;;
        stop)
            stop_pihole "$@"
            ;;
        restart)
            restart_pihole "$@"
            ;;
        uninstall)
            uninstall_pihole "$@"
            ;;
        *)
            echo "Error: Unknown manage command: $subcmd" >&2
            echo "Valid commands: install, start, stop, restart, uninstall" >&2
            exit 1
            ;;
    esac
}

# Handle test commands
handle_test() {
    local subcmd="${1:-all}"
    shift || true
    
    case "$subcmd" in
        smoke)
            test_smoke "$@"
            ;;
        integration)
            test_integration "$@"
            ;;
        unit)
            test_unit "$@"
            ;;
        all)
            test_all "$@"
            ;;
        *)
            echo "Error: Unknown test command: $subcmd" >&2
            echo "Valid commands: smoke, integration, unit, all" >&2
            exit 1
            ;;
    esac
}

# Handle content management
handle_content() {
    local subcmd="${1:-}"
    shift || true
    
    case "$subcmd" in
        stats)
            show_stats "$@"
            ;;
        update)
            update_blocklists "$@"
            ;;
        blacklist)
            manage_blacklist "$@"
            ;;
        whitelist)
            manage_whitelist "$@"
            ;;
        logs)
            query_logs "$@"
            ;;
        disable)
            disable_blocking "$@"
            ;;
        enable)
            enable_blocking "$@"
            ;;
        *)
            echo "Error: Unknown content command: $subcmd" >&2
            echo "Valid commands: stats, update, blacklist, whitelist, logs, disable, enable" >&2
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"