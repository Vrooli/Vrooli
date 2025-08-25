#!/usr/bin/env bash
########################################
# Enhanced Browser Operations with Retry and Debug
# 
# Adds retry logic, debug screenshots, and better
# error handling to atomic browser operations.
########################################
set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
ENHANCED_OPS_DIR="${APP_ROOT}/resources/browserless/lib"

# Source base operations
source "${ENHANCED_OPS_DIR}/browser-ops.sh"

# Global debug settings
BROWSER_DEBUG="${BROWSER_DEBUG:-false}"
BROWSER_DEBUG_DIR="${BROWSER_DEBUG_DIR:-/tmp/browserless_debug}"
BROWSER_MAX_RETRIES="${BROWSER_MAX_RETRIES:-3}"
BROWSER_RETRY_DELAY="${BROWSER_RETRY_DELAY:-2}"

########################################
# Initialize debug directory
########################################
browser::init_debug() {
    if [[ "$BROWSER_DEBUG" == "true" ]]; then
        mkdir -p "$BROWSER_DEBUG_DIR"
        log::info "Debug mode enabled. Screenshots will be saved to: $BROWSER_DEBUG_DIR"
    fi
}

########################################
# Take debug screenshot
########################################
browser::debug_screenshot() {
    local name="${1:-debug}"
    local session_id="${2:-default}"
    
    if [[ "$BROWSER_DEBUG" != "true" ]]; then
        return 0
    fi
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local filename="${BROWSER_DEBUG_DIR}/${timestamp}_${name}.png"
    
    browser::screenshot "$filename" "$session_id" || true
    log::debug "Debug screenshot saved: $filename"
}

########################################
# Click with retry logic
########################################
browser::click_with_retry() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    local max_retries="${3:-$BROWSER_MAX_RETRIES}"
    
    log::info "Clicking element with retry: $selector"
    
    for ((attempt=1; attempt<=max_retries; attempt++)); do
        log::debug "Attempt $attempt of $max_retries"
        
        # Take debug screenshot before attempt
        browser::debug_screenshot "before_click_${attempt}" "$session_id"
        
        # Check if element exists first
        if ! browser::element_exists "$selector" "$session_id"; then
            log::warn "Element not found: $selector"
            if [[ $attempt -lt $max_retries ]]; then
                log::debug "Waiting ${BROWSER_RETRY_DELAY}s before retry..."
                sleep "$BROWSER_RETRY_DELAY"
                continue
            fi
        fi
        
        # Try to click
        if browser::click "$selector" "$session_id"; then
            log::success "Successfully clicked: $selector"
            browser::debug_screenshot "after_click_success" "$session_id"
            return 0
        fi
        
        # Click failed
        if [[ $attempt -lt $max_retries ]]; then
            log::warn "Click failed, retrying in ${BROWSER_RETRY_DELAY}s..."
            browser::debug_screenshot "after_click_fail_${attempt}" "$session_id"
            sleep "$BROWSER_RETRY_DELAY"
        fi
    done
    
    log::error "Failed to click after $max_retries attempts: $selector"
    browser::debug_screenshot "click_final_failure" "$session_id"
    return 1
}

########################################
# Fill with retry logic
########################################
browser::fill_with_retry() {
    local selector="${1:?Selector required}"
    local text="${2:?Text required}"
    local session_id="${3:-default}"
    local max_retries="${4:-$BROWSER_MAX_RETRIES}"
    
    log::info "Filling element with retry: $selector"
    
    for ((attempt=1; attempt<=max_retries; attempt++)); do
        log::debug "Attempt $attempt of $max_retries"
        
        # Take debug screenshot before attempt
        browser::debug_screenshot "before_fill_${attempt}" "$session_id"
        
        # Wait for element to be visible
        if ! browser::wait_for_element "$selector" 5000 "$session_id"; then
            log::warn "Element not visible: $selector"
            if [[ $attempt -lt $max_retries ]]; then
                sleep "$BROWSER_RETRY_DELAY"
                continue
            fi
        fi
        
        # Try to fill
        if browser::fill "$selector" "$text" "$session_id"; then
            log::success "Successfully filled: $selector"
            browser::debug_screenshot "after_fill_success" "$session_id"
            return 0
        fi
        
        # Fill failed
        if [[ $attempt -lt $max_retries ]]; then
            log::warn "Fill failed, retrying in ${BROWSER_RETRY_DELAY}s..."
            browser::debug_screenshot "after_fill_fail_${attempt}" "$session_id"
            sleep "$BROWSER_RETRY_DELAY"
        fi
    done
    
    log::error "Failed to fill after $max_retries attempts: $selector"
    browser::debug_screenshot "fill_final_failure" "$session_id"
    return 1
}

########################################
# Navigate with retry and verification
########################################
browser::navigate_with_retry() {
    local url="${1:?URL required}"
    local session_id="${2:-default}"
    local max_retries="${3:-$BROWSER_MAX_RETRIES}"
    
    log::info "Navigating with retry to: $url"
    
    for ((attempt=1; attempt<=max_retries; attempt++)); do
        log::debug "Navigation attempt $attempt of $max_retries"
        
        # Try to navigate
        if browser::navigate "$url" "$session_id"; then
            # Verify we're on the right page
            local current_url
            current_url=$(browser::get_url "$session_id")
            
            # Check if URL matches (allowing for redirects)
            if [[ "$current_url" =~ ^${url} ]] || [[ "$url" =~ ^${current_url} ]]; then
                log::success "Successfully navigated to: $current_url"
                browser::debug_screenshot "navigate_success" "$session_id"
                return 0
            else
                log::warn "Navigation completed but URL mismatch"
                log::debug "Expected: $url"
                log::debug "Got: $current_url"
            fi
        fi
        
        # Navigation failed
        if [[ $attempt -lt $max_retries ]]; then
            log::warn "Navigation failed, retrying in ${BROWSER_RETRY_DELAY}s..."
            browser::debug_screenshot "navigate_fail_${attempt}" "$session_id"
            sleep "$BROWSER_RETRY_DELAY"
        fi
    done
    
    log::error "Failed to navigate after $max_retries attempts: $url"
    browser::debug_screenshot "navigate_final_failure" "$session_id"
    return 1
}

########################################
# Wait for element with enhanced logging
########################################
browser::wait_for_element_enhanced() {
    local selector="${1:?Selector required}"
    local timeout="${2:-10000}"
    local session_id="${3:-default}"
    
    log::info "Waiting for element: $selector (timeout: ${timeout}ms)"
    
    # Take initial screenshot
    browser::debug_screenshot "wait_start" "$session_id"
    
    # Calculate intervals for periodic checks
    local interval=2
    local max_checks=$((timeout / 1000 / interval))
    local checks=0
    
    while [[ $checks -lt $max_checks ]]; do
        if browser::element_exists "$selector" "$session_id"; then
            log::success "Element found: $selector"
            browser::debug_screenshot "wait_found" "$session_id"
            return 0
        fi
        
        checks=$((checks + 1))
        log::debug "Waiting for element... ($((checks * interval))s / $((timeout / 1000))s)"
        
        # Take periodic debug screenshots
        if [[ $((checks % 5)) -eq 0 ]]; then
            browser::debug_screenshot "wait_progress_${checks}" "$session_id"
        fi
        
        sleep "$interval"
    done
    
    log::error "Element not found after ${timeout}ms: $selector"
    browser::debug_screenshot "wait_timeout" "$session_id"
    return 1
}

########################################
# Execute with comprehensive error handling
########################################
browser::execute_safe() {
    local action="${1:?Action required}"
    local session_id="${2:-default}"
    shift 2
    local args=("$@")
    
    log::header "Executing: $action"
    
    # Take pre-action screenshot
    browser::debug_screenshot "pre_${action}" "$session_id"
    
    # Execute the action
    local result=0
    case "$action" in
        click)
            browser::click_with_retry "${args[@]}" "$session_id" || result=$?
            ;;
        fill|type)
            browser::fill_with_retry "${args[@]}" "$session_id" || result=$?
            ;;
        navigate)
            browser::navigate_with_retry "${args[@]}" "$session_id" || result=$?
            ;;
        wait)
            browser::wait_for_element_enhanced "${args[@]}" "$session_id" || result=$?
            ;;
        *)
            log::warn "Unknown action for safe execution: $action"
            browser::"$action" "${args[@]}" "$session_id" || result=$?
            ;;
    esac
    
    # Take post-action screenshot
    browser::debug_screenshot "post_${action}" "$session_id"
    
    if [[ $result -eq 0 ]]; then
        log::success "Action completed: $action"
    else
        log::error "Action failed: $action"
    fi
    
    return $result
}

########################################
# Get page diagnostics for debugging
########################################
browser::get_diagnostics() {
    local session_id="${1:-default}"
    
    log::header "Page Diagnostics"
    
    # Get current URL
    local url
    url=$(browser::get_url "$session_id")
    log::info "Current URL: $url"
    
    # Get page title
    local title
    title=$(browser::get_title "$session_id")
    log::info "Page Title: $title"
    
    # Get console logs
    local console_logs
    console_logs=$(browser::get_console_logs "$session_id")
    if [[ -n "$console_logs" ]]; then
        log::info "Console Logs:"
        echo "$console_logs" | head -20
    fi
    
    # Check for common error indicators
    local error_selectors=(
        ".error"
        ".alert-danger"
        ".error-message"
        "[class*='error']"
        "[id*='error']"
    )
    
    log::info "Checking for error indicators..."
    for selector in "${error_selectors[@]}"; do
        if browser::element_exists "$selector" "$session_id"; then
            log::warn "Found error element: $selector"
            local error_text
            error_text=$(browser::get_text "$selector" "$session_id" 2>/dev/null || echo "")
            if [[ -n "$error_text" ]]; then
                log::error "Error text: $error_text"
            fi
        fi
    done
    
    # Take diagnostic screenshot
    browser::debug_screenshot "diagnostics" "$session_id"
    
    log::info "Diagnostics complete"
}

########################################
# Initialize enhanced operations
########################################
browser::init_debug

# Export enhanced functions
export -f browser::click_with_retry
export -f browser::fill_with_retry
export -f browser::navigate_with_retry
export -f browser::wait_for_element_enhanced
export -f browser::execute_safe
export -f browser::get_diagnostics
export -f browser::debug_screenshot

log::success "Enhanced browser operations loaded"