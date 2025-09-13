#!/usr/bin/env bash
# Unstructured.io Content Management Functions
# Business functionality for document processing and analysis

# Add document for processing (not applicable - processes directly)
unstructured_io::content::add() {
    echo "Document storage not applicable - Unstructured.io processes files directly"
    echo "Use 'content process <file>' to process documents"
}

# List processed documents
unstructured_io::content::list() {
    echo "Recent processing results:"
    local data_dir="${UNSTRUCTURED_DATA_DIR:-$HOME/.unstructured}"
    if command -v find >/dev/null 2>&1; then
        find "${data_dir}/cache" -name "*.json" -type f 2>/dev/null | head -10 || echo "No processed documents found"
    else
        echo "Unable to list - find command not available"
    fi
}

# Get processed document result
unstructured_io::content::get() {
    local file_id="${1:-}"
    if [[ -z "$file_id" ]]; then
        echo "Usage: content get <file_id_or_path>"
        return 1
    fi
    
    # Try to find and display cached result
    if [[ -f "$file_id" ]]; then
        cat "$file_id"
    else
        echo "File not found: $file_id"
        return 1
    fi
}

# Remove processed document result
unstructured_io::content::remove() {
    local file_id="${1:-}"
    if [[ -z "$file_id" ]]; then
        echo "Usage: content remove <file_id_or_path>"
        return 1
    fi
    
    if [[ -f "$file_id" ]]; then
        rm -f "$file_id"
        echo "Removed: $file_id"
    else
        echo "File not found: $file_id"
        return 1
    fi
}

# Execute document processing (main business functionality)
unstructured_io::content::execute() {
    local file="${1:-}"
    local strategy="${2:-$UNSTRUCTURED_IO_DEFAULT_STRATEGY}"
    local output="${3:-}"
    
    if [[ -z "$file" ]]; then
        echo "Usage: content execute <file> [strategy] [output]"
        echo "Strategies: auto, fast, hi_res, ocr_only"
        return 1
    fi
    
    # Delegate to the main processing function
    unstructured_io::process_document "$file" "$strategy" "$output"
}

# Process document (alias for execute)
unstructured_io::content::process() {
    unstructured_io::content::execute "$@"
}

# Process directory of documents
unstructured_io::content::process_directory() {
    local dir="${1:-}"
    if [[ -z "$dir" ]]; then
        echo "❌ Error: Directory path required" >&2
        echo "Usage: vrooli resource unstructured-io content process-directory <directory> [strategy] [format] [recursive]" >&2
        return 1
    fi
    
    # Call the actual process_directory function with proper arguments
    unstructured_io::process_directory "$dir" "${2:-}" "${3:-}" "${4:-}"
}

# Clear processing cache
unstructured_io::content::clear_cache() {
    local file="${1:-}"
    if [[ -n "$file" ]]; then
        unstructured_io::clear_cache "$file"
    else
        unstructured_io::clear_all_cache
    fi
}

# Show processing results
unstructured_io::content::results() {
    local data_dir="${UNSTRUCTURED_DATA_DIR:-$HOME/.unstructured}"
    echo "Processing results directory: $data_dir/results"
    
    if [[ -d "$data_dir/results" ]]; then
        ls -la "$data_dir/results" 2>/dev/null || echo "No results found"
    else
        echo "Results directory does not exist"
    fi
}