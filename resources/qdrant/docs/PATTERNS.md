# Patterns

## Collection Design Patterns

### Namespaced Collections
**Category:** Structural
**Purpose:** Isolate data by application or tenant
**When to Use:** Multi-app or multi-tenant systems
**When NOT to Use:** Single application with shared data

#### Implementation
```bash
# Collection naming convention
{app_id}-{content_type}

# Examples
user-portal-v1-embeddings
analytics-dashboard-metrics
payment-processor-transactions
```

#### Benefits
- Complete data isolation
- Independent scaling per app
- Simplified access control
- Easy cleanup/deletion

### Content Type Separation
**Category:** Structural
**Purpose:** Optimize search by content type
**When to Use:** Mixed content types with different search patterns
**When NOT to Use:** Homogeneous data

#### Implementation
```bash
# Separate collections by type
app-workflows    # N8n workflows
app-code        # Source code
app-knowledge   # Documentation
app-resources   # Resource configs
```

#### Benefits
- Type-specific optimization
- Faster filtered searches
- Independent index tuning
- Clear data organization

## Indexing Patterns

### Batch Insertion
**Category:** Performance
**Purpose:** Efficient bulk data loading
**When to Use:** Initial load or large updates
**When NOT to Use:** Real-time single updates

#### Implementation
```bash
# Batch upsert with error handling
qdrant::collections::batch_upsert() {
    local collection="$1"
    local batch_size=100
    local points=()
    
    while read -r line; do
        points+=("$line")
        if [[ ${#points[@]} -ge $batch_size ]]; then
            curl -X PUT "http://localhost:6333/collections/$collection/points/batch" \
                -H 'Content-Type: application/json' \
                -d "$(jq -n --argjson points "$(printf '%s\n' "${points[@]}" | jq -s .)" \
                    '{points: $points}')"
            points=()
        fi
    done
}
```

#### Benefits
- 10x faster than individual inserts
- Reduced network overhead
- Atomic batch operations
- Better resource utilization

### Incremental Updates
**Category:** Performance
**Purpose:** Update only changed content
**When to Use:** Regular refreshes
**When NOT to Use:** Complete rebuilds

#### Implementation
```bash
# Track changes with git
last_commit=$(jq -r '.last_indexed_commit' .vrooli/app-identity.json)
current_commit=$(git rev-parse HEAD)

if [[ "$last_commit" != "$current_commit" ]]; then
    # Extract only changed files
    changed_files=$(git diff --name-only "$last_commit" "$current_commit")
    # Index only changed content
fi
```

## Search Patterns

### Semantic Search with Filtering
**Category:** Behavioral
**Purpose:** Combine vector and metadata search
**When to Use:** Precise semantic search
**When NOT to Use:** Simple keyword matching

#### Implementation
```json
{
  "vector": [0.1, 0.2, ...],
  "filter": {
    "must": [
      {"key": "type", "match": {"value": "workflow"}},
      {"key": "app", "match": {"value": "user-portal"}}
    ]
  },
  "limit": 10,
  "with_payload": true
}
```

#### Benefits
- Precise results
- Reduced search space
- Better performance
- Contextual relevance

### Multi-Collection Search
**Category:** Behavioral
**Purpose:** Search across multiple collections
**When to Use:** Cross-app discovery
**When NOT to Use:** Single app queries

#### Implementation
```bash
# Parallel search across collections
search_all_collections() {
    local query="$1"
    local collections=($(list_collections))
    
    for collection in "${collections[@]}"; do
        (search_collection "$collection" "$query") &
    done
    wait
    
    # Merge and rank results
}
```

## Integration Patterns

### Embedding Pipeline
**Category:** Integration
**Purpose:** Generate and store embeddings
**Systems:** Ollama → Qdrant
**Direction:** Unidirectional

#### Architecture
```
Content → Ollama (embed) → Vectors → Qdrant (store)
```

#### Implementation
```bash
# Generate embeddings with Ollama
generate_embedding() {
    local text="$1"
    curl -X POST http://localhost:11434/api/embed \
        -d "{\"model\": \"mxbai-embed-large\", \"input\": [\"$text\"]}"
}

# Store in Qdrant
store_vector() {
    local collection="$1"
    local vector="$2"
    local payload="$3"
    
    curl -X PUT "http://localhost:6333/collections/$collection/points" \
        -d "{\"points\": [{\"id\": \"$(uuidgen)\", \"vector\": $vector, \"payload\": $payload}]}"
}
```

### RAG Pattern
**Category:** Integration
**Purpose:** Retrieval-augmented generation
**Systems:** Qdrant → LLM
**Direction:** Bidirectional

#### Implementation
```bash
# Retrieve context from Qdrant
retrieve_context() {
    local query="$1"
    local embedding=$(generate_embedding "$query")
    local results=$(search_vector "$embedding")
    echo "$results" | jq -r '.result[].payload.content'
}

# Augment LLM prompt
generate_response() {
    local query="$1"
    local context=$(retrieve_context "$query")
    local prompt="Context: $context\n\nQuestion: $query\n\nAnswer:"
    call_llm "$prompt"
}
```

## Performance Patterns

### Index Optimization
**Category:** Performance
**Purpose:** Optimize search speed vs accuracy
**When to Use:** Large datasets
**When NOT to Use:** Small collections (<10k vectors)

#### Implementation
```json
{
  "vectors": {
    "size": 1536,
    "distance": "Cosine"
  },
  "optimizers_config": {
    "default_segment_number": 4,
    "indexing_threshold": 20000
  },
  "hnsw_config": {
    "m": 16,
    "ef_construct": 100,
    "full_scan_threshold": 10000
  }
}
```

### Caching Strategy
**Category:** Performance
**Purpose:** Reduce repeated computations
**When to Use:** Frequent identical queries
**When NOT to Use:** Unique queries

#### Implementation
```bash
# Redis cache for search results
cache_search() {
    local query="$1"
    local cache_key="qdrant:search:$(echo -n "$query" | md5sum | cut -d' ' -f1)"
    
    # Check cache
    cached=$(redis-cli GET "$cache_key")
    if [[ -n "$cached" ]]; then
        echo "$cached"
        return 0
    fi
    
    # Perform search
    result=$(search_qdrant "$query")
    
    # Cache result (5 minute TTL)
    redis-cli SETEX "$cache_key" 300 "$result"
    echo "$result"
}
```

## Anti-Patterns to Avoid

### Unbounded Collection Growth
**Why It's Problematic:** Memory exhaustion, slow searches
**How to Recognize:** Ever-growing collection size
**Common Causes:** No deletion strategy

#### Bad Example
```bash
# DON'T: Never delete old vectors
while true; do
    add_vectors_to_collection
    # No cleanup
done
```

#### Good Alternative
```bash
# DO: Implement retention policy
add_with_ttl() {
    local vector="$1"
    local ttl_days=30
    local expiry=$(date -d "+$ttl_days days" +%s)
    
    # Add with expiry metadata
    add_vector "$vector" "{\"expiry\": $expiry}"
    
    # Regular cleanup job
    cleanup_expired_vectors
}
```

### Single Large Collection
**Why It's Problematic:** Poor search performance
**How to Recognize:** Slow queries, high memory usage
**Common Causes:** Simplistic design

#### Bad Example
```bash
# DON'T: Everything in one collection
all_embeddings_collection
```

#### Good Alternative
```bash
# DO: Logical separation
app1-workflows
app1-code
app2-workflows
app2-code
```

### Synchronous Embedding Generation
**Why It's Problematic:** Blocks user operations
**How to Recognize:** Slow response times
**Common Causes:** Direct API calls

#### Bad Example
```bash
# DON'T: Block on embedding generation
handle_request() {
    embedding=$(generate_embedding "$text")  # Blocks
    store_vector "$embedding"
    respond_to_user
}
```

#### Good Alternative
```bash
# DO: Queue for async processing
handle_request() {
    # Queue for processing
    redis-cli LPUSH embedding_queue "$text"
    
    # Return immediately
    respond_with_job_id
}

# Background worker
process_embeddings() {
    while true; do
        text=$(redis-cli BRPOP embedding_queue 0)
        embedding=$(generate_embedding "$text")
        store_vector "$embedding"
    done
}
```