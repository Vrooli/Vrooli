#!/usr/bin/env bash
# MinIO API Functions
# Functions for interacting with MinIO S3 API

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
MINIO_API_DIR="${APP_ROOT}/resources/minio/lib"

# Source required dependencies
source "${MINIO_API_DIR}/common.sh"
source "${MINIO_API_DIR}/docker.sh"

# Source and initialize messages
source "${APP_ROOT}/resources/minio/config/messages.sh"
if command -v minio::messages::init &>/dev/null; then
    minio::messages::init
fi

#######################################
# Execute MinIO client command
# Arguments:
#   $@ - mc command arguments
# Returns: Command exit code
#######################################
minio::api::mc() {
    # Use docker exec to run mc commands inside the container
    minio::docker::exec mc "$@"
}

#######################################
# Configure MinIO client alias
# Returns: 0 on success, 1 on failure
#######################################
minio::api::configure_mc() {
    log::debug "Configuring MinIO client..."
    
    # Load credentials - check file first, then use defaults
    local user="${MINIO_ROOT_USER}"
    local password="${MINIO_ROOT_PASSWORD}"
    
    if [[ -f "${HOME}/.minio/config/credentials" ]]; then
        # Read credentials without sourcing to avoid readonly conflicts
        user=$(grep "^MINIO_ROOT_USER=" "${HOME}/.minio/config/credentials" | cut -d'=' -f2-)
        password=$(grep "^MINIO_ROOT_PASSWORD=" "${HOME}/.minio/config/credentials" | cut -d'=' -f2-)
        log::debug "Loaded credentials from file"
    fi
    
    # Set up mc alias for local MinIO (using new mc syntax)
    if minio::api::mc alias set local \
        "http://localhost:9000" \
        "${user}" \
        "${password}" \
        >/dev/null 2>&1; then
        log::debug "MinIO client configured successfully"
        return 0
    else
        log::error "Failed to configure MinIO client"
        return 1
    fi
}

#######################################
# Create a bucket
# Arguments:
#   $1 - Bucket name
# Returns: 0 on success, 1 on failure
#######################################
minio::api::create_bucket() {
    local bucket_name=$1
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name is required"
        return 1
    fi
    
    log::debug "Creating bucket: $bucket_name"
    
    # Check if bucket already exists
    if minio::api::mc ls "local/${bucket_name}" >/dev/null 2>&1; then
        log::debug "Bucket already exists: $bucket_name"
        return 0
    fi
    
    # Create the bucket
    if minio::api::mc mb "local/${bucket_name}" >/dev/null 2>&1; then
        log::success "${MSG_BUCKET_CREATED}: $bucket_name"
        return 0
    else
        log::error "${MSG_BUCKET_CREATE_FAILED}: $bucket_name"
        return 1
    fi
}

#######################################
# Set bucket policy
# Arguments:
#   $1 - Bucket name
#   $2 - Policy type (public, download, upload, none)
# Returns: 0 on success, 1 on failure
#######################################
minio::api::set_bucket_policy() {
    local bucket_name=$1
    local policy_type=$2
    
    if [[ -z "$bucket_name" || -z "$policy_type" ]]; then
        log::error "Bucket name and policy type are required"
        return 1
    fi
    
    log::debug "Setting policy '$policy_type' for bucket: $bucket_name"
    
    # Apply the policy
    if minio::api::mc anonymous set "$policy_type" "local/${bucket_name}" >/dev/null 2>&1; then
        log::debug "Policy set successfully for bucket: $bucket_name"
        return 0
    else
        log::error "Failed to set policy for bucket: $bucket_name"
        return 1
    fi
}

#######################################
# List all buckets
# Output: List of bucket names
#######################################
minio::api::list_buckets() {
    minio::api::mc ls local | awk '{print $NF}' | sed 's|/$||'
}

#######################################
# Get bucket size
# Arguments:
#   $1 - Bucket name
# Output: Bucket size in human readable format
#######################################
minio::api::get_bucket_size() {
    local bucket_name=$1
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name is required"
        return 1
    fi
    
    minio::api::mc du --summarize "local/${bucket_name}" 2>/dev/null | awk '{print $1}'
}

#######################################
# Upload a file
# Arguments:
#   $1 - Local file path
#   $2 - Bucket name
#   $3 - Object name (optional, defaults to filename)
# Returns: 0 on success, 1 on failure
#######################################
minio::api::upload_file() {
    local file_path=$1
    local bucket_name=$2
    local object_name=${3:-$(basename "$file_path")}
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name is required"
        return 1
    fi
    
    log::debug "Uploading $file_path to $bucket_name/$object_name"
    
    # Copy file to container first
    local temp_path="/tmp/upload-$(basename "$file_path")"
    if docker cp "$file_path" "${MINIO_CONTAINER_NAME}:${temp_path}" >/dev/null 2>&1; then
        # Upload to MinIO
        if minio::api::mc cp "$temp_path" "local/${bucket_name}/${object_name}" >/dev/null 2>&1; then
            # Clean up temp file
            minio::docker::exec rm -f "$temp_path"
            log::debug "File uploaded successfully"
            return 0
        else
            log::error "Failed to upload file to MinIO"
            minio::docker::exec rm -f "$temp_path"
            return 1
        fi
    else
        log::error "Failed to copy file to container"
        return 1
    fi
}

#######################################
# Download a file
# Arguments:
#   $1 - Bucket name
#   $2 - Object name
#   $3 - Local destination path
# Returns: 0 on success, 1 on failure
#######################################
minio::api::download_file() {
    local bucket_name=$1
    local object_name=$2
    local dest_path=$3
    
    if [[ -z "$bucket_name" || -z "$object_name" || -z "$dest_path" ]]; then
        log::error "Bucket name, object name, and destination path are required"
        return 1
    fi
    
    log::debug "Downloading $bucket_name/$object_name to $dest_path"
    
    # Download to container first
    local temp_path="/tmp/download-${object_name}"
    if minio::api::mc cp "local/${bucket_name}/${object_name}" "$temp_path" >/dev/null 2>&1; then
        # Copy from container to host
        if docker cp "${MINIO_CONTAINER_NAME}:${temp_path}" "$dest_path" >/dev/null 2>&1; then
            # Clean up temp file
            minio::docker::exec rm -f "$temp_path"
            log::debug "File downloaded successfully"
            return 0
        else
            log::error "Failed to copy file from container"
            minio::docker::exec rm -f "$temp_path"
            return 1
        fi
    else
        log::error "Failed to download file from MinIO"
        return 1
    fi
}

#######################################
# Set lifecycle policy for bucket
# Arguments:
#   $1 - Bucket name
#   $2 - Days to expire
# Returns: 0 on success, 1 on failure
#######################################
minio::api::set_lifecycle() {
    local bucket_name=$1
    local expire_days=$2
    
    if [[ -z "$bucket_name" || -z "$expire_days" ]]; then
        log::error "Bucket name and expiration days are required"
        return 1
    fi
    
    log::debug "Setting lifecycle policy for bucket: $bucket_name (expire after $expire_days days)"
    
    # Create lifecycle configuration
    local lifecycle_config=$(cat <<EOF
{
    "Rules": [{
        "ID": "expire-after-${expire_days}-days",
        "Status": "Enabled",
        "Expiration": {
            "Days": ${expire_days}
        }
    }]
}
EOF
)
    
    # Save config to temp file in container
    local temp_file="/tmp/lifecycle-${bucket_name}.json"
    if minio::docker::exec sh -c "echo '$lifecycle_config' > $temp_file"; then
        # Apply lifecycle policy
        if minio::api::mc ilm import "local/${bucket_name}" < "$temp_file" >/dev/null 2>&1; then
            minio::docker::exec rm -f "$temp_file"
            log::debug "Lifecycle policy set successfully"
            return 0
        else
            log::error "Failed to set lifecycle policy"
            minio::docker::exec rm -f "$temp_file"
            return 1
        fi
    else
        log::error "Failed to create lifecycle configuration"
        return 1
    fi
}

#######################################
# Generate presigned URL for object
# Arguments:
#   $1 - Bucket name
#   $2 - Object name
#   $3 - Expiry duration (optional, default: 7d)
# Output: Presigned URL
#######################################
minio::api::presign_url() {
    local bucket_name=$1
    local object_name=$2
    local expiry=${3:-"7d"}
    
    if [[ -z "$bucket_name" || -z "$object_name" ]]; then
        log::error "Bucket name and object name are required"
        return 1
    fi
    
    minio::api::mc share download --expire="$expiry" "local/${bucket_name}/${object_name}" 2>/dev/null
}

#######################################
# Test MinIO API with a simple operation
# Returns: 0 if API is working, 1 if not
#######################################
minio::api::test() {
    log::info "Testing MinIO API..."
    
    # Configure mc client first
    minio::api::configure_mc || return 1
    
    # Try to list buckets as a test
    if minio::api::mc ls local >/dev/null 2>&1; then
        log::debug "MinIO API test successful"
        return 0
    else
        log::error "MinIO API test failed"
        return 1
    fi
}

#######################################
# Get storage metrics for all buckets
# Output: JSON with storage statistics
# Returns: 0 on success, 1 on failure
#######################################
minio::api::get_metrics() {
    local format="${1:-text}"  # text or json
    
    # Configure mc client first
    minio::api::configure_mc || return 1
    
    if [[ "$format" == "json" ]]; then
        # Get bucket statistics in JSON format
        minio::api::mc du --json local 2>/dev/null
    else
        # Get human-readable statistics
        echo "MinIO Storage Metrics"
        echo "===================="
        echo ""
        
        # Get total disk usage
        local total_size=$(minio::api::mc du local 2>/dev/null | tail -1 | awk '{print $1}')
        echo "Total Storage Used: ${total_size:-0B}"
        echo ""
        
        # Get per-bucket statistics
        echo "Bucket Statistics:"
        echo "------------------"
        # List all buckets and get their stats
        minio::api::mc ls local 2>/dev/null | while read -r line; do
            # Extract bucket name from the listing (last field, remove trailing /)
            local bucket_name=$(echo "$line" | awk '{print $NF}' | sed 's|/$||')
            if [[ -n "$bucket_name" ]]; then
                local bucket_size=$(minio::api::mc du "local/${bucket_name}" 2>/dev/null | tail -1 | awk '{print $1}')
                local object_count=$(minio::api::mc ls --recursive "local/${bucket_name}" 2>/dev/null | wc -l)
                printf "%-30s %10s (%d objects)\n" "${bucket_name}:" "${bucket_size:-0B}" "${object_count}"
            fi
        done
        
        # Get server info for bandwidth if available
        echo ""
        echo "Server Information:"
        echo "------------------"
        local server_info=$(minio::api::mc admin info local 2>/dev/null | grep -E "Uptime:|Network:" | head -2)
        if [[ -n "$server_info" ]]; then
            echo "$server_info"
        else
            echo "Server info not available"
        fi
    fi
    
    return 0
}

#######################################
# Get storage metrics for a specific bucket
# Arguments:
#   $1 - Bucket name
# Output: Storage statistics for the bucket
# Returns: 0 on success, 1 on failure
#######################################
minio::api::get_bucket_metrics() {
    local bucket_name=$1
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name is required"
        return 1
    fi
    
    # Configure mc client first
    minio::api::configure_mc || return 1
    
    # Get bucket size
    local bucket_size=$(minio::api::mc du "local/${bucket_name}" 2>/dev/null | tail -1 | awk '{print $1}')
    local object_count=$(minio::api::mc ls --recursive "local/${bucket_name}" 2>/dev/null | wc -l)
    
    echo "Bucket: $bucket_name"
    echo "Size: ${bucket_size:-0B}"
    echo "Objects: ${object_count:-0}"
    
    # Get largest objects
    echo ""
    echo "Largest Objects (top 5):"
    minio::api::mc ls --recursive "local/${bucket_name}" 2>/dev/null | \
        sort -k4 -rn | head -5 | \
        awk '{printf "  %-40s %s\n", $NF, $4}'
    
    return 0
}

#######################################
# Upload large file with multi-part support
# Arguments:
#   $1 - File path
#   $2 - Bucket name
#   $3 - Object name (optional)
# Returns: 0 on success, 1 on failure
#######################################
minio::api::upload_large_file() {
    local file_path=$1
    local bucket_name=$2
    local object_name=${3:-$(basename "$file_path")}
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name is required"
        return 1
    fi
    
    # Get file size in bytes
    local file_size=$(stat -c%s "$file_path" 2>/dev/null || stat -f%z "$file_path" 2>/dev/null)
    local file_size_mb=$((file_size / 1024 / 1024))
    
    log::info "Uploading large file (${file_size_mb}MB): $file_path to $bucket_name/$object_name"
    
    # For files >100MB, use multi-part upload
    if [[ $file_size_mb -gt 100 ]]; then
        log::info "Using multi-part upload for large file"
        
        # Copy file to container first
        local temp_path="/tmp/upload-$(basename "$file_path")"
        if docker cp "$file_path" "${MINIO_CONTAINER_NAME}:${temp_path}" >/dev/null 2>&1; then
            # Upload with progress indicator using mc
            if minio::docker::exec mc cp --continue "$temp_path" "local/${bucket_name}/${object_name}"; then
                # Clean up temp file
                minio::docker::exec rm -f "$temp_path"
                log::success "Large file uploaded successfully"
                return 0
            else
                log::error "Failed to upload large file to MinIO"
                minio::docker::exec rm -f "$temp_path"
                return 1
            fi
        else
            log::error "Failed to copy large file to container"
            return 1
        fi
    else
        # For smaller files, use regular upload
        minio::api::upload_file "$file_path" "$bucket_name" "$object_name"
    fi
}

#######################################
# Resume incomplete upload
# Arguments:
#   $1 - File path
#   $2 - Bucket name
#   $3 - Object name
# Returns: 0 on success, 1 on failure
#######################################
minio::api::resume_upload() {
    local file_path=$1
    local bucket_name=$2
    local object_name=$3
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    log::info "Resuming upload for: $file_path"
    
    # Copy file to container
    local temp_path="/tmp/resume-$(basename "$file_path")"
    if docker cp "$file_path" "${MINIO_CONTAINER_NAME}:${temp_path}" >/dev/null 2>&1; then
        # Resume upload using mc --continue flag
        if minio::docker::exec mc cp --continue "$temp_path" "local/${bucket_name}/${object_name}"; then
            minio::docker::exec rm -f "$temp_path"
            log::success "Upload resumed and completed"
            return 0
        else
            log::error "Failed to resume upload"
            minio::docker::exec rm -f "$temp_path"
            return 1
        fi
    else
        log::error "Failed to copy file for resume"
        return 1
    fi
}