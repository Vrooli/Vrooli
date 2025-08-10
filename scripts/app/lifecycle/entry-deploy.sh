#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Vrooli-Specific Deploy Lifecycle
# 
# Handles complex deployment processes including:
# - Multi-environment deployments (dev, staging, production)
# - Kubernetes deployments with Helm charts
# - Docker registry management and image pushing
# - Database migrations and schema updates  
# - Configuration management and secret injection
# - Health checks and deployment verification
# - Rollback capabilities
#
# This script contains all the Vrooli-specific logic extracted from deploy.sh
################################################################################

APP_LIFECYCLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/index.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

#######################################
# Main Vrooli deploy lifecycle
# Arguments:
#   All arguments passed from main deploy.sh
#######################################
vrooli_deploy::main() {
    log::info "Starting Vrooli-specific deployment process..."
    
    # Load environment for deployment configuration
    env::load_secrets
    
    # Set deployment metadata
    export DEPLOY_VERSION="${VERSION:-$(date +%Y%m%d-%H%M%S)}"
    export DEPLOY_COMMIT="${CI_COMMIT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
    export DEPLOY_TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    export DEPLOY_ENVIRONMENT="${ENVIRONMENT:-development}"
    
    log::info "Deployment metadata:"
    log::info "  Version: $DEPLOY_VERSION"
    log::info "  Commit: $DEPLOY_COMMIT"  
    log::info "  Environment: $DEPLOY_ENVIRONMENT"
    log::info "  Target: $TARGET"
    
    # Pre-deployment validation
    vrooli_deploy::validate_deployment
    
    # Execute deployment based on target
    case "$TARGET" in
        docker|docker-only)
            vrooli_deploy::docker "$@"
            ;;
        k8s|k8s-cluster)
            vrooli_deploy::k8s "$@"
            ;;
        native-linux|native-mac|native-win)
            vrooli_deploy::native "$@"
            ;;
        *)
            log::error "Unknown target for Vrooli deployment: $TARGET"
            exit 1
            ;;
    esac
    
    # Post-deployment verification
    vrooli_deploy::post_deployment_verification
}

#######################################
# Validate deployment prerequisites
#######################################
vrooli_deploy::validate_deployment() {
    log::info "Validating deployment prerequisites..."
    
    # Check if images/artifacts exist
    case "$TARGET" in
        docker|k8s*)
            vrooli_deploy::validate_docker_images
            ;;
        native*)
            vrooli_deploy::validate_native_build
            ;;
    esac
    
    # Environment-specific validations
    case "$DEPLOY_ENVIRONMENT" in
        production)
            vrooli_deploy::validate_production_prerequisites
            ;;
        staging)
            vrooli_deploy::validate_staging_prerequisites
            ;;
        development)
            vrooli_deploy::validate_development_prerequisites
            ;;
    esac
    
    # Check required environment variables
    local required_vars=("DATABASE_URL" "REDIS_URL")
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            log::error "Required environment variable not set: $var"
            exit 1
        fi
    done
    
    log::success "Deployment validation passed"
}

#######################################
# Validate Docker images exist
#######################################
vrooli_deploy::validate_docker_images() {
    local services=("server" "ui" "jobs")
    
    for service in "${services[@]}"; do
        local image="vrooli-${service}:${DEPLOY_VERSION}"
        
        if ! docker image inspect "$image" >/dev/null 2>&1; then
            log::error "Docker image not found: $image"
            log::info "Run build process first: scripts/manage.sh build --target docker"
            exit 1
        fi
        
        log::info "Validated Docker image: $image"
    done
}

#######################################
# Validate native build artifacts
#######################################
vrooli_deploy::validate_native_build() {
    if [[ ! -d "${var_ROOT_DIR}/dist/native" ]]; then
        log::error "Native build artifacts not found: dist/native/"
        log::info "Run build process first: scripts/manage.sh build --target native"
        exit 1
    fi
    
    # Check each package has build artifacts
    local packages=("shared" "server" "ui" "jobs")
    for package in "${packages[@]}"; do
        if [[ ! -d "${var_ROOT_DIR}/dist/native/${package}" ]]; then
            log::error "Missing build artifacts for package: $package"
            exit 1
        fi
    done
    
    log::info "Validated native build artifacts"
}

#######################################
# Production deployment validations
#######################################
vrooli_deploy::validate_production_prerequisites() {
    log::info "Validating production deployment prerequisites..."
    
    # Require confirmation for production deployments
    if [[ "${CONFIRM_PRODUCTION:-}" != "yes" ]]; then
        log::warning "This is a PRODUCTION deployment!"
        log::warning "Environment: $DEPLOY_ENVIRONMENT"
        log::warning "Version: $DEPLOY_VERSION"
        
        read -rp "Are you sure you want to proceed? (type 'deploy-production' to confirm): " confirmation
        if [[ "$confirmation" != "deploy-production" ]]; then
            log::info "Deployment cancelled by user"
            exit 0
        fi
    fi
    
    # Check production-specific requirements
    local prod_vars=("STRIPE_SECRET_KEY" "JWT_SECRET" "DATABASE_URL")
    for var in "${prod_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            log::error "Production requires environment variable: $var"
            exit 1
        fi
    done
    
    # Ensure we're on the correct branch/tag for production
    if [[ "${DEPLOY_ENVIRONMENT}" == "production" ]]; then
        local current_branch
        current_branch=$(git branch --show-current 2>/dev/null || echo "detached")
        
        if [[ "$current_branch" != "main" && "$current_branch" != "master" ]]; then
            log::warning "Deploying to production from branch: $current_branch"
            read -rp "Continue? (y/N): " continue_deploy
            if [[ ! "$continue_deploy" =~ ^[Yy]$ ]]; then
                exit 0
            fi
        fi
    fi
}

#######################################
# Staging deployment validations
#######################################
vrooli_deploy::validate_staging_prerequisites() {
    log::info "Validating staging deployment prerequisites..."
    
    # Staging-specific checks
    local staging_vars=("DATABASE_URL" "REDIS_URL")
    for var in "${staging_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            log::warning "Staging deployment missing: $var"
        fi
    done
}

#######################################
# Development deployment validations
#######################################
vrooli_deploy::validate_development_prerequisites() {
    log::info "Validating development deployment prerequisites..."
    
    # Development is more lenient, but still check basics
    if [[ -z "${DATABASE_URL:-}" ]]; then
        log::info "DATABASE_URL not set - using default development database"
        export DATABASE_URL="postgresql://postgres:password@localhost:5432/vrooli_dev"
    fi
}

#######################################
# Docker deployment process
#######################################
vrooli_deploy::docker() {
    log::info "Deploying Docker containers..."
    
    # Push images to registry if configured
    if [[ -n "${DOCKER_REGISTRY:-}" ]]; then
        vrooli_deploy::push_docker_images
    fi
    
    # Update docker-compose with new version
    vrooli_deploy::update_docker_compose
    
    # Deploy with zero-downtime strategy
    log::info "Performing zero-downtime Docker deployment..."
    
    # Pull new images
    docker compose pull
    
    # Start new containers alongside old ones
    docker compose up --no-deps --detach --scale server=2 --scale ui=2 server ui
    
    # Health check new containers
    if vrooli_deploy::health_check_docker; then
        # Stop old containers
        log::info "Health check passed, removing old containers..."
        docker compose up --no-deps --detach --scale server=1 --scale ui=1 server ui
        
        # Clean up old images
        docker image prune -f
        
        log::success "Docker deployment completed successfully"
    else
        log::error "Health check failed, rolling back..."
        docker compose up --no-deps --detach --scale server=1 --scale ui=1 server ui
        exit 1
    fi
}

#######################################
# Kubernetes deployment process
#######################################
vrooli_deploy::k8s() {
    log::info "Deploying to Kubernetes..."
    
    # Check kubectl connectivity
    if ! kubectl cluster-info >/dev/null 2>&1; then
        log::error "Cannot connect to Kubernetes cluster"
        log::info "Check your kubectl configuration and cluster connectivity"
        exit 1
    fi
    
    # Create namespace if it doesn't exist
    local namespace="vrooli-${DEPLOY_ENVIRONMENT}"
    kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f -
    
    # Push images to container registry
    if [[ -n "${CONTAINER_REGISTRY:-}" ]]; then
        vrooli_deploy::push_to_registry
    fi
    
    # Deploy using Helm if chart exists
    if [[ -f "${var_ROOT_DIR}/k8s/helm/Chart.yaml" ]]; then
        vrooli_deploy::helm_deploy "$namespace"
    else
        vrooli_deploy::kubectl_deploy "$namespace"
    fi
    
    # Wait for rollout to complete
    log::info "Waiting for deployment rollout..."
    kubectl rollout status deployment/vrooli-server -n "$namespace" --timeout=600s
    kubectl rollout status deployment/vrooli-ui -n "$namespace" --timeout=600s
    
    # Run post-deployment health checks
    if vrooli_deploy::health_check_k8s "$namespace"; then
        log::success "Kubernetes deployment completed successfully"
    else
        log::error "Kubernetes deployment health check failed"
        vrooli_deploy::rollback_k8s "$namespace"
        exit 1
    fi
}

#######################################
# Native deployment process
#######################################
vrooli_deploy::native() {
    log::info "Deploying native build..."
    
    local deploy_target="${DEPLOY_TARGET:-/opt/vrooli}"
    
    # Create deployment directory
    sudo mkdir -p "$deploy_target"
    
    # Backup current deployment
    if [[ -d "$deploy_target/current" ]]; then
        log::info "Backing up current deployment..."
        sudo mv "$deploy_target/current" "$deploy_target/backup-$(date +%Y%m%d-%H%M%S)"
    fi
    
    # Deploy new version
    log::info "Copying new deployment files..."
    sudo cp -r "${var_ROOT_DIR}/dist/native" "$deploy_target/current"
    
    # Install production dependencies
    cd "$deploy_target/current"
    sudo pnpm install --prod --frozen-lockfile
    
    # Run database migrations if needed
    if [[ -f "$deploy_target/current/server/package.json" ]]; then
        log::info "Running database migrations..."
        cd "$deploy_target/current/server"
        sudo -u vrooli pnpm prisma migrate deploy || {
            log::warning "Database migration failed, but continuing..."
        }
    fi
    
    # Update systemd services
    vrooli_deploy::update_systemd_services
    
    # Restart services
    sudo systemctl restart vrooli-server vrooli-jobs
    sudo systemctl reload nginx  # Assuming nginx serves UI
    
    # Health check
    if vrooli_deploy::health_check_native; then
        log::success "Native deployment completed successfully"
        
        # Clean up old backups (keep last 5)
        while IFS= read -r backup_dir; do
            sudo bash -c "source ${var_LIB_SYSTEM_DIR}/trash.sh && trash::safe_remove '$backup_dir' --no-confirm"
        done < <(find "$deploy_target" -maxdepth 1 -name "backup-*" -type d | sort | head -n -5)
    else
        log::error "Native deployment health check failed"
        vrooli_deploy::rollback_native
        exit 1
    fi
}

#######################################
# Health check for Docker deployment
#######################################
vrooli_deploy::health_check_docker() {
    log::info "Running Docker health checks..."
    
    # Check if containers are running
    if ! docker compose ps | grep -q "running"; then
        log::error "Containers are not running"
        return 1
    fi
    
    # HTTP health checks
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -sf "http://localhost:5329/health" >/dev/null 2>&1; then
            log::success "Server health check passed"
            return 0
        fi
        
        log::info "Health check attempt $attempt/$max_attempts..."
        sleep 10
        ((attempt++))
    done
    
    log::error "Health check failed after $max_attempts attempts"
    return 1
}

