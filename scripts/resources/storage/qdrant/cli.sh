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

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    QDRANT_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    QDRANT_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
QDRANT_CLI_DIR="$(cd "$(dirname "$QDRANT_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${QDRANT_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source qdrant configuration
# shellcheck disable=SC1091
source "${QDRANT_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
qdrant::export_config 2>/dev/null || true

# Source qdrant libraries
for lib in core health collections backup inject content status models embeddings search credentials; do
    lib_file="${QDRANT_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "qdrant" "Qdrant vector database management"

# Override help to provide Qdrant-specific examples
cli::register_command "help" "Show this help message with Qdrant examples" "qdrant_show_help"

# Register additional Qdrant-specific commands
cli::register_command "inject" "[DEPRECATED] Inject data into Qdrant (use 'content add' instead)" "qdrant_inject" "modifies-system"

# Content management commands (replacing inject)
cli::register_command "content" "Manage Qdrant content (add/list/get/remove/execute)" "qdrant_content_dispatch" "modifies-system"

# Collection commands
cli::register_command "collections" "Manage collections (list/create/info/delete/search)" "qdrant_collections_dispatch"

# Embedding commands
cli::register_command "embed" "Generate embeddings from text" "qdrant_embed"
cli::register_command "embed-info" "Show embedding model information" "qdrant_embed_info"

# Model commands
cli::register_command "models" "Manage embedding models (list/info)" "qdrant_models_dispatch"

# Backup commands
cli::register_command "backup" "Manage backups (create/list)" "qdrant_backup_dispatch"

# Other commands
cli::register_command "credentials" "Show connection details for Qdrant" "qdrant_credentials"
cli::register_command "uninstall" "Uninstall Qdrant (requires --force)" "qdrant_uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject data into Qdrant (DEPRECATED - delegates to content add)
qdrant_inject() {
    local file="${1:-}"
    
    # Handle help requests
    if [[ "$file" == "--help" || "$file" == "-h" || "$file" == "help" ]]; then
        echo "Usage: resource-qdrant inject <file.json>"
        echo ""
        echo "⚠️  DEPRECATED: Use 'resource-qdrant content add' instead"
        echo ""
        echo "Inject data from JSON file into Qdrant collections"
        echo ""
        echo "Arguments:"
        echo "  <file.json>    Path to JSON file containing injection data"
        echo ""
        echo "Examples:"
        echo "  resource-qdrant inject vectors.json"
        echo "  resource-qdrant inject shared:initialization/storage/qdrant/collections.json"
        echo ""
        echo "Recommended alternative:"
        echo "  resource-qdrant content add --file vectors.json"
        return 0
    fi
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-qdrant inject <file.json>"
        echo ""
        echo "⚠️  DEPRECATED: Use 'resource-qdrant content add' instead"
        echo ""
        echo "Examples:"
        echo "  resource-qdrant inject vectors.json"
        echo "  resource-qdrant inject shared:initialization/storage/qdrant/collections.json"
        return 1
    fi
    
    # Show deprecation warning but continue with operation
    log::warn "⚠️  DEPRECATED: 'inject' command is deprecated. Use 'resource-qdrant content add --file $file' instead"
    
    # Delegate to content add command with file parameter
    if command -v qdrant::content::add &>/dev/null; then
        qdrant::content::add --file "$file"
    else
        log::error "Content add function not available"
        return 1
    fi
}

# Validate Qdrant configuration
qdrant_validate() {
    if command -v qdrant::health &>/dev/null; then
        qdrant::health
    elif command -v qdrant::check_basic_health &>/dev/null; then
        qdrant::check_basic_health
    else
        log::error "Qdrant health functions not available"
        return 1
    fi
}

# Show Qdrant status
qdrant_status() {
    if command -v qdrant::status::check &>/dev/null; then
        qdrant::status::check
    else
        log::error "Qdrant status functions not available"
        return 1
    fi
}

# Start Qdrant
qdrant_start() {
    if command -v qdrant::docker::start &>/dev/null; then
        qdrant::docker::start
    else
        log::error "Qdrant docker functions not available"
        return 1
    fi
}

# Stop Qdrant
qdrant_stop() {
    if command -v qdrant::docker::stop &>/dev/null; then
        qdrant::docker::stop
    else
        log::error "Qdrant docker functions not available"
        return 1
    fi
}

# Install Qdrant
qdrant_install() {
    if command -v qdrant::install &>/dev/null; then
        qdrant::install
    else
        log::error "qdrant::install not available"
        return 1
    fi
}

# Uninstall Qdrant
qdrant_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Qdrant and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v qdrant::uninstall &>/dev/null; then
        qdrant::uninstall false  # remove data
    else
        log::error "Qdrant uninstall function not available"
        return 1
    fi
}

# Show credentials for n8n integration
qdrant_credentials() {
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "qdrant"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "${QDRANT_CONTAINER_NAME:-qdrant}")
    
    if command -v qdrant::credentials::show &>/dev/null; then
        qdrant::credentials::show "$status"
    else
        log::error "Qdrant credentials functions not available"
        return 1
    fi
}

# Content management dispatcher
qdrant_content_dispatch() {
    local subcommand="${1:-help}"
    shift || true
    
    case "$subcommand" in
        add)
            if command -v qdrant::content::add &>/dev/null; then
                qdrant::content::add "$@"
            else
                log::error "Content add function not available"
                return 1
            fi
            ;;
        list)
            if command -v qdrant::content::list &>/dev/null; then
                qdrant::content::list "$@"
            else
                log::error "Content list function not available"
                return 1
            fi
            ;;
        get)
            if command -v qdrant::content::get &>/dev/null; then
                qdrant::content::get "$@"
            else
                log::error "Content get function not available"
                return 1
            fi
            ;;
        remove)
            if command -v qdrant::content::remove &>/dev/null; then
                qdrant::content::remove "$@"
            else
                log::error "Content remove function not available"
                return 1
            fi
            ;;
        execute)
            if command -v qdrant::content::execute &>/dev/null; then
                qdrant::content::execute "$@"
            else
                log::error "Content execute function not available"
                return 1
            fi
            ;;
        help|--help|-h)
            echo "Usage: resource-qdrant content <subcommand> [options]"
            echo ""
            echo "Subcommands:"
            echo "  add       Add content to Qdrant"
            echo "  list      List stored content"
            echo "  get       Get specific content metadata"
            echo "  remove    Remove content"
            echo "  execute   Execute operations from file"
            echo ""
            echo "Examples:"
            echo "  resource-qdrant content add --file collections.json"
            echo "  resource-qdrant content add --file vectors.ndjson --name my-vectors"
            echo "  resource-qdrant content list"
            echo "  resource-qdrant content list --type collections"
            echo "  resource-qdrant content get --name my-vectors"
            echo "  resource-qdrant content remove --name old-data"
            echo "  resource-qdrant content execute --file operations.json"
            ;;
        *)
            log::error "Unknown content subcommand: $subcommand"
            echo "Use 'resource-qdrant content help' for available commands"
            return 1
            ;;
    esac
}

# Collections dispatcher for subcommands
qdrant_collections_dispatch() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        list)
            qdrant_list_collections "$@"
            ;;
        create)
            qdrant_create_collection "$@"
            ;;
        info)
            qdrant_collection_info "$@"
            ;;
        delete)
            qdrant_delete_collection "$@"
            ;;
        search)
            qdrant_collections_search "$@"
            ;;
        help|--help|-h)
            echo "Usage: resource-qdrant collections <subcommand> [options]"
            echo ""
            echo "Subcommands:"
            echo "  list      List all collections"
            echo "  create    Create a new collection"
            echo "  info      Show collection information"
            echo "  delete    Delete a collection"
            echo "  search    Search in collections"
            echo ""
            echo "📦 Collection Management Examples:"
            echo "  resource-qdrant collections list                          # List all collections"
            echo "  resource-qdrant collections list --show-models            # List with compatible models"
            echo "  resource-qdrant collections create my-docs --model nomic-embed-text  # Auto-detect dimensions"
            echo "  resource-qdrant collections create vectors --dimensions 768          # Manual dimensions"
            echo "  resource-qdrant collections info my-docs                  # Show collection details"
            echo "  resource-qdrant collections delete old-vectors            # Delete collection"
            echo ""
            echo "🔍 Search Examples:"
            echo "  resource-qdrant collections search \"quantum computing\"    # Search with text query"
            echo "  resource-qdrant collections search --text \"AI\" --collection docs --limit 5"
            echo "  resource-qdrant collections search --text \"machine learning\" --model nomic-embed-text"
            echo "  resource-qdrant collections search --embedding '[0.1, 0.2, ...]' --collection my-docs"
            echo ""
            echo "💡 Tips:"
            echo "  • Use --model to auto-detect dimensions when creating collections"
            echo "  • Search without --collection will auto-select if only one exists"
            echo "  • Use --show-models with list to see compatible embedding models"
            ;;
        *)
            log::error "Unknown collections subcommand: $subcommand"
            echo ""
            echo "Available subcommands: list, create, info, delete, search"
            echo "Use 'resource-qdrant collections help' for more information"
            return 1
            ;;
    esac
}

# List collections
qdrant_list_collections() {
    local show_models="${1:-}"
    
    if [[ "$show_models" == "--show-models" ]] || [[ "$show_models" == "-m" ]]; then
        if command -v qdrant::collections::list_with_models &>/dev/null; then
            qdrant::collections::list_with_models
        else
            log::warn "Model compatibility listing not available"
            qdrant::collections::list
        fi
    elif command -v qdrant::collections::list &>/dev/null; then
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
    if ! command -v qdrant::collections::parse_create_args &>/dev/null; then
        log::error "Collection argument parsing not available"
        return 1
    fi
    
    if ! qdrant::collections::parse_create_args "$@"; then
        local parse_result=$?
        if [[ $parse_result -eq 2 ]]; then
            # Help was requested
            echo "Usage: resource-qdrant collections create <name> [options]"
            echo ""
            echo "Options:"
            echo "  --model, -m <model>      Auto-detect dimensions from model"
            echo "  --dimensions, -d <n>     Manual dimension specification"
            echo "  --distance <metric>      Distance metric (Cosine/Dot/Euclid)"
            echo "  --help, -h               Show this help message"
            echo ""
            echo "Examples:"
            echo "  resource-qdrant collections create my-docs --model nomic-embed-text"
            echo "  resource-qdrant collections create embeddings --dimensions 384"
            echo "  resource-qdrant collections create vectors 768 --distance Dot"
            return 0
        else
            # Parsing error
            echo "Usage: resource-qdrant collections create <name> [options]"
            echo ""
            echo "Options:"
            echo "  --model, -m <model>      Auto-detect dimensions from model"
            echo "  --dimensions, -d <n>     Manual dimension specification"
            echo "  --distance <metric>      Distance metric (Cosine/Dot/Euclid)"
            echo "  --help, -h               Show this help message"
            echo ""
            echo "Examples:"
            echo "  resource-qdrant collections create my-docs --model nomic-embed-text"
            echo "  resource-qdrant collections create embeddings --dimensions 384"
            echo "  resource-qdrant collections create vectors 768 --distance Dot"
            return 1
        fi
    fi
    
    if command -v qdrant::collections::create_parsed &>/dev/null; then
        qdrant::collections::create_parsed
    else
        log::error "Collection creation functions not available"
        return 1
    fi
}

# Delete collection
qdrant_delete_collection() {
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
qdrant_collection_info() {
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

# Backup dispatcher for subcommands
qdrant_backup_dispatch() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        create)
            qdrant_create_backup "$@"
            ;;
        list)
            qdrant_list_backups "$@"
            ;;
        help|--help|-h)
            echo "Usage: resource-qdrant backup <subcommand> [options]"
            echo ""
            echo "Subcommands:"
            echo "  create    Create a backup snapshot"
            echo "  list      List all backups"
            echo ""
            echo "💾 Backup Examples:"
            echo "  resource-qdrant backup list                               # List all existing backups"
            echo "  resource-qdrant backup create                             # Create backup with auto-generated name"
            echo "  resource-qdrant backup create my-backup-name              # Create backup with custom name"
            echo "  resource-qdrant backup create pre-upgrade-\$(date +%Y%m%d)  # Timestamped backup"
            echo ""
            echo "💡 Tips:"
            echo "  • Backups include all collections and their data"
            echo "  • Auto-generated names use format: backup-YYYYMMDD-HHMMSS"
            echo "  • Custom names help identify backup purposes"
            echo "  • Both native snapshots and framework backups are supported"
            ;;
        *)
            log::error "Unknown backup subcommand: $subcommand"
            echo ""
            echo "Available subcommands: create, list"
            echo "Use 'resource-qdrant backup help' for more information"
            return 1
            ;;
    esac
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

# Search in collections
qdrant_collections_search() {
    if ! command -v qdrant::search::parse_args &>/dev/null; then
        log::error "Search argument parsing not available"
        return 1
    fi
    
    if ! qdrant::search::parse_args "$@"; then
        echo "Usage: resource-qdrant collections search [options]"
        echo ""
        echo "Options:"
        echo "  --text, -t <text>        Text to embed and search"
        echo "  --embedding, -e <json>   Direct embedding vector"
        echo "  --collection, -c <name>  Target collection (auto-detect if one)"
        echo "  --limit, -l <n>          Number of results (default: 10)"
        echo "  --model, -m <model>      Embedding model (auto-detect from collection)"
        echo ""
        echo "Examples:"
        echo "  resource-qdrant collections search --text \"quantum computing\""
        echo "  resource-qdrant collections search \"machine learning\" --limit 5"
        echo "  resource-qdrant collections search --text \"AI\" --collection docs"
        return 1
    fi
    
    if command -v qdrant::search::execute_parsed &>/dev/null; then
        qdrant::search::execute_parsed
    else
        log::error "Search execution functions not available"
        return 1
    fi
}

