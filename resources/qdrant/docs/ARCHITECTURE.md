# Architecture

## System Overview

Qdrant is a high-performance vector database designed for similarity search and AI applications. In the Vrooli ecosystem, it serves as the central vector storage and search infrastructure, enabling semantic search, recommendations, and RAG systems.

## Core Components

### 1. Storage Engine

**Vector Storage:**
- **HNSW Index**: Hierarchical Navigable Small World graph for fast approximate search
- **Flat Index**: Exact search for small collections
- **Disk-based Storage**: Memory-mapped files for persistent storage
- **In-memory Cache**: Hot data kept in RAM for performance

**Data Model:**
```
Collection
├── Vectors (high-dimensional arrays)
├── Payloads (metadata JSON)
├── Points (vector + payload pairs)
└── Indexes (HNSW graphs)
```

### 2. API Layer

**REST API (Port 6333):**
- HTTP/HTTPS interface
- JSON request/response format
- OpenAPI specification
- WebSocket support

**gRPC API (Port 6334):**
- Binary protocol for performance
- Streaming support
- Lower latency than REST
- Type-safe with protobuf

### 3. Query Engine

**Search Pipeline:**
```
Query → Embedding → Filter → Search → Rank → Result
```

**Query Types:**
- k-NN search (k nearest neighbors)
- Range search (within distance threshold)
- Filtered search (with metadata conditions)
- Recommendation (positive/negative examples)

## Deployment Architecture

### Docker Deployment

```
┌─────────────────────────────────┐
│         Host System             │
│                                 │
│  ┌──────────────────────────┐  │
│  │    Docker Container      │  │
│  │                          │  │
│  │  ┌──────────────────┐    │  │
│  │  │   Qdrant Server  │    │  │
│  │  │                  │    │  │
│  │  │  ┌────────────┐  │    │  │
│  │  │  │  REST API  │←─┼────┼──┼── Port 6333
│  │  │  └────────────┘  │    │  │
│  │  │                  │    │  │
│  │  │  ┌────────────┐  │    │  │
│  │  │  │  gRPC API  │←─┼────┼──┼── Port 6334
│  │  │  └────────────┘  │    │  │
│  │  │                  │    │  │
│  │  │  ┌────────────┐  │    │  │
│  │  │  │   Storage  │←─┼────┼──┼── Volume Mount
│  │  │  └────────────┘  │    │  │
│  │  └──────────────────┘    │  │
│  └──────────────────────────┘  │
└─────────────────────────────────┘
```

### Directory Structure

```
/qdrant/
├── storage/           # Persistent data
│   ├── collections/   # Vector collections
│   │   ├── {name}/   # Per-collection data
│   │   │   ├── segments/  # Data segments
│   │   │   └── meta.json  # Collection metadata
│   └── snapshots/    # Backup snapshots
├── config/           # Configuration
│   └── config.yaml   # Server configuration
└── logs/            # Application logs
```

## Data Flow Architecture

### Write Path

```
1. Client Request
   ↓
2. API Gateway (REST/gRPC)
   ↓
3. Validation Layer
   ↓
4. Vector Processing
   ↓
5. Index Update (HNSW)
   ↓
6. Persistent Storage
   ↓
7. Acknowledgment
```

### Read Path

```
1. Search Query
   ↓
2. Query Parser
   ↓
3. Filter Evaluation
   ↓
4. Vector Search (HNSW)
   ↓
5. Result Ranking
   ↓
6. Payload Retrieval
   ↓
7. Response Assembly
```

## Collection Architecture

### Collection Schema

```json
{
  "name": "app-embeddings",
  "vectors": {
    "size": 1536,
    "distance": "Cosine",
    "hnsw_config": {
      "m": 16,
      "ef_construct": 100,
      "full_scan_threshold": 10000
    },
    "quantization_config": {
      "scalar": {
        "type": "int8",
        "quantile": 0.99,
        "always_ram": true
      }
    }
  },
  "optimizers_config": {
    "default_segment_number": 4,
    "indexing_threshold": 20000,
    "flush_interval_sec": 5,
    "memmap_threshold_kb": 50000
  }
}
```

