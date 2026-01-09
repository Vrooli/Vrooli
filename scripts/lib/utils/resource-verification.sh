#!/usr/bin/env bash
################################################################################
# Resource Verification with Retries
# 
# Provides robust resource startup verification to prevent scenario failures
# Uses vrooli resource status --json and jq for consistent checking
################################################################################

set -euo pipefail

# Source logging utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# **PRIORITY 2 FIX**: Enhanced timeout configuration for reliable startup
RESOURCE_VERIFY_MAX_RETRIES="${RESOURCE_VERIFY_MAX_RETRIES:-30}"  # Increased from 10 to 30
RESOURCE_VERIFY_RETRY_DELAY="${RESOURCE_VERIFY_RETRY_DELAY:-2}"  # Initial delay in seconds
RESOURCE_VERIFY_TIMEOUT="${RESOURCE_VERIFY_TIMEOUT:-120}"  # Increased to 120 seconds
RESOURCE_VERIFY_USE_BACKOFF="${RESOURCE_VERIFY_USE_BACKOFF:-true}"  # Use exponential backoff for retries

#######################################
# Verify a single resource is running with retries
# Arguments:
#   $1 - Resource name
#   $2 - Max retries (optional, defaults to runtime.json or global setting)
# Returns:
#   0 if resource is running, 1 if failed after retries
#######################################
resource_verify::check_single() {
    local resource_name="$1"
    local max_retries="$2"
    local retry_count=0
    
    # Load resource-specific settings from runtime.json
    local runtime_json="${APP_ROOT}/resources/${resource_name}/config/runtime.json"
    local recovery_attempts=$RESOURCE_VERIFY_MAX_RETRIES  # default to global
    
    if [[ -f "$runtime_json" ]]; then
        # Get recovery attempts from runtime.json
        local json_recovery=$(jq -r '.recovery_attempts // null' "$runtime_json" 2>/dev/null)
        if [[ "$json_recovery" != "null" ]]; then
            recovery_attempts=$((json_recovery * 10))  # Convert recovery attempts to retry count
        fi
    fi
    
    # Use provided max_retries or fall back to runtime.json or global
    max_retries="${max_retries:-$recovery_attempts}"
    
    log::info "Verifying resource: $resource_name (max retries: $max_retries)"
    
    # Calculate dynamic delay with exponential backoff
    local current_delay=$RESOURCE_VERIFY_RETRY_DELAY
    local max_delay=10  # Cap at 10 seconds per retry
    local total_elapsed=0
    
    while [[ $retry_count -lt $max_retries ]] && [[ $total_elapsed -lt $RESOURCE_VERIFY_TIMEOUT ]]; do
        # Use vrooli resource status with JSON output and jq to extract running status
        local status_json
        status_json=$(vrooli resource status "$resource_name" --json 2>/dev/null || echo '{}')
        local is_running=$(echo "$status_json" | jq -r '.running // false' 2>/dev/null || echo "false")
        local is_healthy=$(echo "$status_json" | jq -r '.healthy // false' 2>/dev/null || echo "false")
        
        # Check both running and healthy status
        if [[ "$is_running" == "true" ]]; then
            # If resource reports healthy status, trust it
            if [[ "$is_healthy" == "true" ]]; then
                log::success "✅ Resource verified healthy: $resource_name"
                return 0
            fi
            # If only running but not healthy yet, continue checking
            log::debug "Resource running but not yet healthy: $resource_name"
        fi
        
        if [[ $retry_count -eq 0 ]]; then
            log::info "Resource not ready, attempting to start: $resource_name"
            
            # Use the correct vrooli resource start command
            if vrooli resource start "$resource_name" >/dev/null 2>&1; then
                log::info "Start command completed for: $resource_name"
                # Use resource-specific startup timeout from runtime.json if available
                local startup_wait=$(jq -r '.startup_timeout // 10' "$runtime_json" 2>/dev/null || echo "10")
                log::info "Waiting ${startup_wait}s for initial startup of: $resource_name"
                sleep "$startup_wait"
                total_elapsed=$((total_elapsed + startup_wait))
            else
                log::warning "Failed to start resource: $resource_name"
            fi
        fi
        
        retry_count=$((retry_count + 1))
        if [[ $retry_count -lt $max_retries ]]; then
            log::debug "Waiting ${current_delay}s before retry $retry_count/$max_retries for: $resource_name"
            sleep "$current_delay"
            total_elapsed=$((total_elapsed + current_delay))
            
            # Exponential backoff if enabled
            if [[ "$RESOURCE_VERIFY_USE_BACKOFF" == "true" ]]; then
                current_delay=$((current_delay * 2))
                if [[ $current_delay -gt $max_delay ]]; then
                    current_delay=$max_delay
                fi
            fi
        fi
    done
    
    # **PRIORITY 2 FIX**: Add recovery logic before final failure
    log::warning "Resource verification failed, attempting recovery for: $resource_name"
    resource_verify::attempt_recovery "$resource_name"
    
    # Try one final verification after recovery
    local is_running
    is_running=$(vrooli resource status "$resource_name" --json 2>/dev/null | jq -r '.running // false' 2>/dev/null || echo "false")
    if [[ "$is_running" == "true" ]]; then
        log::success "✅ Resource recovered successfully: $resource_name"
        return 0
    fi
    
    log::error "❌ Resource failed to start after $max_retries attempts: $resource_name"
    return 1
}

#######################################
# Attempt recovery for failed resource verification
# Arguments:
#   $1 - Resource name
# Returns:
#   0 on successful recovery attempt, 1 on error
#######################################
resource_verify::attempt_recovery() {
    local resource_name="$1"
    
    log::info "Starting recovery process for: $resource_name"
    
    # Load resource-specific recovery settings from runtime.json
    local runtime_json="${APP_ROOT}/resources/${resource_name}/config/runtime.json"
    local recovery_strategy="restart"  # default strategy
    local recovery_wait=5  # default wait time after recovery
    local recovery_force=false  # whether to force operations
    
    if [[ -f "$runtime_json" ]]; then
        recovery_strategy=$(jq -r '.recovery_strategy // "restart"' "$runtime_json" 2>/dev/null || echo "restart")
        recovery_wait=$(jq -r '.recovery_wait // 5' "$runtime_json" 2>/dev/null || echo "5")
        recovery_force=$(jq -r '.recovery_force // false' "$runtime_json" 2>/dev/null || echo "false")
        log::debug "Using recovery strategy '$recovery_strategy' for $resource_name"
    fi
    
    # Apply recovery strategy based on runtime.json configuration
    case "$recovery_strategy" in
        "restart")
            # Stop and start the resource cleanly
            log::info "Applying restart recovery strategy for: $resource_name"
            
            # Stop the resource (ignore errors as it might already be stopped)
            vrooli resource stop "$resource_name" >/dev/null 2>&1 || true
            sleep 2
            
            # Start the resource using resource-specific CLI for proper cleanup
            if resource-"$resource_name" manage start >/dev/null 2>&1; then
                log::info "Resource restart completed for: $resource_name"
            else
                log::warning "Resource restart failed for: $resource_name"
            fi
            ;;
            
        "recreate")
            # Full reinstall of the resource (useful for corrupted state)
            log::info "Applying recreate recovery strategy for: $resource_name"
            
            # Stop and uninstall using resource-specific CLI
            resource-"$resource_name" manage stop >/dev/null 2>&1 || true
            sleep 2
            
            if [[ "$recovery_force" == "true" ]]; then
                # Force uninstall if configured
                resource-"$resource_name" manage uninstall --force >/dev/null 2>&1 || true
            else
                resource-"$resource_name" manage uninstall >/dev/null 2>&1 || true
            fi
            sleep 2
            
            # Reinstall and start
            if resource-"$resource_name" manage install >/dev/null 2>&1; then
                log::info "Resource recreated successfully: $resource_name"
                resource-"$resource_name" manage start >/dev/null 2>&1 || true
            else
                log::warning "Resource recreation failed for: $resource_name"
            fi
            ;;
            
        "start_only")
            # Just try to start the resource (for resources that self-heal)
            log::info "Applying start_only recovery strategy for: $resource_name"
            if resource-"$resource_name" manage start >/dev/null 2>&1; then
                log::info "Resource started successfully: $resource_name"
            else
                log::warning "Resource start failed for: $resource_name"
            fi
            ;;
            
        "none")
            # No recovery action (for resources that need manual intervention)
            log::info "No recovery action configured for: $resource_name"
            log::info "Manual intervention may be required"
            ;;
            
        *)
            # Unknown strategy, use default restart
            log::warning "Unknown recovery strategy '$recovery_strategy', using default restart"
            vrooli resource stop "$resource_name" >/dev/null 2>&1 || true
            sleep 2
            resource-"$resource_name" manage start >/dev/null 2>&1 || true
            ;;
    esac
    
    # Wait for resource to stabilize
    if [[ "$recovery_strategy" != "none" ]]; then
        log::info "Waiting ${recovery_wait}s for resource to stabilize..."
        sleep "$recovery_wait"
    fi
    
    return 0
}

