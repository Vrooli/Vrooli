#!/usr/bin/env bash
# CLI Command Framework v2.0 for Vrooli Resources
# Supports both flat commands (v1.0) and hierarchical commands (v2.0)
# Maintains backward compatibility while enabling v2.0 Universal Contract compliance

# Source guard to prevent multiple sourcing
if ! declare -F cli::init >/dev/null 2>&1; then
    unset _CLI_COMMAND_FRAMEWORK_V2_SOURCED 2>/dev/null || true
fi
[[ -n "${_CLI_COMMAND_FRAMEWORK_V2_SOURCED:-}" ]] && return 0
_CLI_COMMAND_FRAMEWORK_V2_SOURCED=1

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPTS_RESOURCES_LIB_DIR="${APP_ROOT}/scripts/resources/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true

# Framework state
declare -gA CLI_COMMANDS=()                    # Flat commands for backward compatibility
declare -gA CLI_COMMAND_DESCRIPTIONS=()        # Descriptions for all commands
declare -gA CLI_COMMAND_HANDLERS=()           # Handler functions
declare -gA CLI_COMMAND_FLAGS=()              # Command flags (modifies-system, etc.)
declare -gA CLI_COMMAND_GROUPS=()             # Command groups (manage, test, content)
declare -gA CLI_GROUP_DESCRIPTIONS=()         # Group descriptions
declare -gA CLI_GROUP_SUBCOMMANDS=()          # Subcommands for each group
declare -g CLI_RESOURCE_NAME=""
declare -g CLI_RESOURCE_DESCRIPTION=""
declare -g CLI_DRY_RUN="${DRY_RUN:-false}"
declare -g CLI_V2_MODE="${CLI_V2_MODE:-auto}"  # auto, v1, v2

#######################################
# Initialize CLI framework for a resource
# Args: $1 - resource_name, $2 - description (optional), $3 - mode (optional)
#######################################
cli::init() {
    local resource_name="$1"
    local description="${2:-$resource_name resource management}"
    local mode="${3:-auto}"
    
    CLI_RESOURCE_NAME="$resource_name"
    CLI_RESOURCE_DESCRIPTION="$description"
    CLI_V2_MODE="$mode"
    
    # Always register help command
    cli::register_command "help" "Show this help message" "cli::_handle_help"
    
    # Always register info command for structured resource information
    cli::register_command "info" "Show structured resource information" "cli::_handle_info"
    
    # For v2 mode, register command groups
    if [[ "$CLI_V2_MODE" == "v2" ]] || [[ "$CLI_V2_MODE" == "auto" ]]; then
        # Register v2.0 command groups
        cli::register_command_group "manage" "Resource lifecycle management"
        cli::register_command_group "test" "Testing and validation"
        cli::register_command_group "content" "Content management"
        
        # Register core v2.0 subcommands
        cli::register_subcommand "manage" "install" "Install the resource" "cli::_handle_install" "modifies-system"
        cli::register_subcommand "manage" "uninstall" "Uninstall the resource" "cli::_handle_uninstall" "modifies-system"
        cli::register_subcommand "manage" "start" "Start the resource" "cli::_handle_start" "modifies-system"
        cli::register_subcommand "manage" "stop" "Stop the resource" "cli::_handle_stop" "modifies-system"
        cli::register_subcommand "manage" "restart" "Restart the resource" "cli::_handle_restart" "modifies-system"
        
        cli::register_subcommand "test" "all" "Run all tests" "cli::_handle_test_all"
        cli::register_subcommand "test" "integration" "Run integration tests" "cli::_handle_test_integration"
        cli::register_subcommand "test" "unit" "Run unit tests" "cli::_handle_test_unit"
        cli::register_subcommand "test" "smoke" "Quick health check" "cli::_handle_test_smoke"
        
        cli::register_subcommand "content" "add" "Add content" "cli::_handle_content_add" "modifies-system"
        cli::register_subcommand "content" "list" "List content" "cli::_handle_content_list"
        cli::register_subcommand "content" "get" "Get content" "cli::_handle_content_get"
        cli::register_subcommand "content" "remove" "Remove content" "cli::_handle_content_remove" "modifies-system"
        cli::register_subcommand "content" "execute" "Execute content" "cli::_handle_content_execute" "modifies-system"
    fi
    
    # For backward compatibility, also register flat commands in auto mode
    if [[ "$CLI_V2_MODE" == "v1" ]] || [[ "$CLI_V2_MODE" == "auto" ]]; then
        cli::register_command "status" "Show resource status" "cli::_handle_status"
        cli::register_command "start" "Start the resource" "cli::_handle_start" "modifies-system"
        cli::register_command "stop" "Stop the resource" "cli::_handle_stop" "modifies-system"
        cli::register_command "restart" "Restart the resource" "cli::_handle_restart" "modifies-system"
        cli::register_command "install" "Install the resource" "cli::_handle_install" "modifies-system"
        cli::register_command "validate" "Validate resource configuration" "cli::_handle_validate"
        cli::register_command "logs" "Show resource logs" "cli::_handle_logs"
        cli::register_command "credentials" "Show integration credentials" "cli::_handle_credentials"
    fi
    
    log::debug "CLI framework v2 initialized for: $resource_name (mode: $CLI_V2_MODE)"
}

#######################################
# Register a command group (v2.0)
# Args: $1 - group_name, $2 - description
#######################################
cli::register_command_group() {
    local group="$1"
    local desc="$2"
    
    CLI_COMMAND_GROUPS[$group]=1
    CLI_GROUP_DESCRIPTIONS[$group]="$desc"
    CLI_GROUP_SUBCOMMANDS[$group]=""
    
    # Also register the group as a top-level command for dispatch
    cli::register_command "$group" "$desc" "cli::_handle_group_$group"
    
    log::debug "Registered command group: $group"
}

#######################################
# Register a subcommand within a group (v2.0)
# Args: $1 - group, $2 - subcommand, $3 - description, $4 - handler, $5 - flags
#######################################
cli::register_subcommand() {
    local group="$1"
    local subcmd="$2"
    local desc="$3"
    local handler="$4"
    local flags="${5:-}"
    
    # Store subcommand in group's subcommand list
    local current="${CLI_GROUP_SUBCOMMANDS[$group]:-}"
    if [[ -z "$current" ]]; then
        CLI_GROUP_SUBCOMMANDS[$group]="$subcmd"
    else
        CLI_GROUP_SUBCOMMANDS[$group]="$current,$subcmd"
    fi
    
    # Store subcommand details with group prefix
    local full_cmd="${group}::${subcmd}"
    CLI_COMMAND_DESCRIPTIONS[$full_cmd]="$desc"
    CLI_COMMAND_HANDLERS[$full_cmd]="$handler"
    CLI_COMMAND_FLAGS[$full_cmd]="$flags"
    
    log::debug "Registered subcommand: $group $subcmd -> $handler"
}

#######################################
# Register a flat command (v1.0 backward compatibility)
# Args: $1 - command_name, $2 - description, $3 - handler_function, $4 - flags
#######################################
cli::register_command() {
    local cmd="$1"
    local desc="$2"
    local handler="$3"
    local flags="${4:-}"
    
    CLI_COMMANDS[$cmd]=1
    CLI_COMMAND_DESCRIPTIONS[$cmd]="$desc"
    CLI_COMMAND_HANDLERS[$cmd]="$handler"
    CLI_COMMAND_FLAGS[$cmd]="$flags"
    
    log::debug "Registered command: $cmd -> $handler"
}

#######################################
# Main command dispatcher
# Args: $@ - command line arguments
#######################################
cli::dispatch() {
    local cmd=""
    local subcmd=""
    
    # Parse command and subcommand
    if [[ $# -gt 0 ]]; then
        cmd="$1"
        shift
        
        # Check if this is a command group with a subcommand
        if [[ -n "${CLI_COMMAND_GROUPS[$cmd]:-}" ]] && [[ $# -gt 0 ]] && [[ "$1" != -* ]]; then
            subcmd="$1"
            shift
        fi
    else
        cmd="help"
    fi
    
    # Handle global flags
    while [[ $# -gt 0 ]]; do
        case "${1:-}" in
            --dry-run)
                export CLI_DRY_RUN="true"
                export DRY_RUN="true"
                shift
                ;;
            --help|-h)
                if [[ -n "$subcmd" ]]; then
                    cli::_show_subcommand_help "$cmd" "$subcmd"
                elif [[ -n "${CLI_COMMAND_GROUPS[$cmd]:-}" ]]; then
                    cli::_show_group_help "$cmd"
                else
                    cmd="help"
                fi
                return 0
                ;;
            *)
                break
                ;;
        esac
    done
    
    # Dispatch to appropriate handler
    if [[ -n "$subcmd" ]]; then
        # Handle group subcommand
        cli::_dispatch_subcommand "$cmd" "$subcmd" "$@"
    else
        # Handle flat command or group
        cli::_dispatch_command "$cmd" "$@"
    fi
}

#######################################
# Dispatch a subcommand within a group
#######################################
cli::_dispatch_subcommand() {
    local group="$1"
    local subcmd="$2"
    shift 2
    
    local full_cmd="${group}::${subcmd}"
    
    # Check if subcommand exists
    if [[ -z "${CLI_COMMAND_HANDLERS[$full_cmd]:-}" ]]; then
        log::error "Unknown subcommand: $group $subcmd"
        cli::_show_group_help "$group"
        return 1
    fi
    
    local handler="${CLI_COMMAND_HANDLERS[$full_cmd]}"
    local flags="${CLI_COMMAND_FLAGS[$full_cmd]:-}"
    
    # Check dry-run for system-modifying commands
    if [[ "$flags" == *"modifies-system"* ]] && [[ "$CLI_DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would execute: $group $subcmd"
        return 0
    fi
    
    # Execute the handler
    if command -v "$handler" &>/dev/null; then
        log::debug "Executing subcommand handler: $handler"
        "$handler" "$@"
    else
        log::error "Handler function not found: $handler"
        return 1
    fi
}

#######################################
# Dispatch a flat command
#######################################
cli::_dispatch_command() {
    local cmd="$1"
    shift
    
    # Check if command exists
    if [[ -z "${CLI_COMMANDS[$cmd]:-}" ]]; then
        log::error "Unknown command: $cmd"
        cli::_handle_help
        return 1
    fi
    
    local handler="${CLI_COMMAND_HANDLERS[$cmd]}"
    local flags="${CLI_COMMAND_FLAGS[$cmd]:-}"
    
    # Check dry-run for system-modifying commands
    if [[ "$flags" == *"modifies-system"* ]] && [[ "$CLI_DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would execute: $cmd"
        return 0
    fi
    
    # Execute the handler
    if command -v "$handler" &>/dev/null; then
        log::debug "Executing handler: $handler"
        "$handler" "$@"
    else
        log::error "Handler function not found: $handler"
        return 1
    fi
}

#######################################
# Group handler functions
#######################################
cli::_handle_group_manage() {
    local subcmd="${1:-}"
    if [[ -z "$subcmd" ]] || [[ "$subcmd" == "--help" ]] || [[ "$subcmd" == "-h" ]]; then
        cli::_show_group_help "manage"
        return 0
    fi
    shift
    cli::_dispatch_subcommand "manage" "$subcmd" "$@"
}

cli::_handle_group_test() {
    local subcmd="${1:-}"
    if [[ -z "$subcmd" ]] || [[ "$subcmd" == "--help" ]] || [[ "$subcmd" == "-h" ]]; then
        cli::_show_group_help "test"
        return 0
    fi
    shift
    cli::_dispatch_subcommand "test" "$subcmd" "$@"
}

cli::_handle_group_content() {
    local subcmd="${1:-}"
    if [[ -z "$subcmd" ]] || [[ "$subcmd" == "--help" ]] || [[ "$subcmd" == "-h" ]]; then
        cli::_show_group_help "content"
        return 0
    fi
    shift
    cli::_dispatch_subcommand "content" "$subcmd" "$@"
}

#######################################
# Help display functions
#######################################
cli::_handle_help() {
    # Resource header
    local resource_upper="${CLI_RESOURCE_NAME^^}"
    echo "🔧 ${resource_upper} Resource Management"
    echo
    echo "📋 USAGE:"
    echo "    resource-${CLI_RESOURCE_NAME} <command> [subcommand] [options]"
    echo
    echo "📖 DESCRIPTION:"
    echo "    $CLI_RESOURCE_DESCRIPTION"
    echo
    
    # Show command groups if in v2 mode
    if [[ "$CLI_V2_MODE" == "v2" ]] || ([[ "$CLI_V2_MODE" == "auto" ]] && [[ ${#CLI_COMMAND_GROUPS[@]} -gt 0 ]]); then
        echo "🎯 COMMAND GROUPS:"
        for group in $(printf '%s\n' "${!CLI_COMMAND_GROUPS[@]}" | sort); do
            local desc="${CLI_GROUP_DESCRIPTIONS[$group]}"
            local icon="📦"
            case "$group" in
                manage) icon="⚙️ " ;;
                test) icon="🧪" ;;
                content) icon="📄" ;;
            esac
            printf "    %-20s %s %s\n" "$group" "$icon" "$desc"
        done
        echo
        echo "    💡 Use 'resource-${CLI_RESOURCE_NAME} <group> --help' for subcommands"
        echo
    fi
    
    # Show flat commands
    local flat_commands=()
    for cmd in "${!CLI_COMMANDS[@]}"; do
        # Skip group commands in the flat list
        if [[ -z "${CLI_COMMAND_GROUPS[$cmd]:-}" ]]; then
            flat_commands+=("$cmd")
        fi
    done
    
    if [[ ${#flat_commands[@]} -gt 0 ]]; then
        # Categorize commands
        local info_commands=()
        local deprecated_commands=()
        local other_commands=()
        
        for cmd in $(printf '%s\n' "${flat_commands[@]}" | sort); do
            local desc="${CLI_COMMAND_DESCRIPTIONS[$cmd]}"
            if [[ "$cmd" == "help" || "$cmd" == "status" || "$cmd" == "logs" || "$cmd" == "credentials" ]]; then
                info_commands+=("$cmd")
            elif [[ "$desc" == *"deprecated"* ]]; then
                deprecated_commands+=("$cmd")
            else
                other_commands+=("$cmd")
            fi
        done
        
        # Show information commands
        if [[ ${#info_commands[@]} -gt 0 ]]; then
            echo "ℹ️  INFORMATION COMMANDS:"
            for cmd in "${info_commands[@]}"; do
                local desc="${CLI_COMMAND_DESCRIPTIONS[$cmd]}"
                printf "    %-20s %s\n" "$cmd" "$desc"
            done
            echo
        fi
        
        # Show other commands
        if [[ ${#other_commands[@]} -gt 0 ]]; then
            echo "🔧 OTHER COMMANDS:"
            for cmd in "${other_commands[@]}"; do
                local desc="${CLI_COMMAND_DESCRIPTIONS[$cmd]}"
                local flags="${CLI_COMMAND_FLAGS[$cmd]:-}"
                printf "    %-20s %s" "$cmd" "$desc"
                if [[ "$flags" == *"modifies-system"* ]]; then
                    printf " ⚠️"
                fi
                echo
            done
            echo
        fi
        
        # Show deprecated commands
        if [[ ${#deprecated_commands[@]} -gt 0 ]]; then
            echo "⚠️  LEGACY COMMANDS (deprecated):"
            for cmd in "${deprecated_commands[@]}"; do
                local desc="${CLI_COMMAND_DESCRIPTIONS[$cmd]}"
                printf "    %-20s %s\n" "$cmd" "$desc"
            done
            echo
        fi
    fi
    
    echo "⚙️  OPTIONS:"
    echo "    --dry-run            Show what would be done without making changes"
    echo "    --help, -h           Show help message"
    echo
    
    # Show examples based on mode
    echo "💡 EXAMPLES:"
    if [[ "$CLI_V2_MODE" == "v2" ]] || [[ "$CLI_V2_MODE" == "auto" ]]; then
        echo "    # Resource lifecycle (v2.0 style)"
        echo "    resource-${CLI_RESOURCE_NAME} manage install"
        echo "    resource-${CLI_RESOURCE_NAME} manage start"
        echo "    resource-${CLI_RESOURCE_NAME} manage stop"
        echo
        echo "    # Testing and validation"
        echo "    resource-${CLI_RESOURCE_NAME} test smoke"
        echo "    resource-${CLI_RESOURCE_NAME} test all"
        echo
        echo "    # Content management"
        echo "    resource-${CLI_RESOURCE_NAME} content list"
        echo "    resource-${CLI_RESOURCE_NAME} content add --file data.json"
        echo
    fi
    
    if [[ "$CLI_V2_MODE" == "v1" ]] || [[ "$CLI_V2_MODE" == "auto" ]]; then
        if [[ ${#flat_commands[@]} -gt 0 ]]; then
            echo "    # Information and status"
            echo "    resource-${CLI_RESOURCE_NAME} status"
            echo "    resource-${CLI_RESOURCE_NAME} logs"
            echo "    resource-${CLI_RESOURCE_NAME} --dry-run"
        fi
    fi
    
    echo
    echo "📚 For more help on a specific command:"
    echo "    resource-${CLI_RESOURCE_NAME} <command> --help"
}

cli::_show_group_help() {
    local group="$1"
    local group_upper="${group^^}"
    local icon="📦"
    
    case "$group" in
        manage) icon="⚙️ " ;;
        test) icon="🧪" ;;
        content) icon="📄" ;;
    esac
    
    echo "${icon} ${group_upper} Commands"
    echo
    echo "📋 USAGE:"
    echo "    resource-${CLI_RESOURCE_NAME} $group <subcommand> [options]"
    echo
    echo "📖 DESCRIPTION:"
    echo "    ${CLI_GROUP_DESCRIPTIONS[$group]}"
    echo
    echo "🔧 SUBCOMMANDS:"
    
    # Parse and show subcommands
    IFS=',' read -ra subcmds <<< "${CLI_GROUP_SUBCOMMANDS[$group]}"
    for subcmd in "${subcmds[@]}"; do
        local full_cmd="${group}::${subcmd}"
        local desc="${CLI_COMMAND_DESCRIPTIONS[$full_cmd]}"
        local flags="${CLI_COMMAND_FLAGS[$full_cmd]:-}"
        printf "    %-16s %s" "$subcmd" "$desc"
        if [[ "$flags" == *"modifies-system"* ]]; then
            printf " ⚠️"
        fi
        echo
    done
    echo
    echo "💡 EXAMPLES:"
    case "$group" in
        manage)
            echo "    resource-${CLI_RESOURCE_NAME} manage install"
            echo "    resource-${CLI_RESOURCE_NAME} manage start"
            echo "    resource-${CLI_RESOURCE_NAME} manage stop --dry-run"
            ;;
        test)
            echo "    resource-${CLI_RESOURCE_NAME} test smoke"
            echo "    resource-${CLI_RESOURCE_NAME} test all"
            echo "    resource-${CLI_RESOURCE_NAME} test integration"
            ;;
        content)
            echo "    resource-${CLI_RESOURCE_NAME} content list"
            echo "    resource-${CLI_RESOURCE_NAME} content add --file data.json"
            echo "    resource-${CLI_RESOURCE_NAME} content get --name my-content"
            ;;
    esac
    echo
    echo "📚 For more help:"
    echo "    resource-${CLI_RESOURCE_NAME} help"
}

cli::_show_subcommand_help() {
    local group="$1"
    local subcmd="$2"
    local full_cmd="${group}::${subcmd}"
    
    echo "Usage: resource-${CLI_RESOURCE_NAME} $group $subcmd [options]"
    echo
    echo "${CLI_COMMAND_DESCRIPTIONS[$full_cmd]}"
    echo
    
    # Add subcommand-specific help if available
    local help_handler="${CLI_COMMAND_HANDLERS[$full_cmd]}_help"
    if command -v "$help_handler" &>/dev/null; then
        "$help_handler"
    fi
}

#######################################
# Core command handlers
#######################################
cli::_handle_status() {
    if command -v "${CLI_RESOURCE_NAME}::status" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::status" "$@"
    else
        log::info "Status: Not implemented"
    fi
}

cli::_handle_start() {
    if command -v "${CLI_RESOURCE_NAME}::start" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::start" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::docker::start" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::docker::start" "$@"
    else
        log::error "Start not implemented for ${CLI_RESOURCE_NAME}"
        return 1
    fi
}

cli::_handle_stop() {
    if command -v "${CLI_RESOURCE_NAME}::stop" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::stop" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::docker::stop" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::docker::stop" "$@"
    else
        log::error "Stop not implemented for ${CLI_RESOURCE_NAME}"
        return 1
    fi
}

cli::_handle_restart() {
    log::info "Restarting ${CLI_RESOURCE_NAME}..."
    cli::_handle_stop "$@" && sleep 2 && cli::_handle_start "$@"
}

cli::_handle_install() {
    if command -v "${CLI_RESOURCE_NAME}::install" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::install" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::install::main" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::install::main" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::install::execute" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::install::execute" "$@"
    else
        log::error "Install not implemented for ${CLI_RESOURCE_NAME}"
        return 1
    fi
}

cli::_handle_uninstall() {
    if command -v "${CLI_RESOURCE_NAME}::uninstall" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::uninstall" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::docker::uninstall" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::docker::uninstall" "$@"
    else
        log::error "Uninstall not implemented for ${CLI_RESOURCE_NAME}"
        return 1
    fi
}

cli::_handle_validate() {
    if command -v "${CLI_RESOURCE_NAME}::validate" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::validate" "$@"
    else
        log::info "Validation not implemented for ${CLI_RESOURCE_NAME}"
    fi
}

cli::_handle_logs() {
    if command -v "${CLI_RESOURCE_NAME}::logs" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::logs" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::docker::logs" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::docker::logs" "$@"
    else
        log::info "Logs not available for ${CLI_RESOURCE_NAME}"
    fi
}

cli::_handle_credentials() {
    if command -v "${CLI_RESOURCE_NAME}::credentials" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::credentials" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::core::credentials" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::core::credentials" "$@"
    else
        log::info "No credentials configured for ${CLI_RESOURCE_NAME}"
    fi
}

#######################################
# Test command handlers
#######################################
cli::_handle_test_all() {
    if command -v "${CLI_RESOURCE_NAME}::test::all" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::test::all" "$@"
    else
        log::info "Running all tests..."
        cli::_handle_test_smoke "$@" && \
        cli::_handle_test_integration "$@" && \
        cli::_handle_test_unit "$@"
    fi
}

cli::_handle_test_integration() {
    if command -v "${CLI_RESOURCE_NAME}::test::integration" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::test::integration" "$@"
    else
        log::info "Integration tests not implemented for ${CLI_RESOURCE_NAME}"
        return 2
    fi
}

cli::_handle_test_unit() {
    if command -v "${CLI_RESOURCE_NAME}::test::unit" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::test::unit" "$@"
    else
        log::info "Unit tests not implemented for ${CLI_RESOURCE_NAME}"
        return 2
    fi
}

cli::_handle_test_smoke() {
    if command -v "${CLI_RESOURCE_NAME}::test::smoke" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::test::smoke" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::status" &>/dev/null; then
        log::info "Running smoke test (status check)..."
        "${CLI_RESOURCE_NAME}::status" "$@"
    else
        log::info "Smoke test not implemented for ${CLI_RESOURCE_NAME}"
        return 2
    fi
}

#######################################
# Content command handlers
#######################################
cli::_handle_content_add() {
    if command -v "${CLI_RESOURCE_NAME}::content::add" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::content::add" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::content" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::content" "add" "$@"
    else
        log::info "Content management not implemented for ${CLI_RESOURCE_NAME}"
        return 2
    fi
}

cli::_handle_content_list() {
    if command -v "${CLI_RESOURCE_NAME}::content::list" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::content::list" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::content" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::content" "list" "$@"
    else
        log::info "Content listing not implemented for ${CLI_RESOURCE_NAME}"
        return 2
    fi
}

cli::_handle_content_get() {
    if command -v "${CLI_RESOURCE_NAME}::content::get" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::content::get" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::content" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::content" "get" "$@"
    else
        log::info "Content retrieval not implemented for ${CLI_RESOURCE_NAME}"
        return 2
    fi
}

cli::_handle_content_remove() {
    if command -v "${CLI_RESOURCE_NAME}::content::remove" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::content::remove" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::content" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::content" "remove" "$@"
    else
        log::info "Content removal not implemented for ${CLI_RESOURCE_NAME}"
        return 2
    fi
}

cli::_handle_content_execute() {
    if command -v "${CLI_RESOURCE_NAME}::content::execute" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::content::execute" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::content" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::content" "execute" "$@"
    else
        log::info "Content execution not implemented for ${CLI_RESOURCE_NAME}"
        return 2
    fi
}

cli::_handle_info() {
    # Parse arguments for JSON output
    local output_format="text"
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                output_format="json"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Load runtime configuration from config/runtime.json
    local runtime_json="${APP_ROOT}/resources/${CLI_RESOURCE_NAME}/config/runtime.json"
    local startup_order=500  # default
    local startup_time="unknown"
    local startup_timeout=30
    local recovery_attempts=2
    local priority="medium"
    local dependencies=()
    
    if [[ -f "$runtime_json" ]]; then
        # Extract values from runtime.json
        startup_order=$(jq -r '.startup_order // 500' "$runtime_json" 2>/dev/null || echo "500")
        startup_time=$(jq -r '.startup_time_estimate // "unknown"' "$runtime_json" 2>/dev/null || echo "unknown")
        startup_timeout=$(jq -r '.startup_timeout // 30' "$runtime_json" 2>/dev/null || echo "30")
        recovery_attempts=$(jq -r '.recovery_attempts // 2' "$runtime_json" 2>/dev/null || echo "2")
        priority=$(jq -r '.priority // "medium"' "$runtime_json" 2>/dev/null || echo "medium")
        
        # Extract dependencies array
        if [[ -f "$runtime_json" ]] && jq -e '.dependencies' "$runtime_json" >/dev/null 2>&1; then
            readarray -t dependencies < <(jq -r '.dependencies[]' "$runtime_json" 2>/dev/null)
        fi
    else
        log::debug "No runtime.json found at $runtime_json, using defaults"
    fi
    
    if [[ "$output_format" == "json" ]]; then
        jq -n \
            --arg name "$CLI_RESOURCE_NAME" \
            --arg description "$CLI_RESOURCE_DESCRIPTION" \
            --argjson startup_order "$startup_order" \
            --argjson dependencies "$(printf '%s\n' "${dependencies[@]}" | jq -R . | jq -s .)" \
            --arg startup_time "$startup_time" \
            --argjson startup_timeout "$startup_timeout" \
            --argjson recovery_attempts "$recovery_attempts" \
            --arg priority "$priority" \
            '{
                name: $name,
                description: $description,
                startup_order: $startup_order,
                dependencies: $dependencies,
                startup_time_estimate: $startup_time,
                startup_timeout: $startup_timeout,
                recovery_attempts: $recovery_attempts,
                priority: $priority
            }'
    else
        echo "📋 Resource Information: $CLI_RESOURCE_NAME"
        echo "   Description: $CLI_RESOURCE_DESCRIPTION"
        echo "   Startup Order: $startup_order"
        echo "   Dependencies: ${dependencies[*]:-none}"
        echo "   Startup Time: $startup_time"
        echo "   Startup Timeout: ${startup_timeout}s"
        echo "   Recovery Attempts: $recovery_attempts"
        echo "   Priority: $priority"
    fi
}

#######################################
# Utility functions
#######################################

# Enable v2.0 mode for a resource
cli::enable_v2_mode() {
    CLI_V2_MODE="v2"
    log::debug "Enabled v2.0 mode for ${CLI_RESOURCE_NAME}"
}

# Check if running in v2 mode
cli::is_v2_mode() {
    [[ "$CLI_V2_MODE" == "v2" ]] || [[ "$CLI_V2_MODE" == "auto" ]]
}

# Backward compatibility helper - show deprecation warning
cli::deprecation_warning() {
    local old_cmd="$1"
    local new_cmd="$2"
    
    if [[ "${SUPPRESS_DEPRECATION_WARNINGS:-}" != "true" ]]; then
        log::warning "Command '$old_cmd' is deprecated. Use '$new_cmd' instead."
        log::warning "This command will be removed in v3.0 (December 2025)"
    fi
}