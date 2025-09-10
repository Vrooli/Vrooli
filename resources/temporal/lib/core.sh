#!/usr/bin/env bash
# Temporal Resource - Core Library Functions

set -euo pipefail

# Resource configuration
TEMPORAL_PORT="${TEMPORAL_PORT:-7233}"
TEMPORAL_GRPC_PORT="${TEMPORAL_GRPC_PORT:-7234}"
TEMPORAL_DB_PASSWORD="${TEMPORAL_DB_PASSWORD:-temporal}"
TEMPORAL_NAMESPACE="${TEMPORAL_NAMESPACE:-default}"
CONTAINER_NAME="temporal-server"
DB_CONTAINER_NAME="temporal-postgres"
NETWORK_NAME="temporal-network"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

# Install Temporal and dependencies
install_temporal() {
    log_info "Installing Temporal resource..."
    
    # Check if already installed
    if docker ps -a | grep -q "$CONTAINER_NAME"; then
        log_warning "Temporal is already installed"
        exit 2  # Already installed (skipped)
    fi
    
    # Create network if not exists
    if ! docker network ls | grep -q "$NETWORK_NAME"; then
        log_info "Creating Docker network: $NETWORK_NAME"
        docker network create "$NETWORK_NAME"
    fi
    
    # Pull required images
    log_info "Pulling Docker images..."
    docker pull temporalio/server:latest || {
        log_error "Failed to pull Temporal server image"
        exit 1
    }
    
    docker pull postgres:13 || {
        log_error "Failed to pull PostgreSQL image"
        exit 1
    }
    
    log_info "Temporal installed successfully"
    exit 0
}

# Uninstall Temporal
uninstall_temporal() {
    log_info "Uninstalling Temporal resource..."
    
    # Stop containers if running
    if docker ps | grep -q "$CONTAINER_NAME"; then
        docker stop "$CONTAINER_NAME" 2>/dev/null || true
    fi
    
    if docker ps | grep -q "$DB_CONTAINER_NAME"; then
        docker stop "$DB_CONTAINER_NAME" 2>/dev/null || true
    fi
    
    # Remove containers
    docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
    docker rm -f "$DB_CONTAINER_NAME" 2>/dev/null || true
    
    # Optionally remove network (if no other containers using it)
    if docker network ls | grep -q "$NETWORK_NAME"; then
        docker network rm "$NETWORK_NAME" 2>/dev/null || true
    fi
    
    log_info "Temporal uninstalled successfully"
    exit 0
}

# Start Temporal server
start_temporal() {
    log_info "Starting Temporal server..."
    
    # Ensure network exists
    if ! docker network ls | grep -q "$NETWORK_NAME"; then
        docker network create "$NETWORK_NAME"
    fi
    
    # Start PostgreSQL if not running
    if ! docker ps | grep -q "$DB_CONTAINER_NAME"; then
        log_info "Starting PostgreSQL database..."
        docker run -d \
            --name "$DB_CONTAINER_NAME" \
            --network "$NETWORK_NAME" \
            -e POSTGRES_PASSWORD="$TEMPORAL_DB_PASSWORD" \
            -e POSTGRES_USER=temporal \
            -e POSTGRES_DB=temporal \
            postgres:13
        
        # Wait for PostgreSQL to be ready
        log_info "Waiting for PostgreSQL to be ready..."
        sleep 5
    fi
    
    # Start Temporal server
    if docker ps | grep -q "$CONTAINER_NAME"; then
        log_warning "Temporal is already running"
        exit 0
    fi
    
    log_info "Starting Temporal server container..."
    docker run -d \
        --name "$CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        -p "${TEMPORAL_PORT}:7233" \
        -p "${TEMPORAL_GRPC_PORT}:7234" \
        -e DB=postgresql \
        -e DB_PORT=5432 \
        -e POSTGRES_USER=temporal \
        -e POSTGRES_PWD="$TEMPORAL_DB_PASSWORD" \
        -e POSTGRES_SEEDS="${DB_CONTAINER_NAME}" \
        -e DYNAMIC_CONFIG_FILE_PATH=config/development.yaml \
        temporalio/server:latest
    
    # Wait for startup
    log_info "Waiting for Temporal to start..."
    local retries=30
    while [ $retries -gt 0 ]; do
        if timeout 5 curl -sf "http://localhost:${TEMPORAL_PORT}/health" >/dev/null 2>&1; then
            log_info "Temporal started successfully"
            log_info "Web UI available at: http://localhost:${TEMPORAL_PORT}"
            log_info "gRPC endpoint: localhost:${TEMPORAL_GRPC_PORT}"
            exit 0
        fi
        retries=$((retries - 1))
        sleep 2
    done
    
    log_error "Temporal failed to start within timeout"
    exit 1
}

