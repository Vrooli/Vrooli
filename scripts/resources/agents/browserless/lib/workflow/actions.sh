#!/usr/bin/env bash

#######################################
# Browserless Workflow Action Library
# Defines all available workflow actions and their JavaScript implementations
#######################################

# Get script directory
WORKFLOW_ACTIONS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

#######################################
# Get JavaScript implementation for an action
# Arguments:
#   $1 - Action name
#   $2 - Action parameters (JSON)
# Returns:
#   JavaScript code for the action
#######################################
action::get_implementation() {
    local action="${1:?Action name required}"
    local params="${2:-{}}"
    
    case "$action" in
        # Navigation Actions
        "navigate")
            action::impl_navigate "$params"
            ;;
        "reload")
            action::impl_reload "$params"
            ;;
        "go_back")
            action::impl_go_back "$params"
            ;;
        "go_forward")
            action::impl_go_forward "$params"
            ;;
            
        # Interaction Actions
        "click")
            action::impl_click "$params"
            ;;
        "fill_form")
            action::impl_fill_form "$params"
            ;;
        "type")
            action::impl_type "$params"
            ;;
        "select")
            action::impl_select "$params"
            ;;
        "upload_file")
            action::impl_upload_file "$params"
            ;;
        "hover")
            action::impl_hover "$params"
            ;;
        "focus")
            action::impl_focus "$params"
            ;;
        "clear")
            action::impl_clear "$params"
            ;;
            
        # Wait Actions
        "wait")
            action::impl_wait "$params"
            ;;
        "wait_for_element")
            action::impl_wait_for_element "$params"
            ;;
        "wait_for_text")
            action::impl_wait_for_text "$params"
            ;;
        "wait_for_navigation")
            action::impl_wait_for_navigation "$params"
            ;;
        "wait_for_redirect")
            action::impl_wait_for_redirect "$params"
            ;;
        "wait_for_network_idle")
            action::impl_wait_for_network_idle "$params"
            ;;
            
        # Data Actions
        "screenshot")
            action::impl_screenshot "$params"
            ;;
        "extract_text")
            action::impl_extract_text "$params"
            ;;
        "extract_data")
            action::impl_extract_data "$params"
            ;;
            
        # Validation Actions
        "assert_text")
            action::impl_assert_text "$params"
            ;;
        "assert_url")
            action::impl_assert_url "$params"
            ;;
        "assert_element")
            action::impl_assert_element "$params"
            ;;
            
        # Advanced Actions
        "execute_script")
            action::impl_execute_script "$params"
            ;;
        "scroll")
            action::impl_scroll "$params"
            ;;
        "set_viewport")
            action::impl_set_viewport "$params"
            ;;
            
        *)
            log::error "Unknown action: $action"
            echo "throw new Error('Unknown action: $action');"
            ;;
    esac
}

#######################################
# Navigation Action Implementations
#######################################

action::impl_navigate() {
    local params="$1"
    local url=$(echo "$params" | jq -r '.url // ""')
    local wait_for=$(echo "$params" | jq -r '.wait_for // "networkidle2"')
    local timeout=$(echo "$params" | jq -r '.timeout // 30000')
    
    cat <<EOF
    await page.goto('$url', {
        waitUntil: '$wait_for',
        timeout: $timeout
    });
EOF
}

action::impl_reload() {
    local params="$1"
    local wait_for=$(echo "$params" | jq -r '.wait_for // "networkidle2"')
    
    cat <<EOF
    await page.reload({
        waitUntil: '$wait_for'
    });
EOF
}

action::impl_go_back() {
    cat <<EOF
    await page.goBack();
EOF
}

action::impl_go_forward() {
    cat <<EOF
    await page.goForward();
EOF
}

#######################################
# Interaction Action Implementations
#######################################

action::impl_click() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    local wait_nav=$(echo "$params" | jq -r '.wait_for_navigation // false')
    
    if [[ "$wait_nav" == "true" ]]; then
        cat <<EOF
    await Promise.all([
        page.waitForNavigation(),
        page.click('$selector')
    ]);
EOF
    else
        cat <<EOF
    await page.click('$selector');
EOF
    fi
}

action::impl_fill_form() {
    local params="$1"
    local selectors=$(echo "$params" | jq -r '.selectors // {}')
    local values=$(echo "$params" | jq -r '.values // {}')
    
    cat <<EOF
    const selectors = $selectors;
    const values = $values;
    
    for (const [key, selector] of Object.entries(selectors)) {
        const value = values[key];
        if (value !== undefined) {
            await page.type(selector, value, {delay: 50});
        }
    }
EOF
}

action::impl_type() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    local text=$(echo "$params" | jq -r '.text // ""')
    local delay=$(echo "$params" | jq -r '.delay // 50')
    
    cat <<EOF
    await page.type('$selector', '$text', {delay: $delay});
EOF
}

action::impl_select() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    local value=$(echo "$params" | jq -r '.value // ""')
    
    cat <<EOF
    await page.select('$selector', '$value');
EOF
}

action::impl_upload_file() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    local file_path=$(echo "$params" | jq -r '.file_path // ""')
    
    cat <<EOF
    const inputElement = await page.\$('$selector');
    await inputElement.uploadFile('$file_path');
EOF
}

action::impl_hover() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    
    cat <<EOF
    await page.hover('$selector');
EOF
}

action::impl_focus() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    
    cat <<EOF
    await page.focus('$selector');
EOF
}

action::impl_clear() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    
    cat <<EOF
    await page.evaluate((selector) => {
        const element = document.querySelector(selector);
        if (element) element.value = '';
    }, '$selector');
EOF
}

