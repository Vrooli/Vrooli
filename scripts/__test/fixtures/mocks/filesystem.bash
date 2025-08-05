#!/usr/bin/env bash
# Filesystem System Mocks
# Provides mocks for filesystem operations (find, tar, mkdir, etc.)

# Prevent duplicate loading
if [[ "${FILESYSTEM_MOCKS_LOADED:-}" == "true" ]]; then
    return 0
fi
export FILESYSTEM_MOCKS_LOADED="true"

# Filesystem mock state storage
declare -gA MOCK_FILE_CONTENTS
declare -gA MOCK_DIRECTORY_CONTENTS
declare -gA MOCK_FILE_PERMISSIONS

#######################################
# Set mock file content
# Arguments: $1 - file path, $2 - content
#######################################
mock::file::set_content() {
    local file_path="$1"
    local content="$2"
    
    MOCK_FILE_CONTENTS["$file_path"]="$content"
}

#######################################
# Set mock directory contents
# Arguments: $1 - directory path, $2 - space-separated file list
#######################################
mock::directory::set_contents() {
    local dir_path="$1"
    local contents="$2"
    
    MOCK_DIRECTORY_CONTENTS["$dir_path"]="$contents"
}

#######################################
# Set mock file permissions
# Arguments: $1 - file path, $2 - permissions (e.g., "755")
#######################################
mock::file::set_permissions() {
    local file_path="$1"
    local permissions="$2"
    
    MOCK_FILE_PERMISSIONS["$file_path"]="$permissions"
}

#######################################
# find command mock
#######################################
find() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "find $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local search_path="${1:-.}"
    local criteria=()
    
    # Parse find arguments
    shift
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-name")
                criteria+=("name:$2")
                shift 2
                ;;
            "-type")
                criteria+=("type:$2")
                shift 2
                ;;
            "-exec")
                criteria+=("exec:$2")
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Generate mock results based on criteria
    for criterion in "${criteria[@]}"; do
        local type="${criterion%%:*}"
        local value="${criterion##*:}"
        
        case "$type" in
            "name")
                case "$value" in
                    "*.json")
                        echo "$search_path/config.json"
                        echo "$search_path/package.json"
                        echo "$search_path/settings.json"
                        ;;
                    "*.log")
                        echo "$search_path/app.log"
                        echo "$search_path/error.log"
                        ;;
                    "*.bats")
                        echo "$search_path/test1.bats"
                        echo "$search_path/test2.bats"
                        ;;
                    "docker-compose.yml")
                        echo "$search_path/docker-compose.yml"
                        ;;
                    "Dockerfile")
                        echo "$search_path/Dockerfile"
                        ;;
                    *)
                        echo "$search_path/mock_file_${value//\*/match}"
                        ;;
                esac
                ;;
            "type")
                case "$value" in
                    "f")
                        echo "$search_path/file1.txt"
                        echo "$search_path/file2.txt"
                        ;;
                    "d")
                        echo "$search_path/dir1"
                        echo "$search_path/dir2"
                        ;;
                esac
                ;;
        esac
    done
    
    # If no criteria, just return the search path
    if [[ ${#criteria[@]} -eq 0 ]]; then
        echo "$search_path"
    fi
}

#######################################
# ls command mock
#######################################
ls() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "ls $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local path="${1:-.}"
    local long_format=false
    local all_files=false
    
    # Parse ls arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-l")
                long_format=true
                shift
                ;;
            "-a")
                all_files=true
                shift
                ;;
            "-la"|"-al")
                long_format=true
                all_files=true
                shift
                ;;
            *)
                path="$1"
                shift
                ;;
        esac
    done
    
    # Check for mock directory contents
    if [[ -n "${MOCK_DIRECTORY_CONTENTS[$path]:-}" ]]; then
        local contents="${MOCK_DIRECTORY_CONTENTS[$path]}"
        if [[ "$long_format" == true ]]; then
            echo "total 8"
            for file in $contents; do
                local perms="${MOCK_FILE_PERMISSIONS[$path/$file]:-644}"
                echo "-rw-r--r-- 1 user user 1024 $(date '+%b %d %H:%M') $file"
            done
        else
            echo "$contents" | tr ' ' '\n'
        fi
        return 0
    fi
    
    # Default mock contents
    local files=("file1.txt" "file2.txt" "config.json")
    
    if [[ "$all_files" == true ]]; then
        files=("." ".." ".hidden" "${files[@]}")
    fi
    
    if [[ "$long_format" == true ]]; then
        echo "total 8"
        for file in "${files[@]}"; do
            if [[ "$file" =~ ^\. ]]; then
                echo "drwxr-xr-x 2 user user 4096 $(date '+%b %d %H:%M') $file"
            else
                echo "-rw-r--r-- 1 user user 1024 $(date '+%b %d %H:%M') $file"
            fi
        done
    else
        printf '%s\n' "${files[@]}"
    fi
}

#######################################
# cat command mock
#######################################
cat() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "cat $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    if [[ $# -eq 0 ]]; then
        # Read from stdin
        while IFS= read -r line; do
            echo "$line"
        done
    else
        for file in "$@"; do
            # Check for mock file content
            if [[ -n "${MOCK_FILE_CONTENTS[$file]:-}" ]]; then
                echo "${MOCK_FILE_CONTENTS[$file]}"
            else
                # Generate content based on file type
                case "$file" in
                    *.json)
                        echo '{"status": "ok", "version": "1.0.0"}'
                        ;;
                    *.yaml|*.yml)
                        echo "version: '3.8'"
                        echo "services:"
                        echo "  app:"
                        echo "    image: test:latest"
                        ;;
                    *.log)
                        echo "$(date): [INFO] Application started"
                        echo "$(date): [INFO] Service ready"
                        ;;
                    */VERSION|*version*)
                        echo "1.0.0"
                        ;;
                    *)
                        echo "Mock content for $file"
                        ;;
                esac
            fi
        done
    fi
}

#######################################
# head command mock
#######################################
head() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "head $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local lines=10
    local file=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-n")
                lines="$2"
                shift 2
                ;;
            *)
                file="$1"
                shift
                ;;
        esac
    done
    
    if [[ -n "$file" ]]; then
        cat "$file" | head -n "$lines"
    else
        # Read from stdin
        local count=0
        while IFS= read -r line && [[ $count -lt $lines ]]; do
            echo "$line"
            ((count++))
        done
    fi
}

#######################################
# tail command mock
#######################################
tail() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "tail $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local lines=10
    local file=""
    local follow=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-n")
                lines="$2"
                shift 2
                ;;
            "-f")
                follow=true
                shift
                ;;
            *)
                file="$1"
                shift
                ;;
        esac
    done
    
    if [[ -n "$file" ]]; then
        # Generate tail content
        for ((i=1; i<=lines; i++)); do
            echo "$(date): [INFO] Log line $i from $file"
        done
        
        if [[ "$follow" == true ]]; then
            # Simulate following logs
            echo "$(date): [INFO] Following logs..."
        fi
    else
        # Read from stdin and show last lines
        local input_lines=()
        while IFS= read -r line; do
            input_lines+=("$line")
        done
        
        local start=$((${#input_lines[@]} - lines))
        if [[ $start -lt 0 ]]; then
            start=0
        fi
        
        for ((i=start; i<${#input_lines[@]}; i++)); do
            echo "${input_lines[$i]}"
        done
    fi
}

#######################################
# grep command mock
#######################################
grep() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "grep $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local pattern=""
    local files=()
    local quiet=false
    local invert=false
    local count=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-q"|"--quiet")
                quiet=true
                shift
                ;;
            "-v"|"--invert-match")
                invert=true
                shift
                ;;
            "-c"|"--count")
                count=true
                shift
                ;;
            *)
                if [[ -z "$pattern" ]]; then
                    pattern="$1"
                else
                    files+=("$1")
                fi
                shift
                ;;
        esac
    done
    
    # Simulate grep behavior
    local found=false
    
    if [[ ${#files[@]} -eq 0 ]]; then
        # Read from stdin
        while IFS= read -r line; do
            if [[ "$line" =~ $pattern ]]; then
                found=true
                if [[ "$quiet" != true ]]; then
                    echo "$line"
                fi
            fi
        done
    else
        # Search in files
        for file in "${files[@]}"; do
            # Check mock file content first
            local content
            if [[ -n "${MOCK_FILE_CONTENTS[$file]:-}" ]]; then
                content="${MOCK_FILE_CONTENTS[$file]}"
            else
                content="Mock content containing $pattern for testing"
            fi
            
            local match_count=0
            while IFS= read -r line; do
                if [[ "$line" =~ $pattern ]]; then
                    found=true
                    ((match_count++))
                    if [[ "$quiet" != true && "$count" != true ]]; then
                        if [[ ${#files[@]} -gt 1 ]]; then
                            echo "$file:$line"
                        else
                            echo "$line"
                        fi
                    fi
                fi
            done <<< "$content"
            
            if [[ "$count" == true ]]; then
                if [[ ${#files[@]} -gt 1 ]]; then
                    echo "$file:$match_count"
                else
                    echo "$match_count"
                fi
            fi
        done
    fi
    
    # Return appropriate exit code
    if [[ "$found" == true ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# tar command mock
#######################################
tar() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "tar $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local operation=""
    local archive=""
    local files=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-c"|"-x"|"-t")
                operation="$1"
                shift
                ;;
            "-f")
                archive="$2"
                shift 2
                ;;
            "-czf")
                operation="-c"
                archive="$2"
                shift 2
                ;;
            "-xzf")
                operation="-x"
                archive="$2"
                shift 2
                ;;
            *)
                files+=("$1")
                shift
                ;;
        esac
    done
    
    case "$operation" in
        "-c")
            echo "Creating archive $archive from ${files[*]}"
            # Create a mock archive file
            if [[ -n "$archive" ]]; then
                echo "Mock archive content" > "$archive"
            fi
            ;;
        "-x")
            echo "Extracting archive $archive"
            # Create mock extracted files
            echo "file1.txt" > "file1.txt"
            echo "file2.txt" > "file2.txt"
            ;;
        "-t")
            echo "file1.txt"
            echo "file2.txt"
            echo "directory/"
            ;;
    esac
}

#######################################
# mkdir command mock
#######################################
mkdir() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "mkdir $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # In test environment, always succeed
    return 0
}

#######################################
# rmdir command mock
#######################################
rmdir() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "rmdir $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # In test environment, always succeed
    return 0
}

#######################################
# rm command mock
#######################################
rm() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "rm $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # In test environment, always succeed
    return 0
}

#######################################
# cp command mock
#######################################
cp() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "cp $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # In test environment, always succeed
    return 0
}

#######################################
# mv command mock
#######################################
mv() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "mv $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # In test environment, always succeed
    return 0
}

#######################################
# chmod command mock
#######################################
chmod() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "chmod $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local permissions="$1"
    shift
    
    # Store permissions for files
    for file in "$@"; do
        mock::file::set_permissions "$file" "$permissions"
    done
    
    return 0
}

#######################################
# chown command mock
#######################################
chown() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "chown $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # In test environment, always succeed
    return 0
}

#######################################
# stat command mock
#######################################
stat() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "stat $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local format=""
    local file=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-c"|"--format")
                format="$2"
                shift 2
                ;;
            "-f"|"--file-system")
                # File system stat
                shift
                ;;
            *)
                file="$1"
                shift
                ;;
        esac
    done
    
    if [[ -n "$format" ]]; then
        case "$format" in
            "%a")
                echo "${MOCK_FILE_PERMISSIONS[$file]:-644}"
                ;;
            "%A")
                echo "${MOCK_FILE_PERMISSIONS[$file]:-644}"
                ;;
            "%s")
                echo "1024"
                ;;
            *)
                echo "mock-stat-value"
                ;;
        esac
    else
        echo "  File: $file"
        echo "  Size: 1024      Blocks: 8          IO Block: 4096   regular file"
        echo "Device: 801h/2049d  Inode: 123456      Links: 1"
        echo "Access: (0644/-rw-r--r--)  Uid: ( 1000/    user)   Gid: ( 1000/    user)"
    fi
}

# Export all functions
export -f find ls cat head tail grep tar mkdir rmdir rm cp mv chmod chown stat
export -f mock::file::set_content mock::directory::set_contents mock::file::set_permissions

echo "[FILESYSTEM_MOCKS] Filesystem system mocks loaded"