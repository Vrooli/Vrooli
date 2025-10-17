#!/usr/bin/env bash
# Refresh implementation for Qdrant embeddings
# Orchestrates all extractors to refresh embeddings

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
# Export TEMP_DIR so background jobs can access it
export TEMP_DIR="/tmp/qdrant-embeddings-$$"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source configuration
source "${EMBEDDINGS_DIR}/config/unified.sh"

# Source qdrant libraries
source "${APP_ROOT}/resources/qdrant/lib/collections.sh"
source "${APP_ROOT}/resources/qdrant/lib/embeddings.sh"
source "${APP_ROOT}/resources/qdrant/lib/models.sh"

# Source identity indexer
source "${EMBEDDINGS_DIR}/indexers/identity.sh"

# Source all extractors
for extractor in scenarios docs code resources filetrees initialization; do
    extractor_file="${EMBEDDINGS_DIR}/extractors/${extractor}/main.sh"
    if [[ -f "$extractor_file" ]]; then
        source "$extractor_file"
    fi
done

# Source embedding service
source "${EMBEDDINGS_DIR}/lib/embedding-service.sh"
source "${EMBEDDINGS_DIR}/lib/embedding-service-optimized.sh"

# Track background jobs for cleanup
declare -a REFRESH_BACKGROUND_JOBS=()

