#!/usr/bin/env bash
########################################
# N8n Workflow Execution - Atomic Operations Version
# 
# Simplified implementation using atomic browser operations
# instead of complex YAML compilation.
#
# This replaces the old compile-and-execute approach with
# direct, debuggable browser automation.
########################################
set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/browserless/adapters/n8n"
N8N_ADAPTER_DIR="$SCRIPT_DIR"
BROWSERLESS_DIR="${APP_ROOT}/resources/browserless"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
source "${BROWSERLESS_DIR}/lib/browser-ops.sh"
source "${BROWSERLESS_DIR}/lib/session-manager.sh"

# Default configuration
N8N_BASE_URL="${N8N_BASE_URL:-http://localhost:5678}"
N8N_EMAIL="${N8N_EMAIL:-}"
N8N_PASSWORD="${N8N_PASSWORD:-}"
N8N_API_KEY="${N8N_API_KEY:-}"

########################################
# Handle n8n login if required
########################################
n8n::handle_login() {
    local session_id="$1"
    
    log::info "Handling n8n login"
    
    # Check if we're on the login page
    local current_url
    current_url=$(browser::get_url "$session_id")
    
    if [[ ! "$current_url" =~ signin ]]; then
        log::debug "Not on login page, skipping login"
        return 0
    fi
    
    # Fill login form
    log::debug "Filling login credentials"
    
    # Wait for email field
    if ! browser::wait_for_element "input[type='email']" 30 "$session_id"; then
        log::error "Login form not found"
        return 1
    fi
    
    # Fill email
    browser::fill "input[type='email']" "$N8N_EMAIL" "$session_id"
    
    # Fill password
    browser::fill "input[type='password']" "$N8N_PASSWORD" "$session_id"
    
    # Click submit button
    browser::click "button[type='submit']" "$session_id"
    
    # Wait for navigation to complete
    log::debug "Waiting for login to complete"
    sleep 3
    
    # Verify login succeeded
    current_url=$(browser::get_url "$session_id")
    if [[ "$current_url" =~ signin ]]; then
        log::error "Login failed - still on signin page"
        return 1
    fi
    
    log::success "Login successful"
    return 0
}

