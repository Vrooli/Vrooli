#!/usr/bin/env bash
################################################################################
# Resource Auto-Installation Module
# 
# Automatically installs resources marked as enabled in service.json
# Handles both main Vrooli setup and generated app setup
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
RESOURCE_AUTO_DIR="${APP_ROOT}/scripts/lib/resources"
VROOLI_ROOT="$APP_ROOT"

# Source dependencies
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_LIB_UTILS_DIR}/flow.sh"

# Registry file for tracking installed resources
RESOURCE_REGISTRY="${VROOLI_ROOT}/.vrooli/running-resources.json"

################################################################################
# Resource Registry Functions
################################################################################

#######################################
# Initialize resource registry
#######################################
resource_registry::init() {
    local registry_dir="${RESOURCE_REGISTRY%/*}"
    
    if [[ ! -d "$registry_dir" ]]; then
        mkdir -p "$registry_dir"
    fi
    
    if [[ ! -f "$RESOURCE_REGISTRY" ]]; then
        echo '{"resources": [], "last_updated": ""}' > "$RESOURCE_REGISTRY"
    fi
}

#######################################
# Register a resource as installed/running
#######################################
resource_registry::register() {
    local resource_name="$1"
    local status="${2:-installed}"  # installed, running, stopped
    
    resource_registry::init
    
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    # Check if resource already registered
    local exists=$(jq -r --arg name "$resource_name" '.resources[] | select(.name == $name) | .name' "$RESOURCE_REGISTRY" 2>/dev/null || echo "")
    
    if [[ -n "$exists" ]]; then
        # Update existing entry
        jq --arg name "$resource_name" \
           --arg status "$status" \
           --arg timestamp "$timestamp" \
           '.resources |= map(if .name == $name then .status = $status | .last_updated = $timestamp else . end) | .last_updated = $timestamp' \
           "$RESOURCE_REGISTRY" > "${RESOURCE_REGISTRY}.tmp"
    else
        # Add new entry
        jq --arg name "$resource_name" \
           --arg status "$status" \
           --arg timestamp "$timestamp" \
           '.resources += [{"name": $name, "status": $status, "installed": $timestamp, "last_updated": $timestamp}] | .last_updated = $timestamp' \
           "$RESOURCE_REGISTRY" > "${RESOURCE_REGISTRY}.tmp"
    fi
    
    mv "${RESOURCE_REGISTRY}.tmp" "$RESOURCE_REGISTRY"
    log::debug "Registered resource: $resource_name ($status)"
}

#######################################
# Get list of registered resources
#######################################
resource_registry::list() {
    local filter_status="${1:-}"
    
    resource_registry::init
    
    if [[ -n "$filter_status" ]]; then
        jq -r --arg status "$filter_status" '.resources[] | select(.status == $status) | .name' "$RESOURCE_REGISTRY" 2>/dev/null || true
    else
        jq -r '.resources[].name' "$RESOURCE_REGISTRY" 2>/dev/null || true
    fi
}

################################################################################
# Resource Installation Functions
################################################################################

#######################################
# Validate if a resource is actually functional
# Arguments:
#   $1 - resource name
# Returns:
#   0 if functional, 1 if not
#######################################
resource_auto::validate_resource() {
    local resource_name="$1"
    
    # Check if CLI exists
    if ! command -v "resource-${resource_name}" &>/dev/null; then
        return 1
    fi
    
    # Try to get status - if it succeeds, the resource is functional
    # Use --fast mode for quick health check
    if "resource-${resource_name}" status --fast >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

#######################################
# Install a single resource
#######################################
resource_auto::install_resource() {
    local resource_name="$1"
    local resource_config="$2"
    
    log::info "Installing resource: $resource_name"
    
    # Check if resource is already installed AND functional
    if command -v "resource-${resource_name}" &>/dev/null; then
        log::debug "Resource CLI available: resource-${resource_name}"
        
        # Verify the resource is actually functional by checking its status
        # Use --fast mode for quick health check
        if "resource-${resource_name}" status --fast >/dev/null 2>&1; then
            log::debug "Resource is functional: resource-${resource_name}"
            resource_registry::register "$resource_name" "installed"
            return 0
        else
            log::warning "Resource CLI exists but resource is not functional: resource-${resource_name}"
            log::info "Will reinstall to fix broken installation"
        fi
    fi
    
    # Find resource directory
    local resource_dir=""
    local possible_locations=(
        "${VROOLI_ROOT}/resources/${resource_name}"
    )
    
    for pattern in "${possible_locations[@]}"; do
        for dir in $pattern; do
            if [[ -d "$dir" && -f "$dir/cli.sh" ]]; then
                resource_dir="$dir"
                break 2
            fi
        done
    done
    
    if [[ -z "$resource_dir" ]]; then
        log::warning "Resource directory not found for: $resource_name"
        return 1
    fi
    
    log::debug "Found resource at: $resource_dir"
    
    # Check if resource has a CLI interface
    if [[ -f "$resource_dir/cli.sh" ]]; then
        # Run installation
        log::info "Running installation for $resource_name..."
        
        # Build installation command - use verbose for better feedback
        local install_cmd="$resource_dir/cli.sh install --verbose"
        
        # Execute installation
        if eval "$install_cmd"; then
            log::success "✅ Resource installed: $resource_name"
            resource_registry::register "$resource_name" "installed"
            
            # Note: CLI installation is handled by the resource's install script via resource-cli-helper.sh
            log::debug "Resource installation completed (CLI handled internally)"
            
            return 0
        else
            log::error "Failed to install resource: $resource_name"
            return 1
        fi
    else
        log::warning "No CLI interface found for: $resource_name"
        return 1
    fi
}

#######################################
# Start a resource if it has a start action
#######################################
resource_auto::start_resource() {
    local resource_name="$1"
    local resource_config="$2"
    
    # Check if auto-start is enabled (default: true for enabled resources)
    local auto_start=$(echo "$resource_config" | jq -r '.auto_start // true' 2>/dev/null)
    
    if [[ "$auto_start" != "true" ]]; then
        log::debug "Auto-start disabled for: $resource_name"
        return 0
    fi
    
    # Check if resource is already running first
    # Try using CLI status check
    if command -v "resource-${resource_name}" &>/dev/null; then
        # Use --fast mode for quick health check
        if "resource-${resource_name}" status --fast >/dev/null 2>&1; then
            log::info "Resource already running: $resource_name"
            resource_registry::register "$resource_name" "running"
            return 0
        fi
    fi
    
    # Validate resource is functional before attempting to start
    if ! resource_auto::validate_resource "$resource_name"; then
        log::error "Resource is not functional: $resource_name"
        log::info "Install the resource first with: vrooli resource $resource_name install"
        return 1
    fi
    
    # Try using CLI first
    if command -v "resource-${resource_name}" &>/dev/null; then
        log::info "Starting resource: $resource_name"
        # Don't redirect stderr - sudo needs it for password prompts
        if "resource-${resource_name}" start; then
            resource_registry::register "$resource_name" "running"
            log::success "✅ Resource started: $resource_name"
            return 0
        fi
    fi
    
    # Fallback to direct CLI
    local resource_dir=""
    for dir in "${VROOLI_ROOT}/scripts/resources"/*/"${resource_name}"; do
        if [[ -d "$dir" && -f "$dir/cli.sh" ]]; then
            resource_dir="$dir"
            break
        fi
    done
    
    if [[ -n "$resource_dir" && -f "$resource_dir/cli.sh" ]]; then
        # Check if already running via CLI status
        if "$resource_dir/cli.sh" status >/dev/null 2>&1; then
            log::info "Resource already running (via CLI): $resource_name"
            resource_registry::register "$resource_name" "running"
            return 0
        fi
        
        log::info "Starting resource via CLI: $resource_name"
        if "$resource_dir/cli.sh" start; then
            resource_registry::register "$resource_name" "running"
            log::success "✅ Resource started: $resource_name"
            return 0
        fi
    fi
    
    log::debug "Could not start resource: $resource_name (may not support start action)"
    return 0
}

#######################################
# Install all enabled resources from service.json
#######################################
resource_auto::install_enabled() {
    local service_json="${1:-${VROOLI_ROOT}/.vrooli/service.json}"
    
    if [[ ! -f "$service_json" ]]; then
        log::warning "No service.json found at: $service_json"
        return 0
    fi
    
    log::header "📦 Installing Enabled Resources"
    
    # Initialize registry
    resource_registry::init
    
    # Get list of resources already managed by Vrooli (when running as generated app)
    local managed_resources="${VROOLI_MANAGED_RESOURCES:-}"
    local -A skip_resources
    if [[ -n "$managed_resources" ]]; then
        log::info "Skipping resources managed by Vrooli: ${managed_resources//,/, }"
        # Convert comma-separated list to associative array for fast lookup
        IFS=',' read -ra managed_array <<< "$managed_resources"
        for res in "${managed_array[@]}"; do
            skip_resources["$res"]=1
        done
    fi
    
    # Get list of enabled resources (handle nested structure with categories)
    local resources=$(jq -r '
        .resources | 
        to_entries[] | 
        .value | 
        to_entries[] | 
        select(.value.enabled == true) | 
        .key
    ' "$service_json" 2>/dev/null || echo "")
    
    if [[ -z "$resources" ]]; then
        log::info "No enabled resources found in service.json"
        return 0
    fi
    
    local failed_resources=()
    local installed_count=0
    local skipped_count=0
    
    # Install each enabled resource
    while IFS= read -r resource_name; do
        if [[ -z "$resource_name" ]]; then
            continue
        fi
        
        # Skip if this resource is managed by Vrooli
        if [[ -n "${skip_resources[$resource_name]:-}" ]]; then
            log::debug "Skipping Vrooli-managed resource: $resource_name"
            ((skipped_count++))
            continue
        fi
        
        # Get resource configuration (search through categories)
        local resource_config=$(jq -r --arg name "$resource_name" '
            .resources | 
            to_entries[] | 
            .value[$name] // empty
        ' "$service_json" | jq -s 'add // {}')
        
        # Install resource
        if resource_auto::install_resource "$resource_name" "$resource_config"; then
            ((installed_count++))
            
            # Optionally start the resource
            resource_auto::start_resource "$resource_name" "$resource_config" || true
        else
            failed_resources+=("$resource_name")
        fi
    done <<< "$resources"
    
    # Report results
    if [[ $skipped_count -gt 0 ]]; then
        log::info "⏭️  Skipped $skipped_count Vrooli-managed resource(s)"
    fi
    
    if [[ $installed_count -gt 0 ]]; then
        log::success "✅ Installed $installed_count resource(s)"
    fi
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        log::warning "Failed to install: ${failed_resources[*]}"
        return 1
    fi
    
    return 0
}

#######################################
# Stop all registered resources
#######################################
resource_auto::stop_all() {
    log::header "🛑 Stopping All Resources"
    
    local running_resources=$(resource_registry::list "running")
    
    if [[ -z "$running_resources" ]]; then
        log::info "No running resources to stop"
        return 0
    fi
    
    local stopped_count=0
    
    while IFS= read -r resource_name; do
        if [[ -z "$resource_name" ]]; then
            continue
        fi
        
        log::info "Stopping resource: $resource_name"
        
        # Try CLI first
        if command -v "resource-${resource_name}" &>/dev/null; then
            if "resource-${resource_name}" stop 2>/dev/null; then
                resource_registry::register "$resource_name" "stopped"
                ((stopped_count++))
                log::success "✅ Stopped: $resource_name"
                continue
            fi
        fi
        
        # Fallback to direct CLI
        local resource_dir=""
        for dir in "${VROOLI_ROOT}/scripts/resources"/*/"${resource_name}"; do
            if [[ -d "$dir" && -f "$dir/cli.sh" ]]; then
                resource_dir="$dir"
                break
            fi
        done
        
        if [[ -n "$resource_dir" && -f "$resource_dir/cli.sh" ]]; then
            if "$resource_dir/cli.sh" stop 2>/dev/null; then
                resource_registry::register "$resource_name" "stopped"
                ((stopped_count++))
                log::success "✅ Stopped: $resource_name"
                continue
            fi
        fi
        
        log::warning "Could not stop resource: $resource_name"
    done <<< "$running_resources"
    
    if [[ $stopped_count -gt 0 ]]; then
        log::success "✅ Stopped $stopped_count resource(s)"
    fi
    
    return 0
}

# Export functions
export -f resource_registry::init
export -f resource_registry::register
export -f resource_registry::list
export -f resource_auto::validate_resource
export -f resource_auto::install_resource
export -f resource_auto::start_resource
export -f resource_auto::install_enabled
export -f resource_auto::stop_all