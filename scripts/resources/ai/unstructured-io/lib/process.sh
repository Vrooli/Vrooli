#!/usr/bin/env bash

# Unstructured.io Document Processing Utilities
# This file contains higher-level document processing functions

#######################################
# Process documents from a directory
#######################################
unstructured_io::process_directory() {
    local dir="$1"
    local strategy="${2:-$UNSTRUCTURED_IO_DEFAULT_STRATEGY}"
    local output_format="${3:-json}"
    local recursive="${4:-no}"
    
    if [ ! -d "$dir" ]; then
        echo "❌ Directory not found: $dir"
        return 1
    fi
    
    echo "📁 Processing documents in: $dir"
    echo
    
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
    
    if [ ${#files[@]} -eq 0 ]; then
        echo "No supported documents found in directory"
        return 0
    fi
    
    # Process files in batch
    unstructured_io::batch_process "${files[@]}" --strategy "$strategy" --output "$output_format"
}

#######################################
# Extract tables from documents
#######################################
unstructured_io::extract_tables() {
    local file="$1"
    
    echo "📊 Extracting tables from: $(basename "$file")"
    echo
    
    # Process with table extraction focus
    local response
    response=$(unstructured_io::process_document "$file" "hi_res" "json")
    
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # Extract only table elements
    echo "$response" | jq -r '
        .[] | 
        select(.type == "Table") | 
        "Table found on page \(.metadata.page_number // "unknown"):\n\(.text)\n"
    ' 2>/dev/null
    
    local table_count=$(echo "$response" | jq '[.[] | select(.type == "Table")] | length' 2>/dev/null)
    echo
    echo "✅ Found $table_count table(s)"
}

#######################################
# Extract metadata from documents
#######################################
unstructured_io::extract_metadata() {
    local file="$1"
    
    echo "📋 Extracting metadata from: $(basename "$file")"
    echo
    
    # Process document
    local response
    response=$(unstructured_io::process_document "$file" "auto" "json")
    
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # Extract unique metadata
    echo "Document Metadata:"
    echo "=================="
    
    # Get first element with metadata
    echo "$response" | jq -r '
        .[0].metadata | 
        to_entries | 
        .[] | 
        "\(.key): \(.value)"
    ' 2>/dev/null | sort | uniq
    
    # Summary statistics
    echo
    echo "Content Summary:"
    echo "==============="
    
    local total_elements=$(echo "$response" | jq 'length' 2>/dev/null)
    echo "Total elements: $total_elements"
    
    # Count by type
    echo "$response" | jq -r '
        group_by(.type) | 
        .[] | 
        "\(.[0].type): \(length)"
    ' 2>/dev/null | sort
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
    
    if [ $? -ne 0 ]; then
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
    
    if [ $exit_code -ne 0 ]; then
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
                elements_by_type: ($elements | group_by(.type) | map({type: .[0].type, count: length}) | from_entries),
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