# Stop Temporal server
stop_temporal() {
    log_info "Stopping Temporal server..."
    
    if ! docker ps | grep -q "$CONTAINER_NAME"; then
        log_warning "Temporal is not running"
        exit 0
    fi
    
    docker stop "$CONTAINER_NAME"
    docker stop "$DB_CONTAINER_NAME" 2>/dev/null || true
    
    log_info "Temporal stopped successfully"
    exit 0
}

# Show service status
show_status() {
    echo "Temporal Service Status"
    echo "======================="
    
    # Check if containers are running
    if docker ps | grep -q "$CONTAINER_NAME"; then
        echo -e "Server: ${GREEN}Running${NC}"
        
        # Check health endpoint
        if timeout 5 curl -sf "http://localhost:${TEMPORAL_PORT}/health" >/dev/null 2>&1; then
            echo -e "Health: ${GREEN}Healthy${NC}"
        else
            echo -e "Health: ${YELLOW}Unhealthy${NC}"
        fi
        
        # Show container info
        echo ""
        echo "Container Information:"
        docker ps --filter "name=$CONTAINER_NAME" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        
    else
        echo -e "Server: ${RED}Stopped${NC}"
    fi
    
    # Check database
    echo ""
    if docker ps | grep -q "$DB_CONTAINER_NAME"; then
        echo -e "Database: ${GREEN}Running${NC}"
    else
        echo -e "Database: ${RED}Stopped${NC}"
    fi
    
    echo ""
    echo "Configuration:"
    echo "  Web UI Port: ${TEMPORAL_PORT}"
    echo "  gRPC Port: ${TEMPORAL_GRPC_PORT}"
    echo "  Namespace: ${TEMPORAL_NAMESPACE}"
    
    exit 0
}

# Show service logs
show_logs() {
    local tail_lines="${1:-100}"
    
    if [[ "$1" == "--tail" ]]; then
        tail_lines="${2:-100}"
    fi
    
    log_info "Showing last $tail_lines lines of Temporal logs..."
    docker logs "$CONTAINER_NAME" --tail "$tail_lines" 2>&1
    exit 0
}

# Show connection credentials
show_credentials() {
    echo "Temporal Connection Credentials"
    echo "==============================="
    echo ""
    echo "Web UI:"
    echo "  URL: http://localhost:${TEMPORAL_PORT}"
    echo ""
    echo "gRPC Endpoint:"
    echo "  Address: localhost:${TEMPORAL_GRPC_PORT}"
    echo ""
    echo "Default Namespace: ${TEMPORAL_NAMESPACE}"
    echo ""
    echo "Client Connection Example:"
    echo "  import { WorkflowClient } from '@temporalio/client';"
    echo "  const client = new WorkflowClient({"
    echo "    address: 'localhost:${TEMPORAL_GRPC_PORT}'"
    echo "  });"
    echo ""
    echo "CLI Access:"
    echo "  docker exec $CONTAINER_NAME tctl --help"
    
    exit 0
}

# Workflow content management functions
list_workflows() {
    log_info "Listing workflows..."
    
    if ! docker ps | grep -q "$CONTAINER_NAME"; then
        log_error "Temporal is not running"
        exit 1
    fi
    
    docker exec "$CONTAINER_NAME" tctl workflow list
    exit 0
}

add_workflow() {
    local workflow_file="${1:-}"
    
    if [[ -z "$workflow_file" ]]; then
        log_error "Workflow file required"
        echo "Usage: resource-temporal content add <workflow-file>"
        exit 1
    fi
    
    log_info "Adding workflow from $workflow_file..."
    # Implementation would depend on workflow deployment strategy
    log_warning "Workflow deployment requires worker implementation"
    exit 0
}

get_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log_error "Workflow ID required"
        echo "Usage: resource-temporal content get <workflow-id>"
        exit 1
    fi
    
    log_info "Getting workflow $workflow_id..."
    docker exec "$CONTAINER_NAME" tctl workflow describe --workflow-id "$workflow_id"
    exit 0
}

remove_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log_error "Workflow ID required"
        echo "Usage: resource-temporal content remove <workflow-id>"
        exit 1
    fi
    
    log_info "Canceling workflow $workflow_id..."
    docker exec "$CONTAINER_NAME" tctl workflow cancel --workflow-id "$workflow_id"
    exit 0
}

execute_workflow() {
    local workflow_type="${1:-}"
    local task_queue="${2:-default}"
    local input="${3:-'{}'}"
    
    if [[ -z "$workflow_type" ]]; then
        log_error "Workflow type required"
        echo "Usage: resource-temporal content execute <workflow-type> [task-queue] [input-json]"
        exit 1
    fi
    
    log_info "Executing workflow $workflow_type..."
    docker exec "$CONTAINER_NAME" tctl workflow start \
        --task-queue "$task_queue" \
        --type "$workflow_type" \
        --input "$input"
    exit 0
}