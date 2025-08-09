#!/usr/bin/env bash
# Enhanced Kubernetes Dependencies Checker
# Validates all external dependencies required for Vrooli deployment
set -euo pipefail

APP_LIFECYCLE_DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEPLOY_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args.sh"

# Parse command line arguments
k8s_dependencies::parse_arguments() {
    args::reset
    
    args::register_help
    
    args::register \
        --name "check-only" \
        --flag "c" \
        --desc "Only check dependencies, don't offer fixes" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "fix-issues" \
        --flag "f" \
        --desc "Automatically fix issues where possible" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "environment" \
        --flag "e" \
        --desc "Target environment (dev, staging, prod)" \
        --type "value" \
        --options "dev|staging|prod|production" \
        --default "dev"
    
    if args::is_asking_for_help "$@"; then
        args::usage
        exit 0
    fi
    
    args::parse "$@"
}

# Check if all required operators are installed and healthy
k8s_dependencies::check_operators() {
    log::header "Checking Kubernetes operator dependencies..."
    
    local issues=0
    local operator_status=""
    
    # Check PGO
    if kubectl get crd postgresclusters.postgres-operator.crunchydata.com > /dev/null 2>&1; then
        # Check if operator pods are running
        if kubectl get pods -n postgres-operator -l postgres-operator.crunchydata.com/control-plane=postgres-operator --field-selector=status.phase=Running | grep -q Running; then
            log::success "✓ CrunchyData PostgreSQL Operator (running)"
            operator_status+="PGO: ✓ Installed and running\n"
        else
            log::warning "⚠ CrunchyData PostgreSQL Operator installed but not running"
            operator_status+="PGO: ⚠ Installed but pods not running\n"
            ((issues++))
        fi
    else
        log::error "✗ CrunchyData PostgreSQL Operator NOT installed"
        operator_status+="PGO: ✗ Missing - Install with: bash scripts/app/lifecycle/setup/target/k8s_cluster.sh\n"
        ((issues++))
    fi
    
    # Check Redis Operator
    if kubectl get crd redisfailovers.databases.spotahome.com > /dev/null 2>&1; then
        if kubectl get pods -n redis-operator -l app.kubernetes.io/name=redis-operator --field-selector=status.phase=Running | grep -q Running; then
            log::success "✓ Spotahome Redis Operator (running)"
            operator_status+="Redis Operator: ✓ Installed and running\n"
        else
            log::warning "⚠ Spotahome Redis Operator installed but not running"
            operator_status+="Redis Operator: ⚠ Installed but pods not running\n"
            ((issues++))
        fi
    else
        log::error "✗ Spotahome Redis Operator NOT installed"
        operator_status+="Redis Operator: ✗ Missing - Install with: bash scripts/app/lifecycle/setup/target/k8s_cluster.sh\n"
        ((issues++))
    fi
    
    # Check VSO
    if kubectl get crd vaultsecrets.secrets.hashicorp.com > /dev/null 2>&1; then
        if kubectl get pods -n vault-secrets-operator-system -l app.kubernetes.io/name=vault-secrets-operator --field-selector=status.phase=Running | grep -q Running; then
            log::success "✓ Vault Secrets Operator (running)"
            operator_status+="VSO: ✓ Installed and running\n"
        else
            log::warning "⚠ Vault Secrets Operator installed but not running"
            operator_status+="VSO: ⚠ Installed but pods not running\n"
            ((issues++))
        fi
    else
        log::error "✗ Vault Secrets Operator NOT installed"
        operator_status+="VSO: ✗ Missing - Install with: bash scripts/app/lifecycle/setup/target/k8s_cluster.sh\n"
        ((issues++))
    fi
    
    echo -e "$operator_status"
    return "$issues"
}

# Check Vault connectivity and configuration
k8s_dependencies::check_vault() {
    log::header "Checking Vault dependencies..."
    
    local issues=0
    local vault_status=""
    local environment
    environment=$(args::get "environment")
    
    # Different Vault checks based on environment
    if [[ "$environment" == "dev" ]]; then
        # Development: Check in-cluster Vault
        local vault_address="http://vault.vault.svc.cluster.local:8200"
        
        # Check if Vault pods are running
        if kubectl get pods -n vault -l app.kubernetes.io/name=vault --field-selector=status.phase=Running | grep -q Running; then
            log::success "✓ Vault pod running"
            vault_status+="Vault Pod: ✓ Running\n"
            
            # Test connectivity
            if kubectl exec -n vault vault-0 -- wget -qO- "${vault_address}/v1/sys/health" > /dev/null 2>&1; then
                log::success "✓ Vault API accessible"
                vault_status+="Vault API: ✓ Accessible\n"
                
                # Check if KV secrets engine is enabled
                if kubectl exec -n vault vault-0 -- vault secrets list -format=json | jq -e '.["secret/"]' > /dev/null 2>&1; then
                    log::success "✓ KV secrets engine enabled"
                    vault_status+="KV Engine: ✓ Enabled at secret/\n"
                else
                    log::warning "⚠ KV secrets engine not enabled at secret/"
                    vault_status+="KV Engine: ⚠ Not enabled - Run: kubectl exec -n vault vault-0 -- vault secrets enable -path=secret kv-v2\n"
                    ((issues++))
                fi
                
                # Check Kubernetes auth
                if kubectl exec -n vault vault-0 -- vault auth list -format=json | jq -e '.["kubernetes/"]' > /dev/null 2>&1; then
                    log::success "✓ Kubernetes auth enabled"
                    vault_status+="K8s Auth: ✓ Enabled\n"
                else
                    log::warning "⚠ Kubernetes auth not enabled"
                    vault_status+="K8s Auth: ⚠ Not enabled - Run Vault configuration script\n"
                    ((issues++))
                fi
            else
                log::error "✗ Cannot reach Vault API"
                vault_status+="Vault API: ✗ Not reachable\n"
                ((issues++))
            fi
        else
            log::error "✗ Vault pod not running"
            vault_status+="Vault Pod: ✗ Not running - Install with SECRETS_SOURCE=vault\n"
            ((issues++))
        fi
    else
        # Production: Check external Vault
        local vault_address="${VAULT_ADDR:-}"
        if [[ -z "$vault_address" ]]; then
            log::error "✗ VAULT_ADDR not set for production environment"
            vault_status+="Vault Address: ✗ VAULT_ADDR environment variable required\n"
            ((issues++))
        else
            log::info "Checking external Vault at: $vault_address"
            if curl -s --connect-timeout 5 "${vault_address}/v1/sys/health" > /dev/null 2>&1; then
                log::success "✓ External Vault accessible"
                vault_status+="External Vault: ✓ Accessible at $vault_address\n"
            else
                log::error "✗ Cannot reach external Vault"
                vault_status+="External Vault: ✗ Not reachable at $vault_address\n"
                ((issues++))
            fi
        fi
    fi
    
    echo -e "$vault_status"
    return "$issues"
}

# Check container registry access
k8s_dependencies::check_registry() {
    log::header "Checking container registry dependencies..."
    
    local issues=0
    local registry_status=""
    
    # Check if image pull secrets exist
    local pull_secret_name="vrooli-dockerhub-pull-secret"
    local environment
    environment=$(args::get "environment")
    
    if kubectl get secret "$pull_secret_name" -n "$environment" > /dev/null 2>&1; then
        log::success "✓ Image pull secret exists"
        registry_status+="Pull Secret: ✓ $pull_secret_name exists\n"
        
        # Validate secret format
        if kubectl get secret "$pull_secret_name" -n "$environment" -o jsonpath='{.type}' | grep -q "kubernetes.io/dockerconfigjson"; then
            log::success "✓ Pull secret has correct type"
            registry_status+="Secret Type: ✓ dockerconfigjson\n"
        else
            log::warning "⚠ Pull secret has incorrect type"
            registry_status+="Secret Type: ⚠ Not dockerconfigjson\n"
            ((issues++))
        fi
    else
        log::warning "⚠ Image pull secret not found"
        registry_status+="Pull Secret: ⚠ Create $pull_secret_name or configure VSO\n"
        ((issues++))
    fi
    
    # Check registry connectivity (if credentials available)
    local registry_host="docker.io"
    log::info "Testing registry connectivity to $registry_host..."
    if nslookup "$registry_host" > /dev/null 2>&1; then
        log::success "✓ Registry DNS resolution"
        registry_status+="Registry DNS: ✓ $registry_host resolvable\n"
    else
        log::warning "⚠ Cannot resolve registry hostname"
        registry_status+="Registry DNS: ⚠ $registry_host not resolvable\n"
        ((issues++))
    fi
    
    echo -e "$registry_status"
    return "$issues"
}

# Check storage classes and persistent volumes
k8s_dependencies::check_storage() {
    log::header "Checking storage dependencies..."
    
    local issues=0
    local storage_status=""
    
    # Check for default storage class
    if kubectl get storageclass | grep -q "(default)"; then
        local default_sc
        default_sc=$(kubectl get storageclass | grep "(default)" | awk '{print $1}')
        log::success "✓ Default storage class: $default_sc"
        storage_status+="Default StorageClass: ✓ $default_sc\n"
    else
        log::warning "⚠ No default storage class found"
        storage_status+="Default StorageClass: ⚠ None - PostgreSQL and Redis may need explicit storageClass\n"
        ((issues++))
    fi
    
    # Check for specific storage classes if mentioned in values
    local storage_classes=("gp2" "standard" "fast-ssd" "local-path")
    for sc in "${storage_classes[@]}"; do
        if kubectl get storageclass "$sc" > /dev/null 2>&1; then
            log::info "✓ Optional storage class available: $sc"
            storage_status+="StorageClass $sc: ✓ Available\n"
        fi
    done
    
    echo -e "$storage_status"
    return "$issues"
}

# Check ingress and networking
k8s_dependencies::check_networking() {
    log::header "Checking networking dependencies..."
    
    local issues=0
    local network_status=""
    local environment
    environment=$(args::get "environment")
    
    # Check for ingress controller (mainly for prod)
    if [[ "$environment" != "dev" ]]; then
        if kubectl get pods -A -l app.kubernetes.io/name=ingress-nginx --field-selector=status.phase=Running | grep -q Running; then
            log::success "✓ Nginx Ingress Controller running"
            network_status+="Ingress Controller: ✓ Nginx running\n"
        elif kubectl get pods -A -l app.kubernetes.io/name=traefik --field-selector=status.phase=Running | grep -q Running; then
            log::success "✓ Traefik Ingress Controller running"
            network_status+="Ingress Controller: ✓ Traefik running\n"
        else
            log::warning "⚠ No ingress controller detected"
            network_status+="Ingress Controller: ⚠ None detected - Install ingress-nginx or traefik\n"
            ((issues++))
        fi
    else
        log::info "Skipping ingress check for dev environment"
        network_status+="Ingress Controller: ✓ Not required for dev\n"
    fi
    
    # Check cluster DNS
    if kubectl exec -n kube-system deployment/coredns -- nslookup kubernetes.default.svc.cluster.local > /dev/null 2>&1; then
        log::success "✓ Cluster DNS working"
        network_status+="Cluster DNS: ✓ Working\n"
    else
        log::error "✗ Cluster DNS not working"
        network_status+="Cluster DNS: ✗ CoreDNS issues\n"
        ((issues++))
    fi
    
    echo -e "$network_status"
    return "$issues"
}

