#!/usr/bin/env bash
# OpenMRS Clinical Records Platform CLI Interface

set -euo pipefail

# Get the absolute path to this script's directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="openmrs"

# Source the common libraries
source "${SCRIPT_DIR}/../../scripts/lib/utils/log.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Help function
show_help() {
    cat << EOF
OpenMRS Clinical Records Platform Resource

USAGE:
    resource-${RESOURCE_NAME} <command> [options]

COMMANDS:
    help                    Show this help message
    info                    Show runtime configuration
    status                  Show current status
    logs                    View resource logs
    
    manage install          Install OpenMRS and dependencies
    manage start            Start OpenMRS services
    manage stop             Stop OpenMRS services  
    manage restart          Restart OpenMRS services
    manage uninstall        Remove OpenMRS completely
    
    test smoke              Quick health validation (<30s)
    test integration        End-to-end functionality test
    test unit               Library function tests
    test all                Run all test phases
    
    content list            List available content operations
    content add             Add patient/encounter data
    content get             Retrieve clinical records
    content remove          Remove patient data
    content execute         Execute clinical operations

    api patient             Manage patients via REST API
    api encounter           Manage encounters via REST API
    api concept             Manage clinical concepts
    api provider            Manage healthcare providers
    
    fhir export             Export data in FHIR format
    fhir import             Import FHIR resources

EXAMPLES:
    # Install and start OpenMRS
    resource-${RESOURCE_NAME} manage install
    resource-${RESOURCE_NAME} manage start --wait
    
    # Create a new patient
    resource-${RESOURCE_NAME} api patient create --name "John Doe" --gender "M" --birthdate "1990-01-01"
    
    # Create an encounter
    resource-${RESOURCE_NAME} api encounter create --patient-id "abc123" --type "consultation"
    
    # Export data in FHIR format
    resource-${RESOURCE_NAME} fhir export --resource-type "Patient"
    
    # Run health checks
    resource-${RESOURCE_NAME} test smoke

EOF
}

# Main command router
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        help|"")
            show_help
            ;;
        info)
            openmrs::info "$@"
            ;;
        status)
            openmrs::status "$@"
            ;;
        logs)
            openmrs::logs "$@"
            ;;
        manage)
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                install)
                    openmrs::install "$@"
                    ;;
                start)
                    openmrs::start "$@"
                    ;;
                stop)
                    openmrs::stop "$@"
                    ;;
                restart)
                    openmrs::restart "$@"
                    ;;
                uninstall)
                    openmrs::uninstall "$@"
                    ;;
                *)
                    echo "Unknown manage subcommand: $subcommand" >&2
                    show_help
                    exit 1
                    ;;
            esac
            ;;
        test)
            local phase="${1:-all}"
            shift || true
            openmrs::test "$phase" "$@"
            ;;
        content)
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                list|add|get|remove|execute)
                    openmrs::content::${subcommand} "$@"
                    ;;
                *)
                    echo "Unknown content subcommand: $subcommand" >&2
                    exit 1
                    ;;
            esac
            ;;
        api)
            local subcommand="${1:-}"
            shift || true
            openmrs::api::${subcommand} "$@"
            ;;
        fhir)
            local subcommand="${1:-}"
            shift || true
            openmrs::fhir::${subcommand} "$@"
            ;;
        *)
            echo "Unknown command: $command" >&2
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"