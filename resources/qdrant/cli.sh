#!/usr/bin/env bash
################################################################################
# Qdrant Resource CLI - v2.0 Universal Contract Compliant
# 
# High-performance vector database with full-text search and semantic knowledge system
#
# Usage:
#   resource-qdrant <command> [options]
#   resource-qdrant <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    QDRANT_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${QDRANT_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
QDRANT_CLI_DIR="${APP_ROOT}/resources/qdrant"

# Source standard variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source v2.0 CLI Command Framework
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source qdrant configuration
# shellcheck disable=SC1091
source "${QDRANT_CLI_DIR}/config/defaults.sh"
qdrant::export_config 2>/dev/null || true

# Ensure API client is available before sourcing other libraries
api_client_file="${QDRANT_CLI_DIR}/lib/api-client.sh"
if [[ -f "$api_client_file" ]]; then
    # shellcheck disable=SC1090
    source "$api_client_file" || {
        log::error "Failed to load Qdrant API client"
        exit 1
    }
fi

# Source qdrant libraries
for lib in core health collections backup inject content status models embeddings search credentials; do
    lib_file="${QDRANT_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" || {
            log::warn "Failed to load library: $lib"
        }
    fi
done

# Note: Embeddings management system is loaded on-demand to avoid conflicts
# The embeddings dispatcher will source it when needed

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "qdrant" "Qdrant vector database with semantic knowledge system" "v2"

# ==============================================================================
# REQUIRED HANDLERS - Universal Contract v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="qdrant::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="qdrant::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="qdrant::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="qdrant::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="qdrant::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="qdrant::check_basic_health"
CLI_COMMAND_HANDLERS["test::integration"]="qdrant::test_integration"
CLI_COMMAND_HANDLERS["test::unit"]="qdrant::test_unit"
CLI_COMMAND_HANDLERS["test::all"]="qdrant::test_all"

# Content handlers - Qdrant's business functionality
CLI_COMMAND_HANDLERS["content::add"]="qdrant::content::add"
CLI_COMMAND_HANDLERS["content::list"]="qdrant::content::list"
CLI_COMMAND_HANDLERS["content::get"]="qdrant::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="qdrant::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="qdrant::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed Qdrant status" "qdrant::status"
cli::register_command "logs" "Show Qdrant logs" "qdrant::docker::logs"

# ==============================================================================
# QDRANT-SPECIFIC CUSTOM COMMANDS
# ==============================================================================

# Credentials for integration
cli::register_command "credentials" "Show Qdrant connection details" "qdrant::credentials"

# Collection management commands
cli::register_command "collections" "Manage collections (list/create/info/delete/search)" "qdrant_collections_dispatch"

# Embedding generation commands
cli::register_command "embed" "Generate embeddings from text" "qdrant_embed"
cli::register_command "embed-info" "Show embedding model information" "qdrant_embed_info"

# Model management commands
cli::register_command "models" "Manage embedding models (list/info)" "qdrant_models_dispatch"

# Backup management commands
cli::register_command "backup" "Manage backups (create/list)" "qdrant_backup_dispatch"

# Semantic knowledge system commands
cli::register_command "embeddings" "Manage semantic knowledge system" "qdrant_embeddings_dispatch"

# Legacy inject support with deprecation warning
cli::register_command "inject" "⚠️ DEPRECATED: Inject data (use 'content add' instead)" "qdrant_inject_deprecated"

# ==============================================================================
# QDRANT-SPECIFIC COMMAND IMPLEMENTATIONS
# ==============================================================================

# Collections dispatcher for subcommands
qdrant_collections_dispatch() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        list) qdrant::collections::list "$@" ;;
        create) qdrant::collections::create_parsed_wrapper "$@" ;;
        info) qdrant::collections::info "$@" ;;
        delete) qdrant::collections::delete "$@" "yes" ;;
        search) qdrant::search::execute_wrapper "$@" ;;
        help|--help|-h)
            echo "Usage: resource-qdrant collections <subcommand> [options]"
            echo
            echo "Subcommands:"
            echo "  list      List all collections"
            echo "  create    Create a new collection"
            echo "  info      Show collection information" 
            echo "  delete    Delete a collection"
            echo "  search    Search in collections"
            return 0
            ;;
        *) 
            log::error "Unknown collections subcommand: $subcommand"
            echo "Use 'resource-qdrant collections help' for available commands"
            return 1
            ;;
    esac
}

# Wrapper for collection creation with argument parsing
qdrant::collections::create_parsed_wrapper() {
    if ! command -v qdrant::collections::parse_create_args &>/dev/null; then
        log::error "Collection argument parsing not available"
        return 1
    fi
    
    if ! qdrant::collections::parse_create_args "$@"; then
        return 1
    fi
    
    if command -v qdrant::collections::create_parsed &>/dev/null; then
        qdrant::collections::create_parsed
    else
        log::error "Collection creation functions not available"
        return 1
    fi
}

# Wrapper for search execution
qdrant::search::execute_wrapper() {
    if ! command -v qdrant::search::parse_args &>/dev/null; then
        log::error "Search argument parsing not available"
        return 1
    fi
    
    if ! qdrant::search::parse_args "$@"; then
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

# Models dispatcher
qdrant_models_dispatch() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        list) 
            if command -v qdrant::models::info &>/dev/null; then
                qdrant::models::info
            else
                log::error "Model listing not available"
                return 1
            fi
            ;;
        info)
            local model="${1:-}"
            if command -v qdrant::models::info &>/dev/null; then
                qdrant::models::info "$model"
            else
                log::error "Model info not available"
                return 1
            fi
            ;;
        help|--help|-h)
            echo "Usage: resource-qdrant models <subcommand> [options]"
            echo
            echo "Subcommands:"
            echo "  list      List available embedding models"
            echo "  info      Show model information"
            return 0
            ;;
        *)
            log::error "Unknown models subcommand: $subcommand"
            echo "Use 'resource-qdrant models help' for available commands"
            return 1
            ;;
    esac
}

# Backup dispatcher
qdrant_backup_dispatch() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        create)
            local name="${1:-backup-$(date +%Y%m%d-%H%M%S)}"
            if command -v qdrant::backup::create &>/dev/null; then
                qdrant::backup::create "$name"
            else
                log::error "Backup functionality not available"
                return 1
            fi
            ;;
        list)
            if command -v qdrant::backup::list &>/dev/null; then
                qdrant::backup::list
            else
                log::error "Backup listing not available"
                return 1
            fi
            ;;
        help|--help|-h)
            echo "Usage: resource-qdrant backup <subcommand> [options]"
            echo
            echo "Subcommands:"
            echo "  create    Create a backup snapshot"
            echo "  list      List all backups"
            return 0
            ;;
        *)
            log::error "Unknown backup subcommand: $subcommand"
            echo "Use 'resource-qdrant backup help' for available commands"
            return 1
            ;;
    esac
}

# Embeddings knowledge system dispatcher (v2.0 compliant)
qdrant_embeddings_dispatch() {
    local subcommand="${1:-help}"
    shift || true
    
    # Check if embeddings CLI is available
    local embeddings_cli="${QDRANT_CLI_DIR}/embeddings/cli.sh"
    if [[ ! -f "${embeddings_cli}" ]]; then
        log::error "Embeddings knowledge system not available"
        return 1
    fi
    
    # Delegate directly to the v2.0 compliant embeddings CLI
    "${embeddings_cli}" "$subcommand" "$@"
}

# Deprecated inject command with warning
qdrant_inject_deprecated() {
    local file="${1:-}"
    
    log::warn "⚠️ DEPRECATED: 'inject' command is deprecated. Use 'resource-qdrant content add --file FILE' instead"
    echo
    
    if [[ -z "$file" ]] || [[ "$file" == "--help" ]] || [[ "$file" == "-h" ]]; then
        echo "Usage: resource-qdrant inject <file.json>"
        echo
        echo "⚠️  DEPRECATED: This command will be removed in a future version"
        echo
        echo "Recommended alternative:"
        echo "  resource-qdrant content add --file <file.json>"
        return 0
    fi
    
    # Delegate to content add command
    if command -v qdrant::content::add &>/dev/null; then
        qdrant::content::add --file "$file"
    else
        log::error "Content add function not available"
        return 1
    fi
}

# Show credentials for integration
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

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi