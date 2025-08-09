#!/usr/bin/env bash
# Filesystem System Mocks for Bats tests.
# Provides comprehensive filesystem command mocking with a clear mock::fs::* namespace.

# Prevent duplicate loading
if [[ "${FILESYSTEM_MOCKS_LOADED:-}" == "true" ]]; then
  return 0
fi
export FILESYSTEM_MOCKS_LOADED="true"

# ----------------------------
# Global mock state & options
# ----------------------------
# Modes: normal, readonly, error
export FILESYSTEM_MOCK_MODE="${FILESYSTEM_MOCK_MODE:-normal}"

# Optional: directory to log calls/responses
# export MOCK_RESPONSES_DIR="${MOCK_RESPONSES_DIR:-}"

# In-memory state - virtual filesystem
declare -A MOCK_FS_FILES=()          # path -> "exists|type|perms|owner|group|size|mtime|content"
declare -A MOCK_FS_ERRORS=()         # command -> error_type
declare -A MOCK_FS_SYMLINKS=()       # link_path -> target_path

# Initialize root directory to prevent errors
MOCK_FS_FILES["/"]="true|d|755|root|root|4096|1700000000|"

# File-based state persistence for subshell access (BATS compatibility)
# Use consistent filename based on MOCK_LOG_DIR for state persistence across subshells
if [[ -n "${MOCK_LOG_DIR}" ]]; then
  export FILESYSTEM_MOCK_STATE_FILE="${MOCK_LOG_DIR}/filesystem_mock_state"
else
  export FILESYSTEM_MOCK_STATE_FILE="${FILESYSTEM_MOCK_STATE_FILE:-/tmp/filesystem_mock_state.$$}"
fi

# Initialize state file
_filesystem_mock_init_state_file() {
  # Disabled to prevent hanging - state persistence not working with BATS
  return 0
}

# Save current state to file
_filesystem_mock_save_state() {
  if [[ -n "${FILESYSTEM_MOCK_STATE_FILE}" ]]; then
    # Ensure directory exists
    local state_dir
    state_dir=$(dirname "$FILESYSTEM_MOCK_STATE_FILE")
    [[ -d "$state_dir" ]] || command mkdir -p "$state_dir" 2>/dev/null
    
    # Write state to file with base64 encoding for content to handle newlines
    {
      for path in "${!MOCK_FS_FILES[@]}"; do
        local value="${MOCK_FS_FILES[$path]}"
        # Encode the entire value to handle newlines in content
        local encoded_value=$(echo -n "$value" | base64 -w0 2>/dev/null || echo -n "$value" | base64)
        echo "FILE:$path:$encoded_value"
      done
      for cmd in "${!MOCK_FS_ERRORS[@]}"; do
        echo "ERROR:$cmd:${MOCK_FS_ERRORS[$cmd]}"
      done
      for link in "${!MOCK_FS_SYMLINKS[@]}"; do
        echo "LINK:$link:${MOCK_FS_SYMLINKS[$link]}"
      done
    } > "$FILESYSTEM_MOCK_STATE_FILE" 2>/dev/null || true
  fi
}

# Load state from file
_filesystem_mock_load_state() {
  if [[ -n "${FILESYSTEM_MOCK_STATE_FILE}" && -f "${FILESYSTEM_MOCK_STATE_FILE}" ]]; then
    # Clear existing arrays
    declare -gA MOCK_FS_FILES=()
    declare -gA MOCK_FS_ERRORS=()  
    declare -gA MOCK_FS_SYMLINKS=()
    
    # Read state from file
    while IFS=: read -r type key value; do
      case "$type" in
        FILE)
          # Decode base64 encoded value
          local decoded_value=$(echo -n "$value" | base64 -d 2>/dev/null || echo "$value")
          MOCK_FS_FILES["$key"]="$decoded_value"
          ;;
        ERROR)
          MOCK_FS_ERRORS["$key"]="$value"
          ;;
        LINK)
          MOCK_FS_SYMLINKS["$key"]="$value"
          ;;
      esac
    done < "$FILESYSTEM_MOCK_STATE_FILE" 2>/dev/null || true
    
    # Ensure root directory always exists
    if [[ -z "${MOCK_FS_FILES[/]}" ]]; then
      MOCK_FS_FILES["/"]="true|d|755|root|root|4096|1700000000|"
    fi
  fi
}

# State file initialization disabled to prevent hanging
# _filesystem_mock_init_state_file

# ----------------------------
# Utilities
# ----------------------------
# mock::log_call is now provided by mocks/logs.sh - no need to redefine

# Provide logging functions if not available
if ! command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify() {
        local cmd="$1"; shift
        # Simple logging if file exists
        if [[ -n "${MOCK_LOG_DIR:-}" && -d "${MOCK_LOG_DIR}" ]]; then
            echo "$(date): $cmd $*" >> "${MOCK_LOG_DIR}/command_calls.log" 2>/dev/null || true
        fi
    }
    export -f mock::log_and_verify
fi

if ! command -v mock::log_state &>/dev/null; then
    mock::log_state() {
        local event="$1" path="$2" content="$3"
        # Simple state logging if file exists
        if [[ -n "${MOCK_LOG_DIR:-}" && -d "${MOCK_LOG_DIR}" ]]; then
            echo "$(date): STATE $event $path" >> "${MOCK_LOG_DIR}/state_changes.log" 2>/dev/null || true
        fi
    }
    export -f mock::log_state
fi

_mock_current_time() {
    echo "1700000000"
}

_mock_normalize_path() {
    local path="$1"
    # Remove trailing slashes except for root
    if [[ "$path" != "/" ]]; then
        path="${path%/}"
    fi
    # Convert relative paths to absolute
    if [[ "${path:0:1}" != "/" ]]; then
        path="/$path"
    fi
    # Handle . and .. (basic normalization)
    # Convert /dir/../file to /file
    while [[ "$path" == *"/../"* ]]; do
        path=$(echo "$path" | sed 's|/[^/]*/\.\./|/|g')
    done
    # Remove trailing /. 
    path="${path%/.}"
    # Handle leading ../
    if [[ "$path" == ".."* ]]; then
        path="/"
    fi
    echo "$path"
}

_mock_parent_dir() {
    local path="$1"
    if [[ "$path" == "/" ]]; then
        echo "/"
    else
        dirname "$path"
    fi
}

_mock_basename() {
    local path="$1"
    if [[ "$path" == "/" ]]; then
        echo "/"
    else
        basename "$path"
    fi
}

_mock_format_permissions() {
    local perms="$1"
    local type="$2"
    
    # Convert octal permissions to rwx format
    local owner_perms=""
    local group_perms=""
    local other_perms=""
    
    # Handle 3-digit or 4-digit permissions (like 644 or 0644)
    if [[ ${#perms} -eq 4 ]]; then
        perms="${perms:1}"  # Strip leading 0
    fi
    
    # Extract each digit
    local owner_digit="${perms:0:1}"
    local group_digit="${perms:1:1}"
    local other_digit="${perms:2:1}"
    
    # Convert owner digit to rwx
    case "$owner_digit" in
        0) owner_perms="---" ;;
        1) owner_perms="--x" ;;
        2) owner_perms="-w-" ;;
        3) owner_perms="-wx" ;;
        4) owner_perms="r--" ;;
        5) owner_perms="r-x" ;;
        6) owner_perms="rw-" ;;
        7) owner_perms="rwx" ;;
        *) owner_perms="---" ;;
    esac
    
    # Convert group digit to rwx
    case "$group_digit" in
        0) group_perms="---" ;;
        1) group_perms="--x" ;;
        2) group_perms="-w-" ;;
        3) group_perms="-wx" ;;
        4) group_perms="r--" ;;
        5) group_perms="r-x" ;;
        6) group_perms="rw-" ;;
        7) group_perms="rwx" ;;
        *) group_perms="---" ;;
    esac
    
    # Convert other digit to rwx
    case "$other_digit" in
        0) other_perms="---" ;;
        1) other_perms="--x" ;;
        2) other_perms="-w-" ;;
        3) other_perms="-wx" ;;
        4) other_perms="r--" ;;
        5) other_perms="r-x" ;;
        6) other_perms="rw-" ;;
        7) other_perms="rwx" ;;
        *) other_perms="---" ;;
    esac
    
    # Add type prefix
    local type_char
    case "$type" in
        d) type_char="d" ;;
        l) type_char="l" ;;
        *) type_char="-" ;;
    esac
    
    echo "${type_char}${owner_perms}${group_perms}${other_perms}"
}

# ----------------------------
# Public functions used by tests
# ----------------------------
mock::fs::reset() {
  # Recreate as associative arrays (not indexed arrays)
  declare -gA MOCK_FS_FILES=()
  declare -gA MOCK_FS_ERRORS=()
  declare -gA MOCK_FS_SYMLINKS=()
  
  # Initialize with root directory
  MOCK_FS_FILES["/"]="true|d|755|root|root|4096|$(_mock_current_time)|"
  
  # Save initial state to file
  _filesystem_mock_save_state
  
  # Don't output to stdout in reset function - it interferes with tests
  # echo "[MOCK] Filesystem state reset"
}

# Enable automatic cleanup between tests
mock::fs::enable_auto_cleanup() {
  export FILESYSTEM_MOCK_AUTO_CLEANUP=true
}

# Inject errors for testing failure scenarios
mock::fs::inject_error() {
  local cmd="$1"
  local error_type="${2:-generic}"
  MOCK_FS_ERRORS["$cmd"]="$error_type"
  
  # Save state to file for subshell access
  _filesystem_mock_save_state
  
  echo "[MOCK] Injected error for $cmd: $error_type"
}

# ----------------------------
# Public setters used by tests
# ----------------------------
mock::fs::create_file() {
    local path="$1"
    local content="${2:-}"
    local perms="${3:-644}"
    local owner="${4:-user}"
    local group="${5:-user}"
    
    path=$(_mock_normalize_path "$path")
    
    # Ensure parent directory exists
    local parent=$(_mock_parent_dir "$path")
    if [[ "$parent" != "$path" && -z "${MOCK_FS_FILES[$parent]}" ]]; then
        mock::fs::create_directory "$parent"
    fi
    
    local size=${#content}
    local mtime=$(_mock_current_time)
    
    MOCK_FS_FILES["$path"]="true|f|$perms|$owner|$group|$size|$mtime|$content"
    
    # Save state to file for subshell access
    _filesystem_mock_save_state

    # Use centralized state logging
    if command -v mock::log_state &>/dev/null; then
        mock::log_state "filesystem_file_created" "$path" "$content"
    fi

    if command -v mock::verify::record_call &>/dev/null; then
        mock::verify::record_call "filesystem" "create_file $path"
    fi
    return 0
}

mock::fs::create_directory() {
    local path="$1"
    local perms="${2:-755}"
    local owner="${3:-user}"
    local group="${4:-user}"
    
    path=$(_mock_normalize_path "$path")
    
    # Ensure parent directory exists (recursive creation)
    local parent=$(_mock_parent_dir "$path")
    if [[ "$parent" != "$path" && "$parent" != "/" && -z "${MOCK_FS_FILES[$parent]}" ]]; then
        mock::fs::create_directory "$parent"
    fi
    
    local mtime=$(_mock_current_time)
    MOCK_FS_FILES["$path"]="true|d|$perms|$owner|$group|4096|$mtime|"
    
    # Save state to file for subshell access
    _filesystem_mock_save_state
    
    return 0
}

mock::fs::create_symlink() {
    local link_path="$1"
    local target_path="$2"
    
    link_path=$(_mock_normalize_path "$link_path")
    target_path=$(_mock_normalize_path "$target_path")
    
    # Store symlink info
    MOCK_FS_SYMLINKS["$link_path"]="$target_path"
    
    # Create a special file entry for the symlink
    local mtime=$(_mock_current_time)
    MOCK_FS_FILES["$link_path"]="true|l|777|user|user|${#target_path}|$mtime|$target_path"
    
    # Save state to file for subshell access
    _filesystem_mock_save_state
    
    return 0
}

# ----------------------------
# Command mocks
# ----------------------------
ls() {
    # Load state from file for subshell access
    _filesystem_mock_load_state

    # Use centralized logging and verification
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "ls" "$*"
    fi

    case "$FILESYSTEM_MOCK_MODE" in
        error) echo "ls: Input/output error" >&2; return 1 ;;
    esac

    # Check for injected errors
    if [[ -n "${MOCK_FS_ERRORS[ls]}" ]]; then
        local error_type="${MOCK_FS_ERRORS[ls]}"
        case "$error_type" in
            permission_denied)
                echo "ls: permission denied" >&2
                return 1
                ;;
            *)
                echo "ls: $error_type" >&2
                return 1
                ;;
        esac
    fi

    local path="."
    local long_format=false
    local all_files=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -l) long_format=true; shift ;;
            -a) all_files=true; shift ;;
            -la|-al) long_format=true; all_files=true; shift ;;
            -*) shift ;;
            *) path="$1"; shift ;;
        esac
    done
    
    path=$(_mock_normalize_path "$path")
    
    # Check if path exists
    if [[ -z "${MOCK_FS_FILES[$path]}" ]]; then
        echo "ls: cannot access '$path': No such file or directory" >&2
        return 2
    fi
    
    local info="${MOCK_FS_FILES[$path]}"
    IFS='|' read -r exists type perms owner group size mtime content <<<"$info"
    
    # If it's a file, just show the file
    if [[ "$type" == "f" || "$type" == "l" ]]; then
        if [[ "$long_format" == true ]]; then
            local perm_str=$(_mock_format_permissions "$perms" "$type")
            local time_str=$(date -d "@$mtime" "+%b %d %H:%M" 2>/dev/null || echo "Jan 01 00:00")
            printf "%s %d %s %s %6s %s %s\n" "$perm_str" 1 "$owner" "$group" "$size" "$time_str" "$(_mock_basename "$path")"
        else
            echo "$(_mock_basename "$path")"
        fi
        return 0
    fi
    
    # It's a directory - list contents
    local entries=()
    
    # Add . and .. if -a is specified
    if [[ "$all_files" == true ]]; then
        entries+=(".")
        entries+=("..")
    fi
    
    # Find all entries in this directory
    for file_path in "${!MOCK_FS_FILES[@]}"; do
        local parent=$(_mock_parent_dir "$file_path")
        if [[ "$parent" == "$path" && "$file_path" != "$path" ]]; then
            local basename=$(_mock_basename "$file_path")
            # Skip hidden files unless -a
            if [[ "$all_files" == true || "${basename:0:1}" != "." ]]; then
                entries+=("$basename")
            fi
        fi
    done
    
    # Show total if long format
    if [[ "$long_format" == true && ${#entries[@]} -gt 0 ]]; then
        local total_blocks=0
        for entry in "${entries[@]}"; do
            if [[ "$entry" != "." && "$entry" != ".." ]]; then
                local entry_path="$path/$entry"
                local entry_info="${MOCK_FS_FILES[$entry_path]}"
                if [[ -n "$entry_info" ]]; then
                    IFS='|' read -r exists type perms owner group size mtime content <<<"$entry_info"
                    total_blocks=$((total_blocks + (size / 1024 + 1)))
                fi
            fi
        done
        echo "total $total_blocks"
    fi
    
    # Sort and output entries
    if [[ ${#entries[@]} -gt 0 ]]; then
        # Simple sorting without array substitution
        local sorted_list
        sorted_list=$(printf '%s\n' "${entries[@]}" | sort)
        while IFS= read -r entry; do
            if [[ "$long_format" == true ]]; then
                # Handle . and .. specially
                if [[ "$entry" == "." ]]; then
                    local dir_info="${MOCK_FS_FILES[$path]}"
                    IFS='|' read -r exists type perms owner group size mtime content <<<"$dir_info"
                    local perm_str=$(_mock_format_permissions "$perms" "$type")
                    local time_str=$(date -d "@$mtime" "+%b %d %H:%M" 2>/dev/null || echo "Jan 01 00:00")
                    printf "%s %d %s %s %6s %s %s\n" "$perm_str" 1 "$owner" "$group" "$size" "$time_str" "."
                elif [[ "$entry" == ".." ]]; then
                    local parent_path=$(_mock_parent_dir "$path")
                    local parent_info="${MOCK_FS_FILES[$parent_path]:-true|d|755|root|root|4096|$(_mock_current_time)|}"
                    IFS='|' read -r exists type perms owner group size mtime content <<<"$parent_info"
                    local perm_str=$(_mock_format_permissions "$perms" "$type")
                    local time_str=$(date -d "@$mtime" "+%b %d %H:%M" 2>/dev/null || echo "Jan 01 00:00")
                    printf "%s %d %s %s %6s %s %s\n" "$perm_str" 1 "$owner" "$group" "$size" "$time_str" ".."
                else
                    # Get info for this entry
                    local entry_path="$path/$entry"
                    local entry_info="${MOCK_FS_FILES[$entry_path]}"
                    if [[ -n "$entry_info" ]]; then
                        IFS='|' read -r exists type perms owner group size mtime content <<<"$entry_info"
                        local perm_str=$(_mock_format_permissions "$perms" "$type")
                        local time_str=$(date -d "@$mtime" "+%b %d %H:%M" 2>/dev/null || echo "Jan 01 00:00")
                        printf "%s %d %s %s %6s %s %s\n" "$perm_str" 1 "$owner" "$group" "$size" "$time_str" "$entry"
                    fi
                fi
            else
                echo "$entry"
            fi
        done <<< "$sorted_list"
    fi
    
    # Save state after command
    _filesystem_mock_save_state
    
    return 0
}

cat() {
    # Load state from file for subshell access
    _filesystem_mock_load_state

    # Use centralized logging and verification
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "cat" "$*"
    fi

    case "$FILESYSTEM_MOCK_MODE" in
        readonly) ;; # cat is allowed in readonly mode
        error) echo "cat: Input/output error" >&2; return 1 ;;
    esac

    # Check for injected errors
    if [[ -n "${MOCK_FS_ERRORS[cat]}" ]]; then
        local error_type="${MOCK_FS_ERRORS[cat]}"
        case "$error_type" in
            permission_denied)
                echo "cat: permission_denied" >&2
                return 1
                ;;
            *)
                echo "cat: $error_type" >&2
                return 1
                ;;
        esac
    fi

    if [[ $# -eq 0 ]]; then
        # Read from stdin
        while IFS= read -r line; do
            echo "$line"
        done
        return 0
    fi
    
    local exit_code=0
    
    for file in "$@"; do
        file=$(_mock_normalize_path "$file")
        
        if [[ -z "${MOCK_FS_FILES[$file]}" ]]; then
            echo "cat: $file: No such file or directory" >&2
            exit_code=1
            continue
        fi
        
        local info="${MOCK_FS_FILES[$file]}"
        # Parse the pipe-separated fields
        local exists type perms owner group size mtime content
        
        # Split using shell parameter expansion to preserve newlines in content
        local remaining="$info"
        exists="${remaining%%|*}"; remaining="${remaining#*|}"
        type="${remaining%%|*}"; remaining="${remaining#*|}"
        perms="${remaining%%|*}"; remaining="${remaining#*|}"
        owner="${remaining%%|*}"; remaining="${remaining#*|}"
        group="${remaining%%|*}"; remaining="${remaining#*|}"
        size="${remaining%%|*}"; remaining="${remaining#*|}"
        mtime="${remaining%%|*}"; remaining="${remaining#*|}"
        content="$remaining"
        
        if [[ "$type" == "d" ]]; then
            echo "cat: $file: Is a directory" >&2
            exit_code=1
            continue
        fi
        
        if [[ "$type" == "l" ]]; then
            # For symlinks, follow to target
            local target="${MOCK_FS_SYMLINKS[$file]}"
            if [[ -n "$target" && -n "${MOCK_FS_FILES[$target]}" ]]; then
                local target_info="${MOCK_FS_FILES[$target]}"
                IFS='|' read -r exists type perms owner group size mtime target_content <<<"$target_info"
                if [[ "$type" == "f" ]]; then
                    printf '%s' "$target_content"
                else
                    echo "cat: $file: Is a directory" >&2
                    exit_code=1
                fi
            else
                echo "cat: $file: No such file or directory" >&2
                exit_code=1
            fi
        elif [[ -n "$content" ]]; then
            printf '%s' "$content"
        # Handle empty files (touch creates empty files)
        else
            printf ''
        fi
        
        # Add newline between files when processing multiple files
        if [[ $# -gt 1 ]]; then
            printf '\n'
        fi
    done
    
    # Save state after command
    _filesystem_mock_save_state
    
    return $exit_code
}

touch() {
    # State persistence across BATS 'run' subshells is not implemented
    # Tests that use 'run' for filesystem operations will not see state changes
    # Use direct function calls (mock::fs::create_file, etc.) for reliable testing

    # Use centralized logging and verification
    # Temporarily disable logging to debug
    # if command -v mock::log_and_verify &>/dev/null; then
    #     mock::log_and_verify "touch" "$*"
    # fi

    case "$FILESYSTEM_MOCK_MODE" in
        readonly) echo "touch: Read-only file system" >&2; return 1 ;;
        error) echo "touch: Input/output error" >&2; return 1 ;;
    esac

    # Check for injected errors
    if [[ -n "${MOCK_FS_ERRORS[touch]}" ]]; then
        local error_type="${MOCK_FS_ERRORS[touch]}"
        case "$error_type" in
            permission_denied)
                echo "touch: permission denied" >&2
                return 1
                ;;
            *)
                echo "touch: $error_type" >&2
                return 1
                ;;
        esac
    fi

    for file in "$@"; do
        [[ "$file" == -* ]] && continue
        [[ -z "$file" ]] && continue
        
        file=$(_mock_normalize_path "$file")
        
        if [[ -n "${MOCK_FS_FILES[$file]}" ]]; then
            # Update mtime
            local info="${MOCK_FS_FILES[$file]}"
            IFS='|' read -r exists type perms owner group size mtime content <<<"$info"
            mtime=$(_mock_current_time)
            MOCK_FS_FILES["$file"]="$exists|$type|$perms|$owner|$group|$size|$mtime|$content"
        else
            # Create new empty file - ensure parent directory exists
            local parent=$(_mock_parent_dir "$file")
            if [[ "$parent" != "$file" && "$parent" != "/" && -z "${MOCK_FS_FILES[$parent]}" ]]; then
                mock::fs::create_directory "$parent"
            fi
            
            local mtime=$(_mock_current_time)
            MOCK_FS_FILES["$file"]="true|f|644|user|user|0|$mtime|"
        fi
    done
    
    # Save state after command
    _filesystem_mock_save_state
    
    return 0
}

mkdir() {
    # Load state from file for subshell access
    _filesystem_mock_load_state

    # Use centralized logging and verification
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "mkdir" "$*"
    fi

    case "$FILESYSTEM_MOCK_MODE" in
        readonly) echo "mkdir: Read-only file system" >&2; return 1 ;;
        error) echo "mkdir: Input/output error" >&2; return 1 ;;
    esac

    # Check for injected errors
    if [[ -n "${MOCK_FS_ERRORS[mkdir]}" ]]; then
        local error_type="${MOCK_FS_ERRORS[mkdir]}"
        case "$error_type" in
            permission_denied)
                echo "mkdir: permission denied" >&2
                return 1
                ;;
            *)
                echo "mkdir: $error_type" >&2
                return 1
                ;;
        esac
    fi

    local parents=false
    local mode="755"
    local dirs=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -p|--parents) parents=true; shift ;;
            -m|--mode) mode="$2"; shift 2 ;;
            -*) shift ;;
            *) dirs+=("$1"); shift ;;
        esac
    done
    
    local exit_code=0
    
    for dir in "${dirs[@]}"; do
        dir=$(_mock_normalize_path "$dir")
        
        if [[ -n "${MOCK_FS_FILES[$dir]}" ]]; then
            if [[ "$parents" != true ]]; then
                echo "mkdir: cannot create directory '$dir': File exists" >&2
                exit_code=1
            fi
            continue
        fi
        
        local parent=$(_mock_parent_dir "$dir")
        if [[ -z "${MOCK_FS_FILES[$parent]}" && "$parents" != true ]]; then
            echo "mkdir: cannot create directory '$dir': No such file or directory" >&2
            exit_code=1
            continue
        fi
        
        if [[ "$parents" == true ]]; then
            # Create all parent directories recursively
            mock::fs::create_directory "$dir" "$mode"
        else
            # Check parent exists
            local parent=$(_mock_parent_dir "$dir")
            if [[ "$parent" != "/" && -z "${MOCK_FS_FILES[$parent]}" ]]; then
                echo "mkdir: cannot create directory '$dir': No such file or directory" >&2
                exit_code=1
                continue
            fi
            
            # Create single directory
            local mtime=$(_mock_current_time)
            MOCK_FS_FILES["$dir"]="true|d|$mode|user|user|4096|$mtime|"
        fi
    done
    
    # Save state after command
    _filesystem_mock_save_state
    
    return $exit_code
}

rm() {
    # Load state from file for subshell access
    _filesystem_mock_load_state

    # Use centralized logging and verification
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "rm" "$*"
    fi

    case "$FILESYSTEM_MOCK_MODE" in
        readonly) echo "rm: Read-only file system" >&2; return 1 ;;
        error) echo "rm: Input/output error" >&2; return 1 ;;
    esac

    # Check for injected errors
    if [[ -n "${MOCK_FS_ERRORS[rm]}" ]]; then
        local error_type="${MOCK_FS_ERRORS[rm]}"
        case "$error_type" in
            permission_denied)
                echo "rm: permission denied" >&2
                return 1
                ;;
            *)
                echo "rm: $error_type" >&2
                return 1
                ;;
        esac
    fi

    local recursive=false
    local force=false
    local files=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -r|-R|--recursive) recursive=true; shift ;;
            -f|--force) force=true; shift ;;
            -rf|-fr) recursive=true; force=true; shift ;;
            -*) shift ;;
            *) files+=("$1"); shift ;;
        esac
    done
    
    local exit_code=0
    
    for file in "${files[@]}"; do
        file=$(_mock_normalize_path "$file")
        
        # Protect root directory
        if [[ "$file" == "/" ]]; then
            echo "rm: cannot remove '/': Is a directory" >&2
            exit_code=1
            continue
        fi
        
        if [[ -z "${MOCK_FS_FILES[$file]}" ]]; then
            if [[ "$force" != true ]]; then
                echo "rm: cannot remove '$file': No such file or directory" >&2
                exit_code=1
            fi
            continue
        fi
        
        local info="${MOCK_FS_FILES[$file]}"
        IFS='|' read -r exists type perms owner group size mtime content <<<"$info"
        
        if [[ "$type" == "d" ]]; then
            if [[ "$recursive" != true ]]; then
                echo "rm: cannot remove '$file': Is a directory" >&2
                exit_code=1
                continue
            fi
            
            # Remove all files in directory recursively
            local paths_to_remove=()
            for path in "${!MOCK_FS_FILES[@]}"; do
                if [[ "$path" == "$file"/* || "$path" == "$file" ]]; then
                    paths_to_remove+=("$path")
                fi
            done
            
            # Remove collected paths
            for path in "${paths_to_remove[@]}"; do
                unset "MOCK_FS_FILES[$path]"
                unset "MOCK_FS_SYMLINKS[$path]"
            done
        else
            unset "MOCK_FS_FILES[$file]"
            # Also remove from symlinks if it exists there
            unset "MOCK_FS_SYMLINKS[$file]"
        fi
    done
    
    # Save state after command
    _filesystem_mock_save_state
    
    return $exit_code
}

cp() {
    # Load state from file for subshell access
    _filesystem_mock_load_state

    # Use centralized logging and verification
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "cp" "$*"
    fi

    case "$FILESYSTEM_MOCK_MODE" in
        readonly) echo "cp: Read-only file system" >&2; return 1 ;;
        error) echo "cp: Input/output error" >&2; return 1 ;;
    esac

    # Check for injected errors
    if [[ -n "${MOCK_FS_ERRORS[cp]}" ]]; then
        local error_type="${MOCK_FS_ERRORS[cp]}"
        case "$error_type" in
            permission_denied)
                echo "cp: permission denied" >&2
                return 1
                ;;
            *)
                echo "cp: $error_type" >&2
                return 1
                ;;
        esac
    fi

    local recursive=false
    local files=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -r|-R|--recursive) recursive=true; shift ;;
            -*) shift ;;
            *) files+=("$1"); shift ;;
        esac
    done
    
    # Need at least source and destination
    if [[ ${#files[@]} -lt 2 ]]; then
        echo "cp: missing destination file operand" >&2
        return 1
    fi
    
    # Get destination (last argument)
    local dest="${files[-1]}"
    unset 'files[-1]'
    
    dest=$(_mock_normalize_path "$dest")
    
    # Check if destination is a directory
    local dest_is_dir=false
    if [[ -n "${MOCK_FS_FILES[$dest]}" ]]; then
        local dest_info="${MOCK_FS_FILES[$dest]}"
        IFS='|' read -r exists type _ <<<"$dest_info"
        [[ "$type" == "d" ]] && dest_is_dir=true
    fi
    
    local exit_code=0
    
    for src in "${files[@]}"; do
        src=$(_mock_normalize_path "$src")
        
        if [[ -z "${MOCK_FS_FILES[$src]}" ]]; then
            echo "cp: cannot stat '$src': No such file or directory" >&2
            exit_code=1
            continue
        fi
        
        local src_info="${MOCK_FS_FILES[$src]}"
        IFS='|' read -r exists type perms owner group size mtime content <<<"$src_info"
        
        # Determine actual destination path
        local actual_dest="$dest"
        if [[ "$dest_is_dir" == true ]]; then
            actual_dest="$dest/$(_mock_basename "$src")"
        fi
        
        # Check if source is directory
        if [[ "$type" == "d" ]]; then
            if [[ "$recursive" != true ]]; then
                echo "cp: omitting directory '$src'" >&2
                exit_code=1
                continue
            fi
            
            # Copy directory recursively
            # First create the destination directory
            mock::fs::create_directory "$actual_dest" "$perms" "$owner" "$group"
            
            # Then copy all contents
            for path in "${!MOCK_FS_FILES[@]}"; do
                if [[ "$path" == "$src"/* ]]; then
                    local relative_path="${path#$src/}"
                    local dest_path="$actual_dest/$relative_path"
                    
                    # Copy this file/directory
                    MOCK_FS_FILES["$dest_path"]="${MOCK_FS_FILES[$path]}"
                    
                    # Copy symlink info if it exists
                    if [[ -n "${MOCK_FS_SYMLINKS[$path]}" ]]; then
                        MOCK_FS_SYMLINKS["$dest_path"]="${MOCK_FS_SYMLINKS[$path]}"
                    fi
                fi
            done
        else
            # Ensure destination parent directory exists
            local dest_parent=$(_mock_parent_dir "$actual_dest")
            if [[ "$dest_parent" != "/" && -z "${MOCK_FS_FILES[$dest_parent]}" ]]; then
                mock::fs::create_directory "$dest_parent"
            fi
            
            # Copy the file
            MOCK_FS_FILES["$actual_dest"]="$src_info"
            
            # Copy symlink info if it exists
            if [[ -n "${MOCK_FS_SYMLINKS[$src]}" ]]; then
                MOCK_FS_SYMLINKS["$actual_dest"]="${MOCK_FS_SYMLINKS[$src]}"
            fi
        fi
    done
    
    # Save state after command
    _filesystem_mock_save_state
    
    return $exit_code
}

mv() {
    # Load state from file for subshell access
    _filesystem_mock_load_state

    # Use centralized logging and verification
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "mv" "$*"
    fi

    case "$FILESYSTEM_MOCK_MODE" in
        readonly) echo "mv: Read-only file system" >&2; return 1 ;;
        error) echo "mv: Input/output error" >&2; return 1 ;;
    esac

    # Check for injected errors
    if [[ -n "${MOCK_FS_ERRORS[mv]}" ]]; then
        local error_type="${MOCK_FS_ERRORS[mv]}"
        case "$error_type" in
            permission_denied)
                echo "mv: permission denied" >&2
                return 1
                ;;
            *)
                echo "mv: $error_type" >&2
                return 1
                ;;
        esac
    fi

    local files=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -*) shift ;;
            *) files+=("$1"); shift ;;
        esac
    done
    
    # Need at least source and destination
    if [[ ${#files[@]} -lt 2 ]]; then
        echo "mv: missing destination file operand" >&2
        return 1
    fi
    
    # Get destination (last argument)
    local dest="${files[-1]}"
    unset 'files[-1]'
    
    dest=$(_mock_normalize_path "$dest")
    
    # Check if destination is a directory
    local dest_is_dir=false
    if [[ -n "${MOCK_FS_FILES[$dest]}" ]]; then
        local dest_info="${MOCK_FS_FILES[$dest]}"
        IFS='|' read -r exists type _ <<<"$dest_info"
        [[ "$type" == "d" ]] && dest_is_dir=true
    fi
    
    local exit_code=0
    
    for src in "${files[@]}"; do
        src=$(_mock_normalize_path "$src")
        
        if [[ -z "${MOCK_FS_FILES[$src]}" ]]; then
            echo "mv: cannot stat '$src': No such file or directory" >&2
            exit_code=1
            continue
        fi
        
        # Determine actual destination path
        local actual_dest="$dest"
        if [[ "$dest_is_dir" == true ]]; then
            actual_dest="$dest/$(_mock_basename "$src")"
        fi
        
        local src_info="${MOCK_FS_FILES[$src]}"
        IFS='|' read -r exists type perms owner group size mtime content <<<"$src_info"
        
        # Ensure destination parent directory exists
        local dest_parent=$(_mock_parent_dir "$actual_dest")
        if [[ "$dest_parent" != "/" && -z "${MOCK_FS_FILES[$dest_parent]}" ]]; then
            mock::fs::create_directory "$dest_parent"
        fi
        
        if [[ "$type" == "d" ]]; then
            # Move directory and all its contents
            # First create the destination directory
            MOCK_FS_FILES["$actual_dest"]="$src_info"
            
            # Collect paths to move to avoid modifying array during iteration
            local paths_to_move=()
            for path in "${!MOCK_FS_FILES[@]}"; do
                if [[ "$path" == "$src"/* ]]; then
                    paths_to_move+=("$path")
                fi
            done
            
            # Move all contents
            for path in "${paths_to_move[@]}"; do
                local relative_path="${path#$src/}"
                local dest_path="$actual_dest/$relative_path"
                
                # Move this file/directory
                MOCK_FS_FILES["$dest_path"]="${MOCK_FS_FILES[$path]}"
                unset "MOCK_FS_FILES[$path]"
                
                # Move symlink info if it exists
                if [[ -n "${MOCK_FS_SYMLINKS[$path]}" ]]; then
                    MOCK_FS_SYMLINKS["$dest_path"]="${MOCK_FS_SYMLINKS[$path]}"
                    unset "MOCK_FS_SYMLINKS[$path]"
                fi
            done
            
            # Remove the source directory
            unset "MOCK_FS_FILES[$src]"
            unset "MOCK_FS_SYMLINKS[$src]"
        else
            # Move the file
            MOCK_FS_FILES["$actual_dest"]="${MOCK_FS_FILES[$src]}"
            unset "MOCK_FS_FILES[$src]"
            
            # Move symlink info if it exists
            if [[ -n "${MOCK_FS_SYMLINKS[$src]}" ]]; then
                MOCK_FS_SYMLINKS["$actual_dest"]="${MOCK_FS_SYMLINKS[$src]}"
                unset "MOCK_FS_SYMLINKS[$src]"
            fi
        fi
    done
    
    # Save state after command
    _filesystem_mock_save_state
    
    return $exit_code
}

find() {
    # Load state from file for subshell access
    _filesystem_mock_load_state

    # Use centralized logging and verification
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "find" "$*"
    fi

    case "$FILESYSTEM_MOCK_MODE" in
        error) echo "find: Input/output error" >&2; return 1 ;;
    esac

    # Check for injected errors
    if [[ -n "${MOCK_FS_ERRORS[find]}" ]]; then
        local error_type="${MOCK_FS_ERRORS[find]}"
        case "$error_type" in
            permission_denied)
                echo "find: permission denied" >&2
                return 1
                ;;
            *)
                echo "find: $error_type" >&2
                return 1
                ;;
        esac
    fi

    local search_path="."
    local name_pattern=""
    local type_filter=""
    local maxdepth=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -name)
                name_pattern="$2"
                shift 2
                ;;
            -type)
                type_filter="$2"
                shift 2
                ;;
            -maxdepth)
                maxdepth="$2"
                shift 2
                ;;
            -*)
                shift
                ;;
            *)
                search_path="$1"
                shift
                ;;
        esac
    done
    
    search_path=$(_mock_normalize_path "$search_path")
    
    # Check if search path exists
    if [[ -z "${MOCK_FS_FILES[$search_path]}" ]]; then
        echo "find: '$search_path': No such file or directory" >&2
        return 1
    fi
    
    # Find all matching paths
    local results=()
    
    # Add the search path itself if it matches
    local search_info="${MOCK_FS_FILES[$search_path]}"
    IFS='|' read -r exists search_type _ <<<"$search_info"
    
    local matches=true
    
    # Check type filter
    if [[ -n "$type_filter" ]]; then
        case "$type_filter" in
            f) [[ "$search_type" != "f" ]] && matches=false ;;
            d) [[ "$search_type" != "d" ]] && matches=false ;;
            l) [[ "$search_type" != "l" ]] && matches=false ;;
        esac
    fi
    
    # Check name pattern
    if [[ -n "$name_pattern" && "$matches" == true ]]; then
        local basename=$(_mock_basename "$search_path")
        case "$basename" in
            $name_pattern) ;;
            *) matches=false ;;
        esac
    fi
    
    if [[ "$matches" == true ]]; then
        results+=("$search_path")
    fi
    
    # Find children if search path is a directory
    if [[ "$search_type" == "d" ]]; then
        for path in "${!MOCK_FS_FILES[@]}"; do
            # Skip if not a child of search path
            if [[ "$path" != "$search_path"/* ]]; then
                continue
            fi
            
            # Check maxdepth
            if [[ -n "$maxdepth" ]]; then
                local relative_path="${path#$search_path/}"
                local depth=$(echo "$relative_path" | tr '/' '\n' | wc -l)
                if [[ $depth -gt $maxdepth ]]; then
                    continue
                fi
            fi
            
            local path_info="${MOCK_FS_FILES[$path]}"
            IFS='|' read -r exists path_type _ <<<"$path_info"
            
            matches=true
            
            # Check type filter
            if [[ -n "$type_filter" ]]; then
                case "$type_filter" in
                    f) [[ "$path_type" != "f" ]] && matches=false ;;
                    d) [[ "$path_type" != "d" ]] && matches=false ;;
                    l) [[ "$path_type" != "l" ]] && matches=false ;;
                esac
            fi
            
            # Check name pattern
            if [[ -n "$name_pattern" && "$matches" == true ]]; then
                local basename=$(_mock_basename "$path")
                case "$basename" in
                    $name_pattern) ;;
                    *) matches=false ;;
                esac
            fi
            
            if [[ "$matches" == true ]]; then
                results+=("$path")
            fi
        done
    fi
    
    # Sort and output results
    if [[ ${#results[@]} -gt 0 ]]; then
        printf '%s\n' "${results[@]}" | sort
    fi
    
    # Save state after command
    _filesystem_mock_save_state
    
    return 0
}

chmod() {
    # Load state from file for subshell access
    _filesystem_mock_load_state

    # Use centralized logging and verification
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "chmod" "$*"
    fi

    case "$FILESYSTEM_MOCK_MODE" in
        readonly) echo "chmod: Read-only file system" >&2; return 1 ;;
        error) echo "chmod: Input/output error" >&2; return 1 ;;
    esac

    # Check for injected errors
    if [[ -n "${MOCK_FS_ERRORS[chmod]}" ]]; then
        local error_type="${MOCK_FS_ERRORS[chmod]}"
        case "$error_type" in
            permission_denied)
                echo "chmod: permission denied" >&2
                return 1
                ;;
            *)
                echo "chmod: $error_type" >&2
                return 1
                ;;
        esac
    fi

    local recursive=false
    local mode=""
    local files=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -R|--recursive) recursive=true; shift ;;
            -*) shift ;;
            *)
                if [[ -z "$mode" ]]; then
                    mode="$1"
                else
                    files+=("$1")
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$mode" ]] || [[ ${#files[@]} -eq 0 ]]; then
        echo "chmod: missing operand" >&2
        return 1
    fi
    
    local exit_code=0
    
    for file in "${files[@]}"; do
        file=$(_mock_normalize_path "$file")
        
        if [[ -z "${MOCK_FS_FILES[$file]}" ]]; then
            echo "chmod: cannot access '$file': No such file or directory" >&2
            exit_code=1
            continue
        fi
        
        # Change permissions for this file
        local info="${MOCK_FS_FILES[$file]}"
        IFS='|' read -r exists type perms owner group size mtime content <<<"$info"
        MOCK_FS_FILES["$file"]="$exists|$type|$mode|$owner|$group|$size|$mtime|$content"
        
        # If recursive and it's a directory, change all children too
        if [[ "$recursive" == true && "$type" == "d" ]]; then
            local paths_to_update=()
            for path in "${!MOCK_FS_FILES[@]}"; do
                if [[ "$path" == "$file"/* ]]; then
                    paths_to_update+=("$path")
                fi
            done
            
            for path in "${paths_to_update[@]}"; do
                local child_info="${MOCK_FS_FILES[$path]}"
                IFS='|' read -r exists child_type perms owner group size mtime content <<<"$child_info"
                MOCK_FS_FILES["$path"]="$exists|$child_type|$mode|$owner|$group|$size|$mtime|$content"
            done
        fi
    done
    
    # Save state after command
    _filesystem_mock_save_state
    
    return $exit_code
}

# Get file info helpers
mock::fs::get::file_content() {
    # Load state for subshell access
    _filesystem_mock_load_state
    
    local path="$1"
    path=$(_mock_normalize_path "$path")
    
    local info="${MOCK_FS_FILES[$path]:-}"
    if [[ -n "$info" ]]; then
        IFS='|' read -r exists type perms owner group size mtime content <<<"$info"
        echo "$content"
    fi
}

mock::fs::get::file_permissions() {
    # Load state for subshell access
    _filesystem_mock_load_state
    
    local path="$1"
    path=$(_mock_normalize_path "$path")
    
    local info="${MOCK_FS_FILES[$path]:-}"
    if [[ -n "$info" ]]; then
        IFS='|' read -r exists type perms _ <<<"$info"
        echo "$perms"
    fi
}

# Assertion helpers
mock::fs::assert::file_exists() {
    # Load state for subshell access
    _filesystem_mock_load_state
    
    local path="$1"
    path=$(_mock_normalize_path "$path")
    
    if [[ -z "${MOCK_FS_FILES[$path]}" ]]; then
        echo "ASSERTION FAILED: File '$path' does not exist" >&2
        return 1
    fi
    
    local info="${MOCK_FS_FILES[$path]}"
    IFS='|' read -r exists type _ <<<"$info"
    
    if [[ "$type" != "f" && "$type" != "l" ]]; then
        echo "ASSERTION FAILED: '$path' is not a file or symlink (type: $type)" >&2
        return 1
    fi
    
    return 0
}

mock::fs::assert::directory_exists() {
    # Load state for subshell access
    _filesystem_mock_load_state
    
    local path="$1"
    path=$(_mock_normalize_path "$path")
    
    if [[ -z "${MOCK_FS_FILES[$path]}" ]]; then
        echo "ASSERTION FAILED: Directory '$path' does not exist" >&2
        return 1
    fi
    
    local info="${MOCK_FS_FILES[$path]}"
    IFS='|' read -r exists type _ <<<"$info"
    
    if [[ "$type" != "d" ]]; then
        echo "ASSERTION FAILED: '$path' is not a directory (type: $type)" >&2
        return 1
    fi
    
    return 0
}

mock::fs::assert::not_exists() {
    # Load state for subshell access
    _filesystem_mock_load_state
    
    local path="$1"
    path=$(_mock_normalize_path "$path")
    
    if [[ -n "${MOCK_FS_FILES[$path]}" ]]; then
        echo "ASSERTION FAILED: Path '$path' exists but should not" >&2
        return 1
    fi
    
    return 0
}

# ----------------------------
# Test Helper Functions
# ----------------------------
# Scenario builders for common test patterns
mock::fs::scenario::create_project_structure() {
  local project_name="${1:-test_project}"
  # Handle both with and without leading slash
  local project_path
  if [[ "${project_name:0:1}" == "/" ]]; then
    project_path="$project_name"
  else
    project_path="/$project_name"
  fi
  
  # Create project directory structure
  mock::fs::create_directory "$project_path"
  mock::fs::create_directory "$project_path/src"
  mock::fs::create_directory "$project_path/tests"
  mock::fs::create_directory "$project_path/docs"
  
  # Create common files
  mock::fs::create_file "$project_path/README.md" "# $project_name

This is a test project structure."
  mock::fs::create_file "$project_path/package.json" "{
  \"name\": \"$project_name\",
  \"version\": \"1.0.0\",
  \"description\": \"Test project\"
}"
  mock::fs::create_file "$project_path/.gitignore" "node_modules/
*.log
.env"
  mock::fs::create_file "$project_path/src/index.js" "console.log('Hello World');"
  mock::fs::create_file "$project_path/tests/index.test.js" "// Test file"
  
  # Save state to ensure persistence
  _filesystem_mock_save_state
  
  # Don't output to stdout - it interferes with tests
  # echo "[MOCK] Created project structure: $project_name"
}

# Create a home directory structure for testing
mock::fs::scenario::create_home_directory() {
  local username="${1:-testuser}"
  local home_path="/home/$username"
  
  # Create home directory
  mock::fs::create_directory "$home_path" "755" "$username" "$username"
  
  # Create common subdirectories
  mock::fs::create_directory "$home_path/.ssh" "700" "$username" "$username"
  mock::fs::create_directory "$home_path/Documents" "755" "$username" "$username"
  mock::fs::create_directory "$home_path/Downloads" "755" "$username" "$username"
  
  # Create common files
  mock::fs::create_file "$home_path/.bashrc" "# .bashrc for $username
export PATH=\$PATH:/usr/local/bin" "644" "$username" "$username"
  mock::fs::create_file "$home_path/.profile" "# .profile for $username" "644" "$username" "$username"
  mock::fs::create_file "$home_path/.ssh/config" "# SSH config" "600" "$username" "$username"
  
  echo "[MOCK] Created home directory for: $username"
}

# Debug helper
mock::fs::debug::dump_state() {
  echo "=== Filesystem Mock State Dump ==="
  echo "Files and Directories:"
  for path in "${!MOCK_FS_FILES[@]}"; do
    echo "  $path: ${MOCK_FS_FILES[$path]}"
  done
  echo "Symlinks:"
  for link in "${!MOCK_FS_SYMLINKS[@]}"; do
    echo "  $link -> ${MOCK_FS_SYMLINKS[$link]}"
  done
  echo "Errors:"
  for cmd in "${!MOCK_FS_ERRORS[@]}"; do
    echo "  $cmd: ${MOCK_FS_ERRORS[$cmd]}"
  done
  echo "=========================="
}

# Add ln command for symlinks
ln() {
    # Load state from file for subshell access
    _filesystem_mock_load_state

    # Use centralized logging and verification
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "ln" "$*"
    fi

    case "$FILESYSTEM_MOCK_MODE" in
        readonly) echo "ln: Read-only file system" >&2; return 1 ;;
        error) echo "ln: Input/output error" >&2; return 1 ;;
    esac

    local symbolic=false
    local files=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -s|--symbolic) symbolic=true; shift ;;
            -*) shift ;;
            *) files+=("$1"); shift ;;
        esac
    done
    
    if [[ ${#files[@]} -lt 2 ]]; then
        echo "ln: missing destination file operand" >&2
        return 1
    fi
    
    local target="${files[0]}"
    local link="${files[1]}"
    
    target=$(_mock_normalize_path "$target")
    link=$(_mock_normalize_path "$link")
    
    if [[ "$symbolic" == true ]]; then
        mock::fs::create_symlink "$link" "$target"
    else
        # Hard link - just copy the file
        if [[ -z "${MOCK_FS_FILES[$target]}" ]]; then
            echo "ln: cannot create link '$link': No such file or directory" >&2
            return 1
        fi
        MOCK_FS_FILES["$link"]="${MOCK_FS_FILES[$target]}"
    fi
    
    # Save state after command
    _filesystem_mock_save_state
    
    return 0
}

#######################################
# Compatibility aliases for lifecycle tests
#######################################

#######################################
# Alias for create_file to match test expectations
# Arguments: Same as mock::fs::create_file
#######################################
mock::filesystem::create_file() {
    mock::fs::create_file "$@"
}

#######################################
# Alias for create_directory to match test expectations
# Arguments: Same as mock::fs::create_directory
#######################################
mock::filesystem::create_directory() {
    mock::fs::create_directory "$@"
}

#######################################
# Alias for create_symlink to match test expectations
# Arguments: Same as mock::fs::create_symlink
#######################################
mock::filesystem::create_symlink() {
    mock::fs::create_symlink "$@"
}

#######################################
# Alias for reset to match test expectations
#######################################
mock::filesystem::reset() {
    mock::fs::reset "$@"
}

# ----------------------------
# Export functions into subshells
# ----------------------------
# Exporting lets child bash processes (spawned by scripts under test) inherit mocks.
export -f ls cat touch mkdir rm cp mv find chmod ln
export -f _mock_current_time _mock_normalize_path _mock_parent_dir _mock_basename _mock_format_permissions
export -f _filesystem_mock_init_state_file _filesystem_mock_save_state _filesystem_mock_load_state
export -f mock::fs::reset mock::fs::enable_auto_cleanup mock::fs::inject_error
export -f mock::fs::create_file mock::fs::create_directory mock::fs::create_symlink
export -f mock::fs::get::file_content mock::fs::get::file_permissions
export -f mock::fs::assert::file_exists mock::fs::assert::directory_exists mock::fs::assert::not_exists

# Export test helper functions
export -f mock::fs::scenario::create_project_structure
export -f mock::fs::scenario::create_home_directory
export -f mock::fs::debug::dump_state

# Export compatibility aliases
export -f mock::filesystem::create_file mock::filesystem::create_directory mock::filesystem::create_symlink mock::filesystem::reset

# Initialize with root directory
# Don't call reset here - let the test setup do it
# mock::fs::reset

# Don't output to stdout - it interferes with tests
# echo "[MOCK] Filesystem mocks loaded successfully"