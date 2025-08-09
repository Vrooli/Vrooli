#!/usr/bin/env bash
# Deploy Vrooli to an existing Kubernetes cluster
set -euo pipefail

APP_LIFECYCLE_DEPLOY_TOOLS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEPLOY_TOOLS_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

PROJECT_ROOT="${var_ROOT_DIR}"

# Configuration
NAMESPACE="vrooli"
HELM_RELEASE="vrooli"

deploy::check_prerequisites() {
    log::header "Checking prerequisites..."
    
    # Check if kubectl is connected to a cluster
    if ! kubectl cluster-info &> /dev/null; then
        log::error "kubectl is not connected to a cluster"
        log::info "Please ensure you have a Kubernetes cluster running and kubectl configured"
        return 1
    fi
    
    # Check if helm is installed
    if ! command -v helm &> /dev/null; then
        log::error "helm not found. Please install helm first."
        return 1
    fi
    
    # Show current cluster context
    local current_context
    current_context=$(kubectl config current-context 2>/dev/null || echo "none")
    log::info "Current Kubernetes context: ${current_context}"
    
    # Show cluster info
    log::info "Cluster information:"
    kubectl cluster-info
    
    log::success "Prerequisites check completed"
}

deploy::confirm_cluster() {
    log::header "Cluster Confirmation"
    
    # Show nodes
    echo "Current cluster nodes:"
    kubectl get nodes -o wide
    
    # Show existing namespaces
    echo ""
    echo "Existing namespaces:"
    kubectl get namespaces
    
    # Ask for confirmation
    echo ""
    read -p "🤔 Deploy Vrooli to this cluster? (y/N): " -r response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        log::info "Deployment cancelled by user"
        exit 0
    fi
}

deploy::create_namespace() {
    log::header "Creating namespace..."
    
    kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
    log::success "Namespace '${NAMESPACE}' ready"
}

deploy::create_ssl_issuer() {
    log::header "Creating SSL certificate issuer..."
    
    # Check if cert-manager is installed
    if ! kubectl get crd clusterissuers.cert-manager.io &> /dev/null; then
        log::warning "cert-manager not found. Installing..."
        kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml
        
        log::info "Waiting for cert-manager to be ready..."
        kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=cert-manager -n cert-manager --timeout=300s
    fi
    
    # Create ClusterIssuer for Let's Encrypt
    cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: matthalloran8@gmail.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
    
    log::success "SSL certificate issuer created"
}

deploy::create_secrets() {
    log::header "Creating Kubernetes secrets..."
    
    # Set environment if not already set
    if [[ -z "${ENVIRONMENT:-}" ]]; then
        read -p "Use production environment? (Y/n): " -r env_choice
        if [[ "$env_choice" =~ ^[Nn]$ ]]; then
            read -p "Enter environment (development/staging/production): " -r ENVIRONMENT
        else
            ENVIRONMENT="production"
        fi
        export ENVIRONMENT
        log::info "Environment set to: ${ENVIRONMENT}"
    fi
    
    # Load environment variables using the project's env script
    if [[ -f "${PROJECT_ROOT}/scripts/app/utils/env.sh" ]]; then
        source "${PROJECT_ROOT}/scripts/app/utils/env.sh"
        env::load_secrets "${PROJECT_ROOT}/.env-prod"
        log::info "Loaded environment variables and JWT keys"
    else
        log::error "env.sh script not found"
        return 1
    fi
    
    # Create Docker Hub pull secret
    kubectl create secret docker-registry dockerhub-secret \
        --namespace="${NAMESPACE}" \
        --docker-server=https://index.docker.io/v1/ \
        --docker-username="${DOCKERHUB_USERNAME}" \
        --docker-password="${DOCKERHUB_TOKEN}" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Create application secrets
    kubectl create secret generic vrooli-secrets \
        --namespace="${NAMESPACE}" \
        --from-literal=jwt-private="${JWT_PRIV}" \
        --from-literal=jwt-public="${JWT_PUB}" \
        --from-literal=openai-api-key="${OPENAI_API_KEY}" \
        --from-literal=anthropic-api-key="${ANTHROPIC_API_KEY}" \
        --from-literal=stripe-secret-key="${STRIPE_SECRET_KEY}" \
        --from-literal=admin-password="${ADMIN_PASSWORD}" \
        --from-literal=external-site-key="${EXTERNAL_SITE_KEY}" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    log::success "Application secrets created"
}

deploy::setup_database_secrets() {
    log::header "Setting up database connection..."
    
    # Check if user wants to use external databases
    echo "Database Configuration Options:"
    echo "1. Use existing DigitalOcean managed databases"
    echo "2. Use simple in-cluster databases (for testing)"
    echo ""
    read -p "Choose option (1 or 2): " -r db_option
    
    case $db_option in
        1)
            deploy::setup_external_databases
            ;;
        2)
            deploy::setup_internal_databases
            ;;
        *)
            log::error "Invalid option selected"
            return 1
            ;;
    esac
}

deploy::setup_external_databases() {
    log::info "Setting up external database connections..."
    
    echo "Please provide your DigitalOcean database connection details:"
    echo ""
    
    # Get PostgreSQL details
    read -p "PostgreSQL Host: " -r PG_HOST
    read -p "PostgreSQL Port (default 25060): " -r PG_PORT
    PG_PORT=${PG_PORT:-25060}
    read -p "PostgreSQL User (default doadmin): " -r PG_USER
    PG_USER=${PG_USER:-doadmin}
    read -p "PostgreSQL Password: " -r PG_PASS
    read -p "PostgreSQL Database (default defaultdb): " -r PG_DB
    PG_DB=${PG_DB:-defaultdb}
    
    # Get Redis details
    echo ""
    read -p "Redis Host: " -r REDIS_HOST
    read -p "Redis Port (default 25061): " -r REDIS_PORT
    REDIS_PORT=${REDIS_PORT:-25061}
    read -p "Redis Password: " -r REDIS_PASS
    
    # Create database secrets
    kubectl create secret generic postgres-credentials \
        --namespace="${NAMESPACE}" \
        --from-literal=host="${PG_HOST}" \
        --from-literal=port="${PG_PORT}" \
        --from-literal=username="${PG_USER}" \
        --from-literal=password="${PG_PASS}" \
        --from-literal=database="${PG_DB}" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    kubectl create secret generic redis-credentials \
        --namespace="${NAMESPACE}" \
        --from-literal=host="${REDIS_HOST}" \
        --from-literal=port="${REDIS_PORT}" \
        --from-literal=password="${REDIS_PASS}" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Set deployment to use external databases
    EXTERNAL_DB_ENABLED="true"
    
    log::success "External database secrets created"
}

deploy::setup_internal_databases() {
    log::info "Will deploy simple in-cluster databases"
    
    # Create simple database credentials
    kubectl create secret generic postgres-credentials \
        --namespace="${NAMESPACE}" \
        --from-literal=username="site" \
        --from-literal=password="YnrMnA2DTcN3ae" \
        --from-literal=database="vrooli" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    kubectl create secret generic redis-credentials \
        --namespace="${NAMESPACE}" \
        --from-literal=password="rDEa9XUXgRzUNxv2WXWcKE3y" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Set deployment to use internal databases
    EXTERNAL_DB_ENABLED="false"
    
    log::success "Internal database secrets created"
}

deploy::create_deployment_values() {
    log::header "Creating deployment configuration..."
    
    # Create custom values file for this deployment
    cat > "${PROJECT_ROOT}/k8s/chart/values-current-deployment.yaml" <<EOF
# Auto-generated values for current deployment
nameOverride: ""
fullnameOverride: "vrooli"

image:
  registry: "docker.io/matthalloran8"
  tag: "2.0.0"
  pullPolicy: Always

imagePullSecrets:
  - name: "dockerhub-secret"

replicaCount:
  ui: 2
  server: 2
  jobs: 1

services:
  ui:
    type: ClusterIP
    port: 3000
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
      limits:
        cpu: "500m"
        memory: "512Mi"
    
  server:
    type: ClusterIP
    port: 5329
    env:
      NODE_ENV: "production"
      PORT_API: "5329"
      API_URL: "https://vrooli.com/api"
      UI_URL: "https://vrooli.com"
      CREATE_MOCK_DATA: "false"
    resources:
      requests:
        cpu: "200m"
        memory: "256Mi"
      limits:
        cpu: "1000m"
        memory: "1Gi"
    
  jobs:
    type: ClusterIP
    port: 4001
    env:
      NODE_ENV: "production"
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
      limits:
        cpu: "500m"
        memory: "512Mi"

# External database configuration
postgresql:
  enabled: $([ "$EXTERNAL_DB_ENABLED" = "true" ] && echo "false" || echo "true")

redis:
  enabled: $([ "$EXTERNAL_DB_ENABLED" = "true" ] && echo "false" || echo "true")

# Disable operator CRDs since we're using managed databases
pgoPostgresql:
  enabled: false

spotahomeRedis:
  enabled: false

vso:
  enabled: false

externalDatabase:
  enabled: ${EXTERNAL_DB_ENABLED}
  existingSecret: "postgres-credentials"

externalRedis:
  enabled: ${EXTERNAL_DB_ENABLED}
  existingSecret: "redis-credentials"

# Ingress configuration
ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
  hosts:
    - host: vrooli.com
      paths:
        - path: /api
          pathType: Prefix
          service: server
          port: 5329
        - path: /
          pathType: Prefix
          service: ui
          port: 3000
  tls:
    - secretName: vrooli-tls
      hosts:
        - vrooli.com

# Security
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000

# Health checks
healthChecks:
  enabled: true
  livenessProbe:
    initialDelaySeconds: 30
    periodSeconds: 10
  readinessProbe:
    initialDelaySeconds: 15
    periodSeconds: 5
EOF
    
    log::success "Deployment configuration created"
}

