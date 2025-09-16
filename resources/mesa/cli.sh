#!/usr/bin/env bash
# Mesa Agent-Based Simulation Framework CLI
# v2.0 Universal Contract Compliant

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly LIB_DIR="${SCRIPT_DIR}/lib"

# Source required libraries
source "${LIB_DIR}/core.sh"
source "${LIB_DIR}/test.sh"

# Main command handler
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help)
            mesa::show_help
            ;;
        info)
            mesa::show_info "$@"
            ;;
        manage)
            mesa::manage "$@"
            ;;
        test)
            mesa::test "$@"
            ;;
        status)
            mesa::status "$@"
            ;;
        logs)
            mesa::logs "$@"
            ;;
        content)
            mesa::content "$@"
            ;;
        *)
            echo "Unknown command: $command"
            echo "Run 'resource-mesa help' for usage information"
            exit 1
            ;;
    esac
}

# Show help information
mesa::show_help() {
    cat << EOF
Mesa Agent-Based Simulation Framework

USAGE:
    resource-mesa <COMMAND> [OPTIONS]

COMMANDS:
    help                    Show this help message
    info                    Show resource information
    manage <SUBCOMMAND>     Lifecycle management
        install            Install Mesa and dependencies
        start             Start Mesa service
        stop              Stop Mesa service
        restart           Restart Mesa service
        uninstall         Remove Mesa completely
    test <PHASE>           Run tests
        smoke             Quick health check
        integration       Full functionality test
        unit              Library function tests
        all               Run all tests
    content <SUBCOMMAND>    Manage simulations
        list              List available models
        execute <model>   Run a simulation model
        add <file>        Add custom model
        remove <model>    Remove a model
    status                 Show service status
    logs                   View service logs

EXAMPLES:
    # Install and start Mesa
    resource-mesa manage install
    resource-mesa manage start --wait
    
    # Run example simulation
    resource-mesa content execute schelling
    
    # Check status
    resource-mesa status
    
    # Run tests
    resource-mesa test all

DEFAULT CONFIGURATION:
    Port: 9512 (from port_registry.sh)
    API: http://localhost:9512
    Health: http://localhost:9512/health

For more information, see resources/mesa/README.md
EOF
    return 0
}

# Delegate to appropriate function
main "$@"