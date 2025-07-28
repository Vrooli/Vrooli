# Qdrant Vector Database

Qdrant is a high-performance vector database optimized for storing, searching, and managing vectors for AI applications. It provides fast similarity search and advanced filtering capabilities, making it ideal for AI agent memory, semantic search, and RAG (Retrieval Augmented Generation) systems.

## 🚀 Quick Start

### Installation

```bash
# Install with default settings
./scripts/resources/storage/qdrant/manage.sh --action install

# Install with custom ports
QDRANT_CUSTOM_PORT=6433 QDRANT_CUSTOM_GRPC_PORT=6434 \
  ./scripts/resources/storage/qdrant/manage.sh --action install

# Install with API key authentication
QDRANT_CUSTOM_API_KEY=your-secure-api-key \
  ./scripts/resources/storage/qdrant/manage.sh --action install
```

### Access

After installation:
- **Web UI**: http://localhost:6333/dashboard (interactive dashboard)
- **REST API**: http://localhost:6333 (HTTP API)
- **gRPC API**: grpc://localhost:6334 (high-performance API)

## 📦 Default Collections

Qdrant automatically creates four collections optimized for Vrooli:

| Collection | Purpose | Vector Size | Distance |
|------------|---------|-------------|----------|
| `agent_memory` | AI agent persistent memory | 1536 | Cosine |
| `code_embeddings` | Code similarity search | 768 | Dot |
| `document_chunks` | Document RAG system | 1536 | Cosine |
| `conversation_history` | Chat context storage | 1536 | Cosine |

## 🔧 Common Operations

### Service Management

```bash
# Check status
./manage.sh --action status

# Start/stop/restart
./manage.sh --action start
./manage.sh --action stop
./manage.sh --action restart

# View logs
./manage.sh --action logs --lines 100

# Monitor health continuously
./manage.sh --action monitor --interval 5

# Run diagnostics
./manage.sh --action diagnose
```

### Collection Management

```bash
# List all collections
./manage.sh --action list-collections

# Create a new collection
./manage.sh --action create-collection --collection my_vectors --vector-size 768 --distance Dot

# Get collection information
./manage.sh --action collection-info --collection agent_memory

# Delete a collection
./manage.sh --action delete-collection --collection old_collection

# Force delete non-empty collection
./manage.sh --action delete-collection --collection old_collection --force yes
```

### Backup and Restore

```bash
# Create backup of all collections
./manage.sh --action backup --snapshot-name daily-backup

# Create backup of specific collections
./manage.sh --action backup --collections "agent_memory,code_embeddings" --snapshot-name selective-backup

# List available snapshots
./manage.sh --action list-snapshots

# Restore from backup
./manage.sh --action restore --snapshot-name daily-backup
```

### Performance Monitoring

```bash
# Get index statistics for a collection
./manage.sh --action index-stats --collection agent_memory

# Get index statistics for all collections
./manage.sh --action index-stats

# Monitor resource usage
./manage.sh --action monitor --interval 10
```

## 🔐 Security

### Authentication

By default, Qdrant runs without authentication for local development. For production:

```bash
# Set API key before installation
export QDRANT_CUSTOM_API_KEY="your-secure-api-key-here"
./manage.sh --action install
```

### Network Isolation

- Container runs in isolated Docker network
- Only specified ports are exposed
- Data directory has restricted permissions

## 🏗️ Architecture

### Directory Structure

```
~/.qdrant/
├── data/           # Vector data and indices
│   ├── collections/
│   └── meta/
├── config/         # Configuration files
└── snapshots/      # Backup snapshots
    └── *.tar.gz
```

### Docker Configuration

- **Container**: `qdrant`
- **Network**: `qdrant-network`
- **Ports**: 
  - 6333 (REST API)
  - 6334 (gRPC API)
- **Volumes**:
  - `~/.qdrant/data:/qdrant/storage`
  - `~/.qdrant/snapshots:/qdrant/snapshots`

### Performance Settings

Default configuration optimized for Vrooli:
- **Segment Size**: 20,000 vectors per segment
- **Memory Mapping**: Enabled for segments > 100,000 vectors
- **Indexing Threshold**: 20,000 vectors
- **Distance Metrics**: Cosine (similarity), Dot (performance), Euclidean (precision)

## 🔌 Integration with Vrooli

Qdrant automatically registers with Vrooli's resource discovery system:

```json
{
  "services": {
    "storage": {
      "qdrant": {
        "enabled": true,
        "baseUrl": "http://localhost:6333",
        "grpcUrl": "grpc://localhost:6334",
        "collections": {
          "agent_memory": {"vector_size": 1536, "distance": "Cosine"},
          "code_embeddings": {"vector_size": 768, "distance": "Dot"}
        }
      }
    }
  }
}
```

## 🛠️ Advanced Usage

### Using REST API Directly

```bash
# Check service health
curl http://localhost:6333/

# List collections
curl http://localhost:6333/collections | jq

# Get collection info
curl http://localhost:6333/collections/agent_memory | jq

# With authentication
curl -H "api-key: your-key" http://localhost:6333/collections | jq
```

### Custom Collection Configuration

```bash
# Create high-dimensional collection
./manage.sh --action create-collection \
  --collection large_embeddings \
  --vector-size 4096 \
  --distance Cosine

# Create performance-optimized collection  
./manage.sh --action create-collection \
  --collection fast_search \
  --vector-size 512 \
  --distance Dot
```

### Vector Operations Examples

```bash
# Insert vectors (via API)
curl -X PUT http://localhost:6333/collections/agent_memory/points \
  -H "Content-Type: application/json" \
  -d '{
    "points": [{
      "id": 1,
      "vector": [0.1, 0.2, 0.3, ...],
      "payload": {"text": "AI agent memory", "timestamp": "2024-01-15"}
    }]
  }'

# Search similar vectors
curl -X POST http://localhost:6333/collections/agent_memory/points/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3, ...],
    "limit": 10,
    "with_payload": true
  }'
```

## 🧠 AI Integration Patterns

### With Ollama (Local Embeddings)

```bash
# 1. Generate embeddings with Ollama
curl -X POST http://localhost:11434/api/embeddings \
  -d '{"model": "nomic-embed-text", "prompt": "search query"}'

# 2. Search in Qdrant
curl -X POST http://localhost:6333/collections/code_embeddings/points/search \
  -d '{"vector": [embeddings_from_ollama], "limit": 5}'
```

### Multi-Resource Workflow

```bash
# Document processing pipeline:
# 1. Store document in MinIO
curl -X PUT http://localhost:9000/documents/file.pdf -T document.pdf

# 2. Extract text with AI service
# 3. Generate embeddings
# 4. Store vectors in Qdrant
curl -X PUT http://localhost:6333/collections/document_chunks/points \
  -d '{"points": [{"id": 1, "vector": [...], "payload": {"source": "file.pdf"}}]}'
```

## 🐛 Troubleshooting

### Common Issues

**Port conflicts**
```bash
# Check what's using the port
sudo lsof -i :6333

# Use custom ports
QDRANT_CUSTOM_PORT=6433 ./manage.sh --action install
```

**Container won't start**
```bash
# Check logs
./manage.sh --action logs --lines 100

# Run diagnostics
./manage.sh --action diagnose

# Check disk space
df -h ~/.qdrant/
```

**API not responding**
```bash
# Verify container is running
docker ps | grep qdrant

# Check network connectivity
curl -I http://localhost:6333

# Verify API key if using authentication
curl -H "api-key: your-key" http://localhost:6333/
```

### Performance Issues

**Slow search queries**
- Check collection configuration (vector size, distance metric)
- Monitor segment size and indexing
- Consider using gRPC API for better performance
- Review HNSW parameters

**High memory usage**
- Adjust `memmap_threshold` setting
- Optimize segment size
- Consider using quantization for large collections

### Reset Everything

```bash
# Complete uninstall including data
./manage.sh --action uninstall --remove-data yes

# Fresh installation
./manage.sh --action install
```

## 📊 Performance Benchmarks

Qdrant Performance Characteristics:
- **Search Latency**: ~23ms for 1M vectors (1536D, Cosine)
- **Throughput**: 4x higher RPS than alternatives
- **Memory Efficiency**: Rust-based, no GC overhead
- **Scalability**: Handles 100M+ vectors per collection

Optimal Use Cases:
- **AI Agent Memory**: Fast context retrieval
- **Semantic Search**: Document and code similarity
- **Real-time Recommendations**: Sub-50ms response times
- **RAG Systems**: Efficient chunk retrieval

## 🔄 Maintenance

### Regular Tasks

```bash
# Weekly backup
./manage.sh --action backup --snapshot-name weekly-$(date +%Y%m%d)

# Monitor disk usage
du -sh ~/.qdrant/data/

# Check collection health
./manage.sh --action index-stats
```

### Upgrade Process

```bash
# Upgrade to latest version
./manage.sh --action upgrade
```

### Optimization

```bash
# Monitor performance
./manage.sh --action monitor

# Review collection statistics
./manage.sh --action list-collections

# Cleanup old snapshots (keep 5 recent)
./manage.sh --action cleanup-snapshots --keep 5
```

## 📝 Notes

- Vector data persists between container restarts
- Uninstalling preserves data by default (use `--remove-data yes` to delete)
- Web UI provides visual interface for collections and search
- gRPC API offers better performance than REST for high-throughput applications
- Supports hybrid search (vector + keyword filtering)
- Compatible with popular embedding models (OpenAI, Sentence Transformers, etc.)

## 🔗 Resources

- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [REST API Reference](https://qdrant.github.io/qdrant/redoc/index.html)
- [Python Client](https://github.com/qdrant/qdrant-client)
- [Performance Benchmarks](https://qdrant.tech/benchmarks/)
- [Vrooli Resource System](/packages/server/src/services/resources/README.md)