#!/usr/bin/env bash
################################################################################
# MinIO + N8n Integration Example
# 
# Demonstrates using MinIO for N8n workflow file storage
# Shows how N8n workflows can store and retrieve files from MinIO
################################################################################

set -euo pipefail

# Configuration
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://localhost:9000}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minio123}"
WORKFLOW_BUCKET="n8n-workflows"
ASSETS_BUCKET="n8n-assets"

# N8n configuration
N8N_PORT="${N8N_PORT:-5678}"
N8N_ENDPOINT="http://localhost:${N8N_PORT}"

# Load MinIO credentials if available
CREDS_FILE="${HOME}/.minio/config/credentials"
if [[ -f "$CREDS_FILE" ]]; then
    # shellcheck disable=SC1090
    source "$CREDS_FILE"
    MINIO_ACCESS_KEY="${MINIO_ROOT_USER:-$MINIO_ACCESS_KEY}"
    MINIO_SECRET_KEY="${MINIO_ROOT_PASSWORD:-$MINIO_SECRET_KEY}"
fi

echo "=== N8n + MinIO Integration Example ==="
echo ""

# Function to check services
check_services() {
    local all_ready=true
    
    echo "Checking service availability..."
    
    # Check MinIO
    if timeout 5 curl -sf "${MINIO_ENDPOINT}/minio/health/live" &>/dev/null; then
        echo "✓ MinIO is available at ${MINIO_ENDPOINT}"
    else
        echo "✗ MinIO is not available"
        echo "  Start with: vrooli resource minio develop"
        all_ready=false
    fi
    
    # Check N8n
    if timeout 5 curl -sf "${N8N_ENDPOINT}/healthz" &>/dev/null; then
        echo "✓ N8n is available at ${N8N_ENDPOINT}"
    else
        echo "✗ N8n is not available"
        echo "  Start with: vrooli resource n8n develop"
        all_ready=false
    fi
    
    if [[ "$all_ready" == "false" ]]; then
        return 1
    fi
}

# Function to setup buckets for N8n
setup_n8n_buckets() {
    echo ""
    echo "Setting up N8n storage buckets..."
    
    if command -v aws &>/dev/null; then
        export AWS_ACCESS_KEY_ID="$MINIO_ACCESS_KEY"
        export AWS_SECRET_ACCESS_KEY="$MINIO_SECRET_KEY"
        
        # Create workflow bucket
        if aws s3 ls "s3://${WORKFLOW_BUCKET}" --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1; then
            echo "✓ Workflow bucket already exists"
        else
            aws s3 mb "s3://${WORKFLOW_BUCKET}" --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1
            echo "✓ Created workflow bucket: ${WORKFLOW_BUCKET}"
        fi
        
        # Create assets bucket (public read for serving files)
        if aws s3 ls "s3://${ASSETS_BUCKET}" --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1; then
            echo "✓ Assets bucket already exists"
        else
            aws s3 mb "s3://${ASSETS_BUCKET}" --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1
            echo "✓ Created assets bucket: ${ASSETS_BUCKET}"
        fi
        
        # Make assets bucket public for download
        aws s3api put-bucket-policy \
            --bucket "${ASSETS_BUCKET}" \
            --endpoint-url "$MINIO_ENDPOINT" \
            --policy '{
                "Version": "2012-10-17",
                "Statement": [{
                    "Effect": "Allow",
                    "Principal": {"AWS": "*"},
                    "Action": ["s3:GetObject"],
                    "Resource": ["arn:aws:s3:::'"${ASSETS_BUCKET}"'/*"]
                }]
            }' &>/dev/null 2>&1 || true
            
        echo "✓ Configured public read access for assets bucket"
    else
        echo "⚠ AWS CLI not available, cannot create buckets"
        echo "  Install with: pip install awscli"
    fi
}

# Function to demonstrate workflow export/import via MinIO
demo_workflow_storage() {
    echo ""
    echo "Demonstrating workflow storage..."
    
    # Create a sample workflow JSON
    local workflow_file="/tmp/sample_workflow.json"
    cat > "$workflow_file" << 'EOF'
{
  "name": "MinIO File Storage Example",
  "nodes": [
    {
      "parameters": {
        "httpMethod": "POST",
        "path": "upload-to-minio",
        "responseMode": "onReceived",
        "responseData": "{ \"success\": true }"
      },
      "name": "Webhook",
      "type": "n8n-nodes-base.webhook",
      "position": [250, 300]
    },
    {
      "parameters": {
        "operation": "upload",
        "bucketName": "n8n-assets",
        "fileName": "={{ $json.filename }}"
      },
      "name": "Upload to MinIO",
      "type": "n8n-nodes-base.s3",
      "position": [450, 300]
    }
  ],
  "connections": {
    "Webhook": {
      "main": [[{"node": "Upload to MinIO", "type": "main", "index": 0}]]
    }
  }
}
EOF
    
    if command -v aws &>/dev/null; then
        export AWS_ACCESS_KEY_ID="$MINIO_ACCESS_KEY"
        export AWS_SECRET_ACCESS_KEY="$MINIO_SECRET_KEY"
        
        # Upload workflow to MinIO
        local timestamp=$(date +%Y%m%d_%H%M%S)
        local workflow_name="workflow_example_${timestamp}.json"
        
        if aws s3 cp "$workflow_file" "s3://${WORKFLOW_BUCKET}/${workflow_name}" \
            --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1; then
            echo "✓ Uploaded sample workflow to MinIO"
            echo "  Location: ${WORKFLOW_BUCKET}/${workflow_name}"
            
            # List workflows in bucket
            echo ""
            echo "Workflows stored in MinIO:"
            aws s3 ls "s3://${WORKFLOW_BUCKET}/" --endpoint-url "$MINIO_ENDPOINT" 2>/dev/null | \
                while read -r line; do
                echo "  $line"
            done
        fi
        
        # Upload a sample asset
        local asset_file="/tmp/sample_asset.txt"
        echo "This is a sample asset stored in MinIO for N8n workflows" > "$asset_file"
        
        if aws s3 cp "$asset_file" "s3://${ASSETS_BUCKET}/sample_asset.txt" \
            --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1; then
            echo ""
            echo "✓ Uploaded sample asset to MinIO"
            echo "  Public URL: ${MINIO_ENDPOINT}/${ASSETS_BUCKET}/sample_asset.txt"
        fi
        
        rm -f "$asset_file"
    else
        echo "✗ AWS CLI required for demonstration"
    fi
    
    rm -f "$workflow_file"
}

# Function to show integration configuration
show_integration_config() {
    echo ""
    echo "=== N8n Configuration for MinIO ==="
    echo ""
    echo "To configure N8n to use MinIO for file storage:"
    echo ""
    echo "1. Add S3 credentials in N8n:"
    echo "   - Go to: ${N8N_ENDPOINT}/credentials/new"
    echo "   - Select: AWS S3"
    echo "   - Configuration:"
    echo "     • Access Key ID: ${MINIO_ACCESS_KEY}"
    echo "     • Secret Access Key: ${MINIO_SECRET_KEY}"
    echo "     • Region: us-east-1"
    echo "     • Custom Endpoint: ${MINIO_ENDPOINT}"
    echo ""
    echo "2. Use S3 nodes in workflows:"
    echo "   - Upload files to MinIO buckets"
    echo "   - Download files from MinIO"
    echo "   - List bucket contents"
    echo "   - Generate presigned URLs"
    echo ""
    echo "3. Example use cases:"
    echo "   - Store workflow execution results"
    echo "   - Archive processed files"
    echo "   - Share files between workflows"
    echo "   - Serve static assets"
}

# Main execution
main() {
    echo "This example demonstrates using MinIO with N8n workflows"
    echo "--------------------------------------------------------"
    echo ""
    
    # Check prerequisites
    if ! check_services; then
        exit 1
    fi
    
    # Setup buckets
    setup_n8n_buckets
    
    # Demo workflow storage
    demo_workflow_storage
    
    # Show configuration
    show_integration_config
    
    echo ""
    echo "=== Example Complete ==="
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi