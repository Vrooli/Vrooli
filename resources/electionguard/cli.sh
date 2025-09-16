#!/bin/bash

# ElectionGuard CLI Interface
# Implements v2.0 Universal Resource Contract
# Provides end-to-end verifiable voting infrastructure

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="electionguard"
RESOURCE_DIR="$SCRIPT_DIR"

# Source configuration and libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Main command handler
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
        *)
            echo "Error: Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Show comprehensive help
show_help() {
    cat << EOF
ElectionGuard Resource - End-to-End Verifiable Voting Infrastructure

USAGE:
    vrooli resource $RESOURCE_NAME <command> [options]

COMMANDS:
    help                Show this help message
    info                Show resource information
    manage              Lifecycle management commands
    test                Run validation tests
    content             Election operations
    status              Show service status
    logs                View service logs

MANAGEMENT COMMANDS:
    manage install      Install ElectionGuard and dependencies
    manage start        Start ElectionGuard service
    manage stop         Stop ElectionGuard service
    manage restart      Restart ElectionGuard service
    manage uninstall    Remove ElectionGuard completely

TEST COMMANDS:
    test smoke          Quick health validation (<30s)
    test integration    Full functionality test (<120s)
    test unit          Library function tests (<60s)
    test all           Run all test phases (<180s)

CONTENT COMMANDS:
    content create-election    Create new election
    content generate-keys      Generate guardian keys
    content encrypt-ballot     Encrypt a ballot
    content compute-tally      Compute election tally
    content verify            Verify ballot receipt
    content export            Export election data
    content execute           Run mock election

EXAMPLES:
    # Install and start ElectionGuard
    vrooli resource $RESOURCE_NAME manage install
    vrooli resource $RESOURCE_NAME manage start

    # Create and run a mock election
    vrooli resource $RESOURCE_NAME content execute mock-election

    # Create real election
    vrooli resource $RESOURCE_NAME content create-election \\
        --name "Board Election 2025" \\
        --guardians 5 \\
        --threshold 3

    # Verify service health
    vrooli resource $RESOURCE_NAME test smoke
    vrooli resource $RESOURCE_NAME status

DEFAULT CONFIGURATION:
    Port: ${ELECTIONGUARD_PORT}
    Vault Integration: ${ELECTIONGUARD_VAULT_ENABLED}
    Database Export: ${ELECTIONGUARD_DB_ENABLED}
    Log Level: ${ELECTIONGUARD_LOG_LEVEL}

For more information, see: https://www.electionguard.vote/
EOF
    return 0
}

# Show resource information
show_info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "ElectionGuard Resource Information:"
        echo "===================================="
        python3 -m json.tool "${SCRIPT_DIR}/config/runtime.json" | grep -E '"(startup_order|dependencies|startup_time_estimate|startup_timeout|recovery_attempts|priority)"' | sed 's/[",]//g' | sed 's/^[[:space:]]*/  /'
    fi
    return 0
}

# Main function call
main "$@"