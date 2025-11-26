#!/usr/bin/env bash
# Vrooli CLI - Scenario Management Commands (Modular Architecture)
set -euo pipefail

# Get CLI directory - use unique variable name to avoid conflicts
SCENARIO_CMD_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_CMD_DIR}/../../.." && builtin pwd)}"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${APP_ROOT}/scripts/lib/scenario/runner.sh"

# Source health validator
HEALTH_VALIDATOR="${SCENARIO_CMD_DIR}/validators/health-validator.sh"
if [[ -f "$HEALTH_VALIDATOR" ]]; then
    source "$HEALTH_VALIDATOR"
fi

# Source all modules
for module in "${SCENARIO_CMD_DIR}/modules"/*.sh; do
    if [[ -f "$module" ]]; then
        source "$module"
    fi
done

# Show help for scenario commands
show_scenario_help() {
    cat << EOF
🚀 Vrooli Scenario Commands

USAGE:
    vrooli scenario <subcommand> [options]

SUBCOMMANDS:
    start <name> [options]  Start a scenario
    start <name1> <name2>...Start multiple scenarios
    start-all               Start all available scenarios
    setup <name>            Run the setup lifecycle for a scenario
    restart <name> [options] Restart a scenario (stop then start)
    stop <name>             Stop a running scenario
    stop-all                Stop all running scenarios
    test <name> [phase|all|e2e] Run scenario's test lifecycle event
    list [--json]           List available scenarios (use --include-ports for live port data)
    logs <name> [options]   View logs for a scenario
    status [name] [--json]  Show scenario status (includes test infrastructure validation)
    open <name> [options]   Open scenario in browser
    port <name> [port]      Get port number(s) for scenario (use --json for JSON output)
    ui-smoke <name> [--json] Run Browserless UI smoke harness for a scenario
    template [cmd]          Manage scenario templates (list/show)
    generate <template>     Scaffold a scenario from a template
    requirements <subcommand> Manage scenario requirements (run `vrooli scenario requirements help`)
    completeness <name> [--format json|human] Calculate objective completeness score

OPTIONS FOR START:
    --clean-stale           Clean stale port locks before starting
    --open                  Open scenario in browser after successful start

OPTIONS FOR LOGS:
    --follow, -f            Follow log output (live view)
    --step <name>           View specific background step log
    --runtime               View all background process logs
    --lifecycle             View lifecycle log (default behavior)
    --force-follow          Stream even when non-interactive (may hang automation)

EXAMPLES:
    vrooli scenario start make-it-vegan         # Start a specific scenario
    vrooli scenario start make-it-vegan --clean-stale # Start with stale lock cleanup
    vrooli scenario start app-monitor --open          # Start then open in browser
    vrooli scenario start picker-wheel invoice-generator # Start multiple scenarios
    vrooli scenario start-all                   # Start all scenarios
    vrooli scenario restart ecosystem-manager    # Restart a scenario
    vrooli scenario stop swarm-manager           # Stop a specific scenario
    vrooli scenario stop-all                     # Stop all scenarios
    vrooli scenario test system-monitor          # Run scenario's test lifecycle event
    vrooli scenario list                         # List available scenarios
    vrooli scenario list --include-ports         # Include port info for running scenarios
    vrooli scenario list --json                  # List scenarios in JSON format
    vrooli scenario list --json --include-ports  # JSON with live port metadata
    vrooli scenario logs system-monitor          # Shows lifecycle execution
    vrooli scenario logs system-monitor --follow # Follow lifecycle log
    vrooli scenario status                       # Show all scenarios
    vrooli scenario status swarm-manager --json  # Show specific scenario with test validation
    vrooli scenario open app-monitor             # Open scenario in browser
    vrooli scenario open app-monitor --print-url # Print URL instead of opening
    vrooli scenario port app-monitor             # List all running ports
    vrooli scenario port app-monitor UI_PORT     # Get UI port number
EOF
}

# Main handler
main() {
    if [[ $# -eq 0 ]] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        show_scenario_help
        return 0
    fi
    
    local subcommand="$1"; shift
    case "$subcommand" in
        start|run)  # Support both 'start' and 'run' (run is silent alias)
            scenario::lifecycle::start "$@"
            ;;
        start-all)
            scenario::lifecycle::start_all "$@"
            ;;
        setup)
            scenario::lifecycle::setup "$@"
            ;;
        restart)
            scenario::lifecycle::restart "$@"
            ;;
        stop)
            scenario::lifecycle::stop "$@"
            ;;
        stop-all)
            scenario::lifecycle::stop_all "$@"
            ;;
        test)
            scenario::lifecycle::test "$@"
            ;;
        list)
            scenario::list "$@"
            ;;
        logs)
            scenario::logs::view "$@"
            ;;
        status)
            scenario::status::show "$@"
            ;;
        port)
            scenario::ports::get "$@"
            ;;
        open)
            scenario::browser::open "$@"
            ;;
        requirements)
            scenario::requirements::dispatch "$@"
            ;;
        ui-smoke)
            scenario::smoke::run "$@"
            ;;
        template)
            scenario::template::dispatch "$@"
            ;;
        generate)
            scenario::template::generate "$@"
            ;;
        completeness)
            scenario::completeness::score "$@"
            ;;
        # Removed: convert, convert-all, validate, enable, disable
        *)
            log::error "Unknown scenario command: $subcommand"
            echo ""
            show_scenario_help
            return 1
            ;;
    esac
}

main "$@"
