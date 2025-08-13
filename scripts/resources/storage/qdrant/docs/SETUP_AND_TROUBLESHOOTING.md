# Qdrant Setup & Troubleshooting Guide

Complete configuration and troubleshooting reference for the Qdrant vector database resource.

## Quick Setup

### Default Installation
```bash
./manage.sh --action install
```

### Custom Configuration
```bash
# Custom ports
QDRANT_CUSTOM_PORT=7333 QDRANT_CUSTOM_GRPC_PORT=7334 ./manage.sh --action install

# With authentication
QDRANT_CUSTOM_API_KEY=your-secure-key ./manage.sh --action install

# Custom data location
QDRANT_DATA_DIR=/mnt/fast-ssd/qdrant ./manage.sh --action install
```

## Configuration Reference

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `QDRANT_CUSTOM_PORT` | 6333 | REST API port |
| `QDRANT_CUSTOM_GRPC_PORT` | 6334 | gRPC API port |
| `QDRANT_CUSTOM_API_KEY` | (none) | API authentication key |
| `QDRANT_DATA_DIR` | ~/.qdrant/data | Data storage directory |
| `QDRANT_CONFIG_DIR` | ~/.qdrant/config | Configuration files |
| `QDRANT_SNAPSHOTS_DIR` | ~/.qdrant/snapshots | Backup snapshots |
| `QDRANT_LOG_LEVEL` | INFO | Logging level |

### Performance Tuning Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE` | 20000 | Vectors per segment |
| `QDRANT_STORAGE_MEMMAP_THRESHOLD` | 100000 | Memory mapping threshold |
| `QDRANT_STORAGE_INDEXING_THRESHOLD` | 20000 | Indexing threshold |
| `QDRANT_MAX_REQUEST_SIZE_MB` | 32 | Maximum request size |
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
QDRANT_CUSTOM_API_KEY="$QDRANT_API_KEY" ./manage.sh --action install

# Use in requests
curl -H "api-key: $QDRANT_API_KEY" http://localhost:6333/collections
```

## Performance Optimization

### Memory Optimization
```bash
# Large collections
export QDRANT_STORAGE_MEMMAP_THRESHOLD=500000
export QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE=50000

# Set Docker memory limits
docker update --memory 4g qdrant
```

### Storage Configuration
```bash
# Custom data directory (must exist and be writable)
export QDRANT_DATA_DIR=/mnt/fast-ssd/qdrant
export QDRANT_SNAPSHOTS_DIR=/mnt/backup/qdrant-snapshots
```

## Troubleshooting

### Quick Diagnostics
```bash
# Run comprehensive diagnostics
./manage.sh --action diagnose

# Check service status
./manage.sh --action status

# Test functionality
./manage.sh --action test

# View logs
./manage.sh --action logs --lines 100
```

### Common Issues

#### Installation Problems

**Issue: Installation Fails**
```bash
# Check Docker status
sudo systemctl status docker
sudo systemctl start docker

# Verify Docker works
docker run hello-world
```

**Issue: Port Already in Use**
```bash
# Find what's using the port
sudo lsof -i :6333

# Use custom ports
QDRANT_CUSTOM_PORT=6433 QDRANT_CUSTOM_GRPC_PORT=6434 ./manage.sh --action install
```

**Issue: Permission Denied on Data Directory**
```bash
# Fix permissions
chmod 755 ~/.qdrant
chown -R $USER:$USER ~/.qdrant/

# Or use custom directory
export QDRANT_DATA_DIR=/tmp/qdrant-data
./manage.sh --action install
```

#### Service Not Starting

**Container Exits Immediately:**
```bash
# Check logs
./manage.sh --action logs --lines 50

# Common fixes:
# 1. Insufficient memory
docker update --memory 2g qdrant

# 2. Port conflicts
ss -tlnp | grep -E ':6333|:6334'

# 3. Data corruption - backup and reinstall
cp -r ~/.qdrant/data ~/.qdrant/data.backup
./manage.sh --action uninstall --remove-data yes
./manage.sh --action install
```

#### API Not Responding

**Connection Refused:**
```bash
# Check container status
docker ps | grep qdrant

# Check container health
docker inspect qdrant | jq '.[0].State.Health'

# Restart service
./manage.sh --action restart

# Test with different endpoint
curl -v http://localhost:6333/
```

**API Returns Errors:**
```bash
# Check detailed logs
./manage.sh --action logs --lines 100

# Test basic endpoints
curl http://localhost:6333/
curl http://localhost:6333/cluster

# Verify API key if using authentication
curl -H "api-key: $QDRANT_API_KEY" http://localhost:6333/collections
```

#### Performance Issues

**Slow Search Queries:**
```bash
# Check collection statistics
./manage.sh --action index-stats --collection your_collection

# Monitor resource usage
docker stats qdrant --no-stream

# Optimize collection configuration
./manage.sh --action delete-collection --collection old_collection
./manage.sh --action create-collection \
  --collection optimized_collection \
  --vector-size 768 \
  --distance Dot  # Faster than Cosine for normalized vectors
```

**High Memory Usage:**
```bash
# Enable memory mapping for large collections
export QDRANT_STORAGE_MEMMAP_THRESHOLD=100000

# Reduce in-memory cache
export QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE=5000

# Add memory limits
docker update --memory 4g --memory-swap 6g qdrant
```

#### Collection Issues

**Collection Creation Fails:**
```bash
# Check existing collections
./manage.sh --action list-collections

# Delete existing collection
./manage.sh --action delete-collection --collection name

# Vector dimension mismatch
./manage.sh --action collection-info --collection your_collection
```

**Search Returns No Results:**
```bash
# Check if collection has data
curl http://localhost:6333/collections/your_collection | jq '.result.points_count'

# Test with higher limit and lower threshold
curl -X POST http://localhost:6333/collections/your_collection/points/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, ...],
    "limit": 100,
    "score_threshold": 0.0
  }'
```

#### Backup and Recovery Issues

**Backup Creation Fails:**
```bash
# Check available disk space
df -h ~/.qdrant/snapshots/

# Check permissions
ls -la ~/.qdrant/snapshots/

# Fix permissions
chmod 755 ~/.qdrant/snapshots/
chown -R $USER:$USER ~/.qdrant/
```

**Recovery Fails:**
```bash
# Stop Qdrant before recovery
./manage.sh --action stop

# Manual recovery
./manage.sh --action uninstall --remove-data yes
./manage.sh --action install
./manage.sh --action restore --snapshot-name backup-name
```

## Advanced Diagnostics

### Container Diagnostics
```bash
# Check container configuration
docker inspect qdrant | jq '.[0].Config'

# Check resource limits
docker inspect qdrant | jq '.[0].HostConfig.Memory'

# Execute commands inside container
docker exec -it qdrant sh
```

### Network Diagnostics
```bash
# Check Docker networks
docker network ls
docker network inspect qdrant-network

# Test network connectivity
docker exec qdrant ping host.docker.internal
```

### Performance Profiling
```bash
# Monitor real-time stats
watch -n 1 'docker stats qdrant --no-stream'

# Profile API calls
time curl http://localhost:6333/collections
```

## Log Analysis

### Common Log Messages

**Normal Startup:**
```
INFO qdrant: Service is listening on http://0.0.0.0:6333
INFO qdrant: gRPC service is listening on 0.0.0.0:6334
```

**Common Errors:**
```
ERROR qdrant: Collection 'name' not found
ERROR qdrant: Vector dimension mismatch: expected 1536, got 768
ERROR qdrant: Failed to bind to port 6333: Address already in use
```

### Log Filtering
```bash
# Show only errors
./manage.sh --action logs | grep ERROR

# Monitor logs in real-time
./manage.sh --action logs --follow
```

## Recovery Procedures

### Complete Reset
```bash
# 1. Stop everything
./manage.sh --action stop

# 2. Backup current state (optional)
cp -r ~/.qdrant ~/.qdrant.backup.$(date +%Y%m%d)

# 3. Complete uninstall
./manage.sh --action uninstall --remove-data yes

# 4. Fresh installation
./manage.sh --action install
```

### Partial Reset (Preserve Data)
```bash
# 1. Stop service
./manage.sh --action stop

# 2. Remove container only
docker rm qdrant

# 3. Reinstall (keeps data)
./manage.sh --action install
```

## Getting Help

### Create Debug Report
```bash
cat > debug_report.txt << EOF
=== Qdrant Debug Report ===
Date: $(date)
System: $(uname -a)
Docker Version: $(docker --version)

=== Service Status ===
$(./manage.sh --action status 2>&1)

=== Diagnostics ===
$(./manage.sh --action diagnose 2>&1)

=== Recent Logs ===
$(./manage.sh --action logs --lines 50 2>&1)

=== Container Info ===
$(docker inspect qdrant 2>&1)

=== System Resources ===
Memory: $(free -h)
Disk: $(df -h ~/.qdrant)
EOF

echo "Debug report saved to debug_report.txt"
```

### Support Resources
- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [GitHub Issues](https://github.com/qdrant/qdrant/issues)
- [Discord Community](https://discord.gg/qdrant)
- [Stack Overflow](https://stackoverflow.com/questions/tagged/qdrant)

---

For API reference and detailed usage examples, see [API.md](API.md).