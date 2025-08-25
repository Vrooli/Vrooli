# n8n Configuration Guide

This guide covers all configuration options for n8n, including installation parameters, runtime settings, database configuration, and security setup.

## Installation Configuration

### Basic Installation Options

```bash
# Install with custom image (recommended for host access)
./manage.sh --action install --build-image yes

# Install with standard n8n image
./manage.sh --action install

# Install with PostgreSQL database
./manage.sh --action install --database postgres --build-image yes
```

### Installation Parameters

| Parameter | Description | Default | Options |
|-----------|-------------|---------|---------|
| `--build-image` | Build custom image with host access | `no` | `yes`, `no` |
| `--basic-auth` | Enable basic authentication | `yes` | `yes`, `no` |
| `--username` | Basic auth username | `admin` | Any string |
| `--password` | Basic auth password | Auto-generated | Any string |
| `--database` | Database type | `sqlite` | `sqlite`, `postgres` |
| `--webhook-url` | External webhook URL | None | Full URL |
| `--force` | Force reinstall | `no` | `yes`, `no` |

### Complete Installation Example

```bash
./manage.sh --action install \
  --build-image yes \
  --basic-auth yes \
  --username admin \
  --password secure123 \
  --database postgres \
  --webhook-url https://your-domain.com \
  --force yes
```

## Environment Variables

### Core Configuration

```bash
# Service Configuration
export N8N_CUSTOM_PORT=5678                  # Override default port
export N8N_BASIC_AUTH_USER=admin             # Basic auth username
export N8N_BASIC_AUTH_PASSWORD=secure123     # Basic auth password
export N8N_WEBHOOK_URL=https://domain.com    # External webhook URL

# Database Configuration
export N8N_DB_TYPE=sqlite                    # Database type: sqlite or postgres
export N8N_DB_SQLITE_DATABASE=/data/n8n.db  # SQLite database file
export N8N_DB_POSTGRESDB_HOST=postgres      # PostgreSQL host
export N8N_DB_POSTGRESDB_PORT=5432          # PostgreSQL port
export N8N_DB_POSTGRESDB_DATABASE=n8n       # PostgreSQL database name
export N8N_DB_POSTGRESDB_USER=n8n           # PostgreSQL username
export N8N_DB_POSTGRESDB_PASSWORD=password  # PostgreSQL password

# Security Settings
export N8N_ENCRYPTION_KEY=your-encryption-key # Encryption key for credentials
export N8N_API_KEY=your-api-key              # API key (create in web UI)
```

### Performance Configuration

```bash
# Execution Settings
export N8N_DEFAULT_BINARY_DATA_MODE=filesystem # Binary data storage mode
export N8N_BINARY_DATA_TTL=24                  # Binary data TTL in hours
export N8N_EXECUTIONS_DATA_PRUNE=true          # Auto-prune old executions
export N8N_EXECUTIONS_DATA_MAX_AGE=168         # Keep executions for 7 days

# Resource Limits
export N8N_PAYLOAD_SIZE_MAX=16                 # Max payload size in MB
export N8N_METRICS=true                        # Enable metrics endpoint
export N8N_DIAGNOSTICS_ENABLED=false           # Disable diagnostics
```

### Feature Toggles

```bash
# Feature Configuration
export N8N_TEMPLATES_ENABLED=true              # Enable workflow templates
export N8N_ONBOARDING_FLOW_DISABLED=true       # Disable onboarding
export N8N_HIDE_USAGE_PAGE=true                # Hide usage statistics page
export N8N_PERSONALIZATION_ENABLED=false       # Disable personalization survey

# Security Features
export N8N_BLOCK_ENV_ACCESS_IN_NODE=false      # Allow env access in nodes
export N8N_SECURE_COOKIE=false                 # Use secure cookies (HTTPS only)
export N8N_DISABLE_UI=false                    # Disable web UI completely
```

## Docker Configuration

### Volume Mounts

The n8n container uses several volume mounts:

