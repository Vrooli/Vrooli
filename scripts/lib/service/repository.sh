#!/usr/bin/env bash
# repository.sh - Repository management helper functions
# Centralized repository operations using service.json configuration
set -euo pipefail

LIB_SERVICE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source core utilities
# shellcheck disable=SC1091
source "${LIB_SERVICE_DIR}/../utils/exit_codes.sh"
# shellcheck disable=SC1091
source "${LIB_SERVICE_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${LIB_SERVICE_DIR}/../system/system_commands.sh"
# shellcheck disable=SC1091
source "${LIB_SERVICE_DIR}/../utils/var.sh"

# Path to service.json file
SERVICE_JSON_PATH="${SERVICE_JSON_PATH:-${var_SERVICE_JSON_FILE:-$(cd "${LIB_SERVICE_DIR}/../../.." && pwd)/.vrooli/service.json}}"

#######################################
# Read repository configuration from service.json
# Returns: Repository config as JSON string
#######################################
repository::read_config() {
    local config_path="${1:-$SERVICE_JSON_PATH}"
    
    if [[ ! -f "$config_path" ]]; then
        log::error "Service configuration not found: $config_path"
        return 1
    fi
    
    # Extract repository section from service.json
    if system::is_command "jq"; then
        jq -r '.service.repository // empty' "$config_path"
    elif system::is_command "yq"; then
        yq eval '.service.repository' "$config_path" -o json
    else
        log::error "Neither jq nor yq found. Please install one to read service.json"
        return 1
    fi
}

#######################################
# Get repository URL (primary or fallback)
# Returns: Repository URL string
#######################################
repository::get_url() {
    local config
    config=$(repository::read_config) || return 1
    
    if [[ -z "$config" ]]; then
        log::error "No repository configuration found"
        return 1
    fi
    
    # Try primary URL first
    local url
    if system::is_command "jq"; then
        url=$(echo "$config" | jq -r '.url // empty')
    else
        url=$(echo "$config" | yq eval '.url' -)
    fi
    
    if [[ -n "$url" ]]; then
        echo "$url"
        return 0
    fi
    
    log::error "No repository URL found in configuration"
    return 1
}

#######################################
# Get SSH URL for HTTPS fallback
# Returns: SSH URL string if available
#######################################
repository::get_ssh_url() {
    local config
    config=$(repository::read_config) || return 1
    
    local ssh_url
    if system::is_command "jq"; then
        ssh_url=$(echo "$config" | jq -r '.sshUrl // empty')
    else
        ssh_url=$(echo "$config" | yq eval '.sshUrl // ""' -)
    fi
    
    if [[ -n "$ssh_url" ]]; then
        echo "$ssh_url"
        return 0
    fi
    
    # If no SSH URL configured, try to derive from HTTPS URL
    local https_url
    https_url=$(repository::get_url) || return 1
    
    if [[ "$https_url" =~ https://github\.com/(.+)/(.+)(\.git)?$ ]]; then
        echo "git@github.com:${BASH_REMATCH[1]}/${BASH_REMATCH[2]}.git"
        return 0
    fi
    
    return 1
}

#######################################
# Get default branch from configuration
# Returns: Branch name (defaults to 'main')
#######################################
repository::get_branch() {
    local config
    config=$(repository::read_config) || return 1
    
    local branch
    if system::is_command "jq"; then
        branch=$(echo "$config" | jq -r '.branch // "main"')
    else
        branch=$(echo "$config" | yq eval '.branch // "main"' -)
    fi
    
    echo "${branch:-main}"
}

#######################################
# Get mirror URLs from configuration
# Returns: Space-separated list of mirror URLs
#######################################
repository::get_mirrors() {
    local config
    config=$(repository::read_config) || return 1
    
    local mirrors
    if system::is_command "jq"; then
        mirrors=$(echo "$config" | jq -r '.mirrors[]? // empty' | tr '\n' ' ')
    else
        mirrors=$(echo "$config" | yq eval '.mirrors[]' - 2>/dev/null | tr '\n' ' ')
    fi
    
    echo "$mirrors"
}

#######################################
# Check if repository is private
# Returns: 0 if private, 1 if public
#######################################
repository::is_private() {
    local config
    config=$(repository::read_config) || return 1
    
    local private
    if system::is_command "jq"; then
        private=$(echo "$config" | jq -r '.private // false')
    else
        private=$(echo "$config" | yq eval '.private // false' -)
    fi
    
    [[ "$private" == "true" ]] && return 0 || return 1
}

#######################################
# Get submodules configuration
# Returns: JSON object or boolean
#######################################
repository::get_submodules_config() {
    local config
    config=$(repository::read_config) || return 1
    
    if system::is_command "jq"; then
        echo "$config" | jq -r '.submodules // false'
    else
        echo "$config" | yq eval '.submodules // false' - -o json
    fi
}

#######################################
# Run repository hook if defined
# Arguments:
#   $1 - Hook name (postClone, preBuild, postUpdate)
# Returns: 0 on success or if no hook defined
#######################################
repository::run_hook() {
    local hook_name="${1:-}"
    
    if [[ -z "$hook_name" ]]; then
        log::error "Hook name required"
        return 1
    fi
    
    local config
    config=$(repository::read_config) || return 1
    
    local hook_command
    if system::is_command "jq"; then
        hook_command=$(echo "$config" | jq -r ".hooks.${hook_name} // empty")
    else
        hook_command=$(echo "$config" | yq eval ".hooks.${hook_name} // \"\"" -)
    fi
    
    if [[ -z "$hook_command" ]]; then
        log::debug "No ${hook_name} hook defined"
        return 0
    fi
    
    log::info "Running ${hook_name} hook: ${hook_command}"
    
    # Execute the hook command
    if eval "$hook_command"; then
        log::success "${hook_name} hook completed successfully"
        return 0
    else
        log::error "${hook_name} hook failed"
        return 1
    fi
}

#######################################
# Clone repository with proper configuration
# Arguments:
#   $1 - Target directory (optional)
# Returns: 0 on success
#######################################
repository::clone() {
    local target_dir="${1:-}"
    local url
    url=$(repository::get_url) || return 1
    
    local branch
    branch=$(repository::get_branch)
    
    # Check if we should use SSH URL (for HTTPS blocking)
    if ! curl -Is "$url" >/dev/null 2>&1; then
        log::warning "HTTPS URL not accessible, trying SSH"
        local ssh_url
        if ssh_url=$(repository::get_ssh_url); then
            url="$ssh_url"
        fi
    fi
    
    log::info "Cloning repository from $url (branch: $branch)"
    
    # Build clone command
    local clone_cmd="git clone"
    
    # Add branch if specified
    if [[ -n "$branch" && "$branch" != "main" && "$branch" != "master" ]]; then
        clone_cmd="$clone_cmd -b $branch"
    fi
    
    # Handle submodules
    local submodules_config
    submodules_config=$(repository::get_submodules_config)
    
    if [[ "$submodules_config" == "true" ]]; then
        clone_cmd="$clone_cmd --recurse-submodules"
    elif [[ "$submodules_config" != "false" ]]; then
        # Complex submodules configuration
        local recursive shallow
        if system::is_command "jq"; then
            recursive=$(echo "$submodules_config" | jq -r '.recursive // true')
            shallow=$(echo "$submodules_config" | jq -r '.shallow // false')
        else
            recursive=$(echo "$submodules_config" | yq eval '.recursive // true' -)
            shallow=$(echo "$submodules_config" | yq eval '.shallow // false' -)
        fi
        
        if [[ "$recursive" == "true" ]]; then
            clone_cmd="$clone_cmd --recurse-submodules"
        fi
        
        if [[ "$shallow" == "true" ]]; then
            clone_cmd="$clone_cmd --shallow-submodules"
        fi
    fi
    
    # Add URL and target directory
    clone_cmd="$clone_cmd $url"
    if [[ -n "$target_dir" ]]; then
        clone_cmd="$clone_cmd $target_dir"
    fi
    
    # Execute clone
    if eval "$clone_cmd"; then
        log::success "Repository cloned successfully"
        
        # Change to cloned directory if specified
        if [[ -n "$target_dir" ]]; then
            cd "$target_dir" || return 1
        fi
        
        # Run postClone hook
        repository::run_hook "postClone"
        
        return 0
    else
        log::error "Failed to clone repository"
        
        # Try mirrors if available
        local mirrors
        mirrors=$(repository::get_mirrors)
        
        if [[ -n "$mirrors" ]]; then
            log::info "Trying mirror repositories..."
            for mirror in $mirrors; do
                log::info "Trying mirror: $mirror"
                clone_cmd="${clone_cmd/$url/$mirror}"
                if eval "$clone_cmd"; then
                    log::success "Repository cloned from mirror: $mirror"
                    
                    if [[ -n "$target_dir" ]]; then
                        cd "$target_dir" || return 1
                    fi
                    
                    repository::run_hook "postClone"
                    return 0
                fi
            done
        fi
        
        return 1
    fi
}

#######################################
# Update repository (pull latest changes)
# Arguments:
#   $1 - Repository directory (optional, defaults to current)
# Returns: 0 on success
#######################################
repository::update() {
    local repo_dir="${1:-.}"
    
    # Save current directory
    local original_dir
    original_dir=$(pwd)
    
    # Change to repository directory
    cd "$repo_dir" || return 1
    
    # Check if it's a git repository
    if [[ ! -d .git ]]; then
        log::error "Not a git repository: $repo_dir"
        cd "$original_dir" || return 1
        return 1
    fi
    
    local branch
    branch=$(repository::get_branch)
    
    log::info "Updating repository (branch: $branch)"
    
    # Fetch latest changes
    if ! git fetch origin; then
        log::error "Failed to fetch from origin"
        cd "$original_dir" || return 1
        return 1
    fi
    
    # Pull changes
    if git pull origin "$branch"; then
        log::success "Repository updated successfully"
        
        # Handle submodules if configured
        local submodules_config
        submodules_config=$(repository::get_submodules_config)
        
        if [[ "$submodules_config" != "false" ]]; then
            log::info "Updating submodules..."
            git submodule update --init --recursive
        fi
        
        # Run postUpdate hook
        repository::run_hook "postUpdate"
        
        cd "$original_dir" || return 1
        return 0
    else
        log::error "Failed to update repository"
        cd "$original_dir" || return 1
        return 1
    fi
}

#######################################
# Check repository access
# Returns: 0 if accessible, 1 otherwise
#######################################
repository::check_access() {
    local url
    url=$(repository::get_url) || return 1
    
    log::info "Checking repository access: $url"
    
    # Try to access the repository
    if git ls-remote "$url" >/dev/null 2>&1; then
        log::success "Repository is accessible"
        return 0
    else
        log::warning "Primary repository not accessible, checking mirrors..."
        
        # Try mirrors
        local mirrors
        mirrors=$(repository::get_mirrors)
        
        for mirror in $mirrors; do
            if git ls-remote "$mirror" >/dev/null 2>&1; then
                log::success "Mirror accessible: $mirror"
                return 0
            fi
        done
        
        # Try SSH as last resort
        local ssh_url
        if ssh_url=$(repository::get_ssh_url); then
            if git ls-remote "$ssh_url" >/dev/null 2>&1; then
                log::success "Repository accessible via SSH"
                return 0
            fi
        fi
        
        log::error "Repository not accessible via any method"
        return 1
    fi
}

#######################################
# Get available mirror URL
# Returns: First accessible mirror URL
#######################################
repository::get_mirror() {
    local mirrors
    mirrors=$(repository::get_mirrors)
    
    if [[ -z "$mirrors" ]]; then
        log::debug "No mirrors configured"
        return 1
    fi
    
    for mirror in $mirrors; do
        if git ls-remote "$mirror" >/dev/null 2>&1; then
            echo "$mirror"
            return 0
        fi
    done
    
    log::warning "No accessible mirrors found"
    return 1
}

#######################################
# Display repository information
# Returns: 0 on success
#######################################
repository::info() {
    local config
    config=$(repository::read_config) || return 1
    
    log::header "Repository Information"
    
    local url branch private ssh_url
    
    if system::is_command "jq"; then
        url=$(echo "$config" | jq -r '.url // "Not configured"')
        branch=$(echo "$config" | jq -r '.branch // "main"')
        private=$(echo "$config" | jq -r '.private // false')
        ssh_url=$(echo "$config" | jq -r '.sshUrl // "Not configured"')
    else
        url=$(echo "$config" | yq eval '.url // "Not configured"' -)
        branch=$(echo "$config" | yq eval '.branch // "main"' -)
        private=$(echo "$config" | yq eval '.private // false' -)
        ssh_url=$(echo "$config" | yq eval '.sshUrl // "Not configured"' -)
    fi
    
    echo "URL: $url"
    echo "SSH URL: $ssh_url"
    echo "Branch: $branch"
    echo "Private: $private"
    
    local mirrors
    mirrors=$(repository::get_mirrors)
    if [[ -n "$mirrors" ]]; then
        echo "Mirrors:"
        for mirror in $mirrors; do
            echo "  - $mirror"
        done
    fi
    
    # Display hooks if configured
    local hooks
    if system::is_command "jq"; then
        hooks=$(echo "$config" | jq -r '.hooks // {}')
    else
        hooks=$(echo "$config" | yq eval '.hooks // {}' - -o json)
    fi
    
    if [[ "$hooks" != "{}" && "$hooks" != "null" ]]; then
        echo "Hooks:"
        if system::is_command "jq"; then
            echo "$hooks" | jq -r 'to_entries[] | "  \(.key): \(.value)"'
        else
            echo "$hooks" | yq eval 'to_entries[] | "  " + .key + ": " + .value' -
        fi
    fi
    
    return 0
}