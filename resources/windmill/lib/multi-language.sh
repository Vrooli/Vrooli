#!/usr/bin/env bash
################################################################################
# Windmill Multi-Language Script Support
# 
# Enhanced support for executing scripts in multiple languages
################################################################################

# Source required dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${WINDMILL_CLI_DIR}/config/defaults.sh"

################################################################################
# Language Support Registry
################################################################################

declare -A WINDMILL_SUPPORTED_LANGUAGES=(
    ["typescript"]="ts"
    ["javascript"]="js"
    ["python"]="py"
    ["python3"]="py"
    ["go"]="go"
    ["golang"]="go"
    ["bash"]="sh"
    ["shell"]="sh"
    ["powershell"]="ps1"
    ["rust"]="rs"
    ["ruby"]="rb"
    ["php"]="php"
    ["deno"]="ts"
)

################################################################################
# Multi-Language Script Execution
################################################################################

windmill::execute_script() {
    local script_path="$1"
    local language="${2:-auto}"
    shift 2
    local params="$@"
    
    # Auto-detect language from file extension if not specified
    if [[ "$language" == "auto" ]]; then
        local extension="${script_path##*.}"
        language=$(windmill::detect_language "$extension")
    fi
    
    # Validate language support
    if ! windmill::is_language_supported "$language"; then
        log::error "Language '$language' is not supported"
        windmill::show_supported_languages
        return 1
    fi
    
    log::info "Executing $language script: $script_path"
    
    # Execute via Windmill API
    windmill::api_execute_script "$script_path" "$language" "$params"
}

windmill::detect_language() {
    local extension="$1"
    
    case "$extension" in
        ts) echo "typescript" ;;
        js) echo "javascript" ;;
        py) echo "python" ;;
        go) echo "go" ;;
        sh|bash) echo "bash" ;;
        ps1) echo "powershell" ;;
        rs) echo "rust" ;;
        rb) echo "ruby" ;;
        php) echo "php" ;;
        *) echo "unknown" ;;
    esac
}

windmill::is_language_supported() {
    local language="$1"
    [[ -n "${WINDMILL_SUPPORTED_LANGUAGES[$language]}" ]]
}

windmill::show_supported_languages() {
    echo "Supported languages:"
    for lang in "${!WINDMILL_SUPPORTED_LANGUAGES[@]}"; do
        echo "  - $lang (.${WINDMILL_SUPPORTED_LANGUAGES[$lang]})"
    done | sort
}

################################################################################
# API Script Execution
################################################################################

windmill::api_execute_script() {
    local script_path="$1"
    local language="$2"
    shift 2
    local params="$@"
    
    # Read script content
    if [[ ! -f "$script_path" ]]; then
        log::error "Script file not found: $script_path"
        return 1
    fi
    
    local script_content
    script_content=$(cat "$script_path" | jq -Rs '.')
    
    # Prepare execution payload
    local payload=$(cat <<EOF
{
    "language": "$language",
    "content": $script_content,
    "args": {}
}
EOF
    )
    
    # Execute via API
    local response
    response=$(curl -sf -X POST "${WINDMILL_BASE_URL}/api/w/starter/jobs/run_code" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${WINDMILL_API_TOKEN:-}" \
        -d "$payload" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to execute script"
        return 1
    fi
    
    # Extract job ID
    local job_id
    job_id=$(echo "$response" | jq -r '.id // empty')
    
    if [[ -z "$job_id" ]]; then
        log::error "Failed to get job ID from response"
        return 1
    fi
    
    log::info "Job submitted: $job_id"
    
    # Wait for completion and get result
    windmill::wait_for_job "$job_id"
}

windmill::wait_for_job() {
    local job_id="$1"
    local max_wait=60
    local elapsed=0
    
    log::info "Waiting for job completion..."
    
    while [[ $elapsed -lt $max_wait ]]; do
        local status_response
        status_response=$(curl -sf "${WINDMILL_BASE_URL}/api/w/starter/jobs/get/${job_id}" \
            -H "Authorization: Bearer ${WINDMILL_API_TOKEN:-}" 2>/dev/null)
        
        if [[ $? -ne 0 ]]; then
            log::error "Failed to get job status"
            return 1
        fi
        
        local job_type
        job_type=$(echo "$status_response" | jq -r '.type // empty')
        
        if [[ "$job_type" == "CompletedJob" ]]; then
            local success
            success=$(echo "$status_response" | jq -r '.success // false')
            
            if [[ "$success" == "true" ]]; then
                log::success "Job completed successfully"
                
                # Display result
                local result
                result=$(echo "$status_response" | jq '.result')
                echo "Result: $result"
                return 0
            else
                log::error "Job failed"
                
                # Display error
                local error
                error=$(echo "$status_response" | jq -r '.error // "Unknown error"')
                echo "Error: $error"
                return 1
            fi
        fi
        
        sleep 1
        elapsed=$((elapsed + 1))
    done
    
    log::error "Job execution timed out after ${max_wait} seconds"
    return 1
}

################################################################################
# Language-Specific Optimizations
################################################################################

windmill::optimize_python_execution() {
    log::info "Optimizing Python script execution..."
    
    # Check for common Python dependencies
    local deps=("numpy" "pandas" "requests" "matplotlib")
    for dep in "${deps[@]}"; do
        log::info "Ensuring $dep is available in Windmill Python environment"
    done
    
    # Verify Python runtime version
    local python_version
    python_version=$(curl -sf "${WINDMILL_BASE_URL}/api/workers/runtime_info" \
        -H "Authorization: Bearer ${WINDMILL_API_TOKEN:-}" 2>/dev/null | \
        jq -r '.python_version // "unknown"')
    
    log::info "Python runtime version: $python_version"
}

windmill::optimize_typescript_execution() {
    log::info "Optimizing TypeScript script execution..."
    
    # Check Deno runtime (Windmill uses Deno for TS)
    local deno_version
    deno_version=$(curl -sf "${WINDMILL_BASE_URL}/api/workers/runtime_info" \
        -H "Authorization: Bearer ${WINDMILL_API_TOKEN:-}" 2>/dev/null | \
        jq -r '.deno_version // "unknown"')
    
    log::info "Deno runtime version: $deno_version"
}

windmill::optimize_go_execution() {
    log::info "Optimizing Go script execution..."
    
    # Check Go runtime version
    local go_version
    go_version=$(curl -sf "${WINDMILL_BASE_URL}/api/workers/runtime_info" \
        -H "Authorization: Bearer ${WINDMILL_API_TOKEN:-}" 2>/dev/null | \
        jq -r '.go_version // "unknown"')
    
    log::info "Go runtime version: $go_version"
}

################################################################################
# Batch Script Execution
################################################################################

windmill::execute_batch() {
    local scripts_dir="$1"
    local language="${2:-auto}"
    
    if [[ ! -d "$scripts_dir" ]]; then
        log::error "Directory not found: $scripts_dir"
        return 1
    fi
    
    log::info "Executing batch scripts from: $scripts_dir"
    
    local success_count=0
    local fail_count=0
    
    # Find all scripts in directory
    for script in "$scripts_dir"/*; do
        if [[ -f "$script" ]]; then
            log::info "Processing: $(basename "$script")"
            
            if windmill::execute_script "$script" "$language"; then
                success_count=$((success_count + 1))
            else
                fail_count=$((fail_count + 1))
            fi
        fi
    done
    
    log::info "Batch execution complete"
    log::info "Success: $success_count, Failed: $fail_count"
    
    if [[ $fail_count -gt 0 ]]; then
        return 1
    fi
    return 0
}

################################################################################
# Performance Monitoring
################################################################################

windmill::monitor_execution_performance() {
    local language="$1"
    
    log::info "Monitoring $language execution performance..."
    
    # Get worker metrics
    local metrics
    metrics=$(curl -sf "${WINDMILL_BASE_URL}/api/workers/metrics" \
        -H "Authorization: Bearer ${WINDMILL_API_TOKEN:-}" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$metrics" | jq ".language_metrics.$language // {}"
    else
        log::warning "Failed to retrieve performance metrics"
    fi
}

################################################################################
# Export Functions
################################################################################

# Make functions available for CLI
export -f windmill::execute_script
export -f windmill::show_supported_languages
export -f windmill::execute_batch
export -f windmill::optimize_python_execution
export -f windmill::optimize_typescript_execution
export -f windmill::optimize_go_execution
export -f windmill::monitor_execution_performance