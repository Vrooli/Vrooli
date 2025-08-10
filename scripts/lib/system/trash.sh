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
    local test_cleanup=false
    
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
            --test-cleanup)
                test_cleanup=true
                no_confirm=true  # Test cleanup always skips confirmation
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
    
    # Handle test cleanup mode with special validation
    if [[ "$test_cleanup" == true ]]; then
        if ! trash::validate_test_cleanup_path "$canonical_target"; then
            return 1
        fi
        # For /tmp paths in test mode, skip to direct deletion
        if [[ "$canonical_target" =~ ^/tmp/ ]] || [[ "$canonical_target" =~ ^/var/tmp/ ]]; then
            log::debug "Test cleanup: Direct deletion of temp path: $canonical_target"
            trash::log_operation "TEST_CLEANUP_TEMP" "$canonical_target" "Direct deletion"
            if rm -rf "$canonical_target" 2>/dev/null; then
                return 0
            else
                log::error "Failed to delete test temp path: $canonical_target"
                return 1
            fi
        fi
    else
        # Regular safety check: prevent deletion of critical system paths
        if ! trash::validate_path "$canonical_target"; then
            return 1
        fi
    fi
    
    # Extra protection in CI/test environments
    if [[ -n "${CI:-}" || -n "${BATS_TEST_FILENAME:-}" || -n "${GITHUB_ACTIONS:-}" ]]; then
        log::warn "Running in CI/test environment - extra safety checks enabled"
        
        # In test mode, be extra conservative
        if [[ "$force_permanent" == true && "$no_confirm" == true ]]; then
            log::error "CRITICAL: Test environment detected - refusing permanent deletion without confirmation"
            trash::log_operation "BLOCKED_CI_UNSAFE" "$canonical_target" "REJECTED"
            return 1
        fi
    fi
    
    # Calculate size for large file warning
    local size_mb
    size_mb=$(trash::calculate_size_mb "$canonical_target")
    
    # HARD LIMIT: refuse to delete anything over 5GB without explicit override
    if [[ "$size_mb" -gt 5120 ]]; then
        log::error "CRITICAL: Refusing to delete extremely large item (${size_mb}MB > 5GB limit): $canonical_target"
        log::error "This seems dangerous - manual review required"
        trash::log_operation "BLOCKED_SIZE_LIMIT" "$canonical_target" "${size_mb}MB too large"
        return 1
    fi
    
    # Count files being deleted for directories
    if [[ -d "$canonical_target" ]]; then
        local file_count
        file_count=$(find "$canonical_target" -type f 2>/dev/null | wc -l)
        if [[ "$file_count" -gt 10000 && "$no_confirm" == true ]]; then
            log::error "CRITICAL: Refusing to delete $file_count files without confirmation"
            trash::log_operation "BLOCKED_FILE_COUNT" "$canonical_target" "$file_count files"
            return 1
        fi
        if [[ "$file_count" -gt 1000 ]]; then
            log::warn "Large directory deletion: $file_count files in $canonical_target"
        fi
    fi
    
    # For critical files, create backup first
    if trash::is_critical_file "$canonical_target"; then
        local backup_dir="/tmp"
        # In test environments, use the test temp dir if available
        if [[ -n "${BATS_TEST_TMPDIR:-}" ]]; then
            backup_dir="$BATS_TEST_TMPDIR"
        fi
        local backup_path="$backup_dir/trash_backup_$(date +%Y%m%d_%H%M%S)_$(basename "$canonical_target")"
        log::warn "Creating backup of critical file before deletion: $backup_path"
        if ! cp -r "$canonical_target" "$backup_path" 2>/dev/null; then
            log::error "Failed to create backup - aborting deletion for safety"
            trash::log_operation "BACKUP_FAILED" "$canonical_target" "ABORTED"
            return 1
        fi
        log::info "Safety backup created at: $backup_path"
        trash::log_operation "BACKUP_CREATED" "$canonical_target" "$backup_path"
    fi
    
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
        trash::log_operation "PERMANENT_DELETE_REQUESTED" "$canonical_target" "size_mb=${size_mb}"
        
        if [[ "$no_confirm" == false ]]; then
            echo -n "This will PERMANENTLY delete the file(s). Are you sure? (y/N): "
            read -r response
            if [[ "$response" != "y" && "$response" != "Y" ]]; then
                log::info "Permanent deletion cancelled by user"
                trash::log_operation "PERMANENT_DELETE_CANCELLED" "$canonical_target" "USER_CANCELLED"
                return 1
            fi
        fi
        
        log::info "Permanently deleting: $canonical_target"
        if rm -rf "$canonical_target"; then
            log::success "Permanently deleted: $canonical_target"
            trash::log_operation "PERMANENT_DELETE_SUCCESS" "$canonical_target" "COMPLETED"
            return 0
        else
            log::error "Failed to permanently delete: $canonical_target"
            trash::log_operation "PERMANENT_DELETE_FAILED" "$canonical_target" "rm command failed"
            return 1
        fi
    fi
    
    # Attempt to move to trash
    if trash::move_to_trash "$canonical_target"; then
        log::success "Moved to trash: $canonical_target"
        trash::log_operation "TRASH_SUCCESS" "$canonical_target" "COMPLETED"
        return 0
    else
        log::error "Failed to move to trash, falling back to permanent deletion"
        log::warn "FALLBACK: Permanently deleting: $canonical_target"
        trash::log_operation "TRASH_FAILED_FALLBACK" "$canonical_target" "attempting permanent delete"
        
        if [[ "$no_confirm" == false ]]; then
            echo -n "Trash failed. Permanently delete instead? (y/N): "
            read -r response
            if [[ "$response" != "y" && "$response" != "Y" ]]; then
                log::info "Deletion cancelled by user"
                trash::log_operation "FALLBACK_CANCELLED" "$canonical_target" "USER_CANCELLED"
                return 1
            fi
        fi
        
        if rm -rf "$canonical_target"; then
            log::warn "PERMANENTLY DELETED (trash failed): $canonical_target"
            trash::log_operation "FALLBACK_SUCCESS" "$canonical_target" "rm completed"
            return 0
        else
            log::error "Failed to delete: $canonical_target"
            trash::log_operation "FALLBACK_FAILED" "$canonical_target" "rm failed"
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
# Find the project root directory by looking for markers
# Arguments:
#   $1 - starting path to search from
# Returns: project root path via stdout
#######################################
trash::find_project_root() {
    local start_path="$1"
    local current_dir
    
    if [[ -f "$start_path" ]]; then
        current_dir="$(dirname "$start_path")"
    else
        current_dir="$start_path"
    fi
    
    # Walk up directory tree looking for project markers
    while [[ "$current_dir" != "/" ]]; do
        if [[ -f "$current_dir/package.json" ]] || [[ -d "$current_dir/.git" ]] || [[ -f "$current_dir/pnpm-workspace.yaml" ]]; then
            echo "$current_dir"
            return 0
        fi
        current_dir="$(dirname "$current_dir")"
    done
    
    # Fallback to current working directory
    echo "$PWD"
}

#######################################
# Check if a file is critical and needs special protection
# Arguments:
#   $1 - path to check
# Returns: 0 if critical, 1 if not
#######################################
trash::is_critical_file() {
    local path="$1"
    local basename_file="$(basename "$path")"
    
    case "$basename_file" in
        package.json|pnpm-lock.yaml|yarn.lock|package-lock.json|Dockerfile|docker-compose*.yml|.env*|*.key|*.pem|.git|.gitignore)
            return 0
            ;;
    esac
    
    # Check for critical directories
    case "$path" in
        */.git|*/.git/*|*/node_modules|*/packages|*/src)
            return 0
            ;;
    esac
    
    return 1
}

#######################################
# Log operations to audit trail
# Arguments:
#   $1 - operation type
#   $2 - target path
#   $3 - result
#######################################
trash::log_operation() {
    local operation="$1"
    local target="$2" 
    local result="$3"
    local audit_log="${XDG_DATA_HOME:-$HOME/.local/share}/trash_audit.log"
    
    # Create directory if it doesn't exist
    mkdir -p "$(dirname "$audit_log")"
    
    # Log the operation with timestamp and context
    echo "$(date --iso-8601=seconds 2>/dev/null || date) - $operation - $target - $result - PID:$$ - USER:${USER:-unknown} - PWD:$PWD" >> "$audit_log"
}

#######################################
# Validate that a path is safe to delete
# Arguments:
#   $1 - canonical path to validate
# Returns: 0 if safe, 1 if dangerous
#######################################
trash::validate_path() {
    local canonical_target="$1"
    
    # Find project root for context
    local project_root
    project_root=$(trash::find_project_root "$canonical_target")
    
    # CRITICAL: Prevent deletion of entire project root
    if [[ "$canonical_target" == "$project_root" ]]; then
        log::error "CRITICAL: Refusing to delete entire project root: $canonical_target"
        trash::log_operation "BLOCKED_PROJECT_ROOT" "$canonical_target" "REJECTED"
        return 1
    fi
    
    # CRITICAL: Prevent deletion of current working directory or its parents
    local pwd_canonical
    pwd_canonical="$(trash::canonicalize "$PWD")"
    
    # Check if target is PWD or contains PWD
    if [[ "$pwd_canonical" == "$canonical_target"* ]]; then
        log::error "CRITICAL: Refusing to delete current working directory or its parent: $canonical_target"
        trash::log_operation "BLOCKED_PWD_PARENT" "$canonical_target" "REJECTED"
        return 1
    fi
    
    # CRITICAL: Prevent deletion of critical project files/directories
    if [[ "$canonical_target" =~ ^"$project_root" ]]; then
        local relative_path="${canonical_target#$project_root/}"
        case "$relative_path" in
            ".git"|.git/*|"package.json"|"pnpm-lock.yaml"|"pnpm-workspace.yaml"|"node_modules")
                log::error "CRITICAL: Refusing to delete critical project file/directory: $canonical_target"
                trash::log_operation "BLOCKED_CRITICAL_FILE" "$canonical_target" "REJECTED"
                return 1
                ;;
        esac
    fi
    
    # CRITICAL: Repository integrity protection
    if [[ "$canonical_target" =~ \.git$ ]] || [[ "$canonical_target" =~ \.git/ ]]; then
        log::error "CRITICAL: Refusing to delete Git repository data: $canonical_target"
        trash::log_operation "BLOCKED_GIT_DATA" "$canonical_target" "REJECTED"
        return 1
    fi
    
    # Check for other VCS directories
    case "$(basename "$canonical_target")" in
        .svn|.hg|.bzr)
            log::error "CRITICAL: Refusing to delete version control directory: $canonical_target"
            trash::log_operation "BLOCKED_VCS_DIR" "$canonical_target" "REJECTED"
            return 1
            ;;
    esac
    
    # Safety check: prevent deletion of critical system paths
    case "$canonical_target" in
        /|/bin|/boot|/dev|/etc|/lib|/lib64|/proc|/root|/sbin|/sys|/usr|/var)
            log::error "CRITICAL: Refusing to delete critical system directory: $canonical_target"
            trash::log_operation "BLOCKED_SYSTEM_DIR" "$canonical_target" "REJECTED"
            return 1
            ;;
        /home|"$HOME")
            log::error "CRITICAL: Refusing to delete entire home directory: $canonical_target"
            trash::log_operation "BLOCKED_HOME_DIR" "$canonical_target" "REJECTED"
            return 1
            ;;
    esac
    
    # Check if any processes are using files in this path (if lsof is available)
    if command -v lsof >/dev/null 2>&1 && [[ -d "$canonical_target" ]]; then
        local open_files
        open_files=$(lsof +D "$canonical_target" 2>/dev/null | wc -l)
        if [[ "$open_files" -gt 0 ]]; then
            log::warn "WARNING: $open_files files in $canonical_target are currently open by processes"
            # This is a warning, not a blocker, but we log it
            trash::log_operation "WARNING_FILES_IN_USE" "$canonical_target" "$open_files files in use"
        fi
    fi
    
    # Log successful validation
    trash::log_operation "VALIDATED" "$canonical_target" "ALLOWED"
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

#######################################
# Validate path for test cleanup mode
# Two-tier validation: lenient for /tmp, strict for project paths
# Arguments:
#   $1 - canonical path to validate
# Returns: 0 if safe for test cleanup, 1 if dangerous
#######################################
trash::validate_test_cleanup_path() {
    local canonical_target="$1"
    
    # Tier 1: /tmp and /var/tmp paths - lenient validation
    if [[ "$canonical_target" =~ ^/tmp/ ]] || [[ "$canonical_target" =~ ^/var/tmp/ ]]; then
        # Still do basic safety checks even in /tmp
        case "$canonical_target" in
            /tmp|/var/tmp)
                log::error "CRITICAL: Cannot delete entire temp directory"
                trash::log_operation "BLOCKED_TEST_CLEANUP" "$canonical_target" "ENTIRE_TMP_DIR"
                return 1
                ;;
        esac
        # Check for BATS test directory - extra safe
        if [[ -n "${BATS_TEST_TMPDIR:-}" ]] && [[ "$canonical_target" =~ ^"$BATS_TEST_TMPDIR" ]]; then
            log::debug "Test cleanup: BATS temp directory allowed"
            return 0
        fi
        log::debug "Test cleanup: /tmp path allowed"
        return 0
    fi
    
    # Tier 2: Non-/tmp paths - VERY strict validation
    # Path MUST contain test indicators
    local has_test_indicator=false
    case "$canonical_target" in
        *test*|*spec*|*.bats*|*fixture*|*mock*|*stub*)
            has_test_indicator=true
            ;;
        */coverage/*|*/.nyc_output/*|*/.vitest/*|*/.jest/*)
            has_test_indicator=true
            ;;
        */test-output/*|*/test-results/*|*/test-artifacts/*)
            has_test_indicator=true
            ;;
    esac
    
    if [[ "$has_test_indicator" == false ]]; then
        log::error "CRITICAL: Test cleanup refusing non-test path outside /tmp: $canonical_target"
        log::error "Path must contain 'test', 'spec', 'fixture', etc., or be in /tmp"
        trash::log_operation "BLOCKED_TEST_CLEANUP" "$canonical_target" "NO_TEST_INDICATOR"
        return 1
    fi
    
    # Additional safety: reject if path is too close to project root
    local project_root
    project_root=$(trash::find_project_root "$canonical_target")
    
    # Never allow deletion of project root or critical subdirs
    case "$canonical_target" in
        "$project_root"|"$project_root/.git"|"$project_root/packages"|"$project_root/scripts"|"$project_root/src")
            log::error "CRITICAL: Test cleanup cannot delete critical project directory: $canonical_target"
            trash::log_operation "BLOCKED_TEST_CLEANUP" "$canonical_target" "CRITICAL_PROJECT_DIR"
            return 1
            ;;
    esac
    
    # Reject paths that look like home directories or system paths
    case "$canonical_target" in
        /home/*|/root|/usr/*|/etc/*|/var/*|/opt/*)
            # Exception: allow if it's clearly a test directory
            if [[ ! "$canonical_target" =~ /test[s]?/ ]] && [[ ! "$canonical_target" =~ /fixture[s]?/ ]]; then
                log::error "CRITICAL: Test cleanup cannot delete system path: $canonical_target"
                trash::log_operation "BLOCKED_TEST_CLEANUP" "$canonical_target" "SYSTEM_PATH"
                return 1
            fi
            ;;
    esac
    
    log::debug "Test cleanup: Validated non-/tmp test path: $canonical_target"
    trash::log_operation "TEST_CLEANUP_VALIDATED" "$canonical_target" "ALLOWED"
    return 0
}

#######################################
# Safe cleanup function specifically for test environments (DEPRECATED)
# Use trash::safe_remove with --test-cleanup flag instead
# Arguments:
#   $1 - path to clean up
#   $@ - additional options passed to safe_remove
# Returns: 0 on success, 1 on failure
#######################################
trash::safe_test_cleanup() {
    local target="$1"
    shift  # Remove target from arguments, keep the rest
    
    log::warn "trash::safe_test_cleanup is deprecated. Use trash::safe_remove --test-cleanup instead"
    
    # Call the regular safe_remove with test-cleanup flag
    trash::safe_remove "$target" --test-cleanup "$@"
}

# Export functions if sourced
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    export -f trash::safe_remove
    export -f trash::move_to_trash
    export -f trash::list
    export -f trash::empty
    export -f trash::get_trash_dirs
    export -f trash::validate_path
    export -f trash::validate_test_cleanup_path
    export -f trash::calculate_size_mb
    export -f trash::canonicalize
    export -f trash::find_project_root
    export -f trash::is_critical_file
    export -f trash::log_operation
    export -f trash::safe_test_cleanup
fi