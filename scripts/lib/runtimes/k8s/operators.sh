#!/usr/bin/env bash
################################################################################
# Generic Kubernetes Operator Management
# 
# Provides reusable functions for checking, validating, and managing
# Kubernetes operators in any cluster. NOT specific to any application.
#
# Functions:
#   - k8s::operators::check_crd - Check if a CRD exists
#   - k8s::operators::check_pods - Check if operator pods are running
#   - k8s::operators::wait_ready - Wait for operator to be ready
#   - k8s::operators::check_list - Check multiple operators from a list
#   - k8s::operators::validate_health - Validate operator health
#
# Usage:
#   source scripts/lib/runtimes/k8s/operators.sh
#   
#   # Define your operators
#   declare -A MY_OPERATORS=(
#       ["crd_name"]="Display Name|namespace|label_selector"
#   )
#   k8s::operators::check_list MY_OPERATORS
#
################################################################################

set -euo pipefail

# Get script directory
K8S_OPERATORS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${K8S_OPERATORS_DIR}/../../../utils/var.sh"
source "${var_LOG_FILE}"

################################################################################
# Check if a Custom Resource Definition exists
# Arguments:
#   $1 - CRD name (e.g., "postgresclusters.postgres-operator.crunchydata.com")
#   $2 - Display name for logging (e.g., "PostgreSQL Operator")
# Returns:
#   0 if CRD exists, 1 if not
################################################################################
k8s::operators::check_crd() {
    local crd_name="${1:?CRD name required}"
    local display_name="${2:-$crd_name}"
    
    if kubectl get crd "$crd_name" >/dev/null 2>&1; then
        log::success "✓ $display_name CRD exists"
        return 0
    else
        log::error "✗ $display_name CRD not found"
        return 1
    fi
}

################################################################################
# Check if operator pods are running
# Arguments:
#   $1 - Namespace
#   $2 - Label selector (e.g., "app=postgres-operator")
#   $3 - Display name for logging
#   $4 - Minimum pod count (default: 1)
# Returns:
#   0 if pods are running, 1 if not
################################################################################
k8s::operators::check_pods() {
    local namespace="${1:?Namespace required}"
    local label_selector="${2:?Label selector required}"
    local display_name="${3:-Operator}"
    local min_pods="${4:-1}"
    
    local running_pods
    running_pods=$(kubectl get pods -n "$namespace" \
        -l "$label_selector" \
        --field-selector=status.phase=Running \
        --no-headers 2>/dev/null | wc -l)
    
    if [[ "$running_pods" -ge "$min_pods" ]]; then
        log::success "✓ $display_name has $running_pods pod(s) running"
        return 0
    else
        log::error "✗ $display_name has only $running_pods pod(s) running (expected >= $min_pods)"
        return 1
    fi
}

################################################################################
# Wait for operator to be ready
# Arguments:
#   $1 - Namespace
#   $2 - Label selector
#   $3 - Display name
#   $4 - Timeout in seconds (default: 120)
# Returns:
#   0 if ready within timeout, 1 if timeout exceeded
################################################################################
k8s::operators::wait_ready() {
    local namespace="${1:?Namespace required}"
    local label_selector="${2:?Label selector required}"
    local display_name="${3:-Operator}"
    local timeout="${4:-120}"
    
    log::info "Waiting for $display_name to be ready (timeout: ${timeout}s)..."
    
    local elapsed=0
    local check_interval=5
    
    while [[ "$elapsed" -lt "$timeout" ]]; do
        if k8s::operators::check_pods "$namespace" "$label_selector" "$display_name" 1 >/dev/null 2>&1; then
            log::success "✓ $display_name is ready"
            return 0
        fi
        
        sleep "$check_interval"
        elapsed=$((elapsed + check_interval))
        echo -n "."
    done
    
    echo ""
    log::error "✗ Timeout waiting for $display_name to be ready"
    return 1
}

