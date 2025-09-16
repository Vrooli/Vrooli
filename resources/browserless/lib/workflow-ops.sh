#!/usr/bin/env bash
########################################
# Enhanced Workflow Operations Library
# 
# Provides advanced browser operations specifically
# designed for the browser-automation-studio scenario
# including scenario navigation, element extraction,
# and comprehensive testing capabilities.
########################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSER_OPS_DIR="${APP_ROOT}/resources/browserless/lib"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }
source "${BROWSER_OPS_DIR}/browser-ops.sh" || { echo "FATAL: Failed to load browser operations" >&2; exit 1; }
source "${BROWSER_OPS_DIR}/cache.sh" || { echo "FATAL: Failed to load cache module" >&2; exit 1; }

########################################
# Navigate to a scenario by name
# Uses vrooli CLI to find the scenario port
# Arguments:
#   $1 - Scenario name
#   $2 - Port type (UI_PORT or API_PORT, default: UI_PORT)
#   $3 - Session ID (optional)
# Returns:
#   Navigation result
########################################
browser::navigate_to_scenario() {
    local scenario_name="${1:?Scenario name required}"
    local port_type="${2:-UI_PORT}"
    local session_id="${3:-default}"
    
    log::info "Looking up port for scenario: $scenario_name ($port_type)"
    
    # Use vrooli CLI to get the scenario port
    local port
    port=$(vrooli scenario port "$scenario_name" "$port_type" 2>/dev/null || echo "")
    
    if [[ -z "$port" ]]; then
        log::error "Could not find port for scenario $scenario_name"
        return 1
    fi
    
    local url="http://localhost:${port}"
    log::info "Navigating to scenario $scenario_name at $url"
    
    browser::navigate "$url" "$session_id"
}

########################################
# Extract text content from element
# Arguments:
#   $1 - CSS selector
#   $2 - Session ID (optional)
# Returns:
#   Text content of the element
########################################
browser::extract_text() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    
    local js_code="
        const element = await page.\$('${selector}');
        if (!element) {
            return { success: false, error: 'Element not found: ${selector}' };
        }
        const text = await element.evaluate(el => el.textContent);
        return { success: true, result: text };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    # Check if response is valid JSON
    if ! echo "$response" | jq -e '.' > /dev/null 2>&1; then
        log::error "Invalid JSON response from browser::execute_js"
        return 1
    fi
    
    local success
    success=$(echo "$response" | jq -r '.success // false' 2>/dev/null || echo "false")
    
    if [[ "$success" == "true" ]]; then
        echo "$response" | jq -r '.result // empty'
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // "Unknown error"' 2>/dev/null || echo "Unknown error")
        log::error "Failed to extract text: $error_msg"
        return 1
    fi
}

########################################
# Extract attribute value from element
# Arguments:
#   $1 - CSS selector
#   $2 - Attribute name
#   $3 - Session ID (optional)
# Returns:
#   Attribute value
########################################
browser::extract_attribute() {
    local selector="${1:?Selector required}"
    local attribute="${2:?Attribute name required}"
    local session_id="${3:-default}"
    
    local js_code="
        const element = await page.\$('${selector}');
        if (!element) {
            return { success: false, error: 'Element not found: ${selector}' };
        }
        const value = await element.evaluate((el, attr) => el.getAttribute(attr), '${attribute}');
        return { success: true, result: value };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    # Check if response is valid JSON
    if ! echo "$response" | jq -e '.' > /dev/null 2>&1; then
        log::error "Invalid JSON response from browser::execute_js"
        return 1
    fi
    
    local success
    success=$(echo "$response" | jq -r '.success // false' 2>/dev/null || echo "false")
    
    if [[ "$success" == "true" ]]; then
        echo "$response" | jq -r '.result // empty'
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // "Unknown error"' 2>/dev/null || echo "Unknown error")
        log::error "Failed to extract attribute: $error_msg"
        return 1
    fi
}

