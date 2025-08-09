#!/usr/bin/env bash
################################################################################
# Universal Deploy Phase Handler
# 
# Handles generic deployment tasks:
# - Artifact validation
# - Deployment strategy selection
# - Health checks
# - Rollback support
#
# App-specific logic should be in app/lifecycle/deploy.sh
################################################################################

set -euo pipefail

# Get script directory
LIB_LIFECYCLE_PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${LIB_LIFECYCLE_PHASES_DIR}/../../utils/var.sh"

# Source common utilities
# shellcheck disable=SC1091
source "${LIB_LIFECYCLE_PHASES_DIR}/common.sh"

# Source required libraries
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"

################################################################################
# Deployment Functions
################################################################################

#######################################
# Validate deployment artifacts
# Arguments:
#   $1 - Source type (docker/k8s/zip)
#   $2 - Version
# Returns:
#   0 if valid, 1 otherwise
#######################################
deploy::validate_artifacts() {
    local source_type="$1"
    local version="${2:-latest}"
    
    log::info "Validating $source_type artifacts (version: $version)..."
    
    case "$source_type" in
        docker)
            # Check if Docker images exist
            if command -v docker &> /dev/null; then
                local image_name="${IMAGE_NAME:-app}"
                if docker image inspect "${image_name}:${version}" &> /dev/null; then
                    log::success "✅ Docker image found: ${image_name}:${version}"
                    return 0
                else
                    log::error "Docker image not found: ${image_name}:${version}"
                    log::info "Run './scripts/manage.sh build --artifacts docker' first"
                    return 1
                fi
            else
                log::error "Docker is not installed"
                return 1
            fi
            ;;
            
        k8s|kubernetes)
            # Check if k8s manifests exist
            local k8s_dir="${var_ROOT_DIR}/k8s"
            if [[ -d "$k8s_dir" ]]; then
                log::success "✅ Kubernetes manifests found"
                return 0
            else
                log::error "Kubernetes manifests not found"
                return 1
            fi
            ;;
            
        zip)
            # Check if deployment bundle exists
            local bundle_path="${var_DEST_DIR}/bundle-${version}.zip"
            if [[ -f "$bundle_path" ]]; then
                log::success "✅ Deployment bundle found: $bundle_path"
                return 0
            else
                log::error "Deployment bundle not found: $bundle_path"
                log::info "Run './scripts/manage.sh build --bundles zip' first"
                return 1
            fi
            ;;
            
        *)
            log::error "Unknown source type: $source_type"
            return 1
            ;;
    esac
}

#######################################
# Deploy using Docker
# Arguments:
#   $1 - Version
#   $2 - Environment
# Returns:
#   0 on success
#######################################
deploy::docker() {
    local version="${1:-latest}"
    local environment="${2:-production}"
    
    log::info "Deploying Docker container..."
    
    # Check for docker-compose
    local compose_file="${var_ROOT_DIR}/docker-compose.${environment}.yml"
    if [[ ! -f "$compose_file" ]]; then
        compose_file="${var_DOCKER_COMPOSE_DEV_FILE}"
    fi
    
    if [[ -f "$compose_file" ]]; then
        log::info "Using compose file: $compose_file"
        
        export VERSION="$version"
        export ENVIRONMENT="$environment"
        
        if command -v docker-compose &> /dev/null; then
            (cd "${var_ROOT_DIR}" && docker-compose -f "$compose_file" up -d)
        else
            (cd "${var_ROOT_DIR}" && docker compose -f "$compose_file" up -d)
        fi
        
        log::success "✅ Docker deployment complete"
        return 0
    else
        log::error "No docker-compose file found"
        return 1
    fi
}

#######################################
# Deploy to Kubernetes
# Arguments:
#   $1 - Version
#   $2 - Environment
# Returns:
#   0 on success
#######################################
deploy::kubernetes() {
    local version="${1:-latest}"
    local environment="${2:-production}"
    
    log::info "Deploying to Kubernetes..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log::error "kubectl is not installed"
        return 1
    fi
    
    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        log::error "Cannot connect to Kubernetes cluster"
        log::info "Check your KUBECONFIG or kubectl context"
        return 1
    fi
    
    # Apply manifests
    local k8s_dir="${var_ROOT_DIR}/k8s"
    if [[ -d "$k8s_dir" ]]; then
        log::info "Applying manifests from: $k8s_dir"
        
        # Set namespace
        local namespace="${NAMESPACE:-default}"
        kubectl create namespace "$namespace" 2>/dev/null || true
        
        # Apply manifests
        VERSION="$version" kubectl apply -f "$k8s_dir" -n "$namespace"
        
        # Wait for rollout
        log::info "Waiting for rollout to complete..."
        kubectl rollout status deployment -n "$namespace" --timeout=5m
        
        log::success "✅ Kubernetes deployment complete"
        return 0
    else
        log::error "Kubernetes manifests not found"
        return 1
    fi
}

#######################################
# Run deployment health checks
# Arguments:
#   $1 - Source type
# Returns:
#   0 if healthy
#######################################
deploy::health_check() {
    local source_type="$1"
    
    log::info "Running health checks..."
    
    # Get health check endpoint
    local health_endpoint="${HEALTH_ENDPOINT:-/health}"
    local health_port="${HEALTH_PORT:-8080}"
    local max_attempts=30
    local attempt=0
    
    case "$source_type" in
        docker)
            # Check Docker container health
            local container_name="${CONTAINER_NAME:-app}"
            
            while [[ $attempt -lt $max_attempts ]]; do
                if docker exec "$container_name" curl -f "http://localhost:${health_port}${health_endpoint}" &> /dev/null; then
                    log::success "✅ Health check passed"
                    return 0
                fi
                
                attempt=$((attempt + 1))
                log::info "Waiting for service to be healthy... ($attempt/$max_attempts)"
                sleep 2
            done
            ;;
            
        k8s|kubernetes)
            # Check Kubernetes deployment health
            local namespace="${NAMESPACE:-default}"
            local service_name="${SERVICE_NAME:-app}"
            
            while [[ $attempt -lt $max_attempts ]]; do
                if kubectl exec -n "$namespace" deploy/"$service_name" -- curl -f "http://localhost:${health_port}${health_endpoint}" &> /dev/null; then
                    log::success "✅ Health check passed"
                    return 0
                fi
                
                attempt=$((attempt + 1))
                log::info "Waiting for service to be healthy... ($attempt/$max_attempts)"
                sleep 2
            done
            ;;
    esac
    
    log::error "Health check failed after $max_attempts attempts"
    return 1
}

################################################################################
# Main Deployment Logic
################################################################################

#######################################
# Run universal deployment tasks
# Handles generic deployment operations
# Globals:
#   SOURCE_TYPE
#   VERSION
#   ENVIRONMENT
#   SKIP_HEALTH_CHECK
# Returns:
#   0 on success, 1 on failure
#######################################
deploy::universal::main() {
    # Initialize phase
    phase::init "Deploy"
    
    # Get parameters from environment or defaults
    local source_type="${SOURCE_TYPE:-docker}"
    local version="${VERSION:-latest}"
    local environment="${ENVIRONMENT:-production}"
    local skip_health="${SKIP_HEALTH_CHECK:-no}"
    
    log::info "Universal deploy starting..."
    log::debug "Parameters:"
    log::debug "  Source: $source_type"
    log::debug "  Version: $version"
    log::debug "  Environment: $environment"
    log::debug "  Skip health: $skip_health"
    
    # Step 1: Validate artifacts
    log::header "📋 Validating Artifacts"
    if ! deploy::validate_artifacts "$source_type" "$version"; then
        log::error "Artifact validation failed"
        return 1
    fi
    
    # Step 2: Run pre-deploy hook
    phase::run_hook "preDeploy"
    
    # Step 3: Deploy based on source type
    log::header "🚀 Deploying Application"
    
    case "$source_type" in
        docker)
            if ! deploy::docker "$version" "$environment"; then
                log::error "Docker deployment failed"
                return 1
            fi
            ;;
            
        k8s|kubernetes)
            if ! deploy::kubernetes "$version" "$environment"; then
                log::error "Kubernetes deployment failed"
                return 1
            fi
            ;;
            
        *)
            # Look for app-specific deployment
            local app_deploy="${var_APP_LIFECYCLE_DEPLOY_DIR:-}/${source_type}.sh"
            if [[ -f "$app_deploy" ]]; then
                log::info "Running app-specific deployment: $source_type"
                if ! bash "$app_deploy" "$version" "$environment"; then
                    log::error "Deployment failed"
                    return 1
                fi
            else
                log::error "Unknown deployment source: $source_type"
                return 1
            fi
            ;;
    esac
    
    # Step 4: Health checks
    if ! flow::is_yes "$skip_health"; then
        log::header "🏥 Health Checks"
        if ! deploy::health_check "$source_type"; then
            log::error "Deployment health check failed"
            log::info "The deployment may need more time to initialize"
            log::info "Or there may be an issue with the deployment"
            return 1
        fi
    fi
    
    # Step 5: Run post-deploy hook
    phase::run_hook "postDeploy"
    
    # Complete phase
    phase::complete
    
    # Export deployment information
    export DEPLOY_VERSION="$version"
    export DEPLOY_ENVIRONMENT="$environment"
    export DEPLOY_SOURCE="$source_type"
    
    log::success "✅ Deployment completed successfully"
    log::info "Version: $version"
    log::info "Environment: $environment"
    
    return 0
}

#######################################
# Entry point for direct execution
#######################################
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Check if being called by lifecycle engine
    if [[ "${LIFECYCLE_PHASE:-}" == "deploy" ]]; then
        deploy::universal::main "$@"
    else
        log::error "This script should be called through the lifecycle engine"
        log::info "Use: ./scripts/manage.sh deploy [options]"
        exit 1
    fi
fi