################################################################################
# Check multiple operators from a configuration array
# Arguments:
#   $1 - Name of associative array with operator definitions
#        Format: ["crd_name"]="Display Name|namespace|label_selector"
# Returns:
#   Number of missing/unhealthy operators
################################################################################
k8s::operators::check_list() {
    local -n operators=$1
    local missing_count=0
    local failed_count=0
    
    log::header "Checking Kubernetes Operators"
    
    for crd in "${!operators[@]}"; do
        IFS='|' read -r display_name namespace label_selector <<< "${operators[$crd]}"
        
        # Check CRD exists
        if ! k8s::operators::check_crd "$crd" "$display_name"; then
            ((missing_count++))
            continue
        fi
        
        # Check pods if namespace and selector provided
        if [[ -n "$namespace" ]] && [[ -n "$label_selector" ]]; then
            if ! k8s::operators::check_pods "$namespace" "$label_selector" "$display_name"; then
                ((failed_count++))
            fi
        fi
    done
    
    # Summary
    echo ""
    if [[ "$missing_count" -eq 0 ]] && [[ "$failed_count" -eq 0 ]]; then
        log::success "All operators are installed and healthy"
        return 0
    else
        [[ "$missing_count" -gt 0 ]] && log::error "$missing_count operator(s) not installed"
        [[ "$failed_count" -gt 0 ]] && log::warning "$failed_count operator(s) not healthy"
        return $((missing_count + failed_count))
    fi
}

################################################################################
# Validate operator health with detailed checks
# Arguments:
#   $1 - Namespace
#   $2 - Operator name
#   $3 - Expected CRDs (comma-separated list)
#   $4 - Expected deployments (comma-separated list)
# Returns:
#   0 if healthy, 1 if unhealthy
################################################################################
k8s::operators::validate_health() {
    local namespace="${1:?Namespace required}"
    local operator_name="${2:?Operator name required}"
    local expected_crds="${3:-}"
    local expected_deployments="${4:-}"
    
    local health_issues=0
    
    log::info "Validating $operator_name health in namespace $namespace..."
    
    # Check CRDs if provided
    if [[ -n "$expected_crds" ]]; then
        IFS=',' read -ra crd_array <<< "$expected_crds"
        for crd in "${crd_array[@]}"; do
            if ! kubectl get crd "$crd" >/dev/null 2>&1; then
                log::error "  ✗ Missing CRD: $crd"
                ((health_issues++))
            else
                log::success "  ✓ CRD exists: $crd"
            fi
        done
    fi
    
    # Check deployments if provided
    if [[ -n "$expected_deployments" ]]; then
        IFS=',' read -ra deployment_array <<< "$expected_deployments"
        for deployment in "${deployment_array[@]}"; do
            if ! kubectl get deployment -n "$namespace" "$deployment" >/dev/null 2>&1; then
                log::error "  ✗ Missing deployment: $deployment"
                ((health_issues++))
            else
                # Check if deployment is ready
                local ready_replicas
                ready_replicas=$(kubectl get deployment -n "$namespace" "$deployment" \
                    -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
                local desired_replicas
                desired_replicas=$(kubectl get deployment -n "$namespace" "$deployment" \
                    -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
                
                if [[ "$ready_replicas" -ge "$desired_replicas" ]]; then
                    log::success "  ✓ Deployment ready: $deployment ($ready_replicas/$desired_replicas)"
                else
                    log::warning "  ⚠ Deployment not fully ready: $deployment ($ready_replicas/$desired_replicas)"
                    ((health_issues++))
                fi
            fi
        done
    fi
    
    return "$health_issues"
}

################################################################################
# Check if kubectl is available and cluster is accessible
# Returns:
#   0 if cluster is accessible, 1 if not
################################################################################
k8s::operators::check_cluster_access() {
    if ! command -v kubectl >/dev/null 2>&1; then
        log::error "kubectl is not installed"
        return 1
    fi
    
    if ! kubectl cluster-info >/dev/null 2>&1; then
        log::error "Cannot access Kubernetes cluster"
        log::info "Current context: $(kubectl config current-context 2>/dev/null || echo 'none')"
        return 1
    fi
    
    log::success "✓ Kubernetes cluster is accessible"
    return 0
}