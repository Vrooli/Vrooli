# Kafka Resource - Known Issues and Solutions

## Common Problems

### 1. Port Conflicts
**Problem**: Installation fails with "Port already in use"
**Solution**: 
- Check which process is using the port: `lsof -i :29092`
- Either stop the conflicting service or change Kafka ports in environment variables
- Default ports: 29092 (broker), 29093 (controller), 29094 (external)

### 2. Memory Issues
**Problem**: Kafka broker crashes or runs slowly
**Solution**:
- Increase heap memory: `export KAFKA_HEAP_OPTS="-Xmx2G -Xms2G"`
- Monitor memory usage: `docker stats kafka-broker`
- Default is 1GB which may be insufficient for heavy workloads

### 3. Test Failures
**Problem**: Unit tests fail when Kafka is running
**Solution**: 
- Unit tests check port availability and should be run when service is stopped
- Run `./cli.sh manage stop` before running `./cli.sh test unit`
- Integration and smoke tests require the service to be running

### 4. Slow Startup
**Problem**: Kafka takes longer than expected to start
**Solution**:
- Normal startup time is 30-60 seconds for KRaft mode
- Check logs for errors: `./cli.sh logs`
- Ensure sufficient CPU/memory resources available

### 5. Network Connectivity
**Problem**: Cannot connect to Kafka from other containers
**Solution**:
- Ensure containers are on the same Docker network (vrooli-network)
- Use the container name (kafka-broker) for internal connections
- External connections use localhost:29094

## Troubleshooting Commands

```bash
# Check Kafka status
./cli.sh status

# View Kafka logs
./cli.sh logs

# Test broker health
timeout 5 docker exec kafka-broker \
  /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server localhost:29092

# List all topics
docker exec kafka-broker \
  /opt/kafka/bin/kafka-topics.sh \
  --list --bootstrap-server localhost:29092

# Check resource usage
docker stats kafka-broker --no-stream
```

## Performance Tuning

### Message Throughput
- Adjust batch size: Set producer batch.size property
- Tune compression: Use snappy or lz4 for better performance
- Increase partitions for parallel processing

### Storage Optimization
- Configure log retention: `KAFKA_LOG_RETENTION_HOURS`
- Set segment size: `KAFKA_LOG_SEGMENT_BYTES`
- Enable log compaction for key-based topics

## Migration Notes

### From Zookeeper to KRaft
This resource uses KRaft mode (no Zookeeper dependency) which:
- Simplifies deployment (single process)
- Reduces resource usage
- Improves startup time
- Is the future of Kafka (Zookeeper being deprecated)

### Compatibility
- Requires Kafka 3.3+ for stable KRaft support
- Client libraries should support Kafka 2.8+ protocol
- No changes needed for producer/consumer code