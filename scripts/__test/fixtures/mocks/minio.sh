#!/usr/bin/env bash
# MinIO Mock Implementation
# 
# Provides a comprehensive mock for MinIO S3-compatible object storage including:
# - MinIO client (mc) command interception
# - Docker container state simulation
# - S3 API operations (buckets, objects, policies)
# - File upload/download simulation
# - Health and admin endpoints
# - Bucket lifecycle and policy management
#
# This mock follows the same standards as other mocks with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions

# Prevent duplicate loading
[[ -n "${MINIO_MOCK_LOADED:-}" ]] && return 0
declare -g MINIO_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"
[[ -f "$MOCK_DIR/docker.sh" ]] && source "$MOCK_DIR/docker.sh"

# Global configuration
declare -g MINIO_MOCK_STATE_DIR="${MINIO_MOCK_STATE_DIR:-/tmp/minio-mock-state}"
declare -g MINIO_MOCK_DEBUG="${MINIO_MOCK_DEBUG:-}"

# Global state arrays
declare -gA MINIO_MOCK_BUCKETS=()           # Bucket storage
declare -gA MINIO_MOCK_OBJECTS=()           # Object storage (bucket/key -> data)
declare -gA MINIO_MOCK_OBJECT_METADATA=()   # Object metadata (bucket/key -> json)
declare -gA MINIO_MOCK_BUCKET_POLICIES=()   # Bucket policies
declare -gA MINIO_MOCK_ALIASES=()           # MC client aliases
declare -gA MINIO_MOCK_CONFIG=(             # MinIO configuration
    [container_name]="vrooli_minio"
    [port]="9000"
    [console_port]="9001" 
    [root_user]="minioadmin"
    [root_password]="minioadmin"
    [state]="running"
    [health_status]="healthy"
    [data_dir]="/tmp/minio-data"
    [config_dir]="/tmp/minio-config"
    [version]="RELEASE.2024-01-01T00-00-00Z"
    [error_mode]=""
)

# Temp file storage for simulating file operations
declare -g MINIO_MOCK_FILES_DIR="$MINIO_MOCK_STATE_DIR/files"

# Initialize state directory
mkdir -p "$MINIO_MOCK_STATE_DIR"
mkdir -p "$MINIO_MOCK_FILES_DIR"

# State persistence functions
mock::minio::save_state() {
    local state_file="$MINIO_MOCK_STATE_DIR/minio-state.sh"
    {
        echo "# MinIO mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p MINIO_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MINIO_MOCK_CONFIG=()"
        declare -p MINIO_MOCK_BUCKETS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MINIO_MOCK_BUCKETS=()"
        declare -p MINIO_MOCK_OBJECTS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MINIO_MOCK_OBJECTS=()"
        declare -p MINIO_MOCK_OBJECT_METADATA 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MINIO_MOCK_OBJECT_METADATA=()"
        declare -p MINIO_MOCK_BUCKET_POLICIES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MINIO_MOCK_BUCKET_POLICIES=()"
        declare -p MINIO_MOCK_ALIASES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MINIO_MOCK_ALIASES=()"
        
        # Save temp directory path
        echo "MINIO_MOCK_FILES_DIR=\"$MINIO_MOCK_FILES_DIR\""
    } > "$state_file"
    
    mock::log_state "minio" "Saved MinIO state to $state_file"
}

mock::minio::load_state() {
    local state_file="$MINIO_MOCK_STATE_DIR/minio-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        mock::log_state "minio" "Loaded MinIO state from $state_file"
    fi
}

# Automatically load state when sourced
mock::minio::load_state

