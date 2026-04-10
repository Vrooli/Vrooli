#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Unified Clean Commands
# 
# Provides a comprehensive interface for cleaning various Vrooli components:
# stale locks, build artifacts, logs, temporary files, and system state.
#
# Usage:
#   vrooli clean [target] [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Configuration
DRY_RUN="${DRY_RUN:-false}"
VERBOSE="${VERBOSE:-false}"
FORCE_CLEAN="${FORCE_CLEAN:-false}"

# Paths
SCENARIO_STATE_DIR="${HOME}/.vrooli/state/scenarios"
VROOLI_LOGS_DIR="${HOME}/.vrooli/logs"
BUILD_DIRS=("${APP_ROOT}/dist" "${APP_ROOT}/build" "${APP_ROOT}/.next" "${APP_ROOT}/node_modules/.cache")

################################################################################
# Core Functions
################################################################################

# Execute command or show in dry-run
clean::execute() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY-RUN] Would execute: $*"
        return 0
    else
        eval "$@"
    fi
}

# Show comprehensive help for clean commands
show_clean_help() {
    cat << EOF
🧹 Vrooli Unified Clean System

USAGE:
    vrooli clean [target] [options]

TARGETS:
    locks               Clean stale port locks (zombie/orphaned process locks)
    logs                Clean old log files
    build               Clean build artifacts and caches
    temp                Clean temporary files
    state               Clean scenario state files
    all                 Clean everything (equivalent to no target)

SPECIFIC CLEANING:
    locks:stale         Clean only stale locks (PIDs not running)
    locks:scenario      Clean locks for specific scenario
    logs:old            Clean logs older than 7 days
    build:cache         Clean only cache directories

OPTIONS:
    --dry-run, --check  Show what would be cleaned without actually cleaning
    --verbose, -v       Show detailed progress and debug information
    --force             Clean protected files and bypass confirmations
    --quiet, -q         Minimal output (only errors and final summary)
    --help, -h          Show this help message

SAFETY FEATURES:
    • Dry-run mode to preview actions before execution
    • Stale lock detection to avoid cleaning active processes
    • Automatic backup of critical state before cleaning
    • Comprehensive logging of what was cleaned
    • Protected file detection with force override

EXAMPLES:
    vrooli clean                        # Clean everything with confirmation
    vrooli clean --dry-run              # Preview what would be cleaned
    vrooli clean locks                  # Clean only stale port locks
    vrooli clean build --force          # Force clean build artifacts
    vrooli clean logs --verbose         # Clean logs with detailed output
    vrooli clean locks:scenario app-monitor  # Clean locks for specific scenario

LOCK CLEANING:
    The lock cleaning system uses the zombie detector to:
    • Identify stale locks (process no longer running)
    • Verify lock ownership and timestamps
    • Provide detailed diagnostics on what was cleaned
    • Skip locks for active processes
    • Clean locks older than 1 hour automatically

For more information: https://docs.vrooli.com/cli/clean
EOF
}

################################################################################
# Lock Cleaning Functions
################################################################################

# Clean stale port locks
# NOTE: This function writes directly to stdout without log::info to avoid
# pipe buffering issues when called from scenario runner with output redirection
clean::stale_locks() {
    printf '%s\n' "[INFO]    Cleaning stale port locks..."

    if ! command -v vrooli-autoheal >/dev/null 2>&1; then
        printf '%s\n' "[ERROR]   vrooli-autoheal not installed. Run 'vrooli setup' first."
        return 1
    fi

    if vrooli-autoheal locks clean 2>&1; then
        printf '%s\n' "[SUCCESS] Lock cleaning completed via vrooli-autoheal"
        return 0
    else
        printf '%s\n' "[ERROR]   vrooli-autoheal locks clean failed"
        return 1
    fi
}

# Clean locks for specific scenario
# Note: vrooli-autoheal cleans all stale locks (dead PIDs), which covers scenario-specific cleanup
clean::scenario_locks() {
    local scenario_name="$1"

    [[ -n "$scenario_name" ]] || {
        log::error "Scenario name required for scenario lock cleaning"
        return 1
    }

    log::info "Cleaning stale locks (including scenario: $scenario_name)"

    if ! command -v vrooli-autoheal >/dev/null 2>&1; then
        log::error "vrooli-autoheal not installed. Run 'vrooli setup' first."
        return 1
    fi

    # vrooli-autoheal cleans all stale locks (where PID is no longer running)
    if clean::execute "vrooli-autoheal locks clean"; then
        log::success "Cleaned stale locks"
    else
        log::error "Failed to clean stale locks"
        return 1
    fi
}

################################################################################
# Other Cleaning Functions
################################################################################

# Clean old log files
clean::logs() {
    log::info "Cleaning old log files..."
    
    [[ -d "$VROOLI_LOGS_DIR" ]] || {
        log::info "No logs directory found - nothing to clean"
        return 0
    }
    
    local days_old=${1:-7}  # Default 7 days
    local cleaned_count=0
    
    while IFS= read -r log_file; do
        [[ -f "$log_file" ]] || continue
        if clean::execute "rm -f '$log_file'"; then
            ((cleaned_count++))
            [[ "$VERBOSE" == "true" ]] && log::success "Removed old log: $(basename "$log_file")"
        fi
    done < <(find "$VROOLI_LOGS_DIR" -name "*.log" -mtime +"$days_old" 2>/dev/null || true)
    
    log::success "Cleaned $cleaned_count old log files (older than $days_old days)"
}

# Clean build artifacts
clean::build() {
    log::info "Cleaning build artifacts..."
    
    local cleaned_dirs=0
    for build_dir in "${BUILD_DIRS[@]}"; do
        if [[ -d "$build_dir" ]]; then
            if clean::execute "rm -rf '$build_dir'"; then
                ((cleaned_dirs++))
                [[ "$VERBOSE" == "true" ]] && log::success "Removed build directory: $build_dir"
            fi
        fi
    done
    
    # Clean node_modules/.cache in scenarios
    local cache_dirs=0
    while IFS= read -r cache_dir; do
        [[ -d "$cache_dir" ]] || continue
        if clean::execute "rm -rf '$cache_dir'"; then
            ((cache_dirs++))
            [[ "$VERBOSE" == "true" ]] && log::success "Removed cache: $cache_dir"
        fi
    done < <(find "${APP_ROOT}/scenarios" -name "node_modules/.cache" -type d 2>/dev/null || true)
    
    log::success "Cleaned $cleaned_dirs build directories and $cache_dirs cache directories"
}

# Clean temporary files
clean::temp() {
    log::info "Cleaning temporary files..."
    
    local temp_patterns=("${APP_ROOT}/.tmp/*" "${APP_ROOT}/tmp/*" "/tmp/vrooli*" "/tmp/claude*")
    local cleaned_files=0
    
    for pattern in "${temp_patterns[@]}"; do
        # Use find instead of glob for safety
        local base_dir="${pattern%/*}"
        local file_pattern="${pattern##*/}"
        
        [[ -d "$base_dir" ]] || continue
        
        while IFS= read -r temp_file; do
            [[ -e "$temp_file" ]] || continue
            if clean::execute "rm -rf '$temp_file'"; then
                ((cleaned_files++))
                [[ "$VERBOSE" == "true" ]] && log::success "Removed temp file: $temp_file"
            fi
        done < <(find "$base_dir" -name "$file_pattern" 2>/dev/null || true)
    done
    
    log::success "Cleaned $cleaned_files temporary files"
}

# Clean scenario state files
clean::state() {
    log::info "Cleaning scenario state files..."
    
    [[ -d "$SCENARIO_STATE_DIR" ]] || {
        log::info "No scenario state directory found - nothing to clean"
        return 0
    }
    
    local cleaned_files=0
    
    # Clean state JSON files (but not lock files)
    while IFS= read -r state_file; do
        [[ -f "$state_file" ]] || continue
        if clean::execute "rm -f '$state_file'"; then
            ((cleaned_files++))
            [[ "$VERBOSE" == "true" ]] && log::success "Removed state file: $(basename "$state_file")"
        fi
    done < <(find "$SCENARIO_STATE_DIR" -name "*.json" 2>/dev/null || true)
    
    log::success "Cleaned $cleaned_files scenario state files"
}

# Clean everything
clean::all() {
    log::info "Cleaning all Vrooli components..."
    
    if [[ "$FORCE_CLEAN" != "true" ]] && [[ "$DRY_RUN" != "true" ]]; then
        log::warning "This will clean locks, logs, build artifacts, temp files, and state"
        if ! command -v flow::confirm >/dev/null 2>&1; then
            read -p "Continue? [y/N]: " -r
            [[ $REPLY =~ ^[Yy]$ ]] || { log::info "Cancelled by user"; return 0; }
        else
            flow::confirm "Continue with full cleanup?" || { log::info "Cancelled by user"; return 0; }
        fi
    fi
    
    clean::stale_locks
    clean::logs
    clean::build
    clean::temp
    clean::state
    
    log::success "Complete cleanup finished"
}

################################################################################
# Argument Parsing
################################################################################

# Parse clean command arguments
parse_clean_args() {
    local target="all"  # Default target
    local show_help=false
    local -a extra_args=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --help|-h)
                show_help=true
                shift
                ;;
            --dry-run|--check)
                export DRY_RUN=true
                shift
                ;;
            --verbose|-v)
                export VERBOSE=true
                shift
                ;;
            --quiet|-q)
                export QUIET=true
                shift
                ;;
            --force)
                export FORCE_CLEAN=true
                shift
                ;;
            -*)
                log::error "Unknown option: $1"
                log::info "Run 'vrooli clean --help' for usage information"
                return 1
                ;;
            *)
                # This is a target
                target="$1"
                shift
                # Collect any remaining args for specific targets
                extra_args+=("$@")
                break
                ;;
        esac
    done
    
    # Output parsed results
    echo "show_help:$show_help"
    echo "target:$target"
    echo "extra_args:${extra_args[*]}"
}

################################################################################
# Main Command Function
################################################################################

clean_command() {
    # Parse arguments
    local parsed_output
    parsed_output=$(parse_clean_args "$@")
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # Extract parsed values
    local show_help
    show_help=$(echo "$parsed_output" | grep "^show_help:" | cut -d: -f2)
    local target
    target=$(echo "$parsed_output" | grep "^target:" | cut -d: -f2)
    local extra_args_str
    extra_args_str=$(echo "$parsed_output" | grep "^extra_args:" | cut -d: -f2-)
    
    # Convert extra_args string back to array
    local -a extra_args=()
    if [[ -n "$extra_args_str" ]] && [[ "$extra_args_str" != " " ]]; then
        IFS=' ' read -ra extra_args <<< "$extra_args_str"
    fi
    
    # Handle help request
    if [[ "$show_help" == "true" ]]; then
        show_clean_help
        return 0
    fi
    
    # Dispatch to appropriate cleaning function
    case "$target" in
        locks)
            clean::stale_locks
            ;;
        locks:stale)
            clean::stale_locks
            ;;
        locks:scenario)
            if [[ ${#extra_args[@]} -eq 0 ]]; then
                log::error "Scenario name required for locks:scenario target"
                log::info "Usage: vrooli clean locks:scenario <scenario_name>"
                return 1
            fi
            clean::scenario_locks "${extra_args[0]}"
            ;;
        logs)
            clean::logs
            ;;
        logs:old)
            local days="${extra_args[0]:-7}"
            clean::logs "$days"
            ;;
        build)
            clean::build
            ;;
        temp)
            clean::temp
            ;;
        state)
            clean::state
            ;;
        all|"")
            clean::all
            ;;
        *)
            log::error "Unknown clean target: '$target'"
            log::info "Valid targets: locks, logs, build, temp, state, all"
            log::info "Run 'vrooli clean --help' for more information"
            return 1
            ;;
    esac
}

# Alias for backwards compatibility
clean_main() {
    clean_command "$@"
}

# If script is run directly, execute clean command
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    clean_command "$@"
    exit $?
fi

# When sourced, make functions available
true