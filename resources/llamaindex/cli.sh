#!/usr/bin/env bash
################################################################################
# LlamaIndex Resource CLI - v2.0 Universal Contract Compliant
# 
# RAG and Document Processing Platform
# Provides intelligent document indexing and querying capabilities
#
# Usage:
#   resource-llamaindex <command> [options]
#   resource-llamaindex <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    LLAMAINDEX_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${LLAMAINDEX_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
LLAMAINDEX_CLI_DIR="${APP_ROOT}/resources/llamaindex"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${LLAMAINDEX_CLI_DIR}/config/defaults.sh"

# Source LlamaIndex libraries
for lib in core inject; do
    lib_file="${LLAMAINDEX_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "llamaindex" "LlamaIndex RAG and document processing platform" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="llamaindex::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="llamaindex::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="llamaindex::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="llamaindex::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="llamaindex::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="llamaindex::get_status"

# Content handlers - RAG and document management operations
CLI_COMMAND_HANDLERS["content::add"]="llamaindex::inject"
CLI_COMMAND_HANDLERS["content::list"]="llamaindex::list_indices" 
CLI_COMMAND_HANDLERS["content::get"]="llamaindex::query"
CLI_COMMAND_HANDLERS["content::remove"]="cli::framework::not_implemented"
CLI_COMMAND_HANDLERS["content::execute"]="llamaindex::query"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed LlamaIndex status" "llamaindex::get_status"
cli::register_command "logs" "Show LlamaIndex container logs" "llamaindex::logs"

# ==============================================================================
# CUSTOM LLAMAINDEX-SPECIFIC COMMANDS
# ==============================================================================

# Add LlamaIndex-specific content subcommands for document management
cli::register_subcommand "content" "index" "Index documents from directory" "llamaindex::index_documents" "modifies-system"
cli::register_subcommand "content" "indices" "List all available indices" "llamaindex::list_indices"

# Add utility command for checking indices
cli::register_command "indices" "Quick list of available indices" "llamaindex::list_indices"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi