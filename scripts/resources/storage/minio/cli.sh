#!/usr/bin/env bash
################################################################################
# MinIO Resource CLI
# 
# Lightweight CLI interface for MinIO that delegates to existing lib functions.
#
# Usage:
#   resource-minio <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (handle symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    MINIO_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    MINIO_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
MINIO_CLI_DIR="$(cd "$(dirname "$MINIO_CLI_SCRIPT")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$MINIO_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$MINIO_CLI_DIR"
export MINIO_SCRIPT_DIR="$MINIO_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Source MinIO configuration
# shellcheck disable=SC1091
source "${MINIO_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
minio::export_config 2>/dev/null || true

# Source MinIO libraries
for lib in common docker status api buckets; do
    lib_file="${MINIO_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize with resource name
resource_cli::init "minio"

################################################################################
# Delegate to existing MinIO functions
################################################################################

# Inject bucket configuration or data into MinIO
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-minio inject <file.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would inject: $file"
        return 0
    fi
    
    # Use existing injection function
    if command -v minio::inject_file &>/dev/null; then
        minio::inject_file "$file"
    elif command -v minio::inject_bucket_config &>/dev/null; then
        minio::inject_bucket_config "$file"
    else
        log::error "MinIO injection functions not available"
        return 1
    fi
}

# Validate MinIO configuration
resource_cli::validate() {
    if command -v minio::validate &>/dev/null; then
        minio::validate
    elif command -v minio::check_health &>/dev/null; then
        minio::check_health
    else
        # Basic validation
        log::header "Validating MinIO"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "minio" || {
            log::error "MinIO container not running"
            return 1
        }
        log::success "MinIO is running"
    fi
}

# Show MinIO status
resource_cli::status() {
    if command -v minio::status &>/dev/null; then
        minio::status
    else
        # Basic status
        log::header "MinIO Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "minio"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=minio" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start MinIO
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start MinIO"
        return 0
    fi
    
    if command -v minio::start &>/dev/null; then
        minio::start
    elif command -v minio::docker::start &>/dev/null; then
        minio::docker::start
    else
        docker start minio || log::error "Failed to start MinIO"
    fi
}

# Stop MinIO
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop MinIO"
        return 0
    fi
    
    if command -v minio::stop &>/dev/null; then
        minio::stop
    elif command -v minio::docker::stop &>/dev/null; then
        minio::docker::stop
    else
        docker stop minio || log::error "Failed to stop MinIO"
    fi
}

# Install MinIO
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install MinIO"
        return 0
    fi
    
    if command -v minio::install &>/dev/null; then
        minio::install
    else
        log::error "minio::install not available"
        return 1
    fi
}

# Uninstall MinIO
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove MinIO and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall MinIO"
        return 0
    fi
    
    if command -v minio::uninstall &>/dev/null; then
        minio::uninstall
    else
        docker stop minio 2>/dev/null || true
        docker rm minio 2>/dev/null || true
        log::success "MinIO uninstalled"
    fi
}

# Get credentials for n8n integration
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    # Parse arguments
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "minio"; return 0; }
        return 1
    fi
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "$MINIO_CONTAINER_NAME")
    
    # Build connections array
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # MinIO S3-compatible connection
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "$MINIO_PORT" \
            --argjson ssl false \
            --arg region "$MINIO_REGION" \
            '{
                host: $host,
                port: $port,
                ssl: $ssl,
                region: $region
            }')
        
        local auth_obj
        auth_obj=$(jq -n \
            --arg access_key "$MINIO_ROOT_USER" \
            --arg secret_key "$MINIO_ROOT_PASSWORD" \
            '{
                access_key: $access_key,
                secret_key: $secret_key
            }')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "MinIO S3-compatible object storage" \
            --argjson buckets "$(printf '%s\n' "${MINIO_DEFAULT_BUCKETS[@]}" | jq -R . | jq -s .)" \
            --arg console_url "$MINIO_CONSOLE_URL" \
            '{
                description: $description,
                default_buckets: $buckets,
                console_url: $console_url
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "MinIO S3 Storage" \
            "s3" \
            "$connection_obj" \
            "$auth_obj" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "minio" "$status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

################################################################################
# MinIO-specific commands (if functions exist)
################################################################################

# List buckets
minio_list_buckets() {
    if command -v minio::api::list_buckets &>/dev/null; then
        minio::api::list_buckets
    elif command -v minio::buckets::list &>/dev/null; then
        minio::buckets::list
    else
        # Try using mc command directly
        if command -v minio::api::mc &>/dev/null; then
            minio::api::configure_mc >/dev/null 2>&1 || true
            minio::api::mc ls local
        else
            log::error "Bucket listing not available"
            return 1
        fi
    fi
}

# Create bucket
minio_create_bucket() {
    local bucket_name="${1:-}"
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name required"
        echo "Usage: resource-minio create-bucket <name>"
        return 1
    fi
    
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would create bucket: $bucket_name"
        return 0
    fi
    
    if command -v minio::api::create_bucket &>/dev/null; then
        minio::api::create_bucket "$bucket_name"
    elif command -v minio::buckets::create &>/dev/null; then
        minio::buckets::create "$bucket_name"
    else
        log::error "Bucket creation not available"
        return 1
    fi
}

# Delete bucket
minio_delete_bucket() {
    local bucket_name="${1:-}"
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name required"
        echo "Usage: resource-minio delete-bucket <name>"
        return 1
    fi
    
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will delete bucket '$bucket_name' and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would delete bucket: $bucket_name"
        return 0
    fi
    
    if command -v minio::api::delete_bucket &>/dev/null; then
        minio::api::delete_bucket "$bucket_name"
    elif command -v minio::buckets::delete &>/dev/null; then
        minio::buckets::delete "$bucket_name"
    else
        log::error "Bucket deletion not available"
        return 1
    fi
}

# Configure MinIO client
minio_configure() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would configure MinIO client"
        return 0
    fi
    
    if command -v minio::api::configure_mc &>/dev/null; then
        minio::api::configure_mc
    else
        log::error "MinIO client configuration not available"
        return 1
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 MinIO Resource CLI

USAGE:
    resource-minio <command> [options]

CORE COMMANDS:
    inject <file>       Inject bucket config/data into MinIO
    validate            Validate MinIO configuration
    status              Show MinIO status
    start               Start MinIO container
    stop                Stop MinIO container
    install             Install MinIO
    uninstall           Uninstall MinIO (requires --force)
    credentials         Get connection credentials for n8n integration
    
MINIO COMMANDS:
    list-buckets        List all buckets
    create-bucket <name> Create a new bucket
    delete-bucket <name> Delete a bucket (requires --force)
    configure           Configure MinIO client

OPTIONS:
    --verbose, -v       Show detailed output
    --dry-run           Preview actions without executing
    --force             Force operation (skip confirmations)

EXAMPLES:
    resource-minio status
    resource-minio list-buckets
    resource-minio create-bucket my-bucket
    resource-minio delete-bucket old-bucket --force
    resource-minio inject shared:initialization/storage/minio/buckets.json

For more information: https://docs.vrooli.com/resources/minio
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first
    local remaining_args
    remaining_args=$(resource_cli::parse_options "$@")
    set -- $remaining_args
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall|credentials)
            resource_cli::$command "$@"
            ;;
            
        # MinIO-specific commands
        list-buckets)
            minio_list_buckets "$@"
            ;;
        create-bucket)
            minio_create_bucket "$@"
            ;;
        delete-bucket)
            minio_delete_bucket "$@"
            ;;
        configure)
            minio_configure "$@"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi