#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Vrooli Resource Management Functions
# 
# This file provides the core resource management capabilities that generated
# scenario apps inherit. It bridges the gap between scenario apps and Vrooli's
# resource infrastructure.
#
# Functions provided:
#   - start_required_resources: Start all resources marked as required
#   - wait_for_resource_health: Wait for resources to be healthy
#   - inject_initialization_data: Inject data using runtime manifests
#   - start_application_services: Start app-specific services
#   - stop_required_resources: Clean shutdown of resources
#   - start_health_monitoring: Begin continuous health checks
#   - stop_health_monitoring: Stop health monitoring
#
# Expected environment:
#   - APP_ROOT: Root directory of the generated app
#   - VROOLI_RUNTIME_DIR: Vrooli runtime directory (optional)
#   - INJECTION_MANIFEST: Path to injection manifest JSON
#
################################################################################

# Ensure we have access to Vrooli's resource scripts
if [[ -z "${VROOLI_RUNTIME_DIR:-}" ]]; then
    # Try to find Vrooli installation
    if [[ -d "/opt/vrooli" ]]; then
        export VROOLI_RUNTIME_DIR="/opt/vrooli"
    elif [[ -d "${HOME}/Vrooli" ]]; then
        export VROOLI_RUNTIME_DIR="${HOME}/Vrooli"
    else
        echo "ERROR: Cannot find Vrooli runtime directory"
        echo "Set VROOLI_RUNTIME_DIR environment variable"
        exit 1
    fi
fi

# Source Vrooli's common resource utilities
RESOURCES_DIR="${VROOLI_RUNTIME_DIR}/scripts/resources"
if [[ -f "${RESOURCES_DIR}/common.sh" ]]; then
    # shellcheck disable=SC1091
    source "${RESOURCES_DIR}/common.sh"
else
    echo "ERROR: Cannot find Vrooli resource common utilities"
    exit 1
fi

# Source JSON utilities for robust service.json parsing
if [[ -f "${VROOLI_RUNTIME_DIR}/scripts/lib/utils/json.sh" ]]; then
    # shellcheck disable=SC1091
    source "${VROOLI_RUNTIME_DIR}/scripts/lib/utils/json.sh"
else
    echo "ERROR: Cannot find JSON utilities - ensure Vrooli is properly installed"
    exit 1
fi

# Track health monitoring PID
HEALTH_MONITOR_PID=""

################################################################################
# Start all required resources based on service.json
################################################################################
start_required_resources() {
    local service_config="${APP_ROOT}/config/service.json"
    
    if [[ ! -f "$service_config" ]]; then
        log_error "Service configuration not found: $service_config"
        return 1
    fi
    
    log_info "Starting required resources..."
    
    # Extract required resources using JSON utilities
    local required_resources
    required_resources=$(json::get_required_resources "" "$service_config" || echo "")
    
    if [[ -z "$required_resources" ]]; then
        log_info "No required resources defined"
        return 0
    fi
    
    local failed_resources=()
    
    # Start each required resource
    for resource in $required_resources; do
        log_info "Starting resource: $resource"
        
        # Find the resource's manage.sh script
        local manage_script
        manage_script=$(find "${RESOURCES_DIR}" -name "manage.sh" -path "*/${resource}/*" 2>/dev/null | head -1)
        
        if [[ -z "$manage_script" ]]; then
            log_warning "No management script found for resource: $resource"
            continue
        fi
        
        # Check if resource is already running
        if "$manage_script" --action status >/dev/null 2>&1; then
            log_info "Resource already running: $resource"
            continue
        fi
        
        # Start the resource
        if "$manage_script" --action start; then
            log_success "Started resource: $resource"
        else
            log_error "Failed to start resource: $resource"
            failed_resources+=("$resource")
        fi
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        log_error "Failed to start resources: ${failed_resources[*]}"
        return 1
    fi
    
    log_success "All required resources started"
    return 0
}

################################################################################
# Wait for all resources to be healthy
################################################################################
wait_for_resource_health() {
    local service_config="${APP_ROOT}/config/service.json"
    local max_wait_time=300  # 5 minutes total
    local check_interval=5   # Check every 5 seconds
    
    log_info "Waiting for resources to be healthy..."
    
    # Extract resources with health checks using JSON utilities
    local resources_with_health
    # Note: For complex health check extraction, we use a targeted approach
    if json::load_service_config "$service_config"; then
        resources_with_health=$(echo "$JSON_CONFIG_CACHE" | jq -r '
            .resources | 
            to_entries[] | 
            .value | 
            to_entries[] | 
            select(.value.healthCheck // false) | 
            "\(.key)|\(.value.healthCheck.type // "tcp")|\(.value.healthCheck.port // 0)|\(.value.healthCheck.endpoint // "")"
        ' 2>/dev/null || echo "")
    else
        resources_with_health=""
    fi
    
    if [[ -z "$resources_with_health" ]]; then
        log_info "No health checks defined"
        return 0
    fi
    
    local start_time=$SECONDS
    local all_healthy=false
    
    while [[ $((SECONDS - start_time)) -lt $max_wait_time ]]; do
        all_healthy=true
        
        while IFS='|' read -r resource check_type port endpoint; do
            [[ -z "$resource" ]] && continue
            
            case "$check_type" in
                tcp)
                    if resources::is_service_running "$port"; then
                        log_debug "✓ $resource: healthy (port $port)"
                    else
                        log_debug "✗ $resource: not ready (port $port)"
                        all_healthy=false
                    fi
                    ;;
                http)
                    local url="http://localhost:${port}${endpoint}"
                    if resources::check_http_health "$url"; then
                        log_debug "✓ $resource: healthy (HTTP endpoint)"
                    else
                        log_debug "✗ $resource: not ready (HTTP endpoint)"
                        all_healthy=false
                    fi
                    ;;
            esac
        done <<< "$resources_with_health"
        
        if [[ "$all_healthy" == true ]]; then
            log_success "All resources are healthy"
            return 0
        fi
        
        sleep "$check_interval"
    done
    
    log_error "Timeout waiting for resources to be healthy"
    return 1
}

################################################################################
# Inject initialization data using the injection manifest
################################################################################
inject_initialization_data() {
    local manifest_file="${1:-$INJECTION_MANIFEST}"
    
    if [[ ! -f "$manifest_file" ]]; then
        log_info "No injection manifest found, skipping data injection"
        return 0
    fi
    
    log_info "Injecting initialization data..."
    
    # Use the runtime injection engine
    local runtime_engine="${VROOLI_RUNTIME_DIR}/scripts/scenarios/injection/runtime-engine.sh"
    
    if [[ ! -x "$runtime_engine" ]]; then
        log_error "Runtime injection engine not found: $runtime_engine"
        return 1
    fi
    
    # Execute the runtime injection engine with the manifest
    if "$runtime_engine" "$manifest_file"; then
        log_success "All initialization data injected"
        return 0
    else
        log_error "Initialization data injection failed"
        return 1
    fi
}

################################################################################
# Start application-specific services
################################################################################
start_application_services() {
    # Look for app-specific startup scripts
    local app_startup_dir="${APP_ROOT}/scripts/startup"
    
    if [[ -d "$app_startup_dir" ]]; then
        log_info "Starting application services..."
        
        # Execute all startup scripts in order
        for script in "$app_startup_dir"/*.sh; do
            if [[ -x "$script" ]]; then
                local script_name
                script_name=$(basename "$script")
                log_info "Running startup script: $script_name"
                
                if "$script"; then
                    log_success "✓ Completed: $script_name"
                else
                    log_error "✗ Failed: $script_name"
                    return 1
                fi
            fi
        done
    fi
    
    # Check for main application process
    local main_app="${APP_ROOT}/app/main.sh"
    if [[ -x "$main_app" ]]; then
        log_info "Starting main application..."
        "$main_app" &
        log_success "Main application started (PID: $!)"
    fi
    
    return 0
}

################################################################################
# Stop all required resources
################################################################################
stop_required_resources() {
    local service_config="${APP_ROOT}/config/service.json"
    
    log_info "Stopping required resources..."
    
    # Extract required resources using JSON utilities
    local required_resources
    required_resources=$(json::get_required_resources "" "$service_config" || echo "")
    
    for resource in $required_resources; do
        log_info "Stopping resource: $resource"
        
        # Find the resource's manage.sh script
        local manage_script
        manage_script=$(find "${RESOURCES_DIR}" -name "manage.sh" -path "*/${resource}/*" 2>/dev/null | head -1)
        
        if [[ -n "$manage_script" ]]; then
            "$manage_script" --action stop || true
        fi
    done
    
    log_info "All resources stopped"
}

################################################################################
# Start continuous health monitoring
################################################################################
start_health_monitoring() {
    local health_check_script="${APP_ROOT}/scripts/health-check.sh"
    
    if [[ ! -x "$health_check_script" ]]; then
        log_debug "No health check script found, skipping monitoring"
        return 0
    fi
    
    log_info "Starting health monitoring..."
    
    # Run health check in background loop
    (
        while true; do
            if ! "$health_check_script" >/dev/null 2>&1; then
                log_warning "Health check failed at $(date)"
            fi
            sleep 30  # Check every 30 seconds
        done
    ) &
    
    HEALTH_MONITOR_PID=$!
    log_info "Health monitoring started (PID: $HEALTH_MONITOR_PID)"
}

################################################################################
# Stop health monitoring
################################################################################
stop_health_monitoring() {
    if [[ -n "$HEALTH_MONITOR_PID" ]] && kill -0 "$HEALTH_MONITOR_PID" 2>/dev/null; then
        log_info "Stopping health monitoring..."
        kill "$HEALTH_MONITOR_PID" 2>/dev/null || true
        HEALTH_MONITOR_PID=""
    fi
}

################################################################################
# Stop application services
################################################################################
stop_application_services() {
    log_info "Stopping application services..."
    
    # Stop main application if running
    if [[ -f "${APP_ROOT}/app.pid" ]]; then
        local app_pid
        app_pid=$(cat "${APP_ROOT}/app.pid")
        if kill -0 "$app_pid" 2>/dev/null; then
            kill "$app_pid" || true
            rm -f "${APP_ROOT}/app.pid"
        fi
    fi
    
    # Run any shutdown scripts
    local shutdown_dir="${APP_ROOT}/scripts/shutdown"
    if [[ -d "$shutdown_dir" ]]; then
        for script in "$shutdown_dir"/*.sh; do
            if [[ -x "$script" ]]; then
                "$script" || true
            fi
        done
    fi
    
    log_info "Application services stopped"
}

# Export all functions for use by apps
export -f start_required_resources
export -f wait_for_resource_health
export -f inject_initialization_data
export -f start_application_services
export -f stop_required_resources
export -f start_health_monitoring
export -f stop_health_monitoring
export -f stop_application_services