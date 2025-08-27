#!/usr/bin/env bash
# Qdrant Embeddings Unified Configuration
# Centralized configuration for all embedding-related settings
# Provides consistent environment variable overrides and default values

set -euo pipefail

#######################################
# Export unified embedding configuration
# Idempotent - safe to call multiple times
# Environment variables take precedence over defaults
#######################################
qdrant::embeddings::export_config() {
    # Guard against multiple calls
    if [[ -n "${QDRANT_EMBEDDINGS_CONFIG_EXPORTED:-}" ]]; then
        return 0
    fi
    readonly QDRANT_EMBEDDINGS_CONFIG_EXPORTED="true"
    
    # ================================
    # EMBEDDING MODEL CONFIGURATION
    # ================================
    
    # Default embedding model (can be overridden via QDRANT_EMBEDDING_MODEL)
    if [[ -z "${QDRANT_EMBEDDING_MODEL:-}" ]]; then
        QDRANT_EMBEDDING_MODEL="${QDRANT_EMBEDDING_MODEL_OVERRIDE:-mxbai-embed-large}"
    fi
    
    # Model dimensions (auto-detected if not set)
    if [[ -z "${QDRANT_EMBEDDING_DIMENSIONS:-}" ]]; then
        QDRANT_EMBEDDING_DIMENSIONS="${QDRANT_EMBEDDING_DIMENSIONS_OVERRIDE:-1024}"
    fi
    
    # Distance metric for collections
    if [[ -z "${QDRANT_EMBEDDING_DISTANCE_METRIC:-}" ]]; then
        QDRANT_EMBEDDING_DISTANCE_METRIC="${QDRANT_EMBEDDING_DISTANCE_METRIC_OVERRIDE:-Cosine}"
    fi
    
    # ================================
    # PARALLEL PROCESSING CONFIGURATION
    # ================================
    
    # Maximum parallel workers for embeddings (inherits from main qdrant config if set)
    if [[ -z "${EMBEDDING_MAX_WORKERS:-}" ]]; then
        # Inherit from main qdrant config if available, otherwise use embedding-specific default
        EMBEDDING_MAX_WORKERS="${EMBEDDING_MAX_WORKERS_OVERRIDE:-${QDRANT_MAX_WORKERS:-16}}"
    fi
    
    # Batch size for embedding generation
    if [[ -z "${EMBEDDING_BATCH_SIZE:-}" ]]; then
        EMBEDDING_BATCH_SIZE="${EMBEDDING_BATCH_SIZE_OVERRIDE:-50}"
    fi
    
    # Memory usage threshold (percentage) for embeddings
    if [[ -z "${EMBEDDING_MEMORY_LIMIT:-}" ]]; then
        EMBEDDING_MEMORY_LIMIT="${EMBEDDING_MEMORY_LIMIT_OVERRIDE:-80}"
    fi
    
    # ================================
    # CACHE CONFIGURATION
    # ================================
    
    # Cache TTL in seconds
    if [[ -z "${EMBEDDING_CACHE_TTL:-}" ]]; then
        EMBEDDING_CACHE_TTL="${EMBEDDING_CACHE_TTL_OVERRIDE:-3600}"
    fi
    
    # Enable/disable caching
    if [[ -z "${EMBEDDING_CACHE_ENABLED:-}" ]]; then
        EMBEDDING_CACHE_ENABLED="${EMBEDDING_CACHE_ENABLED_OVERRIDE:-true}"
    fi
    
    # Cache directory
    if [[ -z "${EMBEDDING_CACHE_DIR:-}" ]]; then
        EMBEDDING_CACHE_DIR="${EMBEDDING_CACHE_DIR_OVERRIDE:-${HOME}/.cache/qdrant-embeddings}"
    fi
    
    # ================================
    # PROCESSING CONFIGURATION
    # ================================
    
    # Content processing timeout (seconds)
    if [[ -z "${EMBEDDING_PROCESSING_TIMEOUT:-}" ]]; then
        EMBEDDING_PROCESSING_TIMEOUT="${EMBEDDING_PROCESSING_TIMEOUT_OVERRIDE:-300}"
    fi
    
    # Maximum retry attempts for failed embeddings
    if [[ -z "${EMBEDDING_MAX_RETRIES:-}" ]]; then
        EMBEDDING_MAX_RETRIES="${EMBEDDING_MAX_RETRIES_OVERRIDE:-3}"
    fi
    
    # Retry delay in seconds
    if [[ -z "${EMBEDDING_RETRY_DELAY:-}" ]]; then
        EMBEDDING_RETRY_DELAY="${EMBEDDING_RETRY_DELAY_OVERRIDE:-2}"
    fi
    
    # ================================
    # COLLECTION NAMING CONFIGURATION
    # ================================
    
    # Collection name suffixes
    if [[ -z "${EMBEDDING_WORKFLOWS_COLLECTION_SUFFIX:-}" ]]; then
        EMBEDDING_WORKFLOWS_COLLECTION_SUFFIX="${EMBEDDING_WORKFLOWS_COLLECTION_SUFFIX_OVERRIDE:-workflows}"
    fi
    
    if [[ -z "${EMBEDDING_SCENARIOS_COLLECTION_SUFFIX:-}" ]]; then
        EMBEDDING_SCENARIOS_COLLECTION_SUFFIX="${EMBEDDING_SCENARIOS_COLLECTION_SUFFIX_OVERRIDE:-scenarios}"
    fi
    
    if [[ -z "${EMBEDDING_KNOWLEDGE_COLLECTION_SUFFIX:-}" ]]; then
        EMBEDDING_KNOWLEDGE_COLLECTION_SUFFIX="${EMBEDDING_KNOWLEDGE_COLLECTION_SUFFIX_OVERRIDE:-knowledge}"
    fi
    
    if [[ -z "${EMBEDDING_CODE_COLLECTION_SUFFIX:-}" ]]; then
        EMBEDDING_CODE_COLLECTION_SUFFIX="${EMBEDDING_CODE_COLLECTION_SUFFIX_OVERRIDE:-code}"
    fi
    
    if [[ -z "${EMBEDDING_RESOURCES_COLLECTION_SUFFIX:-}" ]]; then
        EMBEDDING_RESOURCES_COLLECTION_SUFFIX="${EMBEDDING_RESOURCES_COLLECTION_SUFFIX_OVERRIDE:-resources}"
    fi
    
    # ================================
    # PERFORMANCE TUNING
    # ================================
    
    # Minimum content length to embed (avoid noise)
    if [[ -z "${EMBEDDING_MIN_CONTENT_LENGTH:-}" ]]; then
        EMBEDDING_MIN_CONTENT_LENGTH="${EMBEDDING_MIN_CONTENT_LENGTH_OVERRIDE:-50}"
    fi
    
    # Maximum content length before truncation
    if [[ -z "${EMBEDDING_MAX_CONTENT_LENGTH:-}" ]]; then
        EMBEDDING_MAX_CONTENT_LENGTH="${EMBEDDING_MAX_CONTENT_LENGTH_OVERRIDE:-8192}"
    fi
    
    # Progress reporting interval (number of items)
    if [[ -z "${EMBEDDING_PROGRESS_INTERVAL:-}" ]]; then
        EMBEDDING_PROGRESS_INTERVAL="${EMBEDDING_PROGRESS_INTERVAL_OVERRIDE:-100}"
    fi
    
    # ================================
    # AUTO-ADJUSTMENT AND VALIDATION
    # ================================
    
    # Auto-detect CPU cores if EMBEDDING_MAX_WORKERS is 0 (only on first load)
    if [[ "$EMBEDDING_MAX_WORKERS" -eq 0 ]] && [[ -z "${EMBEDDING_MAX_WORKERS_DETECTED:-}" ]]; then
        local detected_cores
        detected_cores=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
        # Cap at 16 workers for optimal embedding performance (I/O bound workload)
        if [[ "$detected_cores" -gt 16 ]]; then
            EMBEDDING_MAX_WORKERS="16"
        else
            EMBEDDING_MAX_WORKERS="$detected_cores"
        fi
        EMBEDDING_MAX_WORKERS_DETECTED="true"
    fi
    
    # Ensure minimum values (only on first load)
    if [[ "$EMBEDDING_MAX_WORKERS" -lt 1 ]] && [[ -z "${EMBEDDING_MAX_WORKERS_ADJUSTED:-}" ]]; then
        EMBEDDING_MAX_WORKERS=1
        EMBEDDING_MAX_WORKERS_ADJUSTED="true"
    fi
    
    if [[ "$EMBEDDING_MEMORY_LIMIT" -lt 10 ]] || [[ "$EMBEDDING_MEMORY_LIMIT" -gt 95 ]]; then
        EMBEDDING_MEMORY_LIMIT=80
    fi
    
    # ================================
    # COMPATIBILITY ALIASES
    # ================================
    # Maintain backward compatibility with existing code
    
    if [[ -z "${DEFAULT_MODEL:-}" ]]; then
        DEFAULT_MODEL="$QDRANT_EMBEDDING_MODEL"
    fi
    
    if [[ -z "${DEFAULT_BATCH_SIZE:-}" ]]; then
        DEFAULT_BATCH_SIZE="$EMBEDDING_BATCH_SIZE"
    fi
    
    if [[ -z "${BATCH_SIZE:-}" ]]; then
        BATCH_SIZE="$EMBEDDING_BATCH_SIZE"
    fi
    
    if [[ -z "${MAX_WORKERS:-}" ]]; then
        MAX_WORKERS="$EMBEDDING_MAX_WORKERS"
    fi
    
    if [[ -z "${CACHE_TTL:-}" ]]; then
        CACHE_TTL="$EMBEDDING_CACHE_TTL"
    fi
    
    # ================================
    # EXPORT ALL CONFIGURATION
    # ================================
    # Note: Variables are not made readonly to allow multiple sourcing
    
    # ================================
    # EXPORT ALL CONFIGURATION
    # ================================
    
    export QDRANT_EMBEDDING_MODEL QDRANT_EMBEDDING_DIMENSIONS QDRANT_EMBEDDING_DISTANCE_METRIC
    export EMBEDDING_MAX_WORKERS EMBEDDING_BATCH_SIZE EMBEDDING_MEMORY_LIMIT
    export EMBEDDING_CACHE_TTL EMBEDDING_CACHE_ENABLED EMBEDDING_CACHE_DIR
    export EMBEDDING_PROCESSING_TIMEOUT EMBEDDING_MAX_RETRIES EMBEDDING_RETRY_DELAY
    export EMBEDDING_WORKFLOWS_COLLECTION_SUFFIX EMBEDDING_SCENARIOS_COLLECTION_SUFFIX
    export EMBEDDING_KNOWLEDGE_COLLECTION_SUFFIX EMBEDDING_CODE_COLLECTION_SUFFIX EMBEDDING_RESOURCES_COLLECTION_SUFFIX
    export EMBEDDING_MIN_CONTENT_LENGTH EMBEDDING_MAX_CONTENT_LENGTH EMBEDDING_PROGRESS_INTERVAL
    
    # Export compatibility aliases
    export DEFAULT_MODEL DEFAULT_BATCH_SIZE BATCH_SIZE MAX_WORKERS CACHE_TTL
    
    return 0
}

#######################################
# Display current configuration
# Returns: Configuration summary
#######################################
qdrant::embeddings::show_config() {
    echo "=== Qdrant Embeddings Configuration ==="
    echo
    echo "📊 Model & Processing:"
    echo "  Embedding Model: $QDRANT_EMBEDDING_MODEL"
    echo "  Model Dimensions: $QDRANT_EMBEDDING_DIMENSIONS"
    echo "  Distance Metric: $QDRANT_EMBEDDING_DISTANCE_METRIC"
    echo "  Batch Size: $EMBEDDING_BATCH_SIZE"
    echo
    echo "⚡ Performance:"
    echo "  Max Workers: $EMBEDDING_MAX_WORKERS"
    echo "  Memory Limit: ${EMBEDDING_MEMORY_LIMIT}%"
    echo "  Processing Timeout: ${EMBEDDING_PROCESSING_TIMEOUT}s"
    echo "  Max Retries: $EMBEDDING_MAX_RETRIES"
    echo
    echo "💾 Caching:"
    echo "  Cache Enabled: $EMBEDDING_CACHE_ENABLED"
    echo "  Cache TTL: ${EMBEDDING_CACHE_TTL}s"
    echo "  Cache Directory: $EMBEDDING_CACHE_DIR"
    echo
    echo "📏 Content Limits:"
    echo "  Min Content Length: $EMBEDDING_MIN_CONTENT_LENGTH chars"
    echo "  Max Content Length: $EMBEDDING_MAX_CONTENT_LENGTH chars"
    echo
    echo "🔧 Override via environment variables:"
    echo "  export QDRANT_EMBEDDING_MODEL_OVERRIDE=\"nomic-embed-text\""
    echo "  export EMBEDDING_MAX_WORKERS_OVERRIDE=8"
    echo "  export EMBEDDING_MEMORY_LIMIT_OVERRIDE=70"
}

#######################################
# Validate configuration values
# Returns: 0 if valid, 1 if issues found
#######################################
qdrant::embeddings::validate_config() {
    local issues=0
    
    # Check model availability (if ollama is available)
    if command -v ollama >/dev/null 2>&1; then
        if ! ollama list | grep -q "$QDRANT_EMBEDDING_MODEL"; then
            echo "⚠️  Warning: Model $QDRANT_EMBEDDING_MODEL not found in Ollama"
            echo "   Run: ollama pull $QDRANT_EMBEDDING_MODEL"
            ((issues++))
        fi
    fi
    
    # Check cache directory permissions
    if [[ ! -d "$EMBEDDING_CACHE_DIR" ]]; then
        if ! mkdir -p "$EMBEDDING_CACHE_DIR" 2>/dev/null; then
            echo "❌ Error: Cannot create cache directory $EMBEDDING_CACHE_DIR"
            ((issues++))
        fi
    fi
    
    # Check numeric values
    if ! [[ "$EMBEDDING_MAX_WORKERS" =~ ^[0-9]+$ ]]; then
        echo "❌ Error: EMBEDDING_MAX_WORKERS must be a number, got: $EMBEDDING_MAX_WORKERS"
        ((issues++))
    fi
    
    if ! [[ "$EMBEDDING_BATCH_SIZE" =~ ^[0-9]+$ ]] || [[ "$EMBEDDING_BATCH_SIZE" -lt 1 ]]; then
        echo "❌ Error: EMBEDDING_BATCH_SIZE must be a positive number, got: $EMBEDDING_BATCH_SIZE"
        ((issues++))
    fi
    
    if [[ $issues -eq 0 ]]; then
        echo "✅ Configuration validation passed"
        return 0
    else
        echo "❌ Configuration validation failed with $issues issues"
        return 1
    fi
}

# Auto-export configuration when sourced (only if not already exported)
if [[ -z "${QDRANT_EMBEDDINGS_CONFIG_EXPORTED:-}" ]]; then
    qdrant::embeddings::export_config
fi