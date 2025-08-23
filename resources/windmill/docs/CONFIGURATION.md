# Windmill Configuration Guide

This guide covers all configuration options for Windmill.

## Environment Variables

### Core Settings

```bash
# Server configuration
WINDMILL_CUSTOM_PORT=5681                    # API server port (default: 5681)
WINDMILL_BASE_URL=http://localhost:5681      # Public URL for Windmill
WINDMILL_CUSTOM_API_KEY=your-secure-key      # API authentication key

# Database configuration
WINDMILL_DATABASE_URL=postgresql://user:pass@localhost:5432/windmill
WINDMILL_EXTERNAL_DB=yes                     # Use external database
```

### Worker Configuration

```bash
# Worker settings
WINDMILL_WORKER_COUNT=3                      # Number of workers (default: 3)
WINDMILL_WORKER_TAGS=gpu,heavy               # Worker capability tags
WINDMILL_WORKER_MEMORY=2048                  # Memory per worker in MB
WINDMILL_WORKER_REPLICAS=1                   # Replicas per worker group

# Language Server Protocol
WINDMILL_LSP_ENABLED=true                    # Enable code completion
```

### Performance Tuning

```bash
# Execution settings
WINDMILL_TIMEOUT_WAIT_RESULT=60              # Max wait time for results (seconds)
WINDMILL_QUEUE_LIMIT_WAIT_RESULT=15          # Queue depth limit
WINDMILL_TIMEOUT_FLOW_JOB=1800               # Flow job timeout (30 minutes)

# Resource limits
WINDMILL_MAX_WAIT_TIME_SAME_WORKER=60000     # Max wait for same worker (ms)
WINDMILL_RESTART_ZOMBIE_JOBS_INTERVAL=30     # Zombie job check interval (seconds)
```

## Worker Scaling

### Dynamic Scaling

```bash
# Scale workers up
./manage.sh --action scale-workers --workers 10

# Scale down
./manage.sh --action scale-workers --workers 2

# Restart all workers
./manage.sh --action restart-workers
```

### Worker Tags

Configure workers with specific capabilities:

```bash
# GPU-enabled worker
WINDMILL_WORKER_TAGS=gpu ./manage.sh --action add-worker

# Heavy computation worker
WINDMILL_WORKER_TAGS=heavy,cpu-intensive ./manage.sh --action add-worker

# Python-specific worker
WINDMILL_WORKER_TAGS=python ./manage.sh --action add-worker
```

## Database Configuration

### Internal Database (Default)

Windmill includes a PostgreSQL container by default:

```bash
# No configuration needed for internal DB
./manage.sh --action install
```

### External Database

```bash
# PostgreSQL connection string
WINDMILL_DATABASE_URL="postgresql://windmill_user:secure_password@db.example.com:5432/windmill?sslmode=require"

# Install with external DB
./manage.sh --action install --external-db yes
```

### Database Requirements

- PostgreSQL 14+ recommended
- Minimum 1GB RAM for database
- Enable pg_trgm extension
- Configure connection pooling for production

## Security Configuration

### API Authentication

```bash
# Generate secure API key
WINDMILL_CUSTOM_API_KEY=$(openssl rand -hex 32)

# Set in environment
export WINDMILL_CUSTOM_API_KEY="$WINDMILL_CUSTOM_API_KEY"
```

### Network Security

```bash
# Bind to specific interface
WINDMILL_BIND_ADDRESS=127.0.0.1

# Enable HTTPS (requires reverse proxy)
WINDMILL_BASE_URL=https://windmill.example.com
```

## Storage Configuration

### Script Storage

```bash
# Local storage path
WINDMILL_DATA_DIR=/opt/windmill/data

# S3-compatible storage
WINDMILL_S3_BUCKET=windmill-scripts
WINDMILL_S3_ENDPOINT=https://s3.amazonaws.com
WINDMILL_S3_ACCESS_KEY=your-key
WINDMILL_S3_SECRET_KEY=your-secret
```

### Log Retention

```bash
# Log settings
WINDMILL_LOG_LEVEL=info                      # debug, info, warn, error
WINDMILL_RETAIN_LOGS_DAYS=30                 # Log retention period
WINDMILL_MAX_LOG_SIZE=100M                   # Max log file size
```

## Integration Configuration

### OAuth/SSO Setup

```bash
# GitHub OAuth
WINDMILL_GITHUB_OAUTH_ID=your-client-id
WINDMILL_GITHUB_OAUTH_SECRET=your-client-secret

# Google OAuth
WINDMILL_GOOGLE_OAUTH_ID=your-client-id
WINDMILL_GOOGLE_OAUTH_SECRET=your-client-secret
```

### Webhook Configuration

```bash
# Webhook settings
WINDMILL_WEBHOOK_TIMEOUT=30                  # Webhook timeout (seconds)
WINDMILL_WEBHOOK_MAX_RETRIES=3               # Max retry attempts
```

## Advanced Configuration

### Resource Limits

```bash
# CPU limits
WINDMILL_WORKER_CPU_LIMIT=2                  # CPU cores per worker
WINDMILL_SERVER_CPU_LIMIT=1                  # CPU cores for server

# Memory limits
WINDMILL_WORKER_MEMORY_LIMIT=4096            # MB per worker
WINDMILL_SERVER_MEMORY_LIMIT=2048            # MB for server
```

### Development Mode

```bash
# Enable development features
WINDMILL_DEV_MODE=true                       # Enable dev tools
WINDMILL_HOT_RELOAD=true                     # Auto-reload on changes
WINDMILL_VERBOSE_LOGS=true                   # Detailed logging
```

## Configuration Files

### Docker Compose Override

Create `docker-compose.override.yml`:

```yaml
version: '3.7'

services:
  windmill_server:
    environment:
      - WINDMILL_WORKER_COUNT=5
      - WINDMILL_LOG_LEVEL=debug
    ports:
      - "8080:8000"

  windmill_worker:
    deploy:
      replicas: 5
    environment:
      - WINDMILL_WORKER_TAGS=gpu,heavy
```

### Custom Worker Configuration

Create `worker-config.json`:

```json
{
  "tags": ["python", "heavy"],
  "memory_limit": 4096,
  "cpu_limit": 2,
  "env": {
    "PYTHONPATH": "/app/custom"
  }
}
```

## Applying Configuration Changes

### Restart Services

```bash
# Restart with new configuration
./manage.sh --action restart

# Verify configuration
./manage.sh --action status --verbose
```

### Validate Configuration

```bash
# Check configuration
./manage.sh --action validate-config

# Test database connection
./manage.sh --action test-db
```

## Next Steps

- [Explore the API](API.md)
- [Set up monitoring](OPERATIONS.md#monitoring)
- [Configure backups](OPERATIONS.md#backup-restore)