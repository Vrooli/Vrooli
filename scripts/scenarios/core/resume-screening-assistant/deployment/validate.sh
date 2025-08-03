#!/bin/bash
# Validation script for Resume Screening Assistant
# Performs pre and post deployment validation checks

set -euo pipefail

# Configuration
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCENARIO_ID="resume-screening-assistant"
VALIDATION_LOG="/tmp/vrooli-${SCENARIO_ID}-validation.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Validation results
VALIDATION_PASSED=0
VALIDATION_FAILED=0
VALIDATION_WARNINGS=0

# Logging functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$VALIDATION_LOG"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$VALIDATION_LOG"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1" | tee -a "$VALIDATION_LOG"
    VALIDATION_PASSED=$((VALIDATION_PASSED + 1))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "$VALIDATION_LOG"
    VALIDATION_WARNINGS=$((VALIDATION_WARNINGS + 1))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1" | tee -a "$VALIDATION_LOG"
    VALIDATION_FAILED=$((VALIDATION_FAILED + 1))
}

# Check if a port is available
check_port() {
    local port=$1
    local service=$2
    
    if nc -z localhost "$port" 2>/dev/null; then
        log_success "Port $port is open for $service"
        return 0
    else
        log_error "Port $port is not accessible for $service"
        return 1
    fi
}

# Check if a URL responds
check_url() {
    local url=$1
    local description=$2
    local expected_status=${3:-200}
    
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    
    if [[ "$response_code" == "$expected_status" ]]; then
        log_success "$description responded with status $response_code"
        return 0
    else
        log_error "$description failed (status: $response_code, expected: $expected_status)"
        return 1
    fi
}

# Check disk space
check_disk_space() {
    local required_gb=$1
    local available_kb
    available_kb=$(df "$SCENARIO_DIR" | awk 'NR==2 {print $4}')
    local available_gb=$((available_kb / 1024 / 1024))
    
    if [[ $available_gb -ge $required_gb ]]; then
        log_success "Sufficient disk space available (${available_gb}GB >= ${required_gb}GB)"
        return 0
    else
        log_error "Insufficient disk space (${available_gb}GB < ${required_gb}GB)"
        return 1
    fi
}

# Check if required files exist
check_required_files() {
    log_info "Checking required files..."
    
    local required_files=(
        "$SCENARIO_DIR/metadata.yaml"
        "$SCENARIO_DIR/manifest.yaml"
        "$SCENARIO_DIR/test.sh"
        "$SCENARIO_DIR/deployment/startup.sh"
        "$SCENARIO_DIR/deployment/validate.sh"
        "$SCENARIO_DIR/deployment/monitor.sh"
    )
    
    for file in "${required_files[@]}"; do
        if [[ -f "$file" ]]; then
            log_success "Found: $(basename "$file")"
        else
            log_error "Missing: $file"
        fi
    done
}

# Check if required directories exist
check_required_directories() {
    log_info "Checking required directories..."
    
    local required_dirs=(
        "$SCENARIO_DIR/initialization"
        "$SCENARIO_DIR/initialization/configuration"
        "$SCENARIO_DIR/initialization/workflows"
        "$SCENARIO_DIR/deployment"
    )
    
    for dir in "${required_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            log_success "Found directory: $(basename "$dir")"
        else
            log_warning "Missing directory: $dir"
        fi
    done
}

# Validate resource availability
validate_resources() {
    log_info "Validating resource availability..."
    
    # Check Unstructured.io
    if check_url "http://localhost:8000/healthcheck" "Unstructured.io"; then
        # Test document processing capability
        if [[ -f "/tmp/test-resume.txt" ]]; then
            echo "Test Resume Content" > /tmp/test-resume.txt
            
            local test_response
            test_response=$(curl -s -X POST "http://localhost:8000/general/v0/general" \
                -F "files=@/tmp/test-resume.txt" \
                -F "strategy=fast" 2>/dev/null || echo "")
            
            if [[ -n "$test_response" ]]; then
                log_success "Unstructured.io can process documents"
            else
                log_warning "Unstructured.io processing test failed"
            fi
            
            rm -f /tmp/test-resume.txt
        fi
    fi
    
    # Check Ollama
    if check_url "http://localhost:11434/api/tags" "Ollama"; then
        # Check for available models
        local models_response
        models_response=$(curl -s "http://localhost:11434/api/tags" 2>/dev/null || echo '{"models":[]}')
        
        if echo "$models_response" | grep -q "models"; then
            local model_count
            model_count=$(echo "$models_response" | jq '.models | length' 2>/dev/null || echo "0")
            
            if [[ $model_count -gt 0 ]]; then
                log_success "Ollama has $model_count model(s) available"
            else
                log_warning "Ollama has no models available"
            fi
        fi
    fi
    
    # Check Qdrant
    if check_url "http://localhost:6333/collections" "Qdrant"; then
        # Check for collections
        local collections_response
        collections_response=$(curl -s "http://localhost:6333/collections" 2>/dev/null || echo '{"result":{"collections":[]}}')
        
        if echo "$collections_response" | grep -q "collections"; then
            log_success "Qdrant is operational"
        fi
    fi
}

# Validate Qdrant collections
validate_vector_collections() {
    log_info "Validating vector collections..."
    
    local qdrant_url="http://localhost:6333"
    local required_collections=("candidate_profiles" "assessment_templates" "job_requirements")
    
    for collection in "${required_collections[@]}"; do
        local collection_response
        collection_response=$(curl -s "$qdrant_url/collections/$collection" 2>/dev/null || echo '{"status":"error"}')
        
        if echo "$collection_response" | grep -q "vectors_count"; then
            log_success "Collection '$collection' exists"
        else
            log_warning "Collection '$collection' not found"
        fi
    done
}

# Validate configuration files
validate_configuration() {
    log_info "Validating configuration files..."
    
    local config_dir="$SCENARIO_DIR/initialization/configuration"
    
    if [[ -d "$config_dir" ]]; then
        # Check for JSON validity
        for config_file in "$config_dir"/*.json; do
            if [[ -f "$config_file" ]]; then
                if jq . "$config_file" > /dev/null 2>&1; then
                    log_success "Valid JSON: $(basename "$config_file")"
                else
                    log_error "Invalid JSON: $(basename "$config_file")"
                fi
            fi
        done
    else
        log_warning "Configuration directory not found"
    fi
}

# Validate workflows
validate_workflows() {
    log_info "Validating workflow files..."
    
    local workflow_dir="$SCENARIO_DIR/initialization/workflows"
    
    if [[ -d "$workflow_dir" ]]; then
        # Check N8N workflows
        if [[ -d "$workflow_dir/n8n" ]]; then
            for workflow in "$workflow_dir/n8n"/*.json; do
                if [[ -f "$workflow" ]]; then
                    if jq '.nodes' "$workflow" > /dev/null 2>&1; then
                        log_success "Valid N8N workflow: $(basename "$workflow")"
                    else
                        log_error "Invalid N8N workflow: $(basename "$workflow")"
                    fi
                fi
            done
        fi
        
        # Check Windmill scripts
        if [[ -d "$workflow_dir/windmill/scripts" ]]; then
            for script in "$workflow_dir/windmill/scripts"/*.py; do
                if [[ -f "$script" ]]; then
                    if python3 -m py_compile "$script" 2>/dev/null; then
                        log_success "Valid Python script: $(basename "$script")"
                    else
                        log_error "Invalid Python script: $(basename "$script")"
                    fi
                fi
            done
        fi
    else
        log_warning "Workflow directory not found"
    fi
}

# Performance validation
validate_performance() {
    log_info "Validating performance requirements..."
    
    # Test resume processing endpoint (if available)
    local api_url="http://localhost:3000/api/resume-screening-assistant"
    
    if curl -s -f "$api_url/health" > /dev/null 2>&1; then
        log_info "Testing response times..."
        
        # Measure health check response time
        local start_time=$(date +%s%N)
        curl -s "$api_url/health" > /dev/null 2>&1
        local end_time=$(date +%s%N)
        local response_time=$(((end_time - start_time) / 1000000))
        
        if [[ $response_time -lt 1000 ]]; then
            log_success "Health check response time: ${response_time}ms (< 1000ms)"
        else
            log_warning "Health check response time: ${response_time}ms (>= 1000ms)"
        fi
    else
        log_warning "API endpoint not available for performance testing"
    fi
}

# Pre-deployment validation
pre_deployment_validation() {
    log "========================================="
    log "Pre-Deployment Validation"
    log "========================================="
    
    # Check system requirements
    check_disk_space 2
    
    # Check required files and directories
    check_required_files
    check_required_directories
    
    # Check ports
    check_port 8000 "Unstructured.io"
    check_port 11434 "Ollama"
    check_port 6333 "Qdrant"
    
    # Validate configurations
    validate_configuration
    validate_workflows
}

# Post-deployment validation
post_deployment_validation() {
    log "========================================="
    log "Post-Deployment Validation"
    log "========================================="
    
    # Validate resources are running
    validate_resources
    
    # Validate vector collections
    validate_vector_collections
    
    # Validate performance
    validate_performance
    
    # Check application endpoints
    check_url "http://localhost:3000/api/resume-screening-assistant/health" "Application health"
}

# Generate validation report
generate_report() {
    log "========================================="
    log "Validation Report"
    log "========================================="
    
    local total_checks=$((VALIDATION_PASSED + VALIDATION_FAILED + VALIDATION_WARNINGS))
    local success_rate=0
    
    if [[ $total_checks -gt 0 ]]; then
        success_rate=$(( (VALIDATION_PASSED * 100) / total_checks ))
    fi
    
    log "Total Checks: $total_checks"
    log "Passed: $VALIDATION_PASSED"
    log "Failed: $VALIDATION_FAILED"
    log "Warnings: $VALIDATION_WARNINGS"
    log "Success Rate: ${success_rate}%"
    
    if [[ $VALIDATION_FAILED -eq 0 ]]; then
        log_success "Validation PASSED - Scenario is ready for deployment"
        return 0
    elif [[ $success_rate -ge 70 ]]; then
        log_warning "Validation PASSED WITH WARNINGS - Review warnings before production use"
        return 0
    else
        log_error "Validation FAILED - Critical issues must be resolved"
        return 1
    fi
}

# Main execution
main() {
    local mode="${1:-all}"
    
    log "Starting validation for $SCENARIO_ID"
    log "Mode: $mode"
    log "Validation log: $VALIDATION_LOG"
    
    case "$mode" in
        pre|pre-deploy)
            pre_deployment_validation
            ;;
        post|post-deploy)
            post_deployment_validation
            ;;
        all|*)
            pre_deployment_validation
            post_deployment_validation
            ;;
    esac
    
    generate_report
}

# Run main function
main "$@"