#!/usr/bin/env bash
# Mathlib (Lean) Resource CLI
# Universal v2.0 contract compliant interface

set -euo pipefail

# Resource identification
readonly RESOURCE_NAME="mathlib"
readonly RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source configurations and libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

# Display help information
show_help() {
    cat << EOF
Mathlib (Lean) Resource - Formal Mathematical Theorem Proving

USAGE:
    vrooli resource mathlib <command> [options]

COMMANDS:
    help                Show this help message
    info                Display resource configuration from runtime.json
    manage              Lifecycle management commands
        install         Install Lean 4 and dependencies
        uninstall       Remove Lean 4 and Mathlib
        start           Start the Mathlib service
        stop            Stop the Mathlib service
        restart         Restart the service
    test                Run validation tests
        smoke           Quick health check (< 30s)
        integration     Full functionality test (< 120s)
        unit            Library function tests (< 60s)
        all             Run all test suites (< 600s)
    content             Manage proof content (future)
        list            List available tactics
        execute         Execute a proof file
    status              Show service status
    logs                View service logs
    credentials         Display API credentials (if applicable)

EXAMPLES:
    # Install and start the resource
    vrooli resource mathlib manage install
    vrooli resource mathlib manage start

    # Check service health
    vrooli resource mathlib status
    vrooli resource mathlib test smoke

    # View logs
    vrooli resource mathlib logs --tail 50

CONFIGURATION:
    Port:           ${MATHLIB_PORT}
    Work Directory: ${MATHLIB_WORK_DIR}
    Cache:          ${MATHLIB_CACHE_DIR}
    Timeout:        ${MATHLIB_TIMEOUT}s

For more information, see: resources/mathlib/README.md
EOF
}

# Main command router
main() {
    local command="${1:-}"
    shift || true

    case "${command}" in
        help|--help|-h|"")
            show_help
            exit 0
            ;;
        info)
            mathlib::info "$@"
            ;;
        manage)
            mathlib::manage "$@"
            ;;
        test)
            mathlib::test "$@"
            ;;
        content)
            mathlib::content "$@"
            ;;
        status)
            mathlib::status "$@"
            ;;
        logs)
            mathlib::logs "$@"
            ;;
        credentials)
            mathlib::credentials "$@"
            ;;
        *)
            echo "Error: Unknown command '${command}'"
            echo "Run 'vrooli resource mathlib help' for usage information"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"