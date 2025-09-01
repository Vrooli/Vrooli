#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Unified Stop Commands
# 
# Provides a single, comprehensive interface for stopping all types of Vrooli
# components: apps, resources, containers, and processes.
#
# Usage:
#   vrooli stop [target] [options]
#
# This is the new unified stop system that replaces scattered stop commands.
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Path to unified stop manager
STOP_MANAGER="${APP_ROOT}/scripts/lib/lifecycle/stop-manager.sh"

# Show comprehensive help for stop commands
show_stop_help() {
    cat << EOF
🛑 Vrooli Unified Stop System

USAGE:
    vrooli stop [target] [options]

TARGETS:
    (none)              Stop everything (apps, resources, containers, processes)
    all                 Stop everything (explicit)
    apps                Stop only generated apps
    resources           Stop only resources (Docker containers managed by Vrooli)
    containers          Stop all Docker containers
    processes           Stop system processes (Python, Node, etc.)
    <name>              Stop specific app or resource by name

SPECIFIC TARGETING:
    app:<name>          Stop specific app (e.g., app:research-assistant)
    resource:<name>     Stop specific resource (e.g., resource:postgres)

OPTIONS:
    --force             Stop protected apps/resources (immediate termination)
    --confirm           Confirm stopping critical system components (use with --force)
    --dry-run, --check  Show what would be stopped without actually stopping
    --verbose, -v       Show detailed progress and debug information
    --quiet, -q         Minimal output (only errors and final summary)
    --timeout <sec>     Timeout for graceful shutdown (default: 30 seconds)
    --help, -h          Show this help message

SAFETY FEATURES:
    • Runtime protection for critical automation scenarios
    • Graceful shutdown first (SIGTERM), then force (SIGKILL) if needed
    • Dry-run mode to preview actions before execution
    • Comprehensive logging of what was stopped and what failed
    • Smart process detection to avoid stopping system-critical processes
    • Protected apps require --force, critical components require --force --confirm

EXAMPLES:
    vrooli stop                        # Stop everything
    vrooli stop --dry-run              # Preview what would be stopped
    vrooli stop apps                   # Stop only generated apps
    vrooli stop resources              # Stop only Vrooli resources
    vrooli stop containers             # Stop all Docker containers
    vrooli stop postgres               # Stop PostgreSQL resource
    vrooli stop research-assistant     # Stop specific app
    vrooli stop app:research-assistant # Stop specific app (explicit)
    vrooli stop --force --verbose      # Force stop with detailed output
    vrooli stop --timeout 60 apps      # Stop apps with 60s timeout

MIGRATION FROM OLD COMMANDS:
    Old Command                    →    New Command
    ─────────────────────────────────────────────────────
    vrooli app stop-all            →    vrooli stop apps
    vrooli resource stop-all       →    vrooli stop resources
    (no equivalent)                →    vrooli stop            # Stop everything
    (no equivalent)                →    vrooli stop containers # All containers
    
    Note: Old commands still work but will show deprecation warnings.

ADVANCED FEATURES (Coming Soon):
    vrooli stop --only postgres,redis      # Selective stopping
    vrooli stop --exclude windmill         # Stop everything except specified
    vrooli stop "app:*assistant*"          # Pattern matching
    vrooli stop --with-deps postgres       # Stop with dependencies
    vrooli stop --interactive              # Interactive selection mode

For more information: https://docs.vrooli.com/cli/stop
EOF
}

# Parse stop command arguments
parse_stop_args() {
    local target="all"  # Default target
    local -a stop_manager_args=()
    local show_help=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --help|-h)
                show_help=true
                shift
                ;;
            --force)
                # Note: will export in main function, not here (subshell issue)
                stop_manager_args+=("--force")
                shift
                ;;
            --confirm)
                # Note: will export in main function, not here (subshell issue)
                stop_manager_args+=("--confirm")
                shift
                ;;
            --dry-run|--check)
                # Note: will export in main function, not here (subshell issue)
                stop_manager_args+=("--dry-run")
                shift
                ;;
            --verbose|-v)
                # Note: will export in main function, not here (subshell issue)
                stop_manager_args+=("--verbose")
                shift
                ;;
            --quiet|-q)
                # Note: will export in main function, not here (subshell issue)
                stop_manager_args+=("--quiet")
                shift
                ;;
            --timeout)
                if [[ $# -lt 2 ]]; then
                    log::error "Missing timeout value"
                    return 1
                fi
                # Note: will export in main function, not here (subshell issue)
                stop_manager_args+=("--timeout" "$2")
                shift 2
                ;;
            --only)
                if [[ $# -lt 2 ]]; then
                    log::error "Missing list of items for --only"
                    return 1
                fi
                log::warning "Advanced feature --only not yet implemented"
                log::info "For now, stop specific items individually: vrooli stop $2"
                shift 2
                ;;
            --exclude)
                if [[ $# -lt 2 ]]; then
                    log::error "Missing list of items for --exclude" 
                    return 1
                fi
                log::warning "Advanced feature --exclude not yet implemented"
                log::info "For now, stop categories individually: vrooli stop apps, vrooli stop resources, etc."
                shift 2
                ;;
            --interactive)
                log::warning "Advanced feature --interactive not yet implemented"
                log::info "For now, use --dry-run to preview, then run without --dry-run"
                shift
                ;;
            -*)
                log::error "Unknown option: $1"
                log::info "Run 'vrooli stop --help' for usage information"
                return 1
                ;;
            *)
                # This is the target
                if [[ "$target" == "all" ]]; then
                    target="$1"
                else
                    log::error "Multiple targets specified: '$target' and '$1'"
                    log::info "Only one target allowed per command"
                    return 1
                fi
                shift
                ;;
        esac
    done
    
    # Output parsed results
    echo "show_help:$show_help"
    echo "target:$target"
    echo "stop_manager_args:${stop_manager_args[*]}"
}

# Validate target
validate_target() {
    local target="$1"
    
    case "$target" in
        all|apps|app|resources|resource|containers|container|docker|processes|process)
            return 0
            ;;
        app:*|resource:*)
            return 0
            ;;
        *)
            # Check if it's a valid app or resource name
            local name="$target"
            
            # Check for apps
            if [[ -d "${GENERATED_APPS_DIR:-$HOME/generated-apps}/$name" ]]; then
                return 0
            fi
            
            # Check for resources
            if [[ -d "${var_RESOURCES_DIR:-${APP_ROOT}/resources}/$name" ]]; then
                return 0
            fi
            
            # Check with find for resources (in case of nested structure)
            if find "${var_RESOURCES_DIR:-${APP_ROOT}/resources}" -mindepth 1 -maxdepth 2 -type d -name "$name" 2>/dev/null | grep -q .; then
                return 0
            fi
            
            return 1
            ;;
    esac
}

# Main stop command function
stop_command() {
    # Check if stop manager exists
    if [[ ! -f "$STOP_MANAGER" ]]; then
        log::error "Stop manager not found at: $STOP_MANAGER"
        log::error "The unified stop system is not properly installed"
        log::info "Falling back to legacy stop commands..."
        
        # Fallback logic
        case "${1:-all}" in
            apps|app)
                if command -v "${APP_ROOT}/cli/commands/app-commands.sh" >/dev/null 2>&1; then
                    bash "${APP_ROOT}/cli/commands/app-commands.sh" stop-all
                else
                    pkill -f "generated-apps" 2>/dev/null || true
                fi
                ;;
            resources|resource)
                if command -v "${APP_ROOT}/cli/commands/resource-commands.sh" >/dev/null 2>&1; then
                    bash "${APP_ROOT}/cli/commands/resource-commands.sh" stop-all
                else
                    docker stop $(docker ps -q) 2>/dev/null || true
                fi
                ;;
            *)
                log::info "Attempting to stop everything with basic commands..."
                pkill -f "generated-apps" 2>/dev/null || true
                docker stop $(docker ps -q) 2>/dev/null || true
                ;;
        esac
        return $?
    fi
    
    # Parse arguments
    local parsed_output
    parsed_output=$(parse_stop_args "$@")
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # Extract parsed values
    local show_help
    show_help=$(echo "$parsed_output" | grep "^show_help:" | cut -d: -f2)
    local target  
    target=$(echo "$parsed_output" | grep "^target:" | cut -d: -f2)
    local stop_manager_args
    stop_manager_args=$(echo "$parsed_output" | grep "^stop_manager_args:" | cut -d: -f2-)
    
    # Handle help request
    if [[ "$show_help" == "true" ]]; then
        show_stop_help
        return 0
    fi
    
    # Validate target
    if ! validate_target "$target"; then
        log::error "Invalid target: '$target'"
        log::info "Target must be one of: all, apps, resources, containers, processes, or a specific app/resource name"
        log::info ""
        log::info "Available apps:"
        if [[ -d "${GENERATED_APPS_DIR:-$HOME/generated-apps}" ]]; then
            ls "${GENERATED_APPS_DIR:-$HOME/generated-apps}" 2>/dev/null | head -5 | sed 's/^/  /' || echo "  (none found)"
        else
            echo "  (no generated-apps directory)"
        fi
        log::info ""
        log::info "Available resources:"
        if [[ -d "${var_RESOURCES_DIR:-${APP_ROOT}/resources}" ]]; then
            find "${var_RESOURCES_DIR:-${APP_ROOT}/resources}" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | \
                xargs -I {} basename {} | head -5 | sed 's/^/  /' || echo "  (none found)"
        else
            echo "  (no resources directory)"
        fi
        log::info ""
        log::info "Run 'vrooli stop --help' for more information"
        return 1
    fi
    
    # Set environment variables based on parsed arguments
    # (This fixes the subshell export issue in parse_stop_args)
    if [[ "$stop_manager_args" == *"--dry-run"* ]]; then
        export DRY_RUN=true
    fi
    if [[ "$stop_manager_args" == *"--force"* ]]; then
        export FORCE_STOP=true
    fi
    if [[ "$stop_manager_args" == *"--confirm"* ]]; then
        export CONFIRM_CRITICAL=true
    fi
    if [[ "$stop_manager_args" == *"--verbose"* ]]; then
        export VERBOSE=true
    fi
    if [[ "$stop_manager_args" == *"--quiet"* ]]; then
        export QUIET=true
    fi
    if [[ "$stop_manager_args" == *"--timeout"* ]]; then
        # Extract timeout value (this is a simple approach)
        local timeout_value
        timeout_value=$(echo "$stop_manager_args" | sed -n 's/.*--timeout \([0-9]*\).*/\1/p')
        export STOP_TIMEOUT="$timeout_value"
    fi
    
    # Source and execute stop manager
    # shellcheck disable=SC1090
    source "$STOP_MANAGER"
    
    # Environment variables are already set (DRY_RUN, FORCE_STOP, etc.)
    # No need to pass arguments again since stop-commands.sh already parsed them
    
    # Execute stop manager with just the target
    stop::main "$target"
}

# Alias for backwards compatibility and clarity
stop_main() {
    stop_command "$@"
}

# If script is run directly, execute stop command
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    stop_command "$@"
    exit $?
fi

# When sourced, make functions available
true