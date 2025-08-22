#!/usr/bin/env bash
# Qdrant Embedding Management - Generate and manage embeddings via Ollama
# Provides embedding generation, caching, and batch processing

set -euo pipefail

# Get directory of this script
QDRANT_EMBEDDINGS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required utilities
# shellcheck disable=SC1091
source "${QDRANT_EMBEDDINGS_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"

# Source configuration and dependencies
# shellcheck disable=SC1091
source "${QDRANT_EMBEDDINGS_DIR}/../config/defaults.sh" 2>/dev/null || true
# Export configuration variables
qdrant::export_config 2>/dev/null || true
# shellcheck disable=SC1091
source "${QDRANT_EMBEDDINGS_DIR}/models.sh"

# Embedding cache settings
QDRANT_EMBEDDING_CACHE_ENABLED="${QDRANT_EMBEDDING_CACHE_ENABLED:-true}"
QDRANT_EMBEDDING_CACHE_TTL="${QDRANT_EMBEDDING_CACHE_TTL:-3600}"  # 1 hour
QDRANT_EMBEDDING_BATCH_SIZE="${QDRANT_EMBEDDING_BATCH_SIZE:-10}"  # Process in batches of 10

# Check if Redis is available for caching
REDIS_AVAILABLE="false"
if command -v redis-cli >/dev/null 2>&1 && redis-cli ping >/dev/null 2>&1; then
    REDIS_AVAILABLE="true"
fi

#######################################
# Parse embedding arguments
# Arguments: All CLI arguments passed to embed command
# Sets global variables: PARSED_TEXT, PARSED_MODEL, PARSED_INFO_ONLY, PARSED_BATCH, PARSED_FROM_FILE
# Returns: 0 on success, 1 on failure
#######################################
qdrant::embeddings::parse_args() {
    local text=""
    local model=""
    local info_only="false"
    local batch="false"
    local from_file=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --model|-m)
                model="$2"
                shift 2
                ;;
            --info)
                info_only="true"
                shift
                ;;
            --batch|-b)
                batch="true"
                shift
                ;;
            --from-file|-f)
                from_file="$2"
                shift 2
                ;;
            *)
                if [[ -z "$text" ]]; then
                    text="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Set global variables for caller
    PARSED_TEXT="$text"
    PARSED_MODEL="$model"
    PARSED_INFO_ONLY="$info_only"
    PARSED_BATCH="$batch"
    PARSED_FROM_FILE="$from_file"
    
    return 0
}

#######################################
# Execute embedding with parsed arguments
# Uses global variables set by parse_args
# Returns: 0 on success, 1 on failure
#######################################
qdrant::embeddings::execute_parsed() {
    if [[ "${PARSED_INFO_ONLY:-}" == "true" ]]; then
        qdrant::embeddings::info "${PARSED_MODEL:-}"
        return $?
    fi
    
    if [[ -n "${PARSED_FROM_FILE:-}" ]]; then
        if command -v qdrant::embeddings::from_file &>/dev/null; then
            qdrant::embeddings::from_file "$PARSED_FROM_FILE" "${PARSED_MODEL:-}"
        else
            log::error "File embedding not available"
            return 1
        fi
    elif [[ "${PARSED_BATCH:-}" == "true" ]]; then
        local text="${PARSED_TEXT:-}"
        if [[ -z "$text" ]]; then
            # Read from stdin
            text=$(cat)
        fi
        if command -v qdrant::embeddings::batch &>/dev/null; then
            qdrant::embeddings::batch "$text" "${PARSED_MODEL:-}"
        else
            log::error "Batch embedding not available"
            return 1
        fi
    else
        if [[ -z "${PARSED_TEXT:-}" ]]; then
            log::error "Text required for embedding"
            return 1
        fi
        
        qdrant::embeddings::generate "${PARSED_TEXT}" "${PARSED_MODEL:-}"
    fi
}

#######################################
# Generate embedding for text using Ollama
# Arguments:
#   $1 - Text to embed
#   $2 - Model name (optional, auto-select if not provided)
#   $3 - Use cache (true/false, default: true)
# Outputs: JSON array of embedding values
# Returns: 0 on success, 1 on failure
#######################################
qdrant::embeddings::generate() {
    local text="$1"
    local model="${2:-}"
    local use_cache="${3:-$QDRANT_EMBEDDING_CACHE_ENABLED}"
    
    if [[ -z "$text" ]]; then
        log::error "No text provided for embedding"
        return 1
    fi
    
    # Auto-select model if not provided
    if [[ -z "$model" ]]; then
        model=$(qdrant::models::auto_select "768" 2>/dev/null || echo "nomic-embed-text:latest")
        log::debug "Auto-selected model: $model"
    fi
    
    # Check cache first
    log::debug "Checking cache: use_cache=$use_cache"
    if [[ "$use_cache" == "true" ]]; then
        log::debug "Cache enabled, checking for cached embedding"
        log::debug "About to call get_from_cache with text='$text', model='$model'"
        local cached_embedding
        log::debug "Calling get_from_cache function"
        
        # Wrap cache call in error handling to prevent script abortion
        if cached_embedding=$(qdrant::embeddings::get_from_cache "$text" "$model" 2>/dev/null) && [[ -n "$cached_embedding" ]]; then
            log::debug "Using cached embedding for text"
            echo "$cached_embedding"
            return 0
        else
            log::debug "No cached embedding found or cache access failed, generating new embedding"
        fi
    else
        log::debug "Cache disabled"
    fi
    
    # Validate model is available
    if ! qdrant::models::get_model_dimensions "$model" >/dev/null 2>&1; then
        log::error "Model '$model' not found or not an embedding model"
        log::info "Available embedding models:"
        qdrant::models::get_embedding_models | jq -r '.[].name'
        return 1
    fi
    
    # Generate embedding via Ollama
    local ollama_url="${OLLAMA_BASE_URL:-http://localhost:11434}"
    local request_body
    request_body=$(jq -n \
        --arg model "$model" \
        --arg prompt "$text" \
        '{model: $model, prompt: $prompt}')
    
    log::debug "Generating embedding with model: $model"
    log::debug "Request body: $request_body"
    log::debug "Ollama URL: ${ollama_url}/api/embeddings"
    
    local response
    response=$(http::request "POST" "${ollama_url}/api/embeddings" "$request_body" "Content-Type: application/json")
    local http_exit_code=$?
    
    log::debug "HTTP response: $response"
    log::debug "HTTP exit code: $http_exit_code"
    
    if [[ -z "$response" ]]; then
        log::error "Failed to generate embedding: empty response"
        return 1
    fi
    
    # Extract embedding from response
    local embedding
    embedding=$(echo "$response" | jq '.embedding' 2>/dev/null)
    
    if [[ -z "$embedding" ]] || [[ "$embedding" == "null" ]]; then
        log::error "Invalid embedding response from Ollama"
        return 1
    fi
    
    # Validate embedding dimensions
    local actual_dims
    actual_dims=$(echo "$embedding" | jq 'length')
    log::debug "Generated embedding with $actual_dims dimensions"
    
    # Cache the embedding
    if [[ "$use_cache" == "true" ]]; then
        qdrant::embeddings::store_in_cache "$text" "$model" "$embedding"
    fi
    
    echo "$embedding"
    return 0
}

