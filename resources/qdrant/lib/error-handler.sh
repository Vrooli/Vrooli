#!/usr/bin/env bash
################################################################################
# Comprehensive Error Handler for Qdrant Operations
# 
# Provides standardized error handling, logging, and recovery mechanisms
# for all Qdrant operations including embeddings, search, and management
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source logging utilities  
source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
source "${var_LIB_UTILS_DIR}/log.sh" 2>/dev/null || echo() { printf "%s\n" "$*"; }

# Error handling configuration
ERROR_LOG_FILE="${QDRANT_ERROR_LOG:-$HOME/.qdrant/errors.log}"
ERROR_METRICS_FILE="$HOME/.qdrant/error-metrics.json"
DEAD_LETTER_QUEUE_DIR="$HOME/.qdrant/failed-operations"
MAX_RETRY_ATTEMPTS="${MAX_RETRY_ATTEMPTS:-3}"
RETRY_BACKOFF_BASE="${RETRY_BACKOFF_BASE:-2}"

# Create required directories
mkdir -p "$(dirname "$ERROR_LOG_FILE")"
mkdir -p "$DEAD_LETTER_QUEUE_DIR"

# Global error state
declare -A ERROR_COUNTERS
declare -A ERROR_LAST_SEEN

#######################################
# Initialize error handling system
#######################################
error_handler::init() {
    # Load existing error metrics
    if [[ -f "$ERROR_METRICS_FILE" ]]; then
        local metrics
        metrics=$(cat "$ERROR_METRICS_FILE" 2>/dev/null || echo '{}')
        
        # Parse counters (simplified for bash)
        local embedding_errors
        embedding_errors=$(echo "$metrics" | jq -r '.embedding_errors // 0' 2>/dev/null || echo "0")
        ERROR_COUNTERS["embedding_errors"]=$embedding_errors
        
        local search_errors  
        search_errors=$(echo "$metrics" | jq -r '.search_errors // 0' 2>/dev/null || echo "0")
        ERROR_COUNTERS["search_errors"]=$search_errors
        
        local api_errors
        api_errors=$(echo "$metrics" | jq -r '.api_errors // 0' 2>/dev/null || echo "0")
        ERROR_COUNTERS["api_errors"]=$api_errors
    fi
    
    log::debug "Error handler initialized"
}

#######################################
# Log error with context and classification
# Arguments:
#   $1 - Error type (embedding|search|api|collection|model)
#   $2 - Error message
#   $3 - Operation context (optional)
#   $4 - Error data (JSON, optional)
#######################################
error_handler::log_error() {
    local error_type="$1"
    local error_message="$2"
    local context="${3:-unknown}"
    local error_data="${4:-{}}"
    local timestamp=$(date -Iseconds)
    
    # Increment error counter
    ERROR_COUNTERS["${error_type}_errors"]=$((${ERROR_COUNTERS["${error_type}_errors"]:-0} + 1))
    ERROR_LAST_SEEN["$error_type"]="$timestamp"
    
    # Create structured error log entry
    local log_entry
    log_entry=$(jq -n \
        --arg timestamp "$timestamp" \
        --arg type "$error_type" \
        --arg message "$error_message" \
        --arg context "$context" \
        --argjson data "$error_data" \
        '{
            timestamp: $timestamp,
            type: $type,
            message: $message,
            context: $context,
            data: $data,
            severity: (if $type == "api" then "high" else "medium" end)
        }')
    
    # Append to error log
    echo "$log_entry" >> "$ERROR_LOG_FILE"
    
    # Log to console based on severity
    case "$error_type" in
        api|collection)
            log::error "[${error_type^^}] $error_message (context: $context)"
            ;;
        embedding|search)
            log::warn "[${error_type^^}] $error_message (context: $context)"
            ;;
        model)
            log::warn "[${error_type^^}] $error_message (context: $context)"
            ;;
        *)
            log::error "[ERROR] $error_message (context: $context)"
            ;;
    esac
    
    # Save metrics
    error_handler::save_metrics
}

#######################################
# Save error metrics to file
#######################################
error_handler::save_metrics() {
    local metrics_json
    metrics_json=$(jq -n \
        --argjson embedding_errors "${ERROR_COUNTERS["embedding_errors"]:-0}" \
        --argjson search_errors "${ERROR_COUNTERS["search_errors"]:-0}" \
        --argjson api_errors "${ERROR_COUNTERS["api_errors"]:-0}" \
        --argjson collection_errors "${ERROR_COUNTERS["collection_errors"]:-0}" \
        --argjson model_errors "${ERROR_COUNTERS["model_errors"]:-0}" \
        --arg last_updated "$(date -Iseconds)" \
        '{
            embedding_errors: $embedding_errors,
            search_errors: $search_errors,
            api_errors: $api_errors,
            collection_errors: $collection_errors,
            model_errors: $model_errors,
            last_updated: $last_updated
        }')
    
    echo "$metrics_json" > "$ERROR_METRICS_FILE"
}

#######################################
# Execute operation with retry logic and error handling
# Arguments:
#   $1 - Operation type (for error classification)
#   $2 - Operation description
#   $3 - Command to execute
#   $4 - Expected success pattern (optional, default: any non-empty output)
# Returns: 0 on success, 1 on failure
#######################################
error_handler::execute_with_retry() {
    local op_type="$1"
    local description="$2"
    local command="$3"
    local success_pattern="${4:-.*}"
    
    local attempt=1
    local max_attempts=$((MAX_RETRY_ATTEMPTS + 1))
    local backoff_delay=1
    
    log::debug "Executing $op_type operation: $description"
    
    while [[ $attempt -le $max_attempts ]]; do
        local start_time=$(date +%s%3N)
        local result=""
        local exit_code=0
        
        # Execute command with timeout
        if result=$(timeout 60 eval "$command" 2>&1); then
            exit_code=0
        else
            exit_code=$?
        fi
        
        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))
        
        # Check success
        if [[ $exit_code -eq 0 && "$result" =~ $success_pattern ]]; then
            log::debug "$op_type operation succeeded in ${duration}ms (attempt $attempt)"
            echo "$result"
            return 0
        fi
        
        # Handle failure
        local error_data
        error_data=$(jq -n \
            --arg command "$command" \
            --argjson attempt "$attempt" \
            --argjson exit_code "$exit_code" \
            --argjson duration "$duration" \
            --arg result "$result" \
            '{
                command: $command,
                attempt: $attempt,
                exit_code: $exit_code,
                duration: $duration,
                result: $result
            }')
        
        if [[ $attempt -eq $max_attempts ]]; then
            # Final failure
            error_handler::log_error "$op_type" "Operation failed after $max_attempts attempts: $description" "retry_exhausted" "$error_data"
            
            # Save to dead letter queue
            error_handler::save_failed_operation "$op_type" "$description" "$command" "$error_data"
            
            return 1
        else
            # Retry with backoff
            log::debug "$op_type operation failed (attempt $attempt/$max_attempts), retrying in ${backoff_delay}s..."
            error_handler::log_error "$op_type" "Operation failed, retrying: $description" "retry_attempt_$attempt" "$error_data"
            
            sleep "$backoff_delay"
            backoff_delay=$((backoff_delay * RETRY_BACKOFF_BASE))
            ((attempt++))
        fi
    done
}

#######################################
# Save failed operation to dead letter queue
# Arguments:
#   $1 - Operation type
#   $2 - Description
#   $3 - Command
#   $4 - Error data
#######################################
error_handler::save_failed_operation() {
    local op_type="$1"
    local description="$2"  
    local command="$3"
    local error_data="$4"
    local timestamp=$(date +%s)
    
    local failed_op
    failed_op=$(jq -n \
        --arg timestamp "$(date -Iseconds)" \
        --arg type "$op_type" \
        --arg description "$description" \
        --arg command "$command" \
        --argjson error_data "$error_data" \
        '{
            timestamp: $timestamp,
            type: $type,
            description: $description,
            command: $command,
            error_data: $error_data,
            retry_count: 0
        }')
    
    local filename="${DEAD_LETTER_QUEUE_DIR}/${op_type}_${timestamp}.json"
    echo "$failed_op" > "$filename"
    
    log::debug "Saved failed operation to: $filename"
}

#######################################
# Retry failed operations from dead letter queue
# Arguments:
#   $1 - Operation type filter (optional)
#######################################
error_handler::retry_failed_operations() {
    local type_filter="${1:-}"
    local retried=0
    local succeeded=0
    
    log::info "🔄 Processing failed operations from dead letter queue..."
    
    # Find failed operation files
    local pattern="*.json"
    if [[ -n "$type_filter" ]]; then
        pattern="${type_filter}_*.json"
    fi
    
    for file in "$DEAD_LETTER_QUEUE_DIR"/$pattern; do
        [[ -f "$file" ]] || continue
        
        local op_data
        op_data=$(cat "$file" 2>/dev/null || continue)
        
        local op_type
        op_type=$(echo "$op_data" | jq -r '.type')
        local description
        description=$(echo "$op_data" | jq -r '.description')
        local command
        command=$(echo "$op_data" | jq -r '.command')
        local retry_count
        retry_count=$(echo "$op_data" | jq -r '.retry_count // 0')
        
        # Skip if too many retries
        if [[ $retry_count -ge 5 ]]; then
            log::debug "Skipping $file (max retries exceeded)"
            continue
        fi
        
        log::info "Retrying $op_type operation: $description (retry #$((retry_count + 1)))"
        ((retried++))
        
        # Attempt retry
        if error_handler::execute_with_retry "$op_type" "$description" "$command" >/dev/null 2>&1; then
            # Success - remove from queue
            rm -f "$file"
            ((succeeded++))
            log::success "✅ Retry succeeded: $description"
        else
            # Update retry count
            local updated_op
            updated_op=$(echo "$op_data" | jq --argjson count "$((retry_count + 1))" '.retry_count = $count')
            echo "$updated_op" > "$file"
            log::warn "❌ Retry failed: $description"
        fi
    done
    
    if [[ $retried -gt 0 ]]; then
        log::info "🎯 Processed $retried failed operations: $succeeded succeeded"
    else
        log::info "📭 No failed operations to retry"
    fi
}

#######################################
# Get error statistics
#######################################
error_handler::get_stats() {
    echo "=== Error Handler Statistics ==="
    echo
    
    # Load current metrics
    if [[ -f "$ERROR_METRICS_FILE" ]]; then
        local metrics
        metrics=$(cat "$ERROR_METRICS_FILE" 2>/dev/null || echo '{}')
        
        echo "Error Counts:"
        echo "  Embedding errors: $(echo "$metrics" | jq -r '.embedding_errors // 0')"
        echo "  Search errors: $(echo "$metrics" | jq -r '.search_errors // 0')"
        echo "  API errors: $(echo "$metrics" | jq -r '.api_errors // 0')"
        echo "  Collection errors: $(echo "$metrics" | jq -r '.collection_errors // 0')"
        echo "  Model errors: $(echo "$metrics" | jq -r '.model_errors // 0')"
        echo
        echo "Last updated: $(echo "$metrics" | jq -r '.last_updated // "Never"')"
    else
        echo "No error metrics available"
    fi
    
    echo
    
    # Count failed operations in queue
    local failed_count
    failed_count=$(find "$DEAD_LETTER_QUEUE_DIR" -name "*.json" -type f 2>/dev/null | wc -l)
    echo "Failed operations in queue: $failed_count"
    
    # Recent errors
    echo
    echo "Recent errors (last 10):"
    if [[ -f "$ERROR_LOG_FILE" ]]; then
        tail -n 10 "$ERROR_LOG_FILE" | jq -r '"\(.timestamp) [\(.type)] \(.message)"' 2>/dev/null || \
            tail -n 10 "$ERROR_LOG_FILE"
    else
        echo "  No error log available"
    fi
}

#######################################
# Clear old error logs and metrics
# Arguments:
#   $1 - Age in days (default: 7)
#######################################
error_handler::cleanup() {
    local age_days="${1:-7}"
    
    log::info "🧹 Cleaning up error logs older than $age_days days"
    
    # Clean old error log entries (keep recent ones)
    if [[ -f "$ERROR_LOG_FILE" ]]; then
        local temp_file=$(mktemp)
        local cutoff_date=$(date -d "$age_days days ago" -Iseconds)
        
        # Keep only recent entries
        jq -r "select(.timestamp >= \"$cutoff_date\")" "$ERROR_LOG_FILE" > "$temp_file" 2>/dev/null || true
        mv "$temp_file" "$ERROR_LOG_FILE"
    fi
    
    # Clean old failed operations
    find "$DEAD_LETTER_QUEUE_DIR" -name "*.json" -type f -mtime +$age_days -delete 2>/dev/null || true
    
    log::success "Cleanup completed"
}

# Initialize on source
error_handler::init