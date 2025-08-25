#!/usr/bin/env bash
# Path Reference Validation Component - Docs Phase
# 
# Detects broken file path references in documentation, including:
# - Plain text file paths in code blocks
# - File paths mentioned in regular text  
# - Relative and absolute path references
#
# This catches broken file references that standard link checkers miss

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test"
PROJECT_ROOT="${PROJECT_ROOT:-$APP_ROOT}"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"
# shellcheck disable=SC1091  
source "$SCRIPT_DIR/shared/test-helpers.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/cache.bash"

# Check file paths referenced in a document
run_path_reference_validation() {
    local file="$1"
    local relative_path
    relative_path=$(relative_path "$file")
    
    log_test_start "path-references" "$relative_path"
    
    if is_dry_run; then
        log_info "[DRY RUN] Would check file path references: $relative_path"
        return 0
    fi
    
    echo "DEBUG: After dry run check" >&2
    
    local broken_paths=()
    local total_paths=0
    local line_num=0
    local in_code_block="false"
    local code_block_type=""
    
    echo "DEBUG: About to open file: $file" >&2
    
    # Extract potential file paths from the document
    # Use file descriptor 3 to avoid stdin conflicts
    exec 3< "$file"
    echo "DEBUG: File opened, starting read loop" >&2
    while IFS= read -r line <&3; do
        ((line_num++))
        echo "DEBUG: Line $line_num: [$line]" >&2
        
        # Check for code block boundaries
        if [[ "$line" =~ ^'```' ]]; then
            if [[ "$in_code_block" == "false" ]]; then
                in_code_block="true"
                # Extract code block type (e.g., mermaid, bash, etc.)
                code_block_type=$(echo "$line" | sed 's/^```//' | awk '{print $1}')
            else
                in_code_block="false"
                code_block_type=""
            fi
            continue
        fi
        
        # Skip lines inside mermaid/plantuml/graphviz diagrams ONLY
        # We still want to check paths in bash, javascript, etc. code blocks
        if [[ "$in_code_block" == "true" ]] && [[ "$code_block_type" =~ ^(mermaid|plantuml|graphviz|dot|svg)$ ]]; then
            continue
        fi
        
        # Skip empty lines
        [[ -z "$line" ]] && continue
        
        # Extract file paths using multiple patterns
        local paths=""
        
        # Pattern 1: ALL absolute paths starting with /
        # This will catch /home/user/..., /tmp/..., /scripts/..., etc.
        paths+=$(echo "$line" | grep -oE '/[a-zA-Z0-9_/.-]+(\.[a-zA-Z0-9]+)?' || true)$'\n'
        
        # Pattern 2: Relative paths starting with ./
        paths+=$(echo "$line" | grep -oE '\./[a-zA-Z0-9_./-]+(\.[a-zA-Z0-9]+)?' || true)$'\n'
        
        # Pattern 3: File paths with common extensions (no leading /)
        paths+=$(echo "$line" | grep -oE '[a-zA-Z0-9_-]+(/[a-zA-Z0-9_.-]+)*\.(sh|py|js|ts|tsx|jsx|json|yaml|yml|md|sql|txt|toml|conf|cfg|ini|log)' || true)$'\n'
        
        # Process paths one by one using a for loop to avoid stdin issues
        # Convert newline-separated paths to an array
        local path_array=()
        if [[ -n "$paths" ]]; then
            while IFS= read -r path; do
                [[ -n "$path" ]] && path_array+=("$path")
            done <<< "$paths"
        fi
        
        # Check each potential path
        for path in "${path_array[@]}"; do
            [[ -z "$path" ]] && continue
            
            # Skip if path doesn't contain any letters (filters out things like "80/443")
            if ! [[ "$path" =~ [a-zA-Z] ]]; then
                continue
            fi
            
            # Skip certain patterns that aren't actually file paths
            [[ "$path" =~ ^https?:// ]] && continue
            [[ "$path" =~ ^mailto: ]] && continue
            [[ "$path" =~ ^[0-9]+\.[0-9]+\.[0-9]+ ]] && continue  # Skip version numbers
            [[ "$path" =~ \$\{ ]] && continue  # Skip shell variables like ${APP_ROOT}
            [[ "$path" =~ ^example\. ]] && continue  # Skip example.com, example.json etc.
            [[ "$path" =~ localhost: ]] && continue  # Skip localhost URLs
            
            ((total_paths++))
            
            # Handle different types of paths
            local check_path="$path"
            local path_status="valid"
            local error_reason=""
            
            if [[ "$path" == /* ]]; then
                # Absolute paths - handle specially
                
                # System paths that are OK (common temporary/system locations)
                if [[ "$path" =~ ^/(tmp|var/tmp|dev|proc|sys|etc|usr/bin|usr/local/bin)/ ]]; then
                    # These are system paths that are expected to exist on most systems
                    # We'll skip validation for these as they're not project-specific
                    is_verbose && log_info "    ℹ️  Line $line_num: $path (system path, skipped)"
                    continue
                else
                    # ALL other absolute paths are errors (including /scripts/..., /resources/...)
                    # These should be written as relative paths without the leading /
                    path_status="invalid"
                    error_reason="absolute path - should be relative (remove leading /)"
                    broken_paths+=("$path - ABSOLUTE PATH ERROR (line $line_num)")
                    is_verbose && log_warning "    ❌ Line $line_num: $path - absolute path should be relative (remove leading /)"
                fi
                
            elif [[ "$path" == ./* ]]; then
                # Relative paths starting with ./ - relative to document directory
                local doc_dir
                doc_dir=$(dirname "$file")
                check_path="${doc_dir}/${path#./}"
                if [[ ! -f "$check_path" && ! -d "$check_path" ]]; then
                    path_status="invalid"
                    error_reason="not found relative to document"
                fi
                
            else
                # Plain paths without prefix - try multiple interpretations
                # First try as relative to project root
                check_path="${PROJECT_ROOT}/$path"
                
                if [[ ! -f "$check_path" && ! -d "$check_path" ]]; then
                    # Try relative to document directory
                    local doc_dir
                    doc_dir=$(dirname "$file")
                    local alt_path="${doc_dir}/$path"
                    if [[ -f "$alt_path" || -d "$alt_path" ]]; then
                        check_path="$alt_path"
                        path_status="valid"
                    else
                        path_status="invalid"
                        error_reason="not found (tried project root and document-relative)"
                    fi
                fi
            fi
            
            # Report results based on path status
            if [[ "$path_status" == "invalid" ]]; then
                if [[ -n "$error_reason" ]]; then
                    broken_paths+=("$path - $error_reason (line $line_num)")
                    is_verbose && log_warning "    ❌ Line $line_num: $path - $error_reason"
                else
                    broken_paths+=("$path (line $line_num)")
                    is_verbose && log_warning "    ❌ Line $line_num: $path"
                fi
            elif [[ "$path_status" == "valid" ]]; then
                is_verbose && log_info "    ✅ Line $line_num: $path"
            fi
            
        done
        
    done
    exec 3<&-
    
    # Report results
    local broken_count=${#broken_paths[@]}
    if [[ $broken_count -eq 0 ]]; then
        log_test_pass "path-references" "$relative_path"
        update_cache "$file" "path-references" "passed" "$total_paths" "" 0
        return 0
    else
        log_test_fail "path-references" "$relative_path"
        
        # Show broken paths (limit to first 5)
        local show_count=$((broken_count > 5 ? 5 : broken_count))
        for ((i=0; i<show_count; i++)); do
            log_warning "    └─ ${broken_paths[i]}"
        done
        
        if [[ $broken_count -gt 5 ]]; then
            log_warning "    └─ ... and $((broken_count - 5)) more broken paths"
        fi
        
        update_cache "$file" "path-references" "failed" "$total_paths" "$(printf '%s\n' "${broken_paths[@]}")" "$broken_count"
        return 1
    fi
}

# If this script is run directly, provide basic functionality
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        echo "Usage: $0 <file.md> [file2.md ...]"
        exit 1
    fi
    
    for file in "$@"; do
        if [[ -f "$file" ]]; then
            run_path_reference_validation "$file"
        else
            echo "File not found: $file"
        fi
    done
fi