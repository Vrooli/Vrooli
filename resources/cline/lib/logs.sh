#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
CLINE_LOGS_DIR="${APP_ROOT}/resources/cline/lib"

# Source common functions
source "$CLINE_LOGS_DIR/common.sh"

# Show Cline logs
cline::logs() {
    local lines="${1:-50}"
    
    log::info "Cline logs are managed by VS Code"
    
    # Check for VS Code extension logs
    local vscode_log_dir="${HOME}/.config/Code/logs"
    if [[ -d "$vscode_log_dir" ]]; then
        log::info "VS Code logs directory: $vscode_log_dir"
        
        # Look for extension host logs
        local latest_log=$(find "$vscode_log_dir" -name "*exthost*.log" -type f 2>/dev/null | sort -r | head -1)
        if [[ -n "$latest_log" ]]; then
            log::info "Latest extension host log: $latest_log"
            echo "---"
            tail -n "$lines" "$latest_log" | grep -i "cline\|claude-dev" || echo "No Cline-specific logs found"
        fi
    fi
    
    # Check for any local logs
    if [[ -f "$CLINE_CONFIG_DIR/activity.log" ]]; then
        log::info "Cline activity log:"
        echo "---"
        tail -n "$lines" "$CLINE_CONFIG_DIR/activity.log"
    else
        log::info "No Cline activity logs found"
        log::info "Logs will appear here when Cline is used in VS Code"
    fi
    
    return 0
}

# Main - only run if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cline::logs "$@"
fi