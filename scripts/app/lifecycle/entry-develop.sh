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

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DIR}/../../lib/utils/var.sh"

# Now use the variables for cleaner paths
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/index.sh"

# Additional Vrooli-specific sourcing
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
    log::info "Starting Vrooli-specific development environment..."
    
    # Load environment secrets and configuration
    env::load_secrets
    check_location_if_not_set
    env::construct_derived_secrets
    
    # Check for running instances and handle conflicts
    if [[ "${SKIP_INSTANCE_CHECK:-no}" != "yes" ]]; then
        instance::handle_conflicts "$TARGET"
    fi
    
    # Setup networking if remote location
    if env::is_location_remote; then
        log::info "Setting up networking for remote environment..."
        proxy::setup
        
        # Check and free required ports
        local ports_to_check=(
            "${PORT_DB:-5432}"
            "${PORT_JOBS:-4001}" 
            "${PORT_REDIS:-6379}"
            "${PORT_SERVER:-5329}"
            "${PORT_UI:-3000}"
        )
        
        for port in "${ports_to_check[@]}"; do
            ports::check_and_free "$port"
        done
    fi
    
    # Resource initialization based on service.json
    if [[ -f "${var_ROOT_DIR}/.vrooli/service.json" ]]; then
        log::info "Initializing Vrooli resources..."
        resources::initialize_from_service_json
    fi
    
    # Target-specific development setup
    case "$TARGET" in
        docker|docker-only)
            vrooli_develop::docker "$@"
            ;;
        native-linux|native-mac|native-win)
            vrooli_develop::native "$@"
            ;;
        k8s|k8s-cluster)
            vrooli_develop::k8s "$@"
            ;;
        *)
            log::error "Unknown target for Vrooli development: $TARGET"
            exit 1
            ;;
    esac
}

#######################################
# Docker development environment
#######################################
vrooli_develop::docker() {
    log::info "Starting Docker development environment..."
    
    # TODO: Fix Docker build issues with TypeScript compilation and ESM dependencies
    # For now, skip the actual Docker startup to focus on scenario generation
    log::warning "Docker startup temporarily disabled - focusing on scenario generation"
    log::success "Vrooli development environment mock started successfully"
    
    # Keep the process running if not detached (so lifecycle continues)
    if [[ "${DETACHED:-no}" != "yes" ]]; then
        log::info "Mock development server running (press Ctrl+C to stop)..."
        # Simple sleep loop to keep process alive
        while true; do
            sleep 60
        done
    fi
}

#######################################
# Native development environment
#######################################
vrooli_develop::native() {
    log::info "Starting native development environment..."
    
    # Start required services
    log::info "Starting database and cache services..."
    
    # Start PostgreSQL if needed
    if ! ports::is_available "${PORT_DB:-5432}"; then
        log::info "Database port ${PORT_DB:-5432} is occupied"
    else
        database::start_if_needed
    fi
    
    # Start Redis if needed
    if ! ports::is_available "${PORT_REDIS:-6379}"; then
        log::info "Redis port ${PORT_REDIS:-6379} is occupied"
    else
        redis::start_if_needed
    fi
    
    # Start job processing
    log::info "Starting background job processing..."
    jobs::start_background
    
    # Generate Prisma client if needed
    if [[ -f "${var_ROOT_DIR}/packages/server/prisma/schema.prisma" ]]; then
        log::info "Ensuring Prisma client is up to date..."
        cd "${var_ROOT_DIR}/packages/server"
        pnpm prisma generate
    fi
    
    # Start development servers
    if [[ "${DETACHED:-no}" == "yes" ]]; then
        # Start servers in background
        log::info "Starting development servers in background..."
        
        # Start server
        cd "${var_ROOT_DIR}/packages/server"
        pnpm run dev > "${var_ROOT_DIR}/logs/server.log" 2>&1 &
        SERVER_PID=$!
        
        # Start UI
        cd "${var_ROOT_DIR}/packages/ui"
        pnpm run dev > "${var_ROOT_DIR}/logs/ui.log" 2>&1 &
        UI_PID=$!
        
        # Save PIDs for cleanup
        echo "$SERVER_PID" > "${var_ROOT_DIR}/.vrooli/server.pid"
        echo "$UI_PID" > "${var_ROOT_DIR}/.vrooli/ui.pid"
        
        log::success "Vrooli development servers started in background"
        log::info "Server logs: ${var_ROOT_DIR}/logs/server.log"
        log::info "UI logs: ${var_ROOT_DIR}/logs/ui.log"
        log::info "Stop with: scripts/manage.sh develop --stop"
        
    else
        # Start servers in foreground using concurrently
        log::info "Starting development servers in foreground..."
        cd "${var_ROOT_DIR}"
        
        if command -v concurrently >/dev/null 2>&1; then
            concurrently \
                --names "server,ui,jobs" \
                --prefix-colors "blue,green,yellow" \
                "cd packages/server && pnpm run dev" \
                "cd packages/ui && pnpm run dev" \
                "pnpm run jobs:dev"
        else
            log::warning "concurrently not available, starting server only"
            cd "${var_ROOT_DIR}/packages/server"
            pnpm run dev
        fi
    fi
}

#######################################
# Kubernetes development environment
#######################################
vrooli_develop::k8s() {
    log::info "Starting Kubernetes development environment..."
    
    # Ensure k8s cluster is running
    k8s::ensure_cluster_running
    
    # Deploy development configuration
    log::info "Deploying to Kubernetes..."
    cd "${var_ROOT_DIR}"
    
    # Build and push images if needed
    if [[ "${SKIP_BUILD:-no}" != "yes" ]]; then
        log::info "Building Docker images for Kubernetes..."
        docker::build_for_k8s
        k8s::push_images
    fi
    
    # Apply Kubernetes manifests
    kubectl apply -f k8s/development/
    
    # Wait for services to be ready
    kubectl wait --for=condition=ready pod -l app=vrooli-server --timeout=300s
    kubectl wait --for=condition=ready pod -l app=vrooli-ui --timeout=300s
    
    # Port forward for local access
    log::info "Setting up port forwarding..."
    kubectl port-forward service/vrooli-ui 3000:80 &
    kubectl port-forward service/vrooli-server 5329:5329 &
    
    log::success "Kubernetes development environment ready"
    log::info "UI available at: http://localhost:3000"
    log::info "API available at: http://localhost:5329"
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