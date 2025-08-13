#!/usr/bin/env bash
# Claude Code Enhanced Batch Execution Framework
# Provides clean, reusable batch execution patterns

# Set CLAUDE_CODE_SCRIPT_DIR if not already set (for BATS test compatibility)
CLAUDE_CODE_SCRIPT_DIR="${CLAUDE_CODE_SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"

# Source trash module for safe cleanup
# shellcheck disable=SC1091
source "${CLAUDE_CODE_SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

#######################################
# Simple batch runner - cleaner alternative to maintenance-agent.sh
# Arguments:
#   $1 - Prompt
#   $2 - Total turns
#   $3 - Batch size (optional, default 50)
#   $4 - Allowed tools (optional, default "Read,Edit,Write")
#   $5 - Output directory (optional, generates temp dir)
# Returns: 0 on success, 1 on failure
#######################################
claude_code::batch_simple() {
    local prompt="$1"
    local total_turns="$2"
    local batch_size="${3:-50}"
    local allowed_tools="${4:-Read,Edit,Write}"
    local output_dir="${5:-}"
    
    if [[ -z "$prompt" || -z "$total_turns" ]]; then
        log::error "Prompt and total turns are required"
        return 1
    fi
    
    if ! [[ "$total_turns" =~ ^[0-9]+$ ]]; then
        log::error "Total turns must be a number"
        return 1
    fi
    
    # Generate output directory if not provided
    if [[ -z "$output_dir" ]]; then
        output_dir="/tmp/claude_batch_$(date +%Y%m%d_%H%M%S)_$(openssl rand -hex 4 2>/dev/null || echo $RANDOM)"
        mkdir -p "$output_dir"
    fi
    
    log::header "🚀 Starting Batch Execution"
    log::info "Prompt: $prompt"
    log::info "Total turns: $total_turns"
    log::info "Batch size: $batch_size"
    log::info "Allowed tools: $allowed_tools"
    log::info "Output directory: $output_dir"
    
    local session_id=""
    local remaining=$total_turns
    local batch_num=0
    local total_files_modified=()
    local failed_batches=0
    local max_failures=2
    
    while (( remaining > 0 )); do
        local current_batch=$(( remaining < batch_size ? remaining : batch_size ))
        batch_num=$((batch_num + 1))
        
        log::info "📦 Batch $batch_num: $current_batch turns (remaining: $remaining)"
        
        local batch_output="$output_dir/batch_${batch_num}.json"
        local batch_prompt="$prompt"
        
        # Add date prefix to prompt for context
        batch_prompt="[$(date +%Y-%m-%d)] $batch_prompt"
        
        # Build and execute command
        local cmd=""
        if claude_code::run_automation "$batch_prompt" "$allowed_tools" "$current_batch" "$batch_output" >/dev/null 2>&1; then
            log::success "✅ Batch $batch_num completed"
            
            # Extract session ID for continuation
            local new_session_id
            new_session_id=$(claude_code::extract "$batch_output" "session")
            if [[ -n "$new_session_id" ]]; then
                session_id="$new_session_id"
            fi
            
            # Collect modified files
            local batch_files
            mapfile -t batch_files < <(claude_code::extract "$batch_output" "files")
            for file in "${batch_files[@]}"; do
                [[ -n "$file" ]] && total_files_modified+=("$file")
            done
            
            remaining=$((remaining - current_batch))
            failed_batches=0  # Reset failure count on success
            
        else
            log::warn "⚠️  Batch $batch_num failed"
            failed_batches=$((failed_batches + 1))
            
            if [[ $failed_batches -gt $max_failures ]]; then
                log::error "Too many consecutive failures ($failed_batches), aborting"
                break
            fi
            
            log::info "Retrying with smaller batch size..."
            batch_size=$((batch_size / 2))
            if [[ $batch_size -lt 5 ]]; then
                batch_size=5
            fi
        fi
        
        # Brief cooldown between batches
        if (( remaining > 0 )); then
            sleep 2
        fi
    done
    
    # Generate summary
    local status="completed"
    if [[ $remaining -gt 0 ]]; then
        status="partial"
    fi
    
    # Remove duplicates from files list
    local unique_files
    mapfile -t unique_files < <(printf '%s\n' "${total_files_modified[@]}" | sort -u)
    
    log::header "📊 Batch Execution Summary"
    log::info "Status: $([ "$status" = "completed" ] && echo "✅ Complete" || echo "⚠️  Partial ($remaining turns remaining)")"
    log::info "Batches executed: $batch_num"
    log::info "Files modified: ${#unique_files[@]}"
    log::info "Session ID: ${session_id:-none}"
    log::info "Output directory: $output_dir"
    
    if [[ ${#unique_files[@]} -gt 0 ]]; then
        log::info "Modified files:"
        printf '  - %s\n' "${unique_files[@]}"
    fi
    
    # Write summary file
    local summary_file="$output_dir/summary.json"
    cat > "$summary_file" <<EOF
{
  "status": "$status",
  "session_id": "${session_id:-}",
  "total_turns_requested": $total_turns,
  "turns_remaining": $remaining,
  "batches_executed": $batch_num,
  "failed_batches": $failed_batches,
  "files_modified": $(printf '%s\n' "${unique_files[@]}" | jq -R . | jq -s .),
  "output_directory": "$output_dir",
  "timestamp": "$(date -Iseconds)"
}
EOF
    
    echo "$summary_file"  # Return summary file path
    
    return $([ "$status" = "completed" ] && echo 0 || echo 1)
}

#######################################
# Configuration-based batch execution
# Arguments:
#   $1 - Configuration file path (YAML or JSON)
# Returns: 0 on success, 1 on failure
#######################################
claude_code::batch_config() {
    local config_file="$1"
    
    if [[ ! -f "$config_file" ]]; then
        log::error "Configuration file not found: $config_file"
        return 1
    fi
    
    log::header "📋 Configuration-Based Batch Execution"
    log::info "Config file: $config_file"
    
    # Check if it's YAML or JSON
    local is_yaml=false
    if [[ "$config_file" == *.yaml || "$config_file" == *.yml ]]; then
        is_yaml=true
        if ! command -v yq >/dev/null 2>&1; then
            log::error "yq is required for YAML configuration files"
            log::info "Install: https://github.com/mikefarah/yq"
            return 1
        fi
    fi
    
    # Parse configuration
    local name description stages
    if [[ "$is_yaml" == true ]]; then
        name=$(yq eval '.name' "$config_file")
        description=$(yq eval '.description' "$config_file")
        # YAML parsing would need more complex handling for stages
        log::error "YAML configuration not yet implemented"
        return 1
    else
        # JSON parsing
        if ! command -v jq >/dev/null 2>&1; then
            log::error "jq is required for JSON configuration files"
            return 1
        fi
        
        name=$(jq -r '.name // "Batch Workflow"' "$config_file")
        description=$(jq -r '.description // ""' "$config_file")
        
        log::info "Workflow: $name"
        if [[ -n "$description" ]]; then
            log::info "Description: $description"
        fi
        
        # Execute each stage
        local stage_count
        stage_count=$(jq '.stages | length' "$config_file")
        
        if [[ $stage_count -eq 0 ]]; then
            log::error "No stages defined in configuration"
            return 1
        fi
        
        local workflow_session_id=""
        local workflow_output_dir="/tmp/claude_workflow_$(date +%Y%m%d_%H%M%S)"
        mkdir -p "$workflow_output_dir"
        
        for (( i=0; i<stage_count; i++ )); do
            local stage_name stage_prompt stage_tools stage_turns stage_required
            stage_name=$(jq -r ".stages[$i].name" "$config_file")
            stage_prompt=$(jq -r ".stages[$i].prompt" "$config_file")
            stage_tools=$(jq -r ".stages[$i].tools // [\"Read\", \"Edit\", \"Write\"] | join(\",\")" "$config_file")
            stage_turns=$(jq -r ".stages[$i].max_turns // 20" "$config_file")
            stage_required=$(jq -r ".stages[$i].required // false" "$config_file")
            
            log::info "🔧 Stage $((i+1))/$stage_count: $stage_name"
            log::info "  Prompt: $stage_prompt"
            log::info "  Tools: $stage_tools"
            log::info "  Max turns: $stage_turns"
            
            local stage_output="$workflow_output_dir/stage_$((i+1))_${stage_name//[^a-zA-Z0-9]/_}.json"
            
            if claude_code::run_automation "$stage_prompt" "$stage_tools" "$stage_turns" "$stage_output" >/dev/null 2>&1; then
                log::success "  ✅ Stage completed"
                
                # Update workflow session
                local stage_session
                stage_session=$(claude_code::extract "$stage_output" "session")
                if [[ -n "$stage_session" ]]; then
                    workflow_session_id="$stage_session"
                fi
                
            else
                log::error "  ❌ Stage failed"
                if [[ "$stage_required" == "true" ]]; then
                    log::error "Required stage failed, aborting workflow"
                    return 1
                else
                    log::warn "Optional stage failed, continuing"
                fi
            fi
        done
        
        log::success "✅ Workflow completed: $name"
        log::info "Output directory: $workflow_output_dir"
        echo "$workflow_output_dir"
    fi
}

#######################################
# Multi-task batch execution - execute multiple prompts in sequence
# Arguments:
#   $@ - Alternating prompt and turn count pairs
# Usage: claude_code::batch_multi "Task 1" 50 "Task 2" 30 "Task 3" 20
# Returns: 0 on success, 1 on failure
#######################################
claude_code::batch_multi() {
    if [[ $# -lt 2 || $(( $# % 2 )) -ne 0 ]]; then
        log::error "Usage: batch_multi \"PROMPT1\" TURNS1 [\"PROMPT2\" TURNS2 ...]"
        return 1
    fi
    
    log::header "🔀 Multi-Task Batch Execution"
    log::info "Tasks: $(( $# / 2 ))"
    
    local task_num=0
    local total_tasks=$(( $# / 2 ))
    local output_dir="/tmp/claude_multi_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$output_dir"
    
    while (( $# )); do
        local prompt="$1"
        local turns="$2"
        shift 2
        task_num=$((task_num + 1))
        
        log::info "📝 Task $task_num/$total_tasks: $(echo "$prompt" | cut -c1-50)..."
        
        # Use simple batch for each task
        local task_output
        if task_output=$(claude_code::batch_simple "$prompt" "$turns" 50 "Read,Edit,Write,Bash(npm test)" "$output_dir/task_$task_num"); then
            log::success "  ✅ Task $task_num completed"
            
            # Brief cooldown between tasks
            if (( $# > 0 )); then
                sleep 5
            fi
        else
            log::error "  ❌ Task $task_num failed"
            # Continue with next task even if this one failed
        fi
    done
    
    log::success "✅ All tasks processed"
    log::info "Output directory: $output_dir"
    
    # Create overall summary
    local summary_file="$output_dir/multi_summary.json"
    cat > "$summary_file" <<EOF
{
  "type": "multi_task_batch",
  "total_tasks": $total_tasks,
  "output_directory": "$output_dir",
  "timestamp": "$(date -Iseconds)"
}
EOF
    
    echo "$summary_file"
}

#######################################
# Parallel batch execution - run multiple prompts concurrently
# Arguments:
#   $1 - Number of parallel workers
#   $@ - Remaining args are alternating prompt and turn count pairs
# Returns: 0 on success, 1 on failure
#######################################
claude_code::batch_parallel() {
    local workers="$1"
    shift
    
    if [[ ! "$workers" =~ ^[0-9]+$ ]] || [[ $workers -lt 1 ]]; then
        log::error "Number of workers must be a positive integer"
        return 1
    fi
    
    if [[ $# -lt 2 || $(( $# % 2 )) -ne 0 ]]; then
        log::error "Usage: batch_parallel WORKERS \"PROMPT1\" TURNS1 [\"PROMPT2\" TURNS2 ...]"
        return 1
    fi
    
    log::header "⚡ Parallel Batch Execution"
    log::info "Workers: $workers"
    log::info "Tasks: $(( $# / 2 ))"
    
    local output_dir="/tmp/claude_parallel_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$output_dir"
    
    # Create task queue
    local task_queue=()
    local task_num=0
    while (( $# )); do
        local prompt="$1"
        local turns="$2"
        shift 2
        task_num=$((task_num + 1))
        task_queue+=("$task_num|$prompt|$turns")
    done
    
    # Function to process tasks from queue
    process_task_queue() {
        local worker_id="$1"
        local queue_file="$2"
        
        while true; do
            # Atomically get next task
            local task_line
            {
                flock 200
                task_line=$(head -n 1 "$queue_file" 2>/dev/null)
                if [[ -n "$task_line" ]]; then
                    sed -i '1d' "$queue_file"
                fi
            } 200>"$queue_file.lock"
            
            [[ -z "$task_line" ]] && break
            
            local task_id prompt turns
            IFS='|' read -r task_id prompt turns <<< "$task_line"
            
            log::info "Worker $worker_id processing task $task_id"
            
            local task_output="$output_dir/worker_${worker_id}_task_${task_id}"
            claude_code::batch_simple "$prompt" "$turns" 50 "Read,Edit,Write" "$task_output" >/dev/null 2>&1
        done
    }
    
    # Write task queue to file
    local queue_file="$output_dir/task_queue"
    printf '%s\n' "${task_queue[@]}" > "$queue_file"
    
    # Start workers
    local worker_pids=()
    for (( i=1; i<=workers; i++ )); do
        process_task_queue "$i" "$queue_file" &
        worker_pids+=($!)
    done
    
    # Wait for all workers to complete
    log::info "Waiting for $workers workers to complete..."
    for pid in "${worker_pids[@]}"; do
        wait "$pid"
    done
    
    # Clean up
    trash::safe_remove "$queue_file" --temp
    trash::safe_remove "$queue_file.lock" --temp
    
    log::success "✅ All parallel tasks completed"
    log::info "Output directory: $output_dir"
    echo "$output_dir"
}

#######################################
# Progress callback for automation systems
# Arguments:
#   $1 - Current batch number
#   $2 - Current batch size
#   $3 - Remaining turns
#######################################
claude_code::batch_progress_callback() {
    local batch_num="$1"
    local batch_size="$2"
    local remaining="$3"
    
    local total_processed=$((batch_num * 50))  # Estimate based on default batch size
    local progress_percent=$(( (total_processed * 100) / (total_processed + remaining) ))
    
    log::info "Progress: $progress_percent% (batch $batch_num, $remaining turns remaining)"
    
    # Could be extended to write to progress file, webhook, etc.
    echo "{\"batch\": $batch_num, \"remaining\": $remaining, \"progress\": $progress_percent}" > "/tmp/claude_batch_progress.json"
}

# Export functions for external use
export -f claude_code::batch_simple
export -f claude_code::batch_config
export -f claude_code::batch_multi
export -f claude_code::batch_parallel
export -f claude_code::batch_progress_callback