#!/usr/bin/env bash

# Simple code extractor for basic functionality

# Extract basic code information (simplified version)
qdrant::extract::code() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local file_ext="${file##*.}"
    local filename=$(basename "$file")
    local dir=$(dirname "$file")
    
    echo "Code file: $filename"
    echo "Path: $file"
    echo "Language: $file_ext"
    
    # Extract key code features
    local functions=$(grep -E "^[[:space:]]*(function|def|func|const.*=.*=>|export.*function)" "$file" 2>/dev/null | head -5 | sed 's/^[[:space:]]*//' | tr '\n' ' ')
    local classes=$(grep -E "^[[:space:]]*(class|interface|struct|type)" "$file" 2>/dev/null | head -3 | sed 's/^[[:space:]]*//' | tr '\n' ' ')
    local imports=$(grep -E "^(import|require|from|use)" "$file" 2>/dev/null | head -3 | sed 's/^[[:space:]]*//' | tr '\n' ' ')
    
    [[ -n "$functions" ]] && echo "Functions: $functions"
    [[ -n "$classes" ]] && echo "Classes/Types: $classes"
    [[ -n "$imports" ]] && echo "Imports: $imports"
    
    # Extract first comment block as description
    local description=$(head -20 "$file" | grep -E "^[[:space:]]*(//|#|\*|/\*)" | head -3 | sed 's/^[[:space:]]*//' | tr '\n' ' ')
    [[ -n "$description" ]] && echo "Description: $description"
    
    echo "---CODE---"
}

# Find code files (simplified)
qdrant::extract::find_code() {
    local directory="$1"
    
    find "$directory" -type f \( \
        -name "*.js" -o -name "*.ts" -o -name "*.jsx" -o -name "*.tsx" -o \
        -name "*.py" -o -name "*.go" -o -name "*.sh" -o -name "*.bash" -o \
        -name "*.java" -o -name "*.cpp" -o -name "*.c" -o -name "*.h" \
    \) 2>/dev/null
}

# Process code files in batch (simplified)
qdrant::extract::code_batch() {
    local directory="$1" 
    local output_file="$2"
    
    if [[ -z "$directory" || -z "$output_file" ]]; then
        log::error "Usage: qdrant::extract::code_batch <directory> <output_file>"
        return 1
    fi
    
    # Clear output file
    > "$output_file"
    
    local count=0
    while IFS= read -r file; do
        {
            echo "=== Code File ==="
            qdrant::extract::code "$file"
            echo ""
        } >> "$output_file"
        ((count++))
    done < <(qdrant::extract::find_code "$directory")
    
    log::info "Processed $count code files"
    return 0
}