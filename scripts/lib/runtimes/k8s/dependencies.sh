#!/usr/bin/env bash
################################################################################
# Generic Kubernetes Dependencies Validation
# 
# Framework for validating and checking Kubernetes dependencies including
# pods, services, configmaps, secrets, and external connectivity.
#
# Functions:
#   - k8s::deps::check_pod_running - Check if pods are running
#   - k8s::deps::check_service - Check if service exists and has endpoints
#   - k8s::deps::check_configmap - Check if configmap exists
#   - k8s::deps::check_secret - Check if secret exists
#   - k8s::deps::check_pvc - Check if PVC is bound
#   - k8s::deps::validate_all - Run all dependency checks from config
#
################################################################################

set -euo pipefail

# Get script directory
K8S_DEPS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${K8S_DEPS_DIR}/../../../utils/var.sh"
source "${var_LOG_FILE}"

################################################################################
# Check if pods matching selector are running
# Arguments:
#   $1 - Namespace
#   $2 - Label selector or pod name
#   $3 - Display name for logging
#   $4 - Minimum expected pods (default: 1)
# Returns:
#   0 if minimum pods are running, 1 otherwise
################################################################################
k8s::deps::check_pod_running() {
    local namespace="${1:?Namespace required}"
    local selector="${2:?Selector required}"
    local display_name="${3:-Pod}"
    local min_pods="${4:-1}"
    
    local running_pods
    
    # Check if selector is a pod name or label selector
    if [[ "$selector" == *"="* ]]; then
        # Label selector
        running_pods=$(kubectl get pods -n "$namespace" \
            -l "$selector" \
            --field-selector=status.phase=Running \
            --no-headers 2>/dev/null | wc -l)
    else
        # Pod name
        if kubectl get pod -n "$namespace" "$selector" \
            --field-selector=status.phase=Running >/dev/null 2>&1; then
            running_pods=1
        else
            running_pods=0
        fi
    fi
    
    if [[ "$running_pods" -ge "$min_pods" ]]; then
        log::success "✓ $display_name: $running_pods pod(s) running"
        return 0
    else
        log::error "✗ $display_name: only $running_pods pod(s) running (expected >= $min_pods)"
        return 1
    fi
}

################################################################################
# Check if service exists and has endpoints
# Arguments:
#   $1 - Namespace
#   $2 - Service name
#   $3 - Check endpoints (true/false, default: true)
# Returns:
#   0 if service exists (and has endpoints if checked), 1 otherwise
################################################################################
k8s::deps::check_service() {
    local namespace="${1:?Namespace required}"
    local service="${2:?Service name required}"
    local check_endpoints="${3:-true}"
    
    # Check service exists
    if ! kubectl get service -n "$namespace" "$service" >/dev/null 2>&1; then
        log::error "✗ Service not found: $namespace/$service"
        return 1
    fi
    
    # Check endpoints if requested
    if [[ "$check_endpoints" == "true" ]]; then
        local endpoints
        endpoints=$(kubectl get endpoints -n "$namespace" "$service" \
            -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null | wc -w)
        
        if [[ "$endpoints" -gt 0 ]]; then
            log::success "✓ Service $service has $endpoints endpoint(s)"
            return 0
        else
            log::error "✗ Service $service has no endpoints"
            return 1
        fi
    else
        log::success "✓ Service exists: $namespace/$service"
        return 0
    fi
}

################################################################################
# Check if ConfigMap exists
# Arguments:
#   $1 - Namespace
#   $2 - ConfigMap name
#   $3 - Required keys (comma-separated, optional)
# Returns:
#   0 if configmap exists (with required keys if specified), 1 otherwise
################################################################################
k8s::deps::check_configmap() {
    local namespace="${1:?Namespace required}"
    local configmap="${2:?ConfigMap name required}"
    local required_keys="${3:-}"
    
    # Check configmap exists
    if ! kubectl get configmap -n "$namespace" "$configmap" >/dev/null 2>&1; then
        log::error "✗ ConfigMap not found: $namespace/$configmap"
        return 1
    fi
    
    # Check required keys if specified
    if [[ -n "$required_keys" ]]; then
        IFS=',' read -ra keys <<< "$required_keys"
        local missing_keys=0
        
        for key in "${keys[@]}"; do
            if ! kubectl get configmap -n "$namespace" "$configmap" \
                -o jsonpath="{.data.$key}" 2>/dev/null | grep -q .; then
                log::error "  ✗ Missing key in ConfigMap: $key"
                ((missing_keys++))
            fi
        done
        
        if [[ "$missing_keys" -eq 0 ]]; then
            log::success "✓ ConfigMap $configmap has all required keys"
            return 0
        else
            return 1
        fi
    else
        log::success "✓ ConfigMap exists: $namespace/$configmap"
        return 0
    fi
}

################################################################################
# Check if Secret exists
# Arguments:
#   $1 - Namespace
#   $2 - Secret name
#   $3 - Required keys (comma-separated, optional)
# Returns:
#   0 if secret exists (with required keys if specified), 1 otherwise
################################################################################
k8s::deps::check_secret() {
    local namespace="${1:?Namespace required}"
    local secret="${2:?Secret name required}"
    local required_keys="${3:-}"
    
    # Check secret exists
    if ! kubectl get secret -n "$namespace" "$secret" >/dev/null 2>&1; then
        log::error "✗ Secret not found: $namespace/$secret"
        return 1
    fi
    
    # Check required keys if specified
    if [[ -n "$required_keys" ]]; then
        IFS=',' read -ra keys <<< "$required_keys"
        local missing_keys=0
        
        for key in "${keys[@]}"; do
            if ! kubectl get secret -n "$namespace" "$secret" \
                -o jsonpath="{.data.$key}" 2>/dev/null | grep -q .; then
                log::error "  ✗ Missing key in Secret: $key"
                ((missing_keys++))
            fi
        done
        
        if [[ "$missing_keys" -eq 0 ]]; then
            log::success "✓ Secret $secret has all required keys"
            return 0
        else
            return 1
        fi
    else
        log::success "✓ Secret exists: $namespace/$secret"
        return 0
    fi
}

################################################################################
# Check if PersistentVolumeClaim is bound
# Arguments:
#   $1 - Namespace
#   $2 - PVC name
# Returns:
#   0 if PVC is bound, 1 otherwise
################################################################################
k8s::deps::check_pvc() {
    local namespace="${1:?Namespace required}"
    local pvc="${2:?PVC name required}"
    
    local status
    status=$(kubectl get pvc -n "$namespace" "$pvc" \
        -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
    
    if [[ "$status" == "Bound" ]]; then
        log::success "✓ PVC $pvc is bound"
        return 0
    elif [[ "$status" == "NotFound" ]]; then
        log::error "✗ PVC not found: $namespace/$pvc"
        return 1
    else
        log::error "✗ PVC $pvc status: $status (expected: Bound)"
        return 1
    fi
}

################################################################################
# Check deployment or statefulset readiness
# Arguments:
#   $1 - Resource type (deployment/statefulset)
#   $2 - Namespace
#   $3 - Resource name
# Returns:
#   0 if resource is ready, 1 otherwise
################################################################################
k8s::deps::check_workload_ready() {
    local resource_type="${1:?Resource type required}"
    local namespace="${2:?Namespace required}"
    local name="${3:?Resource name required}"
    
    if ! kubectl get "$resource_type" -n "$namespace" "$name" >/dev/null 2>&1; then
        log::error "✗ $resource_type not found: $namespace/$name"
        return 1
    fi
    
    local ready_replicas
    ready_replicas=$(kubectl get "$resource_type" -n "$namespace" "$name" \
        -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    local desired_replicas
    if [[ "$resource_type" == "deployment" ]]; then
        desired_replicas=$(kubectl get "$resource_type" -n "$namespace" "$name" \
            -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
    else
        # StatefulSet
        desired_replicas=$(kubectl get "$resource_type" -n "$namespace" "$name" \
            -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
    fi
    
    if [[ "$ready_replicas" -eq "$desired_replicas" ]]; then
        log::success "✓ $resource_type $name is ready ($ready_replicas/$desired_replicas)"
        return 0
    else
        log::error "✗ $resource_type $name not ready ($ready_replicas/$desired_replicas)"
        return 1
    fi
}

################################################################################
# Validate all dependencies from a configuration
# Arguments:
#   $1 - JSON configuration file or inline JSON
# Returns:
#   Number of failed checks
#
# Configuration format:
# {
#   "pods": [
#     {"namespace": "default", "selector": "app=myapp", "name": "MyApp", "min": 2}
#   ],
#   "services": [
#     {"namespace": "default", "name": "myservice", "checkEndpoints": true}
#   ],
#   "configmaps": [
#     {"namespace": "default", "name": "myconfig", "keys": "key1,key2"}
#   ],
#   "secrets": [
#     {"namespace": "default", "name": "mysecret", "keys": "username,password"}
#   ],
#   "pvcs": [
#     {"namespace": "default", "name": "mypvc"}
#   ],
#   "deployments": [
#     {"namespace": "default", "name": "myapp"}
#   ],
#   "statefulsets": [
#     {"namespace": "default", "name": "mystatefulset"}
#   ]
# }
################################################################################
k8s::deps::validate_all() {
    local config="${1:?Configuration required}"
    local failed_checks=0
    
    log::header "Validating Kubernetes Dependencies"
    
    # Determine if config is a file or inline JSON
    local json_data
    if [[ -f "$config" ]]; then
        json_data=$(cat "$config")
    else
        json_data="$config"
    fi
    
    # Check pods
    if [[ $(echo "$json_data" | jq -r '.pods | length' 2>/dev/null) -gt 0 ]]; then
        log::info "Checking pods..."
        while IFS= read -r pod_config; do
            local namespace=$(echo "$pod_config" | jq -r '.namespace')
            local selector=$(echo "$pod_config" | jq -r '.selector')
            local name=$(echo "$pod_config" | jq -r '.name // .selector')
            local min=$(echo "$pod_config" | jq -r '.min // 1')
            
            if ! k8s::deps::check_pod_running "$namespace" "$selector" "$name" "$min"; then
                ((failed_checks++))
            fi
        done < <(echo "$json_data" | jq -c '.pods[]' 2>/dev/null)
    fi
    
    # Check services
    if [[ $(echo "$json_data" | jq -r '.services | length' 2>/dev/null) -gt 0 ]]; then
        log::info "Checking services..."
        while IFS= read -r service_config; do
            local namespace=$(echo "$service_config" | jq -r '.namespace')
            local name=$(echo "$service_config" | jq -r '.name')
            local check_endpoints=$(echo "$service_config" | jq -r '.checkEndpoints // true')
            
            if ! k8s::deps::check_service "$namespace" "$name" "$check_endpoints"; then
                ((failed_checks++))
            fi
        done < <(echo "$json_data" | jq -c '.services[]' 2>/dev/null)
    fi
    
    # Check configmaps
    if [[ $(echo "$json_data" | jq -r '.configmaps | length' 2>/dev/null) -gt 0 ]]; then
        log::info "Checking configmaps..."
        while IFS= read -r cm_config; do
            local namespace=$(echo "$cm_config" | jq -r '.namespace')
            local name=$(echo "$cm_config" | jq -r '.name')
            local keys=$(echo "$cm_config" | jq -r '.keys // ""')
            
            if ! k8s::deps::check_configmap "$namespace" "$name" "$keys"; then
                ((failed_checks++))
            fi
        done < <(echo "$json_data" | jq -c '.configmaps[]' 2>/dev/null)
    fi
    
    # Check secrets
    if [[ $(echo "$json_data" | jq -r '.secrets | length' 2>/dev/null) -gt 0 ]]; then
        log::info "Checking secrets..."
        while IFS= read -r secret_config; do
            local namespace=$(echo "$secret_config" | jq -r '.namespace')
            local name=$(echo "$secret_config" | jq -r '.name')
            local keys=$(echo "$secret_config" | jq -r '.keys // ""')
            
            if ! k8s::deps::check_secret "$namespace" "$name" "$keys"; then
                ((failed_checks++))
            fi
        done < <(echo "$json_data" | jq -c '.secrets[]' 2>/dev/null)
    fi
    
    # Summary
    echo ""
    if [[ "$failed_checks" -eq 0 ]]; then
        log::success "All dependencies validated successfully"
    else
        log::error "$failed_checks dependency check(s) failed"
    fi
    
    return "$failed_checks"
}