#######################################
# Generate embeddings for multiple texts (batch)
# Arguments:
#   $1 - Array of texts (JSON array or newline-separated)
#   $2 - Model name (optional)
#   $3 - Show progress (true/false, default: true)
# Outputs: JSON array of embeddings
# Returns: 0 on success, 1 on failure
#######################################
qdrant::embeddings::batch() {
    local texts="$1"
    local model="${2:-}"
    local show_progress="${3:-true}"
    
    # Convert input to JSON array if needed
    local text_array
    if echo "$texts" | jq -e 'type == "array"' >/dev/null 2>&1; then
        text_array="$texts"
    else
        # Convert newline-separated text to JSON array
        text_array="[]"
        while IFS= read -r line; do
            if [[ -n "$line" ]]; then
                text_array=$(echo "$text_array" | jq --arg text "$line" '. += [$text]')
            fi
        done <<< "$texts"
    fi
    
    local count
    count=$(echo "$text_array" | jq 'length')
    
    if [[ "$count" -eq 0 ]]; then
        log::error "No texts provided for batch embedding"
        echo "[]"
        return 1
    fi
    
    # Auto-select model if not provided
    if [[ -z "$model" ]]; then
        model=$(qdrant::models::auto_select "768" 2>/dev/null || echo "nomic-embed-text:latest")
        log::info "Using model: $model for batch embedding"
    fi
    
    log::info "Generating embeddings for $count texts..."
    
    local embeddings="[]"
    local processed=0
    local failed=0
    
    # Process in batches
    for ((i=0; i<count; i+=QDRANT_EMBEDDING_BATCH_SIZE)); do
        local batch_end=$((i + QDRANT_EMBEDDING_BATCH_SIZE))
        if [[ $batch_end -gt $count ]]; then
            batch_end=$count
        fi
        
        # Process batch
        for ((j=i; j<batch_end; j++)); do
            local text
            text=$(echo "$text_array" | jq -r ".[$j]")
            
            if [[ "$show_progress" == "true" ]]; then
                echo -ne "\rProcessing: $((j+1))/$count"
            fi
            
            local embedding
            embedding=$(qdrant::embeddings::generate "$text" "$model" "true" 2>/dev/null)
            
            if [[ -n "$embedding" ]] && [[ "$embedding" != "null" ]]; then
                embeddings=$(echo "$embeddings" | jq ". += [$embedding]")
                ((processed++))
            else
                log::debug "Failed to generate embedding for text $((j+1))"
                # Add null for failed embeddings to maintain index alignment
                embeddings=$(echo "$embeddings" | jq '. += [null]')
                ((failed++))
            fi
        done
        
        # Brief pause between batches to avoid overwhelming Ollama
        if [[ $batch_end -lt $count ]]; then
            sleep 0.1
        fi
    done
    
    if [[ "$show_progress" == "true" ]]; then
        echo  # New line after progress
    fi
    
    if [[ $failed -gt 0 ]]; then
        log::warn "Failed to generate $failed out of $count embeddings"
    fi
    
    log::success "Generated $processed embeddings successfully"
    
    echo "$embeddings"
    return 0
}

#######################################
# Get embedding from cache
# Arguments:
#   $1 - Text
#   $2 - Model name
# Outputs: Cached embedding or empty string
# Returns: 0 if found, 1 if not
#######################################
qdrant::embeddings::get_from_cache() {
    local text="$1"
    local model="$2"
    
    log::debug "get_from_cache called with text='$text', model='$model'"
    log::debug "QDRANT_EMBEDDING_CACHE_ENABLED='$QDRANT_EMBEDDING_CACHE_ENABLED'"
    
    if [[ "$QDRANT_EMBEDDING_CACHE_ENABLED" != "true" ]]; then
        log::debug "Cache not enabled, returning 1"
        return 1
    fi
    
    log::debug "Generating cache key"
    local cache_key
    cache_key=$(qdrant::embeddings::generate_cache_key "$text" "$model" 2>/dev/null) || {
        log::debug "Failed to generate cache key, disabling cache for this request"
        return 1
    }
    log::debug "Cache key: $cache_key"
    
    if [[ "$REDIS_AVAILABLE" == "true" ]]; then
        # Try Redis cache
        local cached
        cached=$(redis-cli GET "qdrant:embed:$cache_key" 2>/dev/null || echo "") || {
            log::debug "Redis cache access failed, falling back"
            cached=""
        }
        
        if [[ -n "$cached" ]]; then
            log::debug "Cache hit (Redis)"
            echo "$cached"
            return 0
        fi
    else
        # Try file cache
        local cache_file="${QDRANT_DATA_DIR:-/var/lib/qdrant}/.embed_cache/${cache_key}.json"
        
        if [[ -f "$cache_file" ]]; then
            local cache_age
            cache_age=$(( $(date +%s) - $(stat -c %Y "$cache_file" 2>/dev/null || echo 0) ))
            
            if [[ $cache_age -lt $QDRANT_EMBEDDING_CACHE_TTL ]]; then
                log::debug "Cache hit (file)"
                cat "$cache_file"
                return 0
            fi
        fi
    fi
    
    return 1
}

