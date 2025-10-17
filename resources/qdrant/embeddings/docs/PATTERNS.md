# Patterns

## Extraction Patterns

### Semantic Chunking Pattern
**Category:** Structural
**Purpose:** Break content into meaningful segments
**When to Use:** Processing long documents
**When NOT to Use:** Short, atomic content

#### Implementation
```bash
# Use embedding markers for semantic boundaries
chunk_with_markers() {
    local content="$1"
    
    # Extract marked sections
    awk '/<!-- EMBED:.*:START -->/,/<!-- EMBED:.*:END -->/' "$content"
    
    # Fall back to paragraph chunking
    split_by_paragraphs "$content" 1000  # 1000 char chunks
}
```

#### Benefits
- Preserves semantic context
- Better search relevance
- Consistent chunk sizes
- Maintains relationships

### Incremental Update Pattern
**Category:** Performance
**Purpose:** Update only changed content
**When to Use:** Regular refresh cycles
**When NOT to Use:** Initial indexing

#### Implementation
```bash
# Track changes with git
detect_changes() {
    local last_commit=$(jq -r '.last_indexed_commit' .vrooli/app-identity.json)
    local current_commit=$(git rev-parse HEAD)
    
    if [[ "$last_commit" != "$current_commit" ]]; then
        git diff --name-only "$last_commit" "$current_commit"
    fi
}

# Process only changed files
refresh_incremental() {
    local changed_files=$(detect_changes)
    for file in $changed_files; do
        extract_and_embed "$file"
    done
}
```

## Search Patterns

### Multi-Type Search Pattern
**Category:** Behavioral
**Purpose:** Search across different content types
**When to Use:** Comprehensive discovery
**When NOT to Use:** Type-specific queries

#### Implementation
```bash
# Search all content types
search_all_types() {
    local query="$1"
    local app_id="$2"
    
    local types=(workflows code knowledge resources)
    local results=()
    
    for type in "${types[@]}"; do
        collection="${app_id}-${type}"
        results+=($(search_collection "$collection" "$query"))
    done
    
    # Merge and rank results
    merge_results "${results[@]}"
}
```

### Pattern Discovery Pattern
**Category:** Intelligence
**Purpose:** Identify recurring solutions
**When to Use:** Cross-app analysis
**When NOT to Use:** Single app queries

#### Implementation
```bash
# Discover patterns across apps
discover_patterns() {
    local pattern_query="$1"
    local threshold=0.8
    
    # Search all apps
    local all_results=$(search_all_apps "$pattern_query")
    
    # Group similar results
    group_by_similarity "$all_results" "$threshold"
    
    # Identify patterns
    extract_common_patterns
}
```

## Integration Patterns

### Embedding Pipeline Pattern
**Category:** Integration
**Purpose:** Efficient embedding generation
**Systems:** Content → Ollama → Qdrant
**Direction:** Unidirectional

#### Architecture
```
Extract → Chunk → Embed → Store
   ↓        ↓       ↓       ↓
Content  Segments Vectors Qdrant
```

#### Implementation
```bash
# Complete pipeline
process_content() {
    local file="$1"
    
    # Extract content
    local content=$(extract_content "$file")
    
    # Chunk into segments
    local chunks=$(chunk_content "$content")
    
    # Generate embeddings
    local embeddings=$(generate_embeddings "$chunks")
    
    # Store in Qdrant
    store_embeddings "$embeddings"
}
```

### RAG Enhancement Pattern
**Category:** Integration
**Purpose:** Improve AI responses with context
**Systems:** Query → Qdrant → LLM
**Direction:** Bidirectional

#### Implementation
```bash
# Retrieve context for RAG
get_rag_context() {
    local query="$1"
    local limit=5
    
    # Search for relevant content
    local results=$(search_embeddings "$query" "$limit")
    
    # Extract text from results
    local context=""
    for result in $results; do
        context+=$(extract_text "$result")
    done
    
    echo "$context"
}

# Enhance LLM prompt
enhance_prompt() {
    local query="$1"
    local context=$(get_rag_context "$query")
    
    echo "Context: $context

Question: $query

Based on the context above, please answer:"
}
```

## Code Extraction Patterns

### Function Extraction Pattern
**Category:** Extraction
**Purpose:** Extract function definitions
**When to Use:** Code analysis
**When NOT to Use:** Non-code content

#### Implementation
```bash
# Extract shell functions
extract_shell_functions() {
    local file="$1"
    
    grep -E "^[[:alnum:]_]+::[[:alnum:]_]+\(\)" "$file" | \
    while read -r line; do
        func_name=$(echo "$line" | sed 's/().*//')
        func_body=$(awk "/^$func_name\(\)/,/^}/" "$file")
        
        echo "{
            \"type\": \"function\",
            \"name\": \"$func_name\",
            \"content\": \"$func_body\",
            \"file\": \"$file\"
        }"
    done
}
```

