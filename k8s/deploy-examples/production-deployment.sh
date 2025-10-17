#!/usr/bin/env bash
# Example production deployment script with parameterized values
set -euo pipefail

# Production configuration - CUSTOMIZE THESE VALUES
PRODUCTION_DOMAIN="your-production-domain.com"
VAULT_ADDRESS="https://vault.your-company.com"
TLS_SECRET_NAME="vrooli-tls-secret"
ENVIRONMENT="prod"
VERSION=""  # Leave empty to use version from package.json

# Kubernetes configuration
KUBECONFIG_PATH="$HOME/.kube/config"
NAMESPACE="production"

# Image registry configuration
DOCKER_REGISTRY="docker.io/your-dockerhub-username"

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Deploy Vrooli to production Kubernetes environment with custom configuration.

REQUIRED CUSTOMIZATION:
    Edit this script to set your production values:
    - PRODUCTION_DOMAIN: Your production domain name
    - VAULT_ADDRESS: Your Vault server address
    - TLS_SECRET_NAME: Name of your TLS certificate secret
    - DOCKER_REGISTRY: Your Docker registry URL

OPTIONS:
    -d, --domain DOMAIN         Production domain (overrides script default)
    -v, --vault-addr ADDR       Vault server address (overrides script default)
    -t, --tls-secret NAME       TLS secret name (overrides script default)
    -e, --environment ENV       Environment name (default: prod)
    --version VERSION          Version to deploy (default: from package.json)
    --namespace NAMESPACE      Kubernetes namespace (default: production)
    --dry-run                 Show Helm template without deploying
    -h, --help                Show this help message

EXAMPLES:
    # Deploy with custom domain
    $0 --domain myapp.example.com

    # Deploy specific version
    $0 --version 1.2.3

    # Dry run to see generated manifests
    $0 --dry-run

    # Deploy to staging
    $0 --environment staging --namespace staging

PREREQUISITES:
    1. Kubernetes cluster access configured
    2. Helm 3.x installed
    3. Docker images pushed to registry
    4. Vault configured with production secrets
    5. TLS certificate available as Kubernetes secret
    6. Required operators installed (PGO, Redis, VSO)

EOF
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -d|--domain)
                PRODUCTION_DOMAIN="$2"
                shift 2
                ;;
            -v|--vault-addr)
                VAULT_ADDRESS="$2"
                shift 2
                ;;
            -t|--tls-secret)
                TLS_SECRET_NAME="$2"
                shift 2
                ;;
            -e|--environment)
                ENVIRONMENT="$2"
                shift 2
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

validate_configuration() {
    echo "🔍 Validating configuration..."
    
    # Check for placeholder values
    if [[ "$PRODUCTION_DOMAIN" == "your-production-domain.com" ]]; then
        echo "❌ ERROR: Please customize PRODUCTION_DOMAIN in this script"
        echo "   Current value: $PRODUCTION_DOMAIN"
        exit 1
    fi
    
    if [[ "$VAULT_ADDRESS" == "https://vault.your-company.com" ]]; then
        echo "❌ ERROR: Please customize VAULT_ADDRESS in this script"
        echo "   Current value: $VAULT_ADDRESS"
        exit 1
    fi
    
    # Check prerequisites
    if ! command -v helm >/dev/null 2>&1; then
        echo "❌ ERROR: helm not found. Please install Helm 3.x"
        exit 1
    fi
    
    if ! command -v kubectl >/dev/null 2>&1; then
        echo "❌ ERROR: kubectl not found. Please install kubectl"
        exit 1
    fi
    
    # Test cluster connectivity
    if ! kubectl cluster-info >/dev/null 2>&1; then
        echo "❌ ERROR: Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    echo "✅ Configuration validated"
}

get_version() {
    if [[ -n "$VERSION" ]]; then
        echo "$VERSION"
    else
        # Get version from package.json
        if [[ -f "package.json" ]]; then
            node -p "require('./package.json').version"
        else
            echo "❌ ERROR: No package.json found and no version specified"
            exit 1
        fi
    fi
}

deploy_to_kubernetes() {
    local deploy_version
    deploy_version=$(get_version)
    
    local release_name="vrooli-$ENVIRONMENT"
    local chart_path="k8s/chart"
    local values_file="k8s/chart/values-${ENVIRONMENT}.yaml"
    
    echo "🚀 Deploying Vrooli to Kubernetes"
    echo "   Environment: $ENVIRONMENT"
    echo "   Namespace: $NAMESPACE"
    echo "   Version: $deploy_version"
    echo "   Domain: $PRODUCTION_DOMAIN"
    echo "   Vault: $VAULT_ADDRESS"
    
    # Verify chart exists
    if [[ ! -f "$chart_path/Chart.yaml" ]]; then
        echo "❌ ERROR: Helm chart not found at $chart_path"
        exit 1
    fi
    
    # Verify values file exists
    if [[ ! -f "$values_file" ]]; then
        echo "❌ ERROR: Values file not found at $values_file"
        exit 1
    fi
    
    # Lint the chart
    echo "🔍 Linting Helm chart..."
    if ! helm lint "$chart_path" -f "$values_file"; then
        echo "❌ ERROR: Helm chart linting failed"
        exit 1
    fi
    
    # Build Helm command
    local helm_cmd=(
        helm upgrade --install "$release_name" "$chart_path"
        --namespace "$NAMESPACE"
        --create-namespace
        --values "$values_file"
        --set "productionDomain=$PRODUCTION_DOMAIN"
        --set "vaultAddr=$VAULT_ADDRESS"
        --set "tlsSecretName=$TLS_SECRET_NAME"
        --set "services.ui.tag=$deploy_version"
        --set "services.server.tag=$deploy_version"
        --set "services.jobs.tag=$deploy_version"
        --atomic
        --timeout 10m
    )
    
    # Add dry-run if specified
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        helm_cmd+=(--dry-run --debug)
        echo "🔍 Dry run - showing generated manifests:"
    fi
    
    # Execute deployment
    echo "📦 Executing Helm deployment..."
    "${helm_cmd[@]}"
    
    if [[ "${DRY_RUN:-false}" != "true" ]]; then
        echo "✅ Deployment completed!"
        
        # Show deployment status
        echo ""
        echo "📊 Deployment Status:"
        kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/instance=$release_name"
        
        echo ""
        echo "🌐 Services:"
        kubectl get services -n "$NAMESPACE" -l "app.kubernetes.io/instance=$release_name"
        
        if kubectl get ingress -n "$NAMESPACE" >/dev/null 2>&1; then
            echo ""
            echo "🔗 Ingress:"
            kubectl get ingress -n "$NAMESPACE"
        fi
        
        echo ""
        echo "🎉 Deployment Summary:"
        echo "   Application: $release_name"
        echo "   Namespace: $NAMESPACE"
        echo "   Version: $deploy_version"
        echo "   URL: https://$PRODUCTION_DOMAIN"
        echo ""
        echo "📋 Next Steps:"
        echo "   1. Verify DNS points to your ingress controller"
        echo "   2. Check SSL certificate status"
        echo "   3. Monitor application logs: kubectl logs -f deployment/$release_name-server -n $NAMESPACE"
        echo "   4. Run health checks on https://$PRODUCTION_DOMAIN/healthcheck"
    fi
}

main() {
    parse_arguments "$@"
    validate_configuration
    deploy_to_kubernetes
}

# Run main function
main "$@"