#!/usr/bin/env bash
################################################################################
# Qdrant Resource CLI
# 
# Lightweight CLI interface for Qdrant using the CLI Command Framework
#
# Usage:
#   resource-qdrant <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (handle symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    REAL_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    QDRANT_CLI_DIR="$(cd "$(dirname "$REAL_SCRIPT")" && pwd)"
else
    QDRANT_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$QDRANT_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$QDRANT_CLI_DIR"
export QDRANT_SCRIPT_DIR="$QDRANT_CLI_DIR"  # For compatibility with existing libs

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

# Source qdrant configuration
# shellcheck disable=SC1091
source "${QDRANT_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
qdrant::export_config 2>/dev/null || true

# Source qdrant libraries
for lib in core health collections backup inject; do
    lib_file="${QDRANT_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "qdrant" "Qdrant vector database management"

# Override help to provide Qdrant-specific examples
cli::register_command "help" "Show this help message with Qdrant examples" "resource_cli::show_help"

# Register additional Qdrant-specific commands
cli::register_command "inject" "Inject data into Qdrant" "resource_cli::inject" "modifies-system"
cli::register_command "list-collections" "List all collections" "resource_cli::list_collections"
cli::register_command "create-collection" "Create a new collection" "resource_cli::create_collection" "modifies-system"
cli::register_command "delete-collection" "Delete a collection" "resource_cli::delete_collection" "modifies-system"
cli::register_command "collection-info" "Show collection information" "resource_cli::collection_info"
cli::register_command "create-backup" "Create a backup snapshot" "resource_cli::create_backup" "modifies-system"
cli::register_command "list-backups" "List all backups" "resource_cli::list_backups"
cli::register_command "credentials" "Show n8n credentials for Qdrant" "resource_cli::credentials"
cli::register_command "uninstall" "Uninstall Qdrant (requires --force)" "resource_cli::uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject data into Qdrant
resource_cli::inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-qdrant inject <file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-qdrant inject vectors.json"
        echo "  resource-qdrant inject shared:initialization/storage/qdrant/collections.json"
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
    if command -v qdrant::inject &>/dev/null; then
        INJECTION_CONFIG="$(cat "$file")" qdrant::inject
    else
        log::error "qdrant injection functions not available"
        return 1
    fi
}

# Validate Qdrant configuration
resource_cli::validate() {
    if command -v qdrant::health &>/dev/null; then
        qdrant::health
    elif command -v qdrant::check_basic_health &>/dev/null; then
        qdrant::check_basic_health
    else
        # Basic validation
        log::header "Validating Qdrant"
        local container_name="${QDRANT_CONTAINER_NAME:-qdrant}"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name" || {
            log::error "Qdrant container not running"
            return 1
        }
        log::success "Qdrant is running"
    fi
}

# Show Qdrant status
resource_cli::status() {
    if command -v qdrant::status::check &>/dev/null; then
        qdrant::status::check
    else
        # Basic status
        log::header "Qdrant Status"
        local container_name="${QDRANT_CONTAINER_NAME:-qdrant}"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=$container_name" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start Qdrant
resource_cli::start() {
    if command -v qdrant::docker::start &>/dev/null; then
        qdrant::docker::start
    else
        local container_name="${QDRANT_CONTAINER_NAME:-qdrant}"
        docker start "$container_name" || log::error "Failed to start Qdrant"
    fi
}

# Stop Qdrant
resource_cli::stop() {
    if command -v qdrant::docker::stop &>/dev/null; then
        qdrant::docker::stop
    else
        local container_name="${QDRANT_CONTAINER_NAME:-qdrant}"
        docker stop "$container_name" || log::error "Failed to stop Qdrant"
    fi
}

# Install Qdrant
resource_cli::install() {
    if command -v qdrant::install &>/dev/null; then
        qdrant::install
    else
        log::error "qdrant::install not available"
        return 1
    fi
}

# Uninstall Qdrant
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Qdrant and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v qdrant::uninstall &>/dev/null; then
        qdrant::uninstall false  # remove data
    else
        local container_name="${QDRANT_CONTAINER_NAME:-qdrant}"
        docker stop "$container_name" 2>/dev/null || true
        docker rm "$container_name" 2>/dev/null || true
        log::success "Qdrant uninstalled"
    fi
}

# Show credentials for n8n integration
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "qdrant"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "${QDRANT_CONTAINER_NAME:-qdrant}")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Build connection JSON manually
        local connection_json
        if [[ -n "${QDRANT_API_KEY:-}" ]]; then
            # With API key authentication
            connection_json=$(jq -n \
                --arg id "api" \
                --arg name "Qdrant REST API" \
                --arg n8n_credential_type "httpHeaderAuth" \
                --arg host "localhost" \
                --arg port "${QDRANT_PORT:-6333}" \
                --arg path "/collections" \
                --arg description "Qdrant vector database REST API" \
                --arg version "${QDRANT_VERSION:-latest}" \
                --arg header_name "api-key" \
                --arg header_value "${QDRANT_API_KEY}" \
                '{
                    id: $id,
                    name: $name,
                    n8n_credential_type: $n8n_credential_type,
                    connection: {
                        host: $host,
                        port: ($port | tonumber),
                        path: $path,
                        ssl: false
                    },
                    auth: {
                        header_name: $header_name,
                        header_value: $header_value
                    },
                    metadata: {
                        description: $description,
                        capabilities: ["vectors", "search", "clustering", "embeddings"],
                        version: $version
                    }
                }')
        else
            # Without authentication
            connection_json=$(jq -n \
                --arg id "api" \
                --arg name "Qdrant REST API" \
                --arg n8n_credential_type "httpRequest" \
                --arg host "localhost" \
                --arg port "${QDRANT_PORT:-6333}" \
                --arg path "/collections" \
                --arg description "Qdrant vector database REST API" \
                --arg version "${QDRANT_VERSION:-latest}" \
                '{
                    id: $id,
                    name: $name,
                    n8n_credential_type: $n8n_credential_type,
                    connection: {
                        host: $host,
                        port: ($port | tonumber),
                        path: $path,
                        ssl: false
                    },
                    metadata: {
                        description: $description,
                        capabilities: ["vectors", "search", "clustering", "embeddings"],
                        version: $version
                    }
                }')
        fi
        
        connections_array=$(echo "$connection_json" | jq -s '.')
    fi
    
    local response
    response=$(credentials::build_response "qdrant" "$status" "$connections_array")
    credentials::format_output "$response"
}

# List collections
resource_cli::list_collections() {
    if command -v qdrant::collections::list &>/dev/null; then
        qdrant::collections::list
    elif command -v qdrant::api::list_collections &>/dev/null; then
        qdrant::api::list_collections
    else
        log::error "Collection listing not available"
        return 1
    fi
}

# Create collection
resource_cli::create_collection() {
    local name="${1:-}"
    local vector_size="${2:-1536}"
    local distance="${3:-Cosine}"
    
    if [[ -z "$name" ]]; then
        log::error "Collection name required"
        echo "Usage: resource-qdrant create-collection <name> [vector_size] [distance]"
        echo ""
        echo "Examples:"
        echo "  resource-qdrant create-collection my-vectors              # 1536-dim, Cosine"
        echo "  resource-qdrant create-collection embeddings 384          # 384-dim, Cosine"
        echo "  resource-qdrant create-collection text-vectors 768 Dot    # 768-dim, Dot product"
        echo ""
        echo "Distance options: Cosine, Dot, Euclid"
        return 1
    fi
    
    if command -v qdrant::collections::create &>/dev/null; then
        qdrant::collections::create "$name" "$vector_size" "$distance"
    elif command -v qdrant::api::create_collection &>/dev/null; then
        qdrant::api::create_collection "$name" "$vector_size" "$distance"
    else
        log::error "Collection creation not available"
        return 1
    fi
}

# Delete collection
resource_cli::delete_collection() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Collection name required"
        echo "Usage: resource-qdrant delete-collection <name>"
        echo ""
        echo "Examples:"
        echo "  resource-qdrant delete-collection old-vectors"
        echo "  resource-qdrant delete-collection test-embeddings"
        return 1
    fi
    
    if command -v qdrant::collections::delete &>/dev/null; then
        qdrant::collections::delete "$name" "yes"
    elif command -v qdrant::api::delete_collection &>/dev/null; then
        qdrant::api::delete_collection "$name"
    else
        log::error "Collection deletion not available"
        return 1
    fi
}

# Get collection info
resource_cli::collection_info() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Collection name required"
        echo "Usage: resource-qdrant collection-info <name>"
        echo ""
        echo "Examples:"
        echo "  resource-qdrant collection-info my-vectors"
        echo "  resource-qdrant collection-info embeddings"
        return 1
    fi
    
    if command -v qdrant::collections::info &>/dev/null; then
        qdrant::collections::info "$name"
    elif command -v qdrant::api::get_collection &>/dev/null; then
        qdrant::api::get_collection "$name"
    else
        log::error "Collection info not available"
        return 1
    fi
}

# Create backup
resource_cli::create_backup() {
    local name="${1:-backup-$(date +%Y%m%d-%H%M%S)}"
    
    if command -v qdrant::backup::create &>/dev/null; then
        qdrant::backup::create "$name"
    else
        log::error "Backup functionality not available"
        return 1
    fi
}

# List backups
resource_cli::list_backups() {
    if command -v qdrant::backup::list &>/dev/null; then
        qdrant::backup::list
    else
        log::error "Backup listing not available"
        return 1
    fi
}

# Custom help function with Qdrant-specific examples
resource_cli::show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add Qdrant-specific examples
    echo ""
    echo "🔍 Qdrant Vector Database Examples:"
    echo ""
    echo "Collection Management:"
    echo "  resource-qdrant list-collections                          # List all collections"
    echo "  resource-qdrant create-collection my-vectors              # Create collection (1536-dim)"
    echo "  resource-qdrant create-collection embeddings 384          # Create 384-dim collection"
    echo "  resource-qdrant create-collection text-vectors 768 Dot    # Create with Dot distance"
    echo "  resource-qdrant collection-info my-vectors                # Show collection details"
    echo "  resource-qdrant delete-collection old-vectors             # Delete collection"
    echo ""
    echo "Data Management:"
    echo "  resource-qdrant inject vectors.json                       # Import vector data"
    echo "  resource-qdrant inject shared:init/qdrant/collections.json # Import shared config"
    echo ""
    echo "Backup & Monitoring:"
    echo "  resource-qdrant create-backup my-backup                   # Create named backup"
    echo "  resource-qdrant create-backup                             # Create timestamped backup"
    echo "  resource-qdrant list-backups                              # List all backups"
    echo "  resource-qdrant status                                    # Check service status"
    echo ""
    echo "Integration:"
    echo "  resource-qdrant credentials --format pretty               # Show connection details"
    echo ""
    echo "Vector Features:"
    echo "  • High-performance similarity search"
    echo "  • Multiple distance metrics (Cosine, Dot, Euclidean)"
    echo "  • Payload filtering and hybrid search"
    echo "  • Clustering and recommendation systems"
    echo ""
    echo "Default Port: 6333"
    echo "Web UI: http://localhost:6333/dashboard"
    echo "Common Vector Sizes: 384 (sentence-transformers), 768 (BERT), 1536 (OpenAI)"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi