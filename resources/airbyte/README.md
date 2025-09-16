# Airbyte Resource

Open-source ELT data integration platform with 600+ connectors for APIs, databases, and data warehouses.

## Quick Start

```bash
# Install and start Airbyte
vrooli resource airbyte manage install
vrooli resource airbyte manage start

# Check health
vrooli resource airbyte status

# View available connectors
vrooli resource airbyte content list --type sources
vrooli resource airbyte content list --type destinations
```

## Features

- 600+ pre-built source and destination connectors
- Automatic schema discovery and evolution
- Incremental data synchronization
- Built-in error handling and retries
- DBT integration for transformations
- REST API for programmatic control

## Usage Examples

### Create a PostgreSQL to S3 Pipeline

```bash
# Add source connector (PostgreSQL)
cat > postgres-source.json << EOF
{
  "name": "postgres-prod",
  "sourceDefinitionId": "decd338e-5647-4c0b-adf4-da0e75f5a750",
  "connectionConfiguration": {
    "host": "localhost",
    "port": 25432,
    "database": "vrooli",
    "username": "vrooli_user",
    "password": "${POSTGRES_PASSWORD}"
  }
}
EOF
vrooli resource airbyte content add --type source --config postgres-source.json

# Add destination connector (S3)
cat > s3-destination.json << EOF
{
  "name": "s3-backup",
  "destinationDefinitionId": "4816b78f-1489-44c1-9060-4b19d5fa9362",
  "connectionConfiguration": {
    "s3_bucket_name": "vrooli-backups",
    "s3_bucket_path": "postgres",
    "s3_bucket_region": "us-east-1",
    "access_key_id": "${AWS_ACCESS_KEY}",
    "secret_access_key": "${AWS_SECRET_KEY}"
  }
}
EOF
vrooli resource airbyte content add --type destination --config s3-destination.json

# Create connection
cat > connection.json << EOF
{
  "sourceId": "postgres-prod",
  "destinationId": "s3-backup",
  "schedule": {
    "units": 1,
    "timeUnit": "hours"
  },
  "status": "active"
}
EOF
vrooli resource airbyte content add --type connection --config connection.json

# Trigger manual sync
vrooli resource airbyte content execute --connection-id postgres-to-s3
```

## Architecture

Airbyte consists of several services:
- **Webapp** (port 8002): Web UI for configuration
- **Server** (port 8003): API server
- **Worker**: Executes sync jobs
- **Scheduler**: Manages job scheduling
- **Database**: PostgreSQL for metadata
- **Temporal**: Workflow orchestration

## Configuration

Default ports (from port registry):
- Webapp: 8002
- API Server: 8003
- Temporal: 8006

Environment variables:
- `AIRBYTE_VERSION`: Airbyte version (default: 0.50.0)
- `AIRBYTE_DATA_DIR`: Data directory (default: ./data)
- `AIRBYTE_WORKSPACE_DIR`: Workspace directory

## Testing

```bash
# Run smoke tests (health check)
vrooli resource airbyte test smoke

# Run integration tests
vrooli resource airbyte test integration

# Run all tests
vrooli resource airbyte test all
```

## Troubleshooting

### Services not starting
- Check Docker is running: `docker ps`
- Verify port availability: `netstat -tlnp | grep 8002`
- Check logs: `vrooli resource airbyte logs --service server`

### Connection failures
- Verify source/destination credentials
- Check network connectivity
- Review job logs: `vrooli resource airbyte logs --job-id [id]`

### Performance issues
- Increase worker memory in docker-compose.yml
- Enable staging for large syncs
- Use incremental sync mode when possible

## API Reference

See [Airbyte API Documentation](https://airbyte-public-api-docs.s3.us-east-2.amazonaws.com/rapidoc-api-docs.html)

## Related Resources

- `postgres` - Common data source
- `minio` - S3-compatible destination
- `qdrant` - Store sync metrics
- `n8n` - Orchestrate complex pipelines