#######################################
# Store embedding in cache
# Arguments:
#   $1 - Text
#   $2 - Model name
#   $3 - Embedding (JSON array)
# Returns: 0 on success
#######################################
qdrant::embeddings::store_in_cache() {
    local text="$1"
    local model="$2"
    local embedding="$3"
    
    if [[ "$QDRANT_EMBEDDING_CACHE_ENABLED" != "true" ]]; then
        return 0
    fi
    
    local cache_key
    cache_key=$(qdrant::embeddings::generate_cache_key "$text" "$model")
    
    if [[ "$REDIS_AVAILABLE" == "true" ]]; then
        # Store in Redis with TTL
        redis-cli SETEX "qdrant:embed:$cache_key" "$QDRANT_EMBEDDING_CACHE_TTL" "$embedding" >/dev/null 2>&1 || true
        log::debug "Cached in Redis"
    else
        # Store in file cache
        local cache_dir="${QDRANT_DATA_DIR:-/var/lib/qdrant}/.embed_cache"
        mkdir -p "$cache_dir"
        
        echo "$embedding" > "${cache_dir}/${cache_key}.json"
        log::debug "Cached in file"
        
        # Clean old cache files
        find "$cache_dir" -type f -mmin +$((QDRANT_EMBEDDING_CACHE_TTL / 60)) -delete 2>/dev/null || true
    fi
    
    return 0
}

#######################################
# Generate cache key for text and model
# Arguments:
#   $1 - Text
#   $2 - Model name
# Outputs: Cache key
#######################################
qdrant::embeddings::generate_cache_key() {
    local text="$1"
    local model="$2"
    
    # Create hash of text and model for cache key
    echo -n "${model}:${text}" | sha256sum | cut -d' ' -f1
}

#######################################
# Clear embedding cache
# Returns: 0 on success
#######################################
qdrant::embeddings::clear_cache() {
    log::info "Clearing embedding cache..."
    
    if [[ "$REDIS_AVAILABLE" == "true" ]]; then
        # Clear Redis cache
        redis-cli --scan --pattern "qdrant:embed:*" | xargs -r redis-cli DEL 2>/dev/null || true
        log::debug "Cleared Redis cache"
    fi
    
    # Clear file cache
    local cache_dir="${QDRANT_DATA_DIR:-/var/lib/qdrant}/.embed_cache"
    if [[ -d "$cache_dir" ]]; then
        rm -rf "${cache_dir:?}"/*
        log::debug "Cleared file cache"
    fi
    
    log::success "Embedding cache cleared"
    return 0
}

#######################################
# Get embedding info (dimensions, model details)
# Arguments:
#   $1 - Model name (optional)
# Outputs: Model and embedding information
# Returns: 0 on success
#######################################
qdrant::embeddings::info() {
    local model="${1:-}"
    
    # Auto-select model if not provided
    if [[ -z "$model" ]]; then
        model=$(qdrant::models::auto_select "768" 2>/dev/null || echo "nomic-embed-text:latest")
    fi
    
    echo "=== Embedding Information ==="
    echo "Model: $model"
    
    # Get model dimensions
    local dimensions
    dimensions=$(qdrant::models::get_model_dimensions "$model")
    echo "Dimensions: $dimensions"
    
    # Cache status
    echo "Cache Enabled: $QDRANT_EMBEDDING_CACHE_ENABLED"
    if [[ "$QDRANT_EMBEDDING_CACHE_ENABLED" == "true" ]]; then
        echo "Cache Type: $([ "$REDIS_AVAILABLE" == "true" ] && echo "Redis" || echo "File")"
        echo "Cache TTL: ${QDRANT_EMBEDDING_CACHE_TTL}s"
    fi
    
    # Ollama status
    if qdrant::models::check_ollama; then
        echo "Ollama Status: ✅ Available"
    else
        echo "Ollama Status: ❌ Not available"
    fi
    
    echo
    echo "Available Embedding Models:"
    qdrant::models::get_embedding_models | jq -r '.[] | "  • \(.name) (\(.dimensions) dims)"'
}

#######################################
# Embed text from file
# Arguments:
#   $1 - File path
#   $2 - Model name (optional)
#   $3 - Output format (json/raw, default: json)
# Outputs: Embeddings
# Returns: 0 on success, 1 on failure
#######################################
qdrant::embeddings::from_file() {
    local file_path="$1"
    local model="${2:-}"
    local output_format="${3:-json}"
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    # Read file content
    local content
    content=$(cat "$file_path")
    
    if [[ -z "$content" ]]; then
        log::error "File is empty: $file_path"
        return 1
    fi
    
    # Check if file contains multiple lines (batch) or single text
    local line_count
    line_count=$(wc -l < "$file_path")
    
    if [[ $line_count -gt 1 ]]; then
        log::info "Processing $line_count lines as batch..."
        qdrant::embeddings::batch "$content" "$model"
    else
        # Single text embedding
        local embedding
        embedding=$(qdrant::embeddings::generate "$content" "$model")
        
        if [[ "$output_format" == "raw" ]]; then
            echo "$embedding" | jq -r '.[]'
        else
            echo "$embedding"
        fi
    fi
}

#######################################
# Validate embedding dimensions against collection
# Arguments:
#   $1 - Embedding (JSON array)
#   $2 - Collection name or expected dimensions
# Returns: 0 if valid, 1 if not
#######################################
qdrant::embeddings::validate_dimensions() {
    local embedding="$1"
    local collection_or_dims="$2"
    
    # Get embedding dimensions
    local actual_dims
    actual_dims=$(echo "$embedding" | jq 'length' 2>/dev/null || echo "0")
    
    if [[ "$actual_dims" -eq 0 ]]; then
        log::error "Invalid embedding format"
        return 1
    fi
    
    # Get expected dimensions
    local expected_dims
    if [[ "$collection_or_dims" =~ ^[0-9]+$ ]]; then
        expected_dims="$collection_or_dims"
    else
        # Get collection dimensions
        expected_dims=$(qdrant::collections::get_dimensions "$collection_or_dims" 2>/dev/null || echo "unknown")
        
        if [[ "$expected_dims" == "unknown" ]]; then
            log::error "Cannot determine dimensions for collection: $collection_or_dims"
            return 1
        fi
    fi
    
    if [[ "$actual_dims" -eq "$expected_dims" ]]; then
        log::debug "Embedding dimensions valid: $actual_dims"
        return 0
    else
        log::error "Dimension mismatch: Embedding has $actual_dims dimensions, expected $expected_dims"
        return 1
    fi
}

#######################################
# Get optimal model for text type
# Arguments:
#   $1 - Text type hint (code/document/short/long)
#   $2 - Preferred dimensions (optional)
# Outputs: Recommended model name
# Returns: 0 on success
#######################################
qdrant::embeddings::recommend_model() {
    local text_type="${1:-general}"
    local preferred_dims="${2:-768}"
    
    # Model recommendations based on use case
    case "$text_type" in
        code)
            # Prefer code-specific models
            if qdrant::models::get_model_dimensions "codellama:7b" >/dev/null 2>&1; then
                echo "codellama:7b"
            else
                qdrant::models::auto_select "$preferred_dims"
            fi
            ;;
        document|long)
            # Prefer larger models for documents
            if qdrant::models::get_model_dimensions "mxbai-embed-large:latest" >/dev/null 2>&1; then
                echo "mxbai-embed-large:latest"
            else
                qdrant::models::auto_select "1024"
            fi
            ;;
        short|general|*)
            # Default to efficient general model
            qdrant::models::auto_select "$preferred_dims"
            ;;
    esac
}