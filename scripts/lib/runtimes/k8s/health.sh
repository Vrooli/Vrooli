#!/usr/bin/env bash
################################################################################
# Generic Kubernetes Health Checking Utilities
# 
# Provides health checking and monitoring functions for Kubernetes resources.
#
# Functions:
#   - k8s::health::check_deployment - Check deployment health
#   - k8s::health::check_statefulset - Check statefulset health
#   - k8s::health::check_daemonset - Check daemonset health
#   - k8s::health::check_pod_logs - Check for errors in pod logs
#   - k8s::health::wait_for_rollout - Wait for rollout to complete
#   - k8s::health::check_resource_usage - Check resource usage
#
################################################################################

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
K8S_HEALTH_DIR="${APP_ROOT}/scripts/lib/runtimes/k8s"

# Source dependencies
source "${K8S_HEALTH_DIR}/../../../utils/var.sh"
source "${var_LOG_FILE}"

################################################################################
# Check deployment health and readiness
# Arguments:
#   $1 - Namespace
#   $2 - Deployment name
#   $3 - Check pod health (true/false, default: true)
# Returns:
#   0 if healthy, 1 if unhealthy
################################################################################
k8s::health::check_deployment() {
    local namespace="${1:?Namespace required}"
    local deployment="${2:?Deployment name required}"
    local check_pods="${3:-true}"
    
    log::info "Checking deployment health: $namespace/$deployment"
    
    # Check deployment exists
    if ! kubectl get deployment -n "$namespace" "$deployment" >/dev/null 2>&1; then
        log::error "✗ Deployment not found: $namespace/$deployment"
        return 1
    fi
    
    # Get deployment status
    local available_replicas
    available_replicas=$(kubectl get deployment -n "$namespace" "$deployment" \
        -o jsonpath='{.status.availableReplicas}' 2>/dev/null || echo "0")
    
    local desired_replicas
    desired_replicas=$(kubectl get deployment -n "$namespace" "$deployment" \
        -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
    
    local updated_replicas
    updated_replicas=$(kubectl get deployment -n "$namespace" "$deployment" \
        -o jsonpath='{.status.updatedReplicas}' 2>/dev/null || echo "0")
    
    # Check if deployment is healthy
    if [[ "$available_replicas" -eq "$desired_replicas" ]] && \
       [[ "$updated_replicas" -eq "$desired_replicas" ]]; then
        log::success "✓ Deployment healthy: $available_replicas/$desired_replicas replicas available"
    else
        log::error "✗ Deployment unhealthy: $available_replicas/$desired_replicas available, $updated_replicas updated"
        return 1
    fi
    
    # Check pod health if requested
    if [[ "$check_pods" == "true" ]]; then
        local unhealthy_pods=0
        while IFS= read -r pod; do
            local pod_status
            pod_status=$(kubectl get pod -n "$namespace" "$pod" \
                -o jsonpath='{.status.phase}' 2>/dev/null)
            
            if [[ "$pod_status" != "Running" ]]; then
                log::warning "  ⚠ Pod $pod status: $pod_status"
                ((unhealthy_pods++))
            fi
            
            # Check container statuses
            local container_count
            container_count=$(kubectl get pod -n "$namespace" "$pod" \
                -o jsonpath='{.status.containerStatuses[*].ready}' 2>/dev/null | wc -w)
            
            local ready_containers
            ready_containers=$(kubectl get pod -n "$namespace" "$pod" \
                -o jsonpath='{.status.containerStatuses[?(@.ready==true)].name}' 2>/dev/null | wc -w)
            
            if [[ "$ready_containers" -lt "$container_count" ]]; then
                log::warning "  ⚠ Pod $pod: only $ready_containers/$container_count containers ready"
                ((unhealthy_pods++))
            fi
        done < <(kubectl get pods -n "$namespace" \
            -l "$(kubectl get deployment -n "$namespace" "$deployment" \
                -o jsonpath='{.spec.selector.matchLabels}' | jq -r 'to_entries | map("\(.key)=\(.value)") | join(",")' 2>/dev/null)" \
            --no-headers -o custom-columns=":metadata.name" 2>/dev/null)
        
        if [[ "$unhealthy_pods" -gt 0 ]]; then
            log::warning "Found $unhealthy_pods unhealthy pod(s)"
            return 1
        fi
    fi
    
    return 0
}

################################################################################
# Check StatefulSet health
# Arguments:
#   $1 - Namespace
#   $2 - StatefulSet name
# Returns:
#   0 if healthy, 1 if unhealthy
################################################################################
k8s::health::check_statefulset() {
    local namespace="${1:?Namespace required}"
    local statefulset="${2:?StatefulSet name required}"
    
    log::info "Checking statefulset health: $namespace/$statefulset"
    
    # Check statefulset exists
    if ! kubectl get statefulset -n "$namespace" "$statefulset" >/dev/null 2>&1; then
        log::error "✗ StatefulSet not found: $namespace/$statefulset"
        return 1
    fi
    
    # Get statefulset status
    local ready_replicas
    ready_replicas=$(kubectl get statefulset -n "$namespace" "$statefulset" \
        -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    local desired_replicas
    desired_replicas=$(kubectl get statefulset -n "$namespace" "$statefulset" \
        -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
    
    local current_replicas
    current_replicas=$(kubectl get statefulset -n "$namespace" "$statefulset" \
        -o jsonpath='{.status.currentReplicas}' 2>/dev/null || echo "0")
    
    # Check if statefulset is healthy
    if [[ "$ready_replicas" -eq "$desired_replicas" ]] && \
       [[ "$current_replicas" -eq "$desired_replicas" ]]; then
        log::success "✓ StatefulSet healthy: $ready_replicas/$desired_replicas replicas ready"
        return 0
    else
        log::error "✗ StatefulSet unhealthy: $ready_replicas/$desired_replicas ready, $current_replicas current"
        return 1
    fi
}

################################################################################
# Check DaemonSet health
# Arguments:
#   $1 - Namespace
#   $2 - DaemonSet name
# Returns:
#   0 if healthy, 1 if unhealthy
################################################################################
k8s::health::check_daemonset() {
    local namespace="${1:?Namespace required}"
    local daemonset="${2:?DaemonSet name required}"
    
    log::info "Checking daemonset health: $namespace/$daemonset"
    
    # Check daemonset exists
    if ! kubectl get daemonset -n "$namespace" "$daemonset" >/dev/null 2>&1; then
        log::error "✗ DaemonSet not found: $namespace/$daemonset"
        return 1
    fi
    
    # Get daemonset status
    local number_ready
    number_ready=$(kubectl get daemonset -n "$namespace" "$daemonset" \
        -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")
    
    local desired_number
    desired_number=$(kubectl get daemonset -n "$namespace" "$daemonset" \
        -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "1")
    
    # Check if daemonset is healthy
    if [[ "$number_ready" -eq "$desired_number" ]]; then
        log::success "✓ DaemonSet healthy: $number_ready/$desired_number pods ready"
        return 0
    else
        log::error "✗ DaemonSet unhealthy: $number_ready/$desired_number pods ready"
        return 1
    fi
}

################################################################################
# Check pod logs for errors
# Arguments:
#   $1 - Namespace
#   $2 - Pod name or label selector
#   $3 - Number of lines to check (default: 100)
#   $4 - Error patterns to search (default: "error|fail|exception|fatal")
# Returns:
#   0 if no errors found, 1 if errors found
################################################################################
k8s::health::check_pod_logs() {
    local namespace="${1:?Namespace required}"
    local pod_selector="${2:?Pod selector required}"
    local lines="${3:-100}"
    local error_patterns="${4:-error|fail|exception|fatal}"
    
    local pods=()
    
    # Determine if selector is a pod name or label selector
    if [[ "$pod_selector" == *"="* ]]; then
        # Label selector
        mapfile -t pods < <(kubectl get pods -n "$namespace" -l "$pod_selector" \
            --no-headers -o custom-columns=":metadata.name" 2>/dev/null)
    else
        # Pod name
        pods=("$pod_selector")
    fi
    
    local errors_found=0
    
    for pod in "${pods[@]}"; do
        log::info "Checking logs for pod: $pod"
        
        # Get container names
        local containers
        mapfile -t containers < <(kubectl get pod -n "$namespace" "$pod" \
            -o jsonpath='{.spec.containers[*].name}' 2>/dev/null | tr ' ' '\n')
        
        for container in "${containers[@]}"; do
            local error_count
            error_count=$(kubectl logs -n "$namespace" "$pod" -c "$container" \
                --tail="$lines" 2>/dev/null | grep -iE "$error_patterns" | wc -l)
            
            if [[ "$error_count" -gt 0 ]]; then
                log::warning "  ⚠ Found $error_count error(s) in container $container"
                ((errors_found++))
            else
                log::success "  ✓ No errors in container $container (last $lines lines)"
            fi
        done
    done
    
    if [[ "$errors_found" -gt 0 ]]; then
        return 1
    fi
    
    return 0
}

################################################################################
# Wait for rollout to complete
# Arguments:
#   $1 - Resource type (deployment/statefulset/daemonset)
#   $2 - Namespace
#   $3 - Resource name
#   $4 - Timeout in seconds (default: 300)
# Returns:
#   0 if rollout completed, 1 if timeout
################################################################################
k8s::health::wait_for_rollout() {
    local resource_type="${1:?Resource type required}"
    local namespace="${2:?Namespace required}"
    local name="${3:?Resource name required}"
    local timeout="${4:-300}"
    
    log::info "Waiting for $resource_type/$name rollout to complete (timeout: ${timeout}s)..."
    
    if kubectl rollout status "$resource_type" -n "$namespace" "$name" \
        --timeout="${timeout}s" 2>/dev/null; then
        log::success "✓ Rollout completed successfully"
        return 0
    else
        log::error "✗ Rollout failed or timed out"
        
        # Show current status
        kubectl rollout status "$resource_type" -n "$namespace" "$name" 2>&1 || true
        
        return 1
    fi
}

################################################################################
# Check resource usage against limits
# Arguments:
#   $1 - Namespace
#   $2 - Pod name or label selector
#   $3 - CPU threshold percentage (default: 80)
#   $4 - Memory threshold percentage (default: 80)
# Returns:
#   0 if within limits, 1 if exceeding thresholds
################################################################################
k8s::health::check_resource_usage() {
    local namespace="${1:?Namespace required}"
    local pod_selector="${2:?Pod selector required}"
    local cpu_threshold="${3:-80}"
    local memory_threshold="${4:-80}"
    
    # Check if metrics-server is available
    if ! kubectl top nodes >/dev/null 2>&1; then
        log::warning "Metrics server not available, skipping resource usage check"
        return 0
    fi
    
    local pods=()
    
    # Determine if selector is a pod name or label selector
    if [[ "$pod_selector" == *"="* ]]; then
        # Label selector
        mapfile -t pods < <(kubectl get pods -n "$namespace" -l "$pod_selector" \
            --no-headers -o custom-columns=":metadata.name" 2>/dev/null)
    else
        # Pod name
        pods=("$pod_selector")
    fi
    
    local high_usage=0
    
    for pod in "${pods[@]}"; do
        # Get pod metrics
        local metrics
        metrics=$(kubectl top pod -n "$namespace" "$pod" --no-headers 2>/dev/null || echo "")
        
        if [[ -z "$metrics" ]]; then
            log::warning "No metrics available for pod: $pod"
            continue
        fi
        
        # Parse CPU and memory usage
        local cpu_usage
        cpu_usage=$(echo "$metrics" | awk '{print $2}' | sed 's/m//')
        
        local memory_usage
        memory_usage=$(echo "$metrics" | awk '{print $3}' | sed 's/Mi//')
        
        # Get pod limits
        local cpu_limit
        cpu_limit=$(kubectl get pod -n "$namespace" "$pod" \
            -o jsonpath='{.spec.containers[0].resources.limits.cpu}' 2>/dev/null | sed 's/m//')
        
        local memory_limit
        memory_limit=$(kubectl get pod -n "$namespace" "$pod" \
            -o jsonpath='{.spec.containers[0].resources.limits.memory}' 2>/dev/null | sed 's/Mi//')
        
        # Calculate percentages if limits are set
        if [[ -n "$cpu_limit" ]] && [[ "$cpu_limit" -gt 0 ]]; then
            local cpu_percent=$((cpu_usage * 100 / cpu_limit))
            if [[ "$cpu_percent" -gt "$cpu_threshold" ]]; then
                log::warning "  ⚠ Pod $pod CPU usage: ${cpu_percent}% (threshold: ${cpu_threshold}%)"
                ((high_usage++))
            else
                log::success "  ✓ Pod $pod CPU usage: ${cpu_percent}%"
            fi
        fi
        
        if [[ -n "$memory_limit" ]] && [[ "$memory_limit" -gt 0 ]]; then
            local mem_percent=$((memory_usage * 100 / memory_limit))
            if [[ "$mem_percent" -gt "$memory_threshold" ]]; then
                log::warning "  ⚠ Pod $pod memory usage: ${mem_percent}% (threshold: ${memory_threshold}%)"
                ((high_usage++))
            else
                log::success "  ✓ Pod $pod memory usage: ${mem_percent}%"
            fi
        fi
    done
    
    if [[ "$high_usage" -gt 0 ]]; then
        return 1
    fi
    
    return 0
}