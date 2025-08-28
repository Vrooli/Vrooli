#!/usr/bin/env bash
# Qdrant Model Management - Ollama Embedding Model Discovery and Validation
# Provides model discovery, dimension detection, and compatibility validation

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
QDRANT_MODELS_DIR="${APP_ROOT}/resources/qdrant/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"

# Source configuration
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/qdrant/config/defaults.sh"

# Model cache file location
QDRANT_MODEL_CACHE="${QDRANT_DATA_DIR:-/var/lib/qdrant}/.model_cache.json"
QDRANT_MODEL_CACHE_TTL=3600  # 1 hour cache TTL

# Known embedding model patterns
declare -A EMBEDDING_MODEL_PATTERNS=(
    ["embed"]="embedding model"
    ["e5"]="embedding model"
    ["bge"]="embedding model"
    ["gte"]="embedding model"
    ["instructor"]="embedding model"
    ["sentence-transformers"]="embedding model"
)

# Model dimension registry (fallback for known models)
declare -A KNOWN_MODEL_DIMENSIONS=(
    ["nomic-embed-text"]="768"
    ["nomic-embed-text:latest"]="768"
    ["mxbai-embed-large"]="1024"
    ["mxbai-embed-large:latest"]="1024"
    ["all-minilm"]="384"
    ["all-minilm:latest"]="384"
    ["bge-small"]="384"
    ["bge-base"]="768"
    ["bge-large"]="1024"
)

#######################################
# Check if Ollama is available
# Returns: 0 if available, 1 if not
#######################################
qdrant::models::check_ollama() {
    local ollama_url="${OLLAMA_BASE_URL:-http://localhost:11434}"
    
    if ! http::request "GET" "${ollama_url}/api/tags" "" "" 2>/dev/null | jq -e '.models' >/dev/null 2>&1; then
        log::debug "Ollama not available at ${ollama_url}"
        return 1
    fi
    
    return 0
}

#######################################
# Discover all available Ollama models
# Outputs: JSON array of model information
# Returns: 0 on success, 1 on failure
#######################################
qdrant::models::discover_ollama() {
    local ollama_url="${OLLAMA_BASE_URL:-http://localhost:11434}"
    local use_cache="${1:-true}"
    
    # Check cache if enabled
    if [[ "$use_cache" == "true" ]] && [[ -f "$QDRANT_MODEL_CACHE" ]]; then
        local cache_age
        cache_age=$(( $(date +%s) - $(stat -c %Y "$QDRANT_MODEL_CACHE" 2>/dev/null || echo 0) ))
        
        if [[ $cache_age -lt $QDRANT_MODEL_CACHE_TTL ]]; then
            log::debug "Using cached model information (age: ${cache_age}s)"
            cat "$QDRANT_MODEL_CACHE"
            return 0
        fi
    fi
    
    # Check if Ollama is available
    if ! qdrant::models::check_ollama; then
        log::error "Ollama is not available. Please ensure it's running."
        echo "[]"
        return 1
    fi
    
    log::debug "Discovering Ollama models..."
    
    # Get all models from Ollama
    local models_response
    models_response=$(http::request "GET" "${ollama_url}/api/tags" "" "" 2>/dev/null)
    
    if [[ -z "$models_response" ]]; then
        log::error "Failed to retrieve models from Ollama"
        echo "[]"
        return 1
    fi
    
    # Extract model names
    local model_names
    model_names=$(echo "$models_response" | jq -r '.models[]?.name // empty' 2>/dev/null)
    
    if [[ -z "$model_names" ]]; then
        log::warn "No models found in Ollama"
        echo "[]"
        return 0
    fi
    
    # Build model information array
    local models_info="[]"
    
    while IFS= read -r model_name; do
        if [[ -n "$model_name" ]]; then
            # Check if it's likely an embedding model
            local is_embedding="false"
            local model_type="general"
            
            for pattern in "${!EMBEDDING_MODEL_PATTERNS[@]}"; do
                if [[ "$model_name" == *"$pattern"* ]]; then
                    is_embedding="true"
                    model_type="embedding"
                    break
                fi
            done
            
            # Try to get dimensions if it's an embedding model
            local dimensions="unknown"
            if [[ "$is_embedding" == "true" ]]; then
                # Check known dimensions first
                if [[ -n "${KNOWN_MODEL_DIMENSIONS[$model_name]:-}" ]]; then
                    dimensions="${KNOWN_MODEL_DIMENSIONS[$model_name]}"
                else
                    # Try to detect dimensions by testing the model
                    dimensions=$(qdrant::models::detect_dimensions "$model_name")
                fi
            fi
            
            # Add model to array
            local model_info
            model_info=$(jq -n \
                --arg name "$model_name" \
                --arg type "$model_type" \
                --arg dimensions "$dimensions" \
                --arg is_embedding "$is_embedding" \
                '{
                    name: $name,
                    type: $type,
                    dimensions: ($dimensions | tonumber? // $dimensions),
                    is_embedding: ($is_embedding | test("true"))
                }')
            
            models_info=$(echo "$models_info" | jq ". += [$model_info]")
        fi
    done <<< "$model_names"
    
    # Cache the results
    mkdir -p "${QDRANT_MODEL_CACHE%/*}"
    echo "$models_info" > "$QDRANT_MODEL_CACHE"
    
    echo "$models_info"
    return 0
}

#######################################
# Detect embedding dimensions for a model
# Arguments:
#   $1 - Model name
# Outputs: Dimension count or "unknown"
# Returns: 0 on success, 1 on failure
#######################################
qdrant::models::detect_dimensions() {
    local model_name="$1"
    local ollama_url="${OLLAMA_BASE_URL:-http://localhost:11434}"
    
    # First check known dimensions
    if [[ -n "${KNOWN_MODEL_DIMENSIONS[$model_name]:-}" ]]; then
        echo "${KNOWN_MODEL_DIMENSIONS[$model_name]}"
        return 0
    fi
    
    log::debug "Detecting dimensions for model: $model_name"
    
    # First, check if the model exists at all
    local models_response
    models_response=$(timeout 3 curl -s "${ollama_url}/api/tags" 2>/dev/null || echo "")
    
    if [[ -z "$models_response" ]]; then
        log::error "Cannot connect to Ollama at $ollama_url"
        echo "unknown"
        return 1
    fi
    
    # Check if model is in the list
    local model_exists
    model_exists=$(echo "$models_response" | jq -r ".models[]? | select(.name == \"$model_name\") | .name" 2>/dev/null || echo "")
    
    if [[ -z "$model_exists" ]]; then
        log::error "Model '$model_name' not found in Ollama"
        log::info "Available models: $(echo "$models_response" | jq -r '.models[]?.name' 2>/dev/null | head -3 | tr '\n' ' ')"
        echo "unknown"
        return 1
    fi
    
    # Try to generate a test embedding with timeout
    local test_text="test"
    local request_body
    request_body=$(jq -n \
        --arg model "$model_name" \
        --arg prompt "$test_text" \
        '{model: $model, prompt: $prompt}')
    
    local response
    # Use timeout command to prevent hanging (5 seconds should be enough for a simple test)
    response=$(timeout 5 curl -s -X POST "${ollama_url}/api/embeddings" \
        -H "Content-Type: application/json" \
        -d "$request_body" 2>/dev/null || echo "")
    
    if [[ -n "$response" ]]; then
        # Check if response contains an error
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // empty' 2>/dev/null)
        
        if [[ -n "$error_msg" ]]; then
            log::debug "Model $model_name error: $error_msg"
            echo "unknown"
            return 1
        fi
        
        # Extract dimensions from successful response
        local dimensions
        dimensions=$(echo "$response" | jq '.embedding | length' 2>/dev/null || echo "unknown")
        
        if [[ "$dimensions" =~ ^[0-9]+$ ]]; then
            log::debug "Model $model_name has $dimensions dimensions"
            echo "$dimensions"
            return 0
        fi
    fi
    
    echo "unknown"
    return 1
}

#######################################
# Get embedding models only
# Arguments:
#   $1 - Include dimensions (true/false, default: true)
# Outputs: JSON array of embedding models
# Returns: 0 on success
#######################################
qdrant::models::get_embedding_models() {
    local include_dimensions="${1:-true}"
    
    local all_models
    all_models=$(qdrant::models::discover_ollama)
    
    # Filter for embedding models only
    local embedding_models
    embedding_models=$(echo "$all_models" | jq '[.[] | select(.is_embedding == true)]' 2>/dev/null || echo "[]")
    
    if [[ "$include_dimensions" != "true" ]]; then
        # Remove dimension info if not needed
        embedding_models=$(echo "$embedding_models" | jq '[.[] | {name: .name, type: .type}]')
    fi
    
    echo "$embedding_models"
}

#######################################
# Validate model compatibility with collection
# Arguments:
#   $1 - Model name
#   $2 - Collection name or dimensions
# Returns: 0 if compatible, 1 if not
#######################################
qdrant::models::validate_compatibility() {
    local model_name="$1"
    local collection_or_dims="$2"
    
    # Get model dimensions
    local model_dims
    model_dims=$(qdrant::models::get_model_dimensions "$model_name")
    
    if [[ "$model_dims" == "unknown" ]]; then
        log::error "Cannot determine dimensions for model: $model_name"
        return 1
    fi
    
    # Check if second argument is a collection name or dimensions
    local required_dims
    if [[ "$collection_or_dims" =~ ^[0-9]+$ ]]; then
        required_dims="$collection_or_dims"
    else
        # Get collection dimensions
        required_dims=$(qdrant::collections::get_dimensions "$collection_or_dims" 2>/dev/null || echo "unknown")
        
        if [[ "$required_dims" == "unknown" ]]; then
            log::error "Cannot determine dimensions for collection: $collection_or_dims"
            return 1
        fi
    fi
    
    if [[ "$model_dims" == "$required_dims" ]]; then
        return 0
    else
        log::error "Dimension mismatch: Model '$model_name' has $model_dims dimensions, but $required_dims required"
        return 1
    fi
}

#######################################
# Get model dimensions
# Arguments:
#   $1 - Model name
# Outputs: Dimension count or "unknown"
# Returns: 0 on success
#######################################
qdrant::models::get_model_dimensions() {
    local model_name="$1"
    
    log::debug "get_model_dimensions called for: $model_name"
    log::debug "Cache file: $QDRANT_MODEL_CACHE"
    
    # Check cache first
    if [[ -f "$QDRANT_MODEL_CACHE" ]]; then
        log::debug "Cache file exists, checking for cached dimensions"
        local cached_dims
        cached_dims=$(timeout 5 jq -r ".[] | select(.name == \"$model_name\") | .dimensions" "$QDRANT_MODEL_CACHE" 2>/dev/null || echo "")
        log::debug "Cache lookup result: [$cached_dims]"
        
        if [[ -n "$cached_dims" ]] && [[ "$cached_dims" != "null" ]]; then
            log::debug "Returning cached dimensions: $cached_dims"
            echo "$cached_dims"
            return 0
        fi
    fi
    
    log::debug "No cache hit, calling detect_dimensions"
    # Try to detect dimensions
    qdrant::models::detect_dimensions "$model_name"
}

#######################################
# List compatible models for given dimensions
# Arguments:
#   $1 - Required dimensions
# Outputs: List of compatible model names
# Returns: 0 on success
#######################################
qdrant::models::list_compatible() {
    local required_dims="$1"
    
    if [[ ! "$required_dims" =~ ^[0-9]+$ ]]; then
        log::error "Invalid dimensions: $required_dims"
        return 1
    fi
    
    local all_models
    all_models=$(qdrant::models::get_embedding_models)
    
    # Filter for models with matching dimensions
    local compatible_models
    compatible_models=$(echo "$all_models" | jq -r ".[] | select(.dimensions == $required_dims) | .name" 2>/dev/null)
    
    if [[ -z "$compatible_models" ]]; then
        log::warn "No models found with $required_dims dimensions" >&2
        
        # Suggest alternatives
        log::info "Available embedding models:" >&2
        echo "$all_models" | jq -r '.[] | "\(.name): \(.dimensions) dimensions"' >&2 2>/dev/null
    else
        echo "$compatible_models"
    fi
}

#######################################
# Auto-select best model for collection
# Arguments:
#   $1 - Collection name or dimensions
#   $2 - Preferred model (optional)
# Outputs: Selected model name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::models::auto_select() {
    local collection_or_dims="$1"
    local preferred_model="${2:-}"
    
    # If preferred model is specified and compatible, use it
    if [[ -n "$preferred_model" ]]; then
        if qdrant::models::validate_compatibility "$preferred_model" "$collection_or_dims"; then
            echo "$preferred_model"
            return 0
        fi
        log::warn "Preferred model '$preferred_model' is not compatible"
    fi
    
    # Determine required dimensions
    local required_dims
    if [[ "$collection_or_dims" =~ ^[0-9]+$ ]]; then
        required_dims="$collection_or_dims"
    else
        required_dims=$(qdrant::collections::get_dimensions "$collection_or_dims" 2>/dev/null || echo "")
        
        if [[ -z "$required_dims" ]] || [[ "$required_dims" == "unknown" ]]; then
            # If collection doesn't exist, prefer 1024d model for better compatibility
            log::debug "Collection not found, using default 1024d model"
            echo "mxbai-embed-large:latest"
            return 0
        fi
    fi
    
    # Get compatible models
    local compatible_models
    compatible_models=$(qdrant::models::list_compatible "$required_dims" 2>/dev/null)
    
    if [[ -z "$compatible_models" ]]; then
        log::debug "No compatible models found for $required_dims dimensions, using fallback" >&2
        if [[ "$required_dims" == "1024" ]]; then
            echo "mxbai-embed-large:latest"
        else
            echo "nomic-embed-text:latest"
        fi
        return 0
    fi
    
    # Select the best compatible model based on dimensions
    if [[ "$required_dims" == "1024" ]]; then
        # For 1024d collections, prefer mxbai-embed-large
        while IFS= read -r model; do
            if [[ "$model" == *"mxbai-embed-large"* ]]; then
                echo "$model"
                return 0
            fi
        done <<< "$compatible_models"
    elif [[ "$required_dims" == "768" ]]; then
        # For 768d collections, prefer nomic-embed-text
        while IFS= read -r model; do
            if [[ "$model" == *"nomic-embed"* ]]; then
                echo "$model"
                return 0
            fi
        done <<< "$compatible_models"
    fi
    
    # Return first available model
    echo "$compatible_models" | head -n1
}

