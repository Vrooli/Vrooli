# Apache Kafka Resource

Industry-standard distributed event streaming platform for high-throughput, fault-tolerant, and scalable messaging.

## Quick Start

```bash
# Install and start Kafka
vrooli resource kafka manage install
vrooli resource kafka manage start

# Verify health
vrooli resource kafka status

# Create a topic
vrooli resource kafka content add test-topic --partitions 3

# List topics
vrooli resource kafka content list

# Run custom Kafka command
vrooli resource kafka content execute "kafka-topics.sh --list"

# Produce batch messages (high throughput)
vrooli resource kafka content produce-batch test-topic 1000

# View performance metrics
vrooli resource kafka metrics
```

## Features

- **KRaft Mode**: Runs without Zookeeper dependency
- **High Throughput**: Optimized for 8000+ msg/sec performance
- **Fault Tolerant**: Built-in replication and failover
- **Scalable**: Horizontal scaling with partitions
- **Persistent**: Durable message storage with configurable retention
- **Performance Monitoring**: Built-in metrics and resource tracking
- **Batch Operations**: High-throughput batch produce/consume

## Configuration

Default configuration in `config/defaults.sh`:
- **Port**: 29092 (broker)
- **Controller Port**: 29093 (KRaft consensus)
- **External Port**: 29094 (external access)
- **Memory**: 2GB heap (optimized for performance)
- **Storage**: `/var/lib/kafka` (Docker volume)

## Use Cases

- **Event Streaming**: Real-time data pipelines
- **Message Queue**: Reliable async communication
- **Log Aggregation**: Centralized logging
- **Event Sourcing**: Store all state changes
- **Stream Processing**: Real-time data transformation

## CLI Reference

```bash
# Lifecycle management
resource-kafka manage install    # Install Kafka
resource-kafka manage start      # Start broker
resource-kafka manage stop       # Stop broker
resource-kafka manage restart    # Restart broker

# Testing
resource-kafka test smoke        # Quick health check
resource-kafka test integration  # Full validation

# Content management
resource-kafka content add [topic]      # Create topic
resource-kafka content list              # List topics
resource-kafka content get [topic]       # Describe topic
resource-kafka content remove [topic]    # Delete topic
resource-kafka content execute [cmd]     # Run Kafka command
resource-kafka content produce-batch    # Batch message production
resource-kafka content consume-batch    # Batch message consumption

# Monitoring
resource-kafka status            # Detailed status
resource-kafka logs              # View logs
resource-kafka metrics           # Performance metrics
```

## Integration Examples

### With N8n
```javascript
// N8n webhook → Kafka topic
const kafka = require('kafkajs');
// Produce events to Kafka from N8n workflows
```

### With PostgreSQL
```sql
-- Store Kafka offsets in PostgreSQL
CREATE TABLE kafka_offsets (
  consumer_group VARCHAR(255),
  topic VARCHAR(255),
  partition INT,
  offset BIGINT
);
```

### With Redis
```bash
# Cache hot topics in Redis
redis-cli SET "kafka:topic:latest" "message-data"
```

## Troubleshooting

### Broker Not Starting
```bash
# Check logs
resource-kafka logs

# Verify port availability
netstat -tlnp | grep 29092

# Check Docker status
docker ps -a | grep kafka
```

### Connection Issues
```bash
# Test internal connectivity
docker exec kafka-broker kafka-broker-api-versions.sh --bootstrap-server localhost:29092

# Test external connectivity
telnet localhost 29094
```

## Performance Tuning

### Memory Configuration
```bash
# Edit config/defaults.sh
KAFKA_HEAP_OPTS="-Xmx2G -Xms2G"  # Default optimized for 8000+ msg/sec
```

### Batch Operations
```bash
# High-throughput batch production
resource-kafka content produce-batch my-topic 10000 "perf-test"

# Consume messages in batch
resource-kafka content consume-batch my-topic 1000
```

### Storage Optimization
```bash
# Configure retention
KAFKA_LOG_RETENTION_HOURS=168  # 7 days
KAFKA_LOG_SEGMENT_BYTES=1073741824  # 1GB
```

## Security

- Runs as non-root user in container
- Minimal port exposure
- Support for SSL/TLS (configure in defaults.sh)
- ACL support for topic access control

## References

- [Apache Kafka Documentation](https://kafka.apache.org/documentation/)
- [KRaft Configuration](https://developer.confluent.io/learn/kraft/)
- [Kafka CLI Tools](https://docs.confluent.io/kafka/operations-tools/kafka-tools.html)