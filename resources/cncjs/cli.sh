#!/usr/bin/env bash
################################################################################
# CNCjs CLI - v2.0 Universal Contract Implementation
# Web-based CNC controller for Grbl, Marlin, Smoothieware, and TinyG
################################################################################

set -euo pipefail

# Determine paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Source configuration and libraries
[[ -z "${_CNCJS_DEFAULTS_SOURCED:-}" ]] && source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# CLI metadata
readonly CLI_VERSION="2.0.0"
readonly CLI_NAME="resource-cncjs"
readonly CLI_DESCRIPTION="CNCjs CNC Controller Resource"

#######################################
# Show help information
#######################################
show_help() {
    cat << EOF
${CLI_NAME} v${CLI_VERSION} - ${CLI_DESCRIPTION}

USAGE:
    ${CLI_NAME} [COMMAND] [OPTIONS]

COMMANDS:
    help                    Show this help message
    info                    Show resource information and configuration
    manage                  Lifecycle management commands
        install             Install CNCjs and dependencies
        uninstall           Remove CNCjs completely
        start               Start CNCjs service
        stop                Stop CNCjs service
        restart             Restart CNCjs service
    test                    Run validation tests
        smoke               Quick health check (<30s)
        integration         Full functionality test (<120s)
        unit                Library function tests (<60s)
        all                 Run all tests
    content                 Manage G-code files and jobs
        list                List available G-code files
        add <file>          Upload G-code file
        get <name>          Download G-code file
        remove <name>       Remove G-code file
        execute <name>      Queue G-code for execution
    macro                   Manage automation macros
        list                List available macros
        add <name> <code>   Create new macro
        run <name>          Execute macro
        remove <name>       Delete macro
    workflow                Manage CNC job workflows
        list                List available workflows
        create <name>       Create new workflow
        add-step <wf> <file> Add G-code step to workflow
        show <name>         Display workflow details
        execute <name>      Execute workflow sequence
        remove <name>       Delete workflow
        export <name>       Export workflow as archive
        import <archive>    Import workflow from archive
    controller              Manage controller configurations
        list                List supported controllers
        configure <name>    Create controller profile
        show <name>         Display profile details
        apply <name>        Apply controller profile
        remove <name>       Delete controller profile
        test                Test controller connectivity
    visualization           3D G-code path visualization
        preview <file>      Generate 3D preview of G-code file
        analyze <file>      Analyze G-code paths and statistics
        render <file>       Render static visualization image
        export <file>       Export visualization as HTML
        server              Start visualization server
    camera                  Camera monitoring and control
        list                List available camera devices
        enable [device]     Enable camera monitoring
        disable             Disable camera monitoring
        snapshot [file]     Capture snapshot image
        stream start|stop   Start/stop live video stream
        timelapse start|stop Manage timelapse capture
    widget                  Custom widget management
        list                List available widgets
        create <name> [type] Create new widget
        show <name>         Display widget definition
        install <name>      Install widget to CNCjs
        uninstall <name>    Remove widget from CNCjs
        export <name>       Export widget package
        import <file>       Import widget package
    status                  Show detailed service status
    logs                    View CNCjs logs
    credentials             Display connection information

OPTIONS:
    --json                  Output in JSON format
    --verbose               Enable verbose output
    --force                 Skip confirmation prompts

EXAMPLES:
    # Install and start CNCjs
    ${CLI_NAME} manage install
    ${CLI_NAME} manage start

    # Upload and execute G-code
    ${CLI_NAME} content add mypart.gcode
    ${CLI_NAME} content execute mypart.gcode

    # Monitor status
    ${CLI_NAME} status --json
    ${CLI_NAME} logs --follow

CONFIGURATION:
    Port:           ${CNCJS_PORT}
    Controller:     ${CNCJS_CONTROLLER}
    Serial Port:    ${CNCJS_SERIAL_PORT}
    Data Directory: ${CNCJS_DATA_DIR}

For more information, visit: https://cnc.js.org/
EOF
}

#######################################
# Show resource information
#######################################
show_info() {
    local json_output="${1:-false}"
    
    if [[ "$json_output" == "true" ]]; then
        cat "/home/matthalloran8/Vrooli/resources/cncjs/config/runtime.json"
    else
        echo "CNCjs Resource Information"
        echo "=========================="
        local config_file="/home/matthalloran8/Vrooli/resources/cncjs/config/runtime.json"
        if [[ -f "${config_file}" ]]; then
            jq -r '
                "Startup Order: \(.startup_order)",
                "Dependencies: \(.dependencies | join(", "))",
                "Startup Timeout: \(.startup_timeout)s",
                "Startup Estimate: \(.startup_time_estimate)",
                "Recovery Attempts: \(.recovery_attempts)",
                "Priority: \(.priority)",
                "Category: \(.category)",
                "",
                "Capabilities:",
                (.capabilities[] | "  - \(.)")
            ' "${config_file}"
        else
            echo "Runtime configuration not found"
        fi
    fi
}

#######################################
# Main CLI router
#######################################
main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        help)
            show_help
            ;;
        info)
            local json_flag=false
            [[ "${1:-}" == "--json" ]] && json_flag=true
            show_info "$json_flag"
            ;;
        manage)
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                install)
                    cncjs::install "$@"
                    ;;
                uninstall)
                    cncjs::uninstall "$@"
                    ;;
                start)
                    cncjs::start "$@"
                    ;;
                stop)
                    cncjs::stop "$@"
                    ;;
                restart)
                    cncjs::restart "$@"
                    ;;
                *)
                    echo "Unknown manage subcommand: $subcommand" >&2
                    echo "Use '${CLI_NAME} help' for usage information" >&2
                    exit 1
                    ;;
            esac
            ;;
        test)
            local test_type="${1:-all}"
            shift || true
            cncjs::test "$test_type" "$@"
            ;;
        content)
            local subcommand="${1:-list}"
            shift || true
            cncjs::content "$subcommand" "$@"
            ;;
        status)
            cncjs::status "$@"
            ;;
        logs)
            cncjs::logs "$@"
            ;;
        credentials)
            cncjs::credentials "$@"
            ;;
        macro)
            local subcommand="${1:-list}"
            shift || true
            cncjs::macro "$subcommand" "$@"
            ;;
        workflow)
            local subcommand="${1:-list}"
            shift || true
            cncjs::workflow "$subcommand" "$@"
            ;;
        controller)
            local subcommand="${1:-list}"
            shift || true
            cncjs::controller "$subcommand" "$@"
            ;;
        visualization)
            local subcommand="${1:-}"
            shift || true
            cncjs::visualization "$subcommand" "$@"
            ;;
        camera)
            local subcommand="${1:-list}"
            shift || true
            cncjs::camera "$subcommand" "$@"
            ;;
        widget)
            local subcommand="${1:-list}"
            shift || true
            cncjs::widget "$subcommand" "$@"
            ;;
        *)
            echo "Unknown command: $command" >&2
            echo "Use '${CLI_NAME} help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"