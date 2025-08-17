#!/usr/bin/env bash
# QuestDB Common Functions
# Shared utilities for QuestDB management

#######################################
# Check if QuestDB directories exist
# Returns:
#   0 if directories exist, 1 otherwise
#######################################
questdb::dirs_exist() {
    [[ -d "${QUESTDB_DATA_DIR}" ]] && \
    [[ -d "${QUESTDB_CONFIG_DIR}" ]] && \
    [[ -d "${QUESTDB_LOG_DIR}" ]]
}

#######################################
# Create QuestDB directories
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::create_dirs() {
    log::info "${QUESTDB_INSTALL_MESSAGES["creating_directories"]}"
    
    local dirs=(
        "${QUESTDB_DATA_DIR}"
        "${QUESTDB_CONFIG_DIR}"
        "${QUESTDB_LOG_DIR}"
    )
    
    for dir in "${dirs[@]}"; do
        if ! mkdir -p "$dir"; then
            log::error "Failed to create directory: $dir"
            return 1
        fi
        
        # Fix Docker volume permissions if setup was run with sudo
        docker::fix_volume_permissions "$dir" 2>/dev/null || {
            log::debug "Could not fix Docker volume permissions for $dir, continuing..."
        }
    done
    
    return 0
}

#######################################
# Check available disk space
# Arguments:
#   $1 - Required space in GB
# Returns:
#   0 if enough space, 1 otherwise
#######################################
questdb::check_disk_space() {
    local required_gb="${1:-5}"
    local data_dir_parent
    data_dir_parent=$(dirname "${QUESTDB_DATA_DIR}")
    
    # Get available space in GB
    local available_gb
    available_gb=$(df -BG "${data_dir_parent}" | awk 'NR==2 {print $4}' | sed 's/G//')
    
    if (( available_gb < required_gb )); then
        log::error "${QUESTDB_ERROR_MESSAGES["insufficient_space"]}"
        log::error "Required: ${required_gb}GB, Available: ${available_gb}GB"
        return 1
    fi
    
    return 0
}

#######################################
# Check if ports are available
# Returns:
#   0 if ports are free, 1 otherwise
#######################################
questdb::check_ports() {
    local ports=("${QUESTDB_HTTP_PORT}" "${QUESTDB_PG_PORT}" "${QUESTDB_ILP_PORT}")
    local port
    
    for port in "${ports[@]}"; do
        if lsof -i ":${port}" &> /dev/null; then
            log::error "${QUESTDB_ERROR_MESSAGES["port_conflict"]}"
            log::error "Port ${port} is already in use"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Get QuestDB version from running container
# Returns:
#   Version string or empty if not running
#######################################
questdb::get_version() {
    if ! questdb::docker::is_running; then
        echo ""
        return 1
    fi
    
    # Get version from API
    local version
    version=$(curl -s "${QUESTDB_BASE_URL}/status" | jq -r '.questdb // empty' 2>/dev/null || echo "")
    
    echo "${version}"
}

#######################################
# Format bytes to human readable
# Arguments:
#   $1 - Bytes
# Returns:
#   Human readable string
#######################################
questdb::format_bytes() {
    local bytes="$1"
    local units=("B" "KB" "MB" "GB" "TB")
    local unit=0
    
    while (( bytes > 1024 && unit < ${#units[@]} - 1 )); do
        bytes=$((bytes / 1024))
        ((unit++))
    done
    
    echo "${bytes}${units[$unit]}"
}

#######################################
# Format duration to human readable
# Arguments:
#   $1 - Milliseconds
# Returns:
#   Human readable string
#######################################
questdb::format_duration() {
    local ms="$1"
    
    if (( ms < 1000 )); then
        echo "${ms}ms"
    elif (( ms < 60000 )); then
        echo "$((ms / 1000))s"
    elif (( ms < 3600000 )); then
        echo "$((ms / 60000))m"
    else
        echo "$((ms / 3600000))h"
    fi
}

#######################################
# Validate SQL query syntax (basic)
# Arguments:
#   $1 - SQL query
# Returns:
#   0 if valid, 1 otherwise
#######################################
questdb::validate_query() {
    local query="$1"
    
    # Check for empty query
    if [[ -z "$query" ]]; then
        log::error "Empty query"
        return 1
    fi
    
    # Check for dangerous operations without WHERE clause
    local dangerous_ops=("DELETE" "UPDATE" "DROP" "TRUNCATE")
    for op in "${dangerous_ops[@]}"; do
        if [[ "$query" =~ ^[[:space:]]*${op} ]] && ! [[ "$query" =~ WHERE|where ]]; then
            log::warning "Query contains ${op} without WHERE clause. Are you sure?"
            if ! args::prompt_yes_no "Continue?" "n"; then
                return 1
            fi
        fi
    done
    
    return 0
}

#######################################
# Export common functions
#######################################
export -f questdb::dirs_exist
export -f questdb::create_dirs
export -f questdb::check_disk_space
export -f questdb::check_ports
export -f questdb::get_version
export -f questdb::format_bytes
export -f questdb::format_duration
export -f questdb::validate_query