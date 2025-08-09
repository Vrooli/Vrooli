#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Vrooli-Specific Development Lifecycle
# 
# Handles complex development environment initialization including:
# - Resource management (databases, AI models, etc.)  
# - Port management and conflict resolution
# - Instance management for multi-environment support
# - Proxy and networking setup
# - Environment variable loading and validation
#
# This script contains all the Vrooli-specific logic extracted from develop.sh
################################################################################

APP_LIFECYCLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DIR}/../../lib/utils/var.sh"

# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/index.sh"

# shellcheck disable=SC1091
source "${var_APP_LIFECYCLE_DEVELOP_DIR}/index.sh"
# shellcheck disable=SC1091
source "${var_APP_LIFECYCLE_DEVELOP_DIR}/port_manager.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/instance_manager.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/domainCheck.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/docker.sh"

#######################################
# Main Vrooli development lifecycle
# Arguments:
#   All arguments passed from main develop.sh
#######################################
vrooli_develop::main() {
    log::info "Starting Vrooli..."
    
    # Load environment secrets and configuration
    env::load_secrets
    check_location_if_not_set
    env::construct_derived_secrets
    
    # Check for running instances and handle conflicts
    if [[ "${SKIP_INSTANCE_CHECK:-no}" != "yes" ]]; then
        instance::handle_conflicts "$TARGET"
    fi

    # Resource initialization based on service.json
    if [[ -f "${var_ROOT_DIR}/.vrooli/service.json" ]]; then
        log::info "Initializing Vrooli resources..."
        resources::initialize_from_service_json
    fi

    log::success "🚀 Vrooli started! Please view the app-monitor dashboard to manage Vrooli"
}

#######################################
# Resource initialization from service.json
#######################################
resources::initialize_from_service_json() {
    local service_json="${var_ROOT_DIR}/.vrooli/service.json"
    
    # Get enabled resources from service.json
    local enabled_resources
    enabled_resources=$(jq -r '.resources.enabled[]? // empty' "$service_json" 2>/dev/null)
    
    if [[ -n "$enabled_resources" ]]; then
        log::info "Initializing enabled resources: $enabled_resources"
        
        # Start each enabled resource
        while IFS= read -r resource; do
            if [[ -f "${var_ROOT_DIR}/scripts/resources/${resource}/manage.sh" ]]; then
                log::info "Starting resource: $resource"
                bash "${var_ROOT_DIR}/scripts/resources/${resource}/manage.sh" start || {
                    log::warning "Failed to start resource: $resource"
                }
            fi
        done <<< "$enabled_resources"
    fi
}

# Only run main if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    vrooli_develop::main "$@"
fi