# Main mc (MinIO Client) interceptor
mc() {
    mock::log_and_verify "mc" "$@"
    
    # Check if MinIO container is running
    if [[ "${MINIO_MOCK_CONFIG[state]}" != "running" ]]; then
        echo "mc: Unable to connect to MinIO server" >&2
        return 1
    fi
    
    # Check for error injection
    if [[ -n "${MINIO_MOCK_CONFIG[error_mode]}" ]]; then
        case "${MINIO_MOCK_CONFIG[error_mode]}" in
            "connection_failed")
                echo "mc: Connection refused" >&2
                return 1
                ;;
            "auth_failed")
                echo "mc: Access Denied. Check your credentials" >&2
                return 1
                ;;
            "service_unavailable")
                echo "mc: Service Unavailable" >&2
                return 1
                ;;
        esac
    fi
    
    if [[ $# -eq 0 ]]; then
        echo "mc: No command specified" >&2
        return 1
    fi
    
    # Always reload state at the beginning to handle BATS subshells
    mock::minio::load_state
    
    # Execute the MinIO command
    mock::minio::execute_command "$@"
    local result=$?
    
    # Save state after each command
    mock::minio::save_state
    
    return $result
}

# Execute a MinIO client command
mock::minio::execute_command() {
    local cmd="$1"
    shift
    
    case "$cmd" in
        config)
            mock::minio::cmd_config "$@"
            ;;
        ls)
            mock::minio::cmd_ls "$@"
            ;;
        mb)
            mock::minio::cmd_mb "$@"
            ;;
        rb)
            mock::minio::cmd_rb "$@"
            ;;
        cp)
            mock::minio::cmd_cp "$@"
            ;;
        rm)
            mock::minio::cmd_rm "$@"
            ;;
        du)
            mock::minio::cmd_du "$@"
            ;;
        anonymous)
            mock::minio::cmd_anonymous "$@"
            ;;
        stat)
            mock::minio::cmd_stat "$@"
            ;;
        admin)
            mock::minio::cmd_admin "$@"
            ;;
        version)
            mock::minio::cmd_version "$@"
            ;;
        *)
            echo "mc: Unknown command '$cmd'" >&2
            return 1
            ;;
    esac
}

# MinIO command implementations

mock::minio::cmd_config() {
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        host)
            mock::minio::config_host "$@"
            ;;
        *)
            echo "mc config: Unknown subcommand '$subcommand'" >&2
            return 1
            ;;
    esac
}

mock::minio::config_host() {
    local action="$1"
    shift
    
    case "$action" in
        add)
            local alias="$1"
            local url="$2"
            local access_key="$3"
            local secret_key="$4"
            
            if [[ -z "$alias" || -z "$url" || -z "$access_key" || -z "$secret_key" ]]; then
                echo "mc config host add: Missing required arguments" >&2
                return 1
            fi
            
            MINIO_MOCK_ALIASES[$alias]="$url|$access_key|$secret_key"
            echo "Added \`$alias\` successfully."
            return 0
            ;;
        list|ls)
            if [[ ${#MINIO_MOCK_ALIASES[@]} -eq 0 ]]; then
                echo "No aliases found."
            else
                for alias in "${!MINIO_MOCK_ALIASES[@]}"; do
                    local data="${MINIO_MOCK_ALIASES[$alias]}"
                    local url="${data%%|*}"
                    echo "$alias -> $url"
                done
            fi
            return 0
            ;;
        rm)
            local alias="$1"
            if [[ -n "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
                unset MINIO_MOCK_ALIASES[$alias]
                echo "Removed \`$alias\` successfully."
            else
                echo "mc config host rm: Alias \`$alias\` does not exist." >&2
                return 1
            fi
            return 0
            ;;
        *)
            echo "mc config host: Unknown action '$action'" >&2
            return 1
            ;;
    esac
}

mock::minio::cmd_ls() {
    local target="${1:-}"
    
    if [[ -z "$target" ]]; then
        # List all aliases
        if [[ ${#MINIO_MOCK_ALIASES[@]} -eq 0 ]]; then
            echo "No hosts configured."
        else
            for alias in "${!MINIO_MOCK_ALIASES[@]}"; do
                echo "$alias"
            done
        fi
        return 0
    fi
    
    # Parse target (alias/bucket/object)
    local alias="${target%%/*}"
    local path="${target#*/}"
    
    # Check if alias exists
    if [[ -z "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
        echo "mc ls: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    if [[ "$path" == "$alias" ]]; then
        # List buckets
        if [[ ${#MINIO_MOCK_BUCKETS[@]} -eq 0 ]]; then
            return 0
        fi
        
        for bucket in "${!MINIO_MOCK_BUCKETS[@]}"; do
            local created_date="${MINIO_MOCK_BUCKETS[$bucket]}"
            echo "[${created_date}]      0B $bucket/"
        done
    else
        # List objects in bucket
        local bucket="${path%%/*}"
        local prefix="${path#*/}"
        
        if [[ "$prefix" == "$bucket" ]]; then
            prefix=""
        fi
        
        if [[ -z "${MINIO_MOCK_BUCKETS[$bucket]}" ]]; then
            echo "mc ls: The specified bucket does not exist" >&2
            return 1
        fi
        
        # List objects with matching prefix
        local found_any=false
        for obj_key in "${!MINIO_MOCK_OBJECTS[@]}"; do
            if [[ "$obj_key" == "$bucket/"* ]]; then
                local object_name="${obj_key#*/}"
                if [[ -z "$prefix" || "$object_name" == "$prefix"* ]]; then
                    local metadata="${MINIO_MOCK_OBJECT_METADATA[$obj_key]}"
                    local size="1.0KB"
                    local date="2024-01-15 10:00:00 UTC"
                    if [[ -n "$metadata" ]]; then
                        local raw_size=$(echo "$metadata" | grep -o '"size":"[^"]*"' | cut -d'"' -f4 || echo "1024")
                        # Convert raw size to human readable format
                        if [[ "$raw_size" -lt 1024 ]]; then
                            size="${raw_size}B"
                        elif [[ "$raw_size" -lt 1048576 ]]; then
                            size="$(( raw_size / 1024 ))KB"
                        else
                            size="$(( raw_size / 1048576 ))MB"
                        fi
                        date=$(echo "$metadata" | grep -o '"last_modified":"[^"]*"' | cut -d'"' -f4 || echo "2024-01-15 10:00:00 UTC")
                    fi
                    echo "[${date}] ${size} $object_name"
                    found_any=true
                fi
            fi
        done
        
        if [[ "$found_any" == "false" && -n "$prefix" ]]; then
            return 0  # No objects found with prefix
        fi
    fi
    
    return 0
}

mock::minio::cmd_mb() {
    local target="$1"
    
    if [[ -z "$target" ]]; then
        echo "mc mb: Missing bucket name" >&2
        return 1
    fi
    
    # Parse target (alias/bucket)
    local alias="${target%%/*}"
    local bucket="${target#*/}"
    
    # Check if alias exists
    if [[ -z "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
        echo "mc mb: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    if [[ "$bucket" == "$alias" ]]; then
        echo "mc mb: Missing bucket name" >&2
        return 1
    fi
    
    # Check if bucket already exists
    if [[ -n "${MINIO_MOCK_BUCKETS[$bucket]}" ]]; then
        echo "mc mb: Bucket \`$alias/$bucket\` already exists." >&2
        return 1
    fi
    
    # Create bucket
    MINIO_MOCK_BUCKETS[$bucket]=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
    echo "Bucket created successfully \`$alias/$bucket\`."
    
    return 0
}

mock::minio::cmd_rb() {
    local target="$1"
    local force=""
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force="true"
                shift
                ;;
            *)
                target="$1"
                shift
                ;;
        esac
    done
    
    if [[ -z "$target" ]]; then
        echo "mc rb: Missing bucket name" >&2
        return 1
    fi
    
    # Parse target (alias/bucket)
    local alias="${target%%/*}"
    local bucket="${target#*/}"
    
    # Check if alias exists
    if [[ -z "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
        echo "mc rb: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    if [[ "$bucket" == "$alias" ]]; then
        echo "mc rb: Missing bucket name" >&2
        return 1
    fi
    
    # Check if bucket exists
    if [[ -z "${MINIO_MOCK_BUCKETS[$bucket]}" ]]; then
        echo "mc rb: The specified bucket does not exist" >&2
        return 1
    fi
    
    # Check if bucket is empty (unless force)
    local has_objects=false
    for obj_key in "${!MINIO_MOCK_OBJECTS[@]}"; do
        if [[ "$obj_key" == "$bucket/"* ]]; then
            has_objects=true
            break
        fi
    done
    
    if [[ "$has_objects" == "true" && "$force" != "true" ]]; then
        echo "mc rb: The bucket is not empty. Use --force to remove non-empty bucket." >&2
        return 1
    fi
    
    # Remove all objects in bucket if force
    if [[ "$force" == "true" ]]; then
        for obj_key in "${!MINIO_MOCK_OBJECTS[@]}"; do
            if [[ "$obj_key" == "$bucket/"* ]]; then
                unset MINIO_MOCK_OBJECTS[$obj_key]
                unset MINIO_MOCK_OBJECT_METADATA[$obj_key]
            fi
        done
    fi
    
    # Remove bucket
    unset MINIO_MOCK_BUCKETS[$bucket]
    unset MINIO_MOCK_BUCKET_POLICIES[$bucket]
    echo "Removed \`$alias/$bucket\` successfully."
    
    return 0
}

mock::minio::cmd_cp() {
    local source=""
    local target=""
    local recursive=""
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --recursive|-r)
                recursive="true"
                shift
                ;;
            *)
                if [[ -z "$source" ]]; then
                    source="$1"
                elif [[ -z "$target" ]]; then
                    target="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$source" || -z "$target" ]]; then
        echo "mc cp: Source and target are required" >&2
        return 1
    fi
    
    # Determine if this is upload (local -> remote) or download (remote -> local)
    if [[ "$source" == */* ]] && [[ "$target" != */* ]]; then
        # Download: remote -> local
        mock::minio::download_object "$source" "$target"
    elif [[ "$source" != */* ]] && [[ "$target" == */* ]]; then
        # Upload: local -> remote
        mock::minio::upload_object "$source" "$target"
    else
        echo "mc cp: Invalid source/target combination" >&2
        return 1
    fi
}

mock::minio::upload_object() {
    local local_file="$1"
    local remote_target="$2"
    
    # Parse remote target
    local alias="${remote_target%%/*}"
    local path="${remote_target#*/}"
    local bucket="${path%%/*}"
    local object_key="${path#*/}"
    
    # Check if alias exists
    if [[ -z "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
        echo "mc cp: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    # Check if bucket exists
    if [[ -z "${MINIO_MOCK_BUCKETS[$bucket]}" ]]; then
        echo "mc cp: The specified bucket does not exist" >&2
        return 1
    fi
    
    # Check if local file exists
    if [[ ! -f "$local_file" ]]; then
        echo "mc cp: File '$local_file' not found" >&2
        return 1
    fi
    
    # If object_key is same as bucket, use filename
    if [[ "$object_key" == "$bucket" ]]; then
        object_key=$(basename "$local_file")
    fi
    
    # Handle case where remote_target ends with '/'
    if [[ "$remote_target" == *"/" ]]; then
        object_key=$(basename "$local_file")
    fi
    
    # Copy file to mock storage
    local mock_file="$MINIO_MOCK_FILES_DIR/${bucket}_${object_key//\//_}"
    cp "$local_file" "$mock_file"
    
    # Store object reference
    local full_key="$bucket/$object_key"
    MINIO_MOCK_OBJECTS[$full_key]="$mock_file"
    
    # Store metadata
    local file_size=$(stat -c%s "$local_file" 2>/dev/null || stat -f%z "$local_file" 2>/dev/null || echo "1024")
    MINIO_MOCK_OBJECT_METADATA[$full_key]="{\"size\":\"$file_size\",\"last_modified\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"etag\":\"$(echo -n "$object_key" | md5sum | cut -d' ' -f1 || echo 'abc123')\"}"
    
    echo "\`$local_file\` -> \`$remote_target\`"
    echo "Total: 1 object(s), 1 file(s). $(( file_size / 1024 ))KB uploaded."
    
    return 0
}

mock::minio::download_object() {
    local remote_source="$1"
    local local_target="$2"
    
    # Parse remote source
    local alias="${remote_source%%/*}"
    local path="${remote_source#*/}"
    local bucket="${path%%/*}"
    local object_key="${path#*/}"
    
    # Check if alias exists
    if [[ -z "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
        echo "mc cp: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    # Check if object exists
    local full_key="$bucket/$object_key"
    if [[ -z "${MINIO_MOCK_OBJECTS[$full_key]}" ]]; then
        echo "mc cp: Object does not exist" >&2
        return 1
    fi
    
    # Copy from mock storage to local target
    local mock_file="${MINIO_MOCK_OBJECTS[$full_key]}"
    if [[ -f "$mock_file" ]]; then
        cp "$mock_file" "$local_target"
        echo "\`$remote_source\` -> \`$local_target\`"
        echo "Total: 1 object(s), 1 file(s). Downloaded."
        return 0
    else
        echo "mc cp: Object data not found in mock storage" >&2
        return 1
    fi
}

mock::minio::cmd_rm() {
    local target="$1"
    local recursive=""
    local force=""
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --recursive|-r)
                recursive="true"
                shift
                ;;
            --force)
                force="true"
                shift
                ;;
            *)
                target="$1"
                shift
                ;;
        esac
    done
    
    if [[ -z "$target" ]]; then
        echo "mc rm: Missing target" >&2
        return 1
    fi
    
    # Parse target
    local alias="${target%%/*}"
    local path="${target#*/}"
    local bucket="${path%%/*}"
    local object_key="${path#*/}"
    
    # Check if alias exists
    if [[ -z "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
        echo "mc rm: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    if [[ "$object_key" == "$bucket" ]]; then
        echo "mc rm: Cannot remove bucket, use rb command" >&2
        return 1
    fi
    
    # Remove object
    local full_key="$bucket/$object_key"
    if [[ -n "${MINIO_MOCK_OBJECTS[$full_key]}" ]]; then
        local mock_file="${MINIO_MOCK_OBJECTS[$full_key]}"
        rm -f "$mock_file"
        unset MINIO_MOCK_OBJECTS[$full_key]
        unset MINIO_MOCK_OBJECT_METADATA[$full_key]
        echo "Removed \`$target\`."
        return 0
    else
        echo "mc rm: Object does not exist" >&2
        return 1
    fi
}

mock::minio::cmd_du() {
    local target="${1:-}"
    local summarize=""
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --summarize)
                summarize="true"
                shift
                ;;
            *)
                target="$1"
                shift
                ;;
        esac
    done
    
    if [[ -z "$target" ]]; then
        echo "mc du: Missing target" >&2
        return 1
    fi
    
    # Parse target
    local alias="${target%%/*}"
    local path="${target#*/}"
    local bucket="${path%%/*}"
    
    # Check if alias exists
    if [[ -z "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
        echo "mc du: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    # Check if bucket exists
    if [[ -z "${MINIO_MOCK_BUCKETS[$bucket]}" ]]; then
        echo "mc du: The specified bucket does not exist" >&2
        return 1
    fi
    
    # Calculate total size
    local total_size=0
    local object_count=0
    for obj_key in "${!MINIO_MOCK_OBJECTS[@]}"; do
        if [[ "$obj_key" == "$bucket/"* ]]; then
            local metadata="${MINIO_MOCK_OBJECT_METADATA[$obj_key]}"
            local size=1024
            if [[ -n "$metadata" ]]; then
                size=$(echo "$metadata" | grep -o '"size":"[^"]*"' | cut -d'"' -f4 || echo "1024")
            fi
            total_size=$((total_size + size))
            ((object_count++)) || true
        fi
    done
    
    if [[ "$summarize" == "true" ]]; then
        echo "${total_size}B"
    else
        echo "${total_size}B $target ($object_count objects)"
    fi
    
    return 0
}

mock::minio::cmd_anonymous() {
    local action="$1"
    local policy_type="$2"
    local target="$3"
    
    if [[ "$action" != "set" && "$action" != "get" ]]; then
        echo "mc anonymous: Unknown action '$action'" >&2
        return 1
    fi
    
    if [[ "$action" == "set" ]]; then
        if [[ -z "$policy_type" || -z "$target" ]]; then
            echo "mc anonymous set: Policy type and target are required" >&2
            return 1
        fi
        
        # Parse target
        local alias="${target%%/*}"
        local bucket="${target#*/}"
        
        # Check if alias exists
        if [[ -z "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
            echo "mc anonymous: Alias \`$alias\` does not exist." >&2
            return 1
        fi
        
        # Check if bucket exists
        if [[ -z "${MINIO_MOCK_BUCKETS[$bucket]}" ]]; then
            echo "mc anonymous: The specified bucket does not exist" >&2
            return 1
        fi
        
        # Set policy
        MINIO_MOCK_BUCKET_POLICIES[$bucket]="$policy_type"
        echo "Access permission for \`$target\` is set to \`$policy_type\`"
        
    elif [[ "$action" == "get" ]]; then
        if [[ -z "$policy_type" ]]; then
            echo "mc anonymous get: Target is required" >&2
            return 1
        fi
        
        # Parse target (policy_type is actually target in get)
        local target="$policy_type"
        local alias="${target%%/*}"
        local bucket="${target#*/}"
        
        # Check if alias exists
        if [[ -z "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
            echo "mc anonymous: Alias \`$alias\` does not exist." >&2
            return 1
        fi
        
        # Get policy
        local current_policy="${MINIO_MOCK_BUCKET_POLICIES[$bucket]:-none}"
        echo "Access permission for \`$target\` is \`$current_policy\`"
    fi
    
    return 0
}

mock::minio::cmd_stat() {
    local target="$1"
    
    if [[ -z "$target" ]]; then
        echo "mc stat: Missing target" >&2
        return 1
    fi
    
    # Parse target
    local alias="${target%%/*}"
    local path="${target#*/}"
    local bucket="${path%%/*}"
    local object_key="${path#*/}"
    
    # Check if alias exists
    if [[ -z "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
        echo "mc stat: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    if [[ "$object_key" == "$bucket" ]]; then
        # Bucket stat
        if [[ -z "${MINIO_MOCK_BUCKETS[$bucket]}" ]]; then
            echo "mc stat: The specified bucket does not exist" >&2
            return 1
        fi
        
        local created_date="${MINIO_MOCK_BUCKETS[$bucket]}"
        echo "Name      : $bucket/"
        echo "Date      : $created_date"
        echo "Size      : 0B"
        echo "Type      : folder"
        
    else
        # Object stat
        local full_key="$bucket/$object_key"
        if [[ -z "${MINIO_MOCK_OBJECTS[$full_key]}" ]]; then
            echo "mc stat: Object does not exist" >&2
            return 1
        fi
        
        local metadata="${MINIO_MOCK_OBJECT_METADATA[$full_key]}"
        local size="1024"
        local date="2024-01-15 10:00:00 UTC"
        local etag="abc123"
        
        if [[ -n "$metadata" ]]; then
            size=$(echo "$metadata" | grep -o '"size":"[^"]*"' | cut -d'"' -f4 || echo "1024")
            date=$(echo "$metadata" | grep -o '"last_modified":"[^"]*"' | cut -d'"' -f4 || echo "2024-01-15T10:00:00Z")
            etag=$(echo "$metadata" | grep -o '"etag":"[^"]*"' | cut -d'"' -f4 || echo "abc123")
        fi
        
        echo "Name      : $object_key"
        echo "Date      : $date"
        echo "Size      : ${size}B"
        echo "ETag      : $etag"
        echo "Type      : file"
    fi
    
    return 0
}

mock::minio::cmd_admin() {
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        info)
            mock::minio::admin_info "$@"
            ;;
        user)
            mock::minio::admin_user "$@"
            ;;
        policy)
            mock::minio::admin_policy "$@"
            ;;
        *)
            echo "mc admin: Unknown subcommand '$subcommand'" >&2
            return 1
            ;;
    esac
}

mock::minio::admin_info() {
    local alias="${1:-local}"
    
    # Check if alias exists
    if [[ -z "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
        echo "mc admin info: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    echo "●  ${alias}:${MINIO_MOCK_CONFIG[port]}"
    echo "   Uptime: 1 day"
    echo "   Version: ${MINIO_MOCK_CONFIG[version]}"
    echo "   Network: 1/1 OK"
    echo ""
    echo "Drives: 1/1 OK"
    echo ""
    echo "Pool 1, Set 1: 1 drive, 1024 MB Total, 512 MB Available"
    echo "└─ 1: http://localhost:9000/data OK"
    
    return 0
}

mock::minio::admin_user() {
    local action="$1"
    shift
    
    case "$action" in
        add)
            local alias="$1"
            local username="$2"
            local password="$3"
            
            echo "Added user \`$username\` successfully."
            return 0
            ;;
        list)
            local alias="$1"
            echo "enabled    ${MINIO_MOCK_CONFIG[root_user]}"
            return 0
            ;;
        *)
            echo "mc admin user: Unknown action '$action'" >&2
            return 1
            ;;
    esac
}

mock::minio::admin_policy() {
    echo "mc admin policy: Policy management mock not implemented" >&2
    return 1
}

mock::minio::cmd_version() {
    echo "mc version ${MINIO_MOCK_CONFIG[version]}"
    return 0
}

# Test helper functions
mock::minio::reset() {
    local save_state="${1:-true}"
    
    # Clear all data
    MINIO_MOCK_BUCKETS=()
    MINIO_MOCK_OBJECTS=()
    MINIO_MOCK_OBJECT_METADATA=()
    MINIO_MOCK_BUCKET_POLICIES=()
    MINIO_MOCK_ALIASES=()
    
    # Reset configuration
    MINIO_MOCK_CONFIG=(
        [container_name]="vrooli_minio"
        [port]="9000"
        [console_port]="9001"
        [root_user]="minioadmin"
        [root_password]="minioadmin"
        [state]="running"
        [health_status]="healthy"
        [data_dir]="/tmp/minio-data"
        [config_dir]="/tmp/minio-config"
        [version]="RELEASE.2024-01-01T00-00-00Z"
        [error_mode]=""
    )
    
    # Clean up mock files
    rm -rf "$MINIO_MOCK_FILES_DIR"
    mkdir -p "$MINIO_MOCK_FILES_DIR"
    
    # Save the reset state if requested
    if [[ "$save_state" == "true" ]]; then
        mock::minio::save_state
    fi
    
    mock::log_state "minio" "MinIO mock reset to initial state"
}

mock::minio::set_error() {
    local error_mode="$1"
    MINIO_MOCK_CONFIG[error_mode]="$error_mode"
    mock::minio::save_state
    mock::log_state "minio" "Set MinIO error mode: $error_mode"
}

mock::minio::set_state() {
    local state="$1"
    MINIO_MOCK_CONFIG[state]="$state"
    mock::minio::save_state
    mock::log_state "minio" "Set MinIO state: $state"
}

mock::minio::set_config() {
    local key="$1"
    local value="$2"
    MINIO_MOCK_CONFIG[$key]="$value"
    mock::minio::save_state
    mock::log_state "minio" "Set MinIO config: $key=$value"
}

# Test assertions
mock::minio::assert_bucket_exists() {
    local bucket="$1"
    if [[ -n "${MINIO_MOCK_BUCKETS[$bucket]}" ]]; then
        return 0
    else
        echo "Assertion failed: Bucket '$bucket' does not exist" >&2
        return 1
    fi
}

mock::minio::assert_object_exists() {
    local bucket="$1"
    local object_key="$2"
    local full_key="$bucket/$object_key"
    
    if [[ -n "${MINIO_MOCK_OBJECTS[$full_key]}" ]]; then
        return 0
    else
        echo "Assertion failed: Object '$object_key' does not exist in bucket '$bucket'" >&2
        return 1
    fi
}

mock::minio::assert_bucket_policy() {
    local bucket="$1"
    local expected_policy="$2"
    local actual_policy="${MINIO_MOCK_BUCKET_POLICIES[$bucket]:-none}"
    
    if [[ "$actual_policy" == "$expected_policy" ]]; then
        return 0
    else
        echo "Assertion failed: Bucket '$bucket' policy mismatch" >&2
        echo "  Expected: '$expected_policy'" >&2
        echo "  Actual: '$actual_policy'" >&2
        return 1
    fi
}

mock::minio::assert_alias_configured() {
    local alias="$1"
    if [[ -n "${MINIO_MOCK_ALIASES[$alias]}" ]]; then
        return 0
    else
        echo "Assertion failed: Alias '$alias' is not configured" >&2
        return 1
    fi
}

# Debug functions
mock::minio::dump_state() {
    echo "=== MinIO Mock State ==="
    echo "Configuration:"
    for key in "${!MINIO_MOCK_CONFIG[@]}"; do
        echo "  $key: ${MINIO_MOCK_CONFIG[$key]}"
    done
    
    echo "Aliases:"
    for alias in "${!MINIO_MOCK_ALIASES[@]}"; do
        echo "  $alias: ${MINIO_MOCK_ALIASES[$alias]}"
    done
    
    echo "Buckets:"
    for bucket in "${!MINIO_MOCK_BUCKETS[@]}"; do
        echo "  $bucket: ${MINIO_MOCK_BUCKETS[$bucket]}"
    done
    
    echo "Objects:"
    for obj_key in "${!MINIO_MOCK_OBJECTS[@]}"; do
        echo "  $obj_key: ${MINIO_MOCK_OBJECTS[$obj_key]}"
    done
    
    echo "Bucket Policies:"
    for bucket in "${!MINIO_MOCK_BUCKET_POLICIES[@]}"; do
        echo "  $bucket: ${MINIO_MOCK_BUCKET_POLICIES[$bucket]}"
    done
    echo "======================="
}

# Export all functions
export -f mc
export -f mock::minio::save_state
export -f mock::minio::load_state
export -f mock::minio::execute_command
export -f mock::minio::cmd_config
export -f mock::minio::config_host
export -f mock::minio::cmd_ls
export -f mock::minio::cmd_mb
export -f mock::minio::cmd_rb
export -f mock::minio::cmd_cp
export -f mock::minio::upload_object
export -f mock::minio::download_object
export -f mock::minio::cmd_rm
export -f mock::minio::cmd_du
export -f mock::minio::cmd_anonymous
export -f mock::minio::cmd_stat
export -f mock::minio::cmd_admin
export -f mock::minio::admin_info
export -f mock::minio::admin_user
export -f mock::minio::admin_policy
export -f mock::minio::cmd_version
export -f mock::minio::reset
export -f mock::minio::set_error
export -f mock::minio::set_state
export -f mock::minio::set_config
export -f mock::minio::assert_bucket_exists
export -f mock::minio::assert_object_exists
export -f mock::minio::assert_bucket_policy
export -f mock::minio::assert_alias_configured
export -f mock::minio::dump_state

# Save initial state
mock::minio::save_state