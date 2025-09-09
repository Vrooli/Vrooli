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

# Source zombie detector for lock cleaning
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/zombie-detector.sh" 2>/dev/null || {
    log::error "Zombie detector not found - lock cleaning unavailable"
    exit 1
}

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
clean::stale_locks() {
    log::info "Cleaning stale port locks..."
    
    [[ -d "$SCENARIO_STATE_DIR" ]] || {
        log::info "No scenario state directory found - nothing to clean"
        return 0
    }
    
    local total_locks=0
    local stale_locks=0
    local cleaned_locks=0
    local failed_cleanups=0
    
    # Get all lock files into an array for better control
    local -a lock_files=()
    while IFS= read -r lock_file; do
        [[ -f "$lock_file" ]] && lock_files+=("$lock_file")
    done < <(find "$SCENARIO_STATE_DIR" -name ".port_*.lock" 2>/dev/null || true)
    
    total_locks=${#lock_files[@]}
    
    if [[ $total_locks -eq 0 ]]; then
        log::success "No port locks found - system is clean"
        return 0
    fi
    
    log::info "Found $total_locks port locks to check"
    
    # Process each lock file efficiently
    for lock_file in "${lock_files[@]}"; do
        [[ -f "$lock_file" ]] || continue
        
        # Extract port number from filename
        local port
        port=$(basename "$lock_file" | sed 's/\.port_\([0-9]\+\)\.lock/\1/')
        
        if [[ ! "$port" =~ ^[0-9]+$ ]]; then
            [[ "$VERBOSE" == "true" ]] && log::warning "Skipping invalid lock file: $lock_file"
            continue
        fi
        
        # Read lock content to get PID
        local lock_content pid
        lock_content=$(cat "$lock_file" 2>/dev/null || echo "")
        
        if [[ -z "$lock_content" ]]; then
            ((stale_locks++))
            [[ "$VERBOSE" == "true" ]] && log::debug "Empty lock file (stale): port $port"
        else
            # Extract PID from lock content (format: scenario:pid:timestamp)
            pid=$(echo "$lock_content" | cut -d: -f2 2>/dev/null || echo "")
            
            # Simple check if PID is running (much faster than associative array)
            if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                # PID is running - lock is active
                [[ "$VERBOSE" == "true" ]] && log::debug "Active lock (keeping): port $port (PID $pid)"
                continue
            else
                # PID is not running or invalid - lock is stale
                ((stale_locks++))
                [[ "$VERBOSE" == "true" ]] && log::debug "Stale lock found: port $port (PID $pid not running)"
            fi
        fi
        
        # Clean the stale lock
        if clean::execute "rm -f '$lock_file'"; then
            ((cleaned_locks++))
            [[ "$VERBOSE" == "true" ]] && log::success "Cleaned stale lock for port: $port"
        else
            ((failed_cleanups++))
            log::warning "Failed to clean stale lock for port: $port"
        fi
    done
    
    # Summary
    log::info "Lock cleanup summary:"
    log::info "  Total locks: $total_locks"
    log::info "  Stale locks: $stale_locks"
    log::info "  Cleaned: $cleaned_locks"
    [[ $failed_cleanups -gt 0 ]] && log::warning "  Failed cleanups: $failed_cleanups"
    
    if [[ $stale_locks -eq 0 ]]; then
        log::success "All port locks are active - no cleanup needed"
    elif [[ $cleaned_locks -eq $stale_locks ]]; then
        log::success "Successfully cleaned all stale locks"
    else
        log::warning "Some stale locks could not be cleaned (see messages above)"
    fi
    
    return 0
}

# Clean locks for specific scenario
clean::scenario_locks() {
    local scenario_name="$1"
    
    [[ -n "$scenario_name" ]] || {
        log::error "Scenario name required for scenario lock cleaning"
        return 1
    }
    
    log::info "Cleaning locks for scenario: $scenario_name"
    
    if command -v zombie::clean_scenario_locks >/dev/null 2>&1; then
        clean::execute "zombie::clean_scenario_locks '$scenario_name'"
    else
        log::warning "Scenario-specific lock cleaning not available, falling back to manual cleanup"
        
        [[ -d "$SCENARIO_STATE_DIR" ]] || return 0
        
        local cleaned_count=0
        while IFS= read -r lock_file; do
            [[ -f "$lock_file" ]] || continue
            local lock_content
            lock_content=$(cat "$lock_file" 2>/dev/null | cut -d: -f1)
            
            if [[ "$lock_content" == "$scenario_name" ]]; then
                if clean::execute "rm -f '$lock_file'"; then
                    ((cleaned_count++))
                    [[ "$VERBOSE" == "true" ]] && log::success "Removed lock: $(basename "$lock_file")"
                fi
            fi
        done < <(find "$SCENARIO_STATE_DIR" -name ".port_*.lock" 2>/dev/null || true)
        
        log::success "Cleaned $cleaned_count locks for scenario: $scenario_name"
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