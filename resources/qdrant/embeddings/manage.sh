#!/usr/bin/env bash
# Main Embedding Management Orchestrator for Qdrant
# Coordinates all embedding operations: refresh, validate, search, status

set -euo pipefail

# Get directory of this script
EMBEDDINGS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
QDRANT_DIR="$(dirname "$EMBEDDINGS_DIR")"

# Source required utilities
source "${QDRANT_DIR}/../../scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"
# source "${var_LIB_UTILS_DIR}/validation.sh"  # Not used, commented out

# Source qdrant libraries
source "${QDRANT_DIR}/lib/collections.sh"
source "${QDRANT_DIR}/lib/embeddings.sh"
source "${QDRANT_DIR}/lib/models.sh"

# Source embedding components
source "${EMBEDDINGS_DIR}/indexers/identity.sh"
source "${EMBEDDINGS_DIR}/extractors/workflows.sh"
source "${EMBEDDINGS_DIR}/extractors/scenarios.sh"
source "${EMBEDDINGS_DIR}/extractors/docs.sh"
source "${EMBEDDINGS_DIR}/extractors/code.sh"
source "${EMBEDDINGS_DIR}/extractors/resources.sh"
source "${EMBEDDINGS_DIR}/search/single.sh"
source "${EMBEDDINGS_DIR}/search/multi.sh"

# Default settings optimized for parallel processing
DEFAULT_MODEL="mxbai-embed-large"
DEFAULT_DIMENSIONS=1024
BATCH_SIZE=${QDRANT_EMBEDDING_BATCH_SIZE:-50}  # Increased from 10 to 50 for better throughput
MAX_PARALLEL_WORKERS=${QDRANT_MAX_WORKERS:-16}  # Match CPU cores
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
        qdrant::collections::create "$collection" "$DEFAULT_DIMENSIONS" "Cosine" || {
                log::error "Failed to create collection: $collection"
                return 1
            }
    done
    
    # Extract and embed content by type (PARALLEL PROCESSING)
    local total_embeddings=0
    
    log::info "Processing all content types in parallel with $MAX_PARALLEL_WORKERS workers..."
    
    # Create background jobs for each content type
    local workflow_count_file="$TEMP_DIR/workflow_count"
    local scenario_count_file="$TEMP_DIR/scenario_count"
    local doc_count_file="$TEMP_DIR/doc_count"
    local code_count_file="$TEMP_DIR/code_count"
    local resource_count_file="$TEMP_DIR/resource_count"
    
    # Start all content type processing in parallel
    {
        log::info "Processing workflows..."
        workflow_count=$(qdrant::embeddings::process_workflows "$app_id")
        echo "$workflow_count" > "$workflow_count_file"
    } &
    local workflow_pid=$!
    
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
    
    # Wait for all background jobs to complete with monitoring
    log::info "Waiting for all content type processing to complete..."
    
    # Monitor memory usage during parallel processing
    {
        while kill -0 $workflow_pid $scenario_pid $doc_pid $code_pid $resource_pid 2>/dev/null; do
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
    local job_names=("workflows" "scenarios" "documentation" "code" "resources")
    local job_pids=($workflow_pid $scenario_pid $doc_pid $code_pid $resource_pid)
    
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
    local workflow_count=$(cat "$workflow_count_file" 2>/dev/null || echo "0")
    local scenario_count=$(cat "$scenario_count_file" 2>/dev/null || echo "0")
    local doc_count=$(cat "$doc_count_file" 2>/dev/null || echo "0")
    local code_count=$(cat "$code_count_file" 2>/dev/null || echo "0")
    local resource_count=$(cat "$resource_count_file" 2>/dev/null || echo "0")
    
    # Calculate total
    total_embeddings=$((workflow_count + scenario_count + doc_count + code_count + resource_count))
    
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
# Process and embed workflows
# Arguments:
#   $1 - App ID
# Returns: Number of embeddings created
#######################################
qdrant::embeddings::process_workflows() {
    local app_id="$1"
    local collection="${app_id}-workflows"
    local count=0
    
    # Extract workflows
    local output_file="$TEMP_DIR/workflows.txt"
    qdrant::extract::workflows_batch "." "$output_file" >&2
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No workflows found"
        echo "0"
        return 0
    fi
    
    # Check if parallel processing is available
    if [[ -f "${EMBEDDINGS_DIR}/lib/parallel.sh" ]] && command -v xargs >/dev/null 2>&1; then
        # Use parallel processing
        source "${EMBEDDINGS_DIR}/lib/parallel.sh"
        count=$(qdrant::parallel::process_batch "$output_file" "qdrant::parallel::process_workflow_chunk" "$collection" "$app_id")
    else
        # Fall back to sequential processing
        local content=""
        local workflow_id=""
        
        while IFS= read -r line; do
            if [[ "$line" == "---SEPARATOR---" ]]; then
                if [[ -n "$content" ]]; then
                    # Generate embedding
                    local embedding=$(qdrant::embeddings::generate "$content" "$DEFAULT_MODEL")
                    
                    if [[ -n "$embedding" ]]; then
                        # Create unique ID from content hash
                        workflow_id=$(echo -n "$content" | sha256sum | cut -d' ' -f1)
                        
                        # Store in collection
                        qdrant::collections::upsert_point "$collection" \
                            "$workflow_id" \
                            "$embedding" \
                            "{\"content\": $(echo "$content" | jq -Rs .), \"type\": \"workflow\", \"app_id\": \"$app_id\"}"
                        
                        ((count++))
                    fi
                fi
                content=""
            else
            content="${content}${line}"$'\n'
        fi
    done < "$output_file"
    fi
    
    log::debug "Created $count workflow embeddings"
    echo "$count"
}

#######################################
# Process and embed scenarios
# Arguments:
#   $1 - App ID
# Returns: Number of embeddings created
#######################################
qdrant::embeddings::process_scenarios() {
    local app_id="$1"
    local collection="${app_id}-scenarios"
    local count=0
    
    # Extract scenarios
    local output_file="$TEMP_DIR/scenarios.txt"
    qdrant::extract::scenarios_batch "." "$output_file" >&2
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No scenarios found"
        echo "0"
        return 0
    fi
    
    # Process each scenario
    local content=""
    
    while IFS= read -r line; do
        if [[ "$line" == "---SEPARATOR---" ]]; then
            if [[ -n "$content" ]]; then
                # Generate embedding
                local embedding=$(qdrant::embeddings::generate "$content" "$DEFAULT_MODEL")
                
                if [[ -n "$embedding" ]]; then
                    # Create unique ID from content hash
                    local scenario_id=$(echo -n "$content" | sha256sum | cut -d' ' -f1)
                    
                    # Store in collection
                    qdrant::collections::upsert_point "$collection" \
                        "$scenario_id" \
                        "$embedding" \
                        "{\"content\": $(echo "$content" | jq -Rs .), \"type\": \"scenario\", \"app_id\": \"$app_id\"}"
                    
                    ((count++))
                fi
            fi
            content=""
        else
            content="${content}${line}"$'\n'
        fi
    done < "$output_file"
    
    log::debug "Created $count scenario embeddings"
    echo "$count"
}

#######################################
# Process and embed documentation
# Arguments:
#   $1 - App ID
# Returns: Number of embeddings created
#######################################
qdrant::embeddings::process_documentation() {
    local app_id="$1"
    local collection="${app_id}-knowledge"
    local count=0
    
    # Extract documentation
    local output_file="$TEMP_DIR/docs.txt"
    qdrant::extract::docs_batch "." "$output_file" >&2
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No documentation found"
        echo "0"
        return 0
    fi
    
    # Process each document section
    local content=""
    
    while IFS= read -r line; do
        if [[ "$line" == "---SEPARATOR---" ]] || [[ "$line" == "---SECTION---" ]]; then
            if [[ -n "$content" ]]; then
                # Generate embedding
                local embedding=$(qdrant::embeddings::generate "$content" "$DEFAULT_MODEL")
                
                if [[ -n "$embedding" ]]; then
                    # Create unique ID from content hash
                    local doc_id=$(echo -n "$content" | sha256sum | cut -d' ' -f1)
                    
                    # Store in collection
                    qdrant::collections::upsert_point "$collection" \
                        "$doc_id" \
                        "$embedding" \
                        "{\"content\": $(echo "$content" | jq -Rs .), \"type\": \"documentation\", \"app_id\": \"$app_id\"}"
                    
                    ((count++))
                fi
            fi
            content=""
        else
            content="${content}${line}"$'\n'
        fi
    done < "$output_file"
    
    log::debug "Created $count documentation embeddings"
    echo "$count"
}

#######################################
# Process and embed code
# Arguments:
#   $1 - App ID
# Returns: Number of embeddings created
#######################################
qdrant::embeddings::process_code() {
    local app_id="$1"
    local collection="${app_id}-code"
    local count=0
    
    # Extract code
    local output_file="$TEMP_DIR/code.txt"
    qdrant::extract::code_batch "." "$output_file" >&2
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No code found"
        echo "0"
        return 0
    fi
    
    # Process each code element
    local content=""
    
    while IFS= read -r line; do
        if [[ "$line" == "---SEPARATOR---" ]] || [[ "$line" =~ ^---(FUNCTION|ENDPOINT|COMMAND|QUERY|PATTERN)---$ ]]; then
            if [[ -n "$content" ]]; then
                # Generate embedding
                local embedding=$(qdrant::embeddings::generate "$content" "$DEFAULT_MODEL")
                
                if [[ -n "$embedding" ]]; then
                    # Create unique ID from content hash
                    local code_id=$(echo -n "$content" | sha256sum | cut -d' ' -f1)
                    
                    # Store in collection
                    qdrant::collections::upsert_point "$collection" \
                        "$code_id" \
                        "$embedding" \
                        "{\"content\": $(echo "$content" | jq -Rs .), \"type\": \"code\", \"app_id\": \"$app_id\"}"
                    
                    ((count++))
                fi
            fi
            content=""
        else
            content="${content}${line}"$'\n'
        fi
    done < "$output_file"
    
    log::debug "Created $count code embeddings"
    echo "$count"
}

#######################################
# Process and embed resources
# Arguments:
#   $1 - App ID
# Returns: Number of embeddings created
#######################################
qdrant::embeddings::process_resources() {
    local app_id="$1"
    local collection="${app_id}-resources"
    local count=0
    
    # Extract resources
    local output_file="$TEMP_DIR/resources.txt"
    qdrant::extract::resources_batch "." "$output_file" >&2
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No resources found"
        echo "0"
        return 0
    fi
    
    # Process each resource
    local content=""
    
    while IFS= read -r line; do
        if [[ "$line" == "---SEPARATOR---" ]]; then
            if [[ -n "$content" ]]; then
                # Generate embedding
                local embedding=$(qdrant::embeddings::generate "$content" "$DEFAULT_MODEL")
                
                if [[ -n "$embedding" ]]; then
                    # Create unique ID from content hash
                    local resource_id=$(echo -n "$content" | sha256sum | cut -d' ' -f1)
                    
                    # Store in collection
                    qdrant::collections::upsert_point "$collection" \
                        "$resource_id" \
                        "$embedding" \
                        "{\"content\": $(echo "$content" | jq -Rs .), \"type\": \"resource\", \"app_id\": \"$app_id\"}"
                    
                    ((count++))
                fi
            fi
            content=""
        else
            content="${content}${line}"$'\n'
        fi
    done < "$output_file"
    
    log::debug "Created $count resource embeddings"
    echo "$count"
}

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
    if qdrant::models::is_available "$DEFAULT_MODEL"; then
        echo "  ✅ $DEFAULT_MODEL available"
    else
        echo "  ❌ $DEFAULT_MODEL not installed"
        echo "     Run: ollama pull $DEFAULT_MODEL"
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
    
    # Show current app status
    local current_app_id=$(qdrant::identity::get_app_id)
    if [[ -n "$current_app_id" ]]; then
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
    if qdrant::models::is_available "$DEFAULT_MODEL"; then
        echo "  ✅ $DEFAULT_MODEL (default)"
    else
        echo "  ❌ $DEFAULT_MODEL (not installed)"
    fi
    
    # Check for other embedding models
    local other_models=$(ollama list 2>/dev/null | grep -E "(embed|embedding)" | grep -v "$DEFAULT_MODEL" | cut -d' ' -f1)
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
    local all_collections=$(qdrant::collections::list 2>/dev/null | jq -r '.collections[]' || echo "")
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
  
Options:
  --app-id ID               Specify app ID (auto-detected if not provided)
  --force yes              Force operation without confirmation
  --model MODEL            Embedding model (default: $DEFAULT_MODEL)
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
  scripts/resources/storage/qdrant/embeddings/README.md
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
            local app_id=$(qdrant::identity::get_app_id)
            if [[ -z "$app_id" ]]; then
                log::error "No app identity found. Run 'embeddings init' first"
                return 1
            fi
            qdrant::search::explain "$query" "$app_id" "$@"
            ;;
        search-all)
            local query="$1"
            shift || true
            local type="${1:-all}"
            qdrant::search::report "$query" "text"
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