#!/usr/bin/env bash
# Enhanced Claude Code Session Management and Result Extraction
# Provides advanced session tracking, analytics, result extraction, and recovery

# Source trash module for safe cleanup
# Note: CLAUDE_CODE_SCRIPT_DIR should be set by manage.sh before sourcing this file
# Set CLAUDE_CODE_SCRIPT_DIR if not already set (for BATS test compatibility)
CLAUDE_CODE_SCRIPT_DIR="${CLAUDE_CODE_SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"

# shellcheck disable=SC1091
source "${CLAUDE_CODE_SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Session metadata directory for enhanced tracking
CLAUDE_SESSION_METADATA_DIR="${CLAUDE_SESSIONS_DIR}/.metadata"

# Session analytics cache
CLAUDE_SESSION_ANALYTICS_CACHE="${CLAUDE_SESSION_METADATA_DIR}/.analytics_cache"

#######################################
# Initialize enhanced session management
# Creates necessary directories and data structures
#######################################
claude_code::session_enhanced_init() {
    # Create metadata directory if it doesn't exist
    mkdir -p "$CLAUDE_SESSION_METADATA_DIR"
    
    # Initialize analytics cache if it doesn't exist
    if [[ ! -f "$CLAUDE_SESSION_ANALYTICS_CACHE" ]]; then
        echo "[]" > "$CLAUDE_SESSION_ANALYTICS_CACHE"
    fi
    
    return 0
}

#######################################
# Create session metadata entry
# Arguments:
#   $1 - Session ID
#   $2 - Session start time (ISO 8601)
#   $3 - Initial prompt (optional)
#   $4 - Max turns (optional)
#   $5 - Allowed tools (optional)
#######################################
claude_code::session_create_metadata() {
    local session_id="$1"
    local start_time="$2"
    local initial_prompt="${3:-}"
    local max_turns="${4:-}"
    local allowed_tools="${5:-}"
    
    claude_code::session_enhanced_init
    
    local metadata_file="$CLAUDE_SESSION_METADATA_DIR/${session_id}.json"
    
    # Create metadata structure
    local metadata
    metadata=$(jq -n \
        --arg session_id "$session_id" \
        --arg start_time "$start_time" \
        --arg initial_prompt "$initial_prompt" \
        --arg max_turns "$max_turns" \
        --arg allowed_tools "$allowed_tools" \
        --arg status "active" \
        --arg created_by "$(whoami)" \
        --arg host "$(hostname)" \
        --arg cwd "$(pwd)" \
        '{
            session_id: $session_id,
            start_time: $start_time,
            end_time: null,
            status: $status,
            initial_prompt: $initial_prompt,
            max_turns: ($max_turns | tonumber // null),
            allowed_tools: $allowed_tools,
            created_by: $created_by,
            host: $host,
            working_directory: $cwd,
            turn_count: 0,
            files_modified: [],
            errors: [],
            result_summary: null,
            tags: []
        }')
    
    echo "$metadata" > "$metadata_file"
    
    log::info "📝 Created session metadata: $session_id"
    return 0
}

#######################################
# Update session metadata
# Arguments:
#   $1 - Session ID
#   $2 - Update type (turn_count|status|end_time|files|error|summary|tag)
#   $3 - Update value
#######################################
claude_code::session_update_metadata() {
    local session_id="$1"
    local update_type="$2"
    local update_value="$3"
    
    local metadata_file="$CLAUDE_SESSION_METADATA_DIR/${session_id}.json"
    
    if [[ ! -f "$metadata_file" ]]; then
        log::warn "Session metadata not found: $session_id"
        return 1
    fi
    
    local updated_metadata
    case "$update_type" in
        "turn_count")
            updated_metadata=$(jq --arg turns "$update_value" '.turn_count = ($turns | tonumber)' "$metadata_file")
            ;;
        "status")
            updated_metadata=$(jq --arg status "$update_value" '.status = $status' "$metadata_file")
            ;;
        "end_time")
            updated_metadata=$(jq --arg end_time "$update_value" '.end_time = $end_time' "$metadata_file")
            ;;
        "files")
            updated_metadata=$(jq --arg file "$update_value" '.files_modified += [$file] | .files_modified |= unique' "$metadata_file")
            ;;
        "error")
            updated_metadata=$(jq --arg error "$update_value" '.errors += [$error]' "$metadata_file")
            ;;
        "summary")
            updated_metadata=$(jq --arg summary "$update_value" '.result_summary = $summary' "$metadata_file")
            ;;
        "tag")
            updated_metadata=$(jq --arg tag "$update_value" '.tags += [$tag] | .tags |= unique' "$metadata_file")
            ;;
        *)
            log::error "Unknown update type: $update_type"
            return 1
            ;;
    esac
    
    echo "$updated_metadata" > "$metadata_file"
    return 0
}

#######################################
# Extract comprehensive results from session output
# Arguments:
#   $1 - Session output file path
#   $2 - Session ID (optional)
#   $3 - Output format (json|automation|detailed)
# Outputs: Extracted session results
#######################################
claude_code::session_extract_results() {
    local output_file="$1"
    local session_id="${2:-unknown}"
    local format="${3:-json}"
    
    if [[ ! -f "$output_file" ]]; then
        log::error "Output file not found: $output_file"
        return 1
    fi
    
    log::info "📊 Extracting results from session: $session_id"
    
    # Initialize extraction variables
    local status="unknown"
    local turns_used=0
    local files_modified=()
    local errors=()
    local summary=""
    local tool_usage=()
    local performance_metrics=()
    
    # Parse stream-json output for comprehensive extraction
    local line_count=0
    while IFS= read -r line; do
        line_count=$((line_count + 1))
        
        # Skip empty lines
        [[ -z "$line" ]] && continue
        
        # Try to parse as JSON
        if echo "$line" | jq . >/dev/null 2>&1; then
            local line_type
            line_type=$(echo "$line" | jq -r '.type // "unknown"')
            
            case "$line_type" in
                "session_start")
                    session_id=$(echo "$line" | jq -r '.session_id // "unknown"')
                    ;;
                "turn_start")
                    turns_used=$((turns_used + 1))
                    ;;
                "tool_use")
                    local tool_name
                    tool_name=$(echo "$line" | jq -r '.tool_name // "unknown"')
                    tool_usage+=("$tool_name")
                    ;;
                "file_modified")
                    local file_path
                    file_path=$(echo "$line" | jq -r '.file_path // ""')
                    if [[ -n "$file_path" ]]; then
                        files_modified+=("$file_path")
                    fi
                    ;;
                "error")
                    local error_msg
                    error_msg=$(echo "$line" | jq -r '.message // "unknown error"')
                    errors+=("$error_msg")
                    ;;
                "session_complete")
                    status="completed"
                    summary=$(echo "$line" | jq -r '.summary // ""')
                    ;;
                "session_error")
                    status="error"
                    local error_detail
                    error_detail=$(echo "$line" | jq -r '.error // "unknown error"')
                    errors+=("$error_detail")
                    ;;
            esac
        else
            # Handle non-JSON lines - could be error messages or plain text
            if [[ "$line" == *"error"* ]] || [[ "$line" == *"Error"* ]] || [[ "$line" == *"ERROR"* ]]; then
                errors+=("$line")
            fi
        fi
    done < "$output_file"
    
    # Deduplicate arrays
    readarray -t files_modified < <(printf '%s\n' "${files_modified[@]}" | sort -u)
    readarray -t tool_usage < <(printf '%s\n' "${tool_usage[@]}" | sort -u)
    
    # Determine final status if still unknown
    if [[ "$status" == "unknown" ]]; then
        if [[ ${#errors[@]} -gt 0 ]]; then
            status="error"
        elif [[ $turns_used -gt 0 ]]; then
            status="completed"
        else
            status="incomplete"
        fi
    fi
    
    # Generate performance metrics
    local file_size
    file_size=$(wc -c < "$output_file" 2>/dev/null || echo "0")
    local processing_time=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
    
    # Build result structure based on format
    case "$format" in
        "json")
            jq -n \
                --arg session_id "$session_id" \
                --arg status "$status" \
                --argjson turns_used "$turns_used" \
                --argjson files_modified "$(printf '%s\n' "${files_modified[@]}" | jq -R . | jq -s '.')" \
                --argjson errors "$(printf '%s\n' "${errors[@]}" | jq -R . | jq -s '.')" \
                --arg summary "$summary" \
                --argjson tool_usage "$(printf '%s\n' "${tool_usage[@]}" | jq -R . | jq -s '.' | jq 'group_by(.) | map({tool: .[0], count: length})')" \
                --argjson file_size "$file_size" \
                --argjson line_count "$line_count" \
                --arg extracted_at "$processing_time" \
                '{
                    session_id: $session_id,
                    status: $status,
                    turns_used: $turns_used,
                    files_modified: $files_modified,
                    errors: $errors,
                    summary: $summary,
                    tool_usage: $tool_usage,
                    performance: {
                        output_file_size: $file_size,
                        output_line_count: $line_count
                    },
                    extracted_at: $extracted_at
                }'
            ;;
        "automation")
            echo "status=$status"
            echo "turns_used=$turns_used" 
            echo "files_modified=$(IFS=','; echo "${files_modified[*]}")"
            echo "error_count=${#errors[@]}"
            echo "tool_count=${#tool_usage[@]}"
            echo "session_id=$session_id"
            ;;
        "detailed")
            echo "=== Session Results ==="
            echo "Session ID: $session_id"
            echo "Status: $status"
            echo "Turns Used: $turns_used"
            echo "Files Modified: ${#files_modified[@]}"
            if [[ ${#files_modified[@]} -gt 0 ]]; then
                printf '  - %s\n' "${files_modified[@]}"
            fi
            echo "Errors: ${#errors[@]}"
            if [[ ${#errors[@]} -gt 0 ]]; then
                printf '  - %s\n' "${errors[@]}"
            fi
            echo "Tools Used: ${#tool_usage[@]}"
            if [[ ${#tool_usage[@]} -gt 0 ]]; then
                printf '  - %s\n' "${tool_usage[@]}"
            fi
            if [[ -n "$summary" ]]; then
                echo "Summary: $summary"
            fi
            echo "Output Size: ${file_size} bytes"
            echo "Extracted: $processing_time"
            ;;
        *)
            log::error "Unknown format: $format"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Generate session analytics report
# Arguments:
#   $1 - Analysis period (day|week|month|all)
#   $2 - Output format (json|text|csv)
# Outputs: Session analytics report
#######################################
claude_code::session_analytics() {
    local period="${1:-week}"
    local format="${2:-text}"
    
    claude_code::session_enhanced_init
    
    log::header "📈 Session Analytics ($period)"
    
    # Calculate time range
    local start_time
    case "$period" in
        "day")
            start_time=$(date -d '1 day ago' -u '+%Y-%m-%dT%H:%M:%SZ')
            ;;
        "week")
            start_time=$(date -d '1 week ago' -u '+%Y-%m-%dT%H:%M:%SZ')
            ;;
        "month")
            start_time=$(date -d '1 month ago' -u '+%Y-%m-%dT%H:%M:%SZ')
            ;;
        "all")
            start_time="1970-01-01T00:00:00Z"
            ;;
        *)
            log::error "Invalid period: $period (use day|week|month|all)"
            return 1
            ;;
    esac
    
    # Collect session data
    local total_sessions=0
    local completed_sessions=0
    local error_sessions=0
    local total_turns=0
    local total_files_modified=0
    local most_used_tools=()
    local session_durations=()
    
    # Process all metadata files
    for metadata_file in "$CLAUDE_SESSION_METADATA_DIR"/*.json; do
        [[ ! -f "$metadata_file" ]] && continue
        
        local session_start
        session_start=$(jq -r '.start_time' "$metadata_file" 2>/dev/null || echo "")
        
        # Skip if outside time range
        if [[ -n "$session_start" ]] && [[ "$session_start" > "$start_time" ]]; then
            total_sessions=$((total_sessions + 1))
            
            local status
            status=$(jq -r '.status' "$metadata_file" 2>/dev/null || echo "unknown")
            case "$status" in
                "completed") completed_sessions=$((completed_sessions + 1)) ;;
                "error") error_sessions=$((error_sessions + 1)) ;;
            esac
            
            local turns
            turns=$(jq -r '.turn_count // 0' "$metadata_file" 2>/dev/null || echo "0")
            total_turns=$((total_turns + turns))
            
            local files_count
            files_count=$(jq -r '.files_modified | length' "$metadata_file" 2>/dev/null || echo "0")
            total_files_modified=$((total_files_modified + files_count))
            
            # Collect tool usage
            local tools
            tools=$(jq -r '.allowed_tools // ""' "$metadata_file" 2>/dev/null || echo "")
            if [[ -n "$tools" ]]; then
                IFS=',' read -ra tool_array <<< "$tools"
                most_used_tools+=("${tool_array[@]}")
            fi
            
            # Calculate duration if both start and end times exist
            local end_time
            end_time=$(jq -r '.end_time // null' "$metadata_file" 2>/dev/null || echo "null")
            if [[ "$end_time" != "null" ]] && [[ -n "$end_time" ]]; then
                local duration
                duration=$(( $(date -d "$end_time" +%s) - $(date -d "$session_start" +%s) ))
                session_durations+=("$duration")
            fi
        fi
    done
    
    # Calculate statistics
    local success_rate=0
    if [[ $total_sessions -gt 0 ]]; then
        success_rate=$(( completed_sessions * 100 / total_sessions ))
    fi
    
    local avg_turns=0
    if [[ $total_sessions -gt 0 ]]; then
        avg_turns=$(( total_turns / total_sessions ))
    fi
    
    local avg_files=0
    if [[ $total_sessions -gt 0 ]]; then
        avg_files=$(( total_files_modified / total_sessions ))
    fi
    
    # Calculate average duration
    local avg_duration=0
    if [[ ${#session_durations[@]} -gt 0 ]]; then
        local sum_duration=0
        for duration in "${session_durations[@]}"; do
            sum_duration=$((sum_duration + duration))
        done
        avg_duration=$(( sum_duration / ${#session_durations[@]} ))
    fi
    
    # Output analytics based on format
    case "$format" in
        "json")
            jq -n \
                --arg period "$period" \
                --argjson total_sessions "$total_sessions" \
                --argjson completed_sessions "$completed_sessions" \
                --argjson error_sessions "$error_sessions" \
                --argjson success_rate "$success_rate" \
                --argjson total_turns "$total_turns" \
                --argjson avg_turns "$avg_turns" \
                --argjson total_files_modified "$total_files_modified" \
                --argjson avg_files "$avg_files" \
                --argjson avg_duration "$avg_duration" \
                --arg generated_at "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
                '{
                    period: $period,
                    total_sessions: $total_sessions,
                    completed_sessions: $completed_sessions,
                    error_sessions: $error_sessions,
                    success_rate_percent: $success_rate,
                    total_turns: $total_turns,
                    avg_turns_per_session: $avg_turns,
                    total_files_modified: $total_files_modified,
                    avg_files_per_session: $avg_files,
                    avg_duration_seconds: $avg_duration,
                    generated_at: $generated_at
                }'
            ;;
        "csv")
            echo "metric,value"
            echo "period,$period"
            echo "total_sessions,$total_sessions"
            echo "completed_sessions,$completed_sessions"
            echo "error_sessions,$error_sessions"
            echo "success_rate_percent,$success_rate"
            echo "total_turns,$total_turns"
            echo "avg_turns_per_session,$avg_turns"
            echo "total_files_modified,$total_files_modified"
            echo "avg_files_per_session,$avg_files"
            echo "avg_duration_seconds,$avg_duration"
            ;;
        "text")
            echo "📊 Claude Code Session Analytics"
            echo "Period: $period"
            echo
            echo "Session Summary:"
            echo "  Total Sessions: $total_sessions"
            echo "  Completed: $completed_sessions"
            echo "  Errors: $error_sessions"
            echo "  Success Rate: ${success_rate}%"
            echo
            echo "Activity Metrics:"
            echo "  Total Turns: $total_turns"
            echo "  Avg Turns/Session: $avg_turns"
            echo "  Total Files Modified: $total_files_modified"
            echo "  Avg Files/Session: $avg_files"
            if [[ $avg_duration -gt 0 ]]; then
                local duration_formatted
                duration_formatted=$(printf "%02d:%02d:%02d" $((avg_duration/3600)) $((avg_duration%3600/60)) $((avg_duration%60)))
                echo "  Avg Session Duration: $duration_formatted"
            fi
            ;;
        *)
            log::error "Unknown format: $format"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Advanced session recovery with automatic retry
# Arguments:
#   $1 - Session ID
#   $2 - Recovery strategy (auto|clean|force)
#   $3 - Additional prompt for continuation (optional)
# Returns: 0 on success, 1 on failure
#######################################
claude_code::session_recover() {
    local session_id="$1"
    local strategy="${2:-auto}"
    local continuation_prompt="${3:-}"
    
    log::header "🔄 Advanced Session Recovery: $session_id"
    
    # Check if session exists
    local session_file="$CLAUDE_SESSIONS_DIR/$session_id"
    local metadata_file="$CLAUDE_SESSION_METADATA_DIR/${session_id}.json"
    
    if [[ ! -f "$session_file" ]]; then
        log::error "Session file not found: $session_file"
        return 1
    fi
    
    # Load session metadata if available
    local session_status="unknown"
    local last_error=""
    local turn_count=0
    
    if [[ -f "$metadata_file" ]]; then
        session_status=$(jq -r '.status // "unknown"' "$metadata_file" 2>/dev/null || echo "unknown")
        last_error=$(jq -r '.errors[-1] // ""' "$metadata_file" 2>/dev/null || echo "")
        turn_count=$(jq -r '.turn_count // 0' "$metadata_file" 2>/dev/null || echo "0")
    fi
    
    log::info "Session status: $session_status"
    log::info "Turn count: $turn_count"
    if [[ -n "$last_error" ]]; then
        log::warn "Last error: $last_error"
    fi
    
    # Apply recovery strategy
    case "$strategy" in
        "auto")
            log::info "🤖 Applying automatic recovery strategy"
            
            # Determine best recovery approach based on session state
            if [[ "$session_status" == "error" ]]; then
                log::info "Error detected, attempting clean recovery"
                strategy="clean"
            elif [[ $turn_count -gt 0 ]]; then
                log::info "Active session detected, attempting continuation"
                strategy="continue"
            else
                log::info "New session detected, attempting force recovery"
                strategy="force"
            fi
            ;;
        "clean")
            log::info "🧹 Applying clean recovery strategy"
            # Clear error state and retry
            if [[ -f "$metadata_file" ]]; then
                local cleaned_metadata
                cleaned_metadata=$(jq '.status = "recovering" | .errors = []' "$metadata_file")
                echo "$cleaned_metadata" > "$metadata_file"
            fi
            ;;
        "force")
            log::info "💪 Applying force recovery strategy"
            # Force continuation regardless of state
            ;;
        "continue")
            log::info "▶️  Applying continuation strategy"
            ;;
        *)
            log::error "Unknown recovery strategy: $strategy"
            return 1
            ;;
    esac
    
    # Prepare recovery command
    local recovery_cmd="claude --resume \"$session_id\""
    
    # Add continuation prompt if provided
    if [[ -n "$continuation_prompt" ]]; then
        recovery_cmd="$recovery_cmd \"$continuation_prompt\""
    fi
    
    # Add timeout and error handling
    recovery_cmd="timeout ${DEFAULT_TIMEOUT:-300} $recovery_cmd"
    
    log::info "Executing recovery: $recovery_cmd"
    
    # Execute recovery with error handling
    local recovery_output
    local recovery_exit_code
    
    if recovery_output=$(eval "$recovery_cmd" 2>&1); then
        recovery_exit_code=0
        log::success "✅ Session recovery successful"
        
        # Update metadata on success
        if [[ -f "$metadata_file" ]]; then
            claude_code::session_update_metadata "$session_id" "status" "recovered"
            claude_code::session_update_metadata "$session_id" "end_time" "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
        fi
        
        echo "$recovery_output"
    else
        recovery_exit_code=$?
        log::error "❌ Session recovery failed with exit code: $recovery_exit_code"
        
        # Update metadata on failure
        if [[ -f "$metadata_file" ]]; then
            claude_code::session_update_metadata "$session_id" "status" "recovery_failed"
            claude_code::session_update_metadata "$session_id" "error" "Recovery failed: $recovery_output"
        fi
        
        # Show recovery output for debugging
        echo "Recovery output:"
        echo "$recovery_output"
        
        return 1
    fi
    
    return 0
}

#######################################
# Session cleanup and maintenance
# Arguments:
#   $1 - Cleanup strategy (old|failed|all|size)
#   $2 - Age threshold in days (default 30) or size limit in MB
# Returns: Number of sessions cleaned up
#######################################
claude_code::session_cleanup() {
    local strategy="${1:-old}"
    local threshold="${2:-30}"
    
    log::header "🧹 Session Cleanup: $strategy"
    
    claude_code::session_enhanced_init
    
    local sessions_cleaned=0
    local space_freed=0
    
    case "$strategy" in
        "old")
            log::info "Cleaning sessions older than $threshold days"
            
            # Find sessions older than threshold
            local cutoff_date
            cutoff_date=$(date -d "${threshold} days ago" -u '+%Y-%m-%dT%H:%M:%SZ')
            
            for metadata_file in "$CLAUDE_SESSION_METADATA_DIR"/*.json; do
                [[ ! -f "$metadata_file" ]] && continue
                
                local session_start
                session_start=$(jq -r '.start_time // "1970-01-01T00:00:00Z"' "$metadata_file" 2>/dev/null)
                
                if [[ "$session_start" < "$cutoff_date" ]]; then
                    local session_id
                    session_id=$(basename "$metadata_file" .json)
                    
                    local session_file="$CLAUDE_SESSIONS_DIR/$session_id"
                    local file_size=0
                    
                    if [[ -f "$session_file" ]]; then
                        file_size=$(wc -c < "$session_file" 2>/dev/null || echo "0")
                        trash::safe_remove "$session_file" --temp
                    fi
                    
                    trash::safe_remove "$metadata_file" --temp
                    
                    sessions_cleaned=$((sessions_cleaned + 1))
                    space_freed=$((space_freed + file_size))
                    
                    log::info "Cleaned session: $session_id (${file_size} bytes)"
                fi
            done
            ;;
        "failed")
            log::info "Cleaning failed sessions"
            
            for metadata_file in "$CLAUDE_SESSION_METADATA_DIR"/*.json; do
                [[ ! -f "$metadata_file" ]] && continue
                
                local status
                status=$(jq -r '.status // "unknown"' "$metadata_file" 2>/dev/null)
                
                if [[ "$status" == "error" ]] || [[ "$status" == "recovery_failed" ]]; then
                    local session_id
                    session_id=$(basename "$metadata_file" .json)
                    
                    local session_file="$CLAUDE_SESSIONS_DIR/$session_id"
                    local file_size=0
                    
                    if [[ -f "$session_file" ]]; then
                        file_size=$(wc -c < "$session_file" 2>/dev/null || echo "0")
                        trash::safe_remove "$session_file" --temp
                    fi
                    
                    trash::safe_remove "$metadata_file" --temp
                    
                    sessions_cleaned=$((sessions_cleaned + 1))
                    space_freed=$((space_freed + file_size))
                    
                    log::info "Cleaned failed session: $session_id"
                fi
            done
            ;;
        "size")
            log::info "Cleaning to stay under $threshold MB total size"
            
            local target_size=$((threshold * 1024 * 1024))  # Convert MB to bytes
            local current_size=0
            
            # Calculate current total size
            for session_file in "$CLAUDE_SESSIONS_DIR"/*; do
                [[ ! -f "$session_file" ]] && continue
                local file_size
                file_size=$(wc -c < "$session_file" 2>/dev/null || echo "0")
                current_size=$((current_size + file_size))
            done
            
            log::info "Current total size: $((current_size / 1024 / 1024)) MB"
            
            if [[ $current_size -gt $target_size ]]; then
                # Remove oldest sessions until under limit
                while IFS= read -r -d '' session_file; do
                    [[ $current_size -le $target_size ]] && break
                    
                    local session_id
                    session_id=$(basename "$session_file")
                    local metadata_file="$CLAUDE_SESSION_METADATA_DIR/${session_id}.json"
                    
                    local file_size
                    file_size=$(wc -c < "$session_file" 2>/dev/null || echo "0")
                    
                    trash::safe_remove "$session_file" --temp
                    [[ -f "$metadata_file" ]] && trash::safe_remove "$metadata_file" --temp
                    
                    current_size=$((current_size - file_size))
                    sessions_cleaned=$((sessions_cleaned + 1))
                    space_freed=$((space_freed + file_size))
                    
                    log::info "Cleaned session: $session_id (${file_size} bytes)"
                done < <(find "$CLAUDE_SESSIONS_DIR" -type f -printf '%T+ %p\0' | sort -z | cut -z -d' ' -f2-)
            fi
            ;;
        "all")
            if confirm "⚠️  Delete ALL sessions? This cannot be undone!"; then
                log::info "Cleaning all sessions"
                
                for session_file in "$CLAUDE_SESSIONS_DIR"/*; do
                    [[ ! -f "$session_file" ]] && continue
                    local file_size
                    file_size=$(wc -c < "$session_file" 2>/dev/null || echo "0")
                    trash::safe_remove "$session_file" --temp
                    space_freed=$((space_freed + file_size))
                    sessions_cleaned=$((sessions_cleaned + 1))
                done
                
                for metadata_file in "$CLAUDE_SESSION_METADATA_DIR"/*.json; do
                    [[ ! -f "$metadata_file" ]] && continue
                    trash::safe_remove "$metadata_file" --temp
                done
            else
                log::info "Cleanup cancelled"
                return 0
            fi
            ;;
        *)
            log::error "Unknown cleanup strategy: $strategy"
            return 1
            ;;
    esac
    
    # Summary
    log::success "✅ Cleanup completed"
    log::info "Sessions cleaned: $sessions_cleaned"
    log::info "Space freed: $((space_freed / 1024 / 1024)) MB"
    
    return "$sessions_cleaned"
}

#######################################
# List sessions with enhanced filtering and sorting
# Arguments:
#   $1 - Filter (all|active|completed|error|recent)
#   $2 - Sort by (date|turns|files|duration)
#   $3 - Output format (text|json|table)
#   $4 - Limit (number of sessions to show)
# Outputs: Filtered and sorted session list
#######################################
claude_code::session_list_enhanced() {
    local filter="${1:-recent}"
    local sort_by="${2:-date}"
    local format="${3:-text}"
    local limit="${4:-20}"
    
    claude_code::session_enhanced_init
    
    log::header "📋 Enhanced Session List"
    
    # Collect session data
    local sessions=()
    
    # Check if metadata directory exists and has JSON files
    if [[ ! -d "$CLAUDE_SESSION_METADATA_DIR" ]]; then
        log::info "No sessions found. Run a Claude session first."
        return 0
    fi
    
    # Check if there are any JSON files
    local has_json_files=false
    for check_file in "$CLAUDE_SESSION_METADATA_DIR"/*.json; do
        if [[ -f "$check_file" ]]; then
            has_json_files=true
            break
        fi
    done
    
    if [[ "$has_json_files" == "false" ]]; then
        log::info "No session metadata found. Run a Claude session first."
        return 0
    fi
    
    for metadata_file in "$CLAUDE_SESSION_METADATA_DIR"/*.json; do
        [[ ! -f "$metadata_file" ]] && continue
        
        local session_data
        session_data=$(jq -c '{
            session_id: .session_id,
            status: .status,
            start_time: .start_time,
            end_time: .end_time,
            turn_count: .turn_count,
            files_count: (.files_modified | length),
            duration: (if .end_time and .start_time then 
                (((.end_time | fromdateiso8601) - (.start_time | fromdateiso8601)) | tostring) 
                else null end),
            initial_prompt: ((.initial_prompt // "") | tostring | if length > 50 then .[0:50] + "..." else . end)
        }' "$metadata_file" 2>/dev/null)
        
        if [[ -n "$session_data" ]]; then
            sessions+=("$session_data")
        fi
    done
    
    # Apply filters
    local filtered_sessions=()
    for session in "${sessions[@]}"; do
        local status
        status=$(echo "$session" | jq -r '.status')
        
        case "$filter" in
            "all")
                filtered_sessions+=("$session")
                ;;
            "active")
                [[ "$status" == "active" ]] && filtered_sessions+=("$session")
                ;;
            "completed")
                [[ "$status" == "completed" ]] && filtered_sessions+=("$session")
                ;;
            "error")
                [[ "$status" == "error" ]] && filtered_sessions+=("$session")
                ;;
            "recent")
                # Include active, completed, and recently modified sessions
                if [[ "$status" == "active" ]] || [[ "$status" == "completed" ]] || [[ "$status" == "recovered" ]]; then
                    filtered_sessions+=("$session")
                fi
                ;;
        esac
    done
    
    # Sort sessions
    local sorted_sessions=()
    case "$sort_by" in
        "date")
            readarray -t sorted_sessions < <(printf '%s\n' "${filtered_sessions[@]}" | jq -s 'sort_by(.start_time) | reverse')
            ;;
        "turns")
            readarray -t sorted_sessions < <(printf '%s\n' "${filtered_sessions[@]}" | jq -s 'sort_by(.turn_count) | reverse')
            ;;
        "files")
            readarray -t sorted_sessions < <(printf '%s\n' "${filtered_sessions[@]}" | jq -s 'sort_by(.files_count) | reverse')
            ;;
        "duration")
            readarray -t sorted_sessions < <(printf '%s\n' "${filtered_sessions[@]}" | jq -s 'sort_by(.duration) | reverse')
            ;;
        *)
            sorted_sessions=("${filtered_sessions[@]}")
            ;;
    esac
    
    # Apply limit
    local final_sessions=("${sorted_sessions[@]:0:$limit}")
    
    # Output based on format
    case "$format" in
        "json")
            printf '%s\n' "${final_sessions[@]}" | jq -s '.'
            ;;
        "table")
            echo "ID                     Status      Turns  Files  Start Time           Duration"
            echo "─────────────────────  ──────────  ─────  ─────  ───────────────────  ────────"
            for session in "${final_sessions[@]}"; do
                local session_id status turns files start_time duration
                session_id=$(echo "$session" | jq -r '.session_id[0:20]')
                status=$(echo "$session" | jq -r '.status')
                turns=$(echo "$session" | jq -r '.turn_count // 0')
                files=$(echo "$session" | jq -r '.files_count // 0')
                start_time=$(echo "$session" | jq -r '.start_time[0:19]')
                duration=$(echo "$session" | jq -r '.duration // "N/A"')
                
                printf "%-21s  %-10s  %5s  %5s  %-19s  %s\n" \
                    "$session_id" "$status" "$turns" "$files" "$start_time" "$duration"
            done
            ;;
        "text")
            for session in "${final_sessions[@]}"; do
                echo "$session" | jq -r '"Session: \(.session_id)
  Status: \(.status)
  Start: \(.start_time)
  Turns: \(.turn_count // 0)
  Files: \(.files_count // 0)
  Prompt: \(.initial_prompt)
"'
            done
            ;;
        *)
            log::error "Unknown format: $format"
            return 1
            ;;
    esac
    
    log::info "Showing ${#final_sessions[@]} sessions (filter: $filter, sort: $sort_by)"
    return 0
}

# Export enhanced session management functions
export -f claude_code::session_enhanced_init
export -f claude_code::session_create_metadata
export -f claude_code::session_update_metadata
export -f claude_code::session_extract_results
export -f claude_code::session_analytics
export -f claude_code::session_recover
export -f claude_code::session_cleanup
export -f claude_code::session_list_enhanced