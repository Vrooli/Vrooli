#!/usr/bin/env bash
set -euo pipefail

# Embedding Validation Utilities
# Prevents silent failures in vector operations by validating dimensions and models

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source logging utilities
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/log.sh"

#######################################
# Validate embedding model exists and works
# Arguments:
#   $1 - model name
# Returns:
#   0 if model is valid, 1 otherwise
#######################################
embedding::validate_model() {
    local model="$1"
    
    log::info "Validating model: $model"
    
    # Check if Ollama is running
    if ! curl -s "http://localhost:11434/" >/dev/null 2>&1; then
        log::error "Ollama service not running on port 11434"
        return 1
    fi
    
    # Test model with simple embedding
    local response
    response=$(curl -s -X POST "http://localhost:11434/api/embed" \
               -H "Content-Type: application/json" \
               -d "{\"model\": \"$model\", \"input\": \"test\"}" 2>/dev/null)
    
    if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error')
        log::error "Model validation failed: $error_msg"
        if [[ "$error_msg" == *"not found"* ]]; then
            log::info "Try installing the model: ollama pull $model"
        fi
        return 1
    fi
    
    if echo "$response" | jq -e '.embeddings[0]' >/dev/null 2>&1; then
        local dimensions
        dimensions=$(echo "$response" | jq '.embeddings[0] | length')
        log::success "Model $model validated successfully ($dimensions dimensions)"
        return 0
    else
        log::error "Model $model did not return valid embeddings"
        return 1
    fi
}

#######################################
# Get embedding dimensions for a model
# Arguments:
#   $1 - model name
# Outputs:
#   Dimension count
# Returns:
#   0 on success, 1 on failure
#######################################
embedding::get_model_dimensions() {
    local model="$1"
    
    local response
    response=$(curl -s -X POST "http://localhost:11434/api/embed" \
               -H "Content-Type: application/json" \
               -d "{\"model\": \"$model\", \"input\": \"test\"}" 2>/dev/null)
    
    if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
        return 1
    fi
    
    echo "$response" | jq '.embeddings[0] | length'
}

#######################################
# Get Qdrant collection dimensions
# Arguments:
#   $1 - collection name
# Outputs:
#   Dimension count
# Returns:
#   0 on success, 1 on failure
#######################################
embedding::get_collection_dimensions() {
    local collection="$1"
    
    # Check if Qdrant is running
    if ! curl -s "http://localhost:6333/" >/dev/null 2>&1; then
        log::error "Qdrant service not running on port 6333"
        return 1
    fi
    
    local response
    response=$(curl -s "http://localhost:6333/collections/$collection" 2>/dev/null)
    
    if echo "$response" | jq -e '.result.config.params.vectors.size' >/dev/null 2>&1; then
        echo "$response" | jq '.result.config.params.vectors.size'
    else
        log::error "Collection '$collection' not found or invalid"
        return 1
    fi
}

#######################################
# Validate model-collection compatibility
# Arguments:
#   $1 - model name
#   $2 - collection name
# Returns:
#   0 if compatible, 1 if not
#######################################
embedding::validate_compatibility() {
    local model="$1"
    local collection="$2"
    
    log::info "Validating compatibility: $model → $collection"
    
    # Get model dimensions
    local model_dims
    if ! model_dims=$(embedding::get_model_dimensions "$model"); then
        log::error "Failed to get dimensions for model: $model"
        return 1
    fi
    
    # Get collection dimensions
    local collection_dims
    if ! collection_dims=$(embedding::get_collection_dimensions "$collection"); then
        log::error "Failed to get dimensions for collection: $collection"
        return 1
    fi
    
    # Compare dimensions
    if [[ "$model_dims" != "$collection_dims" ]]; then
        log::error "❌ Dimension mismatch!"
        log::error "   Model '$model' produces $model_dims dimensions"
        log::error "   Collection '$collection' expects $collection_dims dimensions"
        log::info ""
        log::info "💡 Recommended solutions:"
        
        # Provide specific recommendations
        if [[ "$collection_dims" == "768" ]]; then
            log::info "   • Use model: nomic-embed-text:latest (768 dimensions)"
            log::info "   • Install with: ollama pull nomic-embed-text:latest"
        elif [[ "$collection_dims" == "1536" ]]; then
            log::info "   • No local Ollama model available for 1536 dimensions"
            log::info "   • Consider using OpenAI text-embedding-ada-002 via API"
            log::info "   • Or recreate collection with 768 dimensions"
        elif [[ "$model_dims" == "4096" && "$collection_dims" == "768" ]]; then
            log::info "   • Replace '$model' with 'nomic-embed-text:latest'"
            log::info "   • Run: ollama pull nomic-embed-text:latest"
        fi
        
        return 1
    fi
    
    log::success "✅ Compatible! Both use $model_dims dimensions"
    return 0
}

#######################################
# Generate embedding with validation
# Arguments:
#   $1 - model name
#   $2 - input text
#   $3 - target collection (optional, for validation)
# Outputs:
#   JSON response with embedding
# Returns:
#   0 on success, 1 on failure
#######################################
embedding::generate_validated() {
    local model="$1"
    local input="$2"
    local collection="${3:-}"
    
    # Validate model first
    if ! embedding::validate_model "$model"; then
        return 1
    fi
    
    # Validate compatibility if collection specified
    if [[ -n "$collection" ]]; then
        if ! embedding::validate_compatibility "$model" "$collection"; then
            log::error "Aborting embedding generation due to compatibility issues"
            return 1
        fi
    fi
    
    # Generate embedding
    log::info "Generating embedding with model: $model"
    local response
    local json_input
    json_input=$(echo "$input" | jq -R .)
    response=$(curl -s -X POST "http://localhost:11434/api/embed" \
               -H "Content-Type: application/json" \
               -d "{\"model\": \"$model\", \"input\": $json_input}")
    
    # Check for errors
    if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error')
        log::error "Embedding generation failed: $error_msg"
        return 1
    fi
    
    # Validate response has embeddings
    if ! echo "$response" | jq -e '.embeddings[0]' >/dev/null 2>&1; then
        log::error "Response does not contain valid embeddings"
        return 1
    fi
    
    log::success "Embedding generated successfully"
    echo "$response"
}

#######################################
# Insert vector into Qdrant with validation
# Arguments:
#   $1 - collection name
#   $2 - point ID
#   $3 - vector JSON array
#   $4 - payload JSON object
# Returns:
#   0 on success, 1 on failure
#######################################
embedding::insert_validated() {
    local collection="$1"
    local point_id="$2"
    local vector="$3"
    local payload="$4"
    
    log::info "Inserting point $point_id into collection: $collection"
    
    # Validate vector dimensions
    local vector_dims
    vector_dims=$(echo "$vector" | jq 'length')
    
    local collection_dims
    if ! collection_dims=$(embedding::get_collection_dimensions "$collection"); then
        return 1
    fi
    
    if [[ "$vector_dims" != "$collection_dims" ]]; then
        log::error "❌ Cannot insert vector: dimension mismatch"
        log::error "   Vector has $vector_dims dimensions"
        log::error "   Collection '$collection' expects $collection_dims dimensions"
        return 1
    fi
    
    # Insert vector
    local request_body
    request_body=$(jq -n \
        --argjson vector "$vector" \
        --argjson payload "$payload" \
        --arg id "$point_id" \
        '{points: [{id: $id, vector: $vector, payload: $payload}]}')
    
    local response
    response=$(curl -s -X PUT "http://localhost:6333/collections/$collection/points" \
               -H "Content-Type: application/json" \
               -d "$request_body")
    
    # Check response
    if echo "$response" | jq -e '.result.operation_id' >/dev/null 2>&1; then
        local op_id
        op_id=$(echo "$response" | jq -r '.result.operation_id')
        log::success "✅ Vector inserted successfully (operation: $op_id)"
        
        # Verify insertion by retrieving the point
        log::info "Verifying insertion..."
        local verify_response
        verify_response=$(curl -s "http://localhost:6333/collections/$collection/points/$point_id")
        
        if echo "$verify_response" | jq -e '.result.id' >/dev/null 2>&1; then
            log::success "✅ Insertion verified"
            return 0
        else
            log::warn "⚠️  Could not verify insertion"
            return 1
        fi
    else
        log::error "❌ Vector insertion failed"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 1
    fi
}

#######################################
# Complete pipeline: generate embedding and insert
# Arguments:
#   $1 - model name
#   $2 - input text
#   $3 - collection name
#   $4 - point ID
#   $5 - payload JSON object
# Returns:
#   0 on success, 1 on failure
#######################################
embedding::pipeline_validated() {
    local model="$1"
    local input="$2"
    local collection="$3"
    local point_id="$4"
    local payload="$5"
    
    log::header "🔄 Validated Embedding Pipeline"
    log::info "Model: $model"
    log::info "Collection: $collection"
    log::info "Point ID: $point_id"
    
    # Generate embedding with validation
    local embedding_response
    if ! embedding_response=$(embedding::generate_validated "$model" "$input" "$collection"); then
        log::error "Pipeline failed at embedding generation"
        return 1
    fi
    
    # Extract vector
    local vector
    vector=$(echo "$embedding_response" | jq '.embeddings[0]')
    
    # Insert with validation
    if embedding::insert_validated "$collection" "$point_id" "$vector" "$payload"; then
        log::success "🎉 Pipeline completed successfully!"
        return 0
    else
        log::error "Pipeline failed at vector insertion"
        return 1
    fi
}

#######################################
# Diagnostic function to check system health
# Returns:
#   0 if all systems healthy, 1 otherwise
#######################################
embedding::diagnose() {
    log::header "🔍 Embedding System Diagnostics"
    
    local all_healthy=true
    
    # Check Ollama
    log::info "Checking Ollama service..."
    if curl -s "http://localhost:11434/" >/dev/null 2>&1; then
        log::success "✅ Ollama is running"
    else
        log::error "❌ Ollama is not running on port 11434"
        all_healthy=false
    fi
    
    # Check Qdrant
    log::info "Checking Qdrant service..."
    if curl -s "http://localhost:6333/" >/dev/null 2>&1; then
        log::success "✅ Qdrant is running"
    else
        log::error "❌ Qdrant is not running on port 6333"
        all_healthy=false
    fi
    
    # Check recommended model
    log::info "Checking recommended embedding model..."
    if embedding::validate_model "nomic-embed-text:latest" >/dev/null 2>&1; then
        log::success "✅ nomic-embed-text:latest is available"
    else
        log::warn "⚠️  nomic-embed-text:latest not available"
        log::info "   Install with: ollama pull nomic-embed-text:latest"
    fi
    
    # List available collections
    log::info "Listing Qdrant collections..."
    local collections_response
    if collections_response=$(curl -s "http://localhost:6333/collections" 2>/dev/null); then
        local collections
        collections=$(echo "$collections_response" | jq -r '.result.collections[].name' 2>/dev/null | tr '\n' ' ')
        if [[ -n "$collections" ]]; then
            log::info "   Available collections: $collections"
        else
            log::warn "   No collections found"
        fi
    fi
    
    if $all_healthy; then
        log::success "🎉 All systems healthy!"
        return 0
    else
        log::error "⚠️  Some issues detected - see messages above"
        return 1
    fi
}

# If script is run directly, run diagnostics
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    embedding::diagnose
fi