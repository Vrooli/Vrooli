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
    
    # Build command
    local cmd="claude"
    cmd="$cmd --prompt \"$PROMPT\""
    cmd="$cmd --max-turns $MAX_TURNS"
    
    if [[ "$OUTPUT_FORMAT" == "stream-json" ]]; then
        cmd="$cmd --output-format stream-json"
    fi
    
    # Add allowed tools if specified
    local tools_params
    tools_params=$(claude_code::build_allowed_tools "$ALLOWED_TOOLS")
    if [[ -n "$tools_params" ]]; then
        cmd="$cmd $tools_params"
    fi
    
    # Add skip permissions flag if specified
    if [[ "$SKIP_PERMISSIONS" == "yes" ]]; then
        cmd="$cmd --dangerously-skip-permissions"
        log::warn "⚠️  WARNING: Permission checks are disabled!"
    fi
    
    # Set timeout environment variables
    claude_code::set_timeouts "$TIMEOUT"
    
    log::info "Executing: $cmd"
    echo
    
    # Execute Claude
    eval "$cmd"
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
    
    # Run with batch-friendly settings
    local cmd="claude"
    cmd="$cmd --prompt \"$PROMPT\""
    cmd="$cmd --max-turns $MAX_TURNS"
    cmd="$cmd --output-format stream-json"
    cmd="$cmd --no-interactive"
    
    # Add allowed tools if specified
    local tools_params
    tools_params=$(claude_code::build_allowed_tools "$ALLOWED_TOOLS")
    if [[ -n "$tools_params" ]]; then
        cmd="$cmd $tools_params"
    fi
    
    # Add skip permissions flag if specified
    if [[ "$SKIP_PERMISSIONS" == "yes" ]]; then
        cmd="$cmd --dangerously-skip-permissions"
        log::warn "⚠️  WARNING: Permission checks are disabled for batch operation!"
    fi
    
    # Set extended timeouts for batch operations
    claude_code::set_timeouts "$TIMEOUT"
    
    log::info "Executing batch: $cmd"
    echo
    
    # Execute and capture output
    local output_file="/tmp/claude-batch-${RANDOM}.json"
    eval "$cmd" > "$output_file" 2>&1
    
    if [[ -f "$output_file" ]]; then
        log::success "✓ Batch completed. Output saved to: $output_file"
        log::info "To view results: cat $output_file | jq ."
    else
        log::error "Batch execution failed"
        return 1
    fi
}