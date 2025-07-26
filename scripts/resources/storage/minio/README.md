# MinIO Local Object Storage

MinIO is a high-performance, S3-compatible object storage server that provides local storage capabilities for Vrooli. It enables file uploads, artifact storage, and serves as a foundation for future cloud storage migrations.

## 🚀 Quick Start

### Installation

```bash
# Install with default settings
./scripts/resources/storage/minio/manage.sh --action install

# Install with custom ports
MINIO_CUSTOM_PORT=9100 MINIO_CUSTOM_CONSOLE_PORT=9101 \
  ./scripts/resources/storage/minio/manage.sh --action install

# Install with custom credentials
MINIO_CUSTOM_ROOT_USER=admin MINIO_CUSTOM_ROOT_PASSWORD=secretpass \
  ./scripts/resources/storage/minio/manage.sh --action install
```

### Access

After installation:
- **Console UI**: http://localhost:9001 (web interface)
- **API Endpoint**: http://localhost:9000 (S3-compatible API)
- **Credentials**: Run `./manage.sh --action show-credentials`

## 📦 Default Buckets

MinIO automatically creates four buckets for Vrooli:

| Bucket | Purpose | Access Policy |
|--------|---------|---------------|
| `vrooli-user-uploads` | User profile pictures, attachments | Public read |
| `vrooli-agent-artifacts` | AI-generated images, documents | Private |
| `vrooli-model-cache` | Downloaded AI models | Private |
| `vrooli-temp-storage` | Temporary files (24hr auto-delete) | Private |

## 🔧 Common Operations

### Service Management

```bash
# Check status
./manage.sh --action status

# Start/stop/restart
./manage.sh --action start
./manage.sh --action stop
./manage.sh --action restart

# View logs
./manage.sh --action logs --lines 100

# Monitor health
./manage.sh --action monitor --interval 5
```

### Bucket Management

```bash
# List all buckets with statistics
./manage.sh --action list-buckets

# Create a new bucket
./manage.sh --action create-bucket --bucket my-bucket --policy download

# Remove a bucket
./manage.sh --action remove-bucket --bucket my-bucket
./manage.sh --action remove-bucket --bucket my-bucket --force yes  # For non-empty
```

### Credentials

```bash
# Show current credentials
./manage.sh --action show-credentials

# Reset credentials (generates new secure ones)
./manage.sh --action reset-credentials
```

### Testing

```bash
# Test file upload/download functionality
./manage.sh --action test-upload

# Run diagnostics
./manage.sh --action diagnose
```

## 🔐 Security

### Default Security

- Generates secure credentials on first install (if not provided)
- Credentials stored in `~/.minio/config/credentials` with 600 permissions
- Network isolated via Docker network
- Health checks enabled

### Custom Credentials

Set before installation:
```bash
export MINIO_CUSTOM_ROOT_USER="myuser"
export MINIO_CUSTOM_ROOT_PASSWORD="mysecurepassword"
./manage.sh --action install
```

### Bucket Policies

Available policies:
- `none` - Private (default)
- `download` - Public read access
- `upload` - Public write access
- `public` - Full public access

## 🏗️ Architecture

### Directory Structure

```
~/.minio/
├── data/           # Object storage data
│   └── .minio.sys/ # System metadata
└── config/         # Configuration and credentials
    └── credentials # Access credentials file
```

### Docker Configuration

- **Container**: `minio`
- **Network**: `minio-network`
- **Ports**: 
  - 9000 (API)
  - 9001 (Console)
- **Volumes**:
  - `~/.minio/data:/data`
  - `~/.minio/config:/root/.minio`

## 🔌 Integration with Vrooli

MinIO automatically registers with Vrooli's resource discovery system. The configuration is stored in `~/.vrooli/resources.local.json`:

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
        "buckets": {
          "userUploads": "vrooli-user-uploads",
          "agentArtifacts": "vrooli-agent-artifacts",
          "modelCache": "vrooli-model-cache",
          "tempStorage": "vrooli-temp-storage"
        }
      }
    }
  }
}
```

## 🛠️ Advanced Usage

### Using MinIO Client (mc)

The MinIO client is available inside the container:

```bash
# Execute mc commands
docker exec minio mc ls local/

# Create alias for external mc client
mc alias set vrooli http://localhost:9000 $(./manage.sh --action show-credentials | grep Username | cut -d' ' -f2) $(./manage.sh --action show-credentials | grep Password | cut -d' ' -f2)
```

### Lifecycle Policies

Temporary storage bucket has automatic cleanup:
```bash
# Files older than 24 hours are automatically deleted
# To set custom lifecycle:
docker exec minio mc ilm add local/my-bucket --expire-days 7
```

### Backup and Restore

```bash
# Backup all data
tar -czf minio-backup-$(date +%Y%m%d).tar.gz ~/.minio/data/

# Restore from backup
./manage.sh --action stop
tar -xzf minio-backup-20240115.tar.gz -C ~/
./manage.sh --action start
```

## 🐛 Troubleshooting

### Common Issues

**Port conflicts**
```bash
# Check what's using the port
sudo lsof -i :9000

# Use custom ports
MINIO_CUSTOM_PORT=9100 ./manage.sh --action install
```

**Container won't start**
```bash
# Check logs
./manage.sh --action logs --lines 100

# Run diagnostics
./manage.sh --action diagnose

# Check disk space
df -h ~/.minio/
```

**Can't access console**
```bash
# Verify container is running
docker ps | grep minio

# Check network
curl -I http://localhost:9001

# Ensure credentials are correct
./manage.sh --action show-credentials
```

### Reset Everything

```bash
# Complete uninstall including data
./manage.sh --action uninstall --remove-data yes

# Fresh install
./manage.sh --action install
```

## 📊 Performance Tuning

### Environment Variables

```bash
# Increase connection limit
MINIO_API_REQUESTS_MAX=1000 ./manage.sh --action install

# Set region
MINIO_REGION=us-west-1 ./manage.sh --action install
```

### Resource Limits

Edit docker run command in `lib/docker.sh` to add:
```bash
--memory="2g" \
--cpus="2.0" \
```

## 🔄 Maintenance

### Upgrade MinIO

```bash
# Upgrade to latest version
./manage.sh --action upgrade
```

### Monitoring

```bash
# Real-time monitoring
./manage.sh --action monitor

# Check resource usage
docker stats minio

# Disk usage
du -sh ~/.minio/data/
```

## 📝 Notes

- MinIO data persists between container restarts
- Uninstalling preserves data by default (use `--remove-data yes` to delete)
- Console provides a user-friendly interface for bucket management
- S3-compatible API enables easy migration to cloud storage
- Supports presigned URLs for secure temporary access

## 🔗 Resources

- [MinIO Documentation](https://docs.min.io/)
- [MinIO Client Guide](https://docs.min.io/docs/minio-client-complete-guide.html)
- [S3 API Reference](https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html)
- [Vrooli Resource System](/packages/server/src/services/resources/README.md)