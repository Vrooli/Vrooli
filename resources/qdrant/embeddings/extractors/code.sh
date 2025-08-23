#!/usr/bin/env bash
# Code File Content Extractor for Qdrant Embeddings
# Extracts FILE-LEVEL summaries instead of individual functions

# Extract FILE-LEVEL summary for better performance
qdrant::extract::code() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local file_ext="${file##*.}"
    local filename=$(basename "$file")
    local dir=$(dirname "$file")
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
    
    # Build natural language summary (optimized for semantic search)
    echo "This is a $language code file named '$filename' located in $dir."
    
    if [[ -n "$purpose" ]]; then
        echo "Purpose: $purpose"
    fi
    
    if [[ "$is_test" == "yes" ]]; then
        echo "This is a test file containing test cases and specifications."
    fi
    
    echo "The file contains $line_count lines of code with:"
    [[ $function_count -gt 0 ]] && echo "- $function_count functions/methods"
    [[ $class_count -gt 0 ]] && echo "- $class_count classes/interfaces/types"
    [[ $import_count -gt 0 ]] && echo "- $import_count imports/dependencies"
    
    if [[ -n "$exports" ]]; then
        echo "Main exports/API: $exports"
    fi
    
    # Add searchable metadata
    echo "File: $file"
    echo "Language: $language ($file_ext)"
    echo "Size: $file_size bytes"
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
    mkdir -p "$(dirname "$output_file")"
    > "$output_file"
    
    local count=0
    local total_files=$(qdrant::extract::find_code "$directory" | wc -l)
    
    log::info "Extracting content from $total_files code files"
    
    while IFS= read -r file; do
        {
            # Each file gets ONE embedding entry (not one per function)
            qdrant::extract::code "$file"
            echo "---SEPARATOR---"
        } >> "$output_file"
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

# Export functions
export -f qdrant::extract::code
export -f qdrant::extract::find_code
export -f qdrant::extract::code_batch