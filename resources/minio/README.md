# MinIO - Local Object Storage

MinIO is a high-performance, S3-compatible object storage server that provides local storage capabilities for Vrooli. It enables file uploads, artifact storage, and serves as a foundation for future cloud storage migrations.

## Quick Reference
- **Category**: Storage
- **Ports**: 9000 (API), 9001 (Console)
- **Container**: minio
- **API Docs**: [Complete API Reference](docs/API.md)
- **Status**: Production Ready

## When to Use
- **Local file storage** with S3-compatible API access
- **AI model and artifact storage** for generated content
- **File uploads and downloads** for user content
- **Multi-application file sharing** within Vrooli ecosystem

**Alternative**: Cloud storage (AWS S3, Google Cloud Storage) for scale, local filesystem for simplicity

## 🚀 Quick Start

```bash
# Install MinIO with defaults
resource-minio manage install

# Check status
resource-minio status

# View credentials
resource-minio credentials

# Test functionality
resource-minio test smoke
```

## Access Points

After installation:
- **Console UI**: http://localhost:9001 (web interface)
- **API Endpoint**: http://localhost:9000 (S3-compatible API)
- **Health Check**: http://localhost:9000/minio/health/live

## 📚 Documentation

- 📖 [**Complete API Reference**](docs/API.md) - S3 API, service management, bucket operations, integration examples
- ⚙️ [**Configuration Guide**](docs/CONFIGURATION.md) - Installation options, security, performance tuning
- 🔧 [**Troubleshooting**](docs/TROUBLESHOOTING.md) - Common issues, diagnostics, recovery procedures
- 📂 [**Examples**](examples/README.md) - S3 integration patterns and automation workflows

## Default Buckets

MinIO automatically creates four buckets for Vrooli:

| Bucket | Purpose | Access Policy |
|--------|---------|---------------|
| `vrooli-user-uploads` | User profile pictures, attachments | Public read |
| `vrooli-agent-artifacts` | AI-generated images, documents | Private |
| `vrooli-model-cache` | Downloaded AI models | Private |
| `vrooli-temp-storage` | Temporary files (24hr auto-delete) | Private |

## Service Management

```bash
# Installation and setup
resource-minio manage install
resource-minio status

# Service control
resource-minio manage start
resource-minio manage stop
resource-minio manage restart

# Monitoring and metrics
resource-minio logs --tail 100
resource-minio metrics                  # Show storage statistics
resource-minio content execute --name monitor --interval 5

# Performance tuning
resource-minio performance profile minimal      # Low resource usage
resource-minio performance profile balanced     # Default balanced
resource-minio performance profile performance  # High performance
resource-minio performance monitor              # Show performance metrics
resource-minio performance benchmark 100 5      # Run benchmark (100MB, 5 iterations)

# Backup and restore operations
resource-minio backup create            # Create timestamped backup
resource-minio backup create my-backup  # Create named backup  
resource-minio backup list              # List available backups
resource-minio backup restore my-backup # Restore from backup
resource-minio backup delete my-backup  # Delete a backup

# Bucket management
resource-minio content list
resource-minio content add my-bucket    # Create bucket
resource-minio content policy my-bucket download  # Set access policy
resource-minio content versioning my-bucket enable  # Enable versioning
resource-minio content versioning my-bucket status  # Check versioning status
resource-minio content remove my-bucket

# File operations
resource-minio content upload my-bucket /path/to/file.txt
resource-minio content download my-bucket file.txt /local/path

# Credentials
resource-minio credentials
resource-minio content configure        # Configure MC client
```

## Custom Installation

```bash
# Custom ports (avoid conflicts)
MINIO_CUSTOM_PORT=9100 MINIO_CUSTOM_CONSOLE_PORT=9101 \
  resource-minio manage install

# Custom credentials  
MINIO_CUSTOM_ROOT_USER=admin MINIO_CUSTOM_ROOT_PASSWORD=secretpass \
  resource-minio manage install

# Performance tuning
MINIO_API_REQUESTS_MAX=1000 MINIO_REGION=us-west-1 \
  resource-minio manage install
```

## Integration Examples

### With MinIO Client (mc) - Built-in
```bash
# The mc client is included in the MinIO container
# Configure alias (done automatically by resource)
docker exec minio mc alias set local http://localhost:9000 $ACCESS_KEY $SECRET_KEY

# List buckets
docker exec minio mc ls local/

# Upload file
docker exec minio mc cp /path/to/file local/bucket-name/

# Download file  
docker exec minio mc cp local/bucket-name/file /path/to/destination

# Create bucket
docker exec minio mc mb local/new-bucket

# Enable versioning on bucket
docker exec minio mc version enable local/bucket-name

# Check versioning status
docker exec minio mc version info local/bucket-name

# Remove bucket
docker exec minio mc rb --force local/old-bucket
```

### With AWS CLI (Optional - requires separate installation)
```bash
# Load MinIO credentials
source ~/.minio/config/credentials

# Configure AWS CLI
aws configure set aws_access_key_id "$MINIO_ROOT_USER"
aws configure set aws_secret_access_key "$MINIO_ROOT_PASSWORD"

# Use S3 commands
aws s3 ls --endpoint-url http://localhost:9000
aws s3 cp file.txt s3://bucket-name/ --endpoint-url http://localhost:9000
```

