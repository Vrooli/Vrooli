#!/usr/bin/env bash
# Schema Validation for Qdrant Embeddings
# Validates embedding output format and structure

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Validate JSON structure of embedding output
# Arguments:
#   $1 - JSON line to validate
#   $2 - Expected content type
# Returns: 0 if valid, 1 if invalid
#######################################
validate_embedding_schema() {
    local json_line="$1"
    local expected_type="${2:-}"
    
    # Check if it's valid JSON
    if ! echo "$json_line" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON format: $json_line"
        return 1
    fi
    
    # Check required fields
    local required_fields=("content" "metadata")
    for field in "${required_fields[@]}"; do
        if ! echo "$json_line" | jq -e ".$field" >/dev/null 2>&1; then
            log::error "Missing required field: $field"
            return 1
        fi
    done
    
    # Validate metadata structure
    local metadata_fields=("content_type" "extractor")
    for field in "${metadata_fields[@]}"; do
        if ! echo "$json_line" | jq -e ".metadata.$field" >/dev/null 2>&1; then
            log::error "Missing required metadata field: $field"
            return 1
        fi
    done
    
    # Validate content type if specified
    if [[ -n "$expected_type" ]]; then
        local actual_type=$(echo "$json_line" | jq -r '.metadata.content_type')
        if [[ "$actual_type" != "$expected_type" ]]; then
            log::error "Content type mismatch. Expected: $expected_type, Got: $actual_type"
            return 1
        fi
    fi
    
    # Validate content is not empty
    local content=$(echo "$json_line" | jq -r '.content')
    if [[ -z "$content" || "$content" == "null" ]]; then
        log::error "Content field is empty or null"
        return 1
    fi
    
    log::debug "Schema validation passed for content type: $(echo "$json_line" | jq -r '.metadata.content_type')"
    return 0
}

#######################################
# Validate embedding vector format
# Arguments:
#   $1 - Embedding vector (JSON array)
#   $2 - Expected dimensions (optional)
# Returns: 0 if valid, 1 if invalid
#######################################
validate_embedding_vector() {
    local vector="$1"
    local expected_dims="${2:-1024}"
    
    # Check if it's a valid JSON array
    if ! echo "$vector" | jq -e 'type == "array"' >/dev/null 2>&1; then
        log::error "Embedding vector is not a valid JSON array"
        return 1
    fi
    
    # Check dimensions
    local actual_dims=$(echo "$vector" | jq 'length')
    if [[ "$actual_dims" != "$expected_dims" ]]; then
        log::error "Dimension mismatch. Expected: $expected_dims, Got: $actual_dims"
        return 1
    fi
    
    # Check that all values are numbers
    local non_numeric_count=$(echo "$vector" | jq '[.[] | select(type != "number")] | length')
    if [[ "$non_numeric_count" -gt 0 ]]; then
        log::error "Embedding vector contains $non_numeric_count non-numeric values"
        return 1
    fi
    
    # Check for reasonable value ranges (-1 to 1 for normalized embeddings)
    local out_of_range_count=$(echo "$vector" | jq '[.[] | select(. < -2 or . > 2)] | length')
    if [[ "$out_of_range_count" -gt 0 ]]; then
        log::warn "Embedding vector has $out_of_range_count values outside expected range [-2, 2]"
    fi
    
    log::debug "Embedding vector validation passed (${actual_dims}D)"
    return 0
}

#######################################
# Validate extractor output file
# Arguments:
#   $1 - Path to extractor output file
#   $2 - Expected content type
# Returns: Number of valid entries
#######################################
validate_extractor_output() {
    local output_file="$1"
    local expected_type="${2:-}"
    
    if [[ ! -f "$output_file" ]]; then
        log::error "Output file not found: $output_file"
        return 0
    fi
    
    local line_count=0
    local valid_count=0
    local error_count=0
    
    while IFS= read -r line; do
        ((line_count++))
        
        if [[ -z "$line" ]]; then
            continue  # Skip empty lines
        fi
        
        if validate_embedding_schema "$line" "$expected_type"; then
            ((valid_count++))
        else
            ((error_count++))
            log::warn "Line $line_count failed validation"
        fi
    done < "$output_file"
    
    log::info "Validation results for $output_file:"
    log::info "  Total lines: $line_count"
    log::info "  Valid entries: $valid_count"
    log::info "  Errors: $error_count"
    
    if [[ $error_count -gt 0 ]]; then
        log::warn "Validation found $error_count errors in $output_file"
    else
        log::success "All entries in $output_file passed validation"
    fi
    
    echo "$valid_count"
}