#######################################
# Extract required resources from service.json
# Returns:
#   Space-separated list of required resource names
#######################################
resource_verify::get_required_resources() {
    local service_json="${SERVICE_JSON_PATH:-${PWD}/.vrooli/service.json}"
    
    if [[ ! -f "$service_json" ]]; then
        log::debug "No service.json found, no resources to verify"
        return 0
    fi
    
    # Extract resources marked as required=true
    local jq_resources="$var_JQ_RESOURCES_EXPR"
    [[ -z "$jq_resources" ]] && jq_resources='(.dependencies.resources // {})'
    local required_resources
    required_resources=$(jq -r "${jq_resources} | to_entries[] | select(.value.required == true) | .key" "$service_json" 2>/dev/null | tr '\n' ' ' || echo "")
    
    echo "$required_resources"
}

#######################################
# Verify all required resources for current scenario
# Returns:
#   0 if all required resources are running, 1 if any failed
#######################################
resource_verify::verify_required() {
    local required_resources
    required_resources=$(resource_verify::get_required_resources)
    
    if [[ -z "$required_resources" ]]; then
        log::info "No required resources to verify"
        return 0
    fi
    
    log::info "Verifying required resources: $required_resources"
    
    local failed_resources=()
    local total_time=0
    
    # Convert to array for processing
    local resource_array=($required_resources)
    
    for resource in "${resource_array[@]}"; do
        local start_time=$(date +%s)
        
        if ! resource_verify::check_single "$resource"; then
            failed_resources+=("$resource")
        fi
        
        local end_time=$(date +%s)
        total_time=$((total_time + end_time - start_time))
        
        # Abort early if we're taking too long
        if [[ $total_time -gt $RESOURCE_VERIFY_TIMEOUT ]]; then
            log::warning "Resource verification timeout reached (${RESOURCE_VERIFY_TIMEOUT}s)"
            break
        fi
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        log::error "Failed to verify resources: ${failed_resources[*]}"
        log::info "You can manually start failed resources with: vrooli resource start <name>"
        return 1
    fi
    
    log::success "✅ All required resources verified running"
    return 0
}

#######################################
# Verify resources with parallel startup for better performance
# Arguments:
#   $@ - List of resource names (optional, defaults to required resources)
# Returns:
#   0 if all resources are running, 1 if any failed
#######################################
resource_verify::verify_parallel() {
    local resources_to_check=("$@")
    
    # If no resources specified, get required ones
    if [[ ${#resources_to_check[@]} -eq 0 ]]; then
        local required_resources
        required_resources=$(resource_verify::get_required_resources)
        if [[ -z "$required_resources" ]]; then
            log::info "No required resources to verify"
            return 0
        fi
        resources_to_check=($required_resources)
    fi
    
    log::info "Starting parallel verification of ${#resources_to_check[@]} resources: ${resources_to_check[*]}"
    
    # Start all resources in parallel first
    local start_pids=()
    for resource in "${resources_to_check[@]}"; do
        {
            vrooli resource start "$resource" >/dev/null 2>&1 || true
        } &
        start_pids+=($!)
    done
    
    # Wait for all start commands to complete
    for pid in "${start_pids[@]}"; do
        wait "$pid" || true
    done
    
    log::info "Start commands completed, verifying resource status..."
    
    # Now verify each resource is actually running
    local failed_resources=()
    for resource in "${resources_to_check[@]}"; do
        local retry_count=0
        local verified=false
        
        while [[ $retry_count -lt $RESOURCE_VERIFY_MAX_RETRIES ]] && [[ "$verified" == "false" ]]; do
            local is_running
            is_running=$(vrooli resource status "$resource" --json 2>/dev/null | jq -r '.running // false' 2>/dev/null || echo "false")
            
            if [[ "$is_running" == "true" ]]; then
                log::success "✅ $resource"
                verified=true
                break
            fi
            
            retry_count=$((retry_count + 1))
            if [[ $retry_count -lt $RESOURCE_VERIFY_MAX_RETRIES ]]; then
                sleep "$RESOURCE_VERIFY_RETRY_DELAY"
            fi
        done
        
        if [[ "$verified" == "false" ]]; then
            failed_resources+=("$resource")
            log::error "❌ $resource (failed after $RESOURCE_VERIFY_MAX_RETRIES attempts)"
        fi
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        log::error "Failed to verify resources: ${failed_resources[*]}"
        return 1
    fi
    
    log::success "🚀 All resources verified and running"
    return 0
}

#######################################
# Verify resources with priority-based startup for better reliability
# Arguments:
#   $@ - List of resource names (optional, defaults to required resources)
# Returns:
#   0 if all resources are running, 1 if any failed
#######################################
resource_verify::verify_with_priority() {
    local resources_to_check=("$@")
    
    # If no resources specified, get required ones
    if [[ ${#resources_to_check[@]} -eq 0 ]]; then
        local required_resources
        required_resources=$(resource_verify::get_required_resources)
        if [[ -z "$required_resources" ]]; then
            log::info "No required resources to verify"
            return 0
        fi
        resources_to_check=($required_resources)
    fi
    
    log::info "Starting priority-based verification of ${#resources_to_check[@]} resources: ${resources_to_check[*]}"
    
    # Get startup_order for each resource from runtime.json
    declare -A resource_priorities
    declare -A resource_timeouts
    declare -A resource_recovery_attempts
    
    for resource in "${resources_to_check[@]}"; do
        local runtime_json="${APP_ROOT}/resources/${resource}/config/runtime.json"
        local startup_order=500  # default
        local startup_timeout=30  # default
        local recovery_attempts=2  # default
        
        if [[ -f "$runtime_json" ]]; then
            startup_order=$(jq -r '.startup_order // 500' "$runtime_json" 2>/dev/null || echo "500")
            startup_timeout=$(jq -r '.startup_timeout // 30' "$runtime_json" 2>/dev/null || echo "30")
            recovery_attempts=$(jq -r '.recovery_attempts // 2' "$runtime_json" 2>/dev/null || echo "2")
        else
            log::debug "No runtime.json for $resource, using defaults"
        fi
        
        resource_priorities["$resource"]="$startup_order"
        resource_timeouts["$resource"]="$startup_timeout"
        resource_recovery_attempts["$resource"]="$recovery_attempts"
        log::debug "Resource $resource: order=$startup_order, timeout=$startup_timeout, recovery=$recovery_attempts"
    done
    
    # Group resources by priority bands
    declare -A priority_groups
    for resource in "${resources_to_check[@]}"; do
        local priority="${resource_priorities[$resource]}"
        local band
        if [[ $priority -ge 1000 ]]; then
            band="1000"
        elif [[ $priority -ge 500 ]]; then
            band="500"
        elif [[ $priority -ge 100 ]]; then
            band="100"
        else
            band="50"
        fi
        
        if [[ -n "${priority_groups[$band]:-}" ]]; then
            priority_groups["$band"]="${priority_groups[$band]} $resource"
        else
            priority_groups["$band"]="$resource"
        fi
    done
    
    # Start each priority band in order (highest first)
    local all_failed_resources=()
    for band in 1000 500 100 50; do
        if [[ -n "${priority_groups[$band]:-}" ]]; then
            local band_resources=(${priority_groups[$band]})
            log::info "Starting priority band $band: ${band_resources[*]}"
            
            # Start this band in parallel
            local start_pids=()
            for resource in "${band_resources[@]}"; do
                {
                    vrooli resource start "$resource" >/dev/null 2>&1 || true
                } &
                start_pids+=($!)
            done
            
            # Wait for all start commands in this band
            for pid in "${start_pids[@]}"; do
                wait "$pid" || true
            done
            
            # Verify this band is running before starting next
            local band_failed=()
            for resource in "${band_resources[@]}"; do
                local retry_count=0
                local verified=false
                
                # Use resource-specific timeout and recovery attempts
                local max_retries=$((${resource_recovery_attempts[$resource]:-2} * 10))
                local resource_timeout=${resource_timeouts[$resource]:-30}
                local start_time=$(date +%s)
                
                while [[ $retry_count -lt $max_retries ]] && [[ "$verified" == "false" ]]; do
                    local is_running
                    is_running=$(vrooli resource status "$resource" --json 2>/dev/null | jq -r '.running // false' 2>/dev/null || echo "false")
                    
                    if [[ "$is_running" == "true" ]]; then
                        log::success "✅ $resource"
                        verified=true
                        break
                    fi
                    
                    # Check if we've exceeded resource-specific timeout
                    local current_time=$(date +%s)
                    local elapsed=$((current_time - start_time))
                    if [[ $elapsed -gt $resource_timeout ]]; then
                        log::warning "Resource $resource exceeded timeout of ${resource_timeout}s"
                        break
                    fi
                    
                    retry_count=$((retry_count + 1))
                    if [[ $retry_count -lt $max_retries ]]; then
                        sleep "$RESOURCE_VERIFY_RETRY_DELAY"
                    fi
                done
                
                if [[ "$verified" == "false" ]]; then
                    band_failed+=("$resource")
                    log::error "❌ $resource (failed after $retry_count attempts)"
                fi
            done
            
            all_failed_resources+=("${band_failed[@]}")
            
            # Add brief pause between priority bands for stability
            if [[ ${#band_failed[@]} -eq 0 ]] && [[ $band != "50" ]]; then
                sleep 2
            fi
        fi
    done
    
    if [[ ${#all_failed_resources[@]} -gt 0 ]]; then
        log::error "Failed to verify resources: ${all_failed_resources[*]}"
        return 1
    fi
    
    log::success "🚀 All resources verified and running"
    return 0
}

#######################################
# Export functions for use in other scripts
#######################################
export -f resource_verify::check_single
export -f resource_verify::get_required_resources  
export -f resource_verify::verify_required
export -f resource_verify::verify_parallel
export -f resource_verify::verify_with_priority
