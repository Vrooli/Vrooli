#!/usr/bin/env bash
########################################
# Conditional Operations Library
# 
# Provides advanced conditional logic for workflows
# including URL checks, element states, text matching,
# and complex branching operations.
########################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSER_OPS_DIR="${APP_ROOT}/resources/browserless/lib"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }
source "${BROWSER_OPS_DIR}/browser-ops.sh" || { echo "FATAL: Failed to load browser operations" >&2; exit 1; }

########################################
# Check if current URL matches a pattern
# Arguments:
#   $1 - URL pattern (can include wildcards)
#   $2 - Session ID (optional)
# Returns:
#   0 if matches, 1 otherwise
########################################
condition::url_matches() {
    local pattern="${1:?URL pattern required}"
    local session_id="${2:-default}"
    
    log::debug "Checking if URL matches pattern: $pattern"
    
    local js_code
    js_code=$(cat <<EOF
        const currentUrl = page.url();
        const pattern = '${pattern}';
        
        // Handle different pattern types
        if (pattern.includes('*')) {
            // Convert wildcard to regex
            const regex = new RegExp('^' + pattern
                .replace(/[.+?^${}()|[\\]\\\\]/g, '\\\\$&')
                .replace(/\\*/g, '.*') + '$');
            return regex.test(currentUrl);
        } else if (pattern.startsWith('/') && pattern.endsWith('/')) {
            // Regex pattern
            const regex = new RegExp(pattern.slice(1, -1));
            return regex.test(currentUrl);
        } else {
            // Exact or contains match
            return currentUrl.includes(pattern);
        }
EOF
    )
    
    local result
    result=$(browser::evaluate "$js_code" "$session_id")
    
    if [[ "$result" == "true" ]]; then
        log::debug "URL matches pattern"
        return 0
    else
        log::debug "URL does not match pattern"
        return 1
    fi
}

########################################
# Check if element exists and is visible
# Arguments:
#   $1 - CSS selector
#   $2 - Session ID (optional)
# Returns:
#   0 if visible, 1 otherwise
########################################
condition::element_visible() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    
    log::debug "Checking if element is visible: $selector"
    
    local js_code
    js_code=$(cat <<EOF
        const element = await page.\$('${selector}');
        if (!element) {
            return false;
        }
        
        // Check if element is visible
        const isVisible = await element.evaluate(el => {
            const style = window.getComputedStyle(el);
            return style.display !== 'none' 
                && style.visibility !== 'hidden' 
                && style.opacity !== '0'
                && el.offsetWidth > 0
                && el.offsetHeight > 0;
        });
        
        return isVisible;
EOF
    )
    
    local result
    result=$(browser::evaluate "$js_code" "$session_id")
    
    if [[ "$result" == "true" ]]; then
        log::debug "Element is visible"
        return 0
    else
        log::debug "Element is not visible"
        return 1
    fi
}

########################################
# Check if element contains specific text
# Arguments:
#   $1 - CSS selector
#   $2 - Expected text
#   $3 - Match type (exact|contains|regex)
#   $4 - Session ID (optional)
# Returns:
#   0 if matches, 1 otherwise
########################################
condition::element_text() {
    local selector="${1:?Selector required}"
    local expected="${2:?Expected text required}"
    local match_type="${3:-contains}"
    local session_id="${4:-default}"
    
    log::debug "Checking element text: $selector for '$expected' (type: $match_type)"
    
    local js_code
    js_code=$(cat <<EOF
        const element = await page.\$('${selector}');
        if (!element) {
            return false;
        }
        
        const text = await element.evaluate(el => el.textContent.trim());
        const expected = '${expected}';
        const matchType = '${match_type}';
        
        switch(matchType) {
            case 'exact':
                return text === expected;
            case 'contains':
                return text.includes(expected);
            case 'regex':
                const regex = new RegExp(expected);
                return regex.test(text);
            default:
                return text.includes(expected);
        }
EOF
    )
    
    local result
    result=$(browser::evaluate "$js_code" "$session_id")
    
    if [[ "$result" == "true" ]]; then
        log::debug "Element text matches"
        return 0
    else
        log::debug "Element text does not match"
        return 1
    fi
}

########################################
# Check if checkbox/radio is checked
# Arguments:
#   $1 - CSS selector
#   $2 - Session ID (optional)
# Returns:
#   0 if checked, 1 otherwise
########################################
condition::input_checked() {
    local selector="${1:?Selector required}"
    local session_id="${2:-default}"
    
    log::debug "Checking if input is checked: $selector"
    
    local js_code
    js_code=$(cat <<EOF
        const element = await page.\$('${selector}');
        if (!element) {
            return false;
        }
        
        const isChecked = await element.evaluate(el => {
            return el.checked === true;
        });
        
        return isChecked;
EOF
    )
    
    local result
    result=$(browser::evaluate "$js_code" "$session_id")
    
    if [[ "$result" == "true" ]]; then
        log::debug "Input is checked"
        return 0
    else
        log::debug "Input is not checked"
        return 1
    fi
}

########################################
# Check if element has specific attribute value
# Arguments:
#   $1 - CSS selector
#   $2 - Attribute name
#   $3 - Expected value
#   $4 - Session ID (optional)
# Returns:
#   0 if matches, 1 otherwise
########################################
condition::attribute_equals() {
    local selector="${1:?Selector required}"
    local attribute="${2:?Attribute name required}"
    local expected="${3:?Expected value required}"
    local session_id="${4:-default}"
    
    log::debug "Checking attribute: $selector[$attribute]='$expected'"
    
    local js_code
    js_code=$(cat <<EOF
        const element = await page.\$('${selector}');
        if (!element) {
            return false;
        }
        
        const value = await element.evaluate((el, attr) => {
            return el.getAttribute(attr);
        }, '${attribute}');
        
        return value === '${expected}';
EOF
    )
    
    local result
    result=$(browser::evaluate "$js_code" "$session_id")
    
    if [[ "$result" == "true" ]]; then
        log::debug "Attribute matches"
        return 0
    else
        log::debug "Attribute does not match"
        return 1
    fi
}

########################################
# Check if page contains error messages
# Arguments:
#   $1 - Error patterns (comma-separated)
#   $2 - Session ID (optional)
# Returns:
#   0 if errors found, 1 otherwise
########################################
condition::has_errors() {
    local patterns="${1:-error,Error,failed,Failed,invalid,Invalid}"
    local session_id="${2:-default}"
    
    log::debug "Checking for error messages: $patterns"
    
    local js_code
    js_code=$(cat <<EOF
        const patterns = '${patterns}'.split(',').map(p => p.trim());
        
        // Common error selectors
        const errorSelectors = [
            '.error', '.alert-danger', '.alert-error',
            '[role="alert"]', '.toast-error', '.notification-error',
            '.snackbar.error', '.message.error'
        ];
        
        // Check for error elements
        for (const selector of errorSelectors) {
            const elements = await page.\$\$(selector);
            if (elements.length > 0) {
                return true;
            }
        }
        
        // Check page text for error patterns
        const pageText = await page.evaluate(() => document.body.textContent);
        for (const pattern of patterns) {
            if (pageText.includes(pattern)) {
                return true;
            }
        }
        
        return false;
EOF
    )
    
    local result
    result=$(browser::evaluate "$js_code" "$session_id")
    
    if [[ "$result" == "true" ]]; then
        log::debug "Error messages found"
        return 0
    else
        log::debug "No error messages found"
        return 1
    fi
}

########################################
# Check if page title matches
# Arguments:
#   $1 - Expected title pattern
#   $2 - Match type (exact|contains|regex)
#   $3 - Session ID (optional)
# Returns:
#   0 if matches, 1 otherwise
########################################
condition::title_matches() {
    local pattern="${1:?Title pattern required}"
    local match_type="${2:-contains}"
    local session_id="${3:-default}"
    
    log::debug "Checking page title: '$pattern' (type: $match_type)"
    
    local js_code
    js_code=$(cat <<EOF
        const title = await page.title();
        const pattern = '${pattern}';
        const matchType = '${match_type}';
        
        switch(matchType) {
            case 'exact':
                return title === pattern;
            case 'contains':
                return title.includes(pattern);
            case 'regex':
                const regex = new RegExp(pattern);
                return regex.test(title);
            default:
                return title.includes(pattern);
        }
EOF
    )
    
    local result
    result=$(browser::evaluate "$js_code" "$session_id")
    
    if [[ "$result" == "true" ]]; then
        log::debug "Title matches"
        return 0
    else
        log::debug "Title does not match"
        return 1
    fi
}

########################################
# Check if cookie exists
# Arguments:
#   $1 - Cookie name
#   $2 - Session ID (optional)
# Returns:
#   0 if exists, 1 otherwise
########################################
condition::cookie_exists() {
    local cookie_name="${1:?Cookie name required}"
    local session_id="${2:-default}"
    
    log::debug "Checking for cookie: $cookie_name"
    
    local js_code
    js_code=$(cat <<EOF
        const cookies = await page.cookies();
        const cookieExists = cookies.some(cookie => cookie.name === '${cookie_name}');
        return cookieExists;
EOF
    )
    
    local result
    result=$(browser::evaluate "$js_code" "$session_id")
    
    if [[ "$result" == "true" ]]; then
        log::debug "Cookie exists"
        return 0
    else
        log::debug "Cookie does not exist"
        return 1
    fi
}