### API Endpoint Extraction
**Category:** Extraction
**Purpose:** Discover API endpoints
**When to Use:** REST/GraphQL APIs
**When NOT to Use:** Non-API code

#### Implementation
```javascript
// Extract Express routes
function extractRoutes(code) {
    const patterns = [
        /app\.(get|post|put|delete|patch)\(['"]([^'"]+)['"]/g,
        /router\.(get|post|put|delete|patch)\(['"]([^'"]+)['"]/g
    ];
    
    const routes = [];
    for (const pattern of patterns) {
        let match;
        while ((match = pattern.exec(code)) !== null) {
            routes.push({
                method: match[1].toUpperCase(),
                path: match[2],
                type: 'rest_api'
            });
        }
    }
    return routes;
}
```

## Documentation Patterns

### Marker-Based Extraction
**Category:** Extraction
**Purpose:** Structured content extraction
**When to Use:** Formatted documentation
**When NOT to Use:** Plain text

#### Implementation
```bash
# Extract marked sections
extract_marked_sections() {
    local file="$1"
    local marker_type="$2"  # PATTERN, DECISION, LESSON
    
    awk -v type="$marker_type" '
        /<!-- EMBED:'"$marker_type"':START -->/ { capture=1; next }
        /<!-- EMBED:'"$marker_type"':END -->/ { capture=0; print "---"; next }
        capture { print }
    ' "$file"
}
```

### Knowledge Graph Pattern
**Category:** Structural
**Purpose:** Build relationship network
**When to Use:** Complex documentation
**When NOT to Use:** Simple content

#### Implementation
```bash
# Build knowledge graph
build_knowledge_graph() {
    local docs_dir="$1"
    
    # Extract entities
    local entities=$(extract_entities "$docs_dir")
    
    # Find relationships
    local relationships=$(find_relationships "$entities")
    
    # Create graph structure
    create_graph "$entities" "$relationships"
    
    # Store as embeddings with metadata
    for entity in $entities; do
        embed_with_relationships "$entity" "$relationships"
    done
}
```

## Performance Patterns

### Batch Processing Pattern
**Category:** Performance
**Purpose:** Process multiple items efficiently
**When to Use:** Bulk operations
**When NOT to Use:** Real-time requirements

#### Implementation
```bash
# Batch embed with size limit
batch_embed() {
    local items=("$@")
    local batch_size=50
    local batch=()
    
    for item in "${items[@]}"; do
        batch+=("$item")
        
        if [[ ${#batch[@]} -ge $batch_size ]]; then
            # Process batch
            ollama_batch_embed "${batch[@]}"
            batch=()
        fi
    done
    
    # Process remaining
    if [[ ${#batch[@]} -gt 0 ]]; then
        ollama_batch_embed "${batch[@]}"
    fi
}
```

### Caching Pattern
**Category:** Performance
**Purpose:** Avoid redundant processing
**When to Use:** Repeated operations
**When NOT to Use:** Unique content

#### Implementation
```bash
# Content-based caching
cache_embeddings() {
    local content="$1"
    local cache_key=$(echo -n "$content" | sha256sum | cut -d' ' -f1)
    local cache_file="~/.vrooli/cache/embeddings/${cache_key}"
    
    # Check cache
    if [[ -f "$cache_file" ]]; then
        cat "$cache_file"
        return 0
    fi
    
    # Generate and cache
    local embedding=$(generate_embedding "$content")
    echo "$embedding" > "$cache_file"
    echo "$embedding"
}
```

## Anti-Patterns to Avoid

### Over-Chunking
**Why It's Problematic:** Loses semantic context
**How to Recognize:** Poor search relevance
**Common Causes:** Fixed small chunk sizes

#### Bad Example
```bash
# DON'T: Arbitrary small chunks
chunk_tiny() {
    split -b 100  # 100 byte chunks - too small!
}
```

#### Good Alternative
```bash
# DO: Semantic chunking
chunk_semantic() {
    # Use natural boundaries
    split_by_paragraph_with_overlap 1000 200
}
```

### Synchronous Extraction
**Why It's Problematic:** Blocks user operations
**How to Recognize:** Long wait times
**Common Causes:** Direct extraction in request handler

#### Bad Example
```bash
# DON'T: Block on extraction
handle_refresh() {
    extract_all_content  # Blocks for minutes!
    respond_to_user
}
```

#### Good Alternative
```bash
# DO: Queue for background processing
handle_refresh() {
    queue_extraction_job
    respond_with_job_id
}
```

### Embedding Everything
**Why It's Problematic:** Wastes resources, poor relevance
**How to Recognize:** Large indexes, irrelevant results
**Common Causes:** No content filtering

#### Bad Example
```bash
# DON'T: Embed all files
find . -type f -exec embed {} \;
```

#### Good Alternative
```bash
# DO: Selective embedding
find . -type f \
    -name "*.md" -o -name "*.ts" -o -name "*.sh" \
    ! -path "*/node_modules/*" \
    ! -path "*/.git/*" \
    -exec embed {} \;
```