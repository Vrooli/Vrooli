# Splink Resource - Known Issues and Solutions

## Issues Resolved (2025-09-16)

### 1. Redis Connection Error Loop
**Problem**: Stream processor continuously tried to connect to Redis, causing error spam in logs
**Symptoms**: Logs filled with "Error connecting to localhost:6380" messages
**Root Cause**: 
- Incorrect default port (6379 instead of 6380)
- No timeout or retry limit on Redis connection
- Stream processor didn't handle Redis unavailability gracefully

**Solution**:
- Fixed Redis port configuration in `stream_processor.py`
- Added connection timeout and proper error handling
- Made stream processing work without Redis (in-memory mode)

### 2. Test Suite Container Name Mismatch
**Problem**: Smoke tests failed because they looked for wrong container name
**Symptoms**: Health check tests reported as failed despite service working
**Root Cause**: Test looked for "splink" but container was named "vrooli-splink"

**Solution**: Updated test script to check for correct container name

### 3. Visualization Endpoint Hanging
**Problem**: Visualization endpoints would hang when accessed
**Symptoms**: Requests to `/visualization/job/*` would timeout
**Root Cause**: Heavy computation in synchronous context without proper async handling

**Solution**: Issue resolved after container restart with updated dependencies

## Known Limitations

### 1. Stream Processing Requires Redis
- Stream processing features only work when Redis resource is running
- Falls back to in-memory processing when Redis unavailable
- To enable full stream features: `vrooli resource redis develop`

### 2. Spark Processing Requires Cluster
- Spark backend requires external Spark cluster for full performance
- Falls back to local mode or DuckDB for smaller datasets
- Configuration via SPARK_MASTER_URL environment variable

### 3. Memory Usage with Large Datasets
- DuckDB backend limited by available RAM
- Recommend using Spark backend for datasets >10M records
- Monitor memory usage with `docker stats vrooli-splink`

## Performance Considerations

1. **Batch Size Tuning**: Adjust batch_size based on dataset characteristics
2. **Blocking Rules**: Use appropriate blocking to reduce comparison space
3. **Threshold Selection**: Higher thresholds = fewer false positives but may miss matches

## Troubleshooting Guide

### Service Won't Start
```bash
# Check if port 8096 is already in use
lsof -i :8096

# Restart the service
vrooli resource splink stop
vrooli resource splink develop
```

### High Memory Usage
```bash
# Check current usage
docker stats vrooli-splink

# Restart with memory limit
docker update --memory="4g" --memory-swap="4g" vrooli-splink
docker restart vrooli-splink
```

### Slow Processing
1. Check if using appropriate backend (DuckDB vs Spark)
2. Review blocking rules efficiency
3. Consider adjusting batch_size parameter
4. Monitor CPU usage during processing

## Future Improvements

1. Add connection pooling for PostgreSQL
2. Implement caching layer for frequent queries
3. Add support for incremental learning
4. Optimize memory usage for large result sets
5. Add Kafka support for stream processing