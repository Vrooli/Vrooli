#!/usr/bin/env bash
# Claude Code Status Functions
# Handles status checking, info display, and log viewing

#######################################
# Check Claude Code status
#######################################
claude_code::status() {
    log::header "📊 Claude Code Status"
    
    # Installation status
    if claude_code::is_installed; then
        local version
        version=$(claude_code::get_version)
        log::success "✓ Claude Code is installed"
        log::info "  Version: $version"
        log::info "  Command: claude"
        
        # Check Node.js
        if claude_code::check_node_version; then
            local node_version
            node_version=$(node --version)
            log::success "✓ Node.js requirements met ($node_version)"
        else
            log::warn "⚠️  Node.js version requirement not met (need v$MIN_NODE_VERSION+)"
        fi
        
        # Show Claude doctor output if available (only in interactive mode)
        if [[ -t 0 && -t 1 ]]; then
            echo
            log::info "Running claude doctor..."
            if claude doctor 2>/dev/null; then
                echo
            else
                log::warn "⚠️  Could not run claude doctor (may need authentication)"
            fi
        else
            echo
            log::info "Run 'claude doctor' interactively for diagnostic information"
        fi
        
        return 0
    else
        log::warn "✗ Claude Code is not installed"
        log::info "  Run: $0 --action install"
        return 1
    fi
}

#######################################
# Display detailed information
#######################################
claude_code::info() {
    log::header "ℹ️  Claude Code Information"
    
    # Display general information
    claude_code::display_info
    
    # Show current status
    echo
    claude_code::status
}

#######################################
# View Claude logs
#######################################
claude_code::logs() {
    log::header "📜 Claude Code Logs"
    
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi
    
    local found_logs=false
    
    for log_dir in "${CLAUDE_LOG_LOCATIONS[@]}"; do
        if [[ -d "$log_dir" ]]; then
            found_logs=true
            log::info "Found logs in: $log_dir"
            
            # Show recent log files
            local recent_logs
            recent_logs=$(ls -t "$log_dir" 2>/dev/null | head -5)
            
            if [[ -n "$recent_logs" ]]; then
                echo "Recent log files:"
                echo "$recent_logs" | nl
                echo
                
                # Show tail of most recent log
                local latest_log
                latest_log=$(ls -t "$log_dir" | head -1)
                if [[ -n "$latest_log" ]]; then
                    log::info "Last 20 lines of $latest_log:"
                    tail -20 "$log_dir/$latest_log"
                fi
            fi
            echo
        fi
    done
    
    if ! $found_logs; then
        log::info "No log files found"
        log::info "Logs are created when Claude runs into issues"
    fi
    
    # Check for debug mode
    echo
    log::info "To enable debug logging:"
    log::info "  export CLAUDE_DEBUG=1"
    log::info "  export LOG_LEVEL=debug"
}