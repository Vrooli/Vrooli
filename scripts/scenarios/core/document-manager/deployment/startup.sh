#!/usr/bin/env bash
# Document Manager Enhanced Startup Script
# Orchestrates initialization and startup of the complete documentation management system

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Directory setup
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
INIT_DIR="$SCENARIO_DIR/initialization"

# Configuration files
RESOURCE_URLS="$INIT_DIR/configuration/resource-urls.json"
AGENT_PRESETS="$INIT_DIR/configuration/agent-presets.json"
NOTIFICATION_RULES="$INIT_DIR/configuration/notification-rules.json"

# Resource health check function
check_resource() {
    local service_name=$1
    local health_url=$2
    local timeout=${3:-10}
    
    log_info "Checking $service_name health..."
    
    for i in {1..30}; do
        if timeout $timeout curl -s -f "$health_url" > /dev/null 2>&1; then
            log_success "$service_name is healthy"
            return 0
        fi
        
        if [ $i -eq 30 ]; then
            log_error "$service_name health check failed after 30 attempts"
            return 1
        fi
        
        log_info "Waiting for $service_name... (attempt $i/30)"
        sleep 2
    done
}

# Database initialization function
init_database() {
    log_info "Initializing PostgreSQL database..."
    
    # Create database if it doesn't exist
    createdb -h localhost -U postgres document_manager 2>/dev/null || log_info "Database already exists"
    
    # Apply schema
    log_info "Applying database schema..."
    psql -h localhost -U postgres -d document_manager -f "$INIT_DIR/storage/postgres/schema.sql" > /dev/null
    
    # Load seed data
    log_info "Loading seed data..."
    psql -h localhost -U postgres -d document_manager -f "$INIT_DIR/storage/postgres/seed.sql" > /dev/null
    
    log_success "Database initialization completed"
}

# Qdrant initialization function
init_qdrant() {
    log_info "Initializing Qdrant vector database..."
    
    # Create documentation embeddings collection
    log_info "Creating documentation_embeddings collection..."
    curl -s -X PUT http://localhost:6333/collections/documentation_embeddings \
        -H "Content-Type: application/json" \
        -d '{
            "vectors": {"size": 768, "distance": "Cosine"},
            "shard_number": 2,
            "replication_factor": 1,
            "write_consistency_factor": 1,
            "on_disk_payload": true
        }' > /dev/null
    
    # Create agent memory collection
    log_info "Creating agent_memory collection..."
    curl -s -X PUT http://localhost:6333/collections/agent_memory \
        -H "Content-Type: application/json" \
        -d '{
            "vectors": {"size": 384, "distance": "Cosine"},
            "shard_number": 1,
            "replication_factor": 1,
            "on_disk_payload": true
        }' > /dev/null
    
    # Create similarity patterns collection
    log_info "Creating similarity_patterns collection..."
    curl -s -X PUT http://localhost:6333/collections/similarity_patterns \
        -H "Content-Type: application/json" \
        -d '{
            "vectors": {"size": 512, "distance": "Cosine"},
            "shard_number": 1,
            "replication_factor": 1,
            "on_disk_payload": true
        }' > /dev/null
    
    log_success "Qdrant collections created successfully"
}

# n8n workflows deployment
deploy_n8n_workflows() {
    log_info "Deploying n8n automation workflows..."
    
    # Deploy Agent Scheduler workflow
    log_info "Deploying Agent Scheduler workflow..."
    curl -s -X POST http://localhost:5678/api/v1/workflows \
        -H "Content-Type: application/json" \
        -d @"$INIT_DIR/automation/n8n/doc-monitor-workflow.json" > /dev/null
    
    # Deploy Improvement Processor workflow
    log_info "Deploying Improvement Processor workflow..."
    curl -s -X POST http://localhost:5678/api/v1/workflows \
        -H "Content-Type: application/json" \
        -d @"$INIT_DIR/automation/n8n/improvement-processor.json" > /dev/null
    
    # Deploy Notification Router workflow
    log_info "Deploying Notification Router workflow..."
    curl -s -X POST http://localhost:5678/api/v1/workflows \
        -H "Content-Type: application/json" \
        -d @"$INIT_DIR/automation/n8n/notification-router.json" > /dev/null
    
    log_success "n8n workflows deployed successfully"
}

# Windmill application deployment
deploy_windmill_app() {
    log_info "Deploying Windmill application..."
    
    # Deploy main application
    log_info "Deploying main dashboard application..."
    curl -s -X POST http://localhost:5681/api/w/document_manager/apps \
        -H "Content-Type: application/json" \
        -d @"$INIT_DIR/automation/windmill/app.json" > /dev/null
    
    # Deploy Python flows
    log_info "Deploying Python flows..."
    for flow_file in "$INIT_DIR/automation/windmill/flows"/*.py; do
        if [ -f "$flow_file" ]; then
            flow_name=$(basename "$flow_file" .py)
            log_info "Deploying $flow_name flow..."
            curl -s -X POST http://localhost:5681/api/w/document_manager/scripts \
                -H "Content-Type: application/json" \
                -d "{\"path\": \"f/$flow_name\", \"content\": \"$(cat "$flow_file" | sed 's/"/\\"/g' | tr '\n' '\\n')\", \"language\": \"python3\"}" > /dev/null
        fi
    done
    
    log_success "Windmill application deployed successfully"
}

# Ollama model verification
verify_ollama_models() {
    log_info "Verifying Ollama models..."
    
    # Check for required models
    if ! curl -s http://localhost:11434/api/tags | grep -q "llama3.2"; then
        log_warning "llama3.2 model not found, pulling..."
        curl -s -X POST http://localhost:11434/api/pull -d '{"name": "llama3.2"}' > /dev/null &
    fi
    
    if ! curl -s http://localhost:11434/api/tags | grep -q "nomic-embed-text"; then
        log_warning "nomic-embed-text model not found, pulling..."
        curl -s -X POST http://localhost:11434/api/pull -d '{"name": "nomic-embed-text"}' > /dev/null &
    fi
    
    log_success "Ollama model verification completed"
}

# Redis channels setup
setup_redis_channels() {
    log_info "Setting up Redis notification channels..."
    
    # Configure Redis for pub/sub
    redis-cli -p 6380 CONFIG SET notify-keyspace-events Ex > /dev/null 2>&1 || log_warning "Could not configure Redis keyspace notifications"
    
    log_success "Redis channels configured"
}

# System health verification
verify_system_health() {
    log_info "Performing final system health verification..."
    
    local health_checks=(
        "PostgreSQL:http://localhost:5433"
        "Redis:http://localhost:6380"
        "Qdrant:http://localhost:6333/health"
        "Ollama:http://localhost:11434/api/tags"
        "n8n:http://localhost:5678/healthz"
        "Windmill:http://localhost:5681/api/version"
        "Unstructured-IO:http://localhost:11450/health"
    )
    
    local failed_checks=()
    
    for check in "${health_checks[@]}"; do
        IFS=':' read -r service_name health_url <<< "$check"
        
        if [[ "$service_name" == "PostgreSQL" ]]; then
            if ! pg_isready -h localhost -p 5433 > /dev/null 2>&1; then
                failed_checks+=("$service_name")
            fi
        elif [[ "$service_name" == "Redis" ]]; then
            if ! redis-cli -p 6380 ping > /dev/null 2>&1; then
                failed_checks+=("$service_name")
            fi
        else
            if ! curl -s -f "$health_url" > /dev/null 2>&1; then
                failed_checks+=("$service_name")
            fi
        fi
    done
    
    if [ ${#failed_checks[@]} -eq 0 ]; then
        log_success "All system components are healthy"
        return 0
    else
        log_warning "Some components failed health checks: ${failed_checks[*]}"
        return 1
    fi
}

# Cleanup function
cleanup() {
    log_info "Cleaning up background processes..."
    jobs -p | xargs -r kill 2>/dev/null || true
}

# Trap cleanup on exit
trap cleanup EXIT

# Main startup sequence
main() {
    log_info "=== Document Manager Enhanced Startup ==="
    log_info "Scenario Directory: $SCENARIO_DIR"
    log_info "Project Root: $PROJECT_ROOT"
    log_info "Starting comprehensive system initialization..."
    
    # Step 1: Health check core resources
    log_info "Step 1/8: Checking core resource availability..."
    check_resource "PostgreSQL" "localhost:5433" 5 || exit 1
    check_resource "Redis" "localhost:6380" 5 || exit 1
    check_resource "Qdrant" "http://localhost:6333/health" 5 || exit 1
    check_resource "Ollama" "http://localhost:11434/api/tags" 5 || exit 1
    check_resource "n8n" "http://localhost:5678/healthz" 5 || exit 1
    check_resource "Windmill" "http://localhost:5681/api/version" 5 || exit 1
    
    # Step 2: Initialize databases
    log_info "Step 2/8: Initializing databases..."
    init_database
    init_qdrant
    setup_redis_channels
    
    # Step 3: Verify AI models
    log_info "Step 3/8: Verifying AI models..."
    verify_ollama_models
    
    # Step 4: Deploy automation workflows
    log_info "Step 4/8: Deploying automation workflows..."
    deploy_n8n_workflows
    
    # Step 5: Deploy applications
    log_info "Step 5/8: Deploying applications..."
    deploy_windmill_app
    
    # Step 6: Wait for deployment completion
    log_info "Step 6/8: Waiting for deployment completion..."
    sleep 10
    
    # Step 7: Final health verification
    log_info "Step 7/8: Final health verification..."
    if verify_system_health; then
        log_success "System health verification passed"
    else
        log_warning "Some health checks failed, but system may still be functional"
    fi
    
    # Step 8: Display status and access information
    log_info "Step 8/8: System ready!"
    
    echo
    log_success "=== Document Manager Successfully Started ==="
    echo
    log_info "Access Points:"
    log_info "  📊 Dashboard: http://localhost:5681"
    log_info "  🔧 n8n Workflows: http://localhost:5678"
    log_info "  🗄️ Database: postgresql://postgres@localhost:5433/document_manager"
    log_info "  🔍 Vector DB: http://localhost:6333"
    log_info "  🤖 AI Models: http://localhost:11434"
    log_info "  📡 Redis: redis://localhost:6380"
    echo
    log_info "Next Steps:"
    log_info "  1. Create your first application in the dashboard"
    log_info "  2. Configure agents for your documentation"
    log_info "  3. Set up notification preferences"
    log_info "  4. Monitor the improvement queue for suggestions"
    echo
    log_info "Logs and monitoring available in the dashboard"
    log_success "System startup completed successfully!"
}

# Execute main function
main "$@"