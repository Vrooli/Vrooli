#!/usr/bin/env bash
# Enhanced Content Optimization for Better Embeddings
# Provides semantic chunking, content enhancement, and preprocessing

set -euo pipefail

# Get directory of this script
CONTENT_OPTIMIZER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EMBEDDINGS_DIR="$(dirname "$CONTENT_OPTIMIZER_DIR")"

# Source required utilities
source "${EMBEDDINGS_DIR}/../../../../lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Content optimization settings
MAX_CHUNK_SIZE=${QDRANT_MAX_CHUNK_SIZE:-1500}  # Characters per chunk
CHUNK_OVERLAP=${QDRANT_CHUNK_OVERLAP:-150}     # Overlap between chunks
MIN_CHUNK_SIZE=${QDRANT_MIN_CHUNK_SIZE:-200}   # Minimum viable chunk size

#######################################
# Optimize content for better semantic understanding
# Arguments:
#   $1 - Raw content
#   $2 - Content type (code/documentation/scenario/resource)
#   $3 - File path (for context)
# Outputs: Enhanced content optimized for embeddings
# Returns: 0 on success
#######################################
content::optimize() {
    local raw_content="$1"
    local content_type="$2"
    local file_path="${3:-unknown}"
    
    if [[ -z "$raw_content" ]]; then
        log::debug "Empty content provided for optimization"
        return 1
    fi
    
    local optimized_content=""
    
    case "$content_type" in
        code)
            optimized_content=$(content::optimize_code "$raw_content" "$file_path")
            ;;
        documentation)
            optimized_content=$(content::optimize_docs "$raw_content" "$file_path")
            ;;
        scenario)
            optimized_content=$(content::optimize_scenario "$raw_content" "$file_path")
            ;;
        resource)
            optimized_content=$(content::optimize_resource "$raw_content" "$file_path")
            ;;
        *)
            # Generic optimization
            optimized_content=$(content::optimize_generic "$raw_content" "$file_path")
            ;;
    esac
    
    echo "$optimized_content"
    return 0
}

#######################################
# Optimize code content for better semantic search
# Arguments:
#   $1 - Raw code content
#   $2 - File path
# Returns: Optimized content
#######################################
content::optimize_code() {
    local raw_content="$1"
    local file_path="$2"
    
    # Add semantic context
    echo "CODE_FILE: $(basename "$file_path")"
    
    # Extract and prioritize important patterns
    echo "CONTEXT: This code file contains implementations and definitions for:"
    
    # Look for function/class names that indicate purpose
    echo "$raw_content" | grep -E "(function|def |class |export)" | head -3 | \
        sed 's/.*function /- Function: /' | \
        sed 's/.*def /- Method: /' | \
        sed 's/.*class /- Class: /' | \
        sed 's/.*export /- Export: /' | \
        cut -c1-100
    
    echo
    echo "CONTENT:"
    
    # Clean up the raw content but preserve structure
    echo "$raw_content" | \
        sed 's/^[[:space:]]*$//' | \        # Remove empty lines
        sed 's/^[[:space:]]*//' | \         # Remove leading whitespace
        head -c $MAX_CHUNK_SIZE             # Limit size
}

#######################################
# Optimize documentation content
# Arguments:
#   $1 - Raw documentation content
#   $2 - File path
# Returns: Optimized content
#######################################
content::optimize_docs() {
    local raw_content="$1"
    local file_path="$2"
    
    echo "DOCUMENTATION: $(basename "$file_path")"
    echo "PURPOSE: This documentation covers:"
    
    # Extract headings for context
    echo "$raw_content" | grep -E "^#+ " | head -3 | sed 's/^#* */- /'
    
    echo
    echo "CONTENT:"
    
    # Preserve important structure while cleaning
    echo "$raw_content" | \
        sed '/^[[:space:]]*$/d' | \         # Remove empty lines
        head -c $MAX_CHUNK_SIZE
}

#######################################
# Optimize scenario content
# Arguments:
#   $1 - Raw scenario content  
#   $2 - File path
# Returns: Optimized content
#######################################
content::optimize_scenario() {
    local raw_content="$1"
    local file_path="$2"
    
    local scenario_name=$(basename "$(dirname "$file_path")")
    
    echo "SCENARIO: $scenario_name"
    echo "BUSINESS_CASE: This scenario implements:"
    
    # Extract key business value
    echo "$raw_content" | \
        grep -E "(Target Users|Value Proposition|Key Features)" -A 2 | \
        grep -v "^--$" | \
        head -5
    
    echo
    echo "DETAILS:"
    echo "$raw_content" | head -c $MAX_CHUNK_SIZE
}

#######################################
# Optimize resource content
# Arguments:
#   $1 - Raw resource content
#   $2 - File path
# Returns: Optimized content
#######################################
content::optimize_resource() {
    local raw_content="$1" 
    local file_path="$2"
    
    echo "RESOURCE: $(basename "$file_path")"
    
    # Look for API endpoints, configuration keys, etc.
    local api_endpoints=$(echo "$raw_content" | grep -E "(/api/|endpoint|GET|POST|PUT|DELETE)" | head -3)
    local config_keys=$(echo "$raw_content" | grep -E "([A-Z_]+:|[a-zA-Z_]+:)" | head -3)
    
    if [[ -n "$api_endpoints" ]]; then
        echo "API_ENDPOINTS:"
        echo "$api_endpoints" | sed 's/^/- /'
    fi
    
    if [[ -n "$config_keys" ]]; then
        echo "CONFIGURATION:"
        echo "$config_keys" | sed 's/^/- /'
    fi
    
    echo
    echo "CONTENT:"
    echo "$raw_content" | head -c $MAX_CHUNK_SIZE
}

#######################################
# Generic content optimization
# Arguments:
#   $1 - Raw content
#   $2 - File path
# Returns: Optimized content
#######################################
content::optimize_generic() {
    local raw_content="$1"
    local file_path="$2"
    
    echo "FILE: $(basename "$file_path")"
    echo "CONTENT:"
    
    # Basic cleanup
    echo "$raw_content" | \
        sed '/^[[:space:]]*$/d' | \
        head -c $MAX_CHUNK_SIZE
}

#######################################
# Add semantic tags to payload
# Arguments:
#   $1 - Optimized content
#   $2 - Content type
#   $3 - File path
# Outputs: JSON payload with enhanced metadata
# Returns: 0 on success
#######################################
content::add_semantic_tags() {
    local content="$1"
    local content_type="$2"
    local file_path="$3"
    
    local filename=$(basename "$file_path")
    local directory=$(dirname "$file_path")
    local file_ext="${filename##*.}"
    
    # Detect semantic categories
    local categories=()
    
    # Content-based categorization
    if echo "$content" | grep -q -i "error\|exception\|try\|catch\|failure"; then
        categories+=("error_handling")
    fi
    
    if echo "$content" | grep -q -i "async\|await\|promise\|parallel\|concurrent"; then
        categories+=("async_programming")  
    fi
    
    if echo "$content" | grep -q -i "database\|sql\|query\|connection"; then
        categories+=("database")
    fi
    
    if echo "$content" | grep -q -i "api\|endpoint\|http\|request\|response"; then
        categories+=("api")
    fi
    
    if echo "$content" | grep -q -i "auth\|login\|password\|token\|security"; then
        categories+=("authentication")
    fi
    
    if echo "$content" | grep -q -i "file\|directory\|path\|filesystem"; then
        categories+=("file_operations")
    fi
    
    if echo "$content" | grep -q -i "test\|spec\|assert\|mock"; then
        categories+=("testing")
    fi
    
    if echo "$content" | grep -q -i "config\|setting\|environment\|env"; then
        categories+=("configuration")
    fi
    
    # Build enhanced payload
    local categories_json
    categories_json=$(printf '%s\n' "${categories[@]}" | jq -R . | jq -s .)
    
    jq -n \
        --arg content "$content" \
        --arg type "$content_type" \
        --arg filename "$filename" \
        --arg directory "$directory" \
        --arg file_ext "$file_ext" \
        --arg file_path "$file_path" \
        --argjson categories "$categories_json" \
        '{
            content: $content,
            type: $type,
            filename: $filename,
            directory: $directory,
            file_extension: $file_ext,
            file_path: $file_path,
            semantic_categories: $categories,
            content_length: ($content | length),
            optimized: true
        }'
}

#######################################
# Split content into semantic chunks if too large
# Arguments:
#   $1 - Content
#   $2 - Content type
#   $3 - File path
# Outputs: Array of optimized chunks
# Returns: 0 on success  
#######################################
content::smart_chunk() {
    local content="$1"
    local content_type="$2"
    local file_path="$3"
    
    local content_length=${#content}
    
    if [[ $content_length -le $MAX_CHUNK_SIZE ]]; then
        # Content is small enough, return as single chunk
        content::add_semantic_tags "$content" "$content_type" "$file_path"
        return 0
    fi
    
    log::debug "Content too large ($content_length chars), splitting into chunks"
    
    # Split content intelligently based on type
    case "$content_type" in
        code)
            content::chunk_code "$content" "$file_path"
            ;;
        documentation)
            content::chunk_docs "$content" "$file_path"
            ;;
        *)
            content::chunk_generic "$content" "$content_type" "$file_path"
            ;;
    esac
}

#######################################
# Chunk code content at natural boundaries
#######################################
content::chunk_code() {
    local content="$1"
    local file_path="$2"
    
    # Try to split at function/class boundaries
    echo "$content" | awk -v chunk_size="$MAX_CHUNK_SIZE" -v file_path="$file_path" '
    BEGIN { current_chunk = ""; chunk_num = 0 }
    /^(function|def |class |export)/ { 
        if (length(current_chunk) > chunk_size) {
            print current_chunk
            current_chunk = ""
            chunk_num++
        }
    }
    { current_chunk = current_chunk $0 "\n" }
    END { 
        if (length(current_chunk) > 0) {
            print current_chunk
        }
    }' | while read -r chunk; do
        if [[ ${#chunk} -ge $MIN_CHUNK_SIZE ]]; then
            content::add_semantic_tags "$chunk" "code" "$file_path"
        fi
    done
}

#######################################
# Chunk documentation at section boundaries
#######################################
content::chunk_docs() {
    local content="$1"
    local file_path="$2"
    
    # Split at header boundaries
    echo "$content" | awk -v chunk_size="$MAX_CHUNK_SIZE" -v file_path="$file_path" '
    BEGIN { current_chunk = ""; chunk_num = 0 }
    /^#+/ {
        if (length(current_chunk) > chunk_size) {
            print current_chunk
            current_chunk = ""
            chunk_num++
        }
    }
    { current_chunk = current_chunk $0 "\n" }
    END {
        if (length(current_chunk) > 0) {
            print current_chunk
        }
    }' | while read -r chunk; do
        if [[ ${#chunk} -ge $MIN_CHUNK_SIZE ]]; then
            content::add_semantic_tags "$chunk" "documentation" "$file_path" 
        fi
    done
}

#######################################
# Generic content chunking by size
#######################################
content::chunk_generic() {
    local content="$1"
    local content_type="$2"
    local file_path="$3"
    
    # Simple size-based chunking with overlap
    echo "$content" | fold -w $MAX_CHUNK_SIZE | while read -r chunk; do
        if [[ ${#chunk} -ge $MIN_CHUNK_SIZE ]]; then
            content::add_semantic_tags "$chunk" "$content_type" "$file_path"
        fi
    done
}