#######################################
# Health check for Kubernetes deployment
#######################################
vrooli_deploy::health_check_k8s() {
    local namespace="$1"
    
    log::info "Running Kubernetes health checks..."
    
    # Check pod status
    if ! kubectl get pods -n "$namespace" | grep -q "Running"; then
        log::error "Pods are not running"
        return 1
    fi
    
    # HTTP health checks via port-forward
    kubectl port-forward -n "$namespace" service/vrooli-server 8080:5329 &
    local port_forward_pid=$!
    sleep 5
    
    local max_attempts=10
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -sf "http://localhost:8080/health" >/dev/null 2>&1; then
            kill $port_forward_pid 2>/dev/null || true
            log::success "Kubernetes health check passed"
            return 0
        fi
        
        log::info "Health check attempt $attempt/$max_attempts..."
        sleep 10
        ((attempt++))
    done
    
    kill $port_forward_pid 2>/dev/null || true
    log::error "Kubernetes health check failed after $max_attempts attempts"
    return 1
}

#######################################
# Health check for native deployment
#######################################
vrooli_deploy::health_check_native() {
    log::info "Running native deployment health checks..."
    
    # Check systemd services
    if ! sudo systemctl is-active --quiet vrooli-server; then
        log::error "vrooli-server service is not active"
        return 1
    fi
    
    if ! sudo systemctl is-active --quiet vrooli-jobs; then
        log::error "vrooli-jobs service is not active"
        return 1
    fi
    
    # HTTP health check
    local max_attempts=20
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -sf "http://localhost:5329/health" >/dev/null 2>&1; then
            log::success "Native deployment health check passed"
            return 0
        fi
        
        log::info "Health check attempt $attempt/$max_attempts..."
        sleep 5
        ((attempt++))
    done
    
    log::error "Native deployment health check failed"
    return 1
}

#######################################
# Post-deployment verification
#######################################
vrooli_deploy::post_deployment_verification() {
    log::info "Running post-deployment verification..."
    
    # Create deployment record
    local deployment_record="${var_ROOT_DIR}/deployments/${DEPLOY_TIMESTAMP}.json"
    mkdir -p "$(dirname "$deployment_record")"
    
    cat > "$deployment_record" << EOF
{
    "version": "${DEPLOY_VERSION}",
    "commit": "${DEPLOY_COMMIT}",
    "timestamp": "${DEPLOY_TIMESTAMP}",
    "environment": "${DEPLOY_ENVIRONMENT}",
    "target": "${TARGET}",
    "status": "completed",
    "deployed_by": "${USER:-unknown}"
}
EOF
    
    # Send deployment notification (if webhook configured)
    if [[ -n "${DEPLOY_WEBHOOK_URL:-}" ]]; then
        curl -X POST "$DEPLOY_WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "{\"message\": \"Vrooli ${DEPLOY_VERSION} deployed to ${DEPLOY_ENVIRONMENT}\"}" \
            >/dev/null 2>&1 || log::warning "Failed to send deployment notification"
    fi
    
    log::success "Vrooli deployment completed successfully!"
    log::info "Deployment record: $deployment_record"
}

# Placeholder functions (would be implemented based on specific deployment needs)
vrooli_deploy::push_docker_images() { log::info "Pushing Docker images..."; }
vrooli_deploy::update_docker_compose() { log::info "Updating docker-compose configuration..."; }
vrooli_deploy::push_to_registry() { log::info "Pushing images to container registry..."; }
vrooli_deploy::helm_deploy() { log::info "Deploying with Helm..."; }
vrooli_deploy::kubectl_deploy() { log::info "Deploying with kubectl..."; }
vrooli_deploy::rollback_k8s() { log::info "Rolling back Kubernetes deployment..."; }
vrooli_deploy::update_systemd_services() { log::info "Updating systemd services..."; }
vrooli_deploy::rollback_native() { log::info "Rolling back native deployment..."; }

# Only run main if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    vrooli_deploy::main "$@"
fi