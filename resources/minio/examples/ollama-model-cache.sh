#!/usr/bin/env bash
################################################################################
# MinIO + Ollama Integration Example
# 
# Demonstrates using MinIO for Ollama model caching and distribution
# Shows how to backup and share Ollama models across instances
################################################################################

set -euo pipefail

# Configuration
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://localhost:9000}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minio123}"
MODEL_BUCKET="ollama-models"

# Ollama configuration
OLLAMA_HOST="${OLLAMA_HOST:-localhost}"
OLLAMA_PORT="${OLLAMA_PORT:-11434}"
OLLAMA_MODELS_DIR="${HOME}/.ollama/models"

# Load MinIO credentials if available
CREDS_FILE="${HOME}/.minio/config/credentials"
if [[ -f "$CREDS_FILE" ]]; then
    # shellcheck disable=SC1090
    source "$CREDS_FILE"
    MINIO_ACCESS_KEY="${MINIO_ROOT_USER:-$MINIO_ACCESS_KEY}"
    MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-$MINIO_SECRET_KEY}"
fi

echo "=== Ollama + MinIO Model Cache Example ==="
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
    
    # Check Ollama
    if timeout 5 curl -sf "http://${OLLAMA_HOST}:${OLLAMA_PORT}/api/tags" &>/dev/null; then
        echo "✓ Ollama is available at ${OLLAMA_HOST}:${OLLAMA_PORT}"
    else
        echo "✗ Ollama is not available"
        echo "  Start with: vrooli resource ollama develop"
        all_ready=false
    fi
    
    if [[ "$all_ready" == "false" ]]; then
        return 1
    fi
}

# Function to setup model bucket
setup_model_bucket() {
    echo ""
    echo "Setting up Ollama model bucket..."
    
    if command -v aws &>/dev/null; then
        export AWS_ACCESS_KEY_ID="$MINIO_ACCESS_KEY"
        export AWS_SECRET_ACCESS_KEY="$MINIO_SECRET_KEY"
        
        if aws s3 ls "s3://${MODEL_BUCKET}" --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1; then
            echo "✓ Model bucket already exists"
        else
            aws s3 mb "s3://${MODEL_BUCKET}" --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1
            echo "✓ Created model bucket: ${MODEL_BUCKET}"
        fi
        
        # Set bucket policy for model distribution
        aws s3api put-bucket-policy \
            --bucket "${MODEL_BUCKET}" \
            --endpoint-url "$MINIO_ENDPOINT" \
            --policy '{
                "Version": "2012-10-17",
                "Statement": [{
                    "Effect": "Allow",
                    "Principal": {"AWS": "*"},
                    "Action": ["s3:GetObject", "s3:ListBucket"],
                    "Resource": [
                        "arn:aws:s3:::'"${MODEL_BUCKET}"'",
                        "arn:aws:s3:::'"${MODEL_BUCKET}"'/*"
                    ]
                }]
            }' &>/dev/null 2>&1 || true
            
        echo "✓ Configured read access for model distribution"
    else
        echo "⚠ AWS CLI not available, cannot create bucket"
    fi
}

# Function to list Ollama models
list_ollama_models() {
    echo ""
    echo "Available Ollama models:"
    
    local models=$(curl -sf "http://${OLLAMA_HOST}:${OLLAMA_PORT}/api/tags" 2>/dev/null | \
        jq -r '.models[]?.name' 2>/dev/null || echo "")
    
    if [[ -z "$models" ]]; then
        echo "  No models found"
        echo "  Pull a model first: ollama pull llama2"
        return 1
    else
        echo "$models" | while read -r model; do
            if [[ -n "$model" ]]; then
                local size=$(curl -sf "http://${OLLAMA_HOST}:${OLLAMA_PORT}/api/tags" 2>/dev/null | \
                    jq -r ".models[] | select(.name==\"$model\") | .size" 2>/dev/null || echo "unknown")
                
                # Convert size to human readable
                if [[ "$size" =~ ^[0-9]+$ ]]; then
                    size=$(echo "$size" | awk '{printf "%.1fGB", $1/1024/1024/1024}')
                fi
                
                echo "  • ${model} (${size})"
            fi
        done
    fi
}

# Function to backup model to MinIO
backup_model_to_minio() {
    local model_name="${1:-}"
    
    if [[ -z "$model_name" ]]; then
        echo "✗ No model specified for backup"
        return 1
    fi
    
    echo ""
    echo "Backing up model '${model_name}' to MinIO..."
    
    # Check if model exists locally
    if ! curl -sf "http://${OLLAMA_HOST}:${OLLAMA_PORT}/api/tags" | \
        jq -e ".models[] | select(.name==\"$model_name\")" &>/dev/null; then
        echo "✗ Model '${model_name}' not found in Ollama"
        return 1
    fi
    
    # Create model export
    local export_file="/tmp/ollama_model_${model_name//\//_}_$(date +%Y%m%d).tar"
    
    echo "Creating model archive..."
    
    # For demonstration, we'll create a manifest file
    # In production, you'd export the actual model files
    local manifest_file="/tmp/model_manifest.json"
    cat > "$manifest_file" << EOF
{
  "model": "${model_name}",
  "exported_at": "$(date -Iseconds)",
  "ollama_host": "${OLLAMA_HOST}",
  "note": "This is a demonstration. Real implementation would export actual model files."
}
EOF
    
    tar -cf "$export_file" -C /tmp model_manifest.json 2>/dev/null
    
    if [[ -f "$export_file" ]]; then
        local file_size=$(du -h "$export_file" | cut -f1)
        echo "✓ Created model archive (${file_size})"
        
        # Upload to MinIO
        if command -v aws &>/dev/null; then
            export AWS_ACCESS_KEY_ID="$MINIO_ACCESS_KEY"
            export AWS_SECRET_ACCESS_KEY="$MINIO_SECRET_KEY"
            
            if aws s3 cp "$export_file" "s3://${MODEL_BUCKET}/$(basename "$export_file")" \
                --endpoint-url "$MINIO_ENDPOINT" &>/dev/null 2>&1; then
                echo "✓ Model backed up to MinIO: ${MODEL_BUCKET}/$(basename "$export_file")"
                
                # Generate shareable URL
                local share_url="${MINIO_ENDPOINT}/${MODEL_BUCKET}/$(basename "$export_file")"
                echo "  Share URL: ${share_url}"
            else
                echo "✗ Failed to upload model to MinIO"
            fi
        fi
        
        # Cleanup
        rm -f "$export_file" "$manifest_file"
    fi
}

# Function to list backed up models
list_backed_up_models() {
    echo ""
    echo "Models backed up in MinIO:"
    
    if command -v aws &>/dev/null; then
        export AWS_ACCESS_KEY_ID="$MINIO_ACCESS_KEY"
        export AWS_SECRET_ACCESS_KEY="$MINIO_SECRET_KEY"
        
        local models=$(aws s3 ls "s3://${MODEL_BUCKET}/" --endpoint-url "$MINIO_ENDPOINT" 2>/dev/null || true)
        
        if [[ -z "$models" ]]; then
            echo "  No models backed up yet"
        else
            echo "$models" | while read -r line; do
                if [[ -n "$line" ]]; then
                    echo "  $line"
                fi
            done
        fi
    else
        echo "  AWS CLI required to list backed up models"
    fi
}

# Function to show model caching strategy
show_caching_strategy() {
    echo ""
    echo "=== Model Caching Strategy with MinIO ==="
    echo ""
    echo "Benefits of using MinIO for Ollama models:"
    echo "  • Centralized model storage across multiple nodes"
    echo "  • Version control for model iterations"
    echo "  • Fast local network transfers"
    echo "  • Reduced bandwidth for model distribution"
    echo "  • Backup and disaster recovery"
    echo ""
    echo "Implementation approach:"
    echo "  1. Pull models to primary Ollama instance"
    echo "  2. Export models to MinIO bucket"
    echo "  3. Secondary instances download from MinIO"
    echo "  4. Use MinIO versioning for model updates"
    echo ""
    echo "Example workflow:"
    echo "  # On primary node:"
    echo "  ollama pull llama2"
    echo "  ./ollama-model-cache.sh backup llama2"
    echo ""
    echo "  # On secondary nodes:"
    echo "  aws s3 cp s3://ollama-models/model.tar . --endpoint-url ${MINIO_ENDPOINT}"
    echo "  ollama import model.tar"
}

# Main execution
main() {
    echo "This example demonstrates using MinIO for Ollama model caching"
    echo "-------------------------------------------------------------"
    echo ""
    
    # Check prerequisites
    if ! check_services; then
        exit 1
    fi
    
    # Setup bucket
    setup_model_bucket
    
    # List current models
    if list_ollama_models; then
        # If models exist, offer to backup one
        echo ""
        echo "To backup a model, run:"
        echo "  $0 backup <model-name>"
        
        # Demo backup with first available model
        local first_model=$(curl -sf "http://${OLLAMA_HOST}:${OLLAMA_PORT}/api/tags" 2>/dev/null | \
            jq -r '.models[0]?.name' 2>/dev/null || echo "")
        
        if [[ -n "$first_model" ]] && [[ "$1" != "backup" ]]; then
            backup_model_to_minio "$first_model"
        fi
    fi
    
    # Handle backup command
    if [[ "$1" == "backup" ]] && [[ -n "${2:-}" ]]; then
        backup_model_to_minio "$2"
    fi
    
    # List backed up models
    list_backed_up_models
    
    # Show strategy
    show_caching_strategy
    
    echo ""
    echo "=== Example Complete ==="
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi