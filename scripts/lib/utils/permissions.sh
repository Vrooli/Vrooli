#!/usr/bin/env bash
################################################################################
# Unified Permissions Utilities
# 
# Consolidated from system/permissions.sh and service/permissions.sh
# Handles all permission-related operations
################################################################################

set -euo pipefail

# Source dependencies
UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "$UTILS_DIR/var.sh"
source "$UTILS_DIR/log.sh"

################################################################################
# System Permission Functions (from system/permissions.sh)
################################################################################

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

################################################################################
# Script Permission Functions (from service/permissions.sh)
################################################################################

#######################################
# Makes all scripts in a directory (recursively) executable
# Arguments:
#   $1 - directory path
#######################################
permissions::make_files_in_dir_executable() {
    log::header "Making scripts in ${1} executable"
    if [ -d "${1}" ]; then
        local count files_output
        files_output=$(find -- "${1}" -type f \( -name '*.sh' -o -name '*.bats' \) 2>&1 || true)
        
        # Show permission errors to user if any
        if echo "$files_output" | grep -q "Permission denied"; then
            echo "$files_output" | grep "Permission denied" || true
        fi
        
        # Filter out permission errors and empty lines, then count actual files
        local clean_files
        if [ -n "$files_output" ]; then
            clean_files=$(echo "$files_output" | grep -v "Permission denied" | grep -v '^$' || true)
        else
            clean_files=""
        fi
        
        # Count actual files (not empty lines)
        if [ -n "$clean_files" ]; then
            count=$(echo "$clean_files" | wc -l | xargs)
        else
            count=0
        fi
        
        if [ "${count}" -gt 0 ]; then
            # Process only the accessible files
            echo "$clean_files" | while IFS= read -r file; do
                [ -n "$file" ] && chmod a+x "$file"
            done
            log::success "Made ${count} script(s) in ${1} executable"
        else
            log::info "No scripts found in ${1}"
        fi
    else
        log::error "Directory not found: ${1}"
        exit 1
    fi
}

#######################################
# Makes every script executable
#######################################
permissions::make_scripts_executable() {
    # Ensure required directories are defined
    : "${var_SCRIPTS_DIR:?var_SCRIPTS_DIR must be set}"
    permissions::make_files_in_dir_executable "$var_SCRIPTS_DIR"
    
    # Only process postgres entrypoint if it exists (for monorepo mode)
    if [[ -n "${var_POSTGRES_ENTRYPOINT_DIR:-}" ]] && [[ -d "$var_POSTGRES_ENTRYPOINT_DIR" ]]; then
        permissions::make_files_in_dir_executable "$var_POSTGRES_ENTRYPOINT_DIR"
    fi
}

# Export functions
export -f permissions::check_sudo
export -f permissions::request_sudo
export -f permissions::ensure_directory
export -f permissions::ensure_file
export -f permissions::can_write
export -f permissions::make_files_in_dir_executable
export -f permissions::make_scripts_executable

# If this script is run directly, apply permissions to all defined directories.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    permissions::make_scripts_executable
fi