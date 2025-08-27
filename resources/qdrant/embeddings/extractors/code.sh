#!/usr/bin/env bash
# Code File Content Extractor for Qdrant Embeddings
# Extracts FILE-LEVEL summaries instead of individual functions

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Define paths from APP_ROOT
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
EXTRACTOR_DIR="${EMBEDDINGS_DIR}/extractors"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified embedding service
source "${EMBEDDINGS_DIR}/lib/embedding-service.sh"

# Extract FILE-LEVEL summary with structured metadata
qdrant::extract::code() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local file_ext="${file##*.}"
    local filename=$(basename "$file")
    local dir=${file%/*}
    local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
    local line_count=$(wc -l < "$file" 2>/dev/null || echo "0")
    
    # Determine language/type
    local language=""
    case "$file_ext" in
        js|jsx) language="JavaScript" ;;
        ts|tsx) language="TypeScript" ;;
        py) language="Python" ;;
        sh|bash) language="Shell/Bash" ;;
        go) language="Go" ;;
        java) language="Java" ;;
        cpp|cc|c) language="C/C++" ;;
        h|hpp) language="C/C++ Header" ;;
        rb) language="Ruby" ;;
        rs) language="Rust" ;;
        php) language="PHP" ;;
        swift) language="Swift" ;;
        kt) language="Kotlin" ;;
        *) language="${file_ext^^}" ;;
    esac
    
    # Extract main purpose from header comments (first 10 lines)
    local purpose=$(head -10 "$file" 2>/dev/null | grep -E "^[[:space:]]*(//|#|\*|/\*)" | head -3 | sed 's/^[[:space:]]*[/#*]*//' | tr '\n' ' ' | sed 's/  */ /g' | cut -c1-200)
    
    # Count key elements for summary (ensure single line output)
    local function_count=$(grep -c -E "^[[:space:]]*(function|def |func |const.*=.*=>|export.*function)" "$file" 2>/dev/null | tr -d '\n' || echo "0")
    local class_count=$(grep -c -E "^[[:space:]]*(class |interface |struct |type )" "$file" 2>/dev/null | tr -d '\n' || echo "0")
    local import_count=$(grep -c -E "^(import |require\(|from |use )" "$file" 2>/dev/null | tr -d '\n' || echo "0")
    
    # Extract main exports/public API (for better searchability)
    local exports=""
    if [[ "$file_ext" == "ts" || "$file_ext" == "js" || "$file_ext" == "tsx" || "$file_ext" == "jsx" ]]; then
        exports=$(grep -E "^export (function|class|const|interface|type) [a-zA-Z_]+" "$file" 2>/dev/null | head -5 | sed 's/export //' | sed 's/[({].*//' | tr '\n' ',' | sed 's/,$//')
    elif [[ "$file_ext" == "py" ]]; then
        exports=$(grep -E "^(def |class )[a-zA-Z_]+[^_]" "$file" 2>/dev/null | head -5 | sed 's/def //' | sed 's/class //' | sed 's/[(:].*//' | tr '\n' ',' | sed 's/,$//')
    elif [[ "$file_ext" == "sh" || "$file_ext" == "bash" ]]; then
        exports=$(grep -E "^[a-zA-Z_][a-zA-Z0-9_:]*\(\)" "$file" 2>/dev/null | head -5 | sed 's/().*//' | tr '\n' ',' | sed 's/,$//')
    fi
    
    # Check for test file
    local is_test="no"
    if [[ "$filename" == *test* ]] || [[ "$filename" == *spec* ]] || [[ "$dir" == *test* ]] || [[ "$dir" == *__test__* ]]; then
        is_test="yes"
    fi
    
    # Build clean content summary (optimized for semantic search)
    local content_summary="This is a $language code file named '$filename' located in $dir."
    
    if [[ -n "$purpose" ]]; then
        content_summary="$content_summary Purpose: $purpose"
    fi
    
    if [[ "$is_test" == "yes" ]]; then
        content_summary="$content_summary This is a test file containing test cases and specifications."
    fi
    
    content_summary="$content_summary The file contains $line_count lines of code"
    if [[ $function_count -gt 0 ]]; then
        content_summary="$content_summary with $function_count functions/methods"
    fi
    if [[ $class_count -gt 0 ]]; then
        content_summary="$content_summary, $class_count classes/interfaces/types"
    fi
    if [[ $import_count -gt 0 ]]; then
        content_summary="$content_summary, $import_count imports"
    fi
    content_summary="${content_summary}."
    
    if [[ -n "$exports" ]]; then
        content_summary="$content_summary Main exports/API: $exports"
    fi
    
    # Output JSON with clean separation of content and metadata
    jq -n \
        --arg content "$content_summary" \
        --arg file "$file" \
        --arg filename "$filename" \
        --arg directory "$dir" \
        --arg language "$language" \
        --arg language_ext "$file_ext" \
        --arg size "$file_size" \
        --arg lines "$line_count" \
        --arg functions "$function_count" \
        --arg classes "$class_count" \
        --arg imports "$import_count" \
        --arg exports "$exports" \
        --arg is_test "$is_test" \
        --arg purpose "$purpose" \
        '{
            content: $content,
            metadata: {
                file: $file,
                filename: $filename,
                directory: $directory,
                language: $language,
                language_ext: $language_ext,
                size: ($size | tonumber),
                lines: ($lines | tonumber),
                functions: ($functions | tonumber),
                classes: ($classes | tonumber),
                imports: ($imports | tonumber),
                exports: $exports,
                is_test: ($is_test == "yes"),
                purpose: $purpose
            }
        }' | jq -c
}

# Find code files (excluding common non-code directories)
qdrant::extract::find_code() {
    local directory="$1"
    
    find "$directory" -type f \( \
        -name "*.js" -o -name "*.ts" -o -name "*.jsx" -o -name "*.tsx" -o \
        -name "*.py" -o -name "*.go" -o -name "*.sh" -o -name "*.bash" -o \
        -name "*.java" -o -name "*.cpp" -o -name "*.c" -o -name "*.h" -o \
        -name "*.rb" -o -name "*.rs" -o -name "*.php" -o -name "*.swift" -o \
        -name "*.kt" -o -name "*.scala" -o -name "*.r" -o -name "*.m" \
    \) \
    ! -path "*/node_modules/*" \
    ! -path "*/.git/*" \
    ! -path "*/dist/*" \
    ! -path "*/build/*" \
    ! -path "*/vendor/*" \
    ! -path "*/.venv/*" \
    ! -path "*/venv/*" \
    2>/dev/null
}

# Process code files in batch with FILE-LEVEL summaries
qdrant::extract::code_batch() {
    local directory="$1" 
    local output_file="$2"
    
    if [[ -z "$directory" || -z "$output_file" ]]; then
        log::error "Usage: qdrant::extract::code_batch <directory> <output_file>"
        return 1
    fi
    
    # Ensure output directory exists and clear output file
    mkdir -p "${output_file%/*}"
    > "$output_file"
    
    local count=0
    local total_files=$(qdrant::extract::find_code "$directory" | wc -l)
    
    log::info "Extracting content from $total_files code files"
    
    while IFS= read -r file; do
        # Each file gets ONE embedding entry (output JSON)
        qdrant::extract::code "$file" >> "$output_file"
        ((count++))
        
        # Progress indicator every 50 files
        if [[ $((count % 50)) -eq 0 ]]; then
            log::debug "Processed $count/$total_files code files..."
        fi
    done < <(qdrant::extract::find_code "$directory")
    
    log::success "Extracted content from $count code files"
    echo "$output_file"
    return 0
}

#######################################
# Process code using unified embedding service
# Arguments:
#   $1 - App ID
# Returns: Number of code files processed
#######################################
qdrant::embeddings::process_code() {
    local app_id="$1"
    local collection="${app_id}-code"
    local count=0
    
    # Extract code to temp file
    local output_file="$TEMP_DIR/code.jsonl"
    qdrant::extract::code_batch "." "$output_file" >&2
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No code found for processing"
        echo "0"
        return 0
    fi
    
    # Process each JSON line through unified embedding service
    while IFS= read -r json_line; do
        if [[ -n "$json_line" ]]; then
            # Parse JSON to extract content and metadata
            local content
            content=$(echo "$json_line" | jq -r '.content // empty' 2>/dev/null)
            
            local metadata
            metadata=$(echo "$json_line" | jq -c '.metadata // {}' 2>/dev/null)
            
            if [[ -n "$content" ]]; then
                # Process through unified embedding service with structured metadata
                if qdrant::embedding::process_item "$content" "code" "$collection" "$app_id" "$metadata"; then
                    ((count++))
                fi
            fi
        fi
    done < "$output_file"
    
    log::debug "Created $count code embeddings"
    echo "$count"
}

#######################################
# Extract metadata from code content text
# Arguments:
#   $1 - Code content text
# Returns: JSON metadata object
#######################################
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
        
        # Determine language from file extension
        case "$file_ext" in
            js|jsx) language="javascript" ;;
            ts|tsx) language="typescript" ;;
            py) language="python" ;;
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
            scala) language="scala" ;;
            r) language="r" ;;
            m) language="matlab" ;;
            *) language="unknown" ;;
        esac
    fi
    
    # Extract language info if present
    local detected_language
    detected_language=$(echo "$content" | grep "^Language: " | cut -d: -f2- | sed 's/^ *//' | cut -d' ' -f1)
    if [[ -n "$detected_language" ]]; then
        language="$detected_language"
    fi
    
    # Extract size if present
    local file_size
    file_size=$(echo "$content" | grep "^Size: " | cut -d: -f2- | sed 's/^ *//' | cut -d' ' -f1)
    
    # Estimate complexity based on content
    local content_length
    content_length=$(echo -n "$content" | wc -c)
    
    local line_count
    line_count=$(echo "$content" | wc -l)
    
    # Determine file category
    local category="source"
    if [[ "$file_ext" == "h" ]] || [[ "$file_ext" == "hpp" ]]; then
        category="header"
    elif [[ "$language" == "shell" ]]; then
        category="script"
    elif [[ "$filename" == *"test"* ]] || [[ "$filename" == *"spec"* ]]; then
        category="test"
    elif [[ "$filename" == *"config"* ]] || [[ "$filename" == *"setup"* ]]; then
        category="configuration"
    fi
    
    # Build metadata JSON - ensure numeric fields have valid defaults
    local safe_file_size="${file_size:-0}"
    local safe_content_length="${content_length:-0}"
    local safe_line_count="${line_count:-0}"
    
    # Ensure numeric fields are not empty
    [[ -z "$safe_file_size" || "$safe_file_size" == " " ]] && safe_file_size="0"
    [[ -z "$safe_content_length" || "$safe_content_length" == " " ]] && safe_content_length="0"
    [[ -z "$safe_line_count" || "$safe_line_count" == " " ]] && safe_line_count="0"
    
    jq -n \
        --arg filename "${filename:-Unknown}" \
        --arg file_path "${file_path:-}" \
        --arg file_ext "${file_ext:-}" \
        --arg language "${language:-unknown}" \
        --arg category "$category" \
        --arg file_size "$safe_file_size" \
        --arg content_length "$safe_content_length" \
        --arg line_count "$safe_line_count" \
        '{
            filename: $filename,
            source_file: $file_path,
            file_extension: $file_ext,
            language: $language,
            category: $category,
            file_size: ($file_size | tonumber),
            content_length: ($content_length | tonumber),
            line_count: ($line_count | tonumber),
            content_type: "code",
            extractor: "code"
        }'
}

# Export functions
export -f qdrant::extract::code
export -f qdrant::extract::find_code
export -f qdrant::extract::code_batch
export -f qdrant::embeddings::process_code