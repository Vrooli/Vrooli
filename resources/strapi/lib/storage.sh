#!/usr/bin/env bash
# Strapi Storage Integration Library
# Provides S3/MinIO integration for media storage

set -euo pipefail

# Prevent multiple sourcing
[[ -n "${STRAPI_STORAGE_LOADED:-}" ]] && return 0
readonly STRAPI_STORAGE_LOADED=1

# Source core library
STORAGE_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${STORAGE_SCRIPT_DIR}/lib/core.sh"

#######################################
# Check if MinIO is available
#######################################
storage::minio_available() {
    local minio_endpoint="${MINIO_ENDPOINT:-http://localhost:9000}"
    
    # Check if MinIO is running
    if timeout 2 curl -sf "${minio_endpoint}/minio/health/live" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

#######################################
# Configure S3 storage provider
#######################################
storage::configure_s3() {
    local config_file="${STRAPI_CONFIG_DIR:-/app/config}/plugins.js"
    
    if ! storage::minio_available; then
        core::warning "MinIO is not available, using local storage"
        return 2
    fi
    
    core::info "Configuring S3 storage provider..."
    
    # Create plugins configuration
    cat > "${config_file}" << 'EOF'
module.exports = ({ env }) => ({
  upload: {
    config: {
      provider: '@strapi/provider-upload-aws-s3',
      providerOptions: {
        s3Options: {
          endpoint: env('MINIO_ENDPOINT', 'http://localhost:9000'),
          accessKeyId: env('MINIO_ACCESS_KEY', 'minioadmin'),
          secretAccessKey: env('MINIO_SECRET_KEY', 'minioadmin'),
          region: env('AWS_REGION', 'us-east-1'),
          s3ForcePathStyle: true,
          signatureVersion: 'v4',
        },
        baseUrl: env('CDN_URL', 'http://localhost:9000'),
        bucket: env('S3_BUCKET', 'strapi-uploads'),
      },
      actionOptions: {
        upload: {},
        uploadStream: {},
        delete: {},
      },
    },
  },
});
EOF
    
    core::success "S3 storage provider configured"
}

#######################################
# Create S3 bucket for uploads
#######################################
storage::create_bucket() {
    local bucket_name="${S3_BUCKET:-strapi-uploads}"
    local minio_endpoint="${MINIO_ENDPOINT:-http://localhost:9000}"
    
    if ! storage::minio_available; then
        core::error "MinIO is not available"
        return 1
    fi
    
    core::info "Creating S3 bucket: ${bucket_name}"
    
    # Use MinIO client if available
    if command -v mc >/dev/null 2>&1; then
        # Configure MinIO client
        mc alias set minio "${minio_endpoint}" \
            "${MINIO_ACCESS_KEY:-minioadmin}" \
            "${MINIO_SECRET_KEY:-minioadmin}" >/dev/null 2>&1
        
        # Create bucket
        mc mb "minio/${bucket_name}" --ignore-existing
        
        # Set public read policy for uploads
        mc anonymous set download "minio/${bucket_name}"
        
        core::success "S3 bucket created and configured"
    else
        core::warning "MinIO client not available, bucket must be created manually"
    fi
}

#######################################
# List uploaded files in S3
#######################################
storage::list_files() {
    local bucket_name="${S3_BUCKET:-strapi-uploads}"
    local minio_endpoint="${MINIO_ENDPOINT:-http://localhost:9000}"
    
    if ! storage::minio_available; then
        core::error "MinIO is not available"
        return 1
    fi
    
    if command -v mc >/dev/null 2>&1; then
        echo "Files in S3 bucket '${bucket_name}':"
        echo "======================================"
        mc ls "minio/${bucket_name}" --recursive
    else
        core::error "MinIO client not available"
        return 1
    fi
}

#######################################
# Test S3 storage integration
#######################################
storage::test() {
    core::info "Testing S3 storage integration..."
    
    # Check MinIO availability
    if ! storage::minio_available; then
        core::error "MinIO is not available at ${MINIO_ENDPOINT:-http://localhost:9000}"
        return 1
    fi
    core::success "MinIO is available"
    
    # Test bucket creation
    if storage::create_bucket; then
        core::success "Bucket operations working"
    else
        core::error "Failed to create/access bucket"
        return 1
    fi
    
    # Test file upload (if Strapi is running)
    if core::is_running; then
        local test_file="/tmp/strapi-test-upload.txt"
        echo "Test upload file" > "${test_file}"
        
        # Upload via Strapi API
        local response=$(curl -sf -X POST \
            -H "Authorization: Bearer ${STRAPI_API_TOKEN:-}" \
            -F "files=@${test_file}" \
            "http://localhost:${STRAPI_PORT:-1337}/api/upload" 2>/dev/null || echo "")
        
        if [[ -n "$response" ]]; then
            core::success "File upload via API working"
        else
            core::warning "Could not test file upload (API token may be required)"
        fi
        
        rm -f "${test_file}"
    else
        core::info "Strapi not running, skipping upload test"
    fi
    
    core::success "S3 storage integration test completed"
}

#######################################
# Enable S3 storage in Strapi
#######################################
storage::enable() {
    core::info "Enabling S3 storage for Strapi..."
    
    # Configure S3 provider
    storage::configure_s3
    
    # Create bucket
    storage::create_bucket
    
    # Test integration
    storage::test
    
    core::success "S3 storage enabled for Strapi"
}

#######################################
# Disable S3 storage (revert to local)
#######################################
storage::disable() {
    core::info "Disabling S3 storage, reverting to local storage..."
    
    local config_file="${STRAPI_CONFIG_DIR:-/app/config}/plugins.js"
    
    # Remove S3 configuration
    if [[ -f "${config_file}" ]]; then
        rm -f "${config_file}"
        core::success "S3 storage disabled, using local storage"
    else
        core::info "S3 storage was not configured"
    fi
}