#######################################
# Validate Qdrant collection structure
# Arguments:
#   $1 - Collection name
# Returns: 0 if valid, 1 if invalid
#######################################
validate_collection_structure() {
    local collection="$1"
    
    # Check if collection exists
    if ! curl -s "http://localhost:6333/collections/$collection" >/dev/null 2>&1; then
        log::error "Collection $collection not accessible"
        return 1
    fi
    
    # Get collection info
    local collection_info=$(curl -s "http://localhost:6333/collections/$collection")
    
    # Validate collection configuration
    local vector_size=$(echo "$collection_info" | jq -r '.result.config.params.vectors.size // .result.config.params.vectors."".size')
    local distance=$(echo "$collection_info" | jq -r '.result.config.params.vectors.distance // .result.config.params.vectors."".distance')
    
    if [[ "$vector_size" != "1024" ]]; then
        log::warn "Unexpected vector size for $collection: $vector_size (expected 1024)"
    fi
    
    if [[ "$distance" != "Cosine" ]]; then
        log::warn "Unexpected distance metric for $collection: $distance (expected Cosine)"
    fi
    
    # Get point count
    local point_count=$(curl -s "http://localhost:6333/collections/$collection/points/count" | jq -r '.result.count // 0')
    
    log::info "Collection $collection validation:"
    log::info "  Vector size: $vector_size"
    log::info "  Distance metric: $distance"
    log::info "  Point count: $point_count"
    
    return 0
}

#######################################
# Main validation function
# Arguments:
#   $1 - Validation type (schema|output|collection|all)
#   $2+ - Additional arguments based on type
#######################################
main() {
    local validation_type="${1:-all}"
    shift || true
    
    case "$validation_type" in
        schema)
            local json_line="$1"
            local content_type="${2:-}"
            validate_embedding_schema "$json_line" "$content_type"
            ;;
        vector)
            local vector="$1"
            local dimensions="${2:-1024}"
            validate_embedding_vector "$vector" "$dimensions"
            ;;
        output)
            local output_file="$1"
            local content_type="${2:-}"
            validate_extractor_output "$output_file" "$content_type"
            ;;
        collection)
            local collection="$1"
            validate_collection_structure "$collection"
            ;;
        all)
            log::info "Running comprehensive validation..."
            
            # Check for common output files
            local temp_files=(
                "/tmp/qdrant-scenarios.jsonl"
                "/tmp/qdrant-docs.jsonl"
                "/tmp/qdrant-code.jsonl"
                "/tmp/qdrant-resources.jsonl"
                "/tmp/qdrant-filetrees.jsonl"
            )
            
            local types=("scenario" "documentation" "code" "resource" "file_tree")
            
            for i in "${!temp_files[@]}"; do
                local file="${temp_files[$i]}"
                local type="${types[$i]}"
                
                if [[ -f "$file" ]]; then
                    log::info "Validating $file..."
                    validate_extractor_output "$file" "$type"
                fi
            done
            
            # Validate common collections
            local collections=("workflows" "scenarios" "docs" "code" "resources" "filetrees")
            for collection in "${collections[@]}"; do
                if curl -s "http://localhost:6333/collections/$collection" >/dev/null 2>&1; then
                    log::info "Validating collection $collection..."
                    validate_collection_structure "$collection"
                fi
            done
            ;;
        *)
            log::error "Unknown validation type: $validation_type"
            log::info "Usage: $0 <schema|vector|output|collection|all> [args...]"
            return 1
            ;;
    esac
}

# Run main if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi