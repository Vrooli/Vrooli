#!/bin/bash
# Semantic Search for Legal Clauses using Qdrant
# Version: 1.0.0
# Description: Provides semantic search capabilities for finding relevant legal clauses

set -euo pipefail

# Check if Qdrant is available
is_qdrant_available() {
    if command -v resource-qdrant &>/dev/null; then
        if resource-qdrant status &>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Initialize Qdrant collection for legal clauses
initialize_legal_clauses_collection() {
    local collection_name="legal_clauses"

    if ! is_qdrant_available; then
        echo "Qdrant is not available. Skipping semantic search initialization." >&2
        return 1
    fi

    # Check if collection already exists
    if resource-qdrant content list-collections 2>/dev/null | grep -q "$collection_name"; then
        echo "Collection '$collection_name' already exists"
        return 0
    fi

    # Create collection with appropriate vector size (384 for all-MiniLM-L6-v2)
    resource-qdrant content create-collection \
        --name "$collection_name" \
        --vector-size 384 \
        --distance "cosine" 2>/dev/null || {
        echo "Failed to create Qdrant collection" >&2
        return 1
    }

    echo "Created Qdrant collection: $collection_name"
    return 0
}

# Index a legal clause in Qdrant
index_legal_clause() {
    local clause_id="$1"
    local clause_text="$2"
    local clause_type="$3"
    local jurisdiction="${4:-}"
    local tags="${5:-}"

    if ! is_qdrant_available; then
        return 1
    fi

    # Generate embedding using Ollama (via resource-ollama)
    # Note: This assumes ollama has an embedding model available
    local embedding=$(echo "$clause_text" | resource-ollama content add \
        --model "llama3.2" \
        --query "Generate embedding" \
        --embedding 2>/dev/null || echo "")

    if [ -z "$embedding" ]; then
        echo "Failed to generate embedding for clause $clause_id" >&2
        return 1
    fi

    # Prepare payload
    local payload=$(cat <<EOF
{
    "clause_id": "$clause_id",
    "clause_type": "$clause_type",
    "jurisdiction": "$jurisdiction",
    "tags": "$tags",
    "text": "$clause_text"
}
EOF
)

    # Index in Qdrant
    resource-qdrant content add \
        --collection "legal_clauses" \
        --id "$clause_id" \
        --vector "$embedding" \
        --payload "$payload" 2>/dev/null || {
        echo "Failed to index clause in Qdrant" >&2
        return 1
    }

    return 0
}

# Search for relevant legal clauses
search_legal_clauses() {
    local query_text="$1"
    local limit="${2:-5}"
    local clause_type_filter="${3:-}"
    local jurisdiction_filter="${4:-}"

    if ! is_qdrant_available; then
        echo "Qdrant not available. Using fallback PostgreSQL search." >&2
        return 1
    fi

    # Generate embedding for query
    local query_embedding=$(echo "$query_text" | resource-ollama content add \
        --model "llama3.2" \
        --query "Generate embedding" \
        --embedding 2>/dev/null || echo "")

    if [ -z "$query_embedding" ]; then
        echo "Failed to generate query embedding" >&2
        return 1
    fi

    # Build filter conditions
    local filter=""
    if [ -n "$clause_type_filter" ]; then
        filter="clause_type:$clause_type_filter"
    fi
    if [ -n "$jurisdiction_filter" ]; then
        [ -n "$filter" ] && filter="$filter,"
        filter="${filter}jurisdiction:$jurisdiction_filter"
    fi

    # Search in Qdrant
    local results=$(resource-qdrant content search \
        --collection "legal_clauses" \
        --vector "$query_embedding" \
        --limit "$limit" \
        ${filter:+--filter "$filter"} 2>/dev/null || echo "")

    if [ -z "$results" ]; then
        echo "No results found" >&2
        return 1
    fi

    echo "$results"
    return 0
}

# Fallback PostgreSQL full-text search
fallback_text_search() {
    local query_text="$1"
    local limit="${2:-5}"
    local clause_type_filter="${3:-}"
    local jurisdiction_filter="${4:-}"

    # Source database query function
    source "$(dirname "$0")/generator.sh"

    local query="SELECT id, clause_type, jurisdiction, content,
                        ts_rank(to_tsvector('english', content),
                                plainto_tsquery('english', '${query_text}')) as rank
                 FROM template_clauses
                 WHERE to_tsvector('english', content) @@ plainto_tsquery('english', '${query_text}')"

    [ -n "$clause_type_filter" ] && query="${query} AND clause_type = '${clause_type_filter}'"
    [ -n "$jurisdiction_filter" ] && query="${query} AND jurisdiction = '${jurisdiction_filter}'"

    query="${query} ORDER BY rank DESC LIMIT ${limit}"

    db_query "$query"
}

# Sync database clauses to Qdrant
sync_clauses_to_qdrant() {
    if ! is_qdrant_available; then
        echo "Qdrant not available. Cannot sync clauses." >&2
        return 1
    fi

    # Initialize collection
    initialize_legal_clauses_collection

    # Source database query function
    source "$(dirname "$0")/generator.sh"

    # Fetch all clauses from database
    local clauses=$(db_query "SELECT id, clause_type, jurisdiction, content, tags
                               FROM template_clauses
                               ORDER BY usage_count DESC")

    if [ -z "$clauses" ]; then
        echo "No clauses found in database" >&2
        return 1
    fi

    local count=0
    # Index each clause
    while IFS='|' read -r id clause_type jurisdiction content tags; do
        # Clean up whitespace
        id=$(echo "$id" | xargs)
        clause_type=$(echo "$clause_type" | xargs)
        jurisdiction=$(echo "$jurisdiction" | xargs)
        content=$(echo "$content" | xargs)
        tags=$(echo "$tags" | xargs)

        if index_legal_clause "$id" "$content" "$clause_type" "$jurisdiction" "$tags"; then
            count=$((count + 1))
            [ $((count % 10)) -eq 0 ] && echo "Indexed $count clauses..."
        fi
    done <<< "$clauses"

    echo "Successfully synced $count clauses to Qdrant"
    return 0
}

# Smart search that tries Qdrant first, falls back to PostgreSQL
smart_search_clauses() {
    local query_text="$1"
    local limit="${2:-5}"
    local clause_type_filter="${3:-}"
    local jurisdiction_filter="${4:-}"

    # Try Qdrant semantic search first
    if is_qdrant_available; then
        local results=$(search_legal_clauses "$query_text" "$limit" "$clause_type_filter" "$jurisdiction_filter" 2>/dev/null)
        if [ -n "$results" ]; then
            echo "$results"
            return 0
        fi
    fi

    # Fallback to PostgreSQL full-text search
    fallback_text_search "$query_text" "$limit" "$clause_type_filter" "$jurisdiction_filter"
}

# Export functions
export -f is_qdrant_available
export -f initialize_legal_clauses_collection
export -f index_legal_clause
export -f search_legal_clauses
export -f fallback_text_search
export -f sync_clauses_to_qdrant
export -f smart_search_clauses

# If script is run directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    case "${1:-}" in
        init)
            initialize_legal_clauses_collection
            ;;
        sync)
            sync_clauses_to_qdrant
            ;;
        search)
            shift
            smart_search_clauses "$@"
            ;;
        *)
            echo "Usage: $0 {init|sync|search} [options]"
            echo ""
            echo "Commands:"
            echo "  init                Initialize Qdrant collection"
            echo "  sync                Sync database clauses to Qdrant"
            echo "  search <query>      Search for relevant clauses"
            exit 1
            ;;
    esac
fi
