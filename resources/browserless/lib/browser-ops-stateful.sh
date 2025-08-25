#!/usr/bin/env bash

#######################################
# Stateful Browser Operations Library
# 
# Provides browser operations using a single browser instance
# that persists state across multiple operations within a workflow.
# This avoids the issue of each operation creating a new browser context.
#######################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
BROWSER_OPS_DIR="${APP_ROOT}/resources/browserless/lib"

# Source log utilities
source "/home/matthalloran8/Vrooli/scripts/lib/utils/log.sh" 2>/dev/null || true

# Default configuration
BROWSERLESS_PORT="${BROWSERLESS_PORT:-4110}"
BROWSERLESS_DATA_DIR="${BROWSERLESS_DATA_DIR:-/home/matthalloran8/.vrooli/browserless}"

# Global variable to store the browser state code
BROWSER_STATE_CODE=""

#######################################
# Initialize a stateful browser session
# This creates a single JavaScript module that maintains state
# Arguments:
#   $1 - Session ID
# Returns:
#   0 on success
#######################################
browser::init_stateful_session() {
    local session_id="${1:?Session ID required}"
    
    log::debug "Initializing stateful browser session: $session_id"
    
    # Create a JavaScript module that maintains browser state
    BROWSER_STATE_CODE="
let _page = null;
let _browser = null;
let _sessionId = '$session_id';
let _navigationPromise = null;

async function ensurePage() {
    if (!_page) {
        _page = page; // Use the page provided by browserless
        await _page.setViewport({ width: 1920, height: 1080 });
    }
    return _page;
}

async function navigate(url) {
    const p = await ensurePage();
    _navigationPromise = p.goto(url, {
        waitUntil: 'networkidle2',
        timeout: 30000
    });
    await _navigationPromise;
    return {
        success: true,
        url: p.url(),
        title: await p.title()
    };
}

async function evaluate(code) {
    const p = await ensurePage();
    
    // Wait for any pending navigation
    if (_navigationPromise) {
        await _navigationPromise.catch(() => {});
        _navigationPromise = null;
    }
    
    // Create a function from the code string and execute it
    const fn = new Function(code);
    const result = await p.evaluate(fn);
    return result;
}

async function screenshot(fullPage = true) {
    const p = await ensurePage();
    
    // Wait for any pending navigation
    if (_navigationPromise) {
        await _navigationPromise.catch(() => {});
        _navigationPromise = null;
    }
    
    const buffer = await p.screenshot({
        fullPage: fullPage,
        type: 'png'
    });
    return {
        success: true,
        data: buffer.toString('base64')
    };
}

async function waitForElement(selector, timeout = 5000) {
    const p = await ensurePage();
    await p.waitForSelector(selector, { timeout });
    return { success: true };
}

async function click(selector) {
    const p = await ensurePage();
    await p.click(selector);
    return { success: true };
}

async function type(selector, text, clearFirst = false) {
    const p = await ensurePage();
    if (clearFirst) {
        await p.click(selector, { clickCount: 3 });
        await p.keyboard.press('Delete');
    }
    await p.type(selector, text);
    return { success: true };
}
"
    
    return 0
}

#######################################
# Execute a stateful browser operation
# Arguments:
#   $1 - Operation name (navigate, evaluate, screenshot, etc.)
#   $2 - Operation parameters (JSON)
#   $3 - Session ID
# Returns:
#   JSON response
#######################################
browser::execute_stateful() {
    local operation="${1:?Operation required}"
    local params="${2:-{}}"
    local session_id="${3:?Session ID required}"
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    # Initialize session if needed
    if [[ -z "$BROWSER_STATE_CODE" ]]; then
        browser::init_stateful_session "$session_id"
    fi
    
    # Build the JavaScript code based on operation
    local js_code=""
    case "$operation" in
        navigate)
            local url=$(echo "$params" | jq -r '.url // ""')
            js_code="
$BROWSER_STATE_CODE

export default async ({ page, context }) => {
    try {
        const result = await navigate('$url');
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
            ;;
            
        evaluate)
            local script=$(echo "$params" | jq -r '.script // ""')
            # Escape the script properly for JavaScript
            local escaped_script=$(echo "$script" | jq -Rs '.')
            js_code="
$BROWSER_STATE_CODE

export default async ({ page, context }) => {
    try {
        const code = $escaped_script;
        const result = await evaluate(code);
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
            ;;
            
        screenshot)
            local fullPage=$(echo "$params" | jq -r '.fullPage // true')
            js_code="
$BROWSER_STATE_CODE

export default async ({ page, context }) => {
    try {
        const result = await screenshot($fullPage);
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
            ;;
            
        *)
            log::error "Unknown operation: $operation"
            return 1
            ;;
    esac
    
    # Execute the code
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data @- \
        "http://localhost:${browserless_port}/chrome/function" <<EOF 2>/dev/null
{
    "code": $(echo "$js_code" | jq -Rs '.'),
    "context": {
        "sessionId": "$session_id"
    }
}
EOF
    )
    
    echo "$response"
}

#######################################
# Navigate to URL using stateful session
# Arguments:
#   $1 - URL
#   $2 - Session ID
# Returns:
#   0 on success, 1 on failure
#######################################
browser::navigate_stateful() {
    local url="${1:?URL required}"
    local session_id="${2:?Session ID required}"
    
    log::debug "Navigating to $url in stateful session $session_id"
    
    local params=$(jq -n --arg url "$url" '{url: $url}')
    local response
    response=$(browser::execute_stateful "navigate" "$params" "$session_id")
    
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
# Evaluate JavaScript using stateful session
# Arguments:
#   $1 - JavaScript code
#   $2 - Session ID
# Returns:
#   JSON result
#######################################
browser::evaluate_stateful() {
    local script="${1:?JavaScript code required}"
    local session_id="${2:?Session ID required}"
    
    log::debug "Evaluating code in stateful session $session_id"
    
    local params=$(jq -n --arg script "$script" '{script: $script}')
    browser::execute_stateful "evaluate" "$params" "$session_id"
}

#######################################
# Take screenshot using stateful session
# Arguments:
#   $1 - Output path
#   $2 - Session ID
# Returns:
#   0 on success, 1 on failure
#######################################
browser::screenshot_stateful() {
    local output_path="${1:?Output path required}"
    local session_id="${2:?Session ID required}"
    
    log::debug "Taking screenshot in stateful session $session_id"
    
    # Ensure output directory exists
    mkdir -p "${output_path%/*"
    
    local params=$(jq -n '{fullPage: true}')
    local response
    response=$(browser::execute_stateful "screenshot" "$params" "$session_id")
    
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

# Export all functions
export -f browser::init_stateful_session
export -f browser::execute_stateful
export -f browser::navigate_stateful
export -f browser::evaluate_stateful
export -f browser::screenshot_stateful