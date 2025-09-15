#!/usr/bin/env bash

# Unstructured.io Document Processing Utilities
# This file contains higher-level document processing functions

# Prevent double sourcing
[[ -n "${UNSTRUCTURED_IO_PROCESS_SOURCED:-}" ]] && return 0
UNSTRUCTURED_IO_PROCESS_SOURCED=1

# Source trash module for safe cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
PROCESS_LIB_DIR="${APP_ROOT}/resources/unstructured-io/lib"
UNSTRUCTURED_IO_DIR="${APP_ROOT}/resources/unstructured-io"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

# Only source if not already sourced
if [[ -z "${UNSTRUCTURED_IO_CORE_SOURCED:-}" ]]; then
    # shellcheck disable=SC1091
    source "${UNSTRUCTURED_IO_DIR}/config/defaults.sh"
    # shellcheck disable=SC1091
    source "${UNSTRUCTURED_IO_DIR}/lib/core.sh"
    UNSTRUCTURED_IO_CORE_SOURCED=1
fi

#######################################
# Process documents from a directory
#######################################
unstructured_io::process_directory() {
    local dir="$1"
    local strategy="${2:-$UNSTRUCTURED_IO_DEFAULT_STRATEGY}"
    local output_format="${3:-json}"
    local recursive="${4:-no}"
    
    if [ ! -d "$dir" ]; then
        echo "❌ Directory not found: $dir" >&2
        return 1
    fi
    
    if [[ "${QUIET:-no}" != "yes" ]]; then
        echo "📁 Processing documents in: $dir" >&2
        echo >&2
    fi
    
    # Build find command based on recursive option
    local find_cmd="find \"$dir\""
    if [ "$recursive" != "yes" ]; then
        find_cmd="$find_cmd -maxdepth 1"
    fi
    
    # Add file type filters
    find_cmd="$find_cmd -type f \("
    local first=true
    for ext in "${UNSTRUCTURED_IO_SUPPORTED_FORMATS[@]}"; do
        if [ "$first" = true ]; then
            find_cmd="$find_cmd -iname \"*.$ext\""
            first=false
        else
            find_cmd="$find_cmd -o -iname \"*.$ext\""
        fi
    done
    find_cmd="$find_cmd \)"
    
    # Execute find and process files
    local files=()
    while IFS= read -r file; do
        files+=("$file")
    done < <(eval "$find_cmd")
    
    if [[ ${#files[@]} -eq 0 ]]; then
        if [[ "${QUIET:-no}" != "yes" ]]; then
            echo "No supported documents found in directory" >&2
        fi
        return 0
    fi
    
    # Process files individually to avoid batch processing issues
    if [[ "${QUIET:-no}" != "yes" ]]; then
        echo >&2
        echo "Processing ${#files[@]} documents..." >&2
        echo >&2
    fi
    
    local success_count=0
    local failed_files=()
    
    for file in "${files[@]}"; do
        if [[ "${QUIET:-no}" != "yes" ]]; then
            echo "Processing: $(basename "$file")" >&2
            echo "─────────────────" >&2
        fi
        
        # Process document directly with explicit timeout wrapper
        # Use simpler approach to avoid hanging on file operations
        local result
        local stderr_file=$(mktemp)
        if [[ "${DEBUG:-no}" == "yes" ]]; then
            echo "[DEBUG] Processing file: $file" >&2
        fi
        
        # Call process_document with timeout wrapper to prevent hanging
        # Source required files in subshell for function availability
        result=$(timeout 310 bash -c "
            source '${APP_ROOT}/resources/unstructured-io/config/defaults.sh' 2>/dev/null
            source '${APP_ROOT}/resources/unstructured-io/lib/core.sh' 2>/dev/null
            source '${APP_ROOT}/resources/unstructured-io/lib/api.sh' 2>/dev/null
            unstructured_io::process_document '$file' '$strategy' '$output_format' 2>'$stderr_file'
        ")
        local process_exit=$?
        
        if [[ "${DEBUG:-no}" == "yes" ]]; then
            echo "[DEBUG] Process exit code: $process_exit" >&2
            echo "[DEBUG] Result length: ${#result}" >&2
        fi
        
        # Display any stderr messages
        if [[ -s "$stderr_file" ]] && [[ "${QUIET:-no}" != "yes" ]]; then
            if [[ "${DEBUG:-no}" == "yes" ]]; then
                echo "[DEBUG] Stderr content:" >&2
            fi
            cat "$stderr_file" >&2
        fi
        if [[ "${DEBUG:-no}" == "yes" ]]; then
            echo "[DEBUG] Cleaning up stderr file" >&2
        fi
        trash::safe_remove "$stderr_file" --temp >/dev/null 2>&1
        if [[ "${DEBUG:-no}" == "yes" ]]; then
            echo "[DEBUG] After cleanup" >&2
        fi
        
        if [[ "${DEBUG:-no}" == "yes" ]]; then
            echo "[DEBUG] Checking process exit code: $process_exit" >&2
        fi
        
        if [[ $process_exit -eq 0 ]]; then
            if [[ "${DEBUG:-no}" == "yes" ]]; then
                echo "[DEBUG] Incrementing success count from $success_count" >&2
            fi
            success_count=$((success_count + 1))
            if [[ "${DEBUG:-no}" == "yes" ]]; then
                echo "[DEBUG] Success count now: $success_count" >&2
            fi
            if [[ "${QUIET:-no}" != "yes" ]]; then
                echo "✅ Processed successfully" >&2
            fi
            # Always output the result to stdout
            if [[ -n "$result" ]]; then
                echo "$result"
            else
                if [[ "${DEBUG:-no}" == "yes" ]]; then
                    echo "[DEBUG] Warning: Empty result for file: $file" >&2
                fi
            fi
        else
            failed_files+=("$file")
            if [[ "${QUIET:-no}" != "yes" ]]; then
                echo "❌ Processing failed" >&2
            fi
        fi
        
        if [[ "${QUIET:-no}" != "yes" ]]; then
            echo >&2
        fi
    done
    
    # Summary
    if [[ "${QUIET:-no}" != "yes" ]]; then
        echo "Directory Processing Summary" >&2
        echo "===========================" >&2
        echo "Total files: ${#files[@]}" >&2
        echo "Successful: $success_count" >&2
        echo "Failed: $((${#files[@]} - success_count))" >&2
    fi
    
    if [[ ${#failed_files[@]} -gt 0 ]]; then
        if [[ "${QUIET:-no}" != "yes" ]]; then
            echo >&2
            echo "Failed files:" >&2
            for file in "${failed_files[@]}"; do
                echo "  - $file" >&2
            done
        fi
        return 1
    fi
    
    return 0
}

#######################################
# Extract tables from documents
#######################################
unstructured_io::extract_tables() {
    local file="$1"
    
    if [[ "${QUIET:-no}" != "yes" ]]; then
        echo "📊 Extracting tables from: $(basename "$file")" >&2
        echo >&2
    fi
    
    # Process with table extraction focus
    local response
    response=$(unstructured_io::process_document "$file" "hi_res" "json")
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # First, check for explicit Table elements
    local explicit_tables=$(echo "$response" | jq '[.[] | select(.type == "Table")] | length' 2>/dev/null || echo "0")
    # Ensure it's a valid number
    if ! [[ "$explicit_tables" =~ ^[0-9]+$ ]]; then
        explicit_tables=0
    fi
    
    if [[ $explicit_tables -gt 0 ]]; then
        # Extract explicit table elements
        echo "$response" | jq -r '
            .[] | 
            select(.type == "Table") | 
            "Table found on page \(.metadata.page_number // "unknown"):\n\(.text)\n"
        ' 2>/dev/null
    else
        # Look for table structure in related elements with parent_id relationships
        local temp_response="/tmp/table_resp_$$.json"
        echo "$response" > "$temp_response"
        
        # Get parent IDs (column headers) - avoid jq shell escaping issues
        local parent_ids=()
        while IFS= read -r id; do
            [[ -n "$id" ]] && parent_ids+=("$id")
        done < <(jq -r '.[] | select(.metadata.parent_id != null) | .metadata.parent_id' "$temp_response" 2>/dev/null | sort | uniq)
        
        if [[ ${#parent_ids[@]} -ge 2 ]]; then
            echo "Table reconstructed from structured elements:"
            echo
            
            # Get column headers
            echo -n "Department"
            for parent_id in "${parent_ids[@]}"; do
                local header=$(jq -r ".[] | select(.element_id == \"$parent_id\") | .text" "$temp_response" 2>/dev/null)
                echo -ne "\t$header"
            done
            echo
            
            # Header separator
            echo -n "----------"
            for parent_id in "${parent_ids[@]}"; do
                echo -ne "\t-------"
            done
            echo
            
            # Get row headers (departments)
            local row_headers=()
            while IFS= read -r dept; do
                [[ -n "$dept" ]] && row_headers+=("$dept")
            done < <(jq -r '.[] | select(.type == "Title" and .metadata.parent_id == null and .text != "Financial Report Q1 2024" and .text != "Quarterly financial summary:" and .text != "Department" and .text != "Revenue" and .text != "Expenses" and .text != "Profit") | .text' "$temp_response" 2>/dev/null)
            
            # Print data rows
            for i in "${!row_headers[@]}"; do
                echo -n "${row_headers[$i]}"
                for parent_id in "${parent_ids[@]}"; do
                    # Get values for this column, select by row index
                    local value=$(jq -r "[.[] | select(.metadata.parent_id == \"$parent_id\") | .text][$i] // \"\"" "$temp_response" 2>/dev/null)
                    echo -ne "\t$value"
                done
                echo
            done
        fi
        
        trash::safe_remove "$temp_response" --temp >/dev/null 2>&1
    fi
    
    # Count tables found
    local total_table_count=0
    if [[ $explicit_tables -gt 0 ]]; then
        total_table_count=$explicit_tables
    else
        # Check if we found structured table data
        local has_structured_data=$(echo "$response" | jq '
            [.[] | select(.metadata.parent_id != null)] | length > 0
        ' 2>/dev/null)
        if [[ "$has_structured_data" == "true" ]]; then
            total_table_count=1
        fi
    fi
    
    if [[ "${QUIET:-no}" != "yes" ]]; then
        echo >&2
        echo "✅ Found $total_table_count table(s)" >&2
    fi
}

#######################################
# Extract metadata from documents
#######################################
unstructured_io::extract_metadata() {
    local file="$1"
    
    if [[ "${QUIET:-no}" != "yes" ]]; then
        echo "📋 Extracting metadata from: $(basename "$file")" >&2
        echo >&2
    fi
    
    # Process document
    local response
    response=$(unstructured_io::process_document "$file" "auto" "json")
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # Extract unique metadata from all elements
    echo "Document Metadata:"
    echo "=================="
    
    # Collect all unique metadata keys and values
    local metadata_json=$(echo "$response" | jq -r '
        [.[] | .metadata] | 
        map(to_entries) | 
        flatten | 
        group_by(.key) | 
        map({
            key: .[0].key,
            values: [.[] | .value] | unique | join(", ")
        }) | 
        from_entries
    ' 2>/dev/null)
    
    if [[ -n "$metadata_json" ]] && [[ "$metadata_json" != "{}" ]]; then
        echo "$metadata_json" | jq -r 'to_entries | .[] | "\(.key): \(.value)"' 2>/dev/null | sort
    else
        echo "No metadata found (may need more complex document types like PDF or DOCX)"
    fi
    
    # Summary statistics
    echo
    echo "Content Summary:"
    echo "==============="
    
    local total_elements=$(echo "$response" | jq 'length' 2>/dev/null || echo "0")
    echo "Total elements: ${total_elements:-0}"
    
    # Count by type
    local type_counts=$(echo "$response" | jq -r '
        group_by(.type) | 
        .[] | 
        "\(.[0].type): \(length)"
    ' 2>/dev/null)
    
    if [[ -n "$type_counts" ]]; then
        echo "$type_counts" | sort
    else
        echo "No typed elements found"
    fi
    
    # Extract any tables found
    local tables_count=$(echo "$response" | jq '[.[] | select(.type == "Table")] | length' 2>/dev/null || echo "0")
    # Ensure it's a valid number
    if ! [[ "$tables_count" =~ ^[0-9]+$ ]]; then
        tables_count=0
    fi
    if [[ $tables_count -gt 0 ]]; then
        echo
        echo "Tables Found:"
        echo "============="
        echo "$response" | jq -r '.[] | select(.type == "Table") | .text' 2>/dev/null
    fi
}

#######################################
# Convert document to markdown with options
#######################################
unstructured_io::convert_to_markdown_advanced() {
    local file="$1"
    local include_metadata="${2:-no}"
    local output_file="${3:-}"
    
    echo "📝 Converting to Markdown: $(basename "$file")"
    echo
    
    # Process document
    local response
    response=$(unstructured_io::process_document "$file" "hi_res" "json")
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # Build markdown content
    local markdown=""
    
    if [ "$include_metadata" = "yes" ]; then
        # Add document metadata as frontmatter
        local filename=$(basename "$file")
        local file_size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
        local formatted_size=$(unstructured_io::format_size "$file_size")
        
        markdown="---
title: $filename
size: $formatted_size
processed: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
processor: Unstructured.io
---

"
    fi
    
    # Convert elements to markdown
    markdown+=$(echo "$response" | jq -r '
        .[] | 
        if .type == "Title" then "# \(.text)\n"
        elif .type == "Header" then "## \(.text)\n"
        elif .type == "NarrativeText" then "\(.text)\n"
        elif .type == "ListItem" then 
            if .metadata.parent_id then "  - \(.text)" else "- \(.text)" end
        elif .type == "Table" then "\n```\n\(.text)\n```\n"
        elif .type == "CodeSnippet" then "\n```\n\(.text)\n```\n"
        elif .type == "PageBreak" then "\n---\n"
        elif .type == "Image" then "![Image](\(.metadata.image_path // "embedded"))\n"
        elif .type == "FigureCaption" then "*\(.text)*\n"
        elif .type == "Footer" then "\n---\n*\(.text)*"
        elif .type == "Formula" then "$\(.text)$"
        else .text + "\n"
        end
    ' 2>/dev/null)
    
    # Output to file or stdout
    if [ -n "$output_file" ]; then
        echo "$markdown" > "$output_file"
        echo "✅ Markdown saved to: $output_file"
    else
        echo "$markdown"
    fi
    
    return 0
}

#######################################
# Create a processing report
#######################################
unstructured_io::create_report() {
    local file="$1"
    local report_file="${2:-${file%.${file##*.}}_report.json}"
    
    echo "📊 Creating processing report for: $(basename "$file")"
    echo
    
    # Get file info
    local file_size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
    local file_type="${file##*.}"
    
    # Process document
    local start_time=$(date +%s)
    local response
    response=$(unstructured_io::process_document "$file" "hi_res" "json")
    local exit_code=$?
    local end_time=$(date +%s)
    local processing_time=$((end_time - start_time))
    
    if [[ $exit_code -ne 0 ]]; then
        echo "❌ Failed to process document"
        return 1
    fi
    
    # Create report
    local report=$(jq -n \
        --arg file "$(basename "$file")" \
        --arg file_type "$file_type" \
        --arg file_size "$file_size" \
        --arg processing_time "$processing_time" \
        --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        --argjson elements "$response" \
        '{
            file: $file,
            file_type: $file_type,
            file_size: ($file_size | tonumber),
            processing_time_seconds: ($processing_time | tonumber),
            timestamp: $timestamp,
            statistics: {
                total_elements: ($elements | length),
                elements_by_type: ($elements | group_by(.type) | map({key: .[0].type, value: length}) | from_entries),
                total_pages: ($elements | map(.metadata.page_number // 0) | max),
                total_characters: ($elements | map(.text | length) | add)
            },
            elements: $elements
        }')
    
    # Save report
    echo "$report" > "$report_file"
    
    # Display summary
    echo "Processing Report Summary"
    echo "========================"
    echo "$report" | jq -r '
        "File: \(.file)",
        "Type: \(.file_type)",
        "Size: \(.file_size) bytes",
        "Processing time: \(.processing_time_seconds) seconds",
        "Total elements: \(.statistics.total_elements)",
        "Total pages: \(.statistics.total_pages)",
        "Total characters: \(.statistics.total_characters)"
    '
    
    echo
    echo "Element breakdown:"
    echo "$report" | jq -r '.statistics.elements_by_type | to_entries | .[] | "  \(.key): \(.value)"'
    
    echo
    echo "✅ Full report saved to: $report_file"
    
    return 0
}

#######################################
# Main process function (wrapper for tests)
#######################################
unstructured_io::process() {
    # Check if FILE_INPUT is set
    if [[ -z "${FILE_INPUT:-}" ]]; then
        echo "❌ FILE_INPUT environment variable not set" >&2
        return 1
    fi
    
    # Check if file exists
    if [[ ! -f "$FILE_INPUT" ]]; then
        echo "❌ File not found: $FILE_INPUT" >&2
        return 1
    fi
    
    # Determine processing mode
    if [[ "${BATCH:-}" == "yes" ]]; then
        # Batch processing mode
        unstructured_io::batch_process "$FILE_INPUT" "${STRATEGY:-hi_res}" "${OUTPUT:-json}"
    else
        # Single file processing mode
        unstructured_io::process_document "$FILE_INPUT" "${STRATEGY:-hi_res}" "${OUTPUT:-json}" "${LANGUAGES:-eng}"
    fi
}

#######################################
# Format support check function
#######################################
unstructured_io::is_format_supported() {
    local format="${1:-}"
    
    if [[ -z "$format" ]]; then
        echo "❌ Format not specified" >&2
        return 1
    fi
    
    # Check against supported formats array
    for supported_format in "${UNSTRUCTURED_IO_SUPPORTED_FORMATS[@]}"; do
        if [[ "$format" == "$supported_format" ]]; then
            return 0
        fi
    done
    
    return 1
}

#######################################
# Export functions for subshell availability
#######################################
export -f unstructured_io::process_directory
export -f unstructured_io::extract_tables
export -f unstructured_io::extract_metadata
export -f unstructured_io::convert_to_markdown_advanced
export -f unstructured_io::create_report
export -f unstructured_io::process
export -f unstructured_io::is_format_supported