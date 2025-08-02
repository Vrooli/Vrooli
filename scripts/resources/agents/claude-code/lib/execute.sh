#!/usr/bin/env bash
# Claude Code Execution Functions
# Handles running Claude with prompts and batch operations

#######################################
# Run Claude with a single prompt
#######################################
claude_code::run() {
    log::header "🤖 Running Claude Code"
    
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi
    
    if [[ -z "${PROMPT:-}" ]]; then
        log::error "No prompt provided. Use --prompt \"Your prompt here\""
        return 1
    fi
    
    # Build command arguments with TTY compatibility
    local cmd_args=()
    cmd_args+=("--print")  # Use non-interactive mode for script execution
    cmd_args+=("--max-turns" "$MAX_TURNS")
    
    # Ensure non-interactive mode for automation environments
    if ! claude_code::is_tty; then
        log::info "Non-TTY environment detected - using automation-friendly settings"
    fi
    
    if [[ "$OUTPUT_FORMAT" == "stream-json" ]]; then
        cmd_args+=("--output-format" "stream-json")
    fi
    
    # Add allowed tools if specified
    if [[ -n "$ALLOWED_TOOLS" ]]; then
        cmd_args+=("--allowedTools" "$ALLOWED_TOOLS")
    fi
    
    # Add skip permissions flag if specified
    if [[ "$SKIP_PERMISSIONS" == "yes" ]]; then
        cmd_args+=("--dangerously-skip-permissions")
        log::warn "⚠️  WARNING: Permission checks are disabled!"
    fi
    
    # Set timeout environment variables
    claude_code::set_timeouts "$TIMEOUT"
    
    log::info "Executing: echo \"$PROMPT\" | claude ${cmd_args[*]}"
    echo
    
    # Execute Claude with prompt via stdin
    local exit_code
    local output
    output=$(echo "$PROMPT" | claude "${cmd_args[@]}" 2>&1)
    exit_code=$?
    
    # Handle common error patterns with enhanced TTY support
    if [[ $exit_code -ne 0 ]]; then
        if [[ "$output" =~ "unknown option" ]]; then
            log::error "CLI interface error: Unknown option detected"
            log::info "This may indicate the claude CLI has changed"
            log::info "Please check: claude --help"
        elif [[ "$output" =~ "Raw mode is not supported" ]]; then
            log::error "TTY error: Interactive mode not supported in current environment"
            log::info "Claude Code requires a TTY for some operations"
            log::info "Fallback: Use the health-check action for non-interactive diagnostics"
            log::info "  $0 --action health-check --check-type full"
            # Attempt to provide basic status without TTY
            if claude_code::is_installed; then
                local version
                version=$(claude_code::get_version)
                log::info "Basic status: Claude Code $version is installed"
            fi
        elif [[ "$output" =~ [Aa]uthentication ]] || [[ "$output" =~ [Ll]ogin ]] || [[ "$output" =~ "sign.*in" ]]; then
            log::error "Authentication required"
            log::info "To authenticate Claude Code:"
            if claude_code::is_tty; then
                log::info "  1. Run: claude"
                log::info "  2. Follow the authentication prompts"
            else
                log::info "  1. Run claude interactively in a TTY environment"
                log::info "  2. Complete authentication setup"
                log::info "  3. Then retry this command"
            fi
        elif [[ "$output" =~ "usage limit" ]] || [[ "$output" =~ "rate limit" ]]; then
            log::error "Usage limit reached"
            log::info "You've reached your Claude usage limit"
            log::info "  - Wait for the limit to reset (typically every 5 hours)"
            log::info "  - Consider upgrading to Claude Pro or Max for higher limits"
            log::info "  - Check your usage at claude.ai"
        else
            log::error "Claude execution failed with exit code: $exit_code"
            log::info "For detailed diagnostics, run:"
            log::info "  $0 --action health-check --check-type full"
        fi
        echo "$output"
        return $exit_code
    fi
    
    echo "$output"
}

#######################################
# Run Claude in batch mode
#######################################
claude_code::batch() {
    log::header "📦 Running Claude Code in Batch Mode"
    
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi
    
    if [[ -z "${PROMPT:-}" ]]; then
        log::error "No prompt provided. Use --prompt \"Your prompt here\""
        return 1
    fi
    
    log::info "Starting batch execution with max turns: $MAX_TURNS"
    log::info "Timeout: ${TIMEOUT}s per operation"
    
    # Build batch command arguments
    local cmd_args=()
    cmd_args+=("--print")  # Use non-interactive mode
    cmd_args+=("--max-turns" "$MAX_TURNS")
    cmd_args+=("--output-format" "stream-json")
    
    # Add allowed tools if specified
    if [[ -n "$ALLOWED_TOOLS" ]]; then
        cmd_args+=("--allowedTools" "$ALLOWED_TOOLS")
    fi
    
    # Add skip permissions flag if specified
    if [[ "$SKIP_PERMISSIONS" == "yes" ]]; then
        cmd_args+=("--dangerously-skip-permissions")
        log::warn "⚠️  WARNING: Permission checks are disabled for batch operation!"
    fi
    
    # Set extended timeouts for batch operations
    claude_code::set_timeouts "$TIMEOUT"
    
    log::info "Executing batch: echo \"$PROMPT\" | claude ${cmd_args[*]}"
    echo
    
    # Execute and capture output
    local output_file="/tmp/claude-batch-${RANDOM}.json"
    local exit_code
    
    echo "$PROMPT" | claude "${cmd_args[@]}" > "$output_file" 2>&1
    exit_code=$?
    
    if [[ $exit_code -ne 0 ]] && [[ -f "$output_file" ]]; then
        local error_content
        error_content=$(cat "$output_file" 2>/dev/null)
        
        if [[ "$error_content" =~ "unknown option" ]]; then
            log::error "CLI interface error: Unknown option detected in batch mode"
            log::info "This may indicate the claude CLI has changed"
        elif [[ "$error_content" =~ "Authentication" ]] || [[ "$error_content" =~ "login" ]]; then
            log::error "Authentication required for batch operation"
            log::info "Please run: claude"
        else
            log::error "Batch execution failed with exit code: $exit_code"
        fi
        
        cat "$output_file"
        return $exit_code
    fi
    
    if [[ -f "$output_file" ]]; then
        log::success "✓ Batch completed. Output saved to: $output_file"
        log::info "To view results: cat $output_file | jq ."
    else
        log::error "Batch execution failed - no output file created"
        return 1
    fi
}

# Export functions for subshell availability
export -f claude_code::run
export -f claude_code::batch