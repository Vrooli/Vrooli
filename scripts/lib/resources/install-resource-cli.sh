#!/usr/bin/env bash
################################################################################
# Universal Resource CLI Installer
# 
# Installs CLI for any resource that has a cli.sh file.
# Reads metadata from capabilities.yaml if available.
#
# Usage:
#   install-resource-cli.sh <resource-path>
#   install-resource-cli.sh <category>/<resource-name>
#   install-resource-cli.sh --all  # Install all resource CLIs
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/lib/resources"
VROOLI_ROOT="$APP_ROOT"

# Source utilities
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || {
    # Fallback logging
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
}

# Installation configuration
INSTALL_DIR="${HOME}/.local/bin"
REGISTRY_DIR="${VROOLI_ROOT}/.vrooli/resource-registry"

################################################################################
# Helper Functions
################################################################################

# Find installation directory
find_install_dir() {
    local dirs=(
        "$HOME/.local/bin"
        "$HOME/bin"
        "/usr/local/bin"
    )
    
    for dir in "${dirs[@]}"; do
        if [[ -d "$dir" ]] && echo "$PATH" | grep -q "$dir"; then
            if [[ -w "$dir" ]]; then
                echo "$dir"
                return 0
            elif [[ "$dir" == "/usr/local/bin" ]]; then
                echo "$dir"
                return 0
            fi
        fi
    done
    
    # Create ~/.local/bin if needed
    mkdir -p "$HOME/.local/bin"
    echo "$HOME/.local/bin"
}

# Extract metadata from capabilities.yaml
get_resource_metadata() {
    local resource_dir="$1"
    local resource_name="$2"
    local category="$3"
    
    local capabilities_file="${resource_dir}/config/capabilities.yaml"
    local description="Resource: ${resource_name}"
    local version="1.0.0"
    
    # Try to read from capabilities.yaml if it exists
    if [[ -f "$capabilities_file" ]]; then
        # Try to extract description
        local yaml_desc
        yaml_desc=$(grep -E "^description:|^  description:" "$capabilities_file" 2>/dev/null | head -1 | sed 's/.*description: *//' | tr -d '"' || true)
        [[ -n "$yaml_desc" ]] && description="$yaml_desc"
        
        # Try to extract version
        local yaml_version
        yaml_version=$(grep -E "^version:|^  version:" "$capabilities_file" 2>/dev/null | head -1 | sed 's/.*version: *//' | tr -d '"' || true)
        [[ -n "$yaml_version" ]] && version="$yaml_version"
    fi
    
    # Output as JSON
    cat << EOF
{
    "name": "${resource_name}",
    "command": "resource-${resource_name}",
    "category": "${category}",
    "description": "${description}",
    "version": "${version}"
}
EOF
}

# Install CLI for a single resource
install_resource_cli() {
    local resource_path="$1"
    
    # Resolve resource directory
    local resource_dir
    if [[ -d "$resource_path" ]]; then
        resource_dir="$resource_path"
    elif [[ -d "${VROOLI_ROOT}/resources/${resource_path}" ]]; then
        resource_dir="${VROOLI_ROOT}/resources/${resource_path}"
    else
        log::error "Resource not found: $resource_path"
        return 1
    fi
    
    # Check if CLI exists  
    local cli_script="$(cd "$resource_dir" && pwd)/cli.sh"
    # Make path relative to VROOLI_ROOT for registry storage
    local relative_cli_path="${cli_script#$VROOLI_ROOT/}"
    if [[ ! -f "$cli_script" ]]; then
        log::info "No CLI found for $(basename "$resource_dir")"
        return 0
    fi
    
    # Extract resource info
    local resource_name
    resource_name=$(basename "$resource_dir")
    local category
    category=$(basename "${resource_dir%/*}")
    local command_name="resource-${resource_name}"
    
    log::info "Installing CLI for ${resource_name}..."
    
    # Get installation directory
    local install_dir
    install_dir=$(find_install_dir)
    local target="${install_dir}/${command_name}"
    
    # Make CLI script executable
    chmod +x "$cli_script"
    
    # Create symlink
    if [[ ! -w "$install_dir" ]] && [[ "$install_dir" == "/usr/local/bin" ]]; then
        sudo ln -sf "$cli_script" "$target"
    else
        ln -sf "$cli_script" "$target"
    fi
    
    # Verify installation
    if [[ -L "$target" ]] || [[ -f "$target" ]]; then
        log::success "${command_name} installed to ${install_dir}"
    else
        log::error "Failed to install ${command_name}"
        return 1
    fi
    
    # Register with Vrooli CLI
    mkdir -p "$REGISTRY_DIR"
    
    # Get metadata
    local metadata
    metadata=$(get_resource_metadata "$resource_dir" "$resource_name" "$category")
    
    # Add installation info to metadata
    local registry_file="${REGISTRY_DIR}/${resource_name}.json"
    echo "$metadata" | jq \
        --arg path "$relative_cli_path" \
        --arg installed "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        '. + {path: $path, installed: $installed}' > "$registry_file"
    
    log::success "Registered ${resource_name} with Vrooli CLI"
    
    # Check if command is in PATH
    if command -v "$command_name" >/dev/null 2>&1; then
        log::success "${command_name} is available in PATH"
    else
        log::warning "${command_name} installed but not in PATH"
        log::info "Add ${install_dir} to your PATH if needed"
    fi
}

# Install all resource CLIs
install_all_resource_clis() {
    log::info "Installing all resource CLIs..."
    
    local count=0
    local resources_dir="${VROOLI_ROOT}/resources"
    
    # Find all cli.sh files
    while IFS= read -r cli_file; do
        local resource_dir
        resource_dir=${cli_file%/*}
        
        if install_resource_cli "$resource_dir"; then
            ((count++))
        fi
    done < <(find "$resources_dir" -name "cli.sh" -type f 2>/dev/null)
    
    log::success "Installed ${count} resource CLIs"
}

# Uninstall a resource CLI
uninstall_resource_cli() {
    local resource_name="$1"
    local command_name="resource-${resource_name}"
    
    log::info "Uninstalling ${command_name}..."
    
    # Remove symlink
    for dir in "$HOME/.local/bin" "$HOME/bin" "/usr/local/bin"; do
        local target="${dir}/${command_name}"
        if [[ -L "$target" ]] || [[ -f "$target" ]]; then
            if [[ -w "$dir" ]]; then
                rm -f "$target"
            else
                sudo rm -f "$target"
            fi
            log::success "Removed ${target}"
        fi
    done
    
    # Remove registry entry
    local registry_file="${REGISTRY_DIR}/${resource_name}.json"
    if [[ -f "$registry_file" ]]; then
        rm -f "$registry_file"
        log::success "Removed registry entry"
    fi
}

################################################################################
# Main
################################################################################

main() {
    local action="${1:-}"
    
    case "$action" in
        --all|-a)
            install_all_resource_clis
            ;;
        --uninstall|-u)
            shift
            local resource="${1:-}"
            if [[ -z "$resource" ]]; then
                log::error "Resource name required for uninstall"
                exit 1
            fi
            uninstall_resource_cli "$resource"
            ;;
        --help|-h|"")
            cat << EOF
Universal Resource CLI Installer

USAGE:
    $(basename "$0") <resource-path>           # Install single resource CLI
    $(basename "$0") <category>/<name>         # Install by category/name
    $(basename "$0") --all                     # Install all resource CLIs
    $(basename "$0") --uninstall <name>        # Uninstall a resource CLI
    $(basename "$0") --help                    # Show this help

EXAMPLES:
    $(basename "$0") automation/ollama
    $(basename "$0") /path/to/resource/dir
    $(basename "$0") --all

Resources with CLIs will have their commands installed as 'resource-<name>'
EOF
            ;;
        *)
            install_resource_cli "$action"
            ;;
    esac
}

main "$@"
