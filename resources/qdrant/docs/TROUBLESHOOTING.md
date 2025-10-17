# Troubleshooting Guide

## Common Issues

### Connection Problems

#### Cannot Connect to Qdrant

**Symptoms:**
- `Connection refused` errors
- Timeout when accessing API
- Health check failures

**Diagnosis:**
```bash
# Check if Qdrant is running
docker ps | grep qdrant

# Test connectivity
curl -f http://localhost:6333/health

# Check ports
netstat -tulpn | grep -E "6333|6334"

# Check Docker logs
docker logs qdrant
```

**Solutions:**

1. **Start Qdrant if not running:**
```bash
resource-qdrant manage start
# or
docker start qdrant
```

2. **Fix port conflicts:**
```bash
# Find process using port
lsof -i :6333
# Kill conflicting process or change Qdrant port
QDRANT_CUSTOM_PORT=7333 resource-qdrant manage install
```

3. **Network issues:**
```bash
# Check Docker network
docker network inspect vrooli_network
# Recreate if needed
docker network create vrooli_network
docker network connect vrooli_network qdrant
```

#### Authentication Failures

**Symptoms:**
- `401 Unauthorized` errors
- API key rejection

**Solutions:**
```bash
# Check API key configuration
resource-qdrant credentials

# Update API key
docker exec qdrant sh -c 'echo "YOUR_API_KEY" > /qdrant/config/api_key'
docker restart qdrant
```

### Performance Issues

#### Slow Search Queries

**Symptoms:**
- Search taking >1 second
- Timeouts on large collections
- High CPU usage

**Diagnosis:**
```bash
# Check collection statistics
curl http://localhost:6333/collections/{collection}/

# Monitor resource usage
docker stats qdrant

# Check index configuration
curl http://localhost:6333/collections/{collection} | jq '.result.config'
```

**Solutions:**

1. **Optimize index settings:**
```bash
# Update HNSW parameters
curl -X PATCH "http://localhost:6333/collections/{collection}" \
  -H 'Content-Type: application/json' \
  -d '{
    "hnsw_config": {
      "ef": 128,
      "m": 16
    }
  }'
```

2. **Add more resources:**
```bash
# Increase memory limit
docker update --memory="4g" --memory-swap="4g" qdrant
docker restart qdrant
```

3. **Use filtering to reduce search space:**
```json
{
  "filter": {
    "must": [
      {"key": "type", "match": {"value": "recent"}}
    ]
  },
  "vector": [...],
  "limit": 10
}
```

#### High Memory Usage

**Symptoms:**
- Container using all available memory
- OOM (Out of Memory) kills
- Slow response times

**Solutions:**

1. **Enable disk-based storage:**
```bash
# Configure memmap threshold
docker exec qdrant sh -c 'cat > /qdrant/config/config.yaml << EOF
storage:
  optimizers:
    memmap_threshold_kb: 20000
EOF'
docker restart qdrant
```

2. **Implement vector quantization:**
```bash
# Enable scalar quantization
curl -X PUT "http://localhost:6333/collections/{collection}" \
  -H 'Content-Type: application/json' \
  -d '{
    "vectors": {
      "size": 1536,
      "distance": "Cosine"
    },
    "quantization_config": {
      "scalar": {
        "type": "int8",
        "quantile": 0.99,
        "always_ram": false
      }
    }
  }'
```

### Data Issues

#### Missing or Corrupted Collections

**Symptoms:**
- Collection not found errors
- Inconsistent search results
- Crash on collection access

**Diagnosis:**
```bash
# List all collections
curl http://localhost:6333/collections

# Check collection health
curl http://localhost:6333/collections/{collection}

# Verify data directory
docker exec qdrant ls -la /qdrant/storage/collections/
```

**Solutions:**

1. **Restore from snapshot:**
```bash
# List available snapshots
curl http://localhost:6333/collections/{collection}/snapshots

# Restore snapshot
curl -X PUT "http://localhost:6333/collections/{collection}/snapshots/recover" \
  -H 'Content-Type: application/json' \
  -d '{"location": "/qdrant/snapshots/snapshot_name"}'
```

2. **Rebuild collection:**
```bash
# Delete corrupted collection
resource-qdrant collections delete {collection}

# Recreate and re-index
resource-qdrant collections create {collection} --dimensions 1536
resource-qdrant embeddings refresh
```

#### Duplicate Vectors

**Symptoms:**
- Same content appearing multiple times
- Inconsistent point counts
- Search returning duplicates

**Solutions:**

1. **Use upsert instead of insert:**
```bash
# Upsert with consistent IDs
curl -X PUT "http://localhost:6333/collections/{collection}/points" \
  -H 'Content-Type: application/json' \
  -d '{
    "points": [{
      "id": "content_hash_or_unique_id",
      "vector": [...],
      "payload": {...}
    }]
  }'
```

2. **Clean duplicates:**
```bash
# Find and remove duplicates
resource-qdrant embeddings gc --force
```

### Docker Issues

#### Container Won't Start

**Symptoms:**
- Container exits immediately
- Restart loop
- Permission errors

