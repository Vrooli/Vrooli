#!/bin/bash

# OpenFOAM Resource CLI Interface
# Implements v2.0 contract for CFD simulation platform

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$SCRIPT_DIR"

# Source libraries
source "$RESOURCE_DIR/config/defaults.sh"
source "$RESOURCE_DIR/lib/core.sh"
source "$RESOURCE_DIR/lib/test.sh" 2>/dev/null || true

# CLI handler
openfoam::cli() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help)
            openfoam::cli::help
            ;;
        info)
            openfoam::cli::info
            ;;
        manage)
            openfoam::cli::manage "$@"
            ;;
        test)
            openfoam::cli::test "$@"
            ;;
        content)
            openfoam::cli::content "$@"
            ;;
        status)
            openfoam::status
            ;;
        logs)
            openfoam::cli::logs "$@"
            ;;
        credentials)
            openfoam::cli::credentials
            ;;
        *)
            echo "Error: Unknown command: $command"
            echo "Run 'resource-openfoam help' for usage information"
            return 1
            ;;
    esac
}

# Help command
openfoam::cli::help() {
    cat << EOF
OpenFOAM CFD Simulation Platform

USAGE:
    resource-openfoam <command> [options]

COMMANDS:
    help        Show this help message
    info        Display runtime configuration
    manage      Lifecycle management commands
    test        Run validation tests
    content     Manage simulation cases
    status      Show detailed status
    logs        View resource logs
    credentials Display integration credentials

MANAGE COMMANDS:
    manage install    Install dependencies
    manage start      Start OpenFOAM container
    manage stop       Stop OpenFOAM container
    manage restart    Restart OpenFOAM
    manage uninstall  Remove OpenFOAM

TEST COMMANDS:
    test smoke        Quick health validation (<30s)
    test integration  End-to-end functionality test
    test unit         Library function tests
    test all          Run all test phases

CONTENT COMMANDS:
    content list              List available cases
    content add <name> [type] Create new case
    content get <name>        View case details
    content remove <name>     Delete case
    content execute <name>    Run simulation

EXAMPLES:
    # Start OpenFOAM
    resource-openfoam manage start
    
    # Create and run a simulation
    resource-openfoam content add cavity
    resource-openfoam content execute cavity
    
    # Check status
    resource-openfoam status
    
    # Run tests
    resource-openfoam test smoke

ENVIRONMENT:
    OPENFOAM_PORT          API port (default: 8090)
    OPENFOAM_MEMORY_LIMIT  Memory limit (default: 4g)
    OPENFOAM_CPU_LIMIT     CPU limit (default: 2)

For more information, see: resources/openfoam/README.md
EOF
}

# Info command
openfoam::cli::info() {
    cat << EOF
OpenFOAM Runtime Configuration
==============================
Service:     openfoam
Version:     v2312 (ESI OpenCFD)
Category:    simulation
Port:        ${OPENFOAM_PORT:-8090}
Status:      $(openfoam::docker::is_running && echo "Running" || echo "Stopped")

Resources:
  Memory:    ${OPENFOAM_MEMORY_LIMIT:-4g}
  CPU:       ${OPENFOAM_CPU_LIMIT:-2} cores
  Storage:   ${OPENFOAM_STORAGE_LIMIT:-20G}

Features:
  Solvers:   simpleFoam, pimpleFoam, interFoam, buoyantFoam
  Mesh:      blockMesh, snappyHexMesh
  Parallel:  ${OPENFOAM_ENABLE_MPI:-false}
  GPU:       ${OPENFOAM_ENABLE_GPU:-false}

Paths:
  Cases:     ${OPENFOAM_CASES_DIR}
  Results:   ${OPENFOAM_RESULTS_DIR}
  Logs:      ${OPENFOAM_LOG_DIR}

API Endpoints:
  Health:    http://localhost:${OPENFOAM_PORT:-8090}/health
  Status:    http://localhost:${OPENFOAM_PORT:-8090}/api/status
EOF
}

# Manage command
openfoam::cli::manage() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        install)
            openfoam::install "$@"
            ;;
        start)
            openfoam::docker::start "$@"
            if [[ "${1:-}" == "--wait" ]]; then
                echo "Waiting for OpenFOAM to be healthy..."
                local attempts=0
                while ! openfoam::health::check && [ $attempts -lt 30 ]; do
                    sleep 2
                    attempts=$((attempts + 1))
                done
                openfoam::health::check || {
                    echo "Error: OpenFOAM failed to become healthy"
                    return 1
                }
                echo "OpenFOAM is healthy"
            fi
            ;;
        stop)
            openfoam::docker::stop "$@"
            ;;
        restart)
            openfoam::docker::stop
            sleep 2
            openfoam::docker::start "$@"
            ;;
        uninstall)
            openfoam::uninstall "$@"
            ;;
        *)
            echo "Error: Unknown manage action: $action"
            echo "Valid actions: install, start, stop, restart, uninstall"
            return 1
            ;;
    esac
}

# Test command
openfoam::cli::test() {
    local phase="${1:-all}"
    
    case "$phase" in
        smoke)
            openfoam::test::smoke
            ;;
        integration)
            openfoam::test::integration
            ;;
        unit)
            openfoam::test::unit
            ;;
        all)
            openfoam::test::all
            ;;
        *)
            echo "Error: Unknown test phase: $phase"
            echo "Valid phases: smoke, integration, unit, all"
            return 1
            ;;
    esac
}

# Content command
openfoam::cli::content() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            openfoam::content::list "$@"
            ;;
        add)
            openfoam::content::add "$@"
            ;;
        get)
            openfoam::content::get "$@"
            ;;
        remove)
            openfoam::content::remove "$@"
            ;;
        execute)
            openfoam::content::execute "$@"
            ;;
        *)
            echo "Error: Unknown content action: $action"
            echo "Valid actions: list, add, get, remove, execute"
            return 1
            ;;
    esac
}

# Logs command
openfoam::cli::logs() {
    local lines="${1:-50}"
    
    if openfoam::docker::is_running; then
        echo "=== OpenFOAM Container Logs ==="
        docker logs --tail "$lines" openfoam 2>&1 || true
    else
        echo "OpenFOAM is not running"
    fi
    
    # Show application logs if available
    if [[ -d "${OPENFOAM_LOG_DIR}" ]]; then
        echo ""
        echo "=== Application Logs ==="
        find "${OPENFOAM_LOG_DIR}" -type f -name "*.log" -exec tail -n "$lines" {} \; 2>/dev/null || true
    fi
}

# Credentials command
openfoam::cli::credentials() {
    cat << EOF
OpenFOAM Integration Credentials
================================
Service URL:  http://localhost:${OPENFOAM_PORT:-8090}
Health Check: http://localhost:${OPENFOAM_PORT:-8090}/health
API Base:     http://localhost:${OPENFOAM_PORT:-8090}/api

No authentication required for local access.

Example API Usage:
  # Create case
  curl -X POST http://localhost:${OPENFOAM_PORT:-8090}/api/case/create \\
    -H "Content-Type: application/json" \\
    -d '{"name": "test_case", "type": "cavity"}'
  
  # Run simulation
  curl -X POST http://localhost:${OPENFOAM_PORT:-8090}/api/solver/run \\
    -H "Content-Type: application/json" \\
    -d '{"case": "test_case", "solver": "simpleFoam"}'

Integration with other resources:
  - Export geometries from Blender/FreeCAD as STL
  - Store results in Minio for large datasets
  - Index patterns in Qdrant for ML analysis
EOF
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    openfoam::cli "$@"
fi