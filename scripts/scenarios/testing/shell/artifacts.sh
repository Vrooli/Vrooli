#!/usr/bin/env bash
# Artifact management utilities for test logs and outputs
set -euo pipefail

# Global artifact configuration
TESTING_ARTIFACTS_MAX_LOGS=10
TESTING_ARTIFACTS_COMPRESS_OLD=false
TESTING_ARTIFACTS_RETENTION_DAYS=7
TESTING_ARTIFACTS_AUTO_CLEANUP=true
TESTING_ARTIFACTS_DIR=""

# Configure artifact management settings
# Usage: testing::artifacts::configure [options]
# Options:
#   --dir PATH                  Directory for artifacts (required)
#   --max-logs COUNT           Maximum number of logs to keep per phase (default: 10)
#   --compress-old BOOL        Compress old logs with gzip (default: false)
#   --retention-days DAYS      Days to retain logs before deletion (default: 7)
#   --auto-cleanup BOOL        Enable automatic cleanup (default: true)
testing::artifacts::configure() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dir)
                TESTING_ARTIFACTS_DIR="$2"
                shift 2
                ;;
            --max-logs)
                TESTING_ARTIFACTS_MAX_LOGS="$2"
                shift 2
                ;;
            --compress-old)
                TESTING_ARTIFACTS_COMPRESS_OLD="$2"
                shift 2
                ;;
            --retention-days)
                TESTING_ARTIFACTS_RETENTION_DAYS="$2"
                shift 2
                ;;
            --auto-cleanup)
                TESTING_ARTIFACTS_AUTO_CLEANUP="$2"
                shift 2
                ;;
            *)
                echo "Unknown option to testing::artifacts::configure: $1" >&2
                return 1
                ;;
        esac
    done

    if [ -z "$TESTING_ARTIFACTS_DIR" ]; then
        echo "ERROR: --dir is required for artifact configuration" >&2
        return 1
    fi

    mkdir -p "$TESTING_ARTIFACTS_DIR"
    
    # Perform initial cleanup if enabled
    if [ "$TESTING_ARTIFACTS_AUTO_CLEANUP" = "true" ]; then
        testing::artifacts::cleanup
    fi
}

