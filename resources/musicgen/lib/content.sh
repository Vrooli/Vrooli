#!/usr/bin/env bash
################################################################################
# MusicGen Content Management Functions
################################################################################

# Add a new music generation task
musicgen::content::add() {
    local task_name="${1:-}"
    local params="${2:-}"
    
    if [[ -z "$task_name" ]]; then
        log::error "Task name required"
        return 1
    fi
    
    # Store task configuration
    local task_dir="${MUSICGEN_DATA_DIR}/tasks"
    mkdir -p "$task_dir"
    echo "$params" > "${task_dir}/${task_name}.json"
    log::info "Added task: $task_name"
}

# List available tasks
musicgen::content::list() {
    local task_dir="${MUSICGEN_DATA_DIR}/tasks"
    if [[ -d "$task_dir" ]]; then
        log::info "Available tasks:"
        ls -1 "$task_dir" | sed 's/\.json$//'
    else
        log::info "No tasks found"
    fi
}

# Get task details
musicgen::content::get() {
    local task_name="${1:-}"
    
    if [[ -z "$task_name" ]]; then
        log::error "Task name required"
        return 1
    fi
    
    local task_file="${MUSICGEN_DATA_DIR}/tasks/${task_name}.json"
    if [[ -f "$task_file" ]]; then
        cat "$task_file"
    else
        log::error "Task not found: $task_name"
        return 1
    fi
}

# Remove a task
musicgen::content::remove() {
    local task_name="${1:-}"
    
    if [[ -z "$task_name" ]]; then
        log::error "Task name required"
        return 1
    fi
    
    local task_file="${MUSICGEN_DATA_DIR}/tasks/${task_name}.json"
    if [[ -f "$task_file" ]]; then
        rm "$task_file"
        log::info "Removed task: $task_name"
    else
        log::error "Task not found: $task_name"
        return 1
    fi
}