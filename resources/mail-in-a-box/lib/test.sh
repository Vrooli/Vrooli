#!/bin/bash

# Mail-in-a-Box Test Library
# Implements v2.0 contract test commands

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
TEST_DIR="$RESOURCE_DIR/test"

# Source core functions
source "$SCRIPT_DIR/core.sh"

# Test phase execution
run_test_phase() {
    local phase="${1:-all}"
    
    case "$phase" in
        smoke|unit|integration|all)
            "$TEST_DIR/run-tests.sh" "$phase"
            ;;
        *)
            echo "Unknown test phase: $phase"
            echo "Available phases: smoke, unit, integration, all"
            return 1
            ;;
    esac
}

# Main test handler
handle_test_command() {
    local subcommand="${1:-all}"
    
    case "$subcommand" in
        smoke)
            echo "Running smoke tests..."
            run_test_phase "smoke"
            ;;
        unit)
            echo "Running unit tests..."
            run_test_phase "unit"
            ;;
        integration)
            echo "Running integration tests..."
            run_test_phase "integration"
            ;;
        all)
            echo "Running all tests..."
            run_test_phase "all"
            ;;
        help)
            cat << EOF
Mail-in-a-Box Test Commands

Usage: resource-mail-in-a-box test <phase>

Phases:
  smoke        Quick health validation (<30s)
  unit         Library function tests (<60s)
  integration  End-to-end functionality tests (<120s)
  all          Run all test phases (default)
  help         Show this help message

Examples:
  resource-mail-in-a-box test smoke
  resource-mail-in-a-box test all
EOF
            ;;
        *)
            echo "Unknown test command: $subcommand"
            echo "Use 'resource-mail-in-a-box test help' for available commands"
            return 1
            ;;
    esac
}

# Export for CLI use
export -f run_test_phase
export -f handle_test_command