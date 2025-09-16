# Apache Kafka Resource PRD

## Executive Summary
**What**: Industry-standard distributed event streaming platform for high-throughput, fault-tolerant messaging
**Why**: Essential infrastructure for real-time data pipelines, event-driven architectures, and microservices communication  
**Who**: All scenarios requiring reliable message queuing, event streaming, or log aggregation
**Value**: Enables $250K+ in scenario value through event-driven architectures and real-time data processing
**Priority**: P0 - Core infrastructure for event streaming

## P0 Requirements (Must Have)
- [x] **v2.0 Contract Compliance**: Full implementation of universal.yaml with all lifecycle hooks ✅ 2025-01-10
- [x] **KRaft Mode Operation**: Run without Zookeeper dependency using native Raft consensus (3.3+) ✅ 2025-01-10
- [x] **Health Check Endpoint**: Respond to health checks within 5 seconds with broker status ✅ 2025-01-10
- [x] **Topic Management**: Create, list, describe, and delete topics via CLI commands ✅ 2025-01-10
- [x] **Producer/Consumer Support**: Enable message production and consumption with built-in tools ✅ 2025-01-10
- [x] **Persistent Storage**: Configure log segments with proper data retention ✅ 2025-01-10
- [x] **Connection Pooling**: Support multiple client connections with configurable limits ✅ 2025-01-10

## P1 Requirements (Should Have)  
- [ ] **Multi-Broker Support**: Enable cluster mode with multiple brokers
- [ ] **Schema Registry Integration**: Support for Avro/JSON schema management
- [ ] **Performance Monitoring**: Expose JMX metrics for monitoring
- [ ] **Security Configuration**: Support for SASL/SSL authentication

## P2 Requirements (Nice to Have)
- [ ] **Kafka UI**: Web interface for visual cluster management
- [ ] **Connect Framework**: Support for Kafka Connect sources/sinks
- [ ] **Stream Processing**: Integration with Kafka Streams API

## Technical Specifications

### Architecture
- **Base Image**: apache/kafka:latest (official Docker image)
- **Mode**: KRaft (Kafka Raft) - no Zookeeper
- **Protocol**: Native Kafka protocol + REST API wrapper
- **Storage**: Volume-mounted log segments
- **Networking**: Configurable listeners for internal/external access

### Dependencies
- **Runtime**: Java 11+ (included in Docker image)
- **Storage**: Local filesystem for log segments
- **Memory**: Minimum 512MB heap, recommended 2GB+
- **CPU**: 2+ cores recommended for production

### Port Allocation
- **Primary Port**: 29092 (Kafka broker)
- **Controller Port**: 29093 (KRaft consensus)  
- **External Port**: 29094 (external client access)
- **JMX Port**: 29099 (metrics)

### CLI Commands (v2.0 Contract)
```bash
# Required commands
resource-kafka help                      # Show usage
resource-kafka info                      # Runtime configuration
resource-kafka manage install            # Install dependencies
resource-kafka manage start              # Start Kafka broker
resource-kafka manage stop               # Stop gracefully
resource-kafka manage restart            # Restart broker
resource-kafka test smoke                # Quick health check
resource-kafka test integration          # Full validation
resource-kafka content add               # Create topics
resource-kafka content list              # List topics
resource-kafka content execute           # Run Kafka commands
resource-kafka status                    # Detailed status
resource-kafka logs                      # View broker logs
```

### Configuration
```yaml
# config/runtime.json
{
  "startup_order": 300,
  "dependencies": [],
  "startup_timeout": 120,
  "startup_time_estimate": "30-60s",
  "recovery_attempts": 3,
  "priority": "high",
  "category": "data",
  "capabilities": ["event-streaming", "message-queue", "pub-sub", "log-aggregation"]
}
```

### Health Check Implementation
```bash
# Quick broker health check
timeout 5 docker exec kafka-broker \
  /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server localhost:29092
```

### API Endpoints
- `GET /health` - Health status (wrapper around broker check)
- `POST /topics` - Create topic
- `GET /topics` - List topics
- `DELETE /topics/{name}` - Delete topic
- `POST /messages` - Produce message
- `GET /messages` - Consume messages

## Success Metrics

### Completion Targets
- **P0 Completion**: 100% ✅ (all must-have features implemented and tested)
- **P1 Completion**: 0% (future enhancement)
- **P2 Completion**: 0% (future enhancement)

### Quality Metrics
- **Health Check Response**: <1 second
- **Startup Time**: <60 seconds
- **Message Throughput**: >10K msg/sec
- **Latency**: <10ms p99
- **Availability**: 99.9% uptime

### Performance Requirements
- **Connection Limit**: 1000+ concurrent clients
- **Topic Limit**: 1000+ topics
- **Partition Limit**: 10000+ partitions
- **Message Size**: Up to 1MB default
- **Retention**: Configurable (default 7 days)

## Integration Points

### Resource Integrations
- **Postgres**: Store message metadata
- **Redis**: Cache hot topics
- **Qdrant**: Index message content
- **MinIO**: Archive old messages
- **N8n**: Workflow triggers
- **Claude-Code**: Event-driven automation

### Scenario Use Cases
- **Event Sourcing**: Store all state changes as events
- **Log Aggregation**: Centralize application logs
- **Stream Processing**: Real-time data transformation
- **Microservices Communication**: Async service messaging
- **Change Data Capture**: Database replication
- **IoT Data Ingestion**: High-volume sensor data

## Security Considerations
- Run broker as non-root user
- Limit exposed ports to minimum required
- Configure authentication for production use
- Enable SSL/TLS for external connections
- Set appropriate file permissions on log directories
- Implement access control lists (ACLs) for topics

## Testing Strategy

### Smoke Tests (30s)
- Broker responds to health checks
- Can create a test topic
- Can produce/consume test message

### Integration Tests (120s)
- Multi-topic operations
- Producer/consumer groups
- Message ordering guarantees
- Partition rebalancing
- Offset management

### Performance Tests
- Throughput benchmarks
- Latency measurements
- Connection scaling
- Resource utilization

## Migration Path
For existing message queue users:
1. Deploy Kafka alongside existing system
2. Dual-write to both systems
3. Migrate consumers gradually
4. Validate data consistency
5. Decommission old system

## Revenue Model
**Direct Value**: 
- Replaces commercial message brokers ($50K/year)
- Enables real-time analytics ($100K value)
- Powers event-driven architectures ($100K value)

**Indirect Value**:
- Reduces system coupling
- Improves scalability
- Enables new business capabilities
- Reduces operational complexity

**Total Platform Value**: $250K+ annually

## Implementation Roadmap

### Phase 1: Scaffold (Completed)
- [x] Create PRD ✅ 2025-01-10
- [x] Implement v2.0 structure ✅ 2025-01-10
- [x] Basic broker deployment ✅ 2025-01-10
- [x] Health check endpoint ✅ 2025-01-10

### Phase 2: Core Features (Completed)
- [x] Topic management CLI ✅ 2025-01-10
- [x] Producer/consumer tools ✅ 2025-01-10
- [x] Persistence configuration ✅ 2025-01-10
- [x] Integration tests ✅ 2025-01-10

### Phase 3: Production Ready (Improver 2)
- [ ] Multi-broker support
- [ ] Security configuration
- [ ] Performance optimization
- [ ] Monitoring integration

### Phase 4: Advanced Features (Improver 3)
- [ ] Schema Registry
- [ ] Kafka Connect
- [ ] Kafka Streams
- [ ] UI Dashboard

## Decision Log
- **2025-01-10**: Chose KRaft mode over Zookeeper for simplicity
- **2025-01-10**: Selected official apache/kafka image for stability
- **2025-01-10**: Prioritized v2.0 contract compliance over features
- **2025-01-10**: Validated all P0 requirements - resource is fully functional
- **2025-01-10**: Achieved 100% P0 completion with passing tests
- **2025-01-16**: Fixed port conflict detection using bash built-in instead of lsof (no sudo required)
- **2025-01-16**: Updated docker stop command to use --timeout instead of deprecated --time flag

## References
- [Apache Kafka Documentation](https://kafka.apache.org/documentation/)
- [KRaft Mode Guide](https://developer.confluent.io/learn/kraft/)
- [Docker Best Practices](https://www.instaclustr.com/education/apache-spark/running-apache-kafka-kraft-on-docker-tutorial-and-best-practices/)
- [Kafka CLI Reference](https://docs.confluent.io/kafka/operations-tools/kafka-tools.html)