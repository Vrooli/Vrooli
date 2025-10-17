#!/usr/bin/env bash
# Unstructured-IO Mock - Tier 2 (Stateful)
# 
# Provides stateful Unstructured.io document processing mocking for testing:
# - Document processing (PDF, DOCX, HTML, TXT, etc.)
# - Text extraction and chunking
# - Table and metadata extraction
# - Cache management
# - Service lifecycle (start, stop, restart)
# - Error injection for resilience testing
#
# Coverage: ~80% of common Unstructured.io operations in 450 lines

# === Configuration ===
declare -gA UNSTRUCTURED_PROCESSED=()    # File_path -> "status|elements|chunks|tables"
declare -gA UNSTRUCTURED_CACHE=()        # Cache_key -> "result|timestamp"
declare -gA UNSTRUCTURED_JOBS=()         # Job_id -> "file|status|progress"
declare -gA UNSTRUCTURED_CONFIG=(        # Service configuration
    [status]="running"
    [port]="8000"
    [cache_enabled]="true"
    [strategy]="auto"
    [error_mode]=""
    [version]="0.10.30"
)

# Debug mode
declare -g UNSTRUCTURED_DEBUG="${UNSTRUCTURED_DEBUG:-}"

# === Helper Functions ===
unstructured_debug() {
    [[ -n "$UNSTRUCTURED_DEBUG" ]] && echo "[MOCK:UNSTRUCTURED] $*" >&2
}

unstructured_check_error() {
    case "${UNSTRUCTURED_CONFIG[error_mode]}" in
        "service_down")
            echo "Error: Unstructured service is not running" >&2
            return 1
            ;;
        "processing_failed")
            echo "Error: Document processing failed" >&2
            return 1
            ;;
        "unsupported_format")
            echo "Error: Unsupported document format" >&2
            return 1
            ;;
    esac
    return 0
}

unstructured_generate_id() {
    printf "job_%08x" $RANDOM
}

# === Main Unstructured Command ===
unstructured() {
    unstructured_debug "unstructured called with: $*"
    
    if ! unstructured_check_error; then
        return $?
    fi
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        process)
            unstructured_cmd_process "$@"
            ;;
        extract)
            unstructured_cmd_extract "$@"
            ;;
        chunk)
            unstructured_cmd_chunk "$@"
            ;;
        cache)
            unstructured_cmd_cache "$@"
            ;;
        jobs)
            unstructured_cmd_jobs "$@"
            ;;
        status)
            unstructured_cmd_status "$@"
            ;;
        start|stop|restart)
            unstructured_cmd_service "$command" "$@"
            ;;
        *)
            echo "Unstructured.io CLI - Document Processing Service"
            echo "Commands:"
            echo "  process  - Process a document"
            echo "  extract  - Extract text from document"
            echo "  chunk    - Chunk document text"
            echo "  cache    - Manage cache"
            echo "  jobs     - View processing jobs"
            echo "  status   - Show service status"
            echo "  start    - Start service"
            echo "  stop     - Stop service"
            echo "  restart  - Restart service"
            ;;
    esac
}

# === Document Processing ===
unstructured_cmd_process() {
    local file="${1:-}"
    local strategy="${2:-${UNSTRUCTURED_CONFIG[strategy]}}"
    
    [[ -z "$file" ]] && { echo "Error: file path required" >&2; return 1; }
    
    # Check file extension
    local ext="${file##*.}"
    local supported="pdf docx txt html md pptx xlsx csv json xml"
    
    if ! echo "$supported" | grep -qw "$ext"; then
        echo "Error: Unsupported format: .$ext" >&2
        return 1
    fi
    
    local job_id
    job_id=$(unstructured_generate_id)
    UNSTRUCTURED_JOBS[$job_id]="$file|processing|0"
    
    echo "Processing document: $file"
    echo "Job ID: $job_id"
    echo "Strategy: $strategy"
    
    # Simulate processing based on file type
    local elements chunks tables
    case "$ext" in
        pdf)
            elements="42"
            chunks="8"
            tables="3"
            ;;
        docx)
            elements="35"
            chunks="6"
            tables="2"
            ;;
        xlsx)
            elements="120"
            chunks="15"
            tables="5"
            ;;
        *)
            elements="20"
            chunks="4"
            tables="0"
            ;;
    esac
    
    # Update job progress
    UNSTRUCTURED_JOBS[$job_id]="$file|processing|50"
    UNSTRUCTURED_JOBS[$job_id]="$file|completed|100"
    
    # Store processed data
    UNSTRUCTURED_PROCESSED[$file]="processed|$elements|$chunks|$tables"
    
    # Cache if enabled
    if [[ "${UNSTRUCTURED_CONFIG[cache_enabled]}" == "true" ]]; then
        local cache_key="${file}_${strategy}"
        UNSTRUCTURED_CACHE[$cache_key]="$elements,$chunks,$tables|$(date +%s)"
    fi
    
    echo "Processing complete!"
    echo "Elements extracted: $elements"
    echo "Chunks created: $chunks"
    echo "Tables found: $tables"
}

# === Text Extraction ===
unstructured_cmd_extract() {
    local file="${1:-}"
    local output_format="${2:-text}"
    
    [[ -z "$file" ]] && { echo "Error: file path required" >&2; return 1; }
    
    # Check if already processed
    if [[ -z "${UNSTRUCTURED_PROCESSED[$file]}" ]]; then
        # Process first
        unstructured_cmd_process "$file" >/dev/null 2>&1
    fi
    
    local data="${UNSTRUCTURED_PROCESSED[$file]}"
    IFS='|' read -r status elements chunks tables <<< "$data"
    
    case "$output_format" in
        text)
            echo "Extracted text from $file:"
            echo "Lorem ipsum dolor sit amet, consectetur adipiscing elit."
            echo "This is sample extracted text with $elements elements."
            ;;
        json)
            echo '{"file":"'$file'","elements":'$elements',"text":"Sample extracted text"}'
            ;;
        markdown)
            echo "# Extracted Content"
            echo "## Document: $file"
            echo "Sample extracted text with **$elements** elements."
            ;;
        *)
            echo "Error: Unknown format: $output_format" >&2
            return 1
            ;;
    esac
}

# === Text Chunking ===
unstructured_cmd_chunk() {
    local file="${1:-}"
    local chunk_size="${2:-500}"
    
    [[ -z "$file" ]] && { echo "Error: file path required" >&2; return 1; }
    
    # Check if already processed
    if [[ -z "${UNSTRUCTURED_PROCESSED[$file]}" ]]; then
        unstructured_cmd_process "$file" >/dev/null 2>&1
    fi
    
    local data="${UNSTRUCTURED_PROCESSED[$file]}"
    IFS='|' read -r status elements chunks tables <<< "$data"
    
    echo "Chunking document: $file"
    echo "Chunk size: $chunk_size characters"
    echo "Chunks created: $chunks"
    echo ""
    echo "Chunk 1: Lorem ipsum dolor sit amet..."
    echo "Chunk 2: Consectetur adipiscing elit..."
    if [[ $chunks -gt 2 ]]; then
        echo "... ($((chunks-2)) more chunks)"
    fi
}

# === Cache Management ===
unstructured_cmd_cache() {
    local action="${1:-list}"
    
    case "$action" in
        list)
            echo "Cache entries:"
            if [[ ${#UNSTRUCTURED_CACHE[@]} -eq 0 ]]; then
                echo "  (empty)"
            else
                for key in "${!UNSTRUCTURED_CACHE[@]}"; do
                    local data="${UNSTRUCTURED_CACHE[$key]}"
                    IFS='|' read -r result timestamp <<< "$data"
                    echo "  $key - Cached at: $(date -d @$timestamp 2>/dev/null || echo $timestamp)"
                done
            fi
            ;;
        clear)
            UNSTRUCTURED_CACHE=()
            echo "Cache cleared"
            ;;
        enable)
            UNSTRUCTURED_CONFIG[cache_enabled]="true"
            echo "Cache enabled"
            ;;
        disable)
            UNSTRUCTURED_CONFIG[cache_enabled]="false"
            echo "Cache disabled"
            ;;
        *)
            echo "Usage: unstructured cache {list|clear|enable|disable}"
            return 1
            ;;
    esac
}

# === Jobs Management ===
unstructured_cmd_jobs() {
    echo "Processing jobs:"
    if [[ ${#UNSTRUCTURED_JOBS[@]} -eq 0 ]]; then
        echo "  (none)"
    else
        for job_id in "${!UNSTRUCTURED_JOBS[@]}"; do
            local data="${UNSTRUCTURED_JOBS[$job_id]}"
            IFS='|' read -r file status progress <<< "$data"
            echo "  $job_id - File: $file, Status: $status, Progress: ${progress}%"
        done
    fi
}

# === Status Command ===
unstructured_cmd_status() {
    echo "Unstructured.io Status"
    echo "====================="
    echo "Service: ${UNSTRUCTURED_CONFIG[status]}"
    echo "Port: ${UNSTRUCTURED_CONFIG[port]}"
    echo "Cache: ${UNSTRUCTURED_CONFIG[cache_enabled]}"
    echo "Strategy: ${UNSTRUCTURED_CONFIG[strategy]}"
    echo "Version: ${UNSTRUCTURED_CONFIG[version]}"
    echo ""
    echo "Processed Documents: ${#UNSTRUCTURED_PROCESSED[@]}"
    echo "Cache Entries: ${#UNSTRUCTURED_CACHE[@]}"
    echo "Jobs: ${#UNSTRUCTURED_JOBS[@]}"
}

# === Service Management ===
unstructured_cmd_service() {
    local action="$1"
    
    case "$action" in
        start)
            if [[ "${UNSTRUCTURED_CONFIG[status]}" == "running" ]]; then
                echo "Unstructured.io is already running"
            else
                UNSTRUCTURED_CONFIG[status]="running"
                echo "Unstructured.io started on port ${UNSTRUCTURED_CONFIG[port]}"
            fi
            ;;
        stop)
            UNSTRUCTURED_CONFIG[status]="stopped"
            echo "Unstructured.io stopped"
            ;;
        restart)
            UNSTRUCTURED_CONFIG[status]="stopped"
            UNSTRUCTURED_CONFIG[status]="running"
            echo "Unstructured.io restarted"
            ;;
    esac
}

# === HTTP API Mock (via curl interceptor) ===
curl() {
    unstructured_debug "curl called with: $*"
    
    local url="" method="GET" data="" file=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X) method="$2"; shift 2 ;;
            -d|--data) data="$2"; shift 2 ;;
            -F) 
                # Parse form data for file upload
                if [[ "$2" =~ files=@(.+) ]]; then
                    file="${BASH_REMATCH[1]}"
                fi
                shift 2
                ;;
            http*) url="$1"; shift ;;
            *) shift ;;
        esac
    done
    
    # Check if this is an Unstructured API call
    if [[ "$url" =~ localhost:8000 || "$url" =~ 127.0.0.1:8000 ]]; then
        unstructured_handle_api "$method" "$url" "$data" "$file"
        return $?
    fi
    
    echo "curl: Not an Unstructured endpoint"
    return 0
}

unstructured_handle_api() {
    local method="$1" url="$2" data="$3" file="$4"
    
    case "$url" in
        */general/v0/general)
            if [[ "$method" == "POST" ]]; then
                local job_id
                job_id=$(unstructured_generate_id)
                echo '{"job_id":"'$job_id'","status":"processing","elements":[]}'
            else
                echo '{"error":"Method not allowed"}'
            fi
            ;;
        */documents/process)
            if [[ -n "$file" ]]; then
                echo '{"status":"success","elements":20,"chunks":4}'
            else
                echo '{"error":"No file provided"}'
            fi
            ;;
        */health)
            if [[ "${UNSTRUCTURED_CONFIG[status]}" == "running" ]]; then
                echo '{"status":"healthy","version":"'${UNSTRUCTURED_CONFIG[version]}'"}'
            else
                echo '{"status":"unhealthy","error":"Service not running"}'
            fi
            ;;
        *)
            echo '{"status":"ok","version":"'${UNSTRUCTURED_CONFIG[version]}'"}'
            ;;
    esac
}

# === Mock Control Functions ===
unstructured_mock_reset() {
    unstructured_debug "Resetting mock state"
    
    UNSTRUCTURED_PROCESSED=()
    UNSTRUCTURED_CACHE=()
    UNSTRUCTURED_JOBS=()
    UNSTRUCTURED_CONFIG[error_mode]=""
    UNSTRUCTURED_CONFIG[status]="running"
    UNSTRUCTURED_CONFIG[cache_enabled]="true"
}

unstructured_mock_set_error() {
    UNSTRUCTURED_CONFIG[error_mode]="$1"
    unstructured_debug "Set error mode: $1"
}

unstructured_mock_dump_state() {
    echo "=== Unstructured Mock State ==="
    echo "Status: ${UNSTRUCTURED_CONFIG[status]}"
    echo "Port: ${UNSTRUCTURED_CONFIG[port]}"
    echo "Cache: ${UNSTRUCTURED_CONFIG[cache_enabled]}"
    echo "Processed: ${#UNSTRUCTURED_PROCESSED[@]}"
    echo "Cache Entries: ${#UNSTRUCTURED_CACHE[@]}"
    echo "Jobs: ${#UNSTRUCTURED_JOBS[@]}"
    echo "Error Mode: ${UNSTRUCTURED_CONFIG[error_mode]:-none}"
    echo "========================"
}

# === Convention-based Test Functions ===
test_unstructured_connection() {
    unstructured_debug "Testing connection..."
    
    local result
    result=$(curl -s http://localhost:8000/health 2>&1)
    
    if [[ "$result" =~ "status" ]]; then
        unstructured_debug "Connection test passed"
        return 0
    else
        unstructured_debug "Connection test failed"
        return 1
    fi
}

test_unstructured_health() {
    unstructured_debug "Testing health..."
    
    test_unstructured_connection || return 1
    
    unstructured process test.pdf >/dev/null 2>&1 || return 1
    unstructured cache list >/dev/null 2>&1 || return 1
    
    unstructured_debug "Health test passed"
    return 0
}

test_unstructured_basic() {
    unstructured_debug "Testing basic operations..."
    
    unstructured process test.docx >/dev/null 2>&1 || return 1
    unstructured extract test.docx text >/dev/null 2>&1 || return 1
    unstructured chunk test.docx 500 >/dev/null 2>&1 || return 1
    
    unstructured_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f unstructured curl
export -f test_unstructured_connection test_unstructured_health test_unstructured_basic
export -f unstructured_mock_reset unstructured_mock_set_error unstructured_mock_dump_state
export -f unstructured_debug unstructured_check_error

# Initialize
unstructured_mock_reset
unstructured_debug "Unstructured.io Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
