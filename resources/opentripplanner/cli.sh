#!/usr/bin/env bash
################################################################################
# OpenTripPlanner Resource CLI - v2.0 Universal Contract Compliant
# 
# Multimodal trip planning engine for transit, biking, walking
#
# Usage:
#   resource-opentripplanner <command> [options]
#   resource-opentripplanner <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    OTP_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${OTP_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi

RESOURCE_DIR="${APP_ROOT}/resources/opentripplanner"
RESOURCE_NAME="opentripplanner"

# Source common libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/lib/utils/exit_codes.sh"
source "${APP_ROOT}/scripts/resources/port_registry.sh"

# Source local libraries
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Show help message
show_help() {
    cat <<EOF
OpenTripPlanner Resource CLI

Usage:
  resource-opentripplanner <command> [options]

Commands:
  help                Show this help message
  info                Show resource runtime information
  
Management Commands:
  manage install      Install OpenTripPlanner and dependencies
  manage uninstall    Remove OpenTripPlanner completely  
  manage start        Start OpenTripPlanner service
  manage stop         Stop OpenTripPlanner service
  manage restart      Restart OpenTripPlanner service
  
Testing Commands:
  test all           Run all test suites
  test smoke         Quick health validation
  test integration   Full functionality tests
  test unit          Library function tests
  
Content Commands:
  content add        Add GTFS/OSM data
  content list       List available data
  content get        Retrieve specific data
  content remove     Remove data
  content execute    Build graph or run analysis
  
Monitoring Commands:
  status             Show service status
  logs               View service logs

Examples:
  resource-opentripplanner manage start --wait
  resource-opentripplanner content add --file transit.gtfs.zip --type gtfs --name portland
  resource-opentripplanner test smoke
  resource-opentripplanner status --json

EOF
}

# Show runtime info
show_info() {
    if [[ -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
        cat "${RESOURCE_DIR}/config/runtime.json"
    else
        echo "Runtime configuration not found"
        return 1
    fi
}

# Main command dispatcher
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
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                install) opentripplanner::install "$@" ;;
                uninstall) opentripplanner::uninstall "$@" ;;
                start) opentripplanner::start "$@" ;;
                stop) opentripplanner::stop "$@" ;;
                restart) opentripplanner::restart "$@" ;;
                *) echo "Unknown manage subcommand: $subcommand"; show_help; exit 1 ;;
            esac
            ;;
        test)
            local subcommand="${1:-all}"
            shift || true
            case "$subcommand" in
                all) opentripplanner::test::all "$@" ;;
                smoke) opentripplanner::test::smoke "$@" ;;
                integration) opentripplanner::test::integration "$@" ;;
                unit) opentripplanner::test::unit "$@" ;;
                *) echo "Unknown test subcommand: $subcommand"; show_help; exit 1 ;;
            esac
            ;;
        content)
            local subcommand="${1:-}"
            shift || true
            case "$subcommand" in
                add) opentripplanner::content::add "$@" ;;
                list) opentripplanner::content::list "$@" ;;
                get) opentripplanner::content::get "$@" ;;
                remove) opentripplanner::content::remove "$@" ;;
                execute) opentripplanner::content::execute "$@" ;;
                *) echo "Unknown content subcommand: $subcommand"; show_help; exit 1 ;;
            esac
            ;;
        status)
            opentripplanner::status "$@"
            ;;
        logs)
            opentripplanner::logs "$@"
            ;;
        *)
            echo "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main
main "$@"