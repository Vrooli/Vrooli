#!/usr/bin/env bash
# MinIO Injection Functions
# Functions for injecting data, buckets, and configurations into MinIO

# Get script directory
MINIO_INJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${MINIO_INJECT_DIR}/common.sh"
source "${MINIO_INJECT_DIR}/api.sh"
source "${MINIO_INJECT_DIR}/buckets.sh"

#######################################
# Main injection entry point
# Handles various injection types based on file content
# Arguments:
#   $1 - Path to injection file (JSON)
# Returns: 0 on success, 1 on failure
#######################################
minio::inject() {
    local file="${1:-}"
    
    # Check if file is provided via environment variable (for CLI integration)
    if [[ -z "$file" && -n "${INJECTION_CONFIG:-}" ]]; then
        # Process inline config from environment
        echo "$INJECTION_CONFIG" | minio::inject::process_config
        return $?
    fi
    
    if [[ -z "$file" ]]; then
        log::error "Injection file path required"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "Injection file not found: $file"
        return 1
    fi
    
    # Process the injection file
    minio::inject::process_file "$file"
}

#######################################
# Process injection file
# Determines injection type and executes appropriate action
# Arguments:
#   $1 - Path to injection file
# Returns: 0 on success, 1 on failure
#######################################
minio::inject::process_file() {
    local file="$1"
    
    log::info "Processing MinIO injection file: $file"
    
    # Validate JSON
    if ! jq empty "$file" 2>/dev/null; then
        log::error "Invalid JSON in injection file"
        return 1
    fi
    
    # Determine injection type
    local type
    type=$(jq -r '.type // "unknown"' "$file")
    
    case "$type" in
        buckets)
            minio::inject::buckets "$file"
            ;;
        objects)
            minio::inject::objects "$file"
            ;;
        policy)
            minio::inject::policy "$file"
            ;;
        users)
            minio::inject::users "$file"
            ;;
        config)
            minio::inject::config "$file"
            ;;
        *)
            # Try to auto-detect based on content
            if jq -e '.buckets' "$file" >/dev/null 2>&1; then
                minio::inject::buckets "$file"
            elif jq -e '.objects' "$file" >/dev/null 2>&1; then
                minio::inject::objects "$file"
            else
                log::error "Unknown injection type: $type"
                log::info "Supported types: buckets, objects, policy, users, config"
                return 1
            fi
            ;;
    esac
}

#######################################
# Process inline configuration
# Used when config is passed via environment variable
# Returns: 0 on success, 1 on failure
#######################################
minio::inject::process_config() {
    local temp_file
    temp_file=$(mktemp /tmp/minio-inject-XXXXXX.json)
    
    # Write stdin to temp file
    cat > "$temp_file"
    
    # Process the temp file
    minio::inject::process_file "$temp_file"
    local result=$?
    
    # Clean up
    rm -f "$temp_file"
    
    return $result
}

#######################################
# Inject buckets into MinIO
# Creates buckets with optional policies and versioning
# Arguments:
#   $1 - Path to buckets configuration file
# Returns: 0 on success, 1 on failure
#######################################
minio::inject::buckets() {
    local file="$1"
    
    log::info "Injecting MinIO buckets..."
    
    # Ensure MinIO is running
    if ! minio::common::is_running; then
        log::error "MinIO is not running"
        return 1
    fi
    
    # Process each bucket
    local bucket_count
    bucket_count=$(jq '.buckets | length' "$file")
    
    if [[ "$bucket_count" -eq 0 ]]; then
        log::warn "No buckets to inject"
        return 0
    fi
    
    local success_count=0
    local i=0
    
    while [[ $i -lt $bucket_count ]]; do
        local bucket_name
        local bucket_policy
        local versioning
        
        bucket_name=$(jq -r ".buckets[$i].name" "$file")
        bucket_policy=$(jq -r ".buckets[$i].policy // \"none\"" "$file")
        versioning=$(jq -r ".buckets[$i].versioning // false" "$file")
        
        log::info "Creating bucket: $bucket_name"
        
        # Create bucket
        if minio::api::create_bucket "$bucket_name"; then
            ((success_count++))
            
            # Set policy if specified
            if [[ "$bucket_policy" != "none" ]]; then
                minio::inject::set_bucket_policy "$bucket_name" "$bucket_policy"
            fi
            
            # Enable versioning if specified
            if [[ "$versioning" == "true" ]]; then
                minio::inject::enable_versioning "$bucket_name"
            fi
        else
            log::warn "Failed to create bucket: $bucket_name"
        fi
        
        ((i++))
    done
    
    log::success "Successfully created $success_count/$bucket_count buckets"
    
    [[ $success_count -gt 0 ]] && return 0 || return 1
}

#######################################
# Inject objects into MinIO buckets
# Uploads files or creates objects from content
# Arguments:
#   $1 - Path to objects configuration file
# Returns: 0 on success, 1 on failure
#######################################
minio::inject::objects() {
    local file="$1"
    
    log::info "Injecting MinIO objects..."
    
    # Ensure MinIO is running
    if ! minio::common::is_running; then
        log::error "MinIO is not running"
        return 1
    fi
    
    # Process each object
    local object_count
    object_count=$(jq '.objects | length' "$file")
    
    if [[ "$object_count" -eq 0 ]]; then
        log::warn "No objects to inject"
        return 0
    fi
    
    local success_count=0
    local i=0
    
    while [[ $i -lt $object_count ]]; do
        local bucket
        local key
        local source_file
        local content
        
        bucket=$(jq -r ".objects[$i].bucket" "$file")
        key=$(jq -r ".objects[$i].key" "$file")
        source_file=$(jq -r ".objects[$i].file // empty" "$file")
        content=$(jq -r ".objects[$i].content // empty" "$file")
        
        # Ensure bucket exists
        if ! minio::inject::ensure_bucket "$bucket"; then
            log::warn "Skipping object $key - bucket $bucket doesn't exist"
            ((i++))
            continue
        fi
        
        # Upload object
        if [[ -n "$source_file" && -f "$source_file" ]]; then
            log::info "Uploading file to $bucket/$key"
            if minio::inject::upload_file "$bucket" "$key" "$source_file"; then
                ((success_count++))
            fi
        elif [[ -n "$content" ]]; then
            log::info "Creating object $bucket/$key from content"
            if echo "$content" | minio::inject::upload_content "$bucket" "$key"; then
                ((success_count++))
            fi
        else
            log::warn "No source for object $key"
        fi
        
        ((i++))
    done
    
    log::success "Successfully uploaded $success_count/$object_count objects"
    
    [[ $success_count -gt 0 ]] && return 0 || return 1
}

#######################################
# Helper function to ensure bucket exists
# Arguments:
#   $1 - Bucket name
# Returns: 0 if exists or created, 1 on failure
#######################################
minio::inject::ensure_bucket() {
    local bucket="$1"
    
    # Check if bucket exists using mc ls
    if minio::api::mc ls "minio-local/$bucket" >/dev/null 2>&1; then
        return 0
    fi
    
    # Create bucket
    log::info "Creating bucket: $bucket"
    minio::api::create_bucket "$bucket"
}

#######################################
# Set bucket policy
# Arguments:
#   $1 - Bucket name
#   $2 - Policy type (public-read, public-read-write, private)
# Returns: 0 on success, 1 on failure
#######################################
minio::inject::set_bucket_policy() {
    local bucket="$1"
    local policy="$2"
    
    log::info "Setting policy '$policy' for bucket '$bucket'"
    
    # Use mc client to set policy
    local mc_alias="vrooli-minio"
    
    case "$policy" in
        public-read)
            docker exec "${MINIO_CONTAINER_NAME}" \
                mc anonymous set download "${mc_alias}/${bucket}" 2>/dev/null
            ;;
        public-read-write)
            docker exec "${MINIO_CONTAINER_NAME}" \
                mc anonymous set public "${mc_alias}/${bucket}" 2>/dev/null
            ;;
        private)
            docker exec "${MINIO_CONTAINER_NAME}" \
                mc anonymous set private "${mc_alias}/${bucket}" 2>/dev/null
            ;;
        *)
            log::warn "Unknown policy type: $policy"
            return 1
            ;;
    esac
}

#######################################
# Enable versioning for bucket
# Arguments:
#   $1 - Bucket name
# Returns: 0 on success, 1 on failure
#######################################
minio::inject::enable_versioning() {
    local bucket="$1"
    
    log::info "Enabling versioning for bucket '$bucket'"
    
    docker exec "${MINIO_CONTAINER_NAME}" \
        mc version enable "vrooli-minio/${bucket}" 2>/dev/null
}

#######################################
# Upload file to MinIO
# Arguments:
#   $1 - Bucket name
#   $2 - Object key
#   $3 - Source file path
# Returns: 0 on success, 1 on failure
#######################################
minio::inject::upload_file() {
    local bucket="$1"
    local key="$2"
    local file="$3"
    
    # Copy file to container and upload
    docker cp "$file" "${MINIO_CONTAINER_NAME}:/tmp/upload-file"
    docker exec "${MINIO_CONTAINER_NAME}" \
        mc cp "/tmp/upload-file" "vrooli-minio/${bucket}/${key}" 2>/dev/null
    local result=$?
    
    # Clean up
    docker exec "${MINIO_CONTAINER_NAME}" rm -f "/tmp/upload-file"
    
    return $result
}

#######################################
# Upload content to MinIO (from stdin)
# Arguments:
#   $1 - Bucket name
#   $2 - Object key
# Returns: 0 on success, 1 on failure
#######################################
minio::inject::upload_content() {
    local bucket="$1"
    local key="$2"
    
    # Create temp file with content
    local temp_file="/tmp/minio-content-$$"
    cat > "$temp_file"
    
    # Upload via mc
    docker cp "$temp_file" "${MINIO_CONTAINER_NAME}:/tmp/upload-content"
    docker exec "${MINIO_CONTAINER_NAME}" \
        mc cp "/tmp/upload-content" "vrooli-minio/${bucket}/${key}" 2>/dev/null
    local result=$?
    
    # Clean up
    rm -f "$temp_file"
    docker exec "${MINIO_CONTAINER_NAME}" rm -f "/tmp/upload-content"
    
    return $result
}

# Export functions
export -f minio::inject
export -f minio::inject::process_file
export -f minio::inject::process_config
export -f minio::inject::buckets
export -f minio::inject::objects
export -f minio::inject::ensure_bucket
export -f minio::inject::set_bucket_policy
export -f minio::inject::enable_versioning
export -f minio::inject::upload_file
export -f minio::inject::upload_content