**Diagnosis:**
```bash
# Check logs
docker logs qdrant --tail 50

# Inspect container
docker inspect qdrant

# Check volume permissions
ls -la ~/.qdrant/
```

**Solutions:**

1. **Fix permissions:**
```bash
# Fix volume ownership
sudo chown -R $(id -u):$(id -g) ~/.qdrant/

# Recreate with correct permissions
docker rm -f qdrant
resource-qdrant manage install
```

2. **Clear corrupted data:**
```bash
# Backup if needed
cp -r ~/.qdrant/data ~/.qdrant/data.backup

# Clear and restart
rm -rf ~/.qdrant/data/*
docker restart qdrant
```

#### Volume Mount Issues

**Symptoms:**
- Data not persisting
- Permission denied errors
- Wrong volume location

**Solutions:**

```bash
# Verify volume mount
docker inspect qdrant | jq '.[0].Mounts'

# Recreate with correct mount
docker rm -f qdrant
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -p 6334:6334 \
  -v ~/.qdrant/data:/qdrant/storage \
  qdrant/qdrant
```

### API Issues

#### Rate Limiting

**Symptoms:**
- `429 Too Many Requests` errors
- Throttled responses
- Intermittent failures

**Solutions:**

1. **Implement client-side rate limiting:**
```bash
# Add delay between requests
for item in "${items[@]}"; do
    process_item "$item"
    sleep 0.1  # 100ms delay
done
```

2. **Batch operations:**
```bash
# Batch insert instead of individual
curl -X PUT "http://localhost:6333/collections/{collection}/points/batch" \
  -H 'Content-Type: application/json' \
  -d '{"points": [/* array of points */]}'
```

#### Payload Size Limits

**Symptoms:**
- `413 Payload Too Large` errors
- Truncated responses
- Memory errors on large queries

**Solutions:**

1. **Reduce payload size:**
```bash
# Store only essential data in payload
{
  "id": 1,
  "vector": [...],
  "payload": {
    "id": "ref_123",
    "type": "doc"
    // Store full content elsewhere
  }
}
```

2. **Use scroll API for large results:**
```bash
# Paginate through results
curl -X POST "http://localhost:6333/collections/{collection}/points/scroll" \
  -H 'Content-Type: application/json' \
  -d '{
    "limit": 100,
    "offset": 0,
    "with_payload": true
  }'
```

## Debugging Tools

### Logging

```bash
# Enable debug logging
docker exec qdrant sh -c 'echo "log_level: DEBUG" >> /qdrant/config/config.yaml'
docker restart qdrant

# View logs
docker logs -f qdrant

# Search for errors
docker logs qdrant 2>&1 | grep -E "ERROR|WARN"
```

### Metrics

```bash
# Get Prometheus metrics
curl http://localhost:6333/metrics

# Collection metrics
curl http://localhost:6333/collections/{collection} | jq '.result.status'

# System telemetry
curl http://localhost:6333/telemetry
```

### Health Checks

```bash
# Basic health
curl http://localhost:6333/health

# Detailed health
curl http://localhost:6333/

# Collection health
for collection in $(curl -s http://localhost:6333/collections | jq -r '.result.collections[].name'); do
    echo "Checking $collection..."
    curl -s "http://localhost:6333/collections/$collection" | jq '.result.status'
done
```

## Recovery Procedures

### Full System Recovery

```bash
# 1. Stop Qdrant
docker stop qdrant

# 2. Backup current data
cp -r ~/.qdrant/data ~/.qdrant/data.backup.$(date +%Y%m%d)

# 3. Clear problematic data
rm -rf ~/.qdrant/data/collections/problematic_collection

# 4. Start Qdrant
docker start qdrant

# 5. Verify health
curl http://localhost:6333/health

# 6. Re-index if needed
resource-qdrant embeddings refresh --force
```

### Emergency Rollback

```bash
# If update causes issues
docker stop qdrant
docker rm qdrant

# Use previous version
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -v ~/.qdrant/data:/qdrant/storage \
  qdrant/qdrant:v1.7.0  # Previous working version

# Verify functionality
resource-qdrant status
```

## Getting Help

### Diagnostic Information to Collect

```bash
# Generate diagnostic bundle
cat > qdrant-diagnostics.sh << 'EOF'
#!/bin/bash
echo "=== Qdrant Diagnostics ==="
echo "Date: $(date)"
echo ""
echo "=== Container Status ==="
docker ps -a | grep qdrant
echo ""
echo "=== Container Logs (last 100 lines) ==="
docker logs qdrant --tail 100
echo ""
echo "=== Collections ==="
curl -s http://localhost:6333/collections | jq
echo ""
echo "=== Health Check ==="
curl -s http://localhost:6333/health
echo ""
echo "=== Resource Usage ==="
docker stats qdrant --no-stream
echo ""
echo "=== Disk Usage ==="
du -sh ~/.qdrant/
EOF

bash qdrant-diagnostics.sh > qdrant-diagnostics.txt
```

### Support Channels

1. **GitHub Issues**: Report bugs and feature requests
2. **Documentation**: Check official Qdrant docs
3. **Community**: Qdrant Discord/Slack channels
4. **Logs**: Always include diagnostic information