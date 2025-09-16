#!/usr/bin/env bash
# OpenEMR Resource CLI Interface
# Provides comprehensive healthcare EMR and practice management capabilities

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="openemr"

# Source the common libraries
source "${SCRIPT_DIR}/../../scripts/lib/utils/log.sh"
source "${SCRIPT_DIR}/../../scripts/resources/common.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Main CLI router
main() {
    local command="${1:-}"
    shift 1 || true

    case "$command" in
        help|--help|-h|"")
            openemr::help
            ;;
        info)
            openemr::info "$@"
            ;;
        manage)
            openemr::manage "$@"
            ;;
        test)
            openemr::test "$@"
            ;;
        content)
            openemr::content "$@"
            ;;
        status)
            openemr::status "$@"
            ;;
        logs)
            openemr::logs "$@"
            ;;
        credentials)
            openemr::credentials "$@"
            ;;
        *)
            log::error "Unknown command: $command"
            log::info "Run 'resource-${RESOURCE_NAME} help' for usage information"
            return 1
            ;;
    esac
}

# Show comprehensive help
openemr::help() {
    cat << EOF
OpenEMR Resource - Healthcare Information System

USAGE:
    resource-${RESOURCE_NAME} <command> [options]

COMMANDS:
    help                Show this help message
    info [--json]       Show resource information from runtime config
    manage <action>     Lifecycle management commands
    test <phase>        Run validation tests
    content <action>    Manage clinical data and workflows
    status              Show resource status
    logs [options]      View resource logs
    credentials         Display API credentials

MANAGE ACTIONS:
    install            Install OpenEMR and dependencies
    start [--wait]     Start OpenEMR stack
    stop               Stop OpenEMR services
    restart            Restart all services
    uninstall          Remove OpenEMR completely

TEST PHASES:
    smoke              Quick health validation (<30s)
    integration        Full functionality test (<120s)
    unit               Library function tests (<60s)
    all                Run all test phases

CONTENT ACTIONS:
    add <type>         Add clinical data (patient, appointment, encounter)
    list <type>        List clinical data
    get <type> <id>    Get specific record
    remove <type> <id> Delete record
    export <type>      Export data via FHIR

EXAMPLES:
    # Install and start OpenEMR
    resource-${RESOURCE_NAME} manage install
    resource-${RESOURCE_NAME} manage start --wait

    # Create a patient
    resource-${RESOURCE_NAME} content add patient --name "Jane Doe" --dob "1985-03-15"

    # Schedule appointment
    resource-${RESOURCE_NAME} content add appointment --patient 1 --provider "Dr. Smith"

    # Export encounters as FHIR
    resource-${RESOURCE_NAME} content export encounters --format fhir

    # Check status
    resource-${RESOURCE_NAME} status
    resource-${RESOURCE_NAME} test smoke

CONFIGURATION:
    OPENEMR_PORT          Web interface port (default: 8080)
    OPENEMR_API_PORT      REST API port (default: 8082)
    OPENEMR_FHIR_PORT     FHIR API port (default: 8083)
    OPENEMR_DB_PORT       MySQL port (default: 3306)
    OPENEMR_ADMIN_USER    Admin username
    OPENEMR_ADMIN_PASS    Admin password

For more information, see: resources/${RESOURCE_NAME}/README.md
EOF
    return 0
}

# Show resource information
openemr::info() {
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                ;;
        esac
        shift
    done
    
    local runtime_file="${SCRIPT_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        log::error "Runtime configuration not found: $runtime_file"
        return 1
    fi
    
    if [[ "$json_output" == "true" ]]; then
        cat "$runtime_file"
    else
        log::info "OpenEMR Resource Information:"
        jq -r '
            "Startup Order: \(.startup_order)",
            "Dependencies: \(.dependencies | join(", "))",
            "Startup Time: \(.startup_time_estimate)",
            "Startup Timeout: \(.startup_timeout)s",
            "Recovery Attempts: \(.recovery_attempts)",
            "Priority: \(.priority)"
        ' "$runtime_file"
    fi
    
    return 0
}

# Entry point
main "$@"