# Generate setup instructions for missing dependencies
k8s_dependencies::generate_setup_instructions() {
    local issues_found="$1"
    
    if [[ "$issues_found" -gt 0 ]]; then
        log::header "🔧 Setup Instructions for Missing Dependencies"
        
        echo ""
        log::info "1. KUBERNETES OPERATORS:"
        echo "   Run the automated installer:"
        echo "   bash scripts/app/lifecycle/deploy/k8s-prerequisites.sh --yes"
        echo ""
        
        log::info "2. VAULT SETUP (Development):"
        echo "   Enable Vault for development:"
        echo "   export SECRETS_SOURCE=vault"
        echo "   bash scripts/manage.sh develop --target k8s-cluster"
        echo ""
        
        log::info "3. VAULT SETUP (Production):"
        echo "   a) Install external Vault or use managed service"
        echo "   b) Configure environment variables:"
        echo "      export VAULT_ADDR=https://your-vault.example.com:8200"
        echo "      export VAULT_TOKEN=your-vault-token"
        echo "   c) Apply Vault policies:"
        echo "      for policy in k8s/dev-support/vault-policies/*.hcl; do"
        echo "        vault policy write \$(basename \$policy .hcl) \$policy"
        echo "      done"
        echo "   d) Enable Kubernetes auth:"
        echo "      vault auth enable kubernetes"
        echo "      vault write auth/kubernetes/config kubernetes_host=\$K8S_API_SERVER"
        echo "   e) Create role:"
        echo "      vault write auth/kubernetes/role/vrooli-vso-sync-role \\"
        echo "        bound_service_account_names=default \\"
        echo "        bound_service_account_namespaces='*' \\"
        echo "        policies=vrooli-config-shared-all-read,vrooli-secrets-shared-server-jobs-read,vrooli-secrets-postgres-read,vrooli-secrets-redis-read,vrooli-secrets-dockerhub-read"
        echo ""
        
        log::info "4. DOCKER REGISTRY:"
        echo "   Create image pull secret manually:"
        echo "   kubectl create secret docker-registry vrooli-dockerhub-pull-secret \\"
        echo "     --docker-server=docker.io \\"
        echo "     --docker-username=your-username \\"
        echo "     --docker-password=your-password \\"
        echo "     --docker-email=your-email@example.com"
        echo ""
        
        log::info "5. STORAGE:"
        echo "   Ensure storage classes are available for your cluster:"
        echo "   kubectl get storageclass"
        echo "   For local development, install local-path-provisioner:"
        echo "   kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.24/deploy/local-path-storage.yaml"
        echo ""
        
        log::info "6. INGRESS (Production):"
        echo "   Install nginx ingress controller:"
        echo "   helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx"
        echo "   helm install ingress-nginx ingress-nginx/ingress-nginx \\"
        echo "     --namespace ingress-nginx --create-namespace"
        echo ""
        
        log::info "Run this script again after resolving issues:"
        echo "bash scripts/app/lifecycle/deploy/k8s-dependencies-check.sh --environment=\$ENVIRONMENT"
        echo ""
    fi
}

# Main function
k8s_dependencies::main() {
    k8s_dependencies::parse_arguments "$@"
    
    local environment
    environment=$(args::get "environment")
    local check_only
    check_only=$(args::get "check-only")
    
    # Check current cluster context
    local current_context
    current_context=$(kubectl config current-context 2>/dev/null || echo "none")
    log::info "Checking dependencies for environment: $environment"
    log::info "Current Kubernetes context: $current_context"
    echo ""
    
    # Perform all checks
    local total_issues=0
    
    k8s_dependencies::check_operators
    total_issues=$((total_issues + $?))
    
    k8s_dependencies::check_vault
    total_issues=$((total_issues + $?))
    
    k8s_dependencies::check_registry
    total_issues=$((total_issues + $?))
    
    k8s_dependencies::check_storage
    total_issues=$((total_issues + $?))
    
    k8s_dependencies::check_networking
    total_issues=$((total_issues + $?))
    
    # Summary and instructions
    echo ""
    if [[ $total_issues -eq 0 ]]; then
        log::success "🎉 All dependencies are properly configured!"
        log::info "You can proceed with deployment:"
        log::info "  bash scripts/manage.sh deploy --source k8s --environment $environment"
        exit 0
    else
        log::warning "⚠ Found $total_issues dependency issues"
        
        if [[ "$check_only" == "yes" ]]; then
            log::error "Dependencies not ready. Fix the issues above and retry."
            exit 1
        else
            k8s_dependencies::generate_setup_instructions "$total_issues"
            exit 1
        fi
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    k8s_dependencies::main "$@"
fi