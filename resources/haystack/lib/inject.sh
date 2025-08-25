#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HAYSTACK_LIB_DIR="${APP_ROOT}/resources/haystack/lib"

# Source dependencies
source "${HAYSTACK_LIB_DIR}/common.sh"
source "${HAYSTACK_LIB_DIR}/../../../../lib/utils/log.sh"

# Inject data into Haystack
haystack::inject() {
    local inject_file="${1:-}"
    
    if [[ -z "${inject_file}" ]]; then
        log::error "No injection file specified"
        return 1
    fi
    
    if [[ ! -f "${inject_file}" ]]; then
        log::error "Injection file not found: ${inject_file}"
        return 1
    fi
    
    # Check if Haystack is running
    if ! haystack::is_running; then
        log::error "Haystack is not running"
        return 1
    fi
    
    local port
    port=$(haystack::get_port)
    
    # Determine file type and handle accordingly
    local filename
    filename=$(basename "${inject_file}")
    local extension="${filename##*.}"
    
    case "${extension}" in
        json)
            # Inject JSON documents
            log::info "Injecting JSON documents from ${inject_file}"
            
            # Check if the JSON has a "documents" field at the top level
            local json_content
            json_content=$(cat "${inject_file}")
            
            # If it doesn't have a documents field, wrap it
            if ! echo "${json_content}" | jq -e '.documents' &>/dev/null; then
                # Check if it's an array
                if echo "${json_content}" | jq -e 'type == "array"' &>/dev/null; then
                    # It's an array, wrap it with documents
                    json_content=$(echo "${json_content}" | jq '{documents: .}')
                else
                    # It's a single document, wrap it in an array and documents
                    json_content=$(echo "${json_content}" | jq '{documents: [.]}')
                fi
            fi
            
            local response
            response=$(echo "${json_content}" | curl -sf -X POST "http://localhost:${port}/index" \
                -H "Content-Type: application/json" \
                -d @-)
            
            if [[ $? -eq 0 ]]; then
                log::success "Successfully injected documents"
                echo "${response}"
            else
                log::error "Failed to inject documents: ${response}"
                return 1
            fi
            ;;
            
        txt|md|text)
            # Upload text file
            log::info "Uploading text file ${inject_file}"
            local response
            response=$(curl -sf -X POST "http://localhost:${port}/upload" \
                -F "file=@${inject_file}" 2>&1)
            
            if [[ $? -eq 0 ]]; then
                log::success "Successfully uploaded file"
                echo "${response}"
            else
                log::error "Failed to upload file: ${response}"
                return 1
            fi
            ;;
            
        py)
            # Execute Python script in Haystack environment
            log::info "Executing Python script ${inject_file}"
            
            # Copy script to Haystack scripts directory
            cp "${inject_file}" "${HAYSTACK_SCRIPTS_DIR}/"
            
            # Execute using Haystack's Python environment
            "${HAYSTACK_VENV_DIR}/bin/python" "${HAYSTACK_SCRIPTS_DIR}/${filename}"
            
            if [[ $? -eq 0 ]]; then
                log::success "Successfully executed Python script"
            else
                log::error "Failed to execute Python script"
                return 1
            fi
            ;;
            
        *)
            log::error "Unsupported file type: ${extension}"
            log::info "Supported types: .json (documents), .txt/.md (text), .py (scripts)"
            return 1
            ;;
    esac
    
    return 0
}

# List injected data
haystack::list_injected() {
    if ! haystack::is_running; then
        log::error "Haystack is not running"
        return 1
    fi
    
    local port
    port=$(haystack::get_port)
    
    log::info "Getting Haystack statistics..."
    curl -sf "http://localhost:${port}/stats" | jq .
}

# Clear all data
haystack::clear_data() {
    if ! haystack::is_running; then
        log::error "Haystack is not running"
        return 1
    fi
    
    local port
    port=$(haystack::get_port)
    
    log::info "Clearing all Haystack data..."
    local response
    response=$(curl -sf -X DELETE "http://localhost:${port}/clear" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "Successfully cleared data"
        echo "${response}"
    else
        log::error "Failed to clear data: ${response}"
        return 1
    fi
}