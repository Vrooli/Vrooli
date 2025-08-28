#!/usr/bin/env bash
# Main Embedding Management Orchestrator for Qdrant
# Coordinates all embedding operations: refresh, validate, search, status
#
# ⚠️ DEPRECATION NOTICE: This script is deprecated as of v2.0 (January 2025)
# Please use cli.sh instead: resource-qdrant-embeddings <command>
# This file will be removed in v3.0 (target: December 2025)
#
# Migration examples:
#   OLD: ./manage.sh init
#   NEW: ./cli.sh init  OR  resource-qdrant-embeddings init
#
#   OLD: ./manage.sh refresh
#   NEW: ./cli.sh refresh  OR  resource-qdrant-embeddings refresh

set -euo pipefail

# Note: This file is sourced by cli.sh - direct execution is deprecated
# Users should use: ./resources/qdrant/cli.sh embeddings <command>
# This internal routing will be refactored in v3.0 (December 2025)

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Define paths from APP_ROOT
QDRANT_DIR="${APP_ROOT}/resources/qdrant"
EMBEDDINGS_DIR="${QDRANT_DIR}/embeddings"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"
# source "${var_LIB_UTILS_DIR}/validation.sh"  # Not used, commented out

# Source unified configuration first (must come before main qdrant libs to avoid readonly conflicts)
source "${EMBEDDINGS_DIR}/config/unified.sh"
# Export configuration (ensures all variables are available)
qdrant::embeddings::export_config

# Source qdrant libraries (after unified config to allow inheritance)
source "${QDRANT_DIR}/lib/collections.sh"
source "${QDRANT_DIR}/lib/embeddings.sh"
source "${QDRANT_DIR}/lib/models.sh"

# Source embedding components
source "${EMBEDDINGS_DIR}/indexers/identity.sh"
# workflows.sh REMOVED - now processed via resources stream (initialization system)
source "${EMBEDDINGS_DIR}/extractors/scenarios/main.sh"
source "${EMBEDDINGS_DIR}/extractors/docs/main.sh"
source "${EMBEDDINGS_DIR}/extractors/code/main.sh"
source "${EMBEDDINGS_DIR}/extractors/resources/main.sh"
source "${EMBEDDINGS_DIR}/extractors/file-trees/main.sh"
source "${EMBEDDINGS_DIR}/search/single.sh"
source "${EMBEDDINGS_DIR}/search/multi.sh"

# Configuration now loaded from unified.sh
# All settings available via QDRANT_* variables with backward-compatible aliases
TEMP_DIR="/tmp/qdrant-embeddings-$$"

# Cleanup on exit
trap "rm -rf $TEMP_DIR" EXIT