# Cleanup function for refresh
cleanup_refresh() {
    local exit_code="${1:-0}"
    
    # Kill all background jobs if they exist
    if [[ ${#REFRESH_BACKGROUND_JOBS[@]} -gt 0 ]]; then
        log::warn "Cleaning up ${#REFRESH_BACKGROUND_JOBS[@]} background jobs..."
        for pid in "${REFRESH_BACKGROUND_JOBS[@]}"; do
            if kill -0 "$pid" 2>/dev/null; then
                kill -TERM "$pid" 2>/dev/null || true
                # Give process time to terminate gracefully
                sleep 1
                # Force kill if still running
                if kill -0 "$pid" 2>/dev/null; then
                    kill -KILL "$pid" 2>/dev/null || true
                fi
            fi
        done
        # Clear the array
        REFRESH_BACKGROUND_JOBS=()
    fi
    
    # Clean up temp directory
    if [[ -n "${TEMP_DIR:-}" ]] && [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
    fi
    
    if [[ "$exit_code" != "0" ]]; then
        log::warn "Refresh process interrupted by user"
    fi
    
    exit "$exit_code"
}

# Signal handlers for refresh
trap 'cleanup_refresh 130' SIGINT   # Ctrl+C
trap 'cleanup_refresh 143' SIGTERM  # Termination
trap 'cleanup_refresh 0' EXIT       # Normal exit

#######################################
# Initialize embeddings for current app
# Arguments:
#   $1 - App ID (optional, auto-detected if not provided)
# Returns: 0 on success
#######################################
embeddings_init_impl() {
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
#   $1 - App ID (optional, current app if not specified)
#   $2 - Force flag (optional, --force)
# Returns: 0 on success
#######################################
embeddings_refresh_impl() {
    local app_id=""
    local force="no"
    local sequential="no"
    local max_workers=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force="yes"
                shift
                ;;
            --sequential)
                sequential="yes"
                shift
                ;;
            --workers)
                max_workers="$2"
                shift 2
                ;;
            *)
                if [[ -z "$app_id" ]]; then
                    app_id="$1"
                fi
                shift
                ;;
        esac
    done
    
    # If no app_id provided, get current app
    if [[ -z "$app_id" ]]; then
        # app_id=$(qdrant::identity::get_app_id)  # Function doesn't exist yet
        app_id="${QDRANT_APP_ID:-}"
    fi
    
    if [[ -z "$app_id" ]]; then
        log::error "No app identity found. Run 'embeddings init' first"
        return 1
    fi
    
    # Check if refresh is needed
    if [[ "$force" != "yes" ]]; then
        # if ! qdrant::identity::needs_reindex; then  # Function doesn't exist yet
        #     log::info "Embeddings are up to date for app: $app_id"
        #     return 0
        # fi
        log::info "Skipping reindex check (identity module not available)"
    fi
    
    log::info "Refreshing embeddings for app: $app_id"
    local start_time=$(date +%s)
    
    # Create temporary directory for extracted content
    mkdir -p "$TEMP_DIR"
    
    # Get collections for this app
    local collections="workflows scenarios docs code resources filetrees"
    
    # Delete existing collections
    log::info "Clearing existing collections..."
    for collection in $collections; do
        local full_collection="${app_id}-${collection}"
        qdrant::collections::delete "$full_collection" "yes" 2>/dev/null || true
    done
    
    # Create fresh collections
    log::info "Creating collections..."
    for collection in $collections; do
        local full_collection="${app_id}-${collection}"
        qdrant::collections::create "$full_collection" "$QDRANT_EMBEDDING_DIMENSIONS" "$QDRANT_EMBEDDING_DISTANCE_METRIC" || {
            log::error "Failed to create collection: $full_collection"
            return 1
        }
    done
    
    # Configure processing mode and workers
    if [[ -n "$max_workers" ]]; then
        export EMBEDDING_MAX_WORKERS="$max_workers"
        log::info "Using custom worker count: $max_workers"
    else
        # Reduce default worker count for better stability
        export EMBEDDING_MAX_WORKERS="${EMBEDDING_MAX_WORKERS:-4}"
        log::info "Using default worker count: $EMBEDDING_MAX_WORKERS"
    fi
    
    # Configure timeout (default 10 minutes per extractor for sequential, 5 for parallel)
    local EXTRACTOR_TIMEOUT
    if [[ "$sequential" == "yes" ]]; then
        EXTRACTOR_TIMEOUT="${EMBEDDING_PROCESSING_TIMEOUT:-600}"
        log::info "Processing content types SEQUENTIALLY (timeout: ${EXTRACTOR_TIMEOUT}s per extractor)..."
        return $(embeddings_refresh_sequential "$app_id" "$EXTRACTOR_TIMEOUT" "$start_time")
    else
        # Increase timeout for parallel processing to handle large files
        EXTRACTOR_TIMEOUT="${EMBEDDING_PROCESSING_TIMEOUT:-600}"
        log::info "Processing content types in PARALLEL (timeout: ${EXTRACTOR_TIMEOUT}s per extractor)..."
    fi
    
    # Create background jobs for each content type
    local pids=()
    local job_names=()
    
    # Process initialization/workflows
    if [[ -d "${APP_ROOT}/initialization" ]]; then
        {
            bash -c "
                # Source all required libraries and setup environment
                source '${APP_ROOT}/scripts/lib/utils/var.sh'
                source '${var_LIB_UTILS_DIR}/log.sh'
                source '${EMBEDDINGS_DIR}/config/unified.sh'
                source '${APP_ROOT}/resources/qdrant/lib/collections.sh'
                source '${APP_ROOT}/resources/qdrant/lib/embeddings.sh'
                source '${APP_ROOT}/resources/qdrant/lib/models.sh'
                source '${EMBEDDINGS_DIR}/lib/embedding-service.sh'
                
                # Set correct working directory and environment
                cd '${APP_ROOT}'
                export TEMP_DIR='${TEMP_DIR}'
                
                # Source and run initialization extractor
                if [[ -f '${EMBEDDINGS_DIR}/extractors/initialization/main.sh' ]]; then
                    source '${EMBEDDINGS_DIR}/extractors/initialization/main.sh'
                    log::info 'Processing initialization workflows...'
                    qdrant::embeddings::process_initialization '$app_id'
                else
                    echo '0'
                fi
            " > "$TEMP_DIR/init_count" 2>&1
        } &
        pids+=($!)
        job_names+=("initialization")
    fi
    
    # Process scenarios
    {
        # Create wrapper script for proper environment
        cat > "$TEMP_DIR/scenarios_wrapper.sh" << 'EOF'
#!/bin/bash
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"
source "${EMBEDDINGS_DIR}/config/unified.sh"
source "${APP_ROOT}/resources/qdrant/lib/collections.sh"
source "${APP_ROOT}/resources/qdrant/lib/embeddings.sh"
source "${APP_ROOT}/resources/qdrant/lib/models.sh"
source "${EMBEDDINGS_DIR}/lib/embedding-service.sh"

cd "${APP_ROOT}"
if [[ -f "${EMBEDDINGS_DIR}/extractors/scenarios/main.sh" ]]; then
    source "${EMBEDDINGS_DIR}/extractors/scenarios/main.sh"
    qdrant::embeddings::process_scenarios "$1"
else
    echo "0"
fi
EOF
        chmod +x "$TEMP_DIR/scenarios_wrapper.sh"
        APP_ROOT="$APP_ROOT" EMBEDDINGS_DIR="$EMBEDDINGS_DIR" TEMP_DIR="$TEMP_DIR" "$TEMP_DIR/scenarios_wrapper.sh" "$app_id" > "$TEMP_DIR/scenario_count" 2>&1
    } &
    pids+=($!)
    job_names+=("scenarios")
    
    # Process documentation
    {
        bash -c "
            # Source all required libraries and setup environment
            source '${APP_ROOT}/scripts/lib/utils/var.sh'
            source '${var_LIB_UTILS_DIR}/log.sh'
            source '${EMBEDDINGS_DIR}/config/unified.sh'
            source '${APP_ROOT}/resources/qdrant/lib/collections.sh'
            source '${APP_ROOT}/resources/qdrant/lib/embeddings.sh'
            source '${APP_ROOT}/resources/qdrant/lib/models.sh'
            source '${EMBEDDINGS_DIR}/lib/embedding-service.sh'
            
            # Set correct working directory and environment
            cd '${APP_ROOT}'
            export TEMP_DIR='${TEMP_DIR}'
            export APP_ROOT='${APP_ROOT}'
            
            # Source parsers and docs extractor
            source '${EMBEDDINGS_DIR}/parsers/markdown.sh'
            if [[ -f '${EMBEDDINGS_DIR}/extractors/docs/main.sh' ]]; then
                source '${EMBEDDINGS_DIR}/extractors/docs/main.sh'
                log::info 'Processing documentation...'
                qdrant::embeddings::process_documentation '$app_id'
            else
                echo '0'
            fi
        " > "$TEMP_DIR/doc_count" 2>&1
    } &
    pids+=($!)
    job_names+=("documentation")
    
    # Process code
    {
        bash -c "
            # Source all required libraries and setup environment
            source '${APP_ROOT}/scripts/lib/utils/var.sh'
            source '${var_LIB_UTILS_DIR}/log.sh'
            source '${EMBEDDINGS_DIR}/config/unified.sh'
            source '${APP_ROOT}/resources/qdrant/lib/collections.sh'
            source '${APP_ROOT}/resources/qdrant/lib/embeddings.sh'
            source '${APP_ROOT}/resources/qdrant/lib/models.sh'
            source '${EMBEDDINGS_DIR}/lib/embedding-service.sh'
            
            # Set correct working directory and environment
            cd '${APP_ROOT}'
            export TEMP_DIR='${TEMP_DIR}'
            export APP_ROOT='${APP_ROOT}'
            export CODE_MAX_FILES='${CODE_MAX_FILES:-10000}'
            
            # Source and run code extractor
            if [[ -f '${EMBEDDINGS_DIR}/extractors/code/main.sh' ]]; then
                source '${EMBEDDINGS_DIR}/extractors/code/main.sh'
                log::info 'Processing code...'
                qdrant::embeddings::process_code '$app_id'
            else
                echo '0'
            fi
        " > "$TEMP_DIR/code_count" 2>&1
    } &
    pids+=($!)
    job_names+=("code")
    
    # Process resources
    {
        bash -c "
            # Source all required libraries and setup environment
            source '${APP_ROOT}/scripts/lib/utils/var.sh'
            source '${var_LIB_UTILS_DIR}/log.sh'
            source '${EMBEDDINGS_DIR}/config/unified.sh'
            source '${APP_ROOT}/resources/qdrant/lib/collections.sh'
            source '${APP_ROOT}/resources/qdrant/lib/embeddings.sh'
            source '${APP_ROOT}/resources/qdrant/lib/models.sh'
            source '${EMBEDDINGS_DIR}/lib/embedding-service.sh'
            
            # Set correct working directory and environment
            cd '${APP_ROOT}'
            export TEMP_DIR='${TEMP_DIR}'
            export APP_ROOT='${APP_ROOT}'
            
            # Source and run resources extractor
            if [[ -f '${EMBEDDINGS_DIR}/extractors/resources/main.sh' ]]; then
                source '${EMBEDDINGS_DIR}/extractors/resources/main.sh'
                log::info 'Processing resources...'
                qdrant::embeddings::process_resources '$app_id'
            else
                echo '0'
            fi
        " > "$TEMP_DIR/resource_count" 2>&1
    } &
    pids+=($!)
    job_names+=("resources")
    
    # Process file trees
    {
        bash -c "
            # Source all required libraries and setup environment
            source '${APP_ROOT}/scripts/lib/utils/var.sh'
            source '${var_LIB_UTILS_DIR}/log.sh'
            source '${EMBEDDINGS_DIR}/config/unified.sh'
            source '${APP_ROOT}/resources/qdrant/lib/collections.sh'
            source '${APP_ROOT}/resources/qdrant/lib/embeddings.sh'
            source '${APP_ROOT}/resources/qdrant/lib/models.sh'
            source '${EMBEDDINGS_DIR}/lib/embedding-service.sh'
            
            # Set correct working directory and environment
            cd '${APP_ROOT}'
            export TEMP_DIR='${TEMP_DIR}'
            export APP_ROOT='${APP_ROOT}'
            
            # Source and run filetrees extractor
            if [[ -f '${EMBEDDINGS_DIR}/extractors/filetrees/main.sh' ]]; then
                source '${EMBEDDINGS_DIR}/extractors/filetrees/main.sh'
                log::info 'Processing file trees...'
                qdrant::embeddings::process_file_trees '$app_id'
            else
                echo '0'
            fi
        " > "$TEMP_DIR/filetrees_count" 2>&1
    } &
    pids+=($!)
    job_names+=("filetrees")
    
    # Start memory monitoring in background
    {
        while kill -0 "${pids[0]}" 2>/dev/null; do
            local mem_usage=$(free | awk '/Mem:/ {printf "%.0f", $3/$2 * 100}')
            local load_avg=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | tr -d ',')
            if [[ $mem_usage -gt ${EMBEDDING_MEMORY_LIMIT:-80} ]]; then
                log::warn "⚠️  High memory usage: ${mem_usage}% (limit: ${EMBEDDING_MEMORY_LIMIT:-80}%)"
            fi
            sleep 5
        done
    } &
    local monitor_pid=$!
    
    # Wait for all jobs to complete with timeout and progress tracking
    local failed_jobs=0
    local completed_jobs=0
    
    for i in "${!pids[@]}"; do
        local pid="${pids[$i]}"
        local job_name="${job_names[$i]}"
        local wait_start=$(date +%s)
        local last_progress_time=$wait_start
        
        log::info "⏳ Waiting for ${job_name} processing... (PID: $pid)"
        
        # Wait with timeout and progress updates
        while kill -0 "$pid" 2>/dev/null; do
            local current_time=$(date +%s)
            local elapsed=$((current_time - wait_start))
            
            # Progress update every 30 seconds
            if [[ $((current_time - last_progress_time)) -gt 30 ]]; then
                log::info "📊 ${job_name}: ${elapsed}s elapsed, still processing..."
                last_progress_time=$current_time
            fi
            
            # Check timeout
            if [[ $elapsed -gt $EXTRACTOR_TIMEOUT ]]; then
                log::warn "⏱️  ${job_name} processing timed out after ${elapsed}s, killing..."
                kill -TERM "$pid" 2>/dev/null || true
                sleep 3
                kill -KILL "$pid" 2>/dev/null || true
                ((failed_jobs++))
                break
            fi
            
            sleep 2
        done
        
        # Check final exit status
        if wait "$pid" 2>/dev/null; then
            local duration=$(($(date +%s) - wait_start))
            log::success "✅ ${job_name} completed successfully in ${duration}s"
            ((completed_jobs++))
        else
            local exit_code=$?
            if [[ $exit_code -ne 143 ]]; then  # 143 = SIGTERM, already logged timeout
                log::error "❌ ${job_name} failed with exit code $exit_code"
                ((failed_jobs++))
            fi
        fi
    done
    
    # Stop memory monitor
    kill $monitor_pid 2>/dev/null || true
    
    # Report results
    if [[ $failed_jobs -eq 0 ]]; then
        log::success "All content types processed successfully"
    else
        log::warn "$failed_jobs out of ${#job_names[@]} content type jobs failed"
    fi
    
    # Calculate total embeddings
    local total_embeddings=0
    for count_file in "$TEMP_DIR"/*_count; do
        if [[ -f "$count_file" ]]; then
            local count=$(tail -1 "$count_file" | grep -oE '[0-9]+' | tail -1 || echo "0")
            total_embeddings=$((total_embeddings + count))
        fi
    done
    
    # Calculate duration
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Update app identity with results
    qdrant::identity::update_after_index "$total_embeddings" "$duration"
    
    log::success "Embedding refresh complete!"
    log::info "App: $app_id"
    log::info "Total Embeddings: $total_embeddings"
    log::info "Duration: ${duration}s"
    
    # Validation
    embeddings_validate_refresh "$app_id"
    
    # Cleanup
    rm -rf "$TEMP_DIR"
    
    return 0
}

#######################################
# Sequential processing implementation - safer but slower
# Arguments:
#   $1 - App ID
#   $2 - Timeout per extractor
#   $3 - Start time
# Returns: 0 on success
#######################################
embeddings_refresh_sequential() {
    local app_id="$1"
    local extractor_timeout="$2"
    local start_time="$3"
    local total_embeddings=0
    local failed_extractors=0
    
    log::info "🔄 Starting sequential processing for app: $app_id"
    
    # Array of extractors to process
    local extractors=()
    local extractor_names=()
    
    # Add extractors that exist
    if [[ -d "${APP_ROOT}/initialization" ]]; then
        extractors+=("initialization")
        extractor_names+=("initialization/workflows")
    fi
    extractors+=("scenarios" "documentation" "code" "resources" "filetrees")
    extractor_names+=("scenarios" "documentation" "code" "resources" "filetrees")
    
    # Process each extractor sequentially
    for i in "${!extractors[@]}"; do
        local extractor="${extractors[$i]}"
        local extractor_name="${extractor_names[$i]}"
        local step_start_time=$(date +%s)
        
        log::info "📦 [$((i+1))/${#extractors[@]}] Processing $extractor_name..."
        
        # Set timeout for this specific extractor
        timeout --preserve-status --signal=TERM --kill-after=10s "${extractor_timeout}s" bash -c "
            source '${APP_ROOT}/scripts/lib/utils/var.sh'
            source '${var_LIB_UTILS_DIR}/log.sh'
            source '${EMBEDDINGS_DIR}/config/unified.sh'
            source '${APP_ROOT}/resources/qdrant/lib/collections.sh'
            source '${APP_ROOT}/resources/qdrant/lib/embeddings.sh'
            source '${APP_ROOT}/resources/qdrant/lib/models.sh'
            
            # Source embedding service for batch processing
            source '${EMBEDDINGS_DIR}/lib/embedding-service.sh'
            
            # Set correct working directory and environment
            cd '${APP_ROOT}'
            export TEMP_DIR='${TEMP_DIR}'
            export APP_ROOT='${APP_ROOT}'
            
            # Source the specific extractor
            if [[ '$extractor' == 'initialization' ]] && [[ -f '${EMBEDDINGS_DIR}/extractors/initialization/main.sh' ]]; then
                source '${EMBEDDINGS_DIR}/extractors/initialization/main.sh'
                qdrant::embeddings::process_initialization '$app_id'
            elif [[ -f '${EMBEDDINGS_DIR}/extractors/$extractor/main.sh' ]]; then
                source '${EMBEDDINGS_DIR}/extractors/$extractor/main.sh'
                case '$extractor' in
                    scenarios) qdrant::embeddings::process_scenarios '$app_id' ;;
                    documentation) qdrant::embeddings::process_documentation '$app_id' ;;
                    code) qdrant::embeddings::process_code '$app_id' ;;
                    resources) qdrant::embeddings::process_resources '$app_id' ;;
                    filetrees) qdrant::embeddings::process_file_trees '$app_id' ;;
                    *) echo '0' ;;
                esac
            else
                echo '0'
            fi
        " > "$TEMP_DIR/${extractor}_count" 2>&1
        
        local exit_code=$?
        local step_duration=$(($(date +%s) - step_start_time))
        
        # Check results
        if [[ $exit_code -eq 0 ]] && [[ -f "$TEMP_DIR/${extractor}_count" ]]; then
            local count=$(tail -1 "$TEMP_DIR/${extractor}_count" | grep -oE '[0-9]+' | tail -1 || echo "0")
            total_embeddings=$((total_embeddings + count))
            log::success "✅ $extractor_name completed: $count embeddings in ${step_duration}s"
        elif [[ $exit_code -eq 124 ]] || [[ $exit_code -eq 137 ]]; then
            log::error "⏱️ $extractor_name timed out after ${extractor_timeout}s"
            ((failed_extractors++))
        else
            log::error "❌ $extractor_name failed with exit code $exit_code"
            ((failed_extractors++))
            
            # Show last few lines of error output for debugging
            if [[ -f "$TEMP_DIR/${extractor}_count" ]]; then
                log::debug "Last error output from $extractor_name:"
                tail -5 "$TEMP_DIR/${extractor}_count" | while read -r line; do
                    log::debug "  $line"
                done
            fi
        fi
        
        # Progress update
        local overall_elapsed=$(($(date +%s) - start_time))
        local remaining=$((${#extractors[@]} - i - 1))
        local avg_time_per_extractor=$((overall_elapsed / (i + 1)))
        local estimated_remaining=$((avg_time_per_extractor * remaining))
        
        log::info "📊 Progress: $((i+1))/${#extractors[@]} extractors complete | ${overall_elapsed}s elapsed | ~${estimated_remaining}s remaining"
    done
    
    # Calculate duration
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Update app identity with results
    qdrant::identity::update_after_index "$total_embeddings" "$duration"
    
    # Report results
    if [[ $failed_extractors -eq 0 ]]; then
        log::success "🎉 Sequential processing complete!"
    else
        log::warn "⚠️ Sequential processing complete with $failed_extractors failed extractors"
    fi
    
    log::info "App: $app_id"
    log::info "Total Embeddings: $total_embeddings"
    log::info "Duration: ${duration}s"
    log::info "Failed Extractors: $failed_extractors"
    
    # Validation
    embeddings_validate_refresh "$app_id"
    
    # Cleanup
    rm -rf "$TEMP_DIR"
    
    return $(( failed_extractors > 0 ? 1 : 0 ))
}

#######################################
# Process initialization workflows
# Arguments:
#   $1 - App ID
# Returns: Number of embeddings created
#######################################
embeddings_process_initialization() {
    local app_id="$1"
    local collection="${app_id}-workflows"
    local count=0
    
    # Source initialization extractor if available
    if [[ -f "${EMBEDDINGS_DIR}/extractors/initialization/main.sh" ]]; then
        source "${EMBEDDINGS_DIR}/extractors/initialization/main.sh"
        if declare -f qdrant::embeddings::process_initialization >/dev/null; then
            count=$(qdrant::embeddings::process_initialization "$app_id")
        fi
    fi
    
    # Fallback to direct workflow processing
    if [[ $count -eq 0 ]] && [[ -d "${APP_ROOT}/initialization/n8n" ]]; then
        log::debug "Using fallback workflow processing"
        
        # Process n8n workflows
        for workflow_file in "${APP_ROOT}/initialization/n8n"/*.json; do
            if [[ -f "$workflow_file" ]]; then
                local name=$(jq -r '.name // "Unnamed"' "$workflow_file" 2>/dev/null || echo "Unnamed")
                local content="N8n workflow: $name - $(jq -r '.description // "No description"' "$workflow_file" 2>/dev/null || echo "")"
                
                # Create metadata
                local metadata=$(jq -n \
                    --arg name "$name" \
                    --arg file "$workflow_file" \
                    '{name: $name, file: $file}')
                
                # Process through embedding service
                if qdrant::embedding::process_item "$content" "workflow" "$collection" "$app_id" "$metadata"; then
                    ((count++))
                fi
            fi
        done
    fi
    
    echo "$count"
}

#######################################
# Validate refresh results
# Arguments:
#   $1 - App ID
# Returns: 0 on success, 1 on validation failure
#######################################
embeddings_validate_refresh() {
    local app_id="$1"
    local has_issues=false
    
    log::info "Validating refresh results..."
    
    # Check each collection
    for collection_type in workflows scenarios docs code resources filetrees; do
        local collection="${app_id}-${collection_type}"
        local count=$(qdrant::collections::count "$collection" 2>/dev/null || echo "0")
        
        if [[ "$count" -eq 0 ]]; then
            log::warn "⚠️  Collection $collection_type has 0 embeddings"
            has_issues=true
        else
            log::debug "✓ Collection $collection_type has $count embeddings"
        fi
    done
    
    if [[ "$has_issues" == "true" ]]; then
        log::warn "Some collections may need attention"
        return 1
    else
        log::success "All collections have embeddings"
        return 0
    fi
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
        echo "   Run: resource-qdrant-embeddings init"
        echo
    else
        echo "✅ App Identity: $app_id"
        echo
    fi
    
    # Check content availability
    echo "📦 Embeddable Content:"
    
    # Workflows
    local workflow_count=$(find "$dir" -type f -name "*.json" -path "*/initialization/*" 2>/dev/null | wc -l)
    echo "  • Workflows: $workflow_count files"
    
    # Scenarios
    local scenario_count=$(find "$dir" -type f -name "PRD.md" -path "*/scenarios/*" 2>/dev/null | wc -l)
    echo "  • Scenarios: $scenario_count PRDs"
    
    # Documentation
    local doc_count=$(find "$dir" -type f -name "*.md" ! -path "*/node_modules/*" ! -path "*/.git/*" 2>/dev/null | wc -l)
    echo "  • Documentation: $doc_count files"
    
    # Code files
    local code_count=$(find "$dir" -type f \( -name "*.sh" -o -name "*.ts" -o -name "*.js" -o -name "*.py" \) ! -path "*/node_modules/*" 2>/dev/null | wc -l)
    echo "  • Code Files: $code_count"
    
    # Resources
    local resource_count=0
    if [[ -d "$dir/resources" ]]; then
        resource_count=$(find "$dir/resources" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l)
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
    
    return 0
}

# Export functions
export -f embeddings_init_impl
export -f embeddings_refresh_impl
export -f embeddings_process_initialization
export -f embeddings_validate_refresh
export -f embeddings_validate_impl

# Main execution block - run if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Parse command and execute
    case "${1:-}" in
        init)
            shift
            embeddings_init_impl "$@"
            ;;
        refresh)
            shift
            embeddings_refresh_impl "$@"
            ;;
        validate)
            shift
            embeddings_validate_impl "$@"
            ;;
        *)
            # Default to refresh if no command specified
            embeddings_refresh_impl "$@"
            ;;
    esac
fi