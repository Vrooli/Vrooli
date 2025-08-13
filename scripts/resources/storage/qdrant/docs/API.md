# Qdrant API Reference

Complete API reference for the Qdrant vector database resource.

## Base URLs

- **REST API**: `http://localhost:6333`
- **gRPC API**: `grpc://localhost:6334`

## Authentication

If API key is configured during installation:
```bash
# All requests require api-key header
curl -H "api-key: your-api-key" http://localhost:6333/collections
```

For authentication setup, see [CONFIGURATION.md](CONFIGURATION.md#authentication-configuration).

## CLI Management Commands

### Service Management
```bash
# Install Qdrant
./manage.sh --action install

# Start/Stop/Restart
./manage.sh --action start
./manage.sh --action stop
./manage.sh --action restart

# Status and health
./manage.sh --action status
./manage.sh --action test
./manage.sh --action diagnose
./manage.sh --action monitor --interval 10
```

### Collection Management
```bash
# List all collections
./manage.sh --action list-collections

# Create collection
./manage.sh --action create-collection --collection my_vectors --vector-size 1536 --distance Cosine

# Get collection info
./manage.sh --action collection-info --collection my_vectors

# Delete collection
./manage.sh --action delete-collection --collection my_vectors [--force yes]

# Collection statistics
./manage.sh --action index-stats [--collection my_vectors]
```

### Backup and Recovery
```bash
# Create backup
./manage.sh --action backup [--snapshot-name my-backup]

# List backups
./manage.sh --action list-backups

# Get backup info
./manage.sh --action backup-info --snapshot-name my-backup

# Restore from backup
./manage.sh --action restore --snapshot-name my-backup
```

## REST API Endpoints

### Health and Status

#### GET `/`
Health check endpoint
```bash
curl http://localhost:6333/
```

#### GET `/cluster`
Cluster information
```bash
curl http://localhost:6333/cluster | jq
```

#### GET `/telemetry`
Service telemetry data
```bash
curl http://localhost:6333/telemetry | jq
```

#### GET `/metrics`
Prometheus metrics
```bash
curl http://localhost:6333/metrics
```

### Collections

#### GET `/collections`
List all collections
```bash
curl http://localhost:6333/collections | jq
```

#### PUT `/collections/{collection_name}`
Create collection
```bash
curl -X PUT http://localhost:6333/collections/my_vectors \
  -H "Content-Type: application/json" \
  -d '{
    "vectors": {
      "size": 1536,
      "distance": "Cosine"
    }
  }' | jq
```

#### GET `/collections/{collection_name}`
Get collection information
```bash
curl http://localhost:6333/collections/my_vectors | jq
```

#### DELETE `/collections/{collection_name}`
Delete collection
```bash
curl -X DELETE http://localhost:6333/collections/my_vectors | jq
```

### Vector Operations

#### PUT `/collections/{collection_name}/points`
Insert/Update vectors
```bash
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
```

#### POST `/collections/{collection_name}/points/search`
Search for similar vectors
```bash
curl -X POST http://localhost:6333/collections/my_vectors/points/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3, ...],
    "limit": 10,
    "with_payload": true,
    "with_vector": false
  }' | jq
```

#### POST `/collections/{collection_name}/points/recommend`
Get recommendations based on positive/negative examples
```bash
curl -X POST http://localhost:6333/collections/my_vectors/points/recommend \
  -H "Content-Type: application/json" \
  -d '{
    "positive": [1, 2],
    "negative": [3],
    "limit": 10,
    "with_payload": true
  }' | jq
```

#### GET `/collections/{collection_name}/points/{point_id}`
Get specific point by ID
```bash
curl http://localhost:6333/collections/my_vectors/points/1 | jq
```

#### DELETE `/collections/{collection_name}/points/{point_id}`
Delete specific point
```bash
curl -X DELETE http://localhost:6333/collections/my_vectors/points/1 | jq
```

### Advanced Filtering

#### Search with payload filter
```bash
curl -X POST http://localhost:6333/collections/my_vectors/points/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3, ...],
    "limit": 10,
    "filter": {
      "must": [
        {"key": "category", "match": {"value": "test"}}
      ]
    },
    "with_payload": true
  }' | jq
```

#### Available filter conditions
- `match`: Exact match
- `range`: Numeric range
- `geo_bounding_box`: Geographic bounds
- `geo_radius`: Geographic radius
- `values_count`: Count of values

### Batch Operations

#### Batch Insert
```bash
curl -X PUT http://localhost:6333/collections/my_vectors/points \
  -H "Content-Type: application/json" \
  -d '{
    "points": [
      {"id": 1, "vector": [0.1, 0.2, 0.3], "payload": {"text": "first"}},
      {"id": 2, "vector": [0.4, 0.5, 0.6], "payload": {"text": "second"}},
      {"id": 3, "vector": [0.7, 0.8, 0.9], "payload": {"text": "third"}}
    ]
  }'
```

#### Batch Delete
```bash
curl -X POST http://localhost:6333/collections/my_vectors/points/delete \
  -H "Content-Type: application/json" \
  -d '{
    "points": [1, 2, 3]
  }'
```

## Response Formats

### Standard Response
```json
{
  "result": {
    // Response data
  },
  "status": "ok",
  "time": 0.001
}
```

### Collection Info Response
```json
{
  "result": {
    "status": "green",
    "vectors_count": 1000,
    "indexed_vectors_count": 1000,
    "points_count": 1000,
    "segments_count": 1,
    "config": {
      "params": {
        "vectors": {
          "size": 1536,
          "distance": "Cosine"
        }
      }
    }
  }
}
```

### Search Results Response
```json
{
  "result": [
    {
      "id": 1,
      "version": 1,
      "score": 0.95,
      "payload": {"text": "example", "category": "test"},
      "vector": null
    }
  ]
}
```

## Distance Metrics

- **Cosine**: Best for similarity (normalized vectors)
- **Dot**: Fast for positive vectors  
- **Euclid**: Traditional euclidean distance

## Error Handling

```json
{
  "status": "error",
  "error": "Collection not found",
  "time": 0.001
}
```

Common HTTP status codes:
- `200`: Success
- `400`: Bad request (invalid data)
- `404`: Not found (collection/point)
- `409`: Conflict (collection already exists)
- `422`: Unprocessable entity (validation error)

## gRPC API

For high-performance applications, use the gRPC endpoint at `grpc://localhost:6334`.

See [Qdrant gRPC documentation](https://qdrant.tech/documentation/guides/qdrant_api_reference/#grpc-reference) for protocol buffer definitions.

## Performance Tips

1. **Use gRPC** for high-throughput applications
2. **Batch operations** when inserting multiple vectors
3. **Optimize vector dimensions** for your use case
4. **Use filtering** to narrow search space
5. **Configure memory mapping** for large collections (see CONFIGURATION.md)
6. **Monitor segment count** and optimize if needed

---

For configuration, troubleshooting, and setup details, see [CONFIGURATION.md](CONFIGURATION.md).
For detailed API documentation, visit the [official Qdrant API docs](https://qdrant.github.io/qdrant/redoc/index.html).