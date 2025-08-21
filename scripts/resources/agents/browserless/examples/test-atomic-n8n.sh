#!/usr/bin/env bash

#######################################
# Test Script: Atomic N8n Workflow Execution
# 
# Demonstrates the new atomic operations approach
# for executing n8n workflows without compilation.
#
# This is a proof of concept showing how simple
# and debuggable the workflow execution becomes.
#######################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BROWSERLESS_DIR="$(dirname "$SCRIPT_DIR")"

# Source log utilities first
source "/home/matthalloran8/Vrooli/scripts/lib/utils/log.sh"

# Source required libraries
source "${BROWSERLESS_DIR}/lib/common.sh"
source "${BROWSERLESS_DIR}/lib/browser-ops.sh"
source "${BROWSERLESS_DIR}/lib/session-manager.sh"

#######################################
# Execute N8n Workflow using Atomic Operations
# Arguments:
#   $1 - Workflow ID
#   $2 - N8n URL (optional, default: http://localhost:5678)
# Returns:
#   0 on success, 1 on failure
#######################################
execute_n8n_workflow_atomic() {
    local workflow_id="${1:?Workflow ID required}"
    local n8n_url="${2:-http://localhost:5678}"
    
    # Create a unique session for this workflow
    local session_id="n8n_${workflow_id}_$$"
    
    log::header "🚀 Executing N8n Workflow (Atomic Operations)"
    log::info "Workflow ID: $workflow_id"
    log::info "Session ID: $session_id"
    log::info "N8n URL: $n8n_url"
    
    # Create browser session
    log::info "Step 1: Creating browser session"
    session::create "$session_id"
    
    # Navigate to workflow
    log::info "Step 2: Navigating to workflow"
    local workflow_url="${n8n_url}/workflow/${workflow_id}"
    if ! browser::navigate "$workflow_url" "$session_id"; then
        log::error "Failed to navigate to workflow"
        session::destroy "$session_id"
        return 1
    fi
    
    # Check current URL to see if we need to login
    log::info "Step 3: Checking if login is required"
    local current_url=$(browser::get_url "$session_id")
    log::debug "Current URL: $current_url"
    
    if [[ "$current_url" =~ "signin" ]]; then
        log::info "Step 4: Handling authentication"
        
        # Wait for email field
        if ! browser::wait_for_element 'input[type="email"], #email' 5000 "$session_id"; then
            log::error "Login form not found"
            browser::screenshot "error_no_login_form.png" "$session_id"
            session::destroy "$session_id"
            return 1
        fi
        
        # Fill email
        log::debug "Filling email field"
        browser::fill 'input[type="email"], #email' "${N8N_EMAIL:-matthalloran8@gmail.com}" "$session_id"
        
        # Fill password
        log::debug "Filling password field"
        browser::fill 'input[type="password"], #password' "${N8N_PASSWORD:-fallback}" "$session_id"
        
        # Take screenshot before login
        browser::screenshot "debug_before_login.png" "$session_id"
        
        # Click submit button
        log::debug "Clicking submit button"
        browser::click 'button[type="submit"]' "$session_id"
        
        # Wait for navigation
        log::debug "Waiting for login to complete"
        browser::wait 3000 "$session_id"
        
        # Check if we're now on the workflow page
        current_url=$(browser::get_url "$session_id")
        log::debug "URL after login: $current_url"
    else
        log::info "Step 4: No login required, already authenticated"
    fi
    
    # Check if workflow canvas is present
    log::info "Step 5: Checking for workflow canvas"
    local has_canvas=$(browser::element_exists '.workflow-canvas, .node-view-wrapper, [data-test-id="canvas"]' "$session_id")
    
    if [[ "$has_canvas" != "true" ]]; then
        log::error "Workflow canvas not found"
        browser::screenshot "error_no_canvas.png" "$session_id"
        
        # Get page title for debugging
        local page_title=$(browser::get_title "$session_id")
        log::error "Page title: $page_title"
        
        session::destroy "$session_id"
        return 1
    fi
    
    log::success "Workflow canvas found"
    browser::screenshot "workflow_loaded.png" "$session_id"
    
    # Look for execute button
    log::info "Step 6: Looking for execute button"
    local execute_selectors=(
        '[title*="Execute"]'
        '.execute-workflow-button'
        '[data-test-id="execute-workflow-button"]'
        'button:has-text("Execute")'
    )
    
    local execute_button_found=false
    for selector in "${execute_selectors[@]}"; do
        if [[ $(browser::element_exists "$selector" "$session_id") == "true" ]]; then
            log::success "Execute button found: $selector"
            execute_button_found=true
            
            # Click execute button
            log::info "Step 7: Executing workflow"
            browser::click "$selector" "$session_id"
            browser::screenshot "workflow_executing.png" "$session_id"
            break
        fi
    done
    
    if [[ "$execute_button_found" == "false" ]]; then
        log::error "Execute button not found"
        browser::screenshot "error_no_execute_button.png" "$session_id"
        session::destroy "$session_id"
        return 1
    fi
    
    # Wait for execution to complete
    log::info "Step 8: Waiting for execution to complete"
    local max_wait=60
    local elapsed=0
    local execution_complete=false
    
    while [[ $elapsed -lt $max_wait ]]; do
        # Check for success indicators
        if [[ $(browser::element_exists '.execution-success, [data-test-id="execution-success"], .success-box' "$session_id") == "true" ]]; then
            log::success "Workflow execution successful"
            browser::screenshot "workflow_success.png" "$session_id"
            execution_complete=true
            break
        fi
        
        # Check for error indicators
        if [[ $(browser::element_exists '.execution-error, [data-test-id="execution-error"], .error-box' "$session_id") == "true" ]]; then
            log::error "Workflow execution failed"
            browser::screenshot "workflow_error.png" "$session_id"
            
            # Try to get error message
            local error_text=$(browser::get_text '.error-message, .execution-error' "$session_id")
            if [[ -n "$error_text" ]]; then
                log::error "Error: $error_text"
            fi
            
            execution_complete=true
            session::destroy "$session_id"
            return 1
        fi
        
        # Still running
        log::debug "Execution in progress... (${elapsed}s elapsed)"
        sleep 2
        elapsed=$((elapsed + 2))
    done
    
    if [[ "$execution_complete" == "false" ]]; then
        log::error "Workflow execution timed out after ${max_wait} seconds"
        browser::screenshot "workflow_timeout.png" "$session_id"
        session::destroy "$session_id"
        return 1
    fi
    
    # Clean up session
    log::info "Step 9: Cleaning up"
    session::destroy "$session_id"
    
    log::success "✅ Workflow execution completed successfully"
    return 0
}

#######################################
# Main execution
#######################################
main() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        echo "Usage: $0 <workflow-id>"
        echo ""
        echo "Example: $0 nZwMYTAQAYUATglq"
        echo ""
        echo "This script demonstrates the new atomic operations approach"
        echo "for executing n8n workflows. Each step is clearly visible"
        echo "and can be debugged independently."
        exit 1
    fi
    
    # Clean up old sessions first
    session::cleanup 30
    
    # Execute the workflow
    execute_n8n_workflow_atomic "$workflow_id"
    local exit_code=$?
    
    # Show summary
    echo ""
    log::header "📊 Execution Summary"
    log::info "Workflow ID: $workflow_id"
    log::info "Exit Code: $exit_code"
    
    if [[ $exit_code -eq 0 ]]; then
        log::success "Workflow executed successfully"
    else
        log::error "Workflow execution failed"
        log::info "Check the screenshot files for debugging:"
        ls -la *.png 2>/dev/null || true
    fi
    
    return $exit_code
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi