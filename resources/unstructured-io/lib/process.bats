#!/usr/bin/env bats
# Tests for Unstructured.io process.sh functions

# Get script directory first
PROCESS_BATS_DIR="${BATS_TEST_DIRNAME}"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${PROCESS_BATS_DIR}/../../../../lib/utils/var.sh"

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/unstructured-io/lib"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Setup for each test
setup() {
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export UNSTRUCTURED_IO_CUSTOM_PORT="9999"
    export UNSTRUCTURED_IO_BASE_URL="http://localhost:9999"
    export UNSTRUCTURED_IO_DEFAULT_STRATEGY="hi_res"
    export UNSTRUCTURED_IO_DEFAULT_LANGUAGES="eng"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    UNSTRUCTURED_IO_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directory and files
    export TEST_DIR="/tmp/unstructured_io_process_test_$$"
    mkdir -p "$TEST_DIR"/{input,output,nested/deep}
    
    # Create test files
    echo "Test document 1" > "$TEST_DIR/input/doc1.txt"
    echo "Test document 2" > "$TEST_DIR/input/doc2.txt"
    echo "Test PDF content" > "$TEST_DIR/input/test.pdf"
    echo "Test Word content" > "$TEST_DIR/input/test.docx"
    echo "Nested document" > "$TEST_DIR/input/nested/nested.txt"
    echo "Deep nested document" > "$TEST_DIR/input/nested/deep/deep.txt"
    
    # Create invalid files
    echo "Binary data" > "$TEST_DIR/input/invalid.bin"
    
    # Mock system functions
    
    # Mock find command
    find() {
        local dir="$1"
        shift
        
        # Simple mock that returns test files based on pattern
        case "$*" in
            *"-name *.txt"*)
                echo "$TEST_DIR/input/doc1.txt"
                echo "$TEST_DIR/input/doc2.txt"
                if [[ "$*" =~ "maxdepth" ]]; then
                    # Non-recursive
                    :
                else
                    # Recursive
                    echo "$TEST_DIR/input/nested/nested.txt"
                    echo "$TEST_DIR/input/nested/deep/deep.txt"
                fi
                ;;
            *"-type f"*)
                echo "$TEST_DIR/input/doc1.txt"
                echo "$TEST_DIR/input/doc2.txt"
                echo "$TEST_DIR/input/test.pdf"
                echo "$TEST_DIR/input/test.docx"
                if [[ ! "$*" =~ "maxdepth" ]]; then
                    echo "$TEST_DIR/input/nested/nested.txt"
                    echo "$TEST_DIR/input/nested/deep/deep.txt"
                fi
                ;;
            *)
                # Fallback
                ls "$dir" 2>/dev/null | head -5
                ;;
        esac
    }
    
    # Mock parallel command
    parallel() {
        # Simple mock that processes each argument
        while IFS= read -r line; do
            echo "PARALLEL_PROCESSING: $line"
        done
    }
    
    # Mock unstructured_io functions
    unstructured_io::process_document() {
        local file="$1"
        local strategy="${2:-hi_res}"
        local output_format="${3:-json}"
        local languages="${4:-eng}"
        
        echo "PROCESSED: $(basename "$file") with strategy=$strategy format=$output_format languages=$languages"
        return 0
    }
    
    unstructured_io::validate_file() {
        local file="$1"
        
        # Reject binary files
        if [[ "$file" =~ "invalid.bin" ]]; then
            echo "ERROR: File not supported"
            return 1
        fi
        
        # Check if file exists
        if [[ ! -f "$file" ]]; then
            echo "ERROR: File does not exist"
            return 1
        fi
        
        return 0
    }
    
    # Mock log functions
    
    
    
    
    # Load configuration and messages
    source "${UNSTRUCTURED_IO_DIR}/config/defaults.sh"
    source "${UNSTRUCTURED_IO_DIR}/config/messages.sh"
    unstructured_io::export_config
    unstructured_io::export_messages
    
    # Load the functions to test
    source "${UNSTRUCTURED_IO_DIR}/lib/process.sh"
}

# Cleanup after each test
teardown() {
    trash::safe_remove "$TEST_DIR" --test-cleanup
}

# Test directory processing - non-recursive
@test "unstructured_io::process_directory processes directory non-recursively" {
    result=$(unstructured_io::process_directory "$TEST_DIR/input" "fast" "json" "no")
    
    [[ "$result" =~ "Processing documents in:" ]]
    [[ "$result" =~ "PROCESSED: doc1.txt" ]]
    [[ "$result" =~ "PROCESSED: doc2.txt" ]]
    [[ "$result" =~ "PROCESSED: test.pdf" ]]
    [[ "$result" =~ "PROCESSED: test.docx" ]]
    [[ ! "$result" =~ "nested.txt" ]]  # Should not include nested files
}

# Test directory processing - recursive
@test "unstructured_io::process_directory processes directory recursively" {
    result=$(unstructured_io::process_directory "$TEST_DIR/input" "hi_res" "markdown" "yes")
    
    [[ "$result" =~ "Processing documents in:" ]]
    [[ "$result" =~ "PROCESSED: doc1.txt" ]]
    [[ "$result" =~ "PROCESSED: nested.txt" ]]
    [[ "$result" =~ "PROCESSED: deep.txt" ]]
    [[ "$result" =~ "strategy=hi_res" ]]
    [[ "$result" =~ "format=markdown" ]]
}

# Test directory processing with default parameters
@test "unstructured_io::process_directory uses default parameters" {
    result=$(unstructured_io::process_directory "$TEST_DIR/input")
    
    [[ "$result" =~ "strategy=hi_res" ]]  # Default strategy
    [[ "$result" =~ "format=json" ]]     # Default format
}

# Test directory processing with invalid directory
@test "unstructured_io::process_directory handles invalid directory" {
    run unstructured_io::process_directory "/nonexistent/directory"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Directory not found" ]]
}

# Test batch processing with file list
@test "unstructured_io::batch_process processes multiple files" {
    local files=("$TEST_DIR/input/doc1.txt" "$TEST_DIR/input/doc2.txt" "$TEST_DIR/input/test.pdf")
    
    result=$(unstructured_io::batch_process "${files[@]}" --strategy "fast" --output "text")
    
    [[ "$result" =~ "Batch processing" ]]
    [[ "$result" =~ "PROCESSED: doc1.txt" ]]
    [[ "$result" =~ "PROCESSED: doc2.txt" ]]
    [[ "$result" =~ "PROCESSED: test.pdf" ]]
    [[ "$result" =~ "strategy=fast" ]]
    [[ "$result" =~ "format=text" ]]
}

# Test batch processing with parallel processing
@test "unstructured_io::batch_process uses parallel processing when available" {
    local files=("$TEST_DIR/input/doc1.txt" "$TEST_DIR/input/doc2.txt")
    
    result=$(unstructured_io::batch_process "${files[@]}" --parallel "yes")
    
    [[ "$result" =~ "parallel processing" ]]
    [[ "$result" =~ "PARALLEL_PROCESSING:" ]]
}

# Test batch processing without parallel
@test "unstructured_io::batch_process processes sequentially without parallel" {
    # Override system check to disable parallel
    system::is_command() {
        case "$1" in
            "parallel") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    local files=("$TEST_DIR/input/doc1.txt" "$TEST_DIR/input/doc2.txt")
    
    result=$(unstructured_io::batch_process "${files[@]}")
    
    [[ "$result" =~ "sequential processing" ]]
    [[ "$result" =~ "PROCESSED: doc1.txt" ]]
    [[ "$result" =~ "PROCESSED: doc2.txt" ]]
}

# Test batch processing with invalid files
@test "unstructured_io::batch_process handles invalid files gracefully" {
    local files=("$TEST_DIR/input/doc1.txt" "$TEST_DIR/input/invalid.bin" "$TEST_DIR/input/doc2.txt")
    
    result=$(unstructured_io::batch_process "${files[@]}")
    
    [[ "$result" =~ "PROCESSED: doc1.txt" ]]
    [[ "$result" =~ "PROCESSED: doc2.txt" ]]
    [[ "$result" =~ "ERROR:" ]]  # Should log error for invalid file
    [[ "$result" =~ "skipping" ]]
}

# Test file filtering by type
@test "unstructured_io::filter_files_by_type filters files correctly" {
    local files=("$TEST_DIR/input/doc1.txt" "$TEST_DIR/input/test.pdf" "$TEST_DIR/input/test.docx" "$TEST_DIR/input/invalid.bin")
    
    result=$(unstructured_io::filter_files_by_type "${files[@]}")
    
    [[ "$result" =~ "doc1.txt" ]]
    [[ "$result" =~ "test.pdf" ]]
    [[ "$result" =~ "test.docx" ]]
    [[ ! "$result" =~ "invalid.bin" ]]
}

# Test processing queue management
@test "unstructured_io::create_processing_queue creates processing queue" {
    local files=("$TEST_DIR/input/doc1.txt" "$TEST_DIR/input/doc2.txt")
    
    result=$(unstructured_io::create_processing_queue "${files[@]}")
    
    [[ "$result" =~ "Processing queue created" ]]
    [[ "$result" =~ "2 files" ]]
}

# Test processing statistics
@test "unstructured_io::get_processing_stats calculates processing statistics" {
    local processed=5
    local failed=2
    local skipped=1
    
    result=$(unstructured_io::get_processing_stats "$processed" "$failed" "$skipped")
    
    [[ "$result" =~ "Total files: 8" ]]
    [[ "$result" =~ "Processed: 5" ]]
    [[ "$result" =~ "Failed: 2" ]]
    [[ "$result" =~ "Skipped: 1" ]]
    [[ "$result" =~ "Success rate: 62%" ]]
}

# Test processing with output directory
@test "unstructured_io::process_with_output_dir processes files to output directory" {
    mkdir -p "$TEST_DIR/output"
    
    result=$(unstructured_io::process_with_output_dir "$TEST_DIR/input/doc1.txt" "$TEST_DIR/output" "json")
    
    [[ "$result" =~ "Processing with output to:" ]]
    [[ "$result" =~ "$TEST_DIR/output" ]]
    [[ "$result" =~ "PROCESSED: doc1.txt" ]]
}

# Test processing with custom filename pattern
@test "unstructured_io::process_with_pattern processes files matching pattern" {
    result=$(unstructured_io::process_with_pattern "$TEST_DIR/input" "*.txt" "fast" "text")
    
    [[ "$result" =~ "Processing files matching pattern:" ]]
    [[ "$result" =~ "*.txt" ]]
    [[ "$result" =~ "PROCESSED: doc1.txt" ]]
    [[ "$result" =~ "PROCESSED: doc2.txt" ]]
    [[ ! "$result" =~ "test.pdf" ]]  # Should not include PDF files
}

# Test processing progress tracking
@test "unstructured_io::track_processing_progress tracks progress correctly" {
    local total=10
    local current=3
    
    result=$(unstructured_io::track_processing_progress "$current" "$total")
    
    [[ "$result" =~ "Progress:" ]]
    [[ "$result" =~ "3/10" ]]
    [[ "$result" =~ "30%" ]]
}

# Test error handling during batch processing
@test "unstructured_io::handle_processing_error handles processing errors" {
    local file="$TEST_DIR/input/problematic.txt"
    local error="Processing timeout"
    
    result=$(unstructured_io::handle_processing_error "$file" "$error")
    
    [[ "$result" =~ "ERROR:" ]]
    [[ "$result" =~ "problematic.txt" ]]
    [[ "$result" =~ "Processing timeout" ]]
}

# Test processing summary generation
@test "unstructured_io::generate_processing_summary generates summary" {
    local start_time="1640995200"  # Mock timestamp
    local end_time="1640995320"    # Mock timestamp (2 minutes later)
    local processed=15
    local failed=2
    
    # Mock date command
    date() {
        case "$*" in
            *"-d @1640995200"*) echo "2022-01-01 12:00:00" ;;
            *"-d @1640995320"*) echo "2022-01-01 12:02:00" ;;
            *) echo "2022-01-01 12:02:00" ;;
        esac
    }
    
    result=$(unstructured_io::generate_processing_summary "$start_time" "$end_time" "$processed" "$failed")
    
    [[ "$result" =~ "Processing Summary" ]]
    [[ "$result" =~ "Duration:" ]]
    [[ "$result" =~ "Processed: 15" ]]
    [[ "$result" =~ "Failed: 2" ]]
    [[ "$result" =~ "Success rate:" ]]
}

# Test concurrent processing management
@test "unstructured_io::manage_concurrent_processing manages concurrent jobs" {
    local max_concurrent=3
    local files=("file1" "file2" "file3" "file4" "file5")
    
    result=$(unstructured_io::manage_concurrent_processing "$max_concurrent" "${files[@]}")
    
    [[ "$result" =~ "Managing concurrent processing" ]]
    [[ "$result" =~ "max: 3" ]]
    [[ "$result" =~ "files: 5" ]]
}

# Test processing retry mechanism
@test "unstructured_io::retry_processing retries failed processing" {
    local file="$TEST_DIR/input/retry_test.txt"
    echo "retry content" > "$file"
    
    # Mock process function to fail first time, succeed second time
    local attempt_count=0
    unstructured_io::process_document() {
        ((attempt_count++))
        if [[ $attempt_count -eq 1 ]]; then
            echo "ERROR: Processing failed"
            return 1
        else
            echo "PROCESSED: retry_test.txt (attempt $attempt_count)"
            return 0
        fi
    }
    
    result=$(unstructured_io::retry_processing "$file" "hi_res" "json" "eng" 2)
    
    [[ "$result" =~ "Retrying" ]]
    [[ "$result" =~ "PROCESSED: retry_test.txt (attempt 2)" ]]
}

# Test processing with timeout
@test "unstructured_io::process_with_timeout processes with timeout constraints" {
    local timeout=30
    
    result=$(unstructured_io::process_with_timeout "$TEST_DIR/input/doc1.txt" "fast" "json" "eng" "$timeout")
    
    [[ "$result" =~ "Processing with timeout:" ]]
    [[ "$result" =~ "30 seconds" ]]
    [[ "$result" =~ "PROCESSED: doc1.txt" ]]
}