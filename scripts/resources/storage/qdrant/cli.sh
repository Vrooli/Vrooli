#!/usr/bin/env bash
################################################################################
# Qdrant Resource CLI
# 
# Lightweight CLI interface for Qdrant that delegates to existing lib functions.
#
# Usage:
#   resource-qdrant <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory and resolve VROOLI_ROOT
QDRANT_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Handle both direct execution and symlink execution
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this script is a symlink, resolve to the original location
    REAL_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    QDRANT_CLI_DIR="$(cd "$(dirname "$REAL_SCRIPT")" && pwd)"
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

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

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

# Initialize with resource name
resource_cli::init "qdrant"

################################################################################
# Delegate to existing qdrant functions
################################################################################

# Inject data into qdrant
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-qdrant inject <file.json>"
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
    if command -v qdrant::inject &>/dev/null; then
        INJECTION_CONFIG="$(cat "$file")" qdrant::inject
    else
        log::error "qdrant injection functions not available"
        return 1
    fi
}

# Validate qdrant configuration
resource_cli::validate() {
    if command -v qdrant::health &>/dev/null; then
        qdrant::health
    elif command -v qdrant::check_basic_health &>/dev/null; then
        qdrant::check_basic_health
    else
        # Basic validation
        log::header "Validating Qdrant"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "qdrant" || {
            log::error "Qdrant container not running"
            return 1
        }
        log::success "Qdrant is running"
    fi
}

# Show qdrant status
resource_cli::status() {
    if command -v qdrant::status::check &>/dev/null; then
        qdrant::status::check
    else
        # Basic status
        log::header "Qdrant Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "qdrant"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=qdrant" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start qdrant
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start qdrant"
        return 0
    fi
    
    if command -v qdrant::docker::start &>/dev/null; then
        qdrant::docker::start
    else
        docker start qdrant || log::error "Failed to start qdrant"
    fi
}

# Stop qdrant
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop qdrant"
        return 0
    fi
    
    if command -v qdrant::docker::stop &>/dev/null; then
        qdrant::docker::stop
    else
        docker stop qdrant || log::error "Failed to stop qdrant"
    fi
}

# Install qdrant
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install qdrant"
        return 0
    fi
    
    if command -v qdrant::install &>/dev/null; then
        qdrant::install
    else
        log::error "qdrant::install not available"
        return 1
    fi
}

# Uninstall qdrant
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove qdrant and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall qdrant"
        return 0
    fi
    
    if command -v qdrant::uninstall &>/dev/null; then
        qdrant::uninstall false  # remove data
    else
        docker stop qdrant 2>/dev/null || true
        docker rm qdrant 2>/dev/null || true
        log::success "qdrant uninstalled"
    fi
}

################################################################################
# Qdrant-specific commands (if functions exist)
################################################################################

# List collections
qdrant_list_collections() {
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
qdrant_create_collection() {
    local name="${1:-}"
    local vector_size="${2:-1536}"
    local distance="${3:-Cosine}"
    
    if [[ -z "$name" ]]; then
        log::error "Collection name required"
        echo "Usage: resource-qdrant create-collection <name> [vector_size] [distance]"
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
qdrant_delete_collection() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Collection name required"
        echo "Usage: resource-qdrant delete-collection <name>"
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
qdrant_collection_info() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Collection name required"
        echo "Usage: resource-qdrant collection-info <name>"
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
qdrant_create_backup() {
    local name="${1:-backup-$(date +%Y%m%d-%H%M%S)}"
    
    if command -v qdrant::backup::create &>/dev/null; then
        qdrant::backup::create "$name"
    else
        log::error "Backup functionality not available"
        return 1
    fi
}

# List backups
qdrant_list_backups() {
    if command -v qdrant::backup::list &>/dev/null; then
        qdrant::backup::list
    else
        log::error "Backup listing not available"
        return 1
    fi
}

# Get credentials for n8n integration
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    # Parse arguments
    credentials::parse_args "$@"
    local parse_result=$?
    if [[ $parse_result -eq 2 ]]; then
        credentials::show_help "qdrant"
        return 0
    elif [[ $parse_result -ne 0 ]]; then
        return 1
    fi
    
    # Get resource status by checking Docker container
    local status
    status=$(credentials::get_resource_status "$QDRANT_CONTAINER_NAME")
    
    # Build connections array manually for Qdrant (bypassing the helper to avoid jq issues)
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
                --arg port "${QDRANT_PORT}" \
                --arg path "/collections" \
                --arg description "Qdrant vector database REST API" \
                --arg version "${QDRANT_VERSION}" \
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
                --arg n8n_credential_type "httpHeaderAuth" \
                --arg host "localhost" \
                --arg port "${QDRANT_PORT}" \
                --arg path "/collections" \
                --arg description "Qdrant vector database REST API" \
                --arg version "${QDRANT_VERSION}" \
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
    
    # Build and validate response
    local response
    response=$(credentials::build_response "qdrant" "$status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 Qdrant Resource CLI

USAGE:
    resource-qdrant <command> [options]

CORE COMMANDS:
    inject <file>               Inject data into qdrant
    validate                    Validate qdrant configuration  
    status                      Show qdrant status
    start                       Start qdrant container
    stop                        Stop qdrant container
    install                     Install qdrant
    uninstall                   Uninstall qdrant (requires --force)
    credentials                 Get connection credentials for n8n integration
    
QDRANT COMMANDS:
    list-collections            List all collections
    create-collection <name>    Create a new collection
    delete-collection <name>    Delete a collection
    collection-info <name>      Show collection information
    create-backup [name]        Create a backup snapshot
    list-backups                List all backups

OPTIONS:
    --verbose, -v               Show detailed output
    --dry-run                   Preview actions without executing
    --force                     Force operation (skip confirmations)

EXAMPLES:
    resource-qdrant status
    resource-qdrant credentials --format pretty
    resource-qdrant list-collections
    resource-qdrant create-collection my-vectors 384 Cosine
    resource-qdrant inject shared:initialization/storage/qdrant/collections.json
    resource-qdrant create-backup my-backup

For more information: https://docs.vrooli.com/resources/qdrant
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
            
        # Qdrant-specific commands
        list-collections)
            qdrant_list_collections "$@"
            ;;
        create-collection)
            qdrant_create_collection "$@"
            ;;
        delete-collection)
            qdrant_delete_collection "$@"
            ;;
        collection-info)
            qdrant_collection_info "$@"
            ;;
        create-backup)
            qdrant_create_backup "$@"
            ;;
        list-backups)
            qdrant_list_backups "$@"
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