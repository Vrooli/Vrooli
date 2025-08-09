#!/bin/bash
# Smart File Photo Manager - Main startup script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# shellcheck disable=SC1091
source "$(cd "$SCRIPT_DIR" && cd ../../../lib/utils && pwd)/var.sh"
# shellcheck disable=SC1091
source "$var_LOG_FILE"

# Source resource helper functions
source "$var_SCRIPTS_RESOURCES_DIR/lib/resource-helper.sh"


# Check if required resources are running
check_prerequisites() {
    log_info "Checking prerequisite resources..."
    
    local required_resources=("postgres" "redis" "qdrant" "minio" "ollama" "unstructured-io" "n8n" "windmill")
    local failed_resources=()
    
    for resource in "${required_resources[@]}"; do
        if ! resource_is_healthy "$resource"; then
            failed_resources+=("$resource")
        fi
    done
    
    if [ ${#failed_resources[@]} -gt 0 ]; then
        log_error "Required resources not available: ${failed_resources[*]}"
        log_error "Please ensure all resources are running before starting the file manager"
        return 1
    fi
    
    log_success "All prerequisite resources are healthy"
}

# Wait for services to be ready
wait_for_service() {
    local service_name="$1"
    local port="$2"
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for $service_name on port $port..."
    
    while [ $attempt -lt $max_attempts ]; do
        if timeout 3 bash -c "</dev/tcp/localhost/$port" 2>/dev/null; then
            log_success "$service_name is ready"
            return 0
        fi
        
        sleep 2
        ((attempt++))
        
        if [ $((attempt % 10)) -eq 0 ]; then
            log_info "Still waiting for $service_name... (attempt $attempt/$max_attempts)"
        fi
    done
    
    log_error "$service_name failed to start within timeout"
    return 1
}

# Setup PostgreSQL database
setup_database() {
    log_info "Setting up PostgreSQL database..."
    
    if ! "$SCRIPT_DIR/setup-postgres.sh"; then
        log_error "Database setup failed"
        return 1
    fi
    
    log_success "Database setup completed"
}

# Setup Qdrant vector database
setup_vector_db() {
    log_info "Setting up Qdrant vector database..."
    
    if ! "$SCRIPT_DIR/setup-qdrant.sh"; then
        log_error "Vector database setup failed"
        return 1
    fi
    
    log_success "Vector database setup completed"
}

# Setup MinIO storage
setup_storage() {
    log_info "Setting up MinIO object storage..."
    
    if ! "$SCRIPT_DIR/setup-minio.sh"; then
        log_error "Object storage setup failed"
        return 1
    fi
    
    log_success "Object storage setup completed"
}

# Pull required Ollama models
setup_models() {
    log_info "Setting up Ollama models..."
    
    if ! "$SCRIPT_DIR/pull-models.sh"; then
        log_error "Model setup failed"
        return 1
    fi
    
    log_success "Model setup completed"
}

# Import automation workflows
setup_automations() {
    log_info "Setting up automation workflows..."
    
    if ! "$SCRIPT_DIR/import-automations.sh"; then
        log_error "Automation setup failed"
        return 1
    fi
    
    log_success "Automation setup completed"
}

# Start health monitoring
start_monitoring() {
    log_info "Starting health monitoring..."
    
    # Start background health check process
    (
        while true; do
            if ! check_prerequisites >/dev/null 2>&1; then
                log_error "Health check failed - some resources are down"
            fi
            sleep 30
        done
    ) &
    
    local monitor_pid=$!
    echo $monitor_pid > "$SCENARIO_ROOT/.monitor.pid"
    
    log_success "Health monitoring started (PID: $monitor_pid)"
}

# Cleanup function
cleanup() {
    log_info "Performing cleanup..."
    
    if [ -f "$SCENARIO_ROOT/.monitor.pid" ]; then
        local monitor_pid
        monitor_pid=$(cat "$SCENARIO_ROOT/.monitor.pid")
        if kill -0 "$monitor_pid" 2>/dev/null; then
            kill "$monitor_pid"
            log_info "Stopped health monitor"
        fi
        rm -f "$SCENARIO_ROOT/.monitor.pid"
    fi
}

# Signal handlers
trap cleanup EXIT INT TERM

# Main execution
main() {
    log_info "Starting Smart File Photo Manager..."
    
    # Check prerequisites first
    if ! check_prerequisites; then
        log_error "Prerequisites check failed. Ensure all resources are running."
        exit 1
    fi
    
    # Run setup steps
    setup_database
    setup_vector_db
    setup_storage
    setup_models
    setup_automations
    
    # Wait for critical services
    wait_for_service "postgres" "5435"
    wait_for_service "qdrant" "6335"
    wait_for_service "minio" "9001"
    wait_for_service "ollama" "11434"
    wait_for_service "n8n" "5680"
    wait_for_service "windmill" "8002"
    
    # Start monitoring
    start_monitoring
    
    log_success "Smart File Photo Manager is ready!"
    log_info "Access the file manager at: http://localhost:8002"
    log_info "n8n workflows available at: http://localhost:5680"
    log_info "Press Ctrl+C to stop..."
    
    # Keep running
    while true; do
        sleep 10
    done
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi