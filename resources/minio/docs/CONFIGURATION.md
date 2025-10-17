# MinIO Configuration Guide

This guide covers installation, configuration, and optimization of MinIO for object storage in Vrooli.

## 🚀 Installation Options

### Default Installation

```bash
# Install with default settings
./resources/minio/cli.sh manage install

# Default configuration:
# - API Port: 9000
# - Console Port: 9001
# - Auto-generated secure credentials
# - Standard bucket setup
```

### Custom Installation

#### Custom Ports
```bash
# Avoid port conflicts
MINIO_CUSTOM_PORT=9100 MINIO_CUSTOM_CONSOLE_PORT=9101 \
  resource-minio manage install

# Verify custom ports
curl -I http://localhost:9100  # API
curl -I http://localhost:9101  # Console
```

#### Custom Credentials
```bash
# Set before installation
export MINIO_CUSTOM_ROOT_USER="admin"
export MINIO_CUSTOM_ROOT_PASSWORD="supersecurepassword123"
resource-minio manage install

# Or inline
MINIO_CUSTOM_ROOT_USER=admin MINIO_CUSTOM_ROOT_PASSWORD=secretpass \
  resource-minio manage install
```

#### Advanced Installation
```bash
# Combination of custom settings
MINIO_CUSTOM_PORT=9100 \
MINIO_CUSTOM_CONSOLE_PORT=9101 \
MINIO_CUSTOM_ROOT_USER="vrooli-admin" \
MINIO_CUSTOM_ROOT_PASSWORD="secure-pass-2024" \
MINIO_REGION="us-west-1" \
  resource-minio manage install
```

## 🏗️ Architecture Configuration

### Directory Structure

```
~/.minio/
├── data/                    # Object storage data
│   ├── .minio.sys/         # System metadata
│   │   ├── buckets/        # Bucket configurations
│   │   ├── config/         # Server configuration
│   │   └── tmp/            # Temporary files
│   ├── vrooli-user-uploads/
│   ├── vrooli-agent-artifacts/
│   ├── vrooli-model-cache/
│   └── vrooli-temp-storage/
└── config/
    └── credentials          # Access credentials (600 permissions)
```

### Docker Configuration

#### Container Settings
- **Container Name**: `minio`
- **Network**: `minio-network` (isolated)
- **Restart Policy**: `unless-stopped`
- **Health Checks**: Enabled

#### Port Mapping
```yaml
# Default ports
ports:
  - "9000:9000"  # S3 API
  - "9001:9001"  # Web Console

# Custom ports example
ports:
  - "9100:9000"  # Custom API port
  - "9101:9001"  # Custom console port
```

#### Volume Mounts
```yaml
volumes:
  - ~/.minio/data:/data                    # Data persistence
  - ~/.minio/config:/root/.minio          # Configuration
```

## 📦 Bucket Configuration

### Default Buckets

MinIO automatically creates four buckets for Vrooli:

| Bucket | Purpose | Policy | Lifecycle |
|--------|---------|--------|-----------|
| `vrooli-user-uploads` | User profile pictures, attachments | Public read | None |
| `vrooli-agent-artifacts` | AI-generated images, documents | Private | None |
| `vrooli-model-cache` | Downloaded AI models | Private | None |
| `vrooli-temp-storage` | Temporary files | Private | 24hr auto-delete |

### Custom Bucket Creation

```bash
# Create bucket with specific policy
resource-minio content execute --name create-bucket --bucket my-data --policy none

# Available policies:
# - none: Private (default)
# - download: Public read access
# - upload: Public write access  
# - public: Full public access
```

### Lifecycle Policies

#### Automatic Cleanup Configuration
```bash
# Temporary storage: 24-hour cleanup (configured automatically)
docker exec minio mc ilm add local/vrooli-temp-storage --expire-days 1

# Custom lifecycle for other buckets
docker exec minio mc ilm add local/my-bucket --expire-days 30
docker exec minio mc ilm add local/logs-bucket --expire-days 7
```

#### Advanced Lifecycle Rules
```bash
# Transition to infrequent access after 30 days
docker exec minio mc ilm add local/archive-bucket \
  --transition-days 30 \
  --storage-class IA

# Delete incomplete multipart uploads after 7 days
docker exec minio mc ilm add local/uploads-bucket \
  --expiry-days 7 \
  --incomplete-upload-expiry-days 7
```

## 🔐 Security Configuration

### Credential Management

#### Secure Credential Generation
```bash
# Automatic secure generation (default)
resource-minio manage install  # Generates 16-character secure password

# View current credentials
resource-minio credentials

# Reset to new secure credentials
resource-minio content execute --name reset-credentials
```

#### Manual Credential Setting
```bash
# Set specific credentials (ensure they're secure)
export MINIO_CUSTOM_ROOT_USER="admin-$(date +%s)"
export MINIO_CUSTOM_ROOT_PASSWORD="$(openssl rand -base64 32)"
resource-minio manage install
```

### File Permissions

```bash
# Credentials file security
ls -la ~/.minio/config/credentials
# Should show: -rw------- (600 permissions)

# Fix permissions if needed
chmod 600 ~/.minio/config/credentials
chmod 700 ~/.minio/config/
```

### Network Security

#### Docker Network Isolation
```bash
# MinIO runs in isolated Docker network
docker network ls | grep minio
# Should show: minio-network

# Network configuration
docker network inspect minio-network
```

#### Access Control
```bash
# Bind to localhost only (default)
# External access disabled by default
# Console accessible only from local machine
```

## ⚙️ Environment Variables

### Installation-Time Variables

```bash
# Port configuration
MINIO_CUSTOM_PORT=9000                    # API port
MINIO_CUSTOM_CONSOLE_PORT=9001           # Console port

# Credentials
MINIO_CUSTOM_ROOT_USER="admin"           # Username
MINIO_CUSTOM_ROOT_PASSWORD="password"    # Password

# Performance tuning
MINIO_API_REQUESTS_MAX=1000              # Max concurrent requests
MINIO_REGION="us-east-1"                 # Default region

# Storage configuration
MINIO_BROWSER="on"                       # Enable/disable web console
MINIO_DOMAIN=""                          # Custom domain
```

### Runtime Environment Variables

Set these in the Docker container:

```bash
# Memory optimization
MINIO_API_REQUESTS_DEADLINE=10s          # Request timeout
MINIO_API_REQUESTS_MAX=1000              # Max requests

# Security
MINIO_BROWSER_REDIRECT_URL=""            # Console redirect URL
MINIO_IDENTITY_OPENID_CONFIG_URL=""      # OpenID configuration

# Monitoring
MINIO_PROMETHEUS_AUTH_TYPE="public"      # Metrics authentication
```

## 🔧 Performance Tuning

### Resource Limits

#### Docker Resource Configuration
```bash
# Edit lib/docker.sh to add resource limits:
docker run -d \
  --name minio \
  --memory="2g" \
  --cpus="2.0" \
  --restart unless-stopped \
  # ... other options
```

#### Memory Optimization
```bash
# Set memory limits based on usage
# Minimum: 512MB
# Recommended: 2GB for moderate usage
# Heavy usage: 4GB+

# Monitor memory usage
docker stats minio
```

### API Performance

```bash
# Increase connection limits
MINIO_API_REQUESTS_MAX=2000 resource-minio manage install

# Set request deadline
export MINIO_API_REQUESTS_DEADLINE=30s
resource-minio manage restart

# Enable compression for better bandwidth usage
export MINIO_COMPRESS="on"
resource-minio manage restart
```

### Storage Optimization

#### Disk Configuration
```bash
# Use SSD for better performance
# Ensure adequate disk space
df -h ~/.minio/data/

# Monitor disk I/O
iostat -x 1

# Clean up temporary files periodically
docker exec minio find /data/.minio.sys/tmp -type f -mtime +1 -delete
```

#### File System Tuning
```bash
# For better performance on large files
# Consider XFS or ext4 with appropriate block sizes
sudo mkfs.xfs -f -s size=4096 -b size=4096 /dev/disk
```

## 🔗 Integration Configuration

### Vrooli Integration

MinIO automatically registers with Vrooli's resource system:

```json
{
  "services": {
    "storage": {
      "minio": {
        "enabled": true,
        "endpoint": "localhost:9000",
        "useSSL": false,
        "accessKey": "${MINIO_ACCESS_KEY}",
        "secretKey": "${MINIO_SECRET_KEY}",
        "region": "us-east-1",
        "buckets": {
          "userUploads": "vrooli-user-uploads",
          "agentArtifacts": "vrooli-agent-artifacts", 
          "modelCache": "vrooli-model-cache",
          "tempStorage": "vrooli-temp-storage"
        },
        "healthCheck": {
          "intervalMs": 60000,
          "timeoutMs": 5000,
          "endpoint": "/minio/health/live"
        }
      }
    }
  }
}
```

### External Tool Integration

#### AWS CLI Configuration
```bash
# Configure AWS CLI profile for MinIO
aws configure set profile.minio.aws_access_key_id minioadmin
aws configure set profile.minio.aws_secret_access_key minio123
aws configure set profile.minio.region us-east-1
aws configure set profile.minio.output json

# Use with custom endpoint
aws s3 ls --profile minio --endpoint-url http://localhost:9000
```

#### SDK Configuration Examples

**Python (boto3)**:
```python
import boto3
from botocore.client import Config

s3_client = boto3.client(
    's3',
    endpoint_url='http://localhost:9000',
    aws_access_key_id='minioadmin',
    aws_secret_access_key='minio123',
    config=Config(signature_version='s3v4'),
    region_name='us-east-1'
)
```

**Node.js**:
```javascript
const Minio = require('minio');

const minioClient = new Minio.Client({
    endPoint: 'localhost',
    port: 9000,
    useSSL: false,
    accessKey: 'minioadmin',
    secretKey: 'minio123'
});
```

## 🔄 Backup and Recovery Configuration

### Data Backup Strategy

```bash
# Full data backup
tar -czf "minio-backup-$(date +%Y%m%d).tar.gz" ~/.minio/data/

# Incremental backup (using rsync)
rsync -av --delete ~/.minio/data/ /backup/minio-data/

# Automated backup script
cat > ~/.minio/backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/backup/minio"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p "$BACKUP_DIR"
tar -czf "$BACKUP_DIR/minio-$DATE.tar.gz" ~/.minio/data/
# Keep only last 7 backups
ls -t "$BACKUP_DIR"/minio-*.tar.gz | tail -n +8 | xargs rm -f
EOF
chmod +x ~/.minio/backup.sh
```

### Disaster Recovery

```bash
# Create disaster recovery script
cat > ~/.minio/restore.sh << 'EOF'
#!/bin/bash
BACKUP_FILE="$1"
if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup-file>"
    exit 1
fi

# Stop MinIO
resource-minio manage stop

# Backup current data
mv ~/.minio/data ~/.minio/data.backup.$(date +%s)

# Restore from backup
mkdir -p ~/.minio/data
tar -xzf "$BACKUP_FILE" -C ~/.minio/

# Start MinIO
resource-minio manage start
EOF
chmod +x ~/.minio/restore.sh
```

## 📊 Monitoring Configuration

### Health Check Configuration

```bash
# Configure health check intervals
export MINIO_HEALTH_CHECK_INTERVAL=30s
resource-minio manage restart

# Custom health check endpoint
curl -f http://localhost:9000/minio/health/live || exit 1
```

### Metrics and Logging

```bash
# Enable Prometheus metrics
export MINIO_PROMETHEUS_AUTH_TYPE="public"
resource-minio manage restart

# Access metrics
curl http://localhost:9000/minio/v2/metrics/cluster

# Configure log levels
export MINIO_ROOT_LOG_LEVEL="INFO"  # DEBUG, INFO, WARN, ERROR
resource-minio manage restart
```

### Alerting Configuration

```bash
# Disk space monitoring
cat > ~/.minio/monitor.sh << 'EOF'
#!/bin/bash
THRESHOLD=80
USAGE=$(df ~/.minio/data | tail -1 | awk '{print $5}' | sed 's/%//')
if [ "$USAGE" -gt "$THRESHOLD" ]; then
    echo "MinIO disk usage is ${USAGE}% (threshold: ${THRESHOLD}%)"
    # Add alerting logic here
fi
EOF

# Add to crontab
echo "*/5 * * * * ~/.minio/monitor.sh" | crontab -
```

This configuration guide ensures optimal MinIO setup for your object storage requirements in the Vrooli ecosystem.