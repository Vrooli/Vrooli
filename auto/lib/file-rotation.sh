#!/usr/bin/env bash

# File Rotation Library - Log and event file rotation management
# Part of the modular loop system
#
# This module provides focused file rotation and size management functions
# for log files, event files, and temporary files.

set -euo pipefail

# Prevent multiple sourcing
if [[ -n "${_AUTO_FILE_ROTATION_SOURCED:-}" ]]; then
    return 0
fi
readonly _AUTO_FILE_ROTATION_SOURCED=1

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
LIB_DIR="${APP_ROOT}/auto/lib"
# shellcheck disable=SC1091
source "$LIB_DIR/constants.sh"

# Check if a file exceeds size limit
# Args: $1 - file path, $2 - max size in bytes
# Returns: 0 if exceeds limit, 1 otherwise
file_rotation::check_size_exceeded() {
    local file_path="$1"
    local max_size="${2:-$LOG_MAX_BYTES}"
    
    if [[ ! -f "$file_path" ]]; then
        return 1
    fi
    
    local size; size=$(stat -c%s "$file_path" 2>/dev/null || echo 0)
    if [[ $size -gt $max_size ]]; then
        return 0
    else
        return 1
    fi
}

# Check if a file exceeds line count limit
# Args: $1 - file path, $2 - max lines
# Returns: 0 if exceeds limit, 1 otherwise
file_rotation::check_lines_exceeded() {
    local file_path="$1"
    local max_lines="${2:-$EVENTS_MAX_LINES}"
    
    if [[ ! -f "$file_path" ]]; then
        return 1
    fi
    
    local lines; lines=$(wc -l < "$file_path" 2>/dev/null || echo 0)
    if [[ $lines -gt $max_lines ]]; then
        return 0
    else
        return 1
    fi
}

# Rotate a file with timestamp suffix
# Args: $1 - file path, $2 - reason (optional)
# Returns: 0 on success, 1 on failure
# Outputs: new rotated file path
file_rotation::rotate_with_timestamp() {
    local file_path="$1"
    local reason="${2:-manual}"
    
    if [[ ! -f "$file_path" ]]; then
        return 1
    fi
    
    local ts; ts=$(date '+%Y%m%d_%H%M%S')
    local rotated_path="${file_path}.${ts}"
    
    if mv "$file_path" "$rotated_path" 2>/dev/null; then
        if declare -F log_with_timestamp >/dev/null 2>&1; then
            log_with_timestamp "Rotated ${file_path##*/} to ${rotated_path##*/} (reason: $reason)"
        fi
        echo "$rotated_path"
        return 0
    else
        return 1
    fi
}

# Rotate log file if it exceeds size limit
# Args: $1 - log file path, $2 - max size (optional, defaults to LOG_MAX_BYTES)
# Returns: 0 if rotated, 1 if not needed or failed
file_rotation::rotate_log_if_needed() {
    local log_file="$1"
    local max_size="${2:-$LOG_MAX_BYTES}"
    
    if file_rotation::check_size_exceeded "$log_file" "$max_size"; then
        local size; size=$(stat -c%s "$log_file" 2>/dev/null || echo 0)
        if file_rotation::rotate_with_timestamp "$log_file" "size_exceeded:${size}>${max_size}"; then
            return 0
        fi
    fi
    return 1
}

# Rotate events file if it exceeds line limit
# Args: $1 - events file path, $2 - max lines (optional, defaults to EVENTS_MAX_LINES)
# Returns: 0 if rotated, 1 if not needed or failed
file_rotation::rotate_events_if_needed() {
    local events_file="$1"
    local max_lines="${2:-$EVENTS_MAX_LINES}"
    
    if file_rotation::check_lines_exceeded "$events_file" "$max_lines"; then
        local lines; lines=$(wc -l < "$events_file" 2>/dev/null || echo 0)
        if file_rotation::rotate_with_timestamp "$events_file" "lines_exceeded:${lines}>${max_lines}"; then
            return 0
        fi
    fi
    return 1
}

# Prune old rotated files keeping only N most recent
# Args: $1 - base file path, $2 - keep count (optional, defaults to ROTATE_KEEP)
# Returns: count of files pruned
file_rotation::prune_rotated_files() {
    local base_path="$1"
    local keep="${2:-$ROTATE_KEEP}"
    
    if [[ ! -d "$(dirname "$base_path")" ]]; then
        echo "0"
        return 1
    fi
    
    local pruned=0
    local dir; dir=$(dirname "$base_path")
    local base; base=$(basename "$base_path")
    
    # Find and prune old rotated files
    while IFS= read -r old_file; do
        if rm -f "$old_file" 2>/dev/null; then
            ((pruned++))
            if declare -F log_with_timestamp >/dev/null 2>&1; then
                log_with_timestamp "Pruned old rotated file: ${old_file##*/}"
            fi
        fi
    done < <(find "$dir" -maxdepth 1 -type f -name "${base}.*" -printf '%T@ %p\n' 2>/dev/null | \
             sort -nr | awk -v k="$keep" 'NR>(k){print $2}')
    
    echo "$pruned"
    return 0
}

# Clean empty temporary files
# Args: $1 - temp directory path
# Returns: count of files cleaned
file_rotation::clean_empty_temp_files() {
    local temp_dir="$1"
    
    if [[ ! -d "$temp_dir" ]]; then
        echo "0"
        return 1
    fi
    
    local cleaned=0
    cleaned=$(find "$temp_dir" -name "tmp.*" -size 0 -delete -print 2>/dev/null | wc -l)
    
    if [[ $cleaned -gt 0 ]] && declare -F log_with_timestamp >/dev/null 2>&1; then
        log_with_timestamp "Cleaned $cleaned empty temporary files from $temp_dir"
    fi
    
    echo "$cleaned"
    return 0
}

# Clean old iteration logs based on age or count
# Args: $1 - iterations directory, $2 - keep count or age in days, $3 - mode ("count" or "age")
# Returns: count of files cleaned
file_rotation::clean_iteration_logs() {
    local iter_dir="$1"
    local keep_value="${2:-50}"
    local mode="${3:-count}"
    
    if [[ ! -d "$iter_dir" ]]; then
        echo "0"
        return 1
    fi
    
    local cleaned=0
    
    if [[ "$mode" == "age" ]]; then
        # Clean files older than N days
        cleaned=$(find "$iter_dir" -name "iter-*.log" -mtime "+$keep_value" -delete -print 2>/dev/null | wc -l)
    else
        # Keep only N most recent files
        while IFS= read -r old_file; do
            if rm -f "$old_file" 2>/dev/null; then
                ((cleaned++))
            fi
        done < <(find "$iter_dir" -name "iter-*.log" -printf '%T@ %p\n' 2>/dev/null | \
                 sort -nr | awk -v k="$keep_value" 'NR>(k){print $2}')
    fi
    
    if [[ $cleaned -gt 0 ]] && declare -F log_with_timestamp >/dev/null 2>&1; then
        log_with_timestamp "Cleaned $cleaned old iteration logs from $iter_dir (mode: $mode, value: $keep_value)"
    fi
    
    echo "$cleaned"
    return 0
}

# Perform comprehensive rotation check for all managed files
# Args: $1 - log file, $2 - events file, $3 - temp dir, $4 - iterations dir (all optional)
# Returns: 0 if any rotation performed
file_rotation::check_and_rotate_all() {
    local log_file="${1:-}"
    local events_file="${2:-}"
    local temp_dir="${3:-}"
    local iter_dir="${4:-}"
    local any_rotated=1
    
    # Check and rotate log file
    if [[ -n "$log_file" ]] && file_rotation::rotate_log_if_needed "$log_file"; then
        any_rotated=0
    fi
    
    # Check and rotate events file
    if [[ -n "$events_file" ]] && file_rotation::rotate_events_if_needed "$events_file"; then
        any_rotated=0
    fi
    
    # Clean temporary files
    if [[ -n "$temp_dir" ]]; then
        file_rotation::clean_empty_temp_files "$temp_dir" >/dev/null
    fi
    
    # Clean old iteration logs (keep last 50 by default)
    if [[ -n "$iter_dir" ]]; then
        file_rotation::clean_iteration_logs "$iter_dir" 50 "count" >/dev/null
    fi
    
    return $any_rotated
}

# Export functions for external use
export -f file_rotation::check_size_exceeded
export -f file_rotation::check_lines_exceeded
export -f file_rotation::rotate_with_timestamp
export -f file_rotation::rotate_log_if_needed
export -f file_rotation::rotate_events_if_needed
export -f file_rotation::prune_rotated_files
export -f file_rotation::clean_empty_temp_files
export -f file_rotation::clean_iteration_logs
export -f file_rotation::check_and_rotate_all