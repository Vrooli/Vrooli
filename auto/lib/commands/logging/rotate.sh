#!/usr/bin/env bash

# Rotate Command Module - Log and File Rotation
# Handles rotation of logs, events, and temporary file cleanup

set -euo pipefail

#######################################
# Rotate log files and cleanup
# Arguments: 
#   --events [N] - Rotate events file, keep N old files (default: ROTATE_KEEP)
#   --temp       - Clean up temporary files
# Returns: 0 on success
#######################################
cmd_execute() {
    local rotate_events=false
    local rotate_temp=false
    local events_keep="$ROTATE_KEEP"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --events)
                rotate_events=true
                # Check if next argument is a number for keep count
                if [[ $# -gt 1 ]] && [[ "$2" =~ ^[0-9]+$ ]]; then
                    events_keep="$2"
                    shift 2
                else
                    shift
                fi
                ;;
            --temp)
                rotate_temp=true
                shift
                ;;
            *)
                echo "ERROR: Unknown argument: $1" >&2
                return 1
                ;;
        esac
    done
    
    # Always rotate main log
    rotate_main_log
    
    # Rotate events if requested
    if [[ "$rotate_events" == "true" ]]; then
        rotate_events_file "$events_keep"
    fi
    
    # Clean temp files if requested
    if [[ "$rotate_temp" == "true" ]]; then
        cleanup_temp_files
    fi
    
    return 0
}

#######################################
# Rotate the main log file
#######################################
rotate_main_log() {
    if [[ -f "$LOG_FILE" ]]; then
        local timestamp; timestamp=$(date '+%Y%m%d_%H%M%S')
        local rotated_log="${LOG_FILE}.${timestamp}"
        
        mv "$LOG_FILE" "$rotated_log"
        echo "Rotated log to $rotated_log"
        
        # Start fresh log with rotation notice
        log_with_timestamp "Log rotated from $rotated_log"
    else
        echo "No log to rotate"
    fi
}

#######################################
# Rotate events file with pruning
# Arguments:
#   $1 - Number of old files to keep
#######################################
rotate_events_file() {
    local keep="$1"
    
    if [[ -f "$EVENTS_JSONL" ]]; then
        local timestamp; timestamp=$(date '+%Y%m%d_%H%M%S')
        local rotated_events="${EVENTS_JSONL}.${timestamp}"
        
        mv "$EVENTS_JSONL" "$rotated_events"
        echo "Rotated events to $rotated_events"
        
        # Prune old events files safely
        local events_dir; events_dir=${EVENTS_JSONL%/*}
        local events_basename; events_basename=$(basename "$EVENTS_JSONL")
        
        # Find old events files and remove excess
        local removed_count=0
        if command -v find >/dev/null 2>&1; then
            # Use find with printf for better timestamp handling
            while IFS= read -r old_file; do
                rm -f "$old_file"
                ((removed_count++))
            done < <(find "$events_dir" -maxdepth 1 -type f -name "${events_basename}.*" -printf '%T@ %p\n' 2>/dev/null | \
                     sort -nr | \
                     awk -v k="$keep" 'NR>(k+1){print $2}')
        else
            # Fallback using ls and basic sorting
            while IFS= read -r old_file; do
                [[ -f "$old_file" ]] && rm -f "$old_file" && ((removed_count++))
            done < <(ls -t "${events_dir}/${events_basename}."* 2>/dev/null | tail -n +$((keep + 2)))
        fi
        
        if [[ $removed_count -gt 0 ]]; then
            echo "Pruned $removed_count old events files (keeping $keep)"
        fi
    else
        echo "No events file to rotate"
    fi
}

#######################################
# Clean up temporary files
#######################################
cleanup_temp_files() {
    if [[ -d "$TMP_DIR" ]]; then
        local cleaned=0
        
        # Clean empty temporary files
        if command -v find >/dev/null 2>&1; then
            # Use find to safely delete empty tmp files
            while IFS= read -r tmp_file; do
                rm -f "$tmp_file"
                ((cleaned++))
            done < <(find "$TMP_DIR" -name "tmp.*" -size 0 -print 2>/dev/null)
        else
            # Fallback method
            for tmp_file in "$TMP_DIR"/tmp.*; do
                if [[ -f "$tmp_file" && ! -s "$tmp_file" ]]; then
                    rm -f "$tmp_file"
                    ((cleaned++))
                fi
            done 2>/dev/null || true
        fi
        
        echo "Cleaned up $cleaned empty temporary files"
    else
        echo "No temp directory to clean"
    fi
}

#######################################
# Validate rotate command arguments
# Arguments: Command arguments
# Returns: 0 if valid, 1 if invalid
#######################################
cmd_validate() {
    local expecting_number=false
    
    for arg in "$@"; do
        if [[ "$expecting_number" == "true" ]]; then
            if ! [[ "$arg" =~ ^[0-9]+$ ]]; then
                echo "ERROR: --events keep count must be a number, got: $arg" >&2
                return 1
            fi
            expecting_number=false
        elif [[ "$arg" == "--events" ]]; then
            # Next argument might be a number
            expecting_number=false  # Optional number
        elif [[ "$arg" == "--temp" ]]; then
            # Valid argument
            :
        else
            echo "ERROR: Invalid argument: $arg" >&2
            echo "Valid arguments: --events [N], --temp" >&2
            return 1
        fi
    done
    
    return 0
}

#######################################
# Show help for rotate command
#######################################
cmd_help() {
    cat << EOF
rotate - Rotate log files and cleanup temporary files

Usage: rotate [--events [N]] [--temp]

Arguments:
  --events [N]   Rotate events file, keep N old files (default: $ROTATE_KEEP)
  --temp         Clean up empty temporary files

Description:
  Rotates log files by adding a timestamp suffix and starting fresh files.
  Always rotates the main log file. Optionally rotates events and cleans
  temporary files.

  Main log rotation:
  - Moves LOG_FILE to LOG_FILE.YYYYMMDD_HHMMSS
  - Creates fresh log file
  
  Events rotation (--events):
  - Moves events file with timestamp suffix
  - Removes old events files, keeping N most recent
  - Default keep count: $ROTATE_KEEP
  
  Temp cleanup (--temp):
  - Removes empty temporary files from tmp directory
  - Only removes files matching tmp.* pattern with 0 bytes

Examples:
  task-manager.sh --task resource-improvement rotate
  task-manager.sh --task resource-improvement rotate --events 5
  task-manager.sh --task resource-improvement rotate --events --temp
  manage-resource-loop.sh rotate --events 10 --temp

See also: logs, status
EOF
}