deploy::deploy_application() {
    log::header "Deploying Vrooli application..."
    
    # Deploy with Helm
    helm upgrade --install "${HELM_RELEASE}" "${PROJECT_ROOT}/k8s/chart" \
        --namespace "${NAMESPACE}" \
        --values "${PROJECT_ROOT}/k8s/chart/values-current-deployment.yaml" \
        --wait \
        --timeout 10m
    
    log::success "Application deployed successfully"
}

deploy::verify_deployment() {
    log::header "Verifying deployment..."
    
    # Show pod status
    echo "Pod status:"
    kubectl get pods -n "${NAMESPACE}" -o wide
    
    echo ""
    echo "Service status:"
    kubectl get svc -n "${NAMESPACE}"
    
    echo ""
    echo "Ingress status:"
    kubectl get ingress -n "${NAMESPACE}"
    
    # Check for external IP
    local external_ip=""
    if kubectl get svc -n ingress-nginx ingress-nginx-controller &> /dev/null; then
        external_ip=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    fi
    
    echo ""
    if [[ -n "${external_ip}" ]]; then
        log::success "External IP: ${external_ip}"
        log::info "Configure DNS A record: vrooli.com -> ${external_ip}"
    else
        log::warning "No external IP found. Check ingress controller status."
    fi
    
    echo ""
    echo "Certificate status:"
    kubectl get certificates -n "${NAMESPACE}" 2>/dev/null || echo "No certificates found yet"
    
    log::success "Deployment verification completed"
}

deploy::show_next_steps() {
    log::header "Next Steps"
    
    echo "📋 Post-Deployment Tasks:"
    echo "========================="
    echo ""
    echo "1. 🌐 Configure DNS (if not done already):"
    local external_ip=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "EXTERNAL_IP")
    echo "   Create A record: vrooli.com -> ${external_ip}"
    echo ""
    echo "2. 🔒 Wait for SSL certificate (5-10 minutes after DNS):"
    echo "   kubectl get certificates -n ${NAMESPACE}"
    echo ""
    echo "3. 🧪 Test the application:"
    echo "   curl -I https://vrooli.com"
    echo ""
    echo "4. 📊 Monitor the deployment:"
    echo "   kubectl get pods -n ${NAMESPACE} -w"
    echo "   kubectl logs -f deployment/vrooli-server -n ${NAMESPACE}"
    echo ""
    echo "5. 🔧 If issues occur:"
    echo "   kubectl describe pods -n ${NAMESPACE}"
    echo "   kubectl logs deployment/vrooli-server -n ${NAMESPACE}"
    echo ""
    echo "6. 🗑️  To remove the deployment:"
    echo "   helm uninstall ${HELM_RELEASE} -n ${NAMESPACE}"
    echo "   kubectl delete namespace ${NAMESPACE}"
}

deploy::main() {
    log::info "Starting deployment to existing Kubernetes cluster..."
    
    deploy::check_prerequisites
    deploy::confirm_cluster
    deploy::create_namespace
    deploy::create_ssl_issuer
    deploy::create_secrets
    deploy::setup_database_secrets
    deploy::create_deployment_values
    deploy::deploy_application
    deploy::verify_deployment
    deploy::show_next_steps
    
    log::success "🎉 Deployment completed!"
    log::info "Application should be accessible at https://vrooli.com once DNS is configured"
}

# Check if script is being run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    deploy::main "$@"
fi