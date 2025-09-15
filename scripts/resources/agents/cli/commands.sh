#!/usr/bin/env bash
################################################################################
# Agent CLI Commands Framework
# 
# Generic CLI command handling for all agent-capable resources
# Provides standardized argument parsing and command routing
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Main CLI command handler for agent management
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $@ - Command arguments
# Returns:
#   Exit code from subcommand
#######################################
agents::cli::command() {
    local config_array_name="$1"
    shift
    local subcommand="${1:-list}"
    shift || true
    
    # Source required modules
    source "${APP_ROOT}/scripts/resources/agents/core/registry.sh"
    source "${APP_ROOT}/scripts/resources/agents/core/lifecycle.sh"
    source "${APP_ROOT}/scripts/resources/agents/monitoring/logs.sh"
    
    case "$subcommand" in
        "list"|"ls")
            agents::cli::handle_list "$config_array_name" "$@"
            ;;
            
        "stop")
            agents::cli::handle_stop "$config_array_name" "$@"
            ;;
            
        "cleanup")
            agents::cli::handle_cleanup "$config_array_name" "$@"
            ;;
            
        "info")
            agents::cli::handle_info "$config_array_name" "$@"
            ;;
            
        "logs")
            agents::cli::handle_logs "$config_array_name" "$@"
            ;;
            
        "help"|"-h"|"--help")
            agents::cli::show_help "$config_array_name"
            ;;
            
        *)
            # Create nameref for config access
            local -n config_ref="$config_array_name"
            local resource_name="${config_ref[resource_name]}"
            local cli_prefix="${config_ref[cli_prefix]}"
            log::error "Unknown subcommand: $subcommand"
            log::error "Use '$cli_prefix agents help' for usage information"
            return 1
            ;;
    esac
}

#######################################
# Handle 'list' subcommand
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $@ - Command arguments
#######################################
agents::cli::handle_list() {
    local config_array_name="$1"
    local -n config_ref="$config_array_name"
    shift
    local status_filter=""
    local json_output="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output="true"
                shift
                ;;
            --status)
                status_filter="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ "$json_output" == "true" ]]; then
        agents::registry::list "$config_array_name" "$status_filter" "json"
    else
        agents::registry::list "$config_array_name" "$status_filter" "table"
    fi
}

#######################################
# Handle 'stop' subcommand
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $@ - Command arguments
#######################################
agents::cli::handle_stop() {
    local -n config_ref=$1
    shift
    local target="$1"
    local force="false"
    shift || true
    
    if [[ -z "$target" ]]; then
        local cli_prefix="${config_ref[cli_prefix]}"
        log::error "Usage: $cli_prefix agents stop <agent-id|all> [--force]"
        return 1
    fi
    
    # Parse remaining arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force="true"
                shift
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    agents::lifecycle::stop config_ref "$target" "$force"
}

#######################################
# Handle 'cleanup' subcommand
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $@ - Command arguments
#######################################
agents::cli::handle_cleanup() {
    local -n config_ref=$1
    shift
    local resource_name="${config_ref[resource_name]}"
    
    local cleaned
    cleaned=$(agents::lifecycle::cleanup config_ref)
    if [[ $cleaned -gt 0 ]]; then
        log::info "Cleaned up $cleaned dead $resource_name agents"
    else
        log::info "No dead $resource_name agents to clean up"
    fi
}

#######################################
# Handle 'info' subcommand
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $@ - Command arguments
#######################################
agents::cli::handle_info() {
    local -n config_ref=$1
    shift
    local agent_id="$1"
    local json_output="false"
    shift || true
    
    if [[ -z "$agent_id" ]]; then
        local cli_prefix="${config_ref[cli_prefix]}"
        log::error "Usage: $cli_prefix agents info <agent-id> [--json]"
        return 1
    fi
    
    # Parse remaining arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output="true"
                shift
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ "$json_output" == "true" ]]; then
        agents::registry::info config_ref "$agent_id" "json"
    else
        agents::registry::info config_ref "$agent_id" "table"
    fi
}

#######################################
# Handle 'logs' subcommand
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $@ - Command arguments
#######################################
agents::cli::handle_logs() {
    local -n config_ref=$1
    shift
    local agent_id="$1"
    local follow="false"
    local lines="100"
    local json_output="false"
    shift || true
    
    if [[ -z "$agent_id" ]]; then
        local cli_prefix="${config_ref[cli_prefix]}"
        log::error "Usage: $cli_prefix agents logs <agent-id> [--follow|-f] [--lines|-n N] [--json]"
        return 1
    fi
    
    # Parse remaining arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -f|--follow)
                follow="true"
                shift
                ;;
            -n|--lines)
                if [[ -z "$2" || ! "$2" =~ ^[0-9]+$ ]]; then
                    log::error "Invalid lines value: $2"
                    return 1
                fi
                lines="$2"
                shift 2
                ;;
            --json)
                json_output="true"
                shift
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    agents::logs::get config_ref "$agent_id" "$follow" "$lines" "$json_output"
}

#######################################
# Show help text
# Arguments:
#   $1 - Configuration array name (passed by reference)
#######################################
agents::cli::show_help() {
    local -n config_ref=$1
    local cli_prefix="${config_ref[cli_prefix]}"
    local resource_name="${config_ref[resource_name]}"
    
    echo "Usage: $cli_prefix agents <subcommand> [options]"
    echo ""
    echo "Subcommands:"
    echo "  list [--json] [--status STATUS]           List all $resource_name agents (optionally filter by status)"
    echo "  stop <agent-id|all> [--force]             Stop specific agent or all agents"
    echo "  cleanup                                    Remove dead agents from registry"
    echo "  info <agent-id> [--json]                  Show detailed information about an agent"
    echo "  logs <agent-id> [-f] [-n N] [--json]      Show agent logs"
    echo "  help                                       Show this help message"
    echo ""
    echo "Logs options:"
    echo "  -f, --follow     Follow log output (like tail -f)"
    echo "  -n, --lines N    Show last N lines (default: 100)"
    echo "  --json           Output in JSON format"
    echo ""
    echo "Status filters: running, stopped, crashed"
}

# Export functions for use by resource-specific implementations
export -f agents::cli::command
export -f agents::cli::handle_list
export -f agents::cli::handle_stop
export -f agents::cli::handle_cleanup
export -f agents::cli::handle_info
export -f agents::cli::handle_logs
export -f agents::cli::show_help