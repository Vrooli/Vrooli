#!/usr/bin/env bash
# Vrooli CLI - Scenario Management Commands (Direct Execution)
set -euo pipefail

# Get CLI directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${APP_ROOT}/scripts/lib/scenario/runner.sh"

# Show help for scenario commands
show_scenario_help() {
    cat << EOF
🚀 Vrooli Scenario Commands

USAGE:
    vrooli scenario <subcommand> [options]

SUBCOMMANDS:
    run <name>              Run a scenario directly
    test <name>             Test a scenario
    list                    List available scenarios
    logs <name>             View logs for a scenario

EXAMPLES:
    vrooli scenario run make-it-vegan
    vrooli scenario test swarm-manager
    vrooli scenario list
    vrooli scenario logs system-monitor
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
        run)
            local scenario_name="${1:-}"
            [[ -z "$scenario_name" ]] && { 
                log::error "Scenario name required"
                log::info "Usage: vrooli scenario run <name>"
                return 1
            }
            shift
            scenario::run "$scenario_name" develop "$@"
            ;;
        test)
            local scenario_name="${1:-}"
            [[ -z "$scenario_name" ]] && { 
                log::error "Scenario name required"
                log::info "Usage: vrooli scenario test <name>"
                return 1
            }
            shift
            scenario::run "$scenario_name" test "$@"
            ;;
        list)
            scenario::list
            ;;
        logs)
            local scenario_name="${1:-}"
            [[ -z "$scenario_name" ]] && { 
                log::error "Scenario name required"
                log::info "Usage: vrooli scenario logs <name>"
                log::info "Available scenarios with logs:"
                ls -1 "${HOME}/.vrooli/logs/scenarios/" 2>/dev/null || echo "  (none found)"
                return 1
            }
            
            local logs_dir="${HOME}/.vrooli/logs/scenarios/${scenario_name}"
            if [[ ! -d "$logs_dir" ]]; then
                log::warn "No logs found for scenario: $scenario_name"
                log::info "Available scenarios with logs:"
                ls -1 "${HOME}/.vrooli/logs/scenarios/" 2>/dev/null || echo "  (none found)"
                return 1
            fi
            
            # Check for log files
            local log_files=("$logs_dir"/*.log)
            if [[ ! -e "${log_files[0]}" ]]; then
                log::warn "No log files found in $logs_dir"
                return 1
            fi
            
            # Display logs with tail -f for live viewing
            log::info "Showing logs for scenario: $scenario_name"
            log::info "Press Ctrl+C to stop viewing"
            echo ""
            
            # Use tail with multiple files if available
            tail -f "$logs_dir"/*.log
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