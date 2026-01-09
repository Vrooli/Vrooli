#!/usr/bin/env bash

#######################################
# Atomic Browser Operations Library
# 
# Provides simple, single-purpose browser operations
# that execute individual JavaScript commands via browserless.
# 
# Each operation:
#   - Does ONE thing
#   - Returns immediately
#   - Can be retried independently
#   - Provides clear error messages
#######################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSER_OPS_DIR="${APP_ROOT}/resources/browserless/lib"

# Source log utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }

# Source common utilities
source "${BROWSER_OPS_DIR}/common.sh"

#######################################
# Execute raw JavaScript in browserless
# This is the core function all others use
# Arguments:
#   $1 - JavaScript code to execute
#   $2 - Session ID (optional, default: "default")
# Returns:
#   Result from JavaScript execution
#######################################
browser::execute_js() {
    local js_code="${1:?JavaScript code required}"
    local session_id="${2:-default}"
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    # First check if browserless is running
    if ! timeout 2 curl -sf "http://localhost:${browserless_port}/pressure" >/dev/null 2>&1; then
        log::error "Browserless is not accessible at port ${browserless_port}"
        echo "❌ Browserless service is not running. Try: vrooli resource browserless develop" >&2
        return 1
    fi
    
    # Create the JavaScript wrapper using ES6 export default format for browserless v2
    local wrapped_code="export default async ({ page, context }) => {
        try {
            // Set viewport to ensure content is visible
            await page.setViewport({ width: 1920, height: 1080 });
            
            // Execute the user code
            const result = await (async () => {
                ${js_code}
            })();
            
            // Return the result if it's an object, otherwise wrap it
            if (typeof result === 'object' && result !== null) {
                return result;
            } else {
                return { success: true, result: result };
            }
        } catch (error) {
            return { 
                success: false, 
                error: error.message,
                stack: error.stack 
            };
        }
    };"
    
    # Execute via browserless v2 API using Content-Type: application/javascript
    local response
    local http_code
    
    # Capture both response body and HTTP status code
    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/javascript" \
        -d "$wrapped_code" \
        "http://localhost:${browserless_port}/chrome/function" 2>/dev/null)
    
    # Extract HTTP code (last line) and response body
    http_code=$(echo "$response" | tail -n 1)
    response=$(echo "$response" | sed '$d')
    
    # Check HTTP status
    if [[ "$http_code" != "200" ]]; then
        log::error "Browserless API returned HTTP $http_code"
        
        # Provide helpful error messages based on status code
        case "$http_code" in
            404)
                echo "❌ Function endpoint not available. This browserless version may not support the /function API." >&2
                echo "💡 Try using the screenshot or content extraction commands instead." >&2
                ;;
            500|502|503)
                echo "❌ Browserless service error. The container may need to be restarted." >&2
                echo "💡 Try: vrooli resource browserless restart" >&2
                ;;
            429)
                echo "❌ Too many requests. Browser pool may be at capacity." >&2
                echo "💡 Wait a moment or increase MAX_CONCURRENT_SESSIONS." >&2
                ;;
            *)
                echo "❌ Unexpected error from browserless (HTTP $http_code)" >&2
                ;;
        esac
        return 1
    fi
    
    # Check for response
    if [[ -n "$response" ]]; then
        # Check if the response indicates an error
        if echo "$response" | jq -e '.success == false' >/dev/null 2>&1; then
            local error_msg=$(echo "$response" | jq -r '.error // "Unknown error"')
            log::error "JavaScript execution failed: $error_msg"
            echo "❌ Browser operation failed: $error_msg" >&2
        fi
        
        # Return the response for caller to handle
        echo "$response"
        return 0
    else
        log::error "Empty response from browserless"
        echo "❌ No response from browserless. The service may be overloaded." >&2
        return 1
    fi
}

#######################################
# Navigate to a URL
# Arguments:
#   $1 - URL to navigate to
#   $2 - Session ID (optional)
#   $3 - Wait until condition (optional, default: "networkidle2")
# Returns:
#   0 on success, 1 on failure
#######################################
browser::navigate() {
    local url="${1:?URL required}"
    local session_id="${2:-default}"
    local wait_until="${3:-networkidle2}"
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    log::debug "Navigating to: $url"
    
    # Use ES6 export default format for browserless v2
    local ready_selector="${BROWSERLESS_READY_SELECTOR:-[data-app-ready=\"true\"]}"
    local wrapped_code="export default async ({ page, context }) => {
        try {
            await page.setViewport({ width: 1920, height: 1080 });
            
            await page.goto('${url}', { 
                waitUntil: '${wait_until}',
                timeout: ${BROWSERLESS_TIMEOUT:-60000} 
            });
            
            // Prefer explicit app-ready marker; fall back to DOM ready
            try {
                await page.waitForSelector('${ready_selector}', { timeout: 2000 });
            } catch (e) {
                await page.waitForFunction(() => document.readyState === 'complete', { timeout: 3000 }).catch(() => {});
            }
            // Small grace period to stabilize layout (avoid long fixed sleeps)
            await new Promise(resolve => setTimeout(resolve, 200));
            
            return { 
                success: true, 
                url: page.url(),
                title: await page.title()
            };
        } catch (error) {
            return { 
                success: false, 
                error: error.message,
                stack: error.stack 
            };
        }
    };"
    
    # Execute directly via browserless API
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/javascript" \
        -d "$wrapped_code" \
        "http://localhost:${browserless_port}/chrome/function" 2>/dev/null)
    
    echo "$response"
}

#######################################
# Click an element
# Arguments:
#   $1 - CSS selector
#   $2 - Session ID (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
browser::click() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    
    log::debug "Clicking: $selector"
    
    # Escape single quotes in selector
    selector="${selector//\'/\\\'}"
    
    local js_code="
        await page.click('${selector}', { timeout: 10000 });
        return { success: true };"
    
    browser::execute_js "$js_code" "$session_id"
}

#######################################
# Type text into an element
# Arguments:
#   $1 - CSS selector
#   $2 - Text to type
#   $3 - Session ID (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
browser::type() {
    local selector="${1:?Selector required}"
    local text="${2:?Text required}"
    local session_id="${3:-default}"
    
    log::debug "Typing into: $selector"
    
    # Escape single quotes in text
    text="${text//\'/\\\'}"
    
    local js_code="
        await page.type('${selector}', '${text}');
        return { success: true };"
    
    browser::execute_js "$js_code" "$session_id"
}

#######################################
# Clear and type text into an element
# Arguments:
#   $1 - CSS selector
#   $2 - Text to type
#   $3 - Session ID (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
browser::fill() {
    local selector="${1:?Selector required}"
    local text="${2:?Text required}"
    local session_id="${3:-default}"
    
    log::debug "Filling: $selector"
    
    # Escape single quotes in both selector and text
    selector="${selector//\'/\\\'}"
    text="${text//\'/\\\'}"
    
    local js_code="
        await page.click('${selector}', { clickCount: 3 });
        await page.type('${selector}', '${text}');
        return { success: true };"
    
    browser::execute_js "$js_code" "$session_id"
}

#######################################
# Take a screenshot
# Arguments:
#   $1 - Output path (optional, returns base64 if not provided)
#   $2 - Session ID (optional)
#   $3 - Full page (optional, default: false)
# Returns:
#   0 on success, 1 on failure
#######################################
browser::screenshot() {
    local output_path="${1:-}"
    local session_id="${2:-default}"
    local full_page="${3:-false}"
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    log::debug "Taking screenshot"
    
    # Use ES6 export default format for browserless v2
    local wrapped_code="export default async ({ page, context }) => {
        try {
            await page.setViewport({ width: 1920, height: 1080 });
            await page.waitForFunction(() => document.readyState === 'complete', { timeout: 5000 }).catch(() => {});
            await new Promise(resolve => setTimeout(resolve, 1000));
            
            const screenshot = await page.screenshot({ 
                encoding: 'base64',
                fullPage: ${full_page}
            });
            return { 
                success: true, 
                screenshot: screenshot,
                timestamp: new Date().toISOString()
            };
        } catch (error) {
            return { 
                success: false, 
                error: error.message,
                stack: error.stack 
            };
        }
    };"
    
    # Execute directly via browserless API
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/javascript" \
        -d "$wrapped_code" \
        "http://localhost:${browserless_port}/chrome/function" 2>/dev/null)
    
    if [[ $? -eq 0 ]] && [[ -n "$output_path" ]]; then
        local success=$(echo "$response" | jq -r '.success // false')
        if [[ "$success" == "true" ]]; then
            # Save screenshot to file
            echo "$response" | jq -r '.screenshot' | base64 -d > "$output_path"
            log::info "📸 Screenshot saved: $output_path" >&2
            
            # Return response without base64 data to avoid cluttering output
            echo "$response" | jq 'del(.screenshot) + {"screenshot_file": "'"$output_path"'"}'
        else
            local error=$(echo "$response" | jq -r '.error // "Unknown error"')
            log::error "Screenshot failed: $error"
            echo "$response"
            return 1
        fi
    else
        # No output path provided, return full response (this should be avoided in normal usage)
        echo "$response"
    fi
}