### Segmentation Strategy

Collections are divided into segments for:
- **Parallel processing**: Multiple CPU cores
- **Incremental indexing**: New data doesn't rebuild entire index
- **Memory management**: Load segments as needed
- **Optimization**: Background segment merging

## Performance Architecture

### Index Types

**HNSW (Hierarchical Navigable Small World):**
- Build time: O(N log N)
- Search time: O(log N)
- Memory: O(N × M)
- Accuracy: 95-99% (configurable)

**Flat Index:**
- Build time: O(1)
- Search time: O(N)
- Memory: O(N)
- Accuracy: 100%

### Optimization Layers

1. **Query Optimization:**
   - Filter push-down
   - Index selection
   - Result caching

2. **Storage Optimization:**
   - Vector quantization
   - Payload compression
   - Segment compaction

3. **Memory Optimization:**
   - Memory mapping
   - LRU cache
   - Lazy loading

## Scaling Architecture

### Vertical Scaling

```yaml
resources:
  cpu: 4-16 cores      # More cores = parallel queries
  memory: 8-64GB       # More RAM = larger indexes in memory
  storage: SSD         # Fast I/O for disk-based operations
```

### Horizontal Scaling (Future)

```
┌──────────────┐     ┌──────────────┐
│   Qdrant 1   │     │   Qdrant 2   │
│  Shard 1,2   │     │  Shard 3,4   │
└──────┬───────┘     └──────┬───────┘
       │                    │
       └────────┬───────────┘
                │
        ┌───────┴────────┐
        │   Load Balancer│
        └───────┬────────┘
                │
           Client Requests
```

## Security Architecture

### Access Control

```
Client → Authentication → Authorization → Resource Access
         (API Key)       (Collection ACL)  (Read/Write)
```

### Network Security

- **TLS/SSL**: Encrypted communications
- **API Keys**: Token-based authentication
- **Network Isolation**: Docker networks
- **Rate Limiting**: Request throttling

### Data Security

- **Encryption at Rest**: Optional disk encryption
- **Audit Logging**: Operation tracking
- **Backup Encryption**: Secure snapshots
- **Payload Sanitization**: Input validation

## Monitoring Architecture

### Metrics Collection

```
Qdrant → Prometheus Metrics → Grafana Dashboard
       ↓
    Health Checks → Alerting System
```

### Key Metrics

**System Metrics:**
- CPU utilization
- Memory usage
- Disk I/O
- Network traffic

**Application Metrics:**
- Query latency (p50, p95, p99)
- Requests per second
- Collection size
- Index build time

**Business Metrics:**
- Search accuracy
- User satisfaction
- Query patterns
- Content coverage

## Integration Architecture

### Service Mesh

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Ollama    │────▶│   Qdrant    │◀────│   LiteLLM   │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
                    ┌──────┴──────┐
                    │             │
            ┌───────▼───┐   ┌────▼────┐
            │    N8n    │   │  Redis  │
            └───────────┘   └─────────┘
```

### Communication Patterns

1. **Synchronous**: REST/gRPC for real-time queries
2. **Asynchronous**: Queue-based for batch processing
3. **Cached**: Redis layer for frequent queries
4. **Streaming**: WebSocket for continuous updates

## Failure Handling

### Resilience Patterns

1. **Circuit Breaker**: Prevent cascading failures
2. **Retry Logic**: Exponential backoff
3. **Fallback**: Degraded service options
4. **Bulkhead**: Resource isolation

### Recovery Mechanisms

```bash
# Automatic recovery
- Health check failures → Container restart
- Corrupted segment → Segment rebuild
- Memory pressure → Cache eviction

# Manual recovery
- Snapshot restore
- Collection rebuild
- Index optimization
```

## Future Architecture

### Planned Enhancements

1. **Distributed Architecture**: Multi-node clusters
2. **Sharding**: Automatic data distribution
3. **Replication**: High availability
4. **Federation**: Cross-region search
5. **GPU Acceleration**: CUDA/ROCm support

### Architecture Evolution

```
Current: Single Node → Near-term: Primary/Replica → Future: Distributed Cluster
         (2024)              (2025)                      (2026)
```