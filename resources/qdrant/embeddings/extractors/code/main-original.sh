#!/usr/bin/env bash
# Code File Content Extractor for Qdrant Embeddings
# 
# SCOPE: Processes APPLICATION SOURCE CODE files outside of:
# - initialization/ folder (handled by resources/initialization stream)
# - scenarios/ folder (handled by scenarios stream) 
# - resources/ folder configuration files (handled by resources stream)
#
# PROCESSING: Provides FILE-LEVEL summaries with deep language-specific analysis
# - Uses unified code-extractor.sh for 16 language detection and parsing
# - Creates semantic summaries at file level (not individual functions)
# - Extracts: language, functions, classes, imports, dependencies, test indicators
# - Output: High-level file descriptions for semantic search
#
# COVERAGE: Processes source code files (.js, .py, .sh, .ts, .rs, .go, etc.)
# throughout the application codebase for architectural understanding

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

#######################################
# Extract FILE-LEVEL summary with deep language analysis
# 
# Uses unified code-extractor for language detection and parsing,
# then aggregates results to file level
#
# Arguments:
#   $1 - Path to code file
# Returns: JSON line with file-level summary and metadata
#######################################
qdrant::extract::code() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local filename=$(basename "$file")
    local dir=${file%/*}
    local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
    local line_count=$(wc -l < "$file" 2>/dev/null || echo "0")
    local file_ext="${filename##*.}"
    
    # Use unified system for primary language detection
    local detected_lang=$(qdrant::lib::detect_primary_language "$(dirname "$file")")
    
    # If directory detection fails, detect from single file
    if [[ "$detected_lang" == "unknown" ]]; then
        # Create temp directory with just this file for detection
        local temp_detect_dir=$(mktemp -d)
        cp "$file" "$temp_detect_dir/"
        detected_lang=$(qdrant::lib::detect_primary_language "$temp_detect_dir")
        rm -rf "$temp_detect_dir"
    fi
    
    # Map internal language names to display names
    local language=""
    case "$detected_lang" in
        javascript) language="JavaScript" ;;
        python) language="Python" ;;
        go) language="Go" ;;
        rust) language="Rust" ;;
        java) language="Java" ;;
        kotlin) language="Kotlin" ;;
        csharp) language="C#" ;;
        ruby) language="Ruby" ;;
        php) language="PHP" ;;
        swift) language="Swift" ;;
        cpp) language="C++" ;;
        shell) language="Shell/Bash" ;;
        sql) language="SQL" ;;
        css) language="CSS" ;;
        matlab) language="MATLAB" ;;
        r) language="R" ;;
        *) 
            # Fallback to extension-based detection
            case "$file_ext" in
                js|jsx) language="JavaScript" ;;
                ts|tsx) language="TypeScript" ;;
                py|ipynb) language="Python" ;;
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
                cs) language="C#" ;;
                sql) language="SQL" ;;
                css|scss|sass|less) language="CSS" ;;
                m|mlx) language="MATLAB" ;;
                R|r|Rmd|rmd) language="R" ;;
                *) language="${file_ext^^}" ;;
            esac
            ;;
    esac
    
    # Extract purpose from header comments (first 10 lines)
    local purpose=$(head -10 "$file" 2>/dev/null | grep -E "^[[:space:]]*(//|#|\*|/\*)" | head -3 | sed 's/^[[:space:]]*[/#*]*//' | tr '\n' ' ' | sed 's/  */ /g' | cut -c1-200)
    
    # Use unified extractor to get deep language-specific analysis
    local extraction_output=""
    local function_count=0
    local class_count=0
    local import_count=0
    local exports=""
    local has_tests="no"
    
    # Create temp directory with just this file for extraction
    local temp_extract_dir=$(mktemp -d)
    cp "$file" "$temp_extract_dir/"
    
    # Extract using unified system (suppress stderr)
    extraction_output=$(qdrant::lib::extract_code "$temp_extract_dir" "code" "file-analysis" "primary" 2>/dev/null || echo "")
    
    if [[ -n "$extraction_output" ]]; then
        # Parse extraction output to get counts and details
        function_count=$(echo "$extraction_output" | jq -s '[.[] | .metadata.functions // [] | length] | add // 0' 2>/dev/null || echo "0")
        class_count=$(echo "$extraction_output" | jq -s '[.[] | .metadata.classes // [] | length] | add // 0' 2>/dev/null || echo "0")
        
        # Get function names for exports (limit to 5)
        local func_names=$(echo "$extraction_output" | jq -r '[.metadata.functions // [] | .[] // empty] | unique | .[0:5] | join(",")' 2>/dev/null || echo "")
        if [[ -n "$func_names" ]]; then
            exports="$func_names"
        fi
        
        # Check for test indicators in metadata
        if echo "$extraction_output" | jq -e '.metadata | select(.is_test == true)' >/dev/null 2>/dev/null; then
            has_tests="yes"
        fi
    else
        # Fallback to basic grep-based counting if extraction fails
        function_count=$(grep -c -E "^[[:space:]]*(function|def |func |const.*=.*=>|export.*function)" "$file" 2>/dev/null | tr -d '\n' || echo "0")
        class_count=$(grep -c -E "^[[:space:]]*(class |interface |struct |type )" "$file" 2>/dev/null | tr -d '\n' || echo "0")
    fi
    
    # Count imports (still use grep as it's simple and effective)
    import_count=$(grep -c -E "^(import |require\(|from |use |include |#include)" "$file" 2>/dev/null | tr -d '\n' || echo "0")
    
    # Clean up temp directory
    rm -rf "$temp_extract_dir"
    
    # Check for test file indicators
    if [[ "$filename" == *test* ]] || [[ "$filename" == *spec* ]] || [[ "$dir" == *test* ]] || [[ "$dir" == *__test__* ]]; then
        has_tests="yes"
    fi
    
    # Build comprehensive content summary
    local content_summary="$language file: $filename in $dir"
    
    if [[ -n "$purpose" ]]; then
        content_summary="$content_summary | Purpose: $purpose"
    fi
    
    if [[ "$has_tests" == "yes" ]]; then
        content_summary="$content_summary | Contains test cases"
    fi
    
    content_summary="$content_summary | $line_count lines"
    
    if [[ $function_count -gt 0 ]]; then
        content_summary="$content_summary | $function_count functions"
    fi
    if [[ $class_count -gt 0 ]]; then
        content_summary="$content_summary | $class_count classes"
    fi
    if [[ $import_count -gt 0 ]]; then
        content_summary="$content_summary | $import_count imports"
    fi
    
    if [[ -n "$exports" ]]; then
        content_summary="$content_summary | Exports: $exports"
    fi
    
    # Determine content category
    local category="source"
    if [[ "$has_tests" == "yes" ]]; then
        category="test"
    elif [[ "$file_ext" == "h" ]] || [[ "$file_ext" == "hpp" ]]; then
        category="header"
    elif [[ "$detected_lang" == "shell" ]]; then
        category="script"
    elif [[ "$filename" == *"config"* ]] || [[ "$filename" == *"setup"* ]]; then
        category="configuration"
    fi
    
    # Output JSON with comprehensive metadata
    jq -n \
        --arg content "$content_summary" \
        --arg file "$file" \
        --arg filename "$filename" \
        --arg directory "$dir" \
        --arg language "$language" \
        --arg detected_lang "$detected_lang" \
        --arg language_ext "$file_ext" \
        --arg size "$file_size" \
        --arg lines "$line_count" \
        --arg functions "$function_count" \
        --arg classes "$class_count" \
        --arg imports "$import_count" \
        --arg exports "$exports" \
        --arg is_test "$has_tests" \
        --arg purpose "$purpose" \
        --arg category "$category" \
        '{
            content: $content,
            metadata: {
                file: $file,
                filename: $filename,
                directory: $directory,
                language: $language,
                detected_language: $detected_lang,
                language_ext: $language_ext,
                size: ($size | tonumber),
                lines: ($lines | tonumber),
                functions: ($functions | tonumber),
                classes: ($classes | tonumber),
                imports: ($imports | tonumber),
                exports: $exports,
                is_test: ($is_test == "yes"),
                purpose: $purpose,
                category: $category,
                content_type: "code_file",
                extraction_method: "unified_code_extractor"
            }
        }' | jq -c
}

# Find code files (excluding common non-code directories)
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
    
    log::info "Extracting content from $total_files code files using unified extractor"
    
    while IFS= read -r file; do
        # Each file gets ONE embedding entry with deep analysis
        if qdrant::extract::code "$file" >> "$output_file"; then
            ((count++))
        fi
        
        # Progress indicator every 50 files
        if [[ $((count % 50)) -eq 0 ]]; then
            log::debug "Processed $count/$total_files code files..."
        fi
    done < <(qdrant::extract::find_code "$directory")
    
    log::success "Extracted content from $count code files with unified analysis"
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
    
    # Use the new batch processing function for massive speedup!
    count=$(qdrant::embedding::process_jsonl_file "$output_file" "code" "$collection" "$app_id")
    
    log::debug "Created $count code embeddings using real batch processing"
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
            extractor: "code_unified"
        }'
}

# Export functions
export -f qdrant::extract::code
export -f qdrant::extract::find_code
export -f qdrant::extract::code_batch
export -f qdrant::embeddings::process_code
export -f qdrant::extract::code_metadata_from_content