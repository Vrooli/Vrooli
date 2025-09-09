#!/usr/bin/env bash
# Improved Code File Content Extractor with Timeout and Error Handling
# Fixes hanging issues by adding timeouts, error recovery, and batch limits

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Define paths from APP_ROOT
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
EXTRACTOR_DIR="${EMBEDDINGS_DIR}/extractors"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified embedding service and code extractor
source "${EMBEDDINGS_DIR}/lib/embedding-service.sh"
source "${EMBEDDINGS_DIR}/lib/code-extractor.sh"

# Configuration with proper default handling
if [[ -z "${CODE_EXTRACT_TIMEOUT:-}" ]]; then
    readonly CODE_EXTRACT_TIMEOUT=10  # Timeout per file in seconds (reduced from 5)
fi
if [[ -z "${MAX_FILES_PER_BATCH:-}" ]]; then
    readonly MAX_FILES_PER_BATCH=50  # Process in chunks to prevent memory issues (increased for better performance)
fi
if [[ -z "${MAX_RETRIES:-}" ]]; then
    readonly MAX_RETRIES=1  # Retry failed files this many times (reduced from 2)
fi
if [[ -z "${SKIP_LARGE_FILES_MB:-}" ]]; then
    readonly SKIP_LARGE_FILES_MB=5  # Skip files larger than this (in MB) (reduced from 10)
fi

#######################################
# Find code files in a directory
# Arguments:
#   $1 - Directory to search
# Returns: List of code files (one per line)
#######################################
qdrant::extract::find_code() {
    local directory="${1:-.}"
    
    find "$directory" -type f \( \
        -name "*.js" -o -name "*.jsx" -o -name "*.ts" -o -name "*.tsx" \
        -o -name "*.py" -o -name "*.go" -o -name "*.rs" \
        -o -name "*.java" -o -name "*.cs" -o -name "*.rb" \
        -o -name "*.php" -o -name "*.swift" -o -name "*.kt" \
        -o -name "*.cpp" -o -name "*.c" -o -name "*.h" -o -name "*.hpp" \
        -o -name "*.sql" -o -name "*.sh" -o -name "*.bash" \
        -o -name "*.css" -o -name "*.scss" -o -name "*.sass" \
        -o -name "*.m" -o -name "*.r" -o -name "*.R" \
    \) 2>/dev/null | grep -v node_modules | grep -v ".git/" | grep -v "__pycache__" | sort
}

#######################################
# Extract code with timeout and error handling
# Arguments:
#   $1 - Path to code file
# Returns: JSON line with file-level summary and metadata
#######################################
qdrant::extract::code_safe() {
    local file="$1"
    local timeout_sec="${2:-$CODE_EXTRACT_TIMEOUT}"
    
    if [[ ! -f "$file" ]]; then
        log::debug "File not found: $file"
        return 1
    fi
    
    # Check file size
    local file_size_mb=$(( $(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo 0) / 1048576 ))
    if [[ $file_size_mb -gt $SKIP_LARGE_FILES_MB ]]; then
        log::warn "Skipping large file (${file_size_mb}MB): $file"
        return 2
    fi
    
    # Create a temporary file for output
    local temp_output=$(mktemp)
    trap "rm -f $temp_output" RETURN
    
    # Run extraction with timeout
    if timeout "$timeout_sec" bash -c "
        export APP_ROOT='$APP_ROOT'
        source '${EMBEDDINGS_DIR}/extractors/code/main-original.sh'
        qdrant::extract::code '$file'
    " > "$temp_output" 2>/dev/null; then
        # Success - output the result
        cat "$temp_output"
        return 0
    else
        local exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            log::warn "Timeout extracting file after ${timeout_sec}s: $file"
        else
            log::debug "Failed to extract file (exit $exit_code): $file"
        fi
        return $exit_code
    fi
}

#######################################
# Process code files in batch with robust error handling
# Arguments:
#   $1 - Directory to scan
#   $2 - Output file path
# Returns: 0 on success
#######################################
qdrant::extract::code_batch_improved() {
    local directory="$1" 
    local output_file="$2"
    
    if [[ -z "$directory" || -z "$output_file" ]]; then
        log::error "Usage: qdrant::extract::code_batch_improved <directory> <output_file>"
        return 1
    fi
    
    # Ensure output directory exists and clear output file
    mkdir -p "${output_file%/*}"
    > "$output_file"
    
    # Create tracking files
    local failed_files=$(mktemp)
    local processed_files=$(mktemp)
    trap "rm -f $failed_files $processed_files" RETURN
    
    # Get list of files to process with prioritization
    local file_list=$(mktemp)
    local prioritized_list=$(mktemp)
    qdrant::extract::find_code "$directory" > "$file_list"
    
    # Prioritize files: main/index files first, then core files, then others
    {
        grep -E "(main\.|index\.|app\.|server\.|client\.|init\.)" "$file_list" || true
        grep -E "(lib/|src/|core/)" "$file_list" | grep -v -E "(test|spec)" || true
        grep -v -E "(main\.|index\.|app\.|server\.|client\.|init\.|lib/|src/|core/)" "$file_list" | grep -v -E "(test|spec|node_modules)" || true
        grep -E "(test|spec)" "$file_list" || true
    } > "$prioritized_list"
    
    mv "$prioritized_list" "$file_list"
    local total_files=$(wc -l < "$file_list")
    
    if [[ $total_files -eq 0 ]]; then
        log::info "No code files found in $directory"
        return 0
    fi
    
    log::info "Starting extraction of $total_files code files (batch size: $MAX_FILES_PER_BATCH)"
    
    # Limit total files to process to prevent endless runs (configurable via environment)
    local max_total_files=${CODE_MAX_FILES:-1500}
    if [[ $total_files -gt $max_total_files ]]; then
        log::warn "Limiting extraction to first $max_total_files files (out of $total_files)"
        log::info "Increase CODE_MAX_FILES environment variable to process more files"
        total_files=$max_total_files
    fi
    
    local count=0
    local failed_count=0
    local skipped_count=0
    local batch_num=1
    local files_in_batch=0
    local files_processed=0
    
    # Process files with batch limits
    while IFS= read -r file; do
        # Stop if we've reached the max files limit
        if [[ $files_processed -ge $max_total_files ]]; then
            log::info "Reached maximum file limit ($max_total_files). Stopping extraction."
            break
        fi
        ((files_processed++))
        # Check if we've reached batch limit
        if [[ $files_in_batch -ge $MAX_FILES_PER_BATCH ]]; then
            log::info "Completed batch $batch_num ($files_in_batch files processed)"
            ((batch_num++))
            files_in_batch=0
            
            # Small pause between batches to prevent resource exhaustion
            sleep 0.5
        fi
        
        # Skip if already processed
        if grep -q "^$file$" "$processed_files" 2>/dev/null; then
            continue
        fi
        
        # Try to extract with timeout
        if qdrant::extract::code_safe "$file" >> "$output_file"; then
            ((count++))
            ((files_in_batch++))
            echo "$file" >> "$processed_files"
        else
            local exit_code=$?
            if [[ $exit_code -eq 2 ]]; then
                # Large file skipped
                ((skipped_count++))
            else
                # Extraction failed
                ((failed_count++))
                echo "$file" >> "$failed_files"
            fi
        fi
        
        # Progress indicator
        local total_processed=$((count + failed_count + skipped_count))
        if [[ $((total_processed % 50)) -eq 0 ]]; then
            log::info "Progress: $total_processed/$total_files (✓$count ✗$failed_count ⊘$skipped_count)"
        fi
        
        # Early exit if too many failures
        if [[ $failed_count -gt 100 ]] && [[ $failed_count -gt $((count / 2)) ]]; then
            log::error "Too many failures ($failed_count). Stopping extraction."
            break
        fi
    done < "$file_list"
    
    # Retry failed files once
    if [[ -s "$failed_files" ]] && [[ $failed_count -lt 50 ]]; then
        log::info "Retrying $failed_count failed files..."
        local retry_count=0
        
        while IFS= read -r file; do
            if qdrant::extract::code_safe "$file" 10 >> "$output_file"; then
                ((count++))
                ((retry_count++))
                ((failed_count--))
            fi
        done < "$failed_files"
        
        if [[ $retry_count -gt 0 ]]; then
            log::info "Recovered $retry_count files on retry"
        fi
    fi
    
    # Final report
    log::success "Extraction complete: $count successful, $failed_count failed, $skipped_count skipped"
    
    if [[ $failed_count -gt 0 ]]; then
        log::warn "Failed files saved to: $failed_files.log"
        cp "$failed_files" "$failed_files.log"
    fi
    
    echo "$output_file"
    return 0
}

#######################################
# Process code using unified embedding service (improved)
# Arguments:
#   $1 - App ID
# Returns: Number of code files processed
#######################################
qdrant::embeddings::process_code() {
    local app_id="$1"
    local collection="${app_id}-code"
    local count=0
    
    # Extract code to temp file with improved batch processing
    local output_file="$TEMP_DIR/code.jsonl"
    
    # Use the improved batch processor - search from APP_ROOT
    if ! qdrant::extract::code_batch_improved "$APP_ROOT" "$output_file" >&2; then
        log::error "Code extraction failed"
        echo "0"
        return 1
    fi
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No code found for processing"
        echo "0"
        return 0
    fi
    
    # Process with smaller batch sizes for stability
    count=$(qdrant::embedding::process_jsonl_file "$output_file" "code" "$collection" "$app_id" 3)
    
    log::debug "Created $count code embeddings"
    echo "$count"
}

# Keep the original find_code function
qdrant::extract::find_code() {
    local directory="$1"
    
    find "$directory" -type f \( \
        -name "*.js" -o -name "*.ts" -o -name "*.jsx" -o -name "*.tsx" -o \
        -name "*.py" -o -name "*.ipynb" -o -name "*.go" -o -name "*.sh" -o -name "*.bash" -o \
        -name "*.java" -o -name "*.cpp" -o -name "*.c" -o -name "*.h" -o \
        -name "*.rb" -o -name "*.rs" -o -name "*.php" -o -name "*.swift" -o \
        -name "*.kt" -o -name "*.scala" -o -name "*.r" -o -name "*.R" -o \
        -name "*.Rmd" -o -name "*.rmd" -o -name "*.m" -o -name "*.mlx" -o \
        -name "*.cs" -o -name "*.css" -o -name "*.scss" -o -name "*.sass" -o \
        -name "*.less" -o -name "*.sql" -o -name "*.ddl" -o -name "*.dml" \
    \) \
    ! -path "*/node_modules/*" \
    ! -path "*/.git/*" \
    ! -path "*/dist/*" \
    ! -path "*/build/*" \
    ! -path "*/vendor/*" \
    ! -path "*/.venv/*" \
    ! -path "*/venv/*" \
    ! -path "*/target/*" \
    ! -path "*/__pycache__/*" \
    2>/dev/null
}

#######################################
# Extract metadata from code content text
# Arguments:
#   $1 - Code content text
# Returns: JSON metadata object
#######################################
# Wrapper for backward compatibility
qdrant::extract::code_batch() {
    qdrant::extract::code_batch_improved "$@"
}

qdrant::extract::code_metadata_from_content() {
    local content="$1"
    
    # Extract file information from content
    local file_path
    file_path=$(echo "$content" | grep "^File: " | cut -d: -f2- | sed 's/^ *//')
    
    local filename=""
    local file_ext=""
    local language=""
    if [[ -n "$file_path" ]]; then
        filename=$(basename "$file_path")
        file_ext="${filename##*.}"
        
        # Use language mapping consistent with main extraction
        case "$file_ext" in
            js|jsx) language="javascript" ;;
            ts|tsx) language="typescript" ;;
            py|ipynb) language="python" ;;
            sh|bash) language="shell" ;;
            go) language="go" ;;
            rs) language="rust" ;;
            java) language="java" ;;
            cpp|cc|cxx) language="cpp" ;;
            c) language="c" ;;
            h|hpp) language="header" ;;
            rb) language="ruby" ;;
            php) language="php" ;;
            swift) language="swift" ;;
            kt) language="kotlin" ;;
            cs) language="csharp" ;;
            sql|ddl|dml) language="sql" ;;
            css|scss|sass|less) language="css" ;;
            m|mlx) language="matlab" ;;
            R|r|Rmd|rmd) language="r" ;;
            scala) language="scala" ;;
            *) language="unknown" ;;
        esac
    fi
    
    # Build metadata JSON
    jq -n \
        --arg filename "${filename:-Unknown}" \
        --arg file_path "${file_path:-}" \
        --arg file_ext "${file_ext:-}" \
        --arg language "${language:-unknown}" \
        '{
            filename: $filename,
            source_file: $file_path,
            file_extension: $file_ext,
            language: $language,
            content_type: "code",
            extractor: "code_improved"
        }'
}

# Export functions
export -f qdrant::extract::code_safe
export -f qdrant::extract::code_batch
export -f qdrant::extract::code_batch_improved
export -f qdrant::embeddings::process_code
export -f qdrant::extract::find_code
export -f qdrant::extract::code_metadata_from_content