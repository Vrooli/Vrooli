# Airbyte Resource

Open-source ELT data integration platform with 600+ connectors for APIs, databases, and data warehouses.

## Deployment Methods

This resource supports two deployment methods:

1. **abctl (Recommended for v1.x+)**: Uses Airbyte's new CLI with Kubernetes-in-Docker
2. **docker-compose (Legacy)**: Traditional Docker Compose deployment

The resource automatically detects which method to use based on your environment.

## Quick Start

```bash
# Install and start Airbyte
vrooli resource airbyte manage install
vrooli resource airbyte manage start --wait

# Check health
vrooli resource airbyte status

# View available connectors
vrooli resource airbyte content list --type sources
vrooli resource airbyte content list --type destinations
```

### Using abctl (Recommended)

```bash
# Force abctl deployment method
export AIRBYTE_USE_ABCTL=true
vrooli resource airbyte manage install

# The resource will automatically download and install abctl
# Installation may take 10-30 minutes on first run
```

### Using docker-compose (Legacy)

```bash
# Force docker-compose deployment method
export AIRBYTE_USE_ABCTL=false
vrooli resource airbyte manage install
```

## Features

- 600+ pre-built source and destination connectors
- Automatic schema discovery and evolution
- Incremental data synchronization
- Built-in error handling and retries
- DBT integration for transformations
- REST API for programmatic control
- Secure credential storage with encryption
- Cron-based schedule management for sync jobs
- Webhook notifications for sync events
- **NEW**: Pipeline optimization for performance monitoring
- **NEW**: Batch sync orchestration with parallel support
- **NEW**: Data quality analysis and reporting
- **NEW**: Resource usage monitoring for both deployment methods
- **NEW**: Custom Connector Development Kit (CDK) support
- **NEW**: Multi-workspace support for project isolation
- **NEW**: Prometheus metrics export for advanced monitoring

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

### Secure Credential Management

```bash
# Store credentials securely (encrypted at rest)
cat > api-cred.json << EOF
{
  "api_key": "sk-1234567890abcdef",
  "api_secret": "secret123"
}
EOF
vrooli resource airbyte credentials store --name myapi --type api_key --file api-cred.json

# List stored credentials
vrooli resource airbyte credentials list

# Use stored credentials in connectors
vrooli resource airbyte content add --type source --config source.json --credential myapi

# Rotate encryption keys
vrooli resource airbyte credentials rotate
```

### Schedule Management

```bash
# Create a daily sync schedule
vrooli resource airbyte schedule create \
  --name daily-backup \
  --connection-id postgres-to-s3 \
  --cron '0 2 * * *'  # Run at 2 AM every day

# List all schedules
vrooli resource airbyte schedule list

# Enable/disable schedules
vrooli resource airbyte schedule disable --name daily-backup
vrooli resource airbyte schedule enable --name daily-backup

# View schedule status
vrooli resource airbyte schedule status --name daily-backup
```

### Webhook Notifications

```bash
# Register a webhook for sync events
vrooli resource airbyte webhook register \
  --name slack-notify \
  --url https://hooks.slack.com/services/XXX/YYY/ZZZ \
  --events 'sync_completed,sync_failed' \
  --auth-type bearer \
  --auth-value 'xoxb-token'

# Test webhook
vrooli resource airbyte webhook test --name slack-notify

# View webhook statistics
vrooli resource airbyte webhook stats

# List all webhooks
vrooli resource airbyte webhook list
```

### DBT Transformations

```bash
# Initialize DBT project for transformations
vrooli resource airbyte transform init

# Install DBT (creates virtual environment)
vrooli resource airbyte transform install

# Create a transformation model
vrooli resource airbyte transform create staging_users \
  "SELECT * FROM raw_users WHERE active = true" \
  staging

# Run transformations
vrooli resource airbyte transform run

# Apply transformation to a connection
vrooli resource airbyte transform apply postgres-to-warehouse staging_users

# List available transformations
vrooli resource airbyte transform list

# Generate DBT documentation
vrooli resource airbyte transform docs --serve

# Test transformation models
vrooli resource airbyte transform test
```

### Pipeline Optimization

```bash
# Monitor sync performance metrics
vrooli resource airbyte pipeline performance --connection-id conn-123
# Returns: throughput, success rate, average duration

# Optimize sync configuration for large datasets
vrooli resource airbyte pipeline optimize --connection-id conn-123 --batch-size 20000

# Analyze data quality
vrooli resource airbyte pipeline quality --connection-id conn-123
# Returns: records synced, errors, warnings, quality score

# Execute batch syncs (sequential or parallel)
echo -e "conn-1\nconn-2\nconn-3" > connections.txt
vrooli resource airbyte pipeline batch --file connections.txt --parallel

# Monitor resource usage
vrooli resource airbyte pipeline resources
# Shows CPU, memory usage for all Airbyte containers/pods
```

## Architecture

### abctl Deployment (v1.x+)
When using abctl, Airbyte runs in a local Kubernetes cluster (kind) with:
- Kubernetes-in-Docker container hosting all services
- Helm charts for service management
- Ingress controller for routing
- Low-resource mode available for limited systems

### docker-compose Deployment (Legacy)
Traditional deployment with separate containers:
- **Webapp** (port 8002): Web UI for configuration
- **Server** (port 8003): API server
- **Worker**: Executes sync jobs
- **Database**: PostgreSQL for metadata
- **Temporal**: Workflow orchestration

Note: The scheduler component has been integrated into the server in v1.x

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

## Advanced Features

### Custom Connector Development (CDK)

Build and deploy custom connectors when the 600+ pre-built ones don't meet your needs:

```bash
# Initialize a new custom source connector
vrooli resource airbyte cdk init my-api source

# Build the connector Docker image
vrooli resource airbyte cdk build source-my-api

# Test the connector
vrooli resource airbyte cdk test source-my-api

# Deploy to your Airbyte instance
vrooli resource airbyte cdk deploy source-my-api

# List all custom connectors
vrooli resource airbyte cdk list
```

### Multi-Workspace Support

Organize your data pipelines into isolated workspaces for different projects or environments:

```bash
# Create workspaces
vrooli resource airbyte workspace create production "Production data pipelines"
vrooli resource airbyte workspace create staging "Staging environment"

# List all workspaces
vrooli resource airbyte workspace list

# Switch active workspace
vrooli resource airbyte workspace switch production

# Export workspace configuration
vrooli resource airbyte workspace export production

# Clone workspace for testing
vrooli resource airbyte workspace clone production production-test

# Import workspace from backup
vrooli resource airbyte workspace import workspace-backup.tar.gz
```

### Prometheus Metrics Export

Monitor Airbyte performance and health with Prometheus-compatible metrics:

```bash
# Enable metrics endpoint
vrooli resource airbyte metrics enable

# Check metrics status
vrooli resource airbyte metrics status

# View metrics dashboard
vrooli resource airbyte metrics dashboard

# Export current metrics
vrooli resource airbyte metrics export metrics-snapshot.txt

# Generate Prometheus configuration
vrooli resource airbyte metrics configure

# Disable metrics (if needed)
vrooli resource airbyte metrics disable
```

The metrics endpoint provides:
- Sync job performance metrics
- Resource usage statistics
- Connection health status
- Error rates and retry counts
- JVM and system metrics

## Related Resources

- `postgres` - Common data source
- `minio` - S3-compatible destination
- `qdrant` - Store sync metrics
- `n8n` - Orchestrate complex pipelines
- `prometheus` - Metrics collection and monitoring
- `grafana` - Visualize Airbyte metrics