########################################
# Extract inner HTML from element
# Arguments:
#   $1 - CSS selector
#   $2 - Session ID (optional)
# Returns:
#   Inner HTML of the element
########################################
browser::extract_html() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    
    local js_code="
        const element = await page.\$('${selector}');
        if (!element) {
            return { success: false, error: 'Element not found: ${selector}' };
        }
        const html = await element.evaluate(el => el.innerHTML);
        return { success: true, result: html };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    # Check if response is valid JSON
    if ! echo "$response" | jq -e '.' > /dev/null 2>&1; then
        log::error "Invalid JSON response from browser::execute_js"
        return 1
    fi
    
    local success
    success=$(echo "$response" | jq -r '.success // false' 2>/dev/null || echo "false")
    
    if [[ "$success" == "true" ]]; then
        echo "$response" | jq -r '.result // empty'
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // "Unknown error"' 2>/dev/null || echo "Unknown error")
        log::error "Failed to extract HTML: $error_msg"
        return 1
    fi
}

########################################
# Extract input value from form element
# Arguments:
#   $1 - CSS selector
#   $2 - Session ID (optional)
# Returns:
#   Value of the input element
########################################
browser::extract_input_value() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    
    local js_code="
        const element = await page.\$('${selector}');
        if (!element) {
            return { success: false, error: 'Element not found: ${selector}' };
        }
        const value = await element.evaluate(el => el.value);
        return { success: true, result: value };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    # Check if response is valid JSON
    if ! echo "$response" | jq -e '.' > /dev/null 2>&1; then
        log::error "Invalid JSON response from browser::execute_js"
        return 1
    fi
    
    local success
    success=$(echo "$response" | jq -r '.success // false' 2>/dev/null || echo "false")
    
    if [[ "$success" == "true" ]]; then
        echo "$response" | jq -r '.result // empty'
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // "Unknown error"' 2>/dev/null || echo "Unknown error")
        log::error "Failed to extract input value: $error_msg"
        return 1
    fi
}

########################################
# Take a full page screenshot
# Arguments:
#   $1 - Output path
#   $2 - Session ID (optional)
# Returns:
#   0 on success
########################################
browser::screenshot_full_page() {
    local output_path="${1:?Output path required}"
    local session_id="${2:-default}"
    
    log::info "Taking full page screenshot to: $output_path"
    
    local js_code="
        const screenshotBuffer = await page.screenshot({ 
            fullPage: true,
            type: 'png'
        });
        const base64 = screenshotBuffer.toString('base64');
        return { 
            success: true, 
            screenshot: base64,
            path: '${output_path}'
        };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    # Check if response is valid JSON
    if ! echo "$response" | jq -e '.' > /dev/null 2>&1; then
        log::error "Invalid JSON response from browser::execute_js"
        return 1
    fi
    
    local success
    success=$(echo "$response" | jq -r '.success // false' 2>/dev/null || echo "false")
    
    if [[ "$success" == "true" ]]; then
        # Extract and save the screenshot
        echo "$response" | jq -r '.screenshot' | base64 -d > "$output_path"
        log::info "Full page screenshot saved to: $output_path"
        return 0
    else
        log::error "Failed to take full page screenshot"
        return 1
    fi
}

########################################
# Take a screenshot of specific element
# Arguments:
#   $1 - CSS selector
#   $2 - Output path
#   $3 - Session ID (optional)
# Returns:
#   0 on success
########################################
browser::screenshot_element() {
    local selector="${1:?Selector required}"
    local output_path="${2:?Output path required}"
    local session_id="${3:-default}"
    
    log::info "Taking element screenshot of '$selector' to: $output_path"
    
    local js_code="
        const element = await page.\$('${selector}');
        if (!element) {
            return { success: false, error: 'Element not found: ${selector}' };
        }
        const screenshotBuffer = await element.screenshot({ type: 'png' });
        const base64 = screenshotBuffer.toString('base64');
        return { 
            success: true, 
            screenshot: base64,
            path: '${output_path}'
        };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    # Check if response is valid JSON
    if ! echo "$response" | jq -e '.' > /dev/null 2>&1; then
        log::error "Invalid JSON response from browser::execute_js"
        return 1
    fi
    
    local success
    success=$(echo "$response" | jq -r '.success // false' 2>/dev/null || echo "false")
    
    if [[ "$success" == "true" ]]; then
        # Extract and save the screenshot
        echo "$response" | jq -r '.screenshot' | base64 -d > "$output_path"
        log::info "Element screenshot saved to: $output_path"
        return 0
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // "Unknown error"' 2>/dev/null || echo "Unknown error")
        log::error "Failed to take element screenshot: $error_msg"
        return 1
    fi
}

########################################
# Wait for URL to change/redirect
# Arguments:
#   $1 - Expected URL pattern (can be partial)
#   $2 - Timeout in seconds (default: 10)
#   $3 - Session ID (optional)
# Returns:
#   New URL on success
########################################
browser::wait_for_url_change() {
    local expected_pattern="${1:?Expected URL pattern required}"
    local timeout="${2:-10}"
    local session_id="${3:-default}"
    
    log::info "Waiting for URL to match: $expected_pattern (timeout: ${timeout}s)"
    
    local js_code="
        const timeout = ${timeout} * 1000;
        const startTime = Date.now();
        
        while (Date.now() - startTime < timeout) {
            const currentUrl = page.url();
            if (currentUrl.includes('${expected_pattern}')) {
                return { success: true, url: currentUrl };
            }
            await new Promise(resolve => setTimeout(resolve, 500));
        }
        
        return { 
            success: false, 
            error: 'Timeout waiting for URL to match: ${expected_pattern}',
            currentUrl: page.url()
        };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    # Check if response is valid JSON
    if ! echo "$response" | jq -e '.' > /dev/null 2>&1; then
        log::error "Invalid JSON response from browser::execute_js"
        return 1
    fi
    
    local success
    success=$(echo "$response" | jq -r '.success // false' 2>/dev/null || echo "false")
    
    if [[ "$success" == "true" ]]; then
        local new_url=$(echo "$response" | jq -r '.url')
        log::info "URL changed to: $new_url"
        echo "$new_url"
        return 0
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // "Unknown error"' 2>/dev/null || echo "Unknown error")
        log::error "URL change timeout: $error_msg"
        return 1
    fi
}

########################################
# Click element and wait for navigation
# Arguments:
#   $1 - CSS selector
#   $2 - Session ID (optional)
# Returns:
#   0 on success
########################################
browser::click_and_wait() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    
    log::info "Clicking '$selector' and waiting for navigation"
    
    local js_code="
        const element = await page.\$('${selector}');
        if (!element) {
            return { success: false, error: 'Element not found: ${selector}' };
        }
        
        // Click and wait for navigation
        await Promise.all([
            page.waitForNavigation({ waitUntil: 'networkidle2', timeout: 30000 }),
            element.click()
        ]);
        
        return { 
            success: true, 
            url: page.url(),
            title: await page.title()
        };"
    
    browser::execute_js "$js_code" "$session_id"
}

########################################
# Type text with realistic delays
# Arguments:
#   $1 - CSS selector
#   $2 - Text to type
#   $3 - Delay between keystrokes in ms (default: 100)
#   $4 - Session ID (optional)
# Returns:
#   0 on success
########################################
browser::type_with_delay() {
    local selector="${1:?Selector required}"
    local text="${2:?Text required}"
    local delay="${3:-100}"
    local session_id="${4:-default}"
    
    log::info "Typing into '$selector' with ${delay}ms delay"
    
    local js_code="
        const element = await page.\$('${selector}');
        if (!element) {
            return { success: false, error: 'Element not found: ${selector}' };
        }
        
        // Clear the field first
        await element.click({ clickCount: 3 });
        await page.keyboard.press('Backspace');
        
        // Type with delay
        await element.type('${text}', { delay: ${delay} });
        
        return { success: true };"
    
    browser::execute_js "$js_code" "$session_id"
}