# Clean up old artifacts based on retention policy
# Usage: testing::artifacts::cleanup [phase]
testing::artifacts::cleanup() {
    local phase="${1:-}"
    
    if [ -z "$TESTING_ARTIFACTS_DIR" ] || [ ! -d "$TESTING_ARTIFACTS_DIR" ]; then
        return 0
    fi
    
    local pattern="*.log"
    if [ -n "$phase" ]; then
        pattern="${phase}-*.log"
    fi
    
    # Remove logs older than retention days
    if [ "$TESTING_ARTIFACTS_RETENTION_DAYS" -gt 0 ]; then
        find "$TESTING_ARTIFACTS_DIR" -name "$pattern" -type f -mtime +$TESTING_ARTIFACTS_RETENTION_DAYS -delete 2>/dev/null || true
        find "$TESTING_ARTIFACTS_DIR" -name "${pattern}.gz" -type f -mtime +$TESTING_ARTIFACTS_RETENTION_DAYS -delete 2>/dev/null || true
    fi
    
    # Keep only the most recent logs per phase
    if [ "$TESTING_ARTIFACTS_MAX_LOGS" -gt 0 ]; then
        for phase_prefix in structure dependencies unit integration business performance; do
            if [ -n "$phase" ] && [ "$phase" != "$phase_prefix" ]; then
                continue
            fi
            
            # Count all logs for this phase (both .log and .log.gz)
            local uncompressed_logs=($(ls -t "$TESTING_ARTIFACTS_DIR/${phase_prefix}-"*.log 2>/dev/null || true))
            local compressed_logs=($(ls -t "$TESTING_ARTIFACTS_DIR/${phase_prefix}-"*.log.gz 2>/dev/null || true))
            local all_logs=("${uncompressed_logs[@]}" "${compressed_logs[@]}")
            local total_count=${#all_logs[@]}
            
            if [ $total_count -gt $TESTING_ARTIFACTS_MAX_LOGS ]; then
                # Sort all logs by modification time (newest first)
                local sorted_logs=($(ls -t "$TESTING_ARTIFACTS_DIR/${phase_prefix}-"*.log* 2>/dev/null || true))
                local to_process=("${sorted_logs[@]:$TESTING_ARTIFACTS_MAX_LOGS}")
                
                for old_log in "${to_process[@]}"; do
                    if [[ "$old_log" == *.log.gz ]]; then
                        # Already compressed, just delete
                        rm -f "$old_log"
                    elif [ "$TESTING_ARTIFACTS_COMPRESS_OLD" = "true" ] && [ -f "$old_log" ]; then
                        # Compress old logs
                        gzip -9 "$old_log" 2>/dev/null || rm -f "$old_log"
                    else
                        # Delete old logs
                        rm -f "$old_log"
                    fi
                done
            fi
        done
    fi
}

# Rotate logs for a specific phase
# Usage: testing::artifacts::rotate_phase_logs PHASE
testing::artifacts::rotate_phase_logs() {
    local phase="$1"
    
    if [ -z "$TESTING_ARTIFACTS_DIR" ]; then
        return 0
    fi
    
    testing::artifacts::cleanup "$phase"
}

# Get the log file path for a phase
# Usage: testing::artifacts::get_log_path PHASE
testing::artifacts::get_log_path() {
    local phase="$1"
    local timestamp=$(date +%s)
    
    if [ -z "$TESTING_ARTIFACTS_DIR" ]; then
        echo "/tmp/${phase}-${timestamp}.log"
    else
        echo "$TESTING_ARTIFACTS_DIR/${phase}-${timestamp}.log"
    fi
}

# Archive all current test artifacts
# Usage: testing::artifacts::archive [run-id]
testing::artifacts::archive() {
    local run_id="${1:-$(date +%Y%m%d-%H%M%S)}"
    
    if [ -z "$TESTING_ARTIFACTS_DIR" ] || [ ! -d "$TESTING_ARTIFACTS_DIR" ]; then
        return 0
    fi
    
    local archive_name="${TESTING_ARTIFACTS_DIR}/archive-${run_id}.tar.gz"
    
    # Create archive of current logs
    if ls "$TESTING_ARTIFACTS_DIR"/*.log >/dev/null 2>&1; then
        tar -czf "$archive_name" -C "$TESTING_ARTIFACTS_DIR" \
            --exclude="archive-*.tar.gz" \
            $(ls -1 *.log 2>/dev/null | head -$TESTING_ARTIFACTS_MAX_LOGS) 2>/dev/null || true
        
        echo "Artifacts archived to: $archive_name"
    fi
}

# Generate artifact summary report
# Usage: testing::artifacts::summary
testing::artifacts::summary() {
    if [ -z "$TESTING_ARTIFACTS_DIR" ] || [ ! -d "$TESTING_ARTIFACTS_DIR" ]; then
        echo "No artifact directory configured"
        return 0
    fi
    
    echo "📊 Test Artifacts Summary"
    echo "════════════════════════════════════════"
    echo "Directory: $TESTING_ARTIFACTS_DIR"
    echo "Retention: $TESTING_ARTIFACTS_RETENTION_DAYS days"
    echo "Max logs per phase: $TESTING_ARTIFACTS_MAX_LOGS"
    echo "Compression: $TESTING_ARTIFACTS_COMPRESS_OLD"
    echo ""
    
    for phase in structure dependencies unit integration business performance; do
        local count=$(ls -1 "$TESTING_ARTIFACTS_DIR/${phase}-"*.log 2>/dev/null | wc -l || echo 0)
        local compressed=$(ls -1 "$TESTING_ARTIFACTS_DIR/${phase}-"*.log.gz 2>/dev/null | wc -l || echo 0)
        
        if [ $count -gt 0 ] || [ $compressed -gt 0 ]; then
            echo "  $phase: $count active, $compressed compressed"
        fi
    done
    
    local total_size=$(du -sh "$TESTING_ARTIFACTS_DIR" 2>/dev/null | cut -f1 || echo "0")
    echo ""
    echo "Total size: $total_size"
}