### With Python (boto3)
```python
import boto3

s3_client = boto3.client(
    's3',
    endpoint_url='http://localhost:9000',
    aws_access_key_id='minioadmin',
    aws_secret_access_key='minio123'
)

# Upload file
s3_client.upload_file('local-file.txt', 'bucket-name', 'remote-file.txt')
```

### With AI Resources
```bash
# Store AI model artifacts
curl -X PUT --data-binary @model.bin \
  http://localhost:9000/vrooli-model-cache/llama-3.1-8b.bin

# Store agent screenshots
curl -X PUT --data-binary @screenshot.png \
  http://localhost:9000/vrooli-agent-artifacts/task-screenshot.png
```

## New Features (v2.3)

- **Performance Tuning**: Configurable performance profiles (minimal/balanced/performance)
- **Performance Monitoring**: Real-time metrics and resource usage tracking
- **Performance Benchmarking**: Built-in benchmark tool for throughput testing
- **Storage Metrics**: `metrics` command shows per-bucket usage statistics
- **Bucket Policies**: `content policy` command for public/private access control
- **Multi-part Upload**: Automatic for files >100MB with resume capability
- **MC Client Integration**: Updated to latest MinIO client syntax
- **Object Versioning**: `content versioning` command for bucket-level version control

## Performance Tuning

MinIO supports configurable performance profiles to optimize for different workloads:

### Performance Profiles

| Profile | CPU | Cache | API Limits | Use Case |
|---------|-----|-------|------------|----------|
| `minimal` | 2 cores | 64MB | Standard | Development, low resources |
| `balanced` | 4 cores | 256MB | Standard | Default, general use |
| `performance` | 8 cores | 1GB | High | Production, heavy loads |
| `custom` | User-defined | User-defined | User-defined | Advanced tuning |

### Usage
```bash
# Apply a profile (requires restart to fully apply)
resource-minio performance profile balanced
resource-minio manage restart

# Monitor performance
resource-minio performance monitor

# Run benchmark
resource-minio performance benchmark 100 5  # 100MB file, 5 iterations
```

### Performance Tips
- Use `performance` profile for production workloads
- Enable caching for frequently accessed objects
- Monitor with `performance monitor` to identify bottlenecks
- Run benchmarks after configuration changes

## Replication

MinIO supports multi-instance data replication for high availability and disaster recovery:

### Replication Types

| Type | Description | Use Case |
|------|-------------|----------|
| **Active-Active** | Bi-directional sync between instances | Load balancing, HA |
| **Active-Passive** | One-way sync to backup instance | Disaster recovery |

### Setup

```bash
# Configure replication to remote MinIO
resource-minio replication setup http://remote-minio:9000 access_key secret_key active-active

# Check status
resource-minio replication status

# Manual sync (if needed)
resource-minio replication sync both  # or push/pull
```

### Operations

```bash
# Enable/disable replication
resource-minio replication enable
resource-minio replication disable

# Monitor replication lag
resource-minio replication monitor

# Failover management (active-passive only)
resource-minio replication failover status
resource-minio replication failover promote  # Make local primary
resource-minio replication failover demote   # Make local secondary

# Remove replication
resource-minio replication cleanup
```

### Requirements
- Both MinIO instances must be reachable
- Versioning is automatically enabled (required for replication)
- Same bucket names on both instances

## Security Features

- **Secure Credentials**: Auto-generated secure passwords on first install
- **Network Isolation**: Runs in isolated Docker network
- **File Permissions**: Credentials stored with 600 permissions
- **Access Policies**: Configurable bucket-level access control (public/download/upload/private)
- **Health Monitoring**: Built-in health checks and diagnostics

## Architecture

### Directory Structure
```
~/.minio/
├── data/                    # Object storage data
│   ├── vrooli-user-uploads/
│   ├── vrooli-agent-artifacts/
│   ├── vrooli-model-cache/
│   └── vrooli-temp-storage/
└── config/
    └── credentials          # Access credentials (600 perms)
```

### Docker Configuration
- **Container**: `minio` with `unless-stopped` restart policy
- **Network**: `minio-network` (isolated)
- **Volumes**: Data persistence and configuration mounting
- **Health Checks**: Automatic container health monitoring

## Automatic Features

- **Bucket Creation**: Default Vrooli buckets created automatically
- **Lifecycle Policies**: 24-hour cleanup for temporary storage
- **Health Monitoring**: Continuous service health checks
- **Secure Defaults**: Strong passwords and proper permissions

## Integration with Vrooli

MinIO automatically registers with Vrooli's resource discovery system, providing:
- **User Uploads**: Profile pictures and file attachments
- **Agent Artifacts**: AI-generated images and documents  
- **Model Cache**: Downloaded and cached AI models
- **Temporary Storage**: Short-lived files with auto-cleanup

For detailed usage instructions, S3 API integration, and troubleshooting, see the documentation links above.