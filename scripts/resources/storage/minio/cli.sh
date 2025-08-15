#!/usr/bin/env bash
################################################################################
# MinIO Resource CLI
# 
# Lightweight CLI interface for MinIO using the CLI Command Framework
#
# Usage:
#   resource-minio <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (handle symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
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
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/resources/lib/cli-command-framework.sh"

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

# Initialize CLI framework
cli::init "minio" "MinIO S3-compatible object storage management"

# Override help to provide MinIO-specific examples
cli::register_command "help" "Show this help message with MinIO examples" "resource_cli::show_help"

# Register additional MinIO-specific commands
cli::register_command "inject" "Inject bucket config/data into MinIO" "resource_cli::inject" "modifies-system"
cli::register_command "list-buckets" "List all buckets" "resource_cli::list_buckets"
cli::register_command "create-bucket" "Create a new bucket" "resource_cli::create_bucket" "modifies-system"
cli::register_command "delete-bucket" "Delete a bucket (requires --force)" "resource_cli::delete_bucket" "modifies-system"
cli::register_command "configure" "Configure MinIO client" "resource_cli::configure" "modifies-system"
cli::register_command "credentials" "Show n8n credentials for MinIO" "resource_cli::credentials"
cli::register_command "uninstall" "Uninstall MinIO (requires --force)" "resource_cli::uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject bucket configuration or data into MinIO
resource_cli::inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-minio inject <file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-minio inject buckets.json"
        echo "  resource-minio inject shared:initialization/storage/minio/config.json"
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
        local container_name="${MINIO_CONTAINER_NAME:-minio}"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name" || {
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
        local container_name="${MINIO_CONTAINER_NAME:-minio}"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=$container_name" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start MinIO
resource_cli::start() {
    if command -v minio::start &>/dev/null; then
        minio::start
    elif command -v minio::docker::start &>/dev/null; then
        minio::docker::start
    else
        local container_name="${MINIO_CONTAINER_NAME:-minio}"
        docker start "$container_name" || log::error "Failed to start MinIO"
    fi
}

# Stop MinIO
resource_cli::stop() {
    if command -v minio::stop &>/dev/null; then
        minio::stop
    elif command -v minio::docker::stop &>/dev/null; then
        minio::docker::stop
    else
        local container_name="${MINIO_CONTAINER_NAME:-minio}"
        docker stop "$container_name" || log::error "Failed to stop MinIO"
    fi
}

# Install MinIO
resource_cli::install() {
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
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove MinIO and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v minio::uninstall &>/dev/null; then
        minio::uninstall
    else
        local container_name="${MINIO_CONTAINER_NAME:-minio}"
        docker stop "$container_name" 2>/dev/null || true
        docker rm "$container_name" 2>/dev/null || true
        log::success "MinIO uninstalled"
    fi
}

# Show credentials for n8n integration
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "minio"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "${MINIO_CONTAINER_NAME:-minio}")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # MinIO S3-compatible connection
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${MINIO_PORT:-9000}" \
            --argjson ssl false \
            --arg region "${MINIO_REGION:-us-east-1}" \
            '{
                host: $host,
                port: $port,
                ssl: $ssl,
                region: $region
            }')
        
        local auth_obj
        auth_obj=$(jq -n \
            --arg access_key "${MINIO_ROOT_USER:-minioadmin}" \
            --arg secret_key "${MINIO_ROOT_PASSWORD:-minioadmin}" \
            '{
                access_key: $access_key,
                secret_key: $secret_key
            }')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "MinIO S3-compatible object storage" \
            --arg console_url "${MINIO_CONSOLE_URL:-http://localhost:9001}" \
            '{
                description: $description,
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
    
    local response
    response=$(credentials::build_response "minio" "$status" "$connections_array")
    credentials::format_output "$response"
}

# List buckets
resource_cli::list_buckets() {
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
resource_cli::create_bucket() {
    local bucket_name="${1:-}"
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name required"
        echo "Usage: resource-minio create-bucket <name>"
        echo ""
        echo "Examples:"
        echo "  resource-minio create-bucket my-data"
        echo "  resource-minio create-bucket backups"
        echo "  resource-minio create-bucket uploads"
        return 1
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
resource_cli::delete_bucket() {
    local bucket_name="${1:-}"
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name required"
        echo "Usage: resource-minio delete-bucket <name> --force"
        echo ""
        echo "Examples:"
        echo "  resource-minio delete-bucket old-data --force"
        echo "  resource-minio delete-bucket temp-bucket --force"
        return 1
    fi
    
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will delete bucket '$bucket_name' and all its data. Use --force to confirm."
        return 1
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
resource_cli::configure() {
    if command -v minio::api::configure_mc &>/dev/null; then
        minio::api::configure_mc
    else
        log::error "MinIO client configuration not available"
        return 1
    fi
}

# Custom help function with MinIO-specific examples
resource_cli::show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add MinIO-specific examples
    echo ""
    echo "🗄️  MinIO S3-Compatible Object Storage Examples:"
    echo ""
    echo "Bucket Management:"
    echo "  resource-minio list-buckets                     # List all buckets"
    echo "  resource-minio create-bucket my-data           # Create new bucket"
    echo "  resource-minio create-bucket uploads           # Create uploads bucket"
    echo "  resource-minio delete-bucket old-data --force  # Delete bucket"
    echo ""
    echo "Configuration:"
    echo "  resource-minio configure                        # Setup MinIO client"
    echo "  resource-minio inject shared:init/minio/buckets.json  # Import config"
    echo ""
    echo "Management:"
    echo "  resource-minio status                           # Check service status"
    echo "  resource-minio credentials                      # Get S3 credentials"
    echo ""
    echo "S3 Features:"
    echo "  • AWS S3-compatible API"
    echo "  • High-performance object storage"
    echo "  • Bucket versioning and lifecycle policies"
    echo "  • Web console for management"
    echo ""
    echo "Default Ports: API 9000, Console 9001"
    echo "Web Console: http://localhost:9001"
    echo "Default Credentials: minioadmin / minioadmin"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi