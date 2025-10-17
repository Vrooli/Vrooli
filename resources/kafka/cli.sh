#!/bin/bash
# Apache Kafka Resource CLI
# v2.0 Contract Implementation

set -e

# Get the directory of this script (resolve symlinks)
SCRIPT_PATH="$(readlink -f "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$(dirname "$SCRIPT_PATH")"
RESOURCE_NAME="kafka"

# Source configuration
source "$SCRIPT_DIR/config/defaults.sh"

# Source library functions
source "$SCRIPT_DIR/lib/core.sh"
source "$SCRIPT_DIR/lib/test.sh"

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
        credentials)
            show_credentials "$@"
            ;;
        metrics)
            show_metrics "$@"
            ;;
        *)
            echo "Error: Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Show help information
show_help() {
    cat << EOF
Apache Kafka Resource CLI - v2.0

Usage: resource-kafka [COMMAND] [OPTIONS]

Commands:
  help                Show this help message
  info [--json]       Show resource information from runtime.json
  manage              Lifecycle management commands
    install           Install Kafka and dependencies
    start [--wait]    Start Kafka broker
    stop              Stop Kafka broker
    restart           Restart Kafka broker
    uninstall         Remove Kafka completely
  test                Run validation tests
    smoke             Quick health check (<30s)
    integration       Full functionality test (<120s)
    unit              Test library functions (<60s)
    all               Run all tests
  content             Manage Kafka topics and messages
    add [topic]       Create a new topic
    list              List all topics
    get [topic]       Describe a topic
    remove [topic]    Delete a topic
    execute [cmd]     Execute Kafka CLI command
    produce-batch     Produce batch messages
    consume-batch     Consume batch messages
  status [--json]     Show detailed status
  logs [--tail N]     View Kafka logs
  credentials         Show connection details
  metrics             Show performance and JMX metrics

Examples:
  resource-kafka manage start --wait
  resource-kafka test smoke
  resource-kafka content add my-topic --partitions 3
  resource-kafka content list
  resource-kafka content execute "kafka-topics.sh --list"
  resource-kafka status --json

Default Configuration:
  Broker Port: $KAFKA_PORT
  Controller Port: $KAFKA_CONTROLLER_PORT
  External Port: $KAFKA_EXTERNAL_PORT
  Data Directory: $KAFKA_DATA_DIR
  Log Level: $KAFKA_LOG_LEVEL

For more information, see: resources/kafka/README.md
EOF
}

# Show resource information
show_info() {
    local json_flag=""
    
    for arg in "$@"; do
        case "$arg" in
            --json)
                json_flag="--json"
                ;;
        esac
    done
    
    if [ -f "$SCRIPT_DIR/config/runtime.json" ]; then
        if [ "$json_flag" == "--json" ]; then
            cat "$SCRIPT_DIR/config/runtime.json"
        else
            echo "=== Kafka Resource Information ==="
            echo "Startup Order: $(jq -r '.startup_order' "$SCRIPT_DIR/config/runtime.json")"
            echo "Dependencies: $(jq -r '.dependencies | join(", ")' "$SCRIPT_DIR/config/runtime.json")"
            echo "Startup Timeout: $(jq -r '.startup_timeout' "$SCRIPT_DIR/config/runtime.json")s"
            echo "Startup Time Estimate: $(jq -r '.startup_time_estimate' "$SCRIPT_DIR/config/runtime.json")"
            echo "Recovery Attempts: $(jq -r '.recovery_attempts' "$SCRIPT_DIR/config/runtime.json")"
            echo "Priority: $(jq -r '.priority' "$SCRIPT_DIR/config/runtime.json")"
            echo "Category: $(jq -r '.category' "$SCRIPT_DIR/config/runtime.json")"
        fi
    else
        echo "Error: runtime.json not found"
        exit 1
    fi
}

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            install_kafka "$@"
            ;;
        start)
            start_kafka "$@"
            ;;
        stop)
            stop_kafka "$@"
            ;;
        restart)
            restart_kafka "$@"
            ;;
        uninstall)
            uninstall_kafka "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand"
            echo "Available: install, start, stop, restart, uninstall"
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
            test_smoke
            ;;
        integration)
            test_integration
            ;;
        unit)
            test_unit
            ;;
        all)
            test_all
            ;;
        *)
            echo "Error: Unknown test subcommand: $subcommand"
            echo "Available: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Handle content subcommands
handle_content() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        add)
            add_topic "$@"
            ;;
        list)
            list_topics "$@"
            ;;
        get)
            describe_topic "$@"
            ;;
        remove)
            remove_topic "$@"
            ;;
        execute)
            execute_command "$@"
            ;;
        produce-batch)
            produce_batch "$@"
            ;;
        consume-batch)
            consume_batch "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand: $subcommand"
            echo "Available: add, list, get, remove, execute, produce-batch, consume-batch"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"