#######################################
# Wait Action Implementations
#######################################

action::impl_wait() {
    local params="$1"
    local duration=$(echo "$params" | jq -r '.duration // 1000')
    
    cat <<EOF
    await page.waitForTimeout($duration);
EOF
}

action::impl_wait_for_element() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    local timeout=$(echo "$params" | jq -r '.timeout // 30000')
    local visible=$(echo "$params" | jq -r '.visible // true')
    
    cat <<EOF
    await page.waitForSelector('$selector', {
        timeout: $timeout,
        visible: $visible
    });
EOF
}

action::impl_wait_for_text() {
    local params="$1"
    local text=$(echo "$params" | jq -r '.text // ""')
    local timeout=$(echo "$params" | jq -r '.timeout // 30000')
    
    cat <<EOF
    await page.waitForFunction(
        (text) => document.body.innerText.includes(text),
        {timeout: $timeout},
        '$text'
    );
EOF
}

action::impl_wait_for_navigation() {
    local params="$1"
    local wait_until=$(echo "$params" | jq -r '.wait_until // "networkidle2"')
    
    cat <<EOF
    await page.waitForNavigation({
        waitUntil: '$wait_until'
    });
EOF
}

action::impl_wait_for_redirect() {
    local params="$1"
    local expected_url=$(echo "$params" | jq -r '.expected_url // ""')
    local timeout=$(echo "$params" | jq -r '.timeout // 30000')
    
    cat <<EOF
    await page.waitForFunction(
        (url) => window.location.href.includes(url),
        {timeout: $timeout},
        '$expected_url'
    );
EOF
}

action::impl_wait_for_network_idle() {
    local params="$1"
    local idle_time=$(echo "$params" | jq -r '.idle_time // 500')
    
    cat <<EOF
    await page.waitForLoadState('networkidle', {
        timeout: $idle_time
    });
EOF
}

#######################################
# Data Action Implementations
#######################################

action::impl_screenshot() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    local output=$(echo "$params" | jq -r '.output // ""')
    local full_page=$(echo "$params" | jq -r '.full_page // false')
    
    if [[ -n "$selector" && "$selector" != "null" ]]; then
        cat <<EOF
    const element = await page.\$('$selector');
    const screenshot = await element.screenshot({
        path: '$output'
    });
EOF
    else
        cat <<EOF
    const screenshot = await page.screenshot({
        path: '$output',
        fullPage: $full_page
    });
EOF
    fi
}

action::impl_extract_text() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    
    cat <<EOF
    const text = await page.\$eval('$selector', el => el.textContent);
    results.push({text});
EOF
}

action::impl_extract_data() {
    local params="$1"
    local selectors=$(echo "$params" | jq -r '.selectors // {}')
    
    cat <<EOF
    const data = {};
    const selectors = $selectors;
    
    for (const [key, selector] of Object.entries(selectors)) {
        try {
            data[key] = await page.\$eval(selector, el => el.textContent);
        } catch (e) {
            data[key] = null;
        }
    }
    results.push({extracted_data: data});
EOF
}

#######################################
# Validation Action Implementations
#######################################

action::impl_assert_text() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    local expected=$(echo "$params" | jq -r '.expected // ""')
    
    cat <<EOF
    const actualText = await page.\$eval('$selector', el => el.textContent);
    if (!actualText.includes('$expected')) {
        throw new Error(\`Text assertion failed: expected '\${expected}' but got '\${actualText}'\`);
    }
EOF
}

action::impl_assert_url() {
    local params="$1"
    local expected_url=$(echo "$params" | jq -r '.expected_url // ""')
    
    cat <<EOF
    const currentUrl = page.url();
    if (!currentUrl.includes('$expected_url')) {
        throw new Error(\`URL assertion failed: expected '\${expected_url}' but got '\${currentUrl}'\`);
    }
EOF
}

action::impl_assert_element() {
    local params="$1"
    local selector=$(echo "$params" | jq -r '.selector // ""')
    
    cat <<EOF
    const element = await page.\$('$selector');
    if (!element) {
        throw new Error('Element assertion failed: $selector not found');
    }
EOF
}

#######################################
# Advanced Action Implementations
#######################################

action::impl_execute_script() {
    local params="$1"
    local script=$(echo "$params" | jq -r '.script // ""')
    
    cat <<EOF
    const scriptResult = await page.evaluate(() => {
        $script
    });
    results.push({script_result: scriptResult});
EOF
}

action::impl_scroll() {
    local params="$1"
    local direction=$(echo "$params" | jq -r '.direction // "down"')
    local amount=$(echo "$params" | jq -r '.amount // 500')
    
    if [[ "$direction" == "down" ]]; then
        cat <<EOF
    await page.evaluate((amount) => {
        window.scrollBy(0, amount);
    }, $amount);
EOF
    else
        cat <<EOF
    await page.evaluate((amount) => {
        window.scrollBy(0, -amount);
    }, $amount);
EOF
    fi
}

action::impl_set_viewport() {
    local params="$1"
    local width=$(echo "$params" | jq -r '.width // 1920')
    local height=$(echo "$params" | jq -r '.height // 1080')
    
    cat <<EOF
    await page.setViewport({
        width: $width,
        height: $height
    });
EOF
}

action::impl_log() {
    local params="$1"
    local message=$(echo "$params" | jq -r '.message // "Log message"')
    local level=$(echo "$params" | jq -r '.level // "info"')
    
    cat <<EOF
    console.log('[${level^^}]', \`$message\`);
    results.push({
        action: 'log',
        level: '$level',
        message: \`$message\`,
        timestamp: new Date().toISOString()
    });
EOF
}

# Export functions
export -f action::get_implementation