#######################################
# Get current URL
# Arguments:
#   $1 - Session ID (optional)
# Returns:
#   Current URL
#######################################
browser::get_url() {
    local session_id="${1:-default}"
    
    local js_code="return { success: true, url: page.url() };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.url'
    fi
}

#######################################
# Get page title
# Arguments:
#   $1 - Session ID (optional)
# Returns:
#   Page title
#######################################
browser::get_title() {
    local session_id="${1:-default}"
    
    local js_code="return { success: true, title: await page.title() };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.title'
    fi
}

#######################################
# Check if element exists
# Arguments:
#   $1 - CSS selector
#   $2 - Session ID (optional)
# Returns:
#   "true" or "false"
#######################################
browser::element_exists() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    
    # Escape single quotes in selector
    selector="${selector//\'/\\\'}"
    
    local js_code="
        const element = await page.\$('${selector}');
        return { 
            success: true, 
            exists: element !== null 
        };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.exists'
    else
        echo "false"
    fi
}

#######################################
# Check if element is visible
# Arguments:
#   $1 - CSS selector
#   $2 - Session ID (optional)
# Returns:
#   "true" or "false"
#######################################
browser::element_visible() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    
    local js_code="
        const element = await page.$('${selector}');
        let visible = false;
        if (element) {
            visible = await element.isIntersectingViewport();
        }
        return { 
            success: true, 
            visible: visible 
        };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.visible'
    else
        echo "false"
    fi
}

#######################################
# Wait for element to appear
# Arguments:
#   $1 - CSS selector
#   $2 - Timeout in ms (optional, default: 10000)
#   $3 - Session ID (optional)
# Returns:
#   0 on success, 1 on timeout
#######################################
browser::wait_for_element() {
    local selector="${1:?Selector required}"
    local timeout="${2:-10000}"
    local session_id="${3:-default}"
    
    log::debug "Waiting for element: $selector (timeout: ${timeout}ms)"
    
    # Escape single quotes in selector
    selector="${selector//\'/\\\'}"
    
    local js_code="
        await page.waitForSelector('${selector}', { 
            timeout: ${timeout} 
        });
        return { success: true };"
    
    browser::execute_js "$js_code" "$session_id"
}

#######################################
# Get element text
# Arguments:
#   $1 - CSS selector
#   $2 - Session ID (optional)
# Returns:
#   Element text content
#######################################
browser::get_text() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    
    # Properly escape the selector for JavaScript
    local escaped_selector
    escaped_selector=$(printf '%s' "$selector" | sed "s/'/\\\\'/g")
    
    local js_code="
        const element = await page.\$('$escaped_selector');
        let text = '';
        if (element) {
            text = await element.evaluate(el => el.textContent);
        }
        return { 
            success: true, 
            text: text 
        };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.text'
    fi
}

#######################################
# Get element attribute
# Arguments:
#   $1 - CSS selector
#   $2 - Attribute name
#   $3 - Session ID (optional)
# Returns:
#   Attribute value
#######################################
browser::get_attribute() {
    local selector="${1:?Selector required}"
    local attribute="${2:?Attribute required}"
    local session_id="${3:-default}"
    
    local js_code="
        const element = await page.$('${selector}');
        let value = null;
        if (element) {
            value = await element.evaluate((el, attr) => el.getAttribute(attr), '${attribute}');
        }
        return { 
            success: true, 
            value: value 
        };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.value'
    fi
}

#######################################
# Evaluate custom JavaScript in page context
# Arguments:
#   $1 - JavaScript code to evaluate
#   $2 - Session ID (optional)
# Returns:
#   Result of evaluation
#######################################
browser::evaluate() {
    local script="${1:?Script required}"
    local session_id="${2:-default}"
    
    local js_code="
        const result = await page.evaluate(() => {
            ${script}
        });
        // Handle both JSON strings and objects
        let finalResult = result;
        if (typeof result === 'string') {
            try {
                finalResult = JSON.parse(result);
            } catch (e) {
                // If parsing fails, keep as string
                finalResult = result;
            }
        }
        return { 
            success: true, 
            result: finalResult 
        };"
    
    browser::execute_js "$js_code" "$session_id"
}

#######################################
# Wait for navigation to complete
# Arguments:
#   $1 - Timeout in ms (optional, default: 30000)
#   $2 - Session ID (optional)
# Returns:
#   0 on success, 1 on timeout
#######################################
browser::wait_for_navigation() {
    local timeout="${1:-30000}"
    local session_id="${2:-default}"
    
    log::debug "Waiting for navigation (timeout: ${timeout}ms)"
    
    local js_code="
        await page.waitForNavigation({ 
            waitUntil: 'networkidle2',
            timeout: ${timeout} 
        });
        return { 
            success: true,
            url: page.url()
        };"
    
    browser::execute_js "$js_code" "$session_id"
}

#######################################
# Get console logs from the page
# Arguments:
#   $1 - Session ID (optional)
# Returns:
#   JSON array of console messages
#######################################
browser::get_console_logs() {
    local session_id="${1:-default}"
    
    local js_code="
        // This would need to be set up when creating the page
        // For now, return empty array
        return { 
            success: true, 
            logs: [] 
        };"
    
    local response
    response=$(browser::execute_js "$js_code" "$session_id")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq '.logs'
    else
        echo "[]"
    fi
}

#######################################
# Press a key
# Arguments:
#   $1 - Key to press (e.g., "Enter", "Tab", "Escape")
#   $2 - Session ID (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
browser::press_key() {
    local key="${1:?Key required}"
    local session_id="${2:-default}"
    
    log::debug "Pressing key: $key"
    
    local js_code="
        await page.keyboard.press('${key}');
        return { success: true };"
    
    browser::execute_js "$js_code" "$session_id"
}

#######################################
# Select option from dropdown
# Arguments:
#   $1 - CSS selector
#   $2 - Value to select
#   $3 - Session ID (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
browser::select() {
    local selector="${1:?Selector required}"
    local value="${2:?Value required}"
    local session_id="${3:-default}"
    
    log::debug "Selecting '$value' in: $selector"
    
    local js_code="
        await page.select('${selector}', '${value}');
        return { success: true };"
    
    browser::execute_js "$js_code" "$session_id"
}

#######################################
# Scroll to element
# Arguments:
#   $1 - CSS selector
#   $2 - Session ID (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
browser::scroll_to() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    
    log::debug "Scrolling to: $selector"
    
    local js_code="
        const element = await page.$('${selector}');
        if (element) {
            await element.scrollIntoView();
        }
        return { success: true };"
    
    browser::execute_js "$js_code" "$session_id"
}

#######################################
# Wait for a specific amount of time
# Arguments:
#   $1 - Time in milliseconds
#   $2 - Session ID (optional)
# Returns:
#   0 on success
#######################################
browser::wait() {
    local ms="${1:?Milliseconds required}"
    local session_id="${2:-default}"
    
    log::debug "Waiting ${ms}ms"
    
    local js_code="
        await new Promise(resolve => setTimeout(resolve, ${ms}));
        return { success: true };"
    
    browser::execute_js "$js_code" "$session_id"
}

# Export all functions
export -f browser::execute_js
export -f browser::navigate
export -f browser::click
export -f browser::type
export -f browser::fill
export -f browser::screenshot
export -f browser::get_url
export -f browser::get_title
export -f browser::element_exists
export -f browser::element_visible
export -f browser::wait_for_element
export -f browser::get_text
export -f browser::get_attribute
export -f browser::evaluate
export -f browser::wait_for_navigation
export -f browser::get_console_logs
export -f browser::press_key
export -f browser::select
export -f browser::scroll_to
export -f browser::wait
#######################################
# Combined navigate and screenshot for persistent context
# Arguments:
#   $1 - URL to navigate to
#   $2 - Output path for screenshot
#   $3 - Session ID (optional)
#   $4 - Wait until condition (optional, default: "networkidle2")
#   $5 - Full page (optional, default: false)
# Returns:
#   Navigation result with screenshot saved to file
#######################################
browser::navigate_and_screenshot() {
    local url="${1:?URL required}"
    local output_path="${2:?Output path required}"
    local session_id="${3:-default}"
    local wait_until="${4:-networkidle2}"
    local full_page="${5:-false}"
    local wait_ms="${6:-2000}"
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    log::debug "Navigating to: $url and taking screenshot (wait ${wait_ms}ms)"
    
    # Combined approach to maintain browser context
    local wrapped_code="export default async ({ page, context }) => {
        try {
            await page.setViewport({ width: 1920, height: 1080 });
            
            // Navigate
            await page.goto('${url}', { 
                waitUntil: '${wait_until}',
                timeout: 30000 
            });
            
            // Wait for page to be ready
            console.log('Page loaded at:', new Date().toISOString());
            await page.waitForFunction(() => document.readyState === 'complete', { timeout: 5000 }).catch(() => {});
            
            // Add visible wait indicator to page
            await page.evaluate((waitTime) => {
                const waitDiv = document.createElement('div');
                waitDiv.id = 'browserless-wait';
                waitDiv.style.cssText = 'position:fixed;top:40px;right:5px;background:red;color:white;padding:5px;z-index:99999;';
                waitDiv.textContent = \`Waiting \${waitTime}ms starting at \${new Date().toISOString()}\`;
                document.body.appendChild(waitDiv);
            }, ${wait_ms});
            
            // Simple explicit wait
            await new Promise(resolve => setTimeout(resolve, ${wait_ms}));
            
            // Update wait indicator
            await page.evaluate(() => {
                const waitDiv = document.getElementById('browserless-wait');
                if (waitDiv) {
                    waitDiv.style.background = 'green';
                    waitDiv.textContent = \`Wait completed at \${new Date().toISOString()}\`;
                }
            });
            
            // Check if metrics are loaded
            const metricsLoaded = await page.evaluate(() => {
                const cpuValue = document.querySelector('#cpu-value');
                return {
                    hasElement: !!cpuValue,
                    text: cpuValue ? cpuValue.textContent : 'not found',
                    dataAttribute: document.body.getAttribute('data-metrics-loaded')
                };
            });
            console.log('Metrics status:', JSON.stringify(metricsLoaded));
            
            // Take screenshot
            const screenshot = await page.screenshot({ 
                encoding: 'base64',
                fullPage: ${full_page}
            });
            
            return { 
                success: true, 
                screenshot: screenshot,
                url: page.url(),
                title: await page.title(),
                timestamp: new Date().toISOString()
            };
        } catch (error) {
            return { 
                success: false, 
                error: error.message,
                stack: error.stack 
            };
        }
    };"
    
    # Execute combined operation
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{
            \"code\": $(echo "$wrapped_code" | jq -Rs .),
            \"context\": {
                \"sessionId\": \"$session_id\"
            }
        }" \
        "http://localhost:${browserless_port}/chrome/function" 2>/dev/null)
    
    local success=$(echo "$response" | jq -r '.success // false')
    if [[ "$success" == "true" ]]; then
        # Save screenshot to file
        echo "$response" | jq -r '.screenshot' | base64 -d > "$output_path"
        log::info "📸 Screenshot saved: $output_path" >&2
        
        # Return response without base64 data
        echo "$response" | jq 'del(.screenshot) + {"screenshot_file": "'"$output_path"'"}'
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        log::error "Combined operation failed: $error"
        echo "$response"
        return 1
    fi
}

# Export the new function
export -f browser::navigate_and_screenshot
