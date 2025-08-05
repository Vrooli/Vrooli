#!/usr/bin/env bash

################################################################################
# File System Operations Module
#
# Provides file system operations for scenario-to-app conversion:
# - Path resolution and normalization
# - Directory structure creation and validation
# - File operations (copy, backup, load)
# - Permission and safety checks
#
# This module is designed to be:
# - Standalone and testable
# - Free of external dependencies except logging
# - Safe with proper error handling
#
# Usage:
#   source fs-operations.sh
#   resolve_project_paths
#   create_app_directory_structure "/path/to/target"
#
################################################################################

set -euo pipefail

################################################################################
# Configuration and Constants
################################################################################

# Default relative paths from script location (guard against multiple sourcing)
if [[ -z "${DEFAULT_PROJECT_ROOT_OFFSET:-}" ]]; then
    readonly DEFAULT_PROJECT_ROOT_OFFSET="../../../.."
fi

if [[ -z "${DEFAULT_SCENARIOS_SUBDIR:-}" ]]; then
    readonly DEFAULT_SCENARIOS_SUBDIR="scripts/scenarios/core"
fi

if [[ -z "${DEFAULT_APP_DIRS:-}" ]]; then
    readonly DEFAULT_APP_DIRS=("bin" "config" "data" "scripts" "manifests")
fi

# Required system tools (guard against multiple sourcing)
if [[ -z "${REQUIRED_TOOLS:-}" ]]; then
    readonly REQUIRED_TOOLS=("jq" "cp" "mv" "date" "mkdir" "chmod")
fi

################################################################################
# Logging Functions (with fallbacks if not sourced externally)
################################################################################

if ! declare -f log_info >/dev/null 2>&1; then
    log_info() { echo "[INFO] $*"; }
fi

if ! declare -f log_error >/dev/null 2>&1; then
    log_error() { echo "[ERROR] $*" >&2; }
fi

if ! declare -f log_warning >/dev/null 2>&1; then
    log_warning() { echo "[WARNING] $*"; }
fi

if ! declare -f log_success >/dev/null 2>&1; then
    log_success() { echo "[SUCCESS] $*"; }
fi

################################################################################
# Path Resolution Functions
################################################################################

# Resolve project paths based on script location
# Sets global variables: FS_SCRIPT_DIR, FS_PROJECT_ROOT, FS_SCENARIOS_DIR
# Usage: resolve_project_paths [project_root_offset] [scenarios_subdir]
resolve_project_paths() {
    local project_root_offset="${1:-$DEFAULT_PROJECT_ROOT_OFFSET}"
    local scenarios_subdir="${2:-$DEFAULT_SCENARIOS_SUBDIR}"
    
    # Determine script directory (works even when sourced)
    if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
        FS_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    else
        FS_SCRIPT_DIR="$(pwd)"
        log_warning "Could not determine script directory, using current directory"
    fi
    
    # Resolve project root
    if [[ -d "${FS_SCRIPT_DIR}/${project_root_offset}" ]]; then
        FS_PROJECT_ROOT="$(cd "${FS_SCRIPT_DIR}/${project_root_offset}" && pwd)"
    else
        log_error "Project root not found at: ${FS_SCRIPT_DIR}/${project_root_offset}"
        return 1
    fi
    
    # Resolve scenarios directory
    FS_SCENARIOS_DIR="${FS_PROJECT_ROOT}/${scenarios_subdir}"
    if [[ ! -d "$FS_SCENARIOS_DIR" ]]; then
        log_error "Scenarios directory not found: $FS_SCENARIOS_DIR"
        return 1
    fi
    
    return 0
}

# Get absolute path for a scenario by name
# Usage: resolve_scenario_path "scenario-name"
# Returns: absolute path to scenario directory
resolve_scenario_path() {
    local scenario_name="$1"
    
    if [[ -z "${FS_SCENARIOS_DIR:-}" ]]; then
        log_error "FS_SCENARIOS_DIR not set. Call resolve_project_paths first."
        return 1
    fi
    
    local scenario_path="${FS_SCENARIOS_DIR}/${scenario_name}"
    
    if [[ ! -d "$scenario_path" ]]; then
        log_error "Scenario not found: $scenario_name"
        log_info "Available scenarios:"
        list_available_scenarios | sed 's/^/  - /'
        return 1
    fi
    
    echo "$scenario_path"
    return 0
}

# Normalize path (convert relative to absolute)
# Usage: normalize_path "/some/path" or normalize_path "relative/path"
normalize_path() {
    local input_path="$1"
    
    if [[ "$input_path" = /* ]]; then
        # Already absolute
        echo "$input_path"
    else
        # Make absolute relative to current directory
        echo "$(cd "$(dirname "$input_path")" 2>/dev/null && pwd)/$(basename "$input_path")" || echo "$input_path"
    fi
}

################################################################################
# Directory Operations
################################################################################

# List available scenarios in the scenarios directory
# Usage: list_available_scenarios
list_available_scenarios() {
    if [[ -z "${FS_SCENARIOS_DIR:-}" ]]; then
        log_error "FS_SCENARIOS_DIR not set. Call resolve_project_paths first."
        return 1
    fi
    
    if [[ ! -d "$FS_SCENARIOS_DIR" ]]; then
        log_error "Scenarios directory not found: $FS_SCENARIOS_DIR"
        return 1
    fi
    
    ls -1 "$FS_SCENARIOS_DIR" 2>/dev/null || true
}

# Create standard app directory structure
# Usage: create_app_directory_structure "/path/to/target" [additional_dirs...]
create_app_directory_structure() {
    local target_dir="$1"
    shift
    local additional_dirs=("$@")
    
    if [[ -z "$target_dir" ]]; then
        log_error "Target directory not specified"
        return 1
    fi
    
    target_dir="$(normalize_path "$target_dir")"
    
    # Create standard directories
    local all_dirs=("${DEFAULT_APP_DIRS[@]}")
    if [[ ${#additional_dirs[@]} -gt 0 ]]; then
        all_dirs+=("${additional_dirs[@]}")
    fi
    
    for dir in "${all_dirs[@]}"; do
        local full_path="${target_dir}/${dir}"
        if ! mkdir -p "$full_path"; then
            log_error "Failed to create directory: $full_path"
            return 1
        fi
    done
    
    log_info "Created app directory structure in: $target_dir"
    return 0
}

# Check if directory is writable
# Usage: check_directory_writable "/path/to/dir"
check_directory_writable() {
    local dir_path="$1"
    
    if [[ ! -d "$dir_path" ]]; then
        # Try to create parent directory
        local parent_dir
        parent_dir="$(dirname "$dir_path")"
        if [[ ! -d "$parent_dir" ]]; then
            log_error "Parent directory does not exist: $parent_dir"
            return 1
        fi
        if [[ ! -w "$parent_dir" ]]; then
            log_error "No write permission to parent directory: $parent_dir"
            return 1
        fi
    else
        if [[ ! -w "$dir_path" ]]; then
            log_error "No write permission to directory: $dir_path"
            return 1
        fi
    fi
    
    return 0
}

################################################################################
# File Operations
################################################################################

# Check if file exists and is readable
# Usage: check_file_exists "/path/to/file" [description]
check_file_exists() {
    local file_path="$1"
    local description="${2:-file}"
    
    if [[ ! -f "$file_path" ]]; then
        log_error "$description not found: $file_path"
        return 1
    fi
    
    if [[ ! -r "$file_path" ]]; then
        log_error "$description is not readable: $file_path"
        return 1
    fi
    
    return 0
}

# Load and validate JSON file
# Usage: load_service_json "/path/to/service.json"
# Returns: JSON content via stdout
load_service_json() {
    local json_path="$1"
    
    if ! check_file_exists "$json_path" "service.json"; then
        return 1
    fi
    
    local json_content
    if ! json_content=$(cat "$json_path" 2>/dev/null); then
        log_error "Failed to read service.json: $json_path"
        return 1
    fi
    
    # Validate JSON syntax
    if ! echo "$json_content" | jq empty 2>/dev/null; then
        log_error "Invalid JSON syntax in: $json_path"
        return 1
    fi
    
    echo "$json_content"
    return 0
}

# Create safe backup with metadata
# Usage: create_safe_backup "/path/to/file" [reason]
# Returns: backup file path via stdout
create_safe_backup() {
    local source_file="$1"
    local backup_reason="${2:-manual}"
    
    if ! check_file_exists "$source_file" "source file for backup"; then
        return 1
    fi
    
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${source_file}.backup.${timestamp}.${backup_reason}"
    
    if ! cp "$source_file" "$backup_file"; then
        log_error "Failed to create backup of: $source_file"
        return 1
    fi
    
    # Create metadata file
    local metadata_file="${backup_file}.meta"
    cat > "$metadata_file" << EOF
{
  "source_file": "$source_file",
  "backup_time": "$(date -Iseconds)",
  "backup_reason": "$backup_reason",
  "script_version": "fs-operations.sh",
  "user": "$(whoami)",
  "pwd": "$(pwd)"
}
EOF
    
    if [[ "${VERBOSE:-false}" == "true" ]]; then
        log_info "Created backup: $backup_file"
    fi
    
    echo "$backup_file"
    return 0
}

# Copy scenario files to target directory
# Usage: copy_scenario_files "/source/scenario" "/target/data"
copy_scenario_files() {
    local source_dir="$1"
    local target_dir="$2"
    
    if [[ ! -d "$source_dir" ]]; then
        log_error "Source scenario directory not found: $source_dir"
        return 1
    fi
    
    if ! mkdir -p "$target_dir"; then
        log_error "Failed to create target directory: $target_dir"
        return 1
    fi
    
    # Copy all files from source to target
    if ! cp -r "${source_dir}"/* "${target_dir}/" 2>/dev/null; then
        log_warning "Some files may not have been copied from: $source_dir"
    fi
    
    log_info "Copied scenario files from $source_dir to $target_dir"
    return 0
}

# Copy specific file with validation
# Usage: copy_file_safely "/source/file" "/target/file"
copy_file_safely() {
    local source_file="$1"
    local target_file="$2"
    
    if ! check_file_exists "$source_file" "source file"; then
        return 1
    fi
    
    local target_dir
    target_dir="$(dirname "$target_file")"
    if ! mkdir -p "$target_dir"; then
        log_error "Failed to create target directory: $target_dir"
        return 1
    fi
    
    if ! cp "$source_file" "$target_file"; then
        log_error "Failed to copy file: $source_file -> $target_file"
        return 1
    fi
    
    return 0
}

################################################################################
# Safety and Validation Functions
################################################################################

# Check for required system tools
# Usage: check_required_tools [additional_tools...]
check_required_tools() {
    local additional_tools=("$@")
    local all_tools=("${REQUIRED_TOOLS[@]}")
    
    if [[ ${#additional_tools[@]} -gt 0 ]]; then
        all_tools+=("${additional_tools[@]}")
    fi
    
    local missing_tools=()
    
    for tool in "${all_tools[@]}"; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            missing_tools+=("$tool")
        fi
    done
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Required tools not found: ${missing_tools[*]}"
        return 1
    fi
    
    return 0
}

# Validate basic scenario structure
# Usage: validate_scenario_structure "/path/to/scenario"
validate_scenario_structure() {
    local scenario_path="$1"
    
    if [[ ! -d "$scenario_path" ]]; then
        log_error "Scenario directory not found: $scenario_path"
        return 1
    fi
    
    # Check for service.json
    local service_json_path="${scenario_path}/service.json"
    if ! check_file_exists "$service_json_path" "service.json"; then
        log_info "All scenarios must have a service.json file"
        log_info "Expected location: $service_json_path"
        return 1
    fi
    
    # Validate JSON syntax
    if ! load_service_json "$service_json_path" >/dev/null; then
        return 1
    fi
    
    log_info "Basic scenario structure validation passed: $scenario_path"
    return 0
}

# Run comprehensive pre-flight safety checks
# Usage: run_preflight_checks "/path/to/target/dir"
run_preflight_checks() {
    local target_dir="$1"
    
    if [[ "${VERBOSE:-false}" == "true" ]]; then
        log_info "Running pre-flight safety checks..."
    fi
    
    # Check required tools
    if ! check_required_tools; then
        return 1
    fi
    
    # Check target directory permissions
    if ! check_directory_writable "$target_dir"; then
        return 1
    fi
    
    if [[ "${VERBOSE:-false}" == "true" ]]; then
        log_success "Pre-flight safety checks passed"
    fi
    
    return 0
}

################################################################################
# Utility Functions
################################################################################

# Make files executable
# Usage: make_executable "/path/to/dir" "pattern"
make_executable() {
    local target_dir="$1"
    local pattern="${2:-*}"
    
    if [[ ! -d "$target_dir" ]]; then
        log_error "Target directory not found: $target_dir"
        return 1
    fi
    
    find "$target_dir" -name "$pattern" -type f -exec chmod +x {} \; 2>/dev/null || true
    
    return 0
}

# Get scenario name from path
# Usage: get_scenario_name "/path/to/scenarios/scenario-name"
get_scenario_name() {
    local scenario_path="$1"
    basename "$scenario_path"
}

################################################################################
# Module Information
################################################################################

# Display module information (for testing/debugging)
fs_operations_info() {
    echo "File System Operations Module"
    echo "=============================="
    echo "Script Dir: ${FS_SCRIPT_DIR:-not set}"
    echo "Project Root: ${FS_PROJECT_ROOT:-not set}"
    echo "Scenarios Dir: ${FS_SCENARIOS_DIR:-not set}"
    echo ""
    echo "Available Functions:"
    echo "  - resolve_project_paths"
    echo "  - resolve_scenario_path"
    echo "  - create_app_directory_structure"
    echo "  - copy_scenario_files"
    echo "  - create_safe_backup"
    echo "  - load_service_json"
    echo "  - validate_scenario_structure"
    echo "  - run_preflight_checks"
}