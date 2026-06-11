#!/usr/bin/env bash
# Qdrant Model Management - Ollama Embedding Model Discovery and Validation
# Provides model discovery, dimension detection, and compatibility validation

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
QDRANT_MODELS_DIR="${RESOURCE_DIR}/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
# http-utils.sh not found, using curl directly instead

# Source configuration
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/config/defaults.sh"
if [[ -z "${QDRANT_CONFIG_EXPORTED:-}" ]]; then
    qdrant::export_config 2>/dev/null || true
fi

# Model cache file location
QDRANT_MODEL_CACHE="${QDRANT_DATA_DIR:-/var/lib/qdrant}/.model_cache.json"
QDRANT_MODEL_CACHE_TTL=3600  # 1 hour cache TTL
QDRANT_DEFAULT_EMBEDDING_ROLE="${QDRANT_DEFAULT_EMBEDDING_ROLE:-embedding.default}"

# Known embedding model patterns
declare -A EMBEDDING_MODEL_PATTERNS=(
    ["embed"]="embedding model"
    ["e5"]="embedding model"
    ["bge"]="embedding model"
    ["gte"]="embedding model"
    ["instructor"]="embedding model"
    ["sentence-transformers"]="embedding model"
)

#######################################
# Check if Ollama is available
# Returns: 0 if available, 1 if not
#######################################
qdrant::models::check_ollama() {
    local ollama_url="${OLLAMA_BASE_URL:-http://localhost:11434}"
    
    if ! curl -s "${ollama_url}/api/tags" 2>/dev/null | jq -e '.models' >/dev/null 2>&1; then
        log::debug "Ollama not available at ${ollama_url}"
        return 1
    fi
    
    return 0
}

#######################################
# Resolve Ollama policy metadata for a role.
# Arguments:
#   $1 - Role name
# Outputs: policy JSON
# Returns: 0 on success, 1 on failure
#######################################
qdrant::models::policy_resolve_role() {
    local role="${1:-$QDRANT_DEFAULT_EMBEDDING_ROLE}"

    if ! command -v resource-ollama >/dev/null 2>&1; then
        log::error "resource-ollama is required for Ollama model policy resolution"
        return 1
    fi

    resource-ollama policy resolve --role "$role" --json
}

#######################################
# Resolve Ollama policy metadata for a catalog model.
# Arguments:
#   $1 - Model reference
# Outputs: policy JSON
# Returns: 0 on success, 1 on failure
#######################################
qdrant::models::policy_resolve_model() {
    local model="$1"

    if [[ -z "$model" ]]; then
        log::error "Model name is required"
        return 1
    fi
    if ! command -v resource-ollama >/dev/null 2>&1; then
        log::error "resource-ollama is required for Ollama model policy resolution"
        return 1
    fi

    resource-ollama policy resolve --model "$model" --json
}

#######################################
# Resolve role or model metadata. Roles are preferred when both could parse.
# Arguments:
#   $1 - Role or model reference
# Outputs: policy JSON
# Returns: 0 on success, 1 on failure
#######################################
qdrant::models::policy_resolve_ref() {
    local ref="${1:-$QDRANT_DEFAULT_EMBEDDING_ROLE}"

    if qdrant::models::policy_resolve_role "$ref" 2>/dev/null; then
        return 0
    fi
    qdrant::models::policy_resolve_model "$ref"
}

#######################################
# Discover all available Ollama models
# Outputs: JSON array of model information
# Returns: 0 on success, 1 on failure
#######################################
qdrant::models::discover_ollama() {
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
    
    if ! command -v resource-ollama >/dev/null 2>&1; then
        log::error "resource-ollama is required for model policy discovery"
        echo "[]"
        return 1
    fi

    log::debug "Discovering Ollama embedding models from resource policy..."

    local policy_models
    if ! policy_models=$(resource-ollama policy models --json 2>/dev/null); then
        log::error "Failed to retrieve Ollama policy models"
        echo "[]"
        return 1
    fi

    local models_info
    models_info=$(echo "$policy_models" | jq '
        [.models[]?
        | select((.capabilities // []) | index("embedding"))
        | {
            name: .model,
            type: "embedding",
            dimensions: (.embedding_dimensions // "unknown"),
            is_embedding: true
        }]'
    )
    
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

    log::debug "Resolving embedding dimensions from Ollama policy: $model_name"

    local resolved
    if resolved=$(qdrant::models::policy_resolve_ref "$model_name" 2>/dev/null); then
        local dimensions
        dimensions=$(echo "$resolved" | jq -r '.embedding_dimensions // "unknown"' 2>/dev/null || echo "unknown")
        if [[ "$dimensions" =~ ^[0-9]+$ ]]; then
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
    
    local default_policy
    if ! default_policy=$(qdrant::models::policy_resolve_role "$QDRANT_DEFAULT_EMBEDDING_ROLE" 2>/dev/null); then
        log::error "Cannot resolve default embedding role: $QDRANT_DEFAULT_EMBEDDING_ROLE"
        return 1
    fi
    local default_dims
    default_dims=$(echo "$default_policy" | jq -r '.embedding_dimensions // "unknown"')

    # Determine required dimensions
    local required_dims
    if [[ "$collection_or_dims" =~ ^[0-9]+$ ]]; then
        required_dims="$collection_or_dims"
    else
        required_dims=$(qdrant::collections::get_dimensions "$collection_or_dims" 2>/dev/null || echo "")
        
        if [[ -z "$required_dims" ]] || [[ "$required_dims" == "unknown" ]]; then
            log::debug "Collection not found, using default embedding role"
            echo "$QDRANT_DEFAULT_EMBEDDING_ROLE"
            return 0
        fi
    fi

    if [[ "$required_dims" == "$default_dims" ]]; then
        echo "$QDRANT_DEFAULT_EMBEDDING_ROLE"
        return 0
    fi
    
    # Get compatible models
    local compatible_models
    compatible_models=$(qdrant::models::list_compatible "$required_dims" 2>/dev/null)
    
    if [[ -z "$compatible_models" ]]; then
        log::error "No policy-known embedding model found for $required_dims dimensions" >&2
        return 1
    fi

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
            log::info "Install the model resolved by: resource-ollama policy resolve --role $QDRANT_DEFAULT_EMBEDDING_ROLE --field model"
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
    models_response=$(curl -s "${OLLAMA_BASE_URL:-http://localhost:11434}/api/tags" 2>/dev/null)
    
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
        log::error "No embedding models found in Ollama policy"
        log::info "Check: resource-ollama policy resolve --role $QDRANT_DEFAULT_EMBEDDING_ROLE --json"
        return 1
    fi
    
    return 0
}
