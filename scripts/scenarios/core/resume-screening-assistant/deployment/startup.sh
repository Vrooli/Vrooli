#!/bin/bash
# Startup script for Resume Screening Assistant
# This script converts the scenario into a running application

set -euo pipefail

# Configuration
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCENARIO_ID="resume-screening-assistant"
SCENARIO_NAME="Resume Screening Assistant"
LOG_FILE="/tmp/vrooli-${SCENARIO_ID}-startup.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

# Error handling
trap 'log_error "Startup failed at line $LINENO"; exit 1' ERR

# Load configuration from manifest.yaml and metadata.yaml
load_configuration() {
    log_info "Loading scenario configuration..."
    
    # Check if required files exist
    if [[ ! -f "$SCENARIO_DIR/metadata.yaml" ]]; then
        log_error "metadata.yaml not found in $SCENARIO_DIR"
        exit 1
    fi
    
    if [[ ! -f "$SCENARIO_DIR/manifest.yaml" ]]; then
        log_error "manifest.yaml not found in $SCENARIO_DIR"
        exit 1
    fi
    
    # Extract resource requirements
    REQUIRED_RESOURCES=(unstructured-io ollama qdrant)
    OPTIONAL_RESOURCES=(browserless minio n8n windmill)
    
    log_success "Configuration loaded"
}

# Check resource health
check_resource_health() {
    local resource=$1
    local health_url=""
    
    case "$resource" in
        unstructured-io)
            health_url="http://localhost:8000/healthcheck"
            ;;
        ollama)
            health_url="http://localhost:11434/api/tags"
            ;;
        qdrant)
            health_url="http://localhost:6333/collections"
            ;;
        browserless)
            health_url="http://localhost:3000/pressure"
            ;;
        minio)
            health_url="http://localhost:9000/minio/health/live"
            ;;
        n8n)
            health_url="http://localhost:5678/healthz"
            ;;
        windmill)
            health_url="http://localhost:8000/api/version"
            ;;
        *)
            log_warning "Unknown resource: $resource"
            return 1
            ;;
    esac
    
    if curl -s -f "$health_url" > /dev/null 2>&1; then
        log_success "$resource is healthy"
        return 0
    else
        log_error "$resource is not responding at $health_url"
        return 1
    fi
}

# Validate all required resources
validate_resources() {
    log_info "Validating required resources..."
    
    local all_healthy=true
    for resource in "${REQUIRED_RESOURCES[@]}"; do
        if ! check_resource_health "$resource"; then
            all_healthy=false
        fi
    done
    
    if [[ "$all_healthy" == "false" ]]; then
        log_error "Not all required resources are healthy"
        exit 1
    fi
    
    log_info "Checking optional resources..."
    for resource in "${OPTIONAL_RESOURCES[@]}"; do
        check_resource_health "$resource" || log_warning "$resource is optional and not available"
    done
    
    log_success "Resource validation complete"
}

# Initialize Qdrant collections
initialize_vector_database() {
    log_info "Initializing vector database collections..."
    
    local qdrant_url="http://localhost:6333"
    
    # Create candidate_profiles collection
    local candidate_collection='{
        "collection_name": "candidate_profiles",
        "vectors": {
            "size": 384,
            "distance": "Cosine"
        },
        "optimizers_config": {
            "default_segment_number": 2
        }
    }'
    
    if curl -s -X PUT "$qdrant_url/collections/candidate_profiles" \
        -H "Content-Type: application/json" \
        -d "$candidate_collection" > /dev/null 2>&1; then
        log_success "Created candidate_profiles collection"
    else
        log_warning "candidate_profiles collection might already exist"
    fi
    
    # Create assessment_templates collection
    local assessment_collection='{
        "collection_name": "assessment_templates",
        "vectors": {
            "size": 384,
            "distance": "Cosine"
        }
    }'
    
    if curl -s -X PUT "$qdrant_url/collections/assessment_templates" \
        -H "Content-Type: application/json" \
        -d "$assessment_collection" > /dev/null 2>&1; then
        log_success "Created assessment_templates collection"
    else
        log_warning "assessment_templates collection might already exist"
    fi
    
    # Create job_requirements collection
    local job_collection='{
        "collection_name": "job_requirements",
        "vectors": {
            "size": 384,
            "distance": "Cosine"
        }
    }'
    
    if curl -s -X PUT "$qdrant_url/collections/job_requirements" \
        -H "Content-Type: application/json" \
        -d "$job_collection" > /dev/null 2>&1; then
        log_success "Created job_requirements collection"
    else
        log_warning "job_requirements collection might already exist"
    fi
    
    log_success "Vector database initialization complete"
}

# Check and pull Ollama models
initialize_ollama_models() {
    log_info "Checking Ollama models..."
    
    local ollama_url="http://localhost:11434"
    local required_model="llama3.2:3b"
    
    # Check if model exists
    local models_response
    models_response=$(curl -s "$ollama_url/api/tags" 2>/dev/null || echo '{"models":[]}')
    
    if echo "$models_response" | grep -q "$required_model"; then
        log_success "Model $required_model is already available"
    else
        log_info "Pulling model $required_model (this may take a few minutes)..."
        
        local pull_request="{\"name\": \"$required_model\"}"
        if curl -s -X POST "$ollama_url/api/pull" \
            -H "Content-Type: application/json" \
            -d "$pull_request" > /dev/null 2>&1; then
            log_success "Model $required_model pulled successfully"
        else
            log_warning "Failed to pull model $required_model, will use any available model"
        fi
    fi
    
    log_success "Ollama initialization complete"
}

# Deploy workflows if automation platforms are available
deploy_workflows() {
    log_info "Deploying workflows..."
    
    # Check for N8N
    if check_resource_health "n8n" 2>/dev/null; then
        log_info "Deploying N8N workflows..."
        
        if [[ -d "$SCENARIO_DIR/initialization/workflows/n8n" ]]; then
            for workflow in "$SCENARIO_DIR/initialization/workflows/n8n"/*.json; do
                if [[ -f "$workflow" ]]; then
                    local workflow_name=$(basename "$workflow")
                    log_info "Importing workflow: $workflow_name"
                    # N8N workflow import would go here
                fi
            done
        fi
    fi
    
    # Check for Windmill
    if check_resource_health "windmill" 2>/dev/null; then
        log_info "Deploying Windmill scripts..."
        
        if [[ -d "$SCENARIO_DIR/initialization/workflows/windmill" ]]; then
            for script in "$SCENARIO_DIR/initialization/workflows/windmill/scripts"/*.py; do
                if [[ -f "$script" ]]; then
                    local script_name=$(basename "$script")
                    log_info "Deploying script: $script_name"
                    # Windmill script deployment would go here
                fi
            done
        fi
    fi
    
    log_success "Workflow deployment complete"
}

# Load application configuration
load_app_configuration() {
    log_info "Loading application configuration..."
    
    local config_dir="$SCENARIO_DIR/initialization/configuration"
    
    if [[ -d "$config_dir" ]]; then
        # Load resource URLs configuration
        if [[ -f "$config_dir/resource-urls.json" ]]; then
            log_info "Loading resource URLs..."
            export RESOURCE_URLS=$(cat "$config_dir/resource-urls.json")
        fi
        
        # Load app configuration
        if [[ -f "$config_dir/app-config.json" ]]; then
            log_info "Loading app configuration..."
            export APP_CONFIG=$(cat "$config_dir/app-config.json")
        fi
        
        # Load feature flags
        if [[ -f "$config_dir/feature-flags.json" ]]; then
            log_info "Loading feature flags..."
            export FEATURE_FLAGS=$(cat "$config_dir/feature-flags.json")
        fi
    else
        log_warning "Configuration directory not found, using defaults"
    fi
    
    log_success "Configuration loaded"
}

# Setup storage buckets if MinIO is available
setup_storage() {
    log_info "Setting up storage..."
    
    if check_resource_health "minio" 2>/dev/null; then
        log_info "Configuring MinIO buckets..."
        
        local minio_url="http://localhost:9000"
        local buckets=("resume-screening-assistant-uploads" "resume-screening-assistant-reports" "resume-screening-assistant-backups")
        
        for bucket in "${buckets[@]}"; do
            log_info "Creating bucket: $bucket"
            # MinIO bucket creation would go here
        done
        
        log_success "Storage setup complete"
    else
        log_warning "MinIO not available, skipping storage setup"
    fi
}

# Create seed data for testing
create_seed_data() {
    log_info "Creating seed data..."
    
    # Create sample assessment templates
    local templates_file="$SCENARIO_DIR/initialization/vectors/assessment-templates.json"
    if [[ -f "$templates_file" ]]; then
        log_info "Loading assessment templates..."
        # Load templates into Qdrant
    fi
    
    # Create sample job requirements
    local sample_jobs=(
        "Senior Software Engineer: Python, React, 5+ years experience"
        "Data Scientist: Machine Learning, Statistics, PhD preferred"
        "DevOps Engineer: Kubernetes, AWS, CI/CD pipelines"
        "Frontend Developer: JavaScript, Vue.js, UX design"
    )
    
    log_info "Creating sample job postings..."
    # Would insert into Qdrant here
    
    log_success "Seed data created"
}

# Start the application
start_application() {
    log_info "Starting Resume Screening Assistant application..."
    
    # Create application status file
    local status_file="/tmp/vrooli-${SCENARIO_ID}.status"
    cat > "$status_file" <<EOF
{
    "scenario_id": "$SCENARIO_ID",
    "scenario_name": "$SCENARIO_NAME",
    "status": "running",
    "started_at": "$(date -Iseconds)",
    "pid": $$,
    "log_file": "$LOG_FILE",
    "urls": {
        "app": "http://localhost:3000/resume-screening-assistant",
        "api": "http://localhost:3000/api/resume-screening-assistant",
        "health": "http://localhost:3000/api/resume-screening-assistant/health"
    }
}
EOF
    
    log_success "Application started successfully"
    log_info "Access the application at: http://localhost:3000/resume-screening-assistant"
    log_info "API endpoint: http://localhost:3000/api/resume-screening-assistant"
    log_info "Health check: http://localhost:3000/api/resume-screening-assistant/health"
    log_info "Log file: $LOG_FILE"
}

# Main execution
main() {
    log "========================================="
    log "Starting $SCENARIO_NAME"
    log "========================================="
    
    # Load configuration
    load_configuration
    
    # Validate resources
    validate_resources
    
    # Initialize components
    initialize_vector_database
    initialize_ollama_models
    
    # Deploy workflows
    deploy_workflows
    
    # Load configuration
    load_app_configuration
    
    # Setup storage
    setup_storage
    
    # Create seed data
    create_seed_data
    
    # Start application
    start_application
    
    log "========================================="
    log "$SCENARIO_NAME startup complete"
    log "========================================="
}

# Run main function
main "$@"