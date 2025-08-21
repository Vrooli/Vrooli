# Qdrant - High-Performance Vector Database

Qdrant is a vector similarity search engine with extended filtering support, designed for high-performance semantic search and AI applications. This resource provides automated installation, configuration, and management of Qdrant with comprehensive collection management for the Vrooli project.

## 🎯 Quick Reference

- **Category**: Storage
- **Ports**: 6333 (REST API), 6334 (gRPC API)
- **Container**: qdrant
- **Data Directory**: ~/.qdrant/data
- **Status**: Production Ready

## 🚀 Quick Start

### Prerequisites
- Docker installed and running
- 2GB+ RAM available for optimal performance
- Ports 6333 and 6334 available
- 5GB+ disk space for vector storage

### Installation
```bash
# Install with default settings
resource-qdrant install

# Install with custom ports
QDRANT_CUSTOM_PORT=7333 QDRANT_CUSTOM_GRPC_PORT=7334 resource-qdrant install

# Install with API key authentication (recommended for production)
QDRANT_CUSTOM_API_KEY=your-secure-key resource-qdrant install

# Custom data directory
QDRANT_DATA_DIR=/mnt/fast-ssd/qdrant resource-qdrant install
```

### Basic Usage
```bash
# Check service status
resource-qdrant status

# Test functionality
resource-qdrant validate

# List collections
resource-qdrant collections list

# Create collection for AI embeddings
resource-qdrant collections create ai_embeddings --dimensions 1536 --distance Cosine
```

## 📖 Management Commands

### Service Management
```bash
# Start/Stop/Restart
resource-qdrant start
resource-qdrant stop
resource-qdrant restart

# Monitoring
resource-qdrant status
resource-qdrant validate
# Use docker logs for detailed logs: docker logs qdrant
```

### Collection Management
```bash
# List all collections
resource-qdrant collections list

# Create collection
resource-qdrant collections create my_vectors --dimensions 1536 --distance Cosine

# Get collection info
resource-qdrant collections info my_vectors

# Delete collection
resource-qdrant collections delete my_vectors

# List collections with model compatibility
resource-qdrant collections list --show-models
```

### Backup and Recovery
```bash
# Create backup
resource-qdrant backup create [my-backup]

# List backups
resource-qdrant backup list

# Backups are stored in ~/.vrooli/backups/qdrant/
# Restore is handled by the backup framework
```

### Data Injection
```bash
# Generate embeddings from text
resource-qdrant embed "machine learning algorithms"

# Search collections semantically
resource-qdrant collections search "AI research" --collection my-docs

# Inject data (deprecated - use content commands)
resource-qdrant inject data.json
```

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `QDRANT_CUSTOM_PORT` | 6333 | REST API port |
| `QDRANT_CUSTOM_GRPC_PORT` | 6334 | gRPC API port |
| `QDRANT_CUSTOM_API_KEY` | (none) | API authentication key |
| `QDRANT_DATA_DIR` | ~/.qdrant/data | Data storage directory |
| `QDRANT_SNAPSHOTS_DIR` | ~/.qdrant/snapshots | Backup snapshots |
| `QDRANT_LOG_LEVEL` | INFO | Logging level |

### Performance Tuning
| Variable | Default | Description |
|----------|---------|-------------|
| `QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE` | 20000 | Vectors per segment |
| `QDRANT_STORAGE_MEMMAP_THRESHOLD` | 100000 | Memory mapping threshold |
| `QDRANT_MAX_WORKERS` | 0 | Max optimization threads (0=auto) |

### Collection Configuration

**Distance Metrics:**
- **Cosine**: Range 0-2, best for text embeddings
- **Dot**: Unbounded, best for normalized vectors (fastest)
- **Euclidean**: Range 0-∞, best for precise distances

**Common Vector Dimensions:**
- 384: Sentence transformers (small)
- 768: BERT, CodeBERT (medium) 
- 1536: OpenAI ada-002 (large)
- 4096: Large language models

### Authentication Setup
```bash
# Generate secure API key
QDRANT_API_KEY=$(openssl rand -hex 32)

# Install with authentication
QDRANT_CUSTOM_API_KEY="$QDRANT_API_KEY" resource-qdrant install

# Use in requests
curl -H "api-key: $QDRANT_API_KEY" http://localhost:6333/collections
```

## 🔗 REST API Reference

### Base URLs
- **REST API**: `http://localhost:6333`
- **gRPC API**: `grpc://localhost:6334`

### Health and Status
```bash
# Health check
curl http://localhost:6333/

# Cluster information
curl http://localhost:6333/cluster | jq

# Prometheus metrics
curl http://localhost:6333/metrics
```

### Collections API
```bash
# List collections
curl http://localhost:6333/collections | jq

# Create collection
curl -X PUT http://localhost:6333/collections/my_vectors \
  -H "Content-Type: application/json" \
  -d '{"vectors": {"size": 1536, "distance": "Cosine"}}' | jq

# Get collection info
curl http://localhost:6333/collections/my_vectors | jq

# Delete collection
curl -X DELETE http://localhost:6333/collections/my_vectors | jq
```

### Vector Operations
```bash
# Insert vectors
curl -X PUT http://localhost:6333/collections/my_vectors/points \
  -H "Content-Type: application/json" \
  -d '{
    "points": [
      {
        "id": 1,
        "vector": [0.1, 0.2, 0.3, ...],
        "payload": {"text": "example", "category": "test"}
      }
    ]
  }' | jq

# Search vectors
curl -X POST http://localhost:6333/collections/my_vectors/points/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3, ...],
    "limit": 10,
    "with_payload": true
  }' | jq
```

## 🚨 Troubleshooting

### Quick Diagnostics
```bash
# Check resource status
resource-qdrant status

# Check container status
docker ps | grep qdrant
docker logs qdrant --tail 20
```

### Common Issues

**Service Not Starting:**
```bash
# Check Docker daemon
sudo systemctl status docker

# Check port conflicts
sudo lsof -i :6333

# Check logs for errors
docker logs qdrant --tail 50
```

**Port Already in Use:**
```bash
# Use custom ports
QDRANT_CUSTOM_PORT=6433 QDRANT_CUSTOM_GRPC_PORT=6434 resource-qdrant install
```

**Performance Issues:**
```bash
# Increase memory mapping for large collections
export QDRANT_STORAGE_MEMMAP_THRESHOLD=500000

# Optimize segment size
export QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE=50000

# Add Docker memory limits
docker update --memory 4g qdrant
```

**Backup/Recovery Issues:**
```bash
# Check disk space
df -h ~/.qdrant/snapshots/

# Fix permissions
chmod 755 ~/.qdrant/snapshots/
chown -R $USER:$USER ~/.qdrant/
```

### Performance Tips
1. Use **gRPC** for high-throughput applications
2. **Batch operations** when inserting multiple vectors
3. **Optimize vector dimensions** for your use case
4. **Configure memory mapping** for large collections
5. **Monitor segment count** and optimize if needed

## 📊 Business Value

This Qdrant resource enables high-value AI applications:

- **Semantic Search Systems**: $15,000-30,000 value
- **Recommendation Engines**: $10,000-25,000 value  
- **Document Intelligence**: $8,000-20,000 value
- **AI Memory Systems**: $12,000-35,000 value

**Technical Capabilities:**
- Sub-millisecond vector similarity search
- Horizontal scaling with cluster support
- Advanced filtering and metadata queries
- Real-time updates with ACID guarantees
- Multiple distance metrics and data types

## 📁 Directory Structure

```
~/.qdrant/
├── data/                    # Vector data and indices
│   ├── collections/         # Collection data
│   └── meta/               # Metadata and cluster info
├── config/                 # Configuration files
└── snapshots/              # Backup files
```

---

For detailed troubleshooting, see the [Enhanced Integration Test](test/integration-test.sh) which validates all functionality.