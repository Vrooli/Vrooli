#!/usr/bin/env bash
# Unstructured.io Mock Implementation
# 
# Provides a comprehensive mock for Unstructured.io operations including:
# - Service management (install, start, stop, status)
# - Document processing (text extraction, table extraction, etc.)
# - API endpoint simulation
# - Cache management
# - Batch processing
# - State management and persistence
#
# This mock follows the same standards as other mocks with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions

# Prevent duplicate loading
[[ -n "${UNSTRUCTURED_IO_MOCK_LOADED:-}" ]] && return 0
declare -g UNSTRUCTURED_IO_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"

# Global configuration
declare -g UNSTRUCTURED_IO_MOCK_STATE_DIR="${UNSTRUCTURED_IO_MOCK_STATE_DIR:-/tmp/unstructured-io-mock-state}"
declare -g UNSTRUCTURED_IO_MOCK_DEBUG="${UNSTRUCTURED_IO_MOCK_DEBUG:-}"

# Global state arrays
declare -gA UNSTRUCTURED_IO_MOCK_CONFIG=(          # Service configuration
    [installed]="false"
    [running]="false"
    [container_id]="unstructured-io-mock-container"
    [host]="localhost"
    [port]="8000"
    [version]="0.10.30"
    [api_url]="http://localhost:8000"
    [health_status]="healthy"
    [error_mode]=""
    [cache_enabled]="true"
    [default_strategy]="auto"
    [supported_formats]="pdf,docx,txt,html,md,pptx,xlsx"
)
declare -gA UNSTRUCTURED_IO_MOCK_PROCESSED_DOCS=() # document_path -> processing_result
declare -gA UNSTRUCTURED_IO_MOCK_CACHE=()          # cache_key -> cached_result
declare -gA UNSTRUCTURED_IO_MOCK_EXTRACTION_DATA=() # document_path -> extracted_data
declare -gA UNSTRUCTURED_IO_MOCK_ERRORS=()         # command -> error_type

# Initialize state directory
mkdir -p "$UNSTRUCTURED_IO_MOCK_STATE_DIR"

# State persistence functions
mock::unstructured_io::save_state() {
    local state_file="$UNSTRUCTURED_IO_MOCK_STATE_DIR/unstructured-io-state.sh"
    {
        echo "# Unstructured.io mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p UNSTRUCTURED_IO_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA UNSTRUCTURED_IO_MOCK_CONFIG=()"
        declare -p UNSTRUCTURED_IO_MOCK_PROCESSED_DOCS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA UNSTRUCTURED_IO_MOCK_PROCESSED_DOCS=()"
        declare -p UNSTRUCTURED_IO_MOCK_CACHE 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA UNSTRUCTURED_IO_MOCK_CACHE=()"
        declare -p UNSTRUCTURED_IO_MOCK_EXTRACTION_DATA 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA UNSTRUCTURED_IO_MOCK_EXTRACTION_DATA=()"
        declare -p UNSTRUCTURED_IO_MOCK_ERRORS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA UNSTRUCTURED_IO_MOCK_ERRORS=()"
    } > "$state_file"
    
    mock::log_state "unstructured_io" "Saved Unstructured.io state to $state_file"
}

mock::unstructured_io::load_state() {
    local state_file="$UNSTRUCTURED_IO_MOCK_STATE_DIR/unstructured-io-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        mock::log_state "unstructured_io" "Loaded Unstructured.io state from $state_file"
    fi
}

# Automatically load state when sourced
mock::unstructured_io::load_state

# Main manage.sh script interceptor
unstructured-io-manage() {
    mock::log_and_verify "unstructured-io-manage" "$@"
    
    # Always reload state at the beginning to handle BATS subshells
    mock::unstructured_io::load_state
    
    # Check for error injection
    if [[ -n "${UNSTRUCTURED_IO_MOCK_CONFIG[error_mode]}" ]]; then
        case "${UNSTRUCTURED_IO_MOCK_CONFIG[error_mode]}" in
            "service_unavailable")
                echo "Error: Unstructured.io service is unavailable" >&2
                return 1
                ;;
            "installation_failed")
                echo "Error: Installation failed" >&2
                return 1
                ;;
            "processing_error")
                echo "Error: Document processing failed" >&2
                return 1
                ;;
        esac
    fi
    
    # Parse the action from arguments
    local action=""
    local args=()
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --action)
                action="$2"
                shift 2
                ;;
            *)
                args+=("$1")
                shift
                ;;
        esac
    done
    
    # Route to appropriate action
    case "$action" in
        install)
            mock::unstructured_io::action_install "${args[@]}"
            ;;
        uninstall)
            mock::unstructured_io::action_uninstall "${args[@]}"
            ;;
        start)
            mock::unstructured_io::action_start "${args[@]}"
            ;;
        stop)
            mock::unstructured_io::action_stop "${args[@]}"
            ;;
        restart)
            mock::unstructured_io::action_restart "${args[@]}"
            ;;
        status)
            mock::unstructured_io::action_status "${args[@]}"
            ;;
        process)
            mock::unstructured_io::action_process "${args[@]}"
            ;;
        info)
            mock::unstructured_io::action_info "${args[@]}"
            ;;
        logs)
            mock::unstructured_io::action_logs "${args[@]}"
            ;;
        test)
            mock::unstructured_io::action_test "${args[@]}"
            ;;
        extract-tables)
            mock::unstructured_io::action_extract_tables "${args[@]}"
            ;;
        extract-metadata)
            mock::unstructured_io::action_extract_metadata "${args[@]}"
            ;;
        process-directory)
            mock::unstructured_io::action_process_directory "${args[@]}"
            ;;
        create-report)
            mock::unstructured_io::action_create_report "${args[@]}"
            ;;
        cache-stats)
            mock::unstructured_io::action_cache_stats "${args[@]}"
            ;;
        clear-cache)
            mock::unstructured_io::action_clear_cache "${args[@]}"
            ;;
        validate-installation)
            mock::unstructured_io::action_validate_installation "${args[@]}"
            ;;
        *)
            echo "Unknown action: $action" >&2
            echo "Usage: unstructured-io-manage --action <action> [options]" >&2
            return 1
            ;;
    esac
    
    local result=$?
    
    # Save state after each command
    mock::unstructured_io::save_state
    
    return $result
}

# Action implementations
mock::unstructured_io::action_install() {
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[installed]}" == "true" ]]; then
        echo "📦 Unstructured.io is already installed"
        return 0
    fi
    
    echo "🔄 Installing Unstructured.io service..."
    echo "📥 Pulling Docker image: unstructuredai/unstructured:${UNSTRUCTURED_IO_MOCK_CONFIG[version]}"
    echo "✅ Installation completed successfully"
    
    UNSTRUCTURED_IO_MOCK_CONFIG[installed]="true"
    return 0
}

mock::unstructured_io::action_uninstall() {
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[installed]}" == "false" ]]; then
        echo "📦 Unstructured.io is not installed"
        return 0
    fi
    
    # Stop service if running
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[running]}" == "true" ]]; then
        mock::unstructured_io::action_stop
    fi
    
    echo "🗑️  Uninstalling Unstructured.io service..."
    echo "✅ Uninstallation completed"
    
    UNSTRUCTURED_IO_MOCK_CONFIG[installed]="false"
    return 0
}

mock::unstructured_io::action_start() {
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[installed]}" == "false" ]]; then
        echo "❌ Unstructured.io is not installed. Please run install first." >&2
        return 1
    fi
    
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[running]}" == "true" ]]; then
        echo "🟢 Unstructured.io service is already running"
        return 0
    fi
    
    echo "🚀 Starting Unstructured.io service..."
    echo "📍 Service will be available at: ${UNSTRUCTURED_IO_MOCK_CONFIG[api_url]}"
    echo "✅ Unstructured.io started successfully"
    
    UNSTRUCTURED_IO_MOCK_CONFIG[running]="true"
    return 0
}

mock::unstructured_io::action_stop() {
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[running]}" == "false" ]]; then
        echo "🔴 Unstructured.io service is not running"
        return 0
    fi
    
    echo "⏹️  Stopping Unstructured.io service..."
    echo "✅ Unstructured.io stopped successfully"
    
    UNSTRUCTURED_IO_MOCK_CONFIG[running]="false"
    return 0
}

mock::unstructured_io::action_restart() {
    mock::unstructured_io::action_stop
    sleep 1
    mock::unstructured_io::action_start
}

mock::unstructured_io::action_status() {
    echo "🔍 Checking Unstructured.io status..."
    echo ""
    
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[installed]}" == "false" ]]; then
        echo "📦 Status: Not installed"
        return 1
    fi
    
    echo "📦 Status: Installed"
    echo "📍 Container: ${UNSTRUCTURED_IO_MOCK_CONFIG[container_id]}"
    
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[running]}" == "false" ]]; then
        echo "🔴 Service: Stopped"
        return 1
    fi
    
    echo "🟢 Service: Running"
    echo "🌐 API URL: ${UNSTRUCTURED_IO_MOCK_CONFIG[api_url]}"
    echo "🏥 Health: ${UNSTRUCTURED_IO_MOCK_CONFIG[health_status]}"
    echo "📊 Version: ${UNSTRUCTURED_IO_MOCK_CONFIG[version]}"
    
    return 0
}

mock::unstructured_io::action_process() {
    local file=""
    local strategy="${UNSTRUCTURED_IO_MOCK_CONFIG[default_strategy]}"
    local output="json"
    local languages="en"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --strategy)
                strategy="$2"
                shift 2
                ;;
            --output)
                output="$2"
                shift 2
                ;;
            --languages)
                languages="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        echo "Error: No file provided for processing" >&2
        return 1
    fi
    
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[running]}" == "false" ]]; then
        echo "❌ Unstructured.io service is not running" >&2
        return 1
    fi
    
    # Check if file has already been processed (cache simulation)
    local cache_key="${file}|${strategy}|${output}|${languages}"
    if [[ -n "${UNSTRUCTURED_IO_MOCK_CACHE[$cache_key]}" ]]; then
        echo "📦 Using cached result for: $(basename "$file")"
        echo "${UNSTRUCTURED_IO_MOCK_CACHE[$cache_key]}"
        return 0
    fi
    
    # Simulate processing
    echo "📄 Processing document: $(basename "$file")"
    echo "🎯 Strategy: $strategy"
    echo "📊 Output format: $output"
    
    # Generate mock result based on file extension
    local result
    case "$output" in
        json)
            result=$(mock::unstructured_io::generate_json_result "$file" "$strategy")
            ;;
        text)
            result=$(mock::unstructured_io::generate_text_result "$file" "$strategy")
            ;;
        *)
            result=$(mock::unstructured_io::generate_json_result "$file" "$strategy")
            ;;
    esac
    
    # Cache the result
    UNSTRUCTURED_IO_MOCK_CACHE[$cache_key]="$result"
    UNSTRUCTURED_IO_MOCK_PROCESSED_DOCS["$file"]="$result"
    
    echo "✅ Processing completed"
    echo "$result"
    
    return 0
}

mock::unstructured_io::action_info() {
    echo "📋 Unstructured.io Service Information"
    echo "======================================"
    echo "Version: ${UNSTRUCTURED_IO_MOCK_CONFIG[version]}"
    echo "API URL: ${UNSTRUCTURED_IO_MOCK_CONFIG[api_url]}"
    echo "Port: ${UNSTRUCTURED_IO_MOCK_CONFIG[port]}"
    echo "Container: ${UNSTRUCTURED_IO_MOCK_CONFIG[container_id]}"
    echo "Health: ${UNSTRUCTURED_IO_MOCK_CONFIG[health_status]}"
    echo "Cache: ${UNSTRUCTURED_IO_MOCK_CONFIG[cache_enabled]}"
    echo "Supported formats: ${UNSTRUCTURED_IO_MOCK_CONFIG[supported_formats]}"
    echo "Default strategy: ${UNSTRUCTURED_IO_MOCK_CONFIG[default_strategy]}"
}

mock::unstructured_io::action_logs() {
    local follow="${1:-false}"
    
    echo "📜 Unstructured.io Service Logs"
    echo "================================"
    echo "[$(date)] INFO Starting unstructured server on port ${UNSTRUCTURED_IO_MOCK_CONFIG[port]}"
    echo "[$(date)] INFO Health check endpoint available at /healthcheck"
    echo "[$(date)] INFO Document processing endpoint available at /general/v0/general"
    echo "[$(date)] INFO Service ready to accept requests"
    
    if [[ "$follow" == "true" ]]; then
        echo "[$(date)] INFO Monitoring logs... (Press Ctrl+C to stop)"
        while true; do
            sleep 5
            echo "[$(date)] DEBUG Heartbeat - service healthy"
        done
    fi
}

mock::unstructured_io::action_test() {
    echo "🧪 Testing Unstructured.io API..."
    echo "================================="
    
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[running]}" == "false" ]]; then
        echo "❌ Service is not running"
        return 1
    fi
    
    echo "✅ Health check: OK"
    echo "✅ API endpoint: Responding"
    echo "✅ Document processing: Available"
    echo "✅ All tests passed"
    
    return 0
}

mock::unstructured_io::action_extract_tables() {
    local file="$1"
    
    if [[ -z "$file" ]]; then
        echo "Error: No file provided for table extraction" >&2
        return 1
    fi
    
    echo "📊 Extracting tables from: $(basename "$file")"
    
    # Generate mock table data
    local table_data
    table_data=$(cat <<'EOF'
{
  "tables": [
    {
      "table_id": "table_1",
      "rows": 3,
      "columns": 2,
      "data": [
        ["Header 1", "Header 2"],
        ["Row 1 Col 1", "Row 1 Col 2"],
        ["Row 2 Col 1", "Row 2 Col 2"]
      ]
    }
  ],
  "metadata": {
    "filename": "document.pdf",
    "tables_found": 1,
    "extraction_method": "mock"
  }
}
EOF
)
    
    echo "$table_data"
    
    # Store extraction data
    UNSTRUCTURED_IO_MOCK_EXTRACTION_DATA["$file|tables"]="$table_data"
    
    return 0
}

mock::unstructured_io::action_extract_metadata() {
    local file="$1"
    
    if [[ -z "$file" ]]; then
        echo "Error: No file provided for metadata extraction" >&2
        return 1
    fi
    
    echo "📋 Extracting metadata from: $(basename "$file")"
    
    # Generate mock metadata
    local metadata
    metadata=$(cat <<EOF
{
  "metadata": {
    "filename": "$(basename "$file")",
    "file_size": 1024576,
    "created_date": "$(date -Iseconds)",
    "modified_date": "$(date -Iseconds)",
    "page_count": 5,
    "word_count": 1250,
    "author": "Mock Author",
    "title": "Mock Document Title",
    "subject": "Mock Subject",
    "language": "en"
  }
}
EOF
)
    
    echo "$metadata"
    
    # Store extraction data
    UNSTRUCTURED_IO_MOCK_EXTRACTION_DATA["$file|metadata"]="$metadata"
    
    return 0
}

mock::unstructured_io::action_process_directory() {
    local directory=""
    local strategy="${UNSTRUCTURED_IO_MOCK_CONFIG[default_strategy]}"
    local output="json"
    local recursive="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --directory)
                directory="$2"
                shift 2
                ;;
            --strategy)
                strategy="$2"
                shift 2
                ;;
            --output)
                output="$2"
                shift 2
                ;;
            --recursive)
                recursive="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$directory" ]]; then
        echo "Error: No directory provided" >&2
        return 1
    fi
    
    echo "📁 Processing documents in directory: $directory"
    echo "🎯 Strategy: $strategy"
    echo "📊 Output format: $output"
    echo "🔄 Recursive: $recursive"
    echo ""
    
    # Simulate finding and processing files
    local processed=0
    local formats=(pdf docx txt html)
    
    for format in "${formats[@]}"; do
        echo "📄 Processing mock.$format"
        processed=$((processed + 1))
    done
    
    echo ""
    echo "✅ Processed $processed documents"
    echo "📊 Results saved to: ${directory}/unstructured_results/"
    
    return 0
}

mock::unstructured_io::action_create_report() {
    local file=""
    local report_file="processing_report.html"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --report-file)
                report_file="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        echo "Error: No file provided for report generation" >&2
        return 1
    fi
    
    echo "📊 Creating processing report for: $(basename "$file")"
    echo "📝 Report file: $report_file"
    
    # Generate mock report
    local report_content
    report_content=$(cat <<EOF
<!DOCTYPE html>
<html>
<head><title>Processing Report</title></head>
<body>
<h1>Document Processing Report</h1>
<p><strong>File:</strong> $(basename "$file")</p>
<p><strong>Generated:</strong> $(date)</p>
<p><strong>Status:</strong> Successfully processed</p>
<p><strong>Elements extracted:</strong> 25</p>
<p><strong>Tables found:</strong> 2</p>
<p><strong>Images found:</strong> 1</p>
</body>
</html>
EOF
)
    
    echo "✅ Report created: $report_file"
    
    # Store report data
    UNSTRUCTURED_IO_MOCK_EXTRACTION_DATA["$file|report"]="$report_content"
    
    return 0
}

mock::unstructured_io::action_cache_stats() {
    echo "📊 Cache Statistics"
    echo "=================="
    echo "Cache enabled: ${UNSTRUCTURED_IO_MOCK_CONFIG[cache_enabled]}"
    echo "Cached items: ${#UNSTRUCTURED_IO_MOCK_CACHE[@]}"
    echo "Processed documents: ${#UNSTRUCTURED_IO_MOCK_PROCESSED_DOCS[@]}"
    
    if [[ ${#UNSTRUCTURED_IO_MOCK_CACHE[@]} -gt 0 ]]; then
        echo ""
        echo "Cache contents:"
        for key in "${!UNSTRUCTURED_IO_MOCK_CACHE[@]}"; do
            local file=$(echo "$key" | cut -d'|' -f1)
            local strategy=$(echo "$key" | cut -d'|' -f2)
            echo "  - $(basename "$file") (strategy: $strategy)"
        done
    fi
}

mock::unstructured_io::action_clear_cache() {
    local file=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -n "$file" ]]; then
        # Clear cache for specific file
        local keys_to_remove=()
        for key in "${!UNSTRUCTURED_IO_MOCK_CACHE[@]}"; do
            if [[ "$key" == "$file|"* ]]; then
                keys_to_remove+=("$key")
            fi
        done
        
        for key in "${keys_to_remove[@]}"; do
            unset "UNSTRUCTURED_IO_MOCK_CACHE[$key]"
        done
        
        echo "✅ Cleared cache for: $(basename "$file")"
    else
        # Clear all cache
        declare -gA UNSTRUCTURED_IO_MOCK_CACHE=()
        declare -gA UNSTRUCTURED_IO_MOCK_PROCESSED_DOCS=()
        declare -gA UNSTRUCTURED_IO_MOCK_EXTRACTION_DATA=()
        
        echo "✅ Cleared all cache"
    fi
}

mock::unstructured_io::action_validate_installation() {
    echo "🔍 Validating Unstructured.io installation..."
    echo "=============================================="
    
    local errors=0
    
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[installed]}" == "false" ]]; then
        echo "❌ Unstructured.io is not installed"
        errors=$((errors + 1))
    else
        echo "✅ Installation: OK"
    fi
    
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[running]}" == "false" ]]; then
        echo "❌ Service is not running"
        errors=$((errors + 1))
    else
        echo "✅ Service status: OK"
    fi
    
    echo "✅ API endpoint: Available"
    echo "✅ Docker connectivity: OK"
    echo "✅ Port availability: OK"
    
    if [[ $errors -eq 0 ]]; then
        echo ""
        echo "🎉 All validation checks passed!"
        return 0
    else
        echo ""
        echo "⚠️  Found $errors issue(s)"
        return 1
    fi
}

# Helper functions for generating mock responses
mock::unstructured_io::generate_json_result() {
    local file="$1"
    local strategy="$2"
    local filename=$(basename "$file")
    
    cat <<EOF
{
  "elements": [
    {
      "type": "Title",
      "element_id": "title_1",
      "text": "Document Title",
      "metadata": {
        "page_number": 1,
        "filename": "$filename"
      }
    },
    {
      "type": "NarrativeText",
      "element_id": "narrative_1",
      "text": "This is sample extracted text from the document. The content has been processed using the $strategy strategy.",
      "metadata": {
        "page_number": 1,
        "filename": "$filename"
      }
    },
    {
      "type": "Table",
      "element_id": "table_1",
      "text": "Sample Table Data | Column 2\\nRow 1 | Value 1\\nRow 2 | Value 2",
      "metadata": {
        "page_number": 2,
        "filename": "$filename"
      }
    }
  ],
  "metadata": {
    "filename": "$filename",
    "file_path": "$file",
    "processing_strategy": "$strategy",
    "total_elements": 3,
    "pages_processed": 2,
    "processing_time": "1.23s"
  }
}
EOF
}

mock::unstructured_io::generate_text_result() {
    local file="$1"
    local strategy="$2"
    
    cat <<EOF
Document Title

This is sample extracted text from the document. The content has been processed using the $strategy strategy.

Sample Table Data | Column 2
Row 1 | Value 1
Row 2 | Value 2

Additional text content extracted from $(basename "$file").
EOF
}

# Test helper functions
mock::unstructured_io::reset() {
    # Optional parameter to control whether to save state after reset
    local save_state="${1:-true}"
    
    # Clear all data
    declare -gA UNSTRUCTURED_IO_MOCK_CONFIG=(
        [installed]="false"
        [running]="false"
        [container_id]="unstructured-io-mock-container"
        [host]="localhost"
        [port]="8000"
        [version]="0.10.30"
        [api_url]="http://localhost:8000"
        [health_status]="healthy"
        [error_mode]=""
        [cache_enabled]="true"
        [default_strategy]="auto"
        [supported_formats]="pdf,docx,txt,html,md,pptx,xlsx"
    )
    declare -gA UNSTRUCTURED_IO_MOCK_PROCESSED_DOCS=()
    declare -gA UNSTRUCTURED_IO_MOCK_CACHE=()
    declare -gA UNSTRUCTURED_IO_MOCK_EXTRACTION_DATA=()
    declare -gA UNSTRUCTURED_IO_MOCK_ERRORS=()
    
    # Save the reset state to file if requested (default: true)
    if [[ "$save_state" == "true" ]]; then
        mock::unstructured_io::save_state
    fi
    
    mock::log_state "unstructured_io" "Unstructured.io mock reset to initial state"
}

mock::unstructured_io::set_error() {
    local error_mode="$1"
    UNSTRUCTURED_IO_MOCK_CONFIG[error_mode]="$error_mode"
    mock::unstructured_io::save_state
    mock::log_state "unstructured_io" "Set Unstructured.io error mode: $error_mode"
}

mock::unstructured_io::set_config() {
    local key="$1"
    local value="$2"
    UNSTRUCTURED_IO_MOCK_CONFIG[$key]="$value"
    
    # Update related config when port changes
    if [[ "$key" == "port" ]]; then
        UNSTRUCTURED_IO_MOCK_CONFIG[api_url]="http://${UNSTRUCTURED_IO_MOCK_CONFIG[host]}:$value"
    fi
    
    mock::unstructured_io::save_state
    mock::log_state "unstructured_io" "Set Unstructured.io config: $key=$value"
}

mock::unstructured_io::set_installed() {
    local installed="$1"
    UNSTRUCTURED_IO_MOCK_CONFIG[installed]="$installed"
    mock::unstructured_io::save_state
    mock::log_state "unstructured_io" "Set Unstructured.io installed: $installed"
}

mock::unstructured_io::set_running() {
    local running="$1"
    UNSTRUCTURED_IO_MOCK_CONFIG[running]="$running"
    mock::unstructured_io::save_state
    mock::log_state "unstructured_io" "Set Unstructured.io running: $running"
}

# Test assertions
mock::unstructured_io::assert_installed() {
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[installed]}" != "true" ]]; then
        echo "Assertion failed: Unstructured.io is not installed" >&2
        return 1
    fi
    return 0
}

mock::unstructured_io::assert_running() {
    if [[ "${UNSTRUCTURED_IO_MOCK_CONFIG[running]}" != "true" ]]; then
        echo "Assertion failed: Unstructured.io is not running" >&2
        return 1
    fi
    return 0
}

mock::unstructured_io::assert_processed() {
    local file="$1"
    if [[ -z "${UNSTRUCTURED_IO_MOCK_PROCESSED_DOCS[$file]}" ]]; then
        echo "Assertion failed: File '$file' has not been processed" >&2
        return 1
    fi
    return 0
}

mock::unstructured_io::assert_cached() {
    local file="$1"
    local cache_found=false
    
    for key in "${!UNSTRUCTURED_IO_MOCK_CACHE[@]}"; do
        if [[ "$key" == "$file|"* ]]; then
            cache_found=true
            break
        fi
    done
    
    if [[ "$cache_found" == "false" ]]; then
        echo "Assertion failed: File '$file' is not cached" >&2
        return 1
    fi
    return 0
}

# Debug functions
mock::unstructured_io::dump_state() {
    echo "=== Unstructured.io Mock State ==="
    echo "Configuration:"
    for key in "${!UNSTRUCTURED_IO_MOCK_CONFIG[@]}"; do
        echo "  $key: ${UNSTRUCTURED_IO_MOCK_CONFIG[$key]}"
    done
    
    echo "Processed Documents:"
    for key in "${!UNSTRUCTURED_IO_MOCK_PROCESSED_DOCS[@]}"; do
        echo "  $key: [${#UNSTRUCTURED_IO_MOCK_PROCESSED_DOCS[$key]} chars]"
    done
    
    echo "Cache:"
    for key in "${!UNSTRUCTURED_IO_MOCK_CACHE[@]}"; do
        echo "  $key: [${#UNSTRUCTURED_IO_MOCK_CACHE[$key]} chars]"
    done
    
    echo "Extraction Data:"
    for key in "${!UNSTRUCTURED_IO_MOCK_EXTRACTION_DATA[@]}"; do
        echo "  $key: [${#UNSTRUCTURED_IO_MOCK_EXTRACTION_DATA[$key]} chars]"
    done
    
    echo "Errors:"
    for key in "${!UNSTRUCTURED_IO_MOCK_ERRORS[@]}"; do
        echo "  $key: ${UNSTRUCTURED_IO_MOCK_ERRORS[$key]}"
    done
    echo "=================================="
}

# Export all functions
export -f unstructured-io-manage
export -f mock::unstructured_io::save_state
export -f mock::unstructured_io::load_state
export -f mock::unstructured_io::action_install
export -f mock::unstructured_io::action_uninstall
export -f mock::unstructured_io::action_start
export -f mock::unstructured_io::action_stop
export -f mock::unstructured_io::action_restart
export -f mock::unstructured_io::action_status
export -f mock::unstructured_io::action_process
export -f mock::unstructured_io::action_info
export -f mock::unstructured_io::action_logs
export -f mock::unstructured_io::action_test
export -f mock::unstructured_io::action_extract_tables
export -f mock::unstructured_io::action_extract_metadata
export -f mock::unstructured_io::action_process_directory
export -f mock::unstructured_io::action_create_report
export -f mock::unstructured_io::action_cache_stats
export -f mock::unstructured_io::action_clear_cache
export -f mock::unstructured_io::action_validate_installation
export -f mock::unstructured_io::generate_json_result
export -f mock::unstructured_io::generate_text_result
export -f mock::unstructured_io::reset
export -f mock::unstructured_io::set_error
export -f mock::unstructured_io::set_config
export -f mock::unstructured_io::set_installed
export -f mock::unstructured_io::set_running
export -f mock::unstructured_io::assert_installed
export -f mock::unstructured_io::assert_running
export -f mock::unstructured_io::assert_processed
export -f mock::unstructured_io::assert_cached
export -f mock::unstructured_io::dump_state

# Save initial state
mock::unstructured_io::save_state