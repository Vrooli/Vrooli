#!/usr/bin/env bash
# Claude Code Execution Functions
# Handles running Claude with prompts and batch operations

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

#######################################
# Run Claude with a single prompt
#######################################
claude_code::run() {
    log::header "Running Claude Code"

    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi

    # Read prompt from file or environment variable (file takes precedence for large prompts)
    if [[ -n "$PROMPT_FILE" ]] && [[ -f "$PROMPT_FILE" ]]; then
        PROMPT=$(cat "$PROMPT_FILE")
    elif [[ -z "${PROMPT:-}" ]]; then
        log::error "No prompt provided. Use --prompt \"Your prompt here\""
        return 1
    fi

    # Build command arguments
    local cmd_args=()
    cmd_args+=("--print")
    if [[ -n "${CLAUDE_RESUME_SESSION:-}" ]]; then
        cmd_args+=("--resume" "${CLAUDE_RESUME_SESSION}")
    fi
    cmd_args+=("--max-turns" "${MAX_TURNS:-5}")

    if [[ "$OUTPUT_FORMAT" == "stream-json" ]]; then
        cmd_args+=("--output-format" "stream-json")
        cmd_args+=("--verbose")
    fi

    if [[ -n "$ALLOWED_TOOLS" ]]; then
        cmd_args+=("--allowedTools" "$ALLOWED_TOOLS")
    fi

    if [[ "$SKIP_PERMISSIONS" == "yes" ]]; then
        cmd_args+=("--dangerously-skip-permissions")
    fi

    if [[ -n "${APPEND_SYSTEM_PROMPT:-}" ]]; then
        cmd_args+=("--append-system-prompt" "$APPEND_SYSTEM_PROMPT")
    fi

    local prompt_size=${#PROMPT}
    log::info "Executing: timeout ${TIMEOUT:-600} claude ${cmd_args[*]} (prompt: $prompt_size chars)"
    echo

    # Execute Claude with timeout, capturing output for error handling
    local exit_code
    local temp_output_file
    temp_output_file=$(mktemp)

    {
        echo "$PROMPT" | timeout --foreground "${TIMEOUT:-600}" claude "${cmd_args[@]}" 2>&1
        echo ${PIPESTATUS[1]} > "${temp_output_file}.exit"
    } | tee "$temp_output_file"

    exit_code=$(cat "${temp_output_file}.exit" 2>/dev/null || echo "124")

    if [[ $exit_code -eq 124 ]]; then
        log::error "Claude execution timed out after ${TIMEOUT:-600} seconds"
        rm -f "$temp_output_file" "${temp_output_file}.exit"
        return $exit_code
    fi

    if [[ $exit_code -ne 0 ]]; then
        local output
        output=$(cat "$temp_output_file" 2>/dev/null || echo "")

        if [[ "$output" =~ "unknown option" ]]; then
            log::error "CLI interface error: Unknown option detected"
        elif [[ "$output" =~ [Aa]uthentication ]] || [[ "$output" =~ [Ll]ogin ]]; then
            log::error "Authentication required. Run: claude"
        elif [[ "$output" =~ "usage limit" ]] || [[ "$output" =~ "rate limit" ]]; then
            log::error "Usage/rate limit reached. Wait for reset or upgrade plan."
        else
            log::error "Claude execution failed with exit code: $exit_code"
        fi

        rm -f "$temp_output_file" "${temp_output_file}.exit"
        return $exit_code
    fi

    rm -f "$temp_output_file" "${temp_output_file}.exit"
    return 0
}

#######################################
# Run Claude in batch mode
#######################################
claude_code::batch() {
    log::header "Running Claude Code in Batch Mode"

    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi

    if [[ -n "$PROMPT_FILE" ]] && [[ -f "$PROMPT_FILE" ]]; then
        PROMPT=$(cat "$PROMPT_FILE")
    elif [[ -z "${PROMPT:-}" ]]; then
        log::error "No prompt provided. Use --prompt \"Your prompt here\""
        return 1
    fi

    log::info "Starting batch execution with max turns: $MAX_TURNS"

    local cmd_args=()
    cmd_args+=("--print")
    cmd_args+=("--max-turns" "$MAX_TURNS")
    cmd_args+=("--output-format" "stream-json")

    if [[ -n "$ALLOWED_TOOLS" ]]; then
        cmd_args+=("--allowedTools" "$ALLOWED_TOOLS")
    fi

    if [[ "$SKIP_PERMISSIONS" == "yes" ]]; then
        cmd_args+=("--dangerously-skip-permissions")
    fi

    log::info "Executing batch: echo \"\$PROMPT\" | claude ${cmd_args[*]}"
    echo

    local output_file="/tmp/claude-batch-${RANDOM}.json"
    local exit_code

    echo "$PROMPT" | claude "${cmd_args[@]}" > "$output_file" 2>&1
    exit_code=$?

    if [[ $exit_code -ne 0 ]] && [[ -f "$output_file" ]]; then
        log::error "Batch execution failed with exit code: $exit_code"
        cat "$output_file"
        return $exit_code
    fi

    if [[ -f "$output_file" ]]; then
        log::success "Batch completed. Output saved to: $output_file"
    else
        log::error "Batch execution failed - no output file created"
        return 1
    fi
}

# Export functions for subshell availability
export -f claude_code::run
export -f claude_code::batch