########################################
# Execute n8n workflow using atomic operations
########################################
n8n::execute_workflow() {
    local workflow_id="$1"
    local wait_completion="${2:-true}"
    local timeout="${3:-60}"
    
    log::header "Executing n8n Workflow: $workflow_id"
    
    # Create screenshots directory
    local screenshots_dir="${HOME}/.vrooli/browserless/screenshots/n8n/${workflow_id}/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$screenshots_dir"
    log::info "Screenshots will be saved to: $screenshots_dir"
    
    # Create unique session
    local session_id="n8n_${workflow_id}_$$"
    
    # Create browser session
    log::info "Creating browser session: $session_id"
    if ! session::create "$session_id"; then
        log::error "Failed to create browser session"
        return 1
    fi
    
    # Navigate to workflow and take initial screenshot (combined for persistent context)
    local workflow_url="${N8N_BASE_URL}/workflow/${workflow_id}"
    log::info "Navigating to workflow: $workflow_url"
    browser::navigate_and_screenshot "$workflow_url" "$screenshots_dir/01_initial.png" "$session_id"
    
    # Handle login if needed
    if [[ -n "$N8N_EMAIL" ]] && [[ -n "$N8N_PASSWORD" ]]; then
        n8n::handle_login "$session_id"
        
        # Navigate to workflow again after login
        browser::navigate "$workflow_url" "$session_id"
    fi
    
    # Wait for workflow canvas to load
    log::info "Waiting for workflow canvas to load"
    if ! browser::wait_for_element ".workflow-canvas, .canvas-wrapper, [data-test-id='canvas']" 30 "$session_id"; then
        log::error "Workflow canvas not found"
        browser::screenshot "$screenshots_dir/02_canvas_error.png" "$session_id"
        session::destroy "$session_id"
        return 1
    fi
    
    # Look for execute button with multiple possible selectors
    log::info "Looking for execute button"
    local execute_selectors=(
        "button[title*='Execute']"
        "button[aria-label*='Execute']"
        "[data-test-id='execute-workflow-button']"
        ".execute-workflow-button"
        "button:has-text('Execute')"
        "button.el-button--primary:has-text('Execute')"
    )
    
    local button_found=false
    for selector in "${execute_selectors[@]}"; do
        if browser::element_exists "$selector" "$session_id"; then
            log::debug "Found execute button with selector: $selector"
            button_found=true
            
            # Click the execute button
            log::info "Clicking execute button"
            if browser::click "$selector" "$session_id"; then
                log::success "Workflow execution triggered"
                browser::screenshot "$screenshots_dir/03_executing.png" "$session_id"
                break
            else
                log::warn "Failed to click with selector: $selector"
            fi
        fi
    done
    
    if [[ "$button_found" == "false" ]]; then
        log::error "Execute button not found"
        browser::screenshot "$screenshots_dir/04_no_execute_button.png" "$session_id"
        
        # Get page content for debugging
        local page_text
        page_text=$(browser::evaluate_script "document.body.innerText" "$session_id")
        log::debug "Page content: ${page_text:0:500}..."
        
        session::destroy "$session_id"
        return 1
    fi
    
    # Wait for completion if requested
    if [[ "$wait_completion" == "true" ]]; then
        log::info "Waiting for workflow completion (timeout: ${timeout}s)"
        
        local elapsed=0
        local completed=false
        
        while [[ $elapsed -lt $timeout ]]; do
            # Check for success indicators
            if browser::element_exists ".execution-success, .success-badge, [data-test-id='execution-success']" "$session_id"; then
                log::success "Workflow completed successfully"
                completed=true
                break
            fi
            
            # Check for error indicators
            if browser::element_exists ".execution-error, .error-badge, [data-test-id='execution-error']" "$session_id"; then
                log::error "Workflow execution failed"
                browser::screenshot "$screenshots_dir/05_execution_failed.png" "$session_id"
                session::destroy "$session_id"
                return 1
            fi
            
            # Check if still running
            if browser::element_exists ".execution-running, .running-badge" "$session_id"; then
                log::debug "Workflow still running..."
            fi
            
            sleep 2
            elapsed=$((elapsed + 2))
            
            # Take periodic screenshots
            if [[ $((elapsed % 10)) -eq 0 ]]; then
                browser::screenshot "$screenshots_dir/06_progress_${elapsed}s.png" "$session_id"
            fi
        done
        
        if [[ "$completed" == "false" ]]; then
            log::warn "Workflow execution timed out after ${timeout}s"
            browser::screenshot "$screenshots_dir/07_timeout.png" "$session_id"
        fi
    fi
    
    # Take final screenshot
    browser::screenshot "$screenshots_dir/08_final.png" "$session_id"
    
    # Get execution results if available
    log::info "Checking for execution results"
    local results
    results=$(browser::evaluate_script "
        const resultElements = document.querySelectorAll('.execution-data, .node-output');
        const results = [];
        resultElements.forEach(el => {
            results.push(el.textContent.trim());
        });
        JSON.stringify(results);
    " "$session_id")
    
    if [[ -n "$results" ]] && [[ "$results" != "[]" ]]; then
        log::info "Execution results: $results"
    fi
    
    # Cleanup
    log::info "Cleaning up browser session"
    session::destroy "$session_id"
    
    log::success "Workflow execution completed"
    return 0
}

########################################
# Execute workflow by name (requires API)
########################################
n8n::execute_by_name() {
    local workflow_name="$1"
    shift
    
    log::info "Looking up workflow by name: $workflow_name"
    
    # Use n8n API to find workflow ID
    if [[ -z "$N8N_API_KEY" ]]; then
        log::error "N8N_API_KEY required for workflow lookup"
        return 1
    fi
    
    local workflow_id
    workflow_id=$(curl -s -H "X-N8N-API-KEY: $N8N_API_KEY" \
        "${N8N_BASE_URL}/api/v1/workflows" | \
        jq -r ".data[] | select(.name==\"$workflow_name\") | .id")
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow not found: $workflow_name"
        return 1
    fi
    
    log::info "Found workflow ID: $workflow_id"
    n8n::execute_workflow "$workflow_id" "$@"
}

########################################
# Entry point
########################################
main() {
    local workflow_id=""
    local workflow_name=""
    local wait_completion="true"
    local timeout="60"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id)
                workflow_id="$2"
                shift 2
                ;;
            --name)
                workflow_name="$2"
                shift 2
                ;;
            --no-wait)
                wait_completion="false"
                shift
                ;;
            --timeout)
                timeout="$2"
                shift 2
                ;;
            --email)
                N8N_EMAIL="$2"
                shift 2
                ;;
            --password)
                N8N_PASSWORD="$2"
                shift 2
                ;;
            --api-key)
                N8N_API_KEY="$2"
                shift 2
                ;;
            --url)
                N8N_BASE_URL="$2"
                shift 2
                ;;
            -h|--help)
                cat << EOF
Usage: $0 [OPTIONS]

Execute n8n workflows using atomic browser operations.

Options:
    --id ID             Workflow ID to execute
    --name NAME         Workflow name to execute (requires API key)
    --no-wait           Don't wait for completion
    --timeout SECONDS   Timeout for completion (default: 60)
    --email EMAIL       n8n login email
    --password PASS     n8n login password
    --api-key KEY       n8n API key for workflow lookup
    --url URL           n8n base URL (default: http://localhost:5678)
    -h, --help          Show this help message

Examples:
    # Execute by ID
    $0 --id abc123

    # Execute by name with credentials
    $0 --name "My Workflow" --email user@example.com --password secret --api-key xyz789

    # Execute without waiting
    $0 --id abc123 --no-wait

EOF
                exit 0
                ;;
            *)
                log::error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Validate input
    if [[ -z "$workflow_id" ]] && [[ -z "$workflow_name" ]]; then
        log::error "Either --id or --name must be specified"
        exit 1
    fi
    
    # Execute workflow
    if [[ -n "$workflow_name" ]]; then
        n8n::execute_by_name "$workflow_name" "$wait_completion" "$timeout"
    else
        n8n::execute_workflow "$workflow_id" "$wait_completion" "$timeout"
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi