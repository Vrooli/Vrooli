#!/usr/bin/env bash
# trash.sh - Cross-platform safe file deletion with trash support
# Provides safe alternatives to rm -rf to prevent accidental data loss
set -euo pipefail

# Source var.sh with relative path first
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

#######################################
# Cross-platform safe file deletion with trash support
# Moves files to trash instead of permanent deletion to prevent data loss
# Arguments:
#   $1 - path to delete (file or directory)
#   $2 - options: --force-permanent (skip trash), --no-confirm (skip confirmation for large files)
# Returns: 0 on success, 1 on failure
# Usage: trash::safe_remove /path/to/file
#        trash::safe_remove /path/to/dir --no-confirm
#        trash::safe_remove /path/to/file --force-permanent
#######################################
trash::safe_remove() {
    local target="$1"
    local force_permanent=false
    local no_confirm=false
    
    # Parse options
    shift
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force-permanent)
                force_permanent=true
                shift
                ;;
            --no-confirm)
                no_confirm=true
                shift
                ;;
            *)
                log::error "Unknown option for trash::safe_remove: $1"
                return 1
                ;;
        esac
    done
    
    # Validate input
    if [[ -z "$target" ]]; then
        log::error "trash::safe_remove: No target specified"
        return 1
    fi
    
    if [[ ! -e "$target" ]]; then
        log::debug "trash::safe_remove: Target does not exist: $target"
        return 0  # Not an error if already gone
    fi
    
    # Get canonical path to avoid issues with relative paths
    local canonical_target
    canonical_target=$(trash::canonicalize "$target")
    
    # Safety check: prevent deletion of critical system paths
    if ! trash::validate_path "$canonical_target"; then
        return 1
    fi
    
    # Calculate size for large file warning
    local size_mb
    size_mb=$(trash::calculate_size_mb "$canonical_target")
    
    # Warn for large deletions (>100MB) unless confirmation disabled
    if [[ "$no_confirm" == false && "$size_mb" -gt 100 ]]; then
        log::warn "About to move large item to trash: $canonical_target ($size_mb MB)"
        echo -n "Continue? (y/N): "
        read -r response
        if [[ "$response" != "y" && "$response" != "Y" ]]; then
            log::info "Operation cancelled by user"
            return 1
        fi
    fi
    
    # Force permanent deletion if requested
    if [[ "$force_permanent" == true ]]; then
        log::warn "PERMANENT DELETION requested for: $canonical_target"
        if [[ "$no_confirm" == false ]]; then
            echo -n "This will PERMANENTLY delete the file(s). Are you sure? (y/N): "
            read -r response
            if [[ "$response" != "y" && "$response" != "Y" ]]; then
                log::info "Permanent deletion cancelled by user"
                return 1
            fi
        fi
        
        log::info "Permanently deleting: $canonical_target"
        if rm -rf "$canonical_target"; then
            log::success "Permanently deleted: $canonical_target"
            return 0
        else
            log::error "Failed to permanently delete: $canonical_target"
            return 1
        fi
    fi
    
    # Attempt to move to trash
    if trash::move_to_trash "$canonical_target"; then
        log::success "Moved to trash: $canonical_target"
        return 0
    else
        log::error "Failed to move to trash, falling back to permanent deletion"
        log::warn "FALLBACK: Permanently deleting: $canonical_target"
        
        if [[ "$no_confirm" == false ]]; then
            echo -n "Trash failed. Permanently delete instead? (y/N): "
            read -r response
            if [[ "$response" != "y" && "$response" != "Y" ]]; then
                log::info "Deletion cancelled by user"
                return 1
            fi
        fi
        
        if rm -rf "$canonical_target"; then
            log::warn "PERMANENTLY DELETED (trash failed): $canonical_target"
            return 0
        else
            log::error "Failed to delete: $canonical_target"
            return 1
        fi
    fi
}

#######################################
# Cross-platform function to move files to trash
# Arguments:
#   $1 - path to move to trash
# Returns: 0 on success, 1 on failure
#######################################
trash::move_to_trash() {
    local target="$1"
    local trash_dir
    local trash_info_dir
    local basename
    local timestamp
    local safe_name
    local counter=0
    
    # Get trash directories for current platform
    trash::get_trash_dirs trash_dir trash_info_dir
    
    # Create trash directories if they don't exist
    if [[ ! -d "$trash_dir" ]]; then
        if ! mkdir -p "$trash_dir"; then
            log::error "Failed to create trash directory: $trash_dir"
            return 1
        fi
    fi
    
    if [[ -n "$trash_info_dir" && ! -d "$trash_info_dir" ]]; then
        if ! mkdir -p "$trash_info_dir"; then
            log::error "Failed to create trash info directory: $trash_info_dir"
            return 1
        fi
    fi
    
    # Generate unique name in trash
    basename=$(basename "$target")
    timestamp=$(date +"%Y%m%d_%H%M%S")
    safe_name="${basename}_${timestamp}"
    
    # Handle naming conflicts
    while [[ -e "$trash_dir/$safe_name" ]]; do
        counter=$((counter + 1))
        safe_name="${basename}_${timestamp}_${counter}"
        if [[ $counter -gt 1000 ]]; then
            log::error "Too many naming conflicts in trash directory"
            return 1
        fi
    done
    
    # Create .trashinfo file for Linux (XDG standard)
    if [[ -n "$trash_info_dir" ]]; then
        local info_file="$trash_info_dir/${safe_name}.trashinfo"
        local deletion_date
        deletion_date=$(date --iso-8601=seconds 2>/dev/null || date -Iseconds 2>/dev/null || date)
        
        cat > "$info_file" << EOF
[Trash Info]
Path=$target
DeletionDate=$deletion_date
EOF
        
        if [[ ! -f "$info_file" ]]; then
            log::warn "Failed to create trash info file, but continuing with move"
        fi
    fi
    
    # Move file to trash
    if mv "$target" "$trash_dir/$safe_name"; then
        log::debug "Moved to trash: $target -> $trash_dir/$safe_name"
        return 0
    else
        # Clean up info file if move failed
        if [[ -n "$trash_info_dir" && -f "$trash_info_dir/${safe_name}.trashinfo" ]]; then
            rm -f "$trash_info_dir/${safe_name}.trashinfo"
        fi
        return 1
    fi
}

#######################################
# List items in trash
# Returns: 0 on success
#######################################
trash::list() {
    local trash_dir
    local trash_info_dir
    
    trash::get_trash_dirs trash_dir trash_info_dir
    
    if [[ ! -d "$trash_dir" ]]; then
        log::info "Trash directory doesn't exist or is empty"
        return 0
    fi
    
    log::info "Items in trash ($trash_dir):"
    if ! ls -la "$trash_dir" 2>/dev/null; then
        log::info "Trash is empty"
    fi
    
    if [[ -n "$trash_info_dir" && -d "$trash_info_dir" ]]; then
        local info_count
        info_count=$(find "$trash_info_dir" -name "*.trashinfo" 2>/dev/null | wc -l)
        log::info "Trash info files: $info_count"
    fi
}

#######################################
# Empty trash (with confirmation)
# Arguments:
#   $1 - options: --force (skip confirmation)
# Returns: 0 on success
#######################################
trash::empty() {
    local force=false
    
    if [[ "$1" == "--force" ]]; then
        force=true
    fi
    
    local trash_dir
    local trash_info_dir
    
    trash::get_trash_dirs trash_dir trash_info_dir
    
    if [[ ! -d "$trash_dir" ]]; then
        log::info "Trash directory doesn't exist - nothing to empty"
        return 0
    fi
    
    local item_count
    item_count=$(find "$trash_dir" -mindepth 1 -maxdepth 1 2>/dev/null | wc -l)
    
    if [[ "$item_count" -eq 0 ]]; then
        log::info "Trash is already empty"
        return 0
    fi
    
    if [[ "$force" == false ]]; then
        log::warn "About to PERMANENTLY delete $item_count items from trash"
        echo -n "This cannot be undone. Continue? (y/N): "
        read -r response
        if [[ "$response" != "y" && "$response" != "Y" ]]; then
            log::info "Trash emptying cancelled by user"
            return 1
        fi
    fi
    
    log::info "Emptying trash..."
    
    # Remove all files from trash
    if rm -rf "${trash_dir:?}"/*; then
        log::success "Emptied $item_count items from trash"
    else
        log::error "Failed to empty trash completely"
        return 1
    fi
    
    # Remove info files on Linux
    if [[ -n "$trash_info_dir" && -d "$trash_info_dir" ]]; then
        rm -f "$trash_info_dir"/*.trashinfo 2>/dev/null || true
        log::debug "Cleaned up trash info files"
    fi
    
    return 0
}

#######################################
# Get trash directory paths for current platform
# Arguments:
#   $1 - variable name to store trash_dir
#   $2 - variable name to store trash_info_dir (empty for some platforms)
#######################################
trash::get_trash_dirs() {
    local trash_dir_var="$1"
    local trash_info_dir_var="$2"
    
    case "$(uname -s)" in
        Darwin)
            eval "$trash_dir_var=\"\$HOME/.Trash\""
            eval "$trash_info_dir_var=\"\""  # macOS doesn't use .trashinfo files
            ;;
        Linux)
            # Follow XDG Base Directory specification
            eval "$trash_dir_var=\"\${XDG_DATA_HOME:-\$HOME/.local/share}/Trash/files\""
            eval "$trash_info_dir_var=\"\${XDG_DATA_HOME:-\$HOME/.local/share}/Trash/info\""
            ;;
        CYGWIN*|MINGW*|MSYS*)
            # Windows-like environment
            eval "$trash_dir_var=\"\$HOME/.trash\""  # Custom trash for Windows bash environments
            eval "$trash_info_dir_var=\"\""
            ;;
        *)
            log::debug "Unknown platform, using generic trash directory"
            eval "$trash_dir_var=\"\$HOME/.trash\""
            eval "$trash_info_dir_var=\"\""
            ;;
    esac
}

#######################################
# Validate that a path is safe to delete
# Arguments:
#   $1 - canonical path to validate
# Returns: 0 if safe, 1 if dangerous
#######################################
trash::validate_path() {
    local canonical_target="$1"
    
    # Safety check: prevent deletion of critical system paths
    case "$canonical_target" in
        /|/bin|/boot|/dev|/etc|/lib|/lib64|/proc|/root|/sbin|/sys|/usr|/var)
            log::error "trash::safe_remove: Refusing to delete critical system directory: $canonical_target"
            return 1
            ;;
        /home|"$HOME")
            log::error "trash::safe_remove: Refusing to delete entire home directory: $canonical_target"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Calculate size in MB for a file or directory
# Arguments:
#   $1 - path to calculate size for
# Returns: size in MB via stdout
#######################################
trash::calculate_size_mb() {
    local target="$1"
    local size_mb=0
    
    if [[ -d "$target" ]]; then
        if command -v du >/dev/null 2>&1; then
            size_mb=$(du -sm "$target" 2>/dev/null | cut -f1 || echo "0")
        fi
    elif [[ -f "$target" ]]; then
        if command -v stat >/dev/null 2>&1; then
            local size_bytes
            size_bytes=$(stat -c %s "$target" 2>/dev/null || echo "0")
            size_mb=$((size_bytes / 1024 / 1024))
        fi
    fi
    
    echo "$size_mb"
}

#######################################
# Cross-platform path canonicalization
# Arguments:
#   $1 - path to canonicalize
# Returns: canonical path via stdout
#######################################
trash::canonicalize() {
    local input="$1"
    
    # Use realpath if available to fully resolve symlinks
    if command -v realpath >/dev/null 2>&1; then
        realpath "$input"
    elif command -v readlink >/dev/null 2>&1; then
        # readlink -f resolves symlinks and canonicalizes
        readlink -f "$input"
    else
        # Pure Bash fallback: expand relative paths and normalize . and .. components
        if [[ "$input" != /* ]]; then
            input="$PWD/$input"
        fi
        # Remove '/./' segments
        input="${input//\/\.\//\/}"
        # Split into components
        local IFS='/'
        read -r -a _segments <<< "$input"
        local _result=()
        for _seg in "${_segments[@]}"; do
            case "$_seg" in
                ''|'.') continue ;;
                '..')
                    if (( ${#_result[@]} > 0 )); then
                        unset '_result[${#_result[@]}-1]'
                    fi
                    ;;
                *) _result+=("$_seg") ;;
            esac
        done
        # Reconstruct the path
        local canonical="/"
        for _seg in "${_result[@]}"; do
            canonical="${canonical%/}/$_seg"
        done
        echo "$canonical"
    fi
}

# Export functions if sourced
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    export -f trash::safe_remove
    export -f trash::move_to_trash
    export -f trash::list
    export -f trash::empty
    export -f trash::get_trash_dirs
    export -f trash::validate_path
    export -f trash::calculate_size_mb
    export -f trash::canonicalize
fi