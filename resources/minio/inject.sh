#!/usr/bin/env bash
set -euo pipefail

# MinIO Data Injection Adapter
# This script handles injection of buckets and files into MinIO S3-compatible storage
# Part of the Vrooli resource data injection system

export DESCRIPTION="Inject buckets and files into MinIO S3-compatible object storage"

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/minio"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh"

# Source common utilities
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"

# Source MinIO configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default MinIO settings
readonly DEFAULT_MINIO_ENDPOINT="http://localhost:9000"
readonly DEFAULT_MINIO_ACCESS_KEY="minioadmin"
readonly DEFAULT_MINIO_SECRET_KEY="minioadmin"

# MinIO settings (can be overridden by environment)
MINIO_ENDPOINT="${MINIO_ENDPOINT:-$DEFAULT_MINIO_ENDPOINT}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-$DEFAULT_MINIO_ACCESS_KEY}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-$DEFAULT_MINIO_SECRET_KEY}"

# Operation tracking
declare -a MINIO_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
minio_inject::usage() {
    cat << EOF
MinIO Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects buckets and files into MinIO based on scenario configuration.
    Supports validation, injection, status checks, and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the data injection
    --status      Check status of injected data
    --rollback    Rollback injected data
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "buckets": [
        {
          "name": "bucket-name",
          "policy": "public-read",
          "versioning": false
        }
      ],
      "data": [
        {
          "type": "file",
          "bucket": "bucket-name",
          "file": "path/to/file.txt",
          "key": "optional/key/name.txt"
        },
        {
          "type": "directory",
          "bucket": "bucket-name",
          "source": "path/to/directory/",
          "prefix": "optional/prefix/",
          "recursive": true
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"buckets": [{"name": "test-bucket", "policy": "private"}]}'
    
    # Inject buckets and data
    $0 --inject '{"buckets": [{"name": "images"}], "data": [{"type": "file", "bucket": "images", "file": "logo.png"}]}'

EOF
}

#######################################
# Check if MinIO is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
minio_inject::check_accessibility() {
    # Check if mc (MinIO client) is available
    if ! system::is_command "mc"; then
        log::warn "MinIO client (mc) not available, checking via curl"
        
        # Try basic HTTP check
        if curl -s --max-time 5 "$MINIO_ENDPOINT/minio/health/live" >/dev/null 2>&1; then
            log::debug "MinIO is accessible at $MINIO_ENDPOINT"
            return 0
        else
            log::error "MinIO is not accessible at $MINIO_ENDPOINT"
            log::info "Install mc: wget https://dl.min.io/client/mc/release/linux-amd64/mc && chmod +x mc"
            return 1
        fi
    fi
    
    # Configure mc alias if not already configured
    if ! mc alias list 2>/dev/null | grep -q "local"; then
        log::info "Configuring MinIO client alias..."
        mc alias set local "$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" >/dev/null 2>&1
    fi
    
    # Test connection
    if mc admin info local >/dev/null 2>&1; then
        log::debug "MinIO is accessible via mc client"
        return 0
    else
        log::error "MinIO is not accessible at $MINIO_ENDPOINT"
        log::info "Ensure MinIO is running: ./scripts/resources/storage/minio/manage.sh --action start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
minio_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    MINIO_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added MinIO rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
minio_inject::execute_rollback() {
    if [[ ${#MINIO_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No MinIO rollback actions to execute"
        return 0
    fi
    
    log::info "Executing MinIO rollback actions..."
    
    local success_count=0
    local total_count=${#MINIO_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#MINIO_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${MINIO_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "MinIO rollback completed: $success_count/$total_count actions successful"
    MINIO_ROLLBACK_ACTIONS=()
}

#######################################
# Validate bucket configuration
# Arguments:
#   $1 - buckets configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
minio_inject::validate_buckets() {
    local buckets_config="$1"
    
    log::debug "Validating bucket configurations..."
    
    # Check if buckets is an array
    local buckets_type
    buckets_type=$(echo "$buckets_config" | jq -r 'type')
    
    if [[ "$buckets_type" != "array" ]]; then
        log::error "Buckets configuration must be an array, got: $buckets_type"
        return 1
    fi
    
    # Validate each bucket
    local bucket_count
    bucket_count=$(echo "$buckets_config" | jq 'length')
    
    for ((i=0; i<bucket_count; i++)); do
        local bucket
        bucket=$(echo "$buckets_config" | jq -c ".[$i]")
        
        # Check required fields
        local name
        name=$(echo "$bucket" | jq -r '.name // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Bucket at index $i missing required 'name' field"
            return 1
        fi
        
        # Validate bucket name (S3 naming rules)
        if [[ ! "$name" =~ ^[a-z0-9][a-z0-9.-]*[a-z0-9]$ ]] || [[ ${#name} -lt 3 ]] || [[ ${#name} -gt 63 ]]; then
            log::error "Invalid bucket name: $name (must be 3-63 chars, lowercase, start/end with letter/number)"
            return 1
        fi
        
        # Validate policy if specified
        local policy
        policy=$(echo "$bucket" | jq -r '.policy // "private"')
        
        case "$policy" in
            private|public-read|public-read-write)
                log::debug "Bucket '$name' has valid policy: $policy"
                ;;
            *)
                log::error "Bucket '$name' has invalid policy: $policy"
                return 1
                ;;
        esac
        
        log::debug "Bucket '$name' configuration is valid"
    done
    
    log::success "All bucket configurations are valid"
    return 0
}

#######################################
# Validate data configuration
# Arguments:
#   $1 - data configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
minio_inject::validate_data() {
    local data_config="$1"
    
    log::debug "Validating data configurations..."
    
    # Check if data is an array
    local data_type
    data_type=$(echo "$data_config" | jq -r 'type')
    
    if [[ "$data_type" != "array" ]]; then
        log::error "Data configuration must be an array, got: $data_type"
        return 1
    fi
    
    # Validate each data item
    local data_count
    data_count=$(echo "$data_config" | jq 'length')
    
    for ((i=0; i<data_count; i++)); do
        local data_item
        data_item=$(echo "$data_config" | jq -c ".[$i]")
        
        # Check required fields
        local type bucket
        type=$(echo "$data_item" | jq -r '.type // empty')
        bucket=$(echo "$data_item" | jq -r '.bucket // empty')
        
        if [[ -z "$type" ]]; then
            log::error "Data item at index $i missing required 'type' field"
            return 1
        fi
        
        if [[ -z "$bucket" ]]; then
            log::error "Data item at index $i missing required 'bucket' field"
            return 1
        fi
        
        # Validate type and type-specific fields
        case "$type" in
            file)
                local file
                file=$(echo "$data_item" | jq -r '.file // empty')
                
                if [[ -z "$file" ]]; then
                    log::error "File data item missing required 'file' field"
                    return 1
                fi
                
                # Check if file exists
                local source_file="$VROOLI_PROJECT_ROOT/$file"
                if [[ ! -f "$source_file" ]]; then
                    log::error "Source file not found: $source_file"
                    return 1
                fi
                ;;
            directory)
                local source
                source=$(echo "$data_item" | jq -r '.source // empty')
                
                if [[ -z "$source" ]]; then
                    log::error "Directory data item missing required 'source' field"
                    return 1
                fi
                
                # Check if directory exists
                local source_dir="$VROOLI_PROJECT_ROOT/$source"
                if [[ ! -d "$source_dir" ]]; then
                    log::error "Source directory not found: $source_dir"
                    return 1
                fi
                ;;
            *)
                log::error "Data item has invalid type: $type"
                return 1
                ;;
        esac
        
        log::debug "Data item configuration is valid"
    done
    
    log::success "All data configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
minio_inject::validate_config() {
    local config="$1"
    
    log::info "Validating MinIO injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in MinIO injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_buckets has_data
    has_buckets=$(echo "$config" | jq -e '.buckets' >/dev/null 2>&1 && echo "true" || echo "false")
    has_data=$(echo "$config" | jq -e '.data' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_buckets" == "false" && "$has_data" == "false" ]]; then
        log::error "MinIO injection configuration must have 'buckets' or 'data'"
        return 1
    fi
    
    # Validate buckets if present
    if [[ "$has_buckets" == "true" ]]; then
        local buckets
        buckets=$(echo "$config" | jq -c '.buckets')
        
        if ! minio_inject::validate_buckets "$buckets"; then
            return 1
        fi
    fi
    
    # Validate data if present
    if [[ "$has_data" == "true" ]]; then
        local data
        data=$(echo "$config" | jq -c '.data')
        
        if ! minio_inject::validate_data "$data"; then
            return 1
        fi
    fi
    
    log::success "MinIO injection configuration is valid"
    return 0
}

#######################################
# Create bucket
# Arguments:
#   $1 - bucket configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
minio_inject::create_bucket() {
    local bucket_config="$1"
    
    local name policy versioning
    name=$(echo "$bucket_config" | jq -r '.name')
    policy=$(echo "$bucket_config" | jq -r '.policy // "private"')
    versioning=$(echo "$bucket_config" | jq -r '.versioning // false')
    
    log::info "Creating bucket: $name"
    
    # Create bucket
    if mc mb "local/$name" 2>/dev/null; then
        log::success "Created bucket: $name"
        
        # Add rollback action
        minio_inject::add_rollback_action \
            "Remove bucket: $name" \
            "mc rb --force 'local/$name' >/dev/null 2>&1"
    else
        # Bucket might already exist
        if mc ls "local/$name" >/dev/null 2>&1; then
            log::warn "Bucket '$name' already exists"
        else
            log::error "Failed to create bucket: $name"
            return 1
        fi
    fi
    
    # Set bucket policy
    case "$policy" in
        public-read)
            log::info "Setting public-read policy for bucket: $name"
            mc anonymous set download "local/$name" >/dev/null 2>&1
            ;;
        public-read-write)
            log::info "Setting public-read-write policy for bucket: $name"
            mc anonymous set public "local/$name" >/dev/null 2>&1
            ;;
        private)
            # Default is private, no action needed
            ;;
    esac
    
    # Enable versioning if requested
    if [[ "$versioning" == "true" ]]; then
        log::info "Enabling versioning for bucket: $name"
        mc version enable "local/$name" >/dev/null 2>&1
    fi
    
    return 0
}

#######################################
# Upload file to bucket
# Arguments:
#   $1 - data configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
minio_inject::upload_file() {
    local data_config="$1"
    
    local bucket file key
    bucket=$(echo "$data_config" | jq -r '.bucket')
    file=$(echo "$data_config" | jq -r '.file')
    key=$(echo "$data_config" | jq -r '.key // empty')
    
    # Resolve file path
    local source_file="$VROOLI_PROJECT_ROOT/$file"
    
    # Use original filename as key if not specified
    if [[ -z "$key" ]]; then
        key=$(basename "$file")
    fi
    
    log::info "Uploading file to $bucket: $key"
    
    # Upload file
    if mc cp "$source_file" "local/$bucket/$key" >/dev/null 2>&1; then
        log::success "Uploaded file: $key to bucket $bucket"
        
        # Add rollback action
        minio_inject::add_rollback_action \
            "Remove file: $key from bucket $bucket" \
            "mc rm 'local/$bucket/$key' >/dev/null 2>&1"
        
        return 0
    else
        log::error "Failed to upload file: $key to bucket $bucket"
        return 1
    fi
}

#######################################
# Upload directory to bucket
# Arguments:
#   $1 - data configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
minio_inject::upload_directory() {
    local data_config="$1"
    
    local bucket source prefix recursive
    bucket=$(echo "$data_config" | jq -r '.bucket')
    source=$(echo "$data_config" | jq -r '.source')
    prefix=$(echo "$data_config" | jq -r '.prefix // empty')
    recursive=$(echo "$data_config" | jq -r '.recursive // true')
    
    # Resolve directory path
    local source_dir="$VROOLI_PROJECT_ROOT/$source"
    
    log::info "Uploading directory to $bucket: $source"
    
    # Build mc command
    local mc_cmd="mc cp"
    if [[ "$recursive" == "true" ]]; then
        mc_cmd="$mc_cmd --recursive"
    fi
    
    # Upload directory
    if [[ -n "$prefix" ]]; then
        # Upload with prefix
        if $mc_cmd "$source_dir" "local/$bucket/$prefix" >/dev/null 2>&1; then
            log::success "Uploaded directory: $source to bucket $bucket/$prefix"
            
            # Add rollback action
            minio_inject::add_rollback_action \
                "Remove directory: $prefix from bucket $bucket" \
                "mc rm --recursive --force 'local/$bucket/$prefix' >/dev/null 2>&1"
            
            return 0
        else
            log::error "Failed to upload directory: $source to bucket $bucket/$prefix"
            return 1
        fi
    else
        # Upload without prefix
        if $mc_cmd "$source_dir" "local/$bucket/" >/dev/null 2>&1; then
            log::success "Uploaded directory: $source to bucket $bucket"
            
            # Note: Rollback for directory without prefix is more complex
            # Would need to track individual files
            
            return 0
        else
            log::error "Failed to upload directory: $source to bucket $bucket"
            return 1
        fi
    fi
}

#######################################
# Inject buckets
# Arguments:
#   $1 - buckets configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
minio_inject::inject_buckets() {
    local buckets_config="$1"
    
    log::info "Creating MinIO buckets..."
    
    local bucket_count
    bucket_count=$(echo "$buckets_config" | jq 'length')
    
    if [[ "$bucket_count" -eq 0 ]]; then
        log::info "No buckets to create"
        return 0
    fi
    
    local failed_buckets=()
    
    for ((i=0; i<bucket_count; i++)); do
        local bucket
        bucket=$(echo "$buckets_config" | jq -c ".[$i]")
        
        local bucket_name
        bucket_name=$(echo "$bucket" | jq -r '.name')
        
        if ! minio_inject::create_bucket "$bucket"; then
            failed_buckets+=("$bucket_name")
        fi
    done
    
    if [[ ${#failed_buckets[@]} -eq 0 ]]; then
        log::success "All buckets created successfully"
        return 0
    else
        log::error "Failed to create buckets: ${failed_buckets[*]}"
        return 1
    fi
}

#######################################
# Inject data
# Arguments:
#   $1 - data configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
minio_inject::inject_data_items() {
    local data_config="$1"
    
    log::info "Uploading data to MinIO..."
    
    local data_count
    data_count=$(echo "$data_config" | jq 'length')
    
    if [[ "$data_count" -eq 0 ]]; then
        log::info "No data to upload"
        return 0
    fi
    
    local failed_items=()
    
    for ((i=0; i<data_count; i++)); do
        local data_item
        data_item=$(echo "$data_config" | jq -c ".[$i]")
        
        local type
        type=$(echo "$data_item" | jq -r '.type')
        
        case "$type" in
            file)
                if ! minio_inject::upload_file "$data_item"; then
                    failed_items+=("file:$(echo "$data_item" | jq -r '.file')")
                fi
                ;;
            directory)
                if ! minio_inject::upload_directory "$data_item"; then
                    failed_items+=("dir:$(echo "$data_item" | jq -r '.source')")
                fi
                ;;
        esac
    done
    
    if [[ ${#failed_items[@]} -eq 0 ]]; then
        log::success "All data uploaded successfully"
        return 0
    else
        log::error "Failed to upload data: ${failed_items[*]}"
        return 1
    fi
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
minio_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into MinIO"
    
    # Check MinIO accessibility
    if ! minio_inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    MINIO_ROLLBACK_ACTIONS=()
    
    # Inject buckets if present
    local has_buckets
    has_buckets=$(echo "$config" | jq -e '.buckets' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_buckets" == "true" ]]; then
        local buckets
        buckets=$(echo "$config" | jq -c '.buckets')
        
        if ! minio_inject::inject_buckets "$buckets"; then
            log::error "Failed to create buckets"
            minio_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject data if present
    local has_data
    has_data=$(echo "$config" | jq -e '.data' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_data" == "true" ]]; then
        local data
        data=$(echo "$config" | jq -c '.data')
        
        if ! minio_inject::inject_data_items "$data"; then
            log::error "Failed to upload data"
            minio_inject::execute_rollback
            return 1
        fi
    fi
    
    log::success "✅ MinIO data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
minio_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking MinIO injection status"
    
    # Check MinIO accessibility
    if ! minio_inject::check_accessibility; then
        return 1
    fi
    
    # Check bucket status
    local has_buckets
    has_buckets=$(echo "$config" | jq -e '.buckets' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_buckets" == "true" ]]; then
        local buckets
        buckets=$(echo "$config" | jq -c '.buckets')
        
        log::info "Checking bucket status..."
        
        local bucket_count
        bucket_count=$(echo "$buckets" | jq 'length')
        
        for ((i=0; i<bucket_count; i++)); do
            local bucket
            bucket=$(echo "$buckets" | jq -c ".[$i]")
            
            local name
            name=$(echo "$bucket" | jq -r '.name')
            
            # Check if bucket exists
            if mc ls "local/$name" >/dev/null 2>&1; then
                log::success "✅ Bucket '$name' exists"
            else
                log::error "❌ Bucket '$name' not found"
            fi
        done
    fi
    
    # Check data status
    local has_data
    has_data=$(echo "$config" | jq -e '.data' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_data" == "true" ]]; then
        log::info "Checking data status..."
        
        local data
        data=$(echo "$config" | jq -c '.data')
        
        local data_count
        data_count=$(echo "$data" | jq 'length')
        
        for ((i=0; i<data_count; i++)); do
            local data_item
            data_item=$(echo "$data" | jq -c ".[$i]")
            
            local type bucket
            type=$(echo "$data_item" | jq -r '.type')
            bucket=$(echo "$data_item" | jq -r '.bucket')
            
            case "$type" in
                file)
                    local key
                    key=$(echo "$data_item" | jq -r '.key // empty')
                    
                    if [[ -z "$key" ]]; then
                        local file
                        file=$(echo "$data_item" | jq -r '.file')
                        key=$(basename "$file")
                    fi
                    
                    if mc ls "local/$bucket/$key" >/dev/null 2>&1; then
                        log::success "✅ File '$key' exists in bucket '$bucket'"
                    else
                        log::error "❌ File '$key' not found in bucket '$bucket'"
                    fi
                    ;;
                directory)
                    local prefix
                    prefix=$(echo "$data_item" | jq -r '.prefix // empty')
                    
                    if [[ -n "$prefix" ]]; then
                        if mc ls "local/$bucket/$prefix" >/dev/null 2>&1; then
                            log::success "✅ Directory '$prefix' exists in bucket '$bucket'"
                        else
                            log::error "❌ Directory '$prefix' not found in bucket '$bucket'"
                        fi
                    else
                        log::info "Directory upload verification requires specific file checks"
                    fi
                    ;;
            esac
        done
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
minio_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        minio_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            minio_inject::validate_config "$config"
            ;;
        "--inject")
            minio_inject::inject_data "$config"
            ;;
        "--status")
            minio_inject::check_status "$config"
            ;;
        "--rollback")
            minio_inject::execute_rollback
            ;;
        "--help")
            minio_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            minio_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        minio_inject::usage
        exit 1
    fi
    
    minio_inject::main "$@"
fi