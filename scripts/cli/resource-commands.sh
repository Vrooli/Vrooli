#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Resource Management Commands
# 
# Handles resource operations including listing, status checking, and installation.
#
# Usage:
#   vrooli resource <subcommand> [options]
#
################################################################################

set -euo pipefail

# Get CLI directory
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Resource paths
RESOURCES_DIR="${var_SCRIPTS_RESOURCES_DIR}"
RESOURCES_CONFIG="${var_ROOT_DIR}/.vrooli/resources.local.json"

# Show help for resource commands
show_resource_help() {
    cat << EOF
🚀 Vrooli Resource Commands

USAGE:
    vrooli resource <subcommand> [options]

SUBCOMMANDS:
    list                    List all available resources
    status [name]           Show status of resources (or specific resource)
    install <name>          Install a specific resource
    enable <name>           Enable resource in configuration
    disable <name>          Disable resource in configuration
    info <name>             Show detailed information about a resource

OPTIONS:
    --help, -h              Show this help message
    --verbose, -v           Show detailed output

EXAMPLES:
    vrooli resource list                  # List all resources
    vrooli resource status                 # Check status of all resources
    vrooli resource status postgres        # Check PostgreSQL status
    vrooli resource install ollama         # Install Ollama
    vrooli resource enable windmill        # Enable Windmill in config

For more information: https://docs.vrooli.com/cli/resources
EOF
}

# List all available resources
resource_list() {
    log::header "Available Resources"
    
    echo ""
    echo "Configuration: ${RESOURCES_CONFIG:-No local config}"
    echo ""
    
    # Get list of resource directories
    local categories=(
        "agents:AI Agents & Browsers"
        "ai:AI Models & Inference"
        "automation:Workflow Automation"
        "search:Search Engines"
        "storage:Databases & Storage"
        "monitoring:Monitoring & Analytics"
        "communication:Chat & Communication"
    )
    
    for category_info in "${categories[@]}"; do
        local category="${category_info%%:*}"
        local description="${category_info#*:}"
        local category_dir="${RESOURCES_DIR}/${category}"
        
        if [[ ! -d "$category_dir" ]]; then
            continue
        fi
        
        echo ""
        echo "$description"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        for resource_dir in "$category_dir"/*; do
            [[ ! -d "$resource_dir" ]] && continue
            
            local resource_name
            resource_name=$(basename "$resource_dir")
            local status="❓ Unknown"
            local enabled="❌"
            
            # Check if resource is enabled in config
            if [[ -f "$RESOURCES_CONFIG" ]]; then
                local is_enabled
                is_enabled=$(jq -r ".resources.$category.$resource_name.enabled // false" "$RESOURCES_CONFIG" 2>/dev/null)
                [[ "$is_enabled" == "true" ]] && enabled="✅"
            fi
            
            # Check if resource is installed
            if command -v "$resource_name" >/dev/null 2>&1; then
                status="🟢 Installed"
            elif [[ -f "$resource_dir/lib/common.sh" ]]; then
                # Source resource's common.sh if available for status check
                if source "$resource_dir/lib/common.sh" 2>/dev/null; then
                    if type -t "${resource_name}::is_installed" >/dev/null; then
                        if "${resource_name}::is_installed" 2>/dev/null; then
                            status="🟢 Installed"
                        else
                            status="⭕ Not Installed"
                        fi
                    fi
                fi
            fi
            
            printf "  %-20s %s %s\n" "$resource_name" "$enabled" "$status"
        done
    done
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Legend: ✅ Enabled | ❌ Disabled | 🟢 Installed | ⭕ Not Installed"
}

# Show status of resources
resource_status() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        # Show status of all enabled resources
        log::header "Resource Status"
        
        if [[ ! -f "$RESOURCES_CONFIG" ]]; then
            log::warning "No resource configuration found"
            echo "Resources are configured in: $RESOURCES_CONFIG"
            return 0
        fi
        
        echo ""
        echo "Checking enabled resources..."
        echo ""
        
        # Parse enabled resources from config
        local enabled_resources
        enabled_resources=$(jq -r '
            .resources | to_entries[] as $category | 
            $category.value | to_entries[] | 
            select(.value.enabled == true) | 
            "\($category.key)/\(.key)"
        ' "$RESOURCES_CONFIG" 2>/dev/null)
        
        if [[ -z "$enabled_resources" ]]; then
            log::info "No resources are enabled"
            return 0
        fi
        
        while IFS= read -r resource_path; do
            local category="${resource_path%%/*}"
            local name="${resource_path#*/}"
            resource_status_single "$name" "$category"
            echo ""
        done <<< "$enabled_resources"
    else
        # Show status of specific resource
        # Find the resource category
        local category=""
        for cat_dir in "$RESOURCES_DIR"/*; do
            if [[ -d "$cat_dir/$resource_name" ]]; then
                category=$(basename "$cat_dir")
                break
            fi
        done
        
        if [[ -z "$category" ]]; then
            log::error "Resource not found: $resource_name"
            return 1
        fi
        
        resource_status_single "$resource_name" "$category"
    fi
}

# Show status of a single resource
resource_status_single() {
    local name="$1"
    local category="$2"
    local resource_dir="${RESOURCES_DIR}/${category}/${name}"
    
    echo "📦 Resource: $name ($category)"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Check if enabled
    local is_enabled="false"
    if [[ -f "$RESOURCES_CONFIG" ]]; then
        is_enabled=$(jq -r ".resources.$category.$name.enabled // false" "$RESOURCES_CONFIG" 2>/dev/null)
    fi
    
    echo "Configuration: $([ "$is_enabled" == "true" ] && echo "✅ Enabled" || echo "❌ Disabled")"
    
    # Check installation status
    local is_installed=false
    local version="Unknown"
    
    # Try common command check
    if command -v "$name" >/dev/null 2>&1; then
        is_installed=true
        version=$("$name" --version 2>&1 | head -1 || echo "Unknown")
    fi
    
    # Try resource-specific check
    if [[ -f "$resource_dir/lib/common.sh" ]]; then
        if source "$resource_dir/lib/common.sh" 2>/dev/null; then
            if type -t "${name}::is_installed" >/dev/null; then
                if "${name}::is_installed" 2>/dev/null; then
                    is_installed=true
                fi
            fi
            
            if type -t "${name}::get_version" >/dev/null; then
                local resource_version
                resource_version=$("${name}::get_version" 2>/dev/null || echo "")
                [[ -n "$resource_version" ]] && version="$resource_version"
            fi
        fi
    fi
    
    echo "Installation: $([ "$is_installed" == "true" ] && echo "🟢 Installed" || echo "⭕ Not Installed")"
    [[ "$is_installed" == "true" ]] && echo "Version: $version"
    
    # Check if running (for services)
    if [[ "$is_installed" == "true" ]]; then
        local is_running=false
        local status_msg="Not a service"
        
        # Docker-based resources
        if docker ps --format "table {{.Names}}" 2>/dev/null | grep -q "$name"; then
            is_running=true
            status_msg="🟢 Running (Docker)"
        elif systemctl is-active "$name" >/dev/null 2>&1; then
            is_running=true
            status_msg="🟢 Running (systemd)"
        elif pgrep -f "$name" >/dev/null 2>&1; then
            is_running=true
            status_msg="🟢 Running (process)"
        elif [[ -f "$resource_dir/lib/common.sh" ]]; then
            if source "$resource_dir/lib/common.sh" 2>/dev/null; then
                if type -t "${name}::is_running" >/dev/null; then
                    if "${name}::is_running" 2>/dev/null; then
                        is_running=true
                        status_msg="🟢 Running"
                    else
                        status_msg="🔴 Stopped"
                    fi
                fi
            fi
        fi
        
        [[ "$status_msg" != "Not a service" ]] && echo "Status: $status_msg"
    fi
    
    # Show resource directory
    [[ "${VERBOSE:-false}" == "true" ]] && echo "Location: $resource_dir"
}

# Install a resource
resource_install() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        log::error "Resource name required"
        echo "Usage: vrooli resource install <resource-name>"
        return 1
    fi
    
    # Find the resource
    local category=""
    local resource_dir=""
    for cat_dir in "$RESOURCES_DIR"/*; do
        if [[ -d "$cat_dir/$resource_name" ]]; then
            category=$(basename "$cat_dir")
            resource_dir="$cat_dir/$resource_name"
            break
        fi
    done
    
    if [[ -z "$category" ]]; then
        log::error "Resource not found: $resource_name"
        echo "Run 'vrooli resource list' to see available resources"
        return 1
    fi
    
    log::info "Installing resource: $resource_name ($category)"
    
    # Check if resource has an install script
    local install_script="$resource_dir/lib/install.sh"
    if [[ -f "$install_script" ]]; then
        # Source and run install function
        source "$install_script"
        if type -t "${resource_name}::install" >/dev/null; then
            "${resource_name}::install"
        else
            log::error "No install function found for $resource_name"
            return 1
        fi
    else
        # Try using the main setup script with specific resource
        log::info "Using setup script to install $resource_name..."
        "${var_ROOT_DIR}/scripts/manage.sh" setup --resources "$resource_name"
    fi
}

# Enable resource in configuration
resource_enable() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        log::error "Resource name required"
        echo "Usage: vrooli resource enable <resource-name>"
        return 1
    fi
    
    # Find the resource category
    local category=""
    for cat_dir in "$RESOURCES_DIR"/*; do
        if [[ -d "$cat_dir/$resource_name" ]]; then
            category=$(basename "$cat_dir")
            break
        fi
    done
    
    if [[ -z "$category" ]]; then
        log::error "Resource not found: $resource_name"
        return 1
    fi
    
    # Ensure config file exists
    if [[ ! -f "$RESOURCES_CONFIG" ]]; then
        echo '{"resources": {}}' > "$RESOURCES_CONFIG"
    fi
    
    # Update configuration
    local temp_file="${RESOURCES_CONFIG}.tmp"
    if jq ".resources.$category.$resource_name.enabled = true" "$RESOURCES_CONFIG" > "$temp_file"; then
        mv "$temp_file" "$RESOURCES_CONFIG"
        log::success "✅ Resource enabled: $resource_name"
        echo "Run 'vrooli resource install $resource_name' to install it"
    else
        log::error "Failed to update configuration"
        rm -f "$temp_file"
        return 1
    fi
}

# Disable resource in configuration
resource_disable() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        log::error "Resource name required"
        echo "Usage: vrooli resource disable <resource-name>"
        return 1
    fi
    
    # Find the resource category
    local category=""
    for cat_dir in "$RESOURCES_DIR"/*; do
        if [[ -d "$cat_dir/$resource_name" ]]; then
            category=$(basename "$cat_dir")
            break
        fi
    done
    
    if [[ -z "$category" ]]; then
        log::error "Resource not found: $resource_name"
        return 1
    fi
    
    # Ensure config file exists
    if [[ ! -f "$RESOURCES_CONFIG" ]]; then
        echo '{"resources": {}}' > "$RESOURCES_CONFIG"
    fi
    
    # Update configuration
    local temp_file="${RESOURCES_CONFIG}.tmp"
    if jq ".resources.$category.$resource_name.enabled = false" "$RESOURCES_CONFIG" > "$temp_file"; then
        mv "$temp_file" "$RESOURCES_CONFIG"
        log::success "✅ Resource disabled: $resource_name"
        echo "This resource will not be installed during setup"
    else
        log::error "Failed to update configuration"
        rm -f "$temp_file"
        return 1
    fi
}

# Show detailed info about a resource
resource_info() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        log::error "Resource name required"
        echo "Usage: vrooli resource info <resource-name>"
        return 1
    fi
    
    # Find the resource
    local category=""
    local resource_dir=""
    for cat_dir in "$RESOURCES_DIR"/*; do
        if [[ -d "$cat_dir/$resource_name" ]]; then
            category=$(basename "$cat_dir")
            resource_dir="$cat_dir/$resource_name"
            break
        fi
    done
    
    if [[ -z "$category" ]]; then
        log::error "Resource not found: $resource_name"
        return 1
    fi
    
    log::header "Resource: $resource_name"
    echo ""
    echo "📁 Category: $category"
    echo "📍 Location: $resource_dir"
    
    # Show README if available
    if [[ -f "$resource_dir/README.md" ]]; then
        echo ""
        echo "Documentation:"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        head -20 "$resource_dir/README.md" | sed 's/^#//'
        echo ""
        echo "... (see $resource_dir/README.md for full documentation)"
    fi
    
    # Show available scripts
    echo ""
    echo "Available Scripts:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    for script in "$resource_dir"/lib/*.sh; do
        [[ -f "$script" ]] && echo "  • $(basename "$script")"
    done
    
    # Show configuration from resources.local.json
    if [[ -f "$RESOURCES_CONFIG" ]]; then
        echo ""
        echo "Current Configuration:"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        jq ".resources.$category.$resource_name // {}" "$RESOURCES_CONFIG" 2>/dev/null
    fi
}

# Main command handler
main() {
    # Check for verbose flag
    VERBOSE=false
    for arg in "$@"; do
        if [[ "$arg" == "--verbose" ]] || [[ "$arg" == "-v" ]]; then
            VERBOSE=true
        fi
    done
    
    if [[ $# -eq 0 ]] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        show_resource_help
        return 0
    fi
    
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        list)
            resource_list "$@"
            ;;
        status)
            resource_status "$@"
            ;;
        install)
            resource_install "$@"
            ;;
        enable)
            resource_enable "$@"
            ;;
        disable)
            resource_disable "$@"
            ;;
        info)
            resource_info "$@"
            ;;
        *)
            log::error "Unknown resource command: $subcommand"
            echo ""
            show_resource_help
            return 1
            ;;
    esac
}

# Execute main function
main "$@"