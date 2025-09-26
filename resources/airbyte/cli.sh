#!/bin/bash
# Airbyte Resource CLI - v2.0 Contract Compliant
# ELT data integration platform with 600+ connectors

set -euo pipefail

# Resolve symlinks to get actual resource directory
SCRIPT_PATH="${BASH_SOURCE[0]}"
while [ -h "$SCRIPT_PATH" ]; do
    SCRIPT_DIR="$(cd -P "$(dirname "$SCRIPT_PATH")" && pwd)"
    SCRIPT_PATH="$(readlink "$SCRIPT_PATH")"
    [[ $SCRIPT_PATH != /* ]] && SCRIPT_PATH="$SCRIPT_DIR/$SCRIPT_PATH"
done
SCRIPT_DIR="$(cd -P "$(dirname "$SCRIPT_PATH")" && pwd)"

source "${SCRIPT_DIR}/lib/core.sh"

# Display help information
show_help() {
    cat << EOF
Airbyte - Open-source ELT data integration platform

USAGE:
    resource-airbyte <command> [options]

COMMANDS:
    help                          Show this help message
    info                          Show resource information
    
    manage install                Install Airbyte and dependencies
    manage start                  Start Airbyte services
    manage stop                   Stop Airbyte services  
    manage restart                Restart Airbyte services
    manage uninstall              Remove Airbyte completely
    
    test smoke                    Quick health check (<30s)
    test integration              Full integration tests (<120s)
    test unit                     Test library functions (<60s)
    test all                      Run all test suites
    
    content list                  List sources, destinations, or connections
    content add                   Add a new connector or connection
    content get                   Get connector/connection details
    content remove                Remove a connector or connection
    content execute               Trigger a sync job
    
    status                        Show service status
    logs                          View service logs
    
    credentials show              Display API credentials
    credentials store             Store credential securely
    credentials list              List stored credentials
    credentials remove            Remove a credential
    
    schedule create               Create a sync schedule
    schedule list                 List all schedules
    schedule enable               Enable a schedule
    schedule disable              Disable a schedule
    
    webhook register              Register a webhook
    webhook list                  List all webhooks
    webhook test                  Test a webhook
    webhook stats                 Show webhook statistics
    
    transform init                Initialize DBT project
    transform install             Install DBT dependencies
    transform create              Create a transformation model
    transform run                 Run transformations
    transform list                List available transformations
    transform apply               Apply transformation to connection
    
    pipeline performance          Monitor sync performance metrics
    pipeline optimize             Optimize sync configuration
    pipeline quality              Analyze data quality
    pipeline batch                Execute batch syncs
    pipeline resources            Analyze resource usage
    
    cdk init                      Initialize custom connector
    cdk build                     Build connector Docker image
    cdk test                      Test custom connector
    cdk deploy                    Deploy connector to Airbyte
    cdk list                      List custom connectors
    
    workspace create              Create new workspace
    workspace list                List all workspaces
    workspace switch              Switch active workspace
    workspace delete              Delete a workspace
    workspace export              Export workspace configuration
    
    metrics enable                Enable Prometheus metrics
    metrics disable               Disable metrics export
    metrics status                Check metrics configuration
    metrics export                Export current metrics
    metrics dashboard             Show metrics dashboard

EXAMPLES:
    # Start Airbyte
    resource-airbyte manage start
    
    # List available source connectors
    resource-airbyte content list --type sources
    
    # Create a new connection
    resource-airbyte content add --type connection --config connection.json
    
    # Trigger a sync
    resource-airbyte content execute --connection-id my-connection
    
    # View server logs
    resource-airbyte logs --service server

DEFAULT CONFIGURATION:
    Webapp Port: 8002
    API Port: 8003
    Temporal Port: 8006
    Data Directory: ./data

For more information, see: /home/matthalloran8/Vrooli/resources/airbyte/README.md
EOF
}

# Main command dispatcher
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        help|--help|-h|"")
            show_help
            ;;
        info)
            cmd_info "$@"
            ;;
        manage)
            cmd_manage "$@"
            ;;
        test)
            cmd_test "$@"
            ;;
        content)
            cmd_content "$@"
            ;;
        status)
            cmd_status "$@"
            ;;
        logs)
            cmd_logs "$@"
            ;;
        credentials)
            cmd_credentials "$@"
            ;;
        schedule)
            cmd_schedule "$@"
            ;;
        webhook)
            cmd_webhook "$@"
            ;;
        transform)
            cmd_transform "$@"
            ;;
        pipeline)
            cmd_pipeline "$@"
            ;;
        cdk)
            source "${SCRIPT_DIR}/lib/cdk.sh"
            cmd_cdk "$@"
            ;;
        workspace)
            source "${SCRIPT_DIR}/lib/workspace.sh"
            cmd_workspace "$@"
            ;;
        metrics)
            source "${SCRIPT_DIR}/lib/metrics.sh"
            cmd_metrics "$@"
            ;;
        *)
            echo "Error: Unknown command: $command" >&2
            echo "Run 'resource-airbyte help' for usage information" >&2
            exit 1
            ;;
    esac
}

main "$@"