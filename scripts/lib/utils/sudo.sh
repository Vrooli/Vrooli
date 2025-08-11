#!/usr/bin/env bash
################################################################################
# Sudo and User Permission Utilities
# 
# Centralized functions for handling sudo operations and permission management
# Provides consistent patterns for:
# - Detecting sudo execution
# - Getting actual user information
# - Restoring file ownership
# - Running commands as the original user
################################################################################

set -euo pipefail

# Source dependencies
UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "$UTILS_DIR/log.sh"

################################################################################
# Detection Functions
################################################################################

#######################################
# Check if script is running under sudo
# Returns:
#   0 if running under sudo, 1 otherwise
#######################################
sudo::is_running_as_sudo() {
    [[ -n "${SUDO_USER:-}" ]] && [[ "$SUDO_USER" != "root" ]]
}

#######################################
# Check if running as root (with or without sudo)
# Returns:
#   0 if running as root, 1 otherwise
#######################################
sudo::is_root() {
    [[ $EUID -eq 0 ]]
}

#######################################
# Check if sudo is available and can be used
# Returns:
#   0 if sudo can be used, 1 otherwise
#######################################
sudo::can_use_sudo() {
    # Check if sudo command exists
    if ! command -v sudo >/dev/null 2>&1; then
        return 1
    fi
    
    # If running as root, don't need sudo
    if sudo::is_root; then
        return 0
    fi
    
    # Check if user has sudo privileges (with timeout to avoid hanging)
    sudo -n true 2>/dev/null || sudo -v 2>/dev/null
}

#######################################
# Execute command with sudo, falling back to direct execution if sudo unavailable
# Arguments:
#   $1 - Command string to execute
# Returns:
#   Command exit code
#######################################
sudo::exec_with_fallback() {
    local cmd="$1"
    
    if sudo::can_use_sudo; then
        sudo bash -c "$cmd"
    else
        bash -c "$cmd"
    fi
}

################################################################################
# User Information Functions
################################################################################

#######################################
# Get the actual user (original user if sudo, current user otherwise)
# Outputs:
#   Username
#######################################
sudo::get_actual_user() {
    echo "${SUDO_USER:-$USER}"
}

#######################################
# Get the actual user's primary group
# Outputs:
#   Group name
#######################################
sudo::get_actual_group() {
    local user
    user=$(sudo::get_actual_user)
    id -gn "$user"
}

#######################################
# Get the actual user's home directory
# Outputs:
#   Home directory path
#######################################
sudo::get_actual_home() {
    local user
    user=$(sudo::get_actual_user)
    if [[ "$user" == "$USER" ]]; then
        echo "$HOME"
    else
        eval echo "~${user}"
    fi
}

#######################################
# Get user:group string for chown
# Outputs:
#   user:group string
#######################################
sudo::get_owner_string() {
    local user
    user=$(sudo::get_actual_user)
    echo "${user}:$(id -gn "$user")"
}

################################################################################
# Ownership Management Functions
################################################################################

#######################################
# Restore file/directory ownership to actual user
# Arguments:
#   $1 - Path to file or directory
#   $2 - Recursive flag (optional, "recursive" or "r")
# Returns:
#   0 on success, 1 on failure
#######################################
sudo::restore_owner() {
    local path="$1"
    local recursive="${2:-}"
    
    # Only restore if running under sudo
    if ! sudo::is_running_as_sudo; then
        return 0
    fi
    
    local owner
    owner=$(sudo::get_owner_string)
    
    if [[ "$recursive" == "recursive" ]] || [[ "$recursive" == "r" ]]; then
        chown -R "$owner" "$path"
        log::debug "Restored ownership of $path (recursive) to $owner"
    else
        chown "$owner" "$path"
        log::debug "Restored ownership of $path to $owner"
    fi
}

#######################################
# Create directory with proper ownership
# Arguments:
#   $1 - Directory path
#   $2 - Permissions (optional, default: 755)
# Returns:
#   0 on success, 1 on failure
#######################################
sudo::mkdir_as_user() {
    local dir="$1"
    local perms="${2:-755}"
    
    if sudo::is_running_as_sudo; then
        local user
        user=$(sudo::get_actual_user)
        sudo -u "$user" mkdir -p "$dir"
        chmod "$perms" "$dir"
    else
        mkdir -p "$dir"
        chmod "$perms" "$dir"
    fi
}

#######################################
# Write file with proper ownership
# Arguments:
#   $1 - File path
#   $2 - Content (via stdin if not provided)
# Returns:
#   0 on success, 1 on failure
#######################################
sudo::write_file_as_user() {
    local file="$1"
    local content="${2:-}"
    
    if [[ -n "$content" ]]; then
        echo "$content" > "$file"
    else
        cat > "$file"
    fi
    
    sudo::restore_owner "$file"
}

################################################################################
# Command Execution Functions
################################################################################

#######################################
# Run command as actual user
# Arguments:
#   $@ - Command and arguments
# Returns:
#   Command exit code
#######################################
sudo::run_as_actual_user() {
    if sudo::is_running_as_sudo; then
        local user
        user=$(sudo::get_actual_user)
        sudo -u "$user" "$@"
    else
        "$@"
    fi
}

#######################################
# Run bash command as actual user
# Arguments:
#   $1 - Bash command string
# Returns:
#   Command exit code
#######################################
sudo::exec_as_actual_user() {
    local cmd="$1"
    
    if sudo::is_running_as_sudo; then
        local user
        user=$(sudo::get_actual_user)
        sudo -u "$user" bash -c "$cmd"
    else
        bash -c "$cmd"
    fi
}

################################################################################
# Git Operations (with proper user context)
################################################################################

#######################################
# Clone git repository as actual user
# Arguments:
#   $1 - Repository URL
#   $2 - Target directory (optional)
#   $3 - Additional git options (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
sudo::git_clone_as_user() {
    local repo_url="$1"
    local target_dir="${2:-}"
    local git_opts="${3:-}"
    
    local cmd="git"
    
    # Add TLS compatibility for common issues
    cmd="$cmd -c http.sslVersion=tlsv1.2"
    
    # Add any additional options
    if [[ -n "$git_opts" ]]; then
        cmd="$cmd $git_opts"
    fi
    
    # Add clone command
    cmd="$cmd clone '$repo_url'"
    
    # Add target directory if specified
    if [[ -n "$target_dir" ]]; then
        cmd="$cmd '$target_dir'"
    fi
    
    # Execute as actual user
    sudo::exec_as_actual_user "$cmd"
}

################################################################################
# Environment Setup Functions
################################################################################

#######################################
# Set up environment variables for actual user
# Arguments:
#   $1 - Variable name
#   $2 - Variable value
# Returns:
#   Exports variable in appropriate context
#######################################
sudo::export_for_user() {
    local var_name="$1"
    local var_value="$2"
    
    if sudo::is_running_as_sudo; then
        # Export for both current context and user context
        export "$var_name=$var_value"
        sudo::exec_as_actual_user "export $var_name='$var_value'"
    else
        export "$var_name=$var_value"
    fi
}

################################################################################
# Validation Functions
################################################################################

#######################################
# Check if actual user has access to path
# Arguments:
#   $1 - Path to check
#   $2 - Access type (r/w/x, default: r)
# Returns:
#   0 if accessible, 1 otherwise
#######################################
sudo::user_can_access() {
    local path="$1"
    local access_type="${2:-r}"
    
    sudo::run_as_actual_user test "-$access_type" "$path"
}

#######################################
# Check if actual user is in group
# Arguments:
#   $1 - Group name
# Returns:
#   0 if in group, 1 otherwise
#######################################
sudo::user_in_group() {
    local group="$1"
    local user
    user=$(sudo::get_actual_user)
    
    groups "$user" 2>/dev/null | grep -q "\b$group\b"
}

################################################################################
# Cleanup Functions
################################################################################

#######################################
# Fix ownership of all files created during sudo session
# Useful at the end of scripts that create multiple files
# Arguments:
#   $@ - Paths to fix (if empty, fixes common paths)
# Returns:
#   0 on success
#######################################
sudo::fix_created_files() {
    if ! sudo::is_running_as_sudo; then
        return 0
    fi
    
    local paths=("$@")
    
    # If no paths specified, use common locations
    if [[ ${#paths[@]} -eq 0 ]]; then
        paths=(
            "${HOME}/.npm"
            "${HOME}/.pnpm"
            "${HOME}/.cache"
            "${HOME}/.config"
            "${HOME}/.local"
            "${HOME}/generated-apps"
            "${PWD}/data"
            "${PWD}/node_modules"
            "${PWD}/.next"
            "${PWD}/dist"
            "${PWD}/build"
        )
    fi
    
    local owner
    owner=$(sudo::get_owner_string)
    
    for path in "${paths[@]}"; do
        if [[ -e "$path" ]]; then
            log::debug "Fixing ownership of $path"
            chown -R "$owner" "$path" 2>/dev/null || true
        fi
    done
}

################################################################################
# Export Functions
################################################################################

# Detection
export -f sudo::is_running_as_sudo
export -f sudo::is_root
export -f sudo::can_use_sudo
export -f sudo::exec_with_fallback

# User Information
export -f sudo::get_actual_user
export -f sudo::get_actual_group
export -f sudo::get_actual_home
export -f sudo::get_owner_string

# Ownership Management
export -f sudo::restore_owner
export -f sudo::mkdir_as_user
export -f sudo::write_file_as_user

# Command Execution
export -f sudo::run_as_actual_user
export -f sudo::exec_as_actual_user

# Git Operations
export -f sudo::git_clone_as_user

# Environment Setup
export -f sudo::export_for_user

# Validation
export -f sudo::user_can_access
export -f sudo::user_in_group

# Cleanup
export -f sudo::fix_created_files

# If this script is run directly, show usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cat << 'EOF'
Sudo Utilities Library

This library provides consistent functions for handling sudo operations.

Key Functions:
- sudo::is_running_as_sudo    - Check if running under sudo
- sudo::get_actual_user        - Get the actual username
- sudo::restore_owner          - Restore file ownership to actual user
- sudo::run_as_actual_user     - Run commands as the actual user
- sudo::git_clone_as_user      - Clone repos with proper ownership

Example Usage:
    # Restore ownership after creating a file
    echo "data" > /tmp/myfile
    sudo::restore_owner /tmp/myfile
    
    # Run command as actual user
    sudo::run_as_actual_user npm install
    
    # Create directory with proper ownership
    sudo::mkdir_as_user "$HOME/.config/myapp"

For more details, see the function documentation in this file.
EOF
fi