# Generate embeddings
qdrant_embed() {
    if ! command -v qdrant::embeddings::parse_args &>/dev/null; then
        log::error "Embedding argument parsing not available"
        return 1
    fi
    
    if ! qdrant::embeddings::parse_args "$@"; then
        log::error "Failed to parse embedding arguments"
        return 1
    fi
    
    # Show help if no arguments provided and not info-only mode
    if [[ -z "${PARSED_TEXT:-}" && -z "${PARSED_FROM_FILE:-}" && "${PARSED_BATCH:-}" != "true" && "${PARSED_INFO_ONLY:-}" != "true" ]]; then
        echo "Usage: resource-qdrant embed <text> [options]"
        echo ""
        echo "Options:"
        echo "  --model, -m <model>  Specific model to use"
        echo "  --info               Show embedding info only"
        echo "  --batch, -b          Process multiple texts"
        echo "  --from-file, -f      Read text from file"
        echo ""
        echo "Examples:"
        echo "  resource-qdrant embed \"machine learning\""
        echo "  resource-qdrant embed --model nomic-embed-text --info"
        echo "  cat texts.txt | resource-qdrant embed --batch"
        return 1
    fi
    
    if command -v qdrant::embeddings::execute_parsed &>/dev/null; then
        qdrant::embeddings::execute_parsed
    else
        log::error "Embedding execution functions not available"
        return 1
    fi
}

# Show embedding info
qdrant_embed_info() {
    local model="${1:-}"
    
    if command -v qdrant::embeddings::info &>/dev/null; then
        qdrant::embeddings::info "$model"
    else
        log::error "Embedding info not available"
        return 1
    fi
}

# Models dispatcher for subcommands
qdrant_models_dispatch() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        list)
            qdrant_models_list "$@"
            ;;
        info)
            qdrant_models_info "$@"
            ;;
        help|--help|-h)
            echo "Usage: resource-qdrant models <subcommand> [options]"
            echo ""
            echo "Subcommands:"
            echo "  list      List available embedding models"
            echo "  info      Show model information"
            echo ""
            echo "🤖 Model Examples:"
            echo "  resource-qdrant models list                               # List all embedding models"
            echo "  resource-qdrant models info nomic-embed-text              # Show model details"
            echo "  resource-qdrant models info mxbai-embed-large             # Show large model info"
            echo ""
            echo "💡 Tips:"
            echo "  • Models must be installed in Ollama first"
            echo "  • Use model names with collections create --model for auto-sizing"
            echo "  • Common dimensions: 384 (MiniLM), 768 (nomic-embed-text), 1024 (mxbai-embed-large)"
            ;;
        *)
            log::error "Unknown models subcommand: $subcommand"
            echo ""
            echo "Available subcommands: list, info"
            echo "Use 'resource-qdrant models help' for more information"
            return 1
            ;;
    esac
}

# List available models
qdrant_models_list() {
    if command -v qdrant::models::info &>/dev/null; then
        qdrant::models::info
    elif command -v qdrant::models::get_embedding_models &>/dev/null; then
        local models
        models=$(qdrant::models::get_embedding_models)
        echo "=== Available Embedding Models ==="
        echo "$models" | jq -r '.[] | "• \(.name) (\(.dimensions) dimensions)"'
    else
        log::error "Model listing not available"
        return 1
    fi
}

# Show model info
qdrant_models_info() {
    local model="${1:-}"
    
    if command -v qdrant::models::info &>/dev/null; then
        qdrant::models::info "$model"
    else
        log::error "Model info not available"
        return 1
    fi
}

# Custom help function with Qdrant-specific examples
qdrant_show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add concise Qdrant-specific information
    echo ""
    echo "🔍 Common Qdrant Operations:"
    echo ""
    echo "  collections <subcommand>    Manage vector collections"
    echo "  backup <subcommand>         Create and restore backups"
    echo "  models <subcommand>         List and inspect embedding models"
    echo "  embed <text>                Generate embeddings from text"
    echo ""
    echo "Quick Start:"
    echo "  resource-qdrant collections create my-docs --model nomic-embed-text"
    echo "  resource-qdrant embed \"machine learning\""
    echo "  resource-qdrant collections search \"AI research\""
    echo ""
    echo "Use '<command> help' for detailed examples (e.g., 'collections help')"
    echo ""
    echo "Default Port: 6333 | Web UI: http://localhost:6333/dashboard"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi