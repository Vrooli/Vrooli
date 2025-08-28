# MinIO API Reference

MinIO provides a comprehensive S3-compatible API for object storage operations, plus management script APIs for service administration.

## 🔌 S3-Compatible API

### Access Points

After installation:
- **Console UI**: http://localhost:9001 (web interface)
- **API Endpoint**: http://localhost:9000 (S3-compatible API)
- **Health Check**: http://localhost:9000/minio/health/live

### Authentication

```bash
# Get current credentials
resource-minio credentials

# Example output:
# Username: minioadmin
# Password: minio123
```

### Basic S3 Operations

#### Using curl
```bash
# List buckets
curl -X GET \
  --header "Authorization: AWS4-HMAC-SHA256 ..." \
  http://localhost:9000/

# Upload file
curl -X PUT \
  --data-binary @file.txt \
  --header "Authorization: AWS4-HMAC-SHA256 ..." \
  http://localhost:9000/bucket-name/file.txt

# Download file
curl -X GET \
  --header "Authorization: AWS4-HMAC-SHA256 ..." \
  http://localhost:9000/bucket-name/file.txt -o downloaded-file.txt
```

#### Using AWS CLI
```bash
# Configure AWS CLI for MinIO
aws configure set aws_access_key_id minioadmin
aws configure set aws_secret_access_key minio123
aws configure set default.region us-east-1

# Use S3 commands with custom endpoint
export ENDPOINT_URL="http://localhost:9000"

# List buckets
aws s3 ls --endpoint-url $ENDPOINT_URL

# Upload file
aws s3 cp file.txt s3://bucket-name/ --endpoint-url $ENDPOINT_URL

# Download file
aws s3 cp s3://bucket-name/file.txt downloaded.txt --endpoint-url $ENDPOINT_URL

# Sync directory
aws s3 sync ./local-dir s3://bucket-name/remote-dir/ --endpoint-url $ENDPOINT_URL
```

## 🛠️ Management Script API

### Service Management

```bash
# Check status
resource-minio status

# Start/stop/restart service
resource-minio manage start
resource-minio manage stop
resource-minio manage restart

# View logs
resource-minio logs --tail 100

# Monitor health with interval
resource-minio content execute --name monitor --interval 5

# Run diagnostics
resource-minio content execute --name diagnose
```

#### Status Response Format
```bash
# Example status output:
✅ MinIO is running (Container ID: abc123...)
📊 API Endpoint: http://localhost:9000
🖥️  Console: http://localhost:9001
📦 Buckets: 4 configured
🔐 Credentials: Set (use 'resource-minio credentials' to view)
```

### Bucket Management

```bash
# List all buckets with statistics
resource-minio content list

# Create a new bucket
resource-minio content execute --name create-bucket --bucket my-bucket --policy download

# Remove a bucket (must be empty)
resource-minio content execute --name remove-bucket --bucket my-bucket

# Force remove non-empty bucket
resource-minio content execute --name remove-bucket --bucket my-bucket --force yes
```

#### Bucket Policies

| Policy | Description | Use Cases |
|--------|-------------|-----------|
| `none` | Private (default) | Sensitive data, internal files |
| `download` | Public read access | Static assets, public downloads |
| `upload` | Public write access | User uploads, form submissions |
| `public` | Full public access | Public file sharing |

#### List Buckets Output
```bash
# Example output:
📦 vrooli-user-uploads (Policy: download, Size: 1.2GB, Objects: 245)
📦 vrooli-agent-artifacts (Policy: none, Size: 3.4GB, Objects: 1,456)
📦 vrooli-model-cache (Policy: none, Size: 15.7GB, Objects: 23)
📦 vrooli-temp-storage (Policy: none, Size: 234MB, Objects: 67)
```

### Credential Management

```bash
# Show current credentials
resource-minio credentials

# Reset credentials (generates new secure ones)
resource-minio content execute --name reset-credentials
```

#### Credentials Format
```bash
# Standard output format:
MinIO Credentials:
Username: minioadmin
Password: minio123
Endpoint: http://localhost:9000
Console: http://localhost:9001
```

### Testing and Validation

```bash
# Test file upload/download functionality
resource-minio test smoke

# Run comprehensive diagnostics
resource-minio content execute --name diagnose
```

#### Test Upload Process
1. Creates temporary test file
2. Uploads to test bucket
3. Downloads and verifies integrity
4. Cleans up test files
5. Reports success/failure with details

## 🎯 Advanced API Operations

### MinIO Client (mc) Integration

Execute MinIO client commands through Docker:

```bash
# List objects in bucket
docker exec minio mc ls local/bucket-name/

# Copy file to bucket
docker exec minio mc cp /tmp/file.txt local/bucket-name/

# Mirror directory to bucket
docker exec minio mc mirror /source/dir local/bucket-name/target/

# Set bucket policy
docker exec minio mc policy set download local/bucket-name

# Create alias for external mc client
CREDS=$(resource-minio credentials)
USERNAME=$(echo "$CREDS" | grep Username | cut -d' ' -f2)
PASSWORD=$(echo "$CREDS" | grep Password | cut -d' ' -f2)
mc alias set vrooli http://localhost:9000 $USERNAME $PASSWORD
```

### Lifecycle Management

```bash
# Set expiration policy (files older than 7 days)
docker exec minio mc ilm add local/bucket-name --expire-days 7

# List lifecycle rules
docker exec minio mc ilm ls local/bucket-name

# Remove lifecycle rule
docker exec minio mc ilm rm --id rule-id local/bucket-name
```

### Event Notifications

```bash
# Set up webhook notification
docker exec minio mc event add local/bucket-name arn:minio:sqs::webhook:http://localhost:8080/webhook

# List event configurations
docker exec minio mc event list local/bucket-name

# Remove event notification
docker exec minio mc event remove local/bucket-name arn:minio:sqs::webhook:http://localhost:8080/webhook
```

## 📊 Monitoring and Health Checks

### Health Endpoints

```bash
# Liveness check
curl http://localhost:9000/minio/health/live

# Readiness check
curl http://localhost:9000/minio/health/ready

# Cluster health (for distributed setups)
curl http://localhost:9000/minio/health/cluster
```

### Resource Monitoring

```bash
# Real-time container stats
docker stats minio

# Disk usage
du -sh ~/.minio/data/

# API metrics
curl http://localhost:9000/minio/v2/metrics/cluster
```

### Log Analysis

```bash
# View recent logs
./manage.sh --action logs --lines 100

# Follow logs in real-time
docker logs -f minio

# Filter for errors
docker logs minio 2>&1 | grep -i error

# API access logs
docker logs minio 2>&1 | grep "API:"
```

## 🔗 Integration Examples

### Python (boto3)

```python
import boto3
from botocore.client import Config

# Configure client
s3_client = boto3.client(
    's3',
    endpoint_url='http://localhost:9000',
    aws_access_key_id='minioadmin',
    aws_secret_access_key='minio123',
    config=Config(signature_version='s3v4'),
    region_name='us-east-1'
)

# Upload file
s3_client.upload_file('local-file.txt', 'bucket-name', 'remote-file.txt')

# Download file
s3_client.download_file('bucket-name', 'remote-file.txt', 'downloaded-file.txt')

# List objects
response = s3_client.list_objects_v2(Bucket='bucket-name')
for obj in response.get('Contents', []):
    print(f"Object: {obj['Key']}, Size: {obj['Size']}")
```

### Node.js (minio)

```javascript
const Minio = require('minio');

// Configure client
const minioClient = new Minio.Client({
    endPoint: 'localhost',
    port: 9000,
    useSSL: false,
    accessKey: 'minioadmin',
    secretKey: 'minio123'
});

// Upload file
minioClient.fPutObject('bucket-name', 'object-name', 'file-path')
    .then(() => console.log('Upload successful'))
    .catch(err => console.error('Upload failed:', err));

// Download file
minioClient.fGetObject('bucket-name', 'object-name', 'download-path')
    .then(() => console.log('Download successful'))
    .catch(err => console.error('Download failed:', err));

// List objects
const stream = minioClient.listObjects('bucket-name', '', true);
stream.on('data', obj => console.log(obj));
stream.on('error', err => console.error(err));
```

### Bash/Shell Scripts

```bash
#!/bin/bash

# Configuration
ENDPOINT="http://localhost:9000"
BUCKET="my-bucket"
CREDS=$(resource-minio credentials)
ACCESS_KEY=$(echo "$CREDS" | grep Username | cut -d' ' -f2)
SECRET_KEY=$(echo "$CREDS" | grep Password | cut -d' ' -f2)

# Upload function
upload_file() {
    local file="$1"
    local key="$2"
    
    aws s3 cp "$file" "s3://$BUCKET/$key" \
        --endpoint-url "$ENDPOINT" \
        --profile minio
}

# Download function
download_file() {
    local key="$1"
    local file="$2"
    
    aws s3 cp "s3://$BUCKET/$key" "$file" \
        --endpoint-url "$ENDPOINT" \
        --profile minio
}

# List objects
list_objects() {
    aws s3 ls "s3://$BUCKET/" \
        --endpoint-url "$ENDPOINT" \
        --profile minio
}
```

## 🔄 Backup and Restore API

### Data Export

```bash
# Export bucket data
docker exec minio mc mirror local/bucket-name /backup/bucket-name

# Export with metadata
docker exec minio mc mirror --preserve local/bucket-name /backup/bucket-name

# Export compressed
docker exec -it minio sh -c "mc mirror local/bucket-name /tmp/backup && tar -czf /backup/bucket-backup.tar.gz -C /tmp backup"
```

### Data Import

```bash
# Import bucket data
docker exec minio mc mirror /backup/bucket-name local/bucket-name

# Import with overwrite
docker exec minio mc mirror --overwrite /backup/bucket-name local/bucket-name
```

## 📈 Performance Optimization

### Connection Tuning

```bash
# Increase max requests (set during installation)
MINIO_API_REQUESTS_MAX=1000 ./manage.sh --action install

# Configure concurrent connections
export MINIO_API_REQUESTS_DEADLINE=10s
resource-minio manage restart
```

### Batch Operations

```bash
# Parallel uploads using mc
docker exec minio mc cp --recursive /source/dir local/bucket-name/

# Batch delete
docker exec minio mc rm --recursive --force local/bucket-name/prefix/
```

This API reference provides comprehensive coverage of MinIO's storage capabilities and management operations.