########################################
# Check if localStorage item exists
# Arguments:
#   $1 - Storage key
#   $2 - Session ID (optional)
# Returns:
#   0 if exists, 1 otherwise
########################################
condition::storage_exists() {
    local key="${1:?Storage key required}"
    local session_id="${2:-default}"
    
    log::debug "Checking localStorage for key: $key"
    
    local js_code
    js_code=$(cat <<EOF
        const value = await page.evaluate((key) => {
            return localStorage.getItem(key) !== null;
        }, '${key}');
        return value;
EOF
    )
    
    local result
    result=$(browser::evaluate "$js_code" "$session_id")
    
    if [[ "$result" == "true" ]]; then
        log::debug "Storage key exists"
        return 0
    else
        log::debug "Storage key does not exist"
        return 1
    fi
}

########################################
# Complex condition evaluator
# Supports AND, OR, NOT operations
# Arguments:
#   $1 - Condition JSON
#   $2 - Session ID (optional)
# Returns:
#   0 if true, 1 if false
########################################
condition::evaluate_complex() {
    local condition_json="${1:?Condition JSON required}"
    local session_id="${2:-default}"
    
    local operator=$(echo "$condition_json" | jq -r '.operator // "and"')
    local conditions=$(echo "$condition_json" | jq -r '.conditions // []')
    
    case "$operator" in
        "and")
            local all_true=true
            echo "$conditions" | jq -c '.[]' | while read -r cond; do
                if ! condition::evaluate_single "$cond" "$session_id"; then
                    all_true=false
                    break
                fi
            done
            [[ "$all_true" == "true" ]]
            ;;
            
        "or")
            local any_true=false
            echo "$conditions" | jq -c '.[]' | while read -r cond; do
                if condition::evaluate_single "$cond" "$session_id"; then
                    any_true=true
                    break
                fi
            done
            [[ "$any_true" == "true" ]]
            ;;
            
        "not")
            local single_cond=$(echo "$conditions" | jq -c '.[0]')
            ! condition::evaluate_single "$single_cond" "$session_id"
            ;;
            
        *)
            log::error "Unknown operator: $operator"
            return 1
            ;;
    esac
}

########################################
# Evaluate a single condition
# Arguments:
#   $1 - Condition JSON
#   $2 - Session ID (optional)
# Returns:
#   0 if true, 1 if false
########################################
condition::evaluate_single() {
    local condition="${1:?Condition required}"
    local session_id="${2:-default}"
    
    local type=$(echo "$condition" | jq -r '.type')
    
    case "$type" in
        "url")
            local pattern=$(echo "$condition" | jq -r '.pattern')
            condition::url_matches "$pattern" "$session_id"
            ;;
            
        "element_visible")
            local selector=$(echo "$condition" | jq -r '.selector')
            condition::element_visible "$selector" "$session_id"
            ;;
            
        "element_text")
            local selector=$(echo "$condition" | jq -r '.selector')
            local text=$(echo "$condition" | jq -r '.text')
            local match_type=$(echo "$condition" | jq -r '.match_type // "contains"')
            condition::element_text "$selector" "$text" "$match_type" "$session_id"
            ;;
            
        "input_checked")
            local selector=$(echo "$condition" | jq -r '.selector')
            condition::input_checked "$selector" "$session_id"
            ;;
            
        "has_errors")
            local patterns=$(echo "$condition" | jq -r '.patterns // ""')
            condition::has_errors "$patterns" "$session_id"
            ;;
            
        "title")
            local pattern=$(echo "$condition" | jq -r '.pattern')
            local match_type=$(echo "$condition" | jq -r '.match_type // "contains"')
            condition::title_matches "$pattern" "$match_type" "$session_id"
            ;;
            
        "cookie")
            local name=$(echo "$condition" | jq -r '.name')
            condition::cookie_exists "$name" "$session_id"
            ;;
            
        "storage")
            local key=$(echo "$condition" | jq -r '.key')
            condition::storage_exists "$key" "$session_id"
            ;;
            
        *)
            log::error "Unknown condition type: $type"
            return 1
            ;;
    esac
}

# Export all functions
export -f condition::url_matches
export -f condition::element_visible
export -f condition::element_text
export -f condition::input_checked
export -f condition::attribute_equals
export -f condition::has_errors
export -f condition::title_matches
export -f condition::cookie_exists
export -f condition::storage_exists
export -f condition::evaluate_complex
export -f condition::evaluate_single