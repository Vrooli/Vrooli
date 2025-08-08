#!/usr/bin/env bash
################################################################################
# Helm Installation and Management
# 
# Handles Helm package manager installation for Kubernetes
################################################################################

LIB_SYSTEM_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "$LIB_SYSTEM_DIR/../utils/var.sh"

# shellcheck disable=SC1091
source "$var_LIB_UTILS_DIR/log.sh"

#######################################
# Check if Helm is installed
# Returns:
#   0 if installed, 1 otherwise
#######################################
helm::is_installed() {
    command -v helm &> /dev/null
}

#######################################
# Get installed Helm version
# Returns:
#   Version string or empty
#######################################
helm::get_version() {
    if helm::is_installed; then
        helm version --short 2>/dev/null | grep -oP 'v[\d.]+'
    fi
}

#######################################
# Install Helm
# Arguments:
#   $1 - version (optional, default: latest)
# Returns:
#   0 on success, 1 on failure
#######################################
helm::install() {
    local version="${1:-latest}"
    
    if helm::is_installed; then
        log::info "Helm is already installed ($(helm::get_version))"
        return 0
    fi
    
    log::info "Installing Helm ${version}..."
    
    # Download installation script
    local install_script="/tmp/get_helm.sh"
    if ! curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 -o "$install_script"; then
        log::error "Failed to download Helm installation script"
        return 1
    fi
    
    chmod +x "$install_script"
    
    # Install Helm
    if [[ "$version" == "latest" ]]; then
        bash "$install_script"
    else
        bash "$install_script" --version "$version"
    fi
    
    local result=$?
    rm -f "$install_script"
    
    if [[ $result -eq 0 ]] && helm::is_installed; then
        log::success "Helm installed successfully"
        return 0
    else
        log::error "Failed to install Helm"
        return 1
    fi
}

#######################################
# Add a Helm repository
# Arguments:
#   $1 - repository name
#   $2 - repository URL
# Returns:
#   0 on success, 1 on failure
#######################################
helm::add_repo() {
    local name="$1"
    local url="$2"
    
    if ! helm::is_installed; then
        log::error "Helm is not installed"
        return 1
    fi
    
    log::info "Adding Helm repository: $name"
    helm repo add "$name" "$url" && helm repo update
}

#######################################
# Check if Helm repository exists
# Arguments:
#   $1 - repository name
# Returns:
#   0 if exists, 1 otherwise
#######################################
helm::repo_exists() {
    local name="$1"
    
    if ! helm::is_installed; then
        return 1
    fi
    
    helm repo list 2>/dev/null | grep -q "^$name"
}

# Export functions
export -f helm::is_installed
export -f helm::get_version
export -f helm::install
export -f helm::add_repo
export -f helm::repo_exists