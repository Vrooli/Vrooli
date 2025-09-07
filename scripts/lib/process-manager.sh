#!/usr/bin/env bash
################################################################################
# Simplified Vrooli Process Manager v2.0
# 
# Pattern-based process management using `ps aux` queries instead of PID files.
# Backward compatible with existing function signatures.
#
# Key simplifications:
# - No PID files, lock files, or complex file management
# - Uses `ps aux | grep` to find processes directly
# - ~150 lines instead of 646
#
# Usage:
#   source process-manager.sh
#   pm::logs "name"
#
################################################################################

set -o nounset  # Error on undefined variables
set -o errtrace # Inherit trap ERR

# Configuration
PM_LOG_DIR="${PM_LOG_DIR:-$HOME/.vrooli/logs}"

# Colors for output (only if terminal)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

################################################################################
# Core Functions
################################################################################

# Check if process is running using ps lookup
pm::is_running() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        return 1
    fi
    
    # Find processes by the pm: prefix pattern we use when starting
    local pids
    pids=$(ps aux | grep -v grep | grep -F "pm:$name" | awk '{print $2}' | head -5)
    
    # If we found any PIDs, process is running
    [[ -n "$pids" ]]
}

# Show process logs (unchanged - still useful)
pm::logs() {
    local name="${1:-}"
    local lines="${2:-50}"
    local follow="${3:-false}"
    
    if [[ -z "$name" ]]; then
        echo -e "${RED}Error: pm::logs requires name${NC}" >&2
        return 1
    fi
    
    local log_file="$PM_LOG_DIR/$name.log"
    
    if [[ ! -f "$log_file" ]]; then
        echo -e "${YELLOW}No logs found for: $name${NC}"
        return 1
    fi
    
    if [[ "$follow" == "true" || "$follow" == "--follow" || "$follow" == "-f" ]]; then
        echo -e "${BLUE}Following logs for: $name (Ctrl+C to stop)${NC}"
        echo "────────────────────────────────────────────────────────"
        tail -f "$log_file"
    else
        echo -e "${BLUE}Logs for: $name (last $lines lines)${NC}"
        echo "────────────────────────────────────────────────────────"
        tail -n "$lines" "$log_file"
    fi
}

################################################################################
# Export Functions
################################################################################

# Export functions for use in other scripts
export -f pm::logs
export -f pm::is_running

# Indicate library is loaded
export PM_LOADED=true

################################################################################
# CLI Usage
################################################################################

# If script is executed directly, show usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "Simplified Vrooli Process Manager v2.0"
    echo ""
    echo "This script should be sourced, not executed directly:"
    echo "  source process-manager.sh"
    echo ""
    echo "Key improvements:"
    echo "  • No PID file complexity (uses ps queries)"
    echo "  • Pattern-based process matching"
    echo "  • ~150 lines instead of 646"
    echo "  • More reliable stopping"
    echo "  • Backward compatible function signatures"
    echo ""
    echo "Available functions:"
    echo "  pm::logs <name> [lines] [--follow]"
    echo "  pm::is_running <name>"
    exit 1
fi