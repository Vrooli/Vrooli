#!/usr/bin/env bash

#######################################
# Persistent Browser Operations Library
# 
# Provides browser operations with session persistence
# using browserless WebSocket connections for maintaining state
# across multiple operations.
#######################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSER_OPS_DIR="${APP_ROOT}/resources/browserless/lib"

# Source log utilities
source "/home/matthalloran8/Vrooli/scripts/lib/utils/log.sh" 2>/dev/null || true

# Default configuration
BROWSERLESS_PORT="${BROWSERLESS_PORT:-4110}"
BROWSERLESS_DATA_DIR="${BROWSERLESS_DATA_DIR:-/home/matthalloran8/.vrooli/browserless}"
BROWSERLESS_SESSIONS_DIR="${BROWSERLESS_DATA_DIR}/sessions"
mkdir -p "$BROWSERLESS_SESSIONS_DIR"

#######################################
# Create a persistent browser session
# Arguments:
#   $1 - Session ID
#   $2 - TTL in milliseconds (optional, default: 300000 = 5 minutes)
# Returns:
#   WebSocket endpoint URL
#######################################
browser::create_persistent_session() {
    local session_id="${1:?Session ID required}"
    local ttl="${2:-300000}"
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    log::debug "Creating persistent browser session: $session_id (TTL: ${ttl}ms)"
    
    # Check if session already exists
    local session_file="${BROWSERLESS_SESSIONS_DIR}/${session_id}.json"
    if [[ -f "$session_file" ]]; then
        local ws_endpoint=$(jq -r '.wsEndpoint' "$session_file" 2>/dev/null || echo "")
        if [[ -n "$ws_endpoint" ]]; then
            log::debug "Reusing existing session: $ws_endpoint"
            echo "$ws_endpoint"
            return 0
        fi
    fi
    
    # Create new browser session and get WebSocket endpoint
    local js_code="
export default async ({ page }) => {
    try {
        // Create CDP session
        const cdpSession = await page.target().createCDPSession();
        
        // Send reconnect command to keep browser alive
        const result = await cdpSession.send('Browserless.reconnect', {
            timeout: $ttl
        });
        
        // Return the WebSocket endpoint
        return {
            success: true,
            wsEndpoint: result.browserWSEndpoint,
            sessionId: '$session_id'
        };
    } catch (error) {
        return {
            success: false,
            error: error.message
        };
    }
};
    "
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data @- \
        "http://localhost:${browserless_port}/chrome/function" <<EOF 2>/dev/null
{
    "code": $(echo "$js_code" | jq -Rs '.'),
    "context": {}
}
EOF
    )
    
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        local ws_endpoint=$(echo "$response" | jq -r '.wsEndpoint')
        
        # Save session info
        cat > "$session_file" <<EOF
{
    "sessionId": "$session_id",
    "wsEndpoint": "$ws_endpoint",
    "ttl": $ttl,
    "created": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
        
        log::success "Created persistent session: $session_id"
        echo "$ws_endpoint"
        return 0
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        log::error "Failed to create persistent session: $error"
        return 1
    fi
}

#######################################
# Execute JavaScript in a persistent session
# Arguments:
#   $1 - JavaScript code
#   $2 - Session ID
# Returns:
#   JSON response
#######################################
browser::execute_in_session() {
    local js_code="${1:?JavaScript code required}"
    local session_id="${2:?Session ID required}"
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    # Get WebSocket endpoint for session
    local session_file="${BROWSERLESS_SESSIONS_DIR}/${session_id}.json"
    if [[ ! -f "$session_file" ]]; then
        log::error "Session not found: $session_id"
        # Try to create new session
        local ws_endpoint
        ws_endpoint=$(browser::create_persistent_session "$session_id")
        if [[ $? -ne 0 ]]; then
            return 1
        fi
    fi
    
    local ws_endpoint=$(jq -r '.wsEndpoint' "$session_file" 2>/dev/null || echo "")
    
    if [[ -z "$ws_endpoint" ]]; then
        log::error "No WebSocket endpoint found for session: $session_id"
        return 1
    fi
    
    # Execute code using WebSocket connection
    # Note: This uses the /chrome/function endpoint with browserWSEndpoint parameter
    local wrapped_code="
export default async ({ browser }) => {
    try {
        // Connect to existing browser using WebSocket
        const puppeteer = await import('puppeteer-core');
        const existingBrowser = await puppeteer.connect({
            browserWSEndpoint: '$ws_endpoint'
        });
        
        // Get the first page or create new one
        const pages = await existingBrowser.pages();
        const page = pages.length > 0 ? pages[0] : await existingBrowser.newPage();
        
        // Execute user code
        const result = await (async () => {
            $js_code
        })();
        
        // Don't close the browser - keep it alive for next operation
        return result;
    } catch (error) {
        return {
            success: false,
            error: error.message,
            stack: error.stack
        };
    }
};
    "
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data @- \
        "http://localhost:${browserless_port}/chrome/function" <<EOF 2>/dev/null
{
    "code": $(echo "$wrapped_code" | jq -Rs '.'),
    "context": {}
}
EOF
    )
    
    echo "$response"
}

#######################################
# Navigate to URL in persistent session
# Arguments:
#   $1 - URL
#   $2 - Session ID
# Returns:
#   0 on success, 1 on failure
#######################################
browser::navigate_persistent() {
    local url="${1:?URL required}"
    local session_id="${2:?Session ID required}"
    
    log::debug "Navigating to $url in session $session_id"
    
    local js_code="
        await page.goto('$url', {
            waitUntil: 'networkidle2',
            timeout: 30000
        });
        return {
            success: true,
            url: page.url(),
            title: await page.title()
        };
    "
    
    local response
    response=$(browser::execute_in_session "$js_code" "$session_id")
    
    local success=$(echo "$response" | jq -r '.success // false')
    if [[ "$success" == "true" ]]; then
        log::success "Navigated to: $(echo "$response" | jq -r '.url')"
        return 0
    else
        log::error "Navigation failed: $(echo "$response" | jq -r '.error // "Unknown error"')"
        return 1
    fi
}

#######################################
# Evaluate JavaScript in persistent session
# Arguments:
#   $1 - JavaScript code to evaluate in page context
#   $2 - Session ID
# Returns:
#   JSON result
#######################################
browser::evaluate_persistent() {
    local eval_code="${1:?JavaScript code required}"
    local session_id="${2:?Session ID required}"
    
    log::debug "Evaluating code in session $session_id"
    
    # Wrap the evaluation code
    local js_code="
        const result = await page.evaluate(() => {
            $eval_code
        });
        return result;
    "
    
    browser::execute_in_session "$js_code" "$session_id"
}

#######################################
# Take screenshot in persistent session
# Arguments:
#   $1 - Output path
#   $2 - Session ID
# Returns:
#   0 on success, 1 on failure
#######################################
browser::screenshot_persistent() {
    local output_path="${1:?Output path required}"
    local session_id="${2:?Session ID required}"
    
    log::debug "Taking screenshot in session $session_id"
    
    # Ensure output directory exists
    mkdir -p "${output_path%/*"
    
    local js_code="
        const screenshot = await page.screenshot({
            fullPage: true,
            type: 'png'
        });
        return {
            success: true,
            data: screenshot.toString('base64')
        };
    "
    
    local response
    response=$(browser::execute_in_session "$js_code" "$session_id")
    
    local success=$(echo "$response" | jq -r '.success // false')
    if [[ "$success" == "true" ]]; then
        # Decode base64 and save to file
        echo "$response" | jq -r '.data' | base64 -d > "$output_path"
        log::success "Screenshot saved to: $output_path"
        return 0
    else
        log::error "Screenshot failed: $(echo "$response" | jq -r '.error // "Unknown error"')"
        return 1
    fi
}

#######################################
# Close persistent session
# Arguments:
#   $1 - Session ID
# Returns:
#   0 on success
#######################################
browser::close_session() {
    local session_id="${1:?Session ID required}"
    
    log::debug "Closing session: $session_id"
    
    # Remove session file
    local session_file="${BROWSERLESS_SESSIONS_DIR}/${session_id}.json"
    if [[ -f "$session_file" ]]; then
        rm -f "$session_file"
        log::success "Session closed: $session_id"
    else
        log::warn "Session not found: $session_id"
    fi
    
    return 0
}

# Export all functions
export -f browser::create_persistent_session
export -f browser::execute_in_session
export -f browser::navigate_persistent
export -f browser::evaluate_persistent
export -f browser::screenshot_persistent
export -f browser::close_session