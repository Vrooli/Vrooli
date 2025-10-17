# Integration Guide

## Overview

Qdrant integrates with multiple Vrooli resources to provide vector storage and semantic search capabilities across the platform. This guide covers integration patterns, APIs, and best practices.

## Resource Integrations

### Ollama Integration

Ollama provides embedding generation for Qdrant vector storage.

**Connection Flow:**
```
Text → Ollama (embed) → Vector[1536] → Qdrant (store)
```

**Configuration:**
```bash
# Ollama endpoint
OLLAMA_URL="http://localhost:11434"

# Embedding model
EMBEDDING_MODEL="mxbai-embed-large"

# Generate embeddings
curl -X POST "$OLLAMA_URL/api/embed" \
  -d '{
    "model": "mxbai-embed-large",
    "input": ["text to embed"]
  }'
```

**Usage Example:**
```bash
# Generate and store embeddings
text="Understanding vector databases"
embedding=$(ollama embed "$text")
qdrant store "$embedding" --collection knowledge
```

### LiteLLM Integration

LiteLLM uses Qdrant for RAG (Retrieval-Augmented Generation).

**RAG Flow:**
```
Query → Qdrant (search) → Context → LiteLLM (generate) → Response
```

**Implementation:**
```python
# Retrieve relevant context
def get_context(query: str, collection: str = "knowledge"):
    # Generate query embedding
    query_vector = ollama.embed(query)
    
    # Search Qdrant
    results = qdrant.search(
        collection_name=collection,
        query_vector=query_vector,
        limit=5
    )
    
    # Extract text from results
    context = "\n".join([r.payload["text"] for r in results])
    return context

# Augment LLM prompt
def generate_response(query: str):
    context = get_context(query)
    prompt = f"Context: {context}\n\nQuestion: {query}\n\nAnswer:"
    return litellm.completion(prompt)
```

### N8n Integration

N8n workflows can interact with Qdrant for intelligent automation.

**Workflow Nodes:**
1. **Qdrant Search Node**: Find similar content
2. **Qdrant Store Node**: Add new vectors
3. **Qdrant Filter Node**: Query with conditions

**Example Workflow:**
```json
{
  "nodes": [
    {
      "name": "Embed Text",
      "type": "ollama",
      "operation": "embed",
      "model": "mxbai-embed-large"
    },
    {
      "name": "Search Similar",
      "type": "qdrant",
      "operation": "search",
      "collection": "workflows",
      "limit": 10
    },
    {
      "name": "Process Results",
      "type": "code",
      "code": "return items.map(i => i.payload)"
    }
  ]
}
```

### PostgreSQL Integration

Qdrant complements PostgreSQL for hybrid search scenarios.

**Hybrid Search Pattern:**
```sql
-- PostgreSQL: Structured filtering
SELECT id, metadata FROM items 
WHERE category = 'documentation' 
  AND created_at > '2024-01-01';

-- Qdrant: Semantic search on filtered IDs
{
  "filter": {
    "must": [
      {"key": "id", "match": {"any": [/* PostgreSQL IDs */]}}
    ]
  },
  "vector": [/* query embedding */],
  "limit": 10
}
```

### Redis Integration

Redis provides caching and queuing for Qdrant operations.

**Caching Layer:**
```bash
# Cache search results
cache_key="qdrant:search:$(echo -n "$query" | md5sum)"
cached=$(redis-cli GET "$cache_key")

if [[ -z "$cached" ]]; then
    result=$(qdrant_search "$query")
    redis-cli SETEX "$cache_key" 300 "$result"
else
    result="$cached"
fi
```

**Job Queue:**
```bash
# Queue embedding jobs
redis-cli LPUSH embedding_queue "$text"

# Worker processes queue
while true; do
    text=$(redis-cli BRPOP embedding_queue 0)
    embedding=$(generate_embedding "$text")
    store_in_qdrant "$embedding"
done
```

## API Integration

### REST API

**Endpoint:** `http://localhost:6333`

**Common Operations:**

```bash
# Create collection
curl -X PUT "http://localhost:6333/collections/my-collection" \
  -H 'Content-Type: application/json' \
  -d '{
    "vectors": {
      "size": 1536,
      "distance": "Cosine"
    }
  }'

# Insert vectors
curl -X PUT "http://localhost:6333/collections/my-collection/points" \
  -H 'Content-Type: application/json' \
  -d '{
    "points": [{
      "id": 1,
      "vector": [0.1, 0.2, ...],
      "payload": {"text": "sample"}
    }]
  }'

# Search
curl -X POST "http://localhost:6333/collections/my-collection/points/search" \
  -H 'Content-Type: application/json' \
  -d '{
    "vector": [0.1, 0.2, ...],
    "limit": 10
  }'
```

### gRPC API

**Endpoint:** `localhost:6334`

**Proto Definition:**
```protobuf
service QdrantService {
  rpc Search(SearchRequest) returns (SearchResponse);
  rpc Upsert(UpsertRequest) returns (UpsertResponse);
  rpc Delete(DeleteRequest) returns (DeleteResponse);
}
```

**Python Client:**
```python
import grpc
from qdrant_client import QdrantClient

client = QdrantClient(host="localhost", port=6334, grpc_port=6334)

# Search operation
results = client.search(
    collection_name="my-collection",
    query_vector=[0.1, 0.2, ...],
    limit=10
)
```

## Client Libraries

### Shell Integration

```bash
# Source Qdrant functions
source /path/to/resources/qdrant/lib/core.sh

# Use functions
qdrant::collections::create "my-collection" 1536 "Cosine"
qdrant::collections::search "my-collection" "$vector"
qdrant::collections::delete "my-collection"
```

### Python Integration

```python
from qdrant_client import QdrantClient
from qdrant_client.models import Distance, VectorParams

# Initialize client
client = QdrantClient(url="http://localhost:6333")

# Create collection
client.create_collection(
    collection_name="my-collection",
    vectors_config=VectorParams(size=1536, distance=Distance.COSINE)
)

# Insert vectors
client.upsert(
    collection_name="my-collection",
    points=[
        {"id": 1, "vector": [0.1, 0.2, ...], "payload": {"text": "sample"}}
    ]
)
```

### JavaScript Integration

```javascript
import { QdrantClient } from '@qdrant/js-client-rest';

// Initialize client
const client = new QdrantClient({ url: 'http://localhost:6333' });

// Create collection
await client.createCollection('my-collection', {
  vectors: {
    size: 1536,
    distance: 'Cosine'
  }
});

// Search
const results = await client.search('my-collection', {
  vector: [0.1, 0.2, ...],
  limit: 10
});
```

## Docker Compose Integration

```yaml
version: '3.8'

services:
  qdrant:
    image: qdrant/qdrant:latest
    container_name: qdrant
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - qdrant_data:/qdrant/storage
    environment:
      - QDRANT__SERVICE__HTTP_PORT=6333
      - QDRANT__SERVICE__GRPC_PORT=6334
      - QDRANT__LOG_LEVEL=INFO
    networks:
      - vrooli_network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:6333/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  app:
    build: .
    depends_on:
      qdrant:
        condition: service_healthy
    environment:
      - QDRANT_URL=http://qdrant:6333
    networks:
      - vrooli_network

volumes:
  qdrant_data:

networks:
  vrooli_network:
```

## Kubernetes Integration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: qdrant
spec:
  replicas: 1
  selector:
    matchLabels:
      app: qdrant
  template:
    metadata:
      labels:
        app: qdrant
    spec:
      containers:
      - name: qdrant
        image: qdrant/qdrant:latest
        ports:
        - containerPort: 6333
          name: http
        - containerPort: 6334
          name: grpc
        volumeMounts:
        - name: storage
          mountPath: /qdrant/storage
        env:
        - name: QDRANT__SERVICE__HTTP_PORT
          value: "6333"
        - name: QDRANT__SERVICE__GRPC_PORT
          value: "6334"
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2"
        livenessProbe:
          httpGet:
            path: /health
            port: 6333
          initialDelaySeconds: 30
          periodSeconds: 10
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: qdrant-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: qdrant
spec:
  selector:
    app: qdrant
  ports:
  - name: http
    port: 6333
    targetPort: 6333
  - name: grpc
    port: 6334
    targetPort: 6334
```

## Integration Best Practices

### Connection Management

1. **Use connection pooling** for high-throughput applications
2. **Implement retry logic** with exponential backoff
3. **Monitor connection health** with regular heartbeats
4. **Use appropriate timeouts** for different operations

### Error Handling

```bash
# Robust error handling pattern
qdrant_operation_with_retry() {
    local max_retries=3
    local retry_count=0
    
    while [[ $retry_count -lt $max_retries ]]; do
        if qdrant_operation "$@"; then
            return 0
        fi
        
        retry_count=$((retry_count + 1))
        sleep $((2 ** retry_count))  # Exponential backoff
    done
    
    log::error "Qdrant operation failed after $max_retries retries"
    return 1
}
```

### Performance Optimization

1. **Batch operations** for bulk inserts/updates
2. **Use filters** to reduce search space
3. **Cache frequently accessed** results
4. **Monitor query latency** and optimize indexes

### Security

1. **Use API keys** in production
2. **Implement network isolation** with Docker networks
3. **Enable TLS** for external connections
4. **Audit log** all operations

## Troubleshooting Integration Issues

### Connection Issues

```bash
# Test connectivity
curl -f http://localhost:6333/health

# Check Docker network
docker network inspect vrooli_network

# Verify ports
netstat -tulpn | grep -E "6333|6334"
```

### Performance Issues

```bash
# Check collection statistics
curl http://localhost:6333/collections/my-collection

# Monitor resource usage
docker stats qdrant

# Analyze slow queries
curl http://localhost:6333/metrics
```

### Data Consistency

```bash
# Verify collection schema
curl http://localhost:6333/collections/my-collection

# Check point count
curl http://localhost:6333/collections/my-collection/points/count

# Validate payload structure
curl "http://localhost:6333/collections/my-collection/points/1"
```