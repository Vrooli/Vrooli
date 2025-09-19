#!/bin/bash
# OpenCode Docker Functions (stubs - OpenCode is a VS Code extension, not a Docker service)

# Source common functions
source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::docker::start() {
    opencode_detect_vscode
    log::info "OpenCode is a VS Code extension and doesn't require Docker services"
    log::info "The extension runs within VS Code when VS Code is started"
    
    # Check if VS Code can be launched
    if [[ -n "${VSCODE_COMMAND}" ]]; then
        log::success "VS Code is available and can be started with: ${VSCODE_COMMAND}"
        
        # Optionally launch VS Code if --launch flag is provided
        if [[ "${1:-}" == "--launch" ]]; then
            log::info "Launching VS Code..."
            
            # Register agent if agent management is available
            local agent_id=""
            if type -t agents::register &>/dev/null; then
                agent_id=$(agents::generate_id)
                local command_string="resource-opencode manage start --launch"
                if agents::register "$agent_id" $$ "$command_string"; then
                    log::debug "Registered agent: $agent_id"
                    
                    # Set up signal handler for cleanup
                    opencode::setup_agent_cleanup "$agent_id"
                fi
            fi
            
            ${VSCODE_COMMAND} &
            local vscode_pid=$!
            
            # Update agent with VS Code process if registered
            if [[ -n "$agent_id" ]] && type -t agents::register &>/dev/null; then
                # Re-register with the actual VS Code PID
                agents::unregister "$agent_id" >/dev/null 2>&1
                local updated_command="resource-opencode manage start --launch (VS Code PID: $vscode_pid)"
                agents::register "$agent_id" "$vscode_pid" "$updated_command" >/dev/null 2>&1
            fi
            
            log::success "VS Code launched in background (PID: $vscode_pid)"
        fi
    else
        log::error "VS Code is not available. Please install VS Code first."
        return 1
    fi
}

opencode::docker::stop() {
    opencode_detect_vscode
    log::info "OpenCode is a VS Code extension and doesn't have Docker services to stop"
    log::info "The extension stops when VS Code is closed"
    
    # Check if VS Code processes are running and offer to close them
    local vscode_pids=$(pgrep -f "code" 2>/dev/null || true)
    if [[ -n "${vscode_pids}" ]]; then
        log::info "Found running VS Code processes: ${vscode_pids}"
        log::info "To stop OpenCode, close VS Code or kill these processes manually"
    else
        log::success "No VS Code processes found running"
    fi
}

opencode::docker::restart() {
    opencode_detect_vscode
    log::info "OpenCode is a VS Code extension - restart involves reloading the extension or restarting VS Code"
    
    if [[ -n "${VSCODE_COMMAND}" ]]; then
        log::info "To restart OpenCode:"
        log::info "1. In VS Code, use Cmd/Ctrl+Shift+P and run 'Developer: Reload Window'"
        log::info "2. Or close and reopen VS Code"
        log::success "VS Code restart instructions provided"
    else
        log::error "VS Code is not available"
        return 1
    fi
}

opencode::docker::logs() {
    log::info "OpenCode logs are available through VS Code's built-in logging system"
    
    local log_locations=(
        "${OPENCODE_DATA_DIR}/logs"
        "${HOME}/.vscode/logs"
        "${HOME}/Library/Application Support/Code/logs"
        "${HOME}/.config/Code/logs"
    )
    
    log::info "Check these locations for OpenCode logs:"
    for location in "${log_locations[@]}"; do
        if [[ -d "${location}" ]]; then
            echo "✅ ${location}"
            # Show recent log files if they exist
            local recent_logs=$(find "${location}" -name "*.log" -type f -mtime -1 2>/dev/null | head -3)
            if [[ -n "${recent_logs}" ]]; then
                echo "${recent_logs}" | sed 's/^/   - /'
            fi
        else
            echo "❌ ${location}"
        fi
    done
    
    log::info "For real-time VS Code logs, use: ${VSCODE_COMMAND} --verbose"
}