########################################
# Check if element contains specific text
# Arguments:
#   $1 - CSS selector
#   $2 - Expected text
#   $3 - Session ID (optional)
# Returns:
#   0 if text found, 1 otherwise
########################################
browser::element_contains_text() {
    local selector="${1:?Selector required}"
    local expected_text="${2:?Expected text required}"
    local session_id="${3:-default}"
    
    local js_code="
        const element = await page.\$('${selector}');
        if (!element) {
            return { success: false, error: 'Element not found: ${selector}' };
        }
        
        const text = await element.evaluate(el => el.textContent);
        const contains = text.includes('${expected_text}');
        
        return { 
            success: true, 
            contains: contains,
            actualText: text
        };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    # Check if response is valid JSON
    if ! echo "$response" | jq -e '.' > /dev/null 2>&1; then
        log::error "Invalid JSON response from browser::execute_js"
        return 1
    fi
    
    local success
    success=$(echo "$response" | jq -r '.success // false' 2>/dev/null || echo "false")
    
    if [[ "$success" == "true" ]]; then
        local contains=$(echo "$response" | jq -r '.contains')
        if [[ "$contains" == "true" ]]; then
            return 0
        else
            log::debug "Element text does not contain: $expected_text"
            return 1
        fi
    else
        log::error "Failed to check element text"
        return 1
    fi
}

########################################
# Cached version of extract_text
# Uses result caching for repeated extractions
# Arguments:
#   $1 - Selector
#   $2 - Session ID (optional)
#   $3 - Cache TTL in seconds (optional, default 3600)
# Returns:
#   Extracted text (from cache if available)
########################################
browser::extract_text_cached() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    local ttl="${3:-3600}"
    
    # Get current URL for cache key generation
    local current_url
    current_url=$(browser::get_current_url "$session_id" 2>/dev/null || echo "unknown")
    
    # Generate cache key
    local cache_key
    cache_key=$(cache::generate_key "extract_text" "$current_url" "{\"selector\":\"$selector\"}")
    
    # Try to get from cache
    local cached_result
    if cached_result=$(cache::get "$cache_key"); then
        log::debug "Using cached text extraction for $selector"
        echo "$cached_result"
        return 0
    fi
    
    # Execute extraction
    local result
    if result=$(browser::extract_text "$selector" "$session_id"); then
        # Cache the result
        cache::set "$cache_key" "$result" "$ttl"
        echo "$result"
        return 0
    else
        return 1
    fi
}

########################################
# Cached version of extract_html
# Uses result caching for repeated extractions
# Arguments:
#   $1 - Selector
#   $2 - Session ID (optional)
#   $3 - Cache TTL in seconds (optional, default 3600)
# Returns:
#   Extracted HTML (from cache if available)
########################################
browser::extract_html_cached() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    local ttl="${3:-3600}"
    
    # Get current URL for cache key generation
    local current_url
    current_url=$(browser::get_current_url "$session_id" 2>/dev/null || echo "unknown")
    
    # Generate cache key
    local cache_key
    cache_key=$(cache::generate_key "extract_html" "$current_url" "{\"selector\":\"$selector\"}")
    
    # Try to get from cache
    local cached_result
    if cached_result=$(cache::get "$cache_key"); then
        log::debug "Using cached HTML extraction for $selector"
        echo "$cached_result"
        return 0
    fi
    
    # Execute extraction
    local result
    if result=$(browser::extract_html "$selector" "$session_id"); then
        # Cache the result
        cache::set "$cache_key" "$result" "$ttl"
        echo "$result"
        return 0
    else
        return 1
    fi
}

########################################
# Clear workflow cache
# Useful for forcing fresh extractions
########################################
browser::clear_cache() {
    log::info "Clearing workflow result cache..."
    cache::clear
}

########################################
# Show cache statistics
########################################
browser::cache_stats() {
    cache::stats
}

# Export all functions
export -f browser::navigate_to_scenario
export -f browser::extract_text
export -f browser::extract_attribute
export -f browser::extract_html
export -f browser::extract_input_value
export -f browser::screenshot_full_page
export -f browser::screenshot_element
export -f browser::wait_for_url_change
export -f browser::click_and_wait
export -f browser::type_with_delay
export -f browser::element_contains_text
export -f browser::extract_text_cached
export -f browser::extract_html_cached
export -f browser::clear_cache
export -f browser::cache_stats