#!/usr/bin/env bash
set -euo pipefail

# Claude Code management script
# This script manages the Claude Code CLI installation

DESCRIPTION="Manages Claude Code CLI installation and configuration"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Resource configuration
readonly RESOURCE_NAME="claude-code"
readonly MIN_NODE_VERSION="18"
readonly CLAUDE_PACKAGE="@anthropic-ai/claude-code"

#######################################
# Parse command line arguments
#######################################
claude_code::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|status|info|run|batch|session|settings|logs|help" \
        --default "help"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if already installed" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "prompt" \
        --flag "p" \
        --desc "Prompt to send to Claude" \
        --type "value" \
        --default ""
    
    args::register \
        --name "max-turns" \
        --desc "Maximum number of turns for Claude" \
        --type "value" \
        --default "5"
    
    args::register \
        --name "session-id" \
        --desc "Session ID to resume or manage" \
        --type "value" \
        --default ""
    
    args::register \
        --name "allowed-tools" \
        --desc "Comma-separated list of allowed tools" \
        --type "value" \
        --default ""
    
    args::register \
        --name "timeout" \
        --desc "Timeout in seconds for Claude execution" \
        --type "value" \
        --default "600"
    
    args::register \
        --name "output-format" \
        --desc "Output format (stream-json, text)" \
        --type "value" \
        --options "stream-json|text" \
        --default "text"
    
    if args::is_asking_for_help "$@"; then
        claude_code::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export PROMPT=$(args::get "prompt")
    export MAX_TURNS=$(args::get "max-turns")
    export SESSION_ID=$(args::get "session-id")
    export ALLOWED_TOOLS=$(args::get "allowed-tools")
    export TIMEOUT=$(args::get "timeout")
    export OUTPUT_FORMAT=$(args::get "output-format")
}

#######################################
# Display usage information
#######################################
claude_code::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Actions:"
    echo "  install      Install Claude Code CLI globally"
    echo "  uninstall    Remove Claude Code CLI"
    echo "  status       Check installation status"
    echo "  info         Display detailed information"
    echo "  run          Run Claude with a prompt"
    echo "  batch        Run Claude in batch mode"
    echo "  session      Manage Claude sessions"
    echo "  settings     View or update Claude settings"
    echo "  logs         View Claude logs"
    echo "  help         Show this help message"
    echo
    echo "Examples:"
    echo "  $0 --action install                    # Install Claude Code"
    echo "  $0 --action status                     # Check installation status"
    echo "  $0 --action run --prompt \"Explain this code\"  # Run a prompt"
    echo "  $0 --action batch --prompt \"Write tests\" --max-turns 10"
    echo "  $0 --action session --session-id abc123  # Resume session"
    echo "  $0 --action settings                   # View current settings"
    echo "  $0 --action uninstall                  # Remove Claude Code"
    echo
    echo "Requirements:"
    echo "  - Node.js version $MIN_NODE_VERSION or newer"
    echo "  - npm package manager"
    echo "  - Valid Claude Pro or Max subscription for full functionality"
}

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
# Install Claude Code
#######################################
claude_code::install() {
    log::header "📦 Installing Claude Code"
    
    # Check if already installed
    if claude_code::is_installed && [[ "$FORCE" != "yes" ]]; then
        local version
        version=$(claude_code::get_version)
        log::warn "Claude Code is already installed (version: $version)"
        log::info "Use --force yes to reinstall"
        return 0
    fi
    
    # Check Node.js requirements
    log::info "Checking prerequisites..."
    if ! claude_code::check_node_version; then
        log::error "Node.js $MIN_NODE_VERSION or newer is required"
        log::info "Please install Node.js from https://nodejs.org/"
        return 1
    fi
    
    local node_version
    node_version=$(node --version)
    log::success "✓ Node.js $node_version detected"
    
    # Check npm
    if ! system::is_command npm; then
        log::error "npm is required but not found"
        log::info "npm should be installed with Node.js"
        return 1
    fi
    
    local npm_version
    npm_version=$(npm --version)
    log::success "✓ npm $npm_version detected"
    
    # Install Claude Code globally
    log::info "Installing Claude Code globally..."
    if npm install -g "$CLAUDE_PACKAGE"; then
        log::success "✓ Claude Code installed successfully"
    else
        log::error "Failed to install Claude Code"
        return 1
    fi
    
    # Verify installation
    if claude_code::is_installed; then
        local version
        version=$(claude_code::get_version)
        log::success "✓ Claude Code $version is ready to use"
        
        # Update resource configuration
        if resources::update_config "agents" "$RESOURCE_NAME" "claude-cli" \
            '{"type":"cli","command":"claude","requiresAuth":true}'; then
            log::success "✓ Resource configuration updated"
        else
            log::warn "⚠️  Failed to update resource configuration"
        fi
        
        # Show next steps
        echo
        log::header "🎯 Next Steps"
        log::info "1. Navigate to your project directory: cd /path/to/project"
        log::info "2. Start Claude Code: claude"
        log::info "3. Login with your Claude credentials when prompted"
        log::info "4. Use 'claude doctor' to verify your installation"
        
        return 0
    else
        log::error "Installation verification failed"
        return 1
    fi
}

#######################################
# Uninstall Claude Code
#######################################
claude_code::uninstall() {
    log::header "🗑️  Uninstalling Claude Code"
    
    if ! claude_code::is_installed; then
        log::warn "Claude Code is not installed"
        return 0
    fi
    
    # Confirm uninstallation
    if ! confirm "Remove Claude Code CLI?"; then
        log::info "Uninstallation cancelled"
        return 0
    fi
    
    # Uninstall globally
    log::info "Removing Claude Code..."
    if npm uninstall -g "$CLAUDE_PACKAGE"; then
        log::success "✓ Claude Code removed successfully"
    else
        log::error "Failed to uninstall Claude Code"
        return 1
    fi
    
    # Update configuration
    if resources::remove_from_config "agents" "$RESOURCE_NAME"; then
        log::success "✓ Resource configuration updated"
    fi
    
    return 0
}

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
    
    echo "Claude Code is Anthropic's official CLI for Claude."
    echo
    echo "Key Features:"
    echo "  • Deep codebase awareness"
    echo "  • Direct file editing and command execution"
    echo "  • Claude Opus 4 model integration"
    echo "  • Composable and scriptable interface"
    echo "  • Automatic updates"
    echo
    echo "Requirements:"
    echo "  • Node.js $MIN_NODE_VERSION or newer"
    echo "  • npm package manager"
    echo "  • Claude Pro or Max subscription"
    echo
    echo "Subscription Plans:"
    echo "  • Pro (\$20/month): Light coding on small repositories"
    echo "  • Max (\$100/month): Heavy usage, ~50-200 prompts per 5 hours"
    echo
    echo "Usage:"
    echo "  1. Install: npm install -g $CLAUDE_PACKAGE"
    echo "  2. Navigate to project: cd /your/project"
    echo "  3. Start Claude: claude"
    echo
    echo "Documentation: https://docs.anthropic.com/claude-code"
    echo "Support: https://support.anthropic.com"
    
    # Show current status
    echo
    claude_code::status
}

#######################################
# Run Claude with a single prompt
#######################################
claude_code::run() {
    log::header "🤖 Running Claude Code"
    
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi
    
    if [[ -z "$PROMPT" ]]; then
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
    if [[ -n "$ALLOWED_TOOLS" ]]; then
        IFS=',' read -ra TOOLS <<< "$ALLOWED_TOOLS"
        for tool in "${TOOLS[@]}"; do
            cmd="$cmd --allowedTools \"$tool\""
        done
    fi
    
    # Set timeout environment variables
    export BASH_DEFAULT_TIMEOUT_MS=$((TIMEOUT * 1000))
    export BASH_MAX_TIMEOUT_MS=$((TIMEOUT * 1000))
    export MCP_TOOL_TIMEOUT=$((TIMEOUT * 1000))
    
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
    
    if [[ -z "$PROMPT" ]]; then
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
    if [[ -n "$ALLOWED_TOOLS" ]]; then
        IFS=',' read -ra TOOLS <<< "$ALLOWED_TOOLS"
        for tool in "${TOOLS[@]}"; do
            cmd="$cmd --allowedTools \"$tool\""
        done
    fi
    
    # Set extended timeouts for batch operations
    export BASH_DEFAULT_TIMEOUT_MS=$((TIMEOUT * 1000))
    export BASH_MAX_TIMEOUT_MS=$((TIMEOUT * 1000))
    export MCP_TOOL_TIMEOUT=$((TIMEOUT * 1000))
    
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

#######################################
# Manage Claude sessions
#######################################
claude_code::session() {
    log::header "🔄 Managing Claude Sessions"
    
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi
    
    if [[ -z "$SESSION_ID" ]]; then
        # List sessions
        log::info "Listing recent sessions..."
        
        # Check for session files
        local session_dir="$HOME/.claude/sessions"
        if [[ -d "$session_dir" ]]; then
            local sessions
            sessions=$(ls -t "$session_dir" 2>/dev/null | head -10)
            if [[ -n "$sessions" ]]; then
                echo "Recent sessions:"
                echo "$sessions" | nl
            else
                log::info "No sessions found"
            fi
        else
            log::info "No session directory found"
        fi
        
        echo
        log::info "To resume a session: $0 --action session --session-id <id>"
    else
        # Resume specific session
        log::info "Resuming session: $SESSION_ID"
        
        local cmd="claude --resume \"$SESSION_ID\""
        
        # Add max turns if specified
        if [[ "$MAX_TURNS" != "5" ]]; then
            cmd="$cmd --max-turns $MAX_TURNS"
        fi
        
        log::info "Executing: $cmd"
        eval "$cmd"
    fi
}

#######################################
# View or update Claude settings
#######################################
claude_code::settings() {
    log::header "⚙️  Claude Code Settings"
    
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi
    
    local settings_file="$HOME/.claude/settings.json"
    local project_settings="$(pwd)/.claude/settings.json"
    
    # Check for settings files
    if [[ -f "$project_settings" ]]; then
        log::info "Project settings found: $project_settings"
        echo
        log::info "Current project settings:"
        cat "$project_settings" | jq . 2>/dev/null || cat "$project_settings"
        echo
    fi
    
    if [[ -f "$settings_file" ]]; then
        log::info "Global settings found: $settings_file"
        echo
        log::info "Current global settings:"
        cat "$settings_file" | jq . 2>/dev/null || cat "$settings_file"
    else
        log::warn "No global settings file found"
        log::info "Settings will be created when you first run Claude"
    fi
    
    echo
    log::info "Tips:"
    log::info "  • Project settings override global settings"
    log::info "  • Use .claude/settings.json for project-specific configuration"
    log::info "  • Set environment variables in settings for tool timeouts"
    log::info "  • Configure allowed tools for security"
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
    
    # Common log locations
    local log_locations=(
        "$HOME/.claude/logs"
        "$HOME/.cache/claude/logs"
        "$HOME/Library/Logs/claude"  # macOS
    )
    
    local found_logs=false
    
    for log_dir in "${log_locations[@]}"; do
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

#######################################
# Main execution function
#######################################
claude_code::main() {
    claude_code::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            claude_code::install
            ;;
        "uninstall")
            claude_code::uninstall
            ;;
        "status")
            claude_code::status
            ;;
        "info")
            claude_code::info
            ;;
        "run")
            claude_code::run
            ;;
        "batch")
            claude_code::batch
            ;;
        "session")
            claude_code::session
            ;;
        "settings")
            claude_code::settings
            ;;
        "logs")
            claude_code::logs
            ;;
        "help"|*)
            claude_code::usage
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    claude_code::main "$@"
fi