```bash
# Data persistence
-v n8n-data:/home/node/.n8n               # n8n user data and workflows

# Host integration (when custom image is used)
-v /var/run/docker.sock:/var/run/docker.sock # Docker control
-v "${PWD}:/workspace:ro"                     # Workspace access (read-only)
-v /usr/bin:/host/usr/bin:ro                  # Host binaries access
-v /bin:/host/bin:ro                          # System binaries access
-v /home:/host/home                           # Home directory access

# Optional mounts
-v /etc/localtime:/etc/localtime:ro           # Timezone sync
-v ./custom-nodes:/home/node/.n8n/custom     # Custom node modules
```

### Custom Docker Options

```bash
# Advanced Docker configuration
export DOCKER_EXTRA_ARGS="--privileged"

# Custom network configuration
export DOCKER_NETWORK="vrooli-network"

# Additional environment variables
export DOCKER_ENV_VARS="-e N8N_LOG_LEVEL=debug"
```

### Resource Limits

```bash
# Memory and CPU limits
docker update n8n --memory 2g --cpus 1.5

# Restart policy
docker update n8n --restart unless-stopped
```

## Database Configuration

### SQLite Configuration (Default)

```bash
# SQLite settings
export N8N_DB_TYPE=sqlite
export N8N_DB_SQLITE_DATABASE=/home/node/.n8n/database.sqlite
export N8N_DB_SQLITE_VACUUM_ON_STARTUP=true

# SQLite optimization
export N8N_DB_SQLITE_ENABLE_WAL=true         # Enable Write-Ahead Logging
export N8N_DB_SQLITE_POOL_SIZE=10            # Connection pool size
```

### PostgreSQL Configuration

```bash
# PostgreSQL settings
export N8N_DB_TYPE=postgres
export N8N_DB_POSTGRESDB_HOST=localhost
export N8N_DB_POSTGRESDB_PORT=5432
export N8N_DB_POSTGRESDB_DATABASE=n8n
export N8N_DB_POSTGRESDB_USER=n8n
export N8N_DB_POSTGRESDB_PASSWORD=secure_password
export N8N_DB_POSTGRESDB_SCHEMA=public

# PostgreSQL optimization
export N8N_DB_POSTGRESDB_POOL_SIZE=20        # Connection pool size
export N8N_DB_POSTGRESDB_SSL_CA=/path/to/ca.pem # SSL certificate
export N8N_DB_POSTGRESDB_SSL_CERT=/path/to/cert.pem
export N8N_DB_POSTGRESDB_SSL_KEY=/path/to/key.pem
export N8N_DB_POSTGRESDB_SSL_REJECT_UNAUTHORIZED=true
```

### Database Migration

```bash
# Backup SQLite database
./manage.sh --action backup-database --output n8n-backup.db

# Migrate from SQLite to PostgreSQL
export N8N_DB_TYPE=postgres
export N8N_DB_POSTGRESDB_HOST=postgres-host
# ... other PostgreSQL settings
./manage.sh --action install --database postgres --force yes

# Restore data (manual process required)
# Export workflows from old instance, import to new instance
```

## Authentication Configuration

### Basic Authentication

```bash
# Enable basic authentication
export N8N_BASIC_AUTH_ACTIVE=true
export N8N_BASIC_AUTH_USER=admin
export N8N_BASIC_AUTH_PASSWORD=secure_password

# Hash password for storage
export N8N_BASIC_AUTH_HASH=true
```

### API Key Authentication

```bash
# API key for REST API access (create in web UI)
export N8N_API_KEY=your-generated-api-key

# API key security
export N8N_API_KEY_PREFIXED=true             # Require n8n_ prefix
```

### External Authentication

```bash
# LDAP Authentication (Enterprise feature)
export N8N_AUTH_LDAP_ENABLED=true
export N8N_AUTH_LDAP_SERVER=ldap://ldap.company.com
export N8N_AUTH_LDAP_BIND_DN=cn=admin,dc=company,dc=com
export N8N_AUTH_LDAP_BIND_PASSWORD=admin_password
export N8N_AUTH_LDAP_BASE_DN=ou=users,dc=company,dc=com

# SAML Authentication (Enterprise feature)
export N8N_AUTH_SAML_ENABLED=true
export N8N_AUTH_SAML_METADATA_URL=https://idp.company.com/metadata
```

## Custom Image Configuration

### Dockerfile Customization

The custom n8n image includes:

```dockerfile
FROM n8n/n8n:latest

# Install system tools
USER root
RUN apk add --no-cache \
    bash \
    git \
    curl \
    wget \
    jq \
    python3 \
    docker-cli \
    openssh-client

# Install additional npm packages
USER node
RUN npm install -g \
    n8n-nodes-base \
    custom-node-package

# Copy custom entrypoint
COPY docker-entrypoint.sh /usr/local/bin/
USER root
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
USER node

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
```

### Custom Entrypoint Script

The `docker-entrypoint.sh` script provides:

```bash
#!/bin/bash
set -e

# Add host directories to PATH for transparent command access
export PATH="/host/usr/bin:/host/bin:$PATH"

# Set up host integration
export N8N_HOST_COMMANDS=true

# Start n8n with original entrypoint
exec tini -- node ./dist/index.js "$@"
```

## Workflow Configuration

### Default Workflow Settings

```bash
# Workflow execution settings
export N8N_DEFAULT_TIMEZONE=America/New_York
export N8N_WORKFLOWS_DEFAULT_NAME="New Workflow"
export N8N_DEFAULT_LOCALE=en

# Execution behavior
export N8N_EXECUTIONS_MODE=regular           # regular or queue
export N8N_EXECUTIONS_TIMEOUT=3600           # Max execution time in seconds
export N8N_EXECUTIONS_TIMEOUT_MAX=7200       # Absolute max timeout
```

### Workflow Templates

```bash
# Template configuration
export N8N_TEMPLATES_ENABLED=true
export N8N_TEMPLATES_HOST=https://api.n8n.io
export N8N_DISABLE_TEMPLATES=false

# Custom template directory
export N8N_CUSTOM_TEMPLATES_DIR=/data/templates
```

## Webhook Configuration

### Webhook Settings

```bash
# Webhook configuration
export N8N_WEBHOOK_URL=https://your-domain.com  # External webhook URL
export N8N_WEBHOOK_TUNNEL_URL=                  # Tunnel URL for development
export WEBHOOK_URL=https://your-domain.com      # Alternative setting

# Webhook security
export N8N_WEBHOOK_CORS_ORIGINS=*               # Allowed CORS origins
export N8N_WEBHOOK_CORS_METHODS=GET,POST,PUT    # Allowed methods
```

### Production Webhook Setup

```bash
# Production webhook configuration
export N8N_WEBHOOK_URL=https://hooks.company.com
export N8N_WEBHOOK_PATH=/webhook               # Webhook path prefix
export N8N_WEBHOOK_TEST_PATH=/webhook-test     # Test webhook path

# SSL/TLS configuration
export N8N_PROTOCOL=https
export N8N_SSL_KEY=/etc/ssl/private/n8n.key
export N8N_SSL_CERT=/etc/ssl/certs/n8n.crt
```

## Logging Configuration

### Log Settings

```bash
# Logging configuration
export N8N_LOG_LEVEL=info                      # debug, info, warn, error
export N8N_LOG_OUTPUT=console                  # console, file
export N8N_LOG_FILE=/data/n8n.log             # Log file path

# Log formatting
export N8N_LOG_FORMAT=json                     # json or text
export N8N_LOG_TIMESTAMP=true                  # Include timestamps
```

### Advanced Logging

```bash
# Debug logging for specific components
export N8N_LOG_LEVEL=debug
export DEBUG=n8n:*                            # Enable all debug logs
export DEBUG=n8n:workflow:*                   # Workflow-specific logs

# Performance logging
export N8N_METRICS=true                       # Enable metrics endpoint
export N8N_METRICS_PREFIX=n8n_                # Metrics prefix
```

## Performance Optimization

### Memory Management

```bash
# Node.js memory settings
export NODE_OPTIONS="--max-old-space-size=2048"

# n8n memory settings
export N8N_BINARY_DATA_MEMORY_LIMIT=100      # MB limit for binary data in memory
export N8N_EXECUTIONS_DATA_HARD_DELETE_BUFFER=1 # Hard delete buffer
```

### Execution Optimization

```bash
# Queue mode for high-load scenarios
export N8N_EXECUTIONS_MODE=queue
export QUEUE_BULL_REDIS_HOST=redis
export QUEUE_BULL_REDIS_PORT=6379
export QUEUE_BULL_REDIS_PASSWORD=redis_password

# Worker processes
export N8N_WORKERS_AUTO_START=true
export N8N_WORKERS_COUNT=4
```

### Binary Data Configuration

```bash
# Binary data handling
export N8N_DEFAULT_BINARY_DATA_MODE=filesystem
export N8N_BINARY_DATA_PATH=/data/binary-data
export N8N_BINARY_DATA_TTL=24                # Hours to keep binary data
export N8N_BINARY_DATA_CLEANUP_INTERVAL=60   # Cleanup interval in minutes
```

## Security Configuration

### General Security

```bash
# Security settings
export N8N_BLOCK_ENV_ACCESS_IN_NODE=true      # Block env access in function nodes
export N8N_SECURE_COOKIE=true                 # Use secure cookies
export N8N_COOKIE_SAME_SITE=strict            # Cookie SameSite policy

# Content Security Policy
export N8N_CSP_ENABLED=true
export N8N_CSP_DIRECTIVES="default-src 'self'; script-src 'self' 'unsafe-inline'"
```

### Network Security

```bash
# Network restrictions
export N8N_BLOCK_EXTERNAL_CONNECTIONS=false   # Block external connections
export N8N_ALLOWED_EXTERNAL_HOSTS=api.github.com,api.slack.com

# Rate limiting
export N8N_API_RATE_LIMIT_ENABLED=true
export N8N_API_RATE_LIMIT_COUNT=120           # Requests per window
export N8N_API_RATE_LIMIT_WINDOW=60           # Window in seconds
```

## Backup and Recovery

### Configuration Backup

```bash
# Backup all workflows
./manage.sh --action export-workflows --output "backup-$(date +%Y%m%d).json"

# Backup database
./manage.sh --action backup-database --output "db-backup-$(date +%Y%m%d).sql"

# Backup configuration
docker exec n8n env | grep N8N_ > n8n-config-backup.env
```

### Automated Backup

```bash
# Cron job for daily backup
0 2 * * * cd /path/to/vrooli && ./resources/n8n/manage.sh --action export-workflows --output "backups/workflows-$(date +\%Y\%m\%d).json"
```

### Recovery Procedure

```bash
# Restore workflows
./manage.sh --action import-workflows --input backup-workflows.json

# Restore database
./manage.sh --action restore-database --input db-backup.sql

# Restart service
./manage.sh --action restart
```

## Monitoring Configuration

### Health Checks

```bash
# Health check endpoint
export N8N_METRICS=true                       # Enable /metrics endpoint
export N8N_DIAGNOSTICS_ENABLED=true           # Enable diagnostics

# Custom health check
curl http://localhost:5678/healthz
curl http://localhost:5678/metrics
```

### Monitoring Integration

```bash
# Prometheus metrics
export N8N_METRICS=true
export N8N_METRICS_PREFIX=n8n_

# External monitoring webhook
export N8N_WEBHOOK_MONITORING_URL=https://monitoring.company.com/webhook
```

## Integration Configuration

### Vrooli Resource Integration

n8n is automatically configured in `~/.vrooli/service.json`:

```json
{
  "services": {
    "automation": {
      "n8n": {
        "enabled": true,
        "baseUrl": "http://localhost:5678",
        "healthCheck": {
          "intervalMs": 60000,
          "timeoutMs": 5000,
          "endpoint": "/healthz"
        },
        "api": {
          "version": "v1",
          "key": "your-api-key-here",
          "endpoints": {
            "workflows": "/api/v1/workflows",
            "executions": "/api/v1/executions",
            "credentials": "/api/v1/credentials"
          }
        }
      }
    }
  }
}
```

### External Service Integration

```bash
# Slack integration
export N8N_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...

# GitHub integration
export N8N_GITHUB_TOKEN=your-github-token

# Email configuration
export N8N_EMAIL_FROM=n8n@company.com
export N8N_EMAIL_SMTP_HOST=smtp.company.com
export N8N_EMAIL_SMTP_PORT=587
export N8N_EMAIL_SMTP_USER=smtp_user
export N8N_EMAIL_SMTP_PASS=smtp_password
```

This configuration guide provides comprehensive coverage of all n8n configuration options. For specific use cases, refer to the [API documentation](API.md) and [troubleshooting guide](TROUBLESHOOTING.md).