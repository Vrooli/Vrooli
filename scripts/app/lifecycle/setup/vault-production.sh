#!/usr/bin/env bash
# Production Vault setup script for Vrooli Kubernetes deployment
set -euo pipefail

APP_LIFECYCLE_SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${APP_LIFECYCLE_SETUP_DIR}/../../../lib/utils/var.sh"

# Now use the variables for cleaner paths
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"

# Default values
VAULT_ADDR="${VAULT_ADDR:-https://vault.example.com}"
VAULT_TOKEN="${VAULT_TOKEN:-}"
ENVIRONMENT="${ENVIRONMENT:-prod}"
K8S_NAMESPACE="${K8S_NAMESPACE:-production}"
AUTH_MOUNT_PATH="${AUTH_MOUNT_PATH:-kubernetes-prod}"
VSO_ROLE_NAME="${VSO_ROLE_NAME:-vrooli-vso-sync-role}"
DRY_RUN="${DRY_RUN:-false}"

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Sets up HashiCorp Vault for production Kubernetes deployment.

OPTIONS:
    -a, --vault-addr ADDR       Vault server address (default: $VAULT_ADDR)
    -t, --vault-token TOKEN     Vault authentication token
    -e, --environment ENV       Environment name (default: $ENVIRONMENT)
    -n, --namespace NAMESPACE   Kubernetes namespace (default: $K8S_NAMESPACE)
    -m, --auth-mount PATH       K8s auth mount path (default: $AUTH_MOUNT_PATH)
    -r, --role-name NAME        VSO role name (default: $VSO_ROLE_NAME)
    -d, --dry-run              Show commands without executing
    -h, --help                 Show this help message

EXAMPLES:
    # Setup production Vault with token auth
    $0 --vault-addr https://vault.company.com --vault-token hvs.xxx

    # Dry run to see what would be executed
    $0 --dry-run --vault-addr https://vault.company.com

    # Setup for staging environment
    $0 --environment staging --namespace staging --auth-mount kubernetes-staging

PREREQUISITES:
    - vault CLI installed and in PATH
    - kubectl configured for target cluster
    - Vault admin permissions or equivalent
    - Kubernetes cluster with VSO installed

EOF
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -a|--vault-addr)
                VAULT_ADDR="$2"
                shift 2
                ;;
            -t|--vault-token)
                VAULT_TOKEN="$2"
                shift 2
                ;;
            -e|--environment)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -n|--namespace)
                K8S_NAMESPACE="$2"
                shift 2
                ;;
            -m|--auth-mount)
                AUTH_MOUNT_PATH="$2"
                shift 2
                ;;
            -r|--role-name)
                VSO_ROLE_NAME="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN="true"
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log::error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

validate_prerequisites() {
    log::info "Validating prerequisites..."
    
    # Check vault CLI
    if ! command -v vault >/dev/null 2>&1; then
        log::error "vault CLI not found. Please install HashiCorp Vault CLI."
        exit 1
    fi
    
    # Check kubectl
    if ! command -v kubectl >/dev/null 2>&1; then
        log::error "kubectl not found. Please install kubectl."
        exit 1
    fi
    
    # Validate Vault token
    if [[ -z "$VAULT_TOKEN" ]]; then
        log::error "Vault token required. Set VAULT_TOKEN environment variable or use --vault-token"
        exit 1
    fi
    
    # Test Vault connectivity
    export VAULT_ADDR
    export VAULT_TOKEN
    
    if ! vault status >/dev/null 2>&1; then
        log::error "Cannot connect to Vault at $VAULT_ADDR. Check address and token."
        exit 1
    fi
    
    # Test kubectl connectivity
    if ! kubectl cluster-info >/dev/null 2>&1; then
        log::error "Cannot connect to Kubernetes cluster. Check kubectl configuration."
        exit 1
    fi
    
    log::success "Prerequisites validated"
}

execute_command() {
    local cmd="$1"
    local description="$2"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] $description"
        log::info "[DRY RUN] Command: $cmd"
    else
        log::info "$description"
        eval "$cmd"
    fi
}

