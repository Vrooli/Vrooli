# MinIO S3 Integration Examples

This directory contains practical examples of integrating MinIO object storage with various applications and workflows.

## 📚 Example Categories

### Basic Operations
- File upload and download
- Bucket creation and management
- Credential configuration

### Programming Language Integration
- Python (boto3) examples
- Node.js (minio client) examples
- Shell script automation

### AI Resource Integration
- Model artifact storage
- Generated content management
- Temporary file handling

### Advanced Workflows
- Backup and restore procedures
- Lifecycle policy management
- Event notifications

## 🚀 Prerequisites

Before running examples, ensure MinIO is installed and running:

```bash
# Check installation
./manage.sh --action status

# If not installed
./manage.sh --action install

# Get credentials for examples
./manage.sh --action show-credentials
```

## 🔧 Basic Setup for Examples

Most examples require these basic configurations:

### Environment Variables
```bash
# Set these based on your MinIO credentials
export MINIO_ENDPOINT="http://localhost:9000"
export MINIO_ACCESS_KEY="minioadmin"  # From show-credentials
export MINIO_SECRET_KEY="minio123"    # From show-credentials
export MINIO_REGION="us-east-1"
```

### AWS CLI Configuration (for AWS CLI examples)
```bash
# Configure AWS CLI for MinIO
aws configure set aws_access_key_id $MINIO_ACCESS_KEY
aws configure set aws_secret_access_key $MINIO_SECRET_KEY
aws configure set default.region $MINIO_REGION

# Test configuration
aws s3 ls --endpoint-url $MINIO_ENDPOINT
```

## 📁 Example Structure

Each example directory contains:
- **README.md** - Detailed instructions and explanation
- **example script** - Runnable code (Python, Node.js, shell)
- **requirements** - Dependencies (package.json, requirements.txt, etc.)
- **sample data** - Test files when applicable

## 🛡️ Safety Notes

- Examples use safe operations on test buckets
- Always review scripts before running
- Some examples create temporary test data
- Clean up scripts are provided where applicable

## 🔄 Available Examples

### python-boto3/
Complete Python integration using the boto3 library with examples for:
- Basic file operations
- Bucket management
- Presigned URLs
- Error handling

### nodejs-minio/
Node.js integration using the official MinIO client with examples for:
- Async/await patterns
- Stream handling
- Event notifications
- Bulk operations

### shell-scripts/
Shell script automation examples for:
- Backup workflows
- Batch file processing
- Integration with other services
- Monitoring and alerts

### ai-integration/
Integration with Vrooli's AI resources:
- Model artifact storage
- Generated content management
- Agent screenshot storage
- Temporary file cleanup

### advanced-workflows/
Complex automation patterns:
- Multi-bucket synchronization
- Lifecycle policy automation
- Monitoring and alerting
- Backup and disaster recovery

## 🔍 Quick Test

Run this quick test to verify your MinIO setup works with the examples:

```bash
# Create test file
echo "Hello MinIO!" > test-file.txt

# Upload using AWS CLI
aws s3 cp test-file.txt s3://vrooli-temp-storage/ --endpoint-url $MINIO_ENDPOINT

# List files
aws s3 ls s3://vrooli-temp-storage/ --endpoint-url $MINIO_ENDPOINT

# Download file
aws s3 cp s3://vrooli-temp-storage/test-file.txt downloaded-file.txt --endpoint-url $MINIO_ENDPOINT

# Verify content
cat downloaded-file.txt

# Cleanup
rm test-file.txt downloaded-file.txt
aws s3 rm s3://vrooli-temp-storage/test-file.txt --endpoint-url $MINIO_ENDPOINT
```

If this test works, you're ready to run the examples!

## 🚀 Getting Started

1. **Choose an example** based on your programming language or use case
2. **Read the example's README** for specific instructions
3. **Install dependencies** if required (pip install, npm install, etc.)
4. **Run the example** following the provided instructions
5. **Modify for your needs** - examples are designed to be adapted

Navigate to specific example directories for detailed instructions and runnable code.