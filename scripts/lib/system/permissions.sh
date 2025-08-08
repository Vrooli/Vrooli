#!/usr/bin/env bash
################################################################################
# System Permissions Utilities
# 
# Handles permission checks and modifications for system setup
################################################################################

LIB_SYSTEM_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "$LIB_SYSTEM_DIR/../utils/var.sh"

#######################################
# Check if running with sudo when needed
# Returns:
#   0 if has proper permissions, 1 otherwise
#######################################
permissions::check_sudo() {
    if [[ $EUID -eq 0 ]]; then
        return 0
    fi
    
    # Check if we can sudo
    if sudo -n true 2>/dev/null; then
        return 0
    fi
    
    return 1
}

#######################################
# Request sudo if needed
# Arguments:
#   $1 - reason for sudo (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
permissions::request_sudo() {
    local reason="${1:-system configuration}"
    
    if permissions::check_sudo; then
        return 0
    fi
    
    echo "🔐 Administrator access needed for: $reason"
    if sudo -v; then
        return 0
    fi
    
    return 1
}

#######################################
# Ensure directory permissions
# Arguments:
#   $1 - directory path
#   $2 - permissions (default: 755)
#   $3 - owner (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
permissions::ensure_directory() {
    local dir="$1"
    local perms="${2:-755}"
    local owner="${3:-}"
    
    # Create directory if it doesn't exist
    if [[ ! -d "$dir" ]]; then
        mkdir -p "$dir" || return 1
    fi
    
    # Set permissions
    chmod "$perms" "$dir" || return 1
    
    # Set owner if specified
    if [[ -n "$owner" ]]; then
        if permissions::check_sudo; then
            sudo chown "$owner" "$dir" || return 1
        else
            echo "Warning: Cannot set owner without sudo access"
        fi
    fi
    
    return 0
}

#######################################
# Ensure file permissions
# Arguments:
#   $1 - file path
#   $2 - permissions (default: 644)
#   $3 - owner (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
permissions::ensure_file() {
    local file="$1"
    local perms="${2:-644}"
    local owner="${3:-}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Set permissions
    chmod "$perms" "$file" || return 1
    
    # Set owner if specified
    if [[ -n "$owner" ]]; then
        if permissions::check_sudo; then
            sudo chown "$owner" "$file" || return 1
        else
            echo "Warning: Cannot set owner without sudo access"
        fi
    fi
    
    return 0
}

#######################################
# Check write permission for path
# Arguments:
#   $1 - path to check
# Returns:
#   0 if writable, 1 otherwise
#######################################
permissions::can_write() {
    local path="$1"
    
    if [[ -e "$path" ]]; then
        [[ -w "$path" ]]
    else
        # Check parent directory
        local parent
        parent=$(dirname "$path")
        [[ -w "$parent" ]]
    fi
}

# Export functions
export -f permissions::check_sudo
export -f permissions::request_sudo
export -f permissions::ensure_directory
export -f permissions::ensure_file
export -f permissions::can_write