#######################################
# Initialize embeddings for current app
# Arguments:
#   $1 - App ID (optional, auto-detected if not provided)
# Returns: 0 on success
#######################################
qdrant::embeddings::init() {
    local app_id="${1:-}"
    
    log::info "Initializing embeddings for project..."
    
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
qdrant::embeddings::refresh() {
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
                if [[ -n "$2" ]]; then
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
    local start_time=$(date +%s)
    
    # Create temporary directory for extracted content
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
        total_embeddings=$(qdrant::embeddings::refresh_sequential "$app_id")
    else
        log::info "Processing all content types in parallel with ${EMBEDDING_MAX_WORKERS:-16} workers..."
    
    # Create background jobs for each content type
    # NOTE: Workflows are now processed by resources stream via initialization system
    local scenario_count_file="$TEMP_DIR/scenario_count"
    local doc_count_file="$TEMP_DIR/doc_count"
    local code_count_file="$TEMP_DIR/code_count"
    local resource_count_file="$TEMP_DIR/resource_count"
    local filetrees_count_file="$TEMP_DIR/filetrees_count"
    
    # Start all content type processing in parallel
    
    {
        log::info "Processing scenarios..."
        scenario_count=$(qdrant::embeddings::process_scenarios "$app_id")
        echo "$scenario_count" > "$scenario_count_file"
    } &
    local scenario_pid=$!
    
    {
        log::info "Processing documentation..."
        doc_count=$(qdrant::embeddings::process_documentation "$app_id")
        echo "$doc_count" > "$doc_count_file"
    } &
    local doc_pid=$!
    
    {
        log::info "Processing code..."
        code_count=$(qdrant::embeddings::process_code "$app_id")
        echo "$code_count" > "$code_count_file"
    } &
    local code_pid=$!
    
    {
        log::info "Processing resources..."
        resource_count=$(qdrant::embeddings::process_resources "$app_id")
        echo "$resource_count" > "$resource_count_file"
    } &
    local resource_pid=$!
    
    {
        log::info "Processing file trees..."
        filetrees_count=$(qdrant::embeddings::process_file_trees "$app_id")
        echo "$filetrees_count" > "$filetrees_count_file"
    } &
    local filetrees_pid=$!
    
    # Wait for all background jobs to complete with monitoring
    log::info "Waiting for all content type processing to complete..."
    
    # Monitor memory usage during parallel processing
    {
        while kill -0 $scenario_pid $doc_pid $code_pid $resource_pid $filetrees_pid 2>/dev/null; do
            local mem_usage
            mem_usage=$(free | awk '/Mem:/ {printf "%.0f", $3/$2 * 100}')
            if [[ $mem_usage -gt 85 ]]; then
                log::warn "High memory usage detected: ${mem_usage}% - Consider reducing parallel workers"
            fi
            sleep 2
        done
    } &
    local monitor_pid=$!
    
    # Wait for all jobs with timeout and error handling
    local failed_jobs=0
    local job_names=("scenarios" "documentation" "code" "resources" "file-trees")
    local job_pids=($scenario_pid $doc_pid $code_pid $resource_pid $filetrees_pid)
    
    for i in "${!job_pids[@]}"; do
        local pid="${job_pids[$i]}"
        local job_name="${job_names[$i]}"
        
        if wait "$pid"; then
            log::info "✅ ${job_name} processing completed successfully"
        else
            log::error "❌ ${job_name} processing failed"
            ((failed_jobs++))
        fi
    done
    
    # Stop memory monitor
    kill $monitor_pid 2>/dev/null || true
    
    # Report parallel processing results
    if [[ $failed_jobs -eq 0 ]]; then
        log::success "All content types processed successfully in parallel"
    else
        log::warn "$failed_jobs out of ${#job_names[@]} content type jobs failed"
    fi
    
    # Read results from temporary files
    # NOTE: Workflow embeddings are now included in resource_count via initialization system
    local scenario_count=$(cat "$scenario_count_file" 2>/dev/null || echo "0")
    local doc_count=$(cat "$doc_count_file" 2>/dev/null || echo "0")
    local code_count=$(cat "$code_count_file" 2>/dev/null || echo "0")
    local resource_count=$(cat "$resource_count_file" 2>/dev/null || echo "0")
    local filetrees_count=$(cat "$filetrees_count_file" 2>/dev/null || echo "0")
    
    # Calculate total
    total_embeddings=$((scenario_count + doc_count + code_count + resource_count + filetrees_count))
    
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
qdrant::embeddings::refresh_sequential() {
    local app_id="$1"
    local total_count=0
    
    log::info "Starting sequential processing for safety..."
    
    # Process workflows directly with API calls
    log::info "Processing workflows..."
    local workflow_count=0
    
    # Find workflow files using a simpler approach
    local workflow_files
    workflow_files=$(find "${APP_ROOT:-/home/matthalloran8/Vrooli}/initialization/n8n" -name "*.json" -type f 2>/dev/null | head -3)
    
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

# REMOVED: qdrant::embeddings::process_workflows() function
# This function is DEPRECATED - workflows now processed by resources/main.sh via initialization system

# REMOVED: qdrant::embeddings::process_scenarios() function
# This function is now provided by extractors/scenarios/main.sh which is sourced above

# REMOVED: qdrant::embeddings::process_documentation() function
# This function is now provided by extractors/docs/main.sh which is sourced above

# REMOVED: qdrant::embeddings::process_code() function
# This function is now provided by extractors/code/main.sh which is sourced above

# REMOVED: qdrant::embeddings::process_resources() function
# This function is now provided by extractors/resources/main.sh which is sourced above

#######################################
# Generate embedding for text
# Arguments:
#   $1 - Text content
#   $2 - Model name (optional)
# Returns: Embedding vector as JSON array
#######################################
# REMOVED: qdrant::embeddings::generate() function
# This function is now provided by lib/embeddings.sh which is sourced above
# The lib/embeddings.sh version correctly uses the Ollama API instead of non-existent 'ollama embed' command

#######################################
# Validate embeddings setup
# Arguments:
#   $1 - Directory to validate (optional, defaults to current)
# Returns: 0 on success
#######################################
qdrant::embeddings::validate() {
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
qdrant::embeddings::status() {
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
qdrant::embeddings::garbage_collect() {
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

#######################################
# Show help for embeddings commands
# Returns: 0
#######################################
qdrant::embeddings::show_help() {
    cat << EOF
Qdrant Embeddings Management

Commands:
  init [app-id]              Initialize embeddings for current project
  refresh [app-id] [--force] Refresh all embeddings for an app
  validate [directory]       Validate embedding setup and coverage
  status                     Show status of all embedding systems
  gc [--force]              Garbage collect orphaned embeddings
  search <query> [options]   Search within current app
  search-all <query> [type]  Search across all apps
  patterns <query>           Discover patterns across apps
  solutions <problem>        Find reusable solutions
  gaps <topic>              Analyze knowledge gaps
  sync                      Sync identity file with actual collection state
  
Options:
  --app-id ID               Specify app ID (auto-detected if not provided)
  --force yes              Force operation without confirmation
  --model MODEL            Embedding model (default: $QDRANT_EMBEDDING_MODEL)
  --type TYPE              Filter by type (all/workflows/scenarios/knowledge/code/resources)
  --limit N                Maximum results (default: 10)
  
Examples:
  resource-qdrant embeddings init
  resource-qdrant embeddings refresh --force yes
  resource-qdrant embeddings search "send email"
  resource-qdrant embeddings search-all "webhook processing" --type workflows
  resource-qdrant embeddings patterns "authentication"
  resource-qdrant embeddings solutions "image processing"
  resource-qdrant embeddings gaps "security"
  
For more information, see:
  resources/qdrant/embeddings/README.md
EOF
}

# Main dispatcher (when sourced by CLI)
qdrant_embeddings_dispatch() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        init)
            qdrant::embeddings::init "$@"
            ;;
        refresh)
            qdrant::embeddings::refresh "$@"
            ;;
        validate)
            qdrant::embeddings::validate "$@"
            ;;
        status)
            qdrant::embeddings::status
            ;;
        gc|garbage-collect)
            qdrant::embeddings::garbage_collect "$@"
            ;;
        search)
            local query="$1"
            shift || true
            
            # Parse --type flag
            local type="all"
            local remaining_args=()
            while [[ $# -gt 0 ]]; do
                case $1 in
                    --type)
                        type="$2"
                        shift 2
                        ;;
                    *)
                        remaining_args+=("$1")
                        shift
                        ;;
                esac
            done
            
            local app_id=$(qdrant::identity::get_app_id)
            if [[ -z "$app_id" ]]; then
                echo '{"error": "No app identity found. Run embeddings init first"}'
                return 1
            fi
            # Return raw JSON instead of formatted text
            qdrant::search::single_app "$query" "$app_id" "$type" "${remaining_args[@]}"
            ;;
        search-all)
            local query="$1"
            shift || true
            local type="${1:-all}"
            # Return JSON format for agents
            qdrant::search::report "$query" "json"
            ;;
        patterns)
            local query="$1"
            qdrant::search::discover_patterns "$query"
            ;;
        solutions)
            local problem="$1"
            qdrant::search::find_solutions "$problem"
            ;;
        gaps)
            local topic="$1"
            qdrant::search::find_gaps "$topic"
            ;;
        sync)
            qdrant::identity::sync_with_collections
            ;;
        explore)
            qdrant::search::explore
            ;;
        help|--help|-h)
            qdrant::embeddings::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            qdrant::embeddings::show_help
            return 1
            ;;
    esac
}

# Wrapper function for CLI integration
# The CLI expects embeddings::main to be available
embeddings::main() {
    qdrant_embeddings_dispatch "$@"
}