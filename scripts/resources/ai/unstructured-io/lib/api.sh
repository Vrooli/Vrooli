#!/usr/bin/env bash

# Unstructured.io API Functions
# This file contains functions for interacting with the Unstructured.io API

#######################################
# Process a single document
#######################################
unstructured_io::process_document() {
    local file="$1"
    local strategy="${2:-$UNSTRUCTURED_IO_DEFAULT_STRATEGY}"
    local output_format="${3:-json}"
    local languages="${4:-$UNSTRUCTURED_IO_DEFAULT_LANGUAGES}"
    
    # Validate file
    if ! unstructured_io::validate_file "$file"; then
        return 1
    fi
    
    # Check service is running
    if ! unstructured_io::status "no"; then
        echo "❌ Unstructured.io service is not available"
        return 1
    fi
    
    local filename=$(basename "$file")
    eval "echo \"$MSG_PROCESSING_FILE\""
    
    # Prepare the API request
    local response
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -F "files=@${file}" \
        -F "strategy=${strategy}" \
        -F "languages=${languages}" \
        -F "pdf_infer_table_structure=true" \
        -F "include_page_breaks=${UNSTRUCTURED_IO_INCLUDE_PAGE_BREAKS}" \
        -F "encoding=${UNSTRUCTURED_IO_ENCODING}" \
        --max-time "$UNSTRUCTURED_IO_TIMEOUT_SECONDS" \
        "${UNSTRUCTURED_IO_BASE_URL}${UNSTRUCTURED_IO_PROCESS_ENDPOINT}" 2>/dev/null)
    
    local exit_code=$?
    if [ $exit_code -eq 28 ]; then
        echo "$MSG_PROCESSING_TIMEOUT"
        return 1
    elif [ $exit_code -ne 0 ]; then
        echo "$MSG_PROCESSING_FAILED"
        return 1
    fi
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        echo "$MSG_PROCESSING_SUCCESS"
        
        # Format output based on requested format
        case "$output_format" in
            "json")
                echo "$body" | jq '.'
                ;;
            "markdown")
                unstructured_io::convert_to_markdown "$body"
                ;;
            "text")
                echo "$body" | jq -r '.[] | .text' 2>/dev/null
                ;;
            "elements")
                echo "$body" | jq -r '.[] | "[\(.type)] \(.text)"' 2>/dev/null
                ;;
            *)
                echo "$body"
                ;;
        esac
        
        return 0
    else
        echo "$MSG_PROCESSING_FAILED"
        echo "HTTP Status: $http_code"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        return 1
    fi
}

#######################################
# Convert JSON response to Markdown
#######################################
unstructured_io::convert_to_markdown() {
    local json="$1"
    
    echo "$json" | jq -r '
        .[] | 
        if .type == "Title" then "# \(.text)"
        elif .type == "Header" then "## \(.text)"
        elif .type == "NarrativeText" then "\(.text)\n"
        elif .type == "ListItem" then "- \(.text)"
        elif .type == "Table" then "```\n\(.text)\n```"
        elif .type == "PageBreak" then "\n---\n"
        else .text
        end
    ' 2>/dev/null
}

#######################################
# Process multiple documents in batch
#######################################
unstructured_io::batch_process() {
    local files=("$@")
    local strategy="${UNSTRUCTURED_IO_DEFAULT_STRATEGY}"
    local output_format="json"
    
    # Extract options from files array
    local actual_files=()
    local i=0
    while [ $i -lt ${#files[@]} ]; do
        case "${files[$i]}" in
            --strategy)
                strategy="${files[$((i+1))]}"
                ((i+=2))
                ;;
            --output)
                output_format="${files[$((i+1))]}"
                ((i+=2))
                ;;
            *)
                actual_files+=("${files[$i]}")
                ((i++))
                ;;
        esac
    done
    
    local count=${#actual_files[@]}
    if [ $count -eq 0 ]; then
        echo "❌ No files provided for batch processing"
        return 1
    fi
    
    eval "echo \"$MSG_BATCH_PROCESSING\""
    
    local success_count=0
    local failed_files=()
    
    for file in "${actual_files[@]}"; do
        echo
        echo "Processing: $file"
        echo "─────────────────"
        
        if unstructured_io::process_document "$file" "$strategy" "$output_format"; then
            ((success_count++))
        else
            failed_files+=("$file")
        fi
    done
    
    echo
    echo "Batch Processing Summary"
    echo "======================="
    echo "Total files: $count"
    echo "Successful: $success_count"
    echo "Failed: $((count - success_count))"
    
    if [ ${#failed_files[@]} -gt 0 ]; then
        echo
        echo "Failed files:"
        for file in "${failed_files[@]}"; do
            echo "  - $file"
        done
        return 1
    fi
    
    return 0
}

#######################################
# Test API with a simple text file
#######################################
unstructured_io::test_api() {
    local test_file="/tmp/unstructured_test_$$.txt"
    
    # Create a test file
    cat > "$test_file" << EOF
# Test Document

This is a test document for Unstructured.io API.

## Features
- Document parsing
- Table extraction
- OCR capabilities

This test verifies the API is working correctly.
EOF
    
    echo "🧪 Testing Unstructured.io API..."
    echo
    
    # Process the test file
    if unstructured_io::process_document "$test_file" "fast" "markdown"; then
        rm -f "$test_file"
        echo
        echo "✅ API test passed successfully"
        return 0
    else
        rm -f "$test_file"
        echo
        echo "❌ API test failed"
        return 1
    fi
}

#######################################
# Get supported file types from API
#######################################
unstructured_io::get_supported_types() {
    if ! unstructured_io::status "no"; then
        echo "❌ Service is not available"
        return 1
    fi
    
    echo "📋 Supported File Types"
    echo "====================="
    
    # The Unstructured API doesn't have a direct endpoint for this,
    # so we'll display our configured list
    local count=0
    for format in "${UNSTRUCTURED_IO_SUPPORTED_FORMATS[@]}"; do
        local desc=""
        case "$format" in
            "pdf") desc="Portable Document Format" ;;
            "docx") desc="Microsoft Word" ;;
            "pptx") desc="Microsoft PowerPoint" ;;
            "xlsx") desc="Microsoft Excel" ;;
            "html") desc="Web pages" ;;
            "md") desc="Markdown" ;;
            "txt") desc="Plain text" ;;
            "png"|"jpg"|"jpeg") desc="Images (with OCR)" ;;
            "eml"|"msg") desc="Email messages" ;;
            *) desc="Document format" ;;
        esac
        
        printf "  %-6s - %s\n" ".$format" "$desc"
    done
}