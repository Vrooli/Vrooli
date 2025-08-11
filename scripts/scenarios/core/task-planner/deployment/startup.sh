#!/bin/bash

# Task Planner - Service Startup Script
# Initializes all required services and components for the AI-powered task management system

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../../" && pwd)"

# Load configuration
source "${SCRIPT_DIR}/../initialization/configuration/resource-urls.json" 2>/dev/null || true

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "${PROJECT_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $*${NC}"
}

success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] ✅ $*${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] ⚠️  $*${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ❌ $*${NC}" >&2
}

# Health check function
check_service() {
    local service_name="$1"
    local url="$2"
    local timeout="${3:-10}"
    local retries="${4:-3}"
    
    for i in $(seq 1 $retries); do
        if curl -s --connect-timeout "$timeout" "$url" > /dev/null 2>&1; then
            success "$service_name is healthy"
            return 0
        fi
        warn "$service_name health check failed (attempt $i/$retries)"
        sleep 2
    done
    
    error "$service_name failed health check after $retries attempts"
    return 1
}

# Wait for service to be ready
wait_for_service() {
    local service_name="$1"
    local url="$2" 
    local timeout="${3:-30}"
    local check_interval="${4:-2}"
    
    log "Waiting for $service_name to be ready..."
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if curl -s --connect-timeout 5 "$url" > /dev/null 2>&1; then
            success "$service_name is ready"
            return 0
        fi
        sleep $check_interval
        elapsed=$((elapsed + check_interval))
    done
    
    error "$service_name did not become ready within ${timeout}s"
    return 1
}

# Import environment variables from scenario configuration
setup_environment() {
    log "Setting up environment variables..."
    
    # Export required environment variables
    export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-$(openssl rand -base64 32)}"
    export TASK_PLANNER_SECRET="${TASK_PLANNER_SECRET:-$(openssl rand -base64 32)}"
    export CLAUDE_API_KEY="${CLAUDE_API_KEY:-}"
    export WEBHOOK_SECRET="${WEBHOOK_SECRET:-$(openssl rand -base64 32)}"
    export TASK_PLANNER_JWT_SECRET="${TASK_PLANNER_JWT_SECRET:-$(openssl rand -base64 32)}"
    
    # Task Planner specific configuration
    export TASK_PLANNER_API_PORT="8092"
    export NODE_ENV="production"
    export LOG_LEVEL="info"
    
    success "Environment variables configured"
}

# Initialize PostgreSQL database
setup_database() {
    log "Setting up PostgreSQL database..."
    
    if ! "${SCRIPT_DIR}/setup-postgres.sh"; then
        error "PostgreSQL setup failed"
        return 1
    fi
    
    success "PostgreSQL database ready"
}

# Initialize Qdrant vector database
setup_vector_db() {
    log "Setting up Qdrant vector database..."
    
    if ! "${SCRIPT_DIR}/setup-qdrant.sh"; then
        error "Qdrant setup failed"
        return 1
    fi
    
    success "Qdrant vector database ready"
}

# Import automation workflows and apps
setup_automations() {
    log "Setting up automation workflows..."
    
    if ! "${SCRIPT_DIR}/import-automations.sh"; then
        error "Automation setup failed"
        return 1
    fi
    
    success "Automation workflows ready"
}

# Start Task Planner API server
start_api_server() {
    log "Starting Task Planner API server..."
    
    # Check if API server script exists
    if [[ -f "${SCRIPT_DIR}/api-server.js" ]]; then
        # Start API server in background
        cd "$SCRIPT_DIR"
        node api-server.js > /var/log/task-planner-api.log 2>&1 &
        API_PID=$!
        echo $API_PID > /var/run/task-planner-api.pid
        
        # Wait for API to be ready
        if wait_for_service "Task Planner API" "http://localhost:8092/api/health" 30; then
            success "Task Planner API started (PID: $API_PID)"
        else
            error "Failed to start Task Planner API"
            return 1
        fi
    else
        warn "API server script not found, skipping direct API startup"
    fi
}

# Verify all core services are running
verify_services() {
    log "Verifying all services are healthy..."
    
    local services_failed=0
    
    # Check PostgreSQL
    if ! check_service "PostgreSQL" "postgresql://localhost:5434/task_planner"; then
        ((services_failed++))
    fi
    
    # Check Redis
    if ! check_service "Redis" "redis://localhost:6380"; then
        ((services_failed++))
    fi
    
    # Check Qdrant
    if ! check_service "Qdrant" "http://localhost:6334/health"; then
        ((services_failed++))
    fi
    
    # Check n8n
    if ! check_service "n8n" "http://localhost:5679/healthz"; then
        ((services_failed++))
    fi
    
    # Check Windmill
    if ! check_service "Windmill" "http://localhost:8001/api/version"; then
        ((services_failed++))
    fi
    
    # Check Ollama
    if ! check_service "Ollama" "http://localhost:11434/api/tags"; then
        ((services_failed++))
    fi
    
    # Check SearXNG
    if ! check_service "SearXNG" "http://localhost:9201/healthz"; then
        warn "SearXNG not available - research functionality may be limited"
    fi
    
    # Check Agent-S2
    if ! check_service "Agent-S2" "http://localhost:4113/api/health"; then
        warn "Agent-S2 not available - web research functionality may be limited"
    fi
    
    # Check Task Planner API if started
    if [[ -f "/var/run/task-planner-api.pid" ]]; then
        if ! check_service "Task Planner API" "http://localhost:8092/api/health"; then
            ((services_failed++))
        fi
    fi
    
    if [ $services_failed -eq 0 ]; then
        success "All core services are healthy"
        return 0
    else
        error "$services_failed core services failed health checks"
        return 1
    fi
}

# Display service information
show_service_info() {
    log "Task Planner Service Information:"
    echo ""
    echo "🎯 Task Planner Dashboard: http://localhost:8001/apps/get/task-dashboard"
    echo "🔧 API Endpoint: http://localhost:8092/api"
    echo "📊 n8n Workflows: http://localhost:5679"
    echo "⚙️  Windmill Workspace: http://localhost:8001/w/task_planner"
    echo ""
    echo "📋 Core Services:"
    echo "  • PostgreSQL: localhost:5434/task_planner"
    echo "  • Redis: localhost:6380"
    echo "  • Qdrant: http://localhost:6334"
    echo "  • Ollama: http://localhost:11434"
    echo ""
    echo "🔍 Optional Services:"
    echo "  • SearXNG: http://localhost:9201"
    echo "  • Agent-S2: http://localhost:4113"
    echo ""
    echo "📚 CLI Usage:"
    echo "  task-planner parse 'Add login page'"
    echo "  task-planner list --status=backlog"
    echo "  task-planner research <task-id>"
    echo "  task-planner implement <task-id>"
    echo ""
}

# Cleanup on exit
cleanup() {
    log "Cleaning up..."
    
    # Stop API server if running
    if [[ -f "/var/run/task-planner-api.pid" ]]; then
        local pid=$(cat /var/run/task-planner-api.pid)
        if kill -0 "$pid" 2>/dev/null; then
            log "Stopping Task Planner API (PID: $pid)"
            kill "$pid"
            trash::safe_remove /var/run/task-planner-api.pid --temp
        fi
    fi
}

trap cleanup EXIT

# Main execution
main() {
    log "Starting Task Planner scenario setup..."
    echo ""
    
    # Setup phases
    setup_environment
    
    # Initialize core components  
    setup_database
    setup_vector_db
    setup_automations
    
    # Start additional services
    start_api_server
    
    # Verify everything is working
    verify_services
    
    # Show information
    show_service_info
    
    success "Task Planner scenario startup completed successfully!"
    
    # If running interactively, keep the script running
    if [[ -t 0 ]]; then
        log "Press Ctrl+C to stop all services"
        while true; do
            sleep 10
            # Periodic health check
            if ! verify_services > /dev/null 2>&1; then
                warn "Some services are unhealthy - check logs"
            fi
        done
    fi
}

# Run main function
main "$@"