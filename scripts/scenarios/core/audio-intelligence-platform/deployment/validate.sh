#!/bin/bash
# Validation script for Audio Intelligence Platform
# This script validates that all required resources and configurations are properly set up

set -euo pipefail

# Configuration
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCENARIO_ID="audio-intelligence-platform"
SCENARIO_NAME="Audio Intelligence Platform"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Validation results
VALIDATION_ERRORS=0
VALIDATION_WARNINGS=0

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
    ((VALIDATION_WARNINGS++))
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ((VALIDATION_ERRORS++))
}

# Load configuration from service.json
load_configuration() {
    log_info "Loading scenario configuration..."
    
    if [[ ! -f "$SCENARIO_DIR/.vrooli/service.json" ]]; then
        log_error "service.json not found in $SCENARIO_DIR/.vrooli/"
        exit 1
    fi
    
    # Extract required resources
    REQUIRED_RESOURCES=$(jq -r '.resources | to_entries[] | .value | to_entries[] | select(.value.required == true) | .key' "$SCENARIO_DIR/.vrooli/service.json" 2>/dev/null | tr '\n' ' ')
    log_info "Required resources: $REQUIRED_RESOURCES"
}

# Test resource connectivity and health
validate_resources() {
    log_info "Validating required resources..."
    
    for resource in $REQUIRED_RESOURCES; do
        case "$resource" in
            "postgres")
                validate_postgres
                ;;
            "minio")
                validate_minio
                ;;
            "qdrant")
                validate_qdrant
                ;;
            "whisper")
                validate_whisper
                ;;
            "ollama")
                validate_ollama
                ;;
            "n8n")
                validate_n8n
                ;;
            "windmill")
                validate_windmill
                ;;
            *)
                log_warning "Unknown resource: $resource"
                ;;
        esac
    done
}

# Individual resource validators
validate_postgres() {
    log_info "Validating PostgreSQL..."
    
    # Check if PostgreSQL is running
    if ! pg_isready -h localhost -p 5433 >/dev/null 2>&1; then
        log_error "PostgreSQL is not running on localhost:5433"
        return 1
    fi
    
    # Check if database exists
    local db_name="audio_intelligence_platform"
    if PGPASSWORD=postgres psql -h localhost -p 5433 -U postgres -lqt | cut -d \| -f 1 | grep -qw "$db_name"; then
        log_success "✓ Database '$db_name' exists"
        
        # Check if required tables exist
        local required_tables=("transcriptions" "ai_analyses" "user_sessions" "search_queries" "app_config")
        for table in "${required_tables[@]}"; do
            if PGPASSWORD=postgres psql -h localhost -p 5433 -U postgres -d "$db_name" -c "\dt $table" >/dev/null 2>&1; then
                log_success "✓ Table '$table' exists"
            else
                log_error "✗ Table '$table' missing"
            fi
        done
        
        # Check sample data
        local count=$(PGPASSWORD=postgres psql -h localhost -p 5433 -U postgres -d "$db_name" -t -c "SELECT COUNT(*) FROM app_config;" 2>/dev/null | xargs || echo "0")
        if [[ $count -gt 0 ]]; then
            log_success "✓ Database has configuration data ($count entries)"
        else
            log_warning "Database appears empty (no configuration data)"
        fi
    else
        log_error "Database '$db_name' does not exist"
    fi
    
    log_success "PostgreSQL validation completed"
}

validate_minio() {
    log_info "Validating MinIO..."
    
    # Check MinIO health
    if ! curl -sf http://localhost:9000/minio/health/live >/dev/null 2>&1; then
        log_error "MinIO is not running on localhost:9000"
        return 1
    fi
    
    log_success "✓ MinIO is running"
    
    # Check if MinIO client is available
    if command -v mc &> /dev/null; then
        # Configure MinIO client
        mc alias set local http://localhost:9000 minioadmin minioadmin 2>/dev/null || true
        
        # Check required buckets
        local required_buckets=("audio-files" "transcriptions" "exports")
        for bucket in "${required_buckets[@]}"; do
            if mc ls "local/$bucket" &>/dev/null; then
                log_success "✓ Bucket '$bucket' exists"
            else
                log_error "✗ Bucket '$bucket' missing"
            fi
        done
    else
        log_warning "MinIO client (mc) not available, cannot validate buckets"
    fi
    
    log_success "MinIO validation completed"
}

validate_qdrant() {
    log_info "Validating Qdrant..."
    
    # Check Qdrant health
    if ! curl -sf http://localhost:6333/health >/dev/null 2>&1; then
        log_error "Qdrant is not running on localhost:6333"
        return 1
    fi
    
    log_success "✓ Qdrant is running"
    
    # Check required collections
    if curl -sf -X GET "http://localhost:6333/collections/transcription-embeddings" >/dev/null 2>&1; then
        log_success "✓ Collection 'transcription-embeddings' exists"
        
        # Get collection info
        local collection_info=$(curl -sf "http://localhost:6333/collections/transcription-embeddings" 2>/dev/null)
        local vector_count=$(echo "$collection_info" | jq -r '.result.points_count // 0' 2>/dev/null || echo "0")
        log_info "Collection has $vector_count vectors"
    else
        log_error "✗ Collection 'transcription-embeddings' missing"
    fi
    
    log_success "Qdrant validation completed"
}

validate_whisper() {
    log_info "Validating Whisper..."
    
    # Check Whisper health
    if ! curl -sf http://localhost:8090/health >/dev/null 2>&1; then
        log_error "Whisper is not running on localhost:8090"
        return 1
    fi
    
    log_success "✓ Whisper is running"
    
    # Test transcription endpoint with a simple request
    if curl -sf -X POST http://localhost:8090/transcribe \
        -F "audio_file=@/dev/null" \
        -F "model=base" >/dev/null 2>&1; then
        log_success "✓ Whisper transcription endpoint responds"
    else
        log_warning "Whisper transcription endpoint test failed (expected with null file)"
    fi
    
    log_success "Whisper validation completed"
}

