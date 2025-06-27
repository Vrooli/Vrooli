#!/usr/bin/env bash
# Script to verify and install Kubernetes prerequisites before deployment
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required utilities
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../utils/system.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../utils/args.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../setup/target/k8s_cluster.sh"

# Parse command line arguments
k8s_prerequisites::parse_arguments() {
    args::reset
    
    args::register_help
    
    args::register \
        --name "check-only" \
        --flag "c" \
        --desc "Only check for operators, don't install" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "yes" \
        --flag "y" \
        --desc "Automatically install missing operators without prompting" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    if args::is_asking_for_help "$@"; then
        args::usage
        exit 0
    fi
    
    args::parse "$@"
}

# Check if all required operators are installed
k8s_prerequisites::check_operators() {
    log::header "Checking Kubernetes operator prerequisites..."
    
    local missing_operators=0
    local operator_status=""
    
    # Check PGO
    if kubectl get crd postgresclusters.postgres-operator.crunchydata.com > /dev/null 2>&1; then
        log::success "✓ CrunchyData PostgreSQL Operator"
        operator_status+="PGO: Installed\n"
    else
        log::error "✗ CrunchyData PostgreSQL Operator NOT installed"
        operator_status+="PGO: Missing\n"
        ((missing_operators++))
    fi
    
    # Check Redis Operator
    if kubectl get crd redisfailovers.databases.spotahome.com > /dev/null 2>&1; then
        log::success "✓ Spotahome Redis Operator"
        operator_status+="Redis: Installed\n"
    else
        log::error "✗ Spotahome Redis Operator NOT installed"
        operator_status+="Redis: Missing\n"
        ((missing_operators++))
    fi
    
    # Check VSO
    if kubectl get crd vaultsecrets.secrets.hashicorp.com > /dev/null 2>&1; then
        log::success "✓ Vault Secrets Operator"
        operator_status+="VSO: Installed\n"
    else
        log::error "✗ Vault Secrets Operator NOT installed"
        operator_status+="VSO: Missing\n"
        ((missing_operators++))
    fi
    
    # Check for Vault instance
    log::info "Checking Vault connectivity..."
    local vault_address="${VAULT_ADDR:-http://vault.vault.svc.cluster.local:8200}"
    if kubectl exec -n default deployment/vrooli-server -- wget -qO- "${vault_address}/v1/sys/health" > /dev/null 2>&1; then
        log::success "✓ Vault instance reachable at ${vault_address}"
        operator_status+="Vault: Reachable\n"
    else
        log::warning "⚠ Cannot reach Vault at ${vault_address}. Make sure Vault is properly configured."
        operator_status+="Vault: Not reachable\n"
    fi
    
    echo ""
    log::info "Operator Status Summary:"
    echo -e "$operator_status"
    
    return "$missing_operators"
}

# Install missing operators
k8s_prerequisites::install_operators() {
    log::header "Installing missing operators..."
    
    # Check which operators need installation
    local install_pgo=false
    local install_redis=false
    local install_vso=false
    
    if ! kubectl get crd postgresclusters.postgres-operator.crunchydata.com > /dev/null 2>&1; then
        install_pgo=true
    fi
    
    if ! kubectl get crd redisfailovers.databases.spotahome.com > /dev/null 2>&1; then
        install_redis=true
    fi
    
    if ! kubectl get crd vaultsecrets.secrets.hashicorp.com > /dev/null 2>&1; then
        install_vso=true
    fi
    
    # Install operators using existing functions
    if [[ "$install_pgo" == "true" ]]; then
        log::info "Installing CrunchyData PostgreSQL Operator..."
        if k8s_cluster::install_pgo_operator; then
            log::success "PGO installed successfully"
        else
            log::error "Failed to install PGO"
            return 1
        fi
    fi
    
    if [[ "$install_redis" == "true" ]]; then
        log::info "Installing Spotahome Redis Operator..."
        if k8s_cluster::install_spotahome_redis_operator; then
            log::success "Redis Operator installed successfully"
        else
            log::error "Failed to install Redis Operator"
            return 1
        fi
    fi
    
    if [[ "$install_vso" == "true" ]]; then
        log::info "Installing Vault Secrets Operator..."
        if k8s_cluster::install_vso_helm_chart; then
            log::success "VSO installed successfully"
        else
            log::error "Failed to install VSO"
            return 1
        fi
    fi
    
    # Wait for operators to be ready
    log::info "Waiting for operators to become ready..."
    sleep 10
    
    # Verify installation
    k8s_prerequisites::check_operators
}

# Main function
k8s_prerequisites::main() {
    k8s_prerequisites::parse_arguments "$@"
    
    local check_only
    check_only=$(args::get "check-only")
    local auto_yes
    auto_yes=$(args::get "yes")
    
    # Check current cluster context
    local current_context
    current_context=$(kubectl config current-context 2>/dev/null || echo "none")
    log::info "Current Kubernetes context: ${current_context}"
    
    # Perform operator check
    local missing_count
    k8s_prerequisites::check_operators
    missing_count=$?
    
    if [[ $missing_count -gt 0 ]]; then
        if [[ "$check_only" == "yes" ]]; then
            log::error "Found $missing_count missing operators. Use without --check-only to install."
            exit 1
        else
            if [[ "$auto_yes" == "yes" ]]; then
                log::info "Auto-installing $missing_count missing operators..."
                k8s_prerequisites::install_operators
            else
                log::warning "Found $missing_count missing operators. Install them? (y/n)"
                read -r response
                if [[ "$response" == "y" || "$response" == "Y" ]]; then
                    k8s_prerequisites::install_operators
                else
                    log::error "Cannot proceed without required operators"
                    exit 1
                fi
            fi
        fi
    else
        log::success "All required operators are installed and ready!"
        
        # Additional checks for Vault configuration
        log::info "Additional recommendations:"
        log::info "1. Ensure Vault is configured with the required policies"
        log::info "2. Populate Vault secrets at the correct paths"
        log::info "3. Configure Vault Kubernetes authentication"
        log::info "4. Build and push Docker images to registry"
        
        return 0
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    k8s_prerequisites::main "$@"
fi