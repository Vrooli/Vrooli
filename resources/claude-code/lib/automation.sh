#!/usr/bin/env bash
# Claude Code Automation Helper Functions
# Provides automation-friendly interfaces for external integration

#######################################
# Parse Claude Code stream-json output into structured format
# Arguments:
#   $1 - Path to stream-json output file
#   $2 - Output format (automation|simple|json)
# Outputs: Structured result based on format
#######################################
claude_code::parse_result() {
    local output_file="$1"
    local format="${2:-automation}"
    
    if [[ ! -f "$output_file" || ! -s "$output_file" ]]; then
        echo '{"status": "error", "message": "No output file or empty output"}' >&2
        return 1
    fi
    
    # Extract key information from stream-json
    local last_line session_id status files_modified summary turns_used
    last_line=$(tail -n 1 "$output_file")
    
    # Parse session ID from any line that contains it
    session_id=$(grep -o '"sessionId":"[^"]*"' "$output_file" | tail -1 | cut -d'"' -f4)
    if [[ -z "$session_id" ]]; then
        session_id=$(grep -o '"session_id":"[^"]*"' "$output_file" | tail -1 | cut -d'"' -f4)
    fi
    
    # Determine status from last line
    local subtype is_error
    subtype=$(echo "$last_line" | jq -r '.subtype // ""' 2>/dev/null)
    is_error=$(echo "$last_line" | jq -r '.is_error // false' 2>/dev/null)
    
    case "$subtype" in
        "error_max_turns") status="max_turns_reached" ;;
        "success") status="completed" ;;
        *)
            if [[ "$is_error" == "false" ]]; then
                status="completed"
            else
                status="error"
            fi
            ;;
    esac
    
    # Count turns used
    turns_used=$(grep -c '^{' "$output_file" || echo "0")
    
    # Extract summary from content (look for tool calls and responses)
    summary=$(grep '"type":"text"' "$output_file" | tail -1 | jq -r '.text // ""' 2>/dev/null | head -c 200)
    if [[ -z "$summary" ]]; then
        summary="Claude Code operation completed"
    fi
    
    # Try to extract files modified (look for file paths in content)
    files_modified=$(grep -oE '"[^"]*\.(ts|js|py|md|json|yaml|yml|txt|sh)"' "$output_file" | sort -u | jq -R . | jq -s '.' 2>/dev/null || echo '[]')
    
    case "$format" in
        "automation")
            cat <<EOF
{
  "status": "$status",
  "session_id": "${session_id:-}",
  "turns_used": $turns_used,
  "summary": $(echo "$summary" | jq -R .),
  "files_modified": $files_modified,
  "timestamp": "$(date -Iseconds)"
}
EOF
            ;;
        "simple")
            echo "$status"
            ;;
        "json")
            cat "$output_file"
            ;;
        *)
            log::error "Unknown format: $format"
            return 1
            ;;
    esac
}

#######################################
# Extract specific information from Claude Code results
# Arguments:
#   $1 - Path to output file or session ID
#   $2 - Extract type (status|files|summary|session|errors)
# Outputs: Extracted information
#######################################
claude_code::extract() {
    local input="$1"
    local extract_type="$2"
    local output_file=""
    
    # Determine if input is file path or session ID
    if [[ -f "$input" ]]; then
        output_file="$input"
    else
        # Try to find output file by session ID
        output_file=$(find . -name "*${input}*.json" -type f | head -1)
        if [[ ! -f "$output_file" ]]; then
            echo "Session or file not found: $input" >&2
            return 1
        fi
    fi
    
    case "$extract_type" in
        "status")
            claude_code::parse_result "$output_file" "simple"
            ;;
        "files"|"files-changed"|"files-modified")  
            claude_code::parse_result "$output_file" "automation" | jq -r '.files_modified[]' 2>/dev/null || echo ""
            ;;
        "summary")
            claude_code::parse_result "$output_file" "automation" | jq -r '.summary' 2>/dev/null || echo ""
            ;;
        "session"|"session-id")
            claude_code::parse_result "$output_file" "automation" | jq -r '.session_id' 2>/dev/null || echo ""
            ;;
        "errors")
            grep '"is_error":true' "$output_file" | jq -r '.content[0].text // ""' 2>/dev/null || echo ""
            ;;
        "turns")
            claude_code::parse_result "$output_file" "automation" | jq -r '.turns_used' 2>/dev/null || echo "0"
            ;;
        *)
            log::error "Unknown extract type: $extract_type"
            echo "Available types: status, files, summary, session, errors, turns"
            return 1
            ;;
    esac
}

#######################################
# Enhanced session management with automation features
# Arguments:
#   $1 - Action (list|status|export|create)
#   $2 - Session ID (for status/export actions)
#   $3 - Format (json|text|diff) for export
#######################################
claude_code::session_manage() {
    local action="$1"
    local session_id="${2:-}"
    local format="${3:-text}"
    
    case "$action" in
        "list")
            if [[ -d "$CLAUDE_SESSIONS_DIR" ]]; then
                find "$CLAUDE_SESSIONS_DIR" -name "*.json" -type f -exec basename {} .json \; 2>/dev/null | sort -r | head -20
            else
                echo "No session directory found"
                return 1
            fi
            ;;
        "status")
            if [[ -z "$session_id" ]]; then
                echo "Session ID required for status check"
                return 1
            fi
            
            # Look for recent output files with this session ID
            local session_files
            session_files=$(find . -name "*${session_id}*.json" -type f -mtime -1 2>/dev/null)
            
            if [[ -n "$session_files" ]]; then
                local latest_file
                latest_file=$(echo "$session_files" | head -1)
                claude_code::extract "$latest_file" "status"
            else
                echo "unknown"
            fi
            ;;
        "export")
            if [[ -z "$session_id" ]]; then
                echo "Session ID required for export"
                return 1
            fi
            
            local session_files
            session_files=$(find . -name "*${session_id}*.json" -type f 2>/dev/null)
            
            if [[ -z "$session_files" ]]; then
                echo "No files found for session: $session_id"
                return 1
            fi
            
            case "$format" in
                "json")
                    cat $session_files | jq -s '.'
                    ;;
                "diff")
                    # Extract file modifications and show diffs
                    local files_modified
                    files_modified=$(claude_code::extract "$session_files" "files")
                    if [[ -n "$files_modified" ]]; then
                        echo "Files modified in session $session_id:"
                        echo "$files_modified" | while read -r file; do
                            [[ -n "$file" ]] && echo "  - $file"
                        done
                    else
                        echo "No file modifications detected"
                    fi
                    ;;
                "summary")
                    echo "Session: $session_id"
                    echo "Status: $(claude_code::extract "$session_files" "status")"
                    echo "Summary: $(claude_code::extract "$session_files" "summary")"
                    echo "Files: $(claude_code::extract "$session_files" "files" | tr '\n' ' ')"
                    ;;
                *)
                    # Default text format
                    cat $session_files | jq -r '.content[]?.text // empty' | grep -v '^$'
                    ;;
            esac
            ;;
        "create")
            # Generate a new session identifier
            echo "session_$(date +%Y%m%d_%H%M%S)_$(openssl rand -hex 4 2>/dev/null || echo $(($RANDOM * $RANDOM)))"
            ;;
        *)
            log::error "Unknown session action: $action"
            echo "Available actions: list, status, export, create"
            return 1
            ;;
    esac
}

#######################################
# Health check for automation systems
# Arguments:
#   $1 - Check type (basic|full|capabilities)
#   $2 - Output format (json|text)
# Outputs: Health status information
#######################################
claude_code::health_check() {
    local check_type="${1:-basic}"
    local output_format="${2:-text}"
    
    local claude_available=false
    local api_connected=false
    local auth_status="unknown"
    local version=""
    local capabilities=()
    local tty_compatible=true
    local errors=()
    
    # Check if Claude CLI is available
    if claude_code::is_installed; then
        claude_available=true
        version=$(claude_code::get_version 2>/dev/null || echo "unknown")
    else
        errors+=("claude_not_installed")
    fi
    
    # Check API connectivity and authentication (with TTY handling)
    if [[ "$check_type" == "full" && "$claude_available" == true ]]; then
        local test_output
        # Use a minimal non-interactive test to avoid TTY issues
        if test_output=$(timeout 10 claude --version 2>&1); then
            api_connected=true
            auth_status="available"
        else
            # Analyze the error to determine the issue
            if [[ "$test_output" =~ "Raw mode is not supported" ]]; then
                tty_compatible=false
                errors+=("tty_not_supported")
                # Try to get auth status differently
                auth_status="tty_required"
            elif [[ "$test_output" =~ [Aa]uthentication ]] || [[ "$test_output" =~ [Ll]ogin ]]; then
                auth_status="authentication_required"
                errors+=("authentication_required")
            elif [[ "$test_output" =~ "usage limit" ]] || [[ "$test_output" =~ "rate limit" ]]; then
                auth_status="rate_limited"
                errors+=("rate_limited")
            else
                auth_status="error"
                errors+=("connection_failed")
            fi
        fi
    fi
    
    # Get capabilities
    if [[ "$check_type" == "capabilities" || "$check_type" == "full" ]]; then
        # List available tools (based on our known tools)
        capabilities=("Bash" "Read" "Edit" "MultiEdit" "Write" "Grep" "Glob" "LS" "WebSearch" "WebFetch" "TodoWrite" "NotebookRead" "NotebookEdit")
    fi
    
    case "$output_format" in
        "json")
            local caps_json errors_json
            caps_json=$(printf '%s\n' "${capabilities[@]}" | jq -R . | jq -s '.' 2>/dev/null || echo '[]')
            errors_json=$(printf '%s\n' "${errors[@]}" | jq -R . | jq -s '.' 2>/dev/null || echo '[]')
            
            local overall_status="healthy"
            if [[ "$claude_available" != true ]]; then
                overall_status="unhealthy"
            elif [[ "${#errors[@]}" -gt 0 ]]; then
                overall_status="degraded"
            fi
            
            cat <<EOF
{
  "status": "$overall_status",
  "claude_available": $claude_available,
  "api_connected": $api_connected,
  "auth_status": "$auth_status",
  "tty_compatible": $tty_compatible,
  "version": "$version",
  "capabilities": $caps_json,
  "errors": $errors_json,
  "check_type": "$check_type",
  "timestamp": "$(date -Iseconds)"
}
EOF
            ;;
        "text")
            local overall_status="✅ Healthy"
            if [[ "$claude_available" != true ]]; then
                overall_status="❌ Unhealthy"
            elif [[ "${#errors[@]}" -gt 0 ]]; then
                overall_status="⚠️  Degraded" 
            fi
            
            echo "Claude Code Health Check ($check_type)"
            echo "================================"
            echo "Overall Status: $overall_status"
            echo "Claude Available: $([ "$claude_available" = true ] && echo "✅ Yes" || echo "❌ No")"
            echo "Version: $version"
            
            if [[ "$check_type" == "full" ]]; then
                echo "API Connected: $([ "$api_connected" = true ] && echo "✅ Yes" || echo "❌ No")"
                echo "Authentication: $auth_status"
                echo "TTY Compatible: $([ "$tty_compatible" = true ] && echo "✅ Yes" || echo "⚠️  No")"
                
                if [[ "${#errors[@]}" -gt 0 ]]; then
                    echo "Errors: ${#errors[@]} issues found"
                    printf '  - %s\n' "${errors[@]}"
                fi
            fi
            
            if [[ "$check_type" == "capabilities" || "$check_type" == "full" ]]; then
                echo "Available Tools: ${#capabilities[@]} tools"
                printf '  - %s\n' "${capabilities[@]}"
            fi
            ;;
    esac
}

#######################################
# Simple exit code interpretation for automation
# Arguments:
#   $1 - Exit code from Claude Code execution
#   $2 - Path to output file (optional)
# Returns: Simplified exit code (0=success, 1=error, 2=max_turns, 3=permission_denied)
#######################################
claude_code::simple_exit_code() {
    local exit_code="$1"
    local output_file="${2:-}"
    
    # If exit code is 0, it's a success
    if [[ $exit_code -eq 0 ]]; then
        return 0
    fi
    
    # If we have an output file, check the content for more details
    if [[ -n "$output_file" && -f "$output_file" && -s "$output_file" ]]; then
        local status
        status=$(claude_code::extract "$output_file" "status")
        
        case "$status" in
            "completed") return 0 ;;
            "max_turns_reached") return 2 ;;
            "error") return 1 ;;
            *) return 1 ;;
        esac
    fi
    
    # Map common exit codes
    case $exit_code in
        1) return 1 ;;  # General error
        2) return 1 ;;  # Auth error
        3) return 1 ;;  # Network error  
        4) return 3 ;;  # Permission denied
        5) return 1 ;;  # Subscription limit
        *) return 1 ;;  # Unknown error
    esac
}

#######################################
# Execute Claude Code with automation-friendly output
# Arguments:
#   $1 - Prompt
#   $2 - Allowed tools (comma-separated)
#   $3 - Max turns
#   $4 - Output file path
# Returns: 0 on success, appropriate error code on failure
#######################################
claude_code::run_automation() {
    local prompt="$1"
    local allowed_tools="${2:-Read,Edit,Write}"
    local max_turns="${3:-20}"
    local output_file="${4:-}"
    
    if [[ -z "$prompt" ]]; then
        log::error "Prompt is required"
        return 1
    fi
    
    # Generate output file if not provided
    if [[ -z "$output_file" ]]; then
        output_file="/tmp/claude_automation_$(date +%s)_$(openssl rand -hex 4 2>/dev/null || echo $RANDOM).json"
    fi
    
    # Build allowed tools parameters
    local tools_params
    tools_params=$(claude_code::build_allowed_tools "$allowed_tools")
    
    # Build command
    local cmd="$CLAUDE"
    cmd="$cmd $PROMPT_FLAG \"$prompt\""
    cmd="$cmd --max-turns $max_turns"
    cmd="$cmd --output-format stream-json"
    
    if [[ -n "$tools_params" ]]; then
        cmd="$cmd $tools_params"
    fi
    
    # Execute with error handling
    set +e
    eval "$cmd" > "$output_file" 2>&1
    local exit_code=$?
    set -e
    
    # Use simple exit code interpretation
    claude_code::simple_exit_code $exit_code "$output_file"
    local simple_exit=$?
    
    # Store output file path for caller
    echo "$output_file" >&2
    
    return $simple_exit
}

#######################################
# Batch execution with progress tracking for automation
# Arguments:
#   $1 - Prompt
#   $2 - Total turns
#   $3 - Batch size (optional, defaults to 50)
#   $4 - Progress callback function (optional)
# Outputs: Final result in automation format
#######################################
claude_code::batch_automation() {
    local prompt="$1"
    local total_turns="$2"
    local batch_size="${3:-50}"
    local progress_callback="${4:-}"
    
    if [[ -z "$prompt" || -z "$total_turns" ]]; then
        log::error "Prompt and total turns are required"
        return 1
    fi
    
    local session_id=""
    local remaining=$total_turns
    local batch_count=0
    local failed_batches=0
    local output_dir="/tmp/claude_batch_$(date +%s)"
    
    mkdir -p "$output_dir"
    
    while (( remaining > 0 )); do
        local current_batch=$(( remaining < batch_size ? remaining : batch_size ))
        batch_count=$((batch_count + 1))
        
        local batch_output="$output_dir/batch_${batch_count}.json"
        
        # Call progress callback if provided
        if [[ -n "$progress_callback" && $(type -t "$progress_callback") == "function" ]]; then
            $progress_callback $batch_count $current_batch $remaining
        fi
        
        # Build command with session resumption if available
        local cmd="$CLAUDE $PROMPT_FLAG \"$prompt\""
        cmd="$cmd --max-turns $current_batch"
        cmd="$cmd --output-format stream-json"
        
        if [[ -n "$session_id" ]]; then
            cmd="$cmd --resume \"$session_id\""
        fi
        
        # Execute batch
        set +e
        eval "$cmd" > "$batch_output" 2>&1
        local exit_code=$?
        set -e
        
        # Check if batch succeeded
        if claude_code::simple_exit_code $exit_code "$batch_output"; then
            # Extract session ID for next batch
            session_id=$(claude_code::extract "$batch_output" "session")
            remaining=$((remaining - current_batch))
        else
            failed_batches=$((failed_batches + 1))
            if [[ $failed_batches -gt 2 ]]; then
                log::error "Too many failed batches, aborting"
                break
            fi
            # Continue with next batch (maybe it was just a transient error)
        fi
    done
    
    # Consolidate results
    local final_status="completed"
    if [[ $remaining -gt 0 ]]; then
        final_status="partial"
    fi
    
    local all_files_modified
    all_files_modified=$(find "$output_dir" -name "*.json" -exec claude_code::extract {} "files" \; | sort -u | jq -R . | jq -s '.' 2>/dev/null || echo '[]')
    
    local final_summary
    final_summary=$(find "$output_dir" -name "*.json" -exec claude_code::extract {} "summary" \; | tail -1)
    
    cat <<EOF
{
  "status": "$final_status",
  "session_id": "$session_id",
  "total_turns_requested": $total_turns,
  "turns_remaining": $remaining,
  "batches_executed": $batch_count,
  "failed_batches": $failed_batches,
  "files_modified": $all_files_modified,
  "summary": $(echo "$final_summary" | jq -R .),
  "output_directory": "$output_dir",
  "timestamp": "$(date -Iseconds)"
}
EOF
}

#######################################
# Wrapper function to add automation capabilities to existing management script
# This allows gradual migration to automation features
#######################################
claude_code::automation_wrapper() {
    local action="$1"
    shift
    
    case "$action" in
        "parse-result")
            claude_code::parse_result "$@"
            ;;
        "extract")
            claude_code::extract "$@"
            ;;
        "session-manage")
            claude_code::session_manage "$@"
            ;;
        "health-check")
            claude_code::health_check "$@"
            ;;
        "run-automation")
            claude_code::run_automation "$@"
            ;;
        "batch-automation")
            claude_code::batch_automation "$@"
            ;;
        *)
            # Fall back to original management functions
            return 1
            ;;
    esac
}

# Export functions for external use
export -f claude_code::parse_result
export -f claude_code::extract  
export -f claude_code::session_manage
export -f claude_code::health_check
export -f claude_code::simple_exit_code
export -f claude_code::run_automation
export -f claude_code::batch_automation
export -f claude_code::automation_wrapper