create_vault_policies() {
    log::header "Creating Vault policies for environment: $ENVIRONMENT"
    
    local vault_path_prefix
    if [[ "$ENVIRONMENT" == "prod" ]]; then
        vault_path_prefix="vrooli-prod"
    else
        vault_path_prefix="vrooli/$ENVIRONMENT"
    fi
    
    # Policy 1: Shared configuration (non-sensitive)
    execute_command \
        "vault policy write vrooli-${ENVIRONMENT}-config-shared-all-read - <<EOF
path \"secret/data/${vault_path_prefix}/config/shared-all\" {
  capabilities = [\"read\"]
}
EOF" \
        "Creating shared configuration policy"
    
    # Policy 2: Server/Jobs secrets
    execute_command \
        "vault policy write vrooli-${ENVIRONMENT}-secrets-shared-server-jobs-read - <<EOF
path \"secret/data/${vault_path_prefix}/secrets/shared-server-jobs\" {
  capabilities = [\"read\"]
}
EOF" \
        "Creating server/jobs secrets policy"
    
    # Policy 3: PostgreSQL secrets
    execute_command \
        "vault policy write vrooli-${ENVIRONMENT}-secrets-postgres-read - <<EOF
path \"secret/data/${vault_path_prefix}/secrets/postgres\" {
  capabilities = [\"read\"]
}
EOF" \
        "Creating PostgreSQL secrets policy"
    
    # Policy 4: Redis secrets
    execute_command \
        "vault policy write vrooli-${ENVIRONMENT}-secrets-redis-read - <<EOF
path \"secret/data/${vault_path_prefix}/secrets/redis\" {
  capabilities = [\"read\"]
}
EOF" \
        "Creating Redis secrets policy"
    
    # Policy 5: Docker Hub secrets
    execute_command \
        "vault policy write vrooli-${ENVIRONMENT}-secrets-dockerhub-read - <<EOF
path \"secret/data/${vault_path_prefix}/dockerhub/*\" {
  capabilities = [\"read\"]
}
EOF" \
        "Creating Docker Hub secrets policy"
}

setup_kubernetes_auth() {
    log::header "Setting up Kubernetes authentication"
    
    # Enable Kubernetes auth method
    execute_command \
        "vault auth enable -path=$AUTH_MOUNT_PATH kubernetes || true" \
        "Enabling Kubernetes auth method at $AUTH_MOUNT_PATH"
    
    # Get Kubernetes cluster details
    if [[ "$DRY_RUN" == "false" ]]; then
        log::info "Gathering Kubernetes cluster information..."
        K8S_HOST=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.server}')
        K8S_CA_CERT=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}' | base64 -d)
        
        # Try to get service account token (method varies by K8s version)
        if kubectl get secret -n kube-system default-token-* >/dev/null 2>&1; then
            # Older Kubernetes versions with automatic tokens
            TOKEN_REVIEW_JWT=$(kubectl get secret -n kube-system $(kubectl get serviceaccount -n kube-system default -o jsonpath='{.secrets[0].name}') -o jsonpath='{.data.token}' | base64 -d)
        else
            # Newer Kubernetes versions - create a token
            TOKEN_REVIEW_JWT=$(kubectl create token default -n kube-system --duration=8760h)
        fi
        
        log::info "Kubernetes cluster: $K8S_HOST"
    fi
    
    # Configure Kubernetes auth
    execute_command \
        "vault write auth/$AUTH_MOUNT_PATH/config \
            token_reviewer_jwt=\"\$TOKEN_REVIEW_JWT\" \
            kubernetes_host=\"\$K8S_HOST\" \
            kubernetes_ca_cert=\"\$K8S_CA_CERT\"" \
        "Configuring Kubernetes auth method"
}

create_vso_role() {
    log::header "Creating VSO role"
    
    local policies="vrooli-${ENVIRONMENT}-config-shared-all-read,vrooli-${ENVIRONMENT}-secrets-shared-server-jobs-read,vrooli-${ENVIRONMENT}-secrets-postgres-read,vrooli-${ENVIRONMENT}-secrets-redis-read,vrooli-${ENVIRONMENT}-secrets-dockerhub-read"
    
    execute_command \
        "vault write auth/$AUTH_MOUNT_PATH/role/$VSO_ROLE_NAME \
            bound_service_account_names=default \
            bound_service_account_namespaces=$K8S_NAMESPACE \
            policies=$policies \
            ttl=24h" \
        "Creating VSO role: $VSO_ROLE_NAME"
}

verify_setup() {
    log::header "Verifying Vault setup"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Skipping verification in dry-run mode"
        return
    fi
    
    # List policies
    log::info "Checking created policies..."
    vault policy list | grep "vrooli-${ENVIRONMENT}" || log::warning "No vrooli policies found"
    
    # Check auth method
    log::info "Checking Kubernetes auth method..."
    vault auth list | grep "$AUTH_MOUNT_PATH" || log::warning "Kubernetes auth method not found"
    
    # Check role
    log::info "Checking VSO role..."
    vault read "auth/$AUTH_MOUNT_PATH/role/$VSO_ROLE_NAME" >/dev/null 2>&1 && \
        log::success "VSO role verified" || \
        log::warning "VSO role verification failed"
}

print_next_steps() {
    log::header "Next Steps"
    
    cat << EOF

1. POPULATE SECRETS in Vault at these paths:
   $(if [[ "$ENVIRONMENT" == "prod" ]]; then echo "secret/data/vrooli-prod"; else echo "secret/data/vrooli/$ENVIRONMENT"; fi)/config/shared-all
   $(if [[ "$ENVIRONMENT" == "prod" ]]; then echo "secret/data/vrooli-prod"; else echo "secret/data/vrooli/$ENVIRONMENT"; fi)/secrets/shared-server-jobs
   $(if [[ "$ENVIRONMENT" == "prod" ]]; then echo "secret/data/vrooli-prod"; else echo "secret/data/vrooli/$ENVIRONMENT"; fi)/secrets/postgres
   $(if [[ "$ENVIRONMENT" == "prod" ]]; then echo "secret/data/vrooli-prod"; else echo "secret/data/vrooli/$ENVIRONMENT"; fi)/secrets/redis
   $(if [[ "$ENVIRONMENT" == "prod" ]]; then echo "secret/data/vrooli-prod"; else echo "secret/data/vrooli/$ENVIRONMENT"; fi)/dockerhub/pull-credentials

2. UPDATE HELM VALUES in k8s/chart/values-${ENVIRONMENT}.yaml:
   vso:
     vaultAddr: "$VAULT_ADDR"
     k8sAuthMount: "$AUTH_MOUNT_PATH"
     k8sAuthRole: "$VSO_ROLE_NAME"

3. VERIFY VSO DEPLOYMENT:
   kubectl get pods -n vault-secrets-operator-system

4. DEPLOY APPLICATION:
   bash scripts/manage.sh deploy --source k8s --environment $ENVIRONMENT

5. VERIFY SECRET SYNC:
   kubectl get secrets -n $K8S_NAMESPACE | grep vrooli

For detailed instructions, see: docs/deployment/vault-production-setup.md

EOF
}

main() {
    parse_arguments "$@"
    
    log::header "Vrooli Production Vault Setup"
    log::info "Environment: $ENVIRONMENT"
    log::info "Vault Address: $VAULT_ADDR"
    log::info "Kubernetes Namespace: $K8S_NAMESPACE"
    log::info "Auth Mount Path: $AUTH_MOUNT_PATH"
    log::info "VSO Role Name: $VSO_ROLE_NAME"
    log::info "Dry Run: $DRY_RUN"
    
    if [[ "$DRY_RUN" == "false" ]]; then
        if ! flow::is_yes "$(flow::ask "Continue with Vault setup?")"; then
            log::info "Setup cancelled by user"
            exit 0
        fi
    fi
    
    validate_prerequisites
    create_vault_policies
    setup_kubernetes_auth
    create_vso_role
    verify_setup
    print_next_steps
    
    log::success "✅ Vault setup completed for environment: $ENVIRONMENT"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "This was a dry run. No changes were made to Vault."
        log::info "Run without --dry-run to execute the setup."
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi