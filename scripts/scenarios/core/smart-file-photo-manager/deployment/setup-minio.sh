#!/bin/bash
# MinIO setup for Smart File Photo Manager
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VROOLI_ROOT="$(cd "$SCENARIO_ROOT/../../.." && pwd)"

# Load environment variables
source "$VROOLI_ROOT/scripts/resources/lib/resource-helper.sh"

# Configuration
MINIO_HOST="localhost"
MINIO_PORT="9001"
MINIO_ACCESS_KEY="${MINIO_ROOT_USER:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_ROOT_PASSWORD:-minioadmin}"

# Get default minio port if available
if command -v "resources::get_default_port" &> /dev/null; then
    MINIO_PORT=$(resources::get_default_port "minio" || echo "9001")
fi

MINIO_URL="http://$MINIO_HOST:$MINIO_PORT"

# MinIO client alias
MC_ALIAS="filemanager"

log_info() {
    echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

log_success() {
    echo "[SUCCESS] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Wait for MinIO to be ready
wait_for_minio() {
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for MinIO on port $MINIO_PORT..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$MINIO_URL/minio/health/ready" | grep -q "200 OK" 2>/dev/null || \
           timeout 3 bash -c "</dev/tcp/$MINIO_HOST/$MINIO_PORT" 2>/dev/null; then
            log_success "MinIO is ready"
            return 0
        fi
        
        sleep 2
        ((attempt++))
    done
    
    log_error "MinIO failed to become ready"
    return 1
}

# Configure MinIO client
configure_mc() {
    log_info "Configuring MinIO client..."
    
    # Add MinIO server to mc config
    mc alias set "$MC_ALIAS" "$MINIO_URL" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" 2>/dev/null || {
        log_error "Failed to configure MinIO client"
        return 1
    }
    
    log_success "MinIO client configured successfully"
}

# Create bucket
create_bucket() {
    local bucket_name="$1"
    local description="$2"
    
    log_info "Creating bucket: $bucket_name"
    
    # Check if bucket exists
    if mc ls "$MC_ALIAS/$bucket_name" >/dev/null 2>&1; then
        log_info "Bucket '$bucket_name' already exists"
        return 0
    fi
    
    # Create bucket
    if mc mb "$MC_ALIAS/$bucket_name"; then
        log_success "Created bucket: $bucket_name"
    else
        log_error "Failed to create bucket: $bucket_name"
        return 1
    fi
}

# Set bucket policy
set_bucket_policy() {
    local bucket_name="$1"
    local policy="$2"
    
    log_info "Setting policy for bucket: $bucket_name"
    
    mc anonymous set "$policy" "$MC_ALIAS/$bucket_name" 2>/dev/null || {
        log_info "Policy setting skipped or failed for bucket: $bucket_name"
    }
}

# Enable versioning
enable_versioning() {
    local bucket_name="$1"
    
    log_info "Enabling versioning for bucket: $bucket_name"
    
    mc version enable "$MC_ALIAS/$bucket_name" 2>/dev/null || {
        log_info "Versioning setup skipped for bucket: $bucket_name"
    }
}

# Set lifecycle policy
set_lifecycle() {
    local bucket_name="$1"
    local retention_days="${2:-90}"
    
    log_info "Setting lifecycle policy for bucket: $bucket_name"
    
    # Create temporary lifecycle config
    local lifecycle_config="/tmp/${bucket_name}_lifecycle.json"
    
    cat > "$lifecycle_config" << EOF
{
    "Rules": [
        {
            "ID": "DeleteOldVersions",
            "Status": "Enabled",
            "Filter": {
                "Prefix": ""
            },
            "NoncurrentVersionExpiration": {
                "NoncurrentDays": $retention_days
            }
        },
        {
            "ID": "DeleteIncompleteUploads",
            "Status": "Enabled",
            "Filter": {
                "Prefix": ""
            },
            "AbortIncompleteMultipartUpload": {
                "DaysAfterInitiation": 7
            }
        }
    ]
}
EOF
    
    mc ilm import "$MC_ALIAS/$bucket_name" < "$lifecycle_config" 2>/dev/null || {
        log_info "Lifecycle policy setup skipped for bucket: $bucket_name"
    }
    
    rm -f "$lifecycle_config"
}

# Setup all buckets
setup_buckets() {
    log_info "Setting up MinIO buckets..."
    
    # Original files bucket
    create_bucket "original-files" "Original uploaded files"
    set_bucket_policy "original-files" "none"  # Private
    enable_versioning "original-files"
    set_lifecycle "original-files" "365"  # Keep originals longer
    
    # Processed files bucket
    create_bucket "processed-files" "AI-processed file versions"
    set_bucket_policy "processed-files" "none"  # Private
    set_lifecycle "processed-files" "90"
    
    # Thumbnails bucket
    create_bucket "thumbnails" "Generated thumbnails and previews"
    set_bucket_policy "thumbnails" "download"  # Read-only public access
    set_lifecycle "thumbnails" "30"  # Shorter retention for thumbnails
    
    # Exports bucket
    create_bucket "exports" "Temporary exports and downloads"
    set_bucket_policy "exports" "none"  # Private
    set_lifecycle "exports" "7"  # Short retention for exports
    
    log_success "All buckets created successfully"
}

# Create folder structure in buckets
create_folder_structure() {
    log_info "Creating folder structure in buckets..."
    
    local buckets=("original-files" "processed-files" "thumbnails")
    local folders=("images/" "documents/" "videos/" "audio/" "archives/" "temp/")
    
    for bucket in "${buckets[@]}"; do
        for folder in "${folders[@]}"; do
            # Create empty object to establish folder structure
            echo "" | mc pipe "$MC_ALIAS/$bucket/$folder.gitkeep" 2>/dev/null || true
        done
    done
    
    log_success "Folder structure created"
}

# Load bucket configuration if exists
load_bucket_config() {
    local config_file="$SCENARIO_ROOT/initialization/storage/minio/buckets.json"
    
    if [ -f "$config_file" ]; then
        log_info "Loading bucket configuration from $config_file"
        # This would process custom bucket configs if needed
        # For now, we use the standard setup above
        log_info "Using standard bucket configuration"
    fi
}

# Test basic operations
test_operations() {
    log_info "Testing basic MinIO operations..."
    
    # Create test file
    local test_file="/tmp/minio_test.txt"
    echo "MinIO test file" > "$test_file"
    
    # Upload test file
    if mc cp "$test_file" "$MC_ALIAS/original-files/test/"; then
        log_info "Upload test passed"
        
        # Download test file
        local download_file="/tmp/minio_test_download.txt"
        if mc cp "$MC_ALIAS/original-files/test/minio_test.txt" "$download_file"; then
            log_info "Download test passed"
            
            # Verify content
            if diff "$test_file" "$download_file" >/dev/null; then
                log_success "Basic operations test passed"
            else
                log_error "File content verification failed"
                return 1
            fi
        else
            log_error "Download test failed"
            return 1
        fi
    else
        log_error "Upload test failed"
        return 1
    fi
    
    # Clean up test files
    mc rm "$MC_ALIAS/original-files/test/minio_test.txt" 2>/dev/null || true
    rm -f "$test_file" "$download_file"
}

# Verify bucket setup
verify_buckets() {
    log_info "Verifying MinIO bucket setup..."
    
    local buckets=("original-files" "processed-files" "thumbnails" "exports")
    local all_good=true
    
    for bucket in "${buckets[@]}"; do
        if mc ls "$MC_ALIAS/$bucket" >/dev/null 2>&1; then
            log_success "Bucket '$bucket' is accessible"
        else
            log_error "Bucket '$bucket' is not accessible"
            all_good=false
        fi
    done
    
    if [ "$all_good" = true ]; then
        log_success "All buckets verified successfully"
    else
        log_error "Some buckets failed verification"
        return 1
    fi
}

# Main execution
main() {
    log_info "Setting up MinIO object storage for File Manager..."
    
    # Check if mc command is available
    if ! command -v mc >/dev/null 2>&1; then
        log_error "MinIO client (mc) not found. Please install it first."
        return 1
    fi
    
    # Wait for MinIO to be ready
    wait_for_minio
    
    # Configure MinIO client
    configure_mc
    
    # Load bucket configuration
    load_bucket_config
    
    # Setup buckets
    setup_buckets
    
    # Create folder structure
    create_folder_structure
    
    # Verify buckets
    verify_buckets
    
    # Test basic operations
    test_operations
    
    log_success "MinIO setup completed successfully"
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi