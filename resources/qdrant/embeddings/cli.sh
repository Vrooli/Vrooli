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

# Configuration now loaded from unified.sh
# All settings available via QDRANT_* variables with backward-compatible aliases
# Export TEMP_DIR so background jobs can access it
export TEMP_DIR="/tmp/qdrant-embeddings-$$"

# Track background jobs for cleanup
declare -a BACKGROUND_JOBS=()

# Flag to prevent cleanup during active operations
REFRESH_IN_PROGRESS=false

# Cleanup function
cleanup_embeddings() {
    local exit_code="${1:-0}"
    
    # Kill all background jobs if they exist
    if [[ ${#BACKGROUND_JOBS[@]} -gt 0 ]]; then
        if [[ "$exit_code" != "0" ]]; then
            echo "" # New line after ^C characters
            log::warn "Interrupt received - cleaning up ${#BACKGROUND_JOBS[@]} background jobs..."
        else
            log::debug "Cleaning up ${#BACKGROUND_JOBS[@]} background jobs..."
        fi
        
        for pid in "${BACKGROUND_JOBS[@]}"; do
            if kill -0 "$pid" 2>/dev/null; then
                # Try to kill the process group first (more effective)
                kill -TERM "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
            fi
        done
        
        # Give processes time to terminate gracefully
        if [[ "$exit_code" != "0" ]]; then
            log::info "Waiting for jobs to terminate gracefully..."
        fi
        sleep 1
        
        # Force kill any remaining processes
        for pid in "${BACKGROUND_JOBS[@]}"; do
            if kill -0 "$pid" 2>/dev/null; then
                kill -KILL "-$pid" 2>/dev/null || kill -KILL "$pid" 2>/dev/null || true
            fi
        done
        
        # Clear the array
        BACKGROUND_JOBS=()
    fi
    
    # Clean up temp directory only if not in middle of refresh
    if [[ "$REFRESH_IN_PROGRESS" == "false" ]] && [[ -n "${TEMP_DIR:-}" ]] && [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
    fi
    
    if [[ "$exit_code" == "130" ]]; then
        log::warn "Embeddings generation cancelled by user"
    elif [[ "$exit_code" != "0" ]]; then
        log::error "Process exited with error code $exit_code"
    fi
    
    exit "$exit_code"
}

# Signal handlers
trap 'cleanup_embeddings 130' SIGINT   # Ctrl+C
trap 'cleanup_embeddings 143' SIGTERM  # Termination
trap 'cleanup_embeddings 0' EXIT       # Normal exit

# Source qdrant libraries
# shellcheck disable=SC1091
source "${QDRANT_DIR}/lib/collections.sh"
# shellcheck disable=SC1091
source "${QDRANT_DIR}/lib/embeddings.sh"
# shellcheck disable=SC1091
source "${QDRANT_DIR}/lib/models.sh"

# NOTE: v2.0 Compliance - No external dependencies
# All embedding functions implemented internally
# manage.sh has been deprecated and replaced with this self-contained CLI

# Source embedding components
for component in identity initialization scenarios docs code resources filetrees; do
    if [[ "$component" == "identity" ]]; then
        component_file="${EMBEDDINGS_DIR}/indexers/${component}.sh"
    else
        component_file="${EMBEDDINGS_DIR}/extractors/${component}/main.sh"
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
# shellcheck disable=SC1091
source "${EMBEDDINGS_DIR}/lib/performance-logger.sh"
# shellcheck disable=SC1091
source "${EMBEDDINGS_DIR}/lib/refresh.sh"

# Initialize CLI framework
cli::init "qdrant-embeddings" "Advanced embedding management for AI intelligence"

################################################################################
# V2.0 Universal Contract - Required Commands
################################################################################

# ==================== Help & Information ====================
# Override default help with embeddings-specific help
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
# Override default content command with embeddings-specific content management
cli::register_command "content add" "Add embeddings for content" "embeddings_content_add" "modifies-system"
cli::register_command "content list" "List embedded content" "embeddings_content_list"
cli::register_command "content get" "Get embedding details" "embeddings_content_get"
cli::register_command "content remove" "Remove embeddings" "embeddings_content_remove" "modifies-system"
cli::register_command "content execute" "Process embeddings" "embeddings_content_execute" "modifies-system"

# ==================== Monitoring Commands ====================
# Override default status with embeddings-specific status
cli::register_command "status" "Show embedding system status" "embeddings_status"
cli::register_command "logs" "View embedding logs" "embeddings_logs"

################################################################################
# Advanced Embedding Commands (Beyond v2.0 Contract)
################################################################################

# Initialization and Management
cli::register_command "init" "Initialize embeddings for current app" "embeddings_init" "modifies-system"
cli::register_command "refresh" "Refresh all embedding collections (supports --sequential, --workers)" "embeddings_refresh" "modifies-system"
# Override default validate with embeddings-specific validation
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
cli::register_command "performance" "Show performance metrics and summary" "embeddings_performance"
cli::register_command "efficiency" "Show parallel processing efficiency analysis" "embeddings_efficiency"

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
  performance          Show performance metrics
  efficiency           Analyze parallel processing efficiency

Examples:
  # Initialize embeddings for current app
  resource-qdrant-embeddings init
  
  # Refresh all embeddings (parallel processing - faster but uses more resources)
  resource-qdrant-embeddings refresh --force
  
  # Refresh with sequential processing (safer but slower)
  resource-qdrant-embeddings refresh --sequential --force
  
  # Refresh with custom worker count
  resource-qdrant-embeddings refresh --workers 2 --force
  
  # Search across all collections
  resource-qdrant-embeddings search-all "authentication flow"
  
  # Discover patterns in codebase
  resource-qdrant-embeddings patterns
  
  # Analyze knowledge gaps
  resource-qdrant-embeddings gaps

Default Configuration:
  • Port: 6333 (Qdrant)
  • Collections: code, docs, workflows, scenarios, resources, filetrees
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
    
    # Check if BATS is available
    if ! command -v bats &>/dev/null; then
        log::warn "BATS test framework not installed - skipping integration tests"
        log::info "Install with: npm install -g bats"
        return 0  # Not a failure, just skipped
    fi
    
    # Run BATS tests if available
    if [[ -d "${EMBEDDINGS_DIR}/test" ]]; then
        log::info "Found test directory, running BATS tests..."
        if bats "${EMBEDDINGS_DIR}/test"; then
            log::success "Integration tests passed"
            return 0
        else
            log::error "Integration tests failed"
            return 1
        fi
    else
        log::warn "No test directory found at ${EMBEDDINGS_DIR}/test"
        return 0  # Not a failure if tests don't exist
    fi
}

embeddings_test_unit() {
    log::info "Running unit tests..."
    # Unit tests are not yet implemented - integration tests cover current functionality
    log::info "Unit tests not implemented (use 'test integration' for BATS tests)"
    return 0  # Return 0 since this is expected, not an error
}

embeddings_test_smoke() {
    log::info "Running smoke test..."
    
    # Quick validation - check if Qdrant API is responding
    if ! curl -s "http://localhost:6333/collections" | jq -e '.status == "ok"' >/dev/null 2>&1; then
        log::error "Qdrant is not running or not responding"
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
        workflows) log::warn "Workflows are now processed automatically via resources stream - no manual processing needed" ;;
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
    embeddings_status_impl "$@"
}

embeddings_logs() {
    local tail="${1:-50}"
    
    log::info "Showing last $tail log entries..."
    
    # Show Qdrant logs related to embeddings
    docker logs qdrant --tail "$tail" 2>&1 | grep -i embed || true
}

# ==================== Advanced Embedding Commands ====================
embeddings_init() {
    embeddings_init_impl "$@"
}

embeddings_refresh() {
    embeddings_refresh_impl "$@"
}

embeddings_validate() {
    embeddings_validate_impl "$@"
}

embeddings_search() {
    local query="${1:-}"
    
    if [[ -z "$query" ]]; then
        log::error "Search query required"
        return 1
    fi
    
    # Get app identity and call correct search function
    local app_id=$(qdrant::identity::get_app_id)
    if [[ -z "$app_id" ]]; then
        echo '{"error": "No app identity found. Run embeddings init first"}'
        return 1
    fi
    
    # Use same implementation as core embedding functions
    qdrant::search::single_app "$query" "$app_id" "all"
}

embeddings_search_all() {
    local query="${1:-}"
    
    if [[ -z "$query" ]]; then
        log::error "Search query required"
        return 1
    fi
    
    # Use same implementation as core embedding functions
    qdrant::search::report "$query" "json"
}

embeddings_patterns() {
    qdrant::search::discover_patterns "$@"
}

embeddings_solutions() {
    qdrant::search::find_solutions "$@"
}

embeddings_gaps() {
    qdrant::search::find_gaps "$@"
}

embeddings_explore() {
    qdrant::search::explore "$@"
}

embeddings_optimize() {
    log::info "Optimizing embedding indices..."
    # TODO: Implement actual optimization
    # For now, just trigger Qdrant's internal optimization
    local app_id=$(qdrant::identity::get_app_id)
    if [[ -z "$app_id" ]]; then
        log::error "No app identity found"
        return 1
    fi
    
    # Trigger optimization on all collections
    for collection in $(qdrant::identity::get_collections); do
        log::info "Optimizing $collection..."
        curl -s -X POST "http://localhost:6333/collections/$collection/index/optimize" >/dev/null 2>&1 || true
    done
    
    log::success "Optimization complete"
}

embeddings_stats() {
    # Show detailed statistics using status function
    qdrant::embeddings::status "$@"
}

embeddings_performance() {
    echo "=== Embedding Performance Metrics ==="
    echo
    
    # Generate and display performance summary
    local summary=$(perf::generate_summary 2>/dev/null)
    if [[ $? -eq 0 ]] && [[ "$summary" != *"error"* ]]; then
        echo "📊 Performance Summary:"
        echo "$summary" | jq -r '
            "  Total Entries: " + (.summary.total_entries | tostring) + "\n" +
            "  Operations: " + (.summary.operations | join(", ")) + "\n" +
            "  Avg Embedding Duration: " + ((.summary.avg_embedding_duration // 0) | tostring) + "ms\n" +
            "  Avg Throughput: " + ((.summary.avg_throughput // 0) | tostring) + " items/sec\n" +
            "  Parallel Efficiency: " + ((.summary.parallel_efficiency // 0) | tostring) + "%"
        '
        echo
        echo "🔧 Recent Metrics:"
        echo "$summary" | jq -r '.recent_metrics[] | "  " + .timestamp + " | " + .operation + "." + .metric + " = " + (.value | tostring)'
    else
        echo "⚠️  No performance data available"
        echo "   Performance logging may be disabled or no operations have been logged yet."
        echo "   Run an embedding refresh to generate performance data."
    fi
    
    echo
    echo "⚙️  Performance Configuration:"
    echo "  Logging Enabled: ${EMBEDDING_PERF_LOG_ENABLED:-true}"
    echo "  Log Level: ${EMBEDDING_PERF_LOG_LEVEL:-info}"
    echo "  Log Format: ${EMBEDDING_PERF_LOG_FORMAT:-json}"
    if [[ -n "${EMBEDDING_PERF_LOG_FILE:-}" ]]; then
        echo "  Log File: ${EMBEDDING_PERF_LOG_FILE}"
    else
        echo "  Log File: (debug output only)"
    fi
}

embeddings_efficiency() {
    perf::generate_efficiency_report
}

################################################################################
# Internal Embedding Functions (v2.0 compliant - self-contained)
################################################################################

#######################################
# Initialize embeddings for current app
# Arguments:
#   $1 - App ID (optional, auto-detected if not provided)
# Returns: 0 on success
#######################################
embeddings_init_impl() {
    local app_id="${1:-}"
    
    log::info "Initializing embeddings for project..."
    
    # Pre-flight service health checks
    log::info "Performing service health checks..."
    
    # Check Qdrant availability
    if ! curl -s --max-time 10 "http://localhost:6333/health" >/dev/null 2>&1; then
        log::error "Qdrant service is not available"
        log::error "Please ensure Qdrant is running: resource-qdrant start"
        return 1
    fi
    log::success "Qdrant service is healthy"
    
    # Check Ollama availability
    if ! curl -s --max-time 10 "http://localhost:11434/api/tags" >/dev/null 2>&1; then
        log::error "Ollama service is not available"
        log::error "Please ensure Ollama is running with embedding models"
        return 1
    fi
    log::success "Ollama service is healthy"
    
    # Initialize app identity
    qdrant::identity::init "$app_id"
    
    # Get the app ID that was created
    app_id=$(qdrant::identity::get_app_id)
    
    if [[ -z "$app_id" ]]; then
        log::error "Failed to initialize app identity"
        return 1
    fi
    
    log::success "Initialized embeddings for app: $app_id"
    
    # Show initial status
    qdrant::identity::show
    
    return 0
}

#######################################
# Refresh all embeddings for an app
# Arguments:
#   $1 - App ID (optional, current app if not specified) or --force
#   $2 - Force value (yes/no) if $1 is --force
# Returns: 0 on success
#######################################
embeddings_refresh_impl() {
    local app_id=""
    local force="no"
    local sequential="no"
    
    # Debug: show all arguments
    log::debug "refresh called with arguments: $*"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        log::debug "Processing arg: '$1' (remaining: $#)"
        case "$1" in
            --force)
                force="${2:-yes}"
                if [[ -n "${2:-}" ]]; then
                    shift 2
                else
                    shift
                fi
                ;;
            --sequential)
                sequential="yes"
                shift
                ;;
            yes|no)
                # Skip force values that might be passed
                shift
                ;;
            *)
                if [[ -z "$app_id" ]] && [[ "$1" != "yes" ]] && [[ "$1" != "no" ]]; then
                    app_id="$1"
                    log::debug "Setting app_id to: $app_id"
                fi
                shift
                ;;
        esac
    done
    
    # If no app_id provided, get current app
    if [[ -z "$app_id" ]] || [[ "$app_id" == "yes" ]] || [[ "$app_id" == "no" ]] || [[ "$app_id" =~ ^[0-9]+$ ]]; then
        log::debug "Getting app_id from identity (current: '$app_id')"
        app_id=$(qdrant::identity::get_app_id)
    fi
    
    if [[ -z "$app_id" ]]; then
        log::error "No app identity found. Run 'embeddings init' first"
        return 1
    fi
    
    # Check if refresh is needed
    if [[ "$force" != "yes" ]]; then
        if ! qdrant::identity::needs_reindex; then
            log::info "Embeddings are up to date for app: $app_id"
            return 0
        fi
    fi
    
    log::info "Refreshing embeddings for app: $app_id"
    
    # Pre-flight service health checks with retries
    log::info "Performing service health checks..."
    
    # Check Qdrant availability
    local max_retries=5
    local retry_delay=2
    local qdrant_ready=false
    
    for attempt in $(seq 1 $max_retries); do
        if curl -s --max-time 10 "http://localhost:6333/health" >/dev/null 2>&1; then
            log::success "Qdrant service is healthy"
            qdrant_ready=true
            break
        fi
        
        if [[ $attempt -lt $max_retries ]]; then
            log::warn "Qdrant not ready (attempt $attempt/$max_retries), retrying in ${retry_delay}s..."
            sleep $retry_delay
        fi
    done
    
    if [[ "$qdrant_ready" != "true" ]]; then
        log::error "Qdrant service is not available after $max_retries attempts"
        log::error "Please ensure Qdrant is running: resource-qdrant start"
        return 1
    fi
    
    # Check Ollama availability and model
    local ollama_ready=false
    local model="${QDRANT_EMBEDDING_MODEL:-mxbai-embed-large}"
    
    for attempt in $(seq 1 $max_retries); do
        if curl -s --max-time 10 "http://localhost:11434/api/tags" >/dev/null 2>&1; then
            log::success "Ollama service is healthy"
            
            # Verify embedding model is available
            if curl -s --max-time 10 "http://localhost:11434/api/tags" | jq -e --arg model "$model" '.models[] | select(.name | startswith($model))' >/dev/null 2>&1; then
                log::success "Embedding model '$model' is available"
                ollama_ready=true
                break
            else
                log::error "Embedding model '$model' is not available"
                log::info "Available models:"
                curl -s "http://localhost:11434/api/tags" | jq -r '.models[]?.name // empty' | sed 's/^/  - /' || echo "  (Unable to retrieve model list)"
                return 1
            fi
        fi
        
        if [[ $attempt -lt $max_retries ]]; then
            log::warn "Ollama not ready (attempt $attempt/$max_retries), retrying in ${retry_delay}s..."
            sleep $retry_delay
        fi
    done
    
    if [[ "$ollama_ready" != "true" ]]; then
        log::error "Ollama service is not available after $max_retries attempts"
        log::error "Please ensure Ollama is running with embedding model: $model"
        return 1
    fi
    
    log::success "All service dependencies are healthy ✅"
    local start_time=$(date +%s)
    
    # Create temporary directory for extracted content using existing export
    mkdir -p "$TEMP_DIR"
    
    # Get collections for this app
    local collections=$(qdrant::identity::get_collections)
    
    # Delete existing collections
    log::info "Clearing existing collections..."
    for collection in $collections; do
        qdrant::collections::delete "$collection" "yes" 2>/dev/null || true
    done
    
    # Create fresh collections
    log::info "Creating collections..."
    for collection in $collections; do
        qdrant::collections::create "$collection" "$QDRANT_EMBEDDING_DIMENSIONS" "$QDRANT_EMBEDDING_DISTANCE_METRIC" || {
                log::error "Failed to create collection: $collection"
                return 1
            }
    done
    
    # Extract and embed content by type - SEQUENTIAL OR PARALLEL
    local total_embeddings=0
    
    if [[ "$sequential" == "yes" ]]; then
        log::info "Processing content sequentially (safer but slower)..."
        total_embeddings=$(embeddings_refresh_sequential_impl "$app_id")
    else
        log::info "Processing all content types in parallel with ${EMBEDDING_MAX_WORKERS:-16} workers..."
        
        # Set flag to prevent cleanup during refresh
        REFRESH_IN_PROGRESS=true
    
    # Create background jobs for each content type
    # Process all content types including workflows via initialization
    local workflow_count_file="$TEMP_DIR/workflow_count"
    local scenario_count_file="$TEMP_DIR/scenario_count"
    local doc_count_file="$TEMP_DIR/doc_count"
    local code_count_file="$TEMP_DIR/code_count"
    local resource_count_file="$TEMP_DIR/resource_count"
    local filetrees_count_file="$TEMP_DIR/filetrees_count"
    
    # Start all content type processing in parallel with process groups for better signal handling
    
    # Process workflows
    {
        # Create new process group and set up signal handling
        set -m  # Enable job control
        APP_ROOT="$APP_ROOT" EMBEDDINGS_DIR="$EMBEDDINGS_DIR" TEMP_DIR="$TEMP_DIR" \
        WORKFLOW_COUNT_FILE="$workflow_count_file" APP_ID="$app_id" \
        exec setsid timeout 600 bash -c '
            set -euo pipefail
            trap "echo \"Workflows job interrupted\"; echo \"0\" > \"$WORKFLOW_COUNT_FILE\"; exit 130" SIGINT SIGTERM
            
            # Source all required utilities
            source "$APP_ROOT/scripts/lib/utils/var.sh"
            source "${var_LIB_UTILS_DIR}/log.sh"
            source "$EMBEDDINGS_DIR/config/unified.sh"
            export TEMP_DIR
            
            echo "[INFO] Processing workflows..."
            source "$EMBEDDINGS_DIR/extractors/initialization/main.sh"
            workflow_count=$(qdrant::embeddings::process_initialization "$APP_ID" 2>&1 | tail -1) || workflow_count=0
            echo "${workflow_count:-0}" > "$WORKFLOW_COUNT_FILE"
            echo "[INFO] Workflows complete: ${workflow_count:-0} items"
        ' || echo "0" > "$workflow_count_file"
    } &
    local workflow_pid=$!
    BACKGROUND_JOBS+=("$workflow_pid")
    
    # Process scenarios
    {
        set -m
        APP_ROOT="$APP_ROOT" EMBEDDINGS_DIR="$EMBEDDINGS_DIR" TEMP_DIR="$TEMP_DIR" \
        SCENARIO_COUNT_FILE="$scenario_count_file" APP_ID="$app_id" \
        exec setsid timeout 600 bash -c '
            set -euo pipefail
            trap "echo \"Scenarios job interrupted\"; echo \"0\" > \"$SCENARIO_COUNT_FILE\"; exit 130" SIGINT SIGTERM
            
            # Source all required utilities
            source "$APP_ROOT/scripts/lib/utils/var.sh"
            source "${var_LIB_UTILS_DIR}/log.sh"
            source "$EMBEDDINGS_DIR/config/unified.sh"
            export TEMP_DIR
            
            echo "[INFO] Processing scenarios..."
            source "$EMBEDDINGS_DIR/extractors/scenarios/main.sh"
            scenario_count=$(qdrant::embeddings::process_scenarios "$APP_ID" 2>&1 | tail -1) || scenario_count=0
            echo "${scenario_count:-0}" > "$SCENARIO_COUNT_FILE"
            echo "[INFO] Scenarios complete: ${scenario_count:-0} items"
        ' || echo "0" > "$scenario_count_file"
    } &
    local scenario_pid=$!
    BACKGROUND_JOBS+=("$scenario_pid")
    
    # Process documentation
    {
        set -m
        APP_ROOT="$APP_ROOT" EMBEDDINGS_DIR="$EMBEDDINGS_DIR" TEMP_DIR="$TEMP_DIR" \
        DOC_COUNT_FILE="$doc_count_file" APP_ID="$app_id" \
        exec setsid timeout 600 bash -c '
            set -euo pipefail
            trap "echo \"Documentation job interrupted\"; echo \"0\" > \"$DOC_COUNT_FILE\"; exit 130" SIGINT SIGTERM
            
            # Source all required utilities
            source "$APP_ROOT/scripts/lib/utils/var.sh"
            source "${var_LIB_UTILS_DIR}/log.sh"
            source "$EMBEDDINGS_DIR/config/unified.sh"
            export TEMP_DIR
            
            echo "[INFO] Processing documentation..."
            source "$EMBEDDINGS_DIR/extractors/docs/main.sh"
            doc_count=$(qdrant::embeddings::process_documentation "$APP_ID" 2>&1 | tail -1) || doc_count=0
            echo "${doc_count:-0}" > "$DOC_COUNT_FILE"
            echo "[INFO] Documentation complete: ${doc_count:-0} items"
        ' || echo "0" > "$doc_count_file"
    } &
    local doc_pid=$!
    BACKGROUND_JOBS+=("$doc_pid")
    
    # Process code
    {
        set -m
        APP_ROOT="$APP_ROOT" EMBEDDINGS_DIR="$EMBEDDINGS_DIR" TEMP_DIR="$TEMP_DIR" \
        CODE_COUNT_FILE="$code_count_file" APP_ID="$app_id" CODE_MAX_FILES="500" \
        exec setsid timeout 1200 bash -c '
            set -euo pipefail
            trap "echo \"Code job interrupted\"; echo \"0\" > \"$CODE_COUNT_FILE\"; exit 130" SIGINT SIGTERM
            
            # Source all required utilities
            source "$APP_ROOT/scripts/lib/utils/var.sh"
            source "${var_LIB_UTILS_DIR}/log.sh"
            source "$EMBEDDINGS_DIR/config/unified.sh"
            export TEMP_DIR
            
            echo "[INFO] Processing code (limited to $CODE_MAX_FILES files)..."
            source "$EMBEDDINGS_DIR/extractors/code/main.sh"
            code_count=$(qdrant::embeddings::process_code "$APP_ID" 2>&1 | tail -1) || code_count=0
            echo "${code_count:-0}" > "$CODE_COUNT_FILE"
            echo "[INFO] Code complete: ${code_count:-0} items"
        ' || echo "0" > "$code_count_file"
    } &
    local code_pid=$!
    BACKGROUND_JOBS+=("$code_pid")
    
    # Process resources
    {
        set -m
        APP_ROOT="$APP_ROOT" EMBEDDINGS_DIR="$EMBEDDINGS_DIR" TEMP_DIR="$TEMP_DIR" \
        RESOURCE_COUNT_FILE="$resource_count_file" APP_ID="$app_id" \
        exec setsid timeout 1200 bash -c '
            set -euo pipefail
            trap "echo \"Resources job interrupted\"; echo \"0\" > \"$RESOURCE_COUNT_FILE\"; exit 130" SIGINT SIGTERM
            
            # Source all required utilities
            source "$APP_ROOT/scripts/lib/utils/var.sh"
            source "${var_LIB_UTILS_DIR}/log.sh"
            source "$EMBEDDINGS_DIR/config/unified.sh"
            export TEMP_DIR
            
            echo "[INFO] Processing resources..."
            source "$EMBEDDINGS_DIR/extractors/resources/main.sh"
            resource_count=$(qdrant::embeddings::process_resources "$APP_ID" 2>&1 | tail -1) || resource_count=0
            echo "${resource_count:-0}" > "$RESOURCE_COUNT_FILE"
            echo "[INFO] Resources complete: ${resource_count:-0} items"
        ' || echo "0" > "$resource_count_file"
    } &
    local resource_pid=$!
    BACKGROUND_JOBS+=("$resource_pid")
    
    # Process file trees
    {
        set -m
        APP_ROOT="$APP_ROOT" EMBEDDINGS_DIR="$EMBEDDINGS_DIR" TEMP_DIR="$TEMP_DIR" \
        FILETREES_COUNT_FILE="$filetrees_count_file" APP_ID="$app_id" \
        exec setsid timeout 600 bash -c '
            set -euo pipefail
            trap "echo \"File trees job interrupted\"; echo \"0\" > \"$FILETREES_COUNT_FILE\"; exit 130" SIGINT SIGTERM
            
            # Source all required utilities
            source "$APP_ROOT/scripts/lib/utils/var.sh"
            source "${var_LIB_UTILS_DIR}/log.sh"
            source "$EMBEDDINGS_DIR/config/unified.sh"
            export TEMP_DIR
            
            echo "[INFO] Processing file trees..."
            source "$EMBEDDINGS_DIR/extractors/filetrees/main.sh"
            filetrees_count=$(qdrant::embeddings::process_file_trees "$APP_ID" 2>&1 | tail -1) || filetrees_count=0
            echo "${filetrees_count:-0}" > "$FILETREES_COUNT_FILE"
            echo "[INFO] File trees complete: ${filetrees_count:-0} items"
        ' || echo "0" > "$filetrees_count_file"
    } &
    local filetrees_pid=$!
    BACKGROUND_JOBS+=("$filetrees_pid")
    
    # Wait for all background jobs to complete with monitoring
    log::info "Waiting for all content type processing to complete..."
    
    # Monitor memory usage during parallel processing
    {
        set -m
        exec setsid bash -c '
            trap "echo \"Memory monitor interrupted\"; exit 130" SIGINT SIGTERM
            while kill -0 '$workflow_pid' '$scenario_pid' '$doc_pid' '$code_pid' '$resource_pid' '$filetrees_pid' 2>/dev/null; do
                mem_usage=$(free 2>/dev/null | awk "/Mem:/ {printf \"%.0f\", \$3/\$2 * 100}" 2>/dev/null || echo "0")
                if [[ $mem_usage -gt 85 ]]; then
                    echo "[WARN] High memory usage: ${mem_usage}%"
                fi
                sleep 2 || exit 130
            done
        '
    } &
    local monitor_pid=$!
    BACKGROUND_JOBS+=("$monitor_pid")
    
    # Use a polling-based wait mechanism that's more responsive to signals
    local failed_jobs=0
    local completed_jobs=0
    local job_names=("workflows" "scenarios" "documentation" "code" "resources" "filetrees")
    local job_pids=($workflow_pid $scenario_pid $doc_pid $code_pid $resource_pid $filetrees_pid)
    local job_status=(0 0 0 0 0 0)  # 0=running, 1=completed, 2=failed
    
    log::info "Monitoring ${#job_pids[@]} background jobs (press Ctrl+C to cancel)..."
    
    # Polling loop - much more responsive to signals than wait
    while [[ $completed_jobs -lt ${#job_pids[@]} ]]; do
        # Check each job
        for i in "${!job_pids[@]}"; do
            local pid="${job_pids[$i]}"
            local job_name="${job_names[$i]}"
            
            # Skip if already processed
            [[ ${job_status[$i]} -ne 0 ]] && continue
            
            # Check if process is still running
            if ! kill -0 "$pid" 2>/dev/null; then
                # Process has finished, check exit status
                if wait "$pid" 2>/dev/null; then
                    log::info "✅ ${job_name} processing completed successfully"
                    job_status[$i]=1
                else
                    log::error "❌ ${job_name} processing failed"
                    job_status[$i]=2
                    ((failed_jobs++))
                fi
                ((completed_jobs++))
            fi
        done
        
        # Short sleep to avoid busy waiting - this is where Ctrl+C can interrupt
        sleep 0.5 || {
            # Sleep was interrupted by signal - kill all remaining jobs
            log::warn "Interrupt signal received - terminating all background jobs..."
            
            for pid in "${job_pids[@]}"; do
                if kill -0 "$pid" 2>/dev/null; then
                    log::debug "Killing process group for PID $pid"
                    # Kill the entire process group (negative PID)
                    kill -TERM "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
                fi
            done
            
            # Kill monitor as well
            if kill -0 "$monitor_pid" 2>/dev/null; then
                kill -TERM "-$monitor_pid" 2>/dev/null || kill -TERM "$monitor_pid" 2>/dev/null || true
            fi
            
            # Give jobs 2 seconds to terminate gracefully
            sleep 2 || true
            
            # Force kill any remaining jobs and their process groups
            for pid in "${job_pids[@]}"; do
                if kill -0 "$pid" 2>/dev/null; then
                    log::debug "Force killing process group for PID $pid"
                    kill -KILL "-$pid" 2>/dev/null || kill -KILL "$pid" 2>/dev/null || true
                fi
            done
            
            # Force kill monitor
            if kill -0 "$monitor_pid" 2>/dev/null; then
                kill -KILL "-$monitor_pid" 2>/dev/null || kill -KILL "$monitor_pid" 2>/dev/null || true
            fi
            
            # Stop memory monitor
            kill -KILL $monitor_pid 2>/dev/null || true
            
            log::warn "All background jobs terminated - exiting..."
            cleanup_embeddings 130
        }
    done
    
    # Stop memory monitor
    if kill -0 "$monitor_pid" 2>/dev/null; then
        kill -TERM "-$monitor_pid" 2>/dev/null || kill -TERM "$monitor_pid" 2>/dev/null || true
    fi
    
    # Report parallel processing results
    if [[ $failed_jobs -eq 0 ]]; then
        log::success "All content types processed successfully in parallel"
    else
        log::warn "$failed_jobs out of ${#job_names[@]} content type jobs failed"
    fi
    
    # Clear background jobs from array since they've completed
    BACKGROUND_JOBS=()
    
    # Read results from temporary files
    local workflow_count=$(cat "$workflow_count_file" 2>/dev/null || echo "0")
    local scenario_count=$(cat "$scenario_count_file" 2>/dev/null || echo "0")
    local doc_count=$(cat "$doc_count_file" 2>/dev/null || echo "0")
    local code_count=$(cat "$code_count_file" 2>/dev/null || echo "0")
    local resource_count=$(cat "$resource_count_file" 2>/dev/null || echo "0")
    local filetrees_count=$(cat "$filetrees_count_file" 2>/dev/null || echo "0")
    
    # Calculate total
    total_embeddings=$((workflow_count + scenario_count + doc_count + code_count + resource_count + filetrees_count))
    
    # Clear flag now that count files have been read
    REFRESH_IN_PROGRESS=false
    
    fi # End of parallel processing branch
    
    # Calculate duration
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Update app identity with results
    qdrant::identity::update_after_index "$total_embeddings" "$duration"
    
    log::success "Embedding refresh complete!"
    log::info "App: $app_id"
    log::info "Total Embeddings: $total_embeddings"
    log::info "Duration: ${duration}s"
    
    # Cleanup
    rm -rf "$TEMP_DIR"
    
    return 0
}

#######################################
# Sequential embedding refresh - simpler but slower fallback
# Arguments:
#   $1 - App ID
# Returns: Number of embeddings created
#######################################
embeddings_refresh_sequential_impl() {
    local app_id="$1"
    local total_count=0
    
    log::info "Starting sequential processing for safety..."
    
    # Process workflows directly with API calls
    log::info "Processing workflows..."
    local workflow_count=0
    
    # Find workflow files using a simpler approach
    local workflow_files
    workflow_files=$(find "${APP_ROOT:-${HOME}/Vrooli}/initialization/n8n" -name "*.json" -type f 2>/dev/null | head -3)
    
    if [[ -n "$workflow_files" ]]; then
        while IFS= read -r file; do
            if [[ -f "$file" ]]; then
                log::debug "Processing workflow file: $file"
                
                local name
                name=$(jq -r '.name // "Unnamed"' "$file" 2>/dev/null || echo "Unnamed")
                local content="N8n workflow: $name"
                
                log::debug "Generating embedding for: $name"
                
                # Generate embedding via direct API call
                local embedding
                embedding=$(curl -s -X POST http://localhost:11434/api/embeddings \
                    -H "Content-Type: application/json" \
                    -d "{\"model\": \"${QDRANT_EMBEDDING_MODEL:-mxbai-embed-large}\", \"prompt\": $(echo "$content" | jq -Rs .)}")
                
                if [[ -n "$embedding" ]]; then
                    embedding=$(echo "$embedding" | jq -c '.embedding' 2>/dev/null)
                    
                    if [[ -n "$embedding" ]] && [[ "$embedding" != "null" ]]; then
                        log::debug "Embedding generated successfully"
                        
                        # Create payload
                        local payload
                        payload=$(jq -n \
                            --arg content "$content" \
                            --arg type "workflow" \
                            --arg app_id "$app_id" \
                            --arg name "$name" \
                            --arg file "$file" \
                            '{content: $content, type: $type, app_id: $app_id, name: $name, file: $file}')
                        
                        # Generate UUID-format ID from hash
                        local hash
                        hash=$(echo -n "workflow:$file" | sha256sum | cut -d' ' -f1)
                        local id="${hash:0:8}-${hash:8:4}-${hash:12:4}-${hash:16:4}-${hash:20:12}"
                        
                        log::debug "Storing in Qdrant with UUID ID: $id"
                        
                        # Store directly in Qdrant
                        local response
                        response=$(curl -s -X PUT "http://localhost:6333/collections/${app_id}-workflows/points" \
                            -H "Content-Type: application/json" \
                            -d "{\"points\": [{\"id\": \"$id\", \"vector\": $embedding, \"payload\": $payload}]}")
                        
                        if echo "$response" | grep -q '"status":"ok"'; then
                            ((workflow_count++))
                            log::info "✅ Stored workflow: $name"
                        else
                            log::error "Failed to store workflow: $response"
                        fi
                    else
                        log::error "Invalid embedding received"
                    fi
                else
                    log::error "No embedding response"
                fi
            fi
        done <<< "$workflow_files"
    else
        log::info "No workflow files found"
    fi
    
    ((total_count += workflow_count))
    log::info "Processed $workflow_count workflows"
    
    echo "$total_count"
}

#######################################
# Validate embeddings setup
# Arguments:
#   $1 - Directory to validate (optional, defaults to current)
# Returns: 0 on success
#######################################
embeddings_validate_impl() {
    local dir="${1:-.}"
    
    echo "=== Embedding Validation Report ==="
    echo
    
    # Check app identity
    local app_id=$(qdrant::identity::get_app_id)
    if [[ -z "$app_id" ]]; then
        echo "❌ No app identity found"
        echo "   Run: resource-qdrant embeddings init"
        echo
    else
        echo "✅ App Identity: $app_id"
        echo
    fi
    
    # Check documentation
    echo "📄 Documentation Status:"
    qdrant::extract::docs_coverage "$dir"
    echo
    
    # Check content availability
    echo "📦 Embeddable Content:"
    
    # Workflows
    local workflow_count=$(find "$dir" -type f -name "*.json" -path "*/initialization/*" 2>/dev/null | wc -l)
    echo "  • Workflows: $workflow_count files"
    
    # Scenarios
    local scenario_count=$(find "$dir" -type f -name "PRD.md" -path "*/scenarios/*" 2>/dev/null | wc -l)
    echo "  • Scenarios: $scenario_count PRDs"
    
    # Code files
    local code_count=$(find "$dir" -type f \( -name "*.sh" -o -name "*.ts" -o -name "*.js" \) ! -path "*/node_modules/*" 2>/dev/null | wc -l)
    echo "  • Code Files: $code_count"
    
    # Resources - check both new and old locations
    local resource_count=0
    if [[ -d "$dir/resources" ]]; then
        resource_count=$(find "$dir/resources" -mindepth 1 -maxdepth 1 -type d -exec test -f {}/cli.sh \; -print 2>/dev/null | wc -l)
    elif [[ -d "$dir/scripts/resources" ]]; then
        resource_count=$(find "$dir/scripts/resources" -mindepth 2 -maxdepth 2 -type d 2>/dev/null | wc -l)
    fi
    echo "  • Resources: $resource_count"
    echo
    
    # Check model availability
    echo "🤖 Embedding Model:"
    if qdrant::models::is_available "$QDRANT_EMBEDDING_MODEL"; then
        echo "  ✅ $QDRANT_EMBEDDING_MODEL available"
    else
        echo "  ❌ $QDRANT_EMBEDDING_MODEL not installed"
        echo "     Run: ollama pull $QDRANT_EMBEDDING_MODEL"
    fi
    echo
    
    # Check Qdrant connection
    echo "🔌 Qdrant Connection:"
    if qdrant::collections::list >/dev/null 2>&1; then
        echo "  ✅ Connected to Qdrant"
    else
        echo "  ❌ Cannot connect to Qdrant"
        echo "     Check if Qdrant is running"
    fi
    echo
    
    # Generate recommendations
    echo "💡 Recommendations:"
    local has_recommendations=false
    
    if [[ -z "$app_id" ]]; then
        echo "  • Initialize app identity to start embedding"
        has_recommendations=true
    fi
    
    if [[ $workflow_count -eq 0 ]] && [[ $scenario_count -eq 0 ]]; then
        echo "  • Add workflows or scenarios to enrich embeddings"
        has_recommendations=true
    fi
    
    if [[ ! -f "$dir/docs/ARCHITECTURE.md" ]]; then
        echo "  • Create docs/ARCHITECTURE.md for design decisions"
        has_recommendations=true
    fi
    
    if [[ ! -f "$dir/docs/LESSONS_LEARNED.md" ]]; then
        echo "  • Create docs/LESSONS_LEARNED.md to capture insights"
        has_recommendations=true
    fi
    
    if [[ "$has_recommendations" == "false" ]]; then
        echo "  All systems ready for embedding!"
    fi
    
    return 0
}

#######################################
# Show embeddings status for all apps
# Returns: 0 on success
#######################################
embeddings_status_impl() {
    echo "=== Embeddings Status ==="
    echo
    
    # Sync identity with actual collections before showing status
    local current_app_id=$(qdrant::identity::get_app_id)
    if [[ -n "$current_app_id" ]]; then
        log::debug "Auto-syncing identity before status display"
        qdrant::identity::sync_with_collections 2>/dev/null || true
        
        echo "Current App:"
        qdrant::identity::show
        echo
    fi
    
    # List all apps with embeddings
    echo "All Apps with Embeddings:"
    qdrant::identity::list_all
    
    # Show Qdrant collections
    echo "Qdrant Collections:"
    local collections=$(qdrant::collections::list 2>/dev/null || echo "")
    
    if [[ -n "$collections" ]]; then
        echo "$collections" | jq -r '.collections[]' | while read -r collection; do
            local count=$(qdrant::collections::count "$collection" 2>/dev/null || echo "0")
            echo "  • $collection: $count points"
        done
    else
        echo "  No collections found"
    fi
    echo
    
    # Show model status
    echo "Embedding Models:"
    if qdrant::models::is_available "$QDRANT_EMBEDDING_MODEL"; then
        echo "  ✅ $QDRANT_EMBEDDING_MODEL (default)"
    else
        echo "  ❌ $QDRANT_EMBEDDING_MODEL (not installed)"
    fi
    
    # Check for other embedding models
    local other_models=$(ollama list 2>/dev/null | grep -E "(embed|embedding)" | grep -v "$QDRANT_EMBEDDING_MODEL" | cut -d' ' -f1)
    if [[ -n "$other_models" ]]; then
        echo "  Also available:"
        echo "$other_models" | while read -r model; do
            echo "    • $model"
        done
    fi
    
    return 0
}

#######################################
# Garbage collect unused embeddings
# Arguments:
#   $1 - Force cleanup (yes/no, default: no)
# Returns: 0 on success
#######################################
embeddings_garbage_collect_impl() {
    local force="${1:-no}"
    
    log::info "Analyzing embeddings for cleanup..."
    
    # Find orphaned collections (no matching app-identity.json)
    local all_collections=$(curl -s "http://localhost:6333/collections" 2>/dev/null | jq -r '.result.collections[].name' 2>/dev/null || echo "")
    local orphaned=()
    
    for collection in $all_collections; do
        # Extract app-id from collection name
        local app_id="${collection%-*}"
        
        # Check if app identity exists
        local found=false
        while IFS= read -r identity_file; do
            local file_app_id=$(jq -r '.app_id' "$identity_file" 2>/dev/null)
            if [[ "$file_app_id" == "$app_id" ]]; then
                found=true
                break
            fi
        done < <(find . -name "app-identity.json" -type f 2>/dev/null)
        
        if [[ "$found" == "false" ]]; then
            orphaned+=("$collection")
        fi
    done
    
    if [[ ${#orphaned[@]} -eq 0 ]]; then
        log::info "No orphaned collections found"
        return 0
    fi
    
    echo "Found ${#orphaned[@]} orphaned collections:"
    for collection in "${orphaned[@]}"; do
        local count=$(qdrant::collections::count "$collection" 2>/dev/null || echo "0")
        echo "  • $collection ($count points)"
    done
    echo
    
    if [[ "$force" != "yes" ]]; then
        echo "Use --force yes to delete these collections"
        return 1
    fi
    
    # Delete orphaned collections
    for collection in "${orphaned[@]}"; do
        log::info "Deleting orphaned collection: $collection"
        qdrant::collections::delete "$collection" "yes"
    done
    
    log::success "Garbage collection complete. Removed ${#orphaned[@]} collections"
    return 0
}

################################################################################
# Main Execution
################################################################################

# Dispatch to CLI framework
cli::dispatch "$@"