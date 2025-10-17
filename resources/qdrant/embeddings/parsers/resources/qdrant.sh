#!/usr/bin/env bash
# Qdrant Vector Database Parser for Qdrant Embeddings
# Extracts semantic information from Qdrant collection configuration files
#
# Handles:
# - Collection definitions and configurations
# - Vector dimensions and distance metrics
# - Index settings (HNSW parameters)
# - Payload schemas and field types
# - Optimization and quantization settings

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract collection metadata
# 
# Gets basic collection information
#
# Arguments:
#   $1 - Path to Qdrant collection config JSON file
# Returns: JSON with collection metadata
#######################################
extractor::lib::qdrant::extract_collection() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Validate JSON format
    if ! jq empty "$file" 2>/dev/null; then
        log::debug "Invalid JSON format in Qdrant config: $file" >&2
        return 1
    fi
    
    # Extract collection name and basic settings
    local collection_name=$(jq -r '.collection_name // .name // "unknown"' "$file" 2>/dev/null)
    local vector_size=$(jq -r '.vector_size // .vectors.size // 0' "$file" 2>/dev/null)
    local distance_metric=$(jq -r '.distance // .vectors.distance // "unknown"' "$file" 2>/dev/null)
    
    # Check if vectors config is more complex (named vectors)
    local has_named_vectors="false"
    if jq -e '.vectors | type == "object" and has("size") | not' "$file" >/dev/null 2>/dev/null; then
        has_named_vectors="true"
        # Count named vectors
        vector_size=$(jq '.vectors | keys | length' "$file" 2>/dev/null || echo "0")
    fi
    
    # Extract replication settings
    local replication_factor=$(jq -r '.replication_factor // 1' "$file" 2>/dev/null)
    local write_consistency_factor=$(jq -r '.write_consistency_factor // 1' "$file" 2>/dev/null)
    
    # Check for sharding
    local shard_number=$(jq -r '.shard_number // 1' "$file" 2>/dev/null)
    local on_disk_payload=$(jq -r '.on_disk_payload // false' "$file" 2>/dev/null)
    
    jq -n \
        --arg name "$collection_name" \
        --arg vector_size "$vector_size" \
        --arg distance "$distance_metric" \
        --arg named_vectors "$has_named_vectors" \
        --arg replication "$replication_factor" \
        --arg consistency "$write_consistency_factor" \
        --arg shards "$shard_number" \
        --arg disk_payload "$on_disk_payload" \
        '{
            collection_name: $name,
            vector_size: ($vector_size | tonumber),
            distance_metric: $distance,
            has_named_vectors: ($named_vectors == "true"),
            replication_factor: ($replication | tonumber),
            write_consistency_factor: ($consistency | tonumber),
            shard_number: ($shards | tonumber),
            on_disk_payload: ($disk_payload == "true")
        }'
}

#######################################
# Extract index configuration
# 
# Analyzes HNSW and other index settings
#
# Arguments:
#   $1 - Path to Qdrant collection config file
# Returns: JSON with index configuration
#######################################
extractor::lib::qdrant::extract_index_config() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract HNSW parameters
    local hnsw_config=$(jq '.hnsw_config // .vectors.hnsw_config // {}' "$file" 2>/dev/null)
    local m=$(echo "$hnsw_config" | jq -r '.m // 16')
    local ef_construct=$(echo "$hnsw_config" | jq -r '.ef_construct // 100')
    local full_scan_threshold=$(echo "$hnsw_config" | jq -r '.full_scan_threshold // 10000')
    local max_indexing_threads=$(echo "$hnsw_config" | jq -r '.max_indexing_threads // 0')
    
    # Extract optimizer settings
    local optimizer_config=$(jq '.optimizer_config // {}' "$file" 2>/dev/null)
    local deleted_threshold=$(echo "$optimizer_config" | jq -r '.deleted_threshold // 0.2')
    local vacuum_min_vector_number=$(echo "$optimizer_config" | jq -r '.vacuum_min_vector_number // 1000')
    local default_segment_number=$(echo "$optimizer_config" | jq -r '.default_segment_number // 0')
    
    # Extract quantization settings
    local quantization_config=$(jq '.quantization_config // {}' "$file" 2>/dev/null)
    local has_quantization="false"
    local quantization_type="none"
    if [[ "$quantization_config" != "{}" ]]; then
        has_quantization="true"
        quantization_type=$(echo "$quantization_config" | jq -r 'keys[0] // "unknown"')
    fi
    
    jq -n \
        --arg m "$m" \
        --arg ef_construct "$ef_construct" \
        --arg threshold "$full_scan_threshold" \
        --arg threads "$max_indexing_threads" \
        --arg deleted_thresh "$deleted_threshold" \
        --arg vacuum_min "$vacuum_min_vector_number" \
        --arg segments "$default_segment_number" \
        --arg has_quant "$has_quantization" \
        --arg quant_type "$quantization_type" \
        '{
            hnsw: {
                m: ($m | tonumber),
                ef_construct: ($ef_construct | tonumber),
                full_scan_threshold: ($threshold | tonumber),
                max_indexing_threads: ($threads | tonumber)
            },
            optimizer: {
                deleted_threshold: ($deleted_thresh | tonumber),
                vacuum_min_vector_number: ($vacuum_min | tonumber),
                default_segment_number: ($segments | tonumber)
            },
            quantization: {
                enabled: ($has_quant == "true"),
                type: $quant_type
            }
        }'
}

#######################################
# Extract payload schema
# 
# Analyzes payload field definitions and indexes
#
# Arguments:
#   $1 - Path to Qdrant collection config file
# Returns: JSON with payload schema information
#######################################
extractor::lib::qdrant::extract_payload_schema() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract payload indexes
    local payload_indexes=$(jq '.payload_indexes // {}' "$file" 2>/dev/null)
    local index_count=$(echo "$payload_indexes" | jq 'keys | length' 2>/dev/null || echo "0")
    
    # Extract field types from indexes
    local field_types=()
    if [[ $index_count -gt 0 ]]; then
        while IFS= read -r field; do
            [[ -z "$field" ]] && continue
            field_types+=("$field")
        done < <(echo "$payload_indexes" | jq -r 'to_entries[] | "\(.key):\(.value.data_type // .value)"')
    fi
    
    # Check for text indexes
    local has_text_index="false"
    if echo "$payload_indexes" | jq -e 'to_entries[] | select(.value.data_type == "text" or .value == "text")' >/dev/null 2>/dev/null; then
        has_text_index="true"
    fi
    
    # Check for geo indexes
    local has_geo_index="false"
    if echo "$payload_indexes" | jq -e 'to_entries[] | select(.value.data_type == "geo" or .value == "geo")' >/dev/null 2>/dev/null; then
        has_geo_index="true"
    fi
    
    # Extract schema validation if present
    local has_schema_validation="false"
    local schema_validation=$(jq '.schema // {}' "$file" 2>/dev/null)
    if [[ "$schema_validation" != "{}" ]]; then
        has_schema_validation="true"
    fi
    
    local field_types_json="[]"
    if [[ ${#field_types[@]} -gt 0 ]]; then
        field_types_json=$(printf '%s\n' "${field_types[@]}" | jq -R . | jq -s '.')
    fi
    
    jq -n \
        --arg index_count "$index_count" \
        --argjson field_types "$field_types_json" \
        --arg has_text "$has_text_index" \
        --arg has_geo "$has_geo_index" \
        --arg has_schema "$has_schema_validation" \
        --argjson schema "$schema_validation" \
        '{
            payload_index_count: ($index_count | tonumber),
            field_types: $field_types,
            has_text_index: ($has_text == "true"),
            has_geo_index: ($has_geo == "true"),
            has_schema_validation: ($has_schema == "true"),
            schema: $schema
        }'
}

#######################################
# Analyze collection purpose
# 
# Determines collection usage based on configuration
#
# Arguments:
#   $1 - Path to Qdrant collection config file
# Returns: JSON with purpose analysis
#######################################
extractor::lib::qdrant::analyze_purpose() {
    local file="$1"
    local purposes=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"purposes": [], "primary_purpose": "unknown"}'
        return
    fi
    
    local filename=$(basename "$file" | tr '[:upper:]' '[:lower:]')
    local content=$(cat "$file" 2>/dev/null | tr '[:upper:]' '[:lower:]')
    
    # Analyze filename for hints
    if [[ "$filename" == *"embedding"* ]] || [[ "$filename" == *"vector"* ]]; then
        purposes+=("vector_storage")
    elif [[ "$filename" == *"search"* ]]; then
        purposes+=("similarity_search")
    elif [[ "$filename" == *"recommendation"* ]]; then
        purposes+=("recommendation_system")
    elif [[ "$filename" == *"semantic"* ]]; then
        purposes+=("semantic_search")
    elif [[ "$filename" == *"knowledge"* ]]; then
        purposes+=("knowledge_base")
    elif [[ "$filename" == *"document"* ]] || [[ "$filename" == *"doc"* ]]; then
        purposes+=("document_storage")
    elif [[ "$filename" == *"image"* ]]; then
        purposes+=("image_search")
    elif [[ "$filename" == *"text"* ]]; then
        purposes+=("text_search")
    fi
    
    # Analyze configuration for usage patterns
    local vector_size=$(jq -r '.vector_size // .vectors.size // 0' "$file" 2>/dev/null)
    local distance=$(jq -r '.distance // .vectors.distance // ""' "$file" 2>/dev/null)
    
    # Infer purpose from vector dimensions (common embedding sizes)
    case "$vector_size" in
        384)
            purposes+=("sentence_embeddings")
            ;;
        512|768)
            purposes+=("bert_embeddings")
            ;;
        1536)
            purposes+=("openai_embeddings")
            ;;
        1024|2048)
            purposes+=("large_language_model")
            ;;
        300)
            purposes+=("word2vec_embeddings")
            ;;
    esac
    
    # Analyze distance metric
    case "$distance" in
        cosine|Cosine)
            purposes+=("text_similarity")
            ;;
        dot|Dot)
            purposes+=("content_matching")
            ;;
        euclid*|Euclidean)
            purposes+=("spatial_search")
            ;;
    esac
    
    # Check for text indexing (indicates NLP use case)
    if echo "$content" | grep -q '"text"'; then
        purposes+=("text_processing")
    fi
    
    # Check for geo indexing (indicates location-based search)
    if echo "$content" | grep -q '"geo"'; then
        purposes+=("geospatial_search")
    fi
    
    # Determine primary purpose
    local primary_purpose="vector_database"
    if [[ ${#purposes[@]} -gt 0 ]]; then
        primary_purpose="${purposes[0]}"
    fi
    
    local purposes_json=$(printf '%s\n' "${purposes[@]}" | sort -u | jq -R . | jq -s '.')
    
    jq -n \
        --argjson purposes "$purposes_json" \
        --arg primary "$primary_purpose" \
        '{
            purposes: $purposes,
            primary_purpose: $primary
        }'
}

#######################################
# Extract all Qdrant collection information
# 
# Main extraction function that combines all analyses
#
# Arguments:
#   $1 - Qdrant config file path or directory
#   $2 - Component type (collection, vector_db, etc.)
#   $3 - Resource name
# Returns: JSON lines with all collection information
#######################################
extractor::lib::qdrant::extract_all() {
    local path="$1"
    local component_type="${2:-collection}"
    local resource_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        local file="$path"
        local filename=$(basename "$file")
        local file_ext="${filename##*.}"
        
        # Check if it's a supported file type
        case "$file_ext" in
            json)
                ;;
            *)
                return 1
                ;;
        esac
        
        # Get file statistics
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        
        # Extract all components
        local collection=$(extractor::lib::qdrant::extract_collection "$file")
        local index_config=$(extractor::lib::qdrant::extract_index_config "$file")
        local payload_schema=$(extractor::lib::qdrant::extract_payload_schema "$file")
        local purpose=$(extractor::lib::qdrant::analyze_purpose "$file")
        
        # Get key metrics
        local collection_name=$(echo "$collection" | jq -r '.collection_name')
        local vector_size=$(echo "$collection" | jq -r '.vector_size')
        local distance_metric=$(echo "$collection" | jq -r '.distance_metric')
        local primary_purpose=$(echo "$purpose" | jq -r '.primary_purpose')
        local has_named_vectors=$(echo "$collection" | jq -r '.has_named_vectors')
        
        # Build content summary
        local content="Qdrant Collection: $collection_name | Type: $component_type | Resource: $resource_name"
        content="$content | Purpose: $primary_purpose"
        
        if [[ "$has_named_vectors" == "true" ]]; then
            content="$content | Named Vectors: $vector_size"
        else
            content="$content | Vector Size: $vector_size"
        fi
        
        [[ "$distance_metric" != "unknown" ]] && content="$content | Distance: $distance_metric"
        
        # Check for advanced features
        local index_count=$(echo "$payload_schema" | jq -r '.payload_index_count')
        [[ $index_count -gt 0 ]] && content="$content | Payload Indexes: $index_count"
        
        local has_quantization=$(echo "$index_config" | jq -r '.quantization.enabled')
        [[ "$has_quantization" == "true" ]] && content="$content | Has Quantization"
        
        local replication=$(echo "$collection" | jq -r '.replication_factor')
        [[ $replication -gt 1 ]] && content="$content | Replicated: ${replication}x"
        
        # Output comprehensive collection analysis
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_file "$file" \
            --arg filename "$filename" \
            --arg component_type "$component_type" \
            --arg file_size "$file_size" \
            --argjson collection "$collection" \
            --argjson index_config "$index_config" \
            --argjson payload_schema "$payload_schema" \
            --argjson purpose "$purpose" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_file: $source_file,
                    filename: $filename,
                    component_type: $component_type,
                    database_type: "qdrant",
                    file_size: ($file_size | tonumber),
                    collection: $collection,
                    index_config: $index_config,
                    payload_schema: $payload_schema,
                    purpose: $purpose,
                    content_type: "qdrant_collection",
                    extraction_method: "qdrant_parser"
                }
            }' | jq -c
            
        # Output entry for vector configuration (for better searchability)
        if [[ "$has_named_vectors" == "true" ]]; then
            local vector_names=$(jq -r '.vectors | keys | join(", ")' "$file" 2>/dev/null || echo "")
            if [[ -n "$vector_names" && "$vector_names" != "" ]]; then
                local vectors_content="Qdrant Named Vectors: $vector_names | Collection: $collection_name | Resource: $resource_name"
                
                jq -n \
                    --arg content "$vectors_content" \
                    --arg resource "$resource_name" \
                    --arg source_file "$file" \
                    --arg collection_name "$collection_name" \
                    --arg vector_names "$vector_names" \
                    --arg component_type "$component_type" \
                    '{
                        content: $content,
                        metadata: {
                            resource: $resource,
                            source_file: $source_file,
                            collection_name: $collection_name,
                            vector_names: $vector_names,
                            component_type: $component_type,
                            content_type: "qdrant_vectors",
                            extraction_method: "qdrant_parser"
                        }
                    }' | jq -c
            fi
        fi
        
    elif [[ -d "$path" ]]; then
        # Directory - find all Qdrant config files
        local config_files=()
        while IFS= read -r file; do
            config_files+=("$file")
        done < <(find "$path" -type f -name "*.json" 2>/dev/null)
        
        if [[ ${#config_files[@]} -eq 0 ]]; then
            return 1
        fi
        
        for file in "${config_files[@]}"; do
            # Basic validation - check if it looks like a Qdrant config
            if jq -e '.collection_name or .vectors' "$file" >/dev/null 2>/dev/null; then
                extractor::lib::qdrant::extract_all "$file" "$component_type" "$resource_name"
            fi
        done
    fi
}

#######################################
# Check if file is a Qdrant collection config
# 
# Validates if JSON file is a Qdrant collection definition
#
# Arguments:
#   $1 - File path
# Returns: 0 if Qdrant config, 1 otherwise
#######################################
extractor::lib::qdrant::is_collection_config() {
    local file="$1"
    
    if [[ ! -f "$file" ]] || [[ ! "$file" == *.json ]]; then
        return 1
    fi
    
    # Check for Qdrant collection structure
    if jq -e '.collection_name or .vectors' "$file" >/dev/null 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Export all functions
export -f extractor::lib::qdrant::extract_collection
export -f extractor::lib::qdrant::extract_index_config
export -f extractor::lib::qdrant::extract_payload_schema
export -f extractor::lib::qdrant::analyze_purpose
export -f extractor::lib::qdrant::extract_all
export -f extractor::lib::qdrant::is_collection_config