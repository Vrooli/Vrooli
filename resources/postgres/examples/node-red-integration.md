# PostgreSQL Integration with Node-RED

This guide shows how to connect Node-RED flows to PostgreSQL instances for monitoring and data collection.

## Prerequisites

- Node-RED resource running
- PostgreSQL instance with node-red-network access
- node-red-contrib-postgresql installed in Node-RED

## Creating a PostgreSQL Instance for Node-RED

### Option 1: Use Development Template
```bash
./manage.sh --action create --instance monitoring-db --template development
```

### Option 2: Testing Template (includes more networks)
```bash
./manage.sh --action create --instance monitoring-db --template testing
```

## Getting Connection Details

```bash
./manage.sh --action connect --instance monitoring-db --format node-red
```

Output:
```
PostgreSQL Configuration for Node-RED:

Host: vrooli-postgres-monitoring-db
Port: 5432 (internal) / 5445 (external)
Database: vrooli_client
Username: vrooli
Password: your-password-here

For node-red-contrib-postgresql:
{
  "host": "vrooli-postgres-monitoring-db",
  "port": 5432,
  "database": "vrooli_client"
}
```

## Configuring Node-RED PostgreSQL Nodes

### Installing PostgreSQL Nodes

1. Open Node-RED palette manager
2. Search for `node-red-contrib-postgresql`
3. Install the package

### Creating Database Connection

1. Drag a PostgreSQL node onto the flow
2. Double-click to configure
3. Add new PostgreSQL configuration:
   - **Host**: `vrooli-postgres-monitoring-db`
   - **Port**: `5432`
   - **Database**: `vrooli_client`
   - **User**: `vrooli`
   - **Password**: (from connection details)
   - **Name**: "Client Monitoring DB"

## Example Monitoring Flows

### 1. Resource Usage Logger

```json
[
  {
    "id": "timer-node",
    "type": "inject",
    "repeat": 60,
    "name": "Every Minute"
  },
  {
    "id": "collect-metrics",
    "type": "function",
    "func": "msg.payload = {\n  timestamp: new Date(),\n  cpu_usage: Math.random() * 100,\n  memory_usage: Math.random() * 100\n};\nreturn msg;"
  },
  {
    "id": "postgres-insert",
    "type": "postgresql",
    "query": "INSERT INTO metrics (timestamp, cpu, memory) VALUES ($timestamp, $cpu_usage, $memory_usage)"
  }
]
```

### 2. Real-time Dashboard Query

Query latest metrics for dashboard display:

```sql
SELECT * FROM metrics 
ORDER BY timestamp DESC 
LIMIT 100
```

## Setting Up Tables

Create monitoring tables using Node-RED:

1. Add a PostgreSQL node with this query:
```sql
CREATE TABLE IF NOT EXISTS metrics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    cpu DECIMAL(5,2),
    memory DECIMAL(5,2),
    event_type VARCHAR(50),
    details JSONB
);
```

2. Create an inject node to trigger table creation on deploy

## Network Considerations

- Node-RED and PostgreSQL must share a Docker network
- Use container hostnames for internal communication
- External tools should use localhost with the mapped port

## Troubleshooting

### Cannot Connect to Database

Check network connectivity:
```bash
# Verify both containers are on the same network
docker network inspect node-red-network

# Test from Node-RED container
docker exec nodered ping vrooli-postgres-monitoring-db
```

### Performance Tips

1. Use connection pooling in Node-RED
2. Batch inserts when logging high-frequency data
3. Create indexes on frequently queried columns
4. Use JSONB columns for flexible schema

## Advanced: Multi-Database Monitoring

Monitor multiple client databases from one Node-RED instance:

```javascript
// Function node to route to different databases
const clientDb = msg.clientId || 'default';
msg.database = `vrooli-postgres-${clientDb}`;
return msg;
```