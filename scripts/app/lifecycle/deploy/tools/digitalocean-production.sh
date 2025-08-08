#!/usr/bin/env bash
# Production deployment script for DigitalOcean Kubernetes
set -euo pipefail

APP_LIFECYCLE_DEPLOY_TOOLS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEPLOY_TOOLS_DIR}/../../../../lib/utils/var.sh"

# Now use the variables for cleaner paths
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/env.sh"

PROJECT_ROOT="${var_ROOT_DIR}"

# Configuration
CLUSTER_NAME="vrooli-prod"
REGION="nyc1"
NODE_SIZE="s-2vcpu-4gb"
NODE_COUNT=3
NAMESPACE="vrooli"

# Load production environment
env::load_secrets "${PROJECT_ROOT}/.env-prod"

deploy::check_prerequisites() {
    log::header "Checking prerequisites..."
    
    # Check if doctl is installed
    if ! command -v doctl &> /dev/null; then
        log::error "doctl CLI not found. Installing..."
        curl -sL https://github.com/digitalocean/doctl/releases/download/v1.104.0/doctl-1.104.0-linux-amd64.tar.gz | tar -xzv
        sudo mv doctl /usr/local/bin
        log::success "doctl installed successfully"
    fi
    
    # Check if helm is installed
    if ! command -v helm &> /dev/null; then
        log::error "helm not found. Please install helm first."
        return 1
    fi
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log::error "kubectl not found. Please install kubectl first."
        return 1
    fi
    
    log::success "Prerequisites check completed"
}

deploy::authenticate() {
    log::header "Authenticating with DigitalOcean..."
    
    if [[ -z "${DIGITALOCEAN_TOKEN:-}" ]]; then
        log::error "DIGITALOCEAN_TOKEN environment variable not set"
        log::info "Please get your API token from: https://cloud.digitalocean.com/account/api/tokens"
        log::info "Then run: export DIGITALOCEAN_TOKEN=your_token_here"
        return 1
    fi
    
    echo "${DIGITALOCEAN_TOKEN}" | doctl auth init --access-token-stdin
    log::success "Authenticated with DigitalOcean"
}

deploy::create_cluster() {
    log::header "Creating DigitalOcean Kubernetes cluster..."
    
    # Check if cluster already exists
    if doctl kubernetes cluster get "${CLUSTER_NAME}" &> /dev/null; then
        log::info "Cluster ${CLUSTER_NAME} already exists"
        doctl kubernetes cluster kubeconfig save "${CLUSTER_NAME}"
        return 0
    fi
    
    log::info "Creating cluster: ${CLUSTER_NAME}"
    doctl kubernetes cluster create "${CLUSTER_NAME}" \
        --region "${REGION}" \
        --size "${NODE_SIZE}" \
        --count "${NODE_COUNT}" \
        --auto-upgrade=true \
        --maintenance-window="sunday=02:00" \
        --surge-upgrade=true \
        --ha=true \
        --wait
    
    # Save kubeconfig
    doctl kubernetes cluster kubeconfig save "${CLUSTER_NAME}"
    
    log::success "Cluster created successfully"
}

deploy::install_operators() {
    log::header "Installing required operators..."
    
    # Install cert-manager for SSL certificates
    log::info "Installing cert-manager..."
    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml
    
    # Wait for cert-manager to be ready
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=cert-manager -n cert-manager --timeout=300s
    
    # Install nginx-ingress for load balancing
    log::info "Installing nginx-ingress..."
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/do/deploy.yaml
    
    # Wait for ingress controller to get external IP
    log::info "Waiting for load balancer to get external IP..."
    local external_ip=""
    local attempts=0
    while [[ -z "${external_ip}" && ${attempts} -lt 30 ]]; do
        external_ip=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
        if [[ -z "${external_ip}" ]]; then
            log::info "Waiting for external IP... (attempt $((attempts + 1))/30)"
            sleep 10
            ((attempts++))
        fi
    done
    
    if [[ -n "${external_ip}" ]]; then
        log::success "Load balancer external IP: ${external_ip}"
        log::info "Configure DNS A records:"
        log::info "  vrooli.com -> ${external_ip}"
        log::info "  www.vrooli.com -> ${external_ip}"
        log::info "  app.vrooli.com -> ${external_ip}"
    else
        log::warning "Could not get external IP. Check load balancer status manually."
    fi
}

deploy::create_ssl_issuer() {
    log::header "Creating SSL certificate issuer..."
    
    cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: ${LETSENCRYPT_EMAIL}
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
    
    # Create namespace
    kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
    
    # Create database secret
    kubectl create secret generic postgres-credentials \
        --namespace="${NAMESPACE}" \
        --from-literal=username="${DB_USER}" \
        --from-literal=password="${DB_PASSWORD}" \
        --from-literal=database="${DB_NAME}" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Create Redis secret
    kubectl create secret generic redis-credentials \
        --namespace="${NAMESPACE}" \
        --from-literal=password="${REDIS_PASSWORD}" \
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
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Create Docker Hub pull secret
    kubectl create secret docker-registry dockerhub-secret \
        --namespace="${NAMESPACE}" \
        --docker-server=https://index.docker.io/v1/ \
        --docker-username="${DOCKERHUB_USERNAME}" \
        --docker-password="${DOCKERHUB_TOKEN}" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    log::success "Secrets created successfully"
}

deploy::create_databases() {
    log::header "Setting up managed databases..."
    
    log::info "Creating managed PostgreSQL database..."
    if ! doctl databases list | grep -q "vrooli-postgres"; then
        doctl databases create vrooli-postgres \
            --engine pg \
            --size db-s-1vcpu-1gb \
            --region "${REGION}" \
            --num-nodes 1
        
        log::info "Waiting for PostgreSQL database to be ready..."
        sleep 60
    fi
    
    log::info "Creating managed Redis database..."
    if ! doctl databases list | grep -q "vrooli-redis"; then
        doctl databases create vrooli-redis \
            --engine redis \
            --size db-s-1vcpu-1gb \
            --region "${REGION}" \
            --num-nodes 1
        
        log::info "Waiting for Redis database to be ready..."
        sleep 60
    fi
    
    # Get database connection info
    local pg_info=$(doctl databases connection vrooli-postgres --format Host,Port,User,Password,Database --no-header)
    local redis_info=$(doctl databases connection vrooli-redis --format Host,Port,Password --no-header)
    
    log::success "Managed databases created"
    log::info "PostgreSQL: ${pg_info}"
    log::info "Redis: ${redis_info}"
}

deploy::deploy_application() {
    log::header "Deploying Vrooli application..."
    
    # Create production values file
    cat > "${PROJECT_ROOT}/k8s/chart/values-production.yaml" <<EOF
# Production values for Vrooli
replicaCount:
  ui: 2
  server: 3
  jobs: 2

image:
  registry: docker.io/matthalloran8
  tag: "2.0.0"
  pullPolicy: Always

imagePullSecrets:
  - name: dockerhub-secret

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  hosts:
    - host: vrooli.com
      paths:
        - path: /
          pathType: Prefix
          service:
            name: vrooli-ui
            port: 3000
        - path: /api
          pathType: Prefix
          service:
            name: vrooli-server
            port: 5329
    - host: www.vrooli.com
      paths:
        - path: /
          pathType: Prefix
          service:
            name: vrooli-ui
            port: 3000
  tls:
    - secretName: vrooli-tls
      hosts:
        - vrooli.com
        - www.vrooli.com

# Use managed databases (will be configured with actual connection strings)
postgresql:
  enabled: false

redis:
  enabled: false

# Resource limits for production
resources:
  ui:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
  server:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 1Gi
  jobs:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

# Horizontal Pod Autoscaling
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
EOF

    # Deploy with Helm
    helm upgrade --install vrooli "${PROJECT_ROOT}/k8s/chart" \
        --namespace "${NAMESPACE}" \
        --values "${PROJECT_ROOT}/k8s/chart/values-production.yaml" \
        --wait \
        --timeout 10m
    
    log::success "Application deployed successfully"
}

deploy::verify_deployment() {
    log::header "Verifying deployment..."
    
    # Check pod status
    kubectl get pods -n "${NAMESPACE}"
    
    # Check ingress status
    kubectl get ingress -n "${NAMESPACE}"
    
    # Check certificate status
    kubectl get certificates -n "${NAMESPACE}"
    
    # Get external IP
    local external_ip=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
    
    log::success "Deployment verification completed"
    log::info "External IP: ${external_ip}"
    log::info "Application should be accessible at: https://vrooli.com"
}

deploy::main() {
    log::info "Starting DigitalOcean production deployment..."
    
    deploy::check_prerequisites
    deploy::authenticate
    deploy::create_cluster
    deploy::install_operators
    deploy::create_ssl_issuer
    deploy::create_secrets
    deploy::create_databases
    deploy::deploy_application
    deploy::verify_deployment
    
    log::success "🎉 Production deployment completed successfully!"
    log::info "Next steps:"
    log::info "1. Configure DNS A records to point to the load balancer IP"
    log::info "2. Wait for SSL certificates to be issued (may take a few minutes)"
    log::info "3. Test the application at https://vrooli.com"
}

# Check if script is being run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    deploy::main "$@"
fi