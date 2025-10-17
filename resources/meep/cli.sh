#!/usr/bin/env bash
# MEEP Resource CLI - v2.0 Contract Compliant
# MIT Electromagnetic Equation Propagation FDTD Solver

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

# Source configuration and libraries
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Resource metadata
readonly RESOURCE_NAME="meep"
readonly RESOURCE_VERSION="1.0.0"
readonly RESOURCE_DESCRIPTION="MEEP electromagnetic FDTD simulation engine"

#######################################
# Show help message
#######################################
show_help() {
    cat << EOF
MEEP Electromagnetic Simulation Engine
======================================

DESCRIPTION:
    ${RESOURCE_DESCRIPTION}
    
    MEEP (MIT Electromagnetic Equation Propagation) is an open-source
    FDTD solver for electromagnetic simulations with Python bindings.

USAGE:
    resource-${RESOURCE_NAME} <command> [options]

COMMANDS:
    help                Show this help message
    info                Show resource information from runtime.json
    manage              Lifecycle management commands
    test                Run validation tests
    content             Manage simulation templates and examples
    status              Show service status
    logs                View service logs

MANAGE SUBCOMMANDS:
    manage install      Install MEEP and dependencies
    manage start        Start MEEP service
    manage stop         Stop MEEP service
    manage restart      Restart MEEP service
    manage uninstall    Remove MEEP and clean up

TEST SUBCOMMANDS:
    test smoke          Quick health check (<30s)
    test integration    Full functionality test (<120s)
    test unit           Test library functions (<60s)
    test all            Run all test phases

CONTENT SUBCOMMANDS:
    content list        List available simulation templates
    content add <file>  Add new simulation template
    content get <name>  Get simulation template
    content remove <name> Remove simulation template
    content execute <name> Run simulation template

EXAMPLES:
    # Start MEEP service
    resource-${RESOURCE_NAME} manage start --wait

    # Run a waveguide simulation
    resource-${RESOURCE_NAME} content execute waveguide

    # Check service health
    resource-${RESOURCE_NAME} test smoke

    # View service logs
    resource-${RESOURCE_NAME} logs --tail 50

DEFAULT CONFIGURATION:
    Port: ${MEEP_PORT}
    Data Directory: ${MEEP_DATA_DIR}
    MPI Processes: ${MEEP_MPI_PROCESSES}
    Resolution: ${MEEP_DEFAULT_RESOLUTION}

ENVIRONMENT VARIABLES:
    MEEP_PORT              API server port (default: 8193)
    MEEP_MPI_PROCESSES     Number of MPI processes (default: 4)
    MEEP_GPU_ENABLED       Enable GPU acceleration (default: false)
    MEEP_DEBUG             Enable debug output (default: false)

EOF
}

#######################################
# Show resource info from runtime.json
#######################################
show_info() {
    local json_format="${1:-false}"
    local runtime_file="${SCRIPT_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        echo "Error: runtime.json not found" >&2
        return 1
    fi
    
    if [[ "$json_format" == "true" ]]; then
        cat "$runtime_file"
    else
        echo "MEEP Resource Information"
        echo "========================="
        jq -r '
            "Startup Order: \(.startup_order)",
            "Dependencies: \(.dependencies | join(", "))",
            "Optional Dependencies: \(.optional_dependencies | join(", "))",
            "Startup Timeout: \(.startup_timeout)s",
            "Startup Time Estimate: \(.startup_time_estimate)",
            "Recovery Attempts: \(.recovery_attempts)",
            "Priority: \(.priority)",
            "Category: \(.category)"
        ' "$runtime_file"
    fi
}

#######################################
# Main command router
#######################################
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help|--help|-h)
            show_help
            ;;
        info)
            local json_flag=false
            [[ "${1:-}" == "--json" ]] && json_flag=true
            show_info "$json_flag"
            ;;
        manage)
            local subcommand="${1:-help}"
            shift || true
            meep::manage "$subcommand" "$@"
            ;;
        test)
            local subcommand="${1:-all}"
            shift || true
            meep::test "$subcommand" "$@"
            ;;
        content)
            local subcommand="${1:-list}"
            shift || true
            meep::content "$subcommand" "$@"
            ;;
        status)
            meep::status "$@"
            ;;
        logs)
            meep::logs "$@"
            ;;
        *)
            echo "Error: Unknown command: $command" >&2
            echo "Run 'resource-${RESOURCE_NAME} help' for usage" >&2
            exit 1
            ;;
    esac
}

# Run main function
main "$@"