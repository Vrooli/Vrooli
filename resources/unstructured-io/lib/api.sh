#!/usr/bin/env bash

# Unstructured.io API Functions
# This file contains functions for interacting with the Unstructured.io API

# Source trash module for safe cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
API_LIB_DIR="${APP_ROOT}/resources/unstructured-io/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

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
        echo "❌ Unstructured.io service is not available" >&2
        return 1
    fi
    
    # Check cache first (unless caching is disabled)
    if [[ "${UNSTRUCTURED_IO_CACHE_ENABLED:-yes}" == "yes" ]]; then
        local cache_key=$(unstructured_io::get_cache_key "$file" "$strategy" "$output_format" "$languages")
        local cached_result=$(unstructured_io::get_cached "$cache_key")
        
        if [[ -n "$cached_result" ]]; then
            if [[ "${QUIET:-no}" != "yes" ]]; then
                echo "📦 Using cached result for: $(basename "$file")" >&2
            fi
            echo "$cached_result"
            return 0
        fi
    fi
    
    local filename=$(basename "$file")
    if [[ "${QUIET:-no}" != "yes" ]]; then
        eval "echo \"$MSG_PROCESSING_FILE\"" >&2
    fi
    
    # Prepare the API request
    local response
    local curl_stderr
    curl_stderr=$(mktemp)
    
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -F "files=@${file}" \
        -F "strategy=${strategy}" \
        -F "languages=${languages}" \
        -F "pdf_infer_table_structure=true" \
        -F "skip_infer_table_types=[]" \
        -F "include_page_breaks=${INCLUDE_PAGE_BREAKS:-$UNSTRUCTURED_IO_INCLUDE_PAGE_BREAKS}" \
        -F "encoding=${UNSTRUCTURED_IO_ENCODING}" \
        -F "coordinates=false" \
        -F "xml_keep_tags=false" \
        -F "combine_under_n_chars=${COMBINE_UNDER_CHARS:-0}" \
        -F "new_after_n_chars=${NEW_AFTER_CHARS:-500}" \
        -F "max_characters=${CHUNK_CHARS:-1500}" \
        --max-time "$UNSTRUCTURED_IO_TIMEOUT_SECONDS" \
        "${UNSTRUCTURED_IO_BASE_URL}${UNSTRUCTURED_IO_PROCESS_ENDPOINT}" 2>"$curl_stderr")
    
    local exit_code=$?
    local curl_error=$(cat "$curl_stderr" 2>/dev/null)
    trash::safe_remove "$curl_stderr" --temp
    
    # Enhanced error handling with specific codes
    if [ $exit_code -ne 0 ]; then
        case $exit_code in
            6)
                echo "[ERROR:NETWORK] Could not resolve host" >&2
                ;;
            7)
                echo "[ERROR:CONNECTION] Failed to connect to Unstructured.io service at ${UNSTRUCTURED_IO_BASE_URL}" >&2
                echo "       Check if the service is running: ./manage.sh --action status" >&2
                ;;
            28)
                echo "[ERROR:TIMEOUT] $MSG_PROCESSING_TIMEOUT" >&2
                echo "       Consider using --strategy fast for quicker processing" >&2
                ;;
            52)
                echo "[ERROR:EMPTY_RESPONSE] Server returned empty response" >&2
                ;;
            *)
                echo "[ERROR:CURL_$exit_code] $MSG_PROCESSING_FAILED" >&2
                if [[ -n "$curl_error" ]]; then
                    echo "       Details: $curl_error" >&2
                fi
                ;;
        esac
        return 1
    fi
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        if [[ "${QUIET:-no}" != "yes" ]]; then
            echo "$MSG_PROCESSING_SUCCESS" >&2
        fi
        
        # Format output based on requested format
        local formatted_output=""
        case "$output_format" in
            "json")
                formatted_output=$(echo "$body" | jq '.')
                ;;
            "markdown")
                formatted_output=$(unstructured_io::convert_to_markdown "$body")
                ;;
            "text")
                formatted_output=$(echo "$body" | jq -r '.[] | .text' 2>/dev/null)
                ;;
            "elements")
                formatted_output=$(echo "$body" | jq -r '.[] | "[\(.type)] \(.text)"' 2>/dev/null)
                ;;
            *)
                formatted_output="$body"
                ;;
        esac
        
        # Cache the result if caching is enabled
        if [[ "${UNSTRUCTURED_IO_CACHE_ENABLED:-yes}" == "yes" ]] && [[ -n "$cache_key" ]]; then
            unstructured_io::cache_result "$cache_key" "$formatted_output"
        fi
        
        # Output the result
        echo "$formatted_output"
        
        return 0
    else
        # Provide specific error messages based on HTTP status code
        case "$http_code" in
            "400")
                echo "[ERROR:BAD_REQUEST] Invalid request format" >&2
                echo "       Check file path and parameters" >&2
                ;;
            "401")
                echo "[ERROR:UNAUTHORIZED] Authentication required" >&2
                ;;
            "403")
                echo "[ERROR:FORBIDDEN] Access denied to resource" >&2
                ;;
            "404")
                echo "[ERROR:NOT_FOUND] API endpoint not found" >&2
                echo "       Service may be outdated or misconfigured" >&2
                ;;
            "413")
                echo "[ERROR:FILE_TOO_LARGE] $MSG_ERROR_HTTP_413" >&2
                local file_size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
                echo "       File size: $(echo $file_size | numfmt --to=iec-i --suffix=B 2>/dev/null || echo "$file_size bytes")" >&2
                echo "       Try splitting the file or using --strategy fast" >&2
                ;;
            "415")
                echo "[ERROR:UNSUPPORTED_TYPE] $MSG_ERROR_HTTP_415" >&2
                echo "       File type: $(file -b "$file" | head -1)" >&2
                echo "       Supported formats: ${UNSTRUCTURED_IO_SUPPORTED_FORMATS[*]}" >&2
                ;;
            "422")
                echo "[ERROR:INVALID_PARAMS] $MSG_ERROR_HTTP_422" >&2
                echo "       Strategy: $strategy" >&2
                echo "       Languages: $languages" >&2
                echo "       Valid strategies: fast, hi_res, auto" >&2
                ;;
            "429")
                echo "[ERROR:RATE_LIMIT] Too many requests" >&2
                echo "       Please wait before retrying" >&2
                ;;
            "500")
                echo "[ERROR:SERVER_ERROR] $MSG_ERROR_HTTP_500" >&2
                echo "       The document may contain complex formatting that caused processing to fail" >&2
                echo "       Try using --strategy fast for simpler processing" >&2
                ;;
            "502")
                echo "[ERROR:BAD_GATEWAY] Service temporarily unavailable" >&2
                ;;
            "503")
                echo "[ERROR:SERVICE_UNAVAILABLE] Service is overloaded or down" >&2
                echo "       Check service status: ./manage.sh --action status" >&2
                ;;
            "504")
                echo "[ERROR:GATEWAY_TIMEOUT] Processing took too long" >&2
                echo "       Try using --strategy fast or processing smaller files" >&2
                ;;
            *)
                echo "[ERROR:HTTP_$http_code] Request failed with status: $http_code" >&2
                ;;
        esac
        
        # Show detailed error if available
        if [ -n "$body" ]; then
            local error_detail=$(echo "$body" | jq -r '.detail // .message // .error // empty' 2>/dev/null)
            if [[ -n "$error_detail" ]]; then
                echo "       Server response: $error_detail" >&2
            elif [[ ${#body} -lt 500 ]]; then
                echo "       Raw response: $body" >&2
            fi
        fi
        return 1
    fi
}

#######################################
# Convert JSON response to Markdown
#######################################
unstructured_io::convert_to_markdown() {
    local json="$1"
    
    # Process elements and maintain proper formatting
    echo "$json" | jq -r '
        # Helper function to detect code language
        def detect_code_language:
            if test("^\\s*(def|class|import|from)\\s+"; "m") then "python"
            elif test("^\\s*(function|const|let|var|=>|require|import)"; "m") then "javascript"
            elif test("^\\s*(#include|int main|void|typedef|struct)"; "m") then "c"
            elif test("^\\s*(public class|private|protected|package)"; "m") then "java"
            elif test("^\\s*(#!/bin/bash|#!/bin/sh|^[A-Za-z_]+\\(\\))"; "m") then "bash"
            else ""
            end;
        
        # Helper to extract depth level for headers
        def header_level:
            if .metadata.category_depth then
                "#" * ((.metadata.category_depth // 0) + 1) + " "
            elif .type == "Title" then "# "
            elif .type == "Header" then "## "
            else "### "
            end;
        
        # Process each element
        .[] | 
        if .type == "Title" then 
            "\n" + header_level + (.text | gsub("^#+\\s*"; "")) + "\n"
        elif .type == "Header" then 
            "\n" + header_level + (.text | gsub("^#+\\s*"; "")) + "\n"
        elif .type == "NarrativeText" or .type == "UncategorizedText" then
            # Check if this is a flattened list 
            if (.text | contains(" - Item")) then
                # Simple approach: just ensure each list item is on its own line
                .text | 
                split(" - ") | 
                map(select(length > 0)) | 
                map(
                  if test("Nested") then
                    "  - " + .
                  else
                    "- " + .
                  end
                ) | 
                join("\n") | 
                # Remove the leading dash from the first part if it exists
                sub("^- Here[^\\n]*\\n"; "") + "\n"
            # Check if this looks like code
            elif (.text | test("python |javascript |bash |java |def |function |class ")) then
                # Extract language if specified
                if (.text | test("^(python|javascript|bash|java|c|cpp) ")) then
                    .text as $code |
                    ($code | split(" ")[0]) as $lang |
                    "\n```" + $lang + "\n" + 
                    ($code | sub("^[a-z]+ "; "") | 
                     # Simple replacements for common Python patterns
                     gsub(" def "; "\ndef ") |
                     gsub("\\): "; "):\n    ") |
                     gsub(" return "; "\n    return ")
                    ) + 
                    "\n```\n"
                else
                    "\n```\n" + .text + "\n```\n"
                end
            else
                .text + "\n"
            end
        elif .type == "ListItem" then 
            # Handle nested lists with proper indentation
            (if .metadata.category_depth then
                "  " * ((.metadata.category_depth // 1) - 1)
            else
                ""
            end) + "- " + .text
        elif .type == "Table" then 
            if .metadata.text_as_html then
                # Parse HTML table and convert to markdown
                "\n" + (.metadata.text_as_html | 
                 # Clean up the HTML
                 gsub("</?table[^>]*>"; "") |
                 gsub("</?tbody[^>]*>"; "") |
                 gsub("</?thead[^>]*>"; "") |
                 # Process rows
                 gsub("</tr>"; "</tr>\n") |
                 split("\n") |
                 map(select(test("<tr"))) |
                 map(
                   # Extract cells
                   gsub("<tr[^>]*>"; "") |
                   gsub("</tr>"; "") |
                   split("</td>") |
                   map(
                     gsub("<t[dh][^>]*>"; "") |
                     gsub("^\\s+|\\s+$"; "")
                   ) |
                   # Remove empty last element
                   if .[-1] == "" then .[:-1] else . end |
                   # Join with pipes
                   "| " + join(" | ") + " |"
                 ) |
                 # Add header separator after first row
                 if length > 0 then
                   . as $rows |
                   $rows[0:1] + 
                   [($rows[0] | gsub("[^|]"; "-") | gsub("\\|-"; "|:"))] + 
                   $rows[1:]
                 else
                   .
                 end |
                 join("\n")
                ) + "\n"
            else
                # Fallback for non-HTML tables
                "\n```\n" + .text + "\n```\n"
            end
        elif .type == "CodeSnippet" then 
            # Try to detect language for code snippets
            "\n```" + (.text | detect_code_language) + "\n" + .text + "\n```\n"
        elif .type == "Formula" then 
            # Keep formulas in math notation
            if (.text | test("^\\$.*\\$$")) then
                .text + "\n"
            else
                "$" + .text + "$\n"
            end
        elif .type == "PageBreak" then 
            "\n---\n"
        elif .type == "Image" then
            "![Image](" + (.metadata.image_path // "embedded") + ")\n"
        elif .type == "FigureCaption" then
            "*" + .text + "*\n"
        elif .type == "Footer" then
            "\n---\n*" + .text + "*\n"
        else 
            # Default: just output the text
            .text + "\n"
        end
    ' 2>/dev/null | sed '/^$/N;/^\n$/d'  # Remove excessive blank lines
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
        echo "❌ No files provided for batch processing" >&2
        return 1
    fi
    
    if [[ "${QUIET:-no}" != "yes" ]]; then
        echo "🚀 Starting batch processing of $count documents..." >&2
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
    fi
    
    local success_count=0
    local failed_files=()
    local current_file=0
    local start_time=$(date +%s)
    
    # Function to display progress bar
    show_progress() {
        local current=$1
        local total=$2
        local percent=$((current * 100 / total))
        local filled=$((percent / 2))
        local empty=$((50 - filled))
        
        printf "\r[" >&2
        printf "%${filled}s" | tr ' ' '█' >&2
        printf "%${empty}s" | tr ' ' '░' >&2
        printf "] %3d%% (%d/%d)" "$percent" "$current" "$total" >&2
    }
    
    for file in "${actual_files[@]}"; do
        ((current_file++))
        
        if [[ "${QUIET:-no}" != "yes" ]]; then
            # Show progress bar
            show_progress $current_file $count
            echo >&2
            echo "📄 Processing: $(basename "$file")" >&2
            echo "   Strategy: $strategy | Format: $output_format" >&2
        fi
        
        # Time individual file processing
        local file_start=$(date +%s)
        
        if unstructured_io::process_document "$file" "$strategy" "$output_format"; then
            ((success_count++))
            if [[ "${QUIET:-no}" != "yes" ]]; then
                local file_end=$(date +%s)
                local file_duration=$((file_end - file_start))
                echo "   ✅ Completed in ${file_duration}s" >&2
            fi
        else
            failed_files+=("$file")
            if [[ "${QUIET:-no}" != "yes" ]]; then
                echo "   ❌ Failed to process" >&2
            fi
        fi
        
        if [[ "${QUIET:-no}" != "yes" ]]; then
            echo >&2
        fi
    done
    
    if [[ "${QUIET:-no}" != "yes" ]]; then
        local end_time=$(date +%s)
        local total_duration=$((end_time - start_time))
        local avg_time=$((total_duration / count))
        
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
        echo "📊 Batch Processing Summary" >&2
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
        echo "📁 Total files: $count" >&2
        echo "✅ Successful: $success_count ($((success_count * 100 / count))%)" >&2
        echo "❌ Failed: $((count - success_count)) ($((((count - success_count) * 100) / count))%)" >&2
        echo "⏱️  Total time: ${total_duration}s" >&2
        echo "⚡ Average time per file: ${avg_time}s" >&2
        
        # Show cache statistics if available
        if [[ "${UNSTRUCTURED_IO_CACHE_ENABLED:-yes}" == "yes" ]]; then
            local cache_dir="${UNSTRUCTURED_IO_CACHE_DIR:-$HOME/.vrooli/cache/unstructured-io}"
            if [[ -d "$cache_dir" ]]; then
                local cache_files=$(find "$cache_dir" -type f | wc -l)
                echo "💾 Cached documents: $cache_files" >&2
            fi
        fi
    fi
    
    if [ ${#failed_files[@]} -gt 0 ]; then
        if [[ "${QUIET:-no}" != "yes" ]]; then
            echo >&2
            echo "❌ Failed files:" >&2
            for file in "${failed_files[@]}"; do
                echo "   • $(basename "$file")" >&2
                echo "     Path: $file" >&2
            done
        fi
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
    
    if [[ "${QUIET:-no}" != "yes" ]]; then
        echo "🧪 Testing Unstructured.io API..." >&2
        echo >&2
    fi
    
    # Process the test file
    if unstructured_io::process_document "$test_file" "fast" "markdown"; then
        trash::safe_remove "$test_file" --temp
        if [[ "${QUIET:-no}" != "yes" ]]; then
            echo >&2
            echo "✅ API test passed successfully" >&2
        fi
        return 0
    else
        trash::safe_remove "$test_file" --temp
        if [[ "${QUIET:-no}" != "yes" ]]; then
            echo >&2
            echo "❌ API test failed" >&2
        fi
        return 1
    fi
}

#######################################
# Get supported file types from API
#######################################
unstructured_io::get_supported_types() {
    if ! unstructured_io::status "no"; then
        echo "❌ Service is not available" >&2
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

# Export functions for subshell availability
export -f unstructured_io::process_document
export -f unstructured_io::convert_to_markdown
export -f unstructured_io::batch_process
export -f unstructured_io::test_api
export -f unstructured_io::get_supported_types