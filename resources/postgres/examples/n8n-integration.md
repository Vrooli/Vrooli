# PostgreSQL Integration with n8n

This guide shows how to connect n8n workflows to PostgreSQL instances created with the Vrooli resource manager.

## Prerequisites

- n8n resource running (`docker ps | grep n8n`)
- PostgreSQL instance created with n8n-network access

## Creating a PostgreSQL Instance for n8n

### Option 1: Use Development Template (Recommended)
The development template automatically includes n8n-network:

```bash
./manage.sh --action create --instance my-client --template development
```

### Option 2: Manually Specify Networks
```bash
./manage.sh --action create --instance my-client --networks "n8n-network"
```

## Getting Connection Details

Get n8n-formatted connection details:

```bash
./manage.sh --action connect --instance my-client --format n8n
```

Output:
```json
{
  "credentials": {
    "postgres": {
      "host": "vrooli-postgres-my-client",
      "port": 5432,
      "database": "vrooli_client",
      "user": "vrooli",
      "password": "your-password-here",
      "ssl": false
    }
  }
}
```

## Configuring n8n PostgreSQL Node

1. In n8n, add a PostgreSQL node to your workflow
2. Click on the node and select "Create New" credentials
3. Configure with these settings:
   - **Host**: `vrooli-postgres-my-client` (use container name, not localhost)
   - **Port**: `5432` (internal Docker port, not the external mapped port)
   - **Database**: `vrooli_client`
   - **User**: `vrooli`
   - **Password**: (from the connection details)
   - **SSL**: Disabled

## Important Notes

- **Internal vs External Ports**: 
  - Within Docker networks, always use port `5432`
  - External connections use the mapped port (e.g., `5445`)
  
- **Hostname Resolution**:
  - n8n containers can resolve PostgreSQL containers by name
  - Use the container name (`vrooli-postgres-<instance>`) not `localhost`

- **Network Connectivity**:
  - Both containers must be on the same Docker network
  - The development template automatically adds n8n-network

## Troubleshooting

### Connection Refused
If n8n can't connect, verify network connectivity:

```bash
# Check if PostgreSQL container is on n8n-network
docker inspect vrooli-postgres-my-client | jq '.[0].NetworkSettings.Networks'

# Test connectivity
docker run --rm --network n8n-network alpine ping -c 1 vrooli-postgres-my-client
```

### Wrong Port Error
Remember to use port 5432 for internal connections, not the external port shown in the default connection info.

## Example n8n Workflow

Here's a simple workflow that queries the PostgreSQL database:

1. **PostgreSQL Node**: Execute Query
   - Credentials: As configured above
   - Query: `SELECT * FROM your_table LIMIT 10`
   
2. **Set Node**: Transform Results
   - Transform the query results as needed

3. **HTTP Request Node**: Send to API
   - Send processed data to external service