#######################################
# Display model information
# Arguments:
#   $1 - Model name (optional, show all if not specified)
# Returns: 0 on success
#######################################
qdrant::models::info() {
    local model_name="${1:-}"
    
    local all_models
    all_models=$(qdrant::models::discover_ollama)
    
    if [[ -n "$model_name" ]]; then
        # Show specific model info
        local model_info
        model_info=$(echo "$all_models" | jq ".[] | select(.name == \"$model_name\")" 2>/dev/null)
        
        # If not found, try with :latest tag
        if [[ -z "$model_info" ]] && [[ "$model_name" != *":"* ]]; then
            model_info=$(echo "$all_models" | jq ".[] | select(.name == \"${model_name}:latest\")" 2>/dev/null)
            if [[ -n "$model_info" ]]; then
                model_name="${model_name}:latest"
            fi
        fi
        
        if [[ -z "$model_info" ]]; then
            log::error "Model not found: $model_name"
            return 1
        fi
        
        echo "=== Model Information: $model_name ==="
        echo "$model_info" | jq -r '
            "Type: " + .type +
            "\nIs Embedding Model: " + (.is_embedding | tostring) +
            "\nDimensions: " + (.dimensions | tostring)
        '
    else
        # Show all embedding models
        echo "=== Available Embedding Models ==="
        echo
        
        local embedding_models
        embedding_models=$(echo "$all_models" | jq '[.[] | select(.is_embedding == true)]')
        
        if [[ $(echo "$embedding_models" | jq 'length') -eq 0 ]]; then
            log::warn "No embedding models found"
            log::info "Install embedding models with: ollama pull nomic-embed-text"
            return 0
        fi
        
        echo "$embedding_models" | jq -r '.[] | 
            "📊 " + .name + 
            "\n   Dimensions: " + (.dimensions | tostring) +
            "\n"'
    fi
}

