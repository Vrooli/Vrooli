#!/bin/bash

# SimPy Resource - CLI Interface
set -euo pipefail

# Get the script directory
SIMPY_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SIMPY_LIB_DIR="$SIMPY_CLI_DIR/lib"

# Source utilities
source "$SIMPY_CLI_DIR/../../../lib/utils/log.sh"

# Command handling
case "${1:-help}" in
    install)
        "$SIMPY_LIB_DIR/install.sh" install
        ;;
    uninstall)
        "$SIMPY_LIB_DIR/uninstall.sh" uninstall
        ;;
    status)
        shift
        "$SIMPY_LIB_DIR/status.sh" "$@"
        ;;
    start)
        "$SIMPY_LIB_DIR/start.sh"
        ;;
    stop)
        log::info "SimPy is a library resource - no service to stop"
        ;;
    run)
        shift
        "$SIMPY_LIB_DIR/run.sh" "$@"
        ;;
    inject)
        shift
        "$SIMPY_LIB_DIR/inject.sh" "$@"
        ;;
    list-simulations)
        "$SIMPY_LIB_DIR/list.sh" simulations
        ;;
    list-outputs)
        "$SIMPY_LIB_DIR/list.sh" outputs
        ;;
    help|--help|-h)
        cat <<EOF
SimPy Resource CLI

Usage: $(basename "$0") [command] [options]

Commands:
  install           Install SimPy and dependencies
  uninstall         Uninstall SimPy
  status [--json]   Show SimPy status
  start             Start SimPy (no-op for library)
  stop              Stop SimPy (no-op for library)
  run <file>        Run a simulation file
  inject <file>     Add a simulation to the library
  list-simulations  List available simulations
  list-outputs      List simulation outputs
  help              Show this help message

Examples:
  $(basename "$0") install
  $(basename "$0") status --json
  $(basename "$0") run examples/basic_queue.py
  $(basename "$0") inject my_simulation.py
EOF
        ;;
    *)
        log::error "Unknown command: $1"
        "$0" help
        exit 1
        ;;
esac