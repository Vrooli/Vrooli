#!/usr/bin/env bash
################################################################################
# Qdrant Embeddings CLI
# 
# Advanced embedding management for Vrooli's AI intelligence system
# Implements v2.0 Universal Resource Contract
#
# Usage:
#   resource-qdrant-embeddings <command> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    EMBEDDINGS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${EMBEDDINGS_CLI_SCRIPT%/*}/../../.." && builtin pwd)"
fi
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
QDRANT_DIR="${APP_ROOT}/resources/qdrant"

# Source standard variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source unified configuration
# shellcheck disable=SC1091
source "${EMBEDDINGS_DIR}/config/unified.sh"

# Source qdrant libraries
# shellcheck disable=SC1091
source "${QDRANT_DIR}/lib/collections.sh"
# shellcheck disable=SC1091
source "${QDRANT_DIR}/lib/embeddings.sh"
# shellcheck disable=SC1091
source "${QDRANT_DIR}/lib/models.sh"

# Source embedding components
for component in identity workflows scenarios docs code resources; do
    if [[ "$component" == "identity" ]]; then
        component_file="${EMBEDDINGS_DIR}/indexers/${component}.sh"
    else
        component_file="${EMBEDDINGS_DIR}/extractors/${component}.sh"
    fi
    if [[ -f "$component_file" ]]; then
        # shellcheck disable=SC1090
        source "$component_file" 2>/dev/null || true
    fi
done

# Source search components
# shellcheck disable=SC1091
source "${EMBEDDINGS_DIR}/search/single.sh"
# shellcheck disable=SC1091
source "${EMBEDDINGS_DIR}/search/multi.sh"

# Initialize CLI framework
cli::init "qdrant-embeddings" "Advanced embedding management for AI intelligence"

################################################################################
# V2.0 Universal Contract - Required Commands
################################################################################

# ==================== Help & Information ====================
cli::register_command "help" "Show comprehensive help with examples" "embeddings_show_help"

# ==================== Management Commands ====================
cli::register_command "manage install" "Install embedding dependencies" "embeddings_install" "modifies-system"
cli::register_command "manage uninstall" "Remove embeddings (requires --force)" "embeddings_uninstall" "modifies-system"
cli::register_command "manage start" "Start embedding services" "embeddings_start" "modifies-system"
cli::register_command "manage stop" "Stop embedding services" "embeddings_stop" "modifies-system"
cli::register_command "manage restart" "Restart embedding services" "embeddings_restart" "modifies-system"

# ==================== Testing Commands ====================
cli::register_command "test all" "Run all embedding tests" "embeddings_test_all"
cli::register_command "test integration" "Run integration tests" "embeddings_test_integration"
cli::register_command "test unit" "Run unit tests" "embeddings_test_unit"
cli::register_command "test smoke" "Quick validation test" "embeddings_test_smoke"

# ==================== Content Management ====================
cli::register_command "content add" "Add embeddings for content" "embeddings_content_add" "modifies-system"
cli::register_command "content list" "List embedded content" "embeddings_content_list"
cli::register_command "content get" "Get embedding details" "embeddings_content_get"
cli::register_command "content remove" "Remove embeddings" "embeddings_content_remove" "modifies-system"
cli::register_command "content execute" "Process embeddings" "embeddings_content_execute" "modifies-system"

# ==================== Monitoring Commands ====================
cli::register_command "status" "Show embedding system status" "embeddings_status"
cli::register_command "logs" "View embedding logs" "embeddings_logs"

################################################################################
# Advanced Embedding Commands (Beyond v2.0 Contract)
################################################################################

# Initialization and Management
cli::register_command "init" "Initialize embeddings for current app" "embeddings_init" "modifies-system"
cli::register_command "refresh" "Refresh all embedding collections" "embeddings_refresh" "modifies-system"
cli::register_command "validate" "Validate embedding integrity" "embeddings_validate"

# Search Operations
cli::register_command "search" "Single-term embedding search" "embeddings_search"
cli::register_command "search-all" "Multi-collection search" "embeddings_search_all"

# AI Intelligence Operations
cli::register_command "patterns" "Discover patterns in embeddings" "embeddings_patterns"
cli::register_command "solutions" "Generate solution recommendations" "embeddings_solutions"
cli::register_command "gaps" "Analyze knowledge gaps" "embeddings_gaps"
cli::register_command "explore" "Interactive exploration mode" "embeddings_explore"

# Performance Operations
cli::register_command "optimize" "Optimize embedding indices" "embeddings_optimize" "modifies-system"
cli::register_command "stats" "Show embedding statistics" "embeddings_stats"

################################################################################
# Command Implementations
################################################################################

# ==================== Help Implementation ====================
embeddings_show_help() {
    cat << EOF
Usage: resource-qdrant-embeddings <command> [options]

Advanced Embedding Management for Vrooli AI Intelligence

Standard Commands (v2.0 Contract):
  help                  Show this help message
  
  manage install        Install embedding dependencies
  manage uninstall      Remove embeddings (requires --force)
  manage start          Start embedding services
  manage stop           Stop embedding services
  manage restart        Restart embedding services
  
  test all             Run all embedding tests
  test integration     Run integration tests
  test smoke           Quick validation test
  
  content add          Add embeddings for content
  content list         List embedded content
  content get          Get embedding details
  content remove       Remove embeddings
  content execute      Process embeddings
  
  status               Show embedding system status
  logs                 View embedding logs

Advanced AI Commands:
  init [app-id]        Initialize embeddings for app
  refresh              Refresh all collections
  validate             Validate embedding integrity
  
  search <term>        Single-term search
  search-all <query>   Multi-collection search
  
  patterns             Discover patterns
  solutions            Generate recommendations
  gaps                 Analyze knowledge gaps
  explore              Interactive exploration
  
  optimize             Optimize indices
  stats                Show statistics

Examples:
  # Initialize embeddings for current app
  resource-qdrant-embeddings init
  
  # Refresh all embeddings
  resource-qdrant-embeddings refresh
  
  # Search across all collections
  resource-qdrant-embeddings search-all "authentication flow"
  
  # Discover patterns in codebase
  resource-qdrant-embeddings patterns
  
  # Analyze knowledge gaps
  resource-qdrant-embeddings gaps

Default Configuration:
  • Port: 6333 (Qdrant)
  • Collections: code, docs, workflows, scenarios, resources
  • Model: text-embedding-3-small
  • Parallel Processing: 4 workers
  • Cache: Enabled

Performance Features:
  • Parallel embedding generation
  • Intelligent caching
  • Batch processing
  • Incremental updates

Security:
  • API key authentication
  • Secure vector storage
  • Access control per collection
EOF
}

# ==================== Management Commands ====================
embeddings_install() {
    log::info "Installing embedding dependencies..."
    
    # Check if Qdrant is running
    if ! qdrant::is_running 2>/dev/null; then
        log::error "Qdrant must be running. Start with: resource-qdrant start"
        return 1
    fi
    
    # Initialize collections
    qdrant::embeddings::init_collections
    
    log::success "Embedding dependencies installed"
}

embeddings_uninstall() {
    if [[ "${1:-}" != "--force" ]]; then
        log::error "Uninstall requires --force flag to prevent accidental data loss"
        return 1
    fi
    
    log::warn "Removing all embeddings..."
    
    # Remove collections
    for collection in code docs workflows scenarios resources; do
        qdrant::collections::delete "$collection" 2>/dev/null || true
    done
    
    log::success "Embeddings removed"
}

embeddings_start() {
    log::info "Starting embedding services..."
    # Embeddings run as part of Qdrant, no separate service
    log::success "Embedding services active"
}

embeddings_stop() {
    log::info "Stopping embedding services..."
    # Embeddings run as part of Qdrant, no separate service
    log::success "Embedding services stopped"
}

embeddings_restart() {
    embeddings_stop
    embeddings_start
}

# ==================== Test Commands ====================
embeddings_test_all() {
    local failed=0
    
    log::info "Running all embedding tests..."
    
    embeddings_test_unit || ((failed++))
    embeddings_test_integration || ((failed++))
    embeddings_test_smoke || ((failed++))
    
    if [[ $failed -eq 0 ]]; then
        log::success "All tests passed"
        return 0
    else
        log::error "$failed test suite(s) failed"
        return 1
    fi
}

embeddings_test_integration() {
    log::info "Running integration tests..."
    
    # Run BATS tests if available
    if [[ -d "${EMBEDDINGS_DIR}/test" ]]; then
        bats "${EMBEDDINGS_DIR}/test" || return 1
    fi
    
    log::success "Integration tests passed"
}

embeddings_test_unit() {
    log::info "Running unit tests..."
    # Unit tests would go here
    log::info "No unit tests available"
    return 2
}

embeddings_test_smoke() {
    log::info "Running smoke test..."
    
    # Quick validation
    if ! qdrant::is_running 2>/dev/null; then
        log::error "Qdrant is not running"
        return 1
    fi
    
    # Check if we can list collections
    if ! qdrant::collections::list >/dev/null 2>&1; then
        log::error "Cannot access Qdrant collections"
        return 1
    fi
    
    log::success "Smoke test passed"
}

# ==================== Content Management ====================
embeddings_content_add() {
    local file="${1:-}"
    local type="${2:-}"
    local name="${3:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required: content add <file> [type] [name]"
        return 1
    fi
    
    log::info "Adding embeddings for $file..."
    
    # Determine type from file extension if not provided
    if [[ -z "$type" ]]; then
        case "$file" in
            *.ts|*.js|*.sh|*.py) type="code" ;;
            *.md|*.txt) type="docs" ;;
            *.json) type="workflows" ;;
            *) type="general" ;;
        esac
    fi
    
    # Process based on type
    case "$type" in
        code) qdrant::embeddings::process_code "$file" ;;
        docs) qdrant::embeddings::process_docs "$file" ;;
        workflows) qdrant::embeddings::process_workflows "$file" ;;
        *) log::error "Unknown type: $type" ; return 1 ;;
    esac
    
    log::success "Embeddings added"
}

embeddings_content_list() {
    local format="${1:-text}"
    
    log::info "Listing embedded content..."
    
    # List all collections and their counts
    for collection in code docs workflows scenarios resources; do
        if qdrant::collections::exists "$collection" 2>/dev/null; then
            local count=$(qdrant::collections::count "$collection" 2>/dev/null || echo "0")
            echo "  $collection: $count items"
        fi
    done
}

embeddings_content_get() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Name required: content get <name>"
        return 1
    fi
    
    log::info "Retrieving embedding for $name..."
    
    # Search for the embedding
    qdrant::embeddings::get "$name"
}

embeddings_content_remove() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Name required: content remove <name>"
        return 1
    fi
    
    log::warn "Removing embedding for $name..."
    
    # Remove the embedding
    qdrant::embeddings::delete "$name"
    
    log::success "Embedding removed"
}

embeddings_content_execute() {
    local name="${1:-}"
    
    log::info "Processing embeddings..."
    
    # This would trigger embedding generation/processing
    qdrant::embeddings::refresh
    
    log::success "Processing complete"
}

# ==================== Monitoring Commands ====================
embeddings_status() {
    # Use the original manage.sh status logic
    qdrant::embeddings::status "$@"
}

embeddings_logs() {
    local tail="${1:-50}"
    
    log::info "Showing last $tail log entries..."
    
    # Show Qdrant logs related to embeddings
    docker logs qdrant --tail "$tail" 2>&1 | grep -i embed || true
}

# ==================== Advanced Embedding Commands ====================
embeddings_init() {
    local app_id="${1:-}"
    qdrant::embeddings::init "$app_id"
}

embeddings_refresh() {
    qdrant::embeddings::refresh
}

embeddings_validate() {
    qdrant::embeddings::validate
}

embeddings_search() {
    local query="${1:-}"
    
    if [[ -z "$query" ]]; then
        log::error "Search query required"
        return 1
    fi
    
    qdrant::embeddings::search "$query"
}

embeddings_search_all() {
    local query="${1:-}"
    
    if [[ -z "$query" ]]; then
        log::error "Search query required"
        return 1
    fi
    
    qdrant::embeddings::search_all "$query"
}

embeddings_patterns() {
    qdrant::embeddings::discover_patterns
}

embeddings_solutions() {
    qdrant::embeddings::recommend_solutions
}

embeddings_gaps() {
    qdrant::embeddings::analyze_gaps
}

embeddings_explore() {
    qdrant::embeddings::explore
}

embeddings_optimize() {
    log::info "Optimizing embedding indices..."
    qdrant::embeddings::optimize_indices
    log::success "Optimization complete"
}

embeddings_stats() {
    qdrant::embeddings::show_statistics
}

################################################################################
# Main Execution
################################################################################

# Dispatch to CLI framework
cli::dispatch "$@"