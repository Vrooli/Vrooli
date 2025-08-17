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
    # Always use non-interactive mode for autonomous platform integration
    cmd_args+=("--print")  # Use non-interactive mode for script execution
    cmd_args+=("--max-turns" "${MAX_TURNS:-5}")
    
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
    
    # Configure sudo override if enabled
    if [[ "${SUDO_OVERRIDE:-}" == "yes" ]]; then
        local sudo_commands="${SUDO_COMMANDS:-}"
        local sudo_password="${SUDO_PASSWORD:-}"
        
        if claude_code::configure_sudo_override "yes" "$sudo_commands" "$sudo_password"; then
            log::info "🔧 Sudo override configured for Claude Code execution"
            
            # Add sudo-related tools to allowed tools
            if [[ -n "$ALLOWED_TOOLS" ]]; then
                ALLOWED_TOOLS="$ALLOWED_TOOLS,Bash(sudo:*)"
            else
                ALLOWED_TOOLS="Bash(sudo:*)"
            fi
            
            # Set up cleanup trap
            trap 'claude_code::cleanup_sudo_override' EXIT
        else
            log::error "❌ Failed to configure sudo override"
            return 1
        fi
    fi
    
    # Set timeout environment variables
    claude_code::set_timeouts "$TIMEOUT"
    
    log::info "Executing: timeout ${TIMEOUT:-600} claude ${cmd_args[*]} \"$PROMPT\""
    echo
    
    # Execute Claude with timeout and streaming output (no capture to prevent hanging)
    local exit_code
    local temp_output_file
    temp_output_file=$(mktemp)
    
    # Use timeout with streaming to prevent hanging, capture minimal output for error handling
    # Claude expects prompt as argument, not stdin
    {
        timeout "${TIMEOUT:-600}" claude "${cmd_args[@]}" "$PROMPT" 2>&1
        echo $? > "${temp_output_file}.exit"
    } | tee "$temp_output_file"
    
    exit_code=$(cat "${temp_output_file}.exit" 2>/dev/null || echo "124")
    
    # Handle timeout specifically (exit code 124)
    if [[ $exit_code -eq 124 ]]; then
        log::error "Claude execution timed out after ${TIMEOUT:-600} seconds"
        log::info "Try increasing timeout with: --timeout <seconds>"
        log::info "Or use a simpler prompt that requires less processing"
        rm -f "$temp_output_file" "${temp_output_file}.exit"
        return $exit_code
    fi
    
    # Handle other error patterns with enhanced TTY support
    if [[ $exit_code -ne 0 ]]; then
        local output
        output=$(cat "$temp_output_file" 2>/dev/null || echo "")
        
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
        
        # Cleanup temp files
        rm -f "$temp_output_file" "${temp_output_file}.exit"
        return $exit_code
    fi
    
    # Cleanup temp files on success
    rm -f "$temp_output_file" "${temp_output_file}.exit"
}

#######################################
# Configure sudo override for Claude Code
# Arguments:
#   $1 - Enable sudo override (yes/no)
#   $2 - Comma-separated list of allowed commands
#   $3 - Sudo password (optional)
# Returns: 0 on success, 1 on failure
#######################################
claude_code::configure_sudo_override() {
    local enable_sudo="$1"
    local allowed_commands="${2:-}"
    local sudo_password="${3:-}"
    
    if [[ "$enable_sudo" != "yes" ]]; then
        log::debug "Sudo override disabled"
        return 0
    fi
    
    log::info "🔧 Configuring sudo override for Claude Code"
    
    # Check if sudo is available
    if ! command -v sudo >/dev/null 2>&1; then
        log::error "Sudo is not available on this system"
        return 1
    fi
    
    # Check if user has sudo privileges
    if ! sudo -n true 2>/dev/null; then
        log::warn "⚠️  User does not have passwordless sudo access"
        log::info "Sudo override may prompt for password during execution"
        
        # If password provided, test sudo access
        if [[ -n "$sudo_password" ]]; then
            if ! echo "$sudo_password" | sudo -S true 2>/dev/null; then
                log::error "❌ Invalid sudo password provided"
                return 1
            fi
            log::success "✅ Sudo password validated"
        fi
    else
        log::success "✅ Passwordless sudo access confirmed"
    fi
    
    # Create temporary sudoers file for Claude Code
    local temp_sudoers="/tmp/claude-code-sudoers-$$"
    local sudoers_content="# Claude Code temporary sudo permissions\n"
    sudoers_content+="# Generated on $(date)\n"
    sudoers_content+="# User: $(whoami)\n\n"
    
    # Add user with specific command permissions
    if [[ -n "$allowed_commands" ]]; then
        IFS=',' read -ra commands <<< "$allowed_commands"
        for cmd in "${commands[@]}"; do
            cmd=$(echo "$cmd" | xargs)  # Trim whitespace
            if [[ -n "$cmd" ]]; then
                # Find the full path of the command
                local cmd_path
                cmd_path=$(command -v "$cmd" 2>/dev/null)
                
                # Handle shell builtins and special cases
                if [[ -z "$cmd_path" ]]; then
                    # For shell builtins like echo, we need to allow the shell
                    if [[ "$cmd" == "echo" ]]; then
                        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /bin/bash -c echo*\n"
                        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /bin/sh -c echo*\n"
                    else
                        # Try common locations
                        cmd_path="/usr/bin/$cmd"
                        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: $cmd_path\n"
                    fi
                else
                    sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: $cmd_path\n"
                fi
            fi
        done
    else
        # Default permissions for common resource management commands
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/systemctl\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/service\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/docker\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/podman\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/apt-get\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/apt\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/chown\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/chmod\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/mkdir\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/rm\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/cp\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/mv\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/lsof\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/netstat\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/ps\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/kill\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/pkill\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/npm\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/pip\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/brew\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/snap\n"
        sudoers_content+="$(whoami) ALL=(ALL) NOPASSWD: /usr/bin/git\n"
    fi
    
    # Write temporary sudoers file
    echo -e "$sudoers_content" > "$temp_sudoers"
    
    # Install temporary sudoers file
    if sudo cp "$temp_sudoers" "/etc/sudoers.d/claude-code-temp-$$" 2>/dev/null; then
        sudo chmod 0440 "/etc/sudoers.d/claude-code-temp-$$" 2>/dev/null
        log::success "✅ Temporary sudoers file installed: /etc/sudoers.d/claude-code-temp-$$"
        
        # Store cleanup information
        export CLAUDE_SUDOERS_FILE="/etc/sudoers.d/claude-code-temp-$$"
        export CLAUDE_SUDO_PASSWORD="$sudo_password"
        
        # Clean up temp file
        rm -f "$temp_sudoers"
        return 0
    else
        log::error "❌ Failed to install temporary sudoers file"
        rm -f "$temp_sudoers"
        return 1
    fi
}

#######################################
# Clean up sudo override configuration
# Returns: 0 on success, 1 on failure
#######################################
claude_code::cleanup_sudo_override() {
    local sudoers_file="${CLAUDE_SUDOERS_FILE:-}"
    
    if [[ -n "$sudoers_file" && -f "$sudoers_file" ]]; then
        log::info "🧹 Cleaning up sudo override configuration"
        
        if sudo rm -f "$sudoers_file" 2>/dev/null; then
            log::success "✅ Sudo override configuration cleaned up"
        else
            log::warn "⚠️  Failed to clean up sudo override configuration"
            log::info "Manual cleanup required: sudo rm -f $sudoers_file"
        fi
        
        # Clear environment variables
        unset CLAUDE_SUDOERS_FILE
        unset CLAUDE_SUDO_PASSWORD
    fi
}

#######################################
# Test sudo override functionality
# Returns: 0 on success, 1 on failure
#######################################
claude_code::test_sudo_override() {
    log::info "🧪 Testing sudo override functionality"
    
    # Test basic sudo access
    if sudo -n echo "Sudo test" 2>/dev/null; then
        log::success "✅ Passwordless sudo access confirmed"
        return 0
    else
        log::warn "⚠️  Passwordless sudo not available"
        
        # Test with password if available
        local sudo_password="${CLAUDE_SUDO_PASSWORD:-}"
        if [[ -n "$sudo_password" ]]; then
            if echo "$sudo_password" | sudo -S echo "Sudo test with password" 2>/dev/null; then
                log::success "✅ Sudo access with password confirmed"
                return 0
            else
                log::error "❌ Sudo access with password failed"
                return 1
            fi
        else
            log::error "❌ No sudo password available for testing"
            return 1
        fi
    fi
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