#!/bin/bash

# OpenEMS CLI - v2.0 Contract Compliant
# Energy management system for distributed energy resources

set -euo pipefail

CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export RESOURCE_NAME="openems"

# Source the resource contract validator (if needed for advanced features)
# source "${CLI_SCRIPT_DIR}/../../scripts/resources/lib/cli-command-framework-v2.sh"

# Load core functionality
source "${CLI_SCRIPT_DIR}/lib/core.sh"
source "${CLI_SCRIPT_DIR}/lib/test.sh"

# Load P1 integrations if available
for integration in superset_dashboards ditto_integration forecast_models; do
    if [[ -f "${CLI_SCRIPT_DIR}/lib/${integration}.sh" ]]; then
        source "${CLI_SCRIPT_DIR}/lib/${integration}.sh"
    fi
done

# Load P2 features if available
for feature in alerts cosimulation; do
    if [[ -f "${CLI_SCRIPT_DIR}/lib/${feature}.sh" ]]; then
        source "${CLI_SCRIPT_DIR}/lib/${feature}.sh"
    fi
done

# Command router
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help)
            openems::show_help
            ;;
        info)
            openems::show_info
            ;;
        manage)
            openems::manage "$@"
            ;;
        test)
            openems::test "$@"
            ;;
        content)
            openems::content "$@"
            ;;
        status)
            openems::status "$@"
            ;;
        logs)
            openems::logs "$@"
            ;;
        credentials)
            openems::credentials "$@"
            ;;
        superset)
            "${CLI_SCRIPT_DIR}/lib/superset_dashboards.sh" "$@"
            ;;
        ditto)
            "${CLI_SCRIPT_DIR}/lib/ditto_integration.sh" "$@"
            ;;
        forecast)
            "${CLI_SCRIPT_DIR}/lib/forecast_models.sh" "$@"
            ;;
        # P2 Feature Commands
        alerts)
            openems::alerts "$@"
            ;;
        cosim|cosimulation)
            openems::cosim "$@"
            ;;
        metrics)
            openems::metrics "$@"
            ;;
        benchmark)
            openems::benchmark "$@"
            ;;
        *)
            echo "❌ Unknown command: $command"
            openems::show_help
            exit 1
            ;;
    esac
}

# P2 Alert automation command handler
openems::alerts() {
    local subcommand="${1:-help}"
    shift || true
    
    case "$subcommand" in
        init)
            openems::alerts::init
            ;;
        test)
            openems::alerts::test
            ;;
        configure)
            openems::alerts::configure "$@"
            ;;
        history)
            openems::alerts::history "${1:-10}"
            ;;
        clear)
            openems::alerts::clear_history
            ;;
        enable)
            openems::alerts::configure "${1}" true
            ;;
        disable)
            openems::alerts::configure "${1}" false
            ;;
        *)
            echo "Usage: resource-openems alerts {init|test|configure|history|clear|enable|disable}"
            ;;
    esac
}

# P2 Co-simulation command handler
openems::cosim() {
    local subcommand="${1:-help}"
    shift || true
    
    case "$subcommand" in
        init)
            openems::cosim::init
            ;;
        run)
            openems::cosim::run_scenario "${1:-}"
            ;;
        list)
            openems::cosim::list_scenarios
            ;;
        status)
            openems::cosim::status
            ;;
        test)
            openems::cosim::test
            ;;
        *)
            echo "Usage: resource-openems cosim {init|run|list|status|test}"
            ;;
    esac
}

# Show comprehensive help
openems::show_help() {
    cat << EOF
🔋 OpenEMS - Energy Management System

USAGE:
    resource-openems <command> [options]

COMMANDS:
    help        Show this help message
    info        Display runtime configuration
    manage      Lifecycle management (install/start/stop/restart/uninstall)
    test        Run validation tests (smoke/integration/unit/all)
    content     Manage configurations and DER assets
    status      Show service status
    logs        View service logs
    credentials Display integration credentials
    superset    Apache Superset dashboard integration (P1)
    ditto       Eclipse Ditto digital twin integration (P1)
    forecast    Energy forecast models (P1)
    alerts      Alert automation via Pushover/Twilio (P2)
    cosim       Co-simulation with OTP/GeoNode (P2)
    metrics     Display performance metrics and telemetry stats
    benchmark   Run performance benchmark tests

EXAMPLES:
    # Start OpenEMS
    resource-openems manage start
    
    # Check system health
    resource-openems test smoke
    
    # Configure solar asset
    resource-openems content execute configure-der --type solar
    
    # View energy metrics
    resource-openems status --verbose
    
    
    # Generate forecasts
    resource-openems forecast integrated

For detailed documentation, see: resources/openems/README.md
EOF
    return 0
}

# Show runtime configuration
openems::show_info() {
    echo "📋 OpenEMS Runtime Configuration"
    echo "================================"
    cat "${CLI_SCRIPT_DIR}/config/runtime.json" 2>/dev/null || echo "{}"
    return 0
}

# Lifecycle management
openems::manage() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        install)
            openems::install "$@"
            ;;
        start)
            openems::start "$@"
            ;;
        stop)
            openems::stop "$@"
            ;;
        restart)
            openems::restart "$@"
            ;;
        uninstall)
            openems::uninstall "$@"
            ;;
        *)
            echo "❌ Unknown manage action: $action"
            echo "Available: install, start, stop, restart, uninstall"
            return 1
            ;;
    esac
}

# Test management
openems::test() {
    local phase="${1:-all}"
    shift || true
    
    case "$phase" in
        smoke)
            test::smoke "$@"
            ;;
        integration)
            test::integration "$@"
            ;;
        unit)
            test::unit "$@"
            ;;
        all)
            test::all "$@"
            ;;
        *)
            echo "❌ Unknown test phase: $phase"
            echo "Available: smoke, integration, unit, all"
            return 1
            ;;
    esac
}

# Content management
openems::content() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            content::list "$@"
            ;;
        add)
            content::add "$@"
            ;;
        get)
            content::get "$@"
            ;;
        remove)
            content::remove "$@"
            ;;
        execute)
            content::execute "$@"
            ;;
        *)
            echo "❌ Unknown content action: $action"
            echo "Available: list, add, get, remove, execute"
            return 1
            ;;
    esac
}

# Status display
openems::status() {
    status::show "$@"
}

# Logs display
openems::logs() {
    logs::show "$@"
}

# Show credentials
openems::credentials() {
    echo "🔑 OpenEMS Integration Credentials"
    echo "=================================="
    echo "REST API: http://localhost:8084/rest/"
    echo "JSON-RPC: ws://localhost:8085/jsonrpc"
    echo "Web UI: http://localhost:8084/"
    echo ""
    echo "Default credentials:"
    echo "  Username: admin"
    echo "  Password: admin"
    echo ""
    echo "⚠️  Change defaults in production!"
    return 0
}

# Execute main function
main "$@"
