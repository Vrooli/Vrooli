#!/usr/bin/env bash

# Node-RED management script for Vrooli
# This script provides installation, configuration, and management of Node-RED

set -euo pipefail

# Handle Ctrl+C gracefully
trap 'log::info "\nInterrupted. Exiting..."; exit 130' INT TERM

# Get the directory of this script (unique variable name)
NODE_RED_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NODE_RED_LIB_DIR="${NODE_RED_SCRIPT_DIR}/lib"

# Source var.sh first to get standard directory variables
# shellcheck disable=SC1091
source "${NODE_RED_SCRIPT_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"

# Source configuration
# shellcheck disable=SC1091
source "${NODE_RED_SCRIPT_DIR}/config/defaults.sh"

# Export configuration
node_red::export_config

# Source refactored library modules
# Core module contains most shared functionality
# shellcheck disable=SC1091
source "${NODE_RED_LIB_DIR}/core.sh"
# shellcheck disable=SC1091
source "${NODE_RED_LIB_DIR}/docker.sh"
# shellcheck disable=SC1091
source "${NODE_RED_LIB_DIR}/health.sh"
# shellcheck disable=SC1091
source "${NODE_RED_LIB_DIR}/recovery.sh"
# Note: status.sh and api.sh functions are now in core.sh

#######################################
# Display usage information
#######################################
node_red::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  # Install with custom image"
    echo "  $0 --action install --build-image yes"
    echo
    echo "  # Import flows from file"
    echo "  $0 --action flow-import --flow-file ./my-flows.json"
    echo
    echo "  # Execute test flow"
    echo "  $0 --action flow-execute --endpoint /test/exec --data '{\"command\":\"ls\"}'"
    echo
    echo "  # Export all flows"
    echo "  $0 --action flow-export --output ./backup-flows.json"
    echo
}

#######################################
# Parse command line arguments
#######################################
node_red::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|flow-list|flow-export|flow-import|flow-execute|flow-enable|flow-disable|test|metrics|info|health|monitor|inject|validate-injection" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Node-RED appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "build-image" \
        --desc "Build custom Docker image" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    args::register \
        --name "flow-file" \
        --desc "Path to flow file for import" \
        --type "value" \
        --default ""
    
    args::register \
        --name "flow-id" \
        --desc "Flow ID for flow operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "endpoint" \
        --desc "HTTP endpoint for flow execution" \
        --type "value" \
        --default ""
    
    args::register \
        --name "data" \
        --desc "JSON data for flow execution" \
        --type "value" \
        --default ""
    
    args::register \
        --name "output" \
        --desc "Output file for export operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "follow" \
        --desc "Follow logs in real-time" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "lines" \
        --desc "Number of log lines to show" \
        --type "value" \
        --default "100"
    
    args::register \
        --name "interval" \
        --desc "Monitoring interval in seconds" \
        --type "value" \
        --default "5"
    
    
    args::register \
        --name "backup-path" \
        --desc "Path to backup directory for restore operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "injection-config" \
        --desc "JSON configuration for data injection" \
        --type "value" \
        --default ""
    
    if args::is_asking_for_help "$@"; then
        node_red::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export YES=$(args::get "yes")
    export FORCE=$(args::get "force")
    export BUILD_IMAGE=$(args::get "build-image")
    export FLOW_FILE=$(args::get "flow-file")
    export FLOW_ID=$(args::get "flow-id")
    export ENDPOINT=$(args::get "endpoint")
    export DATA=$(args::get "data")
    export OUTPUT=$(args::get "output")
    export FOLLOW=$(args::get "follow")
    export LOG_LINES=$(args::get "lines")
    export INTERVAL=$(args::get "interval")
    export BACKUP_PATH=$(args::get "backup-path")
    export INJECTION_CONFIG=$(args::get "injection-config")
}

#######################################
# Main execution function
#######################################
main() {
    node_red::parse_arguments "$@"
    
    case "$ACTION" in
        install)
            node_red::install
            ;;
        uninstall)
            node_red::uninstall
            ;;
        start)
            node_red::start
            ;;
        stop)
            node_red::stop
            ;;
        restart)
            node_red::restart
            ;;
        status)
            node_red::status
            ;;
        logs)
            node_red::view_logs "$LOG_LINES" "$FOLLOW"
            ;;
        info)
            log::info "Node-RED Service Information"
            echo "Service: Node-RED Visual Programming"
            echo "Port: $NODE_RED_PORT"
            echo "Container: $NODE_RED_CONTAINER_NAME"
            echo "Data Directory: $NODE_RED_DATA_DIR"
            echo "Access URL: http://localhost:$NODE_RED_PORT"
            ;;
        health)
            node_red::health
            ;;
        monitor)
            log::info "Starting Node-RED monitoring (interval: ${INTERVAL}s)"
            while true; do
                clear
                echo "Node-RED Monitor - $(date)"
                echo "========================"
                node_red::status
                sleep "$INTERVAL"
            done
            ;;
        flow-list)
            node_red::flow_operation "list"
            ;;
        flow-export)
            node_red::flow_operation "export" "$OUTPUT"
            ;;
        flow-import)
            node_red::flow_operation "import" "$FLOW_FILE"
            ;;
        flow-execute)
            node_red::flow_operation "execute" "$ENDPOINT" "$DATA"
            ;;
        flow-enable)
            node_red::flow_operation "enable" "$FLOW_ID"
            ;;
        flow-disable)
            node_red::flow_operation "disable" "$FLOW_ID"
            ;;
        test)
            "${NODE_RED_SCRIPT_DIR}/test/integration-test.sh"
            ;;
        metrics)
            node_red::get_status_json | jq .
            ;;
        backup)
            node_red::create_backup "manual-$(date +%Y%m%d-%H%M%S)"
            ;;
        restore)
            node_red::restore_backup "$BACKUP_PATH"
            ;;
        inject)
            node_red::inject
            ;;
        validate-injection)
            node_red::validate_injection
            ;;
        *)
            log::error "Unknown action: $ACTION"
            node_red::usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"