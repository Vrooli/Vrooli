#!/usr/bin/env bash
# MinIO Bucket Management
# Functions for managing MinIO buckets

# Source trash module for safe cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
MINIO_LIB_DIR="${APP_ROOT}/resources/minio/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

#######################################
# Initialize default buckets for Vrooli
# Returns: 0 on success, 1 on failure
#######################################
minio::buckets::init_defaults() {
    log::info "${MSG_CREATING_BUCKETS}"
    
    # Configure mc client first
    minio::api::configure_mc || return 1
    
    local failed=false
    
    # Create each default bucket
    for bucket in "${MINIO_DEFAULT_BUCKETS[@]}"; do
        if ! minio::api::create_bucket "$bucket"; then
            failed=true
            log::error "Failed to create bucket: $bucket"
        fi
    done
    
    if [[ "$failed" == "true" ]]; then
        return 1
    fi
    
    # Configure bucket policies
    log::info "${MSG_CONFIGURING_POLICIES}"
    
    # User uploads bucket - public read for serving files
    if ! minio::api::set_bucket_policy "vrooli-user-uploads" "download"; then
        log::warn "Failed to set policy for user uploads bucket"
    fi
    
    # Agent artifacts - private by default
    if ! minio::api::set_bucket_policy "vrooli-agent-artifacts" "none"; then
        log::warn "Failed to set policy for agent artifacts bucket"
    fi
    
    # Model cache - private
    if ! minio::api::set_bucket_policy "vrooli-model-cache" "none"; then
        log::warn "Failed to set policy for model cache bucket"
    fi
    
    # Temp storage - private with lifecycle
    if ! minio::api::set_bucket_policy "vrooli-temp-storage" "none"; then
        log::warn "Failed to set policy for temp storage bucket"
    fi
    
    # Set lifecycle for temp storage (24 hour expiry)
    if ! minio::api::set_lifecycle "vrooli-temp-storage" 1; then
        log::warn "Failed to set lifecycle policy for temp storage"
    fi
    
    log::success "${MSG_BUCKETS_INITIALIZED}"
    return 0
}

#######################################
# Create test file and upload to verify functionality
# Returns: 0 on success, 1 on failure
#######################################
minio::buckets::test_upload() {
    log::info "Testing file upload functionality..."
    
    # Create a test file
    local test_file="/tmp/minio-test-$(date +%s).txt"
    echo "MinIO test file created at $(date)" > "$test_file"
    
    # Upload to user uploads bucket
    if minio::api::upload_file "$test_file" "vrooli-user-uploads" "test/minio-test.txt"; then
        log::success "Test file uploaded successfully"
        
        # Try to download it back
        local download_file="/tmp/minio-test-download.txt"
        if minio::api::download_file "vrooli-user-uploads" "test/minio-test.txt" "$download_file"; then
            log::success "Test file downloaded successfully"
            
            # Verify content
            if diff "$test_file" "$download_file" >/dev/null 2>&1; then
                log::success "File content verified successfully"
            else
                log::error "Downloaded file content doesn't match"
            fi
            
            trash::safe_remove "$download_file" --temp
        else
            log::error "Failed to download test file"
        fi
        
        # Clean up test files
        trash::safe_remove "$test_file" --temp
        
        # Delete from MinIO
        minio::api::mc rm "local/vrooli-user-uploads/test/minio-test.txt" >/dev/null 2>&1
        
        return 0
    else
        log::error "Failed to upload test file"
        trash::safe_remove "$test_file" --temp
        return 1
    fi
}

#######################################
# Show bucket statistics
#######################################
minio::buckets::show_stats() {
    log::info "MinIO Bucket Statistics:"
    log::info "========================"
    
    # Configure mc client
    minio::api::configure_mc >/dev/null 2>&1
    
    # Get list of buckets
    local buckets=$(minio::api::list_buckets)
    
    if [[ -z "$buckets" ]]; then
        log::info "No buckets found"
        return
    fi
    
    # Show stats for each bucket
    while IFS= read -r bucket; do
        [[ -z "$bucket" ]] && continue
        
        local size=$(minio::api::get_bucket_size "$bucket" 2>/dev/null || echo "0B")
        local objects=$(minio::api::mc ls --recursive "local/${bucket}" 2>/dev/null | wc -l || echo "0")
        
        printf "%-30s Size: %-10s Objects: %s\n" "$bucket" "$size" "$objects"
    done <<< "$buckets"
}

#######################################
# Create custom bucket with optional policy
# Arguments:
#   $1 - Bucket name
#   $2 - Policy (optional: public, download, upload, none)
# Returns: 0 on success, 1 on failure
#######################################
minio::buckets::create_custom() {
    local bucket_name=$1
    local policy=${2:-"none"}
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name is required"
        log::info "Usage: $0 --action create-bucket --bucket <name> [--policy <type>]"
        return 1
    fi
    
    # Validate bucket name (must be lowercase, 3-63 chars, valid DNS name)
    if ! [[ "$bucket_name" =~ ^[a-z0-9][a-z0-9.-]*[a-z0-9]$ ]] || [[ ${#bucket_name} -lt 3 ]] || [[ ${#bucket_name} -gt 63 ]]; then
        log::error "Invalid bucket name. Must be 3-63 lowercase letters, numbers, hyphens, or dots"
        return 1
    fi
    
    # Configure mc client
    minio::api::configure_mc || return 1
    
    # Create the bucket
    if minio::api::create_bucket "$bucket_name"; then
        # Set policy if specified
        if [[ "$policy" != "none" ]]; then
            if minio::api::set_bucket_policy "$bucket_name" "$policy"; then
                log::info "Bucket created with $policy policy"
            else
                log::warn "Bucket created but failed to set policy"
            fi
        else
            log::info "Bucket created with private access"
        fi
        return 0
    else
        return 1
    fi
}

#######################################
# Remove a bucket
# Arguments:
#   $1 - Bucket name
#   $2 - Force removal (optional, default: false)
# Returns: 0 on success, 1 on failure
#######################################
minio::buckets::remove() {
    local bucket_name=$1
    local force=${2:-false}
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name is required"
        return 1
    fi
    
    # Protect default buckets unless forced
    if [[ "$force" != "true" ]]; then
        for default_bucket in "${MINIO_DEFAULT_BUCKETS[@]}"; do
            if [[ "$bucket_name" == "$default_bucket" ]]; then
                log::error "Cannot remove default bucket: $bucket_name"
                log::info "Use --force flag to override this protection"
                return 1
            fi
        done
    fi
    
    # Configure mc client
    minio::api::configure_mc || return 1
    
    log::info "Removing bucket: $bucket_name"
    
    # Remove bucket (--force removes non-empty buckets)
    if [[ "$force" == "true" ]]; then
        if minio::api::mc rb --force "local/${bucket_name}" >/dev/null 2>&1; then
            log::success "Bucket removed successfully"
            return 0
        else
            log::error "Failed to remove bucket"
            return 1
        fi
    else
        if minio::api::mc rb "local/${bucket_name}" >/dev/null 2>&1; then
            log::success "Bucket removed successfully"
            return 0
        else
            log::error "Failed to remove bucket (bucket may not be empty)"
            log::info "Use --force flag to remove non-empty buckets"
            return 1
        fi
    fi
}