#!/usr/bin/env bash
# MinIO Mock - Tier 2 (Stateful)
# 
# Provides stateful MinIO S3-compatible object storage mock for testing:
# - MinIO client (mc) command interception
# - Bucket management (create, list, delete)
# - Object operations (upload, download, list, delete)
# - Alias configuration (config host add/list)
# - Admin operations and health checks
# - Error injection for resilience testing
#
# Coverage: ~80% of common MinIO use cases in 500 lines

# === Configuration ===
declare -gA MINIO_BUCKETS=()            # Bucket storage: name -> "created_date"
declare -gA MINIO_OBJECTS=()             # Object storage: bucket/key -> "size|modified|data"
declare -gA MINIO_ALIASES=()             # MC client aliases: name -> "url|access_key|secret_key"
declare -gA MINIO_CONFIG=(               # MinIO configuration
    [port]="9000"
    [console_port]="9001"
    [root_user]="minioadmin"
    [root_password]="minioadmin"
    [version]="RELEASE.2024-01-01T00-00-00Z"
    [state]="running"
    [error_mode]=""
)

# Debug mode
declare -g MINIO_DEBUG="${MINIO_DEBUG:-}"

# === Helper Functions ===
minio_debug() {
    [[ -n "$MINIO_DEBUG" ]] && echo "[MOCK:MINIO] $*" >&2
}

minio_check_error() {
    case "${MINIO_CONFIG[error_mode]}" in
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
    return 0
}

minio_parse_target() {
    local target="$1"
    local alias="${target%%/*}"
    local path="${target#*/}"
    
    if [[ "$path" == "$alias" ]]; then
        echo "$alias||"
    else
        local bucket="${path%%/*}"
        local object_key="${path#*/}"
        if [[ "$object_key" == "$bucket" ]]; then
            echo "$alias|$bucket|"
        else
            echo "$alias|$bucket|$object_key"
        fi
    fi
}

# === Main mc Command Interceptor ===
mc() {
    minio_debug "mc called with: $*"
    
    # Check server state
    if [[ "${MINIO_CONFIG[state]}" != "running" ]]; then
        echo "mc: Unable to connect to MinIO server" >&2
        return 1
    fi
    
    # Check for error injection
    if ! minio_check_error; then
        return $?
    fi
    
    if [[ $# -eq 0 ]]; then
        echo "mc: No command specified" >&2
        return 1
    fi
    
    # Route to command handlers
    local cmd="$1"
    shift
    
    case "$cmd" in
        config) minio_cmd_config "$@" ;;
        ls) minio_cmd_ls "$@" ;;
        mb) minio_cmd_mb "$@" ;;
        rb) minio_cmd_rb "$@" ;;
        cp) minio_cmd_cp "$@" ;;
        rm) minio_cmd_rm "$@" ;;
        du) minio_cmd_du "$@" ;;
        stat) minio_cmd_stat "$@" ;;
        admin) minio_cmd_admin "$@" ;;
        version) echo "mc version ${MINIO_CONFIG[version]}" ;;
        *)
            echo "mc: Unknown command '$cmd'" >&2
            return 1
            ;;
    esac
}

# === Command Implementations ===

minio_cmd_config() {
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        host)
            local action="$1"
            shift
            case "$action" in
                add)
                    local alias="$1" url="$2" access_key="$3" secret_key="$4"
                    if [[ -z "$alias" || -z "$url" || -z "$access_key" || -z "$secret_key" ]]; then
                        echo "mc config host add: Missing required arguments" >&2
                        return 1
                    fi
                    MINIO_ALIASES[$alias]="$url|$access_key|$secret_key"
                    echo "Added \`$alias\` successfully."
                    ;;
                list|ls)
                    if [[ ${#MINIO_ALIASES[@]} -eq 0 ]]; then
                        echo "No aliases found."
                    else
                        for alias in "${!MINIO_ALIASES[@]}"; do
                            local data="${MINIO_ALIASES[$alias]}"
                            local url="${data%%|*}"
                            echo "$alias -> $url"
                        done
                    fi
                    ;;
                *)
                    echo "mc config host: Unknown action '$action'" >&2
                    return 1
                    ;;
            esac
            ;;
        *)
            echo "mc config: Unknown subcommand '$subcommand'" >&2
            return 1
            ;;
    esac
}

minio_cmd_ls() {
    local target="${1:-}"
    
    if [[ -z "$target" ]]; then
        # List all aliases
        if [[ ${#MINIO_ALIASES[@]} -eq 0 ]]; then
            echo "No hosts configured."
        else
            for alias in "${!MINIO_ALIASES[@]}"; do
                echo "$alias"
            done
        fi
        return 0
    fi
    
    local parse_result
    parse_result=$(minio_parse_target "$target")
    IFS='|' read -r alias bucket object_key <<< "$parse_result"
    
    # Check if alias exists
    if [[ -z "${MINIO_ALIASES[$alias]}" ]]; then
        echo "mc ls: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    if [[ -z "$bucket" ]]; then
        # List buckets
        if [[ ${#MINIO_BUCKETS[@]} -eq 0 ]]; then
            return 0
        fi
        for bucket_name in "${!MINIO_BUCKETS[@]}"; do
            local created_date="${MINIO_BUCKETS[$bucket_name]}"
            echo "[${created_date}]      0B $bucket_name/"
        done
    else
        # List objects in bucket
        if [[ -z "${MINIO_BUCKETS[$bucket]}" ]]; then
            echo "mc ls: The specified bucket does not exist" >&2
            return 1
        fi
        
        for obj_key in "${!MINIO_OBJECTS[@]}"; do
            if [[ "$obj_key" == "$bucket/"* ]]; then
                local object_name="${obj_key#*/}"
                local obj_data="${MINIO_OBJECTS[$obj_key]}"
                IFS='|' read -r size modified data <<< "$obj_data"
                echo "[${modified}] ${size}B $object_name"
            fi
        done
    fi
}

minio_cmd_mb() {
    local target="$1"
    
    if [[ -z "$target" ]]; then
        echo "mc mb: Missing bucket name" >&2
        return 1
    fi
    
    local parse_result
    parse_result=$(minio_parse_target "$target")
    IFS='|' read -r alias bucket object_key <<< "$parse_result"
    
    # Check if alias exists
    if [[ -z "${MINIO_ALIASES[$alias]}" ]]; then
        echo "mc mb: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    if [[ -z "$bucket" ]]; then
        echo "mc mb: Missing bucket name" >&2
        return 1
    fi
    
    # Check if bucket already exists
    if [[ -n "${MINIO_BUCKETS[$bucket]}" ]]; then
        echo "mc mb: Bucket \`$alias/$bucket\` already exists." >&2
        return 1
    fi
    
    # Create bucket
    MINIO_BUCKETS[$bucket]=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
    echo "Bucket created successfully \`$alias/$bucket\`."
    minio_debug "Created bucket: $bucket"
}

minio_cmd_rb() {
    local target="$1"
    local force=""
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force="true"; shift ;;
            *) target="$1"; shift ;;
        esac
    done
    
    if [[ -z "$target" ]]; then
        echo "mc rb: Missing bucket name" >&2
        return 1
    fi
    
    local parse_result
    parse_result=$(minio_parse_target "$target")
    IFS='|' read -r alias bucket object_key <<< "$parse_result"
    
    # Check if alias exists
    if [[ -z "${MINIO_ALIASES[$alias]}" ]]; then
        echo "mc rb: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    # Check if bucket exists
    if [[ -z "${MINIO_BUCKETS[$bucket]}" ]]; then
        echo "mc rb: The specified bucket does not exist" >&2
        return 1
    fi
    
    # Check if bucket is empty (unless force)
    local has_objects=false
    for obj_key in "${!MINIO_OBJECTS[@]}"; do
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
        for obj_key in "${!MINIO_OBJECTS[@]}"; do
            if [[ "$obj_key" == "$bucket/"* ]]; then
                unset MINIO_OBJECTS[$obj_key]
            fi
        done
    fi
    
    # Remove bucket
    unset MINIO_BUCKETS[$bucket]
    echo "Removed \`$alias/$bucket\` successfully."
    minio_debug "Deleted bucket: $bucket"
}

minio_cmd_cp() {
    local source="$1" target="$2"
    
    if [[ -z "$source" || -z "$target" ]]; then
        echo "mc cp: Source and target are required" >&2
        return 1
    fi
    
    # Determine if this is upload (local -> remote) or download (remote -> local)
    if [[ "$source" != */* ]] && [[ "$target" == */* ]]; then
        # Upload: local -> remote
        minio_handle_upload "$source" "$target"
    elif [[ "$source" == */* ]] && [[ "$target" != */* ]]; then
        # Download: remote -> local
        minio_handle_download "$source" "$target"
    else
        echo "mc cp: Invalid source/target combination" >&2
        return 1
    fi
}

minio_handle_upload() {
    local local_file="$1" target="$2"
    
    local parse_result
    parse_result=$(minio_parse_target "$target")
    IFS='|' read -r alias bucket object_key <<< "$parse_result"
    
    # Check if alias and bucket exist
    if [[ -z "${MINIO_ALIASES[$alias]}" ]]; then
        echo "mc cp: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    if [[ -z "${MINIO_BUCKETS[$bucket]}" ]]; then
        echo "mc cp: The specified bucket does not exist" >&2
        return 1
    fi
    
    # Check if local file exists (simulate)
    if [[ ! -f "$local_file" ]]; then
        echo "mc cp: File '$local_file' not found" >&2
        return 1
    fi
    
    # If no object key, use filename
    if [[ -z "$object_key" ]]; then
        object_key=$(basename "$local_file")
    fi
    
    # Simulate file upload
    local file_size=1024  # Simulated size
    if [[ -f "$local_file" ]]; then
        file_size=$(stat -c%s "$local_file" 2>/dev/null || stat -f%z "$local_file" 2>/dev/null || echo "1024")
    fi
    
    local full_key="$bucket/$object_key"
    MINIO_OBJECTS[$full_key]="$file_size|$(date -u +%Y-%m-%dT%H:%M:%SZ)|mock_data"
    
    echo "\`$local_file\` -> \`$target\`"
    echo "Total: 1 object(s), 1 file(s). $(( file_size / 1024 ))KB uploaded."
    minio_debug "Uploaded: $full_key ($file_size bytes)"
}

minio_handle_download() {
    local source="$1" local_target="$2"
    
    local parse_result
    parse_result=$(minio_parse_target "$source")
    IFS='|' read -r alias bucket object_key <<< "$parse_result"
    
    # Check if object exists
    local full_key="$bucket/$object_key"
    if [[ -z "${MINIO_OBJECTS[$full_key]}" ]]; then
        echo "mc cp: Object does not exist" >&2
        return 1
    fi
    
    # Simulate download by creating a mock file
    echo "mock_file_content" > "$local_target"
    echo "\`$source\` -> \`$local_target\`"
    echo "Total: 1 object(s), 1 file(s). Downloaded."
    minio_debug "Downloaded: $full_key to $local_target"
}

minio_cmd_rm() {
    local target="$1"
    
    if [[ -z "$target" ]]; then
        echo "mc rm: Missing target" >&2
        return 1
    fi
    
    local parse_result
    parse_result=$(minio_parse_target "$target")
    IFS='|' read -r alias bucket object_key <<< "$parse_result"
    
    # Check if alias exists
    if [[ -z "${MINIO_ALIASES[$alias]}" ]]; then
        echo "mc rm: Alias \`$alias\` does not exist." >&2
        return 1
    fi
    
    if [[ -z "$object_key" ]]; then
        echo "mc rm: Cannot remove bucket, use rb command" >&2
        return 1
    fi
    
    # Remove object
    local full_key="$bucket/$object_key"
    if [[ -n "${MINIO_OBJECTS[$full_key]}" ]]; then
        unset MINIO_OBJECTS[$full_key]
        echo "Removed \`$target\`."
        minio_debug "Removed object: $full_key"
    else
        echo "mc rm: Object does not exist" >&2
        return 1
    fi
}

minio_cmd_du() {
    local target="$1"
    
    if [[ -z "$target" ]]; then
        echo "mc du: Missing target" >&2
        return 1
    fi
    
    local parse_result
    parse_result=$(minio_parse_target "$target")
    IFS='|' read -r alias bucket object_key <<< "$parse_result"
    
    # Check if bucket exists
    if [[ -z "${MINIO_BUCKETS[$bucket]}" ]]; then
        echo "mc du: The specified bucket does not exist" >&2
        return 1
    fi
    
    # Calculate total size
    local total_size=0 object_count=0
    for obj_key in "${!MINIO_OBJECTS[@]}"; do
        if [[ "$obj_key" == "$bucket/"* ]]; then
            local obj_data="${MINIO_OBJECTS[$obj_key]}"
            local size="${obj_data%%|*}"
            total_size=$((total_size + size))
            ((object_count++))
        fi
    done
    
    echo "${total_size}B $target ($object_count objects)"
}

minio_cmd_stat() {
    local target="$1"
    
    if [[ -z "$target" ]]; then
        echo "mc stat: Missing target" >&2
        return 1
    fi
    
    local parse_result
    parse_result=$(minio_parse_target "$target")
    IFS='|' read -r alias bucket object_key <<< "$parse_result"
    
    if [[ -z "$object_key" ]]; then
        # Bucket stat
        if [[ -z "${MINIO_BUCKETS[$bucket]}" ]]; then
            echo "mc stat: The specified bucket does not exist" >&2
            return 1
        fi
        local created_date="${MINIO_BUCKETS[$bucket]}"
        echo "Name      : $bucket/"
        echo "Date      : $created_date"
        echo "Size      : 0B"
        echo "Type      : folder"
    else
        # Object stat
        local full_key="$bucket/$object_key"
        if [[ -z "${MINIO_OBJECTS[$full_key]}" ]]; then
            echo "mc stat: Object does not exist" >&2
            return 1
        fi
        local obj_data="${MINIO_OBJECTS[$full_key]}"
        IFS='|' read -r size modified data <<< "$obj_data"
        echo "Name      : $object_key"
        echo "Date      : $modified"
        echo "Size      : ${size}B"
        echo "Type      : file"
    fi
}

minio_cmd_admin() {
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        info)
            local alias="${1:-local}"
            if [[ -z "${MINIO_ALIASES[$alias]}" ]]; then
                echo "mc admin info: Alias \`$alias\` does not exist." >&2
                return 1
            fi
            echo "●  ${alias}:${MINIO_CONFIG[port]}"
            echo "   Uptime: 1 day"
            echo "   Version: ${MINIO_CONFIG[version]}"
            echo "   Network: 1/1 OK"
            echo ""
            echo "Drives: 1/1 OK"
            ;;
        *)
            echo "mc admin: Unknown subcommand '$subcommand'" >&2
            return 1
            ;;
    esac
}

# === Convention-based Test Functions ===
test_minio_connection() {
    minio_debug "Testing connection..."
    
    # Test version command
    local result
    result=$(mc version 2>&1)
    
    if [[ "$result" =~ "mc version" ]]; then
        minio_debug "Connection test passed"
        return 0
    else
        minio_debug "Connection test failed: $result"
        return 1
    fi
}

test_minio_health() {
    minio_debug "Testing health..."
    
    # Test connection
    test_minio_connection || return 1
    
    # Test alias configuration
    mc config host add test-alias http://localhost:9000 minioadmin minioadmin >/dev/null 2>&1 || return 1
    
    # Test basic operations
    mc mb test-alias/health-test >/dev/null 2>&1 || return 1
    
    # Create test file and upload
    echo "health test content" > /tmp/health-test.txt
    mc cp /tmp/health-test.txt test-alias/health-test/test.txt >/dev/null 2>&1 || return 1
    
    # List objects
    mc ls test-alias/health-test/ | grep -q "test.txt" || return 1
    
    # Clean up
    mc rb test-alias/health-test --force >/dev/null 2>&1
    rm -f /tmp/health-test.txt
    
    minio_debug "Health test passed"
    return 0
}

test_minio_basic() {
    minio_debug "Testing basic operations..."
    
    # Test version
    mc version >/dev/null 2>&1 || return 1
    
    # Test alias management
    mc config host add basic-test http://localhost:9000 minioadmin minioadmin >/dev/null 2>&1 || return 1
    mc config host list | grep -q "basic-test" || return 1
    
    # Test bucket operations
    mc mb basic-test/basic-test-bucket >/dev/null 2>&1 || return 1
    mc ls basic-test/ | grep -q "basic-test-bucket" || return 1
    
    # Test object operations
    echo "basic test data" > /tmp/basic-test.txt
    mc cp /tmp/basic-test.txt basic-test/basic-test-bucket/test.txt >/dev/null 2>&1 || return 1
    mc ls basic-test/basic-test-bucket/ | grep -q "test.txt" || return 1
    
    # Test admin info
    mc admin info basic-test >/dev/null 2>&1 || return 1
    
    # Clean up
    mc rb basic-test/basic-test-bucket --force >/dev/null 2>&1
    rm -f /tmp/basic-test.txt
    
    minio_debug "Basic test passed"
    return 0
}

# === State Management ===
minio_mock_reset() {
    minio_debug "Resetting mock state (called from: ${BASH_SOURCE[1]:-unknown}:${BASH_LINENO[0]:-unknown})"
    
    MINIO_BUCKETS=()
    MINIO_OBJECTS=()
    MINIO_ALIASES=()
    MINIO_CONFIG[error_mode]=""
    MINIO_CONFIG[state]="running"
    
    # Create default alias for testing
    MINIO_ALIASES["local"]="http://localhost:9000|minioadmin|minioadmin"
}

minio_mock_set_error() {
    MINIO_CONFIG[error_mode]="$1"
    minio_debug "Set error mode: $1"
}

minio_mock_set_state() {
    MINIO_CONFIG[state]="$1"
    minio_debug "Set state: $1"
}

minio_mock_dump_state() {
    echo "=== MinIO Mock State ==="
    echo "Server: localhost:${MINIO_CONFIG[port]} (state: ${MINIO_CONFIG[state]})"
    echo "Version: ${MINIO_CONFIG[version]}"
    echo "Aliases: ${#MINIO_ALIASES[@]}"
    for alias in "${!MINIO_ALIASES[@]}"; do
        echo "  $alias: ${MINIO_ALIASES[$alias]}"
    done
    echo "Buckets: ${#MINIO_BUCKETS[@]}"
    for bucket in "${!MINIO_BUCKETS[@]}"; do
        echo "  $bucket: ${MINIO_BUCKETS[$bucket]}"
    done
    echo "Objects: ${#MINIO_OBJECTS[@]}"
    for obj_key in "${!MINIO_OBJECTS[@]}"; do
        echo "  $obj_key: ${MINIO_OBJECTS[$obj_key]}"
    done
    echo "Error Mode: ${MINIO_CONFIG[error_mode]:-none}"
    echo "===================="
}

# === Export Functions ===
export -f mc
export -f test_minio_connection
export -f test_minio_health
export -f test_minio_basic
export -f minio_mock_reset
export -f minio_mock_set_error
export -f minio_mock_set_state
export -f minio_mock_dump_state
export -f minio_debug
export -f minio_check_error

# Initialize with default state
minio_mock_reset
minio_debug "MinIO Tier 2 mock initialized"