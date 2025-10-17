#!/usr/bin/env bash
# Filesystem Mock - Tier 2 (Stateful)
# 
# Provides stateful filesystem command mocking for testing:
# - File operations (ls, cat, touch, mkdir, rm, cp, mv)
# - File system navigation and inspection
# - Permission and ownership simulation
# - Path resolution and symlink handling
# - Error injection for resilience testing
#
# Coverage: ~80% of common filesystem operations in 600 lines

# === Configuration ===
declare -gA FS_FILES=()                # Virtual filesystem: path -> "exists|type|perms|owner|size|mtime|content"
declare -gA FS_SYMLINKS=()             # Symbolic links: link_path -> target_path
declare -gA FS_CONFIG=(                # Filesystem configuration
    [mode]="normal"
    [error_mode]=""
    [current_user]="testuser"
    [current_group]="testuser"
)

# Debug mode
declare -g FS_DEBUG="${FS_DEBUG:-}"

# === Helper Functions ===
fs_debug() {
    [[ -n "$FS_DEBUG" ]] && echo "[MOCK:FS] $*" >&2
}

fs_check_error() {
    case "${FS_CONFIG[error_mode]}" in
        "permission_denied")
            echo "Permission denied" >&2
            return 1
            ;;
        "no_space_left")
            echo "No space left on device" >&2
            return 1
            ;;
        "read_only")
            echo "Read-only file system" >&2
            return 1
            ;;
        "file_not_found")
            echo "No such file or directory" >&2
            return 2
            ;;
    esac
    return 0
}

fs_normalize_path() {
    local path="$1"
    # Remove trailing slashes except for root
    if [[ "$path" != "/" ]]; then
        path="${path%/}"
    fi
    # Handle empty path
    [[ -z "$path" ]] && path="/"
    echo "$path"
}

fs_parent_dir() {
    local path="$1"
    path="$(fs_normalize_path "$path")"
    if [[ "$path" == "/" ]]; then
        echo "/"
    else
        echo "${path%/*}" | sed 's|^$|/|'
    fi
}

fs_basename() {
    local path="$1"
    path="$(fs_normalize_path "$path")"
    if [[ "$path" == "/" ]]; then
        echo "/"
    else
        echo "${path##*/}"
    fi
}

fs_mock_timestamp() {
    date '+%s' 2>/dev/null || echo '1704067200'
}

fs_format_permissions() {
    local perms="$1"
    local type="$2"
    
    # Convert numeric to rwx format
    case "$type" in
        "d") echo "d$(printf "%03o" "$perms" | sed 's/./&/g' | sed 's/[0-7]/rwx/g' | sed 's/0/-/g; s/1/--x/g; s/2/-w-/g; s/3/-wx/g; s/4/r--/g; s/5/r-x/g; s/6/rw-/g; s/7/rwx/g')" ;;
        "l") echo "l$(printf "%03o" "$perms" | sed 's/./&/g' | sed 's/[0-7]/rwx/g' | sed 's/0/-/g; s/1/--x/g; s/2/-w-/g; s/3/-wx/g; s/4/r--/g; s/5/r-x/g; s/6/rw-/g; s/7/rwx/g')" ;;
        *) echo "-$(printf "%03o" "$perms" | sed 's/./&/g' | sed 's/[0-7]/rwx/g' | sed 's/0/-/g; s/1/--x/g; s/2/-w-/g; s/3/-wx/g; s/4/r--/g; s/5/r-x/g; s/6/rw-/g; s/7/rwx/g')" ;;
    esac
}

# === Core Filesystem Commands ===

ls() {
    fs_debug "ls called with: $*"
    
    if ! fs_check_error; then
        return $?
    fi
    
    local long_format=false all_files=false target="."
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -l) long_format=true; shift ;;
            -a) all_files=true; shift ;;
            -la|-al) long_format=true; all_files=true; shift ;;
            -*) shift ;;  # Skip other options
            *) target="$1"; shift ;;
        esac
    done
    
    target="$(fs_normalize_path "$target")"
    
    # Check if target exists
    local file_data="${FS_FILES[$target]}"
    if [[ -z "$file_data" ]]; then
        echo "ls: cannot access '$target': No such file or directory" >&2
        return 2
    fi
    
    IFS='|' read -r exists type perms owner group size mtime content <<< "$file_data"
    
    if [[ "$type" != "d" ]]; then
        # Single file
        if [[ "$long_format" == "true" ]]; then
            local perm_str=$(fs_format_permissions "$perms" "$type")
            printf "%s %d %s %s %8s %s %s\n" \
                "$perm_str" "1" "$owner" "$group" "$size" \
                "$(date -d "@$mtime" "+%b %d %H:%M" 2>/dev/null || echo "Jan 01 00:00")" \
                "$(fs_basename "$target")"
        else
            fs_basename "$target"
        fi
        return 0
    fi
    
    # Directory listing
    local found_files=()
    
    # Add . and .. if showing all files
    if [[ "$all_files" == "true" ]]; then
        found_files+=("." "..")
    fi
    
    # Find files in this directory
    for path in "${!FS_FILES[@]}"; do
        local parent="$(fs_parent_dir "$path")"
        if [[ "$parent" == "$target" && "$path" != "$target" ]]; then
            local filename="$(fs_basename "$path")"
            # Skip hidden files unless -a
            if [[ "$all_files" == "true" || "${filename:0:1}" != "." ]]; then
                found_files+=("$filename")
            fi
        fi
    done
    
    # Output files
    for filename in "${found_files[@]}"; do
        if [[ "$long_format" == "true" ]]; then
            local file_path="$target/$filename"
            if [[ "$filename" == "." ]]; then
                file_path="$target"
            elif [[ "$filename" == ".." ]]; then
                file_path="$(fs_parent_dir "$target")"
            fi
            
            local file_info="${FS_FILES[$file_path]}"
            if [[ -n "$file_info" ]]; then
                IFS='|' read -r f_exists f_type f_perms f_owner f_group f_size f_mtime f_content <<< "$file_info"
                local perm_str=$(fs_format_permissions "$f_perms" "$f_type")
                printf "%s %d %s %s %8s %s %s\n" \
                    "$perm_str" "1" "$f_owner" "$f_group" "$f_size" \
                    "$(date -d "@$f_mtime" "+%b %d %H:%M" 2>/dev/null || echo "Jan 01 00:00")" \
                    "$filename"
            fi
        else
            echo "$filename"
        fi
    done
}

cat() {
    fs_debug "cat called with: $*"
    
    if ! fs_check_error; then
        return $?
    fi
    
    if [[ $# -eq 0 ]]; then
        # Read from stdin (simulate)
        echo "Mock stdin content"
        return 0
    fi
    
    local exit_code=0
    for file in "$@"; do
        file="$(fs_normalize_path "$file")"
        local file_data="${FS_FILES[$file]}"
        
        if [[ -z "$file_data" ]]; then
            echo "cat: $file: No such file or directory" >&2
            exit_code=1
            continue
        fi
        
        IFS='|' read -r exists type perms owner group size mtime content <<< "$file_data"
        
        if [[ "$type" == "d" ]]; then
            echo "cat: $file: Is a directory" >&2
            exit_code=1
            continue
        fi
        
        # Output file content
        echo -e "$content"
    done
    
    return $exit_code
}

touch() {
    fs_debug "touch called with: $*"
    
    if ! fs_check_error; then
        return $?
    fi
    
    if [[ $# -eq 0 ]]; then
        echo "touch: missing file operand" >&2
        return 1
    fi
    
    local timestamp=$(fs_mock_timestamp)
    
    for file in "$@"; do
        file="$(fs_normalize_path "$file")"
        
        # Check parent directory exists
        local parent="$(fs_parent_dir "$file")"
        if [[ -z "${FS_FILES[$parent]}" ]]; then
            echo "touch: cannot touch '$file': No such file or directory" >&2
            return 1
        fi
        
        local file_data="${FS_FILES[$file]}"
        if [[ -n "$file_data" ]]; then
            # File exists - update timestamp
            IFS='|' read -r exists type perms owner group size mtime content <<< "$file_data"
            FS_FILES[$file]="true|$type|$perms|$owner|$group|$size|$timestamp|$content"
        else
            # Create new file
            FS_FILES[$file]="true|f|644|${FS_CONFIG[current_user]}|${FS_CONFIG[current_group]}|0|$timestamp|"
        fi
        
        fs_debug "Touched file: $file"
    done
}

mkdir() {
    fs_debug "mkdir called with: $*"
    
    if ! fs_check_error; then
        return $?
    fi
    
    local create_parents=false
    local mode="755"
    local dirs=()
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -p|--parents) create_parents=true; shift ;;
            -m|--mode) mode="$2"; shift 2 ;;
            -*) shift ;;  # Skip other options
            *) dirs+=("$1"); shift ;;
        esac
    done
    
    if [[ ${#dirs[@]} -eq 0 ]]; then
        echo "mkdir: missing operand" >&2
        return 1
    fi
    
    local timestamp=$(fs_mock_timestamp)
    
    for dir in "${dirs[@]}"; do
        dir="$(fs_normalize_path "$dir")"
        
        # Check if already exists
        if [[ -n "${FS_FILES[$dir]}" ]]; then
            echo "mkdir: cannot create directory '$dir': File exists" >&2
            return 1
        fi
        
        # Check parent exists (unless -p)
        local parent="$(fs_parent_dir "$dir")"
        if [[ "$create_parents" != "true" && -z "${FS_FILES[$parent]}" ]]; then
            echo "mkdir: cannot create directory '$dir': No such file or directory" >&2
            return 1
        fi
        
        # Create parent directories if needed (-p option)
        if [[ "$create_parents" == "true" ]]; then
            local current_path=""
            IFS='/' read -ra path_parts <<< "$dir"
            for part in "${path_parts[@]}"; do
                if [[ -n "$part" ]]; then
                    current_path="$current_path/$part"
                    if [[ -z "${FS_FILES[$current_path]}" ]]; then
                        FS_FILES[$current_path]="true|d|$mode|${FS_CONFIG[current_user]}|${FS_CONFIG[current_group]}|4096|$timestamp|"
                        fs_debug "Created parent directory: $current_path"
                    fi
                fi
            done
        else
            # Create just the target directory
            FS_FILES[$dir]="true|d|$mode|${FS_CONFIG[current_user]}|${FS_CONFIG[current_group]}|4096|$timestamp|"
        fi
        
        fs_debug "Created directory: $dir"
    done
}

rm() {
    fs_debug "rm called with: $*"
    
    if ! fs_check_error; then
        return $?
    fi
    
    local recursive=false force=false
    local files=()
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -r|-R|--recursive) recursive=true; shift ;;
            -f|--force) force=true; shift ;;
            -rf|-fr) recursive=true; force=true; shift ;;
            -*) shift ;;  # Skip other options
            *) files+=("$1"); shift ;;
        esac
    done
    
    if [[ ${#files[@]} -eq 0 ]]; then
        echo "rm: missing operand" >&2
        return 1
    fi
    
    local exit_code=0
    for file in "${files[@]}"; do
        file="$(fs_normalize_path "$file")"
        local file_data="${FS_FILES[$file]}"
        
        if [[ -z "$file_data" ]]; then
            if [[ "$force" != "true" ]]; then
                echo "rm: cannot remove '$file': No such file or directory" >&2
                exit_code=1
            fi
            continue
        fi
        
        IFS='|' read -r exists type perms owner group size mtime content <<< "$file_data"
        
        if [[ "$type" == "d" ]]; then
            if [[ "$recursive" != "true" ]]; then
                echo "rm: cannot remove '$file': Is a directory" >&2
                exit_code=1
                continue
            fi
            
            # Remove directory and all contents
            for path in "${!FS_FILES[@]}"; do
                if [[ "$path" == "$file"* ]]; then
                    unset FS_FILES[$path]
                    fs_debug "Removed (recursive): $path"
                fi
            done
        else
            # Remove regular file
            unset FS_FILES[$file]
            fs_debug "Removed file: $file"
        fi
        
        # Remove any symlinks pointing to this file
        for link in "${!FS_SYMLINKS[@]}"; do
            if [[ "${FS_SYMLINKS[$link]}" == "$file" ]]; then
                unset FS_SYMLINKS[$link]
            fi
        done
    done
    
    return $exit_code
}

cp() {
    fs_debug "cp called with: $*"
    
    if ! fs_check_error; then
        return $?
    fi
    
    local recursive=false force=false
    local files=()
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -r|-R|--recursive) recursive=true; shift ;;
            -f|--force) force=true; shift ;;
            -*) shift ;;  # Skip other options
            *) files+=("$1"); shift ;;
        esac
    done
    
    if [[ ${#files[@]} -lt 2 ]]; then
        echo "cp: missing file operand" >&2
        return 1
    fi
    
    local target="${files[-1]}"
    local sources=("${files[@]:0:${#files[@]}-1}")
    
    target="$(fs_normalize_path "$target")"
    
    local timestamp=$(fs_mock_timestamp)
    
    for source in "${sources[@]}"; do
        source="$(fs_normalize_path "$source")"
        local source_data="${FS_FILES[$source]}"
        
        if [[ -z "$source_data" ]]; then
            echo "cp: cannot stat '$source': No such file or directory" >&2
            return 1
        fi
        
        IFS='|' read -r exists type perms owner group size mtime content <<< "$source_data"
        
        local dest_path="$target"
        
        # If target is directory, copy into it
        local target_data="${FS_FILES[$target]}"
        if [[ -n "$target_data" ]]; then
            local target_type="${target_data#*|}"
            target_type="${target_type%%|*}"
            if [[ "$target_type" == "d" ]]; then
                dest_path="$target/$(fs_basename "$source")"
            fi
        fi
        
        # Copy file with updated timestamp and ownership
        FS_FILES[$dest_path]="true|$type|$perms|${FS_CONFIG[current_user]}|${FS_CONFIG[current_group]}|$size|$timestamp|$content"
        fs_debug "Copied: $source → $dest_path"
    done
}

mv() {
    fs_debug "mv called with: $*"
    
    if ! fs_check_error; then
        return $?
    fi
    
    if [[ $# -ne 2 ]]; then
        echo "mv: missing file operand" >&2
        return 1
    fi
    
    local source="$1" target="$2"
    source="$(fs_normalize_path "$source")"
    target="$(fs_normalize_path "$target")"
    
    local source_data="${FS_FILES[$source]}"
    if [[ -z "$source_data" ]]; then
        echo "mv: cannot stat '$source': No such file or directory" >&2
        return 1
    fi
    
    # Check if target is directory
    local target_data="${FS_FILES[$target]}"
    if [[ -n "$target_data" ]]; then
        local target_type="${target_data#*|}"
        target_type="${target_type%%|*}"
        if [[ "$target_type" == "d" ]]; then
            target="$target/$(fs_basename "$source")"
        fi
    fi
    
    # Move (copy then delete)
    FS_FILES[$target]="${FS_FILES[$source]}"
    unset FS_FILES[$source]
    
    fs_debug "Moved: $source → $target"
}

# === Convention-based Test Functions ===
test_filesystem_connection() {
    fs_debug "Testing connection..."
    
    # Test basic ls command
    local result
    result=$(ls / 2>/dev/null)
    
    if [[ -n "$result" ]]; then
        fs_debug "Connection test passed"
        return 0
    else
        fs_debug "Connection test failed: empty result"
        return 1
    fi
}

test_filesystem_health() {
    fs_debug "Testing health..."
    
    # Test connection
    test_filesystem_connection || return 1
    
    # Test file operations
    touch /tmp/health-test.txt >/dev/null 2>&1 || return 1
    ls /tmp/health-test.txt >/dev/null 2>&1 || return 1
    filesystem_mock_create_file "/tmp/health-test.txt" "health test" >/dev/null
    cat /tmp/health-test.txt >/dev/null 2>&1 || return 1
    
    # Test directory operations
    mkdir /tmp/health-dir >/dev/null 2>&1 || return 1
    ls /tmp/health-dir >/dev/null 2>&1 || return 1
    rm -rf /tmp/health-dir >/dev/null 2>&1 || return 1
    rm /tmp/health-test.txt >/dev/null 2>&1 || return 1
    
    fs_debug "Health test passed"
    return 0
}

test_filesystem_basic() {
    fs_debug "Testing basic operations..."
    
    # Test file creation and reading
    touch /tmp/basic-test.txt >/dev/null 2>&1 || return 1
    filesystem_mock_create_file "/tmp/basic-test.txt" "test content" >/dev/null
    cat /tmp/basic-test.txt >/dev/null 2>&1 || return 1
    
    # Test file listing
    ls /tmp/ | grep -q "basic-test.txt" || return 1
    ls -l /tmp/basic-test.txt >/dev/null 2>&1 || return 1
    
    # Test copy and move
    cp /tmp/basic-test.txt /tmp/basic-copy.txt >/dev/null 2>&1 || return 1
    mv /tmp/basic-copy.txt /tmp/basic-moved.txt >/dev/null 2>&1 || return 1
    
    # Test directory operations
    mkdir /tmp/basic-dir >/dev/null 2>&1 || return 1
    cp /tmp/basic-test.txt /tmp/basic-dir/ >/dev/null 2>&1 || return 1
    ls /tmp/basic-dir/ | grep -q "basic-test.txt" || return 1
    
    # Cleanup
    rm -rf /tmp/basic-dir >/dev/null 2>&1
    rm /tmp/basic-test.txt /tmp/basic-moved.txt >/dev/null 2>&1
    
    fs_debug "Basic test passed"
    return 0
}

# === State Management ===
filesystem_mock_reset() {
    fs_debug "Resetting mock state (called from: ${BASH_SOURCE[1]:-unknown}:${BASH_LINENO[0]:-unknown})"
    
    FS_FILES=()
    FS_SYMLINKS=()
    FS_CONFIG[error_mode]=""
    FS_CONFIG[mode]="normal"
    
    # Initialize filesystem defaults
    filesystem_mock_init_defaults
}

filesystem_mock_init_defaults() {
    local timestamp=$(fs_mock_timestamp)
    
    # Essential directories
    FS_FILES["/"]="true|d|755|root|root|4096|$timestamp|"
    FS_FILES["/tmp"]="true|d|777|root|root|4096|$timestamp|"
    FS_FILES["/home"]="true|d|755|root|root|4096|$timestamp|"
    FS_FILES["/usr"]="true|d|755|root|root|4096|$timestamp|"
    FS_FILES["/usr/bin"]="true|d|755|root|root|4096|$timestamp|"
    FS_FILES["/var"]="true|d|755|root|root|4096|$timestamp|"
    FS_FILES["/etc"]="true|d|755|root|root|4096|$timestamp|"
    
    # Common files
    FS_FILES["/etc/hostname"]="true|f|644|root|root|8|$timestamp|testhost"
    FS_FILES["/etc/passwd"]="true|f|644|root|root|100|$timestamp|root:x:0:0:root:/root:/bin/bash\ntestuser:x:1000:1000::/home/testuser:/bin/bash"
}

filesystem_mock_set_error() {
    FS_CONFIG[error_mode]="$1"
    fs_debug "Set error mode: $1"
}

filesystem_mock_dump_state() {
    echo "=== Filesystem Mock State ==="
    echo "Mode: ${FS_CONFIG[mode]}"
    echo "Files: ${#FS_FILES[@]}"
    for path in "${!FS_FILES[@]}"; do
        echo "  $path: ${FS_FILES[$path]}"
    done | head -20  # Limit output
    [[ ${#FS_FILES[@]} -gt 20 ]] && echo "  ... and $((${#FS_FILES[@]} - 20)) more"
    echo "Symlinks: ${#FS_SYMLINKS[@]}"
    for link in "${!FS_SYMLINKS[@]}"; do
        echo "  $link → ${FS_SYMLINKS[$link]}"
    done
    echo "Error Mode: ${FS_CONFIG[error_mode]:-none}"
    echo "==================="
}

filesystem_mock_create_file() {
    local path="${1:-/tmp/test_file.txt}"
    local content="${2:-test content}"
    local owner="${3:-${FS_CONFIG[current_user]}}"
    
    path="$(fs_normalize_path "$path")"
    local timestamp=$(fs_mock_timestamp)
    local size=${#content}
    
    FS_FILES[$path]="true|f|644|$owner|${FS_CONFIG[current_group]}|$size|$timestamp|$content"
    fs_debug "Created file: $path ($size bytes)"
    echo "$path"
}

filesystem_mock_create_directory() {
    local path="${1:-/tmp/test_dir}"
    local mode="${2:-755}"
    local owner="${3:-${FS_CONFIG[current_user]}}"
    
    path="$(fs_normalize_path "$path")"
    local timestamp=$(fs_mock_timestamp)
    
    FS_FILES[$path]="true|d|$mode|$owner|${FS_CONFIG[current_group]}|4096|$timestamp|"
    fs_debug "Created directory: $path"
    echo "$path"
}

# === Export Functions ===
export -f ls cat touch mkdir rm cp mv
export -f test_filesystem_connection test_filesystem_health test_filesystem_basic
export -f filesystem_mock_reset filesystem_mock_set_error
export -f filesystem_mock_dump_state filesystem_mock_create_file filesystem_mock_create_directory
export -f fs_debug fs_check_error

# Initialize with defaults
filesystem_mock_reset
fs_debug "Filesystem Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
