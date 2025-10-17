#!/bin/bash
# Elmer FEM Resource CLI - v2.0 Contract Compliant
# Provides multiphysics finite element simulation capabilities

set -euo pipefail

# Get absolute path to resource directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_NAME="elmer-fem"

# Source port registry for dynamic port allocation
# shellcheck source=/dev/null
source "${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh"

# Get assigned port from registry
readonly ELMER_PORT="${ELMER_FEM_PORT:-$(ports::get_resource_port 'elmer-fem')}"

# Export for child processes
export ELMER_PORT
export RESOURCE_NAME

# Source core functions
# shellcheck source=/dev/null
source "${SCRIPT_DIR}/lib/core.sh"

# Display comprehensive help
show_help() {
    cat << EOF
Elmer FEM Multiphysics Resource - v2.0

USAGE:
    resource-elmer-fem [COMMAND] [OPTIONS]

COMMANDS:
    help                Show this help message
    info                Display runtime configuration
    
    manage install      Install Elmer FEM and dependencies
    manage start        Start Elmer service
    manage stop         Stop Elmer service  
    manage restart      Restart Elmer service
    manage uninstall    Remove Elmer completely
    
    test smoke          Quick health check (<30s)
    test integration    Full functionality test (<120s)
    test unit           Library function tests (<60s)
    test all            Run all test phases (<600s)
    
    content add         Add simulation case or mesh
    content list        List available cases/examples
    content get         Retrieve simulation results
    content remove      Delete simulation case
    content execute     Run simulation with parameters
    
    status              Show detailed service status
    logs                View Elmer service logs
    
EXAMPLES:
    # Start the service
    resource-elmer-fem manage start --wait
    
    # Run a heat transfer example
    resource-elmer-fem content execute --case heat_transfer
    
    # Check simulation status
    resource-elmer-fem status --json
    
    # Export results
    resource-elmer-fem content get --case heat_001 --format vtk

DEFAULT CONFIGURATION:
    Port: ${ELMER_PORT}
    Data Directory: \${VROOLI_DATA}/elmer-fem
    MPI Processes: 4 (configurable)
    
For more information: https://www.csc.fi/web/elmer
EOF
}

# Main command router
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help)
            show_help
            ;;
        info)
            core::info "$@"
            ;;
        manage)
            core::manage "$@"
            ;;
        test)
            core::test "$@"
            ;;
        content)
            core::content "$@"
            ;;
        status)
            core::status "$@"
            ;;
        logs)
            core::logs "$@"
            ;;
        *)
            echo "ERROR: Unknown command: $command" >&2
            echo "Run 'resource-elmer-fem help' for usage" >&2
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"