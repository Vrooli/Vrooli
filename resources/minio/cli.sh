#!/usr/bin/env bash
################################################################################
# MinIO Resource CLI - v2.0 Universal Contract Compliant
# 
# S3-compatible object storage server with high-performance and scalability
#
# Usage:
#   resource-minio <command> [options]
#   resource-minio <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    MINIO_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${MINIO_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
MINIO_CLI_DIR="${APP_ROOT}/resources/minio"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${MINIO_CLI_DIR}/config/defaults.sh"

# Export MinIO configuration
minio::export_config 2>/dev/null || true

# Source MinIO libraries
for lib in common docker install status api buckets inject; do
    lib_file="${MINIO_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "minio" "MinIO S3-compatible object storage management" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="minio::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="minio::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="minio::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="minio::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="minio::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="minio::status_wrapper"

# Content handlers for S3 object storage business functionality
CLI_COMMAND_HANDLERS["content::add"]="minio::content::add_bucket"
CLI_COMMAND_HANDLERS["content::list"]="minio::content::list_buckets"
CLI_COMMAND_HANDLERS["content::get"]="minio::content::get_object"
CLI_COMMAND_HANDLERS["content::remove"]="minio::content::remove_bucket"
CLI_COMMAND_HANDLERS["content::execute"]="minio::content::inject_data"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed MinIO status" "minio::status_wrapper"
cli::register_command "logs" "Show MinIO logs" "minio::logs"

# ==============================================================================
# OPTIONAL RESOURCE-SPECIFIC COMMANDS
# ==============================================================================
cli::register_command "credentials" "Show MinIO credentials for integration" "minio::credentials"

# Add custom content subcommands for MinIO-specific operations
cli::register_subcommand "content" "upload" "Upload file to bucket" "minio::content::upload_file" "modifies-system"
cli::register_subcommand "content" "download" "Download file from bucket" "minio::content::download_file"
cli::register_subcommand "content" "configure" "Configure MinIO client" "minio::content::configure"

# ==============================================================================
# CONTENT COMMAND IMPLEMENTATIONS
# ==============================================================================

# Create bucket (content::add)
minio::content::add_bucket() {
    local bucket_name="${1:-}"
    local policy="${2:-private}"
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name required"
        echo "Usage: resource-minio content add <bucket-name> [policy]"
        echo ""
        echo "Examples:"
        echo "  resource-minio content add my-data"
        echo "  resource-minio content add public-images public-read"
        echo "  resource-minio content add uploads private"
        return 1
    fi
    
    if command -v minio::buckets::create_custom &>/dev/null; then
        minio::buckets::create_custom "$bucket_name" "$policy"
    elif command -v minio::api::create_bucket &>/dev/null; then
        minio::api::create_bucket "$bucket_name"
    else
        log::error "Bucket creation functionality not available"
        return 1
    fi
}

# List buckets and objects (content::list)
minio::content::list_buckets() {
    if command -v minio::buckets::show_stats &>/dev/null; then
        minio::buckets::show_stats
    elif command -v minio::api::list_buckets &>/dev/null; then
        minio::api::list_buckets
    else
        log::error "Bucket listing functionality not available"
        return 1
    fi
}

# Get/download object (content::get)
minio::content::get_object() {
    local bucket="${1:-}"
    local object="${2:-}"
    local local_file="${3:-}"
    
    if [[ -z "$bucket" || -z "$object" ]]; then
        log::error "Bucket and object name required"
        echo "Usage: resource-minio content get <bucket> <object> [local-file]"
        echo ""
        echo "Examples:"
        echo "  resource-minio content get my-data file.txt"
        echo "  resource-minio content get images logo.png ./downloaded-logo.png"
        return 1
    fi
    
    if command -v minio::api::download_file &>/dev/null; then
        minio::api::download_file "$bucket" "$object" "$local_file"
    else
        log::error "File download functionality not available"
        return 1
    fi
}

# Remove bucket (content::remove)
minio::content::remove_bucket() {
    local bucket_name="${1:-}"
    
    if [[ -z "$bucket_name" ]]; then
        log::error "Bucket name required"
        echo "Usage: resource-minio content remove <bucket-name> --force"
        echo ""
        echo "Examples:"
        echo "  resource-minio content remove old-data --force"
        echo "  resource-minio content remove temp-bucket --force"
        return 1
    fi
    
    if command -v minio::buckets::remove &>/dev/null; then
        minio::buckets::remove "$bucket_name" "${FORCE:-no}"
    else
        log::error "Bucket removal functionality not available"
        return 1
    fi
}

# Execute data injection (content::execute)
minio::content::inject_data() {
    local config_file="${1:-}"
    
    if [[ -z "$config_file" ]]; then
        log::error "Configuration file required"
        echo "Usage: resource-minio content execute <config-file>"
        echo ""
        echo "Examples:"
        echo "  resource-minio content execute buckets.json"
        echo "  resource-minio content execute shared:initialization/storage/minio/config.json"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$config_file" == shared:* ]]; then
        config_file="${var_ROOT_DIR}/${config_file#shared:}"
    fi
    
    if [[ ! -f "$config_file" ]]; then
        log::error "Configuration file not found: $config_file"
        return 1
    fi
    
    if command -v minio::inject &>/dev/null; then
        minio::inject "$config_file"
    else
        log::error "Data injection functionality not available"
        return 1
    fi
}

# Upload file to bucket
minio::content::upload_file() {
    local bucket="${1:-}"
    local file_path="${2:-}"
    local object_name="${3:-}"
    
    if [[ -z "$bucket" || -z "$file_path" ]]; then
        log::error "Bucket and file path required"
        echo "Usage: resource-minio content upload <bucket> <file-path> [object-name]"
        echo ""
        echo "Examples:"
        echo "  resource-minio content upload my-data /path/to/file.txt"
        echo "  resource-minio content upload images logo.png uploads/logo.png"
        return 1
    fi
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    # Use filename as object name if not specified
    if [[ -z "$object_name" ]]; then
        object_name=$(basename "$file_path")
    fi
    
    if command -v minio::api::upload_file &>/dev/null; then
        minio::api::upload_file "$bucket" "$file_path" "$object_name"
    else
        log::error "File upload functionality not available"
        return 1
    fi
}

# Download file from bucket
minio::content::download_file() {
    minio::content::get_object "$@"
}

# Configure MinIO client
minio::content::configure() {
    if command -v minio::api::configure_mc &>/dev/null; then
        minio::api::configure_mc
    else
        log::error "MinIO client configuration not available"
        return 1
    fi
}

# ==============================================================================
# STATUS IMPLEMENTATION  
# ==============================================================================
minio::status_wrapper() {
    # Basic status check
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    echo "MinIO Status"
    echo "==========="
    
    if docker ps --filter "name=$container_name" --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"; then
        echo "✅ MinIO is running"
        docker ps --filter "name=$container_name" --format "table {{.Status}}\t{{.Ports}}" 2>/dev/null | tail -n 1
    elif docker ps -a --filter "name=$container_name" --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"; then
        echo "⏸️  MinIO is installed but not running"
    else
        echo "❌ MinIO is not installed"
    fi
}

# ==============================================================================
# LOGS IMPLEMENTATION  
# ==============================================================================
minio::logs() {
    if command -v minio::common::show_logs &>/dev/null; then
        minio::common::show_logs "$@"
    else
        # Fallback to basic docker logs
        local container_name="${MINIO_CONTAINER_NAME:-minio}"
        local lines="${1:-50}"
        docker logs --tail "$lines" "$container_name" 2>/dev/null || {
            log::error "Failed to get MinIO logs"
            return 1
        }
    fi
}

# ==============================================================================
# CREDENTIALS IMPLEMENTATION
# ==============================================================================
minio::credentials() {
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
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
            --arg port "${MINIO_PORT:-9000}" \
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

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi