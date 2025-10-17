#!/bin/bash
# SU2 Resource CLI Interface
# Provides access to SU2 CFD simulation and optimization capabilities

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="su2"

# Load configuration
source "${SCRIPT_DIR}/config/defaults.sh"

# Load core functionality
source "${SCRIPT_DIR}/lib/core.sh"

# Load test functionality
source "${SCRIPT_DIR}/lib/test.sh"

# Load integration functionality
[[ -f "${SCRIPT_DIR}/lib/integration.sh" ]] && source "${SCRIPT_DIR}/lib/integration.sh"

# Load visualization functionality
[[ -f "${SCRIPT_DIR}/lib/visualization.sh" ]] && source "${SCRIPT_DIR}/lib/visualization.sh"

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
        integrate)
            handle_integrate "$@"
            ;;
        visualize)
            handle_visualize "$@"
            ;;
        *)
            echo "Error: Unknown command: $command" >&2
            echo "Run '$0 help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Show comprehensive help
show_help() {
    cat << EOF
SU2 Aerospace CFD & Optimization Platform

USAGE:
    $0 [COMMAND] [OPTIONS]

COMMANDS:
    help                Show this help message
    info                Show runtime configuration
    manage              Lifecycle management commands
    test                Run validation tests
    content             Content management operations
    status              Show service status
    logs                View service logs
    integrate           Data integration operations
    visualize           Visualization operations

MANAGE SUBCOMMANDS:
    manage install      Install SU2 and dependencies
    manage start        Start SU2 service
    manage stop         Stop SU2 service
    manage restart      Restart SU2 service
    manage uninstall    Remove SU2 installation

TEST SUBCOMMANDS:
    test smoke          Quick health validation (<30s)
    test integration    Full functionality test (<120s)
    test unit           Library function tests (<60s)
    test all            Run all test phases

CONTENT SUBCOMMANDS:
    content list        List available meshes/configs
    content add         Add mesh or configuration
    content get         Retrieve simulation results
    content remove      Remove mesh/config/results
    content execute     Run CFD simulation

INTEGRATE SUBCOMMANDS:
    integrate questdb   Export results to QuestDB
    integrate qdrant    Export to Qdrant vector DB
    integrate batch     Run batch optimization
    integrate export    Export all results

VISUALIZE SUBCOMMANDS:
    visualize openmct   Setup OpenMCT telemetry
    visualize superset  Setup Superset dashboard
    visualize vtk       Generate VTK visualization
    visualize animate   Create convergence animation

EXAMPLES:
    # Start SU2 service
    $0 manage start

    # Run NACA0012 test case
    $0 content execute --mesh naca0012.su2 --config naca0012.cfg

    # Export results to CSV
    $0 content get --id sim_001 --format csv

    # Run smoke tests
    $0 test smoke

CONFIGURATION:
    Port:       ${SU2_PORT}
    Data Dir:   ${SU2_DATA_DIR}
    MPI Procs:  ${SU2_MPI_PROCESSES}
    Version:    ${SU2_VERSION}

For more information, see: resources/su2/README.md
EOF
}

# Show runtime information
show_info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "SU2 Runtime Information:"
        echo "========================"
        jq -r 'to_entries[] | "  \(.key): \(.value)"' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            install_su2 "$@"
            ;;
        start)
            start_su2 "$@"
            ;;
        stop)
            stop_su2 "$@"
            ;;
        restart)
            restart_su2 "$@"
            ;;
        uninstall)
            uninstall_su2 "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# Handle test subcommands
handle_test() {
    local subcommand="${1:-all}"
    shift || true
    
    case "$subcommand" in
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
            echo "Error: Unknown test subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# Handle content operations
handle_content() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        list)
            content_list "$@"
            ;;
        add)
            content_add "$@"
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
            echo "Error: Unknown content subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# Handle integrate subcommands
handle_integrate() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        questdb)
            export_to_questdb "$@"
            ;;
        qdrant)
            export_to_qdrant "$@"
            ;;
        batch)
            run_batch_optimization "$@"
            ;;
        export)
            export_all_results "$@"
            ;;
        search)
            search_similar_designs "$@"
            ;;
        *)
            echo "Error: Unknown integrate subcommand: $subcommand" >&2
            echo "Available: questdb, qdrant, batch, export, search" >&2
            exit 1
            ;;
    esac
}

# Handle visualize subcommands
handle_visualize() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        openmct)
            setup_openmct_telemetry "$@"
            ;;
        superset)
            setup_superset_dashboard "$@"
            ;;
        vtk)
            generate_vtk_visualization "$@"
            ;;
        animate)
            create_convergence_animation "$@"
            ;;
        stream)
            stream_telemetry "$@"
            ;;
        *)
            echo "Error: Unknown visualize subcommand: $subcommand" >&2
            echo "Available: openmct, superset, vtk, animate, stream" >&2
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"