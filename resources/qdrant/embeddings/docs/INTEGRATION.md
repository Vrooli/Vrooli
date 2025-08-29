# Integration Guide

## Overview

The Embeddings system integrates with multiple components to provide semantic knowledge extraction and search capabilities across Vrooli applications.

## Core Integrations

### Qdrant Integration

The embeddings system uses Qdrant as its vector storage backend.

**Connection Flow:**
```
Content → Extract → Embed → Qdrant Collections
                              ↓
                         Search Results
```

**Configuration:**
```bash
# Qdrant connection
QDRANT_URL="http://localhost:6333"
QDRANT_API_KEY="${QDRANT_API_KEY:-}"

# Collection naming
COLLECTION_NAME="${APP_ID}-${CONTENT_TYPE}"
```

**Collection Management:**
```bash
# Create collection for app
create_app_collections() {
    local app_id="$1"
    local types=(workflows code knowledge resources)
    
    for type in "${types[@]}"; do
        qdrant::collections::create "${app_id}-${type}" 1536 "Cosine"
    done
}
```

### Ollama Integration

Ollama provides the embedding generation capability.

**Embedding Pipeline:**
```bash
# Generate embeddings
generate_embeddings() {
    local text="$1"
    
    curl -X POST http://localhost:11434/api/embed \
        -H "Content-Type: application/json" \
        -d "{
            \"model\": \"mxbai-embed-large\",
            \"input\": [\"$text\"]
        }"
}

# Batch embedding
batch_embed() {
    local texts=("$@")
    
    curl -X POST http://localhost:11434/api/embed \
        -H "Content-Type: application/json" \
        -d "{
            \"model\": \"mxbai-embed-large\",
            \"input\": $(printf '%s\n' "${texts[@]}" | jq -R . | jq -s .)
        }"
}
```

### Git Integration

Git provides change detection and version tracking.

**Change Detection:**
```bash
# Detect changes since last index
detect_changes() {
    local last_commit=$(jq -r '.last_indexed_commit' .vrooli/app-identity.json)
    local current_commit=$(git rev-parse HEAD)
    
    if [[ "$last_commit" != "$current_commit" ]]; then
        # Get changed files
        git diff --name-only "$last_commit" "$current_commit"
        return 0
    fi
    return 1
}

# Update tracking
update_commit_tracking() {
    local commit=$(git rev-parse HEAD)
    jq ".last_indexed_commit = \"$commit\"" .vrooli/app-identity.json > tmp.$$ && \
    mv tmp.$$ .vrooli/app-identity.json
}
```

## App Integration

### App Identity System

Each app has a unique identity for isolation.

**Identity File Structure:**
```json
{
    "app_id": "user-portal-v1",
    "app_name": "User Portal",
    "created_at": "2024-01-15T10:00:00Z",
    "last_indexed_at": "2024-01-20T15:30:00Z",
    "last_indexed_commit": "abc123def456",
    "collections": [
        "user-portal-v1-workflows",
        "user-portal-v1-code",
        "user-portal-v1-knowledge",
        "user-portal-v1-resources"
    ]
}
```

**Identity Management:**
```bash
# Initialize app identity
init_app_identity() {
    local app_dir="$1"
    local app_id=$(generate_app_id "$app_dir")
    
    cat > "$app_dir/.vrooli/app-identity.json" << EOF
{
    "app_id": "$app_id",
    "app_name": "$(basename "$app_dir")",
    "created_at": "$(date -Iseconds)",
    "collections": []
}
EOF
}
```

### N8n Workflow Integration

Extract and index n8n workflows.

**Workflow Extraction:**
```bash
# Extract workflow metadata
extract_workflow() {
    local workflow_file="$1"
    
    jq '{
        name: .name,
        nodes: [.nodes[].type],
        triggers: [.nodes[] | select(.type | contains("trigger"))],
        webhooks: [.nodes[] | select(.type == "n8n-nodes-base.webhook")],
        description: .settings.notes // ""
    }' "$workflow_file"
}

# Index workflow
index_workflow() {
    local workflow_file="$1"
    local metadata=$(extract_workflow "$workflow_file")
    local content=$(jq -r '.name + " " + .description' <<< "$metadata")
    
    local embedding=$(generate_embedding "$content")
    store_embedding "$embedding" "$metadata"
}
```

### Code Integration

Extract and index source code.

**Language Support:**
```bash
# Supported languages
SUPPORTED_LANGUAGES=(
    javascript typescript python shell
    go rust java kotlin swift
    ruby php csharp cpp sql
)

# Language-specific extraction
extract_by_language() {
    local file="$1"
    local ext="${file##*.}"
    
    case "$ext" in
        js|ts|jsx|tsx)
            extract_javascript "$file"
            ;;
        py)
            extract_python "$file"
            ;;
        sh|bash)
            extract_shell "$file"
            ;;
        *)
            extract_generic "$file"
            ;;
    esac
}
```

### Documentation Integration

Process markdown documentation.

**Markdown Processing:**
```bash
# Extract markdown sections
extract_markdown_sections() {
    local file="$1"
    
    # Extract headers and content
    awk '
        /^#+ / {
            if (section) print_section()
            section = $0
            content = ""
        }
        !/^#+ / && NF {
            content = content "\n" $0
        }
        END {
            if (section) print_section()
        }
        function print_section() {
            print section
            print content
            print "---"
        }
    ' "$file"
}

# Process embedding markers
process_markers() {
    local file="$1"
    
    # Extract marked sections
    grep -Pzo '(?s)<!-- EMBED:.*?:START -->.*?<!-- EMBED:.*?:END -->' "$file"
}
```

## CLI Integration

### Vrooli CLI Commands

The embeddings system integrates with the Vrooli CLI.

**Command Registration:**
```bash
# Register embeddings commands
cli::register_group "embeddings" "Semantic knowledge management"

cli::register_command "embeddings" "init" \
    "Initialize embeddings for app" \
    "qdrant::embeddings::init"

cli::register_command "embeddings" "refresh" \
    "Refresh embeddings" \
    "qdrant::embeddings::refresh"

cli::register_command "embeddings" "search" \
    "Search app knowledge" \
    "qdrant::embeddings::search"
```

### Shell Function Library

Direct shell integration for scripts.

**Function Loading:**
```bash
# Source embeddings functions
source /path/to/resources/qdrant/embeddings/lib/embedding-service.sh
source /path/to/resources/qdrant/embeddings/lib/search.sh

# Use functions directly
qdrant::embeddings::init "my-app"
qdrant::embeddings::search "query" "type"
```

## Agent Integration

### AI Agent Discovery

Agents use embeddings for knowledge discovery.

**Agent Workflow:**
```bash
# Agent discovery pattern
agent_discover() {
    local task="$1"
    
    # Search for existing solutions
    local solutions=$(resource-qdrant embeddings search-all "$task" code)
    
    # Find patterns
    local patterns=$(resource-qdrant embeddings patterns "$task")
    
    # Check for lessons learned
    local lessons=$(resource-qdrant embeddings search-all "$task failure" knowledge)
    
    # Make informed decision
    if [[ -n "$solutions" ]]; then
        reuse_solution "$solutions"
    else
        create_new_solution "$patterns" "$lessons"
    fi
}
```

### RAG Integration

Enhance agent responses with context.

**RAG Pipeline:**
```bash
# Retrieve context for agent
get_agent_context() {
    local query="$1"
    local app_context="$2"
    
    # Search relevant knowledge
    local knowledge=$(search_embeddings "$query" "knowledge" 3)
    
    # Find code examples
    local code=$(search_embeddings "$query" "code" 2)
    
    # Get related workflows
    local workflows=$(search_embeddings "$query" "workflows" 2)
    
    # Combine context
    echo "Knowledge: $knowledge
Code Examples: $code
Related Workflows: $workflows"
}
```

## Development Tool Integration

### VS Code Integration

```json
// .vscode/tasks.json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Search Knowledge",
            "type": "shell",
            "command": "resource-qdrant",
            "args": [
                "embeddings",
                "search",
                "${input:searchQuery}"
            ]
        },
        {
            "label": "Refresh Embeddings",
            "type": "shell",
            "command": "resource-qdrant",
            "args": [
                "embeddings",
                "refresh"
            ]
        }
    ]
}
```

### Git Hooks

```bash
#!/bin/bash
# .git/hooks/post-commit

# Refresh embeddings after commit
if [[ -f .vrooli/app-identity.json ]]; then
    echo "Refreshing embeddings..."
    resource-qdrant embeddings refresh &
fi
```

## API Integration

### REST API Access

Direct API access for custom integrations.

**Search API:**
```bash
# Search via API
search_api() {
    local query="$1"
    local collection="$2"
    
    # Generate query embedding
    local query_vector=$(generate_embedding "$query")
    
    # Search Qdrant
    curl -X POST "http://localhost:6333/collections/$collection/points/search" \
        -H "Content-Type: application/json" \
        -d "{
            \"vector\": $query_vector,
            \"limit\": 10,
            \"with_payload\": true
        }"
}
```

### Python Client

```python
from qdrant_client import QdrantClient
import requests

class EmbeddingsClient:
    def __init__(self):
        self.qdrant = QdrantClient(url="http://localhost:6333")
        self.ollama_url = "http://localhost:11434"
    
    def generate_embedding(self, text):
        response = requests.post(
            f"{self.ollama_url}/api/embed",
            json={"model": "mxbai-embed-large", "input": [text]}
        )
        return response.json()["embeddings"][0]
    
    def search(self, query, collection, limit=10):
        query_vector = self.generate_embedding(query)
        return self.qdrant.search(
            collection_name=collection,
            query_vector=query_vector,
            limit=limit
        )
```

## Performance Optimization

### Parallel Processing

```bash
# Parallel extraction
parallel_extract() {
    local files=("$@")
    local num_cores=$(nproc)
    
    printf '%s\n' "${files[@]}" | \
    xargs -P "$num_cores" -I {} bash -c 'extract_and_embed "{}"'
}
```

### Batch Operations

```bash
# Batch upsert to Qdrant
batch_upsert() {
    local collection="$1"
    shift
    local points=("$@")
    
    # Build batch request
    local batch_json=$(printf '%s\n' "${points[@]}" | jq -s '.')
    
    curl -X PUT "http://localhost:6333/collections/$collection/points/batch" \
        -H "Content-Type: application/json" \
        -d "{\"points\": $batch_json}"
}
```

## Troubleshooting Integration

### Connection Issues

```bash
# Test Ollama connection
curl http://localhost:11434/api/tags

# Test Qdrant connection
curl http://localhost:6333/health

# Verify Git
git rev-parse HEAD
```

### Performance Issues

```bash
# Monitor extraction
time resource-qdrant embeddings refresh --verbose

# Check Ollama load
curl http://localhost:11434/api/ps

# Qdrant metrics
curl http://localhost:6333/metrics
```

## Best Practices

1. **Initialize early** in project lifecycle
2. **Refresh regularly** with git hooks
3. **Use batch operations** for efficiency
4. **Monitor performance** metrics
5. **Cache when possible** to reduce load
6. **Handle errors gracefully** with retries
7. **Document patterns** for reuse
8. **Test integrations** thoroughly