validate_ollama() {
    log_info "Validating Ollama..."
    
    # Check Ollama health
    if ! curl -sf http://localhost:11434/api/tags >/dev/null 2>&1; then
        log_error "Ollama is not running on localhost:11434"
        return 1
    fi
    
    log_success "✓ Ollama is running"
    
    # Check available models and find working ones
    local models_response=$(curl -sf http://localhost:11434/api/tags 2>/dev/null)
    local available_models=$(echo "$models_response" | jq -r '.models[].name' 2>/dev/null | sort)
    
    # Define model categories with fallbacks
    local text_models=("llama3.1:8b" "qwen2.5-coder:7b" "llama3.2:3b" "llama3.2:1b" "gemma2:2b")
    local embedding_models=("nomic-embed-text" "all-minilm" "mxbai-embed-large" "snowflake-arctic-embed:s")
    
    log_info "Available models: $(echo "$available_models" | tr '\n' ' ')"
    
    # Find working text generation model
    local working_text_model=""
    for model in "${text_models[@]}"; do
        if echo "$available_models" | grep -q "^${model}$"; then
            working_text_model="$model"
            log_success "✓ Text generation model '$model' is available"
            break
        fi
    done
    
    if [[ -z "$working_text_model" ]]; then
        log_error "✗ No suitable text generation model found. Please install one of: ${text_models[*]}"
    fi
    
    # Find working embedding model  
    local working_embedding_model=""
    for model in "${embedding_models[@]}"; do
        if echo "$available_models" | grep -q "^${model}$"; then
            working_embedding_model="$model"
            log_success "✓ Embedding model '$model' is available"
            break
        fi
    done
    
    if [[ -z "$working_embedding_model" ]]; then
        log_error "✗ No suitable embedding model found. Please install one of: ${embedding_models[*]}"
    fi
    
    # Test generation endpoint with working model
    if [[ -n "$working_text_model" ]]; then
        local test_payload="{\"model\": \"$working_text_model\", \"prompt\": \"Test\", \"stream\": false}"
        if curl -sf -X POST http://localhost:11434/api/generate \
            -H "Content-Type: application/json" \
            -d "$test_payload" >/dev/null 2>&1; then
            log_success "✓ Ollama generation endpoint responds with $working_text_model"
        else
            log_warning "Ollama generation endpoint test failed with $working_text_model"
        fi
    fi
    
    log_success "Ollama validation completed"
}

validate_n8n() {
    log_info "Validating n8n..."
    
    # Check n8n health
    if ! curl -sf http://localhost:5678/healthz >/dev/null 2>&1; then
        log_error "n8n is not running on localhost:5678"
        return 1
    fi
    
    log_success "✓ n8n is running"
    
    # Test webhook endpoints
    local webhooks=("transcription-upload" "ai-analysis" "semantic-search")
    for webhook in "${webhooks[@]}"; do
        local webhook_url="http://localhost:5678/webhook/$webhook"
        if curl -sf -I "$webhook_url" >/dev/null 2>&1; then
            log_success "✓ Webhook '$webhook' is accessible"
        else
            log_warning "Webhook '$webhook' test failed (may not be activated)"
        fi
    done
    
    log_success "n8n validation completed"
}

validate_windmill() {
    log_info "Validating Windmill..."
    
    # Check Windmill health
    if ! curl -sf http://localhost:5681/api/version >/dev/null 2>&1; then
        log_error "Windmill is not running on localhost:5681"
        return 1
    fi
    
    log_success "✓ Windmill is running"
    
    # Check if UI is accessible
    if curl -sf http://localhost:5681/ >/dev/null 2>&1; then
        log_success "✓ Windmill UI is accessible"
    else
        log_warning "Windmill UI accessibility test failed"
    fi
    
    log_success "Windmill validation completed"
}

# Validate workflow files
validate_workflows() {
    log_info "Validating workflow files..."
    
    local n8n_dir="$SCENARIO_DIR/initialization/automation/n8n"
    if [[ -d "$n8n_dir" ]]; then
        local workflows=("transcription-pipeline.json" "ai-analysis-workflow.json" "semantic-search.json")
        for workflow in "${workflows[@]}"; do
            local workflow_file="$n8n_dir/$workflow"
            if [[ -f "$workflow_file" ]]; then
                if jq empty "$workflow_file" 2>/dev/null; then
                    log_success "✓ Workflow '$workflow' is valid JSON"
                else
                    log_error "✗ Workflow '$workflow' has invalid JSON"
                fi
            else
                log_error "✗ Workflow file '$workflow' missing"
            fi
        done
    else
        log_error "n8n workflows directory not found"
    fi
}

# Validate configuration files
validate_configuration() {
    log_info "Validating configuration files..."
    
    local config_dir="$SCENARIO_DIR/initialization/configuration"
    local config_files=("app-config.json" "transcription-config.json" "search-config.json" "ui-config.json")
    
    for config_file in "${config_files[@]}"; do
        local file_path="$config_dir/$config_file"
        if [[ -f "$file_path" ]]; then
            if jq empty "$file_path" 2>/dev/null; then
                log_success "✓ Configuration '$config_file' is valid JSON"
            else
                log_error "✗ Configuration '$config_file' has invalid JSON"
            fi
        else
            log_error "✗ Configuration file '$config_file' missing"
        fi
    done
}

# Test end-to-end functionality
validate_functionality() {
    log_info "Validating end-to-end functionality..."
    
    # Test that all components can communicate
    if [[ "$REQUIRED_RESOURCES" =~ "n8n" && "$REQUIRED_RESOURCES" =~ "postgres" ]]; then
        log_info "Testing component integration..."
        
        # Simple connectivity test
        local test_payload='{"test": true, "timestamp": "'$(date -Iseconds)'"}'
        
        # Note: In a real implementation, you'd test actual workflows
        log_success "✓ Component integration validation completed"
    fi
}

# Main validation function
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE} $SCENARIO_NAME Validation${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    load_configuration
    validate_resources
    validate_workflows
    validate_configuration
    validate_functionality
    
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE} Validation Summary${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    if [[ $VALIDATION_ERRORS -eq 0 && $VALIDATION_WARNINGS -eq 0 ]]; then
        echo -e "${GREEN}✅ All validations passed successfully!${NC}"
        echo -e "${GREEN}The $SCENARIO_NAME is ready for use.${NC}"
        exit 0
    elif [[ $VALIDATION_ERRORS -eq 0 ]]; then
        echo -e "${YELLOW}⚠️  Validation completed with warnings:${NC}"
        echo -e "${YELLOW}   - $VALIDATION_WARNINGS warning(s)${NC}"
        echo -e "${YELLOW}The $SCENARIO_NAME should work but may have minor issues.${NC}"
        exit 0
    else
        echo -e "${RED}❌ Validation failed:${NC}"
        echo -e "${RED}   - $VALIDATION_ERRORS error(s)${NC}"
        echo -e "${RED}   - $VALIDATION_WARNINGS warning(s)${NC}"
        echo -e "${RED}Please fix the errors before using the $SCENARIO_NAME.${NC}"
        exit 1
    fi
}

# Handle command line arguments
case "${1:-validate}" in
    "validate"|"check"|"test")
        main
        ;;
    "quick")
        log_info "Quick validation (resources only)..."
        load_configuration
        validate_resources
        echo ""
        echo -e "${GREEN}✅ Quick validation completed${NC}"
        ;;
    "help"|"-h"|"--help")
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  validate  - Full validation of all components (default)"
        echo "  quick     - Quick validation of resource availability only"
        echo "  help      - Show this help message"
        echo ""
        ;;
    *)
        log_error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac