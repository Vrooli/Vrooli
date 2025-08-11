#!/bin/bash
# Ollama model setup for Smart File Photo Manager
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VROOLI_ROOT="$(cd "$SCENARIO_ROOT/../../.." && pwd)"

# Load environment variables
source "$VROOLI_ROOT/scripts/resources/lib/resource-helper.sh"

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "$VROOLI_ROOT/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Configuration
OLLAMA_HOST="localhost"
OLLAMA_PORT="11434"

# Get default ollama port if available
if command -v "resources::get_default_port" &> /dev/null; then
    OLLAMA_PORT=$(resources::get_default_port "ollama" || echo "11434")
fi

OLLAMA_URL="http://$OLLAMA_HOST:$OLLAMA_PORT"

# Required models
declare -A MODELS=(
    ["llava:13b"]="Vision model for image analysis and description"
    ["llama3.2"]="General language model for reasoning and suggestions"
    ["nomic-embed-text"]="Text embedding model for semantic search"
)

log_info() {
    echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

log_success() {
    echo "[SUCCESS] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Wait for Ollama to be ready
wait_for_ollama() {
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for Ollama on port $OLLAMA_PORT..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$OLLAMA_URL/api/version" >/dev/null 2>&1; then
            log_success "Ollama is ready"
            return 0
        fi
        
        sleep 2
        ((attempt++))
    done
    
    log_error "Ollama failed to become ready"
    return 1
}

# Check if model exists locally
model_exists() {
    local model_name="$1"
    
    if curl -s "$OLLAMA_URL/api/tags" | grep -q "\"name\":\"$model_name\""; then
        return 0
    else
        return 1
    fi
}

# Pull a model
pull_model() {
    local model_name="$1"
    local description="$2"
    
    log_info "Checking model: $model_name"
    
    if model_exists "$model_name"; then
        log_info "Model '$model_name' already exists locally"
        return 0
    fi
    
    log_info "Pulling model: $model_name ($description)"
    log_info "This may take several minutes depending on model size..."
    
    # Pull model with progress tracking
    local pull_payload="{\"name\":\"$model_name\"}"
    
    # Start pull request and capture process
    local temp_output="/tmp/ollama_pull_${model_name//[^a-zA-Z0-9]/_}.log"
    
    curl -s -X POST "$OLLAMA_URL/api/pull" \
        -H "Content-Type: application/json" \
        -d "$pull_payload" > "$temp_output" &
    
    local pull_pid=$!
    
    # Monitor progress
    local last_status=""
    while kill -0 $pull_pid 2>/dev/null; do
        if [ -f "$temp_output" ]; then
            local current_line
            current_line=$(tail -n 1 "$temp_output" 2>/dev/null | grep -o '"status":"[^"]*"' | cut -d'"' -f4 || echo "")
            
            if [ -n "$current_line" ] && [ "$current_line" != "$last_status" ]; then
                log_info "Status: $current_line"
                last_status="$current_line"
            fi
        fi
        sleep 5
    done
    
    # Wait for process to complete
    wait $pull_pid
    local exit_code=$?
    
    # Clean up temp file
    trash::safe_remove "$temp_output" --temp
    
    if [ $exit_code -eq 0 ] && model_exists "$model_name"; then
        log_success "Successfully pulled model: $model_name"
    else
        log_error "Failed to pull model: $model_name"
        return 1
    fi
}

# Test model functionality
test_model() {
    local model_name="$1"
    local test_type="$2"
    
    log_info "Testing model: $model_name"
    
    case "$test_type" in
        "vision")
            # Test vision model with simple prompt
            local test_payload='{
                "model": "'$model_name'",
                "prompt": "What do you see in this image?",
                "stream": false
            }'
            ;;
        "text")
            # Test text model
            local test_payload='{
                "model": "'$model_name'",
                "prompt": "Hello, can you help me organize files?",
                "stream": false
            }'
            ;;
        "embedding")
            # Test embedding model
            local test_payload='{
                "model": "'$model_name'",
                "prompt": "test document content"
            }'
            ;;
    esac
    
    if [ "$test_type" = "embedding" ]; then
        # Test embeddings endpoint
        if curl -s -X POST "$OLLAMA_URL/api/embeddings" \
            -H "Content-Type: application/json" \
            -d "$test_payload" | grep -q "embedding"; then
            log_success "Model test passed: $model_name"
        else
            log_error "Model test failed: $model_name"
            return 1
        fi
    else
        # Test generate endpoint
        if curl -s -X POST "$OLLAMA_URL/api/generate" \
            -H "Content-Type: application/json" \
            -d "$test_payload" | grep -q "response"; then
            log_success "Model test passed: $model_name"
        else
            log_error "Model test failed: $model_name"
            return 1
        fi
    fi
}

# Get model information
get_model_info() {
    local model_name="$1"
    
    log_info "Getting information for model: $model_name"
    
    local model_info
    model_info=$(curl -s "$OLLAMA_URL/api/show" \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"$model_name\"}")
    
    if echo "$model_info" | grep -q "modelfile"; then
        local size
        size=$(echo "$model_info" | grep -o '"size":[0-9]*' | cut -d':' -f2 || echo "unknown")
        log_info "Model size: $(numfmt --to=iec $size 2>/dev/null || echo "$size bytes")"
    fi
}

# Setup all models
setup_models() {
    log_info "Setting up Ollama models for File Manager..."
    
    for model in "${!MODELS[@]}"; do
        pull_model "$model" "${MODELS[$model]}"
        
        # Get model info
        get_model_info "$model"
        
        # Test model based on type
        case "$model" in
            "llava"*)
                test_model "$model" "vision"
                ;;
            "nomic-embed"*)
                test_model "$model" "embedding"
                ;;
            *)
                test_model "$model" "text"
                ;;
        esac
    done
    
    log_success "All models setup completed successfully"
}

# Verify all models
verify_models() {
    log_info "Verifying all models are available..."
    
    local all_good=true
    
    for model in "${!MODELS[@]}"; do
        if model_exists "$model"; then
            log_success "Model verified: $model"
        else
            log_error "Model missing: $model"
            all_good=false
        fi
    done
    
    if [ "$all_good" = true ]; then
        log_success "All models verified successfully"
    else
        log_error "Some models are missing"
        return 1
    fi
}

# Clean up old models (optional)
cleanup_old_models() {
    log_info "Checking for old model versions..."
    
    # List all models and identify old versions
    local model_list
    model_list=$(curl -s "$OLLAMA_URL/api/tags" | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
    
    # This could be expanded to remove specific old versions
    # For now, just report what's available
    log_info "Available models:"
    echo "$model_list" | while read -r model; do
        [ -n "$model" ] && log_info "  - $model"
    done
}

# Main execution
main() {
    log_info "Setting up Ollama models for File Manager..."
    
    # Check if ollama command is available
    if ! command -v ollama >/dev/null 2>&1; then
        log_info "Ollama CLI not found, using API directly"
    fi
    
    # Wait for Ollama to be ready
    wait_for_ollama
    
    # Setup all required models
    setup_models
    
    # Verify all models
    verify_models
    
    # Cleanup if requested
    if [ "${CLEANUP_OLD_MODELS:-false}" = "true" ]; then
        cleanup_old_models
    fi
    
    log_success "Ollama model setup completed successfully"
    log_info "Models are ready for AI-powered file processing"
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi