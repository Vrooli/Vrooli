# QuestDB - Ultra-Fast Time-Series Database

**✅ RESOLVED**: QuestDB port conflict has been fixed. Service is now running on port 9010 (HTTP), 9011 (InfluxDB Line Protocol), and 8812 (PostgreSQL Wire Protocol).

Ultra-high-performance time-series database for AI metrics, resource monitoring, and analytics. QuestDB provides SQL with time-series extensions, achieving 4M+ rows/second ingestion rates with minimal latency queries.

## 🚀 Quick Start

```bash
# Install QuestDB
resource-questdb manage install

# Check status
resource-questdb

# Open web console
resource-questdb console

# Execute a query
resource-questdb query --query "SELECT * FROM tables()"
```

## 🎯 Use Cases

### AI Performance Monitoring
Track model inference times, token usage, costs, and success rates across all AI services.

### Resource Health Tracking
Monitor CPU, memory, response times, and availability of all Vrooli resources in real-time.

### Workflow Analytics
Analyze automation performance, identify bottlenecks, and optimize workflow execution.

### Event Processing
Store and analyze high-frequency events from Vrooli's three-tier AI system.

## 🔌 Multiple Ingestion Protocols

### 1. HTTP REST API (Port 9010)
```bash
# Execute SQL query
curl -G "http://localhost:9010/exec" \
  --data-urlencode "query=SELECT * FROM ai_inference WHERE timestamp > now() - 1h"

# Bulk import CSV
curl -F "data=@metrics.csv" "http://localhost:9010/imp?name=metrics"
```

### 2. PostgreSQL Wire Protocol (Port 8812)
```bash
# Connect with any PostgreSQL client
psql -h localhost -p 8812 -U admin -d qdb

# Use with BI tools, ORMs, or any PostgreSQL-compatible library
```

### 3. InfluxDB Line Protocol (Port 9011)
```bash
# High-speed metric ingestion
echo "cpu,host=server1 usage=45.2 $(date +%s)000000000" | nc localhost 9011

# Batch ingestion
cat metrics.txt | nc localhost 9011
```

## 📊 Default Tables

QuestDB comes pre-configured with tables optimized for Vrooli monitoring:

- **system_metrics** - General system performance metrics
- **ai_inference** - AI model performance and usage tracking
- **resource_health** - Resource availability and health status
- **workflow_metrics** - Automation workflow execution analytics

## 🛠️ Management Commands

### Basic Operations
```bash
# Lifecycle management
resource-questdb manage start      # Start QuestDB
resource-questdb manage stop       # Stop QuestDB
resource-questdb manage restart    # Restart QuestDB
resource-questdb     # Check status
resource-questdb logs       # View logs
resource-questdb logs -f    # Follow logs

# Database operations
resource-questdb tables     # List all tables
resource-questdb query --query "SQL"  # Execute query
```

### Table Management
```bash
# Create table from schema file
resource-questdb tables --table my_metrics --schema schemas/my_metrics.sql

# Get table information
resource-questdb query --query "SHOW COLUMNS FROM ai_inference"
```

## 🔗 Integration Examples

### With n8n Workflows
```javascript
// n8n HTTP Request node
{
  "method": "GET",
  "url": "http://localhost:9010/exec",
  "qs": {
    "query": "SELECT avg(response_time_ms) FROM ai_inference WHERE timestamp > now() - 1h",
    "fmt": "json"
  }
}
```

### With Node-RED
```javascript
// Function node for metric ingestion
msg.payload = `metric,source=nodered value=${msg.payload} ${Date.now()}000000`;
msg.url = "http://localhost:9011/write";
return msg;
```

### With Python
```python
import requests
import time

# REST API
response = requests.get('http://localhost:9010/exec', 
    params={'query': 'SELECT * FROM resource_health ORDER BY timestamp DESC LIMIT 10'})
data = response.json()

# InfluxDB Line Protocol
metric = f"ai_inference,model=gpt4 response_time=250 {int(time.time())}000000000"
requests.post('http://localhost:9011/write', data=metric)
```

## ⚡ Performance Tips

### Ingestion Optimization
- Use InfluxDB Line Protocol for highest ingestion rates
- Batch multiple metrics in single requests
- Configure appropriate partition strategies (default: daily)

### Query Optimization
- Use time-based WHERE clauses to leverage partitioning
- Utilize SAMPLE BY for time-series aggregations
- Create indexes on frequently queried columns

### Resource Tuning
```bash
# Adjust worker counts for your hardware
export QUESTDB_SHARED_WORKER_COUNT=4
export QUESTDB_HTTP_WORKER_COUNT=4

# Increase commit lag for better batching (milliseconds)
export QUESTDB_COMMIT_LAG=600000  # 10 minutes
```

## 📈 Sample Queries

### AI Performance Analytics
```sql
-- Average response time by model (last hour)
SELECT 
    model,
    avg(response_time_ms) as avg_response_time,
    count(*) as request_count
FROM ai_inference
WHERE timestamp > now() - 1h
GROUP BY model;

-- Token usage trends
SELECT 
    timestamp,
    sum(tokens_input + tokens_output) as total_tokens
FROM ai_inference
WHERE timestamp > now() - 24h
SAMPLE BY 1h;
```

### Resource Monitoring
```sql
-- Resources with high response times
SELECT 
    resource_name,
    avg(response_time_ms) as avg_response,
    max(response_time_ms) as max_response
FROM resource_health
WHERE timestamp > now() - 15m
GROUP BY resource_name
HAVING avg_response > 1000;

-- Resource availability over time
SELECT 
    timestamp,
    resource_name,
    CASE WHEN status = 'healthy' THEN 1 ELSE 0 END as is_healthy
FROM resource_health
WHERE timestamp > now() - 1d
SAMPLE BY 5m;
```

## 🔧 Configuration

### Environment Variables
```bash
# Port configuration
export QUESTDB_HTTP_PORT=9010
export QUESTDB_PG_PORT=8812
export QUESTDB_ILP_PORT=9011

# Performance tuning
export QUESTDB_SHARED_WORKER_COUNT=2
export QUESTDB_HTTP_WORKER_COUNT=2
export QUESTDB_WAL_ENABLED=true

# Security
export QUESTDB_PG_USER=admin
export QUESTDB_PG_PASSWORD=quest
```

### Data Persistence
- **Data Directory**: `~/.questdb/data`
- **Config Directory**: `~/.questdb/config`
- **Log Directory**: `~/.questdb/logs`

## 🚨 Troubleshooting

### Container Won't Start
```bash
# Check port availability
lsof -i :9010
lsof -i :8812
lsof -i :9011

# Check Docker logs
resource-questdb logs --tail 50
```

### Query Performance Issues
```bash
# Check table partitioning
resource-questdb query --query "SELECT * FROM tables()"

# Analyze query execution
resource-questdb query --query "EXPLAIN SELECT ..."
```

### High Memory Usage
```bash
# Check current stats
docker stats questdb

# Adjust memory limits in docker-compose.yml
```

## 📚 Resources

- **Web Console**: http://localhost:9010
- **API Documentation**: http://localhost:9010/docs
- **Official Docs**: https://questdb.io/docs/
- **SQL Reference**: https://questdb.io/docs/reference/sql/

## 🔗 Related Resources

- **MinIO**: Object storage for larger artifacts
- **Vault**: Secure credential storage
- **Qdrant**: Vector database for embeddings
- **Node-RED**: Real-time dashboards using QuestDB data
- **n8n**: Automated reporting workflows