# Architecture

## System Overview

The Qdrant Embeddings system provides semantic knowledge extraction and search capabilities for Vrooli apps. It indexes content from multiple sources into vector collections, enabling intelligent pattern discovery and solution reuse across the entire ecosystem.

## Core Components

### 1. Extraction Pipeline

The system processes content through a multi-stage pipeline:

```
App Directory → Extractors → Parsers → Embeddings → Qdrant Collections
```

**Extractors** (`extractors/`):
- `code/` - Extracts functions, APIs, and implementations
- `docs/` - Processes markdown documentation
- `scenarios/` - Parses PRDs and configurations
- `workflows/` - Analyzes n8n workflows
- `resources/` - Captures resource capabilities

**Parsers** (`parsers/`):
- Language-specific code parsers (JavaScript, Python, Shell, etc.)
- Markdown processor with embedding marker support
- Resource-specific parsers for specialized content

**Embedding Service** (`lib/embedding-service.sh`):
- Interfaces with Ollama for vector generation
- Batch processing for efficiency
- Model: `mxbai-embed-large` (1536 dimensions)

### 2. Storage Architecture

**Collection Naming Convention**:
```
{app-id}-{content-type}
```

Example collections:
- `user-portal-v1-workflows`
- `user-portal-v1-code`
- `user-portal-v1-knowledge`

**Vector Storage**:
- Dimensions: 1536 (mxbai-embed-large)
- Distance metric: Cosine similarity
- Metadata: Source file, content type, timestamps

### 3. Search System

**Search Modes**:
- Single app search - Queries specific app collections
- Multi-app search - Queries across all collections
- Pattern discovery - Identifies recurring solutions
- Gap analysis - Finds missing knowledge areas

**Ranking Algorithm**:
- Cosine similarity scoring
- Type-based filtering
- Result deduplication
- Relevance thresholds

## Data Flow

### Initialization Flow

```bash
init → create .vrooli/app-identity.json → validate Qdrant → create collections
```

### Refresh Flow

```bash
detect changes → extract content → generate embeddings → upsert to Qdrant
```

### Search Flow

```bash
query → embed query → search collections → rank results → format output
```

## File Structure

```
embeddings/
├── cli.sh                 # Main CLI interface
├── extractors/           # Content extraction modules
│   ├── code/            # Code extraction
│   ├── docs/            # Documentation extraction
│   ├── scenarios/       # Scenario extraction
│   └── workflows/       # Workflow extraction
├── parsers/             # Content parsing modules
│   ├── code/           # Language-specific parsers
│   └── markdown.sh     # Markdown processor
├── lib/                # Core libraries
│   ├── embedding-service.sh  # Ollama interface
│   ├── cache.sh             # Caching layer
│   ├── parallel.sh          # Parallel processing
│   └── performance-logger.sh # Performance monitoring
├── search/             # Search implementations
│   ├── single.sh      # Single app search
│   └── multi.sh       # Multi-app search
└── validation/        # Validation tools
```

## Key Design Decisions

### 1. App Isolation

Each app has its own namespaced collections to:
- Prevent cross-contamination
- Enable independent updates
- Support permission models
- Allow selective deletion

### 2. Content Type Separation

Content types are stored in separate collections for:
- Optimized search performance
- Type-specific ranking algorithms
- Flexible schema evolution
- Independent scaling

### 3. Git-Based Change Detection

The system uses git commit hashes to:
- Detect when re-indexing is needed
- Avoid unnecessary processing
- Track content evolution
- Support incremental updates

### 4. Embedding Markers

Special markers in documentation enable:
- Semantic chunking of content
- Preservation of context
- Structured knowledge extraction
- Pattern identification

Example markers:
```markdown
<!-- EMBED:PATTERN:START -->
Pattern content here
<!-- EMBED:PATTERN:END -->

<!-- EMBED:DECISION:START -->
Decision content here
<!-- EMBED:DECISION:END -->
```

## Performance Characteristics

### Indexing Performance

- **Small app** (<100 files): ~30 seconds
- **Medium app** (100-500 files): 1-3 minutes
- **Large app** (500+ files): 3-10 minutes

### Search Performance

- **Single app search**: <500ms
- **Multi-app search**: 1-2 seconds
- **Pattern discovery**: 2-5 seconds

### Storage Requirements

- **Per embedding**: ~6KB (vector + metadata)
- **Typical app**: 5-50MB
- **Full ecosystem**: 100MB-1GB

## Integration Points

### 1. Ollama Integration

```bash
# Embedding generation
curl -X POST http://localhost:11434/api/embed \
  -d '{
    "model": "mxbai-embed-large",
    "input": ["text to embed"]
  }'
```

### 2. Qdrant Integration

```bash
# Collection creation
curl -X PUT http://localhost:6333/collections/{collection_name} \
  -H 'Content-Type: application/json' \
  -d '{
    "vectors": {
      "size": 1536,
      "distance": "Cosine"
    }
  }'
```

### 3. Git Integration

```bash
# Change detection
git_hash=$(git rev-parse HEAD 2>/dev/null || echo "no-git")
stored_hash=$(jq -r '.last_indexed_commit' .vrooli/app-identity.json)
if [[ "$git_hash" != "$stored_hash" ]]; then
  # Trigger refresh
fi
```

## Scaling Considerations

### Current Limitations

- Single Qdrant instance
- Sequential embedding generation
- In-memory processing for large files

### Future Enhancements

- Distributed Qdrant cluster support
- Parallel embedding generation
- Streaming processing for large files
- Incremental index updates
- Background refresh workers

## Security Model

### Data Isolation

- App-level collection separation
- No cross-app data access by default
- Metadata sanitization

### Access Control

- Read-only extraction process
- No modification of source files
- Audit logging for searches

### Sensitive Data

- Excludes `.env` files
- Skips credential files
- Configurable exclusion patterns

## Monitoring & Observability

### Metrics Tracked

- Extraction duration
- Embedding generation time
- Search latency
- Collection sizes
- Refresh frequency

### Health Checks

```bash
# System health
resource-qdrant embeddings status

# Validation
resource-qdrant embeddings validate

# Performance metrics
cat ~/.vrooli/embeddings/performance.log
```

## Error Handling

### Graceful Degradation

- Missing Ollama: Skip embedding generation
- Qdrant down: Cache for later processing
- Invalid content: Log and continue
- Network failures: Retry with backoff

### Recovery Mechanisms

- Automatic retry for transient failures
- Rollback on critical errors
- State preservation for resume
- Manual intervention alerts