#!/usr/bin/env bash

#######################################
# Browserless Actions - CLI Interface
# 
# Provides command-line interface for atomic browser operations.
# Each action is designed for single-purpose agent tasks without
# requiring full workflow files.
#######################################

set -euo pipefail

# Get script directory
ACTIONS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required libraries
source "$ACTIONS_DIR/browser-ops.sh"
source "$ACTIONS_DIR/session-manager.sh"
source "/home/matthalloran8/Vrooli/scripts/lib/utils/log.sh" 2>/dev/null || true

# Global variables for universal options
OUTPUT_PATH=""
TIMEOUT_MS="30000"
WAIT_MS="2000"
SESSION_NAME=""
FULL_PAGE="false"
VIEWPORT_WIDTH="1920"
VIEWPORT_HEIGHT="1080"

#######################################
# Parse universal options from arguments
# Sets global variables for common options
#######################################
actions::parse_universal_options() {
    # Reset to defaults (globals are already declared)
    OUTPUT_PATH=""
    TIMEOUT_MS="30000"
    WAIT_MS="2000"
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                TIMEOUT_MS="$2"
                shift 2
                ;;
            --wait-ms)
                WAIT_MS="$2"
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            *)
                # Return remaining arguments
                echo "$@"
                return 0
                ;;
        esac
    done
    
    echo ""
}

#######################################
# Create temporary session for one-off commands
#######################################
actions::create_temp_session() {
    if [[ -n "$SESSION_NAME" ]]; then
        echo "$SESSION_NAME"
    else
        local temp_session="temp_$(date +%s)_$$"
        session::create "$temp_session" >/dev/null 2>&1
        echo "$temp_session"
    fi
}

#######################################
# Cleanup temporary session
#######################################
actions::cleanup_temp_session() {
    local session_id="$1"
    
    if [[ "$session_id" =~ ^temp_ ]] && [[ -z "$SESSION_NAME" ]]; then
        session::destroy "$session_id" >/dev/null 2>&1
    fi
}

#######################################
# Take a screenshot of a URL
# Usage: browserless screenshot <url> [options]
#######################################
actions::screenshot() {
    local remaining_args
    remaining_args=$(actions::parse_universal_options "$@")
    read -ra args <<< "$remaining_args"
    
    local url="${args[0]:-}"
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless screenshot <url> [--output screenshot.png] [--fullpage] [--wait-ms 3000]" >&2
        return 1
    fi
    
    # Set default output if not specified
    if [[ -z "$OUTPUT_PATH" ]]; then
        OUTPUT_PATH="screenshot-$(date +%s).png"
    fi
    
    # Ensure output directory exists
    mkdir -p "$(dirname "$OUTPUT_PATH")"
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "📸 Taking screenshot of $url"
    
    # Use combined navigate and screenshot for consistent context
    local result
    if result=$(browser::navigate_and_screenshot "$url" "$OUTPUT_PATH" "$session_id" "networkidle2" "$FULL_PAGE"); then
        local success=$(echo "$result" | jq -r '.success // false')
        if [[ "$success" == "true" ]]; then
            local title=$(echo "$result" | jq -r '.title // "Unknown"')
            echo "Screenshot saved: $OUTPUT_PATH"
            echo "Page title: $title"
            echo "URL: $(echo "$result" | jq -r '.url')"
        else
            local error=$(echo "$result" | jq -r '.error // "Unknown error"')
            echo "Error: $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Failed to take screenshot" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
    
    actions::cleanup_temp_session "$session_id"
    return 0
}

#######################################
# Navigate to a URL and return basic info
# Usage: browserless navigate <url> [options]
#######################################
actions::navigate() {
    local remaining_args
    remaining_args=$(actions::parse_universal_options "$@")
    read -ra args <<< "$remaining_args"
    
    local url="${args[0]:-}"
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless navigate <url> [--output result.json] [--wait-ms 2000]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "🌐 Navigating to $url"
    
    # Wait initial period before navigation
    browser::wait "$WAIT_MS" "$session_id"
    
    local result
    if result=$(browser::navigate "$url" "$session_id"); then
        local success=$(echo "$result" | jq -r '.success // false')
        if [[ "$success" == "true" ]]; then
            local clean_result
            clean_result=$(echo "$result" | jq '{
                success: .success,
                url: .url,
                title: .title,
                timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ"))
            }')
            
            if [[ -n "$OUTPUT_PATH" ]]; then
                mkdir -p "$(dirname "$OUTPUT_PATH")"
                echo "$clean_result" > "$OUTPUT_PATH"
                echo "Navigation result saved: $OUTPUT_PATH"
            fi
            
            echo "Successfully navigated to: $(echo "$clean_result" | jq -r '.url')"
            echo "Page title: $(echo "$clean_result" | jq -r '.title')"
        else
            local error=$(echo "$result" | jq -r '.error // "Unknown error"')
            echo "Error: $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Navigation failed" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
    
    actions::cleanup_temp_session "$session_id"
    return 0
}

#######################################
# Check if a URL loads successfully
# Usage: browserless health-check <url> [options]
#######################################
actions::health_check() {
    local remaining_args
    remaining_args=$(actions::parse_universal_options "$@")
    read -ra args <<< "$remaining_args"
    
    local url="${args[0]:-}"
    local expected_text="${args[1]:-}"
    
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless health-check <url> [expected-text] [--timeout 10000]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "🏥 Health checking $url"
    
    local result
    if result=$(browser::navigate "$url" "$session_id"); then
        local success=$(echo "$result" | jq -r '.success // false')
        if [[ "$success" == "true" ]]; then
            local healthy=true
            local health_result
            health_result=$(echo "$result" | jq '{
                url: .url,
                title: .title,
                healthy: true,
                timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ"))
            }')
            
            # Check for expected text if provided
            if [[ -n "$expected_text" ]]; then
                local page_content
                if page_content=$(browser::evaluate "return document.body.textContent;" "$session_id"); then
                    local content_success=$(echo "$page_content" | jq -r '.success // false')
                    if [[ "$content_success" == "true" ]]; then
                        local content=$(echo "$page_content" | jq -r '.result // ""')
                        if [[ "$content" == *"$expected_text"* ]]; then
                            health_result=$(echo "$health_result" | jq --arg text "$expected_text" '.expected_text_found = true | .expected_text = $text')
                        else
                            healthy=false
                            health_result=$(echo "$health_result" | jq --arg text "$expected_text" '.healthy = false | .expected_text_found = false | .expected_text = $text')
                        fi
                    else
                        healthy=false
                        health_result=$(echo "$health_result" | jq '.healthy = false | .error = "Could not read page content"')
                    fi
                fi
            fi
            
            if [[ -n "$OUTPUT_PATH" ]]; then
                mkdir -p "$(dirname "$OUTPUT_PATH")"
                echo "$health_result" > "$OUTPUT_PATH"
                echo "Health check result saved: $OUTPUT_PATH"
            fi
            
            if [[ "$healthy" == "true" ]]; then
                echo "✅ Health check passed"
                echo "URL: $(echo "$health_result" | jq -r '.url')"
                echo "Title: $(echo "$health_result" | jq -r '.title')"
                if [[ -n "$expected_text" ]]; then
                    echo "Expected text found: $expected_text"
                fi
            else
                echo "❌ Health check failed"
                if [[ -n "$expected_text" ]]; then
                    echo "Expected text not found: $expected_text"
                fi
                actions::cleanup_temp_session "$session_id"
                return 1
            fi
        else
            local error=$(echo "$result" | jq -r '.error // "Unknown error"')
            echo "❌ Health check failed: $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "❌ Health check failed: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
    
    actions::cleanup_temp_session "$session_id"
    return 0
}

#######################################
# Check if an element exists on a page
# Usage: browserless element-exists <url> --selector "button.login" [options]
#######################################
actions::element_exists() {
    local remaining_args
    remaining_args=$(actions::parse_universal_options "$@")
    read -ra args <<< "$remaining_args"
    
    local url="${args[0]:-}"
    local selector=""
    
    # Parse selector from remaining args
    local i=1
    while [[ $i -lt ${#args[@]} ]]; do
        case "${args[$i]}" in
            --selector)
                selector="${args[$((i+1))]}"
                i=$((i+2))
                ;;
            *)
                i=$((i+1))
                ;;
        esac
    done
    
    if [[ -z "$url" ]] || [[ -z "$selector" ]]; then
        echo "Error: URL and selector required" >&2
        echo "Usage: browserless element-exists <url> --selector \"button.login\" [--timeout 5000]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "🔍 Checking if element exists: $selector"
    
    # Navigate first
    local nav_result
    if nav_result=$(browser::navigate "$url" "$session_id"); then
        local nav_success=$(echo "$nav_result" | jq -r '.success // false')
        if [[ "$nav_success" == "true" ]]; then
            # Wait for page to stabilize
            browser::wait "$WAIT_MS" "$session_id"
            
            # Check if element exists
            local exists
            if exists=$(browser::element_exists "$selector" "$session_id"); then
                if [[ "$exists" == "true" ]]; then
                    echo "✅ Element exists: $selector"
                    actions::cleanup_temp_session "$session_id"
                    return 0
                else
                    echo "❌ Element not found: $selector" >&2
                    actions::cleanup_temp_session "$session_id"
                    return 1
                fi
            else
                echo "❌ Error checking element: $selector" >&2
                actions::cleanup_temp_session "$session_id"
                return 1
            fi
        else
            local error=$(echo "$nav_result" | jq -r '.error // "Unknown error"')
            echo "Error: Navigation failed - $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
}

#######################################
# Extract text content from a page
# Usage: browserless extract-text <url> --selector "h1" [options]
#######################################
actions::extract_text() {
    local remaining_args
    remaining_args=$(actions::parse_universal_options "$@")
    read -ra args <<< "$remaining_args"
    
    local url="${args[0]:-}"
    local selector=""
    
    # Parse selector from remaining args
    local i=1
    while [[ $i -lt ${#args[@]} ]]; do
        case "${args[$i]}" in
            --selector)
                selector="${args[$((i+1))]}"
                i=$((i+2))
                ;;
            *)
                i=$((i+1))
                ;;
        esac
    done
    
    if [[ -z "$url" ]] || [[ -z "$selector" ]]; then
        echo "Error: URL and selector required" >&2
        echo "Usage: browserless extract-text <url> --selector \"h1\" [--output title.txt]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "📄 Extracting text from: $selector"
    
    # Navigate first
    local nav_result
    if nav_result=$(browser::navigate "$url" "$session_id"); then
        local nav_success=$(echo "$nav_result" | jq -r '.success // false')
        if [[ "$nav_success" == "true" ]]; then
            # Wait for page to stabilize
            browser::wait "$WAIT_MS" "$session_id"
            
            # Extract text
            local text
            if text=$(browser::get_text "$selector" "$session_id"); then
                if [[ -n "$OUTPUT_PATH" ]]; then
                    mkdir -p "$(dirname "$OUTPUT_PATH")"
                    echo "$text" > "$OUTPUT_PATH"
                    echo "Text saved: $OUTPUT_PATH"
                fi
                
                echo "Extracted text: $text"
                actions::cleanup_temp_session "$session_id"
                return 0
            else
                echo "Error: Could not extract text from selector: $selector" >&2
                actions::cleanup_temp_session "$session_id"
                return 1
            fi
        else
            local error=$(echo "$nav_result" | jq -r '.error // "Unknown error"')
            echo "Error: Navigation failed - $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
}

#######################################
# Extract structured data using custom JavaScript
# Usage: browserless extract <url> --script "return {title: document.title}" [options]
#######################################
actions::extract() {
    local remaining_args
    remaining_args=$(actions::parse_universal_options "$@")
    read -ra args <<< "$remaining_args"
    
    local url="${args[0]:-}"
    local script=""
    
    # Parse script from remaining args
    local i=1
    while [[ $i -lt ${#args[@]} ]]; do
        case "${args[$i]}" in
            --script)
                script="${args[$((i+1))]}"
                i=$((i+2))
                ;;
            *)
                i=$((i+1))
                ;;
        esac
    done
    
    if [[ -z "$url" ]] || [[ -z "$script" ]]; then
        echo "Error: URL and script required" >&2
        echo "Usage: browserless extract <url> --script \"return {title: document.title}\" [--output data.json]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "⚙️  Extracting data with custom script"
    
    # Navigate first
    local nav_result
    if nav_result=$(browser::navigate "$url" "$session_id"); then
        local nav_success=$(echo "$nav_result" | jq -r '.success // false')
        if [[ "$nav_success" == "true" ]]; then
            # Wait for page to stabilize
            browser::wait "$WAIT_MS" "$session_id"
            
            # Execute custom script
            local result
            if result=$(browser::evaluate "$script" "$session_id"); then
                local eval_success=$(echo "$result" | jq -r '.success // false')
                if [[ "$eval_success" == "true" ]]; then
                    local extracted_data=$(echo "$result" | jq '.result')
                    
                    if [[ -n "$OUTPUT_PATH" ]]; then
                        mkdir -p "$(dirname "$OUTPUT_PATH")"
                        echo "$extracted_data" | jq '.' > "$OUTPUT_PATH"
                        echo "Data saved: $OUTPUT_PATH"
                    fi
                    
                    echo "Extracted data:"
                    echo "$extracted_data" | jq '.'
                    actions::cleanup_temp_session "$session_id"
                    return 0
                else
                    local error=$(echo "$result" | jq -r '.error // "Unknown error"')
                    echo "Error: Script execution failed - $error" >&2
                    actions::cleanup_temp_session "$session_id"
                    return 1
                fi
            else
                echo "Error: Could not execute script" >&2
                actions::cleanup_temp_session "$session_id"
                return 1
            fi
        else
            local error=$(echo "$nav_result" | jq -r '.error // "Unknown error"')
            echo "Error: Navigation failed - $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
}

#######################################
# Perform basic interactions (fill forms, click buttons)
# Usage: browserless interact <url> --fill "input[name=email]:admin@example.com" --click "button[type=submit]" [options]
#######################################
actions::interact() {
    local remaining_args
    remaining_args=$(actions::parse_universal_options "$@")
    read -ra args <<< "$remaining_args"
    
    local url="${args[0]:-}"
    local interactions=()
    
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless interact <url> --fill \"input[name=email]:value\" --click \"button[type=submit]\" [--screenshot result.png]" >&2
        return 1
    fi
    
    # Parse interactions from remaining args
    local i=1
    while [[ $i -lt ${#args[@]} ]]; do
        case "${args[$i]}" in
            --fill)
                interactions+=("fill:${args[$((i+1))]}")
                i=$((i+2))
                ;;
            --click)
                interactions+=("click:${args[$((i+1))]}")
                i=$((i+2))
                ;;
            --wait-for)
                interactions+=("wait-for:${args[$((i+1))]}")
                i=$((i+2))
                ;;
            *)
                i=$((i+1))
                ;;
        esac
    done
    
    if [[ ${#interactions[@]} -eq 0 ]]; then
        echo "Error: At least one interaction required" >&2
        echo "Available interactions: --fill, --click, --wait-for" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "🤖 Performing interactions on $url"
    
    # Navigate first
    local nav_result
    if nav_result=$(browser::navigate "$url" "$session_id"); then
        local nav_success=$(echo "$nav_result" | jq -r '.success // false')
        if [[ "$nav_success" == "true" ]]; then
            # Wait for page to stabilize
            browser::wait "$WAIT_MS" "$session_id"
            
            # Execute interactions in order
            for interaction in "${interactions[@]}"; do
                local action="${interaction%%:*}"
                local param="${interaction#*:}"
                
                case "$action" in
                    fill)
                        # Format: "selector:value"
                        local selector="${param%%:*}"
                        local value="${param#*:}"
                        log::info "✍️  Filling $selector with: $value"
                        if ! browser::fill "$selector" "$value" "$session_id"; then
                            echo "Error: Failed to fill $selector" >&2
                            actions::cleanup_temp_session "$session_id"
                            return 1
                        fi
                        ;;
                    click)
                        log::info "👆 Clicking: $param"
                        if ! browser::click "$param" "$session_id"; then
                            echo "Error: Failed to click $param" >&2
                            actions::cleanup_temp_session "$session_id"
                            return 1
                        fi
                        ;;
                    wait-for)
                        log::info "⏳ Waiting for element: $param"
                        if ! browser::wait_for_element "$param" "$TIMEOUT_MS" "$session_id"; then
                            echo "Error: Element did not appear: $param" >&2
                            actions::cleanup_temp_session "$session_id"
                            return 1
                        fi
                        ;;
                esac
                
                # Small wait between interactions
                browser::wait "500" "$session_id"
            done
            
            # Take final screenshot if output path provided
            if [[ -n "$OUTPUT_PATH" ]]; then
                mkdir -p "$(dirname "$OUTPUT_PATH")"
                if browser::screenshot "$OUTPUT_PATH" "$session_id" "$FULL_PAGE"; then
                    echo "Final screenshot saved: $OUTPUT_PATH"
                else
                    echo "Warning: Could not save screenshot" >&2
                fi
            fi
            
            echo "✅ All interactions completed successfully"
            actions::cleanup_temp_session "$session_id"
            return 0
        else
            local error=$(echo "$nav_result" | jq -r '.error // "Unknown error"')
            echo "Error: Navigation failed - $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
}

#######################################
# Capture console logs from a page
# Usage: browserless console <url> [--filter error] [options]
#######################################
actions::console() {
    local remaining_args
    remaining_args=$(actions::parse_universal_options "$@")
    read -ra args <<< "$remaining_args"
    
    local url="${args[0]:-}"
    local filter=""
    
    # Parse filter from remaining args
    local i=1
    while [[ $i -lt ${#args[@]} ]]; do
        case "${args[$i]}" in
            --filter)
                filter="${args[$((i+1))]}"
                i=$((i+2))
                ;;
            *)
                i=$((i+1))
                ;;
        esac
    done
    
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless console <url> [--filter error] [--output console.json]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "🖥️  Capturing console logs from $url"
    
    # Use combined approach to capture console logs
    local result
    result=$(browser::execute_js "
        // Set up console log capturing
        const logs = [];
        const originalLog = console.log;
        const originalError = console.error;
        const originalWarn = console.warn;
        const originalInfo = console.info;
        
        console.log = function(...args) {
            logs.push({level: 'log', message: args.join(' '), timestamp: new Date().toISOString()});
            originalLog.apply(console, args);
        };
        console.error = function(...args) {
            logs.push({level: 'error', message: args.join(' '), timestamp: new Date().toISOString()});
            originalError.apply(console, args);
        };
        console.warn = function(...args) {
            logs.push({level: 'warn', message: args.join(' '), timestamp: new Date().toISOString()});
            originalWarn.apply(console, args);
        };
        console.info = function(...args) {
            logs.push({level: 'info', message: args.join(' '), timestamp: new Date().toISOString()});
            originalInfo.apply(console, args);
        };
        
        // Navigate to the URL
        await page.goto('$url', { 
            waitUntil: 'networkidle2',
            timeout: 30000 
        });
        
        // Wait for any additional console activity
        await new Promise(resolve => setTimeout(resolve, $WAIT_MS));
        
        // Also capture any existing console messages if available
        const runtimeLogs = await page.evaluate(() => {
            // Try to get existing console messages (this is limited by browser)
            return logs;
        });
        
        return {
            success: true,
            logs: runtimeLogs,
            url: page.url(),
            title: await page.title()
        };
    " "$session_id")
    
    local success=$(echo "$result" | jq -r '.success // false')
    if [[ "$success" == "true" ]]; then
        local logs=$(echo "$result" | jq '.logs')
        
        # Apply filter if specified
        if [[ -n "$filter" ]]; then
            logs=$(echo "$logs" | jq --arg filter "$filter" '[.[] | select(.level == $filter)]')
        fi
        
        local filtered_result
        filtered_result=$(echo "$result" | jq --argjson logs "$logs" '.logs = $logs')
        
        if [[ -n "$OUTPUT_PATH" ]]; then
            mkdir -p "$(dirname "$OUTPUT_PATH")"
            echo "$filtered_result" > "$OUTPUT_PATH"
            echo "Console logs saved: $OUTPUT_PATH"
        fi
        
        local log_count=$(echo "$logs" | jq 'length')
        echo "Captured $log_count console messages"
        if [[ -n "$filter" ]]; then
            echo "Filter: $filter"
        fi
        echo "$logs" | jq '.'
    else
        local error=$(echo "$result" | jq -r '.error // "Unknown error"')
        echo "Error: Console capture failed - $error" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
    
    actions::cleanup_temp_session "$session_id"
    return 0
}

#######################################
# Get page performance metrics
# Usage: browserless performance <url> [options]
#######################################
actions::performance() {
    local remaining_args
    remaining_args=$(actions::parse_universal_options "$@")
    read -ra args <<< "$remaining_args"
    
    local url="${args[0]:-}"
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless performance <url> [--output metrics.json]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "⚡ Measuring performance for $url"
    
    local result
    result=$(browser::execute_js "
        const startTime = performance.now();
        
        // Navigate to the URL
        await page.goto('$url', { 
            waitUntil: 'networkidle2',
            timeout: 30000 
        });
        
        // Get performance metrics
        const metrics = await page.evaluate(() => {
            const perf = performance;
            const navigation = perf.getEntriesByType('navigation')[0];
            
            return {
                // Navigation timing
                dns_lookup: navigation ? navigation.domainLookupEnd - navigation.domainLookupStart : 0,
                tcp_connect: navigation ? navigation.connectEnd - navigation.connectStart : 0,
                request_response: navigation ? navigation.responseEnd - navigation.requestStart : 0,
                dom_interactive: navigation ? navigation.domInteractive - navigation.fetchStart : 0,
                dom_complete: navigation ? navigation.domComplete - navigation.fetchStart : 0,
                load_event: navigation ? navigation.loadEventEnd - navigation.loadEventStart : 0,
                
                // Page metrics
                total_load_time: navigation ? navigation.loadEventEnd - navigation.fetchStart : 0,
                first_paint: null,
                first_contentful_paint: null,
                
                // Resource counts
                resource_count: perf.getEntriesByType('resource').length,
                
                // Memory usage (if available)
                memory_used: navigator.deviceMemory || null,
                memory_limit: navigator.deviceMemory ? navigator.deviceMemory * 1024 * 1024 * 1024 : null,
                
                // Connection info
                connection_type: navigator.connection ? navigator.connection.effectiveType : null,
                connection_downlink: navigator.connection ? navigator.connection.downlink : null
            };
        });
        
        // Try to get paint timing if available
        const paintMetrics = await page.evaluate(() => {
            const paintEntries = performance.getEntriesByType('paint');
            const result = {};
            paintEntries.forEach(entry => {
                if (entry.name === 'first-paint') {
                    result.first_paint = entry.startTime;
                } else if (entry.name === 'first-contentful-paint') {
                    result.first_contentful_paint = entry.startTime;
                }
            });
            return result;
        });
        
        const totalTime = performance.now() - startTime;
        
        return {
            success: true,
            metrics: {
                ...metrics,
                ...paintMetrics,
                total_measurement_time: totalTime
            },
            url: page.url(),
            title: await page.title(),
            timestamp: new Date().toISOString()
        };
    " "$session_id")
    
    local success=$(echo "$result" | jq -r '.success // false')
    if [[ "$success" == "true" ]]; then
        if [[ -n "$OUTPUT_PATH" ]]; then
            mkdir -p "$(dirname "$OUTPUT_PATH")"
            echo "$result" > "$OUTPUT_PATH"
            echo "Performance metrics saved: $OUTPUT_PATH"
        fi
        
        local metrics=$(echo "$result" | jq '.metrics')
        echo "Performance Metrics:"
        echo "  Total Load Time: $(echo "$metrics" | jq -r '.total_load_time // "N/A"')ms"
        echo "  DOM Interactive: $(echo "$metrics" | jq -r '.dom_interactive // "N/A"')ms"
        echo "  DOM Complete: $(echo "$metrics" | jq -r '.dom_complete // "N/A"')ms"
        echo "  First Paint: $(echo "$metrics" | jq -r '.first_paint // "N/A"')ms"
        echo "  First Contentful Paint: $(echo "$metrics" | jq -r '.first_contentful_paint // "N/A"')ms"
        echo "  Resource Count: $(echo "$metrics" | jq -r '.resource_count // "N/A"')"
        echo "  Connection Type: $(echo "$metrics" | jq -r '.connection_type // "N/A"')"
    else
        local error=$(echo "$result" | jq -r '.error // "Unknown error"')
        echo "Error: Performance measurement failed - $error" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
    
    actions::cleanup_temp_session "$session_id"
    return 0
}

#######################################
# Extract form data and input fields
# Usage: browserless extract-forms <url> [options]
#######################################
actions::extract_forms() {
    local remaining_args
    remaining_args=$(actions::parse_universal_options "$@")
    read -ra args <<< "$remaining_args"
    
    local url="${args[0]:-}"
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless extract-forms <url> [--output forms.json]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "📝 Extracting forms from $url"
    
    # Navigate first
    local nav_result
    if nav_result=$(browser::navigate "$url" "$session_id"); then
        local nav_success=$(echo "$nav_result" | jq -r '.success // false')
        if [[ "$nav_success" == "true" ]]; then
            # Wait for page to stabilize
            browser::wait "$WAIT_MS" "$session_id"
            
            # Extract form data
            local result
            if result=$(browser::evaluate "
                const forms = Array.from(document.querySelectorAll('form')).map((form, index) => {
                    const inputs = Array.from(form.querySelectorAll('input, select, textarea')).map(input => ({
                        type: input.type || input.tagName.toLowerCase(),
                        name: input.name || '',
                        id: input.id || '',
                        placeholder: input.placeholder || '',
                        required: input.required || false,
                        value: input.value || '',
                        className: input.className || ''
                    }));
                    
                    return {
                        index: index,
                        action: form.action || '',
                        method: form.method || 'get',
                        id: form.id || '',
                        className: form.className || '',
                        inputs: inputs,
                        inputCount: inputs.length
                    };
                });
                
                return {
                    forms: forms,
                    formCount: forms.length,
                    totalInputs: forms.reduce((sum, form) => sum + form.inputCount, 0)
                };
            " "$session_id"); then
                local eval_success=$(echo "$result" | jq -r '.success // false')
                if [[ "$eval_success" == "true" ]]; then
                    local forms_data=$(echo "$result" | jq '.result')
                    
                    if [[ -n "$OUTPUT_PATH" ]]; then
                        mkdir -p "$(dirname "$OUTPUT_PATH")"
                        echo "$forms_data" | jq '.' > "$OUTPUT_PATH"
                        echo "Forms data saved: $OUTPUT_PATH"
                    fi
                    
                    local form_count=$(echo "$forms_data" | jq -r '.formCount')
                    local total_inputs=$(echo "$forms_data" | jq -r '.totalInputs')
                    echo "Found $form_count forms with $total_inputs total inputs"
                    echo "$forms_data" | jq '.'
                    actions::cleanup_temp_session "$session_id"
                    return 0
                else
                    local error=$(echo "$result" | jq -r '.error // "Unknown error"')
                    echo "Error: Form extraction failed - $error" >&2
                    actions::cleanup_temp_session "$session_id"
                    return 1
                fi
            else
                echo "Error: Could not extract forms" >&2
                actions::cleanup_temp_session "$session_id"
                return 1
            fi
        else
            local error=$(echo "$nav_result" | jq -r '.error // "Unknown error"')
            echo "Error: Navigation failed - $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
}

#######################################
# Dispatch function for CLI routing
#######################################
actions::dispatch() {
    local action="$1"
    shift
    
    case "$action" in
        screenshot)
            actions::screenshot "$@"
            ;;
        navigate)
            actions::navigate "$@"
            ;;
        health-check)
            actions::health_check "$@"
            ;;
        element-exists)
            actions::element_exists "$@"
            ;;
        extract-text)
            actions::extract_text "$@"
            ;;
        extract)
            actions::extract "$@"
            ;;
        extract-forms)
            actions::extract_forms "$@"
            ;;
        interact)
            actions::interact "$@"
            ;;
        console)
            actions::console "$@"
            ;;
        performance)
            actions::performance "$@"
            ;;
        *)
            echo "Error: Unknown action: $action" >&2
            echo "" >&2
            echo "Available actions:" >&2
            echo "  screenshot     - Take screenshots of URLs" >&2
            echo "  navigate       - Navigate to URLs and get basic info" >&2
            echo "  health-check   - Check if URLs load successfully" >&2
            echo "  element-exists - Check if elements exist on pages" >&2
            echo "  extract-text   - Extract text content from elements" >&2
            echo "  extract        - Extract structured data with custom scripts" >&2
            echo "  extract-forms  - Extract form data and input fields" >&2
            echo "  interact       - Perform form fills, clicks, and wait operations" >&2
            echo "  console        - Capture console logs from pages" >&2
            echo "  performance    - Measure page performance metrics" >&2
            echo "" >&2
            echo "Universal options:" >&2
            echo "  --output <path>    - Save result to file" >&2
            echo "  --timeout <ms>     - Max wait time (default: 30000)" >&2
            echo "  --wait-ms <ms>     - Initial wait before action (default: 2000)" >&2
            echo "  --session <name>   - Use persistent session" >&2
            echo "  --fullpage         - Full page screenshots" >&2
            echo "  --mobile           - Use mobile viewport" >&2
            return 1
            ;;
    esac
}

# Export the dispatch function
export -f actions::dispatch