#######################################
# Clear model cache
# Returns: 0 on success
#######################################
qdrant::models::clear_cache() {
    if [[ -f "$QDRANT_MODEL_CACHE" ]]; then
        rm -f "$QDRANT_MODEL_CACHE"
        log::info "Model cache cleared"
    fi
    return 0
}

#######################################
# Check if a specific model is available in Ollama
# Arguments:
#   $1 - Model name
# Returns: 0 if available, 1 if not
#######################################
qdrant::models::is_available() {
    local model_name="$1"
    
    if [[ -z "$model_name" ]]; then
        log::error "Model name is required"
        return 1
    fi
    
    # First check if Ollama is running
    if ! qdrant::models::check_ollama; then
        log::debug "Ollama not available"
        return 1
    fi
    
    # Get list of available models
    local models_response
    models_response=$(http::request "GET" "${OLLAMA_BASE_URL:-http://localhost:11434}/api/tags" "" "" 2>/dev/null)
    
    if [[ -z "$models_response" ]]; then
        log::debug "Failed to get models list from Ollama"
        return 1
    fi
    
    # Check if the model exists (handle both with and without :latest suffix)
    local model_found
    model_found=$(echo "$models_response" | jq -r "
        .models[]? | select(
            .name == \"$model_name\" or 
            .name == \"${model_name}:latest\" or
            (.name | split(\":\")[0]) == \"$model_name\"
        ) | .name" 2>/dev/null | head -1)
    
    if [[ -n "$model_found" ]]; then
        log::debug "Model '$model_name' is available (found as: $model_found)"
        return 0
    else
        log::debug "Model '$model_name' not found in Ollama"
        return 1
    fi
}

#######################################
# Install recommended embedding model if none available
# Returns: 0 on success, 1 on failure
#######################################
qdrant::models::ensure_embedding_model() {
    local embedding_models
    embedding_models=$(qdrant::models::get_embedding_models)
    
    if [[ $(echo "$embedding_models" | jq 'length') -eq 0 ]]; then
        log::info "No embedding models found. Installing nomic-embed-text..."
        
        # Try to pull the model using Ollama
        if command -v ollama >/dev/null 2>&1; then
            if ollama pull nomic-embed-text:latest; then
                log::success "Successfully installed nomic-embed-text embedding model"
                # Clear cache to refresh model list
                qdrant::models::clear_cache
                return 0
            else
                log::error "Failed to install embedding model"
                return 1
            fi
        else
            log::error "Ollama CLI not found. Please install embedding models manually."
            log::info "Run: ollama pull nomic-embed-text"
            return 1
        fi
    fi
    
    return 0
}