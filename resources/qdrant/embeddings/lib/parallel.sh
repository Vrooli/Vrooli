#!/usr/bin/env bash
# Parallel processing utilities for embedding generation

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
PARALLEL_DIR="${APP_ROOT}/resources/qdrant/embeddings/lib"
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Maximum parallel workers (can be overridden)
# Increased to 16 to match CPU cores and Ollama parallel capacity
MAX_WORKERS="${QDRANT_MAX_WORKERS:-16}"

# Auto-detect CPU cores if MAX_WORKERS is 0
if [[ "$MAX_WORKERS" -eq 0 ]]; then
    MAX_WORKERS=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
    log::debug "Auto-detected $MAX_WORKERS CPU cores for parallel processing"
fi

# Ensure MAX_WORKERS is at least 1
if [[ "$MAX_WORKERS" -lt 1 ]]; then
    MAX_WORKERS=1
fi

#######################################
# Process files in parallel batches
# Arguments:
#   $1 - Input file with list of items
#   $2 - Processing function name
#   $3 - Collection name
#   $4 - App ID
# Returns: Number of items processed
#######################################
qdrant::parallel::process_batch() {
    local input_file="$1"
    local process_func="$2"
    local collection="$3"
    local app_id="$4"
    
    if [[ ! -f "$input_file" ]]; then
        echo "0"
        return 0
    fi
    
    # Create temp directory for parallel processing
    local batch_dir="/tmp/qdrant-parallel-$$"
    mkdir -p "$batch_dir"
    trap "rm -rf $batch_dir" EXIT
    
    # Split input into chunks for parallel processing
    local total_lines=$(wc -l < "$input_file")
    local chunk_size=$((total_lines / MAX_WORKERS + 1))
    
    # Split file into chunks
    split -l "$chunk_size" "$input_file" "$batch_dir/chunk_"
    
    # Process chunks in parallel
    local pids=()
    local chunk_num=0
    
    for chunk_file in "$batch_dir"/chunk_*; do
        {
            # Process chunk
            "$process_func" "$chunk_file" "$collection" "$app_id" > "$batch_dir/result_$chunk_num" 2>&1
        } &
        pids+=($!)
        ((chunk_num++))
    done
    
    # Wait for all parallel jobs to complete with enhanced error handling
    local failed=0
    local completed=0
    local total_jobs=${#pids[@]}
    
    for pid in "${pids[@]}"; do
        if wait "$pid"; then
            ((completed++))
            log::debug "Parallel job $pid completed successfully ($completed/$total_jobs)"
        else
            local exit_code=$?
            ((failed++))
            log::warn "Parallel job $pid failed with exit code $exit_code ($failed failures so far)"
            
            # If too many jobs are failing, consider reducing parallelism
            if [[ $failed -gt $((total_jobs / 2)) ]]; then
                log::error "More than 50% of parallel jobs failing - consider reducing MAX_WORKERS"
            fi
        fi
    done
    
    # Performance metrics
    local success_rate=$((completed * 100 / total_jobs))
    log::info "Parallel processing completed: $completed successful, $failed failed (${success_rate}% success rate)"
    
    # Aggregate results
    local total_processed=0
    for result_file in "$batch_dir"/result_*; do
        if [[ -f "$result_file" ]]; then
            local count=$(tail -1 "$result_file" | grep -oE '[0-9]+' | tail -1 || echo "0")
            total_processed=$((total_processed + count))
        fi
    done
    
    if [[ $failed -gt 0 ]]; then
        log::warn "Some parallel jobs failed: $failed/$chunk_num"
    fi
    
    echo "$total_processed"
    return 0
}

#######################################
# Process workflow chunk
# Arguments:
#   $1 - Chunk file
#   $2 - Collection name
#   $3 - App ID
# Returns: Number of embeddings created
#######################################
qdrant::parallel::process_workflow_chunk() {
    local chunk_file="$1"
    local collection="$2"
    local app_id="$3"
    local count=0
    
    # Source required functions
    source "${EMBEDDINGS_DIR}/lib/embeddings.sh"
    source "${EMBEDDINGS_DIR}/../lib/collections.sh"
    
    # Process each workflow in chunk
    local content=""
    local in_workflow=false
    
    while IFS= read -r line; do
        if [[ "$line" == "=== N8n Workflow ===" ]]; then
            in_workflow=true
            content=""
        elif [[ "$line" == "---SEPARATOR---" ]] && [[ "$in_workflow" == true ]]; then
            if [[ -n "$content" ]]; then
                # Generate embedding
                local embedding=$(qdrant::embeddings::generate "$content" "mxbai-embed-large" 2>/dev/null)
                
                if [[ -n "$embedding" ]]; then
                    # Create unique ID from content hash
                    local workflow_id=$(echo -n "$content" | sha256sum | cut -d' ' -f1)
                    
                    # Store in collection
                    if qdrant::collections::upsert_point "$collection" \
                        "$workflow_id" \
                        "$embedding" \
                        "{\"content\": $(echo "$content" | jq -Rs .), \"type\": \"workflow\", \"app_id\": \"$app_id\"}" 2>/dev/null; then
                        ((count++))
                    fi
                fi
            fi
            in_workflow=false
            content=""
        elif [[ "$in_workflow" == true ]]; then
            content="${content}${line}
"
        fi
    done < "$chunk_file"
    
    echo "$count"
    return 0
}

#######################################
# Export functions for parallel execution
#######################################
export -f qdrant::parallel::process_workflow_chunk

#######################################
# Process scenarios chunk (similar to workflows)
#######################################
qdrant::parallel::process_scenario_chunk() {
    local chunk_file="$1"
    local collection="$2"
    local app_id="$3"
    local count=0
    
    # Source required functions
    source "${EMBEDDINGS_DIR}/lib/embeddings.sh"
    source "${EMBEDDINGS_DIR}/../lib/collections.sh"
    
    # Process scenarios similarly to workflows
    local content=""
    local in_scenario=false
    
    while IFS= read -r line; do
        if [[ "$line" == "=== Scenario ===" ]]; then
            in_scenario=true
            content=""
        elif [[ "$line" == "---SEPARATOR---" ]] && [[ "$in_scenario" == true ]]; then
            if [[ -n "$content" ]]; then
                # Generate embedding
                local embedding=$(qdrant::embeddings::generate "$content" "mxbai-embed-large" 2>/dev/null)
                
                if [[ -n "$embedding" ]]; then
                    # Create unique ID from content hash
                    local scenario_id=$(echo -n "$content" | sha256sum | cut -d' ' -f1)
                    
                    # Store in collection
                    if qdrant::collections::upsert_point "$collection" \
                        "$scenario_id" \
                        "$embedding" \
                        "{\"content\": $(echo "$content" | jq -Rs .), \"type\": \"scenario\", \"app_id\": \"$app_id\"}" 2>/dev/null; then
                        ((count++))
                    fi
                fi
            fi
            in_scenario=false
            content=""
        elif [[ "$in_scenario" == true ]]; then
            content="${content}${line}
"
        fi
    done < "$chunk_file"
    
    echo "$count"
    return 0
}

export -f qdrant::parallel::process_scenario_chunk

#######################################
# Enable parallel processing in manage.sh
# This function replaces sequential processing
#######################################
qdrant::parallel::enable() {
    export QDRANT_PARALLEL_ENABLED=true
    export MAX_WORKERS="${1:-4}"
    log::info "Parallel processing enabled with $MAX_WORKERS workers"
}