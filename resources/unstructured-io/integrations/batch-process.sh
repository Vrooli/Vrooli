#!/usr/bin/env bash
# Batch process multiple documents with progress tracking
# Supports various output formats and parallel processing

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/unstructured-io/integrations"
# Source trash module for safe cleanup
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
MANAGE_SCRIPT="${SCRIPT_DIR}/../manage.sh"

# Default values
OUTPUT_FORMAT="json"
OUTPUT_DIR="./processed"
MAX_PARALLEL=3
QUIET="no"

# Usage function
usage() {
    echo "Usage: $0 [OPTIONS] <directory_or_files>"
    echo
    echo "Options:"
    echo "  -o, --output FORMAT     Output format (json|markdown|text|elements) [default: json]"
    echo "  -d, --output-dir DIR    Output directory [default: ./processed]"
    echo "  -p, --parallel N        Max parallel processes [default: 3]"
    echo "  -q, --quiet             Suppress progress messages"
    echo "  -h, --help              Show this help message"
    echo
    echo "Examples:"
    echo "  $0 /path/to/documents                    # Process all supported files in directory"
    echo "  $0 -o markdown -d ./output *.pdf         # Process PDFs to markdown"
    echo "  $0 -p 5 -q document1.pdf document2.docx  # Process specific files quietly"
    exit 0
}

# Parse arguments
FILES=()
while [[ $# -gt 0 ]]; do
    case "$1" in
        -o|--output)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        -d|--output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -p|--parallel)
            MAX_PARALLEL="$2"
            shift 2
            ;;
        -q|--quiet)
            QUIET="yes"
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            # Check if it's a directory
            if [[ -d "$1" ]]; then
                # Find all supported files in directory
                while IFS= read -r -d '' file; do
                    FILES+=("$file")
                done < <(find "$1" -maxdepth 1 -type f \( \
                    -iname "*.pdf" -o -iname "*.docx" -o -iname "*.doc" -o \
                    -iname "*.pptx" -o -iname "*.ppt" -o -iname "*.xlsx" -o \
                    -iname "*.xls" -o -iname "*.txt" -o -iname "*.md" -o \
                    -iname "*.html" -o -iname "*.rtf" -o -iname "*.csv" \
                \) -print0)
            else
                FILES+=("$1")
            fi
            shift
            ;;
    esac
done

# Check if we have files to process
if [[ ${#FILES[@]} -eq 0 ]]; then
    echo "Error: No files to process"
    echo "Use -h for help"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Progress tracking
TOTAL=${#FILES[@]}
COMPLETED=0
FAILED=0
START_TIME=$(date +%s)

# Lock file for counter updates
LOCK_FILE="/tmp/batch_process_$$.lock"
COMPLETED_FILE="/tmp/batch_process_$$.completed"
FAILED_FILE="/tmp/batch_process_$$.failed"

# Initialize counters
echo "0" > "$COMPLETED_FILE"
echo "0" > "$FAILED_FILE"

# Cleanup function
cleanup() {
    trash::safe_remove "$LOCK_FILE" --temp
    trash::safe_remove "$COMPLETED_FILE" --temp
    trash::safe_remove "$FAILED_FILE" --temp
}
trap cleanup EXIT

# Progress display function
show_progress() {
    if [[ "$QUIET" != "yes" ]]; then
        local completed=$(cat "$COMPLETED_FILE")
        local failed=$(cat "$FAILED_FILE")
        local current=$((completed + failed))
        local percent=$((current * 100 / TOTAL))
        echo -ne "\r📊 Progress: [$current/$TOTAL] ${percent}% | ✅ Success: $completed | ❌ Failed: $failed"
    fi
}

# Process single file function
process_file() {
    local file="$1"
    local basename=$(basename "$file" | sed 's/\.[^.]*$//')
    local output_ext=""
    
    case "$OUTPUT_FORMAT" in
        json) output_ext=".json" ;;
        markdown) output_ext=".md" ;;
        text) output_ext=".txt" ;;
        elements) output_ext=".elements.txt" ;;
    esac
    
    local output_file="$OUTPUT_DIR/${basename}${output_ext}"
    
    # Process the file
    if "$MANAGE_SCRIPT" --action process --file "$file" --output "$OUTPUT_FORMAT" --quiet yes > "$output_file" 2>/dev/null; then
        # Update completed counter
        (
            flock -x 200
            local count=$(cat "$COMPLETED_FILE")
            echo $((count + 1)) > "$COMPLETED_FILE"
        ) 200>"$LOCK_FILE"
    else
        # Update failed counter
        (
            flock -x 200
            local count=$(cat "$FAILED_FILE")
            echo $((count + 1)) > "$FAILED_FILE"
            [[ "$QUIET" != "yes" ]] && echo -e "\n⚠️  Failed: $file"
        ) 200>"$LOCK_FILE"
    fi
    
    show_progress
}

# Export functions for parallel execution
export -f process_file show_progress
export MANAGE_SCRIPT OUTPUT_FORMAT OUTPUT_DIR QUIET TOTAL
export LOCK_FILE COMPLETED_FILE FAILED_FILE

# Header
if [[ "$QUIET" != "yes" ]]; then
    echo "🚀 Batch Document Processing"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Files to process: $TOTAL"
    echo "Output format: $OUTPUT_FORMAT"
    echo "Output directory: $OUTPUT_DIR"
    echo "Parallel jobs: $MAX_PARALLEL"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
fi

# Process files in parallel
printf '%s\0' "${FILES[@]}" | xargs -0 -P "$MAX_PARALLEL" -I {} bash -c 'process_file "$@"' _ {}

# Final statistics
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
COMPLETED=$(cat "$COMPLETED_FILE")
FAILED=$(cat "$FAILED_FILE")

if [[ "$QUIET" != "yes" ]]; then
    echo -e "\n\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✅ Processing Complete"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Total files: $TOTAL"
    echo "Successful: $COMPLETED"
    echo "Failed: $FAILED"
    echo "Duration: ${DURATION}s"
    echo "Output directory: $OUTPUT_DIR"
fi

# Exit with error if any files failed
[[ $FAILED -eq 0 ]] || exit 1