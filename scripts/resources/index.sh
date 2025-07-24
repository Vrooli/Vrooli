#!/usr/bin/env bash
set -euo pipefail

# Main orchestrator for local resource setup and management
# This script provides a unified interface for managing local resources

DESCRIPTION="Manages local development resources (AI, automation, storage, agents)"

RESOURCES_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Available resources organized by category
declare -A AVAILABLE_RESOURCES=(
    ["ai"]="ollama localai llamacpp"
    ["automation"]="n8n nodered"
    ["storage"]="minio ipfs"
    ["agents"]="puppeteer playwright"
)

# All available resources as a flat list
ALL_RESOURCES="ollama localai llamacpp n8n nodered minio ipfs puppeteer playwright"

#######################################
# Parse command line arguments
#######################################
resources::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|list|discover" \
        --default "install"
    
    args::register \
        --name "resources" \
        --flag "r" \
        --desc "Resources to manage (comma-separated, or 'all', 'ai-only', 'none')" \
        --type "value" \
        --default "none"
    
    args::register \
        --name "category" \
        --flag "c" \
        --desc "Resource category filter" \
        --type "value" \
        --options "ai|automation|storage|agents" \
        --default ""
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if resource appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    if args::is_asking_for_help "$@"; then
        resources::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export RESOURCES_INPUT=$(args::get "resources")
    export CATEGORY=$(args::get "category")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
}

#######################################
# Display usage information
#######################################
resources::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install --resources ollama                # Install Ollama"
    echo "  $0 --action install --resources ai-only               # Install all AI resources"
    echo "  $0 --action install --resources all                   # Install all resources"
    echo "  $0 --action status --resources ollama,n8n             # Check status of specific resources"
    echo "  $0 --action list                                      # List available resources"
    echo "  $0 --action discover                                  # Discover running resources"
    echo
    echo "Resource Categories:"
    for category in "${!AVAILABLE_RESOURCES[@]}"; do
        echo "  $category: ${AVAILABLE_RESOURCES[$category]}"
    done
}

#######################################
# Resolve resource list from input
# Outputs: space-separated list of resources
#######################################
resources::resolve_list() {
    local input="$RESOURCES_INPUT"
    local resolved=""
    
    case "$input" in
        "all")
            resolved="$ALL_RESOURCES"
            ;;
        "ai-only")
            resolved="${AVAILABLE_RESOURCES[ai]}"
            ;;
        "automation-only")
            resolved="${AVAILABLE_RESOURCES[automation]}"
            ;;
        "storage-only")
            resolved="${AVAILABLE_RESOURCES[storage]}"
            ;;
        "agents-only")
            resolved="${AVAILABLE_RESOURCES[agents]}"
            ;;
        "none"|"")
            resolved=""
            ;;
        *)
            # Parse comma-separated list
            IFS=',' read -ra RESOURCE_ARRAY <<< "$input"
            for resource in "${RESOURCE_ARRAY[@]}"; do
                resource=$(echo "$resource" | tr -d ' ')  # trim whitespace
                if [[ " $ALL_RESOURCES " =~ " $resource " ]]; then
                    resolved="$resolved $resource"
                else
                    log::error "Unknown resource: $resource"
                    log::info "Available resources: $ALL_RESOURCES"
                    exit 1
                fi
            done
            ;;
    esac
    
    # Apply category filter if specified
    if [[ -n "$CATEGORY" && -n "$resolved" ]]; then
        local filtered=""
        local category_resources="${AVAILABLE_RESOURCES[$CATEGORY]}"
        for resource in $resolved; do
            if [[ " $category_resources " =~ " $resource " ]]; then
                filtered="$filtered $resource"
            fi
        done
        resolved="$filtered"
    fi
    
    echo "$resolved"
}

#######################################
# Get the category for a resource
# Arguments:
#   $1 - resource name
# Outputs: category name
#######################################
resources::get_category() {
    local resource="$1"
    for category in "${!AVAILABLE_RESOURCES[@]}"; do
        if [[ " ${AVAILABLE_RESOURCES[$category]} " =~ " $resource " ]]; then
            echo "$category"
            return
        fi
    done
    echo "unknown"
}

#######################################
# Get the script path for a resource
# Arguments:
#   $1 - resource name
# Outputs: script path
#######################################
resources::get_script_path() {
    local resource="$1"
    local category
    category=$(resources::get_category "$resource")
    echo "${RESOURCES_DIR}/${category}/${resource}.sh"
}

#######################################
# Check if a resource script exists
# Arguments:
#   $1 - resource name
# Returns: 0 if exists, 1 otherwise
#######################################
resources::script_exists() {
    local resource="$1"
    local script_path
    script_path=$(resources::get_script_path "$resource")
    [[ -f "$script_path" ]]
}

#######################################
# Execute action on a single resource
# Arguments:
#   $1 - resource name
#   $2 - action
# Returns: exit code from resource script
#######################################
resources::execute_action() {
    local resource="$1"
    local action="$2"
    local script_path
    script_path=$(resources::get_script_path "$resource")
    
    if ! resources::script_exists "$resource"; then
        log::error "Resource script not found: $script_path"
        return 1
    fi
    
    log::header "🔧 $resource: $action"
    
    # Execute the resource script with the action
    bash "$script_path" --action "$action" --force "$FORCE" --yes "$YES"
}

#######################################
# List available resources
#######################################
resources::list_available() {
    log::header "📋 Available Resources"
    
    for category in "${!AVAILABLE_RESOURCES[@]}"; do
        log::info "Category: $category"
        for resource in ${AVAILABLE_RESOURCES[$category]}; do
            local script_path
            script_path=$(resources::get_script_path "$resource")
            local status="❌ Not implemented"
            
            if [[ -f "$script_path" ]]; then
                status="✅ Available"
                # Check if actually installed/running
                local port
                port=$(resources::get_default_port "$resource")
                if resources::is_service_running "$port"; then
                    status="🟢 Running"
                fi
            fi
            
            echo "  - $resource: $status"
        done
        echo
    done
}

#######################################
# Discover running resources
#######################################
resources::discover_running() {
    log::header "🔍 Discovering Running Resources"
    
    local found_any=false
    
    for resource in $ALL_RESOURCES; do
        local port
        port=$(resources::get_default_port "$resource")
        
        if resources::is_service_running "$port"; then
            log::success "✅ $resource is running on port $port"
            found_any=true
            
            # Check if it responds to HTTP
            local base_url="http://localhost:$port"
            if resources::check_http_health "$base_url"; then
                log::info "   HTTP health check: ✅ Healthy"
            else
                log::warn "   HTTP health check: ⚠️  No response"
            fi
        fi
    done
    
    if ! $found_any; then
        log::info "No known resources detected running"
    fi
    
    echo
    log::info "To configure discovered resources for Vrooli, run:"
    log::info "  $0 --action install --resources <resource-name>"
}

#######################################
# Main execution function
#######################################
resources::main() {
    resources::parse_arguments "$@"
    
    case "$ACTION" in
        "list")
            resources::list_available
            return 0
            ;;
        "discover")
            resources::discover_running
            return 0
            ;;
    esac
    
    local resource_list
    resource_list=$(resources::resolve_list)
    
    if [[ -z "$resource_list" ]]; then
        log::error "No resources specified"
        log::info "Use --resources to specify resources, or --action list to see available options"
        exit 1
    fi
    
    log::header "🚀 Resource Management"
    log::info "Action: $ACTION"
    log::info "Resources: $resource_list"
    echo
    
    local overall_success=true
    local processed_count=0
    local success_count=0
    
    for resource in $resource_list; do
        processed_count=$((processed_count + 1))
        
        if resources::execute_action "$resource" "$ACTION"; then
            success_count=$((success_count + 1))
        else
            overall_success=false
            log::error "Failed to $ACTION $resource"
        fi
        
        # Add spacing between resources
        if [[ $processed_count -lt $(echo "$resource_list" | wc -w) ]]; then
            echo
        fi
    done
    
    echo
    log::header "📊 Summary"
    log::info "Processed: $processed_count resources"
    log::info "Successful: $success_count resources"
    
    if $overall_success; then
        log::success "✅ All operations completed successfully"
    else
        log::error "❌ Some operations failed"
        exit 1
    fi
    
    # Show next steps for install action
    if [[ "$ACTION" == "install" && $success_count -gt 0 ]]; then
        echo
        log::header "🎯 Next Steps"
        log::info "1. Resources have been configured in ~/.vrooli/resources.local.json"
        log::info "2. Start your Vrooli development environment to begin using these resources"
        log::info "3. Check resource status with: $0 --action status --resources $RESOURCES_INPUT"
    fi
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resources::main "$@"
fi