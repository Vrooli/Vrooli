# Redis Resource for Vrooli

A comprehensive Redis resource implementation for Vrooli's automation ecosystem, providing high-performance in-memory data storage with multi-tenant support for client project isolation.

## 🎯 Use Cases

**Primary Applications:**
- **Cache Layer**: High-speed data caching for applications and workflows
- **Session Storage**: User session management for web applications
- **Queue Management**: Task queues and job processing
- **Real-time Analytics**: Fast counters, metrics, and real-time data
- **Pub/Sub Messaging**: Event-driven communication between services
- **Client Project Isolation**: Separate Redis instances for different client builds

**When to Use Redis:**
- Need sub-millisecond data access times
- Building real-time applications or dashboards
- Implementing distributed caching layers
- Managing user sessions across multiple services
- Processing high-volume time-series data
- Building event-driven architectures

**Alternatives:**
- PostgreSQL for relational data and ACID compliance
- MinIO for large file storage
- QuestDB for time-series analytics
- Qdrant for vector similarity search

---

## 🚀 Quick Start

### Installation
```bash
# Install Redis resource
./scripts/resources/storage/redis/manage.sh --action install

# Check status
./scripts/resources/storage/redis/manage.sh --action status

# Start monitoring
./scripts/resources/storage/redis/manage.sh --action monitor
```

### Basic Usage
```bash
# Connect to Redis CLI
redis-cli -p 6380

# Or use the resource CLI wrapper
./scripts/resources/storage/redis/manage.sh --action cli

# Basic operations
redis-cli -p 6380 SET mykey "Hello Redis"
redis-cli -p 6380 GET mykey
redis-cli -p 6380 INCR counter
```

### Connection Details
- **Host**: localhost
- **Port**: 6380 (default resource port)
- **Connection URL**: `redis://localhost:6380`
- **Databases**: 0-15 (configurable)
- **Password**: None by default (configurable)

---

## 🔧 Configuration

### Default Configuration
```bash
# Port Configuration (avoids conflict with internal Redis on 6379)
REDIS_PORT=6380

# Memory and Performance
REDIS_MAX_MEMORY="2gb"
REDIS_MAX_MEMORY_POLICY="allkeys-lru"

# Persistence Options
REDIS_PERSISTENCE="rdb"  # Options: rdb, aof, both, none
REDIS_DATABASES=16

# Security (empty by default for development)
REDIS_PASSWORD=""
```

### Customizing Configuration
```bash
# Update memory limit
./manage.sh --action config --memory "4gb"

# Change persistence mode
./manage.sh --action config --persistence "both"

# Apply both changes and restart
./manage.sh --action config --memory "4gb" --persistence "aof"
```

### Persistence Modes
- **RDB**: Point-in-time snapshots (default)
- **AOF**: Append-only file for maximum durability
- **Both**: RDB + AOF for best of both worlds
- **None**: No persistence (fastest, data lost on restart)

---

## 🛠️ Management Commands

### Lifecycle Management
```bash
# Installation and setup
./manage.sh --action install                    # Install Redis resource
./manage.sh --action uninstall                  # Remove container only
./manage.sh --action uninstall --remove-data yes # Remove container and data
./manage.sh --action upgrade                     # Upgrade to latest version

# Container operations
./manage.sh --action start                       # Start Redis
./manage.sh --action stop                        # Stop Redis
./manage.sh --action restart                     # Restart Redis
./manage.sh --action status                      # Show detailed status
```

### Monitoring and Diagnostics
```bash
# Real-time monitoring
./manage.sh --action monitor                     # Default 5-second intervals
./manage.sh --action monitor --interval 1       # 1-second intervals

# Logs and diagnostics
./manage.sh --action logs                        # Show recent logs
./manage.sh --action logs --lines 100           # Show 100 log lines
./manage.sh --action diagnose                    # Full diagnostic report

# Performance testing
./manage.sh --action benchmark                   # Run Redis benchmark
```

### Data Management
```bash
# Backup operations
./manage.sh --action backup                      # Create timestamped backup
./manage.sh --action backup --backup-name "pre-deploy" # Named backup
./manage.sh --action list-backups                # List all backups
./manage.sh --action restore --backup-name "pre-deploy" # Restore backup

# Database operations
./manage.sh --action flush --database 0          # Flush database 0
./manage.sh --action flush --database all        # Flush all databases
./manage.sh --action cli --database 1            # Connect to database 1
```

### Multi-Tenant Client Management
```bash
# Create isolated client instance
./manage.sh --action create-client --client-id "acme-corp"

# List all client instances
./manage.sh --action list-clients

# Remove client instance
./manage.sh --action destroy-client --client-id "acme-corp"
```

---

## 🤖 Multi-Tenant Architecture

### Client Instance Isolation
Each client gets their own isolated Redis instance with:
- **Dedicated Port**: Automatically allocated from range 6381-6399
- **Isolated Network**: Separate Docker network per client
- **Private Data**: Completely separate data directories
- **Independent Config**: Client-specific Redis configuration

### Creating Client Instances
```bash
# Create client instance
./manage.sh --action create-client --client-id "real-estate-client"

# Output provides connection details:
#   Client: real-estate-client
#   Port: 6382
#   URL: redis://localhost:6382
#   Container: vrooli-client-real-estate-client
```

### Client Instance Management
```bash
# List all client instances with status
./manage.sh --action list-clients

# Example output:
#   Client: acme-corp
#     Container: vrooli-client-acme-corp
#     Port: 6381
#     Status: running
#     Health: healthy
#     URL: redis://localhost:6381
```

### Integration with Client Project Builder
```bash
# Automated client environment creation
./scripts/resources/index.sh --action create-client-env \
  --client "e-commerce-store" \
  --resources "redis,postgres,minio"

# This creates isolated instances of all specified resources
# with coordinated networking and configuration
```

---

## 📊 Performance and Monitoring

### Key Metrics
The status and monitor commands display:
- **Memory Usage**: Current and peak memory consumption
- **Connected Clients**: Active client connections
- **Commands Processed**: Total and per-second operation counts
- **Cache Hit Rate**: Effectiveness of caching operations
- **Database Activity**: Keys per database, expiration stats

### Performance Tuning
```bash
# Memory optimization
REDIS_MAX_MEMORY="4gb"                    # Set memory limit
REDIS_MAX_MEMORY_POLICY="allkeys-lru"    # Eviction policy

# Persistence optimization
REDIS_PERSISTENCE="rdb"                   # Faster restarts
REDIS_SAVE_INTERVALS="900 1 300 10"      # Custom save points

# Network optimization
REDIS_TCP_BACKLOG=511                     # Connection queue size
REDIS_TCP_KEEPALIVE=300                   # Keep-alive interval
```

### Monitoring Integration
```bash
# Redis info for external monitoring
redis-cli -p 6380 INFO > redis_metrics.txt

# Integration with Node-RED monitoring
# The resource automatically exposes metrics at:
# http://localhost:1880/api/resources/redis/metrics
```

---

## 🔗 Integration Examples

### n8n Workflow Integration
```javascript
// n8n Redis node configuration
{
  "host": "localhost",
  "port": 6380,
  "database": 0,
  "operation": "set",
  "key": "workflow_state",
  "value": "{{$json.status}}"
}
```

### Node-RED Flow Integration
```javascript
// Node-RED Redis node
{
  "server": "127.0.0.1",
  "port": "6380",
  "command": "get",
  "topic": "sensor_data",
  "database": "2"
}
```

### Python Application Integration
```python
import redis

# Connect to Vrooli Redis resource
r = redis.Redis(host='localhost', port=6380, db=0)

# Basic operations
r.set('user:123:session', '{"id": 123, "active": true}')
session_data = r.get('user:123:session')

# Use as cache
r.setex('expensive_query', 3600, query_result)  # Cache for 1 hour
```

### Node.js Application Integration
```javascript
const redis = require('redis');

// Connect to Vrooli Redis resource
const client = redis.createClient({
  host: 'localhost',
  port: 6380,
  db: 0
});

// Pub/Sub messaging
client.subscribe('notifications');
client.on('message', (channel, message) => {
  console.log(`Received: ${message} on ${channel}`);
});
```

---

## 🔒 Security Configuration

### Password Protection
```bash
# Set password (requires restart)
export REDIS_PASSWORD="your-secure-password"
./manage.sh --action restart

# Connect with password
redis-cli -p 6380 -a "your-secure-password"
```

### Network Security
```bash
# Bind to specific interface only
REDIS_BIND="127.0.0.1"  # Localhost only
REDIS_BIND="0.0.0.0"    # All interfaces (default for containers)

# Enable protected mode
REDIS_PROTECTED_MODE="yes"  # Requires password or bind restrictions
```

### Production Security Checklist
- [ ] Set strong password via `REDIS_PASSWORD`
- [ ] Configure appropriate `REDIS_BIND` settings
- [ ] Enable `REDIS_PROTECTED_MODE`
- [ ] Use TLS/SSL for external connections
- [ ] Implement proper firewall rules
- [ ] Regular backup schedule
- [ ] Monitor access logs

---

## 💾 Backup and Recovery

### Automatic Backups
```bash
# Create backup with timestamp
./manage.sh --action backup

# Create named backup
./manage.sh --action backup --backup-name "before-migration"

# List all backups
./manage.sh --action list-backups
```

### Backup Types
1. **RDB Snapshots**: Binary point-in-time snapshots
   - Fast to create and restore
   - Compact file size
   - May lose recent changes

2. **Data Directory Archives**: Complete data folder backup
   - Includes all persistence files
   - Larger file size
   - More comprehensive

### Recovery Operations
```bash
# Restore from backup
./manage.sh --action restore --backup-name "before-migration"

# Restore with confirmation bypass
./manage.sh --action restore --backup-name "backup-name" --yes yes

# Clean old backups (keep 30 days)
./manage.sh --action cleanup-backups --days 30
```

---

## 🧪 Testing and Validation

### Health Checks
```bash
# Quick health check
./manage.sh --action status

# Comprehensive diagnostics
./manage.sh --action diagnose

# Automated health monitoring
docker exec vrooli-redis-resource redis-cli ping  # Should return PONG
```

### Performance Testing
```bash
# Built-in benchmark
./manage.sh --action benchmark

# Custom benchmark
redis-cli -p 6380 --latency -i 1    # Latency monitoring
redis-cli -p 6380 --stat -i 1       # Statistics monitoring
```

### Data Validation
```bash
# Test basic operations
redis-cli -p 6380 SET test_key "test_value"
redis-cli -p 6380 GET test_key       # Should return "test_value"
redis-cli -p 6380 DEL test_key

# Test persistence (restart and check)
redis-cli -p 6380 SET persist_test "data"
./manage.sh --action restart
redis-cli -p 6380 GET persist_test   # Should return "data"
```

---

## 🔧 Troubleshooting

### Common Issues

**Redis won't start:**
```bash
# Check port conflicts
./manage.sh --action diagnose
netstat -tuln | grep 6380

# Check logs
./manage.sh --action logs --lines 50

# Try different port
export REDIS_PORT=6381
./manage.sh --action restart
```

**Connection refused:**
```bash
# Verify Redis is running
./manage.sh --action status

# Check network connectivity
telnet localhost 6380

# Check Docker network
docker network ls | grep vrooli
```

**Performance issues:**
```bash
# Check memory usage
./manage.sh --action monitor

# Review configuration
./manage.sh --action config

# Run benchmark
./manage.sh --action benchmark
```

**Data loss concerns:**
```bash
# Check persistence configuration
redis-cli -p 6380 CONFIG GET save
redis-cli -p 6380 CONFIG GET appendonly

# Create immediate backup
./manage.sh --action backup --backup-name "emergency-backup"

# Force save
redis-cli -p 6380 BGSAVE
```

### Log Analysis
```bash
# Redis logs location
tail -f ~/.vrooli/redis/logs/redis.log

# Container logs
docker logs vrooli-redis-resource -f

# System resource usage
docker stats vrooli-redis-resource
```

---

## 🔗 Resource Integration

### With Other Vrooli Resources

**+ Ollama (AI Processing)**
```bash
# Use Redis for AI model caching
redis-cli -p 6380 SET "model:llama3.1:cache:prompt_hash" "cached_response"
```

**+ n8n (Workflow Automation)**
```bash
# Store workflow state and intermediate results
# Use Redis nodes in n8n workflows for:
# - Session management
# - Rate limiting
# - Temporary data storage
```

**+ MinIO (Object Storage)**
```bash
# Use Redis for metadata caching
redis-cli -p 6380 HSET "file:metadata:abc123" "size" "1024" "type" "image/png"
```

**+ PostgreSQL (Relational Data)**
```bash
# Use Redis as PostgreSQL query cache
# Store frequently accessed data for faster retrieval
```

### Port Registry Integration
The Redis resource automatically registers its ports with the Vrooli port registry:
- Main instance: 6380
- Client instances: 6381-6399 (dynamically allocated)

---

## 📚 Advanced Usage

### Lua Scripting
```bash
# Execute Lua scripts for atomic operations
redis-cli -p 6380 EVAL "return redis.call('incr', KEYS[1])" 1 counter
```

### Pub/Sub Messaging
```bash
# Publisher
redis-cli -p 6380 PUBLISH notifications "System update completed"

# Subscriber (in another terminal)
redis-cli -p 6380 SUBSCRIBE notifications
```

### Streams and Time Series
```bash
# Add to stream
redis-cli -p 6380 XADD sensor_data * temperature 23.5 humidity 45

# Read from stream
redis-cli -p 6380 XREAD STREAMS sensor_data 0
```

### Redis Modules
To extend Redis with additional capabilities, modify the Docker command in `lib/docker.sh` to include module loading.

---

## 🎯 Best Practices

### Development
- Use database numbers (0-15) to separate application data
- Implement proper key naming conventions
- Set appropriate TTLs for temporary data
- Monitor memory usage and set limits

### Production
- Enable persistence appropriate for your use case
- Set up regular automated backups
- Monitor performance metrics
- Implement proper security measures
- Use connection pooling in applications

### Multi-Tenant Usage
- Create separate client instances for isolation
- Use consistent naming conventions for client IDs
- Monitor resource usage per client
- Implement cleanup procedures for completed projects

---

**🎯 The Redis resource provides high-performance in-memory storage that perfectly complements Vrooli's automation ecosystem, enabling fast data access for AI workflows, caching layers, and real-time applications while supporting complete client project isolation.**