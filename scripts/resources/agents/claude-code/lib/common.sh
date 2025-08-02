#!/usr/bin/env bash
# Claude Code Common Utility Functions
# Shared utilities used across all modules

#######################################
# Check if Node.js is installed and meets version requirements
# Returns: 0 if valid, 1 otherwise
#######################################
claude_code::check_node_version() {
    if ! system::is_command node; then
        return 1
    fi
    
    local node_version
    node_version=$(node --version | sed 's/v//' | cut -d'.' -f1)
    
    if [[ "$node_version" -lt "$MIN_NODE_VERSION" ]]; then
        return 1
    fi
    
    return 0
}

#######################################
# Check if Claude Code is installed
# Returns: 0 if installed, 1 otherwise
#######################################
claude_code::is_installed() {
    if system::is_command claude; then
        return 0
    fi
    return 1
}

#######################################
# Get Claude Code version
# Outputs: version string or "not installed"
#######################################
claude_code::get_version() {
    if claude_code::is_installed; then
        claude --version 2>/dev/null || echo "unknown"
    else
        echo "not installed"
    fi
}

#######################################
# Set timeout environment variables for Claude operations
# Arguments:
#   $1 - timeout in seconds
#######################################
claude_code::set_timeouts() {
    local timeout="${1:-$DEFAULT_TIMEOUT}"
    export BASH_DEFAULT_TIMEOUT_MS=$((timeout * 1000))
    export BASH_MAX_TIMEOUT_MS=$((timeout * 1000))
    export MCP_TOOL_TIMEOUT=$((timeout * 1000))
}

#######################################
# Build allowed tools parameter string
# Arguments:
#   $1 - comma-separated list of tools
# Outputs: parameter string for claude command
#######################################
claude_code::build_allowed_tools() {
    local tools_list="$1"
    local result=""
    
    if [[ -n "$tools_list" ]]; then
        IFS=',' read -ra TOOLS <<< "$tools_list"
        for tool in "${TOOLS[@]}"; do
            result="$result --allowedTools \"$tool\""
        done
    fi
    
    echo "$result"
}

#######################################
# Check if running in a TTY environment
# Returns: 0 if TTY, 1 otherwise
#######################################
claude_code::is_tty() {
    [[ -t 0 && -t 1 ]]
}

#######################################
# Check if command requires TTY and handle appropriately
# Arguments:
#   $1 - command name (e.g., "doctor", "interactive")
# Returns: 0 if can proceed, 1 if should skip
#######################################
claude_code::check_tty_requirement() {
    local command="$1"
    
    if ! claude_code::is_tty; then
        case "$command" in
            "doctor")
                log::warn "Interactive 'claude doctor' not supported in non-TTY environment"
                log::info "Use --action health-check instead for non-interactive diagnostics"
                return 1
                ;;
            "interactive")
                log::error "Interactive mode requires a TTY environment"
                log::info "Use --action run with --prompt for non-interactive execution"
                return 1
                ;;
            *)
                # Command doesn't require TTY
                return 0
                ;;
        esac
    fi
    
    return 0
}