#!/bin/bash
# Home Assistant Custom Components Management

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HOME_ASSISTANT_COMPONENTS_DIR="${APP_ROOT}/resources/home-assistant/lib"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${HOME_ASSISTANT_COMPONENTS_DIR}/core.sh"
source "${HOME_ASSISTANT_COMPONENTS_DIR}/health.sh"

#######################################
# Install HACS (Home Assistant Community Store)
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::components::install_hacs() {
    log::info "Installing HACS (Home Assistant Community Store)..."
    
    # Initialize environment
    home_assistant::init
    
    # Check if Home Assistant is running
    if ! home_assistant::health::is_healthy; then
        log::error "Home Assistant must be running to install HACS"
        return 1
    fi
    
    # Create custom_components directory
    local custom_components_dir="$HOME_ASSISTANT_CONFIG_DIR/custom_components"
    mkdir -p "$custom_components_dir"
    
    # Check if HACS is already installed
    if [[ -d "$custom_components_dir/hacs" ]]; then
        log::warning "HACS appears to be already installed"
        return 0
    fi
    
    # Download HACS
    local hacs_url="https://github.com/hacs/integration/releases/latest/download/hacs.zip"
    local temp_file="/tmp/hacs_$(date +%s).zip"
    
    log::info "Downloading HACS from GitHub..."
    if curl -sfL "$hacs_url" -o "$temp_file"; then
        # Extract to custom_components directory
        if unzip -q "$temp_file" -d "$custom_components_dir"; then
            rm -f "$temp_file"
            log::success "HACS installed successfully"
            log::info "Restart Home Assistant to complete installation"
            log::info "Then configure HACS through the UI at Settings > Devices & Services"
            return 0
        else
            log::error "Failed to extract HACS"
            rm -f "$temp_file"
            return 1
        fi
    else
        log::error "Failed to download HACS"
        return 1
    fi
}

#######################################
# List installed custom components
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::components::list() {
    # Initialize environment
    home_assistant::init
    
    local custom_components_dir="$HOME_ASSISTANT_CONFIG_DIR/custom_components"
    
    if [[ ! -d "$custom_components_dir" ]]; then
        log::info "No custom components directory found"
        return 0
    fi
    
    log::header "Installed Custom Components"
    
    local count=0
    for component in "$custom_components_dir"/*; do
        if [[ -d "$component" ]]; then
            local component_name=$(basename "$component")
            local manifest_file="$component/manifest.json"
            
            if [[ -f "$manifest_file" ]]; then
                # Try to extract version from manifest
                local version=$(jq -r '.version // "unknown"' "$manifest_file" 2>/dev/null || echo "unknown")
                local name=$(jq -r '.name // ""' "$manifest_file" 2>/dev/null || echo "$component_name")
                
                echo "  - $name ($component_name) - v$version"
                ((count++))
            else
                echo "  - $component_name (no manifest)"
                ((count++))
            fi
        fi
    done
    
    if [[ $count -eq 0 ]]; then
        log::info "No custom components installed"
    else
        log::success "Found $count custom component(s)"
    fi
    
    return 0
}

#######################################
# Install a custom component from GitHub
# Arguments:
#   repo_url: GitHub repository URL
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::components::install_from_github() {
    local repo_url="${1:-}"
    
    if [[ -z "$repo_url" ]]; then
        log::error "No repository URL provided"
        log::info "Usage: content add-component <github-url>"
        return 1
    fi
    
    # Initialize environment
    home_assistant::init
    
    # Extract repo info from URL
    local repo_name=$(basename "$repo_url" .git)
    local custom_components_dir="$HOME_ASSISTANT_CONFIG_DIR/custom_components"
    mkdir -p "$custom_components_dir"
    
    log::info "Installing custom component from: $repo_url"
    
    # Clone to temp directory
    local temp_dir="/tmp/ha_component_$(date +%s)"
    if git clone --depth 1 "$repo_url" "$temp_dir" 2>/dev/null; then
        # Look for custom_components directory in repo
        if [[ -d "$temp_dir/custom_components" ]]; then
            # Copy all components from the repo
            cp -r "$temp_dir/custom_components/"* "$custom_components_dir/"
            log::success "Installed component(s) from $repo_name"
        elif [[ -f "$temp_dir/manifest.json" ]]; then
            # Single component in root
            cp -r "$temp_dir" "$custom_components_dir/$repo_name"
            log::success "Installed component: $repo_name"
        else
            log::error "No valid Home Assistant component structure found in repository"
            rm -rf "$temp_dir"
            return 1
        fi
        
        rm -rf "$temp_dir"
        log::info "Restart Home Assistant to load the new component"
        return 0
    else
        log::error "Failed to clone repository: $repo_url"
        return 1
    fi
}

#######################################
# Remove a custom component
# Arguments:
#   component_name: Name of the component to remove
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::components::remove() {
    local component_name="${1:-}"
    
    if [[ -z "$component_name" ]]; then
        log::error "No component name provided"
        log::info "Usage: content remove-component <component-name>"
        return 1
    fi
    
    # Initialize environment
    home_assistant::init
    
    local custom_components_dir="$HOME_ASSISTANT_CONFIG_DIR/custom_components"
    local component_path="$custom_components_dir/$component_name"
    
    if [[ ! -d "$component_path" ]]; then
        log::error "Component not found: $component_name"
        home_assistant::components::list
        return 1
    fi
    
    log::info "Removing custom component: $component_name"
    if rm -rf "$component_path"; then
        log::success "Component removed: $component_name"
        log::info "Restart Home Assistant to complete removal"
        return 0
    else
        log::error "Failed to remove component: $component_name"
        return 1
    fi
}