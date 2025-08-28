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

# Monitoring and maintenance
resource-minio logs --tail 100
resource-minio content execute --name monitor --interval 5
resource-minio content execute --name diagnose

# Bucket management
resource-minio content list
resource-minio content execute --name create-bucket --bucket my-bucket --policy download
resource-minio content execute --name remove-bucket --bucket my-bucket

# Credentials
resource-minio credentials
resource-minio content execute --name reset-credentials
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

### With AWS CLI
```bash
# Configure for MinIO
aws configure set aws_access_key_id minioadmin
aws configure set aws_secret_access_key minio123

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

## Security Features

- **Secure Credentials**: Auto-generated secure passwords on first install
- **Network Isolation**: Runs in isolated Docker network
- **File Permissions**: Credentials stored with 600 permissions
